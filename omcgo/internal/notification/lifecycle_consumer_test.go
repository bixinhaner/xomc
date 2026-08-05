package notification

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

type lifecycleRepositoryStub struct {
	payload event.AlarmLifecyclePayload
	result  LifecycleApplyResult
	err     error
	calls   int
}

type occurrenceOrchestratorStub struct {
	occurrenceID uuid.UUID
	err          error
}

func (s *occurrenceOrchestratorStub) ProcessOccurrence(_ context.Context, id uuid.UUID) error {
	s.occurrenceID = id
	return s.err
}

func (s *lifecycleRepositoryStub) ApplyLifecycle(_ context.Context, payload event.AlarmLifecyclePayload) (LifecycleApplyResult, error) {
	s.calls++
	s.payload = payload
	return s.result, s.err
}

type notificationLifecycleSubscription struct{ unsubscribed bool }

func (s *notificationLifecycleSubscription) Unsubscribe() error {
	s.unsubscribed = true
	return nil
}

type notificationLifecycleBusStub struct {
	subject      string
	config       event.KeyedQueueConfig
	keyFunc      event.EventKeyFunc
	handler      event.EventHandler
	stats        event.QueueStats
	statsSubject string
	statsDurable string
	subscription *notificationLifecycleSubscription
}

func (*notificationLifecycleBusStub) Publish(context.Context, string, event.Event) error { return nil }
func (*notificationLifecycleBusStub) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return &notificationLifecycleSubscription{}, nil
}
func (*notificationLifecycleBusStub) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return &notificationLifecycleSubscription{}, nil
}
func (*notificationLifecycleBusStub) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return &notificationLifecycleSubscription{}, nil
}
func (*notificationLifecycleBusStub) Close() error { return nil }
func (b *notificationLifecycleBusStub) KeyedQueueSubscribe(
	subject string,
	config event.KeyedQueueConfig,
	keyFunc event.EventKeyFunc,
	handler event.EventHandler,
) (event.Subscription, error) {
	b.subject, b.config, b.keyFunc, b.handler = subject, config, keyFunc, handler
	b.subscription = &notificationLifecycleSubscription{}
	return b.subscription, nil
}
func (b *notificationLifecycleBusStub) QueueStats(_ context.Context, subject, durable string) (event.QueueStats, error) {
	b.statsSubject, b.statsDurable = subject, durable
	return b.stats, nil
}

func TestLifecycleConsumer_HandleAppliesValidatedManagedElementEvent(t *testing.T) {
	payload, envelope := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)
	repository := &lifecycleRepositoryStub{result: LifecycleApplyResult{State: LifecycleApplied, AppliedCount: 1}}
	consumer := NewLifecycleConsumer(repository, nil, 0)

	require.NoError(t, consumer.Handle(context.Background(), envelope))
	require.Equal(t, 1, repository.calls)
	require.Equal(t, payload, repository.payload)
}

func TestLifecycleConsumer_HandleRunsOrchestrationBeforeAcknowledgement(t *testing.T) {
	payload, envelope := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)
	repository := &lifecycleRepositoryStub{}
	orchestrator := &occurrenceOrchestratorStub{}
	consumer := NewLifecycleConsumer(repository, nil, 0)
	consumer.SetOrchestrator(orchestrator)

	require.NoError(t, consumer.Handle(context.Background(), envelope))
	require.Equal(t, payload.OccurrenceID, orchestrator.occurrenceID)
}

func TestLifecycleConsumer_HandleReturnsOrchestrationErrorForDurableRetry(t *testing.T) {
	payload, envelope := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)
	wantErr := errors.New("orchestration database unavailable")
	orchestrator := &occurrenceOrchestratorStub{err: wantErr}
	consumer := NewLifecycleConsumer(&lifecycleRepositoryStub{}, nil, 0)
	consumer.SetOrchestrator(orchestrator)

	err := consumer.Handle(context.Background(), envelope)
	require.ErrorIs(t, err, wantErr)
	require.ErrorContains(t, err, "orchestrate notification lifecycle")
	require.Equal(t, payload.OccurrenceID, orchestrator.occurrenceID)
}

func TestLifecycleConsumer_HandleReturnsRepositoryErrorWithoutAcknowledging(t *testing.T) {
	_, envelope := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)
	wantErr := errors.New("database unavailable")
	consumer := NewLifecycleConsumer(&lifecycleRepositoryStub{err: wantErr}, nil, 0)

	err := consumer.Handle(context.Background(), envelope)
	require.ErrorIs(t, err, wantErr)
	require.ErrorContains(t, err, "apply notification lifecycle")
}

func TestLifecycleConsumer_RejectsInvalidCanonicalEnvelope(t *testing.T) {
	_, valid := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)

	tests := []struct {
		name   string
		mutate func(*event.Event)
		want   string
	}{
		{name: "subject mismatch", mutate: func(envelope *event.Event) { envelope.Subject = event.SubjectDomainAlarmLifecycleCleared }, want: "subject"},
		{name: "event identity mismatch", mutate: func(envelope *event.Event) { envelope.ID = uuid.NewString() }, want: "identity mismatch"},
		{name: "unsupported schema", mutate: func(envelope *event.Event) {
			var payload event.AlarmLifecyclePayload
			require.NoError(t, envelope.DecodePayload(&payload))
			payload.SchemaVersion++
			envelope.Payload, _ = json.Marshal(payload)
		}, want: "unsupported schema_version"},
		{name: "system event without managed element", mutate: func(envelope *event.Event) {
			var payload event.AlarmLifecyclePayload
			require.NoError(t, envelope.DecodePayload(&payload))
			payload.Snapshot.DeviceID = uuid.Nil
			envelope.Payload, _ = json.Marshal(payload)
		}, want: "managed-element device ID"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envelope := valid
			test.mutate(&envelope)
			repository := &lifecycleRepositoryStub{}
			err := NewLifecycleConsumer(repository, nil, 0).Handle(context.Background(), envelope)
			require.ErrorContains(t, err, test.want)
			require.Zero(t, repository.calls)
		})
	}
}

func TestLifecycleConsumer_SubscribeUsesFixedOrderedDurable(t *testing.T) {
	bus := &notificationLifecycleBusStub{}
	consumer := NewLifecycleConsumer(&lifecycleRepositoryStub{}, bus, 42)
	require.NoError(t, consumer.Subscribe())

	require.Equal(t, event.SubjectDomainAlarmLifecycleAll, bus.subject)
	require.Equal(t, LifecycleConsumerDurable, bus.config.Durable)
	require.Equal(t, uint64(42), bus.config.StartSequence)
	require.Equal(t, 8, bus.config.Concurrency)
	require.Equal(t, 64, bus.config.QueueDepth)
	require.Equal(t, 512, bus.config.MaxAckPending)
	payload, envelope := notificationLifecycleFixture(t, event.AlarmLifecycleRaised, 1, model.AlarmActive)
	key, err := bus.keyFunc(envelope)
	require.NoError(t, err)
	require.Equal(t, payload.OccurrenceID.String(), key)
	require.NotNil(t, bus.handler)

	require.NoError(t, consumer.Close())
	require.True(t, bus.subscription.unsubscribed)
}

func TestLifecycleConsumer_ReadyRequiresCaughtUpDurable(t *testing.T) {
	bus := &notificationLifecycleBusStub{stats: event.QueueStats{Pending: 2, AckPending: 1}}
	consumer := NewLifecycleConsumer(&lifecycleRepositoryStub{}, bus, 0)
	require.NoError(t, consumer.Subscribe())

	err := consumer.Ready(context.Background())
	require.ErrorIs(t, err, ErrLifecycleConsumerNotCaughtUp)
	require.Equal(t, event.SubjectDomainAlarmLifecycleAll, bus.statsSubject)
	require.Equal(t, LifecycleConsumerDurable, bus.statsDurable)
}

func notificationLifecycleFixture(
	t *testing.T,
	lifecycleType event.AlarmLifecycleType,
	version int64,
	status model.AlarmStatus,
) (event.AlarmLifecyclePayload, event.Event) {
	t.Helper()
	now := time.Date(2026, 8, 5, 10, 0, int(version), 0, time.UTC)
	alarm := model.Alarm{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "LAB-ENB-001", Carrier: model.CarrierCMCC,
		Severity: model.AlarmMajor, AlarmType: "communications", AlarmIdentifier: "CELL_UNAVAILABLE",
		Description: "cell unavailable", Status: status, RaisedAt: now.Add(-time.Minute), LastUpdatedAt: now,
	}
	if status == model.AlarmAcknowledged {
		alarm.AcknowledgedAt = &now
	}
	if status == model.AlarmCleared {
		alarm.ClearedAt = &now
	}
	payload, err := event.NewAlarmLifecyclePayload(lifecycleType, alarm, nil, version, now, nil)
	require.NoError(t, err)
	subject, err := lifecycleType.Subject()
	require.NoError(t, err)
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return payload, event.Event{ID: payload.EventID.String(), Subject: subject, Timestamp: now, Payload: raw}
}
