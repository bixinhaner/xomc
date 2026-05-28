package alarm

import (
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmDiff represents the difference between remote (device) and local (database) alarms.
type AlarmDiff struct {
	ToAdd    []*model.Alarm          // Remote exists, local doesn't → new alarm
	ToUpdate map[string]*model.Alarm // instance-level match key exists on both sides, attributes changed → update
	ToClear  []string                // instance-level match key exists only locally → clear (archive)
}

// SyncResult summarizes the outcome of an alarm sync operation.
type SyncResult struct {
	DeviceSN     string    `json:"device_sn"`
	Added        int       `json:"added"`
	Updated      int       `json:"updated"`
	Cleared       int       `json:"cleared"`
	FailedAdd    int       `json:"failed_add"`
	FailedUpdate int       `json:"failed_update"`
	FailedClear  int       `json:"failed_clear"`
	SyncedAt     time.Time `json:"synced_at"`
}

// SyncStatus represents the current sync state for a device.
type SyncStatus struct {
	DeviceSN        string       `json:"device_sn"`
	LastSyncedAt    *time.Time   `json:"last_synced_at"`
	SyncStatus      string       `json:"sync_status"` // idle, syncing, synced, failed
	LastResult      *SyncResult  `json:"last_result,omitempty"`
	ActiveAlarmCount int        `json:"active_alarm_count"`
}

// SyncStatus constants
const (
	SyncStatusIdle    = "idle"
	SyncStatusSyncing = "syncing"
	SyncStatusSynced  = "synced"
	SyncStatusFailed  = "failed"
)

// ComputeDiff compares remote alarms (from device) with local alarms (from DB)
// and produces a three-way diff. Match key: alarm_identifier + stable instance qualifier.
func ComputeDiff(remote, local []*model.Alarm) *AlarmDiff {
	diff := &AlarmDiff{
		ToUpdate: make(map[string]*model.Alarm),
	}

	remoteMap := indexActiveAlarms(remote)
	localMap := indexActiveAlarms(local)

	// Find ToAdd: remote exists but local doesn't
	for key, rAlarm := range remoteMap {
		if _, exists := localMap[key]; !exists {
			diff.ToAdd = append(diff.ToAdd, rAlarm)
		}
	}

	// Find ToUpdate: both exist, check if attributes changed
	for key, rAlarm := range remoteMap {
		lAlarm, exists := localMap[key]
		if !exists {
			continue
		}
		if hasAlarmChanged(rAlarm, lAlarm) {
			diff.ToUpdate[key] = rAlarm
		}
	}

	// Find ToClear: local exists but remote doesn't
	for key := range localMap {
		if _, exists := remoteMap[key]; !exists {
			diff.ToClear = append(diff.ToClear, key)
		}
	}

	return diff
}

// hasAlarmChanged checks if the remote alarm has meaningful attribute changes
// compared to the local version. Only severity and description are compared
// during sync (other fields like status/acknowledged are locally managed).
func hasAlarmChanged(remote, local *model.Alarm) bool {
	if remote.Severity != local.Severity {
		return true
	}
	if remote.Description != local.Description {
		return true
	}
	if remote.AlarmType != local.AlarmType {
		return true
	}
	if ptrStrNE(remote.AlarmSource, local.AlarmSource) {
		return true
	}
	if ptrStrNE(remote.EventType, local.EventType) {
		return true
	}
	return false
}

// ptrStrNE returns true if two *string values are not equal.
func ptrStrNE(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}
