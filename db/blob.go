package db

type Blob struct {
	Id            int64
	Name          string `gorm:"NOT NULL;uniqueIndex:idx_blob_name;size:96"`
	TxHash        string `gorm:"NOT NULL;index:idx_blob_tx_hash"`
	ToAddr        string `gorm:"NOT NULL;index:idx_blob_to_address"`
	VersionedHash string // `gorm:"NOT NULL;index:idx_blob_versioned_hash"`
	Height        uint64 `gorm:"NOT NULL;index:idx_bsc_block_height_index"`
	Index         int    `gorm:"NOT NULL;index:idx_bsc_block_height_index"`
	BundleName    string `gorm:"NOT NULL"`
}

func (*Blob) TableName() string {
	return "blob"
}
