package kpi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ── 测试夹具：mock 依赖 ──────────────────────────────────────────────

type mockCounterRepo struct {
	queryForKPIFunc      func(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error)
	queryForKPICellsFunc func(ctx context.Context, deviceID uuid.UUID, cellIDs []string, counterNames []string, startTime, endTime time.Time) (map[string]map[string]float64, error)
}

func (m *mockCounterRepo) BatchInsert(_ context.Context, _ []model.PMCounter) error { return nil }
func (m *mockCounterRepo) Query(_ context.Context, _ counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
}
func (m *mockCounterRepo) QueryAggregated(_ context.Context, _ counter.CounterFilter) ([]counter.AggregatedCounter, error) {
	return nil, nil
}
func (m *mockCounterRepo) QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error) {
	if m.queryForKPIFunc != nil {
		return m.queryForKPIFunc(ctx, deviceID, cellID, counterNames, startTime, endTime)
	}
	return nil, nil
}
func (m *mockCounterRepo) QueryForKPICells(ctx context.Context, deviceID uuid.UUID, cellIDs []string, counterNames []string, startTime, endTime time.Time) (map[string]map[string]float64, error) {
	if m.queryForKPICellsFunc != nil {
		return m.queryForKPICellsFunc(ctx, deviceID, cellIDs, counterNames, startTime, endTime)
	}
	// 默认：复用 queryForKPIFunc 对每个 cell 取值，方便仅设置单 cell 行为的旧测试复用。
	if m.queryForKPIFunc != nil {
		out := make(map[string]map[string]float64, len(cellIDs))
		for _, c := range cellIDs {
			v, err := m.queryForKPIFunc(ctx, deviceID, c, counterNames, startTime, endTime)
			if err != nil {
				return nil, err
			}
			out[c] = v
		}
		return out, nil
	}
	return nil, nil
}

type mockKPIRepo struct{ insertedValues []model.KPIValue }

func (m *mockKPIRepo) BatchInsert(_ context.Context, values []model.KPIValue) error {
	m.insertedValues = append(m.insertedValues, values...)
	return nil
}
func (m *mockKPIRepo) Query(_ context.Context, _ KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
}
func (m *mockKPIRepo) ListDefinitions(_ context.Context, _ *model.CarrierCode, _ *model.Technology) ([]model.KPIDefinition, error) {
	return nil, nil
}
func (m *mockKPIRepo) SyncDefinitions(_ context.Context, _ []model.KPIDefinition) error { return nil }

// stubRouter 实现 KPIRouter，按 deviceSN 返指定路由 / 错误。
type stubRouter struct {
	route *router.KPIRoute
	err   error
}

func (s *stubRouter) LookupByDevice(_ context.Context, _ string) (*router.KPIRoute, error) {
	return s.route, s.err
}

// 构造一个典型的 BLQ-LTE-V1 KPI 子集：1 个 KPI（RRC_Succ_Rate = succ/att），2 个 counter。
func sampleRoute() *router.KPIRoute {
	return &router.KPIRoute{
		ProductID:         uuid.New(),
		IndicatorPlatform: "BLQ-LTE-V1",
		KPIs: []router.KPIDef{
			{
				IndicatorID:  "K-001",
				Name:         "RRC_Succ_Rate",
				StatisType:   "pct",
				Formula:      "RRC_Conn_Succ / RRC_Conn_Att",
				Dependencies: []string{"RRC_Conn_Succ", "RRC_Conn_Att"},
			},
		},
	}
}

// ── Cases ────────────────────────────────────────────────────────────

// 主路径：router 返路由 + counterRepo 有齐全 counter → KPIEngine 计算成功
// 并按 route.KPIs[*] 落 KPIValue。
func TestKPIEngine_Calculate_ComputesRoutedKPIs(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, names []string, _, _ time.Time) (map[string]float64, error) {
			// 验证 engine 把 route.KPIs[*].Dependencies 透传给 counterRepo
			assert.ElementsMatch(t, []string{"RRC_Conn_Succ", "RRC_Conn_Att"}, names)
			return map[string]float64{"RRC_Conn_Succ": 950, "RRC_Conn_Att": 1000}, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	engine := NewKPIEngine(counterRepo, kpiRepo, &stubRouter{route: sampleRoute()}, zap.NewNop())

	values, err := engine.Calculate(
		context.Background(),
		uuid.New(), "00A0C6", "TEST-SN-001", "cell-1",
		time.Now().Add(-15*time.Minute), time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "RRC_Succ_Rate", values[0].KPIName)
	assert.InDelta(t, 0.95, values[0].KPIValue, 0.001)
	assert.Equal(t, model.CarrierCMCC, values[0].Carrier, "carrier 作为 passthrough 标签写入 KPIValue")
}

// 产品未匹配 → router 返 ErrProductNotMatched → KPIEngine 跳过设备，返空切片不抛错。
func TestKPIEngine_Calculate_SkipsWhenProductNotMatched(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(context.Context, uuid.UUID, string, []string, time.Time, time.Time) (map[string]float64, error) {
			t.Fatal("counterRepo 不应被调用：路由失败应短路")
			return nil, nil
		},
	}
	engine := NewKPIEngine(counterRepo, &mockKPIRepo{}, &stubRouter{err: router.ErrProductNotMatched}, zap.NewNop())

	values, err := engine.Calculate(
		context.Background(),
		uuid.New(), "00A0C6", "ORPHAN-SN", "cell-1",
		time.Now().Add(-15*time.Minute), time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	assert.Nil(t, values)
}

// counter 缺失 → 该 KPI 跳过（Evaluate 报 counter not found），其它 KPI 仍能跑。
func TestKPIEngine_Calculate_SkipKPIsWithMissingCounters(t *testing.T) {
	route := &router.KPIRoute{
		ProductID:         uuid.New(),
		IndicatorPlatform: "BLQ-LTE-V1",
		KPIs: []router.KPIDef{
			{
				Name:         "RRC_Succ_Rate",
				Formula:      "RRC_Conn_Succ / RRC_Conn_Att",
				Dependencies: []string{"RRC_Conn_Succ", "RRC_Conn_Att"},
			},
			{
				Name:         "ERAB_Succ_Rate",
				Formula:      "ERAB_Succ / ERAB_Att",
				Dependencies: []string{"ERAB_Succ", "ERAB_Att"},
			},
		},
	}
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
			// 只回 RRC counter，ERAB 全缺
			return map[string]float64{"RRC_Conn_Succ": 950, "RRC_Conn_Att": 1000}, nil
		},
	}
	engine := NewKPIEngine(counterRepo, &mockKPIRepo{}, &stubRouter{route: route}, zap.NewNop())

	values, err := engine.Calculate(
		context.Background(),
		uuid.New(), "00A0C6", "SN-001", "cell-1",
		time.Now().Add(-15*time.Minute), time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	require.Len(t, values, 1, "ERAB 应被 skip")
	assert.Equal(t, "RRC_Succ_Rate", values[0].KPIName)
}

// CalculateAndStore：落库走 kpiRepo.BatchInsert，且返回的值与持久化值一致。
func TestKPIEngine_CalculateAndStore_PersistsResults(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
			return map[string]float64{"RRC_Conn_Succ": 950, "RRC_Conn_Att": 1000}, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	engine := NewKPIEngine(counterRepo, kpiRepo, &stubRouter{route: sampleRoute()}, zap.NewNop())

	values, err := engine.CalculateAndStore(
		context.Background(),
		uuid.New(), "00A0C6", "SN-001", "cell-1",
		time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	require.NotEmpty(t, values)
	assert.Equal(t, len(values), len(kpiRepo.insertedValues))
}

// CalculateCellsAndStore：批量算多 cell —— route 只查一次、counter 一次查全（按 cell 分桶）、
// 所有 cell 的 KPIValue 一次性 BatchInsert，且每个 cell 落到正确 CellID。
func TestKPIEngine_CalculateCellsAndStore_BatchesAllCells(t *testing.T) {
	var routeLookups, cellsQueries int
	perCell := map[string]map[string]float64{
		"cell-1": {"RRC_Conn_Succ": 950, "RRC_Conn_Att": 1000}, // 0.95
		"cell-2": {"RRC_Conn_Succ": 800, "RRC_Conn_Att": 1000}, // 0.80
		"cell-3": {"RRC_Conn_Succ": 500, "RRC_Conn_Att": 1000}, // 0.50
	}
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(context.Context, uuid.UUID, string, []string, time.Time, time.Time) (map[string]float64, error) {
			t.Fatal("批量路径不应逐 cell 调 QueryForKPI")
			return nil, nil
		},
		queryForKPICellsFunc: func(_ context.Context, _ uuid.UUID, cellIDs, names []string, _, _ time.Time) (map[string]map[string]float64, error) {
			cellsQueries++
			assert.ElementsMatch(t, []string{"RRC_Conn_Succ", "RRC_Conn_Att"}, names)
			assert.ElementsMatch(t, []string{"cell-1", "cell-2", "cell-3"}, cellIDs)
			return perCell, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	rt := &stubRouter{route: sampleRoute()}
	engine := NewKPIEngine(counterRepo, kpiRepo, &countingRouter{inner: rt, n: &routeLookups}, zap.NewNop())

	n, err := engine.CalculateCellsAndStore(
		context.Background(),
		uuid.New(), "00A0C6", "SN-001",
		[]string{"cell-1", "cell-2", "cell-3"},
		time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, routeLookups, "route 应只查一次（hoist 出 cell 循环）")
	assert.Equal(t, 1, cellsQueries, "counter 应一次查全部 cell")
	assert.Equal(t, 3, n, "3 个 cell × 1 KPI = 3 行")
	require.Len(t, kpiRepo.insertedValues, 3, "应一次性批量落 3 行")

	byCell := map[string]float64{}
	for _, v := range kpiRepo.insertedValues {
		assert.Equal(t, "RRC_Succ_Rate", v.KPIName)
		byCell[v.CellID] = v.KPIValue
	}
	assert.InDelta(t, 0.95, byCell["cell-1"], 0.001)
	assert.InDelta(t, 0.80, byCell["cell-2"], 0.001)
	assert.InDelta(t, 0.50, byCell["cell-3"], 0.001)
}

// 空 cellIDs → 直接返回 0，不查 route、不落库。
func TestKPIEngine_CalculateCellsAndStore_EmptyCells(t *testing.T) {
	kpiRepo := &mockKPIRepo{}
	engine := NewKPIEngine(&mockCounterRepo{}, kpiRepo, &stubRouter{route: sampleRoute()}, zap.NewNop())
	n, err := engine.CalculateCellsAndStore(
		context.Background(), uuid.New(), "00A0C6", "SN-001", nil, time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, kpiRepo.insertedValues)
}

// countingRouter 包装一个 KPIRouter 统计 LookupByDevice 调用次数（验证 route hoist）。
type countingRouter struct {
	inner KPIRouter
	n     *int
}

func (c *countingRouter) LookupByDevice(ctx context.Context, sn string) (*router.KPIRoute, error) {
	*c.n++
	return c.inner.LookupByDevice(ctx, sn)
}

// router 透传非 sentinel 错误 → engine 透传给调用方（让上游决定重试 / 记账）。
func TestKPIEngine_Calculate_PropagatesRouterError(t *testing.T) {
	wantErr := errors.New("redis down")
	engine := NewKPIEngine(&mockCounterRepo{}, &mockKPIRepo{}, &stubRouter{err: wantErr}, zap.NewNop())

	_, err := engine.Calculate(
		context.Background(),
		uuid.New(), "00A0C6", "SN-001", "cell-1",
		time.Now().Add(-15*time.Minute), time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.ErrorIs(t, err, wantErr)
}

// 空路由（router 返空 KPIs）→ 直接返 nil + nil，不调 counterRepo。
func TestKPIEngine_Calculate_EmptyRouteShortCircuit(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(context.Context, uuid.UUID, string, []string, time.Time, time.Time) (map[string]float64, error) {
			t.Fatal("counterRepo 不应被调用：空 KPI 子集应短路")
			return nil, nil
		},
	}
	engine := NewKPIEngine(counterRepo, &mockKPIRepo{}, &stubRouter{route: &router.KPIRoute{}}, zap.NewNop())

	values, err := engine.Calculate(
		context.Background(),
		uuid.New(), "00A0C6", "SN-001", "cell-1",
		time.Now().Add(-15*time.Minute), time.Now(),
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	assert.Nil(t, values)
}
