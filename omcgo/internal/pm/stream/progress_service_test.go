package stream

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProgressServiceLoadSnapshotReusesUnchangedCatalog(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, Fingerprint: "stable"},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	service := NewProgressService(nil, nil, loader)

	first, err := service.loadSnapshot(context.Background())
	if err != nil {
		t.Fatalf("first loadSnapshot returned error: %v", err)
	}
	second, err := service.loadSnapshot(context.Background())
	if err != nil {
		t.Fatalf("second loadSnapshot returned error: %v", err)
	}
	loadCount, _ := loader.counts()
	if loadCount != 1 {
		t.Fatalf("full catalog loads = %d, want 1", loadCount)
	}
	if first != second {
		t.Fatal("unchanged progress catalog did not reuse the published snapshot")
	}
}

func TestProgressServiceUsesPrimedStartupSnapshot(t *testing.T) {
	loader := &revisionMatchableLoader{
		revision: MatchableRevision{TaskCount: 1, Fingerprint: "stable"},
		versions: []*TaskVersionSnapshot{snapshotTestVersion()},
	}
	snapshot := NewSnapshotStore(loader, nil)
	if err := snapshot.Refresh(context.Background()); err != nil {
		t.Fatalf("prime startup snapshot: %v", err)
	}
	service := NewProgressServiceWithSnapshot(nil, nil, snapshot)

	got, err := service.loadSnapshot(context.Background())
	if err != nil {
		t.Fatalf("load primed progress snapshot: %v", err)
	}

	loadCount, _ := loader.counts()
	if loadCount != 1 {
		t.Fatalf("full catalog loads = %d, want startup load only", loadCount)
	}
	if got != snapshot.Current() {
		t.Fatal("progress service did not retain the primed startup snapshot")
	}
}

func TestBuildCurrentWeeklyPreviewIncludesOpenDailyWithoutPersistedWeeklyWindow(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, location)
	thursday := monday.AddDate(0, 0, 3)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	dailyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: thursday.UTC(), End: thursday.AddDate(0, 0, 1).UTC(),
	}
	daily := WindowState{
		ExpectedSlots: 24, ReceivedSlots: 4,
		Accumulators: []Accumulator{{
			Definition: ContributionValue{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			},
			Sum: 10, Count: 4, Min: 1, Max: 4,
		}},
	}
	originalDaily := daily
	originalDaily.Accumulators = append([]Accumulator(nil), daily.Accumulators...)

	key, preview, coverage, err := buildCurrentWeeklyPreview(
		version, dailyKey, daily, WindowState{}, location,
	)
	if err != nil {
		t.Fatalf("buildCurrentWeeklyPreview returned error: %v", err)
	}
	if !key.Start.Equal(monday.UTC()) ||
		!key.End.Equal(monday.AddDate(0, 0, 7).UTC()) ||
		key.Granularity != GranularityWeekly ||
		key.EntityKey != "network" {
		t.Fatalf("unexpected weekly key: %+v", key)
	}
	if len(preview.Accumulators) != 1 {
		t.Fatalf("preview accumulator count = %d, want 1", len(preview.Accumulators))
	}
	accumulator := preview.Accumulators[0]
	if accumulator.Sum != 10 || accumulator.Count != 4 ||
		accumulator.Min != 1 || accumulator.Max != 4 {
		t.Fatalf("unexpected preview accumulator: %+v", accumulator)
	}
	if coverage.received != 4 || coverage.naturalExpected != 168 ||
		coverage.versionExpected != 168 {
		t.Fatalf("unexpected weekly coverage: %+v", coverage)
	}
	if !reflect.DeepEqual(daily, originalDaily) {
		t.Fatalf("daily input mutated: got %+v want %+v", daily, originalDaily)
	}
}

func TestBuildCurrentWeeklyPreviewMergesPublishedDaysWithOpenDailyExactlyOnce(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, location)
	wednesday := monday.AddDate(0, 0, 2)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday.Add(6 * time.Hour),
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationAvg},
		},
	}
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationAvg,
	}
	dailyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: wednesday.UTC(), End: wednesday.AddDate(0, 0, 1).UTC(),
	}
	daily := WindowState{
		ExpectedSlots: 24, ReceivedSlots: 3,
		Accumulators: []Accumulator{{
			Definition: definition, Sum: 15, Count: 3, Min: 3, Max: 7,
		}},
	}
	weekly := WindowState{
		ExpectedSlots: 7, ReceivedSlots: 2,
		Accumulators: []Accumulator{{
			Definition: definition, Sum: 80, Count: 8, Min: 1, Max: 20,
		}},
	}

	_, preview, coverage, err := buildCurrentWeeklyPreview(
		version, dailyKey, daily, weekly, location,
	)
	if err != nil {
		t.Fatalf("buildCurrentWeeklyPreview returned error: %v", err)
	}
	accumulator := preview.Accumulators[0]
	if accumulator.Sum != 95 || accumulator.Count != 11 ||
		accumulator.Min != 1 || accumulator.Max != 20 {
		t.Fatalf("unexpected merged accumulator: %+v", accumulator)
	}
	if coverage.received != 51 || coverage.naturalExpected != 168 ||
		coverage.versionExpected != 162 {
		t.Fatalf("unexpected merged coverage: %+v", coverage)
	}
}

func TestBuildCurrentWeeklyPreviewFromDailiesIncludesEveryOpenDayOnce(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, location)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
	}
	dailyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: monday.AddDate(0, 0, 2).UTC(), End: monday.AddDate(0, 0, 3).UTC(),
	}
	dailies := []WindowState{
		{
			ExpectedSlots: 24, ReceivedSlots: 20,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 20, Count: 20, Min: 1, Max: 1,
			}},
		},
		{
			ExpectedSlots: 24, ReceivedSlots: 3,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 6, Count: 3, Min: 2, Max: 2,
			}},
		},
	}
	weekly := WindowState{
		ExpectedSlots: 7, ReceivedSlots: 2,
		Accumulators: []Accumulator{{
			Definition: definition, Sum: 48, Count: 48, Min: 1, Max: 1,
		}},
	}

	_, preview, coverage, err := buildCurrentWeeklyPreviewFromDailies(
		version, dailyKey, dailies, weekly, location,
	)
	if err != nil {
		t.Fatalf("buildCurrentWeeklyPreviewFromDailies returned error: %v", err)
	}
	accumulator := preview.Accumulators[0]
	if accumulator.Sum != 74 || accumulator.Count != 71 ||
		accumulator.Min != 1 || accumulator.Max != 2 {
		t.Fatalf("unexpected multi-day accumulator: %+v", accumulator)
	}
	if coverage.received != 71 || coverage.naturalExpected != 168 ||
		coverage.versionExpected != 168 {
		t.Fatalf("unexpected multi-day coverage: %+v", coverage)
	}
}

func TestBuildProgressQueryResultSynthesizesCurrentWeekFromOpenDaily(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, location)
	thursday := monday.AddDate(0, 0, 3)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"K1": {
				MetricID: "K1", MetricPath: "K1", MetricType: "kpi",
				Formula: "C1", Dependencies: []string{"C1"},
			},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	key := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: thursday.UTC(), End: thursday.AddDate(0, 0, 1).UTC(),
	}
	candidate := progressCandidate{
		key: key, revision: 3, status: "open",
		openedAt: thursday.UTC(), received: 4, expected: 24,
	}
	state := WindowState{
		ExpectedSlots: 24, ReceivedSlots: 4,
		Accumulators: []Accumulator{{
			Definition: ContributionValue{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			},
			Sum: 42, Count: 4, Min: 8, Max: 13,
		}},
	}

	result, err := buildProgressQueryResult(
		[]progressCandidate{candidate},
		map[WindowKey]WindowState{key: state},
		BuildTaskSnapshot([]*TaskVersionSnapshot{version}),
		location,
	)
	if err != nil {
		t.Fatalf("buildProgressQueryResult returned error: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows = %d, want daily and weekly previews", len(result.Rows))
	}
	rows := make(map[Granularity]ProgressResult, len(result.Rows))
	for _, row := range result.Rows {
		rows[row.Granularity] = row
	}
	if rows[GranularityDaily].Value != 42 ||
		rows[GranularityDaily].ReceivedSlots != 4 ||
		rows[GranularityDaily].ExpectedSlots != 24 {
		t.Fatalf("unexpected daily preview: %+v", rows[GranularityDaily])
	}
	weekly := rows[GranularityWeekly]
	if weekly.Value != 42 || weekly.ReceivedSlots != 4 ||
		weekly.ExpectedSlots != 168 || weekly.VersionExpectedSlots != 168 ||
		!weekly.Partial {
		t.Fatalf("unexpected weekly preview: %+v", weekly)
	}
	if len(result.Periods) != 2 {
		t.Fatalf("periods = %d, want daily and weekly", len(result.Periods))
	}
	periods := make(map[Granularity]PeriodProgress, len(result.Periods))
	for _, period := range result.Periods {
		periods[period.Granularity] = period
	}
	if periods[GranularityWeekly].ReceivedSlots != 4 ||
		periods[GranularityWeekly].ExpectedSlots != 168 ||
		periods[GranularityWeekly].TaskVersionID != version.VersionID ||
		periods[GranularityWeekly].VersionExpectedSlots != 168 ||
		periods[GranularityWeekly].CoverageRatio != float64(4)/168 ||
		periods[GranularityWeekly].VersionSliceComplete ||
		periods[GranularityWeekly].PeriodComplete ||
		periods[GranularityWeekly].State != "partial" {
		t.Fatalf("unexpected weekly period: %+v", periods[GranularityWeekly])
	}
}

func TestBuildProgressQueryResultDropsNonNaturalWeeklyDuplicate(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, location)
	tuesdayBusinessDay := time.Date(2026, 8, 4, 8, 0, 0, 0, location)
	versionEffective := time.Date(2026, 8, 3, 17, 0, 0, 0, location)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: versionEffective,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
	}
	dailyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: tuesdayBusinessDay.UTC(),
		End:   tuesdayBusinessDay.AddDate(0, 0, 1).UTC(),
	}
	nonNaturalWeeklyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityWeekly,
		Start: monday.Add(8 * time.Hour).UTC(),
		End:   monday.AddDate(0, 0, 7).Add(8 * time.Hour).UTC(),
	}
	candidates := []progressCandidate{
		{
			key: dailyKey, revision: 2, status: "open",
			received: 1, expected: 24,
		},
		{
			key: nonNaturalWeeklyKey, revision: 2, status: "open",
			received: 1, expected: 7,
		},
	}
	states := map[WindowKey]WindowState{
		dailyKey: {
			ExpectedSlots: 24, ReceivedSlots: 1,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 10, Count: 1, Min: 10, Max: 10,
			}},
		},
		nonNaturalWeeklyKey: {
			ExpectedSlots: 7, ReceivedSlots: 1,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 90, Count: 24, Min: 1, Max: 8,
			}},
		},
	}

	result, err := buildProgressQueryResult(
		candidates,
		states,
		BuildTaskSnapshot([]*TaskVersionSnapshot{version}),
		location,
	)
	if err != nil {
		t.Fatalf("buildProgressQueryResult returned error: %v", err)
	}

	var weeklyPeriods []PeriodProgress
	for _, period := range result.Periods {
		if period.Granularity == GranularityWeekly {
			weeklyPeriods = append(weeklyPeriods, period)
		}
	}
	if len(weeklyPeriods) != 1 {
		t.Fatalf("weekly period count = %d, want 1: %+v", len(weeklyPeriods), weeklyPeriods)
	}
	weekly := weeklyPeriods[0]
	if !weekly.WindowStart.Equal(monday.UTC()) ||
		!weekly.WindowEnd.Equal(monday.AddDate(0, 0, 7).UTC()) {
		t.Fatalf("weekly period used non-natural window: %+v", weekly)
	}
	if weekly.ReceivedSlots != 1 ||
		weekly.ExpectedSlots != 168 ||
		weekly.VersionExpectedSlots != 151 ||
		weekly.CoverageRatio != float64(1)/168 {
		t.Fatalf("unexpected weekly progress coverage: %+v", weekly)
	}
}

func TestBuildProgressQueryResultReportsWeeklyNaturalHourlyCoverage(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, location)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday.Add(2 * time.Hour),
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	key := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityWeekly,
		Start: monday.UTC(), End: monday.AddDate(0, 0, 7).UTC(),
	}
	candidate := progressCandidate{
		key: key, revision: 1, status: "open",
		received: 1, expected: 7,
	}
	state := WindowState{
		ExpectedSlots: 7, ReceivedSlots: 1,
		Accumulators: []Accumulator{{
			Definition: ContributionValue{
				Dimension: DimensionNetwork, DimensionKey: "network",
				MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
			},
			Sum: 10, Count: 1, Min: 10, Max: 10,
		}},
	}

	result, err := buildProgressQueryResult(
		[]progressCandidate{candidate},
		map[WindowKey]WindowState{key: state},
		BuildTaskSnapshot([]*TaskVersionSnapshot{version}),
		location,
	)
	if err != nil {
		t.Fatalf("buildProgressQueryResult returned error: %v", err)
	}
	if len(result.Periods) != 1 {
		t.Fatalf("periods = %d, want 1: %+v", len(result.Periods), result.Periods)
	}
	period := result.Periods[0]
	if period.ExpectedSlots != 168 ||
		period.VersionExpectedSlots != 7 ||
		period.CoverageRatio != float64(1)/168 {
		t.Fatalf("unexpected weekly period coverage: %+v", period)
	}
}

func TestBuildProgressQueryResultDropsNonNaturalWeeklyDuplicateWithoutDaily(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, location)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday.Add(2 * time.Hour),
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
	}
	naturalKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityWeekly,
		Start: monday.UTC(), End: monday.AddDate(0, 0, 7).UTC(),
	}
	boundaryKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityWeekly,
		Start: monday.Add(8 * time.Hour).UTC(),
		End:   monday.AddDate(0, 0, 7).Add(8 * time.Hour).UTC(),
	}
	stateFor := func(value float64) WindowState {
		return WindowState{
			ExpectedSlots: 7, ReceivedSlots: 1,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: value, Count: 1, Min: value, Max: value,
			}},
		}
	}

	result, err := buildProgressQueryResult(
		[]progressCandidate{
			{key: naturalKey, revision: 1, status: "open", received: 1, expected: 7},
			{key: boundaryKey, revision: 1, status: "open", received: 1, expected: 7},
		},
		map[WindowKey]WindowState{
			naturalKey:  stateFor(10),
			boundaryKey: stateFor(20),
		},
		BuildTaskSnapshot([]*TaskVersionSnapshot{version}),
		location,
	)
	if err != nil {
		t.Fatalf("buildProgressQueryResult returned error: %v", err)
	}

	var weeklyPeriods []PeriodProgress
	for _, period := range result.Periods {
		if period.Granularity == GranularityWeekly {
			weeklyPeriods = append(weeklyPeriods, period)
		}
	}
	if len(weeklyPeriods) != 1 {
		t.Fatalf("weekly period count = %d, want 1: %+v", len(weeklyPeriods), weeklyPeriods)
	}
	weekly := weeklyPeriods[0]
	if !weekly.WindowStart.Equal(naturalKey.Start) ||
		!weekly.WindowEnd.Equal(naturalKey.End) {
		t.Fatalf("weekly period used non-natural window: %+v", weekly)
	}
	if weekly.ReceivedSlots != 1 ||
		weekly.ExpectedSlots != 168 ||
		weekly.VersionExpectedSlots != 7 ||
		weekly.CoverageRatio != float64(1)/168 {
		t.Fatalf("unexpected weekly progress coverage: %+v", weekly)
	}
}

func TestBuildProgressQueryResultDoesNotDoubleCountFinalizingDailyIntoWeekly(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, location)
	thursday := monday.AddDate(0, 0, 3)
	version := &TaskVersionSnapshot{
		TaskID: uuid.New(), VersionID: uuid.New(), Enabled: true,
		EffectiveFrom: monday,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly,
		},
		Metrics: map[string]MetricRule{
			"C1": {MetricID: "C1", MetricPath: "C1", MetricType: "counter"},
		},
		Counters: map[string]CounterRule{
			"C1": {MetricPath: "C1", Aggregation: AggregationSum},
		},
	}
	definition := ContributionValue{
		Dimension: DimensionNetwork, DimensionKey: "network",
		MetricPath: "C1", MetricType: "counter", Operation: AggregationSum,
	}
	dailyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityDaily,
		Start: thursday.UTC(), End: thursday.AddDate(0, 0, 1).UTC(),
	}
	weeklyKey := WindowKey{
		TaskID: version.TaskID, TaskVersionID: version.VersionID,
		EntityKey: "network", Granularity: GranularityWeekly,
		Start: monday.UTC(), End: monday.AddDate(0, 0, 7).UTC(),
	}
	candidates := []progressCandidate{
		{
			key: dailyKey, revision: 1, status: "finalizing",
			received: 4, expected: 24,
		},
		{
			key: weeklyKey, revision: 1, status: "open",
			received: 2, expected: 7,
		},
	}
	states := map[WindowKey]WindowState{
		dailyKey: {
			ExpectedSlots: 24, ReceivedSlots: 4,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 15, Count: 4, Min: 2, Max: 5,
			}},
		},
		weeklyKey: {
			ExpectedSlots: 7, ReceivedSlots: 2,
			Accumulators: []Accumulator{{
				Definition: definition, Sum: 80, Count: 48, Min: 1, Max: 3,
			}},
		},
	}

	result, err := buildProgressQueryResult(
		candidates,
		states,
		BuildTaskSnapshot([]*TaskVersionSnapshot{version}),
		location,
	)
	if err != nil {
		t.Fatalf("buildProgressQueryResult returned error: %v", err)
	}
	var weekly ProgressResult
	for _, row := range result.Rows {
		if row.Granularity == GranularityWeekly {
			weekly = row
			break
		}
	}
	if weekly.Value != 80 {
		t.Fatalf("weekly finalizing race value = %v, want persisted 80", weekly.Value)
	}
}

func TestValidateOpenDailyCandidateStatusesRejectsFinalizeDuringRead(t *testing.T) {
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		EntityKey: "network", Granularity: GranularityDaily,
		Start: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	candidates := []progressCandidate{{key: key, status: "open"}}

	err := validateOpenDailyCandidateStatuses(
		candidates,
		map[string]string{progressCandidateIdentity(key): "published"},
	)
	if !errors.Is(err, ErrProgressUnavailable) {
		t.Fatalf("finalized window error = %v, want ErrProgressUnavailable", err)
	}
}

func TestValidateOpenDailyCandidateStatusesAcceptsStableOpenSnapshot(t *testing.T) {
	key := WindowKey{
		TaskID: uuid.New(), TaskVersionID: uuid.New(),
		EntityKey: "network", Granularity: GranularityDaily,
		Start: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	candidates := []progressCandidate{{key: key, status: "open"}}

	err := validateOpenDailyCandidateStatuses(
		candidates,
		map[string]string{progressCandidateIdentity(key): "open"},
	)
	if err != nil {
		t.Fatalf("stable open snapshot returned error: %v", err)
	}
}

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
		{"week", WindowKey{Granularity: GranularityWeekly}, 168},
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

func TestMetricVersionIntervalsUseAllCatalogVersionsNotRollupLineage(t *testing.T) {
	taskID := uuid.New()
	lineageStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	currentStart := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	oldStart := currentStart.Add(-24 * time.Hour)
	oldEnd := currentStart
	oldVersion := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: uuid.New(), VersionNo: 1, Enabled: true,
		EffectiveFrom: oldStart, EffectiveTo: &oldEnd,
		Metrics: map[string]MetricRule{
			"K1": {MetricID: "K1", MetricPath: "K1", MetricType: "kpi"},
		},
	}
	currentVersion := &TaskVersionSnapshot{
		TaskID: taskID, VersionID: uuid.New(), VersionNo: 2, Enabled: true,
		EffectiveFrom: currentStart, LineageEffectiveFrom: lineageStart,
		Metrics: map[string]MetricRule{
			"K1": {MetricID: "K1", MetricPath: "K1", MetricType: "kpi"},
		},
	}

	intervals := metricVersionIntervals(
		[]*TaskVersionSnapshot{currentVersion, oldVersion}, taskID,
	)

	if len(intervals) != 2 {
		t.Fatalf("expected both catalog metric intervals, got %d", len(intervals))
	}
	if intervals[0].MetricPath != "K1" ||
		!intervals[0].EffectiveFrom.Equal(oldStart) ||
		intervals[0].EffectiveTo == nil ||
		!intervals[0].EffectiveTo.Equal(oldEnd) {
		t.Fatalf("unexpected old catalog metric interval: %+v", intervals[0])
	}
	if !intervals[1].EffectiveFrom.Equal(currentStart) {
		t.Fatalf("unexpected current catalog metric interval: %+v", intervals[1])
	}
}

func TestMetricVersionIntervalsRetainEnabledVersionBeforeDisable(t *testing.T) {
	taskID := uuid.New()
	disabledAt := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	oldStart := disabledAt.Add(-24 * time.Hour)
	oldEnd := disabledAt
	versions := []*TaskVersionSnapshot{
		{
			TaskID: taskID, VersionID: uuid.New(), VersionNo: 1, Enabled: true,
			EffectiveFrom: oldStart, EffectiveTo: &oldEnd,
			Metrics: map[string]MetricRule{
				"K1": {MetricID: "K1", MetricPath: "K1", MetricType: "kpi"},
			},
		},
		{
			TaskID: taskID, VersionID: uuid.New(), VersionNo: 2, Enabled: false,
			EffectiveFrom: disabledAt,
			Metrics: map[string]MetricRule{
				"K1": {MetricID: "K1", MetricPath: "K1", MetricType: "kpi"},
			},
		},
	}

	intervals := metricVersionIntervals(versions, taskID)

	if len(intervals) != 1 || !intervals[0].EffectiveFrom.Equal(oldStart) {
		t.Fatalf("expected pre-disable interval, got %+v", intervals)
	}
}
