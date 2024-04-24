package db

type InnerBundleStatus int

const (
	Finalizing InnerBundleStatus = 0
	Finalized  InnerBundleStatus = 1 // when a bundle is uploaded to bundle service, its status will be Finalized
	Sealed     InnerBundleStatus = 2 //
	Deprecated InnerBundleStatus = 3
)

type Bundle struct {
	Id          int64
	Name        string            `gorm:"NOT NULL;uniqueIndex:idx_bundle_name;size:64"`
	Status      InnerBundleStatus `gorm:"NOT NULL"`
	Calibrated  bool
	CreatedTime int64 `gorm:"NOT NULL;comment:created_time"`
}

func (*Bundle) TableName() string {
	return "bundle"
}
