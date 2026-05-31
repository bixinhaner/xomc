package aggregator

// kpi_recompute.go 实现 T-0191（D3-1）：百分比类 / 派生 KPI 在「聚合到组」维度按公式
// 分子分母 counter 各自组内汇总后重算。
//
// 背景（旧坏逻辑）：query.go 的即席组聚合各函数对 device 级 KPI 行直接 SUM / AVG（落 ELSE 分支），
// 把「各设备已算好的百分比」相加，语义错；或撞 precheckStatisType 报 pct 不支持错。
//
// 新逻辑（统一替代）：对所有请求里 is_counter='0' 的派生 KPI（含 pct）：
//  1. 解析 K-code → {code, statisType, formula, deps}（deps = 公式里引用的 counter 名）；
//  2. 把 SQL 聚合的 metric_paths 换成「用户 counter ∪ 所有 KPI 的 deps」，且只聚 metric_type='counter' 行；
//  3. 按 (维度键, time bucket) 把聚合出的 counter 行归成 map[counterName]value，逐 KPI 用 expr 重算；
//     除零 / counter 缺失 → 跳过该桶该 KPI（不产假 0）；
//  4. 产出前剔除「仅为重算引入、用户没主动请求」的 deps counter 行；
//  5. DisplayName 由 Query 末尾的 backfillDisplayNames 统一回填。
//
// 落点：Query 入口在路由到分维度函数前后包一层（recompute*）。device 维度不做重算
// （单设备行就是设备自己算好的 KPI，无需跨设备汇总）。

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// kpiMeta 是一个派生 KPI 的重算元数据。
type kpiMeta struct {
	code       string   // K-code（= 结果行 MetricPath）
	statisType string   // perf_indicators_*.statis_type（如 pct），写进结果行 StatisType
	formula    string   // rela_platform_indicator_formula_*.formula
	deps       []string // 公式里引用的 counter 名（3GPP 名，对应 pm_metrics.metric_path）
}

// kpiRecomputeNeeded 判断是否需要走 KPI 重算（组维度 + 请求里含派生 KPI）。
// device 维度与无 metric_paths 的请求直接跳过。
func kpiRecomputeNeeded(q QueryRequest) bool {
	if q.Dimension == DimensionDevice || q.Dimension == "" {
		return false
	}
	return len(q.MetricPaths) > 0
}

// resolveKPIMetadata 把请求里的 metric_paths 拆成「派生 KPI 集合」+「普通 counter 路径集合」。
//
// 查法照搬 lookupIndicatorNames 的 UNION 范式：跨三张 perf_indicators_* 表 JOIN 各自的
// rela_platform_indicator_formula_* 取 is_counter='0' 的派生 KPI 的 {statis_type, formula}。
// K-code 在三表全局唯一（无跨表重叠），UNION 即可覆盖，无需按制式分表查。
//
// 同一 K-code 跨平台公式已确认一致；防御性兜底：若取到多行（多平台 / 多 formula），
// 取第一个并对「不同 formula」log warn。
//
// 返回：
//   - kpis：请求里命中 is_counter='0' 且有 formula 的派生 KPI（含 deps）
//   - userCounters：请求里其余路径（用户主动请求的 counter，原样保留）
func (a *Aggregator) resolveKPIMetadata(ctx context.Context, paths []string) (kpis []kpiMeta, userCounters []string) {
	if len(paths) == 0 {
		return nil, nil
	}
	const tmpl = `
SELECT pi.id, pi.statis_type, rf.formula
FROM perf_indicators_enb pi
JOIN rela_platform_indicator_formula_enb rf ON rf.indicator_id = pi.id
WHERE pi.id = ANY($1) AND pi.is_counter = '0' AND COALESCE(rf.formula, '') <> ''
UNION ALL
SELECT pi.id, pi.statis_type, rf.formula
FROM perf_indicators_gnb pi
JOIN rela_platform_indicator_formula_gnb rf ON rf.indicator_id = pi.id
WHERE pi.id = ANY($1) AND pi.is_counter = '0' AND COALESCE(rf.formula, '') <> ''
UNION ALL
SELECT pi.id, pi.statis_type, rf.formula
FROM perf_indicators_gsm pi
JOIN rela_platform_indicator_formula_gsm rf ON rf.indicator_id = pi.id
WHERE pi.id = ANY($1) AND pi.is_counter = '0' AND COALESCE(rf.formula, '') <> ''`

	metaByCode := make(map[string]kpiMeta)
	rows, err := a.db.Query(ctx, tmpl, paths)
	if err != nil {
		// 查不到元数据时退化：把所有路径当 counter 处理（与旧 counter 行为一致，不报错）。
		a.logger.Warn("kpi recompute metadata query failed; treat all paths as counters", zap.Error(err))
		return nil, paths
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var statis, formula *string
		if err := rows.Scan(&id, &statis, &formula); err != nil {
			a.logger.Warn("kpi recompute metadata scan failed", zap.Error(err))
			return nil, paths
		}
		f := ""
		if formula != nil {
			f = *formula
		}
		if f == "" {
			continue
		}
		st := ""
		if statis != nil {
			st = *statis
		}
		if prev, ok := metaByCode[id]; ok {
			if prev.formula != f {
				a.logger.Warn("kpi has multiple distinct formulas across platforms; keep first",
					zap.String("code", id), zap.String("kept", prev.formula), zap.String("ignored", f))
			}
			continue // 取第一个
		}
		parsed, perr := expr.Parse(f)
		if perr != nil {
			a.logger.Warn("kpi formula parse failed; skip recompute for code",
				zap.String("code", id), zap.String("formula", f), zap.Error(perr))
			continue
		}
		metaByCode[id] = kpiMeta{
			code:       id,
			statisType: st,
			formula:    f,
			deps:       parsed.Identifiers(),
		}
	}
	if err := rows.Err(); err != nil {
		a.logger.Warn("kpi recompute metadata rows err", zap.Error(err))
		return nil, paths
	}

	for _, p := range paths {
		if m, ok := metaByCode[p]; ok {
			kpis = append(kpis, m)
		} else {
			userCounters = append(userCounters, p)
		}
	}
	return kpis, userCounters
}

// effectiveCounterPaths 计算需要从聚合表拉取的 counter 路径 = 用户 counter ∪ 所有 KPI 的 deps（去重）。
func effectiveCounterPaths(userCounters []string, kpis []kpiMeta) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(userCounters))
	add := func(p string) {
		if _, dup := seen[p]; dup {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, c := range userCounters {
		add(c)
	}
	for _, k := range kpis {
		for _, d := range k.deps {
			add(d)
		}
	}
	return out
}

// groupKey 由结果行的维度身份键 + 时间桶构成，用于把 counter 行归到「同一组同一桶」再重算 KPI。
//
// 各维度身份字段：
//   - device_group：DeviceGroupID
//   - product：ProductID
//   - aggregate_group / network：无实体键（DeviceSN 恒为 "AGGREGATED"）
//   - band：ObjectLDN（'Band=<值>'）
//
// 时间用 RFC3339Nano 串入键，保证不同桶不混算。
type groupKey struct {
	groupID string
	product string
	ldn     string
	t       string
}

func rowGroupKey(r Row) groupKey {
	ldn := ""
	if r.ObjectLDN != nil {
		ldn = *r.ObjectLDN
	}
	return groupKey{
		groupID: r.DeviceGroupID.String(),
		product: r.ProductID.String(),
		ldn:     ldn,
		t:       r.Time.Format(time.RFC3339Nano),
	}
}

// recomputeKPIs 是组维度 KPI 重算后处理：在已聚合出的 counter 行基础上，按 (维度键, 桶) 重算 KPI 行。
//
// counterRows 是分维度函数返回的「只含 counter 行」的聚合结果（调用方已把 metric_paths 换成
// effectiveCounterPaths 且强制 metric_type='counter'）。
//
// 返回：用户请求的 counter 行（剔除仅为重算引入的 deps）+ 重算出的 KPI 行。
func recomputeKPIs(counterRows []Row, kpis []kpiMeta, userCounters []string) []Row {
	// 1. 按 (维度键, 桶) 归拢 counter 值 + 记录每个键的代表行（透传维度身份 / 时间字段）。
	type bucket struct {
		values map[string]float64
		sample Row
	}
	buckets := make(map[groupKey]*bucket)
	for _, r := range counterRows {
		k := rowGroupKey(r)
		b, ok := buckets[k]
		if !ok {
			b = &bucket{values: make(map[string]float64)}
			buckets[k] = b
			b.sample = r
		}
		b.values[r.MetricPath] = r.MetricValue
	}

	// 2. 输出：先放用户主动请求的 counter 行（剔除 deps-only）。
	userSet := make(map[string]struct{}, len(userCounters))
	for _, c := range userCounters {
		userSet[c] = struct{}{}
	}
	out := make([]Row, 0, len(counterRows))
	for _, r := range counterRows {
		if _, ok := userSet[r.MetricPath]; ok {
			out = append(out, r)
		}
	}

	// 3. 逐 (桶, KPI) 重算。为保证输出顺序稳定（先按 counter 行出现序遍历桶），
	//    用 counterRows 的出现序锚定桶遍历序。
	seenBucket := make(map[groupKey]struct{})
	for _, r := range counterRows {
		k := rowGroupKey(r)
		if _, done := seenBucket[k]; done {
			continue
		}
		seenBucket[k] = struct{}{}
		b := buckets[k]
		for _, km := range kpis {
			f, err := expr.Parse(km.formula)
			if err != nil {
				continue
			}
			val, err := f.Evaluate(b.values)
			if err != nil {
				// 除零 / counter 缺失 → 跳过该桶该 KPI，不产假 0。
				continue
			}
			kr := b.sample
			kr.MetricPath = km.code
			kr.MetricType = metrics.MetricTypeKPI
			kr.MetricValue = val
			kr.DisplayName = "" // 交给 backfillDisplayNames 回填
			if km.statisType != "" {
				st := metrics.StatisType(km.statisType)
				kr.StatisType = &st
			} else {
				kr.StatisType = nil
			}
			kr.Extra = nil // KPI 行不透传 counter 行的 extra
			out = append(out, kr)
		}
	}
	return out
}

// queryWithKPIRecompute 在组维度下做「KPI 公式重算」：解析请求 → 改查 effective counter →
// 调分维度函数（强制只聚 counter）→ 重算 KPI。device 维度 / 无派生 KPI 时退回原路径。
func (a *Aggregator) queryWithKPIRecompute(
	ctx context.Context,
	table string,
	q QueryRequest,
	dimFn func(context.Context, string, QueryRequest) ([]Row, error),
) ([]Row, error) {
	kpis, userCounters := a.resolveKPIMetadata(ctx, q.MetricPaths)
	if len(kpis) == 0 {
		// 请求里没有派生 KPI：原样查（仍强制只聚 counter 行，避免撞 device 级 KPI 行 SUM）。
		q.MetricPaths = userCounters
		q.MetricType = counterMetricType()
		return dimFn(ctx, table, q)
	}

	eff := effectiveCounterPaths(userCounters, kpis)
	cq := q
	cq.MetricPaths = eff
	cq.MetricType = counterMetricType() // 只聚 counter 行
	// 重算路径不下推 Limit/Offset：Limit 截行会切掉某桶的部分 deps counter，导致该桶 KPI
	// 漏算/算错。重算需要每个桶的全部分子分母 counter，故拉全量 counter 再在内存重算
	// （激活路径 adhoc executor 本就传 Limit=100000，行为不变；仅防御直连 API 小 Limit）。
	cq.Limit = 0
	cq.Offset = 0
	counterRows, err := dimFn(ctx, table, cq)
	if err != nil {
		return nil, err
	}
	return recomputeKPIs(counterRows, kpis, userCounters), nil
}

// counterMetricType 返回 metric_type='counter' 过滤值的指针（重算只聚 counter 行）。
func counterMetricType() *metrics.MetricType {
	ct := metrics.MetricTypeCounter
	return &ct
}
