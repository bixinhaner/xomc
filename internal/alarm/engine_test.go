package alarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockAlarmStore implements AlarmStore for testing.
type mockAlarmStore struct {
	active  map[uuid.UUID]*model.Alarm
	history []*model.Alarm
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
		return nil, fmt.Errorf("not found")
	}
	return a, nil
}

func (m *mockAlarmStore) GetActiveByDeviceAndCode(_ context.Context, deviceSN, alarmCode string) (*model.Alarm, error) {
	for _, a := range m.active {
		if a.DeviceSN == deviceSN && a.AlarmCode == alarmCode && a.Status != model.AlarmCleared {
			return a, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockAlarmStore) UpdateActive(_ context.Context, alarm *model.Alarm) error {
	m.active[alarm.ID] = alarm
	return nil
}

func (m *mockAlarmStore) RemoveActive(_ context.Context, id uuid.UUID) error {
	delete(m.active, id)
	return nil
}

func (m *mockAlarmStore) ListActive(_ context.Context, _ AlarmFilter) (*model.ListResponse[model.Alarm], error) {
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

func (m *mockAlarmStore) ListHistory(_ context.Context, _ AlarmFilter) (*model.ListResponse[model.Alarm], error) {
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

func newTestEngine(store AlarmStore) *AlarmEngine {
	return &AlarmEngine{
		store:  store,
		logger: zap.NewNop(),
	}
}

func TestProcessNewAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmCode: "ALM001",
		AlarmType: "equipment",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}

	err := engine.Process(context.Background(), alarm)
	require.NoError(t, err)

	assert.Len(t, store.active, 1)
	assert.Equal(t, model.AlarmActive, alarm.Status)
	assert.NotEqual(t, uuid.Nil, alarm.ID)
}

func TestProcessDuplicateAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm1 := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmCode: "ALM001",
		AlarmType: "equipment",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now().Add(-10 * time.Minute),
	}
	require.NoError(t, engine.Process(ctx, alarm1))

	alarm2 := &model.Alarm{
		DeviceSN:    "TEST001",
		DeviceID:    alarm1.DeviceID,
		Carrier:     model.CarrierCMCC,
		AlarmCode:   "ALM001",
		AlarmType:   "equipment",
		Severity:    model.AlarmCritical,
		RaisedAt:    time.Now(),
		Description: "updated description",
	}
	require.NoError(t, engine.Process(ctx, alarm2))

	// Should still have only 1 alarm (dedup via DB fallback)
	assert.Len(t, store.active, 1)

	// The existing alarm should have updated fields
	for _, a := range store.active {
		assert.Equal(t, model.AlarmCritical, a.Severity)
		assert.Equal(t, "updated description", a.Description)
	}
}

func TestAcknowledgeAlarm(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmCode: "ALM001",
		Severity:  model.AlarmMajor,
		RaisedAt:  time.Now(),
	}
	require.NoError(t, engine.Process(ctx, alarm))

	err := engine.Acknowledge(ctx, alarm.ID, "admin@test.com")
	require.NoError(t, err)

	stored := store.active[alarm.ID]
	assert.Equal(t, model.AlarmAcknowledged, stored.Status)
	assert.Equal(t, "admin@test.com", stored.AcknowledgedBy)
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
		AlarmCode: "ALM001",
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
		AlarmCode: "ALM001",
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

func TestAlarmFullLifecycle(t *testing.T) {
	store := newMockAlarmStore()
	engine := newTestEngine(store)
	ctx := context.Background()

	alarm := &model.Alarm{
		DeviceSN:  "TEST001",
		DeviceID:  uuid.New(),
		Carrier:   model.CarrierCMCC,
		AlarmCode: "ALM001",
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
			AlarmCode: code,
			Severity:  model.AlarmMajor,
			RaisedAt:  time.Now(),
		}
		require.NoError(t, engine.Process(ctx, alarm))
	}

	assert.Len(t, store.active, 3)
}
