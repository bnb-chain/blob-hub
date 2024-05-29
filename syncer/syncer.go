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
	"github.com/bnb-chain/blob-hub/logging"
	"github.com/bnb-chain/blob-hub/metrics"
	"github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

const (
	BundleStatusFinalized      = 1
	BundleStatusCreatedOnChain = 2
	BundleStatusSealedOnChain  = 3

	LoopSleepTime        = 10 * time.Millisecond
	PauseTime            = 90 * time.Second
	RPCTimeout           = 20 * time.Second
	MonitorQuotaInterval = 5 * time.Minute
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
	spClient     *external.SPClient
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
	bs := &BlobSyncer{
		blobDao:      blobDao,
		ethClients:   clients,
		bundleClient: bundleClient,
		config:       config,
	}
	if config.MetricsConfig.Enable && len(config.MetricsConfig.SPEndpoint) > 0 {
		spClient, err := external.NewSPClient(config.MetricsConfig.SPEndpoint)
		if err != nil {
			panic(err)
		}
		bs.spClient = spClient
	}
	return bs
}

func (s *BlobSyncer) StartLoop() {
	go func() {
		nextSlot, err := s.calNextSlot()
		if err != nil {
			panic(err)
		}
		err = s.LoadProgressAndResume(nextSlot)
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
		nextSlot               uint64
		err                    error
		block, latestBlockResp *structs.GetBlockV2Response
	)
	nextSlot, err = s.calNextSlot()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
	defer cancel()
	var isForkedBlock bool
	block, err = s.ethClients.BeaconClient.GetBlock(ctx, nextSlot)
	if err != nil {
		if err != external.ErrBlockNotFound {
			return err
		}
		// Both try to get forked block and non-exist block will return 404. When the response is ErrBlockNotFound,
		// check whether nextSlot is >= latest slot, otherwise it is a forked block, should skip it.
		latestBlockResp, err = s.ethClients.BeaconClient.GetLatestBlock(ctx)
		if err != nil {
			logging.Logger.Errorf("failed to get latest becon block, err=%s", err.Error())
			return err
		}
		clBlock, _, err := ToBlockAndExecutionPayloadDeneb(latestBlockResp)
		if err != nil {
			logging.Logger.Errorf("failed to ToBlockAndExecutionPayloadDeneb, err=%s", err.Error())
			return err
		}
		if nextSlot >= uint64(clBlock.Slot) {
			logging.Logger.Debugf("the next slot %d is larger than current block slot %d\n", nextSlot, clBlock.Slot)
			time.Sleep(PauseTime)
			return nil
		}
		isForkedBlock = true
	}

	if block != nil && !block.Finalized {
		logging.Logger.Infof("current block(slot=%d) is not finalized yet", nextSlot)
		time.Sleep(PauseTime)
		return nil
	}

	var sideCars []*structs.Sidecar
	if !isForkedBlock {
		ctx, cancel = context.WithTimeout(context.Background(), RPCTimeout)
		defer cancel()
		sideCars, err = s.ethClients.BeaconClient.GetBlob(ctx, nextSlot)
		if err != nil {
			return err
		}
	}

	bundleName := s.bundleDetail.name
	// create a new bundle in local.
	if nextSlot == s.bundleDetail.startSlot {
		if err = s.createLocalBundleDir(); err != nil {
			logging.Logger.Errorf("failed to create local bundle dir, bundle=%s, err=%s", bundleName, err.Error())
			return err
		}
	}
	if err = s.writeBlobToFile(nextSlot, bundleName, sideCars); err != nil {
		return err
	}
	if nextSlot == s.bundleDetail.finalizeSlot {
		err = s.finalizeCurBundle(bundleName)
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
			Slot:       nextSlot,
			BundleName: bundleName,
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
	metrics.SyncedSlotGauge.Set(float64(nextSlot))
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

func (s *BlobSyncer) writeBlobToFile(slot uint64, bundleName string, blobs []*structs.Sidecar) error {
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

		// might no longer need to process the bundle even-thought it is not finalized if the user set the config to skip it.
		if nextSlot > endSlot {
			err = s.blobDao.UpdateBlocksStatus(startSlot, endSlot, db.Skipped)
			if err != nil {
				logging.Logger.Errorf("failed to update blocks status, startSlot=%d, endSlot=%d", startSlot, endSlot)
				return err
			}
			logging.Logger.Infof("the config slot number %d is larger than the recorded bundle end slot %d, will resume from the config slot", nextSlot, endSlot)
			if err = s.blobDao.UpdateBundleStatus(finalizingBundle.Name, db.Deprecated); err != nil {
				return err
			}
			startSlot = nextSlot
			endSlot = nextSlot + s.getCreateBundleSlotInterval() - 1
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
		ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
		defer cancel()
		header, err := s.ethClients.BeaconClient.GetHeader(ctx, slot)
		if err != nil {
			logging.Logger.Errorf("failed to get header, slot=%d, err=%s", slot, err.Error())
			return nil, nil, err
		}
		rootBz, err := hexutil.Decode(header.Data.Root)
		if err != nil {
			logging.Logger.Errorf("failed to decode header.Data.Root=%s, err=%s", header.Data.Root, err.Error())
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
			BlobCount:     len(blobs),
			BundleName:    bundleName,
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
			KzgProof:                 blob.KzgProof,
			KzgCommitment:            blob.KzgCommitment,
			CommitmentInclusionProof: util.JoinWithComma(blob.CommitmentInclusionProof),
		}
		blobsReturn = append(blobsReturn, b)
	}

	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
	defer cancel()
	elBlock, err := s.ethClients.Eth1Client.BlockByNumber(ctx, big.NewInt(int64(executionPayload.GetBlockNumber())))
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
	return blockReturn, blobsReturn, nil
}
