// Package aggregator 实现 T-0164-P5 / G5 自然日历桶预聚合。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.5
// 实施 plan：docs/project/plan-T-0164-P5-natural-bucket-aggregation.md
//
// 流水线：pm_metrics (15min) ──hourly──▶ pm_metrics_hourly
//                                       ──daily ──▶ pm_metrics_daily
//                                                  ──weekly──▶ pm_metrics_weekly
//                                                             ──monthly──▶ pm_metrics_monthly
//
// 每个 cron runner 内做两步：
//
//  1. AggregateCounters：把上一级粒度的 counter 行按 statis_type（sum/avg/max）GROUP BY 聚合
//     直接落到目标表（SQL 单批，CASE WHEN 路由聚合方式）。
//  2. AggregateKPIs：扫目标表的 counter 行 → 按 device 拉 KPI 路由 → arithmetic 求值
//     → INSERT metric_type='kpi' 的行（一次/设备一次/KPI）。
//
// 与 pm_metrics 写入幂等一致：所有 INSERT 走 ON CONFLICT DO UPDATE，cron 重跑安全。
package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// WindowSpec 描述一个自然桶聚合窗口。
//   - Granularity：目标粒度（hourly / daily / weekly / monthly）
//   - Start, End：源数据时间区间（半开区间 [Start, End)），自然桶对齐
//     例：hourly 桶 [10:00, 11:00)；daily 桶 [00:00, 24:00)
//
// cron 调度器负责按整点对齐生成 Start/End；Aggregator 不再做时间对齐。
type WindowSpec struct {
	Granularity metrics.Granularity
	Start, End  time.Time
}

// KPIRouter 是 Aggregator 计算 KPI 时的路由依赖（取设备所属产品声明的 KPI 子集）。
//
// 真实实现是 *router.Router；测试可注入 stub 返回固定路由。
type KPIRouter interface {
	LookupByDevice(ctx context.Context, deviceSN string) (*router.KPIRoute, error)
}

// PgQuerier 是 Aggregator 所需的最小 pgxpool 子集，便于单测 stub。
// 真实实现是 *pgxpool.Pool。
type PgQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Aggregator 是 G5 聚合主入口；线程安全，可被多 cron runner 共享。
type Aggregator struct {
	db        PgQuerier
	kpiRouter KPIRouter
	logger    *zap.Logger
}

// New 构造 Aggregator。kpiRouter 可为 nil（表示禁用 KPI 聚合）。
func New(db PgQuerier, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Aggregator{db: db, kpiRouter: kpiRouter, logger: logger.Named("pm.aggregator")}
}

// NewWithPool 便利构造器（外部传 *pgxpool.Pool 时省去接口断言）。
func NewWithPool(pool *pgxpool.Pool, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	return New(pool, kpiRouter, logger)
}

// AggregateCounters 把 source 表中 [w.Start, w.End) 内的 counter 行按
// (device_oui, device_sn, metric_path, statis_type) GROUP BY 聚合写入 target 表。
//
// 路由规则（CASE WHEN m.statis_type）：
//   - 'sum' → SUM(metric_value)
//   - 'avg' → AVG(metric_value)
//   - 'max' → MAX(metric_value)
//   - 其余（pct / NULL）跳过（KPI 走 AggregateKPIs；NULL counter 视为脏数据丢弃）
//
// 返回写入行数（含 UPSERT 冲突更新）。
func (a *Aggregator) AggregateCounters(ctx context.Context, source, target string, w WindowSpec) (int, error) {
	sql, args := buildCountersSQL(source, target, w)
	tag, err := a.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("aggregator.AggregateCounters %s→%s: %w", source, target, err)
	}
	return int(tag.RowsAffected()), nil
}

// AggregateKPIs 在 target 表内做"二阶段"KPI 聚合：
//  1. SELECT 出 [w.Start, w.End) 桶内 (device_oui, device_sn) DISTINCT 设备
//  2. 对每个设备：KPIRouter.LookupByDevice → KPI 列表
//  3. 对每个 KPI：从 target 表拉公式依赖的 counter 值 → expr.Evaluate → INSERT KPI 行
//
// 注意：本函数假设 AggregateCounters 已写入对应桶的 counter 行（否则 KPI 拿不到入参）。
// 单 cron runner 内调用顺序：AggregateCounters → AggregateKPIs。
//
// 返回插入的 KPI 行数。
func (a *Aggregator) AggregateKPIs(ctx context.Context, target string, w WindowSpec) (int, error) {
	if a.kpiRouter == nil {
		return 0, nil
	}
	devices, err := a.listDevicesInBucket(ctx, target, w)
	if err != nil {
		return 0, fmt.Errorf("aggregator.AggregateKPIs list devices: %w", err)
	}
	if len(devices) == 0 {
		return 0, nil
	}

	total := 0
	for _, dev := range devices {
		route, err := a.kpiRouter.LookupByDevice(ctx, dev.sn)
		if err != nil {
			// 单设备失败不阻塞整体（orphan / 元数据残缺都是稳定状态，记 WARN 继续）。
			a.logger.Warn("kpi route lookup failed; skip device",
				zap.String("device_oui", dev.oui), zap.String("device_sn", dev.sn),
				zap.Error(err))
			continue
		}
		if route == nil || len(route.KPIs) == 0 {
			continue
		}
		counterValues, err := a.loadCountersForDevice(ctx, target, dev.oui, dev.sn, w)
		if err != nil {
			a.logger.Warn("load counters failed; skip device",
				zap.String("device_oui", dev.oui), zap.String("device_sn", dev.sn),
				zap.Error(err))
			continue
		}
		written, err := a.evalAndInsertKPIs(ctx, target, dev, w, route.KPIs, counterValues)
		if err != nil {
			a.logger.Warn("eval/insert kpis failed; skip device",
				zap.String("device_oui", dev.oui), zap.String("device_sn", dev.sn),
				zap.Error(err))
			continue
		}
		total += written
	}
	return total, nil
}

// ── 内部 helper ───────────────────────────────────────────────────────────

type deviceKey struct{ oui, sn string }

func (a *Aggregator) listDevicesInBucket(ctx context.Context, target string, w WindowSpec) ([]deviceKey, error) {
	sql, args := buildListDevicesInBucketSQL(target, w)
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []deviceKey
	for rows.Next() {
		var d deviceKey
		if err := rows.Scan(&d.oui, &d.sn); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (a *Aggregator) loadCountersForDevice(ctx context.Context, target, oui, sn string, w WindowSpec) (map[string]float64, error) {
	sql, args := buildLoadCountersForDeviceSQL(target, oui, sn, w)
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err != nil {
			return nil, err
		}
		out[name] = val
	}
	return out, rows.Err()
}

func (a *Aggregator) evalAndInsertKPIs(
	ctx context.Context,
	target string,
	dev deviceKey,
	w WindowSpec,
	kpis []router.KPIDef,
	counters map[string]float64,
) (int, error) {
	var rows []kpiRow
	for _, k := range kpis {
		if k.Formula == "" {
			continue
		}
		f, err := expr.Parse(k.Formula)
		if err != nil {
			a.logger.Warn("kpi formula parse failed; skip",
				zap.String("kpi", k.Name), zap.String("formula", k.Formula), zap.Error(err))
			continue
		}
		val, err := f.Evaluate(counters)
		if err != nil {
			// 公式依赖某 counter 不在 counters 表（设备未上报 / 未聚合）— 跳过该 KPI 不报错
			continue
		}
		statis := k.StatisType
		if statis == "" {
			statis = string(metrics.StatisPct) // KPI 缺省按 pct 写
		}
		rows = append(rows, kpiRow{path: k.IndicatorID, value: val, stype: statis})
	}
	if len(rows) == 0 {
		return 0, nil
	}

	sql, args := buildKPIInsertSQL(target, dev, w, rows)
	tag, err := a.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ── 纯 SQL builder（便于 TDD 单测，不依赖 DB）─────────────────────────────

// buildCountersSQL 构造 source→target 的 GROUP BY 聚合 INSERT。
// 路由规则：
//   - sum / avg / max 三路 CASE WHEN
//   - GROUP BY 包含 statis_type 防止同 metric_path 但 statis 不同时合并冲突
//   - 时间列 time/start_time/end_time 全部由 cron 注入的 w.Start/w.End 决定（桶整对齐）
//   - object_ldn 用 MIN 取代表值（同 GROUP BY 内业务上应一致）
//
// 目标表 ON CONFLICT 分两种：
//   - hourly：UNIQUE (oui, sn, metric_path, granularity, end_time, time)
//   - daily/weekly/monthly：PRIMARY KEY (oui, sn, metric_path, granularity, end_time)
//     time 列恒等于 w.Start（桶起点），两种 conflict target 在业务上唯一。
func buildCountersSQL(source, target string, w WindowSpec) (string, []any) {
	conflictTarget := conflictTargetForTable(target)
	withID := targetHasIDColumn(target)

	// SELECT 列：hourly 表 PK 含 id，所以列名前加 id；其余表直接进 13 列。
	insertCols := "device_oui, device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time, ingest_time, object_ldn, extra"
	selectIDExpr := ""
	if withID {
		insertCols = "id, " + insertCols
		selectIDExpr = "gen_random_uuid(),\n    "
	}

	// 参数顺序：$1=granularity, $2=bucket_start, $3=bucket_end, $4=where_start, $5=where_end
	// bucket_start/end 等于 where_start/end（cron 同步驱动），但用独立位允许未来错峰回填。
	sql := fmt.Sprintf(`
INSERT INTO %s (%s)
SELECT
    %sm.device_oui,
    m.device_sn,
    m.metric_path,
    'counter',
    CASE m.statis_type
        WHEN 'sum' THEN SUM(m.metric_value)
        WHEN 'avg' THEN AVG(m.metric_value)
        WHEN 'max' THEN MAX(m.metric_value)
        WHEN 'min' THEN MIN(m.metric_value)
    END,
    m.statis_type,
    $1,
    $2,
    $2,
    $3,
    NOW(),
    MIN(m.object_ldn),
    NULL::jsonb
FROM %s m
WHERE m.metric_type = 'counter'
  AND m.end_time >= $4
  AND m.end_time <  $5
  AND m.statis_type IN ('sum','avg','max','min')
GROUP BY m.device_oui, m.device_sn, m.metric_path, m.statis_type
ON CONFLICT %s DO UPDATE SET
    metric_value = EXCLUDED.metric_value,
    ingest_time  = NOW()`,
		target, insertCols,
		selectIDExpr,
		source,
		conflictTarget,
	)
	return sql, []any{string(w.Granularity), w.Start, w.End, w.Start, w.End}
}

// buildListDevicesInBucketSQL 构造"列出本桶有计数的设备" SELECT。
//
// 读的是 target 表（每行即一个已聚合完整桶，桶尾时刻 end_time = w.End），
// 因此必须**精确命中本桶**（end_time = w.End），而非源表式半开区间扫描。
// 叠加 time = w.Start（hourly 表 PK 含 time，daily/weekly/monthly 该列恒为桶起点）
// 既无害又自证锁定到唯一本桶；配合 granularity = $1 唯一定位。
//
// 参数顺序：$1=granularity, $2=bucket_end(=w.End), $3=bucket_start(=w.Start)
func buildListDevicesInBucketSQL(target string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT DISTINCT device_oui, device_sn
FROM %s
WHERE metric_type = 'counter'
  AND granularity = $1
  AND end_time = $2
  AND time = $3`, target)
	return sql, []any{string(w.Granularity), w.End, w.Start}
}

// buildLoadCountersForDeviceSQL 构造"取某设备本桶各 counter 值" SELECT。
//
// 与 buildListDevicesInBucketSQL 同理：读 target 表必须精确命中本桶
// （end_time = w.End），不能用半开区间——否则会命中上一桶（其 end_time = 本桶 w.Start）。
//
// 参数顺序：$1=oui, $2=sn, $3=granularity, $4=bucket_end(=w.End), $5=bucket_start(=w.Start)
func buildLoadCountersForDeviceSQL(target, oui, sn string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT metric_path, metric_value
FROM %s
WHERE device_oui = $1
  AND device_sn  = $2
  AND metric_type = 'counter'
  AND granularity = $3
  AND end_time = $4
  AND time = $5`, target)
	return sql, []any{oui, sn, string(w.Granularity), w.End, w.Start}
}

// buildKPIInsertSQL 给 target 表构造批量 KPI INSERT。
// 与 buildCountersSQL 不同，KPI 行是 Go 端 evaluate 后逐行插入；这里复用同样的桶时间列设计。
func buildKPIInsertSQL(target string, dev deviceKey, w WindowSpec, rows []kpiRow) (string, []any) {
	conflictTarget := conflictTargetForTable(target)
	withID := targetHasIDColumn(target)

	columnList := "device_oui, device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time, ingest_time, object_ldn, extra"
	if withID {
		columnList = "id, " + columnList
	}

	valueRows := make([]string, 0, len(rows))
	args := []any{}
	pos := 1
	add := func(v any) string {
		args = append(args, v)
		p := fmt.Sprintf("$%d", pos)
		pos++
		return p
	}

	for _, r := range rows {
		oui := add(dev.oui)
		sn := add(dev.sn)
		path := add(r.path)
		val := add(r.value)
		stype := add(r.stype)
		gran := add(string(w.Granularity))
		bktStart := add(w.Start)
		bktEnd := add(w.End)
		core := fmt.Sprintf("(%s, %s, %s, 'kpi', %s, %s, %s, %s, %s, %s, NOW(), NULL, NULL::jsonb)",
			oui, sn, path, val, stype, gran, bktStart, bktStart, bktEnd)
		if withID {
			core = "(gen_random_uuid(), " + core[1:]
		}
		valueRows = append(valueRows, core)
	}

	sql := fmt.Sprintf(`
INSERT INTO %s (%s)
VALUES %s
ON CONFLICT %s DO UPDATE SET
    metric_value = EXCLUDED.metric_value,
    ingest_time  = NOW()`,
		target, columnList,
		joinComma(valueRows),
		conflictTarget,
	)
	return sql, args
}

// ── 表名路由 / 冲突列 ──────────────────────────────────────────────────────

// conflictTargetForTable 返回 ON CONFLICT 的列清单。
//   - hourly 用 UNIQUE INDEX (oui, sn, metric_path, granularity, end_time, time)
//   - daily/weekly/monthly 用 PRIMARY KEY (oui, sn, metric_path, granularity, end_time)
//   - group_hourly 用 UNIQUE (device_group_id, metric_path, granularity, end_time, time)
//   - group_daily/weekly/monthly 用 PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
func conflictTargetForTable(target string) string {
	switch target {
	case "pm_metrics_hourly":
		return "(device_oui, device_sn, metric_path, granularity, end_time, time)"
	case "pm_metrics_daily", "pm_metrics_weekly", "pm_metrics_monthly":
		return "(device_oui, device_sn, metric_path, granularity, end_time)"
	case "pm_group_metrics_hourly":
		return "(device_group_id, metric_path, granularity, end_time, time)"
	case "pm_group_metrics_daily", "pm_group_metrics_weekly", "pm_group_metrics_monthly":
		return "(device_group_id, metric_path, granularity, end_time)"
	}
	return "(device_oui, device_sn, metric_path, granularity, end_time)"
}

// targetHasIDColumn hourly 表（hypertable 要求 PK 含分区列 time，故保留 id 列）。
func targetHasIDColumn(target string) bool {
	return target == "pm_metrics_hourly" || target == "pm_group_metrics_hourly"
}

// joinComma 用 ",\n" 拼接 VALUES 行。
func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ",\n"
		}
		out += p
	}
	return out
}

// kpiRow 是 evalAndInsertKPIs / 测试用的 KPI 求值结果。
type kpiRow struct {
	path  string
	value float64
	stype string
}
