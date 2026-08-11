package pageconfig

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

type AlarmEventHandler func(ctx context.Context, subject string, evt event.Event) error

type AlarmEventConsumer struct {
	bus      event.EventBus
	logger   *zap.Logger
	handlers []AlarmEventHandler

	mu   sync.Mutex
	subs []event.Subscription
}

func NewAlarmEventConsumer(bus event.EventBus, logger *zap.Logger, handlers ...AlarmEventHandler) *AlarmEventConsumer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AlarmEventConsumer{
		bus:      bus,
		logger:   logger.Named("page-config-alarm-consumer"),
		handlers: handlers,
	}
}

func (c *AlarmEventConsumer) Start() error {
	if c == nil || c.bus == nil || len(c.handlers) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.subs) > 0 {
		return nil
	}

	for _, subject := range alarmRealtimeSubjects() {
		subject := subject
		sub, err := c.bus.QueueSubscribe(subject, alarmRealtimeQueueForSubject(subject), func(ctx context.Context, evt event.Event) error {
			return c.dispatch(ctx, subject, evt)
		})
		if err != nil {
			for _, existing := range c.subs {
				_ = existing.Unsubscribe()
			}
			c.subs = nil
			return fmt.Errorf("subscribe northbound alarm event %s: %w", subject, err)
		}
		c.subs = append(c.subs, sub)
	}

	c.logger.Info("northbound page-config alarm event consumer started")
	return nil
}

func (c *AlarmEventConsumer) Stop() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	for _, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	c.subs = nil
	return firstErr
}

func (c *AlarmEventConsumer) dispatch(ctx context.Context, subject string, evt event.Event) error {
	for _, handler := range c.handlers {
		if handler == nil {
			continue
		}
		if err := handler(ctx, subject, evt); err != nil {
			c.logger.Warn("northbound page-config alarm handler failed",
				zap.String("subject", subject),
				zap.String("event_id", evt.ID),
				zap.Error(err))
		}
	}
	return nil
}

func alarmRealtimeSubjects() []string {
	return []string{event.SubjectAlarmRaised, event.SubjectAlarmCleared}
}

func alarmRealtimeQueueForSubject(subject string) string {
	switch subject {
	case event.SubjectAlarmRaised:
		// Keep the original durable for rolling upgrades from early page-config builds.
		return snmpForwarderQueue
	case event.SubjectAlarmCleared:
		return snmpForwarderQueue + "-cleared"
	default:
		return snmpForwarderQueue + "-event"
	}
}
