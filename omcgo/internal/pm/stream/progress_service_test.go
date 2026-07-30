package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildProgressResultsReportsCoverageAndVersionSlice(t *testing.T) {
	start := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: start.Add(6 * time.Hour),
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
	}
	key := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		Granularity: GranularityDaily, Start: start, End: start.Add(24 * time.Hour),
	}
	state := WindowState{
		ExpectedSlots: 1, ReceivedSlots: 1,
		SourceExpectedSlots: 4, SourceReceivedSlots: 4,
		Accumulators: []Accumulator{{
			Definition: ContributionValue{
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
				Dimension: DimensionNetwork, DimensionKey: "Network",
			},
			Sum: 42, Count: 1,
		}},
	}

	rows, err := BuildProgressResults(version, key, 2, state)
	if err != nil {
		t.Fatalf("BuildProgressResults returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if !row.Partial || row.PeriodComplete || row.VersionSliceComplete {
		t.Fatalf("unexpected completeness: %+v", row)
	}
	if row.ReceivedSlots != 1 || row.ExpectedSlots != 24 ||
		row.VersionExpectedSlots != 1 || row.NaturalExpectedSlots != 24 ||
		row.Revision != 2 {
		t.Fatalf("unexpected coverage/version: %+v", row)
	}
	if row.Value != 42 {
		t.Fatalf("value = %v, want 42", row.Value)
	}
}

func TestNaturalExpectedSlotsUsesCalendarPeriods(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monthStart := time.Date(2026, 2, 1, 0, 0, 0, 0, location)
	tests := []struct {
		name string
		key  WindowKey
		want int64
	}{
		{"hour", WindowKey{Granularity: GranularityHourly}, 4},
		{"day", WindowKey{Granularity: GranularityDaily}, 24},
		{"week", WindowKey{Granularity: GranularityWeekly}, 7},
		{
			"calendar month",
			WindowKey{Granularity: GranularityMonthly, Start: monthStart, End: monthStart.AddDate(0, 1, 0)},
			28,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := &TaskVersionSnapshot{DevicePipeline: true}
			if got := naturalExpectedSlots(version, test.key, WindowState{}); got != test.want {
				t.Fatalf("naturalExpectedSlots() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestNaturalExpectedSlotsUsesRuleMemberCountForHourly(t *testing.T) {
	key := WindowKey{Granularity: GranularityHourly}
	state := WindowState{ExpectedSlots: 37}
	version := &TaskVersionSnapshot{DevicePipeline: false}
	if got := naturalExpectedSlots(version, key, state); got != 37 {
		t.Fatalf("rule hourly natural expected slots = %d, want 37", got)
	}
}

func TestProgressVersionSelectionIsTaskWindowScopedAndEffectiveOrdered(t *testing.T) {
	taskID := uuid.New()
	oldVersion := uuid.New()
	newVersion := uuid.New()
	start := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	oldEffective := start.Add(-30 * 24 * time.Hour)
	newEffective := start.Add(-24 * time.Hour)
	old := progressCandidate{
		key: WindowKey{
			TaskID: taskID, TaskVersionID: oldVersion, EntityKey: "removed-entity",
			Granularity: GranularityDaily, Start: start, End: start.Add(24 * time.Hour),
		},
		effective: &oldEffective,
		openedAt:  start.Add(10 * time.Hour),
	}
	current := progressCandidate{
		key: WindowKey{
			TaskID: taskID, TaskVersionID: newVersion, EntityKey: "current-entity",
			Granularity: GranularityDaily, Start: start, End: start.Add(24 * time.Hour),
		},
		effective: &newEffective,
		openedAt:  start,
	}
	if progressVersionGroup(old.key) != progressVersionGroup(current.key) {
		t.Fatal("entities in one natural task window must share the active-version group")
	}
	if !progressVersionAfter(current, old, nil) {
		t.Fatal("newer effective version must win even when the old version opened later")
	}
}

func TestProgressStatusMarksFailedAndRebuildingUnavailable(t *testing.T) {
	for _, status := range []string{"failed", "rebuilding"} {
		if !progressStatusUnavailable(status) {
			t.Fatalf("%s window must make current progress unavailable", status)
		}
	}
	for _, status := range []string{"open", "finalizing"} {
		if progressStatusUnavailable(status) {
			t.Fatalf("%s window must remain readable", status)
		}
	}
}

func TestLowerProgressCoverageIncludesWindowWithoutMetricRows(t *testing.T) {
	full := PeriodProgress{
		EntityKey: "with-metrics", ReceivedSlots: 24, ExpectedSlots: 24,
	}
	empty := PeriodProgress{
		EntityKey: "without-metrics", ReceivedSlots: 0, ExpectedSlots: 24,
	}

	if !lowerProgressCoverage(empty, full) {
		t.Fatal("zero-coverage window without metrics must become task-level minimum")
	}
}
