package service

import (
	"fmt"
	"strconv"

	"github.com/bnb-chain/blob-syncer/cache"
	"github.com/bnb-chain/blob-syncer/config"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/models"
	"github.com/bnb-chain/blob-syncer/util"
)

type Blob interface {
	GetBlobSidecarsByRoot(root string, indices []int64) ([]*models.Sidecar, error)
	GetBlobSidecarsBySlot(slot uint64, indices []int64) ([]*models.Sidecar, error)
}

type BlobService struct {
	blobDB       db.BlobDao
	bundleClient *external.BundleClient
	cacheService cache.Cache
	config       *config.ServerConfig
}

func NewBlobService(blobDB db.BlobDao, bundleClient *external.BundleClient, cache cache.Cache, config *config.ServerConfig) Blob {
	return &BlobService{
		blobDB:       blobDB,
		bundleClient: bundleClient,
		cacheService: cache,
		config:       config,
	}
}

func (b BlobService) GetBlobSidecarsBySlot(slot uint64, indices []int64) ([]*models.Sidecar, error) {
	var err error
	blobs, found := b.cacheService.Get(strconv.FormatUint(slot, 10))
	if found {
		blobsFound := blobs.([]*models.Sidecar)
		if len(indices) != 0 {
			blobReturn := make([]*models.Sidecar, 0)
			for _, idx := range indices {
				if int(idx) >= len(blobsFound) {
					return nil, fmt.Errorf("index %d out of bound, only %d blob at slot %d", idx, len(blobsFound), slot)
				}
				blobReturn = append(blobReturn, blobsFound[idx])
			}
			return blobReturn, nil
		}
		return blobsFound, nil
	}

	block, err := b.blobDB.GetBlock(slot)
	if err != nil {
		return nil, err
	}

	var blobMetas []*db.Blob
	if len(indices) == 0 {
		blobMetas, err = b.blobDB.GetBlobBySlot(slot)
		if err != nil {
			return nil, err
		}
	} else {
		blobMetas, err = b.blobDB.GetBlobBySlotAndIndices(slot, indices)
		if err != nil {
			return nil, err
		}
	}

	sideCars := make([]*models.Sidecar, 0)
	for _, meta := range blobMetas {
		bundleObject, err := b.bundleClient.GetObject(b.config.BucketName, meta.BundleName, meta.Name)
		if err != nil {
			return nil, err
		}
		header := &models.SidecarSignedBlockHeader{
			Message: &models.SidecarSignedBlockHeaderMessage{
				BodyRoot:      block.BodyRoot,
				ParentRoot:    block.ParentRoot,
				ProposerIndex: strconv.FormatUint(block.ProposerIndex, 10),
				Slot:          strconv.FormatUint(block.Slot, 10),
				StateRoot:     block.StateRoot,
			},
		}
		sideCars = append(sideCars,
			&models.Sidecar{
				Blob:                        bundleObject,
				Index:                       strconv.FormatInt(int64(meta.Idx), 10),
				KzgCommitmentInclusionProof: util.SplitByComma(meta.CommitmentInclusionProof),
				KzgCommitment:               meta.KzgCommitment,
				KzgProof:                    meta.KzgProof,
				SignedBlockHeader:           header,
			})
	}

	// cache all blobs at a specified slot
	if len(indices) == 0 {
		b.cacheService.Set(strconv.FormatUint(slot, 10), sideCars)
	}
	return sideCars, nil
}

func (b BlobService) GetBlobSidecarsByRoot(root string, indices []int64) ([]*models.Sidecar, error) {
	block, err := b.blobDB.GetBlockByRoot(root)
	if err != nil {
		return nil, err
	}
	return b.GetBlobSidecarsBySlot(block.Slot, indices)
}
