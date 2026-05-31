package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// metaRow 是 resolveKPIMetadata UNION 查询的一行：id, statis_type, formula。
func metaRow(id, statis, formula string) []any {
	return []any{id, statis, formula}
}

// productRow 是 queryProductTable SELECT 的一行：
// product_id, path, type, value, statis, gran, time×4。
func productRow(pid uuid.UUID, path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{pid, path, mtype, val, statis, gran, t, t, t, t}
}

// groupTableRow 是 queryGroupTable SELECT 的一行：
// device_group_id, path, type, value, statis, gran, time×4, extra([]byte)。
func groupTableRow(gid uuid.UUID, path, mtype string, val float64, statis, gran string, t time.Time) []any {
	return []any{gid, path, mtype, val, statis, gran, t, t, t, t, []byte(nil)}
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
				groupTableRow(gid, "RRC.SuccConnEstab", "counter", 100, "sum", "hourly", now),
				groupTableRow(gid, "RRC.AttConnEstab", "counter", 400, "sum", "hourly", now),
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
	assert.InDelta(t, 25.0, rows[0].MetricValue, 1e-9)
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
