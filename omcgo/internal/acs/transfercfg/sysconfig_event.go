package transfercfg

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// HandleSysConfigSavedEvent returns an EventBus handler that invalidates the
// transfer config cache only when acs_transfer is updated.
func HandleSysConfigSavedEvent(policy interface{ InvalidateCache() }, logger *zap.Logger) event.EventHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx context.Context, evt event.Event) error {
		_ = ctx
		if policy == nil {
			return nil
		}

		var payload event.SysConfigSavedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			logger.Warn("decode sys config saved event", zap.Error(err))
			return nil
		}
		if payload.Category != Category {
			return nil
		}

		policy.InvalidateCache()
		logger.Info("invalidated transfer config cache from sys config event",
			zap.String("category", payload.Category))
		return nil
	}
}
