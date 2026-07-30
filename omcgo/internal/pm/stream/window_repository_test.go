package stream

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSortedSnapshotVersionIDsUsesStableLockOrder(t *testing.T) {
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	second := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	snapshot := &TaskSnapshot{
		ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
			second: {},
			first:  {},
		},
	}
	got := sortedSnapshotVersionIDs(snapshot)
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("version lock order = %v, want [%s %s]", got, first, second)
	}
}

func TestVersionMetadataBackfillLockSQL(t *testing.T) {
	blocking := versionMetadataBackfillLockSQL(false)
	if !strings.Contains(blocking, "pg_advisory_xact_lock") ||
		strings.Contains(blocking, "pg_try_advisory_xact_lock") {
		t.Fatalf("blocking lock SQL = %q", blocking)
	}

	nonBlocking := versionMetadataBackfillLockSQL(true)
	if !strings.Contains(nonBlocking, "pg_try_advisory_xact_lock") {
		t.Fatalf("non-blocking lock SQL = %q", nonBlocking)
	}
}

func TestObserveReceivedReopensRecoveredFailedWindow(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		EntityKey:     "device-1",
		Granularity:   GranularityHourly,
	}
	query, args, err := observeReceivedUpdate(key, 7).ToSql()
	if err != nil {
		t.Fatalf("build observe received SQL: %v", err)
	}
	for _, fragment := range []string{
		"status = $",
		"last_error = $",
		"status IN ($",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("observe received SQL %q missing %q", query, fragment)
		}
	}
	var hasOpen, hasFailed bool
	for _, arg := range args {
		hasOpen = hasOpen || arg == "open"
		hasFailed = hasFailed || arg == "failed"
	}
	if !hasOpen || !hasFailed {
		t.Fatalf("observe received args %v must reopen both open and failed windows", args)
	}
}

func TestFinalizationClaimSynchronizesCalibratedSlotsAndVersionMetadata(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		EntityKey:     "network",
		Granularity:   GranularityDaily,
		Start:         time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
	}
	closedAt := key.Start.Add(16 * time.Hour)
	version := &TaskVersionSnapshot{
		EffectiveFrom: key.Start.Add(5 * time.Hour),
		EffectiveTo:   &closedAt,
	}
	state := WindowState{ExpectedSlots: 11, ReceivedSlots: 11}
	coverage := finalizationCoverage{SourceExpectedSlots: 11, SourceReceivedSlots: 11}

	query, args, err := finalizationClaimUpdate(key, CloseTimeout, state, coverage, version).ToSql()
	if err != nil {
		t.Fatalf("build finalization claim SQL: %v", err)
	}
	for _, field := range []string{
		"expected_slots", "version_effective_from", "version_effective_to",
	} {
		if !strings.Contains(query, field) {
			t.Fatalf("finalization claim SQL %q missing %q", query, field)
		}
	}
	for _, want := range []any{int64(11), version.EffectiveFrom, closedAt} {
		found := false
		for _, arg := range args {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("finalization claim args %v missing %v", args, want)
		}
	}
}

func TestFinalizationClaimPreservesVersionMetadataWhenSnapshotIsMissing(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		EntityKey:     "network",
		Granularity:   GranularityDaily,
		Start:         time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
	}
	state := WindowState{ExpectedSlots: 11, ReceivedSlots: 11}
	coverage := finalizationCoverage{SourceExpectedSlots: 11, SourceReceivedSlots: 11}

	query, _, err := finalizationClaimUpdate(key, CloseTimeout, state, coverage, nil).ToSql()
	if err != nil {
		t.Fatalf("build finalization claim SQL: %v", err)
	}
	for _, field := range []string{"version_effective_from", "version_effective_to"} {
		if strings.Contains(query, field) {
			t.Fatalf("finalization claim SQL %q must preserve existing %q when snapshot is missing", query, field)
		}
	}
}
