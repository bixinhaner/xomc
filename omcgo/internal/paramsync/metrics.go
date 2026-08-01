package paramsync

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	RequestsTotal         *prometheus.CounterVec
	RunsActive            *prometheus.GaugeVec
	RequestDuration       prometheus.Histogram
	RunDuration           prometheus.Histogram
	TasksExpected         prometheus.Gauge
	TasksTerminal         prometheus.Gauge
	TasksProcessed        prometheus.Gauge
	TasksFailed           prometheus.Gauge
	ResultRedelivery      prometheus.Counter
	ResultCountMode       *prometheus.CounterVec
	FinalizeTotal         *prometheus.CounterVec
	OutboxBacklog         prometheus.Gauge
	StagingRows           prometheus.Gauge
	ReconcileRepairs      *prometheus.CounterVec
	ResultShardQueueDepth *prometheus.GaugeVec
	ResultShardQueueFull  *prometheus.CounterVec
	ResultShardInflight   *prometheus.GaugeVec
	ResultProcessDuration prometheus.Histogram
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		RequestsTotal:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_requests_total", Help: "Parameter sync requests by scope, reason and status."}, []string{"scope", "reason", "status"}),
		RunsActive:            prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_runs_active", Help: "Active parameter sync runs by status."}, []string{"status"}),
		RequestDuration:       prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_request_duration_seconds", Help: "Parameter sync request duration.", Buckets: prometheus.DefBuckets}),
		RunDuration:           prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_run_duration_seconds", Help: "Parameter sync run duration.", Buckets: prometheus.DefBuckets}),
		TasksExpected:         prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_expected", Help: "Expected tasks in active parameter sync runs."}),
		TasksTerminal:         prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_terminal", Help: "Terminal tasks in active parameter sync runs."}),
		TasksProcessed:        prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_processed", Help: "Processed task results in active parameter sync runs."}),
		TasksFailed:           prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_failed", Help: "Failed task results in active parameter sync runs."}),
		ResultRedelivery:      prometheus.NewCounter(prometheus.CounterOpts{Name: "param_sync_result_redelivery_total", Help: "Duplicate/redelivered parameter sync results."}),
		ResultCountMode:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_result_count_total", Help: "Parameter sync result counter updates by incremental or authoritative mode."}, []string{"mode"}),
		FinalizeTotal:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_finalize_total", Help: "Parameter sync finalizations by result."}, []string{"result"}),
		OutboxBacklog:         prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_outbox_backlog", Help: "Pending/failed parameter sync outbox rows."}),
		StagingRows:           prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_staging_rows", Help: "Estimated current parameter sync staging rows from PostgreSQL planner statistics."}),
		ReconcileRepairs:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_reconcile_repairs_total", Help: "Parameter sync reconciliation repairs by kind."}, []string{"kind"}),
		ResultShardQueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_result_shard_queue_depth", Help: "Queued parameter sync result work items by shard."}, []string{"shard"}),
		ResultShardQueueFull:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_result_shard_queue_full_total", Help: "Parameter sync result shard queue-full events."}, []string{"shard"}),
		ResultShardInflight:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_result_shard_inflight", Help: "In-flight parameter sync result workers by shard."}, []string{"shard"}),
		ResultProcessDuration: prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_result_process_duration_seconds", Help: "Parameter sync result processing duration.", Buckets: prometheus.DefBuckets}),
	}
	if reg != nil {
		reg.MustRegister(m.RequestsTotal, m.RunsActive, m.RequestDuration, m.RunDuration,
			m.TasksExpected, m.TasksTerminal, m.TasksProcessed, m.TasksFailed,
			m.ResultRedelivery, m.ResultCountMode, m.FinalizeTotal, m.OutboxBacklog, m.StagingRows, m.ReconcileRepairs,
			m.ResultShardQueueDepth, m.ResultShardQueueFull, m.ResultShardInflight, m.ResultProcessDuration)
	}
	return m
}
