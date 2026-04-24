package task

import "github.com/prometheus/client_golang/prometheus"

// TaskMetrics holds Prometheus metrics for the task queue module.
//
// P4 扩展（docs/design/mml-task-flow-design-20260424.md §3.3 C10）：
//   - CompletedTotal 新增 `source` 标签，便于按上游分组
//   - DurationSeconds 记录 device_task 从 sent → 终态的耗时，histogram
//   - NoHandlerTotal 记录 CompletionRouter 未匹配 source 的次数，帮助排查
//     漏注册问题
type TaskMetrics struct {
	PendingTotal     prometheus.Gauge
	CompletedTotal   *prometheus.CounterVec
	DurationSeconds  *prometheus.HistogramVec
	NoHandlerTotal   *prometheus.CounterVec
}

// NewTaskMetrics creates and registers task metrics.
func NewTaskMetrics(reg prometheus.Registerer) *TaskMetrics {
	m := &TaskMetrics{
		PendingTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_tasks_pending_total",
			Help: "Current number of pending tasks",
		}),
		CompletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mml_task_total",
			Help: "Total number of device_task completions by source and terminal status",
		}, []string{"source", "status"}),
		DurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mml_task_duration_seconds",
			Help:    "device_task latency from sent to terminal state, by source",
			Buckets: []float64{0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 180, 600},
		}, []string{"source"}),
		NoHandlerTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "completion_no_handler_total",
			Help: "CompletionRouter received task events whose source has no registered handler",
		}, []string{"source"}),
	}

	reg.MustRegister(
		m.PendingTotal,
		m.CompletedTotal,
		m.DurationSeconds,
		m.NoHandlerTotal,
	)
	return m
}
