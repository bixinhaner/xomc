package export

import (
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
)

// deviceSelectCols 是 device / adhoc 行级表流式取数的固定列序（与 Scan 一一对应）。
var deviceSelectCols = []string{
	"id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time", "object_ldn",
}

var deviceSelectColsNoID = []string{
	"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time", "object_ldn",
}

// buildDeviceKeysetSQL 构造 device 维度表的 (time, id) keyset 流式查询。
//
// 过滤语义与仪表盘聚合查询的 device 维度一致（成对 OUI/SN、metric_paths、metric_type、
// granularity、时窗、制式）；ORDER BY time, id 配 keyset 游标保证不漏不重。
// started=false 时取首批（无游标谓词）；之后用 (time, id) > (curTime, curID) 推进。
func buildDeviceKeysetSQL(table string, req aggregator.QueryRequest, objectLDNs []string, started bool, curTime time.Time, curID uuid.UUID, limit int) (string, []any) {
	inner := newRawAwareExportSelect(table, req, appendExportColumn(deviceSelectCols, "ingest_time")...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	inner = applyDeviceExportFilters(inner, req, objectLDNs)
	inner = inner.OrderBy(
		"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC", "id DESC",
	)
	b := storage.Psql.Select(prefixedExportColumns("d", deviceSelectCols)...).FromSelect(inner, "d")
	if started {
		// keyset：(time, id) 严格大于游标。time 列名带引号避免与保留字冲突。
		b = b.Where(sq.Expr(`("time", id) > (?, ?)`, curTime, curID))
	}
	b = b.OrderBy(`"time" ASC`, "id ASC").Limit(uint64(limit))
	q, args, _ := b.ToSql()
	return q, args
}

func buildDeviceOffsetSQL(table string, req aggregator.QueryRequest, objectLDNs []string, offset, limit int) (string, []any) {
	inner := newRawAwareExportSelect(table, req, appendExportColumn(deviceSelectColsNoID, "ingest_time")...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	inner = applyDeviceExportFilters(inner, req, objectLDNs)
	inner = inner.OrderBy(
		"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
	)
	b := storage.Psql.Select(prefixedExportColumns("d", deviceSelectColsNoID)...).FromSelect(inner, "d")
	b = b.OrderBy(`"time" ASC`, "device_oui ASC", "device_sn ASC", "object_ldn ASC", "metric_path ASC", "metric_type ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset))
	q, args, _ := b.ToSql()
	return q, args
}

func newRawAwareExportSelect(
	table string,
	req aggregator.QueryRequest,
	columns ...string,
) sq.SelectBuilder {
	if len(req.MetricPaths) == 0 || table != "pm_metrics" {
		return storage.Psql.Select(columns...).From(table)
	}
	targeted := storage.Psql.Select(
		"md5(a.anchor_id::text||':'||d.metric_id::text)::uuid AS id",
		"COALESCE(dev.oui,'')::text AS device_oui",
		"COALESCE(dev.serial_number,f.device_sn)::text AS device_sn",
		"d.metric_path", "d.metric_type", "v.metric_value", "d.statis_type", "a.granularity",
		`a."time"`, "a.start_time", "a.end_time",
		"COALESCE(b.committed_at,f.created_at,now()) AS ingest_time", "a.object_ldn",
	).From("pm_measurement_anchors a").
		Join("pm_metric_sets s ON s.metric_set_id=a.metric_set_id").
		Join("pm_metric_dictionary d ON d.metric_id=ANY(s.metric_ids)").
		LeftJoin(`pm_metric_values v ON v."time"=a."time" AND v.anchor_id=a.anchor_id AND v.metric_id=d.metric_id`).
		LeftJoin("pm_files f ON f.id=a.source_file_id").
		LeftJoin("pm_ingest_batches b ON b.ingest_batch_id=a.ingest_batch_id").
		LeftJoin("device_dim dev ON dev.id=a.device_dim_id")
	targeted = targeted.Where(sq.Eq{"d.metric_path": req.MetricPaths})
	return storage.Psql.Select(columns...).FromSelect(targeted, table)
}

func prefixedExportColumns(prefix string, cols []string) []string {
	out := make([]string, 0, len(cols))
	for _, col := range cols {
		out = append(out, prefix+"."+col)
	}
	return out
}

func appendExportColumn(cols []string, col string) []string {
	out := make([]string, len(cols), len(cols)+1)
	copy(out, cols)
	return append(out, col)
}

// adhocSelectCols 是 adhoc 取数的扩展列序（含 product_id::text 与关联名），与 adhocSource 扫描一一对应。
// 镜像网页 buildResultsQuery 的关联：LEFT JOIN product_dim / device_group_dim 把分组键 ID 解析成可读名。
var adhocSelectCols = []string{
	"r.id", "r.device_oui", "r.device_sn", "r.metric_path", "r.metric_type", "r.metric_value",
	"r.statis_type", "r.granularity", "r.time", "r.start_time", "r.end_time", "r.object_ldn",
	"r.product_id::text AS product_id", "p.product_name", "g.name AS device_group_name",
}

// buildAdhocKeysetSQL 构造 pm_adhoc_aggregation_results 的 (time, id) keyset 流式查询。
// pm_adhoc_aggregation_results 在时序库，故此 SQL 跑在 TsPool（wiring 注入 AdhocDB=TsPool）；
// products / device_groups 改读本库影子表 product_dim / device_group_dim（跨库分离）。
// LEFT JOIN 一次性把 product 维度的产品名、device_group 维度的设备组名读出，
// 不破坏流式（单次 SQL 无 N+1）。名缺失返 NULL，由 adhocSource 用 *string 承接（空 → 回退 ID 前 8）。
// device_group 维度 object_ldn 形如 'DeviceGroup=<uuid>,Tech=<制式>'，故取组名 JOIN 用
// split_part(object_ldn, ',', 1) 剥逗号前段再等值（与网页 buildResultsQuery 同口径，老行无逗号原样返回）。
func buildAdhocKeysetSQL(taskID uuid.UUID, metricPaths []string, startTime, endTime time.Time, started bool, curTime time.Time, curID uuid.UUID, limit int) (string, []any) {
	b := storage.Psql.Select(adhocSelectCols...).
		From("pm_adhoc_aggregation_results r").
		LeftJoin("product_dim p ON p.id = r.product_id").
		LeftJoin("device_group_dim g ON ('DeviceGroup=' || g.id::text) = split_part(r.object_ldn, ',', 1)").
		Where(sq.Eq{"r.task_id": taskID})
	if len(metricPaths) > 0 {
		b = b.Where(sq.Eq{"r.metric_path": metricPaths})
	}
	if !startTime.IsZero() {
		b = b.Where(sq.GtOrEq{"r.time": startTime})
	}
	if !endTime.IsZero() {
		b = b.Where(sq.Lt{"r.time": endTime})
	}
	if started {
		b = b.Where(sq.Expr(`("r"."time", r.id) > (?, ?)`, curTime, curID))
	}
	b = b.OrderBy(`"r"."time" ASC`, "r.id ASC").Limit(uint64(limit))
	q, args, _ := b.ToSql()
	return q, args
}

// buildDistinctMetricsSQL 发现 dashboard 源的横表指标列集：DISTINCT(metric_path, metric_type)。
// 指标的编号/类型与设备/小区无关，故只按 metric_paths（非空时）+ 时窗收口即得列全集（含 counter/kpi 类型）。
func buildDistinctMetricsSQL(table string, metricPaths []string, start, end time.Time) (string, []any) {
	b := newRawAwareExportSelect(
		table, aggregator.QueryRequest{MetricPaths: metricPaths},
		"DISTINCT metric_path", "metric_type",
	)
	if len(metricPaths) > 0 {
		b = b.Where(sq.Eq{"metric_path": metricPaths})
	}
	if !start.IsZero() {
		b = b.Where(sq.GtOrEq{"time": start})
	}
	if !end.IsZero() {
		b = b.Where(sq.Lt{"time": end})
	}
	q, args, _ := b.ToSql()
	return q, args
}

// buildAdhocDistinctMetricsSQL 发现 adhoc 源的横表指标列集：按 task_id（+ 可选时窗）DISTINCT(metric_path, metric_type)。
func buildAdhocDistinctMetricsSQL(taskID uuid.UUID, metricPaths []string, start, end time.Time) (string, []any) {
	b := storage.Psql.Select("DISTINCT metric_path", "metric_type").
		From("pm_adhoc_aggregation_results").
		Where(sq.Eq{"task_id": taskID})
	if len(metricPaths) > 0 {
		b = b.Where(sq.Eq{"metric_path": metricPaths})
	}
	if !start.IsZero() {
		b = b.Where(sq.GtOrEq{"time": start})
	}
	if !end.IsZero() {
		b = b.Where(sq.Lt{"time": end})
	}
	q, args, _ := b.ToSql()
	return q, args
}

func buildAdhocResultMetricScopeSQL(taskID uuid.UUID, startTime, endTime time.Time) (string, []any, error) {
	b := storage.Psql.Select("DISTINCT metric_path").
		From("pm_adhoc_aggregation_results").
		Where(sq.Eq{"task_id": taskID})
	if !startTime.IsZero() {
		b = b.Where(sq.GtOrEq{"time": startTime})
	}
	if !endTime.IsZero() {
		b = b.Where(sq.Lt{"time": endTime})
	}
	return b.OrderBy("metric_path").ToSql()
}

// applyDeviceExportFilters 复刻 aggregator 的 device 维度过滤（成对 OUI/SN + 公共过滤），
// 外加 A1 小区/PLMN 下钻白名单 objectLDNs（QueryRequest 无此字段，单独传入）。
// 与 aggregator.applyDeviceFilters 同语义，独立实现以避免改动其签名（scope 要求只读复用）。
func applyDeviceExportFilters(b sq.SelectBuilder, req aggregator.QueryRequest, objectLDNs []string) sq.SelectBuilder {
	if len(req.DeviceOUIs) > 0 && len(req.DeviceSNs) > 0 {
		n := len(req.DeviceOUIs)
		if len(req.DeviceSNs) < n {
			n = len(req.DeviceSNs)
		}
		or := sq.Or{}
		for i := 0; i < n; i++ {
			or = append(or, sq.And{
				sq.Eq{"device_oui": req.DeviceOUIs[i]},
				sq.Eq{"device_sn": req.DeviceSNs[i]},
			})
		}
		b = b.Where(or)
	} else if len(req.DeviceOUIs) > 0 {
		b = b.Where(sq.Eq{"device_oui": req.DeviceOUIs})
	} else if len(req.DeviceSNs) > 0 {
		b = b.Where(sq.Eq{"device_sn": req.DeviceSNs})
	}
	if len(objectLDNs) > 0 {
		// A1 小区/PLMN 下钻：只取白名单命中的 object_ldn 行（sq.Eq 切片 → IN(...)，等价 = ANY）。
		b = b.Where(sq.Eq{"object_ldn": objectLDNs})
	}
	if len(req.MetricPaths) > 0 {
		if req.MetricType != nil {
			b = b.Where(sq.Eq{"metric_path": req.MetricPaths})
		} else {
			or := sq.Or{}
			for _, raw := range req.MetricPaths {
				path := strings.TrimSpace(raw)
				if path == "" {
					continue
				}
				or = append(or, sq.And{
					sq.Eq{"metric_path": path},
					sq.Eq{"metric_type": string(metricTypeFromPath(path))},
				})
			}
			if len(or) > 0 {
				b = b.Where(or)
			}
		}
	}
	if req.MetricType != nil {
		b = b.Where(sq.Eq{"metric_type": string(*req.MetricType)})
	}
	if req.Granularity != "" {
		b = b.Where(sq.Eq{"granularity": string(req.Granularity)})
	}
	if !req.StartTime.IsZero() {
		b = b.Where(sq.GtOrEq{"time": req.StartTime})
	}
	if !req.EndTime.IsZero() {
		b = b.Where(sq.Lt{"time": req.EndTime})
	}
	if len(req.Technologies) > 0 {
		// buildDeviceKeysetSQL 跑在 metricDB=TsPool（device 维度直查 pm_metrics/pm_metrics_hourly），
		// devices 制式子查询改读本库影子表 device_dim（跨库分离）。
		if len(req.DeviceSNs) > 0 && len(req.DeviceOUIs) == 0 {
			b = b.Where(
				"(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE serial_number = ANY(?) AND technology = ANY(?))",
				req.DeviceSNs,
				req.Technologies,
			)
		} else {
			b = b.Where(
				"(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE technology = ANY(?))",
				req.Technologies,
			)
		}
	}
	return b
}
