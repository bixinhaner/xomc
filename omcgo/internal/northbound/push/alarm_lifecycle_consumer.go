package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
)

// AlarmLifecycleConsumerDurable is the fixed northbound projection consumer.
const AlarmLifecycleConsumerDurable = "northbound-alarm-lifecycle-v1"

var (
	ErrAlarmLifecycleConsumerNotSubscribed = errors.New("northbound alarm lifecycle consumer is not subscribed")
	ErrAlarmLifecycleConsumerNotCaughtUp   = errors.New("northbound alarm lifecycle consumer durable is not caught up")
)

type alarmLifecycleEnqueuer interface {
	EnqueueEvent(context.Context, event.Event) error
}

type alarmLifecycleDurableBus interface {
	KeyedQueueSubscribe(
		string,
		event.KeyedQueueConfig,
		event.EventKeyFunc,
		event.EventHandler,
	) (event.Subscription, error)
	QueueStats(context.Context, string, string) (event.QueueStats, error)
}

// AlarmLifecycleConsumer projects committed managed-element alarm facts into
// the existing northbound delivery Outbox before JetStream acknowledges them.
type AlarmLifecycleConsumer struct {
	enqueuer      alarmLifecycleEnqueuer
	bus           event.EventBus
	startSequence uint64
	shadow        bool

	mu           sync.Mutex
	subscription event.Subscription
}

// SetShadow keeps the durable and validation path active without creating
// externally deliverable northbound rows during migration comparison.
func (c *AlarmLifecycleConsumer) SetShadow(shadow bool) { c.shadow = shadow }

// Subscribe binds the fixed durable to canonical managed-element alarm facts.
// Legacy alarm.* system-health events are outside this filter by construction.
func (c *AlarmLifecycleConsumer) Subscribe() error {
	if c == nil || c.enqueuer == nil || c.bus == nil {
		return fmt.Errorf("subscribe alarm lifecycle northbound consumer: dependencies are required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscription != nil {
		return nil
	}
	durableBus, ok := c.bus.(alarmLifecycleDurableBus)
	if !ok {
		return fmt.Errorf("subscribe alarm lifecycle northbound consumer: event bus does not support fixed durable consumers")
	}
	subscription, err := durableBus.KeyedQueueSubscribe(
		event.SubjectDomainAlarmLifecycleAll,
		event.KeyedQueueConfig{
			Durable:       AlarmLifecycleConsumerDurable,
			StartSequence: c.startSequence,
			Concurrency:   8,
			QueueDepth:    64,
			MaxAckPending: 512,
		},
		func(envelope event.Event) (string, error) {
			var payload event.AlarmLifecyclePayload
			if err := envelope.DecodePayload(&payload); err != nil {
				return "", fmt.Errorf("decode alarm lifecycle northbound key: %w", err)
			}
			if payload.OccurrenceID.String() == "00000000-0000-0000-0000-000000000000" {
				return "", fmt.Errorf("decode alarm lifecycle northbound key: occurrence ID is required")
			}
			return payload.OccurrenceID.String(), nil
		},
		c.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe alarm lifecycle northbound durable: %w", err)
	}
	c.subscription = subscription
	return nil
}

func NewAlarmLifecycleConsumer(
	enqueuer alarmLifecycleEnqueuer,
	bus event.EventBus,
	startSequence uint64,
) *AlarmLifecycleConsumer {
	return &AlarmLifecycleConsumer{
		enqueuer: enqueuer, bus: bus, startSequence: startSequence,
	}
}

// Handle synchronously persists one canonical lifecycle fact to the
// northbound Outbox. Returning an error leaves the JetStream message unacked.
func (c *AlarmLifecycleConsumer) Handle(ctx context.Context, envelope event.Event) error {
	if c == nil || c.enqueuer == nil {
		return fmt.Errorf("enqueue alarm lifecycle for northbound: dependency is required")
	}

	var payload event.AlarmLifecyclePayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode alarm lifecycle for northbound: %w", err)
	}
	if payload.SchemaVersion != event.AlarmLifecycleSchemaVersion {
		return fmt.Errorf("decode alarm lifecycle for northbound: unsupported schema_version=%d", payload.SchemaVersion)
	}
	expectedSubject, err := payload.LifecycleType.Subject()
	if err != nil {
		return fmt.Errorf("validate alarm lifecycle for northbound: %w", err)
	}
	if envelope.Subject != expectedSubject {
		return fmt.Errorf(
			"validate alarm lifecycle for northbound: subject=%q lifecycle=%q",
			envelope.Subject,
			payload.LifecycleType,
		)
	}
	if envelope.ID != payload.EventID.String() {
		return fmt.Errorf("validate alarm lifecycle for northbound: event identity mismatch")
	}
	if payload.OccurrenceID == uuid.Nil || payload.Snapshot.AlarmID != payload.OccurrenceID {
		return fmt.Errorf("validate alarm lifecycle for northbound: occurrence identity is invalid")
	}
	if payload.AlarmVersion < 1 {
		return fmt.Errorf("validate alarm lifecycle for northbound: alarm version must be positive")
	}
	if payload.Snapshot.DeviceID == uuid.Nil {
		return fmt.Errorf("validate alarm lifecycle for northbound: managed-element device ID is required")
	}
	if c.shadow {
		return nil
	}

	forwardedPayload, err := json.Marshal(payload.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal alarm lifecycle snapshot for northbound: %w", err)
	}
	forwarded := event.Event{
		ID:        payload.EventID.String(),
		Subject:   event.SubjectOSSAlarmForward,
		Payload:   forwardedPayload,
		Metadata:  envelope.Metadata,
		Timestamp: payload.OccurredAt,
	}
	if err := c.enqueuer.EnqueueEvent(ctx, forwarded); err != nil {
		return fmt.Errorf("enqueue alarm lifecycle for northbound: %w", err)
	}
	return nil
}

// Ready verifies that the fixed durable exists and has applied every message
// visible at the time of the canonical cutover check.
func (c *AlarmLifecycleConsumer) Ready(ctx context.Context) error {
	if c == nil {
		return ErrAlarmLifecycleConsumerNotSubscribed
	}
	c.mu.Lock()
	subscribed := c.subscription != nil
	c.mu.Unlock()
	if !subscribed {
		return ErrAlarmLifecycleConsumerNotSubscribed
	}
	return AlarmLifecycleDurableReady(ctx, c.bus)
}

// AlarmLifecycleDurableReady lets an alarm-producing worker verify the
// northbound durable owned by the app process before enabling canonical mode.
func AlarmLifecycleDurableReady(ctx context.Context, bus event.EventBus) error {
	durableBus, ok := bus.(alarmLifecycleDurableBus)
	if !ok {
		return fmt.Errorf("read northbound alarm lifecycle durable health: unsupported event bus")
	}
	stats, err := durableBus.QueueStats(
		ctx,
		event.SubjectDomainAlarmLifecycleAll,
		AlarmLifecycleConsumerDurable,
	)
	if err != nil {
		return fmt.Errorf("read northbound alarm lifecycle durable health: %w", err)
	}
	if stats.Pending > 0 || stats.AckPending > 0 {
		return fmt.Errorf(
			"%w: pending=%d ack_pending=%d",
			ErrAlarmLifecycleConsumerNotCaughtUp,
			stats.Pending,
			stats.AckPending,
		)
	}
	return nil
}

func (c *AlarmLifecycleConsumer) Close() error {
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
		return fmt.Errorf("unsubscribe alarm lifecycle northbound consumer: %w", err)
	}
	return nil
}
