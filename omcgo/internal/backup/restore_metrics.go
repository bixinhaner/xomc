// Package backup — Prometheus metrics for restore (T-0072).
//
// Lives in a separate file from policy_metrics.go so the restore subsystem
// can be wired or skipped independently in DI without dragging policy
// collectors along (worker process, for example, has policy metrics but does
// not host the restore endpoint).
package backup

import "github.com/prometheus/client_golang/prometheus"

// RestoreMetrics holds the restore-specific Prometheus collectors. T-0079
// adds two more on the same `omc_backup_*` namespace covering the
// task→file_path linkage and the by-task-id restore mode.
type RestoreMetrics struct {
	requestsTotal       *prometheus.CounterVec
	devicesEnqueued     prometheus.Counter
	filePathRecorded    *prometheus.CounterVec // T-0079
	restoreByTaskTotal  *prometheus.CounterVec // T-0079
	snapshotPromoted    *prometheus.CounterVec // T-0164 B3
	restoreBySnapshot   *prometheus.CounterVec // T-0164 B5
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
		filePathRecorded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_filepath_recorded_total",
			Help: "backup.file.received subscriber outcomes (T-0079). result ∈ {recorded, skipped_already_set, skipped_no_match, error}.",
		}, []string{"result"}),
		restoreByTaskTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_restore_by_task_total",
			Help: "Outcomes of POST /backup/restore/by-task-id (T-0079). result ∈ {accepted, rejected_not_uploaded, rejected_invalid_input}.",
		}, []string{"result"}),
		snapshotPromoted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_config_snapshot_promote_total",
			Help: "config_snapshots promote-from-backup outcomes (T-0164 B3). result ∈ {success, failed}.",
		}, []string{"result"}),
		restoreBySnapshot: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_restore_by_snapshot_total",
			Help: "Outcomes of POST /backup/restore/by-snapshot (T-0164 B5). result ∈ {accepted, rejected_missing_snapshot, rejected_invalid_input}.",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(
			m.requestsTotal, m.devicesEnqueued,
			m.filePathRecorded, m.restoreByTaskTotal,
			m.snapshotPromoted, m.restoreBySnapshot,
		)
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

// RecordFilePathRecord increments the file_path linkage counter (T-0079).
// result ∈ {"recorded", "skipped_already_set", "skipped_no_match", "error"}.
func (m *RestoreMetrics) RecordFilePathRecord(result string) {
	if m == nil {
		return
	}
	m.filePathRecorded.WithLabelValues(result).Inc()
}

// RecordRestoreByTask increments the by-task-id restore mode counter (T-0079).
// result ∈ {"accepted", "rejected_not_uploaded", "rejected_invalid_input"}.
func (m *RestoreMetrics) RecordRestoreByTask(result string) {
	if m == nil {
		return
	}
	m.restoreByTaskTotal.WithLabelValues(result).Inc()
}

// RecordSnapshotPromote increments the config_snapshot promote counter (T-0164 B3).
// result ∈ {"success", "failed"}.
func (m *RestoreMetrics) RecordSnapshotPromote(result string) {
	if m == nil {
		return
	}
	m.snapshotPromoted.WithLabelValues(result).Inc()
}

// RecordRestoreBySnapshot increments the by-snapshot restore mode counter (T-0164 B5).
// result ∈ {"accepted", "rejected_missing_snapshot", "rejected_invalid_input"}.
func (m *RestoreMetrics) RecordRestoreBySnapshot(result string) {
	if m == nil {
		return
	}
	m.restoreBySnapshot.WithLabelValues(result).Inc()
}
