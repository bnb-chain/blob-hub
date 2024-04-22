package syncer

import (
	"context"
	"time"

	"github.com/bnb-chain/blob-hub/logging"
	"github.com/bnb-chain/blob-hub/metrics"
)

func (s *BlobSyncer) monitorQuota() {
	if s.spClient == nil {
		return
	}
	monitorTicket := time.NewTicker(MonitorQuotaInterval)
	for range monitorTicket.C {
		quota, err := s.spClient.GetBucketReadQuota(context.Background(), s.getBucketName())
		if err != nil {
			logging.Logger.Errorf("failed to get bucket info from SP, err=%s", err.Error())
			continue
		}
		remaining := quota.ReadQuotaSize + quota.SPFreeReadQuotaSize - quota.ReadConsumedSize - quota.FreeConsumedSize
		metrics.BucketRemainingQuotaGauge.Set(float64(remaining))
		logging.Logger.Infof("remaining quota in bytes is %d", remaining)
	}
}
