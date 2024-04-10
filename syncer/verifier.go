package syncer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/types"
	"github.com/bnb-chain/blob-syncer/util"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrVerificationFailed = errors.New("verification failed")
)

const bundleNotSealedAlertThresh = int64(1 * time.Hour)

// Verify is used to verify the blob uploaded to bundle service is indeed in Greenfield, and integrity.
func (s *BlobSyncer) verify() error {

	var err error
	latestVerifiedBlock, err := s.blobDao.GetLatestVerifiedBlock()
	if err != nil {
		logging.Logger.Errorf("failed to get latest verified block from DB, err=%s", err.Error())
		return err
	}
	var verifyBlockSlot uint64

	if latestVerifiedBlock.Slot == 0 {
		firstBlock, err := s.blobDao.GetFirstBlock()
		if err != nil {
			logging.Logger.Errorf("failed to get latest verified block from DB, err=%s", err.Error())
			return err
		}
		verifyBlockSlot = firstBlock.Slot
	} else {
		verifyBlockSlot = latestVerifiedBlock.Slot + 1
	}

	verifyBlock, err := s.blobDao.GetBlock(verifyBlockSlot)
	if err != nil {
		logging.Logger.Errorf("failed to get block from DB, slot=%d err=%s", verifyBlockSlot, err.Error())
		return err
	}

	if verifyBlock.BlobCount == 0 {
		if err = s.blobDao.UpdateBlockToVerifiedStatus(verifyBlockSlot); err != nil {
			logging.Logger.Errorf("failed to update block status, slot=%d err=%s", verifyBlockSlot, err.Error())
			return err
		}
		return nil
	}

	// get blob from beacon chain again
	sideCars, err := s.ethClients.BeaconClient.GetBlob(context.Background(), verifyBlockSlot)
	if err != nil {
		logging.Logger.Errorf("failed to get blob at slot=%d, err=%s", verifyBlockSlot, err.Error())
		return err
	}

	// get blob meta from DB
	blobMetas, err := s.blobDao.GetBlobBySlot(verifyBlockSlot)
	if err != nil {
		return err
	}
	bundleName := blobMetas[0].BundleName

	if len(blobMetas) != len(sideCars) {
		logging.Logger.Errorf("found blob number mismatch at slot=%d, bundleName=%s", verifyBlockSlot, bundleName)
		return s.reUploadBundle(bundleName)
	}

	err = s.verifyBlobAtSlot(verifyBlockSlot, sideCars, blobMetas, bundleName)
	if err != nil {
		logging.Logger.Errorf("failed to verifyBlobAtSlot, slot=%d err=%s", verifyBlockSlot, err.Error())
		if err == ErrVerificationFailed {
			return s.reUploadBundle(bundleName)
		}
		return err
	}
	err = s.blobDao.UpdateBlockToVerifiedStatus(verifyBlockSlot)
	if err != nil {
		logging.Logger.Errorf("failed to update block status, slot=%d err=%s", verifyBlockSlot, err.Error())
		return err
	}
	logging.Logger.Infof("successfully verify at block slot %d ", verifyBlockSlot)
	return nil
}

func (s *BlobSyncer) verifyBlobAtSlot(slot uint64, sidecars []*structs.Sidecar, blobMetas []*db.Blob, bundleName string) error {
	// validate the bundle is sealed
	bundleInfo, err := s.bundleClient.GetBundleInfo(s.getBucketName(), bundleName)
	if err != nil {
		return err
	}
	if bundleInfo.Status == BundleStatusFinalized || bundleInfo.Status == BundleStatusCreatedOnChain {
		if time.Now().Unix()-bundleInfo.CreatedTimestamp > bundleNotSealedAlertThresh {
			// todo alarm
			return fmt.Errorf("the bundle is supposed to be sealed")
		}
		return nil
	} else if bundleInfo.Status != BundleStatusSealedOnChain {
		return fmt.Errorf("unexpect status, should not happen")
	}

	for i := 0; i < len(sidecars); i++ {
		// get blob from bundle servic
		blobFromBundle, err := s.bundleClient.GetObject(s.getBucketName(), bundleName, types.GetBlobName(slot, i))
		if err != nil {
			return err
		}

		expectedIdx, err := util.StringToInt64(sidecars[i].Index)
		if err != nil {
			return err
		}

		if int64(blobMetas[i].Idx) != expectedIdx {
			logging.Logger.Errorf("found index mismatch")
			return ErrVerificationFailed
		}

		expectedKzgProofHash, err := util.GenerateHash(sidecars[i].KzgProof)
		if err != nil {
			return err
		}
		actualKzgProofHash, err := util.GenerateHash(blobMetas[i].KzgProof)
		if err != nil {
			return err
		}
		if !bytes.Equal(actualKzgProofHash, expectedKzgProofHash) {
			logging.Logger.Errorf("found kzg proof mismatch")
			return ErrVerificationFailed
		}

		actualBlobHash, err := util.GenerateHash(blobFromBundle)
		if err != nil {
			return err
		}
		expectedBlobHash, err := util.GenerateHash(sidecars[i].Blob)
		if err != nil {
			return err
		}
		if !bytes.Equal(actualBlobHash, expectedBlobHash) {
			logging.Logger.Errorf("found blob mismatch")
			return ErrVerificationFailed
		}
	}
	return nil
}

func (s *BlobSyncer) reUploadBundle(bundleName string) error {
	newBundleName := bundleName + "-calibrated-" + util.Int64ToString(time.Now().Unix())
	startSlot, endSlot, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	calibratedBundleDir := fmt.Sprintf("%s/%s/", s.config.TempDir, newBundleName)
	_, err = os.Stat(calibratedBundleDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(calibratedBundleDir), os.ModePerm)
		if err != nil {
			return err
		}
	}
	if err = s.blobDao.CreateBundle(&db.Bundle{
		Name:   newBundleName,
		Status: db.Finalizing,
	}); err != nil {
		return err
	}
	for slot := startSlot; slot < endSlot; slot++ {
		sideCars, err := s.ethClients.BeaconClient.GetBlob(context.Background(), slot)
		if err != nil {
			return err
		}
		if err = s.writeBlobToFile(slot, newBundleName, sideCars); err != nil {
			return err
		}
		block, err := s.ethClients.BeaconClient.GetBlock(context.Background(), slot)
		if err != nil {
			return err
		}
		blockToSave, blobToSave, err := s.ToBlockAndBlobs(block, sideCars, slot, newBundleName)
		if err != nil {
			return err
		}
		err = s.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
		if err != nil {
			logging.Logger.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Slot, len(blobToSave), err.Error())
			return err
		}
		logging.Logger.Infof("save calibrated block(slot=%d) and blobs(num=%d) to DB \n", slot, len(blobToSave))
	}

	bundleFilePath := fmt.Sprintf("%s/%s.bundle", s.config.TempDir, bundleName)
	if err := s.finalizeBundle(bundleName, calibratedBundleDir, bundleFilePath); err != nil {
		return err
	}
	return nil
}
