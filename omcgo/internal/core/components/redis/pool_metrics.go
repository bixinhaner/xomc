package redis

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// DefaultPoolMetricsInterval 默认采样周期。
const DefaultPoolMetricsInterval = 5 * time.Second

// poolSnapshot 是 Redis 连接池一次采样的结果。
//
// 来源：go-redis v9 的 *redis.PoolStats（Hits/Misses/Timeouts/TotalConns/IdleConns/StaleConns 等）。
// 这里只保留章程关心的三类核心指标 + max。
type poolSnapshot struct {
	InUse int32 // 使用中（Total - Idle）
	Idle  int32 // 空闲
	Max   int32 // 配置上限（PoolSize）
}

// poolSampler 在每次轮询时返回一个连接池快照。
type poolSampler func() poolSnapshot

// PoolMetrics 持有 go-redis 连接池的 Prometheus 指标集合。
//
// 暴露的指标（章程 W3.F.3 / T-0061）：
//   - redis_pool_in_use (gauge) 使用中连接数
//   - redis_pool_idle   (gauge) 空闲连接数
//   - redis_pool_max    (gauge) 最大连接数（PoolSize）
type PoolMetrics struct {
	InUse prometheus.Gauge
	Idle  prometheus.Gauge
	Max   prometheus.Gauge

	sampler  poolSampler
	interval time.Duration

	cancel   context.CancelFunc
	stopped  chan struct{}
	stopOnce sync.Once
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

// poolStatsProvider 抽象 go-redis client 的 PoolStats() 方法。
//
// redis.UniversalClient 与 *redis.Client / *redis.ClusterClient 都实现该方法，
// 但因 UniversalClient 是接口、SDK 没有公开统一抽象，这里显式声明最小接口。
type poolStatsProvider interface {
	PoolStats() *redis.PoolStats
}

// RegisterPoolMetrics 为 go-redis 客户端注册连接池指标，并启动后台采样 goroutine。
//
// poolSize 为配置中的 PoolSize（go-redis 默认 10×CPU），如果 ≤ 0 则视为未知，
// max gauge 退化为 TotalConns（Hits+Misses 不参与，避免误导）。
func RegisterPoolMetrics(client poolStatsProvider, poolSize int, reg prometheus.Registerer, opts ...PoolMetricsOption) *PoolMetrics {
	if client == nil {
		panic("redis.RegisterPoolMetrics: client is nil")
	}
	sampler := func() poolSnapshot {
		st := client.PoolStats()
		if st == nil {
			return poolSnapshot{}
		}
		total := int32(st.TotalConns)
		idle := int32(st.IdleConns)
		inUse := total - idle
		if inUse < 0 {
			inUse = 0
		}
		max := int32(poolSize)
		if max <= 0 {
			max = total
		}
		return poolSnapshot{InUse: inUse, Idle: idle, Max: max}
	}
	return registerPoolMetricsWithSampler(sampler, reg, opts...)
}

// registerPoolMetricsWithSampler 接受可注入的快照函数，便于单测构造确定性数据。
func registerPoolMetricsWithSampler(sampler poolSampler, reg prometheus.Registerer, opts ...PoolMetricsOption) *PoolMetrics {
	if sampler == nil {
		panic("redis.registerPoolMetricsWithSampler: sampler is nil")
	}
	if reg == nil {
		panic("redis.RegisterPoolMetrics: registerer is nil")
	}

	pm := &PoolMetrics{
		sampler:  sampler,
		interval: DefaultPoolMetricsInterval,
		stopped:  make(chan struct{}),
		InUse: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_pool_in_use",
			Help: "Number of in-use Redis connections (TotalConns - IdleConns).",
		}),
		Idle: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_pool_idle",
			Help: "Number of idle Redis connections.",
		}),
		Max: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_pool_max",
			Help: "Configured maximum number of Redis connections (PoolSize).",
		}),
	}
	for _, opt := range opts {
		opt(pm)
	}

	reg.MustRegister(pm.InUse, pm.Idle, pm.Max)

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

func (p *PoolMetrics) sample() {
	if p.sampler == nil {
		return
	}
	snap := p.sampler()
	p.InUse.Set(float64(snap.InUse))
	p.Idle.Set(float64(snap.Idle))
	p.Max.Set(float64(snap.Max))
}
