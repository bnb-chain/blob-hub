package entity

type Blob struct {
	TxHash     string
	Height     uint64
	Index      int
	BundleName string
}

type Block struct {
	BlockHash     string
	ParentHash    string
	Height        uint64
	ELBlockHeight uint64
	BlockTime     int64
	BlobCount     int
}

type Bundle struct {
	Name string
}
