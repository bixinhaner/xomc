package asyncjob

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 是 G8 通用任务框架的 Prometheus 指标。
//
// 实施 plan：docs/project/plan-T-0164-followup-gaps.md G8-Gap-4
//
// 注册：worker 启动期 NewMetrics(reg) 并通过 Registry.SetMetrics / Sweeper.SetMetrics 注入。
type Metrics struct {
	// QueueDepth 当前 pending / running 任务数（按 job_type + status 分桶，gauge）。
	// Registry 周期性扫 async_jobs 表更新。
	QueueDepth *prometheus.GaugeVec

	// Duration 单任务耗时（按 job_type）。
	Duration *prometheus.HistogramVec

	// Failed 失败任务计数（按 job_type）。
	Failed *prometheus.CounterVec

	// Zombie 僵尸任务计数（按 job_type）。Sweeper 触发 ResetZombie 时 inc。
	Zombie *prometheus.CounterVec

	// Catchup 启动期补跑次数（按 job_type）。worker/aggregator.go 调用。
	Catchup *prometheus.CounterVec
}

// NewMetrics 构造并注册指标。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		QueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_async_jobs_queue_depth",
			Help: "Current async_jobs queue depth by job_type and status",
		}, []string{"job_type", "status"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_async_jobs_duration_seconds",
			Help:    "Async job execution duration by job_type",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 300, 600, 1800},
		}, []string{"job_type"}),
		Failed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_async_jobs_failed_total",
			Help: "Total async jobs failed by job_type",
		}, []string{"job_type"}),
		Zombie: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_async_jobs_zombie_total",
			Help: "Total async jobs reset from zombie state by job_type",
		}, []string{"job_type"}),
		Catchup: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_async_jobs_catchup_total",
			Help: "Total async jobs enqueued by startup catchup by job_type",
		}, []string{"job_type"}),
	}
	if reg != nil {
		reg.MustRegister(m.QueueDepth, m.Duration, m.Failed, m.Zombie, m.Catchup)
	}
	return m
}

// IncFailed 记一次失败。
func (m *Metrics) IncFailed(jobType string) {
	if m == nil {
		return
	}
	m.Failed.WithLabelValues(jobType).Inc()
}

// IncZombie Sweeper reset zombie 时调。
func (m *Metrics) IncZombie(jobType string) {
	if m == nil {
		return
	}
	m.Zombie.WithLabelValues(jobType).Inc()
}

// IncCatchup 启动补跑 enqueue 时调。
func (m *Metrics) IncCatchup(jobType string) {
	if m == nil {
		return
	}
	m.Catchup.WithLabelValues(jobType).Inc()
}

// ObserveDuration 任务完成时记耗时。
func (m *Metrics) ObserveDuration(jobType string, seconds float64) {
	if m == nil {
		return
	}
	m.Duration.WithLabelValues(jobType).Observe(seconds)
}

// SetQueueDepth 设单 (job_type, status) 队列深度。
func (m *Metrics) SetQueueDepth(jobType, status string, n int) {
	if m == nil {
		return
	}
	m.QueueDepth.WithLabelValues(jobType, status).Set(float64(n))
}

// ── 队列深度采集 ─────────────────────────────────────────────────────────

// QueueDepthSampler 定期扫 async_jobs 表更新 QueueDepth gauge。
//
// repo 必须提供 CountByJobTypeAndStatus 方法（Repository 接口需扩）。
// 为不破坏现有 Repository 契约，sampler 用接口收窄注入。
type QueueDepthSampler interface {
	CountByJobTypeAndStatus(ctx context.Context) (map[string]map[string]int, error)
}
