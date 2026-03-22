package task

import "github.com/prometheus/client_golang/prometheus"

// TaskMetrics holds Prometheus metrics for the task queue module.
type TaskMetrics struct {
	PendingTotal   prometheus.Gauge
	CompletedTotal *prometheus.CounterVec
}

// NewTaskMetrics creates and registers task metrics.
func NewTaskMetrics(reg prometheus.Registerer) *TaskMetrics {
	m := &TaskMetrics{
		PendingTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_tasks_pending_total",
			Help: "Current number of pending tasks",
		}),
		CompletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_tasks_completed_total",
			Help: "Total number of completed tasks by status",
		}, []string{"status"}),
	}

	reg.MustRegister(m.PendingTotal, m.CompletedTotal)
	return m
}
