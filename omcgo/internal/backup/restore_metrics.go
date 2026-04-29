// Package backup — Prometheus metrics for restore (T-0072).
//
// Lives in a separate file from policy_metrics.go so the restore subsystem
// can be wired or skipped independently in DI without dragging policy
// collectors along (worker process, for example, has policy metrics but does
// not host the restore endpoint).
package backup

import "github.com/prometheus/client_golang/prometheus"

// RestoreMetrics holds the restore-specific Prometheus collectors.
type RestoreMetrics struct {
	requestsTotal    *prometheus.CounterVec
	devicesEnqueued  prometheus.Counter
}

// NewRestoreMetrics registers collectors on the given registry.
// Pass nil for tests; the returned struct is still usable.
func NewRestoreMetrics(reg prometheus.Registerer) *RestoreMetrics {
	m := &RestoreMetrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_restore_requests_total",
			Help: "Backup restore request outcomes by classification.",
		}, []string{"result"}),
		devicesEnqueued: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_backup_restore_devices_enqueued_total",
			Help: "Total number of device tasks enqueued by restore (cumulative).",
		}),
	}
	if reg != nil {
		reg.MustRegister(m.requestsTotal, m.devicesEnqueued)
	}
	return m
}

// RecordRequest increments the request outcome counter.
// result ∈ {"accepted", "rejected_invalid_path", "rejected_no_target",
//           "rejected_object_not_found"}.
func (m *RestoreMetrics) RecordRequest(result string) {
	if m == nil {
		return
	}
	m.requestsTotal.WithLabelValues(result).Inc()
}

// RecordDevicesEnqueued adds n to the cumulative enqueued device-task count.
func (m *RestoreMetrics) RecordDevicesEnqueued(n int64) {
	if m == nil || n <= 0 {
		return
	}
	m.devicesEnqueued.Add(float64(n))
}
