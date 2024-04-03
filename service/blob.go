package service

import (
	"github.com/bnb-chain/blob-syncer/config"
	"github.com/bnb-chain/blob-syncer/db"
	"github.com/bnb-chain/blob-syncer/models"
	"github.com/bnb-chain/blob-syncer/syncer"
	lru "github.com/hashicorp/golang-lru"
)

const cacheBlock = 4096

type Blob interface {
	GetBlobSidecars(blockNum int64) ([]*models.Sidecar, error)
}

type BlobService struct {
	blobDB       db.BlobDao
	bundleClient *syncer.BundleClient
	lru          *lru.Cache
	config       *config.Config
}

func NewBlobService(blobDB db.BlobDao, bundleClient *syncer.BundleClient, config *config.Config) Blob {
	//bundleClient, err := syncer.NewBundleClient(config.SyncerConfig.BundleServiceAddrs[0], time.Second*3, config.SyncerConfig.PrivateKey)
	//if err != nil {
	//	panic(err)
	//}
	cache, _ := lru.New(cacheBlock)

	return &BlobService{
		blobDB:       blobDB,
		bundleClient: bundleClient,
		lru:          cache,
		config:       config,
	}
}

func (b BlobService) GetBlobSidecars(blockNum int64) ([]*models.Sidecar, error) {
	blobs, found := b.lru.Get(blockNum)
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
	b.lru.Add(blockNum, sideCars)
	return sideCars, nil
}
