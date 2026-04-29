// Package backup — Prometheus metrics for BackupPolicy enforcement (T-0073).
//
// Three counters cover the Phase 1 enforcement:
//   - backup_cleanup_runs_total{result}            — cron tick outcomes (success/failure/skipped)
//   - backup_cleanup_rows_deleted_total            — rows removed (sum across all runs)
//   - backup_failure_alarm_published_total{kind}   — alarm.raised publish events (published/skipped)
//
// All methods are nil-safe so production wiring (DI passes a registry) and
// tests (no registry) share one method surface.
package backup

import "github.com/prometheus/client_golang/prometheus"

// PolicyMetrics holds the BackupPolicy-specific Prometheus collectors.
type PolicyMetrics struct {
	cleanupRuns         *prometheus.CounterVec
	cleanupRowsDeleted  prometheus.Counter
	failureAlarmTotal   *prometheus.CounterVec
}

// NewPolicyMetrics registers the three counters on the given registry.
// Pass nil for tests; the returned struct is still usable.
func NewPolicyMetrics(reg prometheus.Registerer) *PolicyMetrics {
	m := &PolicyMetrics{
		cleanupRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "backup_cleanup_runs_total",
			Help: "Cron-driven backup_tasks cleanup tick outcomes.",
		}, []string{"result"}),
		cleanupRowsDeleted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "backup_cleanup_rows_deleted_total",
			Help: "Total backup_tasks rows deleted across all cleanup runs.",
		}),
		failureAlarmTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "backup_failure_alarm_published_total",
			Help: "Backup task failure alarm.raised publish events, by outcome.",
		}, []string{"kind"}),
	}
	if reg != nil {
		reg.MustRegister(m.cleanupRuns, m.cleanupRowsDeleted, m.failureAlarmTotal)
	}
	return m
}

// RecordCleanupRun increments the cleanup tick outcome counter.
// result ∈ {"success", "failure", "skipped"}.
func (m *PolicyMetrics) RecordCleanupRun(result string) {
	if m == nil {
		return
	}
	m.cleanupRuns.WithLabelValues(result).Inc()
}

// RecordCleanupDeleted adds n to the rows-deleted counter (no-op when n ≤ 0).
func (m *PolicyMetrics) RecordCleanupDeleted(n int64) {
	if m == nil || n <= 0 {
		return
	}
	m.cleanupRowsDeleted.Add(float64(n))
}

// RecordFailureAlarm increments the failure-alarm publish counter.
// kind ∈ {"published", "skipped"} (skipped = alertOnFailure=false).
func (m *PolicyMetrics) RecordFailureAlarm(kind string) {
	if m == nil {
		return
	}
	m.failureAlarmTotal.WithLabelValues(kind).Inc()
}
