package db

type Blob struct {
	Id                       int64
	Name                     string `gorm:"NOT NULL;uniqueIndex:idx_blob_name;size:96"` // the identifier of blob object in bundle service
	TxHash                   string `gorm:"NOT NULL;index:idx_blob_tx_hash"`
	ToAddr                   string `gorm:"NOT NULL;index:idx_blob_to_address"`
	VersionedHash            string `gorm:"NOT NULL"`
	Slot                     uint64 `gorm:"NOT NULL;index:idx_blob_slot_index"`
	Idx                      int    `gorm:"NOT NULL;index:idx_blob_slot_idx"`
	TxIndex                  int
	KzgCommitment            string `gorm:"NOT NULL"`
	KzgProof                 string `gorm:"NOT NULL"`
	CommitmentInclusionProof string `gorm:"NOT NULL"`
}

func (*Blob) TableName() string {
	return "blob"
}
