package adhoc

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// issue #393：InsertResults 写库分批单测（纯函数路径，不依赖 DB）。
// 验证越过旧 ~4600 行上限时，按 resultInsertBatchSize 切批后每条 SQL 的绑定参数
// 都不越过 pgx 扩展协议 65535 上限，且全部行被覆盖、ON CONFLICT 语义保留。

const pgxMaxParams = 65535 // pgx 扩展协议单条语句绑定参数上限

// makeDistinctRows 造 n 行业务键互不相同的结果行（device_sn 递增，避免被全局去重并掉）。
func makeDistinctRows(n int) []ResultRow {
	task := uuid.New()
	t0 := time.Date(2026, 6, 15, 13, 0, 0, 0, time.UTC)
	rows := make([]ResultRow, n)
	for i := 0; i < n; i++ {
		rows[i] = ResultRow{
			TaskID:      task,
			Granularity: "hourly",
			MetricPath:  "RRC.ConnEstabSucc",
			DeviceSN:    "SN-" + strconv.Itoa(i), // 互不相同
			MetricType:  "kpi",
			MetricValue: float64(i),
			Time:        t0,
			StartTime:   t0,
			EndTime:     t0,
		}
	}
	return rows
}

// batchRanges 复刻 InsertResults 的切批边界逻辑，供断言切批数量与覆盖完整性。
func batchRanges(total, size int) [][2]int {
	var out [][2]int
	for start := 0; start < total; start += size {
		end := start + size
		if end > total {
			end = total
		}
		out = append(out, [2]int{start, end})
	}
	return out
}

// 批大小常量必须留足余量：单批参数（行数×14）远低于 65535。
func Test_resultInsertBatchSize_WithinParamLimit(t *testing.T) {
	cols := len(resultInsertCols)
	require.Equal(t, 14, cols, "结果表应为 14 列；列数变了需重算批大小")
	assert.LessOrEqual(t, resultInsertBatchSize*cols, pgxMaxParams,
		"单批参数数必须 ≤ pgx 65535 上限")
}

// 成功路径：造 5000 行（越过旧 ~4600 行上限），切批后每批 SQL 参数都不越界，
// 且所有行被覆盖、无遗漏无重叠。
func Test_buildInsertResultsSQL_BatchesUnder65535(t *testing.T) {
	const n = 5000 // > 4600，旧实现单条 INSERT 会 14*5000=70000 > 65535 报错
	rows := dedupResultRows(makeDistinctRows(n))
	require.Len(t, rows, n, "互不相同的业务键不应被去重并掉")

	ranges := batchRanges(len(rows), resultInsertBatchSize)
	require.GreaterOrEqual(t, len(ranges), 2, "5000 行必须切成 ≥2 批")

	// 覆盖完整性：批区间首尾相接、全覆盖、无重叠。
	assert.Equal(t, 0, ranges[0][0])
	assert.Equal(t, n, ranges[len(ranges)-1][1])
	for i := 1; i < len(ranges); i++ {
		assert.Equal(t, ranges[i-1][1], ranges[i][0], "批区间必须首尾相接无缝")
	}

	// 每批单独构建 SQL，断言参数数不越 65535，且含 ON CONFLICT 语义。
	total := 0
	for _, rg := range ranges {
		q, args, err := buildInsertResultsSQL(rows[rg[0]:rg[1]])
		require.NoError(t, err)
		batchRows := rg[1] - rg[0]
		assert.Equal(t, batchRows*len(resultInsertCols), len(args),
			"参数数应为 行数×列数")
		assert.LessOrEqual(t, len(args), pgxMaxParams,
			"单批参数数必须 ≤ pgx 65535 上限（旧实现整条 INSERT 在此越界）")
		// ON CONFLICT 守门口径每批都带，保证分批不破坏冲突处理。
		assert.Contains(t, q, "ON CONFLICT")
		assert.Contains(t, q, "DO UPDATE SET")
		total += batchRows
	}
	assert.Equal(t, n, total, "所有行必须被分批覆盖，无遗漏")
}

// 边界路径：恰好等于批大小的整数倍时不产生空尾批。
func Test_buildInsertResultsSQL_ExactMultipleNoEmptyBatch(t *testing.T) {
	n := resultInsertBatchSize * 2
	ranges := batchRanges(n, resultInsertBatchSize)
	require.Len(t, ranges, 2)
	assert.Equal(t, resultInsertBatchSize, ranges[0][1]-ranges[0][0])
	assert.Equal(t, resultInsertBatchSize, ranges[1][1]-ranges[1][0])
}

// 跨批次不共享业务键：先全局去重，重复键被并到同批前的同一行，
// 不会出现「批 A 一行与批 B 一行同业务键」导致 ON CONFLICT 跨批重复命中。
func Test_InsertResults_GlobalDedupBeforeBatching(t *testing.T) {
	task := uuid.New()
	t0 := time.Date(2026, 6, 15, 13, 0, 0, 0, time.UTC)
	// 造 resultInsertBatchSize+10 行，其中末尾 10 行与开头 10 行业务键重复。
	base := makeDistinctRows(resultInsertBatchSize + 10)
	base[0].TaskID = task
	dup := make([]ResultRow, 0, len(base)+10)
	dup = append(dup, base...)
	for i := 0; i < 10; i++ {
		// 复制前 10 行业务键，值改变 → 应被去重保留最后一条值
		r := base[i]
		r.MetricValue = 9999
		r.Time = t0
		dup = append(dup, r)
	}
	out := dedupResultRows(dup)
	// 去重后行数 = base 行数（重复的 10 行被并回）
	assert.Equal(t, len(base), len(out))
	// 前 10 行的值被最后一条覆盖
	for i := 0; i < 10; i++ {
		assert.Equal(t, float64(9999), out[i].MetricValue, "重复业务键应保留最后一条值")
	}
}
