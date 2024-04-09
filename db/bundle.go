package db

type InnerBundleStatus int

const (
	Finalizing InnerBundleStatus = 0
	Finalized  InnerBundleStatus = 1 // when a bundle is uploaded to bundle service, its status will be Finalized
	Sealed     InnerBundleStatus = 2 // todo The post verification process should check if a bundle is indeed sealed onchain
)

type Bundle struct {
	Id     int64
	Name   string            `gorm:"NOT NULL;uniqueIndex:idx_bundle_name;size:96"`
	Status InnerBundleStatus `gorm:"NOT NULL"`
}

func (*Bundle) TableName() string {
	return "bundle"
}
