package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClaimDueUsesLeaseAndSkipLocked(t *testing.T) {
	leaseOwner := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	claimAt := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	dueBefore := claimAt.Add(-12 * time.Minute)
	leaseUntil := claimAt.Add(2 * time.Minute)

	query, args, err := claimDueUpdate(
		GranularityHourly, dueBefore, 4, leaseOwner, leaseUntil.Sub(claimAt),
		claimVersionFilter{}, claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build claim due SQL: %v", err)
	}
	for _, fragment := range []string{
		"WITH candidates AS",
		"FOR UPDATE SKIP LOCKED",
		"finalize_lease_until IS NULL",
		"finalize_lease_until <= CURRENT_TIMESTAMP",
		"finalize_next_attempt_at <= CURRENT_TIMESTAMP",
		"finalize_lease_owner = $",
		"CURRENT_TIMESTAMP + ($",
		"RETURNING",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("claim due SQL %q missing %q", query, fragment)
		}
	}
	for _, want := range []any{
		string(GranularityHourly), dueBefore, leaseOwner, leaseUntil.Sub(claimAt).Seconds(),
	} {
		if !containsSQLArg(args, want) {
			t.Fatalf("claim due args %v missing %v", args, want)
		}
	}
	lastPlaceholder := fmt.Sprintf("$%d", len(args))
	if !strings.Contains(query, lastPlaceholder) {
		t.Fatalf("claim due SQL reuses nested placeholders: last placeholder %s missing from %q", lastPlaceholder, query)
	}
}

func TestClaimDueNewestOrderCanUseDueClaimIndexBackwards(t *testing.T) {
	query, _, err := claimDueUpdate(
		GranularityHourly,
		time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
		32,
		uuid.New(),
		time.Minute,
		claimVersionFilter{},
		claimNewestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build newest claim SQL: %v", err)
	}
	if !strings.Contains(query,
		"w.window_end DESC, w.entity_key DESC, w.task_version_id DESC, w.window_start DESC") {
		t.Fatalf("newest claim order must be the exact reverse of the due-claim index: %q", query)
	}
}

func TestFailFinalizeClaimPersistsRetryAndFencesByClaimToken(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("abababab-abab-4bab-8bab-abababababab"),
		EntityKey:     "poison",
		Granularity:   GranularityHourly,
		Start:         time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
	}
	token := uuid.MustParse("cdcdcdcd-cdcd-4dcd-8dcd-cdcdcdcdcdcd")
	query, args, err := failFinalizeClaimUpdate(
		key, token, errors.New("permanent formula error"), 2*time.Minute,
	).ToSql()
	if err != nil {
		t.Fatalf("build failed finalize claim SQL: %v", err)
	}
	for _, fragment := range []string{
		"status = $",
		"finalize_attempts = finalize_attempts + 1",
		"finalize_next_attempt_at = CURRENT_TIMESTAMP + ($",
		"finalize_lease_owner = $",
		"finalize_lease_owner = $",
		"finalize_lease_until = $",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("failed finalize claim SQL %q missing %q", query, fragment)
		}
	}
	for _, want := range []any{"failed", "permanent formula error", token, (2 * time.Minute).Microseconds()} {
		if !containsSQLArg(args, want) {
			t.Fatalf("failed finalize claim args %v missing %v", args, want)
		}
	}
}

func TestListActiveAfterSkipsFailedWindowsInBackoff(t *testing.T) {
	query, _, err := listActiveAfterSelect(nil, 100).ToSql()
	if err != nil {
		t.Fatalf("build active recovery SQL: %v", err)
	}
	for _, fragment := range []string{
		"status IN",
		"status <> 'failed' OR finalize_next_attempt_at <= CURRENT_TIMESTAMP",
		"ORDER BY window_start, task_version_id, entity_key, granularity",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("active recovery SQL %q missing %q", query, fragment)
		}
	}
}

func TestMarkFailedAfterPersistsRecoveryBackoff(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"),
		EntityKey:     "missing-raw",
		Granularity:   GranularityHourly,
		Start:         time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
	}
	query, args, err := markFailedUpdate(
		key, errors.New("raw source missing"), 6*time.Hour,
	).ToSql()
	if err != nil {
		t.Fatalf("build mark failed SQL: %v", err)
	}
	for _, fragment := range []string{
		"status = $",
		"last_error = $",
		"finalize_next_attempt_at = CURRENT_TIMESTAMP + ($",
		"status <> $",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("mark failed SQL %q missing %q", query, fragment)
		}
	}
	for _, want := range []any{"failed", "raw source missing", (6 * time.Hour).Microseconds()} {
		if !containsSQLArg(args, want) {
			t.Fatalf("mark failed args %v missing %v", args, want)
		}
	}
}

func TestMarkFailedAfterReturnsRepositoryErrors(t *testing.T) {
	repo := &WindowRepository{}
	err := repo.MarkFailedAfter(context.Background(), WindowKey{}, errors.New("raw source missing"), time.Hour)
	if err == nil || !strings.Contains(err.Error(), "repository is not configured") {
		t.Fatalf("MarkFailedAfter error = %v, want repository configuration failure", err)
	}
}

func TestRecoveryTerminalStatusAllowsAbandonedRaw15mWindows(t *testing.T) {
	for _, status := range []string{"orphaned", "retired", "abandoned"} {
		if !validRecoveryTerminalStatus(status) {
			t.Fatalf("recovery terminal status %q must be supported", status)
		}
	}
	for _, status := range []string{"failed", "published", "rebuilding"} {
		if validRecoveryTerminalStatus(status) {
			t.Fatalf("recovery terminal status %q must not be accepted", status)
		}
	}
}

func TestOldestDueQueryUsesDatabaseTimeAndIncludesActiveLeases(t *testing.T) {
	grace := map[Granularity]time.Duration{
		GranularityHourly:  12 * time.Minute,
		GranularityDaily:   15 * time.Minute,
		GranularityWeekly:  30 * time.Minute,
		GranularityMonthly: 30 * time.Minute,
	}
	query, args, err := oldestDueSelect(grace, 5*time.Minute).ToSql()
	if err != nil {
		t.Fatalf("build oldest due query: %v", err)
	}
	for _, fragment := range []string{
		"CURRENT_TIMESTAMP",
		"MIN(eligible.due_at)",
		"w.window_end <= CURRENT_TIMESTAMP - ($",
		"status IN ($",
		"pm_aggregation_outbox",
		"pm_aggregation_rollup_outbox",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("oldest due SQL %q missing %q", query, fragment)
		}
	}
	for _, forbidden := range []string{
		"finalize_lease_owner",
		"finalize_lease_until",
		"finalize_next_attempt_at",
	} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("oldest due SQL %q must include eligible rows regardless of %q", query, forbidden)
		}
	}
	for _, want := range []any{
		(12 * time.Minute).Microseconds(),
		(15 * time.Minute).Microseconds(),
		(30 * time.Minute).Microseconds(),
	} {
		if !containsSQLArg(args, want) {
			t.Fatalf("oldest due args %v missing grace %v", args, want)
		}
	}
}

func TestCountWatermarkBlockedGroupsSharedPeriodsBeforeQueueProbe(t *testing.T) {
	now := time.Date(2026, 7, 31, 6, 30, 0, 0, time.UTC)
	query, args, err := countWatermarkBlockedSelect(now, map[Granularity]time.Duration{
		GranularityHourly:  12 * time.Minute,
		GranularityDaily:   15 * time.Minute,
		GranularityWeekly:  30 * time.Minute,
		GranularityMonthly: 30 * time.Minute,
	}, 5*time.Minute).ToSql()
	if err != nil {
		t.Fatalf("build count watermark-blocked SQL: %v", err)
	}
	for _, fragment := range []string{
		"GROUP BY w.granularity, w.window_start, w.window_end",
		"SUM(due_windows.window_count)",
		"source_event.event_window_start >= due_windows.window_start",
		"source_rollup.window_start >= due_windows.window_start",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("watermark-blocked SQL %q missing grouped-period fragment %q", query, fragment)
		}
	}
	if strings.Contains(query, "source_event.event_window_start >= w.window_start") {
		t.Fatalf("watermark-blocked SQL must not probe the queue once per window: %q", query)
	}
	for _, want := range []any{
		now.Add(-12 * time.Minute),
		now.Add(-15 * time.Minute),
		now.Add(-30 * time.Minute),
	} {
		if !containsSQLArg(args, want) {
			t.Fatalf("watermark-blocked args %v missing cutoff %v", args, want)
		}
	}
}

func TestClaimDueTwoScannersCannotClaimAnUnexpiredLease(t *testing.T) {
	firstOwner := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	secondOwner := uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	claimAt := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)

	firstQuery, _, err := claimDueUpdate(
		GranularityHourly, claimAt, 1, firstOwner, 2*time.Minute,
		claimVersionFilter{}, claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build first scanner claim: %v", err)
	}
	secondQuery, secondArgs, err := claimDueUpdate(
		GranularityHourly, claimAt, 1, secondOwner, 2*time.Minute,
		claimVersionFilter{}, claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build second scanner claim: %v", err)
	}
	if !strings.Contains(firstQuery, "FOR UPDATE SKIP LOCKED") ||
		!strings.Contains(secondQuery, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("both scanner claims must skip rows locked by the other transaction")
	}
	if !strings.Contains(secondQuery, "finalize_lease_until <= CURRENT_TIMESTAMP") ||
		!containsSQLArg(secondArgs, (2*time.Minute).Seconds()) {
		t.Fatalf("second scanner claim must exclude the first scanner's unexpired database-time lease")
	}
}

func TestClaimDueExpiredLeaseCanBeReclaimedAfterProcessExit(t *testing.T) {
	leaseOwner := uuid.MustParse("cccccccc-cccc-4ccc-8ccc-cccccccccccc")
	restartedAt := time.Date(2026, 7, 30, 10, 5, 0, 0, time.UTC)

	query, args, err := claimDueUpdate(
		GranularityDaily, restartedAt, 1, leaseOwner, 2*time.Minute,
		claimVersionFilter{}, claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build expired lease reclaim: %v", err)
	}
	if !strings.Contains(query, "finalize_lease_until <= CURRENT_TIMESTAMP") ||
		!containsSQLArg(args, (2*time.Minute).Seconds()) {
		t.Fatalf("claim SQL must make database-time expired leases eligible: %q %v", query, args)
	}
}

func TestClaimDueVersionFilterUsesOneArrayArgument(t *testing.T) {
	versionIDs := []uuid.UUID{
		uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		uuid.MustParse("22222222-2222-4222-8222-222222222222"),
	}
	claimAt := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	query, args, err := claimDueUpdate(
		GranularityHourly, claimAt, 1, uuid.New(), time.Minute,
		claimVersionFilter{versionIDs: versionIDs}, claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build version-filtered claim: %v", err)
	}
	if !strings.Contains(query, "task_version_id = ANY($") {
		t.Fatalf("version-filtered claim SQL = %q, want one PostgreSQL array predicate", query)
	}
	arrayArgs := 0
	for _, arg := range args {
		if _, ok := arg.([]uuid.UUID); ok {
			arrayArgs++
		}
	}
	if arrayArgs != 1 {
		t.Fatalf("version-filtered claim args = %v, want exactly one UUID array", args)
	}
}

func TestClaimDueHourlyRuleWaitsForDeviceWindowsAndRollupConsumption(t *testing.T) {
	deviceVersionIDs := []uuid.UUID{
		uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		uuid.MustParse("22222222-2222-4222-8222-222222222222"),
	}
	query, args, err := claimDueUpdate(
		GranularityHourly,
		time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
		32,
		uuid.New(),
		time.Minute,
		claimVersionFilter{versionIDs: deviceVersionIDs, exclude: true},
		claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build hourly rule claim: %v", err)
	}
	for _, fragment := range []string{
		"source_window.task_version_id = ANY($",
		"source_window.window_start = w.window_start",
		"source_window.status IN ('open', 'failed', 'finalizing', 'rebuilding')",
		"source_rollup.subject = $",
		"source_rollup.window_start = w.window_start",
		"source_rollup.consumed_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("hourly rule claim SQL %q missing hierarchy barrier %q", query, fragment)
		}
	}
	for _, want := range []any{"pmaggregation.hourly.rollup"} {
		if !containsSQLArg(args, want) {
			t.Fatalf("hourly rule claim args %v missing %v", args, want)
		}
	}
}

func TestClaimDueHourlyDeviceDoesNotWaitForItsOwnFutureRollup(t *testing.T) {
	query, _, err := claimDueUpdate(
		GranularityHourly,
		time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
		32,
		uuid.New(),
		time.Minute,
		claimVersionFilter{
			versionIDs: []uuid.UUID{
				uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			},
		},
		claimOldestFirst,
	).ToSql()
	if err != nil {
		t.Fatalf("build hourly device claim: %v", err)
	}
	for _, forbidden := range []string{
		"source_window.task_version_id",
		"source_rollup.window_start = w.window_start",
	} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("hourly device claim SQL %q must not self-block on %q", query, forbidden)
		}
	}
}

func TestClaimDueConflictUsesOnlyUnexpiredDatabaseTimeLeases(t *testing.T) {
	dueBefore := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	query, args, err := claimConflictSelect(
		GranularityHourly, dueBefore, claimVersionFilter{},
		uuid.MustParse("77777777-7777-4777-8777-777777777777"),
	).ToSql()
	if err != nil {
		t.Fatalf("build claim conflict SQL: %v", err)
	}
	for _, fragment := range []string{
		"finalize_lease_until > CURRENT_TIMESTAMP",
		"finalize_lease_owner <> $",
		"window_end <= $",
		"LIMIT 1",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("claim conflict SQL %q missing %q", query, fragment)
		}
	}
	if !containsSQLArg(args, dueBefore) {
		t.Fatalf("claim conflict args %v missing due cutoff %v", args, dueBefore)
	}
}

func TestCompleteAndReleaseClaimCannotClearAnotherScannerLease(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("dddddddd-dddd-4ddd-8ddd-dddddddddddd"),
		EntityKey:     "device-1",
		Granularity:   GranularityHourly,
		Start:         time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
	}
	owner := uuid.MustParse("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
	for _, name := range []string{"complete", "release"} {
		t.Run(name, func(t *testing.T) {
			query, args, err := clearFinalizeClaimUpdate(key, owner).ToSql()
			if err != nil {
				t.Fatalf("build %s claim SQL: %v", name, err)
			}
			if !strings.Contains(query, "finalize_lease_owner = $") ||
				!containsSQLArg(args, owner) {
				t.Fatalf("%s claim SQL must require the current lease owner: %q %v", name, query, args)
			}
		})
	}
}

func TestRenewFinalizeClaimUsesDatabaseClockAndClaimToken(t *testing.T) {
	key := WindowKey{
		TaskVersionID: uuid.MustParse("12121212-1212-4212-8212-121212121212"),
		EntityKey:     "device-1",
		Granularity:   GranularityHourly,
		Start:         time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
	}
	token := uuid.MustParse("34343434-3434-4434-8434-343434343434")
	query, args, err := renewFinalizeClaimUpdate(key, token, 2*time.Minute).ToSql()
	if err != nil {
		t.Fatalf("build renew finalize claim SQL: %v", err)
	}
	for _, fragment := range []string{
		"finalize_lease_until = CURRENT_TIMESTAMP + ($",
		"updated_at = CURRENT_TIMESTAMP",
		"finalize_lease_owner = $",
		"finalize_lease_until > CURRENT_TIMESTAMP",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("renew finalize claim SQL %q missing %q", query, fragment)
		}
	}
	if !containsSQLArg(args, token) {
		t.Fatalf("renew finalize claim args %v missing token %s", args, token)
	}
}

func containsSQLArg(args []any, want any) bool {
	for _, arg := range args {
		if fmt.Sprint(arg) == fmt.Sprint(want) {
			return true
		}
	}
	return false
}

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

func TestVersionMetadataBackfillUpdateIsBoundedToBatch(t *testing.T) {
	versionID := uuid.MustParse("77777777-7777-4777-8777-777777777777")
	effectiveFrom := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	effectiveTo := effectiveFrom.Add(24 * time.Hour)
	version := &TaskVersionSnapshot{
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   &effectiveTo,
	}

	query, args := versionMetadataBackfillUpdateSQL(versionID, version, 123)
	for _, fragment := range []string{
		"WITH candidates AS",
		"version_effective_from IS NULL",
		"ORDER BY entity_key, granularity, window_start",
		"LIMIT $4",
		"UPDATE pm_aggregation_windows AS w",
		"FROM candidates AS c",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("version metadata backfill SQL %q missing %q", query, fragment)
		}
	}
	for _, forbidden := range []string{
		"UPDATE pm_aggregation_windows SET version_effective_from",
		"LIMIT 123",
	} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("version metadata backfill SQL %q must keep bounded placeholders, found %q", query, forbidden)
		}
	}
	for _, want := range []any{effectiveFrom, effectiveTo, versionID, uint64(123)} {
		if !containsSQLArg(args, want) {
			t.Fatalf("version metadata backfill args %v missing %v", args, want)
		}
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

func TestFinalizationClaimedWindowFencesStateTransitionByLeaseOwner(t *testing.T) {
	owner := uuid.MustParse("ffffffff-ffff-4fff-8fff-ffffffffffff")
	key := WindowKey{
		TaskVersionID: uuid.MustParse("66666666-6666-4666-8666-666666666666"),
		EntityKey:     "device-1",
		Granularity:   GranularityHourly,
		Start:         time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC),
	}
	query, args, err := finalizationClaimUpdateForOwner(
		key,
		CloseTimeout,
		WindowState{ExpectedSlots: 4},
		finalizationCoverage{},
		nil,
		&owner,
	).ToSql()
	if err != nil {
		t.Fatalf("build owner-fenced finalization claim: %v", err)
	}
	if !strings.Contains(query, "finalize_lease_owner = $") ||
		!strings.Contains(query, "finalize_lease_until > CURRENT_TIMESTAMP") ||
		!containsSQLArg(args, owner) {
		t.Fatalf("finalization claim must fence state transition by owner: %q %v", query, args)
	}
}
