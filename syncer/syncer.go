package syncer

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"gorm.io/gorm"

	"github.com/bnb-chain/blob-syncer/config"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/types"
)

const (
	BundleStatusFinalized      = 1
	BundleStatusCreatedOnChain = 2
	BundleStatusSealedOnChain  = 3 // todo The post verification process should check if a bundle is indeed sealed onchain
)

type curBundleDetail struct {
	name         string
	startSlot    uint64
	finalizeSlot uint64
}

type BlobSyncer struct {
	blobDao      db.BlobDao
	ethClients   *external.ETHClient
	bundleClient *external.BundleClient
	config       *config.SyncerConfig
	bundleDetail *curBundleDetail
}

func NewBlobSyncer(
	blobDao db.BlobDao,
	config *config.SyncerConfig,
) *BlobSyncer {
	pkBz, err := hex.DecodeString(config.PrivateKey)
	if err != nil {
		panic(err)
	}
	bundleClient, err := external.NewBundleClient(config.BundleServiceEndpoints[0], external.WithPrivateKey(pkBz))
	if err != nil {
		panic(err)
	}
	clients := external.NewETHClient(config.ETHRPCAddrs[0], config.BeaconRPCAddrs[0])
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
	nextSlot, err := l.calNextSlot()
	if err != nil {
		return err
	}

	// the app is just re-started.
	if l.bundleDetail == nil {
		// get latest bundle from DB or bundle service(if DB data is lost)
		err := l.LoadProgressAndResume(nextSlot)
		if err != nil {
			return fmt.Errorf("failed to LoadProgressAndResume, err=%s", err.Error())
		}
	}
	var isForkedBlock bool
	block, err := l.ethClients.BeaconClient.GetBlock(ctx, nextSlot)
	if err != nil {
		if err != external.ErrBlockNotFound {
			return err
		}
		// Both try to get forked block and non-exist block will return 404. When the response is ErrBlockNotFound,
		// check whether nextSlot is >= latest slot, otherwise it is a forked block, should skip it.
		blockResp, err := l.ethClients.BeaconClient.GetLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest becon block, err=%s", err.Error())
		}
		clBlock, _, err := ToBlockAndExecutionPayloadDeneb(blockResp)
		if err != nil {
			return fmt.Errorf("failed to ToBlockAndExecutionPayloadDeneb, err=%s", err.Error())
		}
		if nextSlot >= uint64(clBlock.Slot) {
			logging.Logger.Debugf("the nextSlot %d is larger than current block slot %d\n", nextSlot, clBlock.Slot)
			return nil
		} else {
			isForkedBlock = true
		}
	}

	if !isForkedBlock && !block.Finalized {
		logging.Logger.Infof("current block(h=%d) is not finalized yet", nextSlot)
		time.Sleep(1 * time.Minute) // around 15 minutes to finalize
		return nil
	}

	var sideCars []*structs.Sidecar
	if !isForkedBlock {
		sideCars, err = l.ethClients.BeaconClient.GetBlob(ctx, nextSlot)
		if err != nil {
			return err
		}
	}

	bundleName := l.bundleDetail.name
	// create a new bundle
	if nextSlot == l.bundleDetail.startSlot {
		if err = l.createLocalBundleDir(); err != nil {
			logging.Logger.Errorf("failed to create local bundle dir, bundle=%s, err=%s", bundleName, err.Error())
			return err
		}
	}
	err = l.writeBlobToFile(nextSlot, sideCars)
	if err != nil {
		return err
	}

	if nextSlot == l.bundleDetail.finalizeSlot {
		err = l.finalizeBundle(bundleName)
		if err != nil {
			return err
		}
		logging.Logger.Infof("finalized bundle, bundle_name=%s, bucket_name=%s\n", bundleName, l.getBucketName())

		// init next bundle
		startSlot := nextSlot + 1
		endSlot := nextSlot + l.getCreateBundleSlotInterval()
		l.bundleDetail = &curBundleDetail{
			name:         types.GetBundleName(startSlot, endSlot),
			startSlot:    startSlot,
			finalizeSlot: endSlot,
		}
	}

	if isForkedBlock {
		return l.blobDao.SaveBlockAndBlob(&db.Block{
			Slot: nextSlot,
		}, nil)
	}

	blockToSave, blobToSave, err := l.ToBlockAndBlobs(block, sideCars, nextSlot, bundleName)
	if err != nil {
		return err
	}
	err = l.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
	if err != nil {
		logging.Logger.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Slot, len(blobToSave), err.Error())
		return err
	}
	logging.Logger.Infof("saved block(slot=%d) and blobs(num=%d) to DB \n", nextSlot, len(blobToSave))
	return nil
}

func (l *BlobSyncer) getBucketName() string {
	return l.config.BucketName
}

func (l *BlobSyncer) getCreateBundleSlotInterval() uint64 {
	return l.config.GetCreateBundleSlotInterval()
}

func (l *BlobSyncer) calNextSlot() (uint64, error) {
	latestProcessedBlock, err := l.blobDao.GetLatestProcessedBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest polled block from db, error: %s", err.Error())
	}
	latestPolledBlockSlot := latestProcessedBlock.Slot
	nextSlot := l.config.StartSlot
	if nextSlot <= latestPolledBlockSlot {
		nextSlot = latestPolledBlockSlot + 1
	}
	return nextSlot, nil
}

// createLocalBundleDir creates an empty dir to hold blob files among a range of blocks, the blobs in this dir will be assembled into a bundle and uploaded to bundle service
func (l *BlobSyncer) createLocalBundleDir() error {
	_, err := os.Stat(l.getBundleDir())
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(l.getBundleDir()), os.ModePerm)
		if err != nil {
			return err
		}
	}
	return l.blobDao.CreateBundle(
		&db.Bundle{
			Name:   l.bundleDetail.name,
			Status: db.Finalizing,
		})
}

func (l *BlobSyncer) finalizeBundle(bundleName string) error {
	err := l.bundleClient.UploadAndFinalizeBundle(bundleName, l.getBucketName(), l.getBundleDir(), l.getBundleFilePath())
	if err != nil {
		if !strings.Contains(err.Error(), "empty bundle") {
			return err
		}
	}
	return l.blobDao.UpdateBundleStatus(bundleName, db.Finalized)
}

func (l *BlobSyncer) writeBlobToFile(slot uint64, blobs []*structs.Sidecar) error {
	for i, b := range blobs {
		blobName := types.GetBlobName(slot, uint64(i))
		file, err := os.Create(l.getBlobPath(blobName))
		if err != nil {
			logging.Logger.Errorf("failed to create file, err=%s", err.Error())
			return err
		}
		defer file.Close()
		_, err = file.WriteString(b.Blob)
		if err != nil {
			return fmt.Errorf("failed to write string, err=%s", err.Error())
		}
	}
	return nil
}

func (l *BlobSyncer) getBundleDir() string {
	return fmt.Sprintf("%s/%s/", l.config.TempDir, l.bundleDetail.name)
}

func (l *BlobSyncer) getBlobPath(blobName string) string {
	return fmt.Sprintf("%s/%s/%s", l.config.TempDir, l.bundleDetail.name, blobName)
}

func (l *BlobSyncer) getBundleFilePath() string {
	return fmt.Sprintf("%s/%s.bundle", l.config.TempDir, l.bundleDetail.name)
}

func (l *BlobSyncer) LoadProgressAndResume(nextSlot uint64) error {
	var (
		startSlot uint64
		endSlot   uint64
		err       error
	)
	finalizingBundle, err := l.blobDao.GetLatestFinalizingBundle()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		// There is no pending(finalizing) bundle, start a new bundle. e.g. a bundle includes
		// blobs from block slot 0-9 when the block interval is config to 10
		startSlot = nextSlot
		endSlot = nextSlot + l.getCreateBundleSlotInterval() - 1
	} else {
		// resume
		startSlot, endSlot, err = types.ParseBundleName(finalizingBundle.Name)
		if err != nil {
			return err
		}
	}
	l.bundleDetail = &curBundleDetail{
		name:         types.GetBundleName(startSlot, endSlot),
		startSlot:    startSlot,
		finalizeSlot: endSlot,
	}
	return nil
}

func (l *BlobSyncer) ToBlockAndBlobs(blockResp *structs.GetBlockV2Response, blobs []*structs.Sidecar, slot uint64, bundleName string) (*db.Block, []*db.Blob, error) {
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
			logging.Logger.Errorf("failed to convert to ToBlockAndExecutionPayloadDeneb, err=%s", err.Error())
			return nil, nil, err
		}

		bodyRoot, err := clBlock.GetBody().HashTreeRoot()
		if err != nil {
			return nil, nil, err
		}
		header, err := l.ethClients.BeaconClient.GetHeader(context.Background(), slot)
		if err != nil {
			logging.Logger.Errorf("failed to get header, err=%s", header.Data.Root, err.Error())
			return nil, nil, err
		}
		rootBz, err := hexutil.Decode(header.Data.Root)
		if err != nil {
			logging.Logger.Errorf("failed to decode header.Data.Root=%s, err=%s", header.Data.Root, err.Error())
			return nil, nil, err
		}
		blockReturn = &db.Block{
			Root:          hex.EncodeToString(rootBz), // get rid of 0x saved to DB
			ParentRoot:    hex.EncodeToString(clBlock.GetParentRoot()),
			StateRoot:     hex.EncodeToString(clBlock.GetStateRoot()),
			BodyRoot:      hex.EncodeToString(bodyRoot[:]),
			ProposerIndex: uint64(clBlock.ProposerIndex),
			Slot:          uint64(clBlock.GetSlot()),
			ELBlockHeight: executionPayload.GetBlockNumber(),
			BlobCount:     len(blobs),
		}
	default:
		return nil, nil, fmt.Errorf("un-expected block version %s", blockResp.Version)
	}

	if len(blobs) == 0 {
		return blockReturn, blobsReturn, nil
	}

	for _, blob := range blobs {
		index, err := strconv.Atoi(blob.Index)
		if err != nil {
			return nil, nil, err
		}
		b := &db.Blob{
			Name:                     types.GetBlobName(slot, uint64(index)),
			Slot:                     slot,
			Idx:                      index,
			BundleName:               bundleName,
			KzgProof:                 blob.KzgProof,
			KzgCommitment:            blob.KzgCommitment,
			CommitmentInclusionProof: JoinWithComma(blob.CommitmentInclusionProof),
		}
		blobsReturn = append(blobsReturn, b)
	}

	elBlock, err := l.ethClients.Eth1Client.BlockByNumber(context.Background(), big.NewInt(int64(executionPayload.GetBlockNumber())))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block at slot %d, err=%s", executionPayload.GetBlockNumber(), err.Error())
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

	return blockReturn, blobsReturn, nil
}
