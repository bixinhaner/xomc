package metrics

import appcontext "github.com/omcgo/omcgo/internal/core/context"

// IndicatorDisplayNameExpr 按 locale 返回指标显示名的 SQL 取名表达式（COALESCE 方向）。
//
//   - 中文（默认）：cn_name 优先，空则回退 en_name —— COALESCE(NULLIF(cn_name,''), en_name)
//   - 英文：en_name 优先，空则回退 cn_name —— COALESCE(NULLIF(en_name,''), cn_name)
//
// 抽出一处，供 adhoc 结果回填与 aggregator 聚合查询两条路径共用同一取名口径，避免两边漂移。
func IndicatorDisplayNameExpr(loc appcontext.Locale) string {
	if loc == appcontext.LocaleEN {
		return "COALESCE(NULLIF(en_name, ''), cn_name)"
	}
	return "COALESCE(NULLIF(cn_name, ''), en_name)"
}
