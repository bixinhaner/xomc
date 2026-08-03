package paramsync

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	RequestsTotal             *prometheus.CounterVec
	RunsActive                *prometheus.GaugeVec
	RequestDuration           prometheus.Histogram
	RunDuration               prometheus.Histogram
	TasksExpected             prometheus.Gauge
	TasksTerminal             prometheus.Gauge
	TasksProcessed            prometheus.Gauge
	TasksFailed               prometheus.Gauge
	ResultRedelivery          prometheus.Counter
	ResultCountMode           *prometheus.CounterVec
	FinalizeTotal             *prometheus.CounterVec
	OutboxBacklog             prometheus.Gauge
	StagingRows               prometheus.Gauge
	ReconcileRepairs          *prometheus.CounterVec
	RunsReadyButNotFinalized  prometheus.Gauge
	RunCounterDrift           prometheus.Gauge
	RunCounterDriftOldestIdle prometheus.Gauge
	ActiveRunOldestAge        prometheus.Gauge
	ReconcileFinalized        *prometheus.CounterVec
	ReconcileDuration         prometheus.Histogram
	RunsBlocked               *prometheus.GaugeVec
	RunsBlockedOldestIdle     *prometheus.GaugeVec
	ResultShardQueueDepth     *prometheus.GaugeVec
	ResultShardQueueFull      *prometheus.CounterVec
	ResultShardInflight       *prometheus.GaugeVec
	ResultProcessDuration     prometheus.Histogram
	AutomaticReservedRuns     prometheus.Gauge
	AutomaticQueuedRequests   prometheus.Gauge
	AutomaticOldestQueueAge   prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		RequestsTotal:             prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_requests_total", Help: "Parameter sync requests by scope, reason and status."}, []string{"scope", "reason", "status"}),
		RunsActive:                prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_runs_active", Help: "Active parameter sync runs by status."}, []string{"status"}),
		RequestDuration:           prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_request_duration_seconds", Help: "Parameter sync request duration.", Buckets: prometheus.DefBuckets}),
		RunDuration:               prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_run_duration_seconds", Help: "Parameter sync run duration.", Buckets: prometheus.DefBuckets}),
		TasksExpected:             prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_expected", Help: "Expected tasks in active parameter sync runs."}),
		TasksTerminal:             prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_terminal", Help: "Terminal tasks in active parameter sync runs."}),
		TasksProcessed:            prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_processed", Help: "Processed task results in active parameter sync runs."}),
		TasksFailed:               prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_failed", Help: "Failed task results in active parameter sync runs."}),
		ResultRedelivery:          prometheus.NewCounter(prometheus.CounterOpts{Name: "param_sync_result_redelivery_total", Help: "Duplicate/redelivered parameter sync results."}),
		ResultCountMode:           prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_result_count_total", Help: "Parameter sync result counter updates by incremental or authoritative mode."}, []string{"mode"}),
		FinalizeTotal:             prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_finalize_total", Help: "Parameter sync finalizations by result."}, []string{"result"}),
		OutboxBacklog:             prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_outbox_backlog", Help: "Pending/failed parameter sync outbox rows."}),
		StagingRows:               prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_staging_rows", Help: "Estimated current parameter sync staging rows from PostgreSQL planner statistics."}),
		ReconcileRepairs:          prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_reconcile_repairs_total", Help: "Parameter sync reconciliation repairs by kind."}, []string{"kind"}),
		RunsReadyButNotFinalized:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_runs_ready_but_not_finalized", Help: "Active parameter sync runs whose durable task and result counts are complete but are not finalized."}),
		RunCounterDrift:           prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_run_counter_drift", Help: "Active parameter sync runs whose stored counters differ from durable task and result state."}),
		RunCounterDriftOldestIdle: prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_run_counter_drift_oldest_idle_seconds", Help: "Longest time since durable progress among active parameter sync runs with counter drift."}),
		ActiveRunOldestAge:        prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_active_run_oldest_age_seconds", Help: "Age in seconds of the oldest active parameter sync run."}),
		ReconcileFinalized:        prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_reconcile_finalized_total", Help: "Parameter sync runs finalized by maintenance reconciliation."}, []string{"result"}),
		ReconcileDuration:         prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_reconcile_duration_seconds", Help: "Duration of one parameter sync convergence maintenance sweep.", Buckets: prometheus.DefBuckets}),
		RunsBlocked:               prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_runs_blocked", Help: "Active parameter sync runs blocked from convergence by reason."}, []string{"reason"}),
		RunsBlockedOldestIdle:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_runs_blocked_oldest_idle_seconds", Help: "Longest time since durable progress among blocked active parameter sync runs by reason."}, []string{"reason"}),
		ResultShardQueueDepth:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_result_shard_queue_depth", Help: "Queued parameter sync result work items by shard."}, []string{"shard"}),
		ResultShardQueueFull:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_result_shard_queue_full_total", Help: "Parameter sync result shard queue-full events."}, []string{"shard"}),
		ResultShardInflight:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_result_shard_inflight", Help: "In-flight parameter sync result workers by shard."}, []string{"shard"}),
		ResultProcessDuration:     prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_result_process_duration_seconds", Help: "Parameter sync result processing duration.", Buckets: prometheus.DefBuckets}),
		AutomaticReservedRuns:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_automatic_admission_reserved_runs", Help: "Automatic parameter sync runs currently holding global admission capacity."}),
		AutomaticQueuedRequests:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_automatic_admission_queued_requests", Help: "Automatic parameter sync requests queued by global admission backpressure."}),
		AutomaticOldestQueueAge:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_automatic_admission_oldest_queue_age_seconds", Help: "Age in seconds of the oldest automatic request queued by global admission backpressure."}),
	}
	if reg != nil {
		reg.MustRegister(m.RequestsTotal, m.RunsActive, m.RequestDuration, m.RunDuration,
			m.TasksExpected, m.TasksTerminal, m.TasksProcessed, m.TasksFailed,
			m.ResultRedelivery, m.ResultCountMode, m.FinalizeTotal, m.OutboxBacklog, m.StagingRows, m.ReconcileRepairs,
			m.RunsReadyButNotFinalized, m.RunCounterDrift, m.RunCounterDriftOldestIdle, m.ActiveRunOldestAge,
			m.ReconcileFinalized, m.ReconcileDuration, m.RunsBlocked, m.RunsBlockedOldestIdle,
			m.ResultShardQueueDepth, m.ResultShardQueueFull, m.ResultShardInflight, m.ResultProcessDuration,
			m.AutomaticReservedRuns, m.AutomaticQueuedRequests, m.AutomaticOldestQueueAge)
	}
	return m
}
