package paramsync

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RunsActive       *prometheus.GaugeVec
	RequestDuration  prometheus.Histogram
	RunDuration      prometheus.Histogram
	TasksExpected    prometheus.Gauge
	TasksTerminal    prometheus.Gauge
	TasksProcessed   prometheus.Gauge
	TasksFailed      prometheus.Gauge
	ResultRedelivery prometheus.Counter
	FinalizeTotal    *prometheus.CounterVec
	OutboxBacklog    prometheus.Gauge
	StagingRows      prometheus.Gauge
	ReconcileRepairs *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		RequestsTotal:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_requests_total", Help: "Parameter sync requests by scope, reason and status."}, []string{"scope", "reason", "status"}),
		RunsActive:       prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "param_sync_runs_active", Help: "Active parameter sync runs by status."}, []string{"status"}),
		RequestDuration:  prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_request_duration_seconds", Help: "Parameter sync request duration.", Buckets: prometheus.DefBuckets}),
		RunDuration:      prometheus.NewHistogram(prometheus.HistogramOpts{Name: "param_sync_run_duration_seconds", Help: "Parameter sync run duration.", Buckets: prometheus.DefBuckets}),
		TasksExpected:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_expected", Help: "Expected tasks in active parameter sync runs."}),
		TasksTerminal:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_terminal", Help: "Terminal tasks in active parameter sync runs."}),
		TasksProcessed:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_processed", Help: "Processed task results in active parameter sync runs."}),
		TasksFailed:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_tasks_failed", Help: "Failed task results in active parameter sync runs."}),
		ResultRedelivery: prometheus.NewCounter(prometheus.CounterOpts{Name: "param_sync_result_redelivery_total", Help: "Duplicate/redelivered parameter sync results."}),
		FinalizeTotal:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_finalize_total", Help: "Parameter sync finalizations by result."}, []string{"result"}),
		OutboxBacklog:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_outbox_backlog", Help: "Pending/failed parameter sync outbox rows."}),
		StagingRows:      prometheus.NewGauge(prometheus.GaugeOpts{Name: "param_sync_staging_rows", Help: "Current parameter sync staging rows."}),
		ReconcileRepairs: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "param_sync_reconcile_repairs_total", Help: "Parameter sync reconciliation repairs by kind."}, []string{"kind"}),
	}
	if reg != nil {
		reg.MustRegister(m.RequestsTotal, m.RunsActive, m.RequestDuration, m.RunDuration,
			m.TasksExpected, m.TasksTerminal, m.TasksProcessed, m.TasksFailed,
			m.ResultRedelivery, m.FinalizeTotal, m.OutboxBacklog, m.StagingRows, m.ReconcileRepairs)
	}
	return m
}
