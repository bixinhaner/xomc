package alarm

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// capturingStore records alarms saved via SaveActive so tests can inspect them.
// It satisfies AlarmStore with minimal behaviour for the RebootMonitor paths.
type capturingStore struct {
	mu     sync.Mutex
	active []*model.Alarm
}

func (s *capturingStore) SaveActive(_ context.Context, a *model.Alarm) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = append(s.active, a)
	return nil
}

func (s *capturingStore) saved() []*model.Alarm {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*model.Alarm, len(s.active))
	copy(out, s.active)
	return out
}

func (s *capturingStore) GetActiveByID(_ context.Context, _ uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (s *capturingStore) GetActiveByDeviceAndCode(_ context.Context, _, _ string) (*model.Alarm, error) {
	return nil, nil
}
func (s *capturingStore) GetActiveByDeviceAndIdentifier(_ context.Context, _, _ string) (*model.Alarm, error) {
	return nil, nil
}
func (s *capturingStore) GetActiveByDeviceSN(_ context.Context, _ string) ([]*model.Alarm, error) {
	return nil, nil
}
func (s *capturingStore) UpdateActive(_ context.Context, _ *model.Alarm) error { return nil }
func (s *capturingStore) RemoveActive(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *capturingStore) ListActive(_ context.Context, _ AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return nil, nil
}
func (s *capturingStore) Archive(_ context.Context, _ *model.Alarm) error { return nil }
func (s *capturingStore) ListHistory(_ context.Context, _ AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return nil, nil
}
func (s *capturingStore) Statistics(_ context.Context, _ AlarmFilter) (*AlarmStatistics, error) {
	return &AlarmStatistics{}, nil
}
func (s *capturingStore) HistoryStatistics(_ context.Context, _ AlarmFilter) (*AlarmStatistics, error) {
	return &AlarmStatistics{}, nil
}
func (s *capturingStore) BatchAcknowledge(_ context.Context, _ []uuid.UUID, _, _ string) error {
	return nil
}
func (s *capturingStore) BatchUnacknowledge(_ context.Context, _ []uuid.UUID) error { return nil }
func (s *capturingStore) BatchClear(_ context.Context, _ []uuid.UUID, _, _ string) error {
	return nil
}
func (s *capturingStore) BatchHistoryAcknowledge(_ context.Context, _ []uuid.UUID, _, _ string) error {
	return nil
}
func (s *capturingStore) BatchHistoryUnacknowledge(_ context.Context, _ []uuid.UUID) error {
	return nil
}
func (s *capturingStore) BatchHistoryDelete(_ context.Context, _ []uuid.UUID) error { return nil }
func (s *capturingStore) MarkRead(_ context.Context, _ uuid.UUID) error             { return nil }

func newMonitorHarness(t *testing.T, opts ...RebootMonitorOption) (*RebootMonitor, *capturingStore, redis.UniversalClient) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := &capturingStore{}
	engine := &AlarmEngine{store: store, logger: zap.NewNop()}
	monitor := NewRebootMonitor(engine, client, zap.NewNop(), opts...)
	return monitor, store, client
}

func makePayload(t *testing.T, sn string) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectDeviceRebootAbnormal, RebootAbnormalPayload{
		DeviceID:     uuid.NewString(),
		SerialNumber: sn,
		Carrier:      "cmcc",
		BootCount:    1,
		LastBootAt:   time.Now(),
		Events:       []string{"1 BOOT"},
	})
	require.NoError(t, err)
	return evt
}

func TestRebootMonitor_BelowThreshold_NoAlarm(t *testing.T) {
	monitor, store, _ := newMonitorHarness(t,
		WithRebootWindow(time.Minute),
		WithRebootThreshold(3))

	ctx := context.Background()
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-A")))
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-A")))

	assert.Empty(t, store.saved(), "two events with threshold=3 should not raise alarm")
}

func TestRebootMonitor_AtThreshold_RaisesAlarm(t *testing.T) {
	monitor, store, _ := newMonitorHarness(t,
		WithRebootWindow(time.Minute),
		WithRebootThreshold(3))

	ctx := context.Background()
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-B")))
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-B")))
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-B")))

	saved := store.saved()
	require.Len(t, saved, 1)
	assert.Equal(t, AlarmCodeFrequentReboot, saved[0].AlarmIdentifier)
	assert.Equal(t, "SN-B", saved[0].DeviceSN)
	assert.Equal(t, model.AlarmMajor, saved[0].Severity)
}

func TestRebootMonitor_SlidingWindow_EvictsOldEntries(t *testing.T) {
	monitor, store, client := newMonitorHarness(t,
		WithRebootWindow(50*time.Millisecond),
		WithRebootThreshold(3))

	ctx := context.Background()
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-C")))
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-C")))

	// Wait long enough for the first two events to fall outside the window.
	time.Sleep(60 * time.Millisecond)
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-C")))

	assert.Empty(t, store.saved(), "earlier events should be evicted; only 1 event in window")

	// Verify window size is 1 (just the latest entry)
	count, err := client.ZCard(ctx, rebootWindowKey("SN-C")).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRebootMonitor_PerDeviceIsolation(t *testing.T) {
	monitor, store, _ := newMonitorHarness(t,
		WithRebootWindow(time.Minute),
		WithRebootThreshold(2))

	ctx := context.Background()
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-X")))
	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-Y")))
	// SN-X only got 1 event so far, shouldn't alarm
	assert.Empty(t, store.saved())

	require.NoError(t, monitor.handle(ctx, makePayload(t, "SN-X")))
	saved := store.saved()
	require.Len(t, saved, 1)
	assert.Equal(t, "SN-X", saved[0].DeviceSN)
}
