package dashboard

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/events"
)

type SSENotifier struct {
	hub    *events.MessageHub
	logger *zap.Logger
}

func NewSSENotifier(hub *events.MessageHub, logger *zap.Logger) *SSENotifier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SSENotifier{hub: hub, logger: logger.Named("dashboard-sse-notifier")}
}

func (n *SSENotifier) Subscribe(bus event.EventBus) (func(), error) {
	if bus == nil || n.hub == nil {
		return func() {}, nil
	}
	subs := make([]event.Subscription, 0, 3)
	for _, subject := range []string{
		event.SubjectAlarmRaised,
		event.SubjectAlarmCleared,
		event.SubjectPMFileParsed,
	} {
		sub, err := bus.Subscribe(subject, n.handle)
		if err != nil {
			for _, s := range subs {
				_ = s.Unsubscribe()
			}
			return nil, err
		}
		subs = append(subs, sub)
	}
	n.logger.Info("dashboard SSE notifier subscribed")
	return func() {
		for _, s := range subs {
			_ = s.Unsubscribe()
		}
	}, nil
}

func (n *SSENotifier) handle(_ context.Context, evt event.Event) error {
	// Emit a unified 'dashboard_update' SSE event so frontend can just refetch queries
	
	// We extract what triggered it just so the frontend knows (e.g., 'alarm' vs 'pm')
	var updateType string
	if evt.Subject == event.SubjectPMFileParsed {
		updateType = "pm"
	} else {
		updateType = "alarm"
	}
	
	data, err := json.Marshal(map[string]any{
		"type":    updateType,
		"subject": evt.Subject,
	})
	if err != nil {
		return nil
	}

	msg := &events.SSEMessage{
		ID:    uuid.New().String(),
		Event: "dashboard_update",
		Data:  data,
	}

	if pubErr := n.hub.PublishGlobal(msg); pubErr != nil {
		n.logger.Warn("dashboard SSE publish failed", zap.String("subject", evt.Subject), zap.Error(pubErr))
	}
	return nil
}
