package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Dimension 标识查询维度（device / device_group / product / band）。
type Dimension string

const (
	DimensionDevice         Dimension = "device"
	DimensionDeviceGroup    Dimension = "device_group"
	DimensionAggregateGroup Dimension = "aggregate_group" // adhoc 临时组：N 个 SN 现场聚合成一条
	DimensionProduct        Dimension = "product"         // T-0182：按设备所属产品 (devices.product_id) 现场聚合
	DimensionBand           Dimension = "band"            // T-0182：仅入枚举，聚合实现见 T-0183
	DimensionNetwork        Dimension = "network"         // T-0184：全网，现场汇总成一条总线（仅制式过滤，无实体键）
)

// QueryRequest 是 Aggregator.Query 的输入。
//
// Granularity 必填，路由表名；Dimension 默认 device。
// DeviceOUIs/DeviceSNs 是 device 维度过滤；DeviceGroupIDs 是 group 维度过滤；
// 不能同时设。
type QueryRequest struct {
	Granularity    metrics.Granularity
	Dimension      Dimension
	DeviceOUIs     []string
	DeviceSNs      []string
	DeviceGroupIDs []uuid.UUID
	ProductIDs     []uuid.UUID // product 维度过滤（空 = 不限产品，按全部 product_id 分组）
	MetricPaths    []string
	MetricType     *metrics.MetricType
	// Technologies 是制式过滤（lte/nr/gsm，小写对齐 devices.technology）。
	// 非空时所有维度（device/product 等需 JOIN devices 的路径）只取该制式的设备。
	Technologies []string
	StartTime    time.Time
	EndTime      time.Time
	Limit        int
	Offset       int
}

// Row 是 Aggregator.Query 的输出行。device 维度填 DeviceOUI/DeviceSN/ObjectLDN；
// device_group 维度填 DeviceGroupID。其余字段两维度共用。
// JSON tag 统一 snake_case 对齐其它 REST 响应；前端 mapper 走 snake → camel。
//
// Filled=true 表示该行是 handler 的 fill_empty 补齐占位（DB 实际无样本），MetricValue
// 字段被忽略；前端 mapper 见 filled=true 时把 metricValue 设为 null 以渲染"-"。
type Row struct {
	DeviceOUI     string              `json:"device_oui,omitempty"`
	DeviceSN      string              `json:"device_sn,omitempty"`
	DeviceGroupID uuid.UUID           `json:"device_group_id,omitempty"`
	ProductID     uuid.UUID           `json:"product_id,omitempty"` // product 维度填该产品 id
	MetricPath    string              `json:"metric_path"`
	// DisplayName 是给前端展示的友好名：KPI 行按 metric_path(=K 编号)回填指标库 cn_name；
	// counter 行 = metric_path 本身。前端列头/系列名用它，避免露出 K 编号。
	DisplayName string             `json:"display_name,omitempty"`
	MetricType  metrics.MetricType `json:"metric_type"`
	MetricValue   float64             `json:"metric_value"`
	StatisType    *metrics.StatisType `json:"statis_type,omitempty"`
	Granularity   metrics.Granularity `json:"granularity"`
	Time          time.Time           `json:"time"`
	StartTime     time.Time           `json:"start_time"`
	EndTime       time.Time           `json:"end_time"`
	IngestTime    time.Time           `json:"ingest_time"`
	ObjectLDN     *string             `json:"object_ldn,omitempty"`
	Extra         map[string]any      `json:"extra,omitempty"`
	Filled        bool                `json:"filled,omitempty"`
}

// Query 根据 (Granularity, Dimension) 路由到对应聚合表查询。
//
// 路由表：
//
//   - 15min × device       → pm_metrics
//   - 15min × device_group → 不支持（无 15min 级 group 聚合）
//   - hourly × device      → pm_metrics_hourly
//   - hourly × device_group→ pm_group_metrics_hourly
//   - daily × ...          → pm_metrics_daily / pm_group_metrics_daily
//   - weekly × ...         → pm_metrics_weekly / pm_group_metrics_weekly
//   - monthly × ...        → pm_metrics_monthly / pm_group_metrics_monthly
func (a *Aggregator) Query(ctx context.Context, q QueryRequest) ([]Row, error) {
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	table, err := SelectTable(q.Granularity, q.Dimension)
	if err != nil {
		return nil, err
	}

	var rows []Row
	switch q.Dimension {
	case DimensionDeviceGroup:
		rows, err = a.queryGroupTable(ctx, table, q)
	case DimensionAggregateGroup:
		rows, err = a.queryAggregateGroupTable(ctx, table, q)
	case DimensionProduct:
		rows, err = a.queryProductTable(ctx, table, q)
	case DimensionBand:
		rows, err = a.queryBandTable(ctx, table, q)
	case DimensionNetwork:
		rows, err = a.queryNetworkTable(ctx, table, q)
	default:
		rows, err = a.queryDeviceTable(ctx, table, q)
	}
	if err != nil {
		return nil, err
	}
	a.backfillDisplayNames(ctx, rows)
	return rows, nil
}

// backfillDisplayNames 给结果行补 DisplayName：
//   - counter 行：DisplayName = metric_path（本身就是可读名）
//   - kpi 行：metric_path 是 K 编号，按编号批量查指标库 cn_name 回填
//
// 编号在 perf_indicators_{enb,gnb,gsm} 三表全局唯一（无跨表重叠），故一次 UNION 查询
// 即可覆盖，无需按设备类型分别解析。查不到的编号回退用编号本身，保证不空白。
func (a *Aggregator) backfillDisplayNames(ctx context.Context, rows []Row) {
	codeSet := make(map[string]struct{})
	for i := range rows {
		if rows[i].MetricType == metrics.MetricTypeKPI {
			if rows[i].MetricPath != "" {
				codeSet[rows[i].MetricPath] = struct{}{}
			}
		} else {
			rows[i].DisplayName = rows[i].MetricPath
		}
	}
	if len(codeSet) == 0 {
		return
	}
	codes := make([]string, 0, len(codeSet))
	for c := range codeSet {
		codes = append(codes, c)
	}
	nameByCode := a.lookupIndicatorNames(ctx, codes)
	for i := range rows {
		if rows[i].MetricType != metrics.MetricTypeKPI {
			continue
		}
		if name, ok := nameByCode[rows[i].MetricPath]; ok && name != "" {
			rows[i].DisplayName = name
		} else {
			rows[i].DisplayName = rows[i].MetricPath // 回退：编号本身
		}
	}
}

// lookupIndicatorNames 按编号集合一次性查三张指标表，返回 code → cn_name（缺则 en_name）。
func (a *Aggregator) lookupIndicatorNames(ctx context.Context, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	const tmpl = `
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_enb  WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_gnb  WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_gsm  WHERE id = ANY($1)`
	rows, err := a.db.Query(ctx, tmpl, codes)
	if err != nil {
		a.logger.Warn("backfill display names query failed; fall back to codes", zap.Error(err))
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			a.logger.Warn("backfill display names scan failed", zap.Error(err))
			return out
		}
		out[id] = name
	}
	return out
}

// ErrPctNotSupportedInAggregateGroup 表示 aggregate_group 维度不支持 KPI 类指标
// （statis_type=pct 需公式重算，本阶段未实现）。
var ErrPctNotSupportedInAggregateGroup = fmt.Errorf(
	"aggregate_group 维度暂不支持 KPI 类 (statis_type=pct)，请改用 counter 类指标 (sum/avg/max/min)",
)

// ErrPctNotSupportedInBand 表示 band 维度不支持 KPI 类指标（statis_type=pct 需公式
// 重算，与 aggregate_group / product 维度一致，本阶段不实现）。
var ErrPctNotSupportedInBand = fmt.Errorf(
	"band 维度暂不支持 KPI 类 (statis_type=pct)，请改用 counter 类指标 (sum/avg/max/min)",
)

// ErrPctNotSupportedInNetwork 表示 network（全网）维度不支持 KPI 类指标（statis_type=pct
// 需公式重算，跨全网求和无意义，与 aggregate_group / product / band 维度一致）。
var ErrPctNotSupportedInNetwork = fmt.Errorf(
	"network 维度暂不支持 KPI 类 (statis_type=pct)，请改用 counter 类指标 (sum/avg/max/min)",
)

// queryAggregateGroupTable 现场聚合：从 device 维度表 (pm_metrics / pm_metrics_hourly / 等)
// 按 metric_path + granularity + time + object_ldn GROUP BY，不 GROUP BY device_sn。
// 算子按 statis_type 路由（sum/avg/max/min；pct 报错）。
//
// 输出 Row.DeviceSN = "AGGREGATED"，DeviceOUI = ""；MetricValue = 跨设备算子结果。
//
// 与 G5 cron 路径无关：直接查 device 表的已聚合（hourly/daily/...）数据再做 GROUP BY 折叠。
// 15min 也支持（pm_metrics raw 表本身就是 15min 粒度，GROUP BY 同样规则）。
func (a *Aggregator) queryAggregateGroupTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// 预检查：扫描目标 metric 的 statis_type，若有 pct 立即报错。
	if err := a.precheckStatisType(ctx, table, q, ErrPctNotSupportedInAggregateGroup); err != nil {
		return nil, err
	}

	// 算子按 statis_type 选择。CASE 在 SQL 内做路由，避免多次查询。
	// 各 statis_type 同一 metric_path 应该一致（来自指标定义），MIN(statis_type) 取代表值。
	const aggValueExpr = `
		CASE MIN(statis_type)
			WHEN 'sum' THEN SUM(metric_value)
			WHEN 'avg' THEN AVG(metric_value)
			WHEN 'max' THEN MAX(metric_value)
			WHEN 'min' THEN MIN(metric_value)
			ELSE SUM(metric_value)
		END`

	qb := storage.Psql.Select(
		"metric_path",
		"MIN(metric_type) AS metric_type",
		aggValueExpr+" AS metric_value",
		"MIN(statis_type) AS statis_type",
		"granularity",
		"time",
		"MIN(start_time) AS start_time",
		"MIN(end_time) AS end_time",
		"MAX(ingest_time) AS ingest_time",
		"object_ldn",
	).From(table)
	qb = applyDeviceFilters(qb, q)
	qb = qb.GroupBy("metric_path", "granularity", "time", "object_ldn")
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s (aggregate_group): %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s (aggregate_group): %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis, ldn *string
		var metricType, granularity string
		if err := rows.Scan(
			&r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime, &ldn,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s (aggregate_group): %w", table, err)
		}
		r.DeviceSN = "AGGREGATED" // 聚合后无单设备身份
		r.DeviceOUI = ""
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		r.ObjectLDN = ldn
		out = append(out, r)
	}
	return out, rows.Err()
}

// precheckStatisType 扫描 target metric_path 的 statis_type，若有 pct 立即返回 pctErr。
// pctErr 由调用方传入（不同维度的不支持文案不同）。
func (a *Aggregator) precheckStatisType(ctx context.Context, table string, q QueryRequest, pctErr error) error {
	if len(q.MetricPaths) == 0 {
		return nil
	}
	qb := storage.Psql.Select("DISTINCT statis_type").From(table).
		Where(sq.Eq{"metric_path": q.MetricPaths})
	if !q.StartTime.IsZero() {
		qb = qb.Where(sq.GtOrEq{"time": q.StartTime})
	}
	if !q.EndTime.IsZero() {
		qb = qb.Where(sq.LtOrEq{"time": q.EndTime})
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("aggregator.precheckStatisType build: %w", err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("aggregator.precheckStatisType exec: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s *string
		if err := rows.Scan(&s); err != nil {
			return fmt.Errorf("aggregator.precheckStatisType scan: %w", err)
		}
		if s != nil && *s == "pct" {
			return pctErr
		}
	}
	return rows.Err()
}

func (a *Aggregator) queryDeviceTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	qb := storage.Psql.Select(
		"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn", "extra",
	).From(table)
	qb = applyDeviceFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s: %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s: %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis, ldn *string
		var metricType, granularity string
		var extraBytes []byte
		if err := rows.Scan(
			&r.DeviceOUI, &r.DeviceSN, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime, &ldn, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s: %w", table, err)
		}
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		r.ObjectLDN = ldn
		if len(extraBytes) > 0 {
			extra := make(map[string]any)
			if err := json.Unmarshal(extraBytes, &extra); err == nil {
				r.Extra = extra
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (a *Aggregator) queryGroupTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// group 表 schema：device_group_id + metric_path/type/value/statis_type/granularity/time/start/end/ingest + extra
	qb := storage.Psql.Select(
		"device_group_id", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "extra",
	).From(table)
	qb = applyGroupFilters(qb, q)
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s: %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s: %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis *string
		var metricType, granularity string
		var extraBytes []byte
		if err := rows.Scan(
			&r.DeviceGroupID, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime, &extraBytes,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s: %w", table, err)
		}
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		if len(extraBytes) > 0 {
			extra := make(map[string]any)
			if err := json.Unmarshal(extraBytes, &extra); err == nil {
				r.Extra = extra
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// queryProductTable 现场聚合：从 device 维度表 (pm_metrics / pm_metrics_hourly / 等)
// JOIN devices 按 d.product_id 分组，把同产品所有设备的 counter 行聚到一起。
//
// 照搬 device_group 模式：
//   - JOIN devices d ON d.oui = m.device_oui AND d.serial_number = m.device_sn
//   - GROUP BY d.product_id, metric_path, granularity, time, statis_type
//   - 算子按 statis_type 路由（sum/avg/max/min；pct 不支持，预检查报错）
//
// 只聚 counter 行（KPI 跨设备求和无意义，与 device_group 一致）。
// product_id 为 NULL 的设备（seed 假设备）被 JOIN 自动排除。
//
// 制式过滤通过 d.technology IN (...) 在 JOIN 后施加（Technologies 非空时）。
func (a *Aggregator) queryProductTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// 预检查：扫描目标 metric 的 statis_type，若有 pct 立即报错（与 aggregate_group 一致）。
	if err := a.precheckStatisType(ctx, table, q, ErrPctNotSupportedInAggregateGroup); err != nil {
		return nil, err
	}

	const aggValueExpr = `
		CASE MIN(m.statis_type)
			WHEN 'sum' THEN SUM(m.metric_value)
			WHEN 'avg' THEN AVG(m.metric_value)
			WHEN 'max' THEN MAX(m.metric_value)
			WHEN 'min' THEN MIN(m.metric_value)
			ELSE SUM(m.metric_value)
		END`

	args := []any{}
	pos := 1
	add := func(v any) string {
		args = append(args, v)
		p := fmt.Sprintf("$%d", pos)
		pos++
		return p
	}

	where := []string{"d.product_id IS NOT NULL"}
	if len(q.MetricPaths) > 0 {
		where = append(where, fmt.Sprintf("m.metric_path = ANY(%s)", add(q.MetricPaths)))
	}
	if q.MetricType != nil {
		where = append(where, fmt.Sprintf("m.metric_type = %s", add(string(*q.MetricType))))
	}
	if q.Granularity != "" {
		where = append(where, fmt.Sprintf("m.granularity = %s", add(string(q.Granularity))))
	}
	if !q.StartTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time >= %s", add(q.StartTime)))
	}
	if !q.EndTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time <= %s", add(q.EndTime)))
	}
	if len(q.ProductIDs) > 0 {
		where = append(where, fmt.Sprintf("d.product_id = ANY(%s)", add(q.ProductIDs)))
	}
	if len(q.Technologies) > 0 {
		where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
	}

	whereSQL := ""
	for i, w := range where {
		if i == 0 {
			whereSQL = "WHERE " + w
		} else {
			whereSQL += "\n  AND " + w
		}
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL += fmt.Sprintf("\nLIMIT %s", add(q.Limit))
	}
	if q.Offset > 0 {
		limitSQL += fmt.Sprintf("\nOFFSET %s", add(q.Offset))
	}

	sqlStr := fmt.Sprintf(`
SELECT
    d.product_id,
    m.metric_path,
    MIN(m.metric_type) AS metric_type,
    %s AS metric_value,
    MIN(m.statis_type) AS statis_type,
    m.granularity,
    m.time,
    MIN(m.start_time) AS start_time,
    MIN(m.end_time) AS end_time,
    MAX(m.ingest_time) AS ingest_time
FROM %s m
JOIN devices d
  ON d.oui = m.device_oui AND d.serial_number = m.device_sn
%s
GROUP BY d.product_id, m.metric_path, m.granularity, m.time
ORDER BY m.time DESC%s`,
		aggValueExpr, table, whereSQL, limitSQL)

	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s (product): %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis *string
		var metricType, granularity string
		if err := rows.Scan(
			&r.ProductID, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s (product): %w", table, err)
		}
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// queryNetworkTable 现场汇总「全网一条总线」：从 device 维度表 (pm_metrics / pm_metrics_hourly / 等)
// 只按 metric_path + granularity + time GROUP BY（**不带 object_ldn、不带任何实体键**），
// 把全网所有设备/小区的同指标同时间桶 counter 行汇总成一条。
//
// 与 product 维度的区别：去掉 d.product_id 分组与 JOIN devices（产品键不需要），
// 分组键只剩 metric_path/granularity/time —— 这是最简单的聚合维度。
//
// 制式过滤：device 维度表行无 technology 列，用 (device_oui, device_sn) 子查询 JOIN devices 收口
// （与 applyCommonFilters 同范式）。Technologies 非空时只汇总该制式设备的行。
//
// 只聚 counter 行（KPI 跨全网求和无意义，与 product/band/aggregate_group 一致，pct 预检查报错）。
// 输出 Row.DeviceSN="AGGREGATED"，DeviceOUI=""，无 ObjectLDN（全网无实体身份）。
func (a *Aggregator) queryNetworkTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// 预检查：扫描目标 metric 的 statis_type，若有 pct 立即报错（与其它聚合维度一致）。
	if err := a.precheckStatisType(ctx, table, q, ErrPctNotSupportedInNetwork); err != nil {
		return nil, err
	}

	// 算子按 statis_type 路由（与 aggregate_group 一致）。
	const aggValueExpr = `
		CASE MIN(statis_type)
			WHEN 'sum' THEN SUM(metric_value)
			WHEN 'avg' THEN AVG(metric_value)
			WHEN 'max' THEN MAX(metric_value)
			WHEN 'min' THEN MIN(metric_value)
			ELSE SUM(metric_value)
		END`

	qb := storage.Psql.Select(
		"metric_path",
		"MIN(metric_type) AS metric_type",
		aggValueExpr+" AS metric_value",
		"MIN(statis_type) AS statis_type",
		"granularity",
		"time",
		"MIN(start_time) AS start_time",
		"MIN(end_time) AS end_time",
		"MAX(ingest_time) AS ingest_time",
	).From(table)
	// 复用 device 维度公共过滤（含制式子查询收口），但不带任何设备/组实体过滤。
	qb = applyCommonFilters(qb, q)
	qb = qb.GroupBy("metric_path", "granularity", "time")
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query build %s (network): %w", table, err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s (network): %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var statis *string
		var metricType, granularity string
		if err := rows.Scan(
			&r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s (network): %w", table, err)
		}
		r.DeviceSN = "AGGREGATED" // 全网无单设备身份
		r.DeviceOUI = ""
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		// 无 ObjectLDN：全网汇总不带任何实体键。
		out = append(out, r)
	}
	return out, rows.Err()
}

// band 维度参数路径后缀（小区频段标识来自设备参数）。
//
//   - bandPathSuffixes：FreqBandIndicator(LTE) / FreqBandIndicatorNR(NR)，取值 = band 值
//   - cellIDPathSuffixes：CellIdentity(LTE)，取值 = 小区号（= PM object_ldn 里的 Cellid）
//
// 用后缀（LIKE '%suffix'）匹配而非全路径，因 {i} 实例号在路径中会被实例化为具体数字。
var (
	bandPathSuffixes = []string{
		".CellConfig.LTE.RAN.RF.FreqBandIndicator", // LTE
		".FreqBandIndicatorNR",                     // NR（MultiFrequencyBandListNRSIB.{i}.FreqBandIndicatorNR）
	}
	cellIDPathSuffixes = []string{
		".CellConfig.LTE.RAN.Common.CellIdentity", // LTE 小区标识
		// NR 小区标识路径本仓库暂无真机样本，待确认后补；当前 LTE 链路已闭环。
	}
)

// queryBandTable 现场聚合：按频段（band）把同频段所有小区的 PM counter 行聚到一起。
//
// band 不在 PM 数据里，来自设备参数：
//  1. 从 device_parameters 取每个小区（device_id + fap_instance）的
//     CellIdentity（= 小区号）与 FreqBandIndicator（= band）两行，配成「小区号→band」映射。
//  2. PM 行 object_ldn 形如 'Cellid=111172245,PLMN=46068'，正则抽出 Cellid（= 小区号）。
//  3. 用 (device_oui,device_sn)→devices.id 把 PM 行挂到设备，再用 小区号 JOIN 映射得 band。
//  4. 按 band GROUP BY，算子照搬 statis_type 路由（sum/avg/max/min；pct 预检查报错）。
//
// join 未命中兜底：小区无频段参数（映射里查不到该小区号）→ INNER JOIN 自动跳过该小区，
// 不产出任何 band 行（行为固定，单测覆盖）。
//
// 结果行 ObjectLDN = 'Band=<值>'（复用已有列，不加迁移）；DeviceSN='AGGREGATED'。
// 只聚 counter 行（KPI 跨小区求和无意义，与 product/aggregate_group 一致）。
func (a *Aggregator) queryBandTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// 预检查：扫描目标 metric 的 statis_type，若有 pct 立即报错（与其它聚合维度一致）。
	if err := a.precheckStatisType(ctx, table, q, ErrPctNotSupportedInBand); err != nil {
		return nil, err
	}

	const aggValueExpr = `
		CASE MIN(m.statis_type)
			WHEN 'sum' THEN SUM(m.metric_value)
			WHEN 'avg' THEN AVG(m.metric_value)
			WHEN 'max' THEN MAX(m.metric_value)
			WHEN 'min' THEN MIN(m.metric_value)
			ELSE SUM(m.metric_value)
		END`

	args := []any{}
	pos := 1
	add := func(v any) string {
		args = append(args, v)
		p := fmt.Sprintf("$%d", pos)
		pos++
		return p
	}

	// 小区→band 映射的路径过滤（LIKE '%suffix'）。
	bandLike := make([]string, 0, len(bandPathSuffixes))
	for _, s := range bandPathSuffixes {
		bandLike = append(bandLike, fmt.Sprintf("bp.parameter_path LIKE %s", add("%"+s)))
	}
	cellLike := make([]string, 0, len(cellIDPathSuffixes))
	for _, s := range cellIDPathSuffixes {
		cellLike = append(cellLike, fmt.Sprintf("cp.parameter_path LIKE %s", add("%"+s)))
	}

	// PM 行过滤条件（与 product 维度同构）。
	where := []string{"m.object_ldn IS NOT NULL"}
	if len(q.MetricPaths) > 0 {
		where = append(where, fmt.Sprintf("m.metric_path = ANY(%s)", add(q.MetricPaths)))
	}
	if q.MetricType != nil {
		where = append(where, fmt.Sprintf("m.metric_type = %s", add(string(*q.MetricType))))
	}
	if q.Granularity != "" {
		where = append(where, fmt.Sprintf("m.granularity = %s", add(string(q.Granularity))))
	}
	if !q.StartTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time >= %s", add(q.StartTime)))
	}
	if !q.EndTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time <= %s", add(q.EndTime)))
	}
	if len(q.Technologies) > 0 {
		where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
	}
	whereSQL := ""
	for i, w := range where {
		if i == 0 {
			whereSQL = "WHERE " + w
		} else {
			whereSQL += "\n  AND " + w
		}
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL += fmt.Sprintf("\nLIMIT %s", add(q.Limit))
	}
	if q.Offset > 0 {
		limitSQL += fmt.Sprintf("\nOFFSET %s", add(q.Offset))
	}

	// cell_band CTE：device_id + fap_instance 上把 CellIdentity 与 FreqBandIndicator 配对。
	sqlStr := fmt.Sprintf(`
WITH cell_band AS (
    SELECT
        cp.device_id,
        cp.parameter_value AS cell_id,
        bp.parameter_value AS band
    FROM device_parameters cp
    JOIN device_parameters bp
      ON bp.device_id = cp.device_id AND bp.fap_instance = cp.fap_instance
    WHERE (%s)
      AND (%s)
      AND cp.parameter_value IS NOT NULL
      AND bp.parameter_value IS NOT NULL
)
SELECT
    cb.band,
    m.metric_path,
    MIN(m.metric_type) AS metric_type,
    %s AS metric_value,
    MIN(m.statis_type) AS statis_type,
    m.granularity,
    m.time,
    MIN(m.start_time) AS start_time,
    MIN(m.end_time) AS end_time,
    MAX(m.ingest_time) AS ingest_time
FROM %s m
JOIN devices d
  ON d.oui = m.device_oui AND d.serial_number = m.device_sn
JOIN cell_band cb
  ON cb.device_id = d.id
 AND cb.cell_id = substring(m.object_ldn FROM 'Cellid=([0-9]+)')
%s
GROUP BY cb.band, m.metric_path, m.granularity, m.time
ORDER BY m.time DESC%s`,
		joinOr(cellLike), joinOr(bandLike), aggValueExpr, table, whereSQL, limitSQL)

	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query exec %s (band): %w", table, err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var band string
		var statis *string
		var metricType, granularity string
		if err := rows.Scan(
			&band, &r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan %s (band): %w", table, err)
		}
		r.DeviceSN = "AGGREGATED"
		r.DeviceOUI = ""
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		ldn := "Band=" + band
		r.ObjectLDN = &ldn
		out = append(out, r)
	}
	return out, rows.Err()
}

// joinOr 把多个 LIKE 条件用 OR 连接成 "(a OR b OR ...)" 的括号体（去掉外层括号，由调用方加）。
func joinOr(conds []string) string {
	out := ""
	for i, c := range conds {
		if i == 0 {
			out = c
		} else {
			out += " OR " + c
		}
	}
	return out
}

// ── 过滤条件 ───────────────────────────────────────────────────────────────

func applyDeviceFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0 {
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		or := sq.Or{}
		for i := 0; i < n; i++ {
			or = append(or, sq.And{
				sq.Eq{"device_oui": q.DeviceOUIs[i]},
				sq.Eq{"device_sn": q.DeviceSNs[i]},
			})
		}
		qb = qb.Where(or)
	} else if len(q.DeviceOUIs) > 0 {
		qb = qb.Where(sq.Eq{"device_oui": q.DeviceOUIs})
	} else if len(q.DeviceSNs) > 0 {
		qb = qb.Where(sq.Eq{"device_sn": q.DeviceSNs})
	}
	return applyCommonFilters(qb, q)
}

func applyGroupFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.DeviceGroupIDs) > 0 {
		qb = qb.Where(sq.Eq{"device_group_id": q.DeviceGroupIDs})
	}
	return applyCommonFilters(qb, q)
}

func applyCommonFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.MetricPaths) > 0 {
		qb = qb.Where(sq.Eq{"metric_path": q.MetricPaths})
	}
	if q.MetricType != nil {
		qb = qb.Where(sq.Eq{"metric_type": string(*q.MetricType)})
	}
	if q.Granularity != "" {
		qb = qb.Where(sq.Eq{"granularity": string(q.Granularity)})
	}
	if !q.StartTime.IsZero() {
		qb = qb.Where(sq.GtOrEq{"time": q.StartTime})
	}
	if !q.EndTime.IsZero() {
		qb = qb.Where(sq.LtOrEq{"time": q.EndTime})
	}
	// 制式过滤：限定到指定制式的设备（device 维度表行没有 technology 列，
	// 用 (device_oui, device_sn) 子查询 JOIN devices 收口，不改 SELECT 列形态）。
	if len(q.Technologies) > 0 {
		qb = qb.Where(
			"(device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY(?))",
			q.Technologies,
		)
	}
	return qb
}

// ── 表名路由（导出便于单测 + 其它包查表名） ────────────────────────────────

// ErrUnsupportedQuery 表示 (Granularity, Dimension) 组合不支持。
// 唯一不支持的组合：15min × device_group（无 15min 级 group 聚合源）。
var ErrUnsupportedQuery = fmt.Errorf("aggregator: unsupported (granularity, dimension) combination")

// SelectTable 把 (granularity, dim) 路由到具体表名。
//
// aggregate_group 维度走 device 维度的表 (pm_metrics / pm_metrics_hourly / 等)，
// 然后在查询层做 GROUP BY metric_path+granularity+time+object_ldn 折叠（不依赖 G5 group 表）。
func SelectTable(g metrics.Granularity, dim Dimension) (string, error) {
	if dim == "" {
		dim = DimensionDevice
	}
	switch g {
	case metrics.Granularity15Min:
		if dim == DimensionDeviceGroup {
			return "", fmt.Errorf("%w: 15min × device_group", ErrUnsupportedQuery)
		}
		// device / aggregate_group / product / band / network 都走 pm_metrics raw 表
		// （band 维度按小区行 JOIN device_parameters，源同 device 维度表；
		//   network 维度全网汇总，同样查 device 维度表再现场 GROUP BY 折叠）
		return "pm_metrics", nil
	case metrics.GranularityHourly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_hourly", nil
		}
		return "pm_metrics_hourly", nil
	case metrics.GranularityDaily:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_daily", nil
		}
		return "pm_metrics_daily", nil
	case metrics.GranularityWeekly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_weekly", nil
		}
		return "pm_metrics_weekly", nil
	case metrics.GranularityMonthly:
		if dim == DimensionDeviceGroup {
			return "pm_group_metrics_monthly", nil
		}
		return "pm_metrics_monthly", nil
	}
	return "", fmt.Errorf("aggregator: unknown granularity %q", g)
}
