package stream

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
)

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
