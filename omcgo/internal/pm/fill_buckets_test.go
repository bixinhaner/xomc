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
