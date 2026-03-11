package sync

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type mockDeviceRepo struct {
	devices []model.Device
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error   { return nil }
func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error { return nil }
func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error    { return nil }
func (m *mockDeviceRepo) List(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse(m.devices, int64(len(m.devices)), 1, 100), nil
}
func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	return nil
}
func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}

type mockAlarmStore struct {
	alarms []model.Alarm
}

func (m *mockAlarmStore) SaveActive(ctx context.Context, a *model.Alarm) error   { return nil }
func (m *mockAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndCode(ctx context.Context, deviceSN, alarmCode string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) UpdateActive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) RemoveActive(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *mockAlarmStore) ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse(m.alarms, int64(len(m.alarms)), 1, 100), nil
}
func (m *mockAlarmStore) Archive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) ListHistory(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 100), nil
}
func (m *mockAlarmStore) Statistics(ctx context.Context, filter alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	return &alarm.AlarmStatistics{}, nil
}

type mockCounterRepo struct {
	counters []model.PMCounter
}

func (m *mockCounterRepo) BatchInsert(ctx context.Context, counters []model.PMCounter) error {
	return nil
}
func (m *mockCounterRepo) Query(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	return model.NewListResponse(m.counters, int64(len(m.counters)), 1, 100), nil
}
func (m *mockCounterRepo) QueryAggregated(ctx context.Context, filter counter.CounterFilter) ([]counter.AggregatedCounter, error) {
	return nil, nil
}
func (m *mockCounterRepo) QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error) {
	return nil, nil
}

type mockKPIRepo struct{}

func (m *mockKPIRepo) BatchInsert(ctx context.Context, values []model.KPIValue) error { return nil }
func (m *mockKPIRepo) Query(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	return model.NewListResponse([]model.KPIValue{}, 0, 1, 100), nil
}
func (m *mockKPIRepo) ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error) {
	return nil, nil
}
func (m *mockKPIRepo) SyncDefinitions(ctx context.Context, defs []model.KPIDefinition) error {
	return nil
}

type mockParamRepo struct {
	params []model.DeviceParameter
}

func (m *mockParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	return nil
}
func (m *mockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return m.params, nil
}
func (m *mockParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error { return nil }

// --- Tests ---

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestFullSync_Device(t *testing.T) {
	devices := []model.Device{
		{
			ID:           uuid.New(),
			SerialNumber: "SN001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			Status:       model.DeviceActive,
		},
		{
			ID:           uuid.New(),
			SerialNumber: "SN002",
			Carrier:      model.CarrierCTCC,
			Technology:   model.TechNR,
			Status:       model.DeviceRegistered,
		},
	}

	svc := NewService(
		&mockDeviceRepo{devices: devices},
		&mockAlarmStore{},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	result, err := svc.FullSync(context.Background(), "device")
	require.NoError(t, err)
	assert.Equal(t, "device", result.DataType)
	assert.Equal(t, 2, result.Total)
	assert.False(t, result.Truncated)
	assert.NotZero(t, result.SyncedAt)
}

func TestFullSync_Alarm(t *testing.T) {
	alarms := []model.Alarm{
		{
			ID:        uuid.New(),
			DeviceSN:  "SN001",
			AlarmCode: "A001",
			Severity:  model.AlarmCritical,
			Status:    model.AlarmActive,
		},
	}

	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{alarms: alarms},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	result, err := svc.FullSync(context.Background(), "alarm")
	require.NoError(t, err)
	assert.Equal(t, "alarm", result.DataType)
	assert.Equal(t, 1, result.Total)
}

func TestFullSync_PM(t *testing.T) {
	counters := []model.PMCounter{
		{
			DeviceID:     uuid.New(),
			CounterGroup: "RRC",
			CounterName:  "rrc.conn.succ",
			CounterValue: 42.0,
		},
	}

	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{},
		&mockCounterRepo{counters: counters},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	result, err := svc.FullSync(context.Background(), "pm")
	require.NoError(t, err)
	assert.Equal(t, "pm", result.DataType)
	assert.Equal(t, 1, result.Total)
}

func TestFullSync_UnsupportedType(t *testing.T) {
	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	_, err := svc.FullSync(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
}

func TestIncrementalSync_Alarm(t *testing.T) {
	alarms := []model.Alarm{
		{
			ID:        uuid.New(),
			DeviceSN:  "SN001",
			AlarmCode: "A002",
			Status:    model.AlarmActive,
		},
	}

	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{alarms: alarms},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	since := time.Now().Add(-1 * time.Hour)
	result, err := svc.IncrementalSync(context.Background(), "alarm", since)
	require.NoError(t, err)
	assert.Equal(t, "alarm", result.DataType)
	assert.Equal(t, 1, result.Total)
}

func TestIncrementalSync_PM(t *testing.T) {
	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	since := time.Now().Add(-30 * time.Minute)
	result, err := svc.IncrementalSync(context.Background(), "pm", since)
	require.NoError(t, err)
	assert.Equal(t, "pm", result.DataType)
	assert.Equal(t, 0, result.Total)
}

func TestIncrementalSync_UnsupportedType(t *testing.T) {
	svc := NewService(
		&mockDeviceRepo{},
		&mockAlarmStore{},
		&mockCounterRepo{},
		&mockKPIRepo{},
		&mockParamRepo{},
		testLogger(),
	)

	_, err := svc.IncrementalSync(context.Background(), "config", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
}
