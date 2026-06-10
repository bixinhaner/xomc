package task

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// issue #12 — wakeDevice 有界并发派生的回归测试。
//
// 历史 bug：wakeDevice 每次 CreateTask / BatchCreateTasks 去重设备都无界 `go func()`，
// 10 万 Inform 风暴下 goroutine 暴涨 → 峰值 OOM。修复后唤醒 goroutine 受
// wakeSemaphore 有界控制，满载丢弃（背压）而非无界堆积。

// blockingLookup 让 GetConnectionRequestURL 阻塞在 release 上，
// 使被放行的唤醒 goroutine 保持"在飞"，便于观测峰值并发。
type blockingLookup struct {
	inFlight atomic.Int64 // 当前进入 lookup 的 goroutine 数
	peak     atomic.Int64 // 观测到的并发峰值
	entered  chan struct{}
	release  chan struct{}
}

func newBlockingLookup(bufferedEnter int) *blockingLookup {
	return &blockingLookup{
		entered: make(chan struct{}, bufferedEnter),
		release: make(chan struct{}),
	}
}

func (b *blockingLookup) GetConnectionRequestURL(_ context.Context, _ string) (string, error) {
	cur := b.inFlight.Add(1)
	// 记录峰值（CAS 循环避免覆盖更高值）
	for {
		p := b.peak.Load()
		if cur <= p || b.peak.CompareAndSwap(p, cur) {
			break
		}
	}
	b.entered <- struct{}{}
	<-b.release // 阻塞直到测试放行
	b.inFlight.Add(-1)
	return "http://cpe/connreq", nil
}

// noopSender 计数发送次数。
type noopSender struct{ sent atomic.Int64 }

func (n *noopSender) Send(_ context.Context, _, _ string) error {
	n.sent.Add(1)
	return nil
}

// TestWakeDevice_ConcurrencyBounded 验证并发在飞唤醒不超过配置上界，
// 超过上界的唤醒被背压丢弃（不阻塞调用方、不无界堆积 goroutine）。
func TestWakeDevice_ConcurrencyBounded(t *testing.T) {
	const limit = 4
	const fired = 50

	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)

	lookup := newBlockingLookup(fired)
	sender := &noopSender{}

	svc := &TaskService{logger: zap.NewNop()}
	svc.SetMetrics(metrics)
	svc.SetConnectionRequester(lookup, sender)
	svc.SetWakeConcurrency(limit)

	// 连发 50 次唤醒。最多 limit 个能拿到信号量进入 lookup 并阻塞；
	// 其余应被 TryAcquire 背压丢弃，wakeDevice 立即返回（不阻塞本循环）。
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < fired; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			svc.wakeDevice("SN-BURST")
		}()
	}
	close(start)
	wg.Wait() // 所有 wakeDevice 调用必须返回（证明未阻塞调用方）——超时即死锁

	// 给放行的 goroutine 进入 lookup 一点时间稳定
	require.Eventually(t, func() bool {
		return lookup.inFlight.Load() == int64(limit)
	}, time.Second, 5*time.Millisecond, "应恰有 limit 个唤醒在飞")

	// 峰值并发不得超过上界（核心断言：有界生效）
	assert.LessOrEqual(t, lookup.peak.Load(), int64(limit),
		"并发在飞唤醒不应超过配置上界 %d", limit)

	// 背压丢弃次数 = fired - limit（满载后剩余全丢）
	dropped := readCounter(t, metrics.WakeDropped)
	assert.Equal(t, float64(fired-limit), dropped,
		"超过上界的唤醒应被背压丢弃并计数")

	// 放行在飞的 goroutine，确认它们正常完成并释放信号量
	close(lookup.release)
	require.Eventually(t, func() bool {
		return lookup.inFlight.Load() == 0
	}, time.Second, 5*time.Millisecond)
	assert.Equal(t, int64(limit), sender.sent.Load(),
		"被放行的 limit 个唤醒应各发一次 Connection Request")
}

// TestWakeDevice_DefaultConcurrencyLazyInit 验证未显式配置时走安全默认上界，
// 且信号量懒初始化（既有 `&TaskService{...}` 字面量构造零改动仍受保护）。
func TestWakeDevice_DefaultConcurrencyLazyInit(t *testing.T) {
	svc := &TaskService{logger: zap.NewNop()}
	// 未调 SetWakeConcurrency —— 首次 wakeSemaphore() 应建出默认上界信号量。
	sem := svc.wakeSemaphore()
	require.NotNil(t, sem)
	// 默认上界下可一次性拿满 defaultWakeConcurrency 个，再多一个失败。
	for i := 0; i < defaultWakeConcurrency; i++ {
		require.True(t, sem.TryAcquire(1), "默认上界内应可获取")
	}
	assert.False(t, sem.TryAcquire(1), "超过默认上界应被拒绝")
}

// TestWakeDevice_ReleasedAfterCompletion 验证唤醒完成后信号量被释放，
// 不会因 fire-and-forget 泄漏配额（否则跑久了所有唤醒都被误丢）。
func TestWakeDevice_ReleasedAfterCompletion(t *testing.T) {
	const limit = 2
	reg := prometheus.NewRegistry()
	metrics := NewTaskMetrics(reg)

	sender := &noopSender{}
	svc := &TaskService{logger: zap.NewNop()}
	svc.SetMetrics(metrics)
	// 用非阻塞 lookup：唤醒瞬间完成并释放信号量。
	svc.SetConnectionRequester(&fakeDeviceLookup{url: "http://cpe/cr"}, sender)
	svc.SetWakeConcurrency(limit)

	// 串行连发远多于 limit 次；因每次都很快完成释放，配额应反复复用、不耗尽。
	const rounds = 20
	for i := 0; i < rounds; i++ {
		svc.wakeDevice("SN-SERIAL")
		// 等本次完成（释放信号量）后再发下一次
		require.Eventually(t, func() bool {
			return svc.wakeSemaphore().TryAcquire(int64(limit))
		}, time.Second, 2*time.Millisecond, "信号量应在唤醒完成后被释放")
		svc.wakeSemaphore().Release(int64(limit))
	}

	require.Eventually(t, func() bool {
		return sender.sent.Load() == int64(rounds)
	}, 2*time.Second, 5*time.Millisecond)
	// 全程无丢弃（配额始终因释放而充足）
	assert.Equal(t, float64(0), readCounter(t, metrics.WakeDropped))
}

// readCounter 读取 prometheus.Counter 当前值。
func readCounter(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}
