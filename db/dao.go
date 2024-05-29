package db

import (
	"gorm.io/gorm"
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
	GetBlock(slot uint64) (*Block, error)
	GetBlockByRoot(root string) (*Block, error)
	GetLatestProcessedBlock() (*Block, error)
	GetEarliestUnverifiedBlock() (*Block, error)
	UpdateBlockStatus(slot uint64, status Status) error
	UpdateBlocksStatus(startSlot, endSlot uint64, status Status) error
}

func (d *BlobSvcDB) GetBlock(slot uint64) (*Block, error) {
	block := Block{}
	err := d.db.Model(Block{}).Where("slot = ?", slot).Take(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (d *BlobSvcDB) GetBlockByRoot(root string) (*Block, error) {
	block := Block{}
	err := d.db.Model(Block{}).Where("root = ?", root).Take(&block).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &block, nil
}

func (d *BlobSvcDB) GetLatestProcessedBlock() (*Block, error) {
	block := Block{}
	err := d.db.Model(Block{}).Order("slot desc").Take(&block).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &block, nil
}

func (d *BlobSvcDB) GetEarliestUnverifiedBlock() (*Block, error) {
	block := Block{}
	err := d.db.Model(Block{}).Where("status = ?", Processed).Order("slot asc").Take(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (d *BlobSvcDB) UpdateBlockStatus(slot uint64, status Status) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		return dbTx.Model(Block{}).Where("slot = ?", slot).Updates(
			Block{Status: status}).Error
	})
}

func (d *BlobSvcDB) UpdateBlocksStatus(startSlot, endSlot uint64, status Status) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		return dbTx.Model(Block{}).Where("slot >= ? and slot <= ?", startSlot, endSlot).Updates(
			Block{Status: status}).Error
	})
}

type BlobDB interface {
	GetBlobBySlot(slot uint64) ([]*Blob, error)
	GetBlobBySlotAndIndices(slot uint64, indices []int64) ([]*Blob, error)
	GetBlobBetweenSlots(startSlot, endSlot uint64) ([]*Blob, error)
}

func (d *BlobSvcDB) GetBlobBySlot(slot uint64) ([]*Blob, error) {
	blobs := make([]*Blob, 0)
	if err := d.db.Where("slot = ?", slot).Order("idx asc").Find(&blobs).Error; err != nil {
		return blobs, err
	}
	return blobs, nil
}

func (d *BlobSvcDB) GetBlobBySlotAndIndices(slot uint64, indices []int64) ([]*Blob, error) {
	blobs := make([]*Blob, 0)
	if err := d.db.Where("slot = ? and idx in (?)", slot, indices).Order("idx asc").Find(&blobs).Error; err != nil {
		return blobs, err
	}
	return blobs, nil
}

func (d *BlobSvcDB) GetBlobBetweenSlots(startSlot, endSlot uint64) ([]*Blob, error) {
	blobs := make([]*Blob, 0)
	if err := d.db.Where("slot >= ? and slot <= ?", startSlot, endSlot).Order("idx asc").Find(&blobs).Error; err != nil {
		return blobs, err
	}
	return blobs, nil
}

type BundleDB interface {
	GetBundle(name string) (*Bundle, error)
	GetLatestFinalizingBundle() (*Bundle, error)
	CreateBundle(*Bundle) error
	UpdateBundleStatus(bundleName string, status InnerBundleStatus) error
}

func (d *BlobSvcDB) GetBundle(name string) (*Bundle, error) {
	bundle := Bundle{}
	err := d.db.Model(Bundle{}).Where("name = ?", name).Take(&bundle).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (d *BlobSvcDB) GetLatestFinalizingBundle() (*Bundle, error) {
	bundle := Bundle{}
	err := d.db.Model(Bundle{}).Where("status = ? and calibrated = false", Finalizing).Order("id desc").Take(&bundle).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (d *BlobSvcDB) CreateBundle(b *Bundle) error {
	return d.db.Transaction(func(dbTx *gorm.DB) error {
		err := dbTx.Create(b).Error
		if err != nil && MysqlErrCode(err) == ErrDuplicateEntryCode {
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
		err := dbTx.Save(block).Error
		if err != nil && MysqlErrCode(err) != ErrDuplicateEntryCode {
			return err
		}
		if len(blobs) != 0 {
			err = dbTx.Save(blobs).Error
			if err != nil && MysqlErrCode(err) != ErrDuplicateEntryCode {
				return err
			}
		}
		return nil
	})
}

func AutoMigrateDB(db *gorm.DB) {
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
