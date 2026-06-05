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
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

// AggregateKPIs 在 target 表内做"二阶段"KPI 聚合（T-B 多粒度：按小区/PLMN 算）：
//  1. SELECT 出 [w.Start, w.End) 桶内 (device_oui, device_sn, object_ldn) DISTINCT 实体——
//     每个 object_ldn（基础小区 / 各 PLMN / 设备级空串）各成一个独立 KPI 实体
//  2. 对每个设备：KPIRouter.LookupByDevice → KPI 列表（同设备多实体共用一次路由查询）
//  3. 对每个实体：按本 object_ldn 精确取计数器，对 PLMN 实体做跨层级配对
//     （合并同 cellID 基础小区行的小区级计数器）→ 自身层级门槛过滤（KPI 公式须引用本实体
//     自身行至少一个计数器，纯小区级 KPI 因此不泄漏到 PLMN 行）→ expr.Evaluate →
//     INSERT 带实际 object_ldn 的 KPI 行
//
// 注意：本函数假设 AggregateCounters 已写入对应桶的 counter 行（否则 KPI 拿不到入参）。
// 单 cron runner 内调用顺序：AggregateCounters → AggregateKPIs。
//
// 返回插入的 KPI 行数。
func (a *Aggregator) AggregateKPIs(ctx context.Context, target string, w WindowSpec) (int, error) {
	if a.kpiRouter == nil {
		return 0, nil
	}
	entities, err := a.listEntitiesInBucket(ctx, target, w)
	if err != nil {
		return 0, fmt.Errorf("aggregator.AggregateKPIs list entities: %w", err)
	}
	if len(entities) == 0 {
		return 0, nil
	}

	// 同设备的计数器（全 object_ldn 行）与 KPI 路由各取一次，跨该设备的多个小区/PLMN 实体复用。
	type devCache struct {
		route       *router.KPIRoute
		byObjectLdn map[string]map[string]float64 // object_ldn → metric_path → value
		loaded      bool
		skip        bool
	}
	caches := map[deviceKey]*devCache{}

	total := 0
	for _, ent := range entities {
		dk := deviceKey{ent.oui, ent.sn}
		c := caches[dk]
		if c == nil {
			c = &devCache{}
			caches[dk] = c
			route, err := a.kpiRouter.LookupByDevice(ctx, ent.sn)
			if err != nil {
				// 单设备失败不阻塞整体（orphan / 元数据残缺都是稳定状态，记 WARN 继续）。
				a.logger.Warn("kpi route lookup failed; skip device",
					zap.String("device_oui", ent.oui), zap.String("device_sn", ent.sn),
					zap.Error(err))
				c.skip = true
			} else if route == nil || len(route.KPIs) == 0 {
				c.skip = true
			} else {
				c.route = route
				byLdn, err := a.loadCountersByObjectLdn(ctx, target, ent.oui, ent.sn, w)
				if err != nil {
					a.logger.Warn("load counters failed; skip device",
						zap.String("device_oui", ent.oui), zap.String("device_sn", ent.sn),
						zap.Error(err))
					c.skip = true
				} else {
					c.byObjectLdn = byLdn
					c.loaded = true
				}
			}
		}
		if c.skip || !c.loaded {
			continue
		}

		counters := countersForEntity(c.byObjectLdn, ent.objectLdn)
		// 自身层级门槛：实体本行（未经跨层级合并）的计数器集合。
		// 仅当 KPI 公式依赖与该集合有交集时才在本实体落库——纯小区级 KPI 不引用任何 PLMN
		// 自身计数 → 在 PLMN 实体门槛不过 → 不落（修掉「device 级 KPI 泄漏到 PLMN 行」）。
		ownSet := c.byObjectLdn[ent.objectLdn]
		written, err := a.evalAndInsertKPIs(ctx, target, ent, w, c.route.KPIs, counters, ownSet)
		if err != nil {
			a.logger.Warn("eval/insert kpis failed; skip entity",
				zap.String("device_oui", ent.oui), zap.String("device_sn", ent.sn),
				zap.String("object_ldn", ent.objectLdn), zap.Error(err))
			continue
		}
		total += written
	}
	return total, nil
}

// ── 内部 helper ───────────────────────────────────────────────────────────

type deviceKey struct{ oui, sn string }

// entityKey 是 T-B KPI 聚合的最小实体粒度：设备 + 小区/PLMN（object_ldn）。
// object_ldn=="" 即设备级实体（与 T-A 前行为一致，KPI 行 object_ldn 仍写空串）。
type entityKey struct{ oui, sn, objectLdn string }

func (a *Aggregator) listEntitiesInBucket(ctx context.Context, target string, w WindowSpec) ([]entityKey, error) {
	sql, args := buildListEntitiesInBucketSQL(target, w)
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entityKey
	for rows.Next() {
		var e entityKey
		if err := rows.Scan(&e.oui, &e.sn, &e.objectLdn); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// loadCountersByObjectLdn 取某设备本桶各 object_ldn 行的 counter 值，
// 按 object_ldn 分层返回 object_ldn → metric_path → value（不再折回设备级）。
//
// T-A 后 target 表每 metric_path 按 object_ldn 分多行；T-B 按实体取本行计数器，
// 故这里以 (object_ldn, metric_path) 为粒度精确带出，跨层级配对在 countersForEntity 完成。
func (a *Aggregator) loadCountersByObjectLdn(ctx context.Context, target, oui, sn string, w WindowSpec) (map[string]map[string]float64, error) {
	sql, args := buildLoadCountersByObjectLdnSQL(target, oui, sn, w)
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]map[string]float64)
	for rows.Next() {
		var objectLdn, name string
		var val float64
		if err := rows.Scan(&objectLdn, &name, &val); err != nil {
			return nil, err
		}
		m := out[objectLdn]
		if m == nil {
			m = make(map[string]float64)
			out[objectLdn] = m
		}
		m[name] = val
	}
	return out, rows.Err()
}

// countersForEntity 组装某实体（object_ldn）算 KPI 的入参：
//   - 设备级实体（object_ldn==""）：取空串行的计数器（无配对）。
//   - 基础小区实体（Cellid=N，无 PLMN）：取本行小区级计数器（无需配 PLMN）。
//   - PLMN 实体（Cellid=N,PLMN=M）：取本 PLMN 行的 PLMN 级计数器
//     ∪ 同 cellID=N 基础小区行的小区级计数器（跨层级配对）。
//
// 两类计数器 metric_path 互斥（真机实测重叠 0），合并不撞键；防御上让实体自身的值优先。
// NR/空小区实体（cellID 提取为空）不 panic：plmn=="" 时按基础/设备级处理，仅返回本行计数器。
func countersForEntity(byObjectLdn map[string]map[string]float64, objectLdn string) map[string]float64 {
	out := make(map[string]float64)

	cellID, plmn := metrics.ParseObjectLDN(objectLdn)
	// PLMN 实体先铺基础小区行的小区级计数器（再被本行覆盖，确保自身优先）。
	if plmn != "" && cellID != "" {
		for baseLdn, vals := range byObjectLdn {
			bCell, bPlmn := metrics.ParseObjectLDN(baseLdn)
			if bPlmn == "" && bCell == cellID {
				for k, v := range vals {
					out[k] = v
				}
			}
		}
	}
	// 本实体自身行计数器（优先级最高）。
	for k, v := range byObjectLdn[objectLdn] {
		out[k] = v
	}
	return out
}

// kpiDependsOnOwnCounters 判定一个 KPI 是否「引用了该实体自身层级（本行未经跨层级合并）
// 的至少一个计数器」——自身层级门槛的判据。
//
//   - 用 router.KPIDef.Dependencies（从公式 arithmetic 提取的计数器编号清单，与公式同源）
//     与实体自身行计数器 ownSet 求交集，非空即门槛过。
//   - 纯小区级 KPI 在 PLMN 实体：依赖全在基础小区行、不在 PLMN 自身集 → 交集空 → 门槛不过 →
//     不在 PLMN 落库（跨层级配对的合并仅用于给混合公式补小区级入参，不让纯小区级 KPI 现身 PLMN）。
//   - 混合公式（同时引用小区级 + PLMN 级）在 PLMN 实体：引用了 PLMN 自身计数 → 门槛过 → 落库，
//     并经配对补到小区级入参算出正确值。
//   - 设备级实体（object_ldn=''）ownSet 即空串行计数器，行为不变。
//   - Dependencies 为空（无法判定层级归属）→ 保守放行，保持原行为不被门槛误杀。
func kpiDependsOnOwnCounters(k router.KPIDef, ownSet map[string]float64) bool {
	if len(k.Dependencies) == 0 {
		return true
	}
	for _, dep := range k.Dependencies {
		if _, ok := ownSet[dep]; ok {
			return true
		}
	}
	return false
}

func (a *Aggregator) evalAndInsertKPIs(
	ctx context.Context,
	target string,
	ent entityKey,
	w WindowSpec,
	kpis []router.KPIDef,
	counters map[string]float64,
	ownSet map[string]float64,
) (int, error) {
	var rows []kpiRow
	for _, k := range kpis {
		if k.Formula == "" {
			continue
		}
		// 自身层级门槛：公式必须引用本实体自身层级至少一个计数器才在此实体落库
		// （否则纯小区级 KPI 会因跨层级合并的 map 里依赖齐全而泄漏到 PLMN 行）。
		if !kpiDependsOnOwnCounters(k, ownSet) {
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

	sql, args := buildKPIInsertSQL(target, ent, w, rows)
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
//   - object_ldn（小区/PLMN）进 GROUP BY 尾部，基础小区/各 PLMN 各自成行、各自聚合，
//     不再用 MIN 取代表值拍平（T-A 物化保留小区/PLMN 维度）
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
    m.object_ldn,
    NULL::jsonb
FROM %s m
WHERE m.metric_type = 'counter'
  AND m.end_time >= $4
  AND m.end_time <  $5
  AND m.statis_type IN ('sum','avg','max','min')
GROUP BY m.device_oui, m.device_sn, m.metric_path, m.statis_type, m.object_ldn
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

// buildListEntitiesInBucketSQL 构造"列出本桶有计数的实体（设备 + 小区/PLMN）" SELECT。
//
// 读的是 target 表（每行即一个已聚合完整桶，桶尾时刻 end_time = w.End），
// 因此必须**精确命中本桶**（end_time = w.End），而非源表式半开区间扫描。
// 叠加 time = w.Start（hourly 表 PK 含 time，daily/weekly/monthly 该列恒为桶起点）
// 既无害又自证锁定到唯一本桶；配合 granularity = $1 唯一定位。
//
// T-B：DISTINCT 维加 object_ldn——基础小区 / 各 PLMN / 设备级空串各成一个独立 KPI 实体。
//
// 参数顺序：$1=granularity, $2=bucket_end(=w.End), $3=bucket_start(=w.Start)
func buildListEntitiesInBucketSQL(target string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT DISTINCT device_oui, device_sn, object_ldn
FROM %s
WHERE metric_type = 'counter'
  AND granularity = $1
  AND end_time = $2
  AND time = $3`, target)
	return sql, []any{string(w.Granularity), w.End, w.Start}
}

// buildLoadCountersByObjectLdnSQL 构造"取某设备本桶各 object_ldn 行的 counter 值"SELECT。
//
// 与 buildListEntitiesInBucketSQL 同理：读 target 表必须精确命中本桶
// （end_time = w.End），不能用半开区间——否则会命中上一桶（其 end_time = 本桶 w.Start）。
//
// 【T-B 关键】：拆掉 T-A 的"折回设备级"层——不再 GROUP BY metric_path 把多小区相加，
// 而是按 (object_ldn, metric_path) 精确带出每行计数器（T-A 后该表每对组合恰一行），
// 交由 countersForEntity 按实体取本行 + 对 PLMN 实体做跨层级配对。
// 这从根上杜绝了"把基础小区 + 各 PLMN 计数相加折回设备级"导致的跨小区串味/重复叠加。
//
// 参数顺序：$1=oui, $2=sn, $3=granularity, $4=bucket_end(=w.End), $5=bucket_start(=w.Start)
func buildLoadCountersByObjectLdnSQL(target, oui, sn string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT object_ldn, metric_path, metric_value
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
//
// T-B：object_ldn 由硬写 '' 改为写实体实际 object_ldn（设备级实体 object_ldn=='' 仍写 ''），
// 让 KPI 按小区/PLMN 落库分行下钻。复用 T-A 的唯一键（已含 object_ldn），同桶重跑幂等覆盖。
func buildKPIInsertSQL(target string, ent entityKey, w WindowSpec, rows []kpiRow) (string, []any) {
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
		oui := add(ent.oui)
		sn := add(ent.sn)
		path := add(r.path)
		val := add(r.value)
		stype := add(r.stype)
		gran := add(string(w.Granularity))
		bktStart := add(w.Start)
		bktEnd := add(w.End)
		objectLdn := add(ent.objectLdn)
		// object_ldn 收紧为 NOT NULL DEFAULT '' 后不能再写 NULL；写实体实际 object_ldn
		// （设备级实体为空串），KPI 按小区/PLMN 分行落库。
		core := fmt.Sprintf("(%s, %s, %s, 'kpi', %s, %s, %s, %s, %s, %s, NOW(), %s, NULL::jsonb)",
			oui, sn, path, val, stype, gran, bktStart, bktStart, bktEnd, objectLdn)
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
//   - hourly 用 UNIQUE INDEX (oui, sn, metric_path, granularity, end_time, time, object_ldn)
//   - daily/weekly/monthly 用 PRIMARY KEY (oui, sn, metric_path, granularity, end_time, object_ldn)
//   - group_hourly 用 UNIQUE (device_group_id, metric_path, granularity, end_time, time, technology)
//   - group_daily/weekly/monthly 用 PRIMARY KEY (device_group_id, metric_path, granularity, end_time, technology)
//
// 设备级四表的冲突列尾部含 object_ldn（小区/PLMN）—— 与迁移 000020 的唯一键改动配套，
// 让同设备同指标同桶按小区/PLMN 分多行落库、各自 UPSERT。
// device_group 四表冲突列尾部含 technology —— 与迁移 000026 的唯一键改动配套（设备组制式治本 B 方案），
// 让同组同指标同桶按制式分多行落库、各自 UPSERT；列序必须与迁移唯一键逐字一致。
func conflictTargetForTable(target string) string {
	switch target {
	case "pm_metrics_hourly":
		return "(device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)"
	case "pm_metrics_daily", "pm_metrics_weekly", "pm_metrics_monthly":
		return "(device_oui, device_sn, metric_path, granularity, end_time, object_ldn)"
	case "pm_group_metrics_hourly":
		return "(device_group_id, metric_path, granularity, end_time, time, technology)"
	case "pm_group_metrics_daily", "pm_group_metrics_weekly", "pm_group_metrics_monthly":
		return "(device_group_id, metric_path, granularity, end_time, technology)"
	}
	return "(device_oui, device_sn, metric_path, granularity, end_time, object_ldn)"
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
