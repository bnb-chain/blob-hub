package syncer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"

	"github.com/bnb-chain/blob-hub/db"
	"github.com/bnb-chain/blob-hub/external/cmn"
	"github.com/bnb-chain/blob-hub/external/eth"
	"github.com/bnb-chain/blob-hub/logging"
	"github.com/bnb-chain/blob-hub/metrics"
	"github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

const VerifyPauseTime = 90 * time.Second

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
	verifyBlock, err := s.blobDao.GetEarliestUnverifiedBlock()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logging.Logger.Debugf("found no unverified block in DB")
			time.Sleep(VerifyPauseTime)
			return nil
		}
		return err
	}
	bundleName := verifyBlock.BundleName
	bundleStartBlockID, bundleEndBlockID, err := types.ParseBundleName(bundleName)
	if err != nil {
		return err
	}
	verifyBlockID := verifyBlock.Slot

	// check if the bundle has been submitted to bundle service
	bundle, err := s.blobDao.GetBundle(bundleName)
	if err != nil {
		return err
	}
	if bundle.Status == db.Finalizing {
		logging.Logger.Debugf("the bundle has not been submitted to bundle service yet, bundleName=%s", bundleName)
		time.Sleep(VerifyPauseTime)
		return nil
	}

	// validate the bundle info at the start slot of a bundle
	if verifyBlockID == bundleStartBlockID {
		// the bundle is recorded finalized in DB, validate the bundle is sealed onchain
		bundleInfo, err := s.bundleClient.GetBundleInfo(s.getBucketName(), bundleName)
		if err != nil {
			if err != cmn.ErrorBundleNotExist {
				logging.Logger.Errorf("failed to get bundle info, bundleName=%s", bundleName)
				return err
			}

			// verify if there are no blobs within the range
			blobs, err := s.blobDao.GetBlobBetweenBlocks(bundleStartBlockID, bundleEndBlockID)
			if err != nil {
				return err
			}
			if len(blobs) != 0 {
				return fmt.Errorf("%d blobs within block_id[%d, %d] not found in bundle service", len(blobs), bundleStartBlockID, bundleEndBlockID)
			}
			if err = s.blobDao.UpdateBlocksStatus(bundleStartBlockID, bundleEndBlockID, db.Verified); err != nil {
				return err
			}
			if err = s.blobDao.UpdateBundleStatus(bundleName, db.Sealed); err != nil {
				return err
			}
			return nil
		}
		// the bundle is not sealed yet
		if bundleInfo.Status == BundleStatusFinalized || bundleInfo.Status == BundleStatusCreatedOnChain {
			if bundle.CreatedTime > 0 && time.Now().Unix()-bundle.CreatedTime > s.config.GetReUploadBundleThresh() {
				logging.Logger.Infof("the bundle %s is not sealed and exceed the re-upload threshold %d ", bundleName, s.config.GetReUploadBundleThresh())
				return s.reUploadBundle(bundleName)
			}
			return nil
		}
	}

	if verifyBlock.BlobCount == 0 {
		if err = s.blobDao.UpdateBlockStatus(verifyBlockID, db.Verified); err != nil {
			logging.Logger.Errorf("failed to update block status, block_id=%d err=%s", verifyBlockID, err.Error())
			return err
		}
		if verifyBlockID == bundleEndBlockID {
			logging.Logger.Debugf("update bundle status to sealed, name=%s , block_id %d ", bundleName, verifyBlockID)
			if err = s.blobDao.UpdateBundleStatus(bundleName, db.Sealed); err != nil {
				logging.Logger.Errorf("failed to update bundle status to sealed, name=%s , block_id %d ", bundleName, verifyBlockID)
				return err
			}
		}
		return nil
	}

	// get blob from beacon chain or BSC again
	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
	defer cancel()
	sideCars, err := s.client.GetBlob(ctx, verifyBlockID)
	if err != nil {
		logging.Logger.Errorf("failed to get blob at block_id=%d, err=%s", verifyBlockID, err.Error())
		return err
	}

	// get blob meta from DB
	blobMetas, err := s.blobDao.GetBlobByBlockID(verifyBlockID)
	if err != nil {
		return err
	}

	if len(blobMetas) != len(sideCars) {
		logging.Logger.Errorf("found blob number mismatch at block_id=%d, bundleName=%s, expected=%d, actual=%d", verifyBlockID, bundleName, len(sideCars), len(blobMetas))
		return s.reUploadBundle(bundleName)
	}

	err = s.verifyBlob(verifyBlockID, sideCars, blobMetas, bundleName)
	if err != nil {
		if err == ErrVerificationFailed {
			return s.reUploadBundle(bundleName)
		}
		return err
	}
	if err = s.blobDao.UpdateBlockStatus(verifyBlockID, db.Verified); err != nil {
		logging.Logger.Errorf("failed to update block status to verified, block_id=%d err=%s", verifyBlockID, err.Error())
		return err
	}
	metrics.VerifiedBlockIDGauge.Set(float64(verifyBlockID))
	if bundleEndBlockID == verifyBlockID {
		logging.Logger.Debugf("update bundle status to sealed, name=%s , block_id=%d ", bundleName, verifyBlockID)
		if err = s.blobDao.UpdateBundleStatus(bundleName, db.Sealed); err != nil {
			logging.Logger.Errorf("failed to update bundle status to sealed, name=%s, block_id %d ", bundleName, verifyBlockID)
			return err
		}
	}
	logging.Logger.Infof("successfully verify at block block_id=%d ", verifyBlockID)
	return nil
}

func (s *BlobSyncer) verifyBlob(blockID uint64, sidecars []*types.GeneralSideCar, blobMetas []*db.Blob, bundleName string) error {
	for i := 0; i < len(sidecars); i++ {
		// get blob from bundle service
		blobFromBundle, err := s.bundleClient.GetObject(s.getBucketName(), bundleName, types.GetBlobName(blockID, i))
		if err != nil {
			if err == cmn.ErrorBundleObjectNotExist {
				logging.Logger.Errorf("the bundle object not found in bundle service, object=%s", types.GetBlobName(blockID, i))
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
	startBlockID, endBlockID, err := types.ParseBundleName(bundleName)
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
	for bi := startBlockID; bi <= endBlockID; bi++ {
		ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout)
		defer cancel()
		sideCars, err := s.client.GetBlob(ctx, bi)
		if err != nil {
			return err
		}
		if err = s.writeBlobToFile(bi, newBundleName, sideCars); err != nil {
			return err
		}

		// not needed by BSC
		var block *structs.GetBlockV2Response
		if s.ETHChain() {
			block, err = s.client.GetBeaconBlock(ctx, bi)
			if err != nil {
				if err == eth.ErrBlockNotFound {
					continue
				}
				return err
			}
		}

		blockMeta, err := s.blobDao.GetBlock(bi)
		if err != nil {
			return err
		}
		blobMetas, err := s.blobDao.GetBlobByBlockID(bi)
		if err != nil {
			return err
		}
		blockToSave, blobToSave, err := s.toBlockAndBlobs(block, sideCars, bi, newBundleName)
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
		logging.Logger.Infof("save calibrated block(block_id=%d) and blobs(num=%d) to DB \n", bi, len(blobToSave))
	}
	if err = s.finalizeBundle(newBundleName, s.getBundleDir(newBundleName), s.getBundleFilePath(newBundleName)); err != nil {
		logging.Logger.Errorf("failed to finalized bundle, name=%s, err=%s", newBundleName, err.Error())
		return err
	}
	return nil
}
