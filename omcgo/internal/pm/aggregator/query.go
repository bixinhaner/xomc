package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/jsonx"
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
	// ObjectLDNs 是测量对象过滤（原始 object_ldn 串，如 "Cellid=111,PLMN=46068"）。
	// 非空时只返回匹配的行；空 = 不过滤（向后兼容）。
	ObjectLDNs []string
	StartTime  time.Time
	EndTime    time.Time
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管不过滤 / []=fail-closed 空集 /
	// [g...]=仅这些组）。device/aggregate_group/network/product/band 维度按 device_sn 收口
	// （authz.ApplyDeviceSNVisibilityFilter / VisibleSNSubquerySQL）；device_group 维度直接对
	// device_group_id 取交（authz.ApplyGroupVisibilityFilter）。handler 解析调用者身份后注入。
	VisibleGroups []uuid.UUID
	Limit         int
	Offset        int
	// Weekdays #599：星期过滤（0=周日..6=周六，对齐 PostgreSQL EXTRACT(dow)）。
	// 空/全选 = 不过滤。筛的是 start_time 的星期几。
	Weekdays []int
	// Hours #599：小时段过滤（0..23，对齐 PostgreSQL EXTRACT(hour)）。
	// 空/全选 = 不过滤。筛的是 start_time 的整点小时。
	Hours []int
	// RecomputeAllKPIs（KPI-ALL-IND）：全网/全聚任务放开到全库时置 true。聚合层在汇总
	// 全部 counter 的同时，额外从指标库加载全库「派生 KPI」按公式重算并产出 KPI 行，
	// 使「首页读现成全网预聚合表」时 KPI 也有线（否则全聚只产 counter 行、KPI 面板空线）。
	// 仅用于「空 MetricPaths＝全聚」的聚合任务执行路径；动态从指标库枚举，不在 seed 硬编码。
	RecomputeAllKPIs bool
	// StoreAllEnabled（#532 P2）：store_all_metrics=true 的落库侧全存模式。置 true 时聚合层
	// 不按请求里 MetricPaths 收窄，改按 Technologies 驱动「已启用指标集」
	// （enabled_pm_indicators_{enb,gnb,gsm}）：全部已启用 counter 随全量 counter 汇总产出，
	// 全部已启用派生 KPI 按公式重算产出，一并落库——使事后改任务指标集无需重算。
	// 与 RecomputeAllKPIs（全库枚举）的区别：枚举源限定到「已启用 ∩（counter/派生）」而非全库，
	// 体量可控。已启用集为空时降级（不丢 counter，见 queryEnabledWithKPIs）。
	// 仅经 queryWithKPIRecompute 的维度（product/band/device_group/aggregate_group/network）生效；
	// device 维度由执行器在请求侧把 MetricPaths 直接灌成已启用集（设备级 KPI 已算好，无需重算）。
	StoreAllEnabled bool
}

// Row 是 Aggregator.Query 的输出行。device 维度填 DeviceOUI/DeviceSN/ObjectLDN；
// device_group 维度填 DeviceGroupID。其余字段两维度共用。
// JSON tag 统一 snake_case 对齐其它 REST 响应；前端 mapper 走 snake → camel。
//
// Filled=true 表示该行是 handler 的 fill_empty 补齐占位（DB 实际无样本），MetricValue
// 字段被忽略；前端 mapper 见 filled=true 时把 metricValue 设为 null 以渲染"-"。
type Row struct {
	DeviceOUI     string    `json:"device_oui,omitempty"`
	DeviceSN      string    `json:"device_sn,omitempty"`
	DeviceGroupID uuid.UUID `json:"device_group_id,omitempty"`
	// Technology 是 device_group 维度的制式拆分键（lte/nr/gsm）。设备组快表按「组 × 制式」拆行，
	// queryGroupTable 带出该列；其它维度恒空。
	Technology string    `json:"technology,omitempty"`
	ProductID  uuid.UUID `json:"product_id,omitempty"` // product 维度填该产品 id
	MetricPath string    `json:"metric_path"`
	// DisplayName 是给前端展示的友好名：KPI 行按 metric_path(=K 编号)回填指标库 cn_name；
	// counter 行 = metric_path 本身。前端列头/系列名用它，避免露出 K 编号。
	DisplayName string             `json:"display_name,omitempty"`
	MetricType  metrics.MetricType `json:"metric_type"`
	// MetricValue 用 jsonx.Float（底层 float64）兜底非有限值（NaN/Inf → null），
	// 避免单个 NaN 行致整批 JSON 编码失败、返回空 body（issue #387）。
	// 「平均型/比率型」KPI 分母为 0 时合法地算出 NaN，是真实聚合数据普遍会踩的坑。
	MetricValue jsonx.Float         `json:"metric_value"`
	StatisType  *metrics.StatisType `json:"statis_type,omitempty"`
	Granularity metrics.Granularity `json:"granularity"`
	Time        time.Time           `json:"time"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     time.Time           `json:"end_time"`
	IngestTime  time.Time           `json:"ingest_time"`
	ObjectLDN   *string             `json:"object_ldn,omitempty"`
	Extra       map[string]any      `json:"extra,omitempty"`
	Filled      bool                `json:"filled,omitempty"`
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
		rows, err = a.queryWithKPIRecompute(ctx, table, q, a.queryGroupTable)
	case DimensionAggregateGroup:
		rows, err = a.queryWithKPIRecompute(ctx, table, q, a.queryAggregateGroupTable)
	case DimensionProduct:
		rows, err = a.queryWithKPIRecompute(ctx, table, q, a.queryProductTable)
	case DimensionBand:
		rows, err = a.queryWithKPIRecompute(ctx, table, q, a.queryBandTable)
	case DimensionNetwork:
		rows, err = a.queryWithKPIRecompute(ctx, table, q, a.queryNetworkTable)
	default:
		// device 维度直接读设备级聚合表（counter / KPI 行都是设备自己算好的，无需跨维重算）。
		// #532 P2 store-all-by-enabled：device 维度不走重算 wrapper，故在入口把已启用集直接
		// 灌成 MetricPaths（既存 counter 行 + 设备级已算好的 KPI 行都按已启用集筛取落库）；
		// 已启用集为空时降级为不下推过滤（取设备表全部，不丢行）。
		if q.StoreAllEnabled && len(q.MetricPaths) == 0 {
			if enabled := a.resolveEnabledIndicators(ctx, q.Technologies); len(enabled) > 0 {
				q.MetricPaths = enabled
			}
		}
		rows, err = a.queryDeviceTable(ctx, table, q)
	}
	if err != nil {
		return nil, err
	}
	a.backfillDisplayNames(ctx, rows)
	return rows, nil
}

// Count 返回与 Query 同过滤条件下命中行的真实总数（忽略 Limit / Offset），
// 用于截断诚实提示（T-0194 C2）：handler 拿它和实际返回行数比，命中 limit 时前端提示「已截断」。
//
// 各维度的「行」口径与 Query 一致：
//   - device：直接表行（无聚合），COUNT(*)。
//   - device_group / aggregate_group / product / band / network：现场 GROUP BY 后的分组数，
//     故 COUNT(*) FROM (<同 Query 的 GROUP BY 子查询，去 ORDER BY/LIMIT/OFFSET>) sub。
//
// KPI recompute 不改变分组身份（只把同 (object,time) 分组的 counter 行重算成 KPI 行），
// 故总数以「存储层命中分组数」为准，是诚实的「DB 命中多少」。
func (a *Aggregator) Count(ctx context.Context, q QueryRequest) (int, error) {
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	table, err := SelectTable(q.Granularity, q.Dimension)
	if err != nil {
		return 0, err
	}
	// Count 不分页：清掉 Limit/Offset，避免被带进子查询。
	q.Limit = 0
	q.Offset = 0

	switch q.Dimension {
	case DimensionDevice:
		qb := storage.Psql.Select("COUNT(*)").From(table)
		qb = applyDeviceFilters(qb, q)
		return a.scanCount(ctx, qb)
	case DimensionDeviceGroup:
		inner := storage.Psql.Select("1").From(table)
		inner = applyGroupFilters(inner, q)
		// 行粒度按「组 × 制式 × 指标 × 桶」，GroupBy 必须含 technology，否则截断计数偏小。
		inner = inner.GroupBy("device_group_id", "technology", "metric_path", "granularity", "time")
		return a.scanCountSub(ctx, inner)
	case DimensionAggregateGroup:
		inner := storage.Psql.Select("1").From(table)
		inner = applyDeviceFilters(inner, q)
		// 单条聚合：分组键不含 object_ldn（与 queryAggregateGroupTable 一致），保证截断计数口径相符。
		inner = inner.GroupBy("metric_path", "granularity", "time")
		return a.scanCountSub(ctx, inner)
	case DimensionNetwork:
		inner := storage.Psql.Select("1").From(table)
		inner = applyCommonFilters(inner, q)
		inner = inner.GroupBy("metric_path", "granularity", "time")
		return a.scanCountSub(ctx, inner)
	default:
		// product / band：内部 SQL 是手拼字符串（带 JOIN / CTE），无 squirrel builder 可复用。
		// 复用各自的 Query 取数再数行数——对窄查询（前端按单设备 1:1 拆分）开销可接受，
		// 且这些维度本身分组后行数远小于 device 维度，不构成 100K 规模瓶颈。
		rows, err := a.Query(ctx, q)
		if err != nil {
			return 0, err
		}
		return len(rows), nil
	}
}

// DiscoverObjectLDNs 返回同设备、同时间窗下实际出现过的 object_ldn 列表。
// 用于 "全部小区" 查询补骨架：请求未显式传 object_ldns 时，后端从同一粒度表发现展示全集。
// 指标过滤在这里刻意清空，否则当前指标完全无数据时无法发现 object 集合。
func (a *Aggregator) DiscoverObjectLDNs(ctx context.Context, q QueryRequest) ([]string, error) {
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	if !CanAutoDiscoverObjectSkeletonRequest(q) {
		return nil, nil
	}
	table, err := SelectTable(q.Granularity, q.Dimension)
	if err != nil {
		return nil, err
	}
	discoverReq := q
	discoverReq.MetricPaths = nil
	discoverReq.MetricType = nil
	discoverReq.ObjectLDNs = nil
	discoverReq.Limit = 0
	discoverReq.Offset = 0

	qb := storage.Psql.Select("DISTINCT object_ldn").
		From(table).
		Where("object_ldn <> ''").
		OrderBy("object_ldn")
	qb = applyDeviceFilters(qb, discoverReq)
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.DiscoverObjectLDNs build: %w", err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.DiscoverObjectLDNs exec: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var ldn string
		if err := rows.Scan(&ldn); err != nil {
			return nil, fmt.Errorf("aggregator.DiscoverObjectLDNs scan: %w", err)
		}
		out = append(out, ldn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("aggregator.DiscoverObjectLDNs rows: %w", err)
	}
	return out, nil
}

func (a *Aggregator) scanCount(ctx context.Context, qb sq.SelectBuilder) (int, error) {
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("aggregator.Count build: %w", err)
	}
	var n int
	if err := a.db.QueryRow(ctx, sqlStr, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("aggregator.Count exec: %w", err)
	}
	return n, nil
}

func (a *Aggregator) scanCountSub(ctx context.Context, inner sq.SelectBuilder) (int, error) {
	innerSQL, args, err := inner.ToSql()
	if err != nil {
		return 0, fmt.Errorf("aggregator.Count build sub: %w", err)
	}
	var n int
	q := "SELECT COUNT(*) FROM (" + innerSQL + ") sub"
	if err := a.db.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("aggregator.Count exec sub: %w", err)
	}
	return n, nil
}

// BackfillDisplayNames 对外暴露：对给定行集合整体回填 DisplayName（按 metric_path 查指标库，
// 查不到回退编号本身）。供 pm handler 在补占位行后统一回填——占位行可能因「该指标本次无任何
// 真实行」而取不到名，整体再回填一次让占位行与真实行同口径取名（合成计数器等库里无对应行的
// 编号仍回退编号本身，行为不变）。
func (a *Aggregator) BackfillDisplayNames(ctx context.Context, rows []Row) {
	a.backfillDisplayNames(ctx, rows)
}

// backfillDisplayNames 给结果行补 DisplayName：
//   - kpi 行：metric_path 是 K 编号，按编号批量查指标库 cn_name 回填
//   - counter 行：PM-P2/P3 编号化后 metric_path 是 C 编号，同样按编号查指标库本地化名回填；
//     查不到（如历史遗留的标准名 counter）则回退用 metric_path 本身，保证不空白、不改旧行为。
//
// 编号在 perf_indicators_{enb,gnb,gsm} 三表全局唯一（无跨表重叠），故一次 UNION 查询
// 即可覆盖，无需按设备类型分别解析。
func (a *Aggregator) backfillDisplayNames(ctx context.Context, rows []Row) {
	codeSet := make(map[string]struct{})
	for i := range rows {
		if rows[i].MetricPath != "" {
			codeSet[rows[i].MetricPath] = struct{}{}
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
		if name, ok := nameByCode[rows[i].MetricPath]; ok && name != "" {
			rows[i].DisplayName = name
		} else {
			rows[i].DisplayName = rows[i].MetricPath // 回退：编号 / 标准名本身
		}
	}
}

// lookupIndicatorNames 按编号集合一次性查三张指标表，返回 code → 本地化显示名。
// 取名方向按 ctx 中的 locale 决定（中文 cn_name 优先 / 英文 en_name 优先，空则回退另一种），
// 与 adhoc 结果回填层共用同一取名口径（metrics.IndicatorDisplayNameExpr）。
func (a *Aggregator) lookupIndicatorNames(ctx context.Context, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	nameExpr := metrics.IndicatorDisplayNameExpr(appcontext.GetLocale(ctx))
	tmpl := fmt.Sprintf(`
SELECT id, %[1]s AS display_name FROM perf_indicators_enb  WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gnb  WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gsm  WHERE id = ANY($1)`, nameExpr)
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

// queryAggregateGroupTable 现场聚合：从 device 维度表 (pm_metrics / pm_metrics_hourly / 等)
// 按 metric_path + granularity + time GROUP BY（**不含 object_ldn**），不 GROUP BY device_sn。
// 算子按 statis_type 路由（sum/avg/max/min；pct 报错）。
//
// 临时聚合组 = N 台设备当一个逻辑对象出一条合计线：所有设备、所有小区的同指标同时间桶
// 合成一条（每个 metric_path × granularity × time 一行），不再按小区拆分。
//
// 输出 Row.DeviceSN = "AGGREGATED"，DeviceOUI = ""，ObjectLDN = nil（单条聚合，无小区）；
// MetricValue = 跨设备跨小区算子结果。
//
// 与 G5 cron 路径无关：直接查 device 表的已聚合（hourly/daily/...）数据再做 GROUP BY 折叠。
// 15min 也支持（pm_metrics raw 表本身就是 15min 粒度，GROUP BY 同样规则）。
func (a *Aggregator) queryAggregateGroupTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	// T-0191：pct/KPI 类不再在此报错；Query 入口已通过 queryWithKPIRecompute 把派生 KPI 拆成
	// counter deps，本函数只聚 counter 行（metric_type='counter' 由 wrapper 强制）。

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
	).From(table)
	qb = applyDeviceFilters(qb, q)
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
		var statis *string
		var metricType, granularity string
		if err := rows.Scan(
			&r.MetricPath, &metricType, &r.MetricValue,
			&statis, &granularity, &r.Time, &r.StartTime, &r.EndTime, &r.IngestTime,
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
		// r.ObjectLDN 保持 nil：单条聚合，无小区
		out = append(out, r)
	}
	return out, rows.Err()
}

// deviceTableColumns 是 device 维度直读 pm_metrics 的列集（内外层 SELECT 共用，避免漂移）。
var deviceTableColumns = []string{
	"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn", "extra",
}

// buildDeviceTableSQL 构造 device 维度直读 pm_metrics 的去重查询（纯函数，便于单测）。
//
// 同窗口多文件去重（#208 数值偏差）：删 uq_pm_metrics_natural（#256）后，同设备同 15min 窗口但
// 文件名不同的两个 PM 文件（真机补传换名 / 厂商按 measInfo 拆文件 / 重复上报）会各自落一行，
// device 维度直读不去重时被前端 SUM/重复渲染成翻倍值。这里在 device 维度 SELECT 用 DISTINCT ON
// 折叠同键行、保留 ingest_time 最新一条（保留最后入库文件，与前端 kpiSeries last-wins 一致）。
// DISTINCT ON 要求 ORDER BY 前缀与去重键一致，故内层按去重键 + ingest_time DESC 排序，外层再包
// 一层恢复原有「time DESC + Limit/Offset」语义。WHERE/参数绑定（applyDeviceFilters）全部留在内层、
// 保持不变。
func buildDeviceTableSQL(table string, q QueryRequest) (string, []any, error) {
	inner := storage.Psql.Select(deviceTableColumns...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`).
		From(table)
	inner = applyDeviceFilters(inner, q)
	inner = inner.OrderBy(
		"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
	)

	qb := storage.Psql.Select(deviceTableColumns...).FromSelect(inner, "d")
	qb = qb.OrderBy("time DESC")
	if q.Limit > 0 {
		qb = qb.Limit(uint64(q.Limit))
	}
	if q.Offset > 0 {
		qb = qb.Offset(uint64(q.Offset))
	}
	return qb.ToSql()
}

func (a *Aggregator) queryDeviceTable(ctx context.Context, table string, q QueryRequest) ([]Row, error) {
	sqlStr, args, err := buildDeviceTableSQL(table, q)
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
	// group 表 schema：device_group_id + technology + metric_path/type/value/statis_type/granularity/time/start/end/ingest + extra
	// （设备组制式治本 B 方案：快表按「组 × 制式」拆行，带出 technology 列。）
	qb := storage.Psql.Select(
		"device_group_id", "technology", "metric_path", "metric_type", "metric_value",
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
			&r.DeviceGroupID, &r.Technology, &r.MetricPath, &metricType, &r.MetricValue,
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
	// T-0191：pct/KPI 类由 Query 入口的 queryWithKPIRecompute 拆成 counter deps 后重算，本函数只聚 counter。

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
		where = append(where, fmt.Sprintf("m.time < %s", add(q.EndTime)))
	}
	if len(q.ProductIDs) > 0 {
		where = append(where, fmt.Sprintf("d.product_id = ANY(%s)", add(q.ProductIDs)))
	}
	if len(q.Technologies) > 0 {
		where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
	}
	// #64 设备组数据权限：JOIN devices 后按 m.device_sn 收口到可见分组（fail-closed）。
	where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)

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
JOIN device_dim d
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
	// T-0191：pct/KPI 类由 Query 入口的 queryWithKPIRecompute 拆成 counter deps 后重算，本函数只聚 counter。

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
	)
	// 15min raw 表可能存在同设备同对象同窗口重复上报；network 汇总前按 device 直读口径
	// 保留最新 ingest 行，避免首页尾部补点把重复 raw 行计入全网 counter。
	if table == "pm_metrics" {
		inner := storage.Psql.Select(
			"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
			"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn",
		).
			Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`).
			From(table)
		inner = applyCommonFilters(inner, q)
		inner = inner.OrderBy(
			"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
		)
		qb = qb.FromSelect(inner, "m")
	} else {
		qb = qb.From(table)
		// 复用 device 维度公共过滤（含制式子查询收口），但不带任何设备/组实体过滤。
		qb = applyCommonFilters(qb, q)
	}
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
	// T-0191：pct/KPI 类由 Query 入口的 queryWithKPIRecompute 拆成 counter deps 后重算，本函数只聚 counter。

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

	// 小区→band 映射改读 cell_band_dim（band 已预派生），无需再按 device_parameters 参数路径过滤；
	// bandPathSuffixes/cellIDPathSuffixes 仅作为同步任务派生 cell_band_dim 的语义参考（见 worker tsdbsync）。

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
		where = append(where, fmt.Sprintf("m.time < %s", add(q.EndTime)))
	}
	if len(q.Technologies) > 0 {
		where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
	}
	// #64 设备组数据权限：JOIN devices 后按 m.device_sn 收口到可见分组（fail-closed）。
	where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)
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

	// 小区→band 映射改读时序库影子表 cell_band_dim（device_id, cell_id, band），
	// 由 worker 同步任务从主库 device_parameters 的 CellIdentity/FreqBandIndicator 配对派生后刷入。
	// 原先此处用 device_parameters 现拼 CTE，跨库分离后改读本库影子表，SQL 形状收敛为直接 JOIN。
	sqlStr := fmt.Sprintf(`
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
JOIN device_dim d
  ON d.oui = m.device_oui AND d.serial_number = m.device_sn
JOIN cell_band_dim cb
  ON cb.device_id = d.id
 AND cb.cell_id = COALESCE(
     substring(m.object_ldn FROM 'Cellid=([0-9]+)'),
     substring(m.object_ldn FROM 'NrCGI=([0-9]+)'),
     substring(m.object_ldn FROM 'Uid=([^,]+)')
 )
%s
GROUP BY cb.band, m.metric_path, m.granularity, m.time
ORDER BY m.time DESC%s`,
		aggValueExpr, table, whereSQL, limitSQL)

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

// appendVisibleSNWhere 把 #64 设备组可见性的 device_sn 收口条件追加到手拼 SQL 的 where 切片里
// （product / band 维度走带 CTE/JOIN 的字符串 SQL，无 squirrel builder）。add 是各自闭包：
// 追加一个位置参数并返回其 $N 占位符。三态：
//
//	nil       → 不追加（超管不过滤）
//	[]        → 追加 "FALSE"（fail-closed，无参数）
//	[g...]    → 追加 "m.device_sn IN (... = ANY($N))" 并 add(groups) 注册参数
func appendVisibleSNWhere(where []string, snColumn string, visibleGroups []uuid.UUID, add func(any) string) []string {
	if visibleGroups == nil {
		return where
	}
	if len(visibleGroups) == 0 {
		return append(where, "FALSE")
	}
	ref := add(visibleGroups)
	sql, _ := authz.VisibleSNSubquerySQL(snColumn, ref, visibleGroups)
	return append(where, sql)
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

// applyGroupFilters 设备组维度过滤：组 id + 标量过滤 + 制式直接按列筛。
//
// 设备组制式治本（B 方案）：设备组快表自带 technology 列（每组每制式一行），制式过滤直接
// WHERE technology = ANY(?)，**不**走「按设备编号子查询 JOIN devices」那套——快表根本没有
// device_oui/device_sn 列，旧路径会引用不存在的列报错（本任务修复的根因）。
func applyGroupFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.DeviceGroupIDs) > 0 {
		qb = qb.Where(sq.Eq{"device_group_id": q.DeviceGroupIDs})
	}
	qb = applyScalarFilters(qb, q)
	if len(q.Technologies) > 0 {
		qb = qb.Where(sq.Eq{"technology": q.Technologies})
	}
	// #64 设备组数据权限：group 维度快表以 device_group_id 为键，直接对可见分组取交（fail-closed）。
	qb = authz.ApplyGroupVisibilityFilter(qb, "device_group_id", q.VisibleGroups)
	return qb
}

// applyScalarFilters 仅标量过滤（metric_path / metric_type / granularity / time 区间），
// 不含任何制式过滤——制式过滤按维度分流由调用方各自补（device/network 走设备编号子查询，
// device_group 走直接列筛）。
func applyScalarFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
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
		// PM 桶和聚合窗口统一使用半开区间 [start,end)：结束时刻若恰好等于
		// 下一桶桶头，不应把下一桶也纳入自定义时间范围（#29）。
		qb = qb.Where(sq.Lt{"time": q.EndTime})
	}
	// #599：星期/小时段后端过滤（全选/空 = 不加条件，向后兼容）。
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		qb = qb.Where("EXTRACT(dow FROM start_time)::int = ANY(?)", q.Weekdays)
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		qb = qb.Where("EXTRACT(hour FROM start_time)::int = ANY(?)", q.Hours)
	}
	// #619：测量对象（object_ldn）后端过滤（空 = 不过滤，向后兼容）。
	if len(q.ObjectLDNs) > 0 {
		qb = qb.Where(sq.Eq{"object_ldn": q.ObjectLDNs})
	}
	return qb
}

// applyCommonFilters 设备维度表（device / aggregate_group / network）公共过滤：标量过滤 +
// 制式过滤走「按设备编号子查询 JOIN devices」收口（这些表行无 technology 列）。
func applyCommonFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	qb = applyScalarFilters(qb, q)
	// 制式过滤：限定到指定制式的设备（device 维度表行没有 technology 列，
	// 用 (device_oui, device_sn) 子查询 JOIN devices 收口，不改 SELECT 列形态）。
	if len(q.Technologies) > 0 {
		qb = qb.Where(
			"(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE technology = ANY(?))",
			q.Technologies,
		)
	}
	// #64 设备组数据权限：device 维度表（device / aggregate_group / network）以 device_sn 为
	// 设备键，按可见分组 fail-closed 收口（nil 超管不过滤 / [] WHERE FALSE / [g...] 子查询限定）。
	qb = authz.ApplyDeviceSNVisibilityFilter(qb, "device_sn", q.VisibleGroups)
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
