package stream

import (
	"errors"
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

func TestRecoveryTerminalForMissingRaw15mSource(t *testing.T) {
	now := time.Date(2026, 8, 23, 2, 0, 0, 0, time.UTC)
	recovery := &Recovery{
		now:             func() time.Time { return now },
		replayRetention: 45 * 24 * time.Hour,
	}
	key := WindowKey{
		Granularity: GranularityHourly,
		Start:       now.Add(-46 * 24 * time.Hour),
		End:         now.Add(-46 * 24 * time.Hour).Add(time.Hour),
	}

	terminal, ok := recovery.recoveryTerminalForMissingSource(key, recoveryRaw15m, nil)
	require.True(t, ok)
	require.Equal(t, "abandoned", terminal.status)
	require.Contains(t, terminal.reason, "raw_15m")
}

func TestRecoveryTerminalForMissingSourceKeepsTransientAndRollupFailuresRetriable(t *testing.T) {
	now := time.Date(2026, 8, 23, 2, 0, 0, 0, time.UTC)
	recovery := &Recovery{
		now:             func() time.Time { return now },
		replayRetention: 45 * 24 * time.Hour,
	}
	oldKey := WindowKey{
		Granularity: GranularityHourly,
		Start:       now.Add(-46 * 24 * time.Hour),
		End:         now.Add(-46 * 24 * time.Hour).Add(time.Hour),
	}
	recentKey := WindowKey{
		Granularity: GranularityHourly,
		Start:       now.Add(-time.Hour),
		End:         now,
	}

	_, ok := recovery.recoveryTerminalForMissingSource(oldKey, recoveryRaw15m, errors.New("nats unavailable"))
	require.False(t, ok)

	_, ok = recovery.recoveryTerminalForMissingSource(recentKey, recoveryRaw15m, nil)
	require.False(t, ok)

	for _, source := range []recoverySource{recoveryDeviceHours, recoveryHourly, recoveryDaily} {
		_, ok = recovery.recoveryTerminalForMissingSource(oldKey, source, nil)
		require.False(t, ok)
	}
}

func TestClassifyMissingRecoveryRecordsSplitsMatchedAbandonedAndDeferred(t *testing.T) {
	now := time.Date(2026, 8, 23, 2, 0, 0, 0, time.UTC)
	recovery := &Recovery{
		now:             func() time.Time { return now },
		replayRetention: 45 * 24 * time.Hour,
	}
	old := recoveryTestRecord("old", now.Add(-46*24*time.Hour))
	recent := recoveryTestRecord("recent", now.Add(-time.Hour))
	matched := recoveryTestRecord("matched", now.Add(-46*24*time.Hour))

	outcome := recovery.classifyMissingRecoveryRecords(
		recoveryRaw15m,
		[]WindowRecord{old, recent, matched},
		map[string]struct{}{windowRecoveryID(matched.Key): {}},
		nil,
	)
	require.Len(t, outcome.abandoned, 1)
	require.Equal(t, "old", outcome.abandoned[0].Key.EntityKey)
	require.Len(t, outcome.deferred, 1)
	require.Equal(t, "recent", outcome.deferred[0].key.EntityKey)
	require.ErrorContains(t, outcome.deferred[0].err, "no retained raw_15m")
}

func TestClassifyMissingRecoveryRecordsDefersReplayErrorsAndRollupMisses(t *testing.T) {
	now := time.Date(2026, 8, 23, 2, 0, 0, 0, time.UTC)
	recovery := &Recovery{
		now:             func() time.Time { return now },
		replayRetention: 45 * 24 * time.Hour,
	}
	old := recoveryTestRecord("old", now.Add(-46*24*time.Hour))

	rawWithReplayErr := recovery.classifyMissingRecoveryRecords(
		recoveryRaw15m,
		[]WindowRecord{old},
		nil,
		errors.New("nats unavailable"),
	)
	require.Empty(t, rawWithReplayErr.abandoned)
	require.Len(t, rawWithReplayErr.deferred, 1)
	require.ErrorContains(t, rawWithReplayErr.deferred[0].err, "nats unavailable")

	rollupMissing := recovery.classifyMissingRecoveryRecords(
		recoveryHourly,
		[]WindowRecord{old},
		nil,
		nil,
	)
	require.Empty(t, rollupMissing.abandoned)
	require.Len(t, rollupMissing.deferred, 1)
	require.ErrorContains(t, rollupMissing.deferred[0].err, "no retained hourly")
}

func TestChunkRecoveryRecordsBoundsBatchSize(t *testing.T) {
	records := []WindowRecord{{}, {}, {}, {}, {}}
	chunks := chunkRecoveryRecords(records, 2)
	require.Len(t, chunks, 3)
	require.Len(t, chunks[0], 2)
	require.Len(t, chunks[1], 2)
	require.Len(t, chunks[2], 1)
}

func recoveryTestRecord(entity string, start time.Time) WindowRecord {
	return WindowRecord{
		Key: WindowKey{
			TaskID:        uuid.New(),
			TaskVersionID: uuid.New(),
			EntityKey:     entity,
			Granularity:   GranularityHourly,
			Start:         start,
			End:           start.Add(time.Hour),
		},
	}
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
