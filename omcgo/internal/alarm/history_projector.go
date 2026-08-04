package alarm

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/omcgo/omcgo/internal/core/event"
)

const AlarmHistoryProjectorDurable = "alarm-history-projector-v1"

var (
	ErrAlarmHistoryShadowMismatch         = errors.New("alarm history shadow projection mismatch")
	ErrAlarmHistoryProjectorNotSubscribed = errors.New("alarm history projector is not subscribed")
	ErrAlarmHistoryProjectorNotCaughtUp   = errors.New("alarm history projector durable is not caught up")
)

type HistoryProjectionErrorCode string

const (
	HistoryProjectionErrorUnsupportedSchema HistoryProjectionErrorCode = "unsupported_schema"
	HistoryProjectionErrorInvalidEvent      HistoryProjectionErrorCode = "invalid_event"
)

// HistoryProjectionError is stable enough for structured logs and NATS retry
// diagnosis without introducing a general-purpose projection error framework.
type HistoryProjectionError struct {
	Code   HistoryProjectionErrorCode
	Detail string
	Err    error
}

func (e *HistoryProjectionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("alarm history projection %s: %s: %v", e.Code, e.Detail, e.Err)
	}
	return fmt.Sprintf("alarm history projection %s: %s", e.Code, e.Detail)
}

func (e *HistoryProjectionError) Unwrap() error { return e.Err }

type historyProjectionStore interface {
	ProjectAlarmHistory(context.Context, event.AlarmLifecyclePayload) (bool, error)
	CompareAlarmHistory(context.Context, event.AlarmLifecyclePayload) (bool, error)
}

type historyDurableBus interface {
	KeyedQueueSubscribe(
		string,
		event.KeyedQueueConfig,
		event.EventKeyFunc,
		event.EventHandler,
	) (event.Subscription, error)
	QueueStats(context.Context, string, string) (event.QueueStats, error)
}

// HistoryProjector is intentionally a single-purpose cleared-alarm consumer.
// It owns no scheduler, checkpoint table, or retry loop; JetStream durable state
// and the TSDB unique key provide those guarantees.
type HistoryProjector struct {
	store         historyProjectionStore
	bus           event.EventBus
	startSequence uint64
	shadow        bool

	mu           sync.Mutex
	subscription event.Subscription
}

func NewHistoryProjector(
	store historyProjectionStore,
	bus event.EventBus,
	startSequence uint64,
) *HistoryProjector {
	return &HistoryProjector{
		store: store, bus: bus, startSequence: startSequence,
	}
}

func (p *HistoryProjector) SetShadow(shadow bool) { p.shadow = shadow }

func (p *HistoryProjector) Subscribe() error {
	if p == nil || p.store == nil || p.bus == nil {
		return fmt.Errorf("subscribe alarm history projector: dependencies are required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.subscription != nil {
		return nil
	}
	durableBus, ok := p.bus.(historyDurableBus)
	if !ok {
		return fmt.Errorf("subscribe alarm history projector: event bus does not support fixed durable consumers")
	}
	subscription, err := durableBus.KeyedQueueSubscribe(
		event.SubjectDomainAlarmLifecycleCleared,
		event.KeyedQueueConfig{
			Durable: AlarmHistoryProjectorDurable, StartSequence: p.startSequence,
			Concurrency: 1, QueueDepth: 64, MaxAckPending: 64,
		},
		func(envelope event.Event) (string, error) {
			payload, err := decodeHistoryProjectionPayload(envelope)
			if err != nil {
				return "", err
			}
			return payload.OccurrenceID.String(), nil
		},
		p.Handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe alarm history projector durable: %w", err)
	}
	p.subscription = subscription
	return nil
}

func (p *HistoryProjector) Handle(ctx context.Context, envelope event.Event) error {
	payload, err := decodeHistoryProjectionPayload(envelope)
	if err != nil {
		return err
	}
	if p.shadow {
		exists, err := p.store.CompareAlarmHistory(ctx, payload)
		if err != nil {
			return fmt.Errorf("compare shadow alarm history: %w", err)
		}
		if !exists {
			return fmt.Errorf("%w: occurrence=%s version=%d", ErrAlarmHistoryShadowMismatch, payload.OccurrenceID, payload.AlarmVersion)
		}
		return nil
	}
	if _, err := p.store.ProjectAlarmHistory(ctx, payload); err != nil {
		return fmt.Errorf("project cleared alarm history: %w", err)
	}
	return nil
}

func decodeHistoryProjectionPayload(envelope event.Event) (event.AlarmLifecyclePayload, error) {
	var payload event.AlarmLifecyclePayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return payload, &HistoryProjectionError{
			Code: HistoryProjectionErrorInvalidEvent, Detail: "decode payload", Err: err,
		}
	}
	if payload.SchemaVersion != event.AlarmLifecycleSchemaVersion {
		return payload, &HistoryProjectionError{
			Code:   HistoryProjectionErrorUnsupportedSchema,
			Detail: fmt.Sprintf("schema_version=%d", payload.SchemaVersion),
		}
	}
	if envelope.Subject != event.SubjectDomainAlarmLifecycleCleared || payload.LifecycleType != event.AlarmLifecycleCleared {
		return payload, &HistoryProjectionError{
			Code: HistoryProjectionErrorInvalidEvent, Detail: "only cleared lifecycle events are accepted",
		}
	}
	if envelope.ID != payload.EventID.String() {
		return payload, &HistoryProjectionError{
			Code: HistoryProjectionErrorInvalidEvent, Detail: "event identity mismatch",
		}
	}
	if _, _, err := alarmHistoryFromLifecycle(payload); err != nil {
		return payload, &HistoryProjectionError{
			Code: HistoryProjectionErrorInvalidEvent, Detail: "invalid cleared snapshot", Err: err,
		}
	}
	return payload, nil
}

func (p *HistoryProjector) Ready(ctx context.Context) error {
	if p == nil {
		return ErrAlarmHistoryProjectorNotSubscribed
	}
	p.mu.Lock()
	subscribed := p.subscription != nil
	p.mu.Unlock()
	if !subscribed {
		return ErrAlarmHistoryProjectorNotSubscribed
	}
	durableBus, ok := p.bus.(historyDurableBus)
	if !ok {
		return fmt.Errorf("read history projector durable health: unsupported event bus")
	}
	stats, err := durableBus.QueueStats(ctx, event.SubjectDomainAlarmLifecycleCleared, AlarmHistoryProjectorDurable)
	if err != nil {
		return fmt.Errorf("read history projector durable health: %w", err)
	}
	if stats.Pending > 0 || stats.AckPending > 0 {
		return fmt.Errorf("%w: pending=%d ack_pending=%d", ErrAlarmHistoryProjectorNotCaughtUp, stats.Pending, stats.AckPending)
	}
	return nil
}

func (p *HistoryProjector) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	subscription := p.subscription
	p.subscription = nil
	p.mu.Unlock()
	if subscription == nil {
		return nil
	}
	if err := subscription.Unsubscribe(); err != nil {
		return fmt.Errorf("unsubscribe alarm history projector: %w", err)
	}
	return nil
}

var _ historyProjectionStore = (*PgAlarmStore)(nil)
