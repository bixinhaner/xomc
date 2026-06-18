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
//
// 双库说明（KPI/时序库物理分离，#532 P2）：
//   - db：时序库（TsPool）——pm_metrics_* 聚合源表 + perf_indicators_*（两库都有副本）。
//   - metaDB：主库（PgPool）——控制面元数据，含 enabled_pm_indicators_*（仅主库有，时序库无）。
//
// resolveEnabledIndicators 必须走 metaDB（启用集表只在主库）；其余聚合/重算走 db。
// metaDB 为 nil（旧构造 / 单测）时退回 db，保持向后兼容（旧路径不读 enabled_pm_indicators_*）。
type Aggregator struct {
	db        PgQuerier
	metaDB    PgQuerier
	kpiRouter KPIRouter
	logger    *zap.Logger
}

// New 构造 Aggregator。kpiRouter 可为 nil（表示禁用 KPI 聚合）。
// metaDB 缺省退回 db（见 NewWithMeta / NewWithPools）。
func New(db PgQuerier, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	return NewWithMeta(db, nil, kpiRouter, logger)
}

// NewWithMeta 构造 Aggregator 并显式注入控制面元数据库 metaDB（主库）。
// metaDB 为 nil 时退回 db（向后兼容旧调用 / 单测）。
func NewWithMeta(db, metaDB PgQuerier, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	if logger == nil {
		logger = zap.NewNop()
	}
	if metaDB == nil {
		metaDB = db
	}
	return &Aggregator{db: db, metaDB: metaDB, kpiRouter: kpiRouter, logger: logger.Named("pm.aggregator")}
}

// NewWithPool 便利构造器（外部传 *pgxpool.Pool 时省去接口断言）。
// 不注入 metaDB（metaDB 退回 db）——仅用于不需要读 enabled_pm_indicators_* 的场景（如纯 cron 聚合）。
func NewWithPool(pool *pgxpool.Pool, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	return New(pool, kpiRouter, logger)
}

// NewWithPools 便利构造器：tsPool=时序库（聚合源 + perf_indicators_*）、pgPool=主库（enabled_pm_indicators_*）。
// 落库侧全存已启用（#532 P2）必须用此构造，否则 resolveEnabledIndicators 在时序库查 enabled_pm_indicators_* 报 relation 不存在。
func NewWithPools(tsPool, pgPool *pgxpool.Pool, kpiRouter KPIRouter, logger *zap.Logger) *Aggregator {
	return NewWithMeta(tsPool, pgPool, kpiRouter, logger)
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

// loadCountersBatchSize 是第二段"整桶载入计数器"的分设备批大小（#516 N+1 批量化）。
//
// 整桶一次性载入会把全桶所有设备的全部 counter 行一次吞进内存，10 万级设备桶可能 GB 量级；
// 故按设备分批：每批取 N 个设备的全部 object_ldn 计数器一次查询载入，处理完该批的 KPI 后释放，
// 再载下一批——往返次数从"逐设备一次（上千次）"降到"每批一次（设备数/N 次）"，同时内存只驻留单批。
// 200 是经验阈值（单批 ≈ 数百设备 × 上千 metric_path，内存可控、往返足够少）；非性能死判，可调。
const loadCountersBatchSize = 200

// kpiInsertBatchSize 是第二段"批量写 KPI"的单条 INSERT 最大行数（#516 N+1 批量化）。
//
// 全桶 KPI 行累积后批量写替代逐实体单条 INSERT；但单条多行 VALUES 受 PG 参数上限（65535）约束，
// 每行 9 个占位符 → 上限约 7000 行/条，取 1000 留足余量并控单条 SQL 体积。非性能死判，可调。
const kpiInsertBatchSize = 1000

// AggregateKPIs 在 target 表内做"二阶段"KPI 聚合（T-B 多粒度：按小区/PLMN 算）：
//  1. SELECT 出 [w.Start, w.End) 桶内 (device_oui, device_sn, object_ldn) DISTINCT 实体——
//     每个 object_ldn（基础小区 / 各 PLMN / 设备级空串）各成一个独立 KPI 实体
//  2. #516 批量化：把"逐设备一次取计数器"改为"按设备分批整桶载入"——每批一次查询拿该批
//     全设备全 object_ldn 的 counter，在内存按 设备→实体→指标 组织（分批控内存，避免一次性
//     吞整桶）；KPI 路由仍按设备 LookupByDevice（已缓存则复用，不重复查）
//  3. 对每个实体：按本 object_ldn 精确取计数器，对 PLMN 实体做跨层级配对
//     （合并同 cellID 基础小区行的小区级计数器）→ 自身层级门槛过滤（KPI 公式须引用本实体
//     自身行至少一个计数器，纯小区级 KPI 因此不泄漏到 PLMN 行）→ expr.Evaluate →
//     KPI 行累积
//  4. #516 批量化：全桶 KPI 行累积后**批量插入**（单条多行 VALUES，按 kpiInsertBatchSize 分条），
//     替代逐实体单条 INSERT（往返从"每实体一次（上万次）"降到"每千行一次"）
//
// KPI 公式求值仍在内存逐实体算（无法下推 SQL，保持不动）。
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

	// 按设备聚合实体，便于"取一次路由 + 整桶分批载计数器"在设备粒度复用。
	// devOrder 保持首次出现顺序，让批划分与产出稳定（测试可断言、行集合稳定）。
	devEntities := map[deviceKey][]entityKey{}
	var devOrder []deviceKey
	for _, ent := range entities {
		dk := deviceKey{ent.oui, ent.sn}
		if _, seen := devEntities[dk]; !seen {
			devOrder = append(devOrder, dk)
		}
		devEntities[dk] = append(devEntities[dk], ent)
	}

	// 累积全桶 KPI 行，最后批量写（#516）。每行带自身实体（落库需写实体 object_ldn）。
	var pending []entityRow

	// 按设备分批：每批整桶载入该批全设备的计数器（一次查询），算完即释放该批内存。
	for start := 0; start < len(devOrder); start += loadCountersBatchSize {
		end := start + loadCountersBatchSize
		if end > len(devOrder) {
			end = len(devOrder)
		}
		batchDevs := devOrder[start:end]

		// 整桶（本批设备）一次性载入：deviceKey → object_ldn → metric_path → value。
		byDevice, err := a.loadCountersForDevices(ctx, target, batchDevs, w)
		if err != nil {
			// 整批载入失败：本批所有设备跳过不阻塞其余批（记 WARN 继续，沿用单设备失败语义）。
			a.logger.Warn("batch load counters failed; skip device batch",
				zap.Int("batch_devices", len(batchDevs)), zap.Error(err))
			continue
		}

		for _, dk := range batchDevs {
			// KPI 路由按设备取一次（缺失/空 → 跳过整设备，沿用 WARN 跳过语义）。
			route, err := a.kpiRouter.LookupByDevice(ctx, dk.sn)
			if err != nil {
				a.logger.Warn("kpi route lookup failed; skip device",
					zap.String("device_oui", dk.oui), zap.String("device_sn", dk.sn),
					zap.Error(err))
				continue
			}
			if route == nil || len(route.KPIs) == 0 {
				continue
			}
			byObjectLdn := byDevice[dk]
			if byObjectLdn == nil {
				// 该设备列出了实体但整桶载入没拿到其计数器（数据态/竞态）→ 跳过不阻塞。
				a.logger.Warn("counters missing for listed device; skip device",
					zap.String("device_oui", dk.oui), zap.String("device_sn", dk.sn))
				continue
			}

			for _, ent := range devEntities[dk] {
				counters := countersForEntity(byObjectLdn, ent.objectLdn)
				// 自身层级门槛：实体本行（未经跨层级合并）的计数器集合。
				// 仅当 KPI 公式依赖与该集合有交集时才在本实体落库——纯小区级 KPI 不引用任何
				// PLMN 自身计数 → 在 PLMN 实体门槛不过 → 不落（修掉「device 级 KPI 泄漏到 PLMN 行」）。
				ownSet := byObjectLdn[ent.objectLdn]
				rows := a.evalKPIs(ent, route.KPIs, counters, ownSet)
				for _, r := range rows {
					pending = append(pending, entityRow{ent: ent, row: r})
				}
			}
		}
	}

	if len(pending) == 0 {
		return 0, nil
	}
	return a.batchInsertKPIs(ctx, target, w, pending)
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
//
// #516：单设备版保留（仅供边界单测验"不串相邻桶"用）；AggregateKPIs 已改走整桶分批的
// loadCountersForDevices，不再逐设备调用本函数。
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

// loadCountersForDevices 整桶一次性载入一批设备的全部 counter 行（#516 N+1 批量化）：
// 单条 SQL 用 (device_oui, device_sn) IN (...) 把该批所有设备的本桶 counter 一次拉回，
// 在内存按 deviceKey → object_ldn → metric_path → value 组织——把"逐设备一次查询"折成"每批一次"。
//
// devs 为本批设备（调用方按 loadCountersBatchSize 切批，控内存只驻留单批）；空批返回空 map。
func (a *Aggregator) loadCountersForDevices(ctx context.Context, target string, devs []deviceKey, w WindowSpec) (map[deviceKey]map[string]map[string]float64, error) {
	out := make(map[deviceKey]map[string]map[string]float64, len(devs))
	if len(devs) == 0 {
		return out, nil
	}
	sql, args := buildLoadCountersForDevicesSQL(target, devs, w)
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var oui, sn, objectLdn, name string
		var val float64
		if err := rows.Scan(&oui, &sn, &objectLdn, &name, &val); err != nil {
			return nil, err
		}
		dk := deviceKey{oui, sn}
		byLdn := out[dk]
		if byLdn == nil {
			byLdn = make(map[string]map[string]float64)
			out[dk] = byLdn
		}
		m := byLdn[objectLdn]
		if m == nil {
			m = make(map[string]float64)
			byLdn[objectLdn] = m
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

// entityRow 把一条求值后的 KPI 行与其所属实体绑定，供全桶累积后批量写（#516）。
type entityRow struct {
	ent entityKey
	row kpiRow
}

// evalKPIs 在内存对单实体逐 KPI 求值（无 DB 往返）——#516 把求值与写入解耦：
// 求值结果不再立刻 INSERT，而是返回给调用方累积，最后全桶批量写。
// 公式解析/求值/门槛过滤语义与原 evalAndInsertKPIs 完全一致（KPI 公式求值保持不动）。
func (a *Aggregator) evalKPIs(
	ent entityKey,
	kpis []router.KPIDef,
	counters map[string]float64,
	ownSet map[string]float64,
) []kpiRow {
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
				zap.String("kpi", k.Name), zap.String("formula", k.Formula),
				zap.String("device_sn", ent.sn), zap.Error(err))
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
	return rows
}

// batchInsertKPIs 把全桶累积的 KPI 行批量写入 target（#516 N+1 批量化）：
// 单条多行 VALUES（按 kpiInsertBatchSize 分条以守 PG 参数上限），替代逐实体单条 INSERT，
// 往返从"每实体一次（上万次）"降到"每千行一次"。返回总写入行数（含 UPSERT 冲突更新）。
func (a *Aggregator) batchInsertKPIs(ctx context.Context, target string, w WindowSpec, pending []entityRow) (int, error) {
	total := 0
	for start := 0; start < len(pending); start += kpiInsertBatchSize {
		end := start + kpiInsertBatchSize
		if end > len(pending) {
			end = len(pending)
		}
		sql, args := buildKPIInsertSQLMulti(target, w, pending[start:end])
		tag, err := a.db.Exec(ctx, sql, args...)
		if err != nil {
			return total, err
		}
		total += int(tag.RowsAffected())
	}
	return total, nil
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
	//
	// 源筛选窗口（#516 分区裁剪）：按**分区列** time 落入半开窗口 [w.Start, w.End) 框桶
	// （time >= w.Start AND time < w.End）。源表/上级表均为 TimescaleDB 超表、按 time 列分区；
	// 改用分区列过滤后查询只命中目标分片、走索引（分区裁剪），消除原按非分区列 start_time
	// 全表扫（公共环境 EXPLAIN 约 106 倍成本差）。
	// 等价性：#479 已统一桶头语义 time == start_time，查的是同一批源行、聚合行为完全不变，
	// 仅获得分区裁剪 + 索引收益。半开区间语义不变；args 顺序不变（$4=w.Start, $5=w.End）。
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
  AND m.time >= $4
  AND m.time <  $5
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
// 读的是 target 表（每行即一个已聚合完整桶）。target 同样是 TimescaleDB 超表、按分区列
// time 分区；改前按非分区的桶尾时刻列 end_time 精确命中会跨全部分片扫，故 #516 改为
// 按**分区列（桶头）time = w.Start** 精确命中——获得分区裁剪 + time 索引，只碰单分片。
//
// 等价性（#479 背书）：#479 已统一桶头语义 time == start_time == 桶头，且同一 granularity 内
// 每个桶的桶头唯一，故 (granularity = $1 AND time = w.Start) 精确锁定唯一本桶，与改前
// (end_time = w.End AND time = w.Start) 命中同一行集合（仅去掉冗余的桶尾时刻列谓词）。
// 不串相邻桶：相邻桶桶头不同（前桶 time = 本桶 w.Start - 桶宽，后桶 time = w.End），time = w.Start
// 精确等值不会命中前后桶（边界单测 Test_buildListEntitiesInBucketSQL_DoesNotLeakToAdjacentBuckets 守卫）。
//
// T-B：DISTINCT 维加 object_ldn——基础小区 / 各 PLMN / 设备级空串各成一个独立 KPI 实体。
//
// 参数顺序：$1=granularity, $2=bucket_start(=w.Start，分区列桶头)
func buildListEntitiesInBucketSQL(target string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT DISTINCT device_oui, device_sn, object_ldn
FROM %s
WHERE metric_type = 'counter'
  AND granularity = $1
  AND time = $2`, target)
	return sql, []any{string(w.Granularity), w.Start}
}

// buildLoadCountersByObjectLdnSQL 构造"取某设备本桶各 object_ldn 行的 counter 值"SELECT。
//
// 与 buildListEntitiesInBucketSQL 同理：#516 把精确命中从非分区的桶尾时刻列 end_time
// 改为**分区列（桶头）time = w.Start**——target 是按 time 分区的超表，按分区列精确等值
// 命中获得分区裁剪 + time 索引、只碰单分片，消除原按 end_time 全分片扫。
// 等价性（#479 背书）：time == start_time == 桶头、同 granularity 内桶头唯一，故
// (granularity AND time = w.Start) 命中与改前 (end_time = w.End AND time = w.Start) 同一行集合；
// 相邻桶桶头不同（前桶 time = w.Start - 桶宽），time = w.Start 精确等值不会命中相邻桶。
//
// 【T-B 关键】：拆掉 T-A 的"折回设备级"层——不再 GROUP BY metric_path 把多小区相加，
// 而是按 (object_ldn, metric_path) 精确带出每行计数器（T-A 后该表每对组合恰一行），
// 交由 countersForEntity 按实体取本行 + 对 PLMN 实体做跨层级配对。
// 这从根上杜绝了"把基础小区 + 各 PLMN 计数相加折回设备级"导致的跨小区串味/重复叠加。
//
// 参数顺序：$1=oui, $2=sn, $3=granularity, $4=bucket_start(=w.Start，分区列桶头)
func buildLoadCountersByObjectLdnSQL(target, oui, sn string, w WindowSpec) (string, []any) {
	sql := fmt.Sprintf(`
SELECT object_ldn, metric_path, metric_value
FROM %s
WHERE device_oui = $1
  AND device_sn  = $2
  AND metric_type = 'counter'
  AND granularity = $3
  AND time = $4`, target)
	return sql, []any{oui, sn, string(w.Granularity), w.Start}
}

// buildLoadCountersForDevicesSQL 构造"整桶一次性载入一批设备全部 counter 行"SELECT（#516 N+1 批量化）。
//
// 与 buildLoadCountersByObjectLdnSQL 的精确命中语义完全一致（按**分区列（桶头）time = w.Start**
// 精确等值命中本桶，获分区裁剪 + time 索引、只碰单分片），区别仅在：
//   - 一次带一批设备（(device_oui, device_sn) IN ((..),(..)) 复合 IN），把"逐设备一次查询"
//     折成"每批一次查询"——往返条数从设备数降到批数（#516 第一个慢源 N+1 的根治）。
//   - SELECT 多带 device_oui/device_sn 两列，供调用方按设备归位（单设备版无需带，因 oui/sn 已知）。
//
// 不串相邻桶：与单设备版同理，time = w.Start 精确等值，相邻桶桶头不同，不命中前后桶。
// 等价性（#479 背书）：time == start_time == 桶头、同 granularity 内桶头唯一。
//
// 参数顺序：$1=granularity, $2=bucket_start(=w.Start，分区列桶头)，
// 其后每设备两参（oui, sn）依次排布在复合 IN 里（$3=oui1,$4=sn1,$5=oui2,$6=sn2,...）。
func buildLoadCountersForDevicesSQL(target string, devs []deviceKey, w WindowSpec) (string, []any) {
	args := make([]any, 0, 2+len(devs)*2)
	args = append(args, string(w.Granularity), w.Start)

	pairs := make([]string, 0, len(devs))
	pos := 3 // $1=granularity, $2=time 已占
	for _, d := range devs {
		pairs = append(pairs, fmt.Sprintf("($%d, $%d)", pos, pos+1))
		args = append(args, d.oui, d.sn)
		pos += 2
	}

	sql := fmt.Sprintf(`
SELECT device_oui, device_sn, object_ldn, metric_path, metric_value
FROM %s
WHERE metric_type = 'counter'
  AND granularity = $1
  AND time = $2
  AND (device_oui, device_sn) IN (%s)`, target, joinComma(pairs))
	return sql, args
}

// buildKPIInsertSQL 给 target 表构造单实体多 KPI 行的 INSERT（保留：供单实体单测用）。
// 内部委托 buildKPIInsertSQLMulti——把单实体的多行包装成跨实体的统一格式，行为完全一致。
func buildKPIInsertSQL(target string, ent entityKey, w WindowSpec, rows []kpiRow) (string, []any) {
	pending := make([]entityRow, 0, len(rows))
	for _, r := range rows {
		pending = append(pending, entityRow{ent: ent, row: r})
	}
	return buildKPIInsertSQLMulti(target, w, pending)
}

// buildKPIInsertSQLMulti 给 target 表构造**跨实体批量** KPI INSERT（#516 N+1 批量化）。
// 与 buildCountersSQL 不同，KPI 行是 Go 端 evaluate 后批量插入；这里复用同样的桶时间列设计。
//
// #516：从"每实体一条单实体 INSERT"改为"全桶累积行一次多行 VALUES"——单条 SQL 跨多个实体/小区/PLMN，
// 每行各自带自身实体的 oui/sn/object_ldn。往返从"每实体一次（上万次）"降到"每批一次"。
//
// T-B：object_ldn 写实体实际 object_ldn（设备级实体 object_ldn=='' 仍写 ''），
// 让 KPI 按小区/PLMN 落库分行下钻。复用 T-A 的唯一键（已含 object_ldn），同桶重跑幂等覆盖。
func buildKPIInsertSQLMulti(target string, w WindowSpec, pending []entityRow) (string, []any) {
	conflictTarget := conflictTargetForTable(target)
	withID := targetHasIDColumn(target)

	columnList := "device_oui, device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time, ingest_time, object_ldn, extra"
	if withID {
		columnList = "id, " + columnList
	}

	valueRows := make([]string, 0, len(pending))
	args := []any{}
	pos := 1
	add := func(v any) string {
		args = append(args, v)
		p := fmt.Sprintf("$%d", pos)
		pos++
		return p
	}

	for _, pr := range pending {
		ent := pr.ent
		r := pr.row
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
