// Package backup — Prometheus metrics for BackupPolicy enforcement (T-0073, T-0074).
//
// T-0073 cleanup counters:
//   - backup_cleanup_runs_total{result}            — cron tick outcomes (success/failure/skipped)
//   - backup_cleanup_rows_deleted_total            — rows removed (sum across all runs)
//   - backup_failure_alarm_published_total{kind}   — alarm.raised publish events (published/skipped)
//
// T-0074 compression counters/histograms (acs/upload/handler.go integration):
//   - omc_backup_compression_bytes_in_total{format}     — plaintext bytes consumed
//   - omc_backup_compression_bytes_out_total{format}    — compressed bytes produced
//   - omc_backup_compression_duration_seconds{format}   — single-stream compression latency
//   - omc_backup_compression_errors_total{format,reason} — failure tally (open|copy|close)
//
// All methods are nil-safe so production wiring (DI passes a registry) and
// tests (no registry) share one method surface.
package backup

import "github.com/prometheus/client_golang/prometheus"

// PolicyMetrics holds the BackupPolicy-specific Prometheus collectors.
type PolicyMetrics struct {
	cleanupRuns        *prometheus.CounterVec
	cleanupRowsDeleted prometheus.Counter
	failureAlarmTotal  *prometheus.CounterVec

	compressionBytesIn  *prometheus.CounterVec
	compressionBytesOut *prometheus.CounterVec
	compressionDuration *prometheus.HistogramVec
	compressionErrors   *prometheus.CounterVec
}

// NewPolicyMetrics registers all collectors on the given registry.
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

		compressionBytesIn: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_compression_bytes_in_total",
			Help: "Plaintext bytes consumed by backup compression, per format.",
		}, []string{"format"}),
		compressionBytesOut: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_compression_bytes_out_total",
			Help: "Compressed bytes produced by backup compression, per format.",
		}, []string{"format"}),
		compressionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omc_backup_compression_duration_seconds",
			Help:    "Single-stream backup compression latency, per format.",
			Buckets: prometheus.DefBuckets,
		}, []string{"format"}),
		compressionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_compression_errors_total",
			Help: "Backup compression error tally, per format and reason (open|copy|close).",
		}, []string{"format", "reason"}),
	}
	if reg != nil {
		reg.MustRegister(
			m.cleanupRuns, m.cleanupRowsDeleted, m.failureAlarmTotal,
			m.compressionBytesIn, m.compressionBytesOut,
			m.compressionDuration, m.compressionErrors,
		)
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

// RecordCompressionBytes adds plaintext-in / compressed-out byte counts.
// Negative values are ignored (no-op).
func (m *PolicyMetrics) RecordCompressionBytes(format string, bytesIn, bytesOut int64) {
	if m == nil {
		return
	}
	if bytesIn > 0 {
		m.compressionBytesIn.WithLabelValues(format).Add(float64(bytesIn))
	}
	if bytesOut > 0 {
		m.compressionBytesOut.WithLabelValues(format).Add(float64(bytesOut))
	}
}

// RecordCompressionDuration observes a compression latency sample (seconds).
func (m *PolicyMetrics) RecordCompressionDuration(format string, seconds float64) {
	if m == nil {
		return
	}
	m.compressionDuration.WithLabelValues(format).Observe(seconds)
}

// RecordCompressionError tallies a compression failure by reason.
// reason ∈ {"open", "copy", "close"}.
func (m *PolicyMetrics) RecordCompressionError(format, reason string) {
	if m == nil {
		return
	}
	m.compressionErrors.WithLabelValues(format, reason).Inc()
}
