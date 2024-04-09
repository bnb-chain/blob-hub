package db

type Status int

const (
	Processed Status = 0
	Verified  Status = 1 // each block's blobs will be verified by the post-verification process
)

type Block struct {
	Id            int64
	Root          string `gorm:"NOT NULL;index:idx_block_root;size:64"`
	ParentRoot    string
	StateRoot     string
	BodyRoot      string
	ProposerIndex uint64
	Slot          uint64 `gorm:"NOT NULL;uniqueIndex:idx_block_slot"`
	ELBlockHeight uint64 // the eth1 block height
	BlobCount     int
	KzgCommitment string

	Status Status
}

func (*Block) TableName() string {
	return "block"
}
