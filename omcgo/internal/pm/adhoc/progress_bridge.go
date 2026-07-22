package adhoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/realtime"
)

type RealtimeSubscriber interface {
	Subscribe(subject string, handler func([]byte)) (realtime.Subscription, error)
}

// ProgressBridge owns one upstream subscription per realtime subject for an
// APP instance and routes decoded task events into its local ProgressHub.
type ProgressBridge struct {
	realtime RealtimeSubscriber
	hub      *ProgressHub
	logger   *zap.Logger
}

func NewProgressBridge(realtimeBus RealtimeSubscriber, hub *ProgressHub, logger *zap.Logger) *ProgressBridge {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ProgressBridge{realtime: realtimeBus, hub: hub, logger: logger.Named("pm.adhoc.progress-bridge")}
}

func (b *ProgressBridge) Subscribe() (func() error, error) {
	if b.realtime == nil || b.hub == nil {
		return nil, errors.New("realtime progress bridge dependencies not wired")
	}
	progress, err := b.realtime.Subscribe(SubjectRealtimeProgress, b.handler("progress"))
	if err != nil {
		return nil, fmt.Errorf("subscribe progress events: %w", err)
	}
	completed, err := b.realtime.Subscribe(SubjectRealtimeCompleted, b.handler("completed"))
	if err != nil {
		subscribeErr := fmt.Errorf("subscribe completed events: %w", err)
		rollbackErr := wrapRealtimeUnsubscribe(
			"rollback progress subscription",
			SubjectRealtimeProgress,
			progress.Unsubscribe(),
		)
		b.hub.Close()
		return nil, errors.Join(subscribeErr, rollbackErr)
	}

	var once sync.Once
	var stopErr error
	return func() error {
		once.Do(func() {
			stopErr = errors.Join(
				wrapRealtimeUnsubscribe("stop progress subscription", SubjectRealtimeProgress, progress.Unsubscribe()),
				wrapRealtimeUnsubscribe("stop completed subscription", SubjectRealtimeCompleted, completed.Unsubscribe()),
			)
			b.hub.Close()
		})
		return stopErr
	}, nil
}

func wrapRealtimeUnsubscribe(operation, subject string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s for realtime subject %s: %w", operation, subject, err)
}

func (b *ProgressBridge) handler(eventName string) func([]byte) {
	return func(data []byte) {
		var envelope struct {
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			b.logger.Warn("ignore invalid realtime progress payload", zap.String("event", eventName), zap.Error(err))
			return
		}
		if envelope.TaskID == "" {
			b.logger.Warn("ignore realtime progress payload without task_id", zap.String("event", eventName))
			return
		}
		b.hub.Publish(envelope.TaskID, ProgressEvent{Name: eventName, Data: data})
	}
}
