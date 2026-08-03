package pm

import (
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/pm/slothealth"
	"github.com/prometheus/client_golang/prometheus"
)

// PMMetrics holds Prometheus metrics for the PM collection module.
type PMMetrics struct {
	FilesProcessedTotal    *prometheus.CounterVec
	FilesDiscardedTotal    *prometheus.CounterVec
	ProcessingDurationSecs prometheus.Histogram

	// G4-Gap-1: 上报延迟 ingest_time - end_time（秒）。
	// 标签 carrier × technology 低基数（3 × 2 = 6 组合）；不加 device 避免基数爆炸。
	// 桶覆盖 1 分钟到 24 小时（PM 文件 15 分钟周期，超 1 小时即明显异常）。
	ReportDelaySeconds *prometheus.HistogramVec

	// issue #14: 迟到补传命中 TimescaleDB 压缩 chunk 被降级跳过的文件数。
	// 与存储层 omc_pm_late_arrival_total（按批次计）互补：本指标按 PM 文件计，
	// 标签 carrier × technology 便于定位哪类设备频繁补传历史数据。
	LateArrivalFilesTotal *prometheus.CounterVec

	// 厂家上报名未在当前产品指标库登记、因此未入库的值数。
	WhitelistMissValuesTotal *prometheus.CounterVec

	// 已登记但未启用、因此按配置未入库的值数。与 whitelist miss 互斥。
	KnownDisabledValuesTotal *prometheus.CounterVec

	// XML 内容制式与设备元数据不一致、在指标路由前隔离的文件数。
	TechnologyMismatchFilesTotal *prometheus.CounterVec

	IngestLastSuccessTimestamp *prometheus.GaugeVec
	SlotEndTimestamp           *prometheus.GaugeVec
	SlotExpectedDevices        *prometheus.GaugeVec
	SlotReceivedDevices        *prometheus.GaugeVec
	SlotCoverageRatio          *prometheus.GaugeVec
	SlotAlertEligible          *prometheus.GaugeVec
	SlotObserverLastSuccess    prometheus.Gauge
}

// NewPMMetrics creates and registers PM metrics.
func NewPMMetrics(reg prometheus.Registerer) *PMMetrics {
	m := &PMMetrics{
		FilesProcessedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_files_processed_total",
			Help: "Total number of PM files processed by status",
		}, []string{"status"}),
		FilesDiscardedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_files_discarded_total",
			Help: "PM files intentionally discarded before parsing, classified by bounded reason.",
		}, []string{"reason"}),
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
		WhitelistMissValuesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_whitelist_miss_values_total",
			Help: "PM counter values removed because the vendor report key is not registered in the routed indicator library.",
		}, []string{"carrier", "technology"}),
		KnownDisabledValuesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_known_disabled_values_total",
			Help: "PM counter values removed because their registered indicator is disabled.",
		}, []string{"carrier", "technology"}),
		TechnologyMismatchFilesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_technology_mismatch_files_total",
			Help: "PM files quarantined because XML evidence conflicts with the declared device technology.",
		}, []string{"declared_technology", "detected_technology"}),
		IngestLastSuccessTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_ingest_last_success_timestamp_seconds",
			Help: "Unix timestamp of the latest successfully ingested PM file.",
		}, []string{"carrier", "technology"}),
		SlotEndTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_slot_end_timestamp_seconds",
			Help: "Unix timestamp of the latest evaluated PM slot end.",
		}, []string{"carrier", "technology"}),
		SlotExpectedDevices: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_slot_expected_devices",
			Help: "Expected devices for the latest evaluated PM slot.",
		}, []string{"carrier", "technology"}),
		SlotReceivedDevices: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_slot_received_devices",
			Help: "Devices successfully ingested for the latest evaluated PM slot.",
		}, []string{"carrier", "technology"}),
		SlotCoverageRatio: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_slot_coverage_ratio",
			Help: "Received divided by expected devices for the latest evaluated PM slot.",
		}, []string{"carrier", "technology"}),
		SlotAlertEligible: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_slot_alert_eligible",
			Help: "Whether the latest evaluated PM slot is eligible for coverage alerts.",
		}, []string{"carrier", "technology"}),
		SlotObserverLastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_slot_observer_last_success_timestamp_seconds",
			Help: "Unix timestamp of the latest successful PM slot observation.",
		}),
	}

	reg.MustRegister(
		m.FilesProcessedTotal,
		m.FilesDiscardedTotal,
		m.ProcessingDurationSecs,
		m.ReportDelaySeconds,
		m.LateArrivalFilesTotal,
		m.WhitelistMissValuesTotal,
		m.KnownDisabledValuesTotal,
		m.TechnologyMismatchFilesTotal,
		m.IngestLastSuccessTimestamp,
		m.SlotEndTimestamp,
		m.SlotExpectedDevices,
		m.SlotReceivedDevices,
		m.SlotCoverageRatio,
		m.SlotAlertEligible,
		m.SlotObserverLastSuccess,
	)
	return m
}

func (m *PMMetrics) RecordIngestSuccess(carrier, technology string, at time.Time) {
	if m == nil {
		return
	}
	m.IngestLastSuccessTimestamp.WithLabelValues(
		strings.ToLower(strings.TrimSpace(carrier)),
		strings.ToLower(strings.TrimSpace(technology)),
	).Set(float64(at.Unix()))
}

func (m *PMMetrics) PublishSlotHealth(snapshots []slothealth.Snapshot) {
	if m == nil {
		return
	}
	m.SlotEndTimestamp.Reset()
	m.SlotExpectedDevices.Reset()
	m.SlotReceivedDevices.Reset()
	m.SlotCoverageRatio.Reset()
	m.SlotAlertEligible.Reset()
	observedAt := time.Now().UTC()
	if len(snapshots) > 0 {
		observedAt = snapshots[0].EvaluatedAt
	}
	for _, snapshot := range snapshots {
		labels := []string{snapshot.Carrier, snapshot.Technology}
		m.SlotEndTimestamp.WithLabelValues(labels...).Set(float64(snapshot.SlotEnd.Unix()))
		m.SlotExpectedDevices.WithLabelValues(labels...).Set(float64(snapshot.ExpectedDevices))
		m.SlotReceivedDevices.WithLabelValues(labels...).Set(float64(snapshot.ReceivedDevices))
		m.SlotCoverageRatio.WithLabelValues(labels...).Set(snapshot.CoverageRatio)
		eligible := 1.0
		if snapshot.Status == slothealth.StatusBootstrapIgnored {
			eligible = 0
		}
		m.SlotAlertEligible.WithLabelValues(labels...).Set(eligible)
		if snapshot.EvaluatedAt.After(observedAt) {
			observedAt = snapshot.EvaluatedAt
		}
	}
	m.SlotObserverLastSuccess.Set(float64(observedAt.Unix()))
}
