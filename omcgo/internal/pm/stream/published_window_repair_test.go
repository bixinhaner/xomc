package stream

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

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
