package alarm

import (
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmDiff represents the difference between remote (device) and local (database) alarms.
type AlarmDiff struct {
	ToAdd    []*model.Alarm          // Remote exists, local doesn't → new alarm
	ToUpdate map[string]*model.Alarm // alarm_identifier match, attributes changed → update
	ToClear  []string               // alarm_identifiers in local but not in remote → clear (archive)
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
// and produces a three-way diff. Match key: alarm_identifier.
func ComputeDiff(remote, local []*model.Alarm) *AlarmDiff {
	diff := &AlarmDiff{
		ToUpdate: make(map[string]*model.Alarm),
	}

	remoteMap := make(map[string]*model.Alarm, len(remote))
	for i := range remote {
		if remote[i] != nil {
			remoteMap[remote[i].AlarmIdentifier] = remote[i]
		}
	}

	localMap := make(map[string]*model.Alarm, len(local))
	localIdentifiers := make(map[string]bool, len(local))
	for i := range local {
		if local[i] != nil {
			localMap[local[i].AlarmIdentifier] = local[i]
			localIdentifiers[local[i].AlarmIdentifier] = true
		}
	}

	// Find ToAdd: remote exists but local doesn't
	for id, rAlarm := range remoteMap {
		if !localIdentifiers[id] {
			diff.ToAdd = append(diff.ToAdd, rAlarm)
		}
	}

	// Find ToUpdate: both exist, check if attributes changed
	for id, rAlarm := range remoteMap {
		lAlarm, exists := localMap[id]
		if !exists {
			continue
		}
		if hasAlarmChanged(rAlarm, lAlarm) {
			diff.ToUpdate[id] = rAlarm
		}
	}

	// Find ToClear: local exists but remote doesn't
	for id := range localIdentifiers {
		if _, exists := remoteMap[id]; !exists {
			diff.ToClear = append(diff.ToClear, id)
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
