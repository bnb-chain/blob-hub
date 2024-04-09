package syncer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bnb-chain/blob-syncer/logging"
	"github.com/bnb-chain/blob-syncer/types"
	"github.com/bnb-chain/blob-syncer/util"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"time"
)

const bundleNotSealedAlertThresh = int64(1 * time.Hour)

// Verify is used to verify the blob uploaded to bundle service is indeed in Greenfield, the intergirty
func (s *BlobSyncer) Verify() error {

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
		err = s.blobDao.UpdateBlockToVerifiedStatus(verifyBlockSlot)
		if err != nil {
			logging.Logger.Errorf("failed to update block status, slot=%d err=%s", verifyBlockSlot, err.Error())
			return err
		}
		return nil
	}

	// get blob from beacon chain again
	var sideCars []*structs.Sidecar
	sideCars, err = s.ethClients.BeaconClient.GetBlob(context.Background(), verifyBlockSlot)
	if err != nil {
		return err
	}

	if verifyBlock.BlobCount != len(sideCars) {
		// todo recreate bundle
		err = fmt.Errorf("found blob count mismatch")
		return err
	}

	// get blob meta from DB
	blobMetas, err := s.blobDao.GetBlobBySlot(verifyBlockSlot)
	if err != nil {
		return err
	}
	bundleName := blobMetas[0].BundleName

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

	for i := 0; i < verifyBlock.BlobCount; i++ {
		// get blob from bundle servic
		blobFromBundle, err := s.bundleClient.GetObject(s.getBucketName(), bundleName, types.GetBlobName(verifyBlockSlot, i))
		if err != nil {
			return err
		}

		expectedIdx, err := util.StringToInt64(sideCars[i].Index)
		if err != nil {
			return err
		}

		if int64(blobMetas[i].Idx) != expectedIdx {
			err = fmt.Errorf("index mismatch")
			return err
		}

		expectedKzgProofHash, err := util.GenerateHash(sideCars[i].KzgProof)
		if err != nil {
			return err
		}
		actualKzgProofHash, err := util.GenerateHash(blobMetas[i].KzgProof)
		if !bytes.Equal(actualKzgProofHash, expectedKzgProofHash) {
			err = fmt.Errorf("kzg proof mismatch")
			return err
		}
		actualBlobHash, err := util.GenerateHash(blobFromBundle)
		if err != nil {
			return err
		}
		expectedBlobHash, err := util.GenerateHash(sideCars[i].Blob)
		if err != nil {
			return err
		}
		if !bytes.Equal(actualBlobHash, expectedBlobHash) {
			err = fmt.Errorf("blob mismatch")
			return err
		}
	}
	err = s.blobDao.UpdateBlockToVerifiedStatus(verifyBlockSlot)
	if err != nil {
		logging.Logger.Errorf("failed to update block status, slot=%d err=%s", verifyBlockSlot, err.Error())
		return err
	}
	logging.Logger.Errorf("successfully verify at block slot %d ", verifyBlockSlot)

	return nil
}
