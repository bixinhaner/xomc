package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// DefaultPoolMetricsInterval 是默认的 pool 指标采样周期。
const DefaultPoolMetricsInterval = 5 * time.Second

// poolSnapshot 是 pgxpool.Stat 的可复制快照，用于解耦采样源与 Prometheus 指标。
// 抽出快照后，单元测试可以直接构造，不再依赖只能由 pgxpool 内部生成的 *pgxpool.Stat。
type poolSnapshot struct {
	InUse        int32
	Idle         int32
	Max          int32
	AcquireCount int64
}

// poolSampler 在每次轮询时返回一个连接池快照。
type poolSampler func() poolSnapshot

// PoolMetrics 持有 pgxpool 连接池的 Prometheus 指标集合。
//
// 暴露的指标（章程 W3.F.3 / T-0061）：
//   - pgxpool_in_use        (gauge)   使用中连接数
//   - pgxpool_idle          (gauge)   空闲连接数
//   - pgxpool_max           (gauge)   最大连接数（来自 pool.Config().MaxConns）
//   - pgxpool_acquire_total (counter) 累计 Acquire 次数
//
// 指标在 RegisterPoolMetrics 内一次性注册到 prometheus.Registerer，
// 然后通过后台 goroutine 周期性轮询 pool.Stat() 更新 gauge。
type PoolMetrics struct {
	InUse        prometheus.Gauge
	Idle         prometheus.Gauge
	Max          prometheus.Gauge
	AcquireTotal prometheus.Counter

	sampler  poolSampler
	interval time.Duration

	cancel    context.CancelFunc
	stopped   chan struct{}
	stopOnce  sync.Once
	lastTotal int64
}

// PoolMetricsOption 配置 PoolMetrics 行为。
type PoolMetricsOption func(*PoolMetrics)

// WithPoolMetricsInterval 自定义采样周期，默认 5s。
func WithPoolMetricsInterval(interval time.Duration) PoolMetricsOption {
	return func(p *PoolMetrics) {
		if interval > 0 {
			p.interval = interval
		}
	}
}

// RegisterPoolMetrics 为 pgxpool.Pool 注册 Prometheus 指标，并启动后台
// 采样 goroutine。返回的 *PoolMetrics 可用于停止采样（调用 Stop()），
// 适合在进程优雅关闭时调用。
//
// pool 不能为 nil；reg 不能为 nil；重复注册会 panic（prometheus 默认行为）。
func RegisterPoolMetrics(pool *pgxpool.Pool, reg prometheus.Registerer, opts ...PoolMetricsOption) *PoolMetrics {
	if pool == nil {
		panic("postgres.RegisterPoolMetrics: pool is nil")
	}
	sampler := func() poolSnapshot {
		stat := pool.Stat()
		return poolSnapshot{
			InUse:        stat.AcquiredConns(),
			Idle:         stat.IdleConns(),
			Max:          stat.MaxConns(),
			AcquireCount: stat.AcquireCount(),
		}
	}
	return registerPoolMetricsWithSampler(sampler, reg, opts...)
}

// registerPoolMetricsWithSampler 接受可注入的快照函数，便于单测构造确定性数据。
func registerPoolMetricsWithSampler(sampler poolSampler, reg prometheus.Registerer, opts ...PoolMetricsOption) *PoolMetrics {
	if sampler == nil {
		panic("postgres.registerPoolMetricsWithSampler: sampler is nil")
	}
	if reg == nil {
		panic("postgres.RegisterPoolMetrics: registerer is nil")
	}

	pm := &PoolMetrics{
		sampler:  sampler,
		interval: DefaultPoolMetricsInterval,
		stopped:  make(chan struct{}),
		InUse: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pgxpool_in_use",
			Help: "Number of in-use (acquired) PostgreSQL connections in the pgxpool.",
		}),
		Idle: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pgxpool_idle",
			Help: "Number of idle PostgreSQL connections in the pgxpool.",
		}),
		Max: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pgxpool_max",
			Help: "Configured maximum number of PostgreSQL connections in the pgxpool.",
		}),
		AcquireTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "pgxpool_acquire_total",
			Help: "Cumulative number of successful Acquire operations on the pgxpool.",
		}),
	}
	for _, opt := range opts {
		opt(pm)
	}

	reg.MustRegister(pm.InUse, pm.Idle, pm.Max, pm.AcquireTotal)

	// 立即采样一次，避免暴露默认 0 值导致瞬时误报。
	pm.sample()

	ctx, cancel := context.WithCancel(context.Background())
	pm.cancel = cancel
	go pm.run(ctx)

	return pm
}

// Stop 停止后台采样并等待 goroutine 退出。可重复调用。
func (p *PoolMetrics) Stop() {
	p.stopOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
		<-p.stopped
	})
}

func (p *PoolMetrics) run(ctx context.Context) {
	defer close(p.stopped)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.sample()
		}
	}
}

// sample 读取一次 pool 快照并刷新 gauge / counter。
//
// AcquireTotal 是单调递增 counter；pgxpool.Stat().AcquireCount() 本身就是
// 进程内累计值，因此这里以 delta 方式 Add 到 prom counter，避免覆盖。
func (p *PoolMetrics) sample() {
	if p.sampler == nil {
		return
	}
	snap := p.sampler()

	p.InUse.Set(float64(snap.InUse))
	p.Idle.Set(float64(snap.Idle))
	p.Max.Set(float64(snap.Max))

	current := snap.AcquireCount
	if current > p.lastTotal {
		p.AcquireTotal.Add(float64(current - p.lastTotal))
		p.lastTotal = current
	} else if current < p.lastTotal {
		// 极端情况：pool 重建导致计数器重置；以当前值重置 baseline。
		p.lastTotal = current
	}
}
