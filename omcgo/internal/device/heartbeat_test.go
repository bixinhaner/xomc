package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: DeviceRepository (hbMock prefix)
// ---------------------------------------------------------------------------

type hbMockDeviceRepo struct {
	createFn                 func(ctx context.Context, device *model.Device) error
	getByIDFn                func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn      func(ctx context.Context, sn string) (*model.Device, error)
	updateFn                 func(ctx context.Context, device *model.Device) error
	deleteFn                 func(ctx context.Context, id uuid.UUID) error
	listFn                   func(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	updateStatusFn           func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	updateLastInformFn       func(ctx context.Context, sn string, at time.Time, events []string) error
	countByStatusFn          func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
	listActiveByLastInformFn func(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error)
}

func (m *hbMockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	if m.createFn != nil {
		return m.createFn(ctx, d)
	}
	return nil
}
func (m *hbMockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *hbMockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySerialNumberFn != nil {
		return m.getBySerialNumberFn(ctx, sn)
	}
	return nil, nil
}
func (m *hbMockDeviceRepo) Update(ctx context.Context, d *model.Device) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, d)
	}
	return nil
}
func (m *hbMockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *hbMockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *hbMockDeviceRepo) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}
func (m *hbMockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}
func (m *hbMockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *hbMockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.updateLastInformFn != nil {
		return m.updateLastInformFn(ctx, sn, at, events)
	}
	return nil
}
func (m *hbMockDeviceRepo) CountByStatus(ctx context.Context, c *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, c)
	}
	return map[model.DeviceStatus]int64{}, nil
}
func (m *hbMockDeviceRepo) ListActiveByLastInform(ctx context.Context, cursorTime *time.Time, cursorID *uuid.UUID, limit int) ([]model.Device, error) {
	if m.listActiveByLastInformFn != nil {
		return m.listActiveByLastInformFn(ctx, cursorTime, cursorID, limit)
	}
	return []model.Device{}, nil
}
func (m *hbMockDeviceRepo) ListGeo(_ context.Context, _ GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *hbMockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*GeoStats, error) {
	return &GeoStats{}, nil
}
func (m *hbMockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]GeoDevice, error) {
	return nil, nil
}
func (m *hbMockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *hbMockDeviceRepo) ListRecycleBin(_ context.Context, _ RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *hbMockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *hbMockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newHBTestMonitor(t *testing.T, repo *hbMockDeviceRepo) (*HeartbeatMonitor, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewHeartbeatMonitor(rdb, repo, zap.NewNop()), mr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewHeartbeatMonitor(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := &hbMockDeviceRepo{}
	hm := NewHeartbeatMonitor(rdb, repo, zap.NewNop())

	require.NotNil(t, hm)
	assert.Nil(t, hm.cron, "cron should be nil before Start")
}

func TestRefreshHeartbeat_Success(t *testing.T) {
	hm, mr := newHBTestMonitor(t, &hbMockDeviceRepo{})

	hm.RefreshHeartbeat(context.Background(), "SN-HB-001", 300)

	key := heartbeatKeyPrefix + "SN-HB-001"
	assert.True(t, mr.Exists(key))
	assert.Equal(t, 600*time.Second, mr.TTL(key))
}

func TestRefreshHeartbeat_MinTTL(t *testing.T) {
	hm, mr := newHBTestMonitor(t, &hbMockDeviceRepo{})

	// 10*2=20s < 60s → should clamp to 600s
	hm.RefreshHeartbeat(context.Background(), "SN-HB-MIN", 10)

	key := heartbeatKeyPrefix + "SN-HB-MIN"
	assert.True(t, mr.Exists(key))
	assert.Equal(t, 600*time.Second, mr.TTL(key))
}

func TestCheckHeartbeats_MarkOffline(t *testing.T) {
	deviceID := uuid.New()
	var markedStatus model.DeviceStatus

	repo := &hbMockDeviceRepo{
		listActiveByLastInformFn: func(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
			return []model.Device{{ID: deviceID, SerialNumber: "SN-EXPIRED", Status: model.DeviceActive}}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status model.DeviceStatus) error {
			markedStatus = status
			return nil
		},
	}

	hm, _ := newHBTestMonitor(t, repo)
	hm.CheckHeartbeats(context.Background())

	assert.Equal(t, model.DeviceOffline, markedStatus)
}

func TestCheckHeartbeats_HeartbeatExists(t *testing.T) {
	updateCalled := false
	repo := &hbMockDeviceRepo{
		listActiveByLastInformFn: func(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
			return []model.Device{{ID: uuid.New(), SerialNumber: "SN-ALIVE", Status: model.DeviceActive}}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
			updateCalled = true
			return nil
		},
	}

	hm, mr := newHBTestMonitor(t, repo)
	mr.Set(heartbeatKeyPrefix+"SN-ALIVE", "1")

	hm.CheckHeartbeats(context.Background())
	assert.False(t, updateCalled, "should not mark device offline when heartbeat exists")
}

func TestCheckHeartbeats_RepoError(t *testing.T) {
	repo := &hbMockDeviceRepo{
		listActiveByLastInformFn: func(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
			return nil, errors.New("db down")
		},
	}

	hm, _ := newHBTestMonitor(t, repo)
	assert.NotPanics(t, func() {
		hm.CheckHeartbeats(context.Background())
	})
}

func TestHeartbeatMonitor_StartStop(t *testing.T) {
	hm, _ := newHBTestMonitor(t, &hbMockDeviceRepo{})

	hm.Start()
	require.NotNil(t, hm.cron)
	assert.GreaterOrEqual(t, len(hm.cron.Entries()), 1)

	hm.Stop()
}

func TestHeartbeatMonitor_StopBeforeStart(t *testing.T) {
	hm, _ := newHBTestMonitor(t, &hbMockDeviceRepo{})
	assert.NotPanics(t, func() { hm.Stop() })
}
