package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wideTestCols() []WideColumn {
	return []WideColumn{
		{Code: "K001", Type: "kpi", Name: "上行吞吐"},
		{Code: "C002", Type: "counter", Name: "下行包数"},
	}
}

func TestWideCSVWriter_BOMAndHeader(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, wideTestCols())
	require.NoError(t, err)
	require.NoError(t, cw.Flush())

	b := buf.Bytes()
	// 头三字节 UTF-8 BOM。
	require.GreaterOrEqual(t, len(b), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, b[:3])

	// 去 BOM 后第一行：固定行键列 + 指标列「编号(名·类型)」，无「统计方式」。
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	firstLine := strings.SplitN(body, "\n", 2)[0]
	assert.Contains(t, firstLine, "设备")
	assert.Contains(t, firstLine, "小区/PLMN")
	assert.Contains(t, firstLine, "时窗起")
	assert.Contains(t, firstLine, "K001(上行吞吐·kpi)")
	assert.Contains(t, firstLine, "C002(下行包数·counter)")
	assert.NotContains(t, firstLine, "统计方式")
}

// 同一 (设备×小区×时间) 行键、不同指标 → 摊成一行，每指标一列。
func TestWideCSVWriter_PivotSameKey(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, wideTestCols())
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	base := ExportRow{Device: "ABCDEF/SN1", CellPLMN: "Cellid=111", Granularity: "hourly", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour)}
	r1 := base
	r1.MetricCode, r1.Value = "K001", 1.5
	r2 := base
	r2.MetricCode, r2.Value = "C002", 2.5
	require.NoError(t, cw.AddRow(r1))
	require.NoError(t, cw.AddRow(r2))
	require.NoError(t, cw.Flush())

	assert.Equal(t, int64(1), cw.RowCount()) // 两数据点摊成一横行

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 2) // 表头 + 1 横行
	row := recs[1]
	assert.Equal(t, "ABCDEF/SN1", row[0])
	assert.Equal(t, "Cellid=111", row[1])
	assert.Equal(t, "hourly", row[2])
	assert.Equal(t, "2026-06-04 10:00:00", row[3]) // 时间
	assert.Equal(t, "2026-06-04 11:00:00", row[5]) // 时窗止
	assert.Equal(t, "1.5", row[6])                 // K001 列
	assert.Equal(t, "2.5", row[7])                 // C002 列
}

// 某指标在该行键缺值 → 空单元格。
func TestWideCSVWriter_MissingMetricEmptyCell(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, wideTestCols())
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{Device: "d2", Time: tm, MetricCode: "K001", Value: 9}))
	require.NoError(t, cw.Flush())

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	row := recs[1]
	assert.Equal(t, "9", row[6]) // K001 有值
	assert.Equal(t, "", row[7])  // C002 缺值 → 空
}

// 不同时间 → 时间桶切换，各成一横行。
func TestWideCSVWriter_TimeBucketFlush(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, wideTestCols())
	require.NoError(t, err)

	t1 := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 4, 11, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{Device: "d1", Time: t1, MetricCode: "K001", Value: 1}))
	require.NoError(t, cw.AddRow(ExportRow{Device: "d1", Time: t2, MetricCode: "K001", Value: 2}))
	require.NoError(t, cw.Flush())

	assert.Equal(t, int64(2), cw.RowCount()) // 两个时间桶 → 两横行
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 3) // 表头 + 2 行
}

// 零列（无指标）：只写固定行键列表头。
func TestWideCSVWriter_NoColumns(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, nil)
	require.NoError(t, err)
	require.NoError(t, cw.Flush())
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Len(t, recs[0], len(wideFixedHeader))
}
