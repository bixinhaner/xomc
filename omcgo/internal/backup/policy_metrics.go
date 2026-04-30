// Package backup — Prometheus metrics for BackupPolicy enforcement (T-0073, T-0074, T-0082).
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
// T-0082 storage threshold gauges/counters:
//   - omc_backup_storage_used_bytes                — current bucket usage (gauge)
//   - omc_backup_storage_capacity_bytes            — MaxStorageGB×1024³ (gauge)
//   - omc_backup_storage_usage_ratio               — used/capacity, may exceed 1 (gauge)
//   - omc_backup_storage_check_total{result}       — poll outcomes (success/failure/skipped)
//   - omc_backup_storage_threshold_alarm_total{kind} — raise/clear/skipped (counter)
//
// T-0089 decrypt semaphore (concurrency cap to bound 64MB×N memory amplification):
//   - omc_backup_decrypt_in_flight                 — current decrypt slots held (gauge)
//   - omc_backup_decrypt_wait_seconds              — acquire wait latency (histogram)
//   - omc_backup_decrypt_rejected_total{reason}    — timeout/ctx_cancel/oversize_config (counter)
//
// T-0083 multi-device orphan reaper (weekly housekeeping for files left
// behind by first-write-wins multi-device backups):
//   - omc_backup_orphan_reaped_total               — successful reap count (counter)
//   - omc_backup_orphan_skipped_total{reason}      — pattern_mismatch|live_task|age_recent|api_error (counter)
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

	// T-0076 physical file delete (MinIO object removal alongside DB cleanup):
	fileDeletedTotal      prometheus.Counter
	fileDeleteErrorsTotal *prometheus.CounterVec

	// T-0075 backup encryption (AES-256-GCM envelope; CBC/ChaCha20 stub):
	encryptedTotal        prometheus.Counter
	encryptionErrorsTotal *prometheus.CounterVec
	decryptionErrorsTotal *prometheus.CounterVec

	// T-0082 storage threshold (poll bucket size + edge-trigger alarm):
	storageUsedBytes       prometheus.Gauge
	storageCapacityBytes   prometheus.Gauge
	storageUsageRatio      prometheus.Gauge
	storageCheckTotal      *prometheus.CounterVec
	storageThresholdAlarms *prometheus.CounterVec

	// T-0089 decrypt semaphore (concurrency cap):
	decryptInFlight      prometheus.Gauge
	decryptWaitSeconds   prometheus.Histogram
	decryptRejectedTotal *prometheus.CounterVec

	// T-0083 multi-device orphan reaper (weekly housekeeping):
	orphanReapedTotal  prometheus.Counter
	orphanSkippedTotal *prometheus.CounterVec
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

		fileDeletedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_backup_file_deleted_total",
			Help: "Total MinIO objects physically deleted by cleanup cron (T-0076).",
		}),
		fileDeleteErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_file_delete_errors_total",
			Help: "Cleanup MinIO RemoveObject errors by reason (not_found|network|parse_path|other).",
		}, []string{"reason"}),

		encryptedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_backup_encrypted_total",
			Help: "Backup files successfully AES-256-GCM encrypted before MinIO PutObject (T-0075).",
		}),
		encryptionErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_encryption_errors_total",
			Help: "Backup encrypt-side failures by reason (key_unavailable|oversize|encrypt_fail|format_invalid).",
		}, []string{"reason"}),
		decryptionErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_decryption_errors_total",
			Help: "Backup decrypt-side failures by reason (wrong_aad|tamper|key_unavailable|format_invalid).",
		}, []string{"reason"}),

		storageUsedBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_backup_storage_used_bytes",
			Help: "Current backup bucket usage in bytes (T-0082; updated by hourly storage check).",
		}),
		storageCapacityBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_backup_storage_capacity_bytes",
			Help: "Backup storage capacity in bytes from BackupPolicy.MaxStorageGB×1024³ (T-0082).",
		}),
		storageUsageRatio: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_backup_storage_usage_ratio",
			Help: "Backup storage usage ratio used/capacity; may exceed 1 when over capacity (T-0082).",
		}),
		storageCheckTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_storage_check_total",
			Help: "Backup storage threshold check tick outcomes (success|failure|skipped) (T-0082).",
		}, []string{"result"}),
		storageThresholdAlarms: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_storage_threshold_alarm_total",
			Help: "Backup storage threshold alarm publish events (raised|cleared|skipped) (T-0082).",
		}, []string{"kind"}),

		decryptInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_backup_decrypt_in_flight",
			Help: "Current concurrent backup-decrypt slots held (T-0089 semaphore).",
		}),
		decryptWaitSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_backup_decrypt_wait_seconds",
			Help:    "Backup-decrypt semaphore acquire wait latency (T-0089).",
			Buckets: prometheus.DefBuckets,
		}),
		decryptRejectedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_decrypt_rejected_total",
			Help: "Backup-decrypt semaphore acquire rejections by reason (timeout|ctx_cancel|oversize_config) (T-0089).",
		}, []string{"reason"}),

		orphanReapedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_backup_orphan_reaped_total",
			Help: "Total multi-device orphan files physically deleted by the weekly reaper (T-0083).",
		}),
		orphanSkippedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_orphan_skipped_total",
			Help: "Reaper-skipped objects by reason (pattern_mismatch|live_task|age_recent|api_error) (T-0083).",
		}, []string{"reason"}),
	}
	if reg != nil {
		reg.MustRegister(
			m.cleanupRuns, m.cleanupRowsDeleted, m.failureAlarmTotal,
			m.compressionBytesIn, m.compressionBytesOut,
			m.compressionDuration, m.compressionErrors,
			m.fileDeletedTotal, m.fileDeleteErrorsTotal,
			m.encryptedTotal, m.encryptionErrorsTotal, m.decryptionErrorsTotal,
			m.storageUsedBytes, m.storageCapacityBytes, m.storageUsageRatio,
			m.storageCheckTotal, m.storageThresholdAlarms,
			m.decryptInFlight, m.decryptWaitSeconds, m.decryptRejectedTotal,
			m.orphanReapedTotal, m.orphanSkippedTotal,
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

// RecordFileDeleted increments the cleanup-side MinIO RemoveObject success
// counter (T-0076).
func (m *PolicyMetrics) RecordFileDeleted() {
	if m == nil {
		return
	}
	m.fileDeletedTotal.Inc()
}

// RecordFileDeleteError tallies a cleanup-side MinIO RemoveObject failure.
// reason ∈ {"not_found", "network", "parse_path", "other"}.
func (m *PolicyMetrics) RecordFileDeleteError(reason string) {
	if m == nil {
		return
	}
	m.fileDeleteErrorsTotal.WithLabelValues(reason).Inc()
}

// RecordBackupEncrypted increments the upload-side encryption success counter (T-0075).
func (m *PolicyMetrics) RecordBackupEncrypted() {
	if m == nil {
		return
	}
	m.encryptedTotal.Inc()
}

// RecordBackupEncryptionError tallies an encrypt-side failure.
// reason ∈ {"key_unavailable", "oversize", "encrypt_fail", "format_invalid"}.
func (m *PolicyMetrics) RecordBackupEncryptionError(reason string) {
	if m == nil {
		return
	}
	m.encryptionErrorsTotal.WithLabelValues(reason).Inc()
}

// RecordBackupDecryptionError tallies a decrypt-side failure on the download path.
// reason ∈ {"wrong_aad", "tamper", "key_unavailable", "format_invalid"}.
func (m *PolicyMetrics) RecordBackupDecryptionError(reason string) {
	if m == nil {
		return
	}
	m.decryptionErrorsTotal.WithLabelValues(reason).Inc()
}

// SetStorageUsedBytes updates the bucket usage gauge (T-0082).
func (m *PolicyMetrics) SetStorageUsedBytes(used int64) {
	if m == nil {
		return
	}
	m.storageUsedBytes.Set(float64(used))
}

// SetStorageCapacityBytes updates the configured capacity gauge (T-0082).
func (m *PolicyMetrics) SetStorageCapacityBytes(capacity int64) {
	if m == nil {
		return
	}
	m.storageCapacityBytes.Set(float64(capacity))
}

// SetStorageUsageRatio updates the usage ratio gauge (T-0082). Callers must
// guard capacity=0 to avoid NaN.
func (m *PolicyMetrics) SetStorageUsageRatio(ratio float64) {
	if m == nil {
		return
	}
	m.storageUsageRatio.Set(ratio)
}

// RecordStorageCheck increments the storage check tick outcome counter (T-0082).
// result ∈ {"success", "failure", "skipped"}.
func (m *PolicyMetrics) RecordStorageCheck(result string) {
	if m == nil {
		return
	}
	m.storageCheckTotal.WithLabelValues(result).Inc()
}

// RecordStorageThresholdAlarm increments the storage alarm publish counter (T-0082).
// kind ∈ {"raised", "cleared", "skipped"}.
func (m *PolicyMetrics) RecordStorageThresholdAlarm(kind string) {
	if m == nil {
		return
	}
	m.storageThresholdAlarms.WithLabelValues(kind).Inc()
}

// SetBackupDecryptInFlight updates the decrypt slot gauge (T-0089).
func (m *PolicyMetrics) SetBackupDecryptInFlight(n int) {
	if m == nil {
		return
	}
	m.decryptInFlight.Set(float64(n))
}

// ObserveBackupDecryptWait records an acquire wait sample in seconds (T-0089).
func (m *PolicyMetrics) ObserveBackupDecryptWait(seconds float64) {
	if m == nil {
		return
	}
	m.decryptWaitSeconds.Observe(seconds)
}

// RecordBackupDecryptRejected increments the decrypt rejection counter (T-0089).
// reason ∈ {"timeout", "ctx_cancel", "oversize_config"}.
func (m *PolicyMetrics) RecordBackupDecryptRejected(reason string) {
	if m == nil {
		return
	}
	m.decryptRejectedTotal.WithLabelValues(reason).Inc()
}

// RecordOrphanReaped increments the orphan reap success counter (T-0083).
func (m *PolicyMetrics) RecordOrphanReaped() {
	if m == nil {
		return
	}
	m.orphanReapedTotal.Inc()
}

// RecordOrphanSkipped tallies a reaper skip by reason (T-0083).
// reason ∈ {"pattern_mismatch", "live_task", "age_recent", "api_error"}.
func (m *PolicyMetrics) RecordOrphanSkipped(reason string) {
	if m == nil {
		return
	}
	m.orphanSkippedTotal.WithLabelValues(reason).Inc()
}
