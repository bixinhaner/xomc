package stream

import "github.com/prometheus/client_golang/prometheus"

// Metrics exposes low-cardinality operational signals for the streaming
// aggregation path. Task, device and metric identifiers are deliberately not
// labels because a production system can have tens of thousands of each.
type Metrics struct {
	Ready                                  prometheus.Gauge
	OutboxPublishedTotal                   prometheus.Counter
	OutboxErrorsTotal                      prometheus.Counter
	RollupOutboxPublishedTotal             prometheus.Counter
	RollupOutboxErrorsTotal                prometheus.Counter
	EventsProcessedTotal                   prometheus.Counter
	EventsFailedTotal                      prometheus.Counter
	DuplicateEventsTotal                   prometheus.Counter
	LateEventsTotal                        prometheus.Counter
	WatermarkBlockedTotal                  prometheus.Counter
	RebuildsTotal                          *prometheus.CounterVec
	RebuildErrorsTotal                     prometheus.Counter
	RebuildBatchesTotal                    prometheus.Counter
	RebuildJobsPerBatch                    prometheus.Histogram
	RebuildSnapshotRowsTotal               prometheus.Counter
	RebuildSnapshotPagesTotal              prometheus.Counter
	RebuildSnapshotScanSeconds             prometheus.Histogram
	RebuildCoalescedTotal                  prometheus.Counter
	WindowsFinalizedTotal                  *prometheus.CounterVec
	WindowsPreparedTotal                   prometheus.Counter
	WindowsPublishedTotal                  prometheus.Counter
	PublicationDuration                    prometheus.Histogram
	FinalizeErrorsTotal                    prometheus.Counter
	FinalizeDuration                       prometheus.Histogram
	FinalizeClaims                         prometheus.Counter
	FinalizeInflight                       prometheus.Gauge
	FinalizeOldestDueSeconds               prometheus.Gauge
	FinalizeClaimConflictsTotal            prometheus.Counter
	DailyVersionExpectedSlotsMismatchTotal prometheus.Counter
	ResultReplaceSeconds                   prometheus.Histogram
	RedisSampledActiveWindows              *prometheus.GaugeVec
	RedisSampledKeys                       *prometheus.GaugeVec
	RedisSampledEstimatedBytes             *prometheus.GaugeVec
	RedisSweeperDeletedTotal               prometheus.Counter
	RedisWriteErrorsTotal                  prometheus.Counter
	RuntimeCleanupRowsTotal                *prometheus.CounterVec
	RuntimeCleanupErrorsTotal              *prometheus.CounterVec
	RuntimeCleanupDuration                 *prometheus.HistogramVec
	RuntimeCleanupBacklog                  *prometheus.GaugeVec
	BuiltinReconcileRunsTotal              prometheus.Counter
	BuiltinReconcileErrorsTotal            prometheus.Counter
	BuiltinVersionsChangedTotal            prometheus.Counter
	BuiltinDefinitionsEmpty                prometheus.Gauge
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
		RebuildBatchesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuild_batches_total",
			Help: "Quiet-period rebuild batches claimed for processing.",
		}),
		RebuildJobsPerBatch: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_rebuild_jobs_per_batch",
			Help:    "Number of coalesced rebuild jobs processed in one batch.",
			Buckets: prometheus.ExponentialBuckets(1, 2, 8),
		}),
		RebuildSnapshotRowsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuild_snapshot_rows_total",
			Help: "Published rollup rows read by repeatable-read rebuild scans.",
		}),
		RebuildSnapshotPagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuild_snapshot_pages_total",
			Help: "Keyset pages containing published rollups read by repeatable-read rebuild scans.",
		}),
		RebuildSnapshotScanSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_rebuild_snapshot_scan_seconds",
			Help:    "Time spent scanning published rollups with repeatable-read keyset pagination for a rebuild batch.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 30, 60},
		}),
		RebuildCoalescedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_rebuild_coalesced_total",
			Help: "Additional late-event generations coalesced into claimed rebuild jobs.",
		}),
		WindowsFinalizedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_windows_finalized_total",
			Help: "Aggregation windows finalized by close reason.",
		}, []string{"reason"}),
		WindowsPreparedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_windows_prepared_total",
			Help: "Initial hourly aggregation windows prepared before the publication watermark.",
		}),
		WindowsPublishedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_windows_published_total",
			Help: "Prepared hourly aggregation windows made visible by an atomic publication switch.",
		}),
		PublicationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_publication_duration_seconds",
			Help:    "Time spent atomically publishing ready hourly revisions.",
			Buckets: prometheus.DefBuckets,
		}),
		FinalizeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_finalize_errors_total",
			Help: "Aggregation window finalization failures.",
		}),
		FinalizeDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_finalize_duration_seconds",
			Help:    "Time spent finalizing one aggregation window.",
			Buckets: prometheus.DefBuckets,
		}),
		FinalizeClaims: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_finalize_claims_total",
			Help: "Aggregation windows leased by the bounded finalization scheduler.",
		}),
		FinalizeInflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_finalize_inflight",
			Help: "Leased aggregation windows currently being finalized.",
		}),
		FinalizeOldestDueSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_finalize_oldest_due_seconds",
			Help: "Current age past close grace of the oldest due aggregation window observed by scheduling class.",
		}),
		FinalizeClaimConflictsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_finalize_claim_conflicts_total",
			Help: "Finalize claim attempts blocked by an unexpired lease.",
		}),
		DailyVersionExpectedSlotsMismatchTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_daily_version_expected_slots_mismatch_total",
			Help: "Finalized rule daily windows whose version-calibrated expected slots differ from the natural daily period.",
		}),
		ResultReplaceSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_result_replace_seconds",
			Help:    "Time spent staging and upserting one PM aggregation result window in TimescaleDB.",
			Buckets: prometheus.DefBuckets,
		}),
		RedisSampledActiveWindows: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_redis_sampled_active_windows",
			Help: "Active Redis aggregation windows observed in the latest bounded sample.",
		}, []string{"granularity"}),
		RedisSampledKeys: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_redis_sampled_keys",
			Help: "Redis aggregation keys observed in the latest bounded sample.",
		}, []string{"granularity"}),
		RedisSampledEstimatedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_redis_sampled_estimated_bytes",
			Help: "Redis MEMORY USAGE bytes observed in the latest bounded aggregation-state sample.",
		}, []string{"granularity"}),
		RedisSweeperDeletedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_redis_sweeper_deleted_total",
			Help: "Published Redis aggregation windows safely removed by the bounded sweeper.",
		}),
		RedisWriteErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_redis_write_errors_total",
			Help: "Redis aggregation state write or UNLINK failures.",
		}),
		RuntimeCleanupRowsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_runtime_cleanup_rows_total",
			Help: "Rows or replay chunks removed by bounded PM runtime cleanup.",
		}, []string{"target"}),
		RuntimeCleanupErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_aggregation_runtime_cleanup_errors_total",
			Help: "Bounded PM runtime cleanup failures by target.",
		}, []string{"target"}),
		RuntimeCleanupDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_pm_aggregation_runtime_cleanup_duration_seconds",
			Help:    "Time spent in one bounded PM runtime cleanup batch.",
			Buckets: prometheus.DefBuckets,
		}, []string{"target"}),
		RuntimeCleanupBacklog: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_pm_aggregation_runtime_cleanup_backlog",
			Help: "Capped sample of rows or replay chunks remaining for PM runtime cleanup.",
		}, []string{"target"}),
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
		m.RebuildBatchesTotal,
		m.RebuildJobsPerBatch,
		m.RebuildSnapshotRowsTotal,
		m.RebuildSnapshotPagesTotal,
		m.RebuildSnapshotScanSeconds,
		m.RebuildCoalescedTotal,
		m.WindowsFinalizedTotal,
		m.WindowsPreparedTotal,
		m.WindowsPublishedTotal,
		m.PublicationDuration,
		m.FinalizeErrorsTotal,
		m.FinalizeDuration,
		m.FinalizeClaims,
		m.FinalizeInflight,
		m.FinalizeOldestDueSeconds,
		m.FinalizeClaimConflictsTotal,
		m.DailyVersionExpectedSlotsMismatchTotal,
		m.ResultReplaceSeconds,
		m.RedisSampledActiveWindows,
		m.RedisSampledKeys,
		m.RedisSampledEstimatedBytes,
		m.RedisSweeperDeletedTotal,
		m.RedisWriteErrorsTotal,
		m.RuntimeCleanupRowsTotal,
		m.RuntimeCleanupErrorsTotal,
		m.RuntimeCleanupDuration,
		m.RuntimeCleanupBacklog,
		m.BuiltinReconcileRunsTotal,
		m.BuiltinReconcileErrorsTotal,
		m.BuiltinVersionsChangedTotal,
		m.BuiltinDefinitionsEmpty,
	)
	return m
}
