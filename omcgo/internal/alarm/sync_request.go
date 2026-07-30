package alarm

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

func publishAlarmSyncRequest(
	ctx context.Context,
	eventBus event.EventBus,
	logger *zap.Logger,
	deviceSN string,
) {
	if eventBus == nil || deviceSN == "" {
		return
	}

	syncPayload := map[string]string{"device_sn": deviceSN}
	syncEvent, err := event.NewEvent(event.SubjectAlarmSyncRequested, syncPayload)
	if err != nil {
		logger.Warn("create alarm sync request", zap.Error(err))
		return
	}
	if err := eventBus.Publish(ctx, event.SubjectAlarmSyncRequested, syncEvent); err != nil {
		logger.Warn("publish alarm sync request", zap.Error(err))
	}
}
