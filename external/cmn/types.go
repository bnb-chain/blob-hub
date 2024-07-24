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

type ObjectInfo struct {
	Checksums    []string `json:"checksums"`
	ObjectStatus string   `json:"object_status"`
}

type GetObjectInfoResponse struct {
	ObjectInfo ObjectInfo `json:"object_info"`
}

type VersionedParams struct {
	MaxSegmentSize          string `json:"max_segment_size"`           // MaxSegmentSize defines the max segment size of the Object
	RedundantDataChunkNum   int    `json:"redundant_data_chunk_num"`   // RedundantDataChunkNum defines the redundant data chunk num of the Object
	RedundantParityChunkNum int    `json:"redundant_parity_chunk_num"` // RedundantParityChunkNum defines the redundant parity chunk num of the Object
}

type Params struct {
	VersionedParams VersionedParams `json:"versioned_params"`
}

type GetParamsResponse struct {
	Params Params `json:"params"`
}
