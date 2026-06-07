package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: statusReconcilerRepo（Pattern B — function fields）
// ---------------------------------------------------------------------------

type mockReconcilerRepo struct {
	findFn func(ctx context.Context, minStaleSec int, limit int) ([]*model.Device, error)
	markFn func(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error)

	// 记录调用,用于断言。
	markCalls []markCall
}

type markCall struct {
	DeviceID uuid.UUID
	Reason   string
	Now      time.Time
}

func (m *mockReconcilerRepo) FindStaleDevicesAdaptive(ctx context.Context, minStaleSec, limit int) ([]*model.Device, error) {
	if m.findFn == nil {
		return nil, nil
	}
	return m.findFn(ctx, minStaleSec, limit)
}

func (m *mockReconcilerRepo) MarkOfflineWithAccounting(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error) {
	m.markCalls = append(m.markCalls, markCall{DeviceID: deviceID, Reason: reason, Now: now})
	if m.markFn == nil {
		return true, nil
	}
	return m.markFn(ctx, deviceID, reason, now)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestReconciler(t *testing.T, repo *mockReconcilerRepo, bus event.EventBus) (*DeviceStatusReconciler, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	r := NewDeviceStatusReconciler(rdb, repo, bus, zap.NewNop())
	return r, mr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewDeviceStatusReconciler_Defaults(t *testing.T) {
	r, _ := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	require.NotNil(t, r)
	assert.Equal(t, 5*time.Minute, r.checkInterval) // 连接状态检查周期：每 5 分钟
	assert.Equal(t, 600, r.minStaleSec)
	assert.Equal(t, 1000, r.batchSize)
}

func TestRefreshHeartbeat_TTLFromInterval(t *testing.T) {
	r, mr := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	r.RefreshHeartbeat(context.Background(), "SN-HB-001", 300)

	key := redisx.Keys.ACSHeartbeat("SN-HB-001")
	assert.True(t, mr.Exists(key), "heartbeat key must exist after refresh")
	assert.Equal(t, 600*time.Second, mr.TTL(key), "TTL should be 2 × inform_interval")
}

func TestRefreshHeartbeat_MinTTLClamp(t *testing.T) {
	r, mr := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	// 10×2 = 20s 远小于 60s,应钳到 600s。
	r.RefreshHeartbeat(context.Background(), "SN-HB-MIN", 10)

	key := redisx.Keys.ACSHeartbeat("SN-HB-MIN")
	require.True(t, mr.Exists(key))
	assert.Equal(t, 600*time.Second, mr.TTL(key))
}

func TestRefreshHeartbeat_NilRedisIsNoOp(t *testing.T) {
	r := &DeviceStatusReconciler{logger: zap.NewNop()}
	assert.NotPanics(t, func() {
		r.RefreshHeartbeat(context.Background(), "SN-NIL", 300)
	})
}

func TestDetect_MarksStaleDevicesOffline(t *testing.T) {
	stale := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN-STALE",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	}
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int) ([]*model.Device, error) {
			return []*model.Device{stale}, nil
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	r.detect(context.Background())

	require.Len(t, repo.markCalls, 1)
	assert.Equal(t, stale.ID, repo.markCalls[0].DeviceID)
	assert.Equal(t, OfflineReasonHeartbeatTimeout, repo.markCalls[0].Reason)
}

func TestDetect_NoStaleDevicesIsNoOp(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int) ([]*model.Device, error) {
			return nil, nil
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	assert.NotPanics(t, func() { r.detect(context.Background()) })
	assert.Empty(t, repo.markCalls)
}

func TestDetect_RepoErrorIsLoggedNotPanic(t *testing.T) {
	repo := &mockReconcilerRepo{
		findFn: func(_ context.Context, _ int, _ int) ([]*model.Device, error) {
			return nil, errors.New("db down")
		},
	}

	r, _ := newTestReconciler(t, repo, nil)
	assert.NotPanics(t, func() { r.detect(context.Background()) })
	assert.Empty(t, repo.markCalls, "should not attempt to mark when find fails")
}

func TestMarkOffline_PublishesEventOnTransition(t *testing.T) {
	stale := &model.Device{
		ID: uuid.New(), SerialNumber: "SN-EVT", Carrier: model.CarrierCMCC, Technology: model.TechLTE,
	}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return true, nil // 真翻转
		},
	}

	bus := event.NewChannelEventBus(64, zap.NewNop())
	received := make(chan event.Event, 1)
	_, err := bus.QueueSubscribe(event.SubjectDeviceOffline, "test-sub",
		func(_ context.Context, evt event.Event) error {
			received <- evt
			return nil
		})
	require.NoError(t, err)

	r, _ := newTestReconciler(t, repo, bus)
	transitioned, err := r.markOffline(context.Background(), stale)
	require.NoError(t, err)
	assert.True(t, transitioned)

	select {
	case evt := <-received:
		assert.Equal(t, event.SubjectDeviceOffline, evt.Subject)
		var payload DeviceOfflineEvent
		require.NoError(t, evt.DecodePayload(&payload))
		assert.Equal(t, stale.ID, payload.DeviceID)
		assert.Equal(t, OfflineReasonHeartbeatTimeout, payload.Reason)
	case <-time.After(1 * time.Second):
		t.Fatal("expected device.offline event not received")
	}
}

func TestMarkOffline_NoEventWhenAlreadyOffline(t *testing.T) {
	stale := &model.Device{ID: uuid.New(), SerialNumber: "SN-IDEM"}
	repo := &mockReconcilerRepo{
		markFn: func(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
			return false, nil // 幂等:已离线
		},
	}

	bus := event.NewChannelEventBus(64, zap.NewNop())
	received := make(chan event.Event, 1)
	_, err := bus.QueueSubscribe(event.SubjectDeviceOffline, "test-sub-idem",
		func(_ context.Context, evt event.Event) error {
			received <- evt
			return nil
		})
	require.NoError(t, err)

	r, _ := newTestReconciler(t, repo, bus)
	transitioned, err := r.markOffline(context.Background(), stale)
	require.NoError(t, err)
	assert.False(t, transitioned)

	select {
	case <-received:
		t.Fatal("should NOT publish event when device was already offline")
	case <-time.After(100 * time.Millisecond):
		// ok - no event published
	}
}

func TestStartStop_GracefulShutdown(t *testing.T) {
	repo := &mockReconcilerRepo{}
	r, _ := newTestReconciler(t, repo, nil)
	r.SetCheckInterval(50 * time.Millisecond)

	r.Start()
	time.Sleep(120 * time.Millisecond) // 让 detect 至少跑两轮
	r.Stop()

	// Stop 后再 Stop 不应 panic。
	assert.NotPanics(t, func() { r.Stop() })
}

func TestStartStop_StopBeforeStartIsNoop(t *testing.T) {
	r := &DeviceStatusReconciler{logger: zap.NewNop()}
	assert.NotPanics(t, func() { r.Stop() })
}

func TestSetters_GuardAgainstNonPositive(t *testing.T) {
	r, _ := newTestReconciler(t, &mockReconcilerRepo{}, nil)

	r.SetCheckInterval(0)
	assert.Equal(t, 5*time.Minute, r.checkInterval, "0 should be ignored")

	r.SetCheckInterval(-5 * time.Second)
	assert.Equal(t, 5*time.Minute, r.checkInterval, "negative should be ignored")

	r.SetCheckInterval(10 * time.Second)
	assert.Equal(t, 10*time.Second, r.checkInterval)

	r.SetMinStaleSec(0)
	assert.Equal(t, 600, r.minStaleSec)

	r.SetMinStaleSec(300)
	assert.Equal(t, 300, r.minStaleSec)

	r.SetBatchSize(0)
	assert.Equal(t, 1000, r.batchSize)

	r.SetBatchSize(50)
	assert.Equal(t, 50, r.batchSize)
}
