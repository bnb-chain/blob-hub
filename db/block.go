package db

type Status int

const (
	Processed Status = 0
	Verified  Status = 1 // each block's blobs will be verified by the post-verification process
)

type Block struct {
	Id            int64
	BlockHash     string `gorm:"NOT NULL"`
	ParentHash    string `gorm:"NOT NULL"`
	Height        uint64 `gorm:"NOT NULL;uniqueIndex:idx_block_height"`
	ELBlockHeight uint64
	BlobCount     int
	Status        Status
}

func (*Block) TableName() string {
	return "block"
}
