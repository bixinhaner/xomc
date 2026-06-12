package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// metaRow 是 resolveKPIMetadata UNION 查询的一行：id, statis_type, arithmetic, is_counter。
// 默认 is_counter='0'（派生 KPI）。原始计数用 metaCounterRow。
func metaRow(id, statis, formula string) []any {
	return []any{id, statis, formula, "0"}
}

// metaCounterRow 是原始计数（is_counter='1'）的元数据行：arithmetic=自身编号。
func metaCounterRow(id, statis string) []any {
	return []any{id, statis, id, "1"}
}

// productRow 是 queryProductTable SELECT 的一行：
// product_id, path, type, value, statis, gran, time×4。
func productRow(pid uuid.UUID, path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{pid, path, mtype, val, statis, gran, t, t, t, t}
}

// groupTableRow 是 queryGroupTable SELECT 的一行（设备组制式治本 B 方案后带 technology 列）：
// device_group_id, technology, path, type, value, statis, gran, time×4, extra([]byte)。
func groupTableRow(gid uuid.UUID, tech, path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{gid, tech, path, mtype, val, statis, gran, t, t, t, t, []byte(nil)}
}

// ── 1. 简单 pct 重算正确，且 ≠ 各设备百分比平均 ───────────────────────────────
//
// succ=[90,10] att=[100,300]：跨设备各自 SUM → succ=100, att=400 → 100/400*100=25%。
// 各设备百分比平均 = (90% + 3.33%)/2 = 46.67%，断言取 25 不取 46.67。
func Test_Recompute_SimplePct_Network_NotDeviceAverage(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	// network 维度：DB 已把全网 counter 各自 SUM 成一条（succ=100, att=400）。
	db := &recordingDB{
		results: []pgx.Rows{
			// call#0 resolveKPIMetadata：K900010002 = succ/att*100
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			// call#1 主聚合查询：全网汇总后的 counter 行
			&fakeRows{rows: [][]any{
				networkRow("RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				networkRow("RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
			}},
			// call#2 backfillDisplayNames（KPI 行回填名）
			&fakeRows{},
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	// 仅 KPI 行（deps counter 用户没请求，被剔除）
	require.Len(t, rows, 1, "deps-only counter 不出现，仅 1 条重算 KPI 行")
	assert.Equal(t, "K900010002", rows[0].MetricPath)
	assert.Equal(t, metrics.MetricTypeKPI, rows[0].MetricType)
	assert.InDelta(t, 25.0, rows[0].MetricValue, 1e-9, "跨设备分子分母各自SUM后重算=25%")
	assert.Greater(t, 46.0, rows[0].MetricValue, "不是各设备百分比平均(46.67%)")
	require.NotNil(t, rows[0].StatisType)
	assert.Equal(t, metrics.StatisPct, *rows[0].StatisType)
}

// ── 2. 多项分子分母 pct ──────────────────────────────────────────────────────
//
// (c1+c2)/(c3+c4)*100：c1=10,c2=30,c3=100,c4=300 → 40/400*100=10%。
func Test_Recompute_MultiTermPct_Product(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	pid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900099999", "pct", "(c1+c2)/(c3+c4)*100"),
			}},
			&fakeRows{rows: [][]any{
				productRow(pid, "c1", "counter", 10, "sum", "hourly", now),
				productRow(pid, "c2", "counter", 30, "sum", "hourly", now),
				productRow(pid, "c3", "counter", 100, "sum", "hourly", now),
				productRow(pid, "c4", "counter", 300, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionProduct,
		MetricPaths: []string{"K900099999"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "K900099999", rows[0].MetricPath)
	assert.Equal(t, pid, rows[0].ProductID, "product 维度分组键透传")
	assert.InDelta(t, 10.0, rows[0].MetricValue, 1e-9)
}

// ── 3. 除零优雅：分母全 0 → 不产 KPI 行 ──────────────────────────────────────
func Test_Recompute_DivByZero_SkipsRow(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				networkRow("RRC.SuccConnEstab", "counter", 0, "sum", "hourly", now),
				networkRow("RRC.AttConnEstab", "counter", 0, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill（无 KPI 行也可能不触发，但留位安全）
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err, "除零不报错")
	assert.Empty(t, rows, "分母为0该桶该KPI跳过，不产假0")
}

// ── 4. counter + KPI 混合请求 ───────────────────────────────────────────────
//
// 请求 [RRC.AttConnEstab（用户主动 counter）, K900010002（pct KPI）]。
// 期望：用户 counter 行返回 + KPI 重算行返回；仅为重算引入的 deps（RRC.SuccConnEstab）不出现。
func Test_Recompute_CounterPlusKPI_Mixed(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			// effective counter = {RRC.AttConnEstab(用户), RRC.SuccConnEstab(dep)}
			&fakeRows{rows: [][]any{
				networkRow("RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				networkRow("RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"RRC.AttConnEstab", "K900010002"},
	})
	require.NoError(t, err)

	var counterPaths, kpiPaths []string
	for _, r := range rows {
		if r.MetricType == metrics.MetricTypeKPI {
			kpiPaths = append(kpiPaths, r.MetricPath)
		} else {
			counterPaths = append(counterPaths, r.MetricPath)
		}
	}
	assert.Equal(t, []string{"RRC.AttConnEstab"}, counterPaths, "仅用户请求的 counter 出现")
	assert.NotContains(t, counterPaths, "RRC.SuccConnEstab", "仅为重算引入的 dep 不出现")
	assert.Equal(t, []string{"K900010002"}, kpiPaths, "KPI 行重算返回")
	for _, r := range rows {
		if r.MetricType == metrics.MetricTypeKPI {
			assert.InDelta(t, 25.0, r.MetricValue, 1e-9)
		}
	}
}

// ── 5. device_group 维度同构断言 ────────────────────────────────────────────
//
// device_group 源是 pm_group_metrics_*（counter 已按组预聚合好），按 device_group_id 归组。
func Test_Recompute_DeviceGroupDimension(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	gid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				groupTableRow(gid, "lte", "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				groupTableRow(gid, "lte", "RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDeviceGroup,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "K900010002", rows[0].MetricPath)
	assert.Equal(t, gid, rows[0].DeviceGroupID, "按 device_group_id 归组透传")
	assert.Equal(t, "lte", rows[0].Technology, "制式透传到 KPI 行")
	assert.InDelta(t, 25.0, rows[0].MetricValue, 1e-9)
}

// 设备组制式治本：同一组内 lte 与 nr 的 counter 不可跨制式混算——KPI 重算分组键含制式，
// 同组同桶 lte / nr 各出一行 KPI，分子分母只取本制式 counter（lte=100/400=25%，nr=300/600=50%）。
func Test_Recompute_DeviceGroupTechnology_NoCrossTechMix(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	gid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				groupTableRow(gid, "lte", "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				groupTableRow(gid, "lte", "RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
				groupTableRow(gid, "nr", "RRC.SuccConnEstab", "counter", 300, "sum", "hourly", now),
				groupTableRow(gid, "nr", "RRC.AttConnEstab", "counter", 600, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDeviceGroup,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2, "同组 lte/nr 各一行 KPI")
	byTech := map[string]float64{}
	for _, r := range rows {
		assert.Equal(t, gid, r.DeviceGroupID)
		byTech[r.Technology] = r.MetricValue
	}
	assert.InDelta(t, 25.0, byTech["lte"], 1e-9, "lte 100/400=25%")
	assert.InDelta(t, 50.0, byTech["nr"], 1e-9, "nr 300/600=50%（不被 lte 污染）")
}

// ── 6. 多组多桶不串算 ───────────────────────────────────────────────────────
//
// 两个产品（pidA/pidB）各自 succ/att，验证 group key 隔离：A=25%，B=50%。
func Test_Recompute_MultiGroupBucket_NoCrossContamination(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	pidA, pidB := uuid.New(), uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				productRow(pidA, "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				productRow(pidA, "RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
				productRow(pidB, "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				productRow(pidB, "RRC.AttConnEstab", "counter", 200, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionProduct,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byPid := map[uuid.UUID]float64{}
	for _, r := range rows {
		byPid[r.ProductID] = r.MetricValue
	}
	assert.InDelta(t, 25.0, byPid[pidA], 1e-9, "产品A 100/400=25%")
	assert.InDelta(t, 50.0, byPid[pidB], 1e-9, "产品B 100/200=50%")
}

// ── PM-P3: arithmetic 编号公式求值成功（派生 KPI 走 arithmetic 列而非 formula 表）────
//
// 编号公式 (C000030140+C000030141)/(C000030142+C000030143)*100：
// 分子 30+10=40，分母 100+300=400 → 40/400*100=10%。验证编号公式经 arithmetic 列解析求值成功。
func Test_Recompute_ArithmeticNumberedFormula_Success(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K000010050", "pct", "(C000030140+C000030141)/(C000030142+C000030143)*100"),
			}},
			&fakeRows{rows: [][]any{
				networkRow("C000030140", "counter", 30, "sum", "hourly", now),
				networkRow("C000030141", "counter", 10, "sum", "hourly", now),
				networkRow("C000030142", "counter", 100, "sum", "hourly", now),
				networkRow("C000030143", "counter", 300, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"K000010050"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "仅 KPI 行，deps counter 用户未请求被剔除")
	assert.Equal(t, "K000010050", rows[0].MetricPath)
	assert.Equal(t, metrics.MetricTypeKPI, rows[0].MetricType)
	assert.InDelta(t, 10.0, rows[0].MetricValue, 1e-9)
}

// ── PM-P3: 依赖 counter 缺失时跳过该桶该 KPI（不产假 0）────────────────────────
//
// 公式引用 C000030142，但聚合结果里只有 C000030140（分母 counter 没出现）→ Evaluate 取不到 → 跳过。
func Test_Recompute_DepCounterMissing_SkipsNoFakeZero(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K000010051", "pct", "C000030140/C000030142*100"),
			}},
			&fakeRows{rows: [][]any{
				// 只有分子，分母 C000030142 这个桶里缺失。
				networkRow("C000030140", "counter", 50, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"K000010051"},
	})
	require.NoError(t, err, "依赖 counter 缺失不报错")
	assert.Empty(t, rows, "依赖 counter 缺失该桶该 KPI 跳过，不产假 0")
}

// ── PM-P3: 原始计数 arithmetic=自身 → 恒等透传该 counter 桶内聚合值 ──────────────
//
// 去掉 is_counter='0' 过滤后，原始计数（is_counter='1'，arithmetic=自身编号）被纳入 kpiMeta。
// recomputeKPIs 对原始计数走 passthrough：直接输出该 counter 的聚合行，
// 保留 metric_type='counter' + StatisType（不被误标成 kpi、不丢行、不假 0）。
func Test_Recompute_RawCounter_IdentityPassthrough(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			// 原始计数 C000060216（小区在服时长），arithmetic=自身，statis=sum。
			&fakeRows{rows: [][]any{
				metaCounterRow("C000060216", "sum"),
			}},
			// effective counter = {C000060216}（其 deps=自身），聚合出该 counter 行。
			&fakeRows{rows: [][]any{
				networkRow("C000060216", "counter", 31.0, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"C000060216"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "原始计数恒等透传一行，不重复不丢行")
	assert.Equal(t, "C000060216", rows[0].MetricPath)
	assert.Equal(t, metrics.MetricTypeCounter, rows[0].MetricType,
		"原始计数保持 metric_type='counter'，不被误标成 kpi")
	assert.InDelta(t, 31.0, rows[0].MetricValue, 1e-9, "输出=该 counter 桶内聚合值，非假 0")
	require.NotNil(t, rows[0].StatisType, "StatisType 正确透传")
	assert.Equal(t, metrics.StatisSum, *rows[0].StatisType)
}

// ── PM-P3: 原始计数 + 派生 KPI 混合请求 ─────────────────────────────────────────
//
// 请求 [C000060216(原始计数), K000010002(派生 pct)]：
// 原始计数 passthrough 出 counter 行；派生 KPI 重算出 kpi 行；纯 dep counter 不出现。
func Test_Recompute_RawCounterPlusDerivedKPI_Mixed(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaCounterRow("C000060216", "sum"),
				metaRow("K000010002", "pct", "C000030001/C000030002*100"),
			}},
			&fakeRows{rows: [][]any{
				networkRow("C000060216", "counter", 31, "sum", "hourly", now),
				networkRow("C000030001", "counter", 100, "sum", "hourly", now),
				networkRow("C000030002", "counter", 400, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	a := New(db, nil, nil)
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		MetricPaths: []string{"C000060216", "K000010002"},
	})
	require.NoError(t, err)

	var counterPaths, kpiPaths []string
	valByPath := map[string]float64{}
	for _, r := range rows {
		valByPath[r.MetricPath] = r.MetricValue
		if r.MetricType == metrics.MetricTypeKPI {
			kpiPaths = append(kpiPaths, r.MetricPath)
		} else {
			counterPaths = append(counterPaths, r.MetricPath)
		}
	}
	assert.Equal(t, []string{"C000060216"}, counterPaths, "原始计数 counter 行透传")
	assert.NotContains(t, counterPaths, "C000030001", "纯 dep counter 不出现")
	assert.NotContains(t, counterPaths, "C000030002", "纯 dep counter 不出现")
	assert.Equal(t, []string{"K000010002"}, kpiPaths, "派生 KPI 重算行")
	assert.InDelta(t, 31.0, valByPath["C000060216"], 1e-9)
	assert.InDelta(t, 25.0, valByPath["K000010002"], 1e-9)
}

// ── 可观测性（#194）：device_group 重算跳过某组某指标时记结构化 debug 日志 ──────────
//
// 一个设备组分母 counter 缺失（RRC.AttConnEstab 桶里没有）→ 该组该 KPI 整条被跳过、不产假 0
//（既有语义不变），但应记一条聚合日志含 group_id + metric + reason=missing:<counter>，
// 便于现场把「该组真没数据」与「某组没上报该 counter」分离，也解释前端 legend 该组消失。
func Test_Recompute_DeviceGroup_SkipLogged(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	gid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				// 只有分子 SuccConnEstab，分母 AttConnEstab 缺 → 该组该 KPI 跳过。
				groupTableRow(gid, "lte", "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	obsCore, logs := observer.New(zapcore.DebugLevel)
	a := New(db, nil, zap.New(obsCore))
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDeviceGroup,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	assert.Empty(t, rows, "分母缺失该组该 KPI 跳过，不产假 0（既有语义不变）")

	entries := logs.FilterMessage("device-group KPI recompute skipped buckets").All()
	require.Len(t, entries, 1, "应记一条聚合跳过日志")
	fields := entries[0].ContextMap()
	assert.EqualValues(t, 1, fields["skipped_total"])
	// observer ContextMap 把 zap.Strings 还原成 []interface{}。
	detailsRaw, ok := fields["details"].([]interface{})
	require.True(t, ok, "details 应为切片")
	require.Len(t, detailsRaw, 1)
	detail := detailsRaw[0].(string)
	assert.Contains(t, detail, "K900010002", "含被跳过的指标编号")
	assert.Contains(t, detail, "missing:RRC.AttConnEstab", "reason 标明缺失的 counter")
	assert.Contains(t, detail, "group="+gid.String(), "采样含 group_id 便于定位")
}

// 全部组都算得出时不应产生跳过日志（避免噪声）。
func Test_Recompute_DeviceGroup_NoSkipLogWhenAllComputed(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	gid := uuid.New()
	db := &recordingDB{
		results: []pgx.Rows{
			&fakeRows{rows: [][]any{
				metaRow("K900010002", "pct", "RRC.SuccConnEstab/RRC.AttConnEstab*100"),
			}},
			&fakeRows{rows: [][]any{
				groupTableRow(gid, "lte", "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				groupTableRow(gid, "lte", "RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
			}},
			&fakeRows{}, // backfill
		},
	}
	obsCore, logs := observer.New(zapcore.DebugLevel)
	a := New(db, nil, zap.New(obsCore))
	rows, err := a.Query(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDeviceGroup,
		MetricPaths: []string{"K900010002"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "该组该 KPI 算得出")
	assert.Empty(t, logs.FilterMessage("device-group KPI recompute skipped buckets").All(),
		"无跳过则不记日志")
}

// ── 纯函数：effectiveCounterPaths 去重并合并 deps ───────────────────────────
func Test_effectiveCounterPaths_DedupsUserAndDeps(t *testing.T) {
	kpis := []kpiMeta{
		{code: "K1", deps: []string{"a", "b"}},
		{code: "K2", deps: []string{"b", "c"}},
	}
	got := effectiveCounterPaths([]string{"a", "x"}, kpis)
	assert.ElementsMatch(t, []string{"a", "x", "b", "c"}, got)
	assert.Len(t, got, 4, "去重后无重复")
}
