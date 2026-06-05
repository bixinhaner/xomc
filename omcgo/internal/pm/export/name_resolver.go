package export

import (
	"context"
	"fmt"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// nameResolver 把指标编号（K/C 编号）批量解析成本地化显示名，带进程内缓存避免重复查库。
// 与 aggregator.lookupIndicatorNames 同口径（三张指标表 UNION + 按 locale 取名）。
type nameResolver struct {
	db    PgQuerier
	loc   appcontext.Locale
	cache map[string]string // code → 显示名（查不到回退编号本身，也入缓存避免重复查）
}

func newNameResolver(db PgQuerier, loc appcontext.Locale) *nameResolver {
	return &nameResolver{db: db, loc: loc, cache: make(map[string]string)}
}

// resolveColumns 把列发现得到的 (编号,类型) 解析成 WideColumn（回填本地化名，回退编号本身），保持入参顺序。
// 横表表头一次性解析全部指标名，流式阶段不再逐行查名。
func (r *nameResolver) resolveColumns(ctx context.Context, keys []colKey) []WideColumn {
	missing := make([]string, 0)
	seen := make(map[string]struct{})
	for _, k := range keys {
		if k.code == "" {
			continue
		}
		if _, ok := r.cache[k.code]; ok {
			continue
		}
		if _, ok := seen[k.code]; ok {
			continue
		}
		seen[k.code] = struct{}{}
		missing = append(missing, k.code)
	}
	if len(missing) > 0 {
		r.loadNames(ctx, missing)
	}
	out := make([]WideColumn, 0, len(keys))
	for _, k := range keys {
		name := r.cache[k.code]
		if name == "" {
			name = k.code // 回退编号本身，保证列名非空、不乱码
		}
		out = append(out, WideColumn{Code: k.code, Type: k.mtype, Name: name})
	}
	return out
}

// loadNames 按编号集合一次查三张指标表，写入缓存；未命中的编号也写回退值入缓存。
func (r *nameResolver) loadNames(ctx context.Context, codes []string) {
	nameExpr := metrics.IndicatorDisplayNameExpr(r.loc)
	q := fmt.Sprintf(`
SELECT id, %[1]s AS display_name FROM perf_indicators_enb WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gnb WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gsm WHERE id = ANY($1)`, nameExpr)
	rows, err := r.db.Query(ctx, q, codes)
	if err != nil {
		// 查库失败：全部回退编号本身（入缓存避免反复重试）。
		for _, c := range codes {
			r.cache[c] = c
		}
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			break
		}
		r.cache[id] = name
	}
	// 未命中的编号补回退值入缓存。
	for _, c := range codes {
		if _, ok := r.cache[c]; !ok {
			r.cache[c] = c
		}
	}
}
