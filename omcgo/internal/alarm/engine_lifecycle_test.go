package alarm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEngineLifecycle_DefaultLegacyDoesNotWriteCanonicalOutbox(t *testing.T) {
	store := newLifecycleRecordingStore()
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())

	require.NoError(t, engine.Process(context.Background(), lifecycleEngineAlarm()))
	require.Empty(t, store.lifecycleCalls)
	require.Len(t, store.active, 1)
}

func TestEngineLifecycle_ShadowRoutesOccurrenceChangesThroughLifecycleStore(t *testing.T) {
	store := newLifecycleRecordingStore()
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
	require.NoError(t, engine.SetLifecycleMode(LifecycleModeShadow))
	ctx := context.Background()

	alarm := lifecycleEngineAlarm()
	require.NoError(t, engine.Process(ctx, alarm))
	update := lifecycleEngineAlarm()
	update.DeviceID = alarm.DeviceID
	update.Severity = model.AlarmCritical
	update.Description = "severity escalated"
	require.NoError(t, engine.Process(ctx, update))
	require.NoError(t, engine.Acknowledge(ctx, alarm.ID, "operator"))
	require.NoError(t, engine.Unacknowledge(ctx, alarm.ID))
	require.NoError(t, engine.Clear(ctx, alarm.ID))

	require.Equal(t, []event.AlarmLifecycleType{
		event.AlarmLifecycleRaised,
		event.AlarmLifecycleUpdated,
		event.AlarmLifecycleAcknowledged,
		event.AlarmLifecycleUnacknowledged,
		event.AlarmLifecycleCleared,
	}, store.lifecycleCalls)
	require.ElementsMatch(t, []event.AlarmChangeField{
		event.AlarmChangeSeverity,
		event.AlarmChangeDescription,
		event.AlarmChangeCount,
	}, store.updateMask)
	require.Empty(t, store.active)
	require.Len(t, store.history, 1, "shadow must preserve synchronous legacy history")
	require.Equal(t, int64(5), store.history[0].Version, "shadow history identity must match cleared lifecycle version")
}

func TestEngineLifecycle_ShadowNewAutoClearWritesPairedFacts(t *testing.T) {
	store := newLifecycleRecordingStore()
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
	require.NoError(t, engine.SetLifecycleMode(LifecycleModeShadow))
	alarm := lifecycleEngineAlarm()

	require.NoError(t, engine.archiveAutoClearedAlarm(context.Background(), alarm))
	require.Equal(t, []event.AlarmLifecycleType{
		event.AlarmLifecycleRaised,
		event.AlarmLifecycleCleared,
	}, store.lifecycleCalls)
	require.Equal(t, int64(2), alarm.Version)
	require.Empty(t, store.active)
	require.Len(t, store.history, 1)
	require.Equal(t, int64(2), store.history[0].Version)
}

func TestEngineLifecycle_ShadowNewAutoAcknowledgeWritesPairedFacts(t *testing.T) {
	store := newLifecycleRecordingStore()
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
	require.NoError(t, engine.SetLifecycleMode(LifecycleModeShadow))
	alarm := lifecycleEngineAlarm()
	alarm.Status = model.AlarmAcknowledged
	acknowledgedAt := time.Now().UTC()
	alarm.AcknowledgedAt = &acknowledgedAt

	require.NoError(t, engine.Process(context.Background(), alarm))
	require.Equal(t, []event.AlarmLifecycleType{
		event.AlarmLifecycleRaised,
		event.AlarmLifecycleAcknowledged,
	}, store.lifecycleCalls)
	require.Equal(t, int64(2), alarm.Version)
	require.Equal(t, model.AlarmAcknowledged, store.active[alarm.ID].Status)
}

func TestEngineLifecycle_RejectsCanonicalBeforeProjectorIsReady(t *testing.T) {
	engine := NewAlarmEngine(newLifecycleRecordingStore(), nil, nil, nil, zap.NewNop())
	require.ErrorIs(t, engine.SetLifecycleMode(LifecycleModeCanonical), ErrCanonicalLifecycleNotReady)
}

func TestEngineLifecycle_CanonicalUsesProjectorInsteadOfSynchronousArchive(t *testing.T) {
	store := newLifecycleRecordingStore()
	engine := NewAlarmEngine(store, nil, nil, nil, zap.NewNop())
	engine.SetCanonicalLifecycleReady(true)
	require.NoError(t, engine.SetLifecycleMode(LifecycleModeCanonical))
	alarm := lifecycleEngineAlarm()

	require.NoError(t, engine.Process(context.Background(), alarm))
	require.NoError(t, engine.Clear(context.Background(), alarm.ID))
	require.Empty(t, store.history, "canonical history has exactly one formal writer: the projector")
	require.Equal(t, []event.AlarmLifecycleType{
		event.AlarmLifecycleRaised,
		event.AlarmLifecycleCleared,
	}, store.lifecycleCalls)
}

func lifecycleEngineAlarm() *model.Alarm {
	return &model.Alarm{
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-ENGINE-LIFECYCLE",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "equipment",
		AlarmIdentifier: "CELL_UNAVAILABLE",
		Description:     "cell unavailable",
		RaisedAt:        time.Now().UTC(),
	}
}

type lifecycleRecordingStore struct {
	*mockAlarmStore
	lifecycleCalls []event.AlarmLifecycleType
	updateMask     []event.AlarmChangeField
}

func newLifecycleRecordingStore() *lifecycleRecordingStore {
	return &lifecycleRecordingStore{mockAlarmStore: newMockAlarmStore()}
}

func (s *lifecycleRecordingStore) PersistRaised(_ context.Context, alarm *model.Alarm) (event.AlarmLifecyclePayload, error) {
	alarm.Version = 1
	s.active[alarm.ID] = alarm
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleRaised)
	return event.AlarmLifecyclePayload{}, nil
}

func (s *lifecycleRecordingStore) PersistUpdated(_ context.Context, alarm *model.Alarm, mask []event.AlarmChangeField) (event.AlarmLifecyclePayload, error) {
	alarm.Version++
	s.active[alarm.ID] = alarm
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleUpdated)
	s.updateMask = append([]event.AlarmChangeField(nil), mask...)
	return event.AlarmLifecyclePayload{}, nil
}

func (s *lifecycleRecordingStore) PersistAcknowledged(_ context.Context, alarm *model.Alarm) (event.AlarmLifecyclePayload, error) {
	alarm.Version++
	s.active[alarm.ID] = alarm
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleAcknowledged)
	return event.AlarmLifecyclePayload{}, nil
}

func (s *lifecycleRecordingStore) PersistUnacknowledged(_ context.Context, alarm *model.Alarm) (event.AlarmLifecyclePayload, error) {
	alarm.Version++
	s.active[alarm.ID] = alarm
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleUnacknowledged)
	return event.AlarmLifecyclePayload{}, nil
}

func (s *lifecycleRecordingStore) PersistCleared(_ context.Context, alarm *model.Alarm) (event.AlarmLifecyclePayload, error) {
	alarm.Version++
	delete(s.active, alarm.ID)
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleCleared)
	return event.AlarmLifecyclePayload{}, nil
}

func (s *lifecycleRecordingStore) PersistRaisedAndCleared(_ context.Context, alarm *model.Alarm) ([]event.AlarmLifecyclePayload, error) {
	alarm.Version = 2
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleRaised, event.AlarmLifecycleCleared)
	return nil, nil
}

func (s *lifecycleRecordingStore) PersistRaisedAndAcknowledged(_ context.Context, alarm *model.Alarm) ([]event.AlarmLifecyclePayload, error) {
	alarm.Version = 2
	s.active[alarm.ID] = alarm
	s.lifecycleCalls = append(s.lifecycleCalls, event.AlarmLifecycleRaised, event.AlarmLifecycleAcknowledged)
	return nil, nil
}
