package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
)

const LifecycleConsumerDurable = "notification-lifecycle-v1"

var (
	ErrLifecycleConsumerNotSubscribed = errors.New("notification lifecycle consumer is not subscribed")
	ErrLifecycleConsumerNotCaughtUp   = errors.New("notification lifecycle consumer durable is not caught up")
)

type notificationLifecycleDurableBus interface {
	KeyedQueueSubscribe(string, event.KeyedQueueConfig, event.EventKeyFunc, event.EventHandler) (event.Subscription, error)
	QueueStats(context.Context, string, string) (event.QueueStats, error)
}

// LifecycleConsumer owns only the durable Inbox projection. Rule matching and
// delivery creation remain separate so notification failures cannot affect the
// alarm transaction.
type LifecycleConsumer struct {
	repository    LifecycleRepository
	bus           event.EventBus
	startSequence uint64
	orchestrator  OccurrenceOrchestrator

	mu           sync.Mutex
	subscription event.Subscription
}

func (c *LifecycleConsumer) SetOrchestrator(orchestrator OccurrenceOrchestrator) {
	c.orchestrator = orchestrator
}

func NewLifecycleConsumer(repository LifecycleRepository, bus event.EventBus, startSequence uint64) *LifecycleConsumer {
	return &LifecycleConsumer{repository: repository, bus: bus, startSequence: startSequence}
}

func (c *LifecycleConsumer) Subscribe() error {
	if c == nil || c.repository == nil || c.bus == nil {
		return fmt.Errorf("subscribe notification lifecycle consumer: dependencies are required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscription != nil {
		return nil
	}
	durableBus, ok := c.bus.(notificationLifecycleDurableBus)
	if !ok {
		return fmt.Errorf("subscribe notification lifecycle consumer: event bus does not support fixed durable consumers")
	}
	subscription, err := durableBus.KeyedQueueSubscribe(
		event.SubjectDomainAlarmLifecycleAll,
		event.KeyedQueueConfig{
			Durable: LifecycleConsumerDurable, StartSequence: c.startSequence,
			Concurrency: 8, QueueDepth: 64, MaxAckPending: 512,
		},
		func(envelope event.Event) (string, error) {
			payload, err := decodeNotificationLifecycle(envelope)
			if err != nil {
				return "", err
			}
			return payload.OccurrenceID.String(), nil
		},
		c.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe notification lifecycle durable: %w", err)
	}
	c.subscription = subscription
	return nil
}

func (c *LifecycleConsumer) Handle(ctx context.Context, envelope event.Event) error {
	if c == nil || c.repository == nil {
		return fmt.Errorf("apply notification lifecycle: repository is required")
	}
	payload, err := decodeNotificationLifecycle(envelope)
	if err != nil {
		return err
	}
	if _, err := c.repository.ApplyLifecycle(ctx, payload); err != nil {
		return fmt.Errorf("apply notification lifecycle: %w", err)
	}
	if c.orchestrator != nil {
		if err := c.orchestrator.ProcessOccurrence(ctx, payload.OccurrenceID); err != nil {
			return fmt.Errorf("orchestrate notification lifecycle: %w", err)
		}
	}
	return nil
}

func decodeNotificationLifecycle(envelope event.Event) (event.AlarmLifecyclePayload, error) {
	var payload event.AlarmLifecyclePayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return payload, fmt.Errorf("decode notification lifecycle payload: %w", err)
	}
	if payload.SchemaVersion != event.AlarmLifecycleSchemaVersion {
		return payload, fmt.Errorf("validate notification lifecycle: unsupported schema_version=%d", payload.SchemaVersion)
	}
	expectedSubject, err := payload.LifecycleType.Subject()
	if err != nil {
		return payload, fmt.Errorf("validate notification lifecycle: %w", err)
	}
	if envelope.Subject != expectedSubject {
		return payload, fmt.Errorf("validate notification lifecycle: subject=%q lifecycle=%q", envelope.Subject, payload.LifecycleType)
	}
	if envelope.ID != payload.EventID.String() {
		return payload, fmt.Errorf("validate notification lifecycle: event identity mismatch")
	}
	if payload.OccurrenceID == uuid.Nil || payload.Snapshot.AlarmID != payload.OccurrenceID {
		return payload, fmt.Errorf("validate notification lifecycle: occurrence identity is invalid")
	}
	if payload.AlarmVersion < 1 {
		return payload, fmt.Errorf("validate notification lifecycle: alarm version must be positive")
	}
	if payload.Snapshot.DeviceID == uuid.Nil {
		return payload, fmt.Errorf("validate notification lifecycle: managed-element device ID is required")
	}
	return payload, nil
}

func (c *LifecycleConsumer) Ready(ctx context.Context) error {
	if c == nil {
		return ErrLifecycleConsumerNotSubscribed
	}
	c.mu.Lock()
	subscribed := c.subscription != nil
	c.mu.Unlock()
	if !subscribed {
		return ErrLifecycleConsumerNotSubscribed
	}
	durableBus, ok := c.bus.(notificationLifecycleDurableBus)
	if !ok {
		return fmt.Errorf("read notification lifecycle durable health: unsupported event bus")
	}
	stats, err := durableBus.QueueStats(ctx, event.SubjectDomainAlarmLifecycleAll, LifecycleConsumerDurable)
	if err != nil {
		return fmt.Errorf("read notification lifecycle durable health: %w", err)
	}
	if stats.Pending > 0 || stats.AckPending > 0 {
		return fmt.Errorf("%w: pending=%d ack_pending=%d", ErrLifecycleConsumerNotCaughtUp, stats.Pending, stats.AckPending)
	}
	return nil
}

func (c *LifecycleConsumer) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	subscription := c.subscription
	c.subscription = nil
	c.mu.Unlock()
	if subscription == nil {
		return nil
	}
	if err := subscription.Unsubscribe(); err != nil {
		return fmt.Errorf("unsubscribe notification lifecycle consumer: %w", err)
	}
	return nil
}
