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

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"

	"github.com/bnb-chain/blob-hub/config"
	"github.com/bnb-chain/blob-hub/db"
	"github.com/bnb-chain/blob-hub/external"
	"github.com/bnb-chain/blob-hub/external/cmn"
	"github.com/bnb-chain/blob-hub/external/eth"
	"github.com/bnb-chain/blob-hub/logging"
	"github.com/bnb-chain/blob-hub/metrics"
	"github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

const (
	BundleStatusFinalized      = 1
	BundleStatusCreatedOnChain = 2
	BundleStatusSealedOnChain  = 3

	LoopSleepTime = 10 * time.Millisecond
	BSCPauseTime  = 3 * time.Second

	ETHPauseTime         = 90 * time.Second
	RPCTimeout           = 20 * time.Second
	MonitorQuotaInterval = 5 * time.Minute
)

type curBundleDetail struct {
	name            string
	startBlockID    uint64
	finalizeBlockID uint64
}

type BlobSyncer struct {
	blobDao      db.BlobDao
	client       external.IClient
	bundleClient *cmn.BundleClient
	chainClient  *cmn.ChainClient
	config       *config.SyncerConfig
	bundleDetail *curBundleDetail
	spClient     *cmn.SPClient
	params       *cmn.VersionedParams
}

func NewBlobSyncer(
	blobDao db.BlobDao,
	cfg *config.SyncerConfig,
) *BlobSyncer {
	pkBz, err := hex.DecodeString(cfg.PrivateKey)
	if err != nil {
		panic(err)
	}
	bundleClient, err := cmn.NewBundleClient(cfg.BundleServiceEndpoints[0], cmn.WithPrivateKey(pkBz))
	if err != nil {
		panic(err)
	}
	chainClient, err := cmn.NewChainClient(cfg.GnfdRpcAddr)
	if err != nil {
		panic(err)
	}

	bs := &BlobSyncer{
		blobDao:      blobDao,
		bundleClient: bundleClient,
		chainClient:  chainClient,
		config:       cfg,
	}
	bs.client = external.NewClient(cfg)
	if cfg.MetricsConfig.Enable && len(cfg.MetricsConfig.SPEndpoint) > 0 {
		spClient, err := cmn.NewSPClient(cfg.MetricsConfig.SPEndpoint)
		if err != nil {
			panic(err)
		}
		bs.spClient = spClient
	}
	return bs
}

func (s *BlobSyncer) StartLoop() {
	go func() {
		// nextBlockID defines the block number (BSC) or slot(ETH)
		nextBlockID, err := s.getNextBlockNumOrSlot()
		if err != nil {
			panic(err)
		}
		err = s.LoadProgressAndResume(nextBlockID)
		if err != nil {
			panic(err)
		}
		syncTicker := time.NewTicker(LoopSleepTime)
		for range syncTicker.C {
			if err = s.sync(); err != nil {
				logging.Logger.Error(err)
				continue
			}
		}
	}()
	go func() {
		verifyTicket := time.NewTicker(LoopSleepTime)
		for range verifyTicket.C {
			if err := s.verify(); err != nil {
				logging.Logger.Error(err)
				continue
			}
		}
	}()
	go s.monitorQuota()
}

func (s *BlobSyncer) sync() error {
	var (
		blockID uint64
		err     error
		block   *structs.GetBlockV2Response
	)
	blockID, err = s.getNextBlockNumOrSlot()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
	defer cancel()

	var isForkedBlock bool
	if s.BSCChain() {
		finalizedBlockNum, err := s.client.GetFinalizedBlockNum(context.Background())
		if err != nil {
			return err
		}
		if int64(blockID) >= int64(finalizedBlockNum) {
			time.Sleep(BSCPauseTime)
			return nil
		}
	} else {
		var latestBlockResp *structs.GetBlockV2Response
		block, err = s.client.GetBeaconBlock(ctx, blockID)
		if err != nil {
			if err != eth.ErrBlockNotFound {
				return err
			}
			// Both try to get forked block and non-exist block will return 404. When the response is ErrBlockNotFound,
			// check whether nextSlot is >= latest slot, otherwise it is a forked block, should skip it.
			latestBlockResp, err = s.client.GetLatestBeaconBlock(ctx)
			if err != nil {
				logging.Logger.Errorf("failed to get latest becon block, err=%s", err.Error())
				return err
			}
			clBlock, _, err := ToBlockAndExecutionPayloadDeneb(latestBlockResp)
			if err != nil {
				logging.Logger.Errorf("failed to ToBlockAndExecutionPayloadDeneb, err=%s", err.Error())
				return err
			}
			if blockID >= uint64(clBlock.Slot) {
				logging.Logger.Debugf("the next slot %d is larger than current block slot %d\n", blockID, clBlock.Slot)
				time.Sleep(ETHPauseTime)
				return nil
			}
			isForkedBlock = true
		}
		if block != nil && !block.Finalized {
			logging.Logger.Infof("current block(slot=%d) is not finalized yet", blockID)
			time.Sleep(ETHPauseTime)
			return nil
		}
	}

	var sideCars []*types.GeneralSideCar

	if !isForkedBlock {
		ctx, cancel = context.WithTimeout(context.Background(), RPCTimeout)
		defer cancel()
		sideCars, err = s.client.GetBlob(ctx, blockID)
		if err != nil {
			return err
		}
	}

	bundleName := s.bundleDetail.name
	err = s.process(bundleName, blockID, sideCars)
	if err != nil {
		return err
	}

	if isForkedBlock {
		return s.blobDao.SaveBlockAndBlob(&db.Block{
			Slot:       blockID,
			BundleName: bundleName,
		}, nil)
	}

	blockToSave, blobToSave, err := s.toBlockAndBlobs(block, sideCars, blockID, bundleName)
	if err != nil {
		return err
	}

	err = s.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
	if err != nil {
		logging.Logger.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Slot, len(blobToSave), err.Error())
		return err
	}
	metrics.SyncedBlockIDGauge.Set(float64(blockID))
	logging.Logger.Infof("saved block(block_id=%d) and blobs(num=%d) to DB \n", blockID, len(blobToSave))
	return nil
}

func (s *BlobSyncer) process(bundleName string, blockID uint64, sidecars []*types.GeneralSideCar) error {
	var err error
	// create a new bundle in local.
	if blockID == s.bundleDetail.startBlockID {
		if err = s.createLocalBundleDir(); err != nil {
			logging.Logger.Errorf("failed to create local bundle dir, bundle=%s, err=%s", bundleName, err.Error())
			return err
		}
	}
	if err = s.writeBlobToFile(blockID, bundleName, sidecars); err != nil {
		return err
	}
	if blockID == s.bundleDetail.finalizeBlockID {
		err = s.finalizeCurBundle(bundleName)
		if err != nil {
			return err
		}
		logging.Logger.Infof("finalized bundle, bundle_name=%s, bucket_name=%s\n", bundleName, s.getBucketName())
		// init next bundle
		startBlockID := blockID + 1
		endBlockID := blockID + s.getCreateBundleInterval()
		s.bundleDetail = &curBundleDetail{
			name:            types.GetBundleName(startBlockID, endBlockID),
			startBlockID:    startBlockID,
			finalizeBlockID: endBlockID,
		}
	}
	return nil
}

func (s *BlobSyncer) getBucketName() string {
	return s.config.BucketName
}

func (s *BlobSyncer) getCreateBundleInterval() uint64 {
	return s.config.GetCreateBundleInterval()
}

func (s *BlobSyncer) getNextBlockNumOrSlot() (uint64, error) {
	latestProcessedBlock, err := s.blobDao.GetLatestProcessedBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest polled block from db, error: %s", err.Error())
	}
	latestPolledBlockSlot := latestProcessedBlock.Slot
	nextBlockID := s.config.StartSlotOrBlock
	if nextBlockID <= latestPolledBlockSlot {
		nextBlockID = latestPolledBlockSlot + 1
	}
	return nextBlockID, nil
}

// createLocalBundleDir creates an empty dir to hold blob files among a range of blocks, the blobs in this dir will be assembled into a bundle and uploaded to bundle service
func (s *BlobSyncer) createLocalBundleDir() error {
	bundleName := s.bundleDetail.name
	_, err := os.Stat(s.getBundleDir(bundleName))
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(s.getBundleDir(bundleName)), os.ModePerm)
		if err != nil {
			return err
		}
	}
	return s.blobDao.CreateBundle(
		&db.Bundle{
			Name:        s.bundleDetail.name,
			Status:      db.Finalizing,
			CreatedTime: time.Now().Unix(),
		})
}
func (s *BlobSyncer) finalizeBundle(bundleName, bundleDir, bundleFilePath string) error {
	err := s.bundleClient.UploadAndFinalizeBundle(bundleName, s.getBucketName(), bundleDir, bundleFilePath)
	if err != nil {
		if !strings.Contains(err.Error(), "Object exists") && !strings.Contains(err.Error(), "empty bundle") {
			return err
		}
	}
	err = os.RemoveAll(bundleDir)
	if err != nil {
		return err
	}
	err = os.Remove(bundleFilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.blobDao.UpdateBundleStatus(bundleName, db.Finalized)
}

func (s *BlobSyncer) finalizeCurBundle(bundleName string) error {
	return s.finalizeBundle(bundleName, s.getBundleDir(bundleName), s.getBundleFilePath(bundleName))
}

func (s *BlobSyncer) writeBlobToFile(slot uint64, bundleName string, blobs []*types.GeneralSideCar) error {
	for i, b := range blobs {
		blobName := types.GetBlobName(slot, i)
		file, err := os.Create(s.getBlobPath(bundleName, blobName))
		if err != nil {
			logging.Logger.Errorf("failed to create file, err=%s", err.Error())
			return err
		}
		defer file.Close()
		_, err = file.WriteString(b.Blob)
		if err != nil {
			logging.Logger.Errorf("failed to  write string, err=%s", err.Error())
			return err
		}
	}
	return nil
}

func (s *BlobSyncer) getBundleDir(bundleName string) string {
	return fmt.Sprintf("%s/%s/", s.config.TempDir, bundleName)
}

func (s *BlobSyncer) getBlobPath(bundleName, blobName string) string {
	return fmt.Sprintf("%s/%s/%s", s.config.TempDir, bundleName, blobName)
}

func (s *BlobSyncer) getBundleFilePath(bundleName string) string {
	return fmt.Sprintf("%s/%s.bundle", s.config.TempDir, bundleName)
}

func (s *BlobSyncer) LoadProgressAndResume(nextBlockID uint64) error {
	var (
		startBlockID uint64
		endBlockID   uint64
		err          error
	)
	finalizingBundle, err := s.blobDao.GetLatestFinalizingBundle()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		// There is no pending(finalizing) bundle, start a new bundle. e.g. a bundle includes
		// blobs from block slot 0-9 when the block interval is config to 10
		startBlockID = nextBlockID
		endBlockID = nextBlockID + s.getCreateBundleInterval() - 1
	} else {
		// resume
		startBlockID, endBlockID, err = types.ParseBundleName(finalizingBundle.Name)
		if err != nil {
			return err
		}

		// might no longer need to process the bundle even-thought it is not finalized if the user set the config to skip it.
		if nextBlockID > endBlockID {
			err = s.blobDao.UpdateBlocksStatus(startBlockID, endBlockID, db.Skipped)
			if err != nil {
				logging.Logger.Errorf("failed to update blocks status, startSlot=%d, endSlot=%d", startBlockID, endBlockID)
				return err
			}
			logging.Logger.Infof("the config slot number %d is larger than the recorded bundle end slot %d, will resume from the config slot", nextBlockID, endBlockID)
			if err = s.blobDao.UpdateBundleStatus(finalizingBundle.Name, db.Deprecated); err != nil {
				return err
			}
			startBlockID = nextBlockID
			endBlockID = nextBlockID + s.getCreateBundleInterval() - 1
		}

	}
	s.bundleDetail = &curBundleDetail{
		name:            types.GetBundleName(startBlockID, endBlockID),
		startBlockID:    startBlockID,
		finalizeBlockID: endBlockID,
	}
	return nil
}

func (s *BlobSyncer) toBlockAndBlobs(blockResp *structs.GetBlockV2Response, sidecars []*types.GeneralSideCar, blockNumOrSlot uint64, bundleName string) (*db.Block, []*db.Blob, error) {

	var blockReturn *db.Block
	blobsReturn := make([]*db.Blob, 0)

	populateBlobTxDetails := func(blockNum uint64) error {
		elBlock, err := s.client.BlockByNumber(context.Background(), big.NewInt(int64(blockNum)))
		if err != nil {
			return fmt.Errorf("failed to get block at height %d, err=%s", blockNum, err.Error())
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
		return nil
	}

	switch {
	case s.BSCChain():
		header, err := s.client.GetBlockHeader(context.Background(), blockNumOrSlot)
		if err != nil {
			return nil, nil, err
		}
		blockReturn = &db.Block{
			Root:       hex.EncodeToString(header.Root.Bytes()),
			Slot:       blockNumOrSlot,
			BlobCount:  len(sidecars),
			BundleName: bundleName,
		}
		if len(sidecars) == 0 {
			return blockReturn, blobsReturn, nil
		}
		for _, blob := range sidecars {
			index, err := strconv.Atoi(blob.Index)
			if err != nil {
				return nil, nil, err
			}
			b := &db.Blob{
				Name:          types.GetBlobName(blockNumOrSlot, index),
				Slot:          blockNumOrSlot,
				Idx:           index,
				TxIndex:       int(blob.TxIndex),
				TxHash:        blob.TxHash,
				KzgProof:      blob.KzgProof,
				KzgCommitment: blob.KzgCommitment,
			}
			blobsReturn = append(blobsReturn, b)
		}
		err = populateBlobTxDetails(blockNumOrSlot)
		if err != nil {
			return nil, nil, err
		}
		return blockReturn, blobsReturn, nil
	case s.ETHChain():
		// Process ETH beacon and execution layer block
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
			ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
			defer cancel()
			header, err := s.client.GetBeaconHeader(ctx, blockNumOrSlot)
			if err != nil {
				logging.Logger.Errorf("failed to get header, slot=%d, err=%s", blockNumOrSlot, err.Error())
				return nil, nil, err
			}

			rootBz, err := hexutil.Decode(header.Data.Root)
			if err != nil {
				logging.Logger.Errorf("failed to decode header.Data.GetObjectInfoResponse=%s, err=%s", header.Data.Root, err.Error())
				return nil, nil, err
			}
			sigBz, err := hexutil.Decode(header.Data.Header.Signature)
			if err != nil {
				logging.Logger.Errorf("failed to decode header.Data.Header.Signature=%s, err=%s", header.Data.Header.Signature, err.Error())
				return nil, nil, err
			}
			blockReturn = &db.Block{
				Root:          hex.EncodeToString(rootBz), // get rid of 0x saved to DB
				ParentRoot:    hex.EncodeToString(clBlock.GetParentRoot()),
				StateRoot:     hex.EncodeToString(clBlock.GetStateRoot()),
				BodyRoot:      hex.EncodeToString(bodyRoot[:]),
				Signature:     hex.EncodeToString(sigBz[:]),
				ProposerIndex: uint64(clBlock.ProposerIndex),
				Slot:          uint64(clBlock.GetSlot()),
				ELBlockHeight: executionPayload.GetBlockNumber(),
				BlobCount:     len(sidecars),
				BundleName:    bundleName,
			}
		default:
			return nil, nil, fmt.Errorf("un-expected block version %s", blockResp.Version)
		}
		if len(sidecars) == 0 {
			return blockReturn, blobsReturn, nil
		}
		for _, blob := range sidecars {
			index, err := strconv.Atoi(blob.Index)
			if err != nil {
				return nil, nil, err
			}
			b := &db.Blob{
				Name:                     types.GetBlobName(blockNumOrSlot, index),
				Slot:                     blockNumOrSlot,
				Idx:                      index,
				KzgProof:                 blob.KzgProof,
				KzgCommitment:            blob.KzgCommitment,
				CommitmentInclusionProof: util.JoinWithComma(blob.CommitmentInclusionProof),
			}
			blobsReturn = append(blobsReturn, b)
		}
		err = populateBlobTxDetails(executionPayload.GetBlockNumber())
		if err != nil {
			return nil, nil, err
		}
		return blockReturn, blobsReturn, nil
	}
	return blockReturn, blobsReturn, nil
}

func (s *BlobSyncer) BSCChain() bool {
	return s.config.Chain == config.BSC
}

func (s *BlobSyncer) ETHChain() bool {
	return s.config.Chain == config.ETH
}

func (s *BlobSyncer) GetParams() (*cmn.VersionedParams, error) {
	if s.params == nil {
		params, err := s.chainClient.GetParams(context.Background())
		if err != nil {
			logging.Logger.Errorf("failed to get params, err=%s", err.Error())
			return nil, err
		}
		s.params = params
	}
	return s.params, nil

}
