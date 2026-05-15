package provision

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type fakeStaleLister struct {
	devices []*model.Device
	err     error
	calls   atomic.Int32
}

func (f *fakeStaleLister) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	f.calls.Add(1)
	return f.devices, f.err
}

type fakeSyncStarter struct {
	mu          sync.Mutex
	calls       []syncCall
	defaultUsed bool
	defaultErr  error
	// Per-device override
	perDevice map[uuid.UUID]syncResult
}

type syncCall struct {
	deviceID uuid.UUID
	sourceID string
	reason   string
}

type syncResult struct {
	used bool
	err  error
}

func newFakeSyncStarter(defaultUsed bool) *fakeSyncStarter {
	return &fakeSyncStarter{defaultUsed: defaultUsed, perDevice: map[uuid.UUID]syncResult{}}
}

func (f *fakeSyncStarter) StartPathBSync(_ context.Context, dev *model.Device, sourceID string, opts ...PathBOption) (bool, error) {
	var pb pathBOptions
	for _, opt := range opts {
		opt(&pb)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, syncCall{deviceID: dev.ID, sourceID: sourceID, reason: pb.reason})
	if r, ok := f.perDevice[dev.ID]; ok {
		return r.used, r.err
	}
	return f.defaultUsed, f.defaultErr
}

func (f *fakeSyncStarter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeLeader struct {
	acquired bool
	err      error
	released atomic.Bool
}

func (f *fakeLeader) TryAcquire(_ context.Context) (bool, error) { return f.acquired, f.err }
func (f *fakeLeader) Release(_ context.Context) error            { f.released.Store(true); return nil }

func mkDevices(n int) []*model.Device {
	out := make([]*model.Device, n)
	for i := 0; i < n; i++ {
		out[i] = &model.Device{ID: uuid.New(), SerialNumber: "SN-" + uuid.New().String()[:8], Status: model.DeviceActive}
	}
	return out
}

// stubLookup 用预设 KV 映射构造 SysConfigLookup（按 (category, key) 二级索引）。
func stubLookup(kv map[string]string) SysConfigLookup {
	return func(_ context.Context, cat, key string) (string, bool) {
		v, ok := kv[cat+"."+key]
		return v, ok
	}
}

// makePolicySnap 把一个 PeriodicSyncSnapshot 反向编码为 SysConfigLookup，
// 让 PeriodicSyncPolicy 在 Snapshot() 时还原出同样的值。
func makePolicySnap(snap PeriodicSyncSnapshot) *PeriodicSyncPolicy {
	kv := map[string]string{
		"device.periodicSyncEnabled":              strconv.FormatBool(snap.Enabled),
		"device.periodicSyncIntervalHours":        strconv.Itoa(int(snap.Interval / time.Hour)),
		"device.periodicSyncBatchSize":            strconv.Itoa(snap.BatchSize),
		"device.periodicSyncMaxConcurrent":        strconv.Itoa(snap.MaxConcurrent),
		"device.periodicSyncStaggerWindowMinutes": strconv.Itoa(int(snap.StaggerWindow / time.Minute)),
	}
	return NewPeriodicSyncPolicy(stubLookup(kv), zap.NewNop())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPeriodicSyncer_Snapshot_DefaultsWhenLookupNil(t *testing.T) {
	p := NewPeriodicSyncPolicy(nil, zap.NewNop())
	snap := p.Snapshot(context.Background())
	assert.False(t, snap.Enabled, "default Enabled=false")
	assert.Equal(t, 24*time.Hour, snap.Interval)
	assert.Equal(t, 200, snap.BatchSize)
	assert.Equal(t, 10, snap.MaxConcurrent)
	assert.Equal(t, time.Duration(0), snap.StaggerWindow)
}

func TestPeriodicSyncer_Snapshot_OverridesFromSysConfig(t *testing.T) {
	kv := map[string]string{
		"device.periodicSyncEnabled":              "true",
		"device.periodicSyncIntervalHours":        "72",
		"device.periodicSyncBatchSize":            "50",
		"device.periodicSyncMaxConcurrent":        "5",
		"device.periodicSyncStaggerWindowMinutes": "5",
	}
	p := NewPeriodicSyncPolicy(stubLookup(kv), zap.NewNop())
	snap := p.Snapshot(context.Background())
	assert.True(t, snap.Enabled)
	assert.Equal(t, 72*time.Hour, snap.Interval)
	assert.Equal(t, 50, snap.BatchSize)
	assert.Equal(t, 5, snap.MaxConcurrent)
	assert.Equal(t, 5*time.Minute, snap.StaggerWindow)
}

func TestPeriodicSyncer_Snapshot_InvalidValuesFallBackToDefault(t *testing.T) {
	// 非数值 / 负值 / 解析失败 → 退化到 default
	kv := map[string]string{
		"device.periodicSyncEnabled":              "yes",  // ParseBool 不接受
		"device.periodicSyncIntervalHours":        "-3",   // 拒绝负值
		"device.periodicSyncBatchSize":            "abc",  // 解析失败
		"device.periodicSyncMaxConcurrent":        "0",    // 拒绝 0
		"device.periodicSyncStaggerWindowMinutes": "-100", // 拒绝负值
	}
	p := NewPeriodicSyncPolicy(stubLookup(kv), zap.NewNop())
	snap := p.Snapshot(context.Background())
	assert.False(t, snap.Enabled, "ParseBool 失败应退化为默认")
	assert.Equal(t, 24*time.Hour, snap.Interval)
	assert.Equal(t, 200, snap.BatchSize)
	assert.Equal(t, 10, snap.MaxConcurrent)
	assert.Equal(t, time.Duration(0), snap.StaggerWindow)
}

func TestPeriodicSyncer_NotLeader_SkipsRun(t *testing.T) {
	lister := &fakeStaleLister{devices: mkDevices(3)}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{acquired: false}

	snap := defaultPeriodicSyncSnapshot()
	snap.Enabled = true
	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), snap)

	assert.Equal(t, int32(0), lister.calls.Load(), "non-leader 应跳过 list")
	assert.Equal(t, 0, syncer.callCount(), "non-leader 应跳过 sync")
}

func TestPeriodicSyncer_LeaderError_SkipsRun(t *testing.T) {
	lister := &fakeStaleLister{}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{err: errors.New("pg pool exhausted")}

	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), defaultPeriodicSyncSnapshot())

	assert.Equal(t, int32(0), lister.calls.Load())
}

func TestPeriodicSyncer_LeaderRunsBatch(t *testing.T) {
	devices := mkDevices(5)
	lister := &fakeStaleLister{devices: devices}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{acquired: true}

	snap := PeriodicSyncSnapshot{
		Enabled:       true,
		Interval:      time.Hour,
		BatchSize:     200,
		MaxConcurrent: 10,
	}
	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), snap)

	assert.Equal(t, int32(1), lister.calls.Load())
	assert.Equal(t, 5, syncer.callCount(), "leader 应入队全部 5 设备")

	// 验证每次调用都带 reason="periodic"
	syncer.mu.Lock()
	defer syncer.mu.Unlock()
	for _, c := range syncer.calls {
		assert.Equal(t, "periodic", c.reason, "reason 应为 periodic")
		// SourceID 是裸 UUID（写入 device_tasks.source_id UUID 列），reason 由 WithReason 独立通道传递
		_, err := uuid.Parse(c.sourceID)
		assert.NoError(t, err, "sourceID 必须是合法 UUID")
	}
}

func TestPeriodicSyncer_NilLeader_RunsBatch(t *testing.T) {
	devices := mkDevices(3)
	lister := &fakeStaleLister{devices: devices}
	syncer := newFakeSyncStarter(true)

	snap := PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 10}
	p := NewPeriodicSyncer(lister, syncer, nil, nil, zap.NewNop()) // nil leader = 单副本
	p.runOnce(context.Background(), snap)

	assert.Equal(t, 3, syncer.callCount(), "nil leader（单副本部署）应直接放行")
}

func TestPeriodicSyncer_EmptyBatch_NoSyncCalls(t *testing.T) {
	lister := &fakeStaleLister{devices: nil}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{acquired: true}

	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 10})

	assert.Equal(t, int32(1), lister.calls.Load(), "应调一次 ListStaleForParamSync")
	assert.Equal(t, 0, syncer.callCount(), "空 batch 不应入队")
}

func TestPeriodicSyncer_StartPathBSyncFailureIsolated(t *testing.T) {
	devices := mkDevices(3)
	lister := &fakeStaleLister{devices: devices}
	syncer := newFakeSyncStarter(true)
	// 第 2 个设备的 sync 失败 — 验证其他设备仍然能跑
	syncer.perDevice[devices[1].ID] = syncResult{used: false, err: errors.New("path-b: enqueue failed")}
	leader := &fakeLeader{acquired: true}

	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 10})

	assert.Equal(t, 3, syncer.callCount(), "单设备失败不应中断 batch，其他设备仍调")
}

func TestPeriodicSyncer_PathBUnavailable_CountsAsSkipped(t *testing.T) {
	devices := mkDevices(3)
	lister := &fakeStaleLister{devices: devices}
	syncer := newFakeSyncStarter(false) // used=false 全部跳过
	leader := &fakeLeader{acquired: true}

	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 10})

	assert.Equal(t, 3, syncer.callCount(), "仍调 StartPathBSync 但内部 used=false → 跳过不算失败")
}

func TestPeriodicSyncer_ListError_NoCrash(t *testing.T) {
	lister := &fakeStaleLister{err: errors.New("db connection lost")}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{acquired: true}

	p := NewPeriodicSyncer(lister, syncer, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 10})

	assert.Equal(t, 0, syncer.callCount(), "list 失败应不入队")
}

func TestPeriodicSyncer_Start_StopsOnCtxCancel(t *testing.T) {
	lister := &fakeStaleLister{}
	syncer := newFakeSyncStarter(true)
	leader := &fakeLeader{acquired: false}
	policy := makePolicySnap(PeriodicSyncSnapshot{
		Enabled:       true,
		Interval:      time.Hour,
		BatchSize:     200,
		MaxConcurrent: 10,
	})
	p := NewPeriodicSyncer(lister, syncer, leader, policy, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.Start(ctx) }()

	time.Sleep(50 * time.Millisecond) // 让 Start 进 select
	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Start should exit within 2s of ctx cancel")
	}
	assert.True(t, leader.released.Load(), "Start 退出时应调 leader.Release")
}

func TestPeriodicSyncer_ConcurrentRequest_RespectsMaxConcurrent(t *testing.T) {
	devices := mkDevices(20)
	lister := &fakeStaleLister{devices: devices}

	// 自定义 syncer 测峰值并发数：维护 inFlight 计数器 + max
	var inFlight, maxInFlight atomic.Int32
	tracking := &trackingSyncStarter{
		onCall: func() {
			cur := inFlight.Add(1)
			for {
				m := maxInFlight.Load()
				if cur > m {
					if maxInFlight.CompareAndSwap(m, cur) {
						break
					}
					continue
				}
				break
			}
			time.Sleep(20 * time.Millisecond) // 让并发窗口稳定
			inFlight.Add(-1)
		},
	}
	leader := &fakeLeader{acquired: true}

	snap := PeriodicSyncSnapshot{Interval: time.Hour, BatchSize: 200, MaxConcurrent: 5}
	p := NewPeriodicSyncer(lister, tracking, leader, nil, zap.NewNop())
	p.runOnce(context.Background(), snap)

	require.LessOrEqual(t, maxInFlight.Load(), int32(5), "并发峰值应 ≤ MaxConcurrent=5")
}

// trackingSyncStarter 计数并发用
type trackingSyncStarter struct {
	onCall func()
}

func (t *trackingSyncStarter) StartPathBSync(_ context.Context, _ *model.Device, _ string, _ ...PathBOption) (bool, error) {
	if t.onCall != nil {
		t.onCall()
	}
	return true, nil
}
