package pm

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 北京时区（T-0192b 业务时区切天的目标时区）。
func beijing(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err, "load Asia/Shanghai must succeed in test env")
	return loc
}

// TestAlignToBucket_DailyLocalMidnight 验证日粒度在业务时区下对齐到本地零点（北京 00:00），
// 其 UTC 瞬间是前一日 16:00（核心断言：填充桶不再落 UTC 零点 = 北京早 8 点）。
func TestAlignToBucket_DailyLocalMidnight(t *testing.T) {
	loc := beijing(t)
	// 北京 2026-06-02 14:30 → 对齐应落北京 2026-06-02 00:00 = UTC 2026-06-01 16:00。
	in := time.Date(2026, 6, 2, 14, 30, 0, 0, loc)
	got := alignToBucket(in.In(loc), metrics.GranularityDaily)

	assert.Equal(t, 0, got.Hour(), "daily bucket must be local midnight")
	assert.Equal(t, loc.String(), got.Location().String())
	// UTC 瞬间应为北京零点对应的 UTC 前一日 16:00。
	assert.Equal(t, time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC), got.UTC())
}

// TestBucketStartsBetween_DailyLandsOnLocalMidnight 验证日粒度桶序列每个起点都落本地零点。
func TestBucketStartsBetween_DailyLandsOnLocalMidnight(t *testing.T) {
	loc := beijing(t)
	// 查询窗：北京 06-01 00:00 ~ 06-03 00:00（用 UTC 瞬间表达，模拟前端按 RFC3339 传入）。
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2026, 6, 3, 0, 0, 0, 0, loc)

	buckets := bucketStartsBetween(start, end, metrics.GranularityDaily, loc)
	require.Len(t, buckets, 2)
	for _, b := range buckets {
		inLoc := b.In(loc)
		assert.Equal(t, 0, inLoc.Hour(), "every daily bucket start must be local midnight")
		assert.Equal(t, 0, inLoc.Minute())
	}
	// 第一个桶 UTC 应为 05-31 16:00（北京 06-01 00:00）。
	assert.Equal(t, time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC), buckets[0].UTC())
}

// TestFillEmptyBuckets_DailyDedupHitsLocalMidnightRow 核心场景：真实行落本地零点时，
// 填充桶按业务时区对齐 → bucketKey 同一 UTC 瞬间 → 去重命中，不再产生"真实点 + 早 8 点空点"。
func TestFillEmptyBuckets_DailyDedupHitsLocalMidnightRow(t *testing.T) {
	loc := beijing(t)
	// 真实行落北京 06-02 00:00（aggregator 已对齐到本地零点，T-0192）。
	realTime := time.Date(2026, 6, 2, 0, 0, 0, 0, loc)
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityDaily,
		Dimension:   aggregator.DimensionDevice,
		DeviceOUIs:  []string{"OUI1"},
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"m.a"},
		StartTime:   time.Date(2026, 6, 1, 0, 0, 0, 0, loc),
		EndTime:     time.Date(2026, 6, 3, 0, 0, 0, 0, loc),
	}
	rows := []aggregator.Row{
		{DeviceOUI: "OUI1", DeviceSN: "SN1", MetricPath: "m.a", Time: realTime, Granularity: metrics.GranularityDaily},
	}

	out := fillEmptyBuckets(rows, req, loc)

	// 窗口含 2 个日桶（06-01、06-02）。06-02 已有真实行 → 仅补 06-01 一个占位行。
	require.Len(t, out, 2, "should have 1 real + 1 filled (06-02 deduped)")

	var filledCount, realCount int
	for _, r := range out {
		if r.Filled {
			filledCount++
			// 填充桶必须落本地零点（不是 UTC 零点 / 北京早 8 点）。
			assert.Equal(t, 0, r.Time.In(loc).Hour(), "filled bucket must be local midnight")
		} else {
			realCount++
			assert.True(t, realTime.Equal(r.Time))
		}
	}
	assert.Equal(t, 1, filledCount)
	assert.Equal(t, 1, realCount)
}

// TestFillEmptyBuckets_NilLocFallsBackUTC 验证 loc 为 nil 时回落 UTC（安全默认，不 panic）。
func TestFillEmptyBuckets_NilLocFallsBackUTC(t *testing.T) {
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityDaily,
		Dimension:   aggregator.DimensionDevice,
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"m.a"},
		StartTime:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	out := fillEmptyBuckets(nil, req, nil)
	require.Len(t, out, 1)
	assert.True(t, out[0].Filled)
	// 回落 UTC：填充桶落 UTC 零点。
	assert.Equal(t, 0, out[0].Time.UTC().Hour())
	assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), out[0].Time.UTC())
}

// TestBucketStartsBetween_HourlyTimezoneAgnostic 验证 15min/hourly 整点对齐行为不随时区变化
// （不回归）：北京时区与 UTC 下同一查询窗产出相同的 UTC 桶起点序列。
func TestBucketStartsBetween_HourlyTimezoneAgnostic(t *testing.T) {
	loc := beijing(t)
	start := time.Date(2026, 6, 2, 10, 17, 0, 0, time.UTC)
	end := time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC)

	utcBuckets := bucketStartsBetween(start, end, metrics.GranularityHourly, time.UTC)
	locBuckets := bucketStartsBetween(start, end, metrics.GranularityHourly, loc)

	require.Equal(t, len(utcBuckets), len(locBuckets))
	for i := range utcBuckets {
		assert.True(t, utcBuckets[i].Equal(locBuckets[i]),
			"hourly bucket %d must be timezone-agnostic", i)
		assert.Equal(t, 0, utcBuckets[i].Minute())
	}
	// 首桶整点对齐到 10:00。
	assert.Equal(t, time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC), utcBuckets[0].UTC())
}

// TestFillEmptyBuckets_FillDisabledBoundaries 验证填充禁用边界：
// 多 SN 复合查询 / 缺 metric_paths 时不补占位，原样返回。
func TestFillEmptyBuckets_FillDisabledBoundaries(t *testing.T) {
	loc := beijing(t)
	base := aggregator.QueryRequest{
		Granularity: metrics.GranularityDaily,
		Dimension:   aggregator.DimensionDevice,
		StartTime:   time.Date(2026, 6, 1, 0, 0, 0, 0, loc),
		EndTime:     time.Date(2026, 6, 3, 0, 0, 0, 0, loc),
	}

	t.Run("multi SN not filled", func(t *testing.T) {
		req := base
		req.DeviceSNs = []string{"SN1", "SN2"}
		req.MetricPaths = []string{"m.a"}
		out := fillEmptyBuckets(nil, req, loc)
		assert.Empty(t, out, "multi-SN composite query must not fill")
	})

	t.Run("missing metric_paths not filled", func(t *testing.T) {
		req := base
		req.DeviceSNs = []string{"SN1"}
		req.MetricPaths = nil
		out := fillEmptyBuckets(nil, req, loc)
		assert.Empty(t, out, "missing metric_paths must not fill")
	})

	t.Run("device_group dimension not filled", func(t *testing.T) {
		req := base
		req.Dimension = aggregator.DimensionDeviceGroup
		req.DeviceSNs = []string{"SN1"}
		req.MetricPaths = []string{"m.a"}
		out := fillEmptyBuckets(nil, req, loc)
		assert.Empty(t, out, "device_group dimension must not fill")
	})
}

// ── T-0192c：月粒度自然月步进 ───────────────────────────────────────────────

// TestNextBucketStart_MonthlyNaturalMonth 验证月粒度走自然月步进（AddDate(0,1,0)），
// 而非固定 30 天：普通月、跨年（12 月→次年 1 月，年份 +1）、闰年（2 月→3 月仍正确）。
func TestNextBucketStart_MonthlyNaturalMonth(t *testing.T) {
	loc := beijing(t)
	cases := []struct {
		name string
		cur  time.Time
		want time.Time
	}{
		{
			name: "ordinary month Jan->Feb",
			cur:  time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			want: time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
		},
		{
			name: "Feb->Mar non-leap",
			cur:  time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
			want: time.Date(2026, 3, 1, 0, 0, 0, 0, loc),
		},
		{
			name: "leap year Feb->Mar (2/29 month still lands 3/1)",
			cur:  time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
			want: time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
		},
		{
			name: "cross year Dec->next Jan (year+1)",
			cur:  time.Date(2026, 12, 1, 0, 0, 0, 0, loc),
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, loc),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := nextBucketStart(c.cur, metrics.GranularityMonthly, loc)
			assert.True(t, c.want.Equal(got), "want %s got %s", c.want, got)
			// 落点恒为下月 1 号本地零点（不漂移）。
			inLoc := got.In(loc)
			assert.Equal(t, 1, inLoc.Day(), "monthly next bucket must land on day 1")
			assert.Equal(t, 0, inLoc.Hour())
		})
	}
}

// TestNextBucketStart_MonthlyNilLocFallsBackUTC 验证月粒度 loc=nil 回落 UTC，不 panic。
func TestNextBucketStart_MonthlyNilLocFallsBackUTC(t *testing.T) {
	cur := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := nextBucketStart(cur, metrics.GranularityMonthly, nil)
	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), got.UTC())
}

// TestNextBucketStart_NonMonthlyFixedDuration 验证非月粒度沿用固定时长步进（零回归）：
// 周=精确 7 天、日=精确 24 小时、时=1h、15min=15min，与改造前一致。
func TestNextBucketStart_NonMonthlyFixedDuration(t *testing.T) {
	loc := beijing(t)
	base := time.Date(2026, 6, 2, 0, 0, 0, 0, loc)
	cases := []struct {
		g   metrics.Granularity
		dur time.Duration
	}{
		{metrics.Granularity15Min, 15 * time.Minute},
		{metrics.GranularityHourly, time.Hour},
		{metrics.GranularityDaily, 24 * time.Hour},
		{metrics.GranularityWeekly, 7 * 24 * time.Hour},
	}
	for _, c := range cases {
		got := nextBucketStart(base, c.g, loc)
		assert.True(t, base.Add(c.dur).Equal(got),
			"granularity %v must step fixed %v", c.g, c.dur)
	}
}

// TestBucketStartsBetween_MonthlyLandsOnFirstOfMonth 验证月桶序列起点恒落每月 1 号本地零点，
// 连续跨多月（含跨 1/31、2/28、12/31）逐月落 1 号、不随月份累积漂移、不出现 5/31 & 6/30。
func TestBucketStartsBetween_MonthlyLandsOnFirstOfMonth(t *testing.T) {
	loc := beijing(t)
	// 查询窗：北京 2026-01-15 ~ 2026-07-15（跨 6 个月边界，含 1/31、2/28）。
	start := time.Date(2026, 1, 15, 8, 0, 0, 0, loc)
	end := time.Date(2026, 7, 15, 0, 0, 0, 0, loc)

	buckets := bucketStartsBetween(start, end, metrics.GranularityMonthly, loc)
	// 对齐后首桶 = 1 月 1 号；含 1~7 月 1 号共 7 个。
	require.Len(t, buckets, 7)

	wantMonths := []time.Month{time.January, time.February, time.March, time.April, time.May, time.June, time.July}
	for i, b := range buckets {
		inLoc := b.In(loc)
		assert.Equal(t, 1, inLoc.Day(), "bucket %d must be day 1, got %s", i, inLoc)
		assert.Equal(t, 0, inLoc.Hour())
		assert.Equal(t, 0, inLoc.Minute())
		assert.Equal(t, wantMonths[i], inLoc.Month(), "bucket %d month mismatch", i)
		assert.Equal(t, 2026, inLoc.Year())
	}
	// 显式断言：不出现 5/31、6/30 这类漂移桶（固定 30 天步进的旧 bug 表征）。
	for _, b := range buckets {
		inLoc := b.In(loc)
		assert.False(t, inLoc.Day() == 30 || inLoc.Day() == 31,
			"must not produce drifted bucket like 5/31 or 6/30, got %s", inLoc)
	}
}

// TestBucketStartsBetween_MonthlyCrossYear 验证月桶跨年正确（12 月→次年 1 月，年份 +1）。
func TestBucketStartsBetween_MonthlyCrossYear(t *testing.T) {
	loc := beijing(t)
	// 查询窗：北京 2026-11-10 ~ 2027-02-10（跨 12/31 年界）。
	start := time.Date(2026, 11, 10, 0, 0, 0, 0, loc)
	end := time.Date(2027, 2, 10, 0, 0, 0, 0, loc)

	buckets := bucketStartsBetween(start, end, metrics.GranularityMonthly, loc)
	// 11 月、12 月、次年 1 月、次年 2 月 共 4 个。
	require.Len(t, buckets, 4)

	want := []time.Time{
		time.Date(2026, 11, 1, 0, 0, 0, 0, loc),
		time.Date(2026, 12, 1, 0, 0, 0, 0, loc),
		time.Date(2027, 1, 1, 0, 0, 0, 0, loc),
		time.Date(2027, 2, 1, 0, 0, 0, 0, loc),
	}
	for i := range want {
		assert.True(t, want[i].Equal(buckets[i]), "bucket %d want %s got %s", i, want[i], buckets[i])
	}
}

// TestFillEmptyBuckets_MonthlyEndTimeIsNextMonthFirst 验证填充月桶 EndTime = 下月 1 号本地零点
// （不是起点 +30 天）。
func TestFillEmptyBuckets_MonthlyEndTimeIsNextMonthFirst(t *testing.T) {
	loc := beijing(t)
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityMonthly,
		Dimension:   aggregator.DimensionDevice,
		DeviceOUIs:  []string{"OUI1"},
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"m.a"},
		// 查询窗：北京 1 月 ~ 3 月（含 1 月、2 月两个桶）。
		StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
		EndTime:   time.Date(2026, 3, 1, 0, 0, 0, 0, loc),
	}

	out := fillEmptyBuckets(nil, req, loc)
	require.Len(t, out, 2, "should fill Jan + Feb")

	byMonth := make(map[time.Month]aggregator.Row, 2)
	for _, r := range out {
		require.True(t, r.Filled)
		byMonth[r.Time.In(loc).Month()] = r
	}
	// 1 月桶 EndTime = 2 月 1 号（28 天后，非 +30 天）。
	jan := byMonth[time.January]
	assert.True(t, time.Date(2026, 2, 1, 0, 0, 0, 0, loc).Equal(jan.EndTime),
		"Jan bucket EndTime must be Feb 1, got %s", jan.EndTime)
	// 2 月桶 EndTime = 3 月 1 号（仅 28 天，固定 +30 天会漂到 3/2）。
	feb := byMonth[time.February]
	assert.True(t, time.Date(2026, 3, 1, 0, 0, 0, 0, loc).Equal(feb.EndTime),
		"Feb bucket EndTime must be Mar 1, got %s", feb.EndTime)
	// 起止边界自洽：StartTime = 桶起点，EndTime = 下月 1 号。
	assert.True(t, jan.StartTime.Equal(jan.Time))
}

// TestFillEmptyBuckets_MonthlyDedupHitsFirstOfMonthRow 验证月桶去重键与真实物化月行
// （落每月 1 号本地零点）对齐：同一月不产生"真实点 + 漂移空点"孪生桶。
func TestFillEmptyBuckets_MonthlyDedupHitsFirstOfMonthRow(t *testing.T) {
	loc := beijing(t)
	// 真实行落北京 2 月 1 号本地零点（worker 物化月数据落每月 1 号，T-0192）。
	realTime := time.Date(2026, 2, 1, 0, 0, 0, 0, loc)
	req := aggregator.QueryRequest{
		Granularity: metrics.GranularityMonthly,
		Dimension:   aggregator.DimensionDevice,
		DeviceOUIs:  []string{"OUI1"},
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"m.a"},
		// 查询窗：1 月 ~ 4 月（含 1/2/3 三个月桶）。
		StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
		EndTime:   time.Date(2026, 4, 1, 0, 0, 0, 0, loc),
	}
	rows := []aggregator.Row{
		{DeviceOUI: "OUI1", DeviceSN: "SN1", MetricPath: "m.a", Time: realTime, Granularity: metrics.GranularityMonthly},
	}

	out := fillEmptyBuckets(rows, req, loc)
	// 3 个月桶，2 月已有真实行 → 补 1 月、3 月两个占位 + 1 真实 = 3 行（无孪生）。
	require.Len(t, out, 3, "should have 1 real (Feb) + 2 filled (Jan, Mar), no twin bucket")

	var filledCount, realCount int
	for _, r := range out {
		if r.Filled {
			filledCount++
			assert.Equal(t, 1, r.Time.In(loc).Day(), "filled monthly bucket must land day 1")
			// 占位行不得与真实行同月（否则就是孪生桶）。
			assert.NotEqual(t, time.February, r.Time.In(loc).Month(),
				"Feb has real row, must not produce twin filled bucket")
		} else {
			realCount++
			assert.True(t, realTime.Equal(r.Time))
		}
	}
	assert.Equal(t, 2, filledCount)
	assert.Equal(t, 1, realCount)
}
