package alarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	carrierpkg "github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockAlarmStore implements AlarmStore for testing.
type mockAlarmStore struct {
	active           map[uuid.UUID]*model.Alarm
	history          []*model.Alarm
	lastActiveFilter AlarmFilter
	lastHistoryFilter AlarmFilter
	returnNotFoundOnLookup bool
}

func newMockAlarmStore() *mockAlarmStore {
	return &mockAlarmStore{active: make(map[uuid.UUID]*model.Alarm)}
}

func (m *mockAlarmStore) SaveActive(_ context.Context, alarm *model.Alarm) error {
	m.active[alarm.ID] = alarm
	return nil
}

func (m *mockAlarmStore) GetActiveByID(_ context.Context, id uuid.UUID) (*model.Alarm, error) {
	a, ok := m.active[id]
	if !ok {
		return nil, fmt.Errorf("alarm not found: %w", commonerrors.ErrNotFound)
	}
	return a, nil
}

func (m *mockAlarmStore) GetHistoryByID(_ context.Context, id uuid.UUID) (*model.Alarm, error) {
	for _, a := range m.history {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("history alarm not found: %w", commonerrors.ErrNotFound)
}

func (m *mockAlarmStore) GetActiveByDeviceAndIdentifier(_ context.Context, deviceSN, alarmIdentifier string) (*model.Alarm, error) {
	for _, a := range m.active {
		if a.DeviceSN == deviceSN && a.AlarmIdentifier == alarmIdentifier && a.Status != model.AlarmCleared {
			return a, nil
		}
	}
	if m.returnNotFoundOnLookup {
		return nil, fmt.Errorf("alarm not found: %w", commonerrors.ErrNotFound)
	}
	return nil, nil // not found — return nil, nil (like a real store)
}

func (m *mockAlarmStore) GetActiveByDeviceSN(_ context.Context, deviceSN string) ([]*model.Alarm, error) {
	var result []*model.Alarm
	for _, a := range m.active {
		if a.DeviceSN == deviceSN && a.Status != model.AlarmCleared {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAlarmStore) UpdateActive(_ context.Context, alarm *model.Alarm) error {
	m.active[alarm.ID] = alarm
	return nil
}

func (m *mockAlarmStore) RemoveActive(_ context.Context, id uuid.UUID) error {
	delete(m.active, id)
	return nil
}

func (m *mockAlarmStore) ListActive(_ context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	m.lastActiveFilter = filter
	var items []model.Alarm
	for _, a := range m.active {
		items = append(items, *a)
	}
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}

func (m *mockAlarmStore) Archive(_ context.Context, alarm *model.Alarm) error {
	m.history = append(m.history, alarm)
	return nil
}

func (m *mockAlarmStore) ListHistory(_ context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	m.lastHistoryFilter = filter
	var items []model.Alarm
	for _, a := range m.history {
		items = append(items, *a)
	}
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}

func (m *mockAlarmStore) Statistics(_ context.Context, _ AlarmFilter) (*AlarmStatistics, error) {
	stats := &AlarmStatistics{
		TotalActive: int64(len(m.active)),
		BySeverity:  make(map[model.AlarmSeverity]int64),
		ByType:      make(map[string]int64),
	}
	for _, a := range m.active {
		stats.BySeverity[a.Severity]++
		if a.AlarmType != "" {
			stats.ByType[a.AlarmType]++
		}
	}
	return stats, nil
}

func (m *mockAlarmStore) BatchAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchUnacknowledge(_ context.Context, _ []uuid.UUID) error                      { return nil }
func (m *mockAlarmStore) BatchClear(_ context.Context, _ []uuid.UUID, _ string, _ string) error         { return nil }
func (m *mockAlarmStore) BatchHistoryAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchHistoryUnacknowledge(_ context.Context, _ []uuid.UUID) error                { return nil }
func (m *mockAlarmStore) BatchHistoryDelete(_ context.Context, _ []uuid.UUID) error                       { return nil }
func (m *mockAlarmStore) MarkRead(_ context.Context, _ uuid.UUID) error                                    { return nil }
func (m *mockAlarmStore) HistoryStatistics(_ context.Context, _ AlarmFilter) (*AlarmStatistics, error)     { return nil, nil }

func newTestEngine(store AlarmStore) *AlarmEngine {
	return &AlarmEngine{
		store:  store,
		logger: zap.NewNop(),
	}
}

func TestProcessNewAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	raisedAt := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		AlarmType: "equipment",
		Severity:  model.AlarmMajor,
		RaisedAt:  raisedAt,
	}

	err := engine.Process(context.Background(), alarm)
	require.NoError(t, err)

	assert.Len(t, store.active, 1)
	assert.Equal(t, model.AlarmActive, alarm.Status)
	assert.NotEqual(t, uuid.Nil, alarm.ID)
	assert.Equal(t, 1, alarm.AckCount)
	assert.Equal(t, raisedAt, alarm.FirstRaisedAt)
	assert.Equal(t, raisedAt, alarm.LastUpdatedAt)
}

func TestProcessLegacyNotificationBarrierContinuesAlarmPersistence(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	engine.filterEngine = NewFilterEngine(&mockFilterRuleRepo{rules: []AlarmFilterRule{
		{
			Name:             "migrated-email-barrier",
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionLegacyNotificationBarrier,
			Priority:         10,
		},
		{
			Name:             "lower-auto-clear",
			AlarmIdentifiers: []string{"DEVICE_OFFLINE"},
			Action:           FilterActionAutoClear,
			Priority:         20,
		},
	}}, store, nil, nil, nil, zap.NewNop())
	alarm := &model.Alarm{
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-BARRIER",
		AlarmIdentifier: "DEVICE_OFFLINE",
		Severity:        model.AlarmMajor,
		Status:          model.AlarmActive,
	}

	err := engine.Process(context.Background(), alarm)

	require.NoError(t, err)
	require.Len(t, store.active, 1)
	require.Nil(t, alarm.ClearedBy)
}

func TestProcessPreservesSourceSeverityWhenCarrierMappingMissing(t *testing.T) {
	store := newMockAlarmStore()
	registry := carrierpkg.NewRegistry()
	registry.Register(cmcc.New())
	engine := &AlarmEngine{
		store:           store,
		carrierRegistry: registry,
		logger:          zap.NewNop(),
	}

	alarm := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "70011",
		AlarmType:       "equipment",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
	}

	require.NoError(t, engine.Process(context.Background(), alarm))

	stored, err := store.GetActiveByDeviceAndIdentifier(context.Background(), "TEST001", "70011")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, model.AlarmMajor, stored.Severity)
}

func TestProcessDuplicateAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()
	firstRaisedAt := time.Now().Add(-10 * time.Minute).UTC().Truncate(time.Second)
	secondRaisedAt := time.Now().UTC().Truncate(time.Second)

	alarm1 := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		AlarmType: "equipment",
		Severity:  model.AlarmMajor,
		RaisedAt:  firstRaisedAt,
	}
	require.NoError(t, engine.Process(ctx, alarm1))

	alarm2 := &model.Alarm{
		DeviceSN:    "TEST001",
		DeviceID:    alarm1.DeviceID,
		Carrier:     model.CarrierCMCC,
		AlarmIdentifier:   "ALM001",
		AlarmType:   "equipment",
		Severity:    model.AlarmCritical,
		RaisedAt:    secondRaisedAt,
		LastUpdatedAt: secondRaisedAt,
		Description: "updated description",
	}
	require.NoError(t, engine.Process(ctx, alarm2))

	// Should still have only 1 alarm (dedup via DB fallback)
	assert.Len(t, store.active, 1)

	// The existing alarm should have updated fields
	for _, a := range store.active {
		assert.Equal(t, model.AlarmCritical, a.Severity)
		assert.Equal(t, "updated description", a.Description)
		assert.Equal(t, 2, a.AckCount)
		assert.Equal(t, firstRaisedAt, a.RaisedAt)
		assert.Equal(t, firstRaisedAt, a.FirstRaisedAt)
		assert.Equal(t, secondRaisedAt, a.LastUpdatedAt)
	}
}

func TestUpdateByEvent_UsesDeviceTimestampForLastUpdatedAt(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	deviceID := uuid.New()
	originalRaisedAt := time.Now().Add(-10 * time.Minute).UTC().Truncate(time.Second)
	changedAt := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)
	alarm := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		Description:     "original description",
		RaisedAt:        originalRaisedAt,
	}
	require.NoError(t, engine.Process(ctx, alarm))

	updated := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmCritical,
		Description:     "updated description",
		LastUpdatedAt:   changedAt,
	}

	require.NoError(t, engine.UpdateByEvent(ctx, updated))

	existing, err := store.GetActiveByDeviceAndIdentifier(ctx, "TEST001", "ALM001")
	require.NoError(t, err)
	assert.Equal(t, changedAt, existing.LastUpdatedAt)
}

func TestProcessKeepsDistinctActiveAlarmsForSameIdentifierWithDifferentAdditionalInformation(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	deviceID := uuid.New()
	alarm1 := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "11184",
		AlarmType:       "equipment",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now().Add(-time.Minute),
		AdditionalInfo:  map[string]string{"additional_information": "cell=1"},
	}
	alarm2 := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "11184",
		AlarmType:       "equipment",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
		AdditionalInfo:  map[string]string{"additional_information": "cell=2"},
	}

	require.NoError(t, engine.Process(ctx, alarm1))
	require.NoError(t, engine.Process(ctx, alarm2))

	assert.Len(t, store.active, 2)
	active := store.active[alarm1.ID]
	require.NotNil(t, active)
	assert.Equal(t, "cell=1", active.AdditionalInfo["additional_information"])
	active = store.active[alarm2.ID]
	require.NotNil(t, active)
	assert.Equal(t, "cell=2", active.AdditionalInfo["additional_information"])
}

func TestAcknowledgeAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))

	err := engine.Acknowledge(ctx, alarm.ID, "admin@test.com")
	require.NoError(t, err)

	stored := store.active[alarm.ID]
	assert.Equal(t, model.AlarmAcknowledged, stored.Status)
	assert.Equal(t, "admin@test.com", *stored.AcknowledgedBy)
	assert.NotNil(t, stored.AcknowledgedAt)
}

func TestAcknowledgeNonActiveAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))
	require.NoError(t, engine.Acknowledge(ctx, alarm.ID, "admin"))

	// Try to acknowledge again
	err := engine.Acknowledge(ctx, alarm.ID, "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in active state")
}

func TestClearAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))

	err := engine.Clear(ctx, alarm.ID)
	require.NoError(t, err)

	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
	assert.NotNil(t, store.history[0].ClearedAt)
}

func TestProcessAutoClearArchivesHistoryWhenNoActiveExists(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	engine.SetFilterEngine(newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"ALM001"}, Action: FilterActionAutoClear, Name: "auto-clear-alm001"},
	}, nil))

	alarm := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
	}

	require.NoError(t, engine.Process(context.Background(), alarm))
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
	assert.NotNil(t, store.history[0].ClearedAt)
	require.NotNil(t, store.history[0].ClearedBy)
	assert.Equal(t, "system", *store.history[0].ClearedBy)
	require.NotNil(t, store.history[0].ClearNote)
	assert.Equal(t, "auto-cleared by alarm rule: auto-clear-alm001", *store.history[0].ClearNote)
}

func TestProcessAutoClearArchivesHistoryWhenStoreReturnsNotFound(t *testing.T) {
	store := newMockAlarmStore()
	store.returnNotFoundOnLookup = true
	engine := newTestEngine(store)
	engine.SetFilterEngine(newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"ALM001"}, Action: FilterActionAutoClear, Name: "auto-clear-alm001"},
	}, nil))

	alarm := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now(),
	}

	require.NoError(t, engine.Process(context.Background(), alarm))
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
}

func TestProcessAutoClearArchivesAndRemovesExistingActive(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now().Add(-time.Minute),
		Status:          model.AlarmActive,
	}
	require.NoError(t, store.SaveActive(ctx, existing))

	engine.SetFilterEngine(newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"ALM001"}, Action: FilterActionAutoClear, Name: "auto-clear-alm001"},
	}, nil))

	incoming := &model.Alarm{
		DeviceSN:        existing.DeviceSN,
		DeviceID:        existing.DeviceID,
		Carrier:         existing.Carrier,
		AlarmIdentifier: existing.AlarmIdentifier,
		Severity:        existing.Severity,
		RaisedAt:        time.Now(),
	}

	require.NoError(t, engine.Process(ctx, incoming))
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, existing.ID, store.history[0].ID)
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
	assert.NotNil(t, store.history[0].ClearedAt)
	require.NotNil(t, store.history[0].ClearedBy)
	assert.Equal(t, "system", *store.history[0].ClearedBy)
	require.NotNil(t, store.history[0].ClearNote)
	assert.Equal(t, "auto-cleared by alarm rule: auto-clear-alm001", *store.history[0].ClearNote)
}

func TestProcessAutoAcknowledgeStoresAckNoteOnExistingActive(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now().Add(-time.Minute),
		Status:          model.AlarmActive,
	}
	require.NoError(t, store.SaveActive(ctx, existing))

	engine.SetFilterEngine(newTestFilterEngine([]AlarmFilterRule{
		{FilterType: FilterTypeAlarmIdentifier, AlarmIdentifiers: []string{"ALM001"}, Action: FilterActionAutoAcknowledge, Name: "auto-ack-alm001", AcknowledgeDesc: "acknowledged by alarm rule"},
	}, nil))

	incoming := &model.Alarm{
		DeviceSN:        existing.DeviceSN,
		DeviceID:        existing.DeviceID,
		Carrier:         existing.Carrier,
		AlarmIdentifier: existing.AlarmIdentifier,
		Severity:        existing.Severity,
		RaisedAt:        time.Now(),
	}

	require.NoError(t, engine.Process(ctx, incoming))
	stored, err := store.GetActiveByID(ctx, existing.ID)
	require.NoError(t, err)
	assert.Equal(t, model.AlarmAcknowledged, stored.Status)
	require.NotNil(t, stored.AcknowledgedBy)
	assert.Equal(t, "system:auto_filter:auto-ack-alm001", *stored.AcknowledgedBy)
	require.NotNil(t, stored.AckNote)
	assert.Equal(t, "acknowledged by alarm rule", *stored.AckNote)
}

func TestAutoClearPreservesUpdateTimeAndSetsSystemClearUser(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()
	lastUpdatedAt := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Second)

	existing := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "TEST001",
		DeviceID:        uuid.New(),
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		RaisedAt:        time.Now().Add(-5 * time.Minute),
		Status:          model.AlarmActive,
		LastUpdatedAt:   lastUpdatedAt,
	}
	require.NoError(t, store.SaveActive(ctx, existing))

	require.NoError(t, engine.AutoClear(ctx, &model.Alarm{
		DeviceSN:        existing.DeviceSN,
		DeviceID:        existing.DeviceID,
		Carrier:         existing.Carrier,
		AlarmIdentifier: existing.AlarmIdentifier,
	}))

	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, lastUpdatedAt, store.history[0].LastUpdatedAt)
	require.NotNil(t, store.history[0].ClearedBy)
	assert.Equal(t, "system", *store.history[0].ClearedBy)
	require.NotNil(t, store.history[0].ClearNote)
	assert.Equal(t, "auto-cleared", *store.history[0].ClearNote)
}

func TestAlarmFullLifecycle(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))
	assert.Equal(t, model.AlarmActive, store.active[alarm.ID].Status)

	require.NoError(t, engine.Acknowledge(ctx, alarm.ID, "admin"))
	assert.Equal(t, model.AlarmAcknowledged, store.active[alarm.ID].Status)

	require.NoError(t, engine.Clear(ctx, alarm.ID))
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
}

func TestMultipleAlarmsDifferentCodes(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	deviceID := uuid.New()
	for _, code := range []string{"ALM001", "ALM002", "ALM003"} {
		alarm := &model.Alarm{
			DeviceSN:  "TEST001",
			DeviceID:  deviceID,
			Carrier:   model.CarrierCMCC,
			AlarmIdentifier: code,
			Severity:  model.AlarmMajor,
			RaisedAt:  time.Now(),
		}
		require.NoError(t, engine.Process(ctx, alarm))
	}

	assert.Len(t, store.active, 3)
}

func TestUpdateByEvent_ExistingAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	// Create initial alarm
	deviceID := uuid.New()
	alarm := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		Description:     "original description",
		EventType:        strPtr("communicationsAlarm"),
		ProbableCause:   strPtr("originalCause"),
		RaisedAt:        time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))

	// Update via expedited event (ChangedAlarm)
	updated := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmCritical,
		Description:     "updated description",
		EventType:        strPtr("equipmentAlarm"),
		ProbableCause:   strPtr("newCause"),
		AdditionalInfo:  map[string]string{"notification_type": "ChangedAlarm"},
	}

	err := engine.UpdateByEvent(ctx, updated)
	require.NoError(t, err)

	// Verify the existing alarm was updated in place
	existing, err := store.GetActiveByDeviceAndIdentifier(ctx, "TEST001", "ALM001")
	require.NoError(t, err)
	assert.Equal(t, model.AlarmCritical, existing.Severity)
	assert.Equal(t, "updated description", existing.Description)
	assert.Equal(t, "equipmentAlarm", *existing.EventType)
	assert.Equal(t, "newCause", *existing.ProbableCause)
	assert.Equal(t, "ChangedAlarm", existing.AdditionalInfo["notification_type"])
	assert.Equal(t, 2, existing.AckCount)
	assert.False(t, existing.LastUpdatedAt.IsZero())

	// Should still have only 1 alarm
	assert.Len(t, store.active, 1)
}

func TestUpdateByEvent_NoExistingAlarm_FallbackToProcess(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	deviceID := uuid.New()
	updated := &model.Alarm{
		DeviceSN:        "TEST001",
		DeviceID:        deviceID,
		Carrier:         model.CarrierCMCC,
		AlarmIdentifier: "ALM001",
		Severity:        model.AlarmMajor,
		Description:     "race condition alarm",
		RaisedAt:        time.Now(),
	}

	// No existing alarm — should fallback to Process() and create a new alarm
	err := engine.UpdateByEvent(ctx, updated)
	require.NoError(t, err)

	assert.Len(t, store.active, 1)
	var found *model.Alarm
	for _, a := range store.active {
		if a.AlarmIdentifier == "ALM001" {
			found = a
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "race condition alarm", found.Description)
}
