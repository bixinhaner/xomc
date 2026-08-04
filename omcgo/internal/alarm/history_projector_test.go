package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

type historyProjectionStoreStub struct {
	inserted map[string]struct{}
	projects []event.AlarmLifecyclePayload
	compares []event.AlarmLifecyclePayload
	err      error
	exists   bool
}

func (s *historyProjectionStoreStub) ProjectAlarmHistory(_ context.Context, payload event.AlarmLifecyclePayload) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	s.projects = append(s.projects, payload)
	if s.inserted == nil {
		s.inserted = make(map[string]struct{})
	}
	key := historyProjectionKey(payload)
	if _, duplicate := s.inserted[key]; duplicate {
		return false, nil
	}
	s.inserted[key] = struct{}{}
	return true, nil
}

func (s *historyProjectionStoreStub) CompareAlarmHistory(_ context.Context, payload event.AlarmLifecyclePayload) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	s.compares = append(s.compares, payload)
	return s.exists, nil
}

func historyProjectionKey(payload event.AlarmLifecyclePayload) string {
	return payload.Snapshot.ClearedAt.UTC().Format(time.RFC3339Nano) + "/" +
		payload.OccurrenceID.String() + "/" + strconv.FormatInt(payload.AlarmVersion, 10)
}

type historyProjectorBusStub struct {
	subject string
	config  event.KeyedQueueConfig
	key     event.EventKeyFunc
	handler event.EventHandler
	stats   event.QueueStats
	statErr error
}

func (s *historyProjectorBusStub) KeyedQueueSubscribe(
	subject string,
	config event.KeyedQueueConfig,
	key event.EventKeyFunc,
	handler event.EventHandler,
) (event.Subscription, error) {
	s.subject, s.config, s.key, s.handler = subject, config, key, handler
	return historyProjectorSubscription{}, nil
}

func (s *historyProjectorBusStub) QueueStats(context.Context, string, string) (event.QueueStats, error) {
	return s.stats, s.statErr
}

func (*historyProjectorBusStub) Publish(context.Context, string, event.Event) error { return nil }
func (*historyProjectorBusStub) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unexpected Subscribe")
}
func (*historyProjectorBusStub) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unexpected QueueSubscribe")
}
func (*historyProjectorBusStub) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, errors.New("unexpected PullSubscribe")
}
func (*historyProjectorBusStub) Close() error { return nil }

type historyProjectorSubscription struct{}

func (historyProjectorSubscription) Unsubscribe() error { return nil }

func TestHistoryProjectorDuplicateClearedEventIsIdempotent(t *testing.T) {
	store := &historyProjectionStoreStub{}
	projector := NewHistoryProjector(store, &historyProjectorBusStub{}, 1)
	envelope := historyProjectorEnvelope(t, 1)

	require.NoError(t, projector.Handle(context.Background(), envelope))
	require.NoError(t, projector.Handle(context.Background(), envelope))
	require.Len(t, store.projects, 2)
	require.Len(t, store.inserted, 1, "same cleared_at + alarm_id + alarm_version must form one history row")
}

func TestHistoryProjectorTSDBFailureReturnsErrorForJetStreamRedelivery(t *testing.T) {
	tsdbErr := errors.New("timescaledb unavailable")
	store := &historyProjectionStoreStub{err: tsdbErr}
	projector := NewHistoryProjector(store, &historyProjectorBusStub{}, 1)

	err := projector.Handle(context.Background(), historyProjectorEnvelope(t, 1))
	require.ErrorIs(t, err, tsdbErr)
}

func TestHistoryProjectorShadowOnlyCompares(t *testing.T) {
	store := &historyProjectionStoreStub{exists: true}
	projector := NewHistoryProjector(store, &historyProjectorBusStub{}, 1)
	projector.SetShadow(true)

	require.NoError(t, projector.Handle(context.Background(), historyProjectorEnvelope(t, 1)))
	require.Empty(t, store.projects)
	require.Len(t, store.compares, 1)

	store.exists = false
	err := projector.Handle(context.Background(), historyProjectorEnvelope(t, 1))
	require.ErrorIs(t, err, ErrAlarmHistoryShadowMismatch)
}

func TestHistoryProjectorRejectsUnsupportedSchemaWithStructuredError(t *testing.T) {
	projector := NewHistoryProjector(&historyProjectionStoreStub{}, &historyProjectorBusStub{}, 1)
	envelope := historyProjectorEnvelope(t, event.AlarmLifecycleSchemaVersion+1)

	err := projector.Handle(context.Background(), envelope)
	var projectionErr *HistoryProjectionError
	require.ErrorAs(t, err, &projectionErr)
	require.Equal(t, HistoryProjectionErrorUnsupportedSchema, projectionErr.Code)
}

func TestHistoryProjectorSubscribeUsesFixedDurableAndStartSequence(t *testing.T) {
	bus := &historyProjectorBusStub{}
	projector := NewHistoryProjector(&historyProjectionStoreStub{}, bus, 417)

	require.NoError(t, projector.Subscribe())
	require.Equal(t, event.SubjectDomainAlarmLifecycleCleared, bus.subject)
	require.Equal(t, AlarmHistoryProjectorDurable, bus.config.Durable)
	require.Equal(t, uint64(417), bus.config.StartSequence)
	require.Equal(t, 1, bus.config.Concurrency, "one TSDB writer is sufficient until measured otherwise")
	require.NotNil(t, bus.key)
	require.NotNil(t, bus.handler)

	envelope := historyProjectorEnvelope(t, 1)
	key, err := bus.key(envelope)
	require.NoError(t, err)
	require.Equal(t, historyProjectorPayload(t, 1).OccurrenceID.String(), key)
}

func TestHistoryProjectorReadinessRequiresAvailableCaughtUpDurable(t *testing.T) {
	bus := &historyProjectorBusStub{stats: event.QueueStats{Pending: 0, AckPending: 0}}
	projector := NewHistoryProjector(&historyProjectionStoreStub{}, bus, 1)

	require.ErrorIs(t, projector.Ready(context.Background()), ErrAlarmHistoryProjectorNotSubscribed)
	require.NoError(t, projector.Subscribe())
	require.NoError(t, projector.Ready(context.Background()))

	bus.stats.Pending = 1
	require.ErrorIs(t, projector.Ready(context.Background()), ErrAlarmHistoryProjectorNotCaughtUp)
	bus.stats.Pending = 0
	bus.statErr = errors.New("nats unavailable")
	require.ErrorContains(t, projector.Ready(context.Background()), "read history projector durable health")
}

func TestHistoryProjectionSQLUsesDeterministicConflictKey(t *testing.T) {
	payload := historyProjectorPayload(t, 1)
	query, args, err := buildAlarmHistoryProjectionInsert(payload)
	require.NoError(t, err)
	normalized := strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "INSERT INTO alarms_history")
	require.Contains(t, normalized, "ON CONFLICT (time, alarm_id, alarm_version) DO NOTHING")
	require.NotEmpty(t, args)
	require.Equal(t, *payload.Snapshot.ClearedAt, args[0], "hypertable partition time must equal cleared_at")
}

func historyProjectorEnvelope(t *testing.T, schemaVersion int) event.Event {
	t.Helper()
	payload := historyProjectorPayload(t, schemaVersion)
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return event.Event{
		ID: payload.EventID.String(), Subject: event.SubjectDomainAlarmLifecycleCleared,
		Payload: data, Timestamp: payload.OccurredAt,
	}
}

func historyProjectorPayload(t *testing.T, schemaVersion int) event.AlarmLifecyclePayload {
	t.Helper()
	clearedAt := time.Date(2026, 8, 5, 1, 2, 3, 456000000, time.UTC)
	alarm := model.Alarm{
		ID:       uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		DeviceID: uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		DeviceSN: "SN-HISTORY-PROJECTOR", Carrier: model.CarrierCMCC,
		Severity: model.AlarmMajor, AlarmType: "equipment",
		AlarmIdentifier: "CELL_UNAVAILABLE", Description: "cell unavailable",
		Status: model.AlarmCleared, RaisedAt: clearedAt.Add(-time.Hour),
		ClearedAt: &clearedAt, AckCount: 1,
	}
	payload, err := event.NewAlarmLifecyclePayload(
		event.AlarmLifecycleCleared, alarm, nil, 3, clearedAt, nil,
	)
	require.NoError(t, err)
	payload.SchemaVersion = schemaVersion
	return payload
}
