package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
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

	pmMetricValueChunkPruneSegment = 4 * time.Hour
	pmMetricValueRawBucketWindow   = 15 * time.Minute
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
	// PageByPivotRow 让 device 维度的 Limit/Offset 作用在透视表行 key 上，而不是原始长表行上。
	// key = device_oui + device_sn + object_ldn + granularity + time；随后再取这些 key 下的全部指标行。
	PageByPivotRow bool
	// PivotRowKeys 是 page_by=pivot_row + fill_empty 时当前页的透视行边界。
	// 它只用于补骨架，避免 FillEmptyBuckets 在分页后重新扩成完整时间窗。
	PivotRowKeys []PivotRowKey
	// Weekdays #599：星期过滤（0=周日..6=周六，对齐 PostgreSQL EXTRACT(dow)）。
	// 空/全选 = 不过滤。筛的是 start_time 的星期几。
	Weekdays []int
	// Hours #599：小时段过滤（0..23，对齐 PostgreSQL EXTRACT(hour)）。
	// 空/全选 = 不过滤。筛的是 start_time 的整点小时。
	Hours []int
	// CalendarTimezone 是星期/小时筛选使用的系统时区 IANA 名称。
	// 空时 Aggregator.Query/Count 会从注入的 TimezoneProvider 取值；无 provider 回落 UTC。
	CalendarTimezone string
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

type PivotRowKey struct {
	DeviceOUI   string
	DeviceSN    string
	ObjectLDN   string
	Granularity metrics.Granularity
	Time        time.Time
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
	q = a.withCalendarTimezone(ctx, q)
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
		// 设备维度的 KPI/counter 已在写入侧落库。页面直查时不能进入现场 KPI 重算路径，
		// 否则会把用户选中的少量设备放大成重算大查询。
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
//   - device：与 Query 一样按自然键保留最新补报后计数。
//   - device_group / aggregate_group / product / band / network：现场 GROUP BY 后的分组数，
//     故 COUNT(*) FROM (<同 Query 的 GROUP BY 子查询，去 ORDER BY/LIMIT/OFFSET>) sub。
//
// KPI recompute 不改变分组身份（只把同 (object,time) 分组的 counter 行重算成 KPI 行），
// 故总数以「存储层命中分组数」为准，是诚实的「DB 命中多少」。
func (a *Aggregator) Count(ctx context.Context, q QueryRequest) (int, error) {
	q = a.withCalendarTimezone(ctx, q)
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
	if q.Dimension != DimensionDevice && usesKPIQueryPath(q) {
		return a.countByQueryResult(ctx, q)
	}

	switch q.Dimension {
	case DimensionDevice:
		if q.PageByPivotRow {
			return a.countDevicePivotRows(ctx, table, q)
		}
		inner := newRawAwareDeviceSelect(storage.Psql, table, q, "1").
			Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
		inner = applyDeviceFilters(inner, q)
		inner = inner.OrderBy(
			"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
		)
		return a.scanCountSub(ctx, inner)
	case DimensionDeviceGroup:
		inner := storage.Psql.Select("1").From(table)
		inner = applyGroupFilters(inner, q)
		// 行粒度按「组 × 制式 × 指标 × 桶」，GroupBy 必须含 technology，否则截断计数偏小。
		inner = inner.GroupBy("device_group_id", "technology", "metric_path", "granularity", "time")
		return a.scanCountSub(ctx, inner)
	case DimensionAggregateGroup:
		inner := newRawAwareDeviceSelect(storage.Psql, table, q, "1")
		inner = applyDeviceFilters(inner, q)
		// 单条聚合：分组键不含 object_ldn（与 queryAggregateGroupTable 一致），保证截断计数口径相符。
		inner = inner.GroupBy("metric_path", "granularity", "time")
		return a.scanCountSub(ctx, inner)
	case DimensionNetwork:
		inner := newRawAwareDeviceSelect(storage.Psql, table, q, "1")
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

func usesKPIQueryPath(q QueryRequest) bool {
	if q.RecomputeAllKPIs || q.StoreAllEnabled {
		return true
	}
	if q.MetricType != nil && *q.MetricType == metrics.MetricTypeKPI && len(q.MetricPaths) > 0 {
		return true
	}
	return hasKPIIndicatorPath(q.MetricPaths)
}

func hasKPIIndicatorPath(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(path)), "K") {
			return true
		}
	}
	return false
}

func (a *Aggregator) countByQueryResult(ctx context.Context, q QueryRequest) (int, error) {
	countReq := q
	countReq.Limit = 0
	countReq.Offset = 0
	rows, err := a.Query(ctx, countReq)
	if err != nil {
		return 0, fmt.Errorf("aggregator.Count query KPI rollup rows: %w", err)
	}
	if !q.PageByPivotRow {
		return len(rows), nil
	}
	seen := make(map[PivotRowKey]struct{}, len(rows))
	for _, row := range rows {
		objectLDN := ""
		if row.ObjectLDN != nil {
			objectLDN = *row.ObjectLDN
		}
		seen[PivotRowKey{
			DeviceOUI:   row.DeviceOUI,
			DeviceSN:    row.DeviceSN,
			ObjectLDN:   objectLDN,
			Granularity: row.Granularity,
			Time:        row.Time,
		}] = struct{}{}
	}
	return len(seen), nil
}

// DiscoverObjectLDNs 返回同设备、同时间窗下实际出现过的 object_ldn 列表。
// 用于 "全部小区" 查询补骨架：请求未显式传 object_ldns 时，后端从同一粒度表发现展示全集。
// 指标过滤在这里刻意清空，否则当前指标完全无数据时无法发现 object 集合。
func (a *Aggregator) DiscoverObjectLDNs(ctx context.Context, q QueryRequest) ([]string, error) {
	q = a.withCalendarTimezone(ctx, q)
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	if !CanAutoDiscoverObjectSkeletonRequest(q) {
		return nil, nil
	}
	discoverReq := q
	discoverReq.MetricPaths = nil
	discoverReq.MetricType = nil
	discoverReq.ObjectLDNs = nil
	discoverReq.Limit = 0
	discoverReq.Offset = 0

	sqlStr, args, err := buildObjectLDNDiscoverySQL(discoverReq)
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

// buildObjectLDNDiscoverySQL 只读取能够表达“设备在窗口内出现过哪些测量对象”的最小数据源。
// 15min 锚点已经持有 object_ldn；小时以上统一结果表也直接持有 object_ldn。这里不能复用
// pm_metrics* 兼容视图：原始视图会展开指标集并关联指标值、PM 文件和入库批次，仅为 DISTINCT
// object_ldn 就放大成数百万行并行扫描。
//
// 两个源都投影成 applyDeviceFilters 熟悉的列名，再复用同一套设备、制式、时间窗和可见分组
// 过滤，避免轻量查询绕过既有权限语义。
func buildObjectLDNDiscoverySQL(q QueryRequest) (string, []any, error) {
	if _, err := SelectTable(q.Granularity, q.Dimension); err != nil {
		return "", nil, err
	}

	source := newDeviceObjectSource(storage.Psql, q)
	qb := storage.Psql.Select("DISTINCT object_ldn").
		FromSelect(source, "object_source").
		Where("object_ldn <> ''").
		OrderBy("object_ldn")
	qb = applyDeviceFilters(qb, q)
	return qb.ToSql()
}

// newDeviceObjectSource 只投影设备透视骨架需要的列。15min 的锚点天然就是
// (device, object_ldn, granularity, time) 存在性记录；小时以上的统一上卷结果同样直接持有
// 这些列。调用方在外层统一套 applyDeviceFilters，既复用权限/过滤语义，也避免为了分页键、
// 计数或对象发现而展开指标字典和值表。
func newDeviceObjectSource(builder sq.StatementBuilderType, q QueryRequest) sq.SelectBuilder {
	switch q.Granularity {
	case metrics.Granularity15Min:
		source := builder.Select(
			"a.object_ldn AS object_ldn",
			"dev.oui AS device_oui",
			"dev.serial_number AS device_sn",
			"dev.technology AS technology",
			"a.granularity AS granularity",
			`a."time" AS "time"`,
			"a.start_time AS start_time",
		).
			From("pm_measurement_anchors a").
			Join("device_dim dev ON dev.id = a.device_dim_id")
		if frag, args, ok := deviceDimIDPrefilter("a.device_dim_id", "id", q); ok {
			source = source.Where(frag, args...)
		}
		return source
	default:
		source := builder.Select(
			"r.object_ldn AS object_ldn",
			"r.device_oui AS device_oui",
			"r.device_sn AS device_sn",
			"r.technology AS technology",
			"r.granularity AS granularity",
			`r.window_start AS "time"`,
			"r.window_start AS start_time",
		).
			From("pm_aggregation_results r").
			Join(`pm_aggregation_publications published_revision
  ON published_revision.task_version_id = r.task_version_id
 AND published_revision.granularity = r.granularity
 AND published_revision.window_start = r.window_start
 AND published_revision.status = 'published'
 AND published_revision.revision = r.revision`).
			Where(sq.Eq{
				"r.dimension":   string(DimensionDevice),
				"r.granularity": string(q.Granularity),
			})
		if frag, args, ok := deviceDimIDPrefilter("r.dimension_key", "id::text", q); ok {
			source = source.Where(frag, args...)
		}
		return source
	}
}

func newDeviceObjectKeySelect(builder sq.StatementBuilderType, q QueryRequest) sq.SelectBuilder {
	source := newDeviceObjectSource(builder, q)
	qb := builder.Select(
		"device_oui",
		"device_sn",
		"COALESCE(object_ldn, '') AS object_ldn",
		"granularity",
		`"time"`,
	).
		FromSelect(source, "object_source").
		Distinct()
	qb = applyDeviceFilters(qb, q)
	return qb
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
	)
	// Raw rows can overlap when a device retransmits the same natural
	// measurement window under another file name. Match the device query's
	// last-write-wins semantics before aggregating across devices and objects.
	if table == "pm_metrics" || (table == "pm_metrics_hourly" && len(q.MetricPaths) > 0) {
		inner := newRawAwareDeviceSelect(storage.Psql, table, q,
			"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
			"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn",
		).Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
		inner = applyDeviceFilters(inner, q)
		inner = inner.OrderBy(
			"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
		)
		qb = qb.FromSelect(inner, "m")
	} else {
		qb = qb.From(table)
		qb = applyDeviceFilters(qb, q)
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

func newRawAwareDeviceSelect(
	builder sq.StatementBuilderType,
	table string,
	q QueryRequest,
	columns ...string,
) sq.SelectBuilder {
	rolledGranularity, isRolledUpTable := rolledUpDeviceGranularity(table)
	isDeviceDimension := q.Dimension == "" || q.Dimension == DimensionDevice
	if isRolledUpTable && isDeviceDimension {
		return newRolledUpDeviceSelect(builder, rolledGranularity, q, columns...)
	}
	if len(q.MetricPaths) == 0 || table != "pm_metrics" {
		return builder.Select(columns...).From(table)
	}
	targeted := builder.Select(
		"COALESCE(dev.oui,'')::text AS device_oui",
		"COALESCE(dev.serial_number,f.device_sn)::text AS device_sn",
		"d.metric_path", "d.metric_type", "v.metric_value", "d.statis_type", "a.granularity",
		`a."time"`, "a.start_time", "a.end_time",
		"COALESCE(b.committed_at,f.created_at,now()) AS ingest_time",
		"a.object_ldn",
		`jsonb_strip_nulls(jsonb_build_object(
			'device_id',a.device_dim_id::text,'counter_group',a.counter_group,
			'carrier',COALESCE(dev.carrier,f.carrier),
			'technology',COALESCE(dev.technology,f.technology))) AS extra`,
	).From("pm_measurement_anchors a").
		Join("pm_metric_sets s ON s.metric_set_id=a.metric_set_id").
		Join("pm_metric_dictionary d ON d.metric_id=ANY(s.metric_ids)").
		LeftJoin(`pm_metric_values v ON v."time"=a."time" AND v.anchor_id=a.anchor_id AND v.metric_id=d.metric_id`).
		LeftJoin("pm_files f ON f.id=a.source_file_id").
		LeftJoin("pm_ingest_batches b ON b.ingest_batch_id=a.ingest_batch_id").
		LeftJoin("device_dim dev ON dev.id=a.device_dim_id")
	targeted = targeted.Where(sq.Eq{"d.metric_path": q.MetricPaths})
	// 性能收口（万级设备规模下曾实测触发 26-30s 超时/500）：applyDeviceFilters 在外层只能对
	// COALESCE(dev.serial_number, f.device_sn) 这种跨两个 LEFT JOIN 算出来的列做等值过滤，
	// 规划器无法把它下推进 JOIN，导致必须先把请求时间窗内【全部设备】的 anchors 都拉出来
	// JOIN 一遍，最后才筛掉绝大多数行（EXPLAIN 实测 cost ~27万，扫 ~278万行只为留 1-2 行）。
	// 这里在 targeted 子查询内部提前对 a.device_dim_id（真实索引列，idx_pm_anchors_device_time /
	// idx_pm_anchors_15min_device_time_default 都以它打头）加一段语义完全冗余、但可被规划器
	// 下推命中索引的等价谓词，把 anchors 扫描收窄到目标设备。device_dim 保留软删行
	// （tsdbsync/runner.go），不会漏历史设备；仅当从未同步过 device_dim（理论上不应发生）
	// 时才可能收窄过头，落回 f.device_sn 兜底路径——比原先「必现的全表 JOIN 超时」风险小得多。
	if frag, args, ok := deviceDimIDPrefilter("a.device_dim_id", "id", q); ok {
		targeted = targeted.Where(frag, args...)
	}
	return builder.Select(columns...).FromSelect(targeted, "pm_metrics")
}

func rolledUpDeviceGranularity(table string) (metrics.Granularity, bool) {
	switch table {
	case "pm_metrics_hourly":
		return metrics.GranularityHourly, true
	case "pm_metrics_daily":
		return metrics.GranularityDaily, true
	case "pm_metrics_weekly":
		return metrics.GranularityWeekly, true
	case "pm_metrics_monthly":
		return metrics.GranularityMonthly, true
	default:
		return "", false
	}
}

// newRolledUpDeviceSelect 绕过 pm_metrics_{hourly,daily,weekly,monthly} 兼容视图。
// 兼容视图会先对全体设备做 DISTINCT ON，外层设备过滤无法下推；这里先按统一结果表的
// dimension_key 和 granularity 收口，再投影成原查询契约，由调用方在 DISTINCT ON 前继续
// 追加指标、时间、对象和权限过滤。
func newRolledUpDeviceSelect(
	builder sq.StatementBuilderType,
	granularity metrics.Granularity,
	q QueryRequest,
	columns ...string,
) sq.SelectBuilder {
	targeted := builder.Select(
		"r.device_oui AS device_oui",
		"r.device_sn AS device_sn",
		"r.metric_path AS metric_path",
		"r.metric_type AS metric_type",
		"r.metric_value AS metric_value",
		"r.aggregation_op::text AS statis_type",
		"r.granularity::text AS granularity",
		`r.window_start AS "time"`,
		"r.window_start AS start_time",
		"r.window_end AS end_time",
		"r.created_at AS ingest_time",
		"r.object_ldn AS object_ldn",
		`jsonb_build_object(
			'task_id',r.task_id,'task_version_id',r.task_version_id,
			'complete',r.complete,'missing_slots',r.missing_slots) AS extra`,
	).
		From("pm_aggregation_results r").
		Join(`pm_aggregation_publications published_revision
  ON published_revision.task_version_id = r.task_version_id
 AND published_revision.granularity = r.granularity
 AND published_revision.window_start = r.window_start
 AND published_revision.status = 'published'
 AND published_revision.revision = r.revision`).
		Where(sq.Eq{
			"r.dimension":   string(DimensionDevice),
			"r.granularity": string(granularity),
		})
	if frag, args, ok := deviceDimIDPrefilter("r.dimension_key", "id::text", q); ok {
		targeted = targeted.Where(frag, args...)
	}
	return builder.Select(columns...).FromSelect(targeted, "pm_metrics")
}

// deviceDimIDPrefilter 把 QueryRequest 里的设备过滤（DeviceOUIs/DeviceSNs/Technologies）
// 镜像映射成一段基于 device_dim 的子查询谓词，供 newRawAwareDeviceSelect 提前收口
// 原始锚点的 a.device_dim_id 或统一结果的 r.dimension_key。deviceIDColumn 和 idProjection
// 只由内部固定调用点传入，不接收外部输入。分支与 applyDeviceFilters 一一对应，纯粹是
// 同一批过滤条件的等价改写（不引入新语义），未传设备过滤时返回 ok=false。
func deviceDimIDPrefilter(deviceIDColumn, idProjection string, q QueryRequest) (string, []any, bool) {
	prefix := deviceIDColumn + " IN (SELECT " + idProjection + " FROM device_dim WHERE "
	switch {
	case len(q.DeviceSNs) > 0 && len(q.Technologies) > 0:
		if len(q.DeviceOUIs) > 0 {
			return prefix + "serial_number = ANY(?) AND technology = ANY(?) AND oui = ANY(?))",
				[]any{q.DeviceSNs, q.Technologies, q.DeviceOUIs}, true
		}
		return prefix + "serial_number = ANY(?) AND technology = ANY(?))",
			[]any{q.DeviceSNs, q.Technologies}, true
	case len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0:
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		var b strings.Builder
		args := make([]any, 0, n*2)
		b.WriteString(prefix)
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(" OR ")
			}
			b.WriteString("(oui = ? AND serial_number = ?)")
			args = append(args, q.DeviceOUIs[i], q.DeviceSNs[i])
		}
		b.WriteString(")")
		return b.String(), args, true
	case len(q.DeviceOUIs) > 0:
		return prefix + "oui = ANY(?))", []any{q.DeviceOUIs}, true
	case len(q.DeviceSNs) > 0:
		return prefix + "serial_number = ANY(?))", []any{q.DeviceSNs}, true
	default:
		return "", nil, false
	}
}

func isDeviceDimension(d Dimension) bool {
	return d == "" || d == DimensionDevice
}

func rolledUpDeviceResultCandidateSelect(builder sq.StatementBuilderType, q QueryRequest, columns ...string) sq.SelectBuilder {
	qb := builder.Select(columns...).
		From("pm_aggregation_results r").
		Where(sq.Eq{
			"r.dimension":   string(DimensionDevice),
			"r.granularity": string(q.Granularity),
		})
	if frag, args, ok := deviceDimIDPrefilter("r.dimension_key", "id::text", q); ok {
		qb = qb.Where(frag, args...)
	} else if len(q.Technologies) > 0 {
		qb = qb.Where(sq.Eq{"r.technology": q.Technologies})
	}
	qb = applyRolledUpResultMetricFilters(qb, q)
	qb = applyRolledUpResultScalarFilters(qb, q)
	qb = authz.ApplyDeviceSNVisibilityFilter(qb, "r.device_sn", q.VisibleGroups)
	return qb
}

func applyRolledUpResultMetricFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.MetricPaths) > 0 {
		if q.MetricType == nil {
			byType := map[metrics.MetricType][]string{}
			unknown := make([]string, 0)
			for _, raw := range q.MetricPaths {
				path := strings.TrimSpace(raw)
				if path == "" {
					continue
				}
				if mt, ok := metricTypeFromIndicatorPath(path); ok {
					byType[mt] = append(byType[mt], path)
				} else {
					unknown = append(unknown, path)
				}
			}
			or := sq.Or{}
			for _, mt := range []metrics.MetricType{metrics.MetricTypeKPI, metrics.MetricTypeCounter} {
				if paths := byType[mt]; len(paths) > 0 {
					or = append(or, sq.And{
						sq.Eq{"r.metric_type": string(mt)},
						sq.Eq{"r.metric_path": paths},
					})
				}
			}
			if len(unknown) > 0 {
				or = append(or, sq.Eq{"r.metric_path": unknown})
			}
			if len(or) > 0 {
				qb = qb.Where(or)
			}
		} else {
			qb = qb.Where(sq.Eq{"r.metric_path": q.MetricPaths})
		}
	}
	if q.MetricType != nil {
		qb = qb.Where(sq.Eq{"r.metric_type": string(*q.MetricType)})
	}
	return qb
}

func applyRolledUpResultScalarFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if !q.StartTime.IsZero() {
		qb = qb.Where(sq.GtOrEq{"r.window_start": q.StartTime})
	}
	if !q.EndTime.IsZero() {
		qb = qb.Where(sq.Lt{"r.window_start": q.EndTime})
	}
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		qb = qb.Where(calendarfilter.ExtractDOWPredicate("r.window_start"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Weekdays)
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		qb = qb.Where(calendarfilter.ExtractHourPredicate("r.window_start"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Hours)
	}
	if len(q.ObjectLDNs) > 0 {
		qb = qb.Where(sq.Eq{"r.object_ldn": q.ObjectLDNs})
	}
	return qb
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
	if q.PageByPivotRow {
		return buildDevicePivotRowPageSQL(table, q)
	}
	inner := newRawAwareDeviceSelect(storage.Psql, table, q, deviceTableColumns...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
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

func buildDevicePivotRowPageSQL(table string, q QueryRequest) (string, []any, error) {
	if len(q.PivotRowKeys) > 0 {
		return buildDevicePivotRowsForKeysSQL(table, q)
	}
	if table != "pm_metrics" && isDeviceDimension(q.Dimension) {
		return buildRolledUpDevicePivotRowPageSQL(q)
	}
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	inner := newRawAwareDeviceSelect(builder, table, q, deviceTableColumns...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	inner = applyDeviceFilters(inner, q)
	inner = inner.OrderBy(
		"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
	)
	innerSQL, args, err := inner.ToSql()
	if err != nil {
		return "", nil, err
	}

	pageReq := q
	if IsExplicitObjectSkeletonRequest(q) {
		pageReq.MetricPaths = nil
		pageReq.MetricType = nil
	}
	var pageInner sq.SelectBuilder
	if IsExplicitObjectSkeletonRequest(q) {
		pageInner = newDeviceObjectKeySelect(builder, pageReq)
	} else {
		pageInner = newRawAwareDeviceSelect(builder, table, pageReq, deviceTableColumns...).
			Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
		pageInner = applyDeviceFilters(pageInner, pageReq)
		pageInner = pageInner.OrderBy(
			"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
		)
	}
	pageInnerSQL, pageArgs, err := pageInner.ToSql()
	if err != nil {
		return "", nil, err
	}
	args = append(args, pageArgs...)

	limitSQL := ""
	if q.Limit > 0 {
		args = append(args, q.Limit)
		limitSQL = "\n  LIMIT ?"
	}
	offsetSQL := ""
	if q.Offset > 0 {
		args = append(args, q.Offset)
		offsetSQL = "\n  OFFSET ?"
	}

	sqlStr := fmt.Sprintf(`
WITH dedup AS (
  %s
),
page_keys AS (
  SELECT DISTINCT device_oui, device_sn, COALESCE(object_ldn, '') AS object_ldn, granularity, "time"
  FROM (
    %s
  ) pk_dedup
  ORDER BY "time" DESC, device_sn ASC, object_ldn ASC%s%s
)
SELECT %s
FROM dedup d
JOIN page_keys pk
  ON pk.device_oui = d.device_oui
 AND pk.device_sn = d.device_sn
 AND pk.object_ldn = COALESCE(d.object_ldn, '')
 AND pk.granularity = d.granularity
 AND pk."time" = d."time"
ORDER BY d."time" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC`,
		innerSQL, pageInnerSQL, limitSQL, offsetSQL, prefixedColumns("d", deviceTableColumns))
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRolledUpDevicePivotRowPageSQL(q QueryRequest) (string, []any, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	candidates := rolledUpDeviceResultCandidateSelect(builder, q,
		"r.device_oui AS device_oui",
		"r.device_sn AS device_sn",
		"r.metric_path AS metric_path",
		"r.metric_type AS metric_type",
		"r.metric_value AS metric_value",
		"r.aggregation_op::text AS statis_type",
		"r.granularity::text AS granularity",
		`r.window_start AS "time"`,
		"r.window_start AS start_time",
		"r.window_end AS end_time",
		"r.created_at AS ingest_time",
		"r.object_ldn AS object_ldn",
		`jsonb_build_object(
			'task_id',r.task_id,'task_version_id',r.task_version_id,
			'complete',r.complete,'missing_slots',r.missing_slots) AS extra`,
		"r.task_version_id",
		"r.revision",
	)
	candidateSQL, args, err := candidates.ToSql()
	if err != nil {
		return "", nil, err
	}
	limitSQL := ""
	if q.Limit > 0 {
		args = append(args, q.Limit)
		limitSQL = "\n  LIMIT ?"
	}
	offsetSQL := ""
	if q.Offset > 0 {
		args = append(args, q.Offset)
		offsetSQL = "\n  OFFSET ?"
	}

	sqlStr := fmt.Sprintf(`
WITH candidate_results AS MATERIALIZED (
  %s
),
dedup AS (
  SELECT DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)
         %s
  FROM candidate_results d
  JOIN pm_aggregation_publications published_revision
    ON published_revision.task_version_id = d.task_version_id
   AND published_revision.granularity = d.granularity
   AND published_revision.window_start = d."time"
   AND published_revision.status = 'published'
   AND published_revision.revision = d.revision
  ORDER BY device_oui, device_sn, metric_path, granularity, "time", object_ldn, ingest_time DESC
),
page_keys AS (
  SELECT DISTINCT device_oui, device_sn, COALESCE(object_ldn, '') AS object_ldn, granularity, "time"
  FROM dedup
  ORDER BY "time" DESC, device_sn ASC, object_ldn ASC%s%s
)
SELECT %s
FROM dedup d
JOIN page_keys pk
  ON pk.device_oui = d.device_oui
 AND pk.device_sn = d.device_sn
 AND pk.object_ldn = COALESCE(d.object_ldn, '')
 AND pk.granularity = d.granularity
 AND pk."time" = d."time"
ORDER BY d."time" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC`,
		candidateSQL, prefixedColumns("d", append(append([]string{}, deviceTableColumns...), "task_version_id", "revision")), limitSQL, offsetSQL, prefixedColumns("d", deviceTableColumns))
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildDevicePivotRowsForKeysSQL(table string, q QueryRequest) (string, []any, error) {
	if table == "pm_metrics" {
		return buildRawDevicePivotRowsForKeysSQL(q)
	}
	if isDeviceDimension(q.Dimension) {
		return buildRolledUpDevicePivotRowsForKeysSQL(q)
	}
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	inner := newRawAwareDeviceSelect(builder, table, q, deviceTableColumns...).
		Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	inner = applyDeviceFilters(inner, q)
	inner = inner.OrderBy(
		"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
	)
	innerSQL, innerArgs, err := inner.ToSql()
	if err != nil {
		return "", nil, err
	}

	valueRows := make([]string, 0, len(q.PivotRowKeys))
	args := make([]any, 0, len(q.PivotRowKeys)*5+len(innerArgs))
	for _, key := range q.PivotRowKeys {
		valueRows = append(valueRows, "(?::text,?::text,?::text,?::text,?::timestamptz)")
		args = append(args, key.DeviceOUI, key.DeviceSN, key.ObjectLDN, string(key.Granularity), key.Time)
	}
	args = append(args, innerArgs...)

	sqlStr := fmt.Sprintf(`
WITH requested_keys(device_oui, device_sn, object_ldn, granularity, "time") AS (
  VALUES %s
),
dedup AS (
  %s
)
SELECT %s
FROM dedup d
JOIN requested_keys pk
  ON pk.device_oui = d.device_oui
 AND pk.device_sn = d.device_sn
 AND pk.object_ldn = COALESCE(d.object_ldn, '')
 AND pk.granularity = d.granularity
 AND pk."time" = d."time"
ORDER BY d."time" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC`,
		strings.Join(valueRows, ", "), innerSQL, prefixedColumns("d", deviceTableColumns))
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRolledUpDevicePivotRowsForKeysSQL(q QueryRequest) (string, []any, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	valueRows := make([]string, 0, len(q.PivotRowKeys))
	args := make([]any, 0, len(q.PivotRowKeys)*5+len(q.MetricPaths)*2+3)
	for _, key := range q.PivotRowKeys {
		valueRows = append(valueRows, "(?::text,?::text,?::text,?::text,?::timestamptz)")
		args = append(args, key.DeviceOUI, key.DeviceSN, key.ObjectLDN, string(key.Granularity), key.Time)
	}

	candidates := builder.Select(
		"r.device_oui AS device_oui",
		"r.device_sn AS device_sn",
		"r.metric_path AS metric_path",
		"r.metric_type AS metric_type",
		"r.metric_value AS metric_value",
		"r.aggregation_op::text AS statis_type",
		"r.granularity::text AS granularity",
		`r.window_start AS "time"`,
		"r.window_start AS start_time",
		"r.window_end AS end_time",
		"r.created_at AS ingest_time",
		"r.object_ldn AS object_ldn",
		`jsonb_build_object(
			'task_id',r.task_id,'task_version_id',r.task_version_id,
			'complete',r.complete,'missing_slots',r.missing_slots) AS extra`,
		"r.task_version_id",
		"r.revision",
	).
		Options(`DISTINCT ON (r.device_oui, r.device_sn, r.metric_path, r.granularity, r.window_start, r.object_ldn)`).
		From("requested_keys pk").
		Join("device_dim dev ON dev.oui = pk.device_oui AND dev.serial_number = pk.device_sn").
		Join(`pm_aggregation_results r
  ON r.dimension_key = dev.id::text
 AND r.device_oui = pk.device_oui
 AND r.device_sn = pk.device_sn
 AND COALESCE(r.object_ldn, '') = pk.object_ldn
 AND r.granularity = pk.granularity
 AND r.window_start = pk."time"`).
		Where(sq.Eq{"r.dimension": string(DimensionDevice)})
	candidates = applyRolledUpResultMetricFilters(candidates, q)
	candidates = candidates.OrderBy(
		"r.device_oui", "r.device_sn", "r.metric_path", "r.granularity", "r.window_start", "r.object_ldn", "r.created_at DESC",
	)
	candidateSQL, candidateArgs, err := candidates.ToSql()
	if err != nil {
		return "", nil, err
	}
	args = append(args, candidateArgs...)

	sqlStr := fmt.Sprintf(`
WITH requested_keys(device_oui, device_sn, object_ldn, granularity, "time") AS (
  VALUES %s
),
candidate_results AS MATERIALIZED (
  %s
)
SELECT %s
FROM candidate_results d
JOIN pm_aggregation_publications published_revision
  ON published_revision.task_version_id = d.task_version_id
 AND published_revision.granularity = d.granularity
 AND published_revision.window_start = d."time"
 AND published_revision.status = 'published'
 AND published_revision.revision = d.revision
ORDER BY d."time" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC`,
		strings.Join(valueRows, ", "), candidateSQL, prefixedColumns("d", deviceTableColumns))
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRawDevicePivotRowsForKeysSQL(q QueryRequest) (string, []any, error) {
	if IsExplicitObjectSkeletonRequest(q) {
		return buildRawDevicePivotSkeletonRowsForKeysSQL(q)
	}
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	valueRows := make([]string, 0, len(q.PivotRowKeys))
	args := make([]any, 0, len(q.PivotRowKeys)*5+len(q.MetricPaths)*2+3)
	for _, key := range q.PivotRowKeys {
		valueRows = append(valueRows, "(?::text,?::text,?::text,?::text,?::timestamptz)")
		args = append(args, key.DeviceOUI, key.DeviceSN, key.ObjectLDN, string(key.Granularity), key.Time)
	}

	valueJoin := `pm_metric_values v ON v."time"=a."time" AND v.anchor_id=a.anchor_id AND v.metric_id=d.metric_id`
	valueJoinArgs := make([]any, 0, 2)
	if !q.StartTime.IsZero() {
		valueJoin += ` AND v."time" >= ?`
		valueJoinArgs = append(valueJoinArgs, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		valueJoin += ` AND v."time" < ?`
		valueJoinArgs = append(valueJoinArgs, q.EndTime)
	}

	targeted := builder.Select(
		"COALESCE(dev.oui,'')::text AS device_oui",
		"COALESCE(dev.serial_number,'')::text AS device_sn",
		"d.metric_path",
		"d.metric_type",
		"v.metric_value",
		"d.statis_type",
		"a.granularity",
		`a."time"`,
		"a.start_time",
		"a.end_time",
		"a.end_time AS ingest_time",
		"a.object_ldn",
		"a.anchor_id AS ingest_sequence",
		`jsonb_strip_nulls(jsonb_build_object(
			'device_id',a.device_dim_id::text,'counter_group',a.counter_group,
			'carrier',dev.carrier,
			'technology',dev.technology)) AS extra`,
	).
		From("requested_keys pk").
		Join("device_dim dev ON dev.oui = pk.device_oui AND dev.serial_number = pk.device_sn").
		Join(`pm_measurement_anchors a
  ON a.device_dim_id = dev.id
 AND a.object_ldn = pk.object_ldn
 AND a.granularity = pk.granularity
 AND a."time" = pk."time"`)
	targeted = targeted.
		Join("pm_metric_sets s ON s.metric_set_id=a.metric_set_id").
		Join("pm_metric_dictionary d ON d.metric_id=ANY(s.metric_ids)")
	targeted = applyMetricDictionaryFilters(targeted, q)
	targeted = targeted.
		LeftJoin(valueJoin, valueJoinArgs...)
	if len(q.DeviceSNs) > 0 {
		targeted = targeted.Where(sq.Eq{"dev.serial_number": q.DeviceSNs})
	}
	if len(q.DeviceOUIs) > 0 {
		targeted = targeted.Where(sq.Eq{"dev.oui": q.DeviceOUIs})
	}
	if len(q.Technologies) > 0 {
		targeted = targeted.Where(sq.Eq{"dev.technology": q.Technologies})
	}
	targeted = authz.ApplyDeviceSNVisibilityFilter(targeted, "dev.serial_number", q.VisibleGroups)

	targetedSQL, targetedArgs, err := targeted.ToSql()
	if err != nil {
		return "", nil, err
	}
	args = append(args, targetedArgs...)

	sqlStr := fmt.Sprintf(`
WITH requested_keys(device_oui, device_sn, object_ldn, granularity, "time") AS (
  VALUES %s
),
dedup AS (
  SELECT DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)
    %s
  FROM (
    %s
  ) pm_metrics
  ORDER BY device_oui, device_sn, metric_path, granularity, "time", object_ldn, ingest_sequence DESC
)
SELECT %s
FROM dedup d
ORDER BY d."time" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC`,
		strings.Join(valueRows, ", "), strings.Join(deviceTableColumns, ", "), targetedSQL, prefixedColumns("d", deviceTableColumns))
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRawDevicePivotSkeletonRowsForKeysSQL(q QueryRequest) (string, []any, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	valueRows := make([]string, 0, len(q.PivotRowKeys))
	args := make([]any, 0, len(q.PivotRowKeys)*5+len(q.MetricPaths)*2+3)
	for _, key := range q.PivotRowKeys {
		valueRows = append(valueRows, "(?::text,?::text,?::text,?::text,?::timestamptz)")
		args = append(args, key.DeviceOUI, key.DeviceSN, key.ObjectLDN, string(key.Granularity), key.Time)
	}

	latestAnchors := builder.Select(
		"COALESCE(dev.oui,'')::text AS device_oui",
		"COALESCE(dev.serial_number,'')::text AS device_sn",
		"a.device_dim_id",
		"a.object_ldn",
		"'15min'::text AS granularity",
		`a."time"`,
		"a.start_time",
		"a.end_time",
		"a.counter_group",
		"a.anchor_id",
		"dev.carrier",
		"dev.technology",
	).
		Options(`DISTINCT ON (dev.oui, dev.serial_number, a.granularity, a."time", a.object_ldn)`).
		From("requested_keys pk").
		Join("device_dim dev ON dev.oui = pk.device_oui AND dev.serial_number = pk.device_sn").
		Join(`pm_measurement_anchors a
  ON a.device_dim_id = dev.id
	AND a.object_ldn = pk.object_ldn
 AND a.granularity = '15min'
 AND a."time" = pk."time"`)
	if len(q.DeviceSNs) > 0 {
		latestAnchors = latestAnchors.Where(sq.Eq{"dev.serial_number": q.DeviceSNs})
	}
	if len(q.DeviceOUIs) > 0 {
		latestAnchors = latestAnchors.Where(sq.Eq{"dev.oui": q.DeviceOUIs})
	}
	if len(q.Technologies) > 0 {
		latestAnchors = latestAnchors.Where(sq.Eq{"dev.technology": q.Technologies})
	}
	if !q.StartTime.IsZero() {
		latestAnchors = latestAnchors.Where(sq.GtOrEq{`a."time"`: q.StartTime})
	}
	if !q.EndTime.IsZero() {
		latestAnchors = latestAnchors.Where(sq.Lt{`a."time"`: q.EndTime})
	}
	latestAnchors = authz.ApplyDeviceSNVisibilityFilter(latestAnchors, "dev.serial_number", q.VisibleGroups)
	latestAnchors = latestAnchors.OrderBy(
		"dev.oui", "dev.serial_number", "a.granularity", `a."time"`, "a.object_ldn", "a.anchor_id DESC",
	)
	latestAnchorsSQL, latestAnchorsArgs, err := latestAnchors.ToSql()
	if err != nil {
		return "", nil, err
	}
	args = append(args, latestAnchorsArgs...)

	requestedMetrics := builder.Select("d.metric_id", "d.metric_path", "d.metric_type", "d.statis_type").
		From("pm_metric_dictionary d")
	requestedMetrics = applyMetricDictionaryFilters(requestedMetrics, q)
	requestedMetricsSQL, requestedMetricsArgs, err := requestedMetrics.ToSql()
	if err != nil {
		return "", nil, err
	}
	args = append(args, requestedMetricsArgs...)

	valuesForAnchorsSQL, valuesForAnchorsArgs := buildRawDevicePivotSkeletonValuesSQL(q)
	args = append(args, valuesForAnchorsArgs...)

	sqlStr := fmt.Sprintf(`
WITH requested_keys(device_oui, device_sn, object_ldn, granularity, "time") AS (
  VALUES %s
),
latest_anchors AS MATERIALIZED (
  %s
),
requested_metrics AS MATERIALIZED (
  %s
),
value_keys AS MATERIALIZED (
  SELECT a.anchor_id, a."time", d.metric_id
  FROM latest_anchors a
  JOIN requested_metrics d ON TRUE
),
values_for_anchors AS MATERIALIZED (
  %s
)
SELECT a.device_oui,
       a.device_sn,
       d.metric_path,
       d.metric_type,
       v.metric_value,
       d.statis_type,
       a.granularity,
       a."time",
       a.start_time,
       a.end_time,
       a.end_time AS ingest_time,
       a.object_ldn,
       jsonb_strip_nulls(jsonb_build_object(
         'device_id',a.device_dim_id::text,'counter_group',a.counter_group,
         'carrier',a.carrier,
         'technology',a.technology)) AS extra
FROM latest_anchors a
JOIN requested_metrics d ON TRUE
LEFT JOIN values_for_anchors v
  ON v.anchor_id=a.anchor_id
 AND v."time"=a."time"
 AND v.metric_id=d.metric_id
ORDER BY a."time" DESC, a.device_sn ASC, COALESCE(a.object_ldn, '') ASC, d.metric_path ASC`,
		strings.Join(valueRows, ", "), latestAnchorsSQL, requestedMetricsSQL, valuesForAnchorsSQL)
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

type timeRange struct {
	start time.Time
	end   time.Time
}

func buildRawDevicePivotSkeletonValuesSQL(q QueryRequest) (string, []any) {
	start, end := metricValueLookupRange(q)
	segments := metricValueLookupSegments(start, end)
	if len(segments) == 0 {
		return `SELECT k.anchor_id, k."time", k.metric_id, v.metric_value
  FROM value_keys k
  JOIN pm_metric_values v
    ON v.anchor_id=k.anchor_id
   AND v.metric_id=k.metric_id
   AND v."time"=k."time"`, nil
	}

	parts := make([]string, 0, len(segments))
	args := make([]any, 0, len(segments)*4)
	for _, segment := range segments {
		parts = append(parts, `SELECT k.anchor_id, k."time", k.metric_id, v.metric_value
  FROM value_keys k
  JOIN pm_metric_values v
    ON v.anchor_id=k.anchor_id
   AND v.metric_id=k.metric_id
   AND v."time"=k."time"
  WHERE k."time" >= ?
    AND k."time" < ?
    AND v."time" >= ?
    AND v."time" < ?`)
		args = append(args, segment.start, segment.end, segment.start, segment.end)
	}
	return strings.Join(parts, "\n  UNION ALL\n  "), args
}

func metricValueLookupRange(q QueryRequest) (time.Time, time.Time) {
	var minTime, maxTime time.Time
	for _, key := range q.PivotRowKeys {
		if key.Time.IsZero() {
			continue
		}
		if minTime.IsZero() || key.Time.Before(minTime) {
			minTime = key.Time
		}
		if maxTime.IsZero() || key.Time.After(maxTime) {
			maxTime = key.Time
		}
	}
	if minTime.IsZero() {
		return q.StartTime, q.EndTime
	}
	return minTime, maxTime.Add(pmMetricValueRawBucketWindow)
}

func metricValueLookupSegments(start, end time.Time) []timeRange {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return nil
	}

	cursor := start.Truncate(pmMetricValueChunkPruneSegment)
	segments := make([]timeRange, 0, int(end.Sub(start)/pmMetricValueChunkPruneSegment)+1)
	for cursor.Before(end) {
		next := cursor.Add(pmMetricValueChunkPruneSegment)
		segmentStart := cursor
		if segmentStart.Before(start) {
			segmentStart = start
		}
		segmentEnd := next
		if segmentEnd.After(end) {
			segmentEnd = end
		}
		if segmentEnd.After(segmentStart) {
			segments = append(segments, timeRange{start: segmentStart, end: segmentEnd})
		}
		cursor = next
	}
	return segments
}

func applyMetricDictionaryFilters(qb sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	if len(q.MetricPaths) > 0 {
		if q.MetricType == nil {
			or := sq.Or{}
			for _, raw := range q.MetricPaths {
				path := strings.TrimSpace(raw)
				if path == "" {
					continue
				}
				if mt, ok := metricTypeFromIndicatorPath(path); ok {
					or = append(or, sq.And{
						sq.Eq{"d.metric_path": path},
						sq.Eq{"d.metric_type": string(mt)},
					})
				} else {
					or = append(or, sq.Eq{"d.metric_path": path})
				}
			}
			if len(or) > 0 {
				qb = qb.Where(or)
			}
		} else {
			qb = qb.Where(sq.Eq{"d.metric_path": q.MetricPaths})
		}
	}
	if q.MetricType != nil {
		qb = qb.Where(sq.Eq{"d.metric_type": string(*q.MetricType)})
	}
	return qb
}

func (a *Aggregator) DevicePivotRowKeys(ctx context.Context, q QueryRequest) ([]PivotRowKey, error) {
	q = a.withCalendarTimezone(ctx, q)
	if q.Dimension == "" {
		q.Dimension = DimensionDevice
	}
	table, err := SelectTable(q.Granularity, q.Dimension)
	if err != nil {
		return nil, err
	}
	if table == "pm_metrics" && IsExplicitObjectSkeletonRequest(q) {
		return a.deviceObjectSkeletonPivotRowKeys(ctx, q)
	}
	sqlStr, args, err := buildDevicePivotRowKeysSQL(table, q)
	if err != nil {
		return nil, fmt.Errorf("aggregator.DevicePivotRowKeys build: %w", err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.DevicePivotRowKeys query %s: %w", table, err)
	}
	defer rows.Close()

	out := make([]PivotRowKey, 0)
	for rows.Next() {
		var key PivotRowKey
		var granularity string
		if err := rows.Scan(&key.DeviceOUI, &key.DeviceSN, &key.ObjectLDN, &granularity, &key.Time); err != nil {
			return nil, fmt.Errorf("aggregator.DevicePivotRowKeys scan %s: %w", table, err)
		}
		key.Granularity = metrics.Granularity(granularity)
		out = append(out, key)
	}
	return out, rows.Err()
}

type devicePivotSkeletonTarget struct {
	oui string
	sn  string
}

func (a *Aggregator) deviceObjectSkeletonPivotRowKeys(ctx context.Context, q QueryRequest) ([]PivotRowKey, error) {
	devices, err := a.devicePivotSkeletonTargets(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, nil
	}
	buckets := skeletonBuckets(q)
	if len(buckets) == 0 {
		return nil, nil
	}
	objectLDNs := append([]string(nil), q.ObjectLDNs...)
	sort.Strings(objectLDNs)

	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	limit := q.Limit
	capHint := 0
	if limit > 0 {
		capHint = limit
	}
	out := make([]PivotRowKey, 0, capHint)
	seen := 0
	for bi := len(buckets) - 1; bi >= 0; bi-- {
		bucket := buckets[bi]
		for _, dev := range devices {
			for _, objectLDN := range objectLDNs {
				if seen < offset {
					seen++
					continue
				}
				out = append(out, PivotRowKey{
					DeviceOUI:   dev.oui,
					DeviceSN:    dev.sn,
					ObjectLDN:   objectLDN,
					Granularity: q.Granularity,
					Time:        bucket,
				})
				seen++
				if limit > 0 && len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func (a *Aggregator) devicePivotSkeletonTargets(ctx context.Context, q QueryRequest) ([]devicePivotSkeletonTarget, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	devices := builder.Select("COALESCE(dev.oui, '')", "COALESCE(dev.serial_number, '')").
		From("device_dim dev")
	devices = applyRawPivotTargetDeviceFilters(devices, q)
	devices = authz.ApplyDeviceSNVisibilityFilter(devices, "dev.serial_number", q.VisibleGroups)
	devices = devices.OrderBy("dev.serial_number ASC", "dev.oui ASC")
	sqlStr, args, err := devices.ToSql()
	if err != nil {
		return nil, fmt.Errorf("aggregator.DevicePivotRowKeys build skeleton devices: %w", err)
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.DevicePivotRowKeys query skeleton devices: %w", err)
	}
	defer rows.Close()

	out := make([]devicePivotSkeletonTarget, 0)
	for rows.Next() {
		var target devicePivotSkeletonTarget
		if err := rows.Scan(&target.oui, &target.sn); err != nil {
			return nil, fmt.Errorf("aggregator.DevicePivotRowKeys scan skeleton devices: %w", err)
		}
		out = append(out, target)
	}
	return out, rows.Err()
}

func buildDevicePivotRowKeysSQL(table string, q QueryRequest) (string, []any, error) {
	if table == "pm_metrics" && IsExplicitObjectSkeletonRequest(q) {
		return buildRawDevicePivotRowKeysSQL(q)
	}
	if table != "pm_metrics" && isDeviceDimension(q.Dimension) {
		return buildRolledUpDevicePivotRowKeysSQL(q)
	}
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	pageReq := q
	if IsExplicitObjectSkeletonRequest(q) {
		pageReq.MetricPaths = nil
		pageReq.MetricType = nil
	}
	var inner sq.SelectBuilder
	if IsExplicitObjectSkeletonRequest(q) {
		inner = newDeviceObjectKeySelect(builder, pageReq)
	} else {
		inner = newRawAwareDeviceSelect(builder, table, pageReq, deviceTableColumns...).
			Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
		inner = applyDeviceFilters(inner, pageReq)
		inner = inner.OrderBy(
			"device_oui", "device_sn", "metric_path", "granularity", `"time"`, "object_ldn", "ingest_time DESC",
		)
	}
	innerSQL, args, err := inner.ToSql()
	if err != nil {
		return "", nil, err
	}
	limitSQL := ""
	if q.Limit > 0 {
		args = append(args, q.Limit)
		limitSQL = "\nLIMIT ?"
	}
	offsetSQL := ""
	if q.Offset > 0 {
		args = append(args, q.Offset)
		offsetSQL = "\nOFFSET ?"
	}
	sqlStr := fmt.Sprintf(`SELECT DISTINCT device_oui, device_sn, COALESCE(object_ldn, '') AS object_ldn, granularity, "time"
FROM (
  %s
) dedup
ORDER BY "time" DESC, device_sn ASC, object_ldn ASC%s%s`, innerSQL, limitSQL, offsetSQL)
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRawDevicePivotRowKeysSQL(q QueryRequest) (string, []any, error) {
	if q.Granularity == "" {
		return "", nil, errors.New("raw device pivot row keys require granularity")
	}
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	devices := builder.Select("id", "oui", "serial_number").
		From("device_dim dev")
	devices = applyRawPivotTargetDeviceFilters(devices, q)
	devices = authz.ApplyDeviceSNVisibilityFilter(devices, "dev.serial_number", q.VisibleGroups)
	devicesSQL, devicesArgs, err := devices.ToSql()
	if err != nil {
		return "", nil, err
	}

	keys := builder.Select(
		"a.device_dim_id",
		"a.object_ldn",
		`a."time"`,
	).
		Distinct().
		From("pm_measurement_anchors a").
		Join("target_devices dev ON dev.id = a.device_dim_id").
		Where(sq.Eq{"a.granularity": string(q.Granularity)})
	if !q.StartTime.IsZero() {
		keys = keys.Where(sq.GtOrEq{`a."time"`: q.StartTime})
	}
	if !q.EndTime.IsZero() {
		keys = keys.Where(sq.Lt{`a."time"`: q.EndTime})
	}
	if len(q.ObjectLDNs) > 0 {
		keys = keys.Where(sq.Eq{"a.object_ldn": q.ObjectLDNs})
	}
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		keys = keys.Where(calendarfilter.ExtractDOWPredicate("a.start_time"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Weekdays)
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		keys = keys.Where(calendarfilter.ExtractHourPredicate("a.start_time"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Hours)
	}
	keysSQL, keysArgs, err := keys.ToSql()
	if err != nil {
		return "", nil, err
	}

	args := make([]any, 0, len(devicesArgs)+len(keysArgs)+1)
	args = append(args, devicesArgs...)
	args = append(args, keysArgs...)
	args = append(args, string(q.Granularity))
	limitSQL := ""
	if q.Limit > 0 {
		args = append(args, q.Limit)
		limitSQL = "\nLIMIT ?"
	}
	offsetSQL := ""
	if q.Offset > 0 {
		args = append(args, q.Offset)
		offsetSQL = "\nOFFSET ?"
	}
	sqlStr := fmt.Sprintf(`
WITH target_devices AS (
  %s
),
anchor_keys AS (
  %s
)
SELECT dev.oui AS device_oui,
       dev.serial_number AS device_sn,
       COALESCE(k.object_ldn, '') AS object_ldn,
       ?::text AS granularity,
       k."time"
FROM anchor_keys k
JOIN target_devices dev ON dev.id = k.device_dim_id
ORDER BY k."time" DESC, dev.serial_number ASC, COALESCE(k.object_ldn, '') ASC%s%s`,
		devicesSQL, keysSQL, limitSQL, offsetSQL)
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func buildRolledUpDevicePivotRowKeysSQL(q QueryRequest) (string, []any, error) {
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	candidates := rolledUpDeviceResultCandidateSelect(builder, q,
		"r.device_oui",
		"r.device_sn",
		"r.object_ldn",
		"r.granularity",
		"r.window_start",
		"r.task_version_id",
		"r.revision",
	)
	candidateSQL, args, err := candidates.ToSql()
	if err != nil {
		return "", nil, err
	}
	limitSQL := ""
	if q.Limit > 0 {
		args = append(args, q.Limit)
		limitSQL = "\nLIMIT ?"
	}
	offsetSQL := ""
	if q.Offset > 0 {
		args = append(args, q.Offset)
		offsetSQL = "\nOFFSET ?"
	}

	sqlStr := fmt.Sprintf(`
WITH candidate_results AS MATERIALIZED (
  %s
)
SELECT DISTINCT r.device_oui, r.device_sn, COALESCE(r.object_ldn, '') AS object_ldn, r.granularity, r.window_start AS "time"
FROM candidate_results r
JOIN pm_aggregation_publications published_revision
  ON published_revision.task_version_id = r.task_version_id
 AND published_revision.granularity = r.granularity
 AND published_revision.window_start = r.window_start
 AND published_revision.status = 'published'
 AND published_revision.revision = r.revision
ORDER BY "time" DESC, device_sn ASC, object_ldn ASC%s%s`, candidateSQL, limitSQL, offsetSQL)
	sqlStr, err = sq.Dollar.ReplacePlaceholders(sqlStr)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func applyRawPivotTargetDeviceFilters(devices sq.SelectBuilder, q QueryRequest) sq.SelectBuilder {
	switch {
	case len(q.DeviceSNs) > 0 && len(q.Technologies) > 0:
		devices = devices.Where(sq.Eq{"dev.serial_number": q.DeviceSNs})
		devices = devices.Where(sq.Eq{"dev.technology": q.Technologies})
		if len(q.DeviceOUIs) > 0 {
			devices = devices.Where(sq.Eq{"dev.oui": q.DeviceOUIs})
		}
	case len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0:
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		pairs := make(sq.Or, 0, n)
		for i := 0; i < n; i++ {
			pairs = append(pairs, sq.And{
				sq.Eq{"dev.oui": q.DeviceOUIs[i]},
				sq.Eq{"dev.serial_number": q.DeviceSNs[i]},
			})
		}
		if len(pairs) > 0 {
			devices = devices.Where(pairs)
		}
	case len(q.DeviceOUIs) > 0:
		devices = devices.Where(sq.Eq{"dev.oui": q.DeviceOUIs})
	case len(q.DeviceSNs) > 0:
		devices = devices.Where(sq.Eq{"dev.serial_number": q.DeviceSNs})
	}
	if len(q.Technologies) > 0 && len(q.DeviceSNs) == 0 {
		devices = devices.Where(sq.Eq{"dev.technology": q.Technologies})
	}
	return devices
}

func prefixedColumns(prefix string, cols []string) string {
	out := make([]string, 0, len(cols))
	for _, col := range cols {
		out = append(out, prefix+"."+col)
	}
	return strings.Join(out, ", ")
}

func (a *Aggregator) countDevicePivotRows(ctx context.Context, table string, q QueryRequest) (int, error) {
	if table == "pm_metrics" && IsExplicitObjectSkeletonRequest(q) {
		countReq := q
		countReq.Limit = 0
		countReq.Offset = 0
		keys, err := a.deviceObjectSkeletonPivotRowKeys(ctx, countReq)
		if err != nil {
			return 0, err
		}
		return len(keys), nil
	}
	if table != "pm_metrics" && isDeviceDimension(q.Dimension) {
		countReq := q
		countReq.Limit = 0
		countReq.Offset = 0
		sqlStr, args, err := buildRolledUpDevicePivotRowKeysSQL(countReq)
		if err != nil {
			return 0, fmt.Errorf("aggregator.Count build rolled-up pivot rows: %w", err)
		}
		rows, err := a.db.Query(ctx, sqlStr, args...)
		if err != nil {
			return 0, fmt.Errorf("aggregator.Count query rolled-up pivot rows %s: %w", table, err)
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			count++
		}
		return count, rows.Err()
	}
	var keySub sq.SelectBuilder
	if len(q.MetricPaths) == 0 && len(q.ObjectLDNs) > 0 {
		keySub = newDeviceObjectKeySelect(storage.Psql, q)
	} else {
		keySub = newRawAwareDeviceSelect(
			storage.Psql, table, q,
			"device_oui", "device_sn", "COALESCE(object_ldn, '') AS object_ldn", "granularity", `"time"`,
		).Distinct()
		keySub = applyDeviceFilters(keySub, q)
	}
	return a.scanCountSub(ctx, keySub)
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
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		where = append(where, fmt.Sprintf("EXTRACT(dow FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Weekdays)))
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		where = append(where, fmt.Sprintf("EXTRACT(hour FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Hours)))
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
	if table == "pm_metrics" || (table == "pm_metrics_hourly" && len(q.MetricPaths) > 0) {
		inner := newRawAwareDeviceSelect(storage.Psql, table, q,
			"device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
			"statis_type", "granularity", "time", "start_time", "end_time", "ingest_time", "object_ldn",
		).Options(`DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
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
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		where = append(where, fmt.Sprintf("EXTRACT(dow FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Weekdays)))
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		where = append(where, fmt.Sprintf("EXTRACT(hour FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Hours)))
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
	if len(q.DeviceSNs) > 0 && len(q.Technologies) > 0 {
		if len(q.DeviceOUIs) > 0 {
			qb = qb.Where(sq.Eq{"device_oui": q.DeviceOUIs})
		}
		qb = qb.Where(
			"(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE serial_number = ANY(?) AND technology = ANY(?))",
			q.DeviceSNs,
			q.Technologies,
		)
		q.Technologies = nil
		return applyCommonFilters(qb, q)
	}
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
		if q.MetricType == nil {
			or := sq.Or{}
			for _, raw := range q.MetricPaths {
				path := strings.TrimSpace(raw)
				if path == "" {
					continue
				}
				if mt, ok := metricTypeFromIndicatorPath(path); ok {
					or = append(or, sq.And{
						sq.Eq{"metric_path": path},
						sq.Eq{"metric_type": string(mt)},
					})
				} else {
					or = append(or, sq.Eq{"metric_path": path})
				}
			}
			if len(or) > 0 {
				qb = qb.Where(or)
			}
		} else {
			qb = qb.Where(sq.Eq{"metric_path": q.MetricPaths})
		}
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
	// 按系统时区转成本地钟面后再判断，避免数据库会话时区把 +08 桶误判成 UTC 前一天/前一小时。
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		qb = qb.Where(calendarfilter.ExtractDOWPredicate("start_time"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Weekdays)
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		qb = qb.Where(calendarfilter.ExtractHourPredicate("start_time"),
			calendarfilter.NormalizeName(q.CalendarTimezone), q.Hours)
	}
	// #619：测量对象（object_ldn）后端过滤（空 = 不过滤，向后兼容）。
	if len(q.ObjectLDNs) > 0 {
		qb = qb.Where(sq.Eq{"object_ldn": q.ObjectLDNs})
	}
	return qb
}

func metricTypeFromIndicatorPath(path string) (metrics.MetricType, bool) {
	switch {
	case strings.HasPrefix(strings.ToUpper(strings.TrimSpace(path)), "K"):
		return metrics.MetricTypeKPI, true
	case strings.HasPrefix(strings.ToUpper(strings.TrimSpace(path)), "C"):
		return metrics.MetricTypeCounter, true
	default:
		return "", false
	}
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
