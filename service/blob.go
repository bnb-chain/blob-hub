package service

import (
	"github.com/bnb-chain/blob-syncer/cache"
	"github.com/bnb-chain/blob-syncer/config"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/external"
	"github.com/bnb-chain/blob-syncer/models"
	"strconv"
)

type Blob interface {
	GetBlobSidecars(blockNum int64) ([]*models.Sidecar, error)
}

type BlobService struct {
	blobDB       db.BlobDao
	bundleClient *external.BundleClient
	cacheService cache.Cache
	config       *config.Config
}

func NewBlobService(blobDB db.BlobDao, bundleClient *external.BundleClient, cache cache.Cache, config *config.Config) Blob {
	return &BlobService{
		blobDB:       blobDB,
		bundleClient: bundleClient,
		cacheService: cache,
		config:       config,
	}
}

func (b BlobService) GetBlobSidecars(blockNum int64) ([]*models.Sidecar, error) {
	blobs, found := b.cacheService.Get(strconv.FormatInt(blockNum, 10))
	if found {
		return blobs.([]*models.Sidecar), nil
	}
	blobsMeta, err := b.blobDB.GetBlobs(blockNum)
	if err != nil {
		return nil, err
	}
	sideCars := make([]*models.Sidecar, 0)
	for _, meta := range blobsMeta {
		object, err := b.bundleClient.GetObject(b.config.SyncerConfig.BucketName, meta.BundleName, meta.Name)
		if err != nil {
			return nil, err
		}
		sideCars = append(sideCars,
			&models.Sidecar{
				Blob:  object,
				Index: int64(meta.Index),
			})
	}
	b.cacheService.Set(strconv.FormatInt(blockNum, 10), sideCars)
	return sideCars, nil
}
