package alarm

import (
	"sort"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmUpdate pairs the local alarm that must be preserved with the remote
// alarm whose device-owned fields should be applied.
type AlarmUpdate struct {
	Local  *model.Alarm
	Remote *model.Alarm
}

// DuplicateAlarmClear identifies a stale duplicate and the active instance
// that remains authoritative after reconciliation.
type DuplicateAlarmClear struct {
	Duplicate *model.Alarm
	Keeper    *model.Alarm
}

// AlarmDiff represents the difference between remote (device) and local (database) alarms.
type AlarmDiff struct {
	ToAdd             []*model.Alarm
	ToUpdate          []AlarmUpdate
	ToClear           []*model.Alarm
	ToClearDuplicates []DuplicateAlarmClear
}

// SyncResult summarizes the outcome of an alarm sync operation.
type SyncResult struct {
	DeviceSN     string    `json:"device_sn"`
	Added        int       `json:"added"`
	Updated      int       `json:"updated"`
	Cleared      int       `json:"cleared"`
	FailedAdd    int       `json:"failed_add"`
	FailedUpdate int       `json:"failed_update"`
	FailedClear  int       `json:"failed_clear"`
	SyncedAt     time.Time `json:"synced_at"`
}

// SyncStatus represents the current sync state for a device.
type SyncStatus struct {
	DeviceSN         string      `json:"device_sn"`
	LastSyncedAt     *time.Time  `json:"last_synced_at"`
	SyncStatus       string      `json:"sync_status"` // idle, syncing, synced, failed
	LastResult       *SyncResult `json:"last_result,omitempty"`
	ActiveAlarmCount int         `json:"active_alarm_count"`
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
	diff := &AlarmDiff{}
	remoteGroups := groupActiveAlarms(remote)
	localGroups := groupActiveAlarms(local)

	keySet := make(map[string]struct{}, len(remoteGroups)+len(localGroups))
	for key := range remoteGroups {
		keySet[key] = struct{}{}
	}
	for key := range localGroups {
		keySet[key] = struct{}{}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		remoteAlarm := latestAlarm(remoteGroups[key])
		localGroup := localGroups[key]

		switch {
		case remoteAlarm == nil:
			sortAlarmsLatestFirst(localGroup)
			diff.ToClear = append(diff.ToClear, localGroup...)
		case len(localGroup) == 0:
			diff.ToAdd = append(diff.ToAdd, remoteAlarm)
		default:
			keeper := closestLocalAlarm(localGroup, remoteAlarm)
			if hasAlarmChanged(remoteAlarm, keeper) {
				diff.ToUpdate = append(diff.ToUpdate, AlarmUpdate{
					Local:  keeper,
					Remote: remoteAlarm,
				})
			}
			for _, alarm := range localGroup {
				if alarm != keeper {
					diff.ToClearDuplicates = append(diff.ToClearDuplicates, DuplicateAlarmClear{
						Duplicate: alarm,
						Keeper:    keeper,
					})
				}
			}
		}
	}

	return diff
}

func groupActiveAlarms(alarms []*model.Alarm) map[string][]*model.Alarm {
	grouped := make(map[string][]*model.Alarm)
	for _, alarm := range alarms {
		if alarm == nil {
			continue
		}
		key := activeAlarmMatchKey(alarm)
		grouped[key] = append(grouped[key], alarm)
	}
	return grouped
}

func closestLocalAlarm(alarms []*model.Alarm, remote *model.Alarm) *model.Alarm {
	if len(alarms) == 0 {
		return nil
	}
	keeper := alarms[0]
	for _, alarm := range alarms[1:] {
		if isBetterLocalKeeper(alarm, keeper, remote) {
			keeper = alarm
		}
	}
	return keeper
}

func isBetterLocalKeeper(candidate, current, remote *model.Alarm) bool {
	candidateDistance := alarmRaisedDistance(candidate, remote)
	currentDistance := alarmRaisedDistance(current, remote)
	if candidateDistance != currentDistance {
		return candidateDistance < currentDistance
	}
	if !candidate.RaisedAt.Equal(current.RaisedAt) {
		return candidate.RaisedAt.After(current.RaisedAt)
	}
	if !candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.CreatedAt.After(current.CreatedAt)
	}
	return candidate.ID.String() > current.ID.String()
}

func alarmRaisedDistance(local, remote *model.Alarm) time.Duration {
	if local == nil || remote == nil || local.RaisedAt.IsZero() || remote.RaisedAt.IsZero() {
		return time.Duration(1<<63 - 1)
	}
	distance := local.RaisedAt.Sub(remote.RaisedAt)
	if distance < 0 {
		return -distance
	}
	return distance
}

func latestAlarm(alarms []*model.Alarm) *model.Alarm {
	if len(alarms) == 0 {
		return nil
	}
	latest := alarms[0]
	for _, alarm := range alarms[1:] {
		if isBetterLocalKeeper(alarm, latest, nil) {
			latest = alarm
		}
	}
	return latest
}

func sortAlarmsLatestFirst(alarms []*model.Alarm) {
	sort.SliceStable(alarms, func(i, j int) bool {
		return isBetterLocalKeeper(alarms[i], alarms[j], nil)
	})
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
