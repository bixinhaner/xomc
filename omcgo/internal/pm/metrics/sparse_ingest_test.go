package metrics

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSparseMeasurementsKeepsAnchorAndDropsMissingPhysicalValue(t *testing.T) {
	end := time.Date(2026, 7, 24, 10, 15, 0, 0, time.UTC)
	deviceID := uuid.New()
	counters := []model.PMCounter{
		{DeviceID: deviceID, OUI: "48BF74", DeviceSN: "SN-1", CellID: "Cellid=1", CounterGroup: "RRC", CounterName: "C1", CounterValue: 10, StatisType: "sum", Unit: "number", Granularity: 15, Time: end},
		{DeviceID: deviceID, OUI: "48BF74", DeviceSN: "SN-1", CellID: "Cellid=1", CounterGroup: "RRC", CounterName: "C2", CounterValue: math.NaN(), StatisType: "sum", Unit: "number", Granularity: 15, Time: end},
	}

	got := BuildSparseMeasurements(counters, nil)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"C1", "C2"}, got[0].MetricPaths)
	require.Len(t, got[0].Metrics, 2)
	assert.Equal(t, "sum", got[0].Metrics[1].StatisType)
	assert.Equal(t, "number", got[0].Metrics[1].Unit)
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, "C1", got[0].Values[0].Path)
	assert.Equal(t, float64(10), got[0].Values[0].Value)
}

func TestBuildSparseMeasurementsKeepsAllMissingKPIMetadata(t *testing.T) {
	end := time.Date(2026, 7, 24, 10, 15, 0, 0, time.UTC)
	got := BuildSparseMeasurements(nil, []model.KPIValue{{
		DeviceID: uuid.New(), DeviceSN: "SN-1", CellID: "Cellid=1",
		IndicatorID: "K1", KPIValue: math.NaN(), StatisType: "pct", Unit: "%",
		Time: end,
	}})

	require.Len(t, got, 1)
	require.Len(t, got[0].Metrics, 1)
	assert.Equal(t, MetricTypeKPI, got[0].Metrics[0].MetricType)
	assert.Equal(t, "pct", got[0].Metrics[0].StatisType)
	assert.Equal(t, "%", got[0].Metrics[0].Unit)
	assert.Empty(t, got[0].Values)
}

func TestBuildSparseMeasurementsKeepsAllMissingMeasurementAnchor(t *testing.T) {
	end := time.Date(2026, 7, 24, 10, 15, 0, 0, time.UTC)
	got := BuildSparseMeasurements([]model.PMCounter{{
		DeviceID: uuid.New(), DeviceSN: "SN-1", CellID: "Cellid=1", CounterGroup: "RRC",
		CounterName: "C1", CounterValue: math.NaN(), Granularity: 15, Time: end,
	}}, nil)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"C1"}, got[0].MetricPaths)
	assert.Empty(t, got[0].Values)
}

func TestMetricSetHashIsOrderIndependent(t *testing.T) {
	assert.Equal(t, MetricSetHash([]int64{9, 2, 5}), MetricSetHash([]int64{5, 9, 2}))
}

func TestSparseProductKeyReusesSetsAcrossDevicesOfSameProduct(t *testing.T) {
	productID := uuid.New()
	assert.Equal(t, productID.String(), sparseProductKey(&productID, uuid.New()))
	assert.Equal(t, productID.String(), sparseProductKey(&productID, uuid.New()))
	deviceID := uuid.New()
	assert.Equal(t, "device:"+deviceID.String(), sparseProductKey(nil, deviceID))
}

func TestBuildSparseMeasurementsFromMetricsKeepsKPIAnchorAndFiniteValues(t *testing.T) {
	deviceID := uuid.New()
	end := time.Date(2026, 7, 24, 11, 15, 0, 0, time.UTC)
	statis := StatisPct
	ldn := "Cellid=1"

	got, err := BuildSparseMeasurementsFromMetrics([]PMMetric{{
		DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "K1",
		MetricType: MetricTypeKPI, MetricValue: 99.5, StatisType: &statis,
		Granularity: Granularity15Min, Time: end.Add(-15 * time.Minute),
		StartTime: end.Add(-15 * time.Minute), EndTime: end, ObjectLDN: &ldn,
		Extra: map[string]any{"device_id": deviceID.String()},
	}})

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, deviceID, got[0].DeviceID)
	assert.Equal(t, "__kpi__", got[0].CounterGroup)
	assert.Equal(t, []string{"K1"}, got[0].MetricPaths)
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, 99.5, got[0].Values[0].Value)
}

func TestBuildSparseMeasurementsFromMetricsPreservesDuplicateMetadataForCanonicalValidation(t *testing.T) {
	deviceID := uuid.New()
	end := time.Date(2026, 7, 25, 11, 15, 0, 0, time.UTC)
	sum, avg := StatisSum, StatisAvg
	input := []PMMetric{
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "RRC.Attempts", MetricType: MetricTypeCounter,
			MetricValue: 1, StatisType: &sum, Granularity: Granularity15Min, Time: end, StartTime: end.Add(-15 * time.Minute), EndTime: end,
			Extra: map[string]any{"device_id": deviceID.String()}},
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "RRC.Attempts", MetricType: MetricTypeCounter,
			MetricValue: 2, StatisType: &avg, Granularity: Granularity15Min, Time: end, StartTime: end.Add(-15 * time.Minute), EndTime: end,
			Extra: map[string]any{"device_id": deviceID.String()}},
	}

	got, err := BuildSparseMeasurementsFromMetrics(input)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Len(t, got[0].Metrics, 2, "metadata must reach canonical validation uncollapsed")
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, float64(2), got[0].Values[0].Value, "reported values retain last-wins behavior")
	_, err = sparseMeasurementDefinitions(got)
	require.ErrorContains(t, err, "duplicate metric definition for path \"RRC.Attempts\" is inconsistent")
}

func TestBuildSparseMeasurementsFromMetricsKeepsConsistentDuplicateMetadataAndLastValue(t *testing.T) {
	deviceID := uuid.New()
	end := time.Date(2026, 7, 25, 11, 15, 0, 0, time.UTC)
	sum := StatisSum
	input := []PMMetric{
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "RRC.Success", MetricType: MetricTypeCounter,
			MetricValue: 1, StatisType: &sum, Granularity: Granularity15Min, Time: end, StartTime: end.Add(-15 * time.Minute), EndTime: end,
			Extra: map[string]any{"device_id": deviceID.String()}},
		{DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "RRC.Success", MetricType: MetricTypeCounter,
			MetricValue: 2, StatisType: &sum, Granularity: Granularity15Min, Time: end, StartTime: end.Add(-15 * time.Minute), EndTime: end,
			Extra: map[string]any{"device_id": deviceID.String()}},
	}

	got, err := BuildSparseMeasurementsFromMetrics(input)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Len(t, got[0].Metrics, 2)
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, float64(2), got[0].Values[0].Value)
	_, err = sparseMeasurementDefinitions(got)
	require.NoError(t, err)
}

func TestBuildSparseMeasurementsPreservesDuplicateMetadataForCanonicalValidation(t *testing.T) {
	deviceID := uuid.New()
	end := time.Date(2026, 7, 25, 11, 15, 0, 0, time.UTC)
	got := BuildSparseMeasurements([]model.PMCounter{
		{DeviceID: deviceID, DeviceSN: "SN-1", CounterGroup: "RRC", CounterName: "RRC.Attempts", CounterValue: 1, StatisType: "sum", Unit: "count", Granularity: 15, Time: end},
		{DeviceID: deviceID, DeviceSN: "SN-1", CounterGroup: "RRC", CounterName: "RRC.Attempts", CounterValue: 2, StatisType: "avg", Unit: "count", Granularity: 15, Time: end},
	}, nil)

	require.Len(t, got, 1)
	assert.Len(t, got[0].Metrics, 2, "metadata must reach canonical validation uncollapsed")
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, float64(2), got[0].Values[0].Value, "reported values retain last-wins behavior")
	_, err := sparseMeasurementDefinitions(got)
	require.ErrorContains(t, err, "duplicate metric definition for path \"RRC.Attempts\" is inconsistent")
}

func TestBuildSparseMeasurementsKeepsConsistentDuplicateMetadataAndLastValue(t *testing.T) {
	deviceID := uuid.New()
	end := time.Date(2026, 7, 25, 11, 15, 0, 0, time.UTC)
	got := BuildSparseMeasurements([]model.PMCounter{
		{DeviceID: deviceID, DeviceSN: "SN-1", CounterGroup: "RRC", CounterName: "RRC.Success", CounterValue: 1, StatisType: "sum", Unit: "count", Granularity: 15, Time: end},
		{DeviceID: deviceID, DeviceSN: "SN-1", CounterGroup: "RRC", CounterName: "RRC.Success", CounterValue: 2, StatisType: "sum", Unit: "count", Granularity: 15, Time: end},
	}, nil)

	require.Len(t, got, 1)
	assert.Len(t, got[0].Metrics, 2)
	require.Len(t, got[0].Values, 1)
	assert.Equal(t, float64(2), got[0].Values[0].Value)
	_, err := sparseMeasurementDefinitions(got)
	require.NoError(t, err)
}

func TestBuildSparseMeasurementsFromMetricsRejectsMissingDeviceIdentity(t *testing.T) {
	_, err := BuildSparseMeasurementsFromMetrics([]PMMetric{{
		DeviceOUI: "48BF74", DeviceSN: "SN-1", MetricPath: "K1",
		MetricType: MetricTypeKPI, MetricValue: 1,
	}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "device_id")
}

func TestBuildDeleteKPIAnchorsSQLTargetsSparsePhysicalTables(t *testing.T) {
	statements := buildDeleteKPIAnchorsSQL()
	require.Len(t, statements, 2)
	assert.Contains(t, statements[0], "DELETE FROM pm_metric_values")
	assert.Contains(t, statements[0], "pm_measurement_anchors")
	assert.Contains(t, statements[1], "DELETE FROM pm_measurement_anchors")
	for _, sql := range statements {
		assert.NotContains(t, sql, "DELETE FROM pm_metrics ")
	}
}

func TestGranularityTextUsesDatabaseEnumValues(t *testing.T) {
	assert.Equal(t, "15min", granularityText(15))
	assert.Equal(t, "hourly", granularityText(60))
	assert.Equal(t, "daily", granularityText(24*60))
	assert.Equal(t, "weekly", granularityText(7*24*60))
	assert.Equal(t, "monthly", granularityText(30*24*60))
}

func TestMarkDirtySQLProtectsActiveAndBuildingHourlyVersions(t *testing.T) {
	sql := buildMarkHourlyBucketsDirtySQL()
	assert.Contains(t, sql, "status IN ('active','building')")
	assert.Contains(t, sql, "bucket_start = ANY($1::timestamptz[])")
	assert.Contains(t, sql, "dirty = true")
}

func TestSparseIngestLockOrder(t *testing.T) {
	deviceID := uuid.New()
	start := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	tx := &recordingSparseIngestTx{
		deviceID: deviceID,
		metricID: 41,
		setID:    73,
	}

	err := writeSparseMeasurements(context.Background(), tx, nil, nil, []SparseMeasurement{{
		DeviceID: deviceID, DeviceSN: "LOCK-ORDER-1", CounterGroup: "RRC",
		Time: start, StartTime: start, EndTime: start.Add(15 * time.Minute), Granularity: 15,
		MetricPaths: []string{"CLOCK1"},
		Metrics: []SparseValue{{
			Path: "CLOCK1", MetricType: MetricTypeCounter, StatisType: "sum", Unit: "number",
		}},
		Values: []SparseValue{{
			Path: "CLOCK1", MetricType: MetricTypeCounter, Value: 1, StatisType: "sum", Unit: "number",
		}},
	}})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"bucket/version", "dictionary", "metric set", "anchor", "values",
	}, tx.operations)
}

func TestWriteSparseMeasurementsRejectsInconsistentRepeatedMetadataBeforeDictionaryResolution(t *testing.T) {
	tx := &recordingSparseIngestTx{deviceID: uuid.New(), metricID: 41, setID: 73}
	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)

	err := writeSparseMeasurements(context.Background(), tx, nil, nil, []SparseMeasurement{{
		DeviceID: tx.deviceID, DeviceSN: "DUPLICATE-METADATA", CounterGroup: "RRC",
		Time: now, StartTime: now, EndTime: now.Add(15 * time.Minute), Granularity: 15,
		MetricPaths: []string{"RRC.Attempts"},
		Metrics: []SparseValue{
			{Path: "RRC.Attempts", MetricType: MetricTypeCounter, StatisType: "sum", Unit: "count"},
			{Path: "RRC.Attempts", MetricType: MetricTypeCounter, StatisType: "avg", Unit: "count"},
		},
	}})

	require.ErrorContains(t, err, "duplicate metric definition for path \"RRC.Attempts\" is inconsistent")
	assert.Empty(t, tx.operations, "invalid metadata must fail before dictionary writes or bucket dirtiness")
}

type recordingSparseIngestTx struct {
	pgx.Tx
	deviceID   uuid.UUID
	metricID   int64
	setID      int64
	operations []string
}

func (tx *recordingSparseIngestTx) record(operation string) {
	if len(tx.operations) == 0 || tx.operations[len(tx.operations)-1] != operation {
		tx.operations = append(tx.operations, operation)
	}
}

func (tx *recordingSparseIngestTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(sql, "FROM pm_metric_dictionary"):
		tx.record("dictionary")
		return &sparseMetadataRows{rows: [][]any{{
			"CLOCK1", tx.metricID, MetricTypeCounter, "sum", "number",
		}}}, nil
	case strings.Contains(sql, "FROM device_dim"):
		return &sparseMetadataRows{}, nil
	case strings.Contains(sql, "FROM pm_metric_sets"):
		tx.record("metric set")
		return &sparseMetadataRows{rows: [][]any{{tx.setID, []int64{tx.metricID}}}}, nil
	default:
		return nil, assert.AnError
	}
}

func (tx *recordingSparseIngestTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "INSERT INTO pm_measurement_anchors") {
		tx.record("anchor")
		return sparseMetadataRow{values: []any{int64(101)}}
	}
	return sparseMetadataRow{err: assert.AnError}
}

func (tx *recordingSparseIngestTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "UPDATE pm_hourly_bucket_versions") {
		tx.record("bucket/version")
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *recordingSparseIngestTx) CopyFrom(
	_ context.Context,
	_ pgx.Identifier,
	_ []string,
	_ pgx.CopyFromSource,
) (int64, error) {
	tx.record("values")
	return 1, nil
}
