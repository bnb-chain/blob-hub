package syncer

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/bnb-chain/blob-syncer/config"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"gorm.io/gorm"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	CreateBundleBlockInterval = 1

	BundleStatusBundling       = 0
	BundleStatusFinalized      = 1
	BundleStatusCreatedOnChain = 2
	BundleStatusSealedOnChain  = 3 // todo The post verification process should check if a bundle is indeed sealed onchain
	BundleStatusExpired        = 4
)

type curBundleDetail struct {
	name           string
	startHeight    uint64
	finalizeHeight uint64
}

type BlobSyncer struct {
	blobDao      db.BlobDao
	ethClients   *external.ETHClient
	bundleClient *external.BundleClient
	config       *config.Config
	bundleDetail *curBundleDetail
}

func NewBlobSyncer(
	blobDao db.BlobDao,
	config *config.Config,
) *BlobSyncer {
	bundleClient, err := external.NewBundleClient(config.SyncerConfig.BundleServiceEndpoints[0], config.SyncerConfig.PrivateKey)
	if err != nil {
		panic(err)
	}
	clients := external.NewETHClient(config.SyncerConfig.ETHRPCAddrs[0], config.SyncerConfig.BeaconRPCAddrs[0])
	return &BlobSyncer{
		blobDao:      blobDao,
		ethClients:   clients,
		bundleClient: bundleClient,
		config:       config,
	}
}

func (l *BlobSyncer) StartLoop() {
	for {
		if err := l.process(); err != nil {
			logging.Logger.Error(err)
			continue
		}
	}
}

func (l *BlobSyncer) process() error {
	ctx := context.Background()
	nextHeight, err := l.calNextHeight()
	if err != nil {
		return err
	}

	// the app is just re-started.
	if l.bundleDetail == nil {
		// get latest bundle from DB or bundle service(if DB data is lost)
		err := l.LoadProgressAndResume(nextHeight)
		if err != nil {
			return fmt.Errorf("failed to LoadProgressAndResume, err=%s", err.Error())
		}
	}
	var isForkedBlock bool
	block, err := l.ethClients.BeaconClient.GetBlock(ctx, nextHeight)
	if err != nil {
		if err != external.ErrBlockNotFound {
			return err
		}
		// Both try to get forked block and non-exist block will return 404. When the response is ErrBlockNotFound,
		// check whether nextHeight is >= latest height, otherwise it is a forked block, should skip it.
		blockResp, err := l.ethClients.BeaconClient.GetLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest becon block, err=%s", err.Error())
		}
		clBlock, _, err := ToBlockAndExecutionPayloadDeneb(blockResp)
		if err != nil {
			return fmt.Errorf("failed to ToBlockAndExecutionPayloadDeneb, err=%s", err.Error())
		}
		if nextHeight >= uint64(clBlock.Slot) {
			logging.Logger.Debugf("the nextHeight %d is larger than current block height %d\n", nextHeight, clBlock.Slot)
			return nil
		} else {
			isForkedBlock = true
		}
	}

	if !isForkedBlock && !block.Finalized {
		logging.Logger.Infof("current block(h=%d) is not finalized yet", nextHeight)
		time.Sleep(1 * time.Minute) // around 15 minutes to finalize
		return nil
	}

	var sideCars []*structs.Sidecar
	if !isForkedBlock {
		sideCars, err = l.ethClients.BeaconClient.GetBlob(ctx, nextHeight)
		if err != nil {
			return err
		}
	}

	bundleName := l.bundleDetail.name
	// create a new bundle
	if nextHeight == l.bundleDetail.startHeight {
		if err := l.createBundle(); err != nil {
			logging.Logger.Errorf("failed to create bundle, bundle=%s, err=%s", bundleName, err.Error())
			return err
		}
	}

	err = l.uploadBlobs(nextHeight, sideCars)
	if err != nil {
		return err
	}

	if nextHeight == l.bundleDetail.finalizeHeight {
		if err = l.finalizeBundle(bundleName); err != nil {
			if strings.Contains(err.Error(), "expired") {
				err = l.bundleClient.DeleteBundle(bundleName, l.getBucketName())
				if err != nil {
					logging.Logger.Infof("failed to delete bundle, bundleName=%s, err=%s", bundleName, err.Error())
					return err
				}
				err = l.reProcessBundleAndFinalize(bundleName)
				if err != nil {
					logging.Logger.Infof("failed to re-process bundle, bundleName=%s, err=%s", bundleName, err.Error())
					return err
				}
			}
			return fmt.Errorf("failed to finalize bundle, bundle=%s, err=%s", bundleName, err.Error())
		}
		logging.Logger.Infof("finalized bundle, bundle_name=%s, bucket_name=%s\n", bundleName, l.getBucketName())

		// init next bundle
		startHeight := nextHeight + 1
		endHeight := nextHeight + l.getBlockInterval()
		l.bundleDetail = &curBundleDetail{
			name:           types.GetBundleName(startHeight, endHeight),
			startHeight:    startHeight,
			finalizeHeight: endHeight,
		}
	}

	if isForkedBlock {
		return l.blobDao.SaveBlockAndBlob(&db.Block{
			Height: nextHeight,
		}, nil)
	}

	blockToSave, blobToSave, err := l.ToBlockAndBlobs(block, sideCars, nextHeight, bundleName)
	if err != nil {
		return err
	}
	err = l.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
	if err != nil {
		return fmt.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Height, len(blobToSave), err.Error())
	}
	logging.Logger.Infof("saved block and blobs(num=%d) at height %d to DB \n", len(blobToSave), nextHeight)
	return nil
}

func (l *BlobSyncer) getBucketName() string {
	return l.config.SyncerConfig.BucketName
}

func (l *BlobSyncer) getBlockInterval() uint64 {
	return CreateBundleBlockInterval
}

func (l *BlobSyncer) calNextHeight() (uint64, error) {
	latestProcessedBlock, err := l.blobDao.GetLatestProcessedBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest polled block from db, error: %s", err.Error())
	}
	latestPolledBlockHeight := latestProcessedBlock.Height
	nextHeight := l.config.SyncerConfig.StartSlot
	if nextHeight <= latestPolledBlockHeight {
		nextHeight = latestPolledBlockHeight + 1
	}
	return nextHeight, nil
}

func (l *BlobSyncer) createBundle() error {
	_, err := l.bundleClient.GetBundleInfo(l.getBucketName(), l.bundleDetail.name)
	if err != nil {
		if err != external.ErrorBundleNotExist {
			return err
		}
		err = l.bundleClient.CreateBundle(l.bundleDetail.name, l.getBucketName())
		if err != nil {
			return fmt.Errorf("failed to created bundle, bundle_name=%s, bucket_name=%s\n, err=%s", l.bundleDetail.name, l.getBucketName(), err.Error())
		}
		logging.Logger.Infof("created bundle, bundle_name=%s, bucket_name=%s\n", l.bundleDetail.name, l.getBucketName())
	}
	return l.blobDao.CreateBundle(
		&db.Bundle{
			Name:   l.bundleDetail.name,
			Status: db.Finalizing,
		})
}

func (l *BlobSyncer) finalizeBundle(bundleName string) error {
	bundleInfo, err := l.bundleClient.GetBundleInfo(l.getBucketName(), bundleName)
	if err != nil {
		return fmt.Errorf("failed to GetBundleInfo, bundle_name=%s, bucket_name=%s err=%s", l.bundleDetail.name, l.getBucketName(), err.Error())
	}
	if bundleInfo.Status == BundleStatusExpired {
		return fmt.Errorf("unexpect bundle status expired, name=%s", bundleInfo.BundleName)
	} else if bundleInfo.Status == BundleStatusBundling {
		// trying to finalize an empty bundle(contains no blob) will fail, so we need to delete it so that we can continue to create a new bundle for future blocks
		if bundleInfo.Size == 0 {
			err = l.bundleClient.DeleteBundle(l.bundleDetail.name, l.getBucketName())
			if err != nil {
				return fmt.Errorf("failed to delete bundle, bundle_name=%s, bucket_name=%s err=%s", l.bundleDetail.name, l.getBucketName(), err.Error())
			}
		} else {
			err = l.bundleClient.FinalizeBundle(l.bundleDetail.name, l.getBucketName())
			if err != nil {
				return fmt.Errorf("failed to finalize bundle, bundle_name=%s, bucket_name=%s err=%s", l.bundleDetail.name, l.getBucketName(), err.Error())
			}
		}
	} else {
		logging.Logger.Infof("bundle has already been finalized")
	}
	return l.blobDao.UpdateBundleStatus(bundleName, db.Finalized)
}

func (l *BlobSyncer) uploadBlobs(height uint64, blobs []*structs.Sidecar) error {
	// TODO concurrent upload in a single block
	for i, b := range blobs {
		blobName := types.GetBlobName(height, uint64(i))
		filePath := l.config.SyncerConfig.TempFilePath + "/" + blobName
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to CreateBlock file, err=%s", err.Error())
		}
		defer file.Close()
		_, err = file.WriteString(b.Blob)
		if err != nil {
			return fmt.Errorf("failed to WriteString, err=%s", err.Error())
		}
		file, err = os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to Open, err=%s", err.Error())
		}
		err = l.bundleClient.UploadObject(blobName, l.getBucketName(), l.bundleDetail.name, "text/plain", file)
		if err != nil {
			if strings.Contains(err.Error(), "Object already exists") {
				_ = os.Remove(filePath)
				return nil
			}
			return fmt.Errorf("failed to upload object to bundle service, err=%s", err.Error())
		}
		_ = os.Remove(filePath)
	}
	return nil
}

func (l *BlobSyncer) LoadProgressAndResume(nextHeight uint64) error {
	var (
		startHeight uint64
		endHeight   uint64
		err         error
	)
	finalizingBundle, err := l.blobDao.GetLatestFinalizingBundle()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		// There is no pending(finalizing) bundle, start a new bundle. e.g. a bundle includes
		// blobs from block height 0-9 when the block interval is config to 10
		startHeight = nextHeight
		endHeight = nextHeight + l.getBlockInterval() - 1
	} else {
		// check against bundle service if still exist before resume
		bundleResp, err := l.bundleClient.GetBundleInfo(l.getBucketName(), finalizingBundle.Name)
		if err != nil {
			// if the bundle recorded in DB not found in bundle service(shouldn't happen), then need to reprocess all blobs within the bundle
			if err == external.ErrorBundleNotExist {
				err = l.bundleClient.CreateBundle(finalizingBundle.Name, l.getBucketName())
				if err != nil {
					return err
				}
				err = l.reProcessBundleUntilHeight(finalizingBundle.Name, nextHeight) // make up missing blobs
				if err != nil {
					return err
				}
				startHeight, endHeight, err = types.ParseBundleName(finalizingBundle.Name)
				if err != nil {
					return err
				}
				l.bundleDetail = &curBundleDetail{
					name:           finalizingBundle.Name,
					startHeight:    startHeight,
					finalizeHeight: endHeight,
				}
				return nil
			} else {
				logging.Logger.Errorf("failed to get bundle info from bundle service")
				return err
			}
		}
		// could fail to update the DB when shutdown the app previously
		if bundleResp.Status == BundleStatusFinalized {
			if err = l.blobDao.UpdateBundleStatus(finalizingBundle.Name, db.Finalized); err != nil {
				return err
			}
			_, endHeight, err = types.ParseBundleName(finalizingBundle.Name)
			if err != nil {
				return err
			}
			// start a new bundle
			startHeight = endHeight + 1
			endHeight = startHeight + l.getBlockInterval() - 1
		} else {
			// resume the current bundle processing, note the config interval might change after the reboot, but it
			// will not take effect until the last existing bundle finalized.
			startHeight, endHeight, err = types.ParseBundleName(finalizingBundle.Name)
			if err != nil {
				return err
			}
		}
	}
	l.bundleDetail = &curBundleDetail{
		name:           types.GetBundleName(startHeight, endHeight),
		startHeight:    startHeight,
		finalizeHeight: endHeight,
	}
	return nil
}

func (l *BlobSyncer) ToBlockAndBlobs(blockResp *structs.GetBlockV2Response, blobs []*structs.Sidecar, height uint64, bundleName string) (*db.Block, []*db.Blob, error) {
	var blockReturn *db.Block
	blobsReturn := make([]*db.Blob, 0)

	var (
		clBlock          *ethpb.BeaconBlockDeneb
		executionPayload *v1.ExecutionPayloadDeneb
		err              error
	)

	switch blockResp.Version {
	case version.String(version.Deneb):
		clBlock, executionPayload, err = ToBlockAndExecutionPayloadDeneb(blockResp)
		if err != nil {
			return nil, nil, err
		}
		blockReturn = &db.Block{
			BlockHash:     hex.EncodeToString(clBlock.GetStateRoot()),
			ParentHash:    hex.EncodeToString(clBlock.GetParentRoot()),
			Height:        uint64(clBlock.GetSlot()),
			ELBlockHeight: executionPayload.GetBlockNumber(),
			BlobCount:     len(blobs),
		}
	default:
		return nil, nil, fmt.Errorf("un-expected block version %s", blockResp.Version)
	}

	for _, blob := range blobs {
		index, err := strconv.Atoi(blob.Index)
		if err != nil {
			return nil, nil, err
		}
		b := &db.Blob{
			Name:       types.GetBlobName(height, uint64(index)),
			Height:     height,
			Index:      index,
			BundleName: bundleName,
		}
		blobsReturn = append(blobsReturn, b)
	}
	if len(blobs) != 0 {
		elBlock, err := l.ethClients.Eth1Client.BlockByNumber(context.Background(), big.NewInt(int64(executionPayload.GetBlockNumber())))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get block at height %d, err=%s", executionPayload.GetBlockNumber(), err.Error())
		}
		blobIndex := 0
		for _, tx := range elBlock.Body().Transactions {
			if tx.Type() == ethtypes.BlobTxType {
				for _, bs := range tx.BlobHashes() {
					blobsReturn[blobIndex].TxHash = hex.EncodeToString(tx.Hash().Bytes())
					blobsReturn[blobIndex].ToAddr = tx.To().String()
					blobsReturn[blobIndex].VersionedHash = bs.String()
					blobIndex++
				}
			}
		}
	}
	return blockReturn, blobsReturn, nil
}

// reProcessBundle is used to re-upload all objects of the bundle to bundle serive and finalize it.
func (l *BlobSyncer) reProcessBundleAndFinalize(bundleName string) error {
	startHeight, endHeight, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	for i := startHeight; i < endHeight; i++ {
		sideCars, err := l.ethClients.BeaconClient.GetBlob(context.Background(), i)
		if err != nil {
			return err
		}
		err = l.uploadBlobs(i, sideCars)
		if err != nil {
			return err
		}
	}
	err = l.finalizeBundle(bundleName)
	if err != nil {
		return err
	}
	return nil
}

// reProcessBundleUntilHeight is used to make up missing blobs until the endHeight(excluded) in a bundle
func (l *BlobSyncer) reProcessBundleUntilHeight(bundleName string, endHeight uint64) error {
	startHeight, _, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	for i := startHeight; i < endHeight; i++ {
		sideCars, err := l.ethClients.BeaconClient.GetBlob(context.Background(), i)
		if err != nil {
			return err
		}
		err = l.uploadBlobs(i, sideCars)
		if err != nil {
			return err
		}
	}
	return nil
}
