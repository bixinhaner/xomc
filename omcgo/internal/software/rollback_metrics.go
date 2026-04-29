// Package software — Prometheus metrics for rollback audit (T-0021 / R-101).
//
// Three counters cover the audit needs spelled out in the PRD:
//   - software_rollback_total{source}                — task creations, by source
//   - software_rollback_devices_total{source}        — devices included, by source
//   - software_rollback_with_target_total            — task creations that override
//     the target firmware (independent
//     of source label)
//
// All three are nil-safe so production wiring (DI passes a registry) and tests
// (no registry) share one method surface.
package software

import "github.com/prometheus/client_golang/prometheus"

// RollbackMetrics holds the rollback-specific Prometheus collectors.
type RollbackMetrics struct {
	totalBySource      *prometheus.CounterVec
	devicesBySource    *prometheus.CounterVec
	withTargetFirmware prometheus.Counter
}

// NewRollbackMetrics registers the three rollback metrics on the given
// registry. Pass nil for tests; the returned struct is still usable.
func NewRollbackMetrics(reg prometheus.Registerer) *RollbackMetrics {
	m := &RollbackMetrics{
		totalBySource: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "software_rollback_total",
			Help: "Count of rollback tasks created, labelled by trigger source (manual/canary_failure/compatibility/scheduled).",
		}, []string{"source"}),
		devicesBySource: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "software_rollback_devices_total",
			Help: "Count of devices included in rollback tasks, labelled by trigger source.",
		}, []string{"source"}),
		withTargetFirmware: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "software_rollback_with_target_total",
			Help: "Count of rollback tasks that override the target firmware (instead of using the device's previous version).",
		}),
	}
	if reg != nil {
		reg.MustRegister(m.totalBySource, m.devicesBySource, m.withTargetFirmware)
	}
	return m
}

// RecordRollback bumps total{source} by 1 and devices{source} by deviceCount.
// Caller is expected to pass a canonical RollbackSource* constant.
func (m *RollbackMetrics) RecordRollback(source string, deviceCount int) {
	if m == nil {
		return
	}
	m.totalBySource.WithLabelValues(source).Inc()
	if deviceCount > 0 {
		m.devicesBySource.WithLabelValues(source).Add(float64(deviceCount))
	}
}

// RecordWithTarget increments the override-firmware counter; called only when
// the request supplied a non-nil target_firmware_id.
func (m *RollbackMetrics) RecordWithTarget() {
	if m == nil {
		return
	}
	m.withTargetFirmware.Inc()
}
