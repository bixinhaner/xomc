package stream

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/prometheus/client_golang/prometheus"
)

func TestRebuildClaimBatchCoalescesUntilDatabaseQuietPeriod(t *testing.T) {
	query, args, err := rebuildClaimBatchSelect(2*time.Minute, 100).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "requested_at <= now() - (") ||
		!strings.Contains(query, "interval '1 microsecond'") {
		t.Fatalf("claim query does not enforce a database-clock quiet period: %s", query)
	}
	if !strings.Contains(query, "FOR UPDATE SKIP LOCKED") {
		t.Fatalf("claim query is not safe for concurrent rebuilders: %s", query)
	}
	if !strings.Contains(fmt.Sprint(args), "120000000") {
		t.Fatalf("claim query does not carry the two-minute quiet period: %v", args)
	}
}

func TestRebuildCoalesceGenerationRunsAtMostOneFollowUp(t *testing.T) {
	if got := rebuildCompletionStatus(100, 100); got != "completed" {
		t.Fatalf("stable generation status = %q, want completed", got)
	}
	if got := rebuildCompletionStatus(100, 101); got != "pending" {
		t.Fatalf("one late generation status = %q, want pending", got)
	}
	if got := rebuildCompletionStatus(100, 200); got != "pending" {
		t.Fatalf("many late generations status = %q, want one pending follow-up", got)
	}
}

func TestRebuildCoalesceDefersParentUntilChildGenerationStable(t *testing.T) {
	if rebuildGenerationStable(RebuildJob{RequestGeneration: 7}, 8) {
		t.Fatal("parent rebuild must not be enqueued while the child generation changed")
	}
	if !rebuildGenerationStable(RebuildJob{RequestGeneration: 8}, 8) {
		t.Fatal("stable child generation must allow exactly one parent enqueue")
	}
}

func TestRebuildCoalesceGroupsSameSourceVersionsAndPeriod(t *testing.T) {
	deviceRollupVersionID := uuid.New()
	ruleVersionID := uuid.New()
	start := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{
			TaskID: uuid.New(), VersionID: deviceRollupVersionID,
			Technology: "LTE", DeviceRollup: true,
		},
		{
			TaskID: uuid.New(), VersionID: ruleVersionID,
			Technology: "LTE", Enabled: true,
		},
	})
	jobs := make([]RebuildJob, 100)
	for index := range jobs {
		jobs[index].Key = WindowKey{
			TaskVersionID: ruleVersionID,
			EntityKey:     fmt.Sprintf("group-%03d", index),
			Granularity:   GranularityHourly,
			Start:         start,
			End:           start.Add(time.Hour),
		}
	}

	groups, ungrouped := groupRollupRebuildJobs(jobs, snapshot)

	if len(ungrouped) != 0 {
		t.Fatalf("rollup rebuilds left ungrouped: %d", len(ungrouped))
	}
	if len(groups) != 1 {
		t.Fatalf("shared source-period groups = %d, want 1", len(groups))
	}
	if len(groups[0].Jobs) != 100 {
		t.Fatalf("jobs in shared scan = %d, want 100", len(groups[0].Jobs))
	}
	if len(groups[0].SourceVersionIDs) != 1 ||
		groups[0].SourceVersionIDs[0] != deviceRollupVersionID {
		t.Fatalf("source versions = %v, want [%s]",
			groups[0].SourceVersionIDs, deviceRollupVersionID)
	}
}

func TestRebuildCoalesceMetricsAreRegistered(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	metrics := NewMetrics(registry)
	metrics.RebuildBatchesTotal.Inc()
	metrics.RebuildJobsPerBatch.Observe(2)
	metrics.RebuildSnapshotRowsTotal.Add(3)
	metrics.RebuildSnapshotScanSeconds.Observe(0.01)
	metrics.RebuildCoalescedTotal.Add(4)

	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(families))
	for _, family := range families {
		got[family.GetName()] = true
	}
	for _, name := range []string{
		"omc_pm_aggregation_rebuild_batches_total",
		"omc_pm_aggregation_rebuild_jobs_per_batch",
		"omc_pm_aggregation_rebuild_snapshot_rows_total",
		"omc_pm_aggregation_rebuild_snapshot_scan_seconds",
		"omc_pm_aggregation_rebuild_coalesced_total",
	} {
		if !got[name] {
			t.Errorf("metric %s is not registered", name)
		}
	}
}

func TestRebuildParentLineageUsesStableDeviceRollupVersion(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	hourlyVersionID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	rollupVersionID := uuid.MustParse("30000000-0000-4000-8000-000000000001")
	snapshot := &TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
		hourlyVersionID: {
			TaskID: taskID, VersionID: hourlyVersionID,
			DevicePipeline: true, RollupVersionID: rollupVersionID,
		},
	}}
	key := WindowKey{
		TaskID: taskID, TaskVersionID: hourlyVersionID,
		Granularity: GranularityHourly,
	}

	gotTaskID, gotVersionID := rebuildParentLineage(key, snapshot)

	if gotTaskID != taskID || gotVersionID != rollupVersionID {
		t.Fatalf(
			"device parent lineage = (%s, %s), want (%s, %s)",
			gotTaskID, gotVersionID, taskID, rollupVersionID,
		)
	}
}

func TestRebuildParentLineageKeepsOrdinaryTaskVersion(t *testing.T) {
	taskID, versionID := uuid.New(), uuid.New()
	snapshot := &TaskSnapshot{ByVersion: map[uuid.UUID]*TaskVersionSnapshot{
		versionID: {TaskID: taskID, VersionID: versionID},
	}}

	gotTaskID, gotVersionID := rebuildParentLineage(WindowKey{
		TaskID: taskID, TaskVersionID: versionID,
	}, snapshot)

	if gotTaskID != taskID || gotVersionID != versionID {
		t.Fatalf("ordinary lineage changed to (%s, %s)", gotTaskID, gotVersionID)
	}
}

func TestPublishedParentsSelectRestrictsTaskAndVersionLineage(t *testing.T) {
	taskID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	versionID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	job := RebuildJob{Key: WindowKey{
		EntityKey: "Network", Granularity: GranularityHourly,
		Start: time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC),
	}}

	query, args, err := publishedParentsSelect(
		job, GranularityDaily, taskID, versionID, "cascade:1:daily",
	).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "task_id = ") ||
		!strings.Contains(query, "task_version_id = ") {
		t.Fatalf("parent select lacks lineage predicates: %s", query)
	}
	joinedArgs := fmt.Sprint(args)
	if !strings.Contains(joinedArgs, taskID.String()) ||
		!strings.Contains(joinedArgs, versionID.String()) {
		t.Fatalf("parent select args lack lineage IDs: %v", args)
	}
}

func TestRebuildCascadeTargets(t *testing.T) {
	start := time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC)
	hour := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		EntityKey: "device-1", Granularity: GranularityHourly,
		Start: start, End: start.Add(time.Hour),
	}
	got := RebuildCascadeTargets(hour)
	want := []Granularity{GranularityDaily}
	if !equalGranularities(got, want) {
		t.Fatalf("hour cascade = %v, want %v", got, want)
	}

	day := hour
	day.Granularity = GranularityDaily
	got = RebuildCascadeTargets(day)
	want = []Granularity{GranularityWeekly, GranularityMonthly}
	if !equalGranularities(got, want) {
		t.Fatalf("day cascade = %v, want %v", got, want)
	}
}

func TestRebuildSourceRoutingUsesRawOnlyForDevicePipelineHours(t *testing.T) {
	pipelineID, ruleID := uuid.New(), uuid.New()
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{TaskID: uuid.New(), VersionID: pipelineID, DevicePipeline: true},
		{TaskID: uuid.New(), VersionID: ruleID},
	})

	if !rebuildUsesRawSources(WindowKey{
		TaskVersionID: pipelineID, Granularity: GranularityHourly,
	}, snapshot) {
		t.Fatal("device-pipeline hour must use normalized raw replay")
	}
	if rebuildUsesRawSources(WindowKey{
		TaskVersionID: ruleID, EntityKey: "Network",
		Granularity: GranularityHourly,
	}, snapshot) {
		t.Fatal("ordinary rule hour must use stable device-hour rollups")
	}
	if rebuildUsesRawSources(WindowKey{
		TaskVersionID: pipelineID, Granularity: GranularityDaily,
	}, snapshot) {
		t.Fatal("daily rebuild must use compact hourly rollups")
	}
}

func TestDeviceRollupVersionsForRuleFiltersTechnology(t *testing.T) {
	lteID, gsmID, ordinaryID := uuid.New(), uuid.New(), uuid.New()
	snapshot := BuildTaskSnapshot([]*TaskVersionSnapshot{
		{
			TaskID: uuid.New(), VersionID: lteID,
			Technology: "LTE", DeviceRollup: true,
		},
		{
			TaskID: uuid.New(), VersionID: gsmID,
			Technology: "gsm", DeviceRollup: true,
		},
		{TaskID: uuid.New(), VersionID: ordinaryID, Technology: "lte"},
	})

	got := deviceRollupVersionsForRule(snapshot, &TaskVersionSnapshot{
		Technology: "lte",
	})

	if len(got) != 1 || got[0] != lteID {
		t.Fatalf("LTE replay versions = %v, want [%s]", got, lteID)
	}
}

func TestRebuildRetryDelayBacksOffAndCaps(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{8, 256 * time.Second},
		{20, 5 * time.Minute},
	}
	for _, test := range tests {
		if got := rebuildRetryDelay(test.attempt); got != test.want {
			t.Fatalf("rebuildRetryDelay(%d) = %s, want %s", test.attempt, got, test.want)
		}
	}
}

func TestRebuildCompletedAtExpressionUsesTimestampType(t *testing.T) {
	query, _, err := storage.Psql.Update("pm_aggregation_rebuilds").
		Set("completed_at", rebuildCompletedAtExpr(3, time.Now().UTC())).
		ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "NULL::timestamptz") ||
		!strings.Contains(query, "::timestamptz") {
		t.Fatalf("completed_at expression lacks timestamp casts: %s", query)
	}
}

func equalGranularities(got, want []Granularity) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
