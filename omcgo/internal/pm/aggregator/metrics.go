package aggregator

import "github.com/prometheus/client_golang/prometheus"

// Metrics 是 G5 自然桶聚合 + G7 adhoc 任务的 Prometheus 指标。
//
// 实施 plan：docs/project/plan-T-0164-followup-gaps.md G5-Gap-3
//
// 注册：worker 启动期 NewMetrics(reg) 并通过 Aggregator.SetMetrics 注入，
// 各 Runner.Run 调用 inc / observe（nil 安全）。
type Metrics struct {
	// Runs 是 cron 触发的运行计数（按 job_type + status 分桶）。
	// status: succeeded / failed
	Runs *prometheus.CounterVec

	// Duration 单次 Run 耗时（按 job_type 分桶）。
	Duration *prometheus.HistogramVec

	// RowsWritten 单次 Run 写入聚合表的行数（按 job_type + kind 分桶；kind=counter/kpi/group）。
	RowsWritten *prometheus.CounterVec

	// BucketLag 当前最后一个成功聚合桶距 now 的滞后秒数（按 job_type 分桶，gauge）。
	// 用于监控聚合是否堵塞 — 正常 hourly job 该值应 < 7200s（2 个桶以内）。
	BucketLag *prometheus.GaugeVec
}

// NewMetrics 构造并注册指标。reg nil 时返不注册的 metrics（测试场景）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Runs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregator_runs_total",
			Help: "Total PM aggregator cron runs by job_type and status",
		}, []string{"job_type", "status"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregator_duration_seconds",
			Help:    "PM aggregator Run duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600},
		}, []string{"job_type"}),
		RowsWritten: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregator_rows_written_total",
			Help: "Total rows written by PM aggregator by job_type and kind (counter/kpi/group)",
		}, []string{"job_type", "kind"}),
		BucketLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_aggregator_bucket_lag_seconds",
			Help: "Seconds between last successfully-aggregated bucket end and now (by job_type)",
		}, []string{"job_type"}),
	}
	if reg != nil {
		reg.MustRegister(m.Runs, m.Duration, m.RowsWritten, m.BucketLag)
	}
	return m
}

// IncRun 记一次 Run 结果。status="succeeded" / "failed"。
func (m *Metrics) IncRun(jobType, status string) {
	if m == nil {
		return
	}
	m.Runs.WithLabelValues(jobType, status).Inc()
}

// ObserveDuration 记 Run 耗时（秒）。
func (m *Metrics) ObserveDuration(jobType string, seconds float64) {
	if m == nil {
		return
	}
	m.Duration.WithLabelValues(jobType).Observe(seconds)
}

// AddRows 累加写入行数。kind="counter" / "kpi" / "group"。
func (m *Metrics) AddRows(jobType, kind string, n int) {
	if m == nil || n <= 0 {
		return
	}
	m.RowsWritten.WithLabelValues(jobType, kind).Add(float64(n))
}

// SetBucketLag 设当前 job_type 的桶滞后秒数。
func (m *Metrics) SetBucketLag(jobType string, seconds float64) {
	if m == nil {
		return
	}
	m.BucketLag.WithLabelValues(jobType).Set(seconds)
}
