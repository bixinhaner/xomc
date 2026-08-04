package push

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

type recordingLifecycleEnqueuer struct {
	events []event.Event
	err    error
}

type lifecycleSubscription struct{}

func (lifecycleSubscription) Unsubscribe() error { return nil }

type recordingLifecycleBus struct {
	subject      string
	config       event.KeyedQueueConfig
	keyFunc      event.EventKeyFunc
	handler      event.EventHandler
	stats        event.QueueStats
	statsSubject string
	statsDurable string
}

func (b *recordingLifecycleBus) Publish(context.Context, string, event.Event) error { return nil }
func (b *recordingLifecycleBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return lifecycleSubscription{}, nil
}
func (b *recordingLifecycleBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return lifecycleSubscription{}, nil
}
func (b *recordingLifecycleBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return lifecycleSubscription{}, nil
}
func (b *recordingLifecycleBus) Close() error { return nil }
func (b *recordingLifecycleBus) KeyedQueueSubscribe(
	subject string,
	config event.KeyedQueueConfig,
	keyFunc event.EventKeyFunc,
	handler event.EventHandler,
) (event.Subscription, error) {
	b.subject = subject
	b.config = config
	b.keyFunc = keyFunc
	b.handler = handler
	return lifecycleSubscription{}, nil
}
func (b *recordingLifecycleBus) QueueStats(_ context.Context, subject, durable string) (event.QueueStats, error) {
	b.statsSubject = subject
	b.statsDurable = durable
	return b.stats, nil
}

func (r *recordingLifecycleEnqueuer) EnqueueEvent(_ context.Context, evt event.Event) error {
	r.events = append(r.events, evt)
	return r.err
}

func TestAlarmLifecycleConsumer_HandleEnqueuesSnapshotBeforeAck(t *testing.T) {
	occurredAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	alarm := model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "ENB-001",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmCritical,
		AlarmType:       "communications",
		AlarmIdentifier: "CELL_UNAVAILABLE",
		Description:     "cell unavailable",
		Status:          model.AlarmActive,
		RaisedAt:        occurredAt,
	}
	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised,
		alarm,
		nil,
		1,
		occurredAt,
		nil,
	)
	require.NoError(t, err)
	subject, err := payload.LifecycleType.Subject()
	require.NoError(t, err)
	envelope := event.Event{
		ID:        payload.EventID.String(),
		Subject:   subject,
		Timestamp: occurredAt,
	}
	envelope.Payload, err = json.Marshal(payload)
	require.NoError(t, err)

	enqueuer := &recordingLifecycleEnqueuer{}
	consumer := NewAlarmLifecycleConsumer(enqueuer, nil, 0)
	require.NoError(t, consumer.Handle(context.Background(), envelope))

	require.Len(t, enqueuer.events, 1)
	forwarded := enqueuer.events[0]
	require.Equal(t, payload.EventID.String(), forwarded.ID)
	require.Equal(t, event.SubjectOSSAlarmForward, forwarded.Subject)
	require.Equal(t, occurredAt, forwarded.Timestamp)

	var forwardedPayload event.AlarmLifecycleSnapshot
	require.NoError(t, forwarded.DecodePayload(&forwardedPayload))
	require.Equal(t, payload.Snapshot, forwardedPayload)
}

func TestAlarmLifecycleConsumer_SubscribeUsesFixedCanonicalDurable(t *testing.T) {
	bus := &recordingLifecycleBus{}
	consumer := NewAlarmLifecycleConsumer(&recordingLifecycleEnqueuer{}, bus, 42)

	require.NoError(t, consumer.Subscribe())
	require.Equal(t, event.SubjectDomainAlarmLifecycleAll, bus.subject)
	require.Equal(t, AlarmLifecycleConsumerDurable, bus.config.Durable)
	require.Equal(t, uint64(42), bus.config.StartSequence)
	require.NotNil(t, bus.keyFunc)
	require.NotNil(t, bus.handler)
	require.NotEqual(t, event.SubjectAlarmRaised, bus.subject)

	payload := event.AlarmLifecyclePayload{OccurrenceID: uuid.New()}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	key, err := bus.keyFunc(event.Event{Payload: data})
	require.NoError(t, err)
	require.Equal(t, payload.OccurrenceID.String(), key)
}

func TestAlarmLifecycleConsumer_HandleDoesNotAckWhenNorthboundOutboxInsertFails(t *testing.T) {
	occurredAt := time.Date(2026, 8, 4, 12, 30, 0, 0, time.UTC)
	alarm := model.Alarm{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "ENB-002",
		Severity: model.AlarmMajor, Status: model.AlarmActive, RaisedAt: occurredAt,
	}
	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised, alarm, nil, 1, occurredAt, nil,
	)
	require.NoError(t, err)
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	repo := newMockOutboxRepo()
	repo.insertErr = errors.New("database unavailable")
	engine := NewEngine(nil, testLogger())
	engine.SetOutboxRepo(repo)
	engine.AddTarget(&Target{ID: "oss-1", DataTypes: []string{"alarm"}, Enabled: true})
	consumer := NewAlarmLifecycleConsumer(engine, nil, 0)

	err = consumer.Handle(context.Background(), event.Event{
		ID: payload.EventID.String(), Subject: event.SubjectDomainAlarmLifecycleRaised,
		Payload: data, Timestamp: occurredAt,
	})
	require.ErrorContains(t, err, "database unavailable")
}

func TestAlarmLifecycleConsumer_ReadyRejectsDurableBacklog(t *testing.T) {
	bus := &recordingLifecycleBus{stats: event.QueueStats{Pending: 3, AckPending: 1}}
	consumer := NewAlarmLifecycleConsumer(&recordingLifecycleEnqueuer{}, bus, 42)
	require.NoError(t, consumer.Subscribe())

	err := consumer.Ready(context.Background())
	require.ErrorContains(t, err, "not caught up")
	require.Equal(t, event.SubjectDomainAlarmLifecycleAll, bus.statsSubject)
	require.Equal(t, AlarmLifecycleConsumerDurable, bus.statsDurable)
}

func TestAlarmLifecycleConsumer_ShadowValidatesWithoutCreatingDelivery(t *testing.T) {
	occurredAt := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)
	alarm := model.Alarm{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "ENB-SHADOW",
		Severity: model.AlarmWarning, Status: model.AlarmActive, RaisedAt: occurredAt,
	}
	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleRaised, alarm, nil, 1, occurredAt, nil,
	)
	require.NoError(t, err)
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	enqueuer := &recordingLifecycleEnqueuer{}
	consumer := NewAlarmLifecycleConsumer(enqueuer, nil, 0)
	consumer.SetShadow(true)
	require.NoError(t, consumer.Handle(context.Background(), event.Event{
		ID: payload.EventID.String(), Subject: event.SubjectDomainAlarmLifecycleRaised,
		Payload: data, Timestamp: occurredAt,
	}))
	require.Empty(t, enqueuer.events)
}

func TestAlarmLifecycleConsumer_RejectsSystemEventWithoutManagedElement(t *testing.T) {
	eventID := uuid.New()
	occurrenceID := uuid.New()
	payload := event.AlarmLifecyclePayload{
		SchemaVersion: event.AlarmLifecycleSchemaVersion,
		EventID:       eventID,
		LifecycleType: event.AlarmLifecycleRaised,
		OccurredAt:    time.Date(2026, 8, 4, 13, 30, 0, 0, time.UTC),
		OccurrenceID:  occurrenceID,
		AlarmVersion:  1,
		Snapshot: event.AlarmLifecycleSnapshot{
			AlarmID: occurrenceID,
			// DeviceID deliberately absent: platform health incidents are not
			// managed-element alarms and must stay outside this consumer.
		},
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	enqueuer := &recordingLifecycleEnqueuer{}
	consumer := NewAlarmLifecycleConsumer(enqueuer, nil, 0)

	err = consumer.Handle(context.Background(), event.Event{
		ID: eventID.String(), Subject: event.SubjectDomainAlarmLifecycleRaised,
		Payload: data, Timestamp: payload.OccurredAt,
	})
	require.ErrorContains(t, err, "managed-element device ID is required")
	require.Empty(t, enqueuer.events)
}
