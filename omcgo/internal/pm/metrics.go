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

	// issue #20: PM 文件解析后被指标库白名单丢弃的孤儿 counter 数。
	// 此前 collector 仅 log Info「filtered orphan counters」，无可观测信号——厂家
	// 上报名漂移 / 指标库未注册导致大批 counter 被静默丢弃时是运维盲区。reason 标签：
	//   - "whitelist_miss" 命中白名单但 report_key 未注册（厂家上报名不在指标库）
	// 持续增长说明某产品的指标库注册缺失或厂家上报名变更，需补库或纠正 report_key。
	DroppedCountersTotal *prometheus.CounterVec
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
		DroppedCountersTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_dropped_counters_total",
			Help: "PM counters dropped after parsing, by reason (e.g. whitelist_miss = report_key not registered in indicator library).",
		}, []string{"carrier", "technology", "reason"}),
	}

	reg.MustRegister(m.FilesProcessedTotal, m.ProcessingDurationSecs, m.ReportDelaySeconds, m.LateArrivalFilesTotal, m.DroppedCountersTotal)
	return m
}
