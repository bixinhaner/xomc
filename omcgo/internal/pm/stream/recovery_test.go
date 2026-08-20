package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRecoverySourceRejectsUnknownTaskVersion(t *testing.T) {
	snapshot := NewSnapshotStore(nil, nil)
	recovery := &Recovery{snapshot: snapshot}
	key := WindowKey{
		TaskID:        uuid.New(),
		TaskVersionID: uuid.New(),
		EntityKey:     "network",
		Granularity:   GranularityDaily,
		Start:         time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
		End:           time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC),
	}

	_, err := recovery.recoverySourceFor(key)
	require.ErrorContains(t, err, "task version is not recoverable")
}

func TestRecoveryTerminalForInvalidTaskVersionStates(t *testing.T) {
	windowStart := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)
	taskID := uuid.New()
	versionID := uuid.New()
	base := TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID,
		TaskEnabled: true, Enabled: true,
		EffectiveFrom: windowStart.Add(-time.Hour),
	}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: versionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: windowStart, End: windowEnd,
	}

	tests := []struct {
		name       string
		version    *TaskVersionSnapshot
		mutateKey  func(*WindowKey)
		wantStatus string
		wantReason string
	}{
		{name: "missing", wantStatus: "orphaned", wantReason: "missing from primary database"},
		{name: "task mismatch", version: cloneTaskVersion(base), mutateKey: func(k *WindowKey) {
			k.TaskID = uuid.New()
		}, wantStatus: "orphaned", wantReason: "identity does not match"},
		{name: "deleted", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "deleted"},
		{name: "task disabled", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "task is disabled"},
		{name: "version disabled", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "version is disabled"},
		{name: "planned end", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "planned end"},
		{name: "before effective range", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "before task version"},
		{name: "after effective range", version: cloneTaskVersion(base), wantStatus: "retired", wantReason: "after task version"},
	}

	deletedAt := windowStart.Add(-time.Minute)
	tests[2].version.TaskDeletedAt = &deletedAt
	tests[3].version.TaskEnabled = false
	tests[4].version.Enabled = false
	plannedEnd := windowStart
	tests[5].version.PlannedEndAt = &plannedEnd
	tests[6].version.EffectiveFrom = windowEnd
	effectiveTo := windowStart
	tests[7].version.EffectiveTo = &effectiveTo

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testKey := key
			if tc.mutateKey != nil {
				tc.mutateKey(&testKey)
			}
			states := map[uuid.UUID]RecoveryVersionState{}
			if tc.version != nil {
				states[versionID] = recoveryStateFromVersion(tc.version)
			}
			terminal, invalid := recoveryTerminalFor(
				testKey, &TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{}}, states,
			)
			require.True(t, invalid)
			require.Equal(t, tc.wantStatus, terminal.status)
			require.Contains(t, terminal.reason, tc.wantReason)
		})
	}
}

func TestRecoveryTerminalForKeepsOverlappingEffectiveWindow(t *testing.T) {
	start := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	taskID, versionID := uuid.New(), uuid.New()
	effectiveTo := start.Add(30 * time.Minute)
	version := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: versionID,
		TaskEnabled: true, Enabled: true,
		EffectiveFrom: start.Add(-time.Hour), EffectiveTo: &effectiveTo,
	}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: versionID,
		EntityKey: "network", Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}

	_, invalid := recoveryTerminalFor(
		key,
		&TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{}},
		map[uuid.UUID]RecoveryVersionState{versionID: recoveryStateFromVersion(version)},
	)
	require.False(t, invalid)
}

func cloneTaskVersion(source TaskVersionSnapshot) *TaskVersionSnapshot {
	clone := source
	return &clone
}

func recoveryStateFromVersion(version *TaskVersionSnapshot) RecoveryVersionState {
	return RecoveryVersionState{
		TaskID: version.TaskID, TaskEnabled: version.TaskEnabled,
		TaskDeletedAt: version.TaskDeletedAt, VersionID: version.VersionID,
		VersionEnabled: version.Enabled, EffectiveFrom: version.EffectiveFrom,
		EffectiveTo: version.EffectiveTo, PlannedEndAt: version.PlannedEndAt,
	}
}
