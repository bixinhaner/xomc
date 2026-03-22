package northbound

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
)

// --- Mock implementations ---

type mockAlarmStore struct {
	alarm.AlarmStore
	listActiveFunc func(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error)
}

func (m *mockAlarmStore) ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return m.listActiveFunc(ctx, filter)
}

type mockCounterRepo struct {
	counter.CounterRepository
	queryFunc func(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error)
}

func (m *mockCounterRepo) Query(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	return m.queryFunc(ctx, filter)
}

type mockKPIRepo struct {
	kpi.KPIRepository
	queryFunc func(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error)
}

func (m *mockKPIRepo) Query(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	return m.queryFunc(ctx, filter)
}

type mockParamRepo struct {
	device.DeviceParameterRepository
	getByDeviceFunc func(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
}

func (m *mockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return m.getByDeviceFunc(ctx, deviceID)
}

func newTestService(
	alarmStore alarm.AlarmStore,
	counterRepo counter.CounterRepository,
	kpiRepo kpi.KPIRepository,
	paramRepo device.DeviceParameterRepository,
) *NorthboundService {
	return NewNorthboundService(alarmStore, counterRepo, kpiRepo, paramRepo, nil, nil, zap.NewNop())
}

// --- Tests ---

func TestExportAlarms_Success(t *testing.T) {
	expected := model.NewListResponse([]model.Alarm{
		{ID: uuid.New(), DeviceSN: "SN001"},
		{ID: uuid.New(), DeviceSN: "SN002"},
	}, 2, 1, 100)

	store := &mockAlarmStore{
		listActiveFunc: func(_ context.Context, _ alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
			return expected, nil
		},
	}

	svc := newTestService(store, nil, nil, nil)
	filter := alarm.AlarmFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportAlarms(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
}

func TestExportAlarms_Error(t *testing.T) {
	store := &mockAlarmStore{
		listActiveFunc: func(_ context.Context, _ alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}

	svc := newTestService(store, nil, nil, nil)
	filter := alarm.AlarmFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportAlarms(context.Background(), filter)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export alarms")
}

func TestListActiveAlarms_Success(t *testing.T) {
	expected := model.NewListResponse([]model.Alarm{
		{ID: uuid.New(), DeviceSN: "SN001"},
	}, 1, 1, 20)

	store := &mockAlarmStore{
		listActiveFunc: func(_ context.Context, f alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
			assert.Equal(t, 1, f.ListRequest.Page)
			assert.Equal(t, 20, f.ListRequest.PageSize)
			return expected, nil
		},
	}

	svc := newTestService(store, nil, nil, nil)
	result, err := svc.ListActiveAlarms(context.Background(), model.ListRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

func TestExportConfig_Success(t *testing.T) {
	deviceID := uuid.New()
	expected := []model.DeviceParameter{
		{DeviceID: deviceID, ParameterPath: "Device.ManagementServer.URL", ParameterValue: "http://acs.example.com"},
		{DeviceID: deviceID, ParameterPath: "Device.DeviceInfo.SerialNumber", ParameterValue: "SN001"},
	}

	repo := &mockParamRepo{
		getByDeviceFunc: func(_ context.Context, id uuid.UUID) ([]model.DeviceParameter, error) {
			assert.Equal(t, deviceID, id)
			return expected, nil
		},
	}

	svc := newTestService(nil, nil, nil, repo)
	params, err := svc.ExportConfig(context.Background(), deviceID)
	require.NoError(t, err)
	assert.Len(t, params, 2)
	assert.Equal(t, "Device.ManagementServer.URL", params[0].ParameterPath)
}

func TestExportConfig_Error(t *testing.T) {
	deviceID := uuid.New()
	repo := &mockParamRepo{
		getByDeviceFunc: func(_ context.Context, _ uuid.UUID) ([]model.DeviceParameter, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := newTestService(nil, nil, nil, repo)
	params, err := svc.ExportConfig(context.Background(), deviceID)
	assert.Nil(t, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export config")
}

func TestExportPM_Success(t *testing.T) {
	expected := model.NewListResponse([]model.PMCounter{
		{DeviceID: uuid.New(), CounterName: "RRC.ConnEstabSucc"},
	}, 1, 1, 100)

	repo := &mockCounterRepo{
		queryFunc: func(_ context.Context, _ counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			return expected, nil
		},
	}

	svc := newTestService(nil, repo, nil, nil)
	filter := counter.CounterFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportPM(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "RRC.ConnEstabSucc", result.Items[0].CounterName)
}

func TestExportPM_Error(t *testing.T) {
	repo := &mockCounterRepo{
		queryFunc: func(_ context.Context, _ counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			return nil, fmt.Errorf("query timeout")
		},
	}

	svc := newTestService(nil, repo, nil, nil)
	filter := counter.CounterFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportPM(context.Background(), filter)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export PM")
}

func TestExportKPI_Success(t *testing.T) {
	expected := model.NewListResponse([]model.KPIValue{
		{DeviceID: uuid.New(), KPIName: "RRC_ConnEstabSuccRate", KPIValue: 99.5},
	}, 1, 1, 100)

	repo := &mockKPIRepo{
		queryFunc: func(_ context.Context, _ kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			return expected, nil
		},
	}

	svc := newTestService(nil, nil, repo, nil)
	filter := kpi.KPIFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportKPI(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 99.5, result.Items[0].KPIValue)
}

func TestExportKPI_Error(t *testing.T) {
	repo := &mockKPIRepo{
		queryFunc: func(_ context.Context, _ kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			return nil, fmt.Errorf("kpi store unavailable")
		},
	}

	svc := newTestService(nil, nil, repo, nil)
	filter := kpi.KPIFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 100}}

	result, err := svc.ExportKPI(context.Background(), filter)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export KPI")
}
