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
// 落点：Query 入口在路由到分维度函数前后包一层（recompute*）。device 维度同样经该入口分流：
// pct/公式类仍按 counter 重算，avg/sum/max/min 从 15min KPI 点直接 rollup。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi/expr"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// kpiMeta 是一个被请求指标的重算元数据。
//
// PM-P3 起统一走编号版 arithmetic：
//   - 派生 KPI（K 编码）：arithmetic = 编号算术式（如 (C0001+C0002)/1000），deps = 引用的 counter 编号；
//   - 原始计数（C 编码）：arithmetic = 自身编号 → 公式=自己 → 恒等输出该 counter 桶内聚合值。
//
// 两类都进 kpiMeta（去掉了旧的 is_counter='0' 过滤），统一经 recomputeKPIs 重算。
type kpiMeta struct {
	code       string   // 指标编号（= 结果行 MetricPath）
	statisType string   // perf_indicators_*.statis_type（如 pct / sum / avg / max），写进结果行 StatisType
	formula    string   // perf_indicators_*.arithmetic（编号公式；原始计数为自身编号）
	deps       []string // 公式里引用的 counter 编号（对应编号化后 pm_metrics.metric_path）
	isCounter  bool     // perf_indicators_*.is_counter='1'：原始计数 → 重算行保持 metric_type='counter'
}

// resolveKPIMetadata 把请求里的 metric_paths 拆成「按 arithmetic 重算的指标集合」+「无元数据的路径集合」。
//
// PM-P3 改造：
//   - 直接读 perf_indicators_*.arithmetic（编号公式），不再 JOIN rela_platform_indicator_formula_*；
//   - 去掉 is_counter='0' 过滤——派生 KPI（K）与原始计数（C）只要 arithmetic 非空都进 kpiMeta。
//     原始计数 arithmetic = 自身编号 → 公式=自己 → 经 recomputeKPIs 恒等输出该 counter 桶内聚合值，
//     从根上消除「原始计数被那道过滤挡掉、零行」的旧 bug。
//   - 额外取 is_counter，用于 recomputeKPIs 给原始计数行保持 metric_type='counter'（不被误标成 kpi）。
//
// 查法照搬 lookupIndicatorNames 的 UNION 范式跨三张 perf_indicators_* 表。
// 编号在三表全局唯一（无跨表重叠），UNION 即可覆盖。
//
// 同一编号跨表 arithmetic 已确认一致；防御性兜底：若取到多行，取第一个并对「不同公式」log warn。
//
// 返回：
//   - kpis：请求里命中且 arithmetic 非空的指标（派生 KPI + 原始计数，含 deps）
//   - userCounters：请求里其余无元数据的路径（原样保留）
func (a *Aggregator) resolveKPIMetadata(ctx context.Context, paths []string) (kpis []kpiMeta, userCounters []string) {
	if len(paths) == 0 {
		return nil, nil
	}
	const tmpl = `
SELECT id, statis_type, arithmetic, is_counter, 'ENB' AS device_type
FROM perf_indicators_enb
WHERE id = ANY($1) AND COALESCE(arithmetic, '') <> ''
UNION ALL
SELECT id, statis_type, arithmetic, is_counter, 'GNB' AS device_type
FROM perf_indicators_gnb
WHERE id = ANY($1) AND COALESCE(arithmetic, '') <> ''
UNION ALL
SELECT id, statis_type, arithmetic, is_counter, 'GSM' AS device_type
FROM perf_indicators_gsm
WHERE id = ANY($1) AND COALESCE(arithmetic, '') <> ''`

	metaByCode := make(map[string]kpiMeta)
	rows, err := a.db.Query(ctx, tmpl, paths)
	if err != nil {
		// 查不到元数据时退化：把所有路径当 counter 处理（与旧 counter 行为一致，不报错）。
		a.logger.Warn("kpi recompute metadata query failed; treat all paths as counters", zap.Error(err))
		return nil, paths
	}
	defer rows.Close()
	for rows.Next() {
		var id, deviceType string
		var statis, formula, isCounter *string
		if err := rows.Scan(&id, &statis, &formula, &isCounter, &deviceType); err != nil {
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
		dt, derr := indicator.ParseDeviceType(deviceType)
		if derr != nil {
			a.logger.Warn("kpi recompute metadata has invalid device type",
				zap.String("code", id), zap.String("device_type", deviceType), zap.Error(derr))
			continue
		}
		f = indicator.CompileRuntimeArithmetic(dt, f)
		st := ""
		if statis != nil {
			st = *statis
		}
		ic := ""
		if isCounter != nil {
			ic = *isCounter
		}
		if prev, ok := metaByCode[id]; ok {
			if prev.formula != f {
				a.logger.Warn("indicator has multiple distinct arithmetic across tables; keep first",
					zap.String("code", id), zap.String("kept", prev.formula), zap.String("ignored", f))
			}
			continue // 取第一个
		}
		parsed, perr := expr.Parse(f)
		if perr != nil {
			a.logger.Warn("indicator arithmetic parse failed; skip recompute for code",
				zap.String("code", id), zap.String("arithmetic", f), zap.Error(perr))
			continue
		}
		metaByCode[id] = kpiMeta{
			code:       id,
			statisType: st,
			formula:    f,
			deps:       parsed.Identifiers(),
			isCounter:  ic == "1",
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
//   - device_group：DeviceGroupID + Technology（设备组快表按「组 × 制式」拆行，KPI 必须制式内重算，
//     否则同组同桶 lte/nr 的 counter 被归一桶 → 分子分母拿了别的制式的 counter → KPI 跨制式混算）
//   - product：ProductID
//   - aggregate_group / network：无实体键（DeviceSN 恒为 "AGGREGATED"）
//   - band：ObjectLDN（'Band=<值>'）
//
// 时间用 RFC3339Nano 串入键，保证不同桶不混算。
type groupKey struct {
	groupID string
	tech    string
	product string
	ldn     string
	t       string
}

// skipKey 标识一类被跳过的重算：哪个指标 + 因何跳过（供可观测性按因聚合）。
type skipKey struct {
	metric string
	reason string // "missing:<counter>" | "divzero" | "parse" | "eval"
}

func rowGroupKey(r Row) groupKey {
	ldn := ""
	if r.ObjectLDN != nil {
		ldn = *r.ObjectLDN
	}
	return groupKey{
		groupID: r.DeviceGroupID.String(),
		tech:    r.Technology,
		product: r.ProductID.String(),
		ldn:     ldn,
		t:       r.Time.Format(time.RFC3339Nano),
	}
}

// recomputeKPIs 是组维度重算后处理：在已聚合出的 counter 行基础上，按 (维度键, 桶) 重算指标行。
//
// counterRows 是分维度函数返回的「只含 counter 行」的聚合结果（调用方已把 metric_paths 换成
// effectiveCounterPaths 且强制 metric_type='counter'）。
//
// PM-P3：kpis 现含两类——
//   - 派生 KPI（km.isCounter=false）：按 arithmetic 编号公式重算，产 metric_type='kpi' 行；
//   - 原始计数（km.isCounter=true，arithmetic=自身）：直接透传该 counter 的聚合行（保留
//     metric_type='counter' / StatisType / Extra），不走公式合成，避免被误标成 kpi 行或丢 Extra。
//
// 返回：用户请求且无元数据的 counter 行 + 原始计数透传行 + 派生 KPI 重算行
// （仅为重算引入、用户没主动请求的 deps counter 行被剔除）。
//
// 可观测性（#194）：某 (组×制式×桶) 分母 counter 缺失 / 为 0 导致整条 KPI 被跳过本是设计内
// 「不产假 0」语义，但前端 legend 按返回行 distinct object_ldn 渲染 → 该组该指标整条曲线消失，
// 现象上像「丢组」。这里把被跳过的 (group_id, metric, reason) 按 reason 聚合后记一条 debug 日志，
// 便于现场判定是「该组真没数据」还是「某组设备没上报该 counter」，把数据问题与代码问题分离。
// 改为 *Aggregator 方法仅为拿到 a.logger，重算逻辑本身未变。
func (a *Aggregator) recomputeKPIs(counterRows []Row, kpis []kpiMeta, userCounters []string) []Row {
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
		b.values[r.MetricPath] = float64(r.MetricValue)
	}

	// 2. passthrough 集合 = 用户请求的无元数据 counter ∪ 原始计数（arithmetic=自身）。
	//    这两类直接透传原始 counter 行（保留 StatisType / Extra / metric_type='counter'）。
	passthrough := make(map[string]struct{}, len(userCounters)+len(kpis))
	for _, c := range userCounters {
		passthrough[c] = struct{}{}
	}
	// 仅派生 KPI 需要走公式重算合成新行。
	derived := make([]kpiMeta, 0, len(kpis))
	for _, km := range kpis {
		if km.isCounter {
			passthrough[km.code] = struct{}{}
		} else {
			derived = append(derived, km)
		}
	}
	out := make([]Row, 0, len(counterRows))
	for _, r := range counterRows {
		if _, ok := passthrough[r.MetricPath]; ok {
			out = append(out, r)
		}
	}

	// 3. 逐 (桶, 派生 KPI) 重算。为保证输出顺序稳定（先按 counter 行出现序遍历桶），
	//    用 counterRows 的出现序锚定桶遍历序。
	if len(derived) == 0 {
		return out
	}
	// 按 (metric, reason) 聚合被跳过的桶数 + 采样首个 group_id，整批一次日志（避免每桶刷屏）。
	skips := make(map[skipKey]int)
	skipSampleGroup := make(map[skipKey]string)
	recordSkip := func(metric, reason, groupID string) {
		k := skipKey{metric: metric, reason: reason}
		if _, seen := skips[k]; !seen {
			skipSampleGroup[k] = groupID
		}
		skips[k]++
	}
	seenBucket := make(map[groupKey]struct{})
	for _, r := range counterRows {
		k := rowGroupKey(r)
		if _, done := seenBucket[k]; done {
			continue
		}
		seenBucket[k] = struct{}{}
		b := buckets[k]
		for _, km := range derived {
			f, err := expr.Parse(km.formula)
			if err != nil {
				recordSkip(km.code, "parse", k.groupID)
				continue
			}
			val, err := f.Evaluate(b.values)
			if err != nil {
				// 除零 / counter 缺失 → 跳过该桶该 KPI，不产假 0。按因归类供可观测性聚合。
				var missing *expr.MissingCounterError
				switch {
				case errors.As(err, &missing):
					recordSkip(km.code, "missing:"+missing.Counter, k.groupID)
				case errors.Is(err, expr.ErrDivByZero):
					recordSkip(km.code, "divzero", k.groupID)
				default:
					recordSkip(km.code, "eval", k.groupID)
				}
				continue
			}
			kr := b.sample
			kr.MetricPath = km.code
			kr.MetricType = metrics.MetricTypeKPI
			kr.MetricValue = jsonx.Float(val)
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
	a.logRecomputeSkips(skips, skipSampleGroup)
	return out
}

// logRecomputeSkips 把 recomputeKPIs 累积的「被跳过 (metric, reason)」聚合记一条 debug 日志。
// 拆成独立方法便于阅读 / 后续可改埋点。零跳过时不打印。
func (a *Aggregator) logRecomputeSkips(skips map[skipKey]int, sampleGroup map[skipKey]string) {
	if len(skips) == 0 {
		return
	}
	details := make([]string, 0, len(skips))
	total := 0
	for k, n := range skips {
		total += n
		details = append(details, fmt.Sprintf("%s|%s×%d(group=%s)", k.metric, k.reason, n, sampleGroup[k]))
	}
	sort.Strings(details) // 稳定输出便于比对
	a.logger.Debug("device-group KPI recompute skipped buckets",
		zap.Int("skipped_total", total),
		zap.Int("distinct_metric_reason", len(skips)),
		zap.Strings("details", details))
}

// queryWithKPIRecompute 在组维度下做「KPI 公式重算」：解析请求 → 改查 effective counter →
// 调分维度函数（强制只聚 counter）→ 重算 KPI。device 维度 / 无派生 KPI 时退回原路径。
func (a *Aggregator) queryWithKPIRecompute(
	ctx context.Context,
	table string,
	q QueryRequest,
	dimFn func(context.Context, string, QueryRequest) ([]Row, error),
) ([]Row, error) {
	// KPI-ALL-IND：全网/全聚放开到全库——汇总全部 counter 的同时，额外重算全库派生 KPI 产 KPI 行。
	// 与默认路径（按请求里显式 KPI 列表重算）互斥：这里指标列表本就为空（全聚），改由指标库枚举驱动。
	if q.RecomputeAllKPIs {
		return a.queryFullLibraryWithKPIs(ctx, table, q, dimFn)
	}

	// #532 P2 store-all-by-enabled：落库侧全存「已启用指标集」——汇总全部 counter 的同时，
	// 按已启用 ∩ 派生重算 KPI 产 KPI 行；枚举源限定到 enabled_pm_indicators_{tech}（体量可控，
	// 非全库）。与 RecomputeAllKPIs 互斥（后者全库）；二者均要求请求侧不下推 MetricPaths。
	if q.StoreAllEnabled {
		return a.queryEnabledWithKPIs(ctx, table, q, dimFn)
	}

	kpis, userCounters := a.resolveKPIMetadata(ctx, q.MetricPaths)
	if len(kpis) == 0 {
		// 请求里没有派生 KPI：原样查（仍强制只聚 counter 行，避免撞 device 级 KPI 行 SUM）。
		q.MetricPaths = userCounters
		q.MetricType = counterMetricType()
		return dimFn(ctx, table, q)
	}

	formulaKPIs, directKPIs := splitKPIMetaByRollupMode(kpis)
	directRows, err := a.queryDirectRollupKPIs(ctx, q, directKPIs)
	if err != nil {
		return nil, err
	}
	if len(formulaKPIs) == 0 {
		if len(userCounters) == 0 {
			return directRows, nil
		}
		cq := q
		cq.MetricPaths = userCounters
		cq.MetricType = counterMetricType()
		counterRows, err := dimFn(ctx, table, cq)
		if err != nil {
			return nil, err
		}
		return append(counterRows, directRows...), nil
	}

	eff := effectiveCounterPaths(userCounters, formulaKPIs)
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
	recomputed := a.recomputeKPIs(counterRows, formulaKPIs, userCounters)
	return append(recomputed, directRows...), nil
}

// counterMetricType 返回 metric_type='counter' 过滤值的指针（重算只聚 counter 行）。
func counterMetricType() *metrics.MetricType {
	ct := metrics.MetricTypeCounter
	return &ct
}

func splitKPIMetaByRollupMode(kpis []kpiMeta) (formula, direct []kpiMeta) {
	for _, km := range kpis {
		if !km.isCounter && isDirectRollupStatisType(km.statisType) {
			direct = append(direct, km)
		} else {
			formula = append(formula, km)
		}
	}
	return formula, direct
}

func (a *Aggregator) queryDirectRollupKPIs(
	ctx context.Context,
	q QueryRequest,
	kpis []kpiMeta,
) ([]Row, error) {
	if len(kpis) == 0 {
		return nil, nil
	}
	sqlStr, args, err := buildDirectRollupKPI15MinSQL(q, kpis)
	if err != nil {
		return nil, err
	}
	rows, err := a.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregator.Query direct KPI rollup from 15min: %w", err)
	}
	defer rows.Close()
	out := make([]Row, 0)
	for rows.Next() {
		var r Row
		var groupID, productID *uuid.UUID
		var objectLDN, statis *string
		var metricType, granularity string
		if err := rows.Scan(
			&r.DeviceOUI, &r.DeviceSN, &groupID, &r.Technology, &productID, &objectLDN,
			&r.MetricPath, &metricType, &r.MetricValue, &statis, &granularity,
			&r.Time, &r.StartTime, &r.EndTime, &r.IngestTime,
		); err != nil {
			return nil, fmt.Errorf("aggregator.Query scan direct KPI rollup from 15min: %w", err)
		}
		if groupID != nil {
			r.DeviceGroupID = *groupID
		}
		if productID != nil {
			r.ProductID = *productID
		}
		r.ObjectLDN = objectLDN
		r.MetricType = metrics.MetricType(metricType)
		r.Granularity = metrics.Granularity(granularity)
		if statis != nil {
			st := metrics.StatisType(*statis)
			r.StatisType = &st
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("aggregator.Query iterate direct KPI rollup from 15min: %w", err)
	}
	return out, nil
}

func buildDirectRollupKPI15MinSQL(q QueryRequest, kpis []kpiMeta) (string, []any, error) {
	bucketExpr, endExpr, err := directRollupBucketExpr(q.Granularity)
	if err != nil {
		return "", nil, err
	}
	args := []any{}
	pos := 1
	add := func(v any) string {
		args = append(args, v)
		p := fmt.Sprintf("$%d", pos)
		pos++
		return p
	}

	defRows := make([]string, 0, len(kpis))
	for _, km := range kpis {
		defRows = append(defRows, fmt.Sprintf("(%s, %s)", add(km.code), add(km.statisType)))
	}

	where := []string{"m.metric_type = 'kpi'", "m.granularity = '15min'"}
	if !q.StartTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time >= %s", add(q.StartTime)))
	}
	if !q.EndTime.IsZero() {
		where = append(where, fmt.Sprintf("m.time < %s", add(q.EndTime)))
	}
	if len(q.DeviceOUIs) > 0 && len(q.DeviceSNs) > 0 {
		n := len(q.DeviceOUIs)
		if len(q.DeviceSNs) < n {
			n = len(q.DeviceSNs)
		}
		pairs := make([]string, 0, n)
		for i := 0; i < n; i++ {
			pairs = append(pairs, fmt.Sprintf("(%s, %s)", add(q.DeviceOUIs[i]), add(q.DeviceSNs[i])))
		}
		where = append(where, "(m.device_oui, m.device_sn) IN ("+joinComma(pairs)+")")
	} else if len(q.DeviceOUIs) > 0 {
		where = append(where, fmt.Sprintf("m.device_oui = ANY(%s)", add(q.DeviceOUIs)))
	} else if len(q.DeviceSNs) > 0 {
		where = append(where, fmt.Sprintf("m.device_sn = ANY(%s)", add(q.DeviceSNs)))
	}
	if len(q.ObjectLDNs) > 0 {
		where = append(where, fmt.Sprintf("m.object_ldn = ANY(%s)", add(q.ObjectLDNs)))
	}
	if len(q.Weekdays) > 0 && len(q.Weekdays) < 7 {
		where = append(where, fmt.Sprintf("EXTRACT(dow FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Weekdays)))
	}
	if len(q.Hours) > 0 && len(q.Hours) < 24 {
		where = append(where, fmt.Sprintf("EXTRACT(hour FROM (m.start_time AT TIME ZONE %s))::int = ANY(%s)",
			add(calendarfilter.NormalizeName(q.CalendarTimezone)), add(q.Hours)))
	}

	joins := []string{"JOIN kpi_defs kd ON kd.metric_path = m.metric_path"}
	selectDeviceOUI := "''::text"
	selectDeviceSN := "'AGGREGATED'::text"
	selectGroupID := "NULL::uuid"
	selectTechnology := "''::text"
	selectProductID := "NULL::uuid"
	selectObjectLDN := "NULL::text"
	groupBy := []string{"bucket", "m.metric_path", "kd.statis_type"}
	distinctOn := []string{"m.device_oui", "m.device_sn", "m.metric_path", "m.granularity", "m.time", "m.object_ldn"}

	switch q.Dimension {
	case DimensionDevice:
		selectDeviceOUI = "m.device_oui"
		selectDeviceSN = "m.device_sn"
		selectObjectLDN = "COALESCE(m.object_ldn, '')"
		if len(q.Technologies) > 0 {
			where = append(where,
				fmt.Sprintf("(m.device_oui, m.device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE technology = ANY(%s))", add(q.Technologies)),
			)
		}
		where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)
	case DimensionDeviceGroup:
		joins = append(joins,
			"JOIN device_dim d ON d.oui = m.device_oui AND d.serial_number = m.device_sn",
			"JOIN device_group_member_dim dgm ON dgm.device_id = d.id",
		)
		selectGroupID = "dgm.group_id"
		selectTechnology = "d.technology"
		distinctOn = append(distinctOn, "dgm.group_id", "d.technology")
		if len(q.DeviceGroupIDs) > 0 {
			where = append(where, fmt.Sprintf("dgm.group_id = ANY(%s)", add(q.DeviceGroupIDs)))
		}
		if len(q.Technologies) > 0 {
			where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
		}
	case DimensionProduct:
		joins = append(joins, "JOIN device_dim d ON d.oui = m.device_oui AND d.serial_number = m.device_sn")
		selectProductID = "d.product_id"
		distinctOn = append(distinctOn, "d.product_id")
		where = append(where, "d.product_id IS NOT NULL")
		if len(q.ProductIDs) > 0 {
			where = append(where, fmt.Sprintf("d.product_id = ANY(%s)", add(q.ProductIDs)))
		}
		if len(q.Technologies) > 0 {
			where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
		}
		where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)
	case DimensionBand:
		joins = append(joins,
			"JOIN device_dim d ON d.oui = m.device_oui AND d.serial_number = m.device_sn",
			`JOIN cell_band_dim cb
  ON cb.device_id = d.id
 AND cb.cell_id = COALESCE(
     substring(m.object_ldn FROM 'Cellid=([0-9]+)'),
     substring(m.object_ldn FROM 'NrCGI=([0-9]+)'),
     substring(m.object_ldn FROM 'Uid=([^,]+)')
 )`,
		)
		selectObjectLDN = "'Band=' || cb.band"
		distinctOn = append(distinctOn, "cb.band")
		where = append(where, "m.object_ldn IS NOT NULL")
		if len(q.Technologies) > 0 {
			where = append(where, fmt.Sprintf("d.technology = ANY(%s)", add(q.Technologies)))
		}
		where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)
	case DimensionAggregateGroup, DimensionNetwork:
		if len(q.Technologies) > 0 {
			where = append(where,
				fmt.Sprintf("(m.device_oui, m.device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE technology = ANY(%s))", add(q.Technologies)),
			)
		}
		where = appendVisibleSNWhere(where, "m.device_sn", q.VisibleGroups, add)
	default:
		return "", nil, fmt.Errorf("aggregator: unsupported direct KPI rollup dimension %s", q.Dimension)
	}
	groupBy = []string{
		"bucket", "m.metric_path", "kd.statis_type",
		"m.out_device_oui", "m.out_device_sn", "m.out_device_group_id",
		"m.out_technology", "m.out_product_id", "m.out_object_ldn",
	}

	aggValue := `CASE kd.statis_type
        WHEN 'sum' THEN SUM(m.metric_value)
        WHEN 'avg' THEN AVG(m.metric_value)
        WHEN 'max' THEN MAX(m.metric_value)
        WHEN 'min' THEN MIN(m.metric_value)
    END`
	sqlStr := fmt.Sprintf(`
WITH kpi_defs(metric_path, statis_type) AS (
    VALUES %s
),
dedup AS (
    SELECT DISTINCT ON (%s)
        m.device_oui, m.device_sn, m.metric_path, m.metric_value, m.object_ldn, m.start_time, m.time, m.ingest_time,
        kd.statis_type,
        %s AS bucket,
        %s AS out_device_oui,
        %s AS out_device_sn,
        %s AS out_device_group_id,
        %s AS out_technology,
        %s AS out_product_id,
        %s AS out_object_ldn
    FROM pm_metrics m
    %s
    WHERE %s
    ORDER BY %s, m.ingest_time DESC
)
SELECT
    m.out_device_oui,
    m.out_device_sn,
    m.out_device_group_id,
    m.out_technology,
    m.out_product_id,
    m.out_object_ldn,
    m.metric_path,
    'kpi' AS metric_type,
    %s AS metric_value,
    kd.statis_type,
    %s AS granularity,
    bucket AS time,
    bucket AS start_time,
    %s AS end_time,
    NOW() AS ingest_time
FROM dedup m
JOIN kpi_defs kd ON kd.metric_path = m.metric_path
%s
GROUP BY %s
ORDER BY bucket DESC`,
		joinComma(defRows),
		strings.Join(distinctOn, ", "),
		bucketExpr,
		selectDeviceOUI,
		selectDeviceSN,
		selectGroupID,
		selectTechnology,
		selectProductID,
		selectObjectLDN,
		strings.Join(joins, "\n    "),
		strings.Join(where, "\n      AND "),
		strings.Join(distinctOn, ", "),
		aggValue,
		add(string(q.Granularity)),
		endExpr,
		"",
		strings.Join(groupBy, ", "),
	)
	if q.Limit > 0 {
		sqlStr += fmt.Sprintf("\nLIMIT %s", add(q.Limit))
	}
	if q.Offset > 0 {
		sqlStr += fmt.Sprintf("\nOFFSET %s", add(q.Offset))
	}
	return sqlStr, args, nil
}

func directRollupBucketExpr(g metrics.Granularity) (bucket string, end string, err error) {
	switch g {
	case metrics.Granularity15Min:
		return "m.time", "bucket + interval '15 minutes'", nil
	case metrics.GranularityHourly:
		return "date_trunc('hour', m.time)", "bucket + interval '1 hour'", nil
	case metrics.GranularityDaily:
		return "date_trunc('day', m.time)", "bucket + interval '1 day'", nil
	case metrics.GranularityWeekly:
		return "date_trunc('week', m.time)", "bucket + interval '1 week'", nil
	case metrics.GranularityMonthly:
		return "date_trunc('month', m.time)", "bucket + interval '1 month'", nil
	default:
		return "", "", fmt.Errorf("aggregator: unsupported direct KPI rollup granularity %s", g)
	}
}

// queryFullLibraryWithKPIs 是「全网/全聚放开到全库」的取数路径（RecomputeAllKPIs=true）：
//  1. 拉全部 counter（不下推 metric_path 过滤、强制只聚 counter 行）—— 与放开前的全聚行为一致；
//  2. 从指标库枚举全库「派生 KPI」元数据（动态加载，不在 seed 硬编码，库变即生效）；
//  3. 复用 recomputeKPIs（userCounters=nil）按公式从同 (维度键,桶) 的 counter 重算出 KPI 行；
//  4. 返回「全部 counter 行 + 全部派生 KPI 行」一并落库 —— 首页读现成全网表时 counter / KPI 都有线。
//
// 跨制式安全：counter 编号三表全局唯一，故某制式任务的 counter 行只能命中本制式 KPI 的 deps，
// 其它制式 KPI 因 deps 缺失被 recomputeKPIs 跳过（不产假 0），无跨制式误算。
func (a *Aggregator) queryFullLibraryWithKPIs(
	ctx context.Context,
	table string,
	q QueryRequest,
	dimFn func(context.Context, string, QueryRequest) ([]Row, error),
) ([]Row, error) {
	cq := q
	cq.MetricPaths = nil // 全库：不下推指标过滤
	cq.MetricType = counterMetricType()
	cq.Limit = 0 // 重算需每桶全部 deps counter，不截行
	cq.Offset = 0
	counterRows, err := dimFn(ctx, table, cq)
	if err != nil {
		return nil, err
	}
	kpis := a.resolveAllDerivedKPIs(ctx)
	if len(kpis) == 0 {
		// 指标库无可重算 KPI（或查询失败已降级）：退回「全部 counter」，不丢 counter 行。
		return counterRows, nil
	}
	formulaKPIs, directKPIs := splitKPIMetaByRollupMode(kpis)
	out := counterRows
	if len(formulaKPIs) > 0 {
		// recomputeKPIs(userCounters=nil, pct 派生 KPI)：passthrough 为空 → 仅产出 KPI 行。
		out = append(out, a.recomputeKPIs(counterRows, formulaKPIs, nil)...)
	}
	directRows, err := a.queryDirectRollupKPIs(ctx, q, directKPIs)
	if err != nil {
		return nil, err
	}
	out = append(out, directRows...)
	return out, nil
}

// queryEnabledWithKPIs 是「落库侧全存已启用指标集」的取数路径（#532 P2，StoreAllEnabled=true）：
//  1. 按请求制式枚举「已启用指标集」（enabled_pm_indicators_{tech}），拆成 已启用 counter / 已启用派生 KPI；
//  2. 已启用集为空 → 降级：退回「全部 counter」原聚合（不下推过滤、只聚 counter 行），不丢 counter；
//  3. 拉 effective counter（已启用 counter ∪ 派生 KPI deps）—— deps 即便自身未启用也要拉来供重算；
//  4. recomputeKPIs(userCounters=已启用 counter, kpis=已启用派生)：
//     passthrough 只含已启用 counter（dep-only 的非启用 counter 行被剔除）+ 已启用派生 KPI 重算行
//     → 输出严格落在「已启用 counter ∪ 已启用派生 KPI」集合内（store-within-enabled 守恒）。
//
// 与 queryFullLibraryWithKPIs（全库）的差异仅在「枚举源 = 已启用集而非全库」，体量受已启用上界约束。
func (a *Aggregator) queryEnabledWithKPIs(
	ctx context.Context,
	table string,
	q QueryRequest,
	dimFn func(context.Context, string, QueryRequest) ([]Row, error),
) ([]Row, error) {
	enabled := a.resolveEnabledIndicators(ctx, q.Technologies)
	if len(enabled) == 0 {
		// 已启用集为空（无配置 / 查询失败已降级）：退回「全库 counter + 全库派生 KPI 重算」口径，
		// 而非裸 counter 原聚合——后者会丢掉本可重算的派生 KPI（相对仅配置反而退化，见 #532 P2 回合1 运行栈）。
		// queryFullLibraryWithKPIs 内部 resolveAllDerivedKPIs 查的是 perf_indicators_*（时序库也有副本），
		// 故该降级不依赖主库 metaDB，纯库降级也不丢 KPI、不丢 counter。
		a.logger.Warn("queryEnabledWithKPIs: enabled set empty; degrade to full-library KPI recompute (no KPI loss)")
		return a.queryFullLibraryWithKPIs(ctx, table, q, dimFn)
	}
	// 用已启用集合走与默认重算同一套元数据解析：命中 arithmetic 的进 kpis（含派生 + 原始计数），
	// 其余无元数据的进 userCounters（纯 counter 编号）。
	kpis, userCounters := a.resolveKPIMetadata(ctx, enabled)
	formulaKPIs, directKPIs := splitKPIMetaByRollupMode(kpis)
	eff := effectiveCounterPaths(userCounters, formulaKPIs)
	var counterRows []Row
	if len(eff) > 0 {
		cq := q
		cq.MetricPaths = eff
		cq.MetricType = counterMetricType() // 只聚 counter 行
		cq.Limit = 0                        // 重算需每桶全部 deps counter，不截行
		cq.Offset = 0
		var err error
		counterRows, err = dimFn(ctx, table, cq)
		if err != nil {
			return nil, err
		}
	}
	if len(kpis) == 0 {
		// 已启用集里没有任何 arithmetic 指标：所有已启用项都是裸 counter，直接返回（已被 eff 收口）。
		return counterRows, nil
	}
	out := counterRows
	if len(formulaKPIs) > 0 {
		// passthrough = 已启用 counter（userCounters）+ 已启用原始计数；dep-only 非启用 counter 被剔除。
		out = a.recomputeKPIs(counterRows, formulaKPIs, userCounters)
	}
	directRows, err := a.queryDirectRollupKPIs(ctx, q, directKPIs)
	if err != nil {
		return nil, err
	}
	return append(out, directRows...), nil
}

// resolveEnabledIndicators 按请求制式枚举「已启用指标编号集」（#532 P2）。
//
// 制式（lte/nr/gsm）→ 启用表后缀（enb/gnb/gsm）；多制式取并集（去重）。
// 跨 operator_code 取并集（default + 各运营商任一启用即视为启用）；编号在三表全局唯一。
// 查询失败 → 返回空（调用方据此降级为「全部 counter」，不报错、不丢 counter）。
func (a *Aggregator) resolveEnabledIndicators(ctx context.Context, technologies []string) []string {
	suffixes := enabledTableSuffixes(technologies)
	if len(suffixes) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, suf := range suffixes {
		// 表名由内部固定后缀拼接（非用户输入），无注入面。
		// enabled_pm_indicators_* 只在主库（metaDB），时序库（db）无此表——必须走 metaDB，
		// 否则真实部署态报 relation "enabled_pm_indicators_enb" does not exist (42P01) 整段降级。
		sql := fmt.Sprintf("SELECT DISTINCT indicator_id FROM enabled_pm_indicators_%s", suf)
		rows, err := a.metaDB.Query(ctx, sql)
		if err != nil {
			a.logger.Warn("resolveEnabledIndicators query failed; degrade this suffix",
				zap.String("suffix", suf), zap.Error(err))
			continue
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				a.logger.Warn("resolveEnabledIndicators scan failed", zap.String("suffix", suf), zap.Error(err))
				break
			}
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
		rows.Close()
	}
	return out
}

// enabledTableSuffixes 把任务制式集合（lte/nr/gsm）映射到启用表后缀（enb/gnb/gsm）。
//
//	lte → enb · nr → gnb · gsm → gsm
//
// 空制式（任务不限制式）→ 三表全取（保守取全部已启用集，与「不限制式即全制式」语义一致）。
// 未知制式忽略。返回去重后缀切片。
func enabledTableSuffixes(technologies []string) []string {
	if len(technologies) == 0 {
		return []string{"enb", "gnb", "gsm"}
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		if _, dup := seen[s]; dup {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range technologies {
		switch t {
		case "lte":
			add("enb")
		case "nr":
			add("gnb")
		case "gsm":
			add("gsm")
		}
	}
	return out
}

// resolveAllDerivedKPIs 从指标库枚举全库「派生 KPI」（arithmetic 非空且 is_counter≠'1'）的重算元数据。
// 用于 RecomputeAllKPIs 路径动态驱动全库 KPI 重算，避免在 seed/任务里硬编码指标列表（库变即漂）。
// 原始计数（is_counter='1'）不在此列：其聚合值已由全部 counter 汇总直接产出，无需公式重算。
// 查法照搬 resolveKPIMetadata 的跨三表 UNION 范式（编号三表全局唯一）；查询失败降级返回空（不报错、不丢 counter）。
func (a *Aggregator) resolveAllDerivedKPIs(ctx context.Context) []kpiMeta {
	const tmpl = `
SELECT id, statis_type, arithmetic, 'ENB' AS device_type FROM perf_indicators_enb WHERE COALESCE(arithmetic, '') <> '' AND COALESCE(is_counter, '0') <> '1'
UNION ALL
SELECT id, statis_type, arithmetic, 'GNB' AS device_type FROM perf_indicators_gnb WHERE COALESCE(arithmetic, '') <> '' AND COALESCE(is_counter, '0') <> '1'
UNION ALL
SELECT id, statis_type, arithmetic, 'GSM' AS device_type FROM perf_indicators_gsm WHERE COALESCE(arithmetic, '') <> '' AND COALESCE(is_counter, '0') <> '1'`
	rows, err := a.db.Query(ctx, tmpl)
	if err != nil {
		a.logger.Warn("resolveAllDerivedKPIs query failed; skip KPI recompute for full-lib aggregation", zap.Error(err))
		return nil
	}
	defer rows.Close()
	seen := make(map[string]struct{})
	var out []kpiMeta
	for rows.Next() {
		var id, deviceType string
		var statis, formula *string
		if err := rows.Scan(&id, &statis, &formula, &deviceType); err != nil {
			a.logger.Warn("resolveAllDerivedKPIs scan failed", zap.Error(err))
			return out
		}
		f := ""
		if formula != nil {
			f = *formula
		}
		if f == "" {
			continue
		}
		dt, derr := indicator.ParseDeviceType(deviceType)
		if derr != nil {
			a.logger.Warn("resolveAllDerivedKPIs has invalid device type",
				zap.String("code", id), zap.String("device_type", deviceType), zap.Error(derr))
			continue
		}
		f = indicator.CompileRuntimeArithmetic(dt, f)
		if _, dup := seen[id]; dup {
			continue // 编号三表全局唯一，防御性去重
		}
		parsed, perr := expr.Parse(f)
		if perr != nil {
			a.logger.Warn("resolveAllDerivedKPIs arithmetic parse failed; skip code",
				zap.String("code", id), zap.String("arithmetic", f), zap.Error(perr))
			continue
		}
		st := ""
		if statis != nil {
			st = *statis
		}
		seen[id] = struct{}{}
		out = append(out, kpiMeta{
			code:       id,
			statisType: st,
			formula:    f,
			deps:       parsed.Identifiers(),
			isCounter:  false,
		})
	}
	if err := rows.Err(); err != nil {
		a.logger.Warn("resolveAllDerivedKPIs rows err", zap.Error(err))
	}
	return out
}
