package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedDashboardTimezoneProvider struct {
	location *time.Location
}

func (p fixedDashboardTimezoneProvider) Location(context.Context) *time.Location {
	return p.location
}

type recordingNetworkRollupReader struct {
	queries []NetworkRollupQuery
	points  []NetworkRollupPoint
	err     error
}

func (r *recordingNetworkRollupReader) ListSeries(_ context.Context, query NetworkRollupQuery) ([]NetworkRollupPoint, error) {
	r.queries = append(r.queries, query)
	return r.points, r.err
}

func (r *recordingNetworkRollupReader) ListLatestHourly(context.Context, time.Time, time.Time) ([]NetworkRollupPoint, error) {
	return r.points, r.err
}

type recordingCounterSeriesReader struct {
	queries []NetworkRollupQuery
	points  []NetworkRollupPoint
}

func (r *recordingCounterSeriesReader) ListSeries(_ context.Context, query NetworkRollupQuery) ([]NetworkRollupPoint, error) {
	r.queries = append(r.queries, query)
	return r.points, nil
}

func TestGetKPITimeSeriesReadsCounterSeriesForDashboard(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)
	counterReader := &recordingCounterSeriesReader{points: []NetworkRollupPoint{{
		Technology:  model.TechNR,
		MetricPath:  "C010070004",
		Granularity: metrics.GranularityHourly,
		WindowStart: start,
		WindowEnd:   start.Add(time.Hour),
		Value:       jsonx.Float(1.25),
	}}}
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		counterSeries:  counterReader,
	}

	result, err := service.GetKPITimeSeries(
		context.Background(), []string{"C010070004"}, model.TechNR,
		metrics.GranularityHourly, start.Add(-time.Hour), now,
	)

	require.NoError(t, err)
	require.Len(t, result["C010070004"], 1)
	assert.Equal(t, jsonx.Float(1.25), result["C010070004"][0].Value)
	require.Len(t, counterReader.queries, 1)
	assert.Equal(t, []string{"C010070004"}, counterReader.queries[0].MetricPaths)
	assert.Equal(t, metrics.MetricTypeCounter, counterReader.queries[0].MetricType)
}

func TestGetKPITimeSeriesKeepsKPIAndCounterReadersSeparate(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	kpiReader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
		Technology: model.TechNR, MetricPath: "KGNB0517",
		Granularity: metrics.GranularityHourly, WindowStart: start,
		WindowEnd: start.Add(time.Hour), Value: jsonx.Float(2.5), Complete: true,
	}}}
	counterReader := &recordingCounterSeriesReader{points: []NetworkRollupPoint{{
		Technology: model.TechNR, MetricPath: "C010070004",
		Granularity: metrics.GranularityHourly, WindowStart: start,
		WindowEnd: start.Add(time.Hour), Value: jsonx.Float(1.25),
	}}}
	service := &Service{networkRollups: kpiReader, counterSeries: counterReader}

	result, err := service.GetKPITimeSeries(
		context.Background(), []string{"KGNB0517", "C010070004"}, model.TechNR,
		metrics.GranularityHourly, start, now.Add(time.Hour),
	)

	require.NoError(t, err)
	assert.Len(t, result["KGNB0517"], 1)
	assert.Len(t, result["C010070004"], 1)
	require.Len(t, kpiReader.queries, 1)
	assert.Equal(t, []string{"KGNB0517"}, kpiReader.queries[0].MetricPaths)
	assert.Equal(t, metrics.MetricTypeKPI, kpiReader.queries[0].MetricType)
	require.Len(t, counterReader.queries, 1)
	assert.Equal(t, []string{"C010070004"}, counterReader.queries[0].MetricPaths)
	assert.Equal(t, metrics.MetricTypeCounter, counterReader.queries[0].MetricType)
}

type fixedNetworkProgressReader struct {
	expectedTaskID uuid.UUID
	result         pmstream.ProgressQueryResult
	err            error
}

func (r *fixedNetworkProgressReader) Query(
	_ context.Context,
	taskID uuid.UUID,
	_, _ time.Time,
) (pmstream.ProgressQueryResult, error) {
	if taskID != r.expectedTaskID {
		return pmstream.ProgressQueryResult{}, errors.New("unexpected network progress task")
	}
	return r.result, r.err
}

func TestGetKPITimeSeriesSnapshotAppendsOnlyRequestedNetworkPartial(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
		Technology: model.TechLTE, MetricPath: "K1",
		Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Value: jsonx.Float(10), Complete: true,
	}}}
	progress := &fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionNetwork, DimensionKey: "network",
					MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
					ReceivedSlots: 4, ExpectedSlots: 24,
				},
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionDevice, DimensionKey: "device-1",
					MetricPath: "K1", MetricType: "kpi", Value: 999, Partial: true,
				},
				{
					TaskID: taskID, Granularity: pmstream.GranularityDaily,
					WindowStart: start.Add(30 * 24 * time.Hour),
					WindowEnd:   end,
					Dimension:   pmstream.DimensionNetwork, DimensionKey: "network",
					MetricPath: "K2", MetricType: "kpi", Value: 888, Partial: true,
				},
			},
			Periods: []pmstream.PeriodProgress{{
				Granularity: pmstream.GranularityDaily,
				WindowStart: start.Add(30 * 24 * time.Hour), WindowEnd: end,
				EntityKey: "network", ReceivedSlots: 4, ExpectedSlots: 24,
			}},
		},
	}
	service := &Service{networkRollups: reader}
	service.SetNetworkProgressReader(progress)

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Equal(t, "available", snapshot.ProgressState)
	require.Len(t, snapshot.Series["K1"], 2)
	require.Equal(t, jsonx.Float(10), snapshot.Series["K1"][0].Value)
	require.False(t, snapshot.Series["K1"][0].Partial)
	require.Equal(t, jsonx.Float(42), snapshot.Series["K1"][1].Value)
	require.True(t, snapshot.Series["K1"][1].Partial)
	require.Len(t, snapshot.PeriodProgress, 1)
	require.EqualValues(t, 4, snapshot.PeriodProgress[0].ReceivedSlots)
	require.EqualValues(t, 24, snapshot.PeriodProgress[0].ExpectedSlots)
	require.NotContains(t, snapshot.Series, "K2")
}

func TestGetKPITimeSeriesSnapshotAppendsCounterNetworkPartial(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("nr")
	require.True(t, ok)
	windowStart := start.Add(30 * 24 * time.Hour)
	reader := &recordingCounterSeriesReader{points: []NetworkRollupPoint{{
		Technology: model.TechNR, MetricPath: "C010070004",
		Granularity: metrics.GranularityDaily, WindowStart: start,
		WindowEnd: start.Add(24 * time.Hour), Value: jsonx.Float(1.1),
	}}}
	service := &Service{counterSeries: reader}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{Rows: []pmstream.ProgressResult{{
			TaskID: taskID, Granularity: pmstream.GranularityDaily,
			WindowStart: windowStart, WindowEnd: end,
			Dimension: pmstream.DimensionNetwork, DimensionKey: "network",
			MetricPath: "C010070004", MetricType: "counter", Value: 1.2,
			Partial: true,
		}}},
	})

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"C010070004"}, model.TechNR,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Len(t, snapshot.Series["C010070004"], 2)
	assert.Equal(t, jsonx.Float(1.2), snapshot.Series["C010070004"][1].Value)
	assert.True(t, snapshot.Series["C010070004"][1].Partial)
}

func TestGetKPITimeSeriesSnapshotKeepsPublishedSeriesWhenProgressUnavailable(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(31 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
		Technology: model.TechLTE, MetricPath: "K1",
		Granularity: metrics.GranularityDaily,
		WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Value: jsonx.Float(10), Complete: true,
	}}}
	service := &Service{networkRollups: reader}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		err:            errors.New("redis timeout"),
	})

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Equal(t, "unavailable", snapshot.ProgressState)
	require.Len(t, snapshot.Series["K1"], 1)
	require.Equal(t, jsonx.Float(10), snapshot.Series["K1"][0].Value)
	require.Empty(t, snapshot.PeriodProgress)
}

func TestGetKPITimeSeriesSnapshotDoesNotReportCurrentPartialPeriodAsMissing(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	end := start.Add(12 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
				Dimension: pmstream.DimensionNetwork, DimensionKey: "network",
				MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
				ReceivedSlots: 2, ExpectedSlots: 24,
			}},
		},
	})

	snapshot, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)

	require.NoError(t, err)
	require.Equal(t, "available", snapshot.ProgressState)
	require.Len(t, snapshot.Series["K1"], 1)
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotEqual(t, "dashboard_kpi_missing_result_total", family.GetName(),
			"a valid current partial period must not increment the published-result-missing counter")
	}
}

func TestGetKPITimeSeriesSnapshotStillReportsMissingClosedPeriod(t *testing.T) {
	setDashboardTimezoneForTest(t, time.UTC)
	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := currentDay.Add(-24 * time.Hour)
	end := now.Add(time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: currentDay, WindowEnd: currentDay.Add(24 * time.Hour),
				Dimension: pmstream.DimensionNetwork, DimensionKey: "network",
				MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
			}},
			Periods: []pmstream.PeriodProgress{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: currentDay, WindowEnd: currentDay.Add(24 * time.Hour),
				EntityKey:            "network",
				VersionEffectiveFrom: start,
			}},
			MetricIntervals: []pmstream.MetricVersionInterval{{
				MetricPath: "K1", EffectiveFrom: start,
			}},
		},
	})

	_, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)
	require.NoError(t, err)

	families, err := registry.Gather()
	require.NoError(t, err)
	var missing float64
	for _, family := range families {
		if family.GetName() == "dashboard_kpi_missing_result_total" {
			for _, metric := range family.GetMetric() {
				missing += metric.GetCounter().GetValue()
			}
		}
	}
	require.Equal(t, float64(1), missing,
		"current partial must not hide a missing final result from a closed period")
}

func TestGetKPITimeSeriesSnapshotDoesNotReportPeriodsBeforeVersionEffectiveFrom(t *testing.T) {
	setDashboardTimezoneForTest(t, time.UTC)
	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := currentDay.Add(-7 * 24 * time.Hour)
	end := now.Add(time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: currentDay, WindowEnd: currentDay.Add(24 * time.Hour),
				Dimension: pmstream.DimensionNetwork, DimensionKey: "network",
				MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
			}},
			Periods: []pmstream.PeriodProgress{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: currentDay, WindowEnd: currentDay.Add(24 * time.Hour),
				EntityKey:            "network",
				VersionEffectiveFrom: currentDay.Add(time.Hour),
			}},
			MetricIntervals: []pmstream.MetricVersionInterval{{
				MetricPath: "K1", EffectiveFrom: currentDay.Add(time.Hour),
			}},
		},
	})

	_, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)
	require.NoError(t, err)

	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotEqual(t, "dashboard_kpi_missing_result_total", family.GetName(),
			"periods before the task version became effective must not be reported as missing")
	}
}

func TestGetKPITimeSeriesSnapshotReportsLatestClosedPeriodEvenWhenOlderPointExists(t *testing.T) {
	setDashboardTimezoneForTest(t, time.UTC)
	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := currentDay.Add(-3 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{points: []NetworkRollupPoint{{
			Technology: model.TechLTE, MetricPath: "K1",
			Granularity: metrics.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
			Value: jsonx.Float(10), Complete: true,
		}}},
		metrics: NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			MetricIntervals: []pmstream.MetricVersionInterval{{
				MetricPath: "K1", EffectiveFrom: start,
			}},
		},
	})

	_, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, now.Add(time.Hour),
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), gatheredMissingResultTotal(t, registry),
		"an older final point must not hide a missing latest closed period")
}

func TestGetKPITimeSeriesSnapshotReportsMissingWithoutOpenProgressWindow(t *testing.T) {
	setDashboardTimezoneForTest(t, time.UTC)
	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := currentDay.Add(-24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows:    nil,
			Periods: nil,
			MetricIntervals: []pmstream.MetricVersionInterval{{
				MetricPath: "K1", EffectiveFrom: start,
			}},
		},
	})

	_, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, now.Add(time.Hour),
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), gatheredMissingResultTotal(t, registry),
		"catalog metadata must keep closed-period monitoring active without an open window")
}

func TestGetKPITimeSeriesSnapshotDoesNotObserveLatestPeriodOutsideQueryRange(t *testing.T) {
	setDashboardTimezoneForTest(t, time.UTC)
	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := currentDay.Add(-7 * 24 * time.Hour)
	end := currentDay.Add(-5 * 24 * time.Hour)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			MetricIntervals: []pmstream.MetricVersionInterval{{
				MetricPath: "K1", EffectiveFrom: start,
			}},
		},
	})

	_, _, err := service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, start, end,
	)
	require.NoError(t, err)
	require.Zero(t, gatheredMissingResultTotal(t, registry),
		"a historical query must not observe a period outside its end boundary")
}

func gatheredMissingResultTotal(
	t *testing.T,
	registry *prometheus.Registry,
) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	var missing float64
	for _, family := range families {
		if family.GetName() != "dashboard_kpi_missing_result_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			missing += metric.GetCounter().GetValue()
		}
	}
	return missing
}

func setDashboardTimezoneForTest(t *testing.T, location *time.Location) {
	t.Helper()
	response.SetTimezoneProvider(fixedDashboardTimezoneProvider{location: location})
	t.Cleanup(func() { response.SetTimezoneProvider(nil) })
}

func TestGetKPITimeSeriesSnapshotUsesBusinessTimezoneForOpenPeriod(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	response.SetTimezoneProvider(fixedDashboardTimezoneProvider{location: location})
	t.Cleanup(func() { response.SetTimezoneProvider(nil) })
	now := time.Now().In(location)
	currentDay := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location,
	)
	taskID, ok := pmstream.BuiltinNetworkTaskID("lte")
	require.True(t, ok)
	registry := prometheus.NewRegistry()
	service := &Service{
		networkRollups: &recordingNetworkRollupReader{},
		metrics:        NewMetrics(registry),
	}
	service.SetNetworkProgressReader(&fixedNetworkProgressReader{
		expectedTaskID: taskID,
		result: pmstream.ProgressQueryResult{
			Rows: []pmstream.ProgressResult{{
				TaskID: taskID, Granularity: pmstream.GranularityDaily,
				WindowStart: currentDay.UTC(),
				WindowEnd:   currentDay.Add(24 * time.Hour).UTC(),
				Dimension:   pmstream.DimensionNetwork, DimensionKey: "network",
				MetricPath: "K1", MetricType: "kpi", Value: 42, Partial: true,
			}},
		},
	})

	_, _, err = service.GetKPITimeSeriesSnapshotWithMetadata(
		context.Background(), []string{"K1"}, model.TechLTE,
		metrics.GranularityDaily, currentDay.UTC(), now.Add(time.Hour).UTC(),
	)
	require.NoError(t, err)
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotEqual(t, "dashboard_kpi_missing_result_total", family.GetName(),
			"the current business-timezone day must not be reported as a missing final period")
	}
}

func TestCurrentNaturalPeriodStartUsesBusinessTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 7, 31, 18, 0, 0, 0, location)

	require.Equal(t,
		time.Date(2026, 7, 30, 16, 0, 0, 0, time.UTC),
		currentNaturalPeriodStart(metrics.GranularityDaily, now),
	)
	require.Equal(t,
		time.Date(2026, 7, 26, 16, 0, 0, 0, time.UTC),
		currentNaturalPeriodStart(metrics.GranularityWeekly, now),
	)
}

func TestGetKPITimeSeries_RoutesRequestedGranularity(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(12 * 7 * 24 * time.Hour)

	for _, granularity := range []metrics.Granularity{
		metrics.GranularityHourly,
		metrics.GranularityDaily,
		metrics.GranularityWeekly,
	} {
		t.Run(string(granularity), func(t *testing.T) {
			reader := &recordingNetworkRollupReader{}
			service := &Service{networkRollups: reader}

			_, err := service.GetKPITimeSeries(
				context.Background(), []string{"K1"}, model.TechLTE, granularity, start, end,
			)

			require.NoError(t, err)
			require.Len(t, reader.queries, 1)
			require.Equal(t, granularity, reader.queries[0].Granularity)
			require.Equal(t, model.TechLTE, reader.queries[0].Technology)
			require.Equal(t, []string{"K1"}, reader.queries[0].MetricPaths)
		})
	}
}

func TestGetKPITimeSeriesReturnsPublishedValuesWithoutFillingMissingBuckets(t *testing.T) {
	start := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	reader := &recordingNetworkRollupReader{points: []NetworkRollupPoint{
		{
			Technology: model.TechNR, MetricPath: "K1",
			Granularity: metrics.GranularityWeekly,
			WindowStart: start.Add(7 * 24 * time.Hour), WindowEnd: start.Add(14 * 24 * time.Hour),
			Value: jsonx.Float(42), Complete: false, MissingSlots: 1,
		},
	}}
	service := &Service{networkRollups: reader}

	got, err := service.GetKPITimeSeries(
		context.Background(), []string{"K1", "K2"}, model.TechNR,
		metrics.GranularityWeekly, start, start.Add(12*7*24*time.Hour),
	)

	require.NoError(t, err)
	require.Len(t, got["K1"], 1)
	require.Equal(t, jsonx.Float(42), got["K1"][0].Value)
	require.Empty(t, got["K2"], "missing buckets and metrics must remain empty")
	require.Len(t, reader.queries, 1, "reader errors or gaps must never trigger a fallback query")
}

func TestCloneKPITimeSeriesResponsePreservesEmptyArrays(t *testing.T) {
	source := KPITimeSeriesResponse{
		"K_WITH_DATA": {{Time: time.Now(), Value: jsonx.Float(42)}},
		"K_NO_DATA":   {},
	}

	cloned := cloneKPITimeSeriesResponse(source)

	require.NotNil(t, cloned["K_NO_DATA"], "JSON contract requires [] instead of null")
	require.Empty(t, cloned["K_NO_DATA"])
	require.Equal(t, source["K_WITH_DATA"], cloned["K_WITH_DATA"])
}
