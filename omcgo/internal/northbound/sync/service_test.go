package sync

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
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

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error { return nil }
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

// T-0162 新接口方法
func (m *mockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	return nil
}
func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ []string, _ []uuid.UUID) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []uuid.UUID) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

type mockAlarmStore struct {
	alarms []model.Alarm
}

func (m *mockAlarmStore) SaveActive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetHistoryByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN, alarmIdentifier string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceSN(_ context.Context, _ string) ([]*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndCode(_ context.Context, _, _ string) (*model.Alarm, error) {
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
func (m *mockAlarmStore) BatchAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchClear(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) HistoryStatistics(_ context.Context, _ alarm.AlarmFilter) (*alarm.AlarmStatistics, error) { return nil, nil }
func (m *mockAlarmStore) BatchUnacknowledge(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) BatchHistoryAcknowledge(_ context.Context, _ []uuid.UUID, _ string, _ string) error { return nil }
func (m *mockAlarmStore) BatchHistoryUnacknowledge(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) BatchHistoryDelete(_ context.Context, _ []uuid.UUID) error { return nil }
func (m *mockAlarmStore) MarkRead(_ context.Context, _ uuid.UUID) error { return nil }

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
func (m *mockCounterRepo) QueryForKPICells(ctx context.Context, deviceID uuid.UUID, cellIDs []string, counterNames []string, startTime, endTime time.Time) (map[string]map[string]float64, error) {
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
func (m *mockParamRepo) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}
func (m *mockParamRepo) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *mockParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}

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
			AlarmIdentifier: "A001",
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
			AlarmIdentifier: "A002",
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
