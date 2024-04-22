package syncer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"

	"github.com/bnb-chain/blob-hub/db"
	"github.com/bnb-chain/blob-hub/external"
	"github.com/bnb-chain/blob-hub/logging"
	"github.com/bnb-chain/blob-hub/metrics"
	"github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

var (
	ErrVerificationFailed = errors.New("verification failed")
)

// Verify is used to verify the blob uploaded to bundle service is indeed in Greenfield, and the integrity.
// In the cases:
//  1. a recorded finalized bundle lost in bundle service
//  2. SP can't seal the object (probably won't seal it anymore)
//  3. verification on a specified blob failed
//
// a new bundle should be re-uploaded.
func (s *BlobSyncer) verify() error {
	var err error
	verifyBlock, err := s.blobDao.GetEarliestUnverifiedBlock()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logging.Logger.Debugf("found no unverified block in DB")
			time.Sleep(PauseTime)
			return nil
		}
		return err
	}
	bundleName := verifyBlock.BundleName
	bundleStartSlot, bundleEndSlot, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	verifyBlockSlot := verifyBlock.Slot

	// check if the bundle has been submitted to bundle service
	bundle, err := s.blobDao.GetBundle(bundleName)
	if err != nil {
		return err
	}
	if bundle.Status == db.Finalizing {
		logging.Logger.Debugf("the bundle has not been submitted to bundle service yet, bundleName=%s", bundleName)
		time.Sleep(PauseTime)
		return nil
	}

	// validate the bundle info at the start slot of a bundle
	if verifyBlockSlot == bundleStartSlot {
		// the bundle is recorded finalized in DB, validate the bundle is sealed onchain
		bundleInfo, err := s.bundleClient.GetBundleInfo(s.getBucketName(), bundleName)
		if err != nil {
			logging.Logger.Errorf("failed to get bundle info, bundleName=%s", bundleName)
			return err
		}
		// the bundle is not sealed yet
		if bundleInfo.Status == BundleStatusFinalized || bundleInfo.Status == BundleStatusCreatedOnChain {
			if bundle.CreatedTime > 0 && time.Now().Unix()-bundle.CreatedTime > s.config.GetReUploadBundleThresh() {
				return s.reUploadBundle(bundleName)
			}
			return nil
		}
	}

	if verifyBlock.BlobCount == 0 {
		if err = s.blobDao.UpdateBlockStatus(verifyBlockSlot, db.Verified); err != nil {
			logging.Logger.Errorf("failed to update block status, slot=%d err=%s", verifyBlockSlot, err.Error())
			return err
		}
		if verifyBlockSlot == bundleEndSlot {
			logging.Logger.Debugf("update bundle status to sealed, name=%s , slot %d ", bundleName, verifyBlockSlot)
			if err = s.blobDao.UpdateBundleStatus(bundleName, db.Sealed); err != nil {
				logging.Logger.Errorf("failed to update bundle status to sealed, name=%s , slot %d ", bundleName, verifyBlockSlot)
				return err
			}
		}
		return nil
	}

	// get blob from beacon chain again
	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
	defer cancel()
	sideCars, err := s.ethClients.BeaconClient.GetBlob(ctx, verifyBlockSlot)
	if err != nil {
		logging.Logger.Errorf("failed to get blob at slot=%d, err=%s", verifyBlockSlot, err.Error())
		return err
	}

	// get blob meta from DB
	blobMetas, err := s.blobDao.GetBlobBySlot(verifyBlockSlot)
	if err != nil {
		return err
	}

	if len(blobMetas) != len(sideCars) {
		logging.Logger.Errorf("found blob number mismatch at slot=%d, bundleName=%s", verifyBlockSlot, bundleName)
		return s.reUploadBundle(bundleName)
	}

	err = s.verifyBlobAtSlot(verifyBlockSlot, sideCars, blobMetas, bundleName)
	if err != nil {
		if err == ErrVerificationFailed {
			return s.reUploadBundle(bundleName)
		}
		return err
	}
	if err = s.blobDao.UpdateBlockStatus(verifyBlockSlot, db.Verified); err != nil {
		logging.Logger.Errorf("failed to update block status to verified, slot=%d err=%s", verifyBlockSlot, err.Error())
		return err
	}
	metrics.VerifiedSlotGauge.Set(float64(verifyBlockSlot))
	if bundleEndSlot == verifyBlockSlot {
		logging.Logger.Debugf("update bundle status to sealed, name=%s , slot=%d ", bundleName, verifyBlockSlot)
		if err = s.blobDao.UpdateBundleStatus(bundleName, db.Sealed); err != nil {
			logging.Logger.Errorf("failed to update bundle status to sealed, name=%s, slot %d ", bundleName, verifyBlockSlot)
			return err
		}
	}
	logging.Logger.Infof("successfully verify at block slot %d ", verifyBlockSlot)
	return nil
}

func (s *BlobSyncer) verifyBlobAtSlot(slot uint64, sidecars []*structs.Sidecar, blobMetas []*db.Blob, bundleName string) error {
	for i := 0; i < len(sidecars); i++ {
		// get blob from bundle service
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
	if err := s.blobDao.UpdateBundleStatus(bundleName, db.Deprecated); err != nil {
		return err
	}

	newBundleName := bundleName + "_calibrated_" + util.Int64ToString(time.Now().Unix())
	startSlot, endSlot, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	logging.Logger.Infof("creating new calibrated bundle %s", newBundleName)

	_, err = os.Stat(s.getBundleDir(newBundleName))
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(s.getBundleDir(newBundleName)), os.ModePerm)
		if err != nil {
			return err
		}
	}
	if err = s.blobDao.CreateBundle(&db.Bundle{
		Name:        newBundleName,
		Status:      db.Finalizing,
		Calibrated:  true,
		CreatedTime: time.Now().Unix(),
	}); err != nil {
		return err
	}
	for slot := startSlot; slot <= endSlot; slot++ {
		ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
		defer cancel()
		sideCars, err := s.ethClients.BeaconClient.GetBlob(ctx, slot)
		if err != nil {
			return err
		}
		if err = s.writeBlobToFile(slot, newBundleName, sideCars); err != nil {
			return err
		}
		block, err := s.ethClients.BeaconClient.GetBlock(ctx, slot)
		if err != nil {
			if err == external.ErrBlockNotFound {
				continue
			}
			return err
		}
		blockMeta, err := s.blobDao.GetBlock(slot)
		if err != nil {
			return err
		}
		blobMetas, err := s.blobDao.GetBlobBySlot(slot)
		if err != nil {
			return err
		}
		blockToSave, blobToSave, err := s.ToBlockAndBlobs(block, sideCars, slot, newBundleName)
		if err != nil {
			return err
		}
		blockToSave.Id = blockMeta.Id
		for i, preBlob := range blobMetas {
			if i < len(blobToSave) {
				blobToSave[i].Id = preBlob.Id
			}
		}
		err = s.blobDao.SaveBlockAndBlob(blockToSave, blobToSave)
		if err != nil {
			logging.Logger.Errorf("failed to save block(h=%d) and Blob(count=%d), err=%s", blockToSave.Slot, len(blobToSave), err.Error())
			return err
		}
		logging.Logger.Infof("save calibrated block(slot=%d) and blobs(num=%d) to DB \n", slot, len(blobToSave))
	}
	if err = s.finalizeBundle(newBundleName, s.getBundleDir(newBundleName), s.getBundleFilePath(newBundleName)); err != nil {
		logging.Logger.Errorf("failed to finalized bundle, name=%s, err=%s", newBundleName, err.Error())
		return err
	}
	return nil
}
