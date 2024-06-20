package cmn

import (
	"encoding/xml"
)

// QuotaInfo indicates the quota info of bucket
type QuotaInfo struct {
	XMLName                 xml.Name `xml:"GetReadQuotaResult"`
	Version                 string   `xml:"version,attr"`
	BucketName              string   `xml:"BucketName"`
	BucketID                string   `xml:"BucketID"`                 // BucketID defines the bucket read quota value on chain
	ReadQuotaSize           uint64   `xml:"ReadQuotaSize"`            // ReadQuotaSize defines the bucket read quota value on chain
	SPFreeReadQuotaSize     uint64   `xml:"SPFreeReadQuotaSize"`      // SPFreeReadQuotaSize defines the free quota of this month
	ReadConsumedSize        uint64   `xml:"ReadConsumedSize"`         // ReadConsumedSize defines the consumed total read quota of this month
	FreeConsumedSize        uint64   `xml:"FreeConsumedSize"`         // FreeConsumedSize defines the consumed free quota
	MonthlyFreeQuota        uint64   `xml:"MonthlyFreeQuota"`         // MonthlyFreeQuota defines the consumed monthly free quota
	MonthlyFreeConsumedSize uint64   `xml:"MonthlyQuotaConsumedSize"` // MonthlyFreeConsumedSize defines the consumed monthly free quota
}
