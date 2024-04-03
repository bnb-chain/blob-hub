package db

import (
	"gorm.io/gorm"
	"strings"
)

type BlobDao interface {
	BlockDB
	BlobDB
	BundleDB
	SaveBlockAndBlob(block *Block, blobs []*Blob) error
}

type BlobSvcDB struct {
	db *gorm.DB
}

func NewBlobSvcDB(db *gorm.DB) BlobDao {
	return &BlobSvcDB{
		db,
	}
}

type BlockDB interface {
	GetLatestProcessedBlock() (*Block, error)
}

func (d *BlobSvcDB) GetLatestProcessedBlock() (*Block, error) {
	block := Block{}
	err := d.db.Model(Block{}).Order("height desc").Take(&block).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &block, nil
}

type BlobDB interface {
	GetBlob(uint64) (*Blob, error)
	//UpdateBlobStatus(blobName string, status BlobStatus) error
}

func (d *BlobSvcDB) GetBlob(u uint64) (*Blob, error) {
	blob := Blob{}
	err := d.db.Model(Block{}).Order("height desc").Take(&blob).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &blob, nil
}

//func (d *BlobSvcDB) UpdateBlobStatus(blobName string, status Status) error {
//	return d.db.Transaction(func(dbTx *gorm.DB) error {
//		return dbTx.Model(Blob{}).Where("name = ?", blobName).Updates(
//			Blob{Status: status}).Error
//	})
//}

type BundleDB interface {
	GetLatestFinalizingBundle() (*Bundle, error)
	CreateBundle(*Bundle) error
	UpdateBundleStatus(bundleName string, status InnerBundleStatus) error
}

func (d *BlobSvcDB) GetLatestFinalizingBundle() (*Bundle, error) {
	bundle := Bundle{}
	err := d.db.Model(Bundle{}).Where("status = ?", Finalizing).Order("id desc").Take(&bundle).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (d *BlobSvcDB) CreateBundle(b *Bundle) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		err := dbTx.Create(b).Error
		if err != nil && strings.Contains(err.Error(), " Duplicate entry") {
			return nil
		}
		return err
	})
}

func (d *BlobSvcDB) UpdateBundleStatus(bundleName string, status InnerBundleStatus) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		return dbTx.Model(Bundle{}).Where("name = ?", bundleName).Updates(
			Bundle{Status: status}).Error
	})
}

func (d *BlobSvcDB) SaveBlockAndBlob(block *Block, blobs []*Blob) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		err := dbTx.Create(block).Error
		if err != nil {
			return err
		}
		if len(blobs) != 0 {
			err := dbTx.Create(blobs).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func InitTables(db *gorm.DB) {
	var err error
	if err = db.AutoMigrate(&Bundle{}); err != nil {
		panic(err)
	}
	if err = db.AutoMigrate(&Block{}); err != nil {
		panic(err)
	}
	if err = db.AutoMigrate(&Blob{}); err != nil {
		panic(err)
	}
}
