package metrics

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SparseValue is an actually reported finite PM value. Missing values remain
// represented only by the containing measurement's MetricPaths.
type SparseValue struct {
	Path       string
	MetricType MetricType
	Value      float64
	StatisType string
	Unit       string
}

func writeSparseMeasurements(ctx context.Context, tx pgx.Tx, fileID, batchID *uuid.UUID, measurements []SparseMeasurement) error {
	if len(measurements) == 0 {
		return nil
	}
	defs, err := sparseMeasurementDefinitions(measurements)
	if err != nil {
		return fmt.Errorf("collect sparse metric definitions: %w", err)
	}
	// Lock order invariant: hourly bucket/version state → dictionary → metric set
	// → anchors/values. This remains in the ingest transaction for atomic rollback.
	if err := markHourlyBucketsDirty(ctx, tx, measurements); err != nil {
		return err
	}
	ids, err := resolveMetricDictionary(ctx, tx, defs)
	if err != nil {
		return fmt.Errorf("resolve pm metric dictionary: %w", err)
	}

	productKeys, err := resolveSparseProductKeys(ctx, tx, measurements)
	if err != nil {
		return err
	}
	valueRows := make([][]any, 0)
	for _, m := range measurements {
		setIDs := make([]int64, 0, len(m.MetricPaths))
		for _, path := range m.MetricPaths {
			id, ok := ids[path]
			if !ok {
				return fmt.Errorf("metric path %q was not resolved", path)
			}
			setIDs = append(setIDs, id)
		}
		sort.Slice(setIDs, func(i, j int) bool { return setIDs[i] < setIDs[j] })
		hash := MetricSetHash(setIDs)
		productKey := productKeys[m.DeviceID]
		setID, err := resolveMetricSet(ctx, tx, productKey, m.CounterGroup, hex.EncodeToString(hash[:]), setIDs)
		if err != nil {
			return fmt.Errorf("resolve pm metric set: %w", err)
		}
		var anchorID int64
		if err := tx.QueryRow(ctx, `
INSERT INTO pm_measurement_anchors
  ("time",device_dim_id,object_ldn,counter_group,metric_set_id,source_file_id,
   granularity,ingest_batch_id,start_time,end_time)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING anchor_id`,
			m.Time, m.DeviceID, m.ObjectLDN, m.CounterGroup, setID, fileID,
			granularityText(m.Granularity), batchID, m.StartTime, m.EndTime).Scan(&anchorID); err != nil {
			return fmt.Errorf("insert pm measurement anchor: %w", err)
		}
		for _, value := range m.Values {
			valueRows = append(valueRows, []any{m.Time, anchorID, ids[value.Path], value.Value})
		}
	}
	if len(valueRows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"pm_metric_values"},
			[]string{"time", "anchor_id", "metric_id", "metric_value"}, pgx.CopyFromRows(valueRows)); err != nil {
			return fmt.Errorf("copy pm sparse values: %w", err)
		}
	}
	return nil
}

func sparseMeasurementDefinitions(measurements []SparseMeasurement) ([]metricDefinition, error) {
	defs := make([]metricDefinition, 0)
	explicitPaths := make(map[string]struct{})
	appendValue := func(value SparseValue) {
		defs = append(defs, metricDefinition{
			path: value.Path, metricType: value.MetricType,
			statisType: value.StatisType, unit: value.Unit,
		})
		explicitPaths[value.Path] = struct{}{}
	}
	for _, measurement := range measurements {
		for _, metric := range measurement.Metrics {
			appendValue(metric)
		}
		for _, value := range measurement.Values {
			appendValue(value)
		}
	}
	for _, measurement := range measurements {
		for _, path := range measurement.MetricPaths {
			if _, ok := explicitPaths[path]; !ok {
				defs = append(defs, metricDefinition{path: path, metricType: MetricTypeCounter})
			}
		}
	}
	return canonicalMetricDefinitions(defs)
}

// BuildSparseMeasurementsFromMetrics adapts administrative counter/KPI writes
// to the same sparse anchor/value representation used by file ingestion.
func BuildSparseMeasurementsFromMetrics(ms []PMMetric) ([]SparseMeasurement, error) {
	type key struct {
		deviceID                  uuid.UUID
		oui, sn, ldn, group, gran string
		startNS, endNS, timeNS    int64
	}
	groups := make(map[key]*SparseMeasurement)
	order := make([]key, 0)
	for _, m := range ms {
		deviceID, err := metricDeviceID(m)
		if err != nil {
			return nil, err
		}
		ldn := ""
		if m.ObjectLDN != nil {
			ldn = *m.ObjectLDN
		}
		group := "__kpi__"
		if m.MetricType != MetricTypeKPI {
			group = "__counter__"
			if v, ok := m.Extra["counter_group"].(string); ok && v != "" {
				group = v
			}
		}
		start, end, point := m.StartTime, m.EndTime, m.Time
		if point.IsZero() {
			point = start
		}
		if start.IsZero() {
			start = point
		}
		if end.IsZero() {
			end = point
		}
		k := key{deviceID, m.DeviceOUI, m.DeviceSN, ldn, group, string(m.Granularity),
			start.UnixNano(), end.UnixNano(), point.UnixNano()}
		g := groups[k]
		if g == nil {
			g = &SparseMeasurement{
				DeviceID: deviceID, DeviceOUI: m.DeviceOUI, DeviceSN: m.DeviceSN,
				ObjectLDN: ldn, CounterGroup: group, Time: point,
				StartTime: start, EndTime: end, Granularity: granularityMinutes(m.Granularity),
			}
			groups[k] = g
			order = append(order, k)
		}
		g.MetricPaths = append(g.MetricPaths, m.MetricPath)
		statis := ""
		if m.StatisType != nil {
			statis = string(*m.StatisType)
		}
		metric := SparseValue{
			Path: m.MetricPath, MetricType: m.MetricType,
			Value: m.MetricValue, StatisType: statis,
		}
		g.Metrics = append(g.Metrics, metric)
		if !math.IsNaN(m.MetricValue) && !math.IsInf(m.MetricValue, 0) {
			g.Values = append(g.Values, metric)
		}
	}
	out := make([]SparseMeasurement, 0, len(order))
	for _, k := range order {
		g := groups[k]
		g.MetricPaths = uniqueSorted(g.MetricPaths)
		lastMeta := make(map[string]SparseValue, len(g.Metrics))
		for _, metric := range g.Metrics {
			lastMeta[metric.Path] = metric
		}
		g.Metrics = g.Metrics[:0]
		for _, path := range g.MetricPaths {
			g.Metrics = append(g.Metrics, lastMeta[path])
		}
		last := make(map[string]SparseValue, len(g.Values))
		for _, value := range g.Values {
			last[value.Path] = value
		}
		g.Values = g.Values[:0]
		for _, path := range g.MetricPaths {
			if value, ok := last[path]; ok {
				g.Values = append(g.Values, value)
			}
		}
		out = append(out, *g)
	}
	return out, nil
}

func metricDeviceID(m PMMetric) (uuid.UUID, error) {
	raw, ok := m.Extra["device_id"]
	if !ok {
		return uuid.Nil, fmt.Errorf("sparse metric %s missing device_id", m.MetricPath)
	}
	text, ok := raw.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("sparse metric %s has non-string device_id", m.MetricPath)
	}
	id, err := uuid.Parse(text)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse sparse metric %s device_id: %w", m.MetricPath, err)
	}
	return id, nil
}

func granularityMinutes(g Granularity) int {
	switch g {
	case GranularityHourly:
		return 60
	case GranularityDaily:
		return 24 * 60
	case GranularityWeekly:
		return 7 * 24 * 60
	case GranularityMonthly:
		return 30 * 24 * 60
	default:
		return 15
	}
}

func buildDeleteKPIAnchorsSQL() []string {
	const candidates = `
SELECT a."time", a.anchor_id
FROM pm_measurement_anchors a
JOIN device_dim d ON d.id = a.device_dim_id
WHERE d.oui = $1 AND d.serial_number = $2
  AND a.object_ldn = $3 AND a.counter_group = '__kpi__'
  AND a.granularity = '15min' AND a.end_time = $4`
	return []string{
		`DELETE FROM pm_metric_values v USING (` + candidates + `) doomed
		 WHERE v."time" = doomed."time" AND v.anchor_id = doomed.anchor_id`,
		`DELETE FROM pm_measurement_anchors a USING (` + candidates + `) doomed
		 WHERE a."time" = doomed."time" AND a.anchor_id = doomed.anchor_id`,
	}
}

// DeleteKPIAnchorsTx removes a recompute scope from physical sparse tables.
func DeleteKPIAnchorsTx(ctx context.Context, tx pgx.Tx, oui, sn, objectLDN string, end time.Time) error {
	for _, sql := range buildDeleteKPIAnchorsSQL() {
		if _, err := tx.Exec(ctx, sql, oui, sn, objectLDN, end); err != nil {
			return fmt.Errorf("delete sparse KPI anchors: %w", err)
		}
	}
	return nil
}

func granularityText(minutes int) string {
	switch minutes {
	case 60:
		return string(GranularityHourly)
	case 24 * 60:
		return string(GranularityDaily)
	case 7 * 24 * 60:
		return string(GranularityWeekly)
	case 30 * 24 * 60:
		return string(GranularityMonthly)
	default:
		return string(Granularity15Min)
	}
}

func buildMarkHourlyBucketsDirtySQL() string {
	return `UPDATE pm_hourly_bucket_versions
SET dirty = true
WHERE status IN ('active','building')
  AND bucket_start = ANY($1::timestamptz[])`
}

func markHourlyBucketsDirty(ctx context.Context, tx pgx.Tx, measurements []SparseMeasurement) error {
	seen := make(map[time.Time]struct{})
	hours := make([]time.Time, 0)
	for _, measurement := range measurements {
		hour := measurement.Time.UTC().Truncate(time.Hour)
		if _, ok := seen[hour]; ok {
			continue
		}
		seen[hour] = struct{}{}
		hours = append(hours, hour)
	}
	return markHourlyStartsDirty(ctx, tx, hours)
}

func markHourlyStartsDirty(ctx context.Context, tx pgx.Tx, hours []time.Time) error {
	if len(hours) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, buildMarkHourlyBucketsDirtySQL(), hours); err != nil {
		return fmt.Errorf("mark hourly buckets dirty: %w", err)
	}
	return nil
}

// SparseMeasurement is the semantic anchor for one device/object/window/group.
type SparseMeasurement struct {
	DeviceID     uuid.UUID
	DeviceOUI    string
	DeviceSN     string
	ObjectLDN    string
	CounterGroup string
	Time         time.Time
	StartTime    time.Time
	EndTime      time.Time
	Granularity  int
	MetricPaths  []string
	Metrics      []SparseValue
	Values       []SparseValue
}

// BuildSparseMeasurements converts the current in-memory compatibility
// representation (including NaN placeholders) into anchors plus finite values.
func BuildSparseMeasurements(counters []model.PMCounter, kpis []model.KPIValue) []SparseMeasurement {
	type key struct {
		deviceID            uuid.UUID
		oui, sn, ldn, group string
		endNS               int64
		gran                int
	}
	groups := make(map[key]*SparseMeasurement)
	order := make([]key, 0)
	add := func(k key, path string, value SparseValue) {
		g := groups[k]
		if g == nil {
			end := time.Unix(0, k.endNS).UTC()
			start := end.Add(-time.Duration(k.gran) * time.Minute)
			g = &SparseMeasurement{
				DeviceID: k.deviceID, DeviceOUI: k.oui, DeviceSN: k.sn,
				ObjectLDN: k.ldn, CounterGroup: k.group, Time: start,
				StartTime: start, EndTime: end, Granularity: k.gran,
			}
			groups[k] = g
			order = append(order, k)
		}
		g.MetricPaths = append(g.MetricPaths, path)
		g.Metrics = append(g.Metrics, value)
		if !math.IsNaN(value.Value) && !math.IsInf(value.Value, 0) {
			g.Values = append(g.Values, value)
		}
	}
	for _, c := range counters {
		add(key{c.DeviceID, c.OUI, c.DeviceSN, c.CellID, c.CounterGroup, c.Time.UnixNano(), c.Granularity},
			c.CounterName, SparseValue{Path: c.CounterName, MetricType: MetricTypeCounter, Value: c.CounterValue, StatisType: c.StatisType, Unit: c.Unit})
	}
	for _, v := range kpis {
		add(key{v.DeviceID, v.OUI, v.DeviceSN, v.CellID, "__kpi__", v.Time.UnixNano(), 15},
			v.IndicatorID, SparseValue{Path: v.IndicatorID, MetricType: MetricTypeKPI, Value: v.KPIValue, StatisType: v.StatisType, Unit: v.Unit})
	}
	for _, k := range order {
		g := groups[k]
		g.MetricPaths = uniqueSorted(g.MetricPaths)
		lastMeta := make(map[string]SparseValue, len(g.Metrics))
		for _, metric := range g.Metrics {
			lastMeta[metric.Path] = metric
		}
		g.Metrics = g.Metrics[:0]
		for _, path := range g.MetricPaths {
			g.Metrics = append(g.Metrics, lastMeta[path])
		}
		// Preserve last-wins behavior for duplicate reported paths.
		last := make(map[string]SparseValue, len(g.Values))
		for _, v := range g.Values {
			last[v.Path] = v
		}
		g.Values = g.Values[:0]
		for _, path := range g.MetricPaths {
			if v, ok := last[path]; ok {
				g.Values = append(g.Values, v)
			}
		}
	}
	out := make([]SparseMeasurement, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

func uniqueSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	for _, item := range in {
		if len(out) == 0 || out[len(out)-1] != item {
			out = append(out, item)
		}
	}
	return out
}

// MetricSetHash returns a stable SHA-256 digest for a set of metric IDs.
func MetricSetHash(ids []int64) [sha256.Size]byte {
	sorted := append([]int64(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	buf := make([]byte, 8*len(sorted))
	for i, id := range sorted {
		binary.BigEndian.PutUint64(buf[i*8:], uint64(id))
	}
	return sha256.Sum256(buf)
}

func sparseProductKey(productID *uuid.UUID, deviceID uuid.UUID) string {
	if productID != nil && *productID != uuid.Nil {
		return productID.String()
	}
	return "device:" + deviceID.String()
}

func resolveSparseProductKeys(
	ctx context.Context,
	tx pgx.Tx,
	measurements []SparseMeasurement,
) (map[uuid.UUID]string, error) {
	deviceIDs := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]struct{})
	for _, measurement := range measurements {
		if _, ok := seen[measurement.DeviceID]; ok {
			continue
		}
		seen[measurement.DeviceID] = struct{}{}
		deviceIDs = append(deviceIDs, measurement.DeviceID)
	}
	productIDs := make(map[uuid.UUID]uuid.UUID)
	rows, err := tx.Query(ctx,
		`SELECT id, product_id FROM device_dim WHERE id=ANY($1::uuid[]) AND product_id IS NOT NULL`,
		deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve sparse product keys: %w", err)
	}
	for rows.Next() {
		var deviceID, productID uuid.UUID
		if err := rows.Scan(&deviceID, &productID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan sparse product key: %w", err)
		}
		productIDs[deviceID] = productID
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate sparse product keys: %w", err)
	}
	rows.Close()
	out := make(map[uuid.UUID]string, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if productID, ok := productIDs[deviceID]; ok {
			out[deviceID] = sparseProductKey(&productID, deviceID)
		} else {
			out[deviceID] = sparseProductKey(nil, deviceID)
		}
	}
	return out, nil
}

func metricSetCacheKey(productKey, group string, ids []int64) string {
	hash := MetricSetHash(ids)
	return strings.Join([]string{productKey, group, strconv.FormatUint(binary.BigEndian.Uint64(hash[:8]), 16)}, "\x00")
}
