package pm

import "github.com/prometheus/client_golang/prometheus"

// PMMetrics holds Prometheus metrics for the PM collection module.
type PMMetrics struct {
	FilesProcessedTotal    *prometheus.CounterVec
	ProcessingDurationSecs prometheus.Histogram

	// G4-Gap-1: 上报延迟 ingest_time - end_time（秒）。
	// 标签 carrier × technology 低基数（3 × 2 = 6 组合）；不加 device 避免基数爆炸。
	// 桶覆盖 1 分钟到 24 小时（PM 文件 15 分钟周期，超 1 小时即明显异常）。
	ReportDelaySeconds *prometheus.HistogramVec

	// issue #14: 迟到补传命中 TimescaleDB 压缩 chunk 被降级跳过的文件数。
	// 与存储层 omc_pm_late_arrival_total（按批次计）互补：本指标按 PM 文件计，
	// 标签 carrier × technology 便于定位哪类设备频繁补传历史数据。
	LateArrivalFilesTotal *prometheus.CounterVec
}

// NewPMMetrics creates and registers PM metrics.
func NewPMMetrics(reg prometheus.Registerer) *PMMetrics {
	m := &PMMetrics{
		FilesProcessedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_files_processed_total",
			Help: "Total number of PM files processed by status",
		}, []string{"status"}),
		ProcessingDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_processing_duration_seconds",
			Help:    "Duration of PM file processing in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}),
		ReportDelaySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_pm_report_delay_seconds",
			Help:    "PM file report delay = ingest_time - end_time (seconds). Negative values indicate device clock ahead of OMC.",
			Buckets: []float64{30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 86400},
		}, []string{"carrier", "technology"}),
		LateArrivalFilesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_late_arrival_files_total",
			Help: "PM files skipped because late-arriving data hit a compressed TimescaleDB chunk (UPSERT unsupported).",
		}, []string{"carrier", "technology"}),
	}

	reg.MustRegister(m.FilesProcessedTotal, m.ProcessingDurationSecs, m.ReportDelaySeconds, m.LateArrivalFilesTotal)
	return m
}
