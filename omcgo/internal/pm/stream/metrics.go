package stream

import "github.com/prometheus/client_golang/prometheus"

// Metrics exposes low-cardinality operational signals for the streaming
// aggregation path. Task, device and metric identifiers are deliberately not
// labels because a production system can have tens of thousands of each.
type Metrics struct {
	Ready                       prometheus.Gauge
	OutboxPublishedTotal        prometheus.Counter
	OutboxErrorsTotal           prometheus.Counter
	RollupOutboxPublishedTotal  prometheus.Counter
	RollupOutboxErrorsTotal     prometheus.Counter
	EventsProcessedTotal        prometheus.Counter
	EventsFailedTotal           prometheus.Counter
	DuplicateEventsTotal        prometheus.Counter
	LateEventsTotal             prometheus.Counter
	WatermarkBlockedTotal       prometheus.Counter
	RebuildsTotal               *prometheus.CounterVec
	RebuildErrorsTotal          prometheus.Counter
	WindowsFinalizedTotal       *prometheus.CounterVec
	FinalizeErrorsTotal         prometheus.Counter
	FinalizeDuration            prometheus.Histogram
	BuiltinReconcileRunsTotal   prometheus.Counter
	BuiltinReconcileErrorsTotal prometheus.Counter
	BuiltinVersionsChangedTotal prometheus.Counter
	BuiltinDefinitionsEmpty     prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Ready: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_ready",
			Help: "Whether the PM streaming aggregation subsystem passed its startup checks.",
		}),
		OutboxPublishedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_outbox_published_total",
			Help: "Normalized PM events successfully published from the transactional outbox.",
		}),
		OutboxErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_outbox_errors_total",
			Help: "Transactional PM outbox publish failures.",
		}),
		RollupOutboxPublishedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rollup_outbox_published_total",
			Help: "Compact hourly and daily Counter rollups published from the transactional outbox.",
		}),
		RollupOutboxErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rollup_outbox_errors_total",
			Help: "Compact Counter rollup outbox publish failures.",
		}),
		EventsProcessedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_events_processed_total",
			Help: "Normalized PM events fully processed and eligible for acknowledgement.",
		}),
		EventsFailedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_events_failed_total",
			Help: "Normalized PM events that failed before acknowledgement.",
		}),
		DuplicateEventsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_duplicate_events_total",
			Help: "Duplicate source-file contributions ignored by the Redis accumulator.",
		}),
		LateEventsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_late_events_total",
			Help: "Events received after their aggregation window was published and queued for revision rebuild.",
		}),
		WatermarkBlockedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_watermark_blocked_total",
			Help: "Timeout-close candidates held open because source queue events remain unconsumed.",
		}),
		RebuildsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuilds_total",
			Help: "Late-event aggregation rebuilds completed by granularity.",
		}, []string{"granularity"}),
		RebuildErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuild_errors_total",
			Help: "Late-event aggregation rebuild failures.",
		}),
		WindowsFinalizedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_windows_finalized_total",
			Help: "Aggregation windows finalized by close reason.",
		}, []string{"reason"}),
		FinalizeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_finalize_errors_total",
			Help: "Aggregation window finalization failures.",
		}),
		FinalizeDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_finalize_duration_seconds",
			Help:    "Time spent finalizing one aggregation window.",
			Buckets: prometheus.DefBuckets,
		}),
		BuiltinReconcileRunsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_builtin_reconcile_runs_total",
			Help: "Built-in PM aggregation reconciliation runs.",
		}),
		BuiltinReconcileErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_builtin_reconcile_errors_total",
			Help: "Built-in PM aggregation definitions that failed reconciliation.",
		}),
		BuiltinVersionsChangedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_builtin_versions_changed_total",
			Help: "Immutable built-in PM aggregation versions created after content changes.",
		}),
		BuiltinDefinitionsEmpty: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_builtin_definitions_empty",
			Help: "Built-in PM aggregation definitions currently resolving no members.",
		}),
	}
	reg.MustRegister(
		m.Ready,
		m.OutboxPublishedTotal,
		m.OutboxErrorsTotal,
		m.RollupOutboxPublishedTotal,
		m.RollupOutboxErrorsTotal,
		m.EventsProcessedTotal,
		m.EventsFailedTotal,
		m.DuplicateEventsTotal,
		m.LateEventsTotal,
		m.WatermarkBlockedTotal,
		m.RebuildsTotal,
		m.RebuildErrorsTotal,
		m.WindowsFinalizedTotal,
		m.FinalizeErrorsTotal,
		m.FinalizeDuration,
		m.BuiltinReconcileRunsTotal,
		m.BuiltinReconcileErrorsTotal,
		m.BuiltinVersionsChangedTotal,
		m.BuiltinDefinitionsEmpty,
	)
	return m
}
