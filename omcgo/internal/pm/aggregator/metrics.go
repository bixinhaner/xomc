package aggregator

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

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

	// BatchDevices records bounded hourly batch completion and device throughput.
	BatchDevices *prometheus.CounterVec

	// HourlyTxRetries and HourlyTxRetryExhausted expose retried and finally
	// failed hourly batch transactions, grouped by PostgreSQL SQLSTATE.
	HourlyTxRetries        *prometheus.CounterVec
	HourlyTxRetryExhausted *prometheus.CounterVec

	HourlyRecoveries         prometheus.Counter
	RecoveryExhaustedBuckets prometheus.Gauge
	StaleBuildingVersions    prometheus.Gauge
	FailedBuckets            prometheus.Gauge
	AgedFailedBuckets        prometheus.Gauge
	FailedVersions           prometheus.Gauge
	WatermarkLag             prometheus.Gauge

	DirtyBuckets        prometheus.Gauge
	WatermarkTimestamp  prometheus.Gauge
	TempBytes           prometheus.Gauge
	SparseAmplification prometheus.Gauge
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
		BatchDevices: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregator_batch_total",
			Help: "Completed PM rollup batches and devices by job_type and kind",
		}, []string{"job_type", "kind"}),
		HourlyTxRetries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_hourly_tx_retries_total",
			Help: "Retried hourly PM batch transactions by PostgreSQL SQLSTATE",
		}, []string{"sqlstate"}),
		HourlyTxRetryExhausted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_hourly_tx_retry_exhausted_total",
			Help: "Hourly PM batch transactions exhausting retries by PostgreSQL SQLSTATE",
		}, []string{"sqlstate"}),
		HourlyRecoveries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_hourly_recoveries_total",
			Help: "Terminal hourly PM bucket jobs atomically requeued by maintenance",
		}),
		RecoveryExhaustedBuckets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_recovery_exhausted_buckets",
			Help: "Recent failed hourly PM buckets that exhausted bounded recovery",
		}),
		StaleBuildingVersions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_stale_building_versions",
			Help: "Hourly PM versions still building beyond the maintenance timeout",
		}),
		FailedBuckets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_failed_buckets",
			Help: "Recent natural hourly PM jobs in failed state",
		}),
		AgedFailedBuckets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_aged_failed_buckets",
			Help: "Recent natural hourly PM jobs failed longer than one maintenance interval",
		}),
		FailedVersions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_failed_versions",
			Help: "Hourly PM bucket versions in failed state",
		}),
		WatermarkLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_watermark_lag_seconds",
			Help: "Seconds since the newest clean active hourly PM bucket ended",
		}),
		DirtyBuckets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_dirty_buckets",
			Help: "Number of active or building hourly buckets marked dirty",
		}),
		WatermarkTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_hourly_watermark_timestamp_seconds",
			Help: "Unix timestamp of the newest clean active hourly bucket end",
		}),
		TempBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_tsdb_temp_bytes_total",
			Help: "Cumulative temporary bytes reported by the PM TimescaleDB",
		}),
		SparseAmplification: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_sparse_logical_to_physical_ratio",
			Help: "Recent logical PM rows divided by sparse physical value rows",
		}),
	}
	if reg != nil {
		reg.MustRegister(
			m.Runs, m.Duration, m.RowsWritten, m.BucketLag, m.BatchDevices,
			m.HourlyTxRetries, m.HourlyTxRetryExhausted,
			m.HourlyRecoveries, m.RecoveryExhaustedBuckets,
			m.StaleBuildingVersions, m.FailedBuckets, m.AgedFailedBuckets,
			m.FailedVersions, m.WatermarkLag,
			m.DirtyBuckets, m.WatermarkTimestamp, m.TempBytes, m.SparseAmplification,
		)
	}
	return m
}

func (m *Metrics) SetDirtyBuckets(n float64) {
	if m != nil {
		m.DirtyBuckets.Set(n)
	}
}

func (m *Metrics) SetWatermark(unixSeconds int64) {
	if m != nil {
		m.WatermarkTimestamp.Set(float64(unixSeconds))
	}
}

func (m *Metrics) SetTempBytes(n float64) {
	if m != nil {
		m.TempBytes.Set(n)
	}
}

func (m *Metrics) SetSparseAmplification(n float64) {
	if m != nil {
		m.SparseAmplification.Set(n)
	}
}

// AddBatch records completed bounded batches and their device count.
func (m *Metrics) AddBatch(jobType string, batches, devices int) {
	if m == nil {
		return
	}
	if batches > 0 {
		m.BatchDevices.WithLabelValues(jobType, "batch").Add(float64(batches))
	}
	if devices > 0 {
		m.BatchDevices.WithLabelValues(jobType, "device").Add(float64(devices))
	}
}

func (m *Metrics) IncHourlyTxRetry(sqlState string) {
	if m != nil {
		m.HourlyTxRetries.WithLabelValues(sqlState).Inc()
	}
}

func (m *Metrics) IncHourlyTxRetryExhausted(sqlState string) {
	if m != nil {
		m.HourlyTxRetryExhausted.WithLabelValues(sqlState).Inc()
	}
}

func (m *Metrics) IncHourlyRecovery() {
	if m != nil {
		m.HourlyRecoveries.Inc()
	}
}

func (m *Metrics) SetRecoveryExhaustedBuckets(n float64) {
	if m != nil {
		m.RecoveryExhaustedBuckets.Set(n)
	}
}

func (m *Metrics) SetStaleBuildingVersions(n float64) {
	if m != nil {
		m.StaleBuildingVersions.Set(n)
	}
}

func (m *Metrics) SetFailedBuckets(n float64) {
	if m != nil {
		m.FailedBuckets.Set(n)
	}
}

func (m *Metrics) SetAgedFailedBuckets(n float64) {
	if m != nil {
		m.AgedFailedBuckets.Set(n)
	}
}

func (m *Metrics) SetFailedVersions(n float64) {
	if m != nil {
		m.FailedVersions.Set(n)
	}
}

func (m *Metrics) SetWatermarkLag(lag time.Duration) {
	if m != nil {
		m.WatermarkLag.Set(lag.Seconds())
	}
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
