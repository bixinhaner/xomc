package aggregator

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

const DefaultHourlyBatchDevices = 2500

// ParseHourlyBatchDevices validates PM_HOURLY_BATCH_DEVICES and returns the
// conservative default for missing, invalid, or non-positive values.
func ParseHourlyBatchDevices(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return DefaultHourlyBatchDevices
	}
	return n
}

// RollupStats describes the physical rows built for one published hourly bucket.
type RollupStats struct {
	BucketVersion    int64
	DeviceCount      int
	CompletedBatches int
	AnchorCount      int64
	ValueCount       int64
}

type poolAcquirer interface {
	Acquire(context.Context) (*pgxpool.Conn, error)
}

// DBTX is the pgx surface shared by a pool connection and a transaction.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type versionedHourlyBatchBeginner interface {
	DBTX
	Begin(context.Context) (pgx.Tx, error)
}

type preparedFormulaKPI struct {
	MetricID   int64
	Definition router.KPIDef
}

type preparedFormulaDevice struct {
	oui, sn string
	kpis    []preparedFormulaKPI
}

type preparedHourlyBaseGroup struct {
	deviceID     uuid.UUID
	objectType   int16
	objectLDN    string
	counterGroup string
	metricSetID  int64
}

func (a *Aggregator) supportsVersionedHourly() bool {
	_, ok := a.db.(poolAcquirer)
	return ok
}

// RunHourlyVersioned builds a complete invisible version in bounded transactions,
// then atomically makes it the only active version for the bucket.
func (a *Aggregator) RunHourlyVersioned(ctx context.Context, w WindowSpec, batchSize int, retryMetrics ...*Metrics) (stats RollupStats, err error) {
	if err := validateVersionedHourlyWindow(w, time.Now()); err != nil {
		return stats, err
	}
	if batchSize <= 0 {
		batchSize = DefaultHourlyBatchDevices
	}
	pool, ok := a.db.(poolAcquirer)
	if !ok {
		return stats, fmt.Errorf("versioned hourly rollup requires a pgx pool")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return stats, fmt.Errorf("acquire hourly rollup connection: %w", err)
	}
	defer conn.Release()

	// This must span the short per-batch transactions, so a session advisory lock
	// on a dedicated pooled connection is used rather than one unbounded xact lock.
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, hourlyAdvisoryLockKey(w.Start)).Scan(&locked); err != nil {
		return stats, fmt.Errorf("acquire hourly advisory lock: %w", err)
	}
	if !locked {
		return stats, fmt.Errorf("hourly bucket %s is already being built", w.Start.Format(time.RFC3339))
	}
	defer func() {
		var unlocked bool
		_ = conn.QueryRow(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, hourlyAdvisoryLockKey(w.Start)).Scan(&unlocked)
	}()

	// The per-bucket session lock is released automatically on process death.
	// Therefore any remaining building version for this bucket is orphaned once
	// this lock has been acquired by a new attempt.
	if _, err := conn.Exec(ctx, `
		UPDATE pm_hourly_bucket_versions
		   SET status='failed'
		 WHERE bucket_start=$1 AND status='building'`, w.Start); err != nil {
		return stats, fmt.Errorf("fail orphaned hourly versions: %w", err)
	}

	var version int64
	if err := conn.QueryRow(ctx, `
		INSERT INTO pm_hourly_bucket_versions (bucket_start, bucket_end, status)
		VALUES ($1,$2,'building')
		RETURNING bucket_version`, w.Start, w.End).Scan(&version); err != nil {
		return stats, fmt.Errorf("create hourly bucket version: %w", err)
	}
	stats.BucketVersion = version
	published := false
	defer func() {
		if err != nil && !published {
			_, _ = conn.Exec(context.WithoutCancel(ctx),
				`UPDATE pm_hourly_bucket_versions SET status='failed' WHERE bucket_version=$1 AND status='building'`,
				version)
		}
	}()

	devices, err := listHourlyDeviceIDs(ctx, conn, w)
	if err != nil {
		return stats, err
	}
	stats.DeviceCount = len(devices)
	numberProcess, err := a.numberProcess(ctx)
	if err != nil {
		return stats, err
	}
	batches := splitDeviceBatches(devices, batchSize)
	if _, err := conn.Exec(ctx,
		`UPDATE pm_hourly_bucket_versions SET expected_batches=$2 WHERE bucket_version=$1`,
		version, len(batches)); err != nil {
		return stats, fmt.Errorf("set hourly expected batches: %w", err)
	}

	for batchNo, deviceIDs := range batches {
		anchorCount, valueCount, batchErr := a.runVersionedHourlyBatch(
			ctx, conn, version, batchNo, deviceIDs, w, numberProcess, retryMetrics...)
		if batchErr != nil {
			return stats, fmt.Errorf("run hourly batch %d: %w", batchNo, batchErr)
		}
		stats.CompletedBatches++
		stats.AnchorCount += anchorCount
		stats.ValueCount += valueCount
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return stats, fmt.Errorf("begin hourly publish: %w", err)
	}
	var bucketStart time.Time
	if err := tx.QueryRow(ctx, `
		SELECT bucket_start FROM pm_hourly_bucket_versions
		 WHERE bucket_version=$1 AND status='building' AND dirty=false
		   AND completed_batches=expected_batches
		   AND anchor_count=$2 AND value_count=$3
		 FOR UPDATE`,
		version, stats.AnchorCount, stats.ValueCount).Scan(&bucketStart); err != nil {
		_ = tx.Rollback(ctx)
		return stats, fmt.Errorf("validate hourly version for publication: %w", err)
	}
	if _, err := tx.Exec(ctx, buildSupersedeHourlyVersionSQL(), bucketStart, version); err != nil {
		_ = tx.Rollback(ctx)
		return stats, fmt.Errorf("supersede previous hourly version: %w", err)
	}
	var activated int64
	if err := tx.QueryRow(ctx, buildPublishHourlyVersionSQL(), version).Scan(&activated); err != nil {
		_ = tx.Rollback(ctx)
		return stats, fmt.Errorf("activate hourly version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return stats, fmt.Errorf("commit hourly publication: %w", err)
	}
	published = true
	return stats, nil
}

func validateVersionedHourlyWindow(w WindowSpec, now time.Time) error {
	if w.Granularity != metrics.GranularityHourly {
		return fmt.Errorf("versioned hourly rollup requires hourly granularity")
	}
	if !w.End.After(w.Start) || w.End.Sub(w.Start) != time.Hour {
		return fmt.Errorf("versioned hourly rollup window must be exactly one hour")
	}
	if w.End.After(now) {
		return fmt.Errorf("versioned hourly rollup only accepts a closed hour")
	}
	return nil
}

func hourlyAdvisoryLockKey(start time.Time) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("pm-hourly:"))
	_, _ = h.Write([]byte(start.UTC().Format(time.RFC3339Nano)))
	return int64(h.Sum64())
}

func splitDeviceBatches(devices []uuid.UUID, size int) [][]uuid.UUID {
	if size <= 0 {
		size = DefaultHourlyBatchDevices
	}
	out := make([][]uuid.UUID, 0, (len(devices)+size-1)/size)
	for start := 0; start < len(devices); start += size {
		end := start + size
		if end > len(devices) {
			end = len(devices)
		}
		out = append(out, devices[start:end])
	}
	return out
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func listHourlyDeviceIDs(ctx context.Context, db queryer, w WindowSpec) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx, `
		SELECT DISTINCT a.device_dim_id
		  FROM pm_measurement_anchors a
		 WHERE a."time" >= $1 AND a."time" < $2
		 ORDER BY a.device_dim_id`, w.Start, w.End)
	if err != nil {
		return nil, fmt.Errorf("list hourly devices: %w", err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan hourly device: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly devices: %w", err)
	}
	return out, nil
}

func (a *Aggregator) prepareAndBeginVersionedHourlyBatch(
	ctx context.Context,
	conn versionedHourlyBatchBeginner,
	version int64,
	deviceIDs []uuid.UUID,
) (map[uuid.UUID]preparedFormulaDevice, pgx.Tx, error) {
	// Formula dictionary rows are resolved before the transaction that locks
	// and writes bucket/version progress. The transaction receives immutable IDs.
	prepared, err := a.prepareVersionedHourlyFormulaKPIs(ctx, conn, deviceIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare hourly formula KPIs: %w", err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin hourly batch: %w", err)
	}
	var lockedVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT bucket_version
		  FROM pm_hourly_bucket_versions
		 WHERE bucket_version=$1 AND status='building'
		 FOR UPDATE`, version).Scan(&lockedVersion); err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, fmt.Errorf("lock hourly bucket version: %w", err)
	}
	if lockedVersion != version {
		_ = tx.Rollback(ctx)
		return nil, nil, fmt.Errorf("lock hourly bucket version: got %d, want %d", lockedVersion, version)
	}
	return prepared, tx, nil
}

func (a *Aggregator) runVersionedHourlyBatch(
	ctx context.Context,
	conn versionedHourlyBatchBeginner,
	version int64,
	batchNo int,
	deviceIDs []uuid.UUID,
	w WindowSpec,
	numberProcess string,
	retryMetrics ...*Metrics,
) (anchorCount, valueCount int64, err error) {
	preparedFormulaKPIs, err := a.prepareVersionedHourlyFormulaKPIs(ctx, conn, deviceIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("prepare hourly formula KPIs: %w", err)
	}
	var m *Metrics
	if len(retryMetrics) > 0 {
		m = retryMetrics[0]
	}
	err = runTransactionWithRetryObserved(ctx, conn.Begin, maxHourlyTxRetryAttempts,
		func(ctx context.Context, tx pgx.Tx) error {
			anchorCount, valueCount, err = a.runPreparedVersionedHourlyBatch(
				ctx, tx, version, batchNo, deviceIDs, w, numberProcess, preparedFormulaKPIs)
			return err
		},
		m.IncHourlyTxRetry,
		m.IncHourlyTxRetryExhausted,
	)
	if err != nil {
		return 0, 0, err
	}
	return anchorCount, valueCount, nil
}

func (a *Aggregator) runPreparedVersionedHourlyBatch(
	ctx context.Context,
	tx pgx.Tx,
	version int64,
	batchNo int,
	deviceIDs []uuid.UUID,
	w WindowSpec,
	numberProcess string,
	preparedFormulaKPIs map[uuid.UUID]preparedFormulaDevice,
) (anchorCount, valueCount int64, err error) {
	if err := lockVersionedHourlyBatch(ctx, tx, version); err != nil {
		return 0, 0, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pm_hourly_rollup_batches
		    (bucket_version, batch_no, status, device_count)
		VALUES ($1,$2,'building',$3)`,
		version, batchNo, len(deviceIDs)); err != nil {
		return 0, 0, fmt.Errorf("start hourly batch: %w", err)
	}
	if _, err := tx.Exec(ctx, buildEnsureVersionedHourlyMetricSetsSQL(),
		w.Start, w.End, deviceIDs); err != nil {
		return 0, 0, fmt.Errorf("resolve base hourly metric sets: %w", err)
	}
	baseGroups, err := resolveVersionedHourlyBaseGroups(ctx, tx, w, deviceIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("prepare base hourly metric-set IDs: %w", err)
	}
	formulaSetIDs, err := resolveVersionedHourlyFormulaMetricSets(ctx, tx, preparedFormulaKPIs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve hourly formula KPI metric sets: %w", err)
	}

	baseDeviceIDs := make([]uuid.UUID, len(baseGroups))
	baseObjectTypes := make([]int16, len(baseGroups))
	baseObjectLDNs := make([]string, len(baseGroups))
	baseCounterGroups := make([]string, len(baseGroups))
	baseMetricSetIDs := make([]int64, len(baseGroups))
	for i, group := range baseGroups {
		baseDeviceIDs[i] = group.deviceID
		baseObjectTypes[i] = group.objectType
		baseObjectLDNs[i] = group.objectLDN
		baseCounterGroups[i] = group.counterGroup
		baseMetricSetIDs[i] = group.metricSetID
	}
	if err := tx.QueryRow(ctx, buildVersionedHourlyBatchSQL(),
		version, w.Start, w.End,
		baseDeviceIDs, baseObjectTypes, baseObjectLDNs, baseCounterGroups, baseMetricSetIDs,
		batchNo, numberProcess).
		Scan(&anchorCount, &valueCount); err != nil {
		return 0, 0, fmt.Errorf("build base hourly batch: %w", err)
	}
	formulaAnchors, formulaValues, err := a.insertVersionedHourlyFormulaKPIs(
		ctx, tx, version, deviceIDs, w, numberProcess, preparedFormulaKPIs, formulaSetIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("build hourly formula KPIs: %w", err)
	}
	if formulaAnchors > 0 || formulaValues > 0 {
		if _, err := tx.Exec(ctx, `
			WITH batch_accounted AS (
			    UPDATE pm_hourly_rollup_batches
			   SET anchor_count=anchor_count+$3, value_count=value_count+$4
			 WHERE bucket_version=$1 AND batch_no=$2
			   RETURNING bucket_version
			)
			UPDATE pm_hourly_bucket_versions v
			   SET anchor_count=anchor_count+$3, value_count=value_count+$4
			  FROM batch_accounted b
			 WHERE v.bucket_version=b.bucket_version AND v.status='building'`,
			version, batchNo, formulaAnchors, formulaValues); err != nil {
			return 0, 0, fmt.Errorf("account hourly formula KPIs: %w", err)
		}
		anchorCount += formulaAnchors
		valueCount += formulaValues
	}
	if _, err := tx.Exec(ctx, buildCompleteVersionedHourlyBatchSQL(), version, batchNo); err != nil {
		return 0, 0, fmt.Errorf("complete hourly batch: %w", err)
	}
	return anchorCount, valueCount, nil
}

func lockVersionedHourlyBatch(ctx context.Context, tx pgx.Tx, version int64) error {
	var lockedVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT bucket_version
		  FROM pm_hourly_bucket_versions
		 WHERE bucket_version=$1 AND status='building'
		 FOR UPDATE`, version).Scan(&lockedVersion); err != nil {
		return fmt.Errorf("lock hourly bucket version: %w", err)
	}
	if lockedVersion != version {
		return fmt.Errorf("lock hourly bucket version: got %d, want %d", lockedVersion, version)
	}
	return nil
}

func (a *Aggregator) prepareVersionedHourlyFormulaKPIs(
	ctx context.Context,
	db DBTX,
	deviceIDs []uuid.UUID,
) (map[uuid.UUID]preparedFormulaDevice, error) {
	prepared := make(map[uuid.UUID]preparedFormulaDevice)
	if a.kpiRouter == nil || len(deviceIDs) == 0 {
		return prepared, nil
	}

	deviceSQL, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "COALESCE(oui,'')", "COALESCE(serial_number,'')").
		From("device_dim").
		Where(sq.Eq{"id": deviceIDs}).
		OrderBy("id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build hourly formula device query: %w", err)
	}
	rows, err := db.Query(ctx, deviceSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list hourly formula devices: %w", err)
	}
	type formulaDevice struct {
		id      uuid.UUID
		oui, sn string
	}
	devices := make([]formulaDevice, 0, len(deviceIDs))
	for rows.Next() {
		var device formulaDevice
		if err := rows.Scan(&device.id, &device.oui, &device.sn); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan hourly formula device: %w", err)
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate hourly formula devices: %w", err)
	}
	rows.Close()

	definitions := make(map[string]router.KPIDef)
	deviceDefinitions := make(map[uuid.UUID][]router.KPIDef, len(devices))
	for _, device := range devices {
		route, err := a.kpiRouter.LookupByDevice(ctx, device.sn)
		if err != nil {
			return nil, fmt.Errorf("lookup hourly KPI route for %s: %w", device.sn, err)
		}
		if route == nil {
			continue
		}
		formulas, _ := splitKPIDefsByRollupMode(route.KPIs)
		if len(formulas) == 0 {
			continue
		}
		canonicalFormulas := make([]router.KPIDef, 0, len(formulas))
		for _, definition := range formulas {
			if strings.TrimSpace(definition.StatisType) == "" {
				definition.StatisType = string(metrics.StatisPct)
			}
			if current, ok := definitions[definition.IndicatorID]; ok {
				if current.StatisType != definition.StatisType || current.Unit != definition.Unit {
					return nil, fmt.Errorf("prepare hourly formula KPI %q: inconsistent immutable metadata", definition.IndicatorID)
				}
			} else {
				definitions[definition.IndicatorID] = definition
			}
			canonicalFormulas = append(canonicalFormulas, definition)
		}
		deviceDefinitions[device.id] = canonicalFormulas
		prepared[device.id] = preparedFormulaDevice{oui: device.oui, sn: device.sn}
	}
	if len(definitions) == 0 {
		return prepared, nil
	}

	metricIDs, err := resolveVersionedHourlyFormulaDictionary(ctx, db, definitions)
	if err != nil {
		return nil, err
	}
	for deviceID, formulas := range deviceDefinitions {
		device := prepared[deviceID]
		device.kpis = make([]preparedFormulaKPI, 0, len(formulas))
		for _, definition := range formulas {
			metricID, ok := metricIDs[definition.IndicatorID]
			if !ok {
				return nil, fmt.Errorf("prepare hourly formula KPI %q: metric ID was not resolved", definition.IndicatorID)
			}
			device.kpis = append(device.kpis, preparedFormulaKPI{
				MetricID: metricID, Definition: definition,
			})
		}
		prepared[deviceID] = device
	}
	return prepared, nil
}

type preparedFormulaDictionaryRow struct {
	metricID         int64
	metricType       metrics.MetricType
	statisType, unit string
}

func resolveVersionedHourlyFormulaDictionary(
	ctx context.Context,
	db DBTX,
	definitions map[string]router.KPIDef,
) (map[string]int64, error) {
	paths := make([]string, 0, len(definitions))
	for path := range definitions {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	existing, err := queryVersionedHourlyFormulaDictionary(ctx, db, paths)
	if err != nil {
		return nil, err
	}
	insert := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("pm_metric_dictionary").
		Columns("metric_path", "report_key", "metric_type", "statis_type", "unit")
	missing := 0
	for _, path := range paths {
		definition := definitions[path]
		if current, ok := existing[path]; ok {
			if err := validatePreparedFormulaDictionaryRow(path, definition, current); err != nil {
				return nil, err
			}
			continue
		}
		insert = insert.Values(path, path, metrics.MetricTypeKPI,
			sq.Expr("NULLIF(?, '')", definition.StatisType),
			sq.Expr("NULLIF(?, '')", definition.Unit))
		missing++
	}
	if missing > 0 {
		insertSQL, args, err := insert.Suffix("ON CONFLICT (metric_path) DO NOTHING").ToSql()
		if err != nil {
			return nil, fmt.Errorf("build hourly formula KPI dictionary insert: %w", err)
		}
		if _, err := db.Exec(ctx, insertSQL, args...); err != nil {
			return nil, fmt.Errorf("register hourly formula KPI dictionary: %w", err)
		}
	}

	resolved, err := queryVersionedHourlyFormulaDictionary(ctx, db, paths)
	if err != nil {
		return nil, err
	}
	metricIDs := make(map[string]int64, len(paths))
	for _, path := range paths {
		current, ok := resolved[path]
		if !ok {
			return nil, fmt.Errorf("resolve hourly formula KPI dictionary: metric path %q was not resolved", path)
		}
		if err := validatePreparedFormulaDictionaryRow(path, definitions[path], current); err != nil {
			return nil, err
		}
		metricIDs[path] = current.metricID
	}
	return metricIDs, nil
}

func queryVersionedHourlyFormulaDictionary(
	ctx context.Context,
	db DBTX,
	paths []string,
) (map[string]preparedFormulaDictionaryRow, error) {
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("metric_path", "metric_id", "metric_type",
			"COALESCE(statis_type,'')", "COALESCE(unit,'')").
		From("pm_metric_dictionary").
		Where(sq.Eq{"metric_path": paths}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build hourly formula KPI dictionary query: %w", err)
	}
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("resolve hourly formula KPI dictionary: %w", err)
	}
	defer rows.Close()
	out := make(map[string]preparedFormulaDictionaryRow, len(paths))
	for rows.Next() {
		var path string
		var row preparedFormulaDictionaryRow
		if err := rows.Scan(&path, &row.metricID, &row.metricType, &row.statisType, &row.unit); err != nil {
			return nil, fmt.Errorf("scan hourly formula KPI dictionary: %w", err)
		}
		out[path] = row
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly formula KPI dictionary: %w", err)
	}
	return out, nil
}

func validatePreparedFormulaDictionaryRow(
	path string,
	definition router.KPIDef,
	row preparedFormulaDictionaryRow,
) error {
	if row.metricType != metrics.MetricTypeKPI ||
		row.statisType != definition.StatisType ||
		row.unit != definition.Unit {
		return fmt.Errorf(
			"resolve hourly formula KPI dictionary: incompatible immutable metadata for path %q",
			path,
		)
	}
	return nil
}

func resolveVersionedHourlyBaseGroups(
	ctx context.Context,
	tx DBTX,
	w WindowSpec,
	deviceIDs []uuid.UUID,
) ([]preparedHourlyBaseGroup, error) {
	rows, err := tx.Query(ctx, buildResolveVersionedHourlyBaseGroupsSQL(),
		w.Start, w.End, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve hourly base groups: %w", err)
	}
	defer rows.Close()
	groups := make([]preparedHourlyBaseGroup, 0)
	for rows.Next() {
		var group preparedHourlyBaseGroup
		if err := rows.Scan(
			&group.deviceID,
			&group.objectType,
			&group.objectLDN,
			&group.counterGroup,
			&group.metricSetID,
		); err != nil {
			return nil, fmt.Errorf("scan hourly base group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly base groups: %w", err)
	}
	return groups, nil
}

func resolveVersionedHourlyFormulaMetricSets(
	ctx context.Context,
	tx DBTX,
	prepared map[uuid.UUID]preparedFormulaDevice,
) (map[uuid.UUID]int64, error) {
	type formulaSet struct {
		hash      []byte
		metricIDs []int64
	}
	setsByKey := make(map[string]formulaSet)
	deviceSetKeys := make(map[uuid.UUID]string, len(prepared))
	for deviceID, device := range prepared {
		metricIDs := make([]int64, 0, len(device.kpis))
		for _, kpi := range device.kpis {
			metricIDs = append(metricIDs, kpi.MetricID)
		}
		if len(metricIDs) == 0 {
			continue
		}
		sort.Slice(metricIDs, func(i, j int) bool { return metricIDs[i] < metricIDs[j] })
		hash := metrics.MetricSetHash(metricIDs)
		key := fmt.Sprintf("%x", hash[:])
		if current, ok := setsByKey[key]; ok {
			if len(current.metricIDs) != len(metricIDs) {
				return nil, fmt.Errorf("prepare hourly formula KPI metric set %q: inconsistent metric IDs", key)
			}
			for i := range metricIDs {
				if current.metricIDs[i] != metricIDs[i] {
					return nil, fmt.Errorf("prepare hourly formula KPI metric set %q: inconsistent metric IDs", key)
				}
			}
		} else {
			setsByKey[key] = formulaSet{hash: hash[:], metricIDs: metricIDs}
		}
		deviceSetKeys[deviceID] = key
	}

	keys := make([]string, 0, len(setsByKey))
	for key := range setsByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	setIDs := make(map[string]int64, len(keys))
	for _, key := range keys {
		set := setsByKey[key]
		setID, err := resolveVersionedHourlyFormulaMetricSet(ctx, tx, set.hash, set.metricIDs)
		if err != nil {
			return nil, err
		}
		setIDs[key] = setID
	}

	out := make(map[uuid.UUID]int64, len(deviceSetKeys))
	for deviceID, key := range deviceSetKeys {
		out[deviceID] = setIDs[key]
	}
	return out, nil
}

func (a *Aggregator) insertVersionedHourlyFormulaKPIs(
	ctx context.Context,
	tx pgx.Tx,
	version int64,
	deviceIDs []uuid.UUID,
	w WindowSpec,
	numberProcess string,
	prepared map[uuid.UUID]preparedFormulaDevice,
	formulaSetIDs map[uuid.UUID]int64,
) (int64, int64, error) {
	if len(prepared) == 0 || len(deviceIDs) == 0 {
		return 0, 0, nil
	}
	type formulaObject struct {
		objectType int16
		objectLDN  string
	}
	type deviceData struct {
		id      uuid.UUID
		oui, sn string
		objects map[formulaObject]map[string]float64
	}
	byID := make(map[uuid.UUID]*deviceData, len(deviceIDs))
	order := make([]uuid.UUID, 0, len(deviceIDs))

	objectRows, err := tx.Query(ctx, `
		SELECT DISTINCT a.device_dim_id, a.object_type, a.object_ldn
		  FROM pm_measurement_anchors a
		 WHERE a."time">=$1 AND a."time"<$2
		   AND a.device_dim_id=ANY($3::uuid[])
		 ORDER BY a.device_dim_id, a.object_ldn`,
		w.Start, w.End, deviceIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("load versioned hourly formula objects: %w", err)
	}
	for objectRows.Next() {
		var id uuid.UUID
		var objectLDN string
		var objectType int16
		if err := objectRows.Scan(&id, &objectType, &objectLDN); err != nil {
			objectRows.Close()
			return 0, 0, fmt.Errorf("scan versioned hourly formula object: %w", err)
		}
		preparedDevice, ok := prepared[id]
		if !ok {
			continue
		}
		dev := byID[id]
		if dev == nil {
			dev = &deviceData{
				id: id, oui: preparedDevice.oui, sn: preparedDevice.sn,
				objects: make(map[formulaObject]map[string]float64),
			}
			byID[id] = dev
			order = append(order, id)
		}
		dev.objects[formulaObject{objectType: objectType, objectLDN: objectLDN}] =
			make(map[string]float64)
	}
	if err := objectRows.Err(); err != nil {
		objectRows.Close()
		return 0, 0, fmt.Errorf("iterate versioned hourly formula objects: %w", err)
	}
	objectRows.Close()

	dependencies, loadCounters := versionedHourlyFormulaDependencies(prepared)
	if loadCounters {
		counterQuery := `
		SELECT a.device_dim_id, a.object_type, a.object_ldn, d.metric_path,
		       ` + normalizeSQLValue(`CASE lower(d.statis_type)
		         WHEN 'sum' THEN sum(v.metric_value)
		         WHEN 'avg' THEN avg(v.metric_value)
		         WHEN 'max' THEN max(v.metric_value)
		         WHEN 'min' THEN min(v.metric_value)
		       END`, "d.unit", "d.statis_type", "$5", "d.metric_path") + `
		  FROM pm_measurement_anchors a
		  JOIN pm_metric_values v
		    ON v."time"=a."time" AND v.anchor_id=a.anchor_id
		  JOIN pm_metric_dictionary d ON d.metric_id=v.metric_id
		 WHERE a."time">=$1 AND a."time"<$2
		   AND a.device_dim_id=ANY($3::uuid[])
		   AND d.metric_type='counter'
		   AND d.metric_path=ANY($4::text[])
		   AND lower(d.statis_type) IN ('sum','avg','max','min')
		 GROUP BY a.device_dim_id, a.object_type, a.object_ldn,
		          d.metric_path, d.unit, d.statis_type
		 ORDER BY a.device_dim_id, a.object_ldn, d.metric_path`
		rows, err := tx.Query(ctx, counterQuery,
			w.Start, w.End, deviceIDs, dependencies, numberProcess)
		if err != nil {
			return 0, 0, fmt.Errorf("load versioned hourly counters: %w", err)
		}
		for rows.Next() {
			var id uuid.UUID
			var objectLDN, path string
			var objectType int16
			var value float64
			if err := rows.Scan(&id, &objectType, &objectLDN, &path, &value); err != nil {
				rows.Close()
				return 0, 0, fmt.Errorf("scan versioned hourly counter: %w", err)
			}
			dev := byID[id]
			if dev == nil {
				continue
			}
			object := formulaObject{objectType: objectType, objectLDN: objectLDN}
			counters := dev.objects[object]
			if counters == nil {
				counters = make(map[string]float64)
				dev.objects[object] = counters
			}
			counters[path] = value
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("iterate versioned hourly counters: %w", err)
		}
		rows.Close()
	}

	type formulaGroup struct {
		deviceID uuid.UUID
		object   formulaObject
		rows     []kpiRow
	}
	var groups []formulaGroup
	for _, id := range order {
		dev := byID[id]
		preparedDevice := prepared[id]
		formulas := make([]router.KPIDef, 0, len(preparedDevice.kpis))
		for _, kpi := range preparedDevice.kpis {
			formulas = append(formulas, kpi.Definition)
		}
		for object, counters := range dev.objects {
			calculated, err := a.evalKPIs(
				entityKey{oui: dev.oui, sn: dev.sn, objectLdn: object.objectLDN},
				formulas, counters, numberProcess)
			if err != nil {
				return 0, 0, err
			}
			if len(calculated) > 0 {
				groups = append(groups, formulaGroup{deviceID: id, object: object, rows: calculated})
			}
		}
	}
	if len(groups) == 0 {
		return 0, 0, nil
	}

	metricIDs := make(map[string]int64)
	for _, device := range prepared {
		for _, kpi := range device.kpis {
			if current, ok := metricIDs[kpi.Definition.IndicatorID]; ok && current != kpi.MetricID {
				return 0, 0, fmt.Errorf("prepared hourly formula KPI %q has inconsistent metric IDs", kpi.Definition.IndicatorID)
			}
			metricIDs[kpi.Definition.IndicatorID] = kpi.MetricID
		}
	}

	groupSetIDs := make([]int64, len(groups))
	for index, group := range groups {
		setID, ok := formulaSetIDs[group.deviceID]
		if !ok {
			return 0, 0, fmt.Errorf(
				"resolve prepared hourly formula KPI metric set for device %s",
				group.deviceID,
			)
		}
		groupSetIDs[index] = setID
	}

	deviceArray := make([]uuid.UUID, len(groups))
	objectTypeArray := make([]int16, len(groups))
	objectLDNArray := make([]string, len(groups))
	for index, group := range groups {
		deviceArray[index] = group.deviceID
		objectTypeArray[index] = group.object.objectType
		objectLDNArray[index] = group.object.objectLDN
	}
	anchorRows, err := tx.Query(ctx, `
		INSERT INTO pm_hourly_anchors
		    ("time",bucket_version,device_dim_id,object_type,object_ldn,counter_group,
		     metric_set_id,granularity,start_time,end_time)
		SELECT $1,$2,device_id,object_type,object_ldn,'__kpi_formula__',
		       metric_set_id,'hourly',$1,$3
		  FROM unnest($4::uuid[],$5::smallint[],$6::text[],$7::bigint[])
		    AS input(device_id,object_type,object_ldn,metric_set_id)
		RETURNING anchor_id,device_dim_id,object_type,object_ldn`,
		w.Start, version, w.End, deviceArray, objectTypeArray, objectLDNArray, groupSetIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("insert hourly formula KPI anchors: %w", err)
	}
	type anchorKey struct {
		deviceID   uuid.UUID
		objectType int16
		objectLDN  string
	}
	anchorIDs := make(map[anchorKey]int64, len(groups))
	for anchorRows.Next() {
		var anchorID int64
		var key anchorKey
		if err := anchorRows.Scan(&anchorID, &key.deviceID, &key.objectType, &key.objectLDN); err != nil {
			anchorRows.Close()
			return 0, 0, fmt.Errorf("scan hourly formula KPI anchor: %w", err)
		}
		anchorIDs[key] = anchorID
	}
	if err := anchorRows.Err(); err != nil {
		anchorRows.Close()
		return 0, 0, fmt.Errorf("iterate hourly formula KPI anchors: %w", err)
	}
	anchorRows.Close()
	if len(anchorIDs) != len(groups) {
		return 0, 0, fmt.Errorf("inserted %d of %d hourly formula KPI anchors", len(anchorIDs), len(groups))
	}

	valueRows := make([][]any, 0)
	for _, group := range groups {
		anchorID, ok := anchorIDs[anchorKey{
			deviceID: group.deviceID, objectType: group.object.objectType, objectLDN: group.object.objectLDN,
		}]
		if !ok {
			return 0, 0, fmt.Errorf("resolve inserted hourly formula KPI anchor")
		}
		for _, row := range group.rows {
			valueRows = append(valueRows, []any{w.Start, version, anchorID, metricIDs[row.path], row.value})
		}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"pm_hourly_values"},
		[]string{"time", "bucket_version", "anchor_id", "metric_id", "metric_value"},
		pgx.CopyFromRows(valueRows)); err != nil {
		return 0, 0, fmt.Errorf("copy hourly formula KPI values: %w", err)
	}
	return int64(len(anchorIDs)), int64(len(valueRows)), nil
}

func versionedHourlyFormulaDependencies(
	prepared map[uuid.UUID]preparedFormulaDevice,
) ([]string, bool) {
	unique := make(map[string]struct{})
	for _, device := range prepared {
		for _, kpi := range device.kpis {
			for _, dependency := range kpi.Definition.Dependencies {
				unique[dependency] = struct{}{}
			}
		}
	}
	dependencies := make([]string, 0, len(unique))
	for dependency := range unique {
		dependencies = append(dependencies, dependency)
	}
	sort.Strings(dependencies)
	return dependencies, len(dependencies) > 0
}

func resolveVersionedHourlyFormulaMetricSet(
	ctx context.Context,
	tx DBTX,
	hash []byte,
	metricIDs []int64,
) (int64, error) {
	query := func() (int64, []int64, bool, error) {
		sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Select("metric_set_id", "metric_ids").
			From("pm_metric_sets").
			Where(sq.Eq{
				"product_key":   "__hourly_formula__",
				"counter_group": "__kpi_formula__",
				"content_hash":  hash,
			}).
			ToSql()
		if err != nil {
			return 0, nil, false, fmt.Errorf("build hourly formula KPI metric set query: %w", err)
		}
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			return 0, nil, false, fmt.Errorf("query hourly formula KPI metric set: %w", err)
		}
		defer rows.Close()
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return 0, nil, false, fmt.Errorf("iterate hourly formula KPI metric set: %w", err)
			}
			return 0, nil, false, nil
		}
		var setID int64
		var storedIDs []int64
		if err := rows.Scan(&setID, &storedIDs); err != nil {
			return 0, nil, false, fmt.Errorf("scan hourly formula KPI metric set: %w", err)
		}
		if rows.Next() {
			return 0, nil, false, fmt.Errorf("query hourly formula KPI metric set: duplicate stable key")
		}
		if err := rows.Err(); err != nil {
			return 0, nil, false, fmt.Errorf("iterate hourly formula KPI metric set: %w", err)
		}
		return setID, storedIDs, true, nil
	}
	validate := func(setID int64, storedIDs []int64) error {
		if len(storedIDs) != len(metricIDs) {
			return fmt.Errorf("resolve hourly formula KPI metric set %d: inconsistent metric IDs", setID)
		}
		for i := range metricIDs {
			if storedIDs[i] != metricIDs[i] {
				return fmt.Errorf("resolve hourly formula KPI metric set %d: inconsistent metric IDs", setID)
			}
		}
		return nil
	}

	setID, storedIDs, ok, err := query()
	if err != nil {
		return 0, err
	}
	if ok {
		if err := validate(setID, storedIDs); err != nil {
			return 0, err
		}
		return setID, nil
	}
	insertSQL, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("pm_metric_sets").
		Columns("product_key", "counter_group", "content_hash", "metric_ids").
		Values("__hourly_formula__", "__kpi_formula__", hash, metricIDs).
		Suffix("ON CONFLICT (product_key,counter_group,content_hash) DO NOTHING").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build hourly formula KPI metric set insert: %w", err)
	}
	if _, err := tx.Exec(ctx, insertSQL, args...); err != nil {
		return 0, fmt.Errorf("insert hourly formula KPI metric set: %w", err)
	}
	setID, storedIDs, ok, err = query()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("resolve hourly formula KPI metric set: stable key was not resolved")
	}
	if err := validate(setID, storedIDs); err != nil {
		return 0, err
	}
	return setID, nil
}

func buildEnsureVersionedHourlyMetricSetsSQL() string {
	return `
WITH group_sets AS (
    SELECT a.counter_group,
           array_agg(DISTINCT member.metric_id ORDER BY member.metric_id) AS metric_ids
      FROM pm_measurement_anchors a
      JOIN pm_metric_sets s ON s.metric_set_id = a.metric_set_id
      CROSS JOIN LATERAL unnest(s.metric_ids) member(metric_id)
     WHERE a."time" >= $1 AND a."time" < $2
       AND a.device_dim_id = ANY($3::uuid[])
     GROUP BY a.device_dim_id, a.object_type, a.object_ldn, a.counter_group
)
INSERT INTO pm_metric_sets (product_key, counter_group, content_hash, metric_ids)
SELECT DISTINCT '__hourly__', gs.counter_group,
       decode(md5(array_to_string(gs.metric_ids, ',')), 'hex'), gs.metric_ids
  FROM group_sets gs
ON CONFLICT (product_key, counter_group, content_hash) DO NOTHING`
}

func buildResolveVersionedHourlyBaseGroupsSQL() string {
	return `
WITH group_sets AS (
    SELECT a.device_dim_id, a.object_type, a.object_ldn, a.counter_group,
           array_agg(DISTINCT member.metric_id ORDER BY member.metric_id) AS metric_ids
      FROM pm_measurement_anchors a
      JOIN pm_metric_sets source_set ON source_set.metric_set_id = a.metric_set_id
      CROSS JOIN LATERAL unnest(source_set.metric_ids) member(metric_id)
     WHERE a."time" >= $1 AND a."time" < $2
       AND a.device_dim_id = ANY($3::uuid[])
     GROUP BY a.device_dim_id, a.object_type, a.object_ldn, a.counter_group
)
SELECT gs.device_dim_id, gs.object_type, gs.object_ldn, gs.counter_group,
       resolved.metric_set_id
  FROM group_sets gs
  JOIN pm_metric_sets resolved
    ON resolved.product_key='__hourly__'
   AND resolved.counter_group=gs.counter_group
   AND resolved.content_hash=decode(md5(array_to_string(gs.metric_ids, ',')), 'hex')
   AND resolved.metric_ids=gs.metric_ids
 ORDER BY gs.device_dim_id, gs.object_type, gs.object_ldn, gs.counter_group`
}

func buildVersionedHourlyBatchSQL() string {
	return `
WITH anchors AS (
    INSERT INTO pm_hourly_anchors
        ("time", bucket_version, device_dim_id, object_type, object_ldn,
         counter_group, metric_set_id, granularity, start_time, end_time)
    SELECT $2, $1, input.device_dim_id, input.object_type, input.object_ldn,
           input.counter_group, input.metric_set_id, 'hourly', $2, $3
      FROM unnest($4::uuid[], $5::smallint[], $6::text[], $7::text[], $8::bigint[])
        AS input(device_dim_id, object_type, object_ldn, counter_group, metric_set_id)
    RETURNING anchor_id, "time", device_dim_id, object_type, object_ldn, counter_group
),
values_inserted AS (
    INSERT INTO pm_hourly_values ("time", bucket_version, anchor_id, metric_id, metric_value)
    SELECT $2, $1, ha.anchor_id, v.metric_id,
           ` + normalizeSQLValue(`CASE lower(d.statis_type)
             WHEN 'sum' THEN sum(v.metric_value)
             WHEN 'avg' THEN avg(v.metric_value)
             WHEN 'max' THEN max(v.metric_value)
             WHEN 'min' THEN min(v.metric_value)
           END`, "d.unit", "d.statis_type", "$10", "d.metric_path") + `
      FROM anchors ha
      JOIN pm_measurement_anchors a
        ON a.device_dim_id = ha.device_dim_id
       AND a.object_type = ha.object_type
       AND a.object_ldn = ha.object_ldn
       AND a.counter_group = ha.counter_group
       AND a."time" >= $2 AND a."time" < $3
      JOIN pm_metric_values v ON v."time" = a."time" AND v.anchor_id = a.anchor_id
      JOIN pm_metric_dictionary d ON d.metric_id = v.metric_id
     WHERE lower(d.statis_type) IN ('sum','avg','max','min')
     GROUP BY ha.anchor_id, v.metric_id, d.metric_path, d.unit, d.statis_type
    RETURNING anchor_id
),
counts AS (
    SELECT (SELECT count(*) FROM anchors)::bigint AS anchors,
           (SELECT count(*) FROM values_inserted)::bigint AS values
),
batch_completed AS (
    UPDATE pm_hourly_rollup_batches b
       SET anchor_count=c.anchors, value_count=c.values
     FROM counts c
     WHERE b.bucket_version=$1 AND b.batch_no=$9
    RETURNING c.anchors, c.values
),
version_progress AS (
    UPDATE pm_hourly_bucket_versions v
       SET completed_batches=completed_batches+1,
           anchor_count=anchor_count+b.anchors,
           value_count=value_count+b.values
      FROM batch_completed b
     WHERE v.bucket_version=$1 AND v.status='building'
    RETURNING b.anchors, b.values
)
SELECT anchors, values FROM version_progress`
}

func buildCompleteVersionedHourlyBatchSQL() string {
	return `
UPDATE pm_hourly_rollup_batches
   SET status='completed', finished_at=clock_timestamp()
 WHERE bucket_version=$1 AND batch_no=$2`
}

func buildPublishHourlyVersionSQL() string {
	return `
UPDATE pm_hourly_bucket_versions
   SET status = 'active', published_at=now()
 WHERE bucket_version=$1 AND status='building'
RETURNING bucket_version`
}

func buildSupersedeHourlyVersionSQL() string {
	return `
UPDATE pm_hourly_bucket_versions
   SET status = 'superseded'
 WHERE bucket_start=$1 AND status='active' AND bucket_version<>$2`
}
