package stream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPublishedVersionRepairerBoundsZeroHitQueriesAcrossManyVersions(t *testing.T) {
	snapshot := publishedRepairTestSnapshot(1_000)
	store := &fakePublishedVersionRepairBatchStore{}
	repairer := newTestPublishedVersionRepairer(store, snapshot, 8)

	if _, err := repairer.runOnce(context.Background()); err != nil {
		t.Fatalf("run bounded zero-hit repair: %v", err)
	}
	if store.queryCount != 8 {
		t.Fatalf("zero-hit version queries = %d, want hard bound 8", store.queryCount)
	}
	if len(store.batches) != 1 || len(store.batches[0]) != 8 {
		t.Fatalf("zero-hit repair batches = %v, want one eight-version batch", store.batches)
	}
}

func TestPublishedVersionRepairerCursorAdvancesAndWrapsFairly(t *testing.T) {
	snapshot := publishedRepairTestSnapshot(10)
	store := &fakePublishedVersionRepairBatchStore{leaveOpen: true}
	repairer := newTestPublishedVersionRepairer(store, snapshot, 3)

	for range 4 {
		if _, err := repairer.runOnce(context.Background()); err != nil {
			t.Fatalf("run cursor repair cycle: %v", err)
		}
	}
	want := [][]uuid.UUID{
		publishedRepairTestVersionIDs(1, 3),
		publishedRepairTestVersionIDs(4, 6),
		publishedRepairTestVersionIDs(7, 9),
		{
			publishedRepairTestVersionID(10),
			publishedRepairTestVersionID(1),
			publishedRepairTestVersionID(2),
		},
	}
	if fmt.Sprint(store.batches) != fmt.Sprint(want) {
		t.Fatalf("cursor batches = %v, want fair advancing/wrapping %v", store.batches, want)
	}
}

func TestPublishedVersionRepairerStopsQueryingAfterEmptySweep(t *testing.T) {
	snapshot := publishedRepairTestSnapshot(5)
	store := &fakePublishedVersionRepairBatchStore{}
	repairer := newTestPublishedVersionRepairer(store, snapshot, 3)

	for range 3 {
		if _, err := repairer.runOnce(context.Background()); err != nil {
			t.Fatalf("run empty repair sweep: %v", err)
		}
	}
	if store.queryCount != 5 {
		t.Fatalf("queries after completed empty sweep = %d, want exactly one per version", store.queryCount)
	}
	if len(store.batches) != 2 {
		t.Fatalf("repository calls after idle cycle = %d, want 2 before becoming idle", len(store.batches))
	}
}

func TestPublishedVersionRepairerRechecksOnlyChangedSnapshotVersion(t *testing.T) {
	snapshot := publishedRepairTestSnapshot(3)
	store := &fakePublishedVersionRepairBatchStore{}
	repairer := newTestPublishedVersionRepairer(store, snapshot, 3)
	if _, err := repairer.runOnce(context.Background()); err != nil {
		t.Fatalf("run initial repair sweep: %v", err)
	}
	if _, err := repairer.runOnce(context.Background()); err != nil {
		t.Fatalf("run idle repair sweep: %v", err)
	}

	changedID := publishedRepairTestVersionID(2)
	closedAt := snapshot.ByVersion[changedID].EffectiveFrom.Add(11 * time.Hour)
	snapshot.ByVersion[changedID].EffectiveTo = &closedAt
	if _, err := repairer.runOnce(context.Background()); err != nil {
		t.Fatalf("run changed-snapshot repair sweep: %v", err)
	}

	if len(store.batches) != 2 {
		t.Fatalf("repository calls after one version changed = %d, want 2", len(store.batches))
	}
	if got := store.batches[1]; len(got) != 1 || got[0] != changedID {
		t.Fatalf("changed-snapshot batch = %v, want only %s", got, changedID)
	}
}

func TestPublishedVersionRepairDoesNotCompleteVersionWhenAuditFenceIsLost(t *testing.T) {
	if publishedVersionRepairBatchCompleted(0, 10, false) {
		t.Fatal("version with a lost audit fence must remain eligible for a later cycle")
	}
	if !publishedVersionRepairBatchCompleted(9, 10, true) {
		t.Fatal("short fully audited batch must complete the version")
	}
	if publishedVersionRepairBatchCompleted(10, 10, true) {
		t.Fatal("full batch needs one bounded follow-up query before completion")
	}
}

func TestPublishedVersionRepairEnqueuesClosedDailySliceWithoutLateEvent(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	windowStart := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(24 * time.Hour)
	closedAt := windowStart.Add(16 * time.Hour)
	version := &TaskVersionSnapshot{
		TaskID:        uuid.MustParse("10101010-1010-4010-8010-101010101010"),
		VersionID:     uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		EffectiveFrom: windowStart.Add(5 * time.Hour),
		EffectiveTo:   &closedAt,
	}
	candidate := publishedVersionRepairCandidate{
		Window: WindowRecord{
			Key: WindowKey{
				TaskID: version.TaskID, TaskVersionID: version.VersionID,
				EntityKey: "network", Granularity: GranularityDaily,
				Start: windowStart, End: windowEnd,
			},
			Status: "published", ExpectedSlots: 19, ReceivedSlots: 11,
		},
		VersionEffectiveFrom: &version.EffectiveFrom,
		VersionEffectiveTo:   nil,
	}

	action, ok := planPublishedVersionRepair(candidate, version, location)
	if !ok {
		t.Fatal("historical published 19-slot row must be audited")
	}
	if action.ExpectedSlots != 11 {
		t.Fatalf("repaired expected slots = %d, want 11", action.ExpectedSlots)
	}
	if !action.EnqueueRebuild {
		t.Fatal("19-to-11 correction must enqueue a rebuild even without a late event")
	}
	if action.SourceEventID == "" {
		t.Fatal("repair rebuild must have a deterministic source event ID")
	}

	query, args, err := publishedVersionRepairEnqueue(action).ToSql()
	if err != nil {
		t.Fatalf("build repair rebuild enqueue SQL: %v", err)
	}
	for _, fragment := range []string{
		"INSERT INTO pm_aggregation_rebuilds",
		"ON CONFLICT",
		"request_generation = pm_aggregation_rebuilds.request_generation + 1",
		"next_attempt_at = now()",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("repair enqueue SQL %q missing %q", query, fragment)
		}
	}
	if !containsSQLArg(args, action.SourceEventID) {
		t.Fatalf("repair enqueue args %v missing source event ID %q", args, action.SourceEventID)
	}
}

type fakePublishedVersionRepairBatchStore struct {
	batches    [][]uuid.UUID
	locations  []string
	queryCount int
	leaveOpen  bool
}

func (s *fakePublishedVersionRepairBatchStore) RepairPublishedVersionBatch(
	_ context.Context,
	versions []*TaskVersionSnapshot,
	location *time.Location,
	_ uint64,
) (PublishedVersionRepairResult, error) {
	ids := make([]uuid.UUID, 0, len(versions))
	completed := make(map[uuid.UUID]string)
	if location != nil {
		s.locations = append(s.locations, location.String())
	}
	for _, version := range versions {
		ids = append(ids, version.VersionID)
		if !s.leaveOpen {
			completed[version.VersionID] = publishedVersionAuditFingerprint(version, location)
		}
	}
	s.batches = append(s.batches, ids)
	s.queryCount += len(versions)
	return PublishedVersionRepairResult{
		VersionQueries:               len(versions),
		CompletedVersionFingerprints: completed,
	}, nil
}

func newTestPublishedVersionRepairer(
	store publishedVersionRepairBatchStore,
	snapshot *TaskSnapshot,
	versionLimit int,
) *PublishedVersionRepairer {
	snapshots := NewSnapshotStore(nil, nil)
	snapshots.value.Store(snapshot)
	repairer := NewPublishedVersionRepairer(store, snapshots, time.UTC, nil)
	repairer.versionQueryLimit = versionLimit
	return repairer
}

func TestPublishedVersionRepairerUsesCurrentLocationProviderEachRun(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	current := tokyo
	store := &fakePublishedVersionRepairBatchStore{leaveOpen: true}
	snapshots := NewSnapshotStore(nil, nil)
	snapshots.value.Store(publishedRepairTestSnapshot(1))
	repairer := NewPublishedVersionRepairerWithLocationProvider(
		store,
		snapshots,
		func() *time.Location { return current },
		nil,
	)

	_, err = repairer.runOnce(context.Background())
	require.NoError(t, err)

	current = newYork
	_, err = repairer.runOnce(context.Background())
	require.NoError(t, err)

	require.Equal(t, []string{"Asia/Tokyo", "America/New_York"}, store.locations)
}

func publishedRepairTestSnapshot(count int) *TaskSnapshot {
	snapshot := &TaskSnapshot{ByVersion: make(map[uuid.UUID]*TaskVersionSnapshot)}
	for index := 1; index <= count; index++ {
		versionID := publishedRepairTestVersionID(index)
		snapshot.ByVersion[versionID] = &TaskVersionSnapshot{
			TaskID: uuid.New(), VersionID: versionID,
			EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}
	}
	return snapshot
}

func publishedRepairTestVersionID(index int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", index))
}

func publishedRepairTestVersionIDs(from, to int) []uuid.UUID {
	result := make([]uuid.UUID, 0, to-from+1)
	for index := from; index <= to; index++ {
		result = append(result, publishedRepairTestVersionID(index))
	}
	return result
}

func TestPublishedVersionRepairIsIdempotentAfterAudit(t *testing.T) {
	location := time.UTC
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	closedAt := start.Add(11 * time.Hour)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(),
		EffectiveFrom: start, EffectiveTo: &closedAt,
	}
	candidate := publishedVersionRepairCandidate{
		Window: WindowRecord{
			Key: WindowKey{
				TaskID: version.TaskID, TaskVersionID: version.VersionID,
				EntityKey: "network", Granularity: GranularityDaily,
				Start: start, End: start.Add(24 * time.Hour),
			},
			Status: "published", ExpectedSlots: 19, ReceivedSlots: 11,
		},
		VersionEffectiveFrom: &version.EffectiveFrom,
	}

	action, ok := planPublishedVersionRepair(candidate, version, location)
	if !ok || !action.EnqueueRebuild {
		t.Fatalf("first repair action = %+v, ok=%v; want one rebuild", action, ok)
	}
	candidate.Window.ExpectedSlots = action.ExpectedSlots
	candidate.VersionEffectiveFrom = &action.VersionEffectiveFrom
	candidate.VersionEffectiveTo = action.VersionEffectiveTo
	candidate.AuditFingerprint = action.AuditFingerprint

	if repeated, ok := planPublishedVersionRepair(candidate, version, location); ok {
		t.Fatalf("repeated repair = %+v, want no audit or generation bump", repeated)
	}
}

func TestPublishedVersionRepairAuditsRowsWithMissingMetadata(t *testing.T) {
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), EffectiveFrom: start,
	}
	candidate := publishedVersionRepairCandidate{
		Window: WindowRecord{
			Key: WindowKey{
				TaskID: version.TaskID, TaskVersionID: version.VersionID,
				EntityKey: "network", Granularity: GranularityDaily,
				Start: start, End: start.Add(24 * time.Hour),
			},
			Status: "published", ExpectedSlots: 24, ReceivedSlots: 24,
		},
		VersionEffectiveFrom: nil,
	}

	action, ok := planPublishedVersionRepair(candidate, version, time.UTC)
	if !ok || !action.EnqueueRebuild {
		t.Fatalf("missing metadata repair = %+v, ok=%v; want audited rebuild", action, ok)
	}
}

func TestPublishedVersionRepairRemainingUsesScannedRowsAsHardBound(t *testing.T) {
	if got := publishedVersionRepairRemaining(100, 100); got != 0 {
		t.Fatalf("remaining after 100 scanned rows = %d, want 0", got)
	}
	if got := publishedVersionRepairRemaining(100, 40); got != 60 {
		t.Fatalf("remaining after 40 scanned rows = %d, want 60", got)
	}
}

func TestPublishedVersionRepairBatchIsCapped(t *testing.T) {
	version := &TaskVersionSnapshot{
		VersionID: uuid.New(), EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	if got := normalizePublishedVersionRepairLimit(10_000); got != publishedVersionRepairBatchSize {
		t.Fatalf("normalized repair limit = %d, want cap %d", got, publishedVersionRepairBatchSize)
	}
	query, args, err := publishedVersionRepairCandidates(
		version,
		publishedVersionAuditFingerprint(version, time.UTC),
		10_000,
	).ToSql()
	if err != nil {
		t.Fatalf("build bounded repair candidate SQL: %v", err)
	}
	for _, fragment := range []string{
		"status = $",
		"version_audit_fingerprint IS NULL",
		"version_audit_fingerprint <> $",
		"ORDER BY window_start",
		"LIMIT 100",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("repair candidate SQL %q missing %q", query, fragment)
		}
	}
	if !containsSQLArg(args, version.VersionID) {
		t.Fatalf("repair candidate args %v missing version %s", args, version.VersionID)
	}
}
