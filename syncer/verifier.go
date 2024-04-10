package syncer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"gorm.io/gorm"

	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/types"
	"github.com/bnb-chain/blob-syncer/util"
)

var (
	ErrVerificationFailed = errors.New("verification failed")
	ErrBundleNotSealed    = errors.New("bundle not sealed yet")
)

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
		if err == gorm.ErrRecordNotFound {
			return nil
		}
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
		if err == external.ErrorBundleNotExist || err == ErrBundleNotSealed {
			return nil
		}
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
		return ErrBundleNotSealed
	}

	for i := 0; i < len(sidecars); i++ {
		// get blob from bundle servic
		blobFromBundle, err := s.bundleClient.GetObject(s.getBucketName(), bundleName, types.GetBlobName(slot, i))
		if err != nil {
			if err == external.ErrorBundleObjectNotExist {
				logging.Logger.Errorf("the bundle object not found in bundle service, object=%s", types.GetBlobName(slot, i))
				return ErrVerificationFailed
			}
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
	newBundleName := bundleName + "_calibrated_" + util.Int64ToString(time.Now().Unix())
	startSlot, endSlot, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	_, err = os.Stat(s.getBundleDir(newBundleName))
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(s.getBundleDir(newBundleName)), os.ModePerm)
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

	if err := s.finalizeBundle(newBundleName, s.getBundleDir(newBundleName), s.getBundleFilePath(newBundleName)); err != nil {
		return err
	}
	return nil
}
