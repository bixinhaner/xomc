package adhoc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ---------------------------------------------------------------------------
// Stub 依赖
// ---------------------------------------------------------------------------

type stubAggr struct {
	rowsByGran map[metrics.Granularity][]aggregator.Row
	lastReq    aggregator.QueryRequest // T-0182：捕获最近一次请求，断言维度/制式透传
}

func (s *stubAggr) Query(_ context.Context, req aggregator.QueryRequest) ([]aggregator.Row, error) {
	s.lastReq = req
	return s.rowsByGran[req.Granularity], nil
}

type stubRepo struct {
	mu           sync.Mutex
	insertedRows []ResultRow
	statusUpdates []struct {
		id       uuid.UUID
		status   Status
		progress *int
	}
}

func (s *stubRepo) Create(context.Context, CreateRequest) (uuid.UUID, error) { return uuid.Nil, nil }
func (s *stubRepo) Update(context.Context, uuid.UUID, UpdateRequest) error   { return nil }
func (s *stubRepo) Get(context.Context, uuid.UUID) (*Task, error)            { return nil, nil }
func (s *stubRepo) List(context.Context, ListFilter) ([]Task, error)         { return nil, nil }
func (s *stubRepo) Cancel(context.Context, uuid.UUID) error                  { return nil }
func (s *stubRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (s *stubRepo) LockNextPending(context.Context, string) (*Task, error)   { return nil, nil }

func (s *stubRepo) UpdateStatus(_ context.Context, id uuid.UUID, status Status, progress *int, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusUpdates = append(s.statusUpdates, struct {
		id       uuid.UUID
		status   Status
		progress *int
	}{id, status, progress})
	return nil
}

func (s *stubRepo) InsertResults(_ context.Context, rows []ResultRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertedRows = append(s.insertedRows, rows...)
	return nil
}

func (s *stubRepo) NextRunSeq(context.Context, uuid.UUID) (int, error)  { return 1, nil }
func (s *stubRepo) InsertRun(context.Context, TaskRun) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *stubRepo) FinishRun(context.Context, uuid.UUID, Status, int, string) error { return nil }
func (s *stubRepo) ListRuns(context.Context, uuid.UUID, int, int) ([]TaskRun, error) {
	return nil, nil
}

type stubPublisher struct {
	mu     sync.Mutex
	events []struct {
		subject string
		payload any
	}
}

func (s *stubPublisher) Publish(_ context.Context, subject string, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, struct {
		subject string
		payload any
	}{subject, payload})
	return nil
}

// ---------------------------------------------------------------------------
// 3 case：ExecuteOneshot 主路径
// ---------------------------------------------------------------------------

func Test_Executor_ExecuteOneshot_SingleGranularity(t *testing.T) {
	stype := metrics.StatisSum
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {
				{
					DeviceOUI: "A", DeviceSN: "S1",
					MetricPath: "L.Cell.Avail.Dur", MetricType: metrics.MetricTypeCounter,
					MetricValue: 700, StatisType: &stype, Granularity: metrics.GranularityHourly,
					Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	repo := &stubRepo{}
	pub := &stubPublisher{}
	e := NewExecutor(aggr, repo, pub, nil)

	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		DeviceSNs:     []string{"S1"},
		MetricPaths:   []string{"L.Cell.Avail.Dur"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}

	n, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Len(t, repo.insertedRows, 1)
	assert.Equal(t, task.ID, repo.insertedRows[0].TaskID)
	assert.Equal(t, "sum", *repo.insertedRows[0].StatisType)

	// 1 个粒度 → 100% progress 一次
	require.Len(t, repo.statusUpdates, 1)
	assert.Equal(t, 100, *repo.statusUpdates[0].progress)

	// 1 个进度事件
	require.Len(t, pub.events, 1)
	assert.Equal(t, SubjectProgress, pub.events[0].subject)
}

func Test_Executor_ExecuteOneshot_MultiGranularityProgress(t *testing.T) {
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 1, Granularity: metrics.GranularityHourly}},
			metrics.GranularityDaily:  {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 2, Granularity: metrics.GranularityDaily}},
			metrics.GranularityWeekly: {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "X", MetricType: metrics.MetricTypeCounter, MetricValue: 3, Granularity: metrics.GranularityWeekly}},
		},
	}
	repo := &stubRepo{}
	pub := &stubPublisher{}
	e := NewExecutor(aggr, repo, pub, nil)

	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly", "daily", "weekly"},
		DeviceSNs:     []string{"S1"},
		WindowStart:   time.Now().Add(-time.Hour),
		WindowEnd:     time.Now(),
	}

	n, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Len(t, repo.insertedRows, 3)

	// 3 个粒度 → 3 次 progress 上报 (33/66/99? or 33/66/100? 100/3*1=33, 100*2/3=66, 100*3/3=100)
	require.Len(t, repo.statusUpdates, 3)
	assert.Equal(t, 33, *repo.statusUpdates[0].progress)
	assert.Equal(t, 66, *repo.statusUpdates[1].progress)
	assert.Equal(t, 100, *repo.statusUpdates[2].progress)

	require.Len(t, pub.events, 3)
}

func Test_Executor_ExecuteOneshot_NoGranularitiesError(t *testing.T) {
	e := NewExecutor(&stubAggr{}, &stubRepo{}, nil, nil)
	task := &Task{ID: uuid.New(), Granularities: nil}
	_, err := e.ExecuteOneshot(context.Background(), task)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// T-0182：存储开关两档（store_all_metrics true/false）
// ---------------------------------------------------------------------------

// 构造一个含 3 个不同 metric_path 行的 stub（hourly 粒度），任务只声明其中 2 个。
func newThreeMetricAggr() *stubAggr {
	stype := metrics.StatisSum
	mkRow := func(path string, v float64) aggregator.Row {
		return aggregator.Row{
			DeviceOUI: "A", DeviceSN: "S1",
			MetricPath: path, MetricType: metrics.MetricTypeCounter,
			MetricValue: jsonx.Float(v), StatisType: &stype, Granularity: metrics.GranularityHourly,
			Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		}
	}
	return &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {mkRow("M.kept.1", 1), mkRow("M.kept.2", 2), mkRow("M.dropped.3", 3)},
		},
	}
}

func newStoreSwitchTask() *Task {
	return &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		DeviceSNs:     []string{"S1"},
		MetricPaths:   []string{"M.kept.1", "M.kept.2"}, // 任务只选 2 个
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
}

// store_all_metrics=true（默认）：聚合返回 3 行全部落库（不按 task.MetricPaths 过滤）。
func Test_Executor_StoreAllMetrics_True_StoresAll(t *testing.T) {
	repo := &stubRepo{}
	e := NewExecutor(newThreeMetricAggr(), repo, nil, nil) // 默认 true
	n, err := e.ExecuteOneshot(context.Background(), newStoreSwitchTask())
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Len(t, repo.insertedRows, 3)
}

// store_all_metrics=false（仅存所选）：落库前按 task.MetricPaths 过滤，只落匹配的 2 行。
func Test_Executor_StoreAllMetrics_False_FiltersToSelected(t *testing.T) {
	repo := &stubRepo{}
	e := NewExecutor(newThreeMetricAggr(), repo, nil, nil).SetStoreAllMetrics(false)
	n, err := e.ExecuteOneshot(context.Background(), newStoreSwitchTask())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.Len(t, repo.insertedRows, 2)
	got := map[string]bool{}
	for _, r := range repo.insertedRows {
		got[r.MetricPath] = true
	}
	assert.True(t, got["M.kept.1"])
	assert.True(t, got["M.kept.2"])
	assert.False(t, got["M.dropped.3"], "dropped metric must not be stored when store_all_metrics=false")
}

// ---------------------------------------------------------------------------
// T-0182：product 维度路由 + 制式（Technology）透传到 aggregator.QueryRequest
// ---------------------------------------------------------------------------

func Test_Executor_ProductDimension_RoutesAndPassesTechnology(t *testing.T) {
	pid := uuid.New() // T-0182-fix：product 维度聚合产出的分组键，必须透传到落库行
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{
				ProductID:  pid,
				MetricPath: "M.x", MetricType: metrics.MetricTypeCounter,
				MetricValue: 9, Granularity: metrics.GranularityHourly,
				Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			}},
		},
	}
	repo := &stubRepo{}
	e := NewExecutor(aggr, repo, nil, nil)
	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		Dimension:     DimensionProduct,
		Technology:    "lte",
		MetricPaths:   []string{"M.x"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	// 维度映射到 aggregator 侧 product 枚举
	assert.Equal(t, aggregator.DimensionProduct, aggr.lastReq.Dimension)
	// 制式透传为单元素 Technologies
	assert.Equal(t, []string{"lte"}, aggr.lastReq.Technologies)

	// T-0182-fix：落库行必须带回 product_id 分组键（非 Nil 且等于聚合产出值），
	// 否则 product 维度多产品结果无法区分（主线端到端发现的缺陷）。
	require.Len(t, repo.insertedRows, 1)
	assert.NotEqual(t, uuid.Nil, repo.insertedRows[0].ProductID, "product 维度落库行 ProductID 不能为 Nil")
	assert.Equal(t, pid, repo.insertedRows[0].ProductID, "落库行 ProductID 必须等于聚合产出的 product_id")
}

// 制式为空时不透传（Technologies 应为 nil，避免误过滤掉全部设备）。
func Test_Executor_EmptyTechnology_NoTechnologiesPassed(t *testing.T) {
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{DeviceOUI: "A", DeviceSN: "S1", MetricPath: "M.x", MetricType: metrics.MetricTypeCounter, MetricValue: 1, Granularity: metrics.GranularityHourly}},
		},
	}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil)
	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		DeviceSNs:     []string{"S1"},
		WindowStart:   time.Now().Add(-time.Hour),
		WindowEnd:     time.Now(),
	}
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)
	assert.Nil(t, aggr.lastReq.Technologies)
}

// ---------------------------------------------------------------------------
// T-0184：network 维度路由 + device_group 维度路由（复用 G5 预聚合）
// ---------------------------------------------------------------------------

// network 维度映射到 aggregator 侧 network 枚举，制式透传，DeviceGroupIDs 留空。
func Test_Executor_NetworkDimension_RoutesAndPassesTechnology(t *testing.T) {
	stype := metrics.StatisSum
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{
				DeviceSN: "AGGREGATED",
				MetricPath: "C000060011", MetricType: metrics.MetricTypeCounter,
				MetricValue: 99999, StatisType: &stype, Granularity: metrics.GranularityHourly,
				Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			}},
		},
	}
	repo := &stubRepo{}
	e := NewExecutor(aggr, repo, nil, nil)
	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		Dimension:     DimensionNetwork,
		Technology:    "lte",
		MetricPaths:   []string{"C000060011"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	assert.Equal(t, aggregator.DimensionNetwork, aggr.lastReq.Dimension, "network 维度映射到 aggregator network")
	assert.Equal(t, []string{"lte"}, aggr.lastReq.Technologies, "制式透传")
	assert.Empty(t, aggr.lastReq.DeviceGroupIDs, "network 不带 DeviceGroupIDs")
	// 全网总线落库行无 group 身份键（ObjectLDN 不被改写）
	require.Len(t, repo.insertedRows, 1)
	assert.Nil(t, repo.insertedRows[0].ObjectLDN, "network 总线落库行无 object_ldn")
}

// device_group 维度映射到 aggregator 既有 device_group 枚举（复用 pm_group_metrics_*），
// DeviceGroupIDs 留空=全部组；hour 粒度可跑；组 id 经 ObjectLDN='DeviceGroup=<uuid>' 承载。
func Test_Executor_DeviceGroupDimension_ReusesPreaggAndCarriesGroupID(t *testing.T) {
	gid := uuid.New()
	stype := metrics.StatisSum
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{
				DeviceGroupID: gid, // group 维度产出的分组键
				MetricPath:    "C000060011", MetricType: metrics.MetricTypeCounter,
				MetricValue: 500, StatisType: &stype, Granularity: metrics.GranularityHourly,
				Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			}},
		},
	}
	repo := &stubRepo{}
	e := NewExecutor(aggr, repo, nil, nil)
	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		Dimension:     DimensionDeviceGroup,
		Technology:    "lte",
		MetricPaths:   []string{"C000060011"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	assert.Equal(t, aggregator.DimensionDeviceGroup, aggr.lastReq.Dimension, "device_group 映射到 aggregator 既有 device_group")
	assert.Empty(t, aggr.lastReq.DeviceGroupIDs, "DeviceGroupIDs 留空=按全部组分组")
	assert.Equal(t, []string{"lte"}, aggr.lastReq.Technologies, "制式透传")
	// 组身份经 ObjectLDN 承载（无独立 device_group_id 列，复用 object_ldn）
	require.Len(t, repo.insertedRows, 1)
	require.NotNil(t, repo.insertedRows[0].ObjectLDN)
	assert.Equal(t, "DeviceGroup="+gid.String(), *repo.insertedRows[0].ObjectLDN, "组 id 编入 object_ldn")
	assert.Equal(t, float64(500), repo.insertedRows[0].MetricValue)
}

// ---------------------------------------------------------------------------
// #532 P2：store_all_metrics=true 时执行器不下传 task 配置指标，改置 StoreAllEnabled
// 让聚合层按已启用集全存；=false 时仍下传配置指标且不置 StoreAllEnabled。
// ---------------------------------------------------------------------------

// store_all=true：请求里 MetricPaths 必须被清空、StoreAllEnabled=true（落库侧全存已启用前提）。
func Test_Executor_StoreAll_True_RequestDropsMetricPathsAndSetsEnabled(t *testing.T) {
	aggr := newThreeMetricAggr()
	e := NewExecutor(aggr, &stubRepo{}, nil, nil) // 默认 store_all=true
	task := newStoreSwitchTask()
	task.Dimension = DimensionProduct
	task.Technology = "lte"
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	assert.True(t, aggr.lastReq.StoreAllEnabled, "store_all=true 必须置 StoreAllEnabled")
	assert.Empty(t, aggr.lastReq.MetricPaths, "store_all=true 不得下传 task 配置指标（否则数据库层先筛掉非配置指标）")
	assert.False(t, aggr.lastReq.RecomputeAllKPIs, "product 维度不走全库 RecomputeAllKPIs")
}

// store_all=false：请求里仍下传 task 配置指标、不置 StoreAllEnabled（仅存所选口径不变）。
func Test_Executor_StoreAll_False_RequestKeepsMetricPathsNoEnabled(t *testing.T) {
	aggr := newThreeMetricAggr()
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetStoreAllMetrics(false)
	task := newStoreSwitchTask()
	task.Dimension = DimensionProduct
	task.Technology = "lte"
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	assert.False(t, aggr.lastReq.StoreAllEnabled, "store_all=false 不得置 StoreAllEnabled")
	assert.Equal(t, []string{"M.kept.1", "M.kept.2"}, aggr.lastReq.MetricPaths, "store_all=false 必须下传 task 配置指标")
}

// store_all=true 且 network 维度：仍走全库 RecomputeAllKPIs（既有特例自洽），且不置 StoreAllEnabled
// （两条全聚路径互斥，避免打架）。
func Test_Executor_StoreAll_True_NetworkStillUsesRecomputeAllKPIs(t *testing.T) {
	aggr := &stubAggr{
		rowsByGran: map[metrics.Granularity][]aggregator.Row{
			metrics.GranularityHourly: {{
				DeviceSN: "AGGREGATED", MetricPath: "M.x", MetricType: metrics.MetricTypeCounter,
				MetricValue: 1, Granularity: metrics.GranularityHourly,
				Time: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
			}},
		},
	}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil) // store_all=true
	task := &Task{
		ID:            uuid.New(),
		Granularities: []string{"hourly"},
		Dimension:     DimensionNetwork,
		Technology:    "lte",
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	}
	_, err := e.ExecuteOneshot(context.Background(), task)
	require.NoError(t, err)

	assert.True(t, aggr.lastReq.RecomputeAllKPIs, "network 维度全存仍走全库 RecomputeAllKPIs（既有特例口径）")
	assert.False(t, aggr.lastReq.StoreAllEnabled, "network 维度与 StoreAllEnabled 互斥，避免双全聚路径打架")
}

// ---------------------------------------------------------------------------
// T-0182：filterByMetricPaths 纯函数（空 allowed 不过滤 / 非空只留匹配）
// ---------------------------------------------------------------------------

func Test_filterByMetricPaths(t *testing.T) {
	rows := []ResultRow{
		{MetricPath: "a"}, {MetricPath: "b"}, {MetricPath: "c"},
	}

	t.Run("empty allowed does not filter", func(t *testing.T) {
		got := filterByMetricPaths(append([]ResultRow(nil), rows...), nil)
		assert.Len(t, got, 3)
	})

	t.Run("non-empty allowed keeps only matches", func(t *testing.T) {
		got := filterByMetricPaths(append([]ResultRow(nil), rows...), []string{"a", "c"})
		require.Len(t, got, 2)
		paths := []string{got[0].MetricPath, got[1].MetricPath}
		assert.Contains(t, paths, "a")
		assert.Contains(t, paths, "c")
		assert.NotContains(t, paths, "b")
	})

	t.Run("allowed with no overlap yields empty", func(t *testing.T) {
		got := filterByMetricPaths(append([]ResultRow(nil), rows...), []string{"z"})
		assert.Len(t, got, 0)
	})
}

// ---------------------------------------------------------------------------
// T-0182-fix：nullableUUID 把 uuid.Nil 映射为 SQL NULL（product_id 列落库用）
// ---------------------------------------------------------------------------

func Test_nullableUUID(t *testing.T) {
	t.Run("Nil maps to SQL NULL", func(t *testing.T) {
		assert.Nil(t, nullableUUID(uuid.Nil), "uuid.Nil 必须落 SQL NULL（device/aggregate_group 维度无 product_id）")
	})
	t.Run("non-Nil maps to value", func(t *testing.T) {
		id := uuid.New()
		got := nullableUUID(id)
		require.NotNil(t, got)
		assert.Equal(t, id, got, "非空 product_id 必须原样落库")
	})
}
