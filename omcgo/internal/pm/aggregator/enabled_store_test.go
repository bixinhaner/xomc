package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// enabledRow 是 resolveEnabledIndicators 的一行（DISTINCT indicator_id 单列）。
func enabledRow(id string) []any { return []any{id} }

// ── enabledTableSuffixes 制式→后缀映射 ──────────────────────────────────────

func Test_enabledTableSuffixes(t *testing.T) {
	assert.Equal(t, []string{"enb"}, enabledTableSuffixes([]string{"lte"}))
	assert.Equal(t, []string{"gnb"}, enabledTableSuffixes([]string{"nr"}))
	assert.Equal(t, []string{"gsm"}, enabledTableSuffixes([]string{"gsm"}))
	// 空制式（不限制式）→ 三表全取
	assert.Equal(t, []string{"enb", "gnb", "gsm"}, enabledTableSuffixes(nil))
	// 多制式去重
	assert.Equal(t, []string{"enb", "gnb"}, enabledTableSuffixes([]string{"lte", "nr", "lte"}))
	// 未知制式忽略
	assert.Empty(t, enabledTableSuffixes([]string{"wifi"}))
}

// ── 路径1：store_all-by-enabled 全存——已启用 counter 透传 + 已启用派生 KPI 重算，
// dep-only 非启用 counter 被剔除（落入已启用超集而非全库、且 store-within-enabled 守恒）。
//
// 已启用集 = {C001(counter), C002(counter), K900(派生=C001/C003*100)}。
// 注意 C003 是 K900 的 dep 但本身未启用 → 必须被拉来重算、但不得作为结果行落库。
func Test_QueryEnabled_StoresEnabledSupersetAndDropsDepOnlyCounters(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	pid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			// call#0 resolveEnabledIndicators（lte → enb 表）：已启用集 3 项
			&fakeRows{rows: [][]any{enabledRow("C001"), enabledRow("C002"), enabledRow("K900")}},
			// call#1 resolveKPIMetadata(已启用集)：C001/C002 是裸 counter（无 arithmetic 不返回），
			// K900 是派生 KPI（arithmetic=C001/C003*100）。原始计数若库里 arithmetic=自身也会进 kpis，
			// 这里 C001/C002 不带 arithmetic → 落 userCounters。
			&fakeRows{rows: [][]any{
				metaRow("K900", "pct", "C001/C003*100"),
			}},
			// call#2 主聚合（queryProductTable，只聚 counter）：含 C001/C002（已启用）+ C003（dep-only 非启用）
			&fakeRows{rows: [][]any{
				productRow(pid, "C001", "counter", 50, "sum", "hourly", now),
				productRow(pid, "C002", "counter", 7, "sum", "hourly", now),
				productRow(pid, "C003", "counter", 200, "sum", "hourly", now),
			}},
			// call#3 backfillDisplayNames
			&fakeRows{},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity:     metrics.GranularityHourly,
		Dimension:       DimensionProduct,
		Technologies:    []string{"lte"},
		StoreAllEnabled: true,
		// 注意：MetricPaths 留空——store_all 模式不下传配置指标
	})
	require.NoError(t, err)

	got := map[string]float64{}
	for _, r := range rows {
		got[r.MetricPath] = float64(r.MetricValue)
	}
	// 已启用 counter C001/C002 透传落库
	assert.Contains(t, got, "C001", "已启用 counter 必须落库")
	assert.Contains(t, got, "C002", "已启用 counter 必须落库")
	// 已启用派生 K900 = 50/200*100 = 25 落库
	require.Contains(t, got, "K900", "已启用派生 KPI 必须重算落库")
	assert.InDelta(t, 25.0, got["K900"], 1e-9)
	// store-superset：distinct metric_path 数 = 3（C001/C002/K900）> 若只存某子集
	assert.Len(t, got, 3, "落库 metric_path 应是已启用超集")
	// store-within-enabled：dep-only 非启用 counter C003 不得落库
	assert.NotContains(t, got, "C003", "dep-only 的非启用 counter 不得作为结果行落库")
}

// ── 路径2：已启用集为空 → 降级为「全库 counter + 全库派生 KPI 重算」，不丢 counter 且不丢派生 KPI。
//
// #532 P2 回合2 修复：旧实现降级只落裸 counter、丢掉本可重算的派生 KPI（相对仅配置反而退化）。
// 现降级走 queryFullLibraryWithKPIs：counter 全留 + 全库派生 KPI 照公式重算。
func Test_QueryEnabled_EmptyEnabled_DegradesToFullLibraryNoKPILoss(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	pid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			// call#0 resolveEnabledIndicators：空集（无任何已启用指标）
			&fakeRows{rows: [][]any{}},
			// call#1 降级主聚合（queryFullLibraryWithKPIs 内部，不下推过滤、只聚 counter）
			&fakeRows{rows: [][]any{
				productRow(pid, "C001", "counter", 50, "sum", "hourly", now),
				productRow(pid, "C002", "counter", 200, "sum", "hourly", now),
			}},
			// call#2 resolveAllDerivedKPIs：全库派生 KPI（K900 = C001/C002*100）
			&fakeRows{rows: [][]any{
				kpiMetaRow("K900", "pct", "C001/C002*100"),
			}},
			// call#3 backfillDisplayNames
			&fakeRows{},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity:     metrics.GranularityHourly,
		Dimension:       DimensionProduct,
		Technologies:    []string{"lte"},
		StoreAllEnabled: true,
	})
	require.NoError(t, err)
	got := map[string]float64{}
	for _, r := range rows {
		got[r.MetricPath] = float64(r.MetricValue)
	}
	// counter 全留
	assert.Contains(t, got, "C001", "降级仍落全部 counter，不丢行")
	assert.Contains(t, got, "C002", "降级仍落全部 counter，不丢行")
	// 关键修复：派生 KPI 仍被重算落库（K900 = 50/200*100 = 25），不再退化丢 KPI
	require.Contains(t, got, "K900", "降级也必须重算派生 KPI（不丢 KPI，修 #532 P2 回合1 退化）")
	assert.InDelta(t, 25.0, got["K900"], 1e-9)
}

// ── 路径3：已启用集枚举查询失败 → 同样降级为「全库 counter + 派生 KPI 重算」，不报错、不丢 counter/KPI。
func Test_QueryEnabled_EnabledQueryError_DegradesNoError(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	pid := uuid.New()
	db := &recordingDB{
		// call#0 resolveEnabledIndicators 查询出错 → 该后缀降级（视为空集）
		errs: []error{errors.New("boom"), nil, nil, nil},
		results: []pgx.Rows{
			nil, // call#0 出错（errs[0]）
			// call#1 降级主聚合（queryFullLibraryWithKPIs 内部）
			&fakeRows{rows: [][]any{
				productRow(pid, "C001", "counter", 9, "sum", "hourly", now),
			}},
			&fakeRows{}, // call#2 resolveAllDerivedKPIs：无派生 KPI（退回全部 counter）
			&fakeRows{}, // call#3 backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity:     metrics.GranularityHourly,
		Dimension:       DimensionProduct,
		Technologies:    []string{"lte"},
		StoreAllEnabled: true,
	})
	require.NoError(t, err, "启用集查询失败应降级不报错")
	require.Len(t, rows, 1, "降级后仍返回全部 counter")
	assert.Equal(t, "C001", rows[0].MetricPath)
}

// ── resolveEnabledIndicators：多制式取并集去重 + 跨 operator 去重（DISTINCT 已在 SQL 侧，
// 这里验跨后缀并集去重）。
func Test_resolveEnabledIndicators_MultiTechUnionDedup(t *testing.T) {
	db := &recordingDB{
		results: []pgx.Rows{
			// lte → enb 表
			&fakeRows{rows: [][]any{enabledRow("X1"), enabledRow("X2")}},
			// nr → gnb 表（X2 与 enb 重复，跨后缀去重）
			&fakeRows{rows: [][]any{enabledRow("X2"), enabledRow("X3")}},
		},
	}
	a := New(db, nil, nil)
	got := a.resolveEnabledIndicators(context.Background(), []string{"lte", "nr"})
	assert.ElementsMatch(t, []string{"X1", "X2", "X3"}, got)
}

// ── #532 P2 回合2 关键回归闸：enabled_pm_indicators_* 只在主库（metaDB），时序库（db）无此表。
// resolveEnabledIndicators 必须走 metaDB——否则真实部署态在时序库查报 42P01 整段降级（回合1 的根因）。
// 这里用「db 一查就报错、metaDB 返回正常」的反向布置：若实现误用 db，会拿到错误降级返回空；
// 走 metaDB 才能拿到 X1/X2，断言据此守卫。
func Test_resolveEnabledIndicators_QueriesMetaDBNotTsDB(t *testing.T) {
	// db（时序库）：任何 enabled_pm_indicators_* 查询都报 relation 不存在（模拟真实部署态）
	tsDB := &recordingDB{errs: []error{errors.New(`relation "enabled_pm_indicators_enb" does not exist`)}}
	// metaDB（主库）：enabled 表正常返回
	pgMeta := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{enabledRow("X1"), enabledRow("X2")}}}}
	a := NewWithMeta(tsDB, pgMeta, nil, nil)
	got := a.resolveEnabledIndicators(context.Background(), []string{"lte"})
	assert.ElementsMatch(t, []string{"X1", "X2"}, got, "必须走 metaDB（主库）读启用集，而非时序库 db")
	// 守卫：时序库 db 不应被用于启用集查询（否则会撞 relation 不存在）
	assert.Empty(t, tsDB.sqls, "启用集查询不得落到时序库（db）")
	require.Len(t, pgMeta.sqls, 1, "启用集查询应落到主库（metaDB）")
}

// ── NewWithMeta：metaDB 为 nil 时退回 db（向后兼容旧构造/单测）。
func Test_NewWithMeta_NilMetaFallsBackToDB(t *testing.T) {
	db := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{enabledRow("X1")}}}}
	a := NewWithMeta(db, nil, nil, nil)
	got := a.resolveEnabledIndicators(context.Background(), []string{"lte"})
	assert.ElementsMatch(t, []string{"X1"}, got)
	require.Len(t, db.sqls, 1, "metaDB 为 nil 时退回 db 查询")
}
