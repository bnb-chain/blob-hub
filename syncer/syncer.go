package syncer

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/bnb-chain/blob-syncer/util"
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

func (s *BlobSyncer) StartLoop() {
	go func() {
		for {
			if err := s.process(); err != nil {
				logging.Logger.Error(err)
				continue
			}
		}
	}()

	go func() {
		for {
			if err := s.verify(); err != nil {
				logging.Logger.Error(err)
				continue
			}
		}
	}()
}

func (s *BlobSyncer) process() error {
	ctx := context.Background()
	nextSlot, err := s.calNextSlot()
	if err != nil {
		return err
	}

	// the app is just re-started.
	if s.bundleDetail == nil {
		// get latest bundle from DB or bundle service(if DB data is lost)
		err := s.LoadProgressAndResume(nextSlot)
		if err != nil {
			return fmt.Errorf("failed to LoadProgressAndResume, err=%s", err.Error())
		}
	}
	var isForkedBlock bool
	block, err := s.ethClients.BeaconClient.GetBlock(ctx, nextSlot)
	if err != nil {
		if err != external.ErrBlockNotFound {
			return err
		}
		// Both try to get forked block and non-exist block will return 404. When the response is ErrBlockNotFound,
		// check whether nextSlot is >= latest slot, otherwise it is a forked block, should skip it.
		blockResp, err := s.ethClients.BeaconClient.GetLatestBlock(ctx)
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
		sideCars, err = s.ethClients.BeaconClient.GetBlob(ctx, nextSlot)
		if err != nil {
			return err
		}
	}

	bundleName := s.bundleDetail.name
	// create a new bundle
	if nextSlot == s.bundleDetail.startSlot {
		if err = s.createLocalBundleDir(); err != nil {
			logging.Logger.Errorf("failed to create local bundle dir, bundle=%s, err=%s", bundleName, err.Error())
			return err
		}
	}
	err = s.writeBlobToFile(nextSlot, sideCars)
	if err != nil {
		return err
	}

	if nextSlot == s.bundleDetail.finalizeSlot {
		err = s.finalizeBundle(bundleName)
		if err != nil {
			return err
		}
		logging.Logger.Infof("finalized bundle, bundle_name=%s, bucket_name=%s\n", bundleName, s.getBucketName())

		// init next bundle
		startSlot := nextSlot + 1
		endSlot := nextSlot + s.getCreateBundleSlotInterval()
		s.bundleDetail = &curBundleDetail{
			name:         types.GetBundleName(startSlot, endSlot),
			startSlot:    startSlot,
			finalizeSlot: endSlot,
		}
	}

	if isForkedBlock {
		return s.blobDao.SaveBlockAndBlob(&db.Block{
			Slot: nextSlot,
		}, nil)
	}

	blockToSave, blobToSave, err := s.ToBlockAndBlobs(block, sideCars, nextSlot, bundleName)
	if err != nil {
		return err
	}
	err = s.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
	if err != nil {
		logging.Logger.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Slot, len(blobToSave), err.Error())
		return err
	}
	logging.Logger.Infof("saved block(slot=%d) and blobs(num=%d) to DB \n", nextSlot, len(blobToSave))
	return nil
}

func (s *BlobSyncer) getBucketName() string {
	return s.config.BucketName
}

func (s *BlobSyncer) getCreateBundleSlotInterval() uint64 {
	return s.config.GetCreateBundleSlotInterval()
}

func (s *BlobSyncer) calNextSlot() (uint64, error) {
	latestProcessedBlock, err := s.blobDao.GetLatestProcessedBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest polled block from db, error: %s", err.Error())
	}
	latestPolledBlockSlot := latestProcessedBlock.Slot
	nextSlot := s.config.StartSlot
	if nextSlot <= latestPolledBlockSlot {
		nextSlot = latestPolledBlockSlot + 1
	}
	return nextSlot, nil
}

// createLocalBundleDir creates an empty dir to hold blob files among a range of blocks, the blobs in this dir will be assembled into a bundle and uploaded to bundle service
func (s *BlobSyncer) createLocalBundleDir() error {
	_, err := os.Stat(s.getBundleDir())
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(s.getBundleDir()), os.ModePerm)
		if err != nil {
			return err
		}
	}
	return s.blobDao.CreateBundle(
		&db.Bundle{
			Name:   s.bundleDetail.name,
			Status: db.Finalizing,
		})
}

func (s *BlobSyncer) finalizeBundle(bundleName string) error {
	err := s.bundleClient.UploadAndFinalizeBundle(bundleName, s.getBucketName(), s.getBundleDir(), s.getBundleFilePath())
	if err != nil {
		if !strings.Contains(err.Error(), "Object exists") && !strings.Contains(err.Error(), "empty bundle") {
			return err
		}
	}
	err = os.RemoveAll(s.getBundleDir())
	if err != nil {
		return err
	}
	err = os.Remove(s.getBundleFilePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.blobDao.UpdateBundleStatus(bundleName, db.Finalized)
}

func (s *BlobSyncer) writeBlobToFile(slot uint64, blobs []*structs.Sidecar) error {
	for i, b := range blobs {
		blobName := types.GetBlobName(slot, i)
		file, err := os.Create(s.getBlobPath(blobName))
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

func (s *BlobSyncer) getBundleDir() string {
	return fmt.Sprintf("%s/%s/", s.config.TempDir, s.bundleDetail.name)
}

func (s *BlobSyncer) getBlobPath(blobName string) string {
	return fmt.Sprintf("%s/%s/%s", s.config.TempDir, s.bundleDetail.name, blobName)
}

func (s *BlobSyncer) getBundleFilePath() string {
	return fmt.Sprintf("%s/%s.bundle", s.config.TempDir, s.bundleDetail.name)
}

func (s *BlobSyncer) LoadProgressAndResume(nextSlot uint64) error {
	var (
		startSlot uint64
		endSlot   uint64
		err       error
	)
	finalizingBundle, err := s.blobDao.GetLatestFinalizingBundle()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		// There is no pending(finalizing) bundle, start a new bundle. e.g. a bundle includes
		// blobs from block slot 0-9 when the block interval is config to 10
		startSlot = nextSlot
		endSlot = nextSlot + s.getCreateBundleSlotInterval() - 1
	} else {
		// resume
		startSlot, endSlot, err = types.ParseBundleName(finalizingBundle.Name)
		if err != nil {
			return err
		}
	}
	s.bundleDetail = &curBundleDetail{
		name:         types.GetBundleName(startSlot, endSlot),
		startSlot:    startSlot,
		finalizeSlot: endSlot,
	}
	return nil
}

func (s *BlobSyncer) ToBlockAndBlobs(blockResp *structs.GetBlockV2Response, blobs []*structs.Sidecar, slot uint64, bundleName string) (*db.Block, []*db.Blob, error) {
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
		header, err := s.ethClients.BeaconClient.GetHeader(context.Background(), slot)
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
			Name:                     types.GetBlobName(slot, index),
			Slot:                     slot,
			Idx:                      index,
			BundleName:               bundleName,
			KzgProof:                 blob.KzgProof,
			KzgCommitment:            blob.KzgCommitment,
			CommitmentInclusionProof: util.JoinWithComma(blob.CommitmentInclusionProof),
		}
		blobsReturn = append(blobsReturn, b)
	}

	elBlock, err := s.ethClients.Eth1Client.BlockByNumber(context.Background(), big.NewInt(int64(executionPayload.GetBlockNumber())))
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
