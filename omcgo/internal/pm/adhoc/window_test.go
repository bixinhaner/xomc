package adhoc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ---------------------------------------------------------------------------
// #528 P2：持续任务取数口径改为「上游完成水位」驱动（替换固定 −2 格猜测）。
// ---------------------------------------------------------------------------

// stubWatermark 是 WatermarkReader 的测试桩。
//
// byKey 命中则返回对应水位；未命中返回 ErrWatermarkNotFound（模拟「上游该粒度尚未卷完任何格」）。
// 记录最近一次读取的 (粒度, 层级)，便于断言维度→层级映射。
type stubWatermark struct {
	byKey     map[wmKey]time.Time
	lastGran  metrics.Granularity
	lastLevel aggregator.WatermarkLevel
	getErr    error // 非 nil 时所有 Get 返回此错误（模拟读取出错）
}

type wmKey struct {
	gran  metrics.Granularity
	level aggregator.WatermarkLevel
}

func (s *stubWatermark) Get(_ context.Context, gran metrics.Granularity, level aggregator.WatermarkLevel) (*aggregator.Watermark, error) {
	s.lastGran = gran
	s.lastLevel = level
	if s.getErr != nil {
		return nil, s.getErr
	}
	bucket, ok := s.byKey[wmKey{gran, level}]
	if !ok {
		return nil, aggregator.ErrWatermarkNotFound
	}
	return &aggregator.Watermark{
		Granularity:          gran,
		Level:                level,
		CompletedBucketStart: bucket,
	}, nil
}

// truncateBucketStart 把任意时刻对齐到各粒度的桶起点（水位防御性对齐）。
func TestTruncateBucketStart_PerGranularity(t *testing.T) {
	loc := time.UTC
	// 2026-06-16（周二）18:44:33。
	t0 := time.Date(2026, 6, 16, 18, 44, 33, 0, loc)
	cases := []struct {
		name string
		g    metrics.Granularity
		want time.Time
	}{
		{"15min → 18:30", metrics.Granularity15Min, time.Date(2026, 6, 16, 18, 30, 0, 0, loc)},
		{"hourly → 18:00", metrics.GranularityHourly, time.Date(2026, 6, 16, 18, 0, 0, 0, loc)},
		{"daily → 06-16 00:00", metrics.GranularityDaily, time.Date(2026, 6, 16, 0, 0, 0, 0, loc)},
		{"weekly → 周一 06-15 00:00", metrics.GranularityWeekly, time.Date(2026, 6, 15, 0, 0, 0, 0, loc)},
		{"monthly → 06-01 00:00", metrics.GranularityMonthly, time.Date(2026, 6, 1, 0, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateBucketStart(c.g, t0, loc)
			assert.True(t, got.Equal(c.want), "got %v want %v", got, c.want)
		})
	}
}

func TestTruncateBucketStart_NilLocDefaultsUTC(t *testing.T) {
	t0 := time.Date(2026, 6, 16, 18, 44, 0, 0, time.UTC)
	got := truncateBucketStart(metrics.GranularityDaily, t0, nil)
	want := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	assert.True(t, got.Equal(want), "got %v want %v", got, want)
}

func TestNextBucketStart_PerGranularity(t *testing.T) {
	loc := time.UTC
	cases := []struct {
		name   string
		g      metrics.Granularity
		bucket time.Time
		want   time.Time
	}{
		{
			name:   "15min",
			g:      metrics.Granularity15Min,
			bucket: time.Date(2026, 7, 18, 9, 0, 0, 0, loc),
			want:   time.Date(2026, 7, 18, 9, 15, 0, 0, loc),
		},
		{
			name:   "hourly",
			g:      metrics.GranularityHourly,
			bucket: time.Date(2026, 7, 18, 9, 0, 0, 0, loc),
			want:   time.Date(2026, 7, 18, 10, 0, 0, 0, loc),
		},
		{
			name:   "daily",
			g:      metrics.GranularityDaily,
			bucket: time.Date(2026, 7, 18, 0, 0, 0, 0, loc),
			want:   time.Date(2026, 7, 19, 0, 0, 0, 0, loc),
		},
		{
			name:   "weekly",
			g:      metrics.GranularityWeekly,
			bucket: time.Date(2026, 7, 13, 0, 0, 0, 0, loc),
			want:   time.Date(2026, 7, 20, 0, 0, 0, 0, loc),
		},
		{
			name:   "monthly jul",
			g:      metrics.GranularityMonthly,
			bucket: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
			want:   time.Date(2026, 8, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "monthly jan",
			g:      metrics.GranularityMonthly,
			bucket: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			want:   time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "monthly feb",
			g:      metrics.GranularityMonthly,
			bucket: time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
			want:   time.Date(2026, 3, 1, 0, 0, 0, 0, loc),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := nextBucketStart(c.g, c.bucket)
			assert.True(t, got.Equal(c.want), "got %v want %v", got, c.want)
		})
	}
}

func TestIsContinuous(t *testing.T) {
	nonZero := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)

	// Mode==continuous → true（即使窗口非零也按持续）。
	assert.True(t, isContinuous(&Task{Mode: ModeContinuous, WindowStart: nonZero, WindowEnd: nonZero}))
	// 零窗口（创建清窗存 NULL）→ true。
	assert.True(t, isContinuous(&Task{Mode: ""}))
	// oneshot + 非零窗口 → false。
	assert.False(t, isContinuous(&Task{Mode: ModeOneshot, WindowStart: nonZero, WindowEnd: nonZero}))
}

// 成功路径：持续任务查询窗口 == [对应水位桶起点, 下一桶起点)。
func TestQueryAndConvert_ContinuousTargetsWatermarkBucket(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		name string
		gran metrics.Granularity
		wm   time.Time
		want time.Time
	}{
		{
			name: "hourly",
			gran: metrics.GranularityHourly,
			wm:   time.Date(2026, 6, 16, 17, 0, 0, 0, loc),
			want: time.Date(2026, 6, 16, 18, 0, 0, 0, loc),
		},
		{
			name: "daily",
			gran: metrics.GranularityDaily,
			wm:   time.Date(2026, 7, 18, 0, 0, 0, 0, loc),
			want: time.Date(2026, 7, 19, 0, 0, 0, 0, loc),
		},
		{
			name: "weekly",
			gran: metrics.GranularityWeekly,
			wm:   time.Date(2026, 7, 13, 0, 0, 0, 0, loc),
			want: time.Date(2026, 7, 20, 0, 0, 0, 0, loc),
		},
		{
			name: "monthly-jan",
			gran: metrics.GranularityMonthly,
			wm:   time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			want: time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
		},
		{
			name: "monthly-feb",
			gran: metrics.GranularityMonthly,
			wm:   time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
			want: time.Date(2026, 3, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			aggr := &stubAggr{}
			wm := &stubWatermark{byKey: map[wmKey]time.Time{
				{c.gran, aggregator.WatermarkLevelDevice}: c.wm,
			}}
			e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

			task := &Task{ID: uuid.New(), Mode: ModeContinuous, Granularities: []string{string(c.gran)}}
			_, err := e.queryAndConvert(context.Background(), task, c.gran)
			require.NoError(t, err)

			// 目标桶用半开窗口 [bucket, nextBucket)，避免 [bucket,bucket) 空区间。
			assert.True(t, aggr.lastReq.StartTime.Equal(c.wm), "start 应=水位桶 %v，实际 %v", c.wm, aggr.lastReq.StartTime)
			assert.True(t, aggr.lastReq.EndTime.Equal(c.want), "end 应=水位下一桶 %v，实际 %v", c.want, aggr.lastReq.EndTime)
		})
	}
}

// 不扑空：水位未到（无该粒度水位）→ 本格不取数（aggregator 不被调用、无结果）。
func TestQueryAndConvert_ContinuousSkipsWhenNoWatermark(t *testing.T) {
	loc := time.UTC
	aggr := &stubAggr{}
	wm := &stubWatermark{byKey: map[wmKey]time.Time{}} // 空 = 任何粒度都返回 NotFound
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

	task := &Task{ID: uuid.New(), Mode: ModeContinuous, Granularities: []string{"hourly"}}
	rows, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err)
	assert.Empty(t, rows, "水位未到应不取数")
	// aggregator 未被调用（lastReq 仍是零值）。
	assert.True(t, aggr.lastReq.StartTime.IsZero(), "水位未到不应调用 aggregator")
}

// 边界：空格水位照推（水位指向一个上游确实卷过但无数据的格）→ 游标照进、目标桶==水位、不卡死。
// aggregator 对该格返回 0 行属正常（不视为错误），下一次水位推进后会取下一格。
func TestQueryAndConvert_ContinuousAdvancesOnEmptyBucketWatermark(t *testing.T) {
	loc := time.UTC
	emptyBucket := time.Date(2026, 6, 16, 6, 0, 0, 0, loc) // 上游卷过但无数据的空格，水位仍推到此

	aggr := &stubAggr{} // rowsByGran 为 nil → 该格返回 0 行（空格）
	wm := &stubWatermark{byKey: map[wmKey]time.Time{
		{metrics.GranularityHourly, aggregator.WatermarkLevelDevice}: emptyBucket,
	}}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

	task := &Task{ID: uuid.New(), Mode: ModeContinuous, Granularities: []string{"hourly"}}
	rows, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err, "空格水位不应报错（不卡死）")
	assert.Empty(t, rows, "空格无数据行")
	// 目标桶仍精确落在水位格（游标照进，未被空格挡住）。
	assert.True(t, aggr.lastReq.StartTime.Equal(emptyBucket), "目标桶应=空格水位 %v，实际 %v", emptyBucket, aggr.lastReq.StartTime)
	assert.True(t, aggr.lastReq.EndTime.Equal(emptyBucket.Add(time.Hour)), "目标窗口结束应=下一小时桶")
}

// GSM 最近桶没有数据时也应正常查询半开窗口并返回空结果，不能因为空结果卡住。
func TestQueryAndConvert_ContinuousGSMEmptyBucketUsesHalfOpenWindow(t *testing.T) {
	loc := time.UTC
	cases := []struct {
		name   string
		gran   metrics.Granularity
		bucket time.Time
		want   time.Time
	}{
		{
			name:   "15min",
			gran:   metrics.Granularity15Min,
			bucket: time.Date(2026, 7, 18, 9, 15, 0, 0, loc),
			want:   time.Date(2026, 7, 18, 9, 30, 0, 0, loc),
		},
		{
			name:   "hourly",
			gran:   metrics.GranularityHourly,
			bucket: time.Date(2026, 7, 18, 9, 0, 0, 0, loc),
			want:   time.Date(2026, 7, 18, 10, 0, 0, 0, loc),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			aggr := &stubAggr{}
			wm := &stubWatermark{byKey: map[wmKey]time.Time{
				{c.gran, aggregator.WatermarkLevelDevice}: c.bucket,
			}}
			e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

			task := &Task{
				ID:            uuid.New(),
				Mode:          ModeContinuous,
				Dimension:     DimensionNetwork,
				Technology:    "gsm",
				Granularities: []string{string(c.gran)},
			}
			rows, err := e.queryAndConvert(context.Background(), task, c.gran)
			require.NoError(t, err)

			assert.Empty(t, rows)
			assert.Equal(t, []string{"gsm"}, aggr.lastReq.Technologies)
			assert.True(t, aggr.lastReq.StartTime.Equal(c.bucket), "start 应=GSM 空桶水位")
			assert.True(t, aggr.lastReq.EndTime.Equal(c.want), "end 应=下一桶")
		})
	}
}

// 维度→层级：device_group 维度看组级水位。
func TestQueryAndConvert_ContinuousDeviceGroupUsesGroupWatermark(t *testing.T) {
	loc := time.UTC
	groupBucket := time.Date(2026, 6, 16, 17, 0, 0, 0, loc)

	aggr := &stubAggr{}
	wm := &stubWatermark{byKey: map[wmKey]time.Time{
		{metrics.GranularityHourly, aggregator.WatermarkLevelGroup}: groupBucket,
	}}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

	task := &Task{ID: uuid.New(), Mode: ModeContinuous, Dimension: DimensionDeviceGroup, Granularities: []string{"hourly"}}
	_, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err)

	assert.Equal(t, aggregator.WatermarkLevelGroup, wm.lastLevel, "device_group 维度应读组级水位")
	assert.True(t, aggr.lastReq.StartTime.Equal(groupBucket), "目标桶应=组级水位 %v", groupBucket)
	assert.True(t, aggr.lastReq.EndTime.Equal(groupBucket.Add(time.Hour)), "目标窗口结束应=下一小时桶")
}

// 维度→层级：device/band/network/product/aggregate_group 维度均看设备级水位。
func TestQueryAndConvert_ContinuousNonGroupDimsUseDeviceWatermark(t *testing.T) {
	loc := time.UTC
	devBucket := time.Date(2026, 6, 16, 17, 0, 0, 0, loc)

	dims := []Dimension{
		DimensionDevice,
		DimensionBand,
		DimensionNetwork,
		DimensionProduct,
		DimensionAggregateGroup,
	}
	for _, dim := range dims {
		t.Run(string(dim), func(t *testing.T) {
			aggr := &stubAggr{}
			wm := &stubWatermark{byKey: map[wmKey]time.Time{
				{metrics.GranularityHourly, aggregator.WatermarkLevelDevice}: devBucket,
			}}
			e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

			task := &Task{ID: uuid.New(), Mode: ModeContinuous, Dimension: dim, Granularities: []string{"hourly"}}
			_, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
			require.NoError(t, err)

			assert.Equal(t, aggregator.WatermarkLevelDevice, wm.lastLevel, "%s 维度应读设备级水位", dim)
			assert.True(t, aggr.lastReq.StartTime.Equal(devBucket), "%s 目标桶应=设备级水位 %v", dim, devBucket)
			assert.True(t, aggr.lastReq.EndTime.Equal(devBucket.Add(time.Hour)), "%s 目标窗口结束应=下一小时桶", dim)
		})
	}
}

// nil 水位读取器（未注入）→ 持续任务保守不取数，不回归不 panic。
func TestQueryAndConvert_ContinuousNoReaderSkips(t *testing.T) {
	loc := time.UTC
	aggr := &stubAggr{}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc) // 未 SetWatermarkReader

	task := &Task{ID: uuid.New(), Mode: ModeContinuous, Granularities: []string{"hourly"}}
	rows, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err)
	assert.Empty(t, rows, "未注入水位读取器应不取数")
	assert.True(t, aggr.lastReq.StartTime.IsZero(), "未注入读取器不应调用 aggregator")
}

// 读取出错（非 NotFound）→ 保守跳过本格，不取数、不越过水位。
func TestQueryAndConvert_ContinuousWatermarkReadErrorSkips(t *testing.T) {
	loc := time.UTC
	aggr := &stubAggr{}
	wm := &stubWatermark{getErr: assert.AnError}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

	task := &Task{ID: uuid.New(), Mode: ModeContinuous, Granularities: []string{"hourly"}}
	rows, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err, "读取出错应保守跳过而非整任务失败")
	assert.Empty(t, rows)
	assert.True(t, aggr.lastReq.StartTime.IsZero(), "读取出错不应调用 aggregator")
}

// oneshot 任务的固定用户窗口保持原样，不被水位覆盖。
func TestQueryAndConvert_OneshotWindowUnchanged(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, loc)
	end := time.Date(2026, 5, 22, 0, 0, 0, 0, loc)

	aggr := &stubAggr{}
	// 即使注入了水位读取器，oneshot 也不读水位。
	wm := &stubWatermark{byKey: map[wmKey]time.Time{
		{metrics.GranularityHourly, aggregator.WatermarkLevelDevice}: time.Date(2026, 6, 16, 17, 0, 0, 0, loc),
	}}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc).SetWatermarkReader(wm)

	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
		WindowStart:   start,
		WindowEnd:     end,
	}
	_, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err)

	assert.True(t, aggr.lastReq.StartTime.Equal(start), "oneshot start 应保持用户窗口 %v，实际 %v", start, aggr.lastReq.StartTime)
	assert.True(t, aggr.lastReq.EndTime.Equal(end), "oneshot end 应保持用户窗口 %v，实际 %v", end, aggr.lastReq.EndTime)
}

// ---------------------------------------------------------------------------
// #528 P3：新建持续任务初始游标 = 建任务时刻当前对应水位桶起点（从「现在」起算，不回扫历史）
// ---------------------------------------------------------------------------

// 初始游标 == 当前水位桶起点（相差 0 格）：网络维度看设备级水位，小时粒度。
func TestInitialCursorForContinuous_EqualsCurrentWatermark(t *testing.T) {
	loc := time.UTC
	wmBucket := time.Date(2026, 6, 16, 17, 0, 0, 0, loc)
	wm := &stubWatermark{byKey: map[wmKey]time.Time{
		{metrics.GranularityHourly, aggregator.WatermarkLevelDevice}: wmBucket,
	}}
	repo := NewPgRepository(nil, nil).SetWatermarkReader(wm).SetLocationFunc(func() *time.Location { return loc })

	cursor, ok := repo.initialCursorForContinuous(context.Background(), CreateRequest{
		Mode:          ModeContinuous,
		Granularities: []string{string(metrics.GranularityHourly)},
		Dimension:     DimensionNetwork,
	})
	require.True(t, ok, "有水位时应取到初始游标")
	assert.Equal(t, wmBucket, cursor, "初始游标必须 == 当前水位桶起点（相差 0 格，不回扫历史）")
	assert.Equal(t, aggregator.WatermarkLevelDevice, wm.lastLevel, "网络维度看设备级水位")
}

// device_group 维度看设备组级水位。
func TestInitialCursorForContinuous_DeviceGroupUsesGroupLevel(t *testing.T) {
	loc := time.UTC
	wmBucket := time.Date(2026, 6, 16, 0, 0, 0, 0, loc)
	wm := &stubWatermark{byKey: map[wmKey]time.Time{
		{metrics.GranularityDaily, aggregator.WatermarkLevelGroup}: wmBucket,
	}}
	repo := NewPgRepository(nil, nil).SetWatermarkReader(wm).SetLocationFunc(func() *time.Location { return loc })

	cursor, ok := repo.initialCursorForContinuous(context.Background(), CreateRequest{
		Mode:          ModeContinuous,
		Granularities: []string{string(metrics.GranularityDaily)},
		Dimension:     DimensionDeviceGroup,
	})
	require.True(t, ok)
	assert.Equal(t, wmBucket, cursor)
	assert.Equal(t, aggregator.WatermarkLevelGroup, wm.lastLevel, "设备组维度看组级水位")
}

// 上游尚未卷完任何格（无水位）→ ok=false（last_fire_at 留 NULL 退化 created_at，水位 gate 兜底）。
func TestInitialCursorForContinuous_NoWatermark_ReturnsNotOK(t *testing.T) {
	wm := &stubWatermark{byKey: map[wmKey]time.Time{}} // 空 = NotFound
	repo := NewPgRepository(nil, nil).SetWatermarkReader(wm)
	_, ok := repo.initialCursorForContinuous(context.Background(), CreateRequest{
		Mode:          ModeContinuous,
		Granularities: []string{string(metrics.GranularityHourly)},
		Dimension:     DimensionNetwork,
	})
	assert.False(t, ok, "上游尚未卷完任何格 → 不设初始游标，水位 gate 兜底")
}

// 未注入水位读取器 → ok=false（不回归，行为安全）。
func TestInitialCursorForContinuous_NoReader_ReturnsNotOK(t *testing.T) {
	repo := NewPgRepository(nil, nil) // 不注入水位读取器
	_, ok := repo.initialCursorForContinuous(context.Background(), CreateRequest{
		Mode:          ModeContinuous,
		Granularities: []string{string(metrics.GranularityHourly)},
		Dimension:     DimensionNetwork,
	})
	assert.False(t, ok)
}
