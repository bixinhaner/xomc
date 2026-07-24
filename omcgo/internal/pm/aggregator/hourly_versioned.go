package aggregator

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

const DefaultHourlyBatchDevices = 200

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

func (a *Aggregator) supportsVersionedHourly() bool {
	_, ok := a.db.(poolAcquirer)
	return ok
}

// RunHourlyVersioned builds a complete invisible version in bounded transactions,
// then atomically makes it the only active version for the bucket.
func (a *Aggregator) RunHourlyVersioned(ctx context.Context, w WindowSpec, batchSize int) (stats RollupStats, err error) {
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
		tx, beginErr := conn.Begin(ctx)
		if beginErr != nil {
			return stats, fmt.Errorf("begin hourly batch %d: %w", batchNo, beginErr)
		}
		var anchorCount, valueCount int64
		if _, insertErr := tx.Exec(ctx, `
			INSERT INTO pm_hourly_rollup_batches
			    (bucket_version, batch_no, status, device_count)
			VALUES ($1,$2,'building',$3)`,
			version, batchNo, len(deviceIDs)); insertErr != nil {
			_ = tx.Rollback(ctx)
			return stats, fmt.Errorf("start hourly batch %d: %w", batchNo, insertErr)
		}
		scanErr := tx.QueryRow(ctx, buildVersionedHourlyBatchSQL(),
			version, w.Start, w.End, deviceIDs, batchNo, numberProcess).Scan(&anchorCount, &valueCount)
		if scanErr != nil {
			_ = tx.Rollback(ctx)
			return stats, fmt.Errorf("build hourly batch %d: %w", batchNo, scanErr)
		}
		formulaAnchors, formulaValues, formulaErr := a.insertVersionedHourlyFormulaKPIs(
			ctx, tx, version, deviceIDs, w, numberProcess)
		if formulaErr != nil {
			_ = tx.Rollback(ctx)
			return stats, fmt.Errorf("build hourly formula KPI batch %d: %w", batchNo, formulaErr)
		}
		if formulaAnchors > 0 || formulaValues > 0 {
			if _, updateErr := tx.Exec(ctx, `
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
				version, batchNo, formulaAnchors, formulaValues); updateErr != nil {
				_ = tx.Rollback(ctx)
				return stats, fmt.Errorf("account hourly formula KPI batch %d: %w", batchNo, updateErr)
			}
			anchorCount += formulaAnchors
			valueCount += formulaValues
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return stats, fmt.Errorf("commit hourly batch %d: %w", batchNo, commitErr)
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

func (a *Aggregator) insertVersionedHourlyFormulaKPIs(
	ctx context.Context,
	tx pgx.Tx,
	version int64,
	deviceIDs []uuid.UUID,
	w WindowSpec,
	numberProcess string,
) (int64, int64, error) {
	if a.kpiRouter == nil || len(deviceIDs) == 0 {
		return 0, 0, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT ha.device_dim_id, COALESCE(dev.oui,''), COALESCE(dev.serial_number,''),
		       ha.object_type, ha.object_ldn, d.metric_path, hv.metric_value
		  FROM pm_hourly_anchors ha
		  JOIN pm_hourly_values hv
		    ON hv.bucket_version=ha.bucket_version
		   AND hv."time"=ha."time" AND hv.anchor_id=ha.anchor_id
		  JOIN pm_metric_dictionary d ON d.metric_id=hv.metric_id
		  LEFT JOIN device_dim dev ON dev.id=ha.device_dim_id
		 WHERE ha.bucket_version=$1
		   AND ha.device_dim_id=ANY($2::uuid[])
		   AND d.metric_type='counter'
		 ORDER BY ha.device_dim_id, ha.object_ldn, d.metric_path`,
		version, deviceIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("load versioned hourly counters: %w", err)
	}
	defer rows.Close()

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
	for rows.Next() {
		var id uuid.UUID
		var oui, sn, objectLDN, path string
		var objectType int16
		var value float64
		if err := rows.Scan(&id, &oui, &sn, &objectType, &objectLDN, &path, &value); err != nil {
			return 0, 0, fmt.Errorf("scan versioned hourly counter: %w", err)
		}
		dev := byID[id]
		if dev == nil {
			dev = &deviceData{id: id, oui: oui, sn: sn, objects: make(map[formulaObject]map[string]float64)}
			byID[id] = dev
			order = append(order, id)
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
		return 0, 0, fmt.Errorf("iterate versioned hourly counters: %w", err)
	}
	rows.Close()

	type formulaGroup struct {
		deviceID uuid.UUID
		object   formulaObject
		rows     []kpiRow
	}
	var groups []formulaGroup
	for _, id := range order {
		dev := byID[id]
		route, err := a.kpiRouter.LookupByDevice(ctx, dev.sn)
		if err != nil {
			return 0, 0, fmt.Errorf("lookup hourly KPI route for %s: %w", dev.sn, err)
		}
		if route == nil {
			continue
		}
		formulas, _ := splitKPIDefsByRollupMode(route.KPIs)
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

	meta := make(map[string]kpiRow)
	for _, group := range groups {
		for _, row := range group.rows {
			meta[row.path] = row
		}
	}
	paths := make([]string, 0, len(meta))
	for path := range meta {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	statis, units := make([]string, len(paths)), make([]string, len(paths))
	for i, path := range paths {
		statis[i], units[i] = meta[path].stype, meta[path].unit
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path, report_key, metric_type, statis_type, unit)
		SELECT path, path, 'kpi', NULLIF(statis,''), NULLIF(unit,'')
		  FROM unnest($1::text[], $2::text[], $3::text[]) x(path,statis,unit)
		ON CONFLICT (metric_path) DO UPDATE SET
		  metric_type='kpi',
		  statis_type=COALESCE(EXCLUDED.statis_type,pm_metric_dictionary.statis_type),
		  unit=COALESCE(EXCLUDED.unit,pm_metric_dictionary.unit),
		  updated_at=now()`, paths, statis, units); err != nil {
		return 0, 0, fmt.Errorf("register hourly formula KPI dictionary: %w", err)
	}
	dictRows, err := tx.Query(ctx,
		`SELECT metric_path, metric_id FROM pm_metric_dictionary WHERE metric_path=ANY($1::text[])`, paths)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve hourly formula KPI dictionary: %w", err)
	}
	metricIDs := make(map[string]int64, len(paths))
	for dictRows.Next() {
		var path string
		var metricID int64
		if err := dictRows.Scan(&path, &metricID); err != nil {
			dictRows.Close()
			return 0, 0, fmt.Errorf("scan hourly formula KPI dictionary: %w", err)
		}
		metricIDs[path] = metricID
	}
	if err := dictRows.Err(); err != nil {
		dictRows.Close()
		return 0, 0, fmt.Errorf("iterate hourly formula KPI dictionary: %w", err)
	}
	dictRows.Close()

	groupSetIDs := make([]int64, len(groups))
	setCache := make(map[string]int64)
	for index, group := range groups {
		setIDs := make([]int64, 0, len(group.rows))
		for _, row := range group.rows {
			setIDs = append(setIDs, metricIDs[row.path])
		}
		sort.Slice(setIDs, func(i, j int) bool { return setIDs[i] < setIDs[j] })
		hash := metrics.MetricSetHash(setIDs)
		cacheKey := string(hash[:])
		setID, ok := setCache[cacheKey]
		if !ok {
			if err := tx.QueryRow(ctx, `
				INSERT INTO pm_metric_sets (product_key,counter_group,content_hash,metric_ids)
				VALUES ('__hourly_formula__','__kpi_formula__',$1,$2)
				ON CONFLICT (product_key,counter_group,content_hash)
				DO UPDATE SET metric_ids=EXCLUDED.metric_ids
				RETURNING metric_set_id`, hash[:], setIDs).Scan(&setID); err != nil {
				return 0, 0, fmt.Errorf("resolve hourly formula KPI metric set: %w", err)
			}
			setCache[cacheKey] = setID
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

func buildVersionedHourlyBatchSQL() string {
	return `
WITH group_sets AS (
    SELECT a.device_dim_id, a.object_type, a.object_ldn, a.counter_group,
           array_agg(DISTINCT member.metric_id ORDER BY member.metric_id) AS metric_ids
      FROM pm_measurement_anchors a
      JOIN pm_metric_sets s ON s.metric_set_id = a.metric_set_id
      CROSS JOIN LATERAL unnest(s.metric_ids) member(metric_id)
     WHERE a."time" >= $2 AND a."time" < $3
       AND a.device_dim_id = ANY($4::uuid[])
     GROUP BY a.device_dim_id, a.object_type, a.object_ldn, a.counter_group
),
resolved_sets AS (
    INSERT INTO pm_metric_sets (product_key, counter_group, content_hash, metric_ids)
    SELECT DISTINCT '__hourly__', gs.counter_group,
           decode(md5(array_to_string(gs.metric_ids, ',')), 'hex'), gs.metric_ids
      FROM group_sets gs
    ON CONFLICT (product_key, counter_group, content_hash)
    DO UPDATE SET metric_ids=EXCLUDED.metric_ids
    RETURNING metric_set_id, counter_group, content_hash
),
anchors AS (
    INSERT INTO pm_hourly_anchors
        ("time", bucket_version, device_dim_id, object_type, object_ldn,
         counter_group, metric_set_id, granularity, start_time, end_time)
    SELECT $2, $1, gs.device_dim_id, gs.object_type, gs.object_ldn,
           gs.counter_group, ms.metric_set_id, 'hourly', $2, $3
      FROM group_sets gs
      JOIN resolved_sets ms
        ON ms.counter_group = gs.counter_group
       AND ms.content_hash = decode(md5(array_to_string(gs.metric_ids, ',')), 'hex')
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
           END`, "d.unit", "d.statis_type", "$6", "d.metric_path") + `
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
       SET status='completed', anchor_count=c.anchors, value_count=c.values, finished_at=now()
      FROM counts c
     WHERE b.bucket_version=$1 AND b.batch_no=$5
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
