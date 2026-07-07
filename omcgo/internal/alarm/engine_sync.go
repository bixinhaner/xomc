package alarm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// syncDeviceSeverityStatsAsync queries the real-time statistics for a device and updates
// the highest_severity and highest_count in Redis asynchronously.
// This bridges the 5-minute gap of the reconciler.
func (e *AlarmEngine) syncDeviceSeverityStatsAsync(deviceID uuid.UUID) {
	if e.redisStore == nil || deviceID.String() == "00000000-0000-0000-0000-000000000000" {
		return
	}

	go func(id uuid.UUID) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idPtr := id
		activeStatus := model.AlarmActive
		stats, err := e.store.Statistics(ctx, AlarmFilter{
			DeviceID: &idPtr,
			Status:   &activeStatus,
		})
		if err != nil {
			e.logger.Error("failed to get async alarm statistics for severity sync", zap.Error(err))
			return
		}

		highestSev := 0
		highestCount := 0

		for sev, count := range stats.BySeverity {
			sevInt := int(sev)
			if sevInt > 0 {
				if highestSev == 0 || sevInt < highestSev {
					highestSev = sevInt
					highestCount = int(count)
				} else if sevInt == highestSev {
					highestCount += int(count)
				}
			}
		}

		key := alarmStatsKey(id.String())
		pipe := e.redisStore.client.Pipeline()
		if highestSev > 0 {
			pipe.HSet(ctx, key, "highest_severity", highestSev)
			pipe.HSet(ctx, key, "highest_count", highestCount)
		} else {
			pipe.HDel(ctx, key, "highest_severity", "highest_count")
		}
		_, err = pipe.Exec(ctx)
		if err != nil {
			e.logger.Warn("failed to save async severity stats to redis", zap.Error(err))
		}
	}(deviceID)
}
