package agentaction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

type fakeDeviceService struct {
	devices       []model.Device
	params        []model.DeviceParameter
	listFilter    device.DeviceFilter
	authorizedIDs []uuid.UUID
}

func (f *fakeDeviceService) ListDevices(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	f.listFilter = filter
	return model.NewListResponse(f.devices, int64(len(f.devices)), filter.Page, filter.PageSize), nil
}

func (f *fakeDeviceService) GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	for _, dev := range f.devices {
		if dev.ID == id {
			return &dev, nil
		}
	}
	return nil, nil
}

func (f *fakeDeviceService) GetDeviceParameters(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return f.params, nil
}

func (f *fakeDeviceService) AuthorizeDeviceGroupAccess(ctx context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error {
	f.authorizedIDs = append(f.authorizedIDs, deviceID)
	return nil
}

type fakeAlarmStore struct {
	filter alarm.AlarmFilter
	stats  *alarm.AlarmStatistics
	items  []model.Alarm
}

func (f *fakeAlarmStore) ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	f.filter = filter
	return model.NewListResponse(f.items, int64(len(f.items)), filter.Page, filter.PageSize), nil
}

func (f *fakeAlarmStore) Statistics(ctx context.Context, filter alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	f.filter = filter
	return f.stats, nil
}

func TestServiceExecuteDeviceSearchPassesVisibleGroups(t *testing.T) {
	groupID := uuid.New()
	deviceID := uuid.New()
	fakeDevices := &fakeDeviceService{
		devices: []model.Device{{
			ID:             deviceID,
			SerialNumber:   "SN-001",
			DeviceName:     "Site A",
			ProductClass:   "pico",
			Technology:     model.TechLTE,
			LifecycleState: model.LifecycleCommissioned,
			IsOnline:       true,
			IPAddress:      "10.0.0.1",
		}},
	}
	svc := NewService(fakeDevices, nil, zap.NewNop())

	result, err := svc.Execute(context.Background(), ActionRequest{
		ActionID: ActionDeviceSearch,
		Input: map[string]any{
			"query": "SN-001",
			"limit": float64(5),
		},
	}, RequestContext{VisibleGroups: []uuid.UUID{groupID}})

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{groupID}, fakeDevices.listFilter.VisibleGroups)
	assert.Equal(t, 5, fakeDevices.listFilter.PageSize)
	assert.Equal(t, ActionDeviceSearch, result.ActionID)
	assert.Equal(t, "ok", result.Status)
}

func TestServiceExecuteDeviceSummaryUsesAuthAndSamplesParameters(t *testing.T) {
	deviceID := uuid.New()
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	fakeDevices := &fakeDeviceService{
		devices: []model.Device{{
			ID:             deviceID,
			SerialNumber:   "SN-002",
			DeviceName:     "Site B",
			Technology:     model.TechNR,
			LifecycleState: model.LifecycleCommissioned,
		}},
		params: []model.DeviceParameter{{
			DeviceID:       deviceID,
			ParameterPath:  "Device.DeviceInfo.SoftwareVersion",
			ParameterValue: "1.0.0",
			Writable:       false,
			LastUpdatedAt:  now,
		}},
	}
	svc := NewService(fakeDevices, nil, zap.NewNop())

	result, err := svc.Execute(context.Background(), ActionRequest{
		ActionID: ActionDeviceSummary,
		Input:    map[string]any{"id": deviceID.String()},
	}, RequestContext{VisibleGroups: []uuid.UUID{uuid.New()}})

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{deviceID}, fakeDevices.authorizedIDs)
	summary, ok := result.Result.(DeviceSummary)
	require.True(t, ok)
	assert.Equal(t, "SN-002", summary.SerialNumber)
	require.Len(t, summary.ParameterSample, 1)
	assert.Equal(t, "Device.DeviceInfo.SoftwareVersion", summary.ParameterSample[0].Path)
}

func TestServiceExecuteAlarmSummaryAppliesVisibleGroups(t *testing.T) {
	groupID := uuid.New()
	store := &fakeAlarmStore{
		stats: &alarm.AlarmStatistics{
			TotalActive: 1,
			BySeverity:  map[model.AlarmSeverity]int64{model.AlarmCritical: 1},
		},
		items: []model.Alarm{{
			ID:              uuid.New(),
			DeviceID:        uuid.New(),
			DeviceSN:        "SN-003",
			Severity:        model.AlarmCritical,
			AlarmIdentifier: "ALM-001",
			Description:     "offline",
			RaisedAt:        time.Date(2026, 7, 6, 11, 0, 0, 0, time.UTC),
		}},
	}
	svc := NewService(nil, store, zap.NewNop())

	result, err := svc.Execute(context.Background(), ActionRequest{
		ActionID: ActionAlarmActiveSummary,
		Input:    map[string]any{"limit": float64(3)},
	}, RequestContext{VisibleGroups: []uuid.UUID{groupID}})

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{groupID}, store.filter.VisibleGroups)
	assert.Equal(t, 3, store.filter.PageSize)
	assert.Equal(t, "ok", result.Status)
}
