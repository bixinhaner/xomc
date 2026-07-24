package aggregator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestVersionedHourlyValidateWindow(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 30, 0, 0, time.UTC)
	valid := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 7, 24, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, validateVersionedHourlyWindow(valid, now))

	open := valid
	open.End = now.Add(time.Minute)
	open.Start = open.End.Add(-time.Hour)
	assert.ErrorContains(t, validateVersionedHourlyWindow(open, now), "closed")

	wrongDuration := valid
	wrongDuration.End = wrongDuration.Start.Add(2 * time.Hour)
	assert.ErrorContains(t, validateVersionedHourlyWindow(wrongDuration, now), "exactly one hour")

	wrongGranularity := valid
	wrongGranularity.Granularity = metrics.GranularityDaily
	assert.ErrorContains(t, validateVersionedHourlyWindow(wrongGranularity, now), "hourly granularity")
}

func TestVersionedHourlyBatchSQLIsBoundedAndAppendOnly(t *testing.T) {
	sql := buildVersionedHourlyBatchSQL()
	assert.Contains(t, sql, "unnest($4::uuid[]")
	assert.Contains(t, sql, "INSERT INTO pm_hourly_anchors")
	assert.Contains(t, sql, "INSERT INTO pm_hourly_values")
	assert.Contains(t, sql, "pm_hourly_rollup_batches")
	valueInsert := sql[strings.Index(sql, "INSERT INTO pm_hourly_values"):]
	assert.NotContains(t, valueInsert, "ON CONFLICT")
	assert.NotContains(t, sql, "DO UPDATE SET metric_ids",
		"stable metric sets must not be updated merely to resolve their IDs")
	assert.NotContains(t, sql, "pm_metric_sets",
		"all base metric-set access must finish before the anchor/value statement")
	assert.NotContains(t, sql, "work_mem")
}

func TestVersionedHourlyMetricSetPreparationIsSeparateAndInsertOnly(t *testing.T) {
	sql := buildEnsureVersionedHourlyMetricSetsSQL()
	assert.Contains(t, sql, "INSERT INTO pm_metric_sets")
	assert.Contains(t, sql, "DO NOTHING")
	assert.NotContains(t, sql, "DO UPDATE")
	assert.NotContains(t, buildVersionedHourlyBatchSQL(), "INSERT INTO pm_metric_sets",
		"the re-query must run in a later statement snapshot after insert-only conflict resolution")
}

func TestVersionedHourlyPublicationSQLIsAtomicStateSwitch(t *testing.T) {
	assert.Contains(t, buildSupersedeHourlyVersionSQL(), "status = 'superseded'")
	assert.Contains(t, buildPublishHourlyVersionSQL(), "status = 'active'")
	assert.NotContains(t, buildPublishHourlyVersionSQL(), "ON CONFLICT")
}

func TestVersionedHourlyAdvisoryLockKeyIsStablePerBucket(t *testing.T) {
	start := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, hourlyAdvisoryLockKey(start), hourlyAdvisoryLockKey(start))
	assert.NotEqual(t, hourlyAdvisoryLockKey(start), hourlyAdvisoryLockKey(start.Add(time.Hour)))
}

func TestVersionedHourlyDeviceBatchCount(t *testing.T) {
	devices := make([]uuid.UUID, 401)
	for i := range devices {
		devices[i] = uuid.New()
	}
	batches := splitDeviceBatches(devices, 200)
	require.Len(t, batches, 3)
	assert.Len(t, batches[0], 200)
	assert.Len(t, batches[1], 200)
	assert.Len(t, batches[2], 1)
}

func TestParseHourlyBatchDevices(t *testing.T) {
	assert.Equal(t, 200, ParseHourlyBatchDevices(""))
	assert.Equal(t, 350, ParseHourlyBatchDevices("350"))
	assert.Equal(t, 200, ParseHourlyBatchDevices("0"))
	assert.Equal(t, 200, ParseHourlyBatchDevices("invalid"))
}

func TestFormulaDictionaryRegisteredBeforeVersionLock(t *testing.T) {
	ctx := context.Background()
	deviceID := uuid.New()
	events := make([]string, 0)
	tx := &recordingFormulaBatchTx{
		events:   &events,
		deviceID: deviceID,
		metricID: 89,
		setID:    144,
	}
	db := &recordingFormulaPreparationDB{
		events:   &events,
		deviceID: deviceID,
		metricID: 89,
		tx:       tx,
	}
	a := New(db, &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"LOCK-FORMULA-1": {
			KPIs: []router.KPIDef{{
				IndicatorID: "KLOCK1", Unit: "%", Formula: "CLOCK1",
			}},
		},
	}}, nil)

	_, _, err := a.prepareAndBeginVersionedHourlyBatch(ctx, db, 55, []uuid.UUID{deviceID})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"dictionary", "begin", "bucket/version transaction",
	}, events)
}

func TestVersionedHourlyProductionLockOrder(t *testing.T) {
	ctx := context.Background()
	deviceID := uuid.New()
	events := make([]string, 0)
	tx := &recordingFormulaBatchTx{
		events:   &events,
		deviceID: deviceID,
		metricID: 89,
		setID:    144,
	}
	db := &recordingFormulaPreparationDB{
		events:   &events,
		deviceID: deviceID,
		metricID: 89,
		tx:       tx,
	}
	a := New(db, &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"LOCK-FORMULA-1": {
			KPIs: []router.KPIDef{{
				IndicatorID: "KLOCK1", Unit: "%", Formula: "CLOCK1",
			}},
		},
	}}, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 7, 24, 11, 0, 0, 0, time.UTC),
	}

	anchors, values, err := a.runVersionedHourlyBatch(
		ctx, db, 55, 0, []uuid.UUID{deviceID}, w, "")

	require.NoError(t, err)
	assert.Equal(t, int64(2), anchors)
	assert.Equal(t, int64(2), values)
	assert.Contains(t, tx.counterQuerySQL, `ha."time"=$3`)
	require.Len(t, tx.counterQueryArgs, 3)
	assert.Equal(t, w.Start, tx.counterQueryArgs[2])
	assert.Equal(t, []string{
		"dictionary",
		"begin",
		"bucket/version transaction",
		"base metric set",
		"formula metric set",
		"base anchors/values",
		"formula anchor",
		"formula values",
		"commit",
	}, events)
}

func TestVersionedHourlyBatchRetriesTransactionWithoutRepeatingFormulaPreparation(t *testing.T) {
	withRetryTiming(t, func(min, max time.Duration) time.Duration { return min }, sleepImmediately)

	ctx := context.Background()
	deviceID := uuid.New()
	events := make([]string, 0)
	first := &recordingFormulaBatchTx{
		events:      &events,
		failLockErr: &pgconn.PgError{Code: "40P01"},
	}
	second := &recordingFormulaBatchTx{
		events:   &events,
		deviceID: deviceID,
		metricID: 89,
		setID:    144,
	}
	db := &recordingFormulaPreparationDB{
		events:       &events,
		deviceID:     deviceID,
		metricID:     89,
		transactions: []pgx.Tx{first, second},
	}
	a := New(db, &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"LOCK-FORMULA-1": {
			KPIs: []router.KPIDef{{
				IndicatorID: "KLOCK1", Unit: "%", Formula: "CLOCK1",
			}},
		},
	}}, nil)
	w := WindowSpec{
		Granularity: metrics.GranularityHourly,
		Start:       time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 7, 24, 11, 0, 0, 0, time.UTC),
	}

	anchors, values, err := a.runVersionedHourlyBatch(
		ctx, db, 55, 0, []uuid.UUID{deviceID}, w, "")

	require.NoError(t, err)
	assert.Equal(t, int64(2), anchors)
	assert.Equal(t, int64(2), values)
	assert.Equal(t, 1, first.rollbacks)
	assert.Equal(t, 0, first.commits)
	assert.Equal(t, 1, second.commits)
	assert.Equal(t, []string{
		"dictionary",
		"begin",
		"bucket/version transaction",
		"begin",
		"bucket/version transaction",
		"base metric set",
		"formula metric set",
		"base anchors/values",
		"formula anchor",
		"formula values",
		"commit",
	}, events)
}

type recordingFormulaPreparationDB struct {
	events            *[]string
	deviceID          uuid.UUID
	metricID          int64
	dictionaryQueries int
	tx                pgx.Tx
	transactions      []pgx.Tx
	beginCalls        int
}

func (db *recordingFormulaPreparationDB) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO pm_metric_dictionary") {
		*db.events = append(*db.events, "dictionary")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (db *recordingFormulaPreparationDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(sql, "FROM device_dim"):
		return &recordingFormulaRows{rows: [][]any{{
			db.deviceID, "48BF74", "LOCK-FORMULA-1",
		}}}, nil
	case strings.Contains(sql, "FROM pm_metric_dictionary"):
		db.dictionaryQueries++
		if db.dictionaryQueries == 1 {
			return &recordingFormulaRows{}, nil
		}
		return &recordingFormulaRows{rows: [][]any{{
			"KLOCK1", db.metricID, metrics.MetricTypeKPI, "pct", "%",
		}}}, nil
	default:
		return nil, fmt.Errorf("unexpected preparation query: %s", sql)
	}
}

func (db *recordingFormulaPreparationDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return recordingFormulaRow{err: fmt.Errorf("unexpected preparation query row: %s", sql)}
}

func (db *recordingFormulaPreparationDB) Begin(_ context.Context) (pgx.Tx, error) {
	*db.events = append(*db.events, "begin")
	if len(db.transactions) > 0 {
		tx := db.transactions[db.beginCalls]
		db.beginCalls++
		return tx, nil
	}
	return db.tx, nil
}

type recordingFormulaBatchTx struct {
	pgx.Tx
	events           *[]string
	deviceID         uuid.UUID
	metricID         int64
	setID            int64
	setQueries       int
	failLockErr      error
	commits          int
	rollbacks        int
	counterQuerySQL  string
	counterQueryArgs []any
}

func (tx *recordingFormulaBatchTx) record(event string) {
	if len(*tx.events) == 0 || (*tx.events)[len(*tx.events)-1] != event {
		*tx.events = append(*tx.events, event)
	}
}

func (tx *recordingFormulaBatchTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(sql, "FROM pm_hourly_anchors"):
		tx.counterQuerySQL = sql
		tx.counterQueryArgs = args
		return &recordingFormulaRows{rows: [][]any{{
			tx.deviceID, int16(0), "", "CLOCK1", float64(5),
		}}}, nil
	case strings.Contains(sql, "SELECT gs.device_dim_id") &&
		strings.Contains(sql, "resolved.metric_set_id"):
		tx.record("base metric set")
		return &recordingFormulaRows{rows: [][]any{{
			tx.deviceID, int16(0), "", "LOCK", int64(201),
		}}}, nil
	case strings.Contains(sql, "FROM pm_metric_sets"):
		tx.record("formula metric set")
		tx.setQueries++
		if tx.setQueries == 1 {
			return &recordingFormulaRows{}, nil
		}
		return &recordingFormulaRows{rows: [][]any{{tx.setID, []int64{tx.metricID}}}}, nil
	case strings.Contains(sql, "INSERT INTO pm_hourly_anchors"):
		tx.record("formula anchor")
		return &recordingFormulaRows{rows: [][]any{{
			int64(233), tx.deviceID, int16(0), "",
		}}}, nil
	default:
		return nil, fmt.Errorf("unexpected batch query: %s", sql)
	}
}

func (tx *recordingFormulaBatchTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "FROM pm_hourly_bucket_versions") &&
		strings.Contains(sql, "FOR UPDATE") {
		tx.record("bucket/version transaction")
		if tx.failLockErr != nil {
			return recordingFormulaRow{err: tx.failLockErr}
		}
		return recordingFormulaRow{values: []any{int64(55)}}
	}
	if strings.Contains(sql, "INSERT INTO pm_hourly_anchors") &&
		strings.Contains(sql, "pm_hourly_rollup_batches") {
		if strings.Contains(sql, "pm_metric_sets") {
			return recordingFormulaRow{err: fmt.Errorf(
				"base anchor/value statement must not access metric sets",
			)}
		}
		tx.record("base anchors/values")
		return recordingFormulaRow{values: []any{int64(1), int64(1)}}
	}
	return recordingFormulaRow{err: fmt.Errorf("unexpected batch query row: %s", sql)}
}

func (tx *recordingFormulaBatchTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(sql, "INSERT INTO pm_metric_dictionary"):
		return pgconn.CommandTag{}, fmt.Errorf("batch transaction must not write dictionary")
	case strings.Contains(sql, "INSERT INTO pm_metric_sets"):
		if strings.Contains(sql, "SELECT DISTINCT '__hourly__'") {
			tx.record("base metric set")
		} else {
			tx.record("formula metric set")
		}
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (tx *recordingFormulaBatchTx) CopyFrom(
	_ context.Context,
	_ pgx.Identifier,
	_ []string,
	_ pgx.CopyFromSource,
) (int64, error) {
	tx.record("formula values")
	return 1, nil
}

func (tx *recordingFormulaBatchTx) Commit(_ context.Context) error {
	tx.record("commit")
	tx.commits++
	return nil
}

func (tx *recordingFormulaBatchTx) Rollback(_ context.Context) error {
	tx.rollbacks++
	return nil
}

type recordingFormulaRows struct {
	rows [][]any
	pos  int
	err  error
}

func (r *recordingFormulaRows) Close()                                       {}
func (r *recordingFormulaRows) Err() error                                   { return r.err }
func (r *recordingFormulaRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *recordingFormulaRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *recordingFormulaRows) RawValues() [][]byte                          { return nil }
func (r *recordingFormulaRows) Conn() *pgx.Conn                              { return nil }
func (r *recordingFormulaRows) Next() bool {
	if r.pos >= len(r.rows) {
		return false
	}
	r.pos++
	return true
}
func (r *recordingFormulaRows) Scan(dest ...any) error {
	if r.pos == 0 || r.pos > len(r.rows) {
		return fmt.Errorf("scan called without a row")
	}
	return scanRecordingFormulaValues(r.rows[r.pos-1], dest...)
}
func (r *recordingFormulaRows) Values() ([]any, error) {
	if r.pos == 0 || r.pos > len(r.rows) {
		return nil, fmt.Errorf("values called without a row")
	}
	return r.rows[r.pos-1], nil
}

type recordingFormulaRow struct {
	values []any
	err    error
}

func (r recordingFormulaRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanRecordingFormulaValues(r.values, dest...)
}

func scanRecordingFormulaValues(values []any, dest ...any) error {
	if len(values) != len(dest) {
		return fmt.Errorf("scan values: got %d columns for %d destinations", len(values), len(dest))
	}
	for i, value := range values {
		out := reflect.ValueOf(dest[i])
		if out.Kind() != reflect.Ptr || out.IsNil() {
			return fmt.Errorf("scan destination %d is not a pointer", i)
		}
		out.Elem().Set(reflect.ValueOf(value))
	}
	return nil
}
