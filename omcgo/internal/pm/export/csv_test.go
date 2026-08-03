package export

import (
	"bytes"
	"encoding/csv"
	"math"
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
	cw, err := NewWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols())
	require.NoError(t, err)
	require.NoError(t, cw.Flush())

	b := buf.Bytes()
	// 头三字节 UTF-8 BOM。
	require.GreaterOrEqual(t, len(b), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, b[:3])

	// 去 BOM 后第一行：与页面表格一致——开始/结束时间 + 设备 + Cell ID/PLMN + 指标友好名；
	// 不含旧的「粒度/时间/时窗起/时窗止」「小区/PLMN」「编号(名·类型)」「统计方式」。
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	firstLine := strings.SplitN(body, "\n", 2)[0]
	assert.Contains(t, firstLine, "开始时间")
	assert.Contains(t, firstLine, "结束时间")
	assert.Contains(t, firstLine, "设备 SN")
	assert.Contains(t, firstLine, "Cell ID")
	assert.Contains(t, firstLine, "PLMN")
	assert.Contains(t, firstLine, "上行吞吐")
	assert.Contains(t, firstLine, "下行包数")
	assert.NotContains(t, firstLine, "K001(")
	assert.NotContains(t, firstLine, "·kpi")
	assert.NotContains(t, firstLine, "粒度")
	assert.NotContains(t, firstLine, "时窗")
	assert.NotContains(t, firstLine, "统计方式")
}

// 同一 (设备×小区×时间) 行键、不同指标 → 摊成一行，每指标一列。
func TestWideCSVWriter_PivotSameKey(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols())
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	base := ExportRow{Device: "SN1", CellPLMN: "Cellid=111,PLMN=46068", Granularity: "hourly", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour)}
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
	// 列序：开始时间 | 结束时间 | 设备 SN | Cell ID | PLMN | K001 | C002
	assert.Equal(t, "2026-06-04 10:00:00", row[0]) // 开始时间（时窗起）
	assert.Equal(t, "2026-06-04 11:00:00", row[1]) // 结束时间（时窗止）
	assert.Equal(t, "SN1", row[2])                 // 设备 SN（无 OUI 前缀）
	assert.Equal(t, "111", row[3])                 // Cell ID（从 LDN 解析）
	assert.Equal(t, "46068", row[4])               // PLMN（从 LDN 解析）
	assert.Equal(t, "1.50", row[5])                // K001 列
	assert.Equal(t, "2.50", row[6])                // C002 列
}

func TestWideCSVWriter_MetricValuesUseFixedTwoDecimals(t *testing.T) {
	var buf bytes.Buffer
	cw, err := newWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols(), "-")
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	base := ExportRow{Device: "SN1", CellPLMN: "Cellid=111,PLMN=46068", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour)}
	longDecimal := base
	longDecimal.MetricCode, longDecimal.Value = "K001", 12.345678901234
	integer := base
	integer.MetricCode, integer.Value = "C002", 12
	require.NoError(t, cw.AddRow(longDecimal))
	require.NoError(t, cw.AddRow(integer))
	require.NoError(t, cw.Flush())

	row := nthCSVRow(t, buf.Bytes(), 1)
	assert.Equal(t, "12.35", row[5])
	assert.Equal(t, "12.00", row[6])
}

func TestWideCSVWriter_KpiQueryMeasurementObjectColumn(t *testing.T) {
	var buf bytes.Buffer
	cw, err := newWideCSVWriterWithMeasurementObject(&buf, "设备 SN", false, true, wideTestCols(), "-", nil)
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	rows := []ExportRow{
		{
			Device:     "LTE-SN",
			CellPLMN:   "Cellid=111,PLMN=46068",
			Time:       tm,
			StartTime:  tm,
			EndTime:    tm.Add(time.Hour),
			MetricCode: "K001",
			Value:      1.5,
		},
		{
			Device:     "NR-SN",
			CellPLMN:   "Type=Cell,Mode=SA,gNBID=123,NrCGI=46068123456",
			Time:       tm,
			StartTime:  tm,
			EndTime:    tm.Add(time.Hour),
			MetricCode: "K001",
			Value:      2.5,
		},
		{
			Device:     "GSM-SN",
			CellPLMN:   "",
			Time:       tm,
			StartTime:  tm,
			EndTime:    tm.Add(time.Hour),
			MetricCode: "K001",
			Value:      3.5,
		},
	}
	for _, row := range rows {
		require.NoError(t, cw.AddRow(row))
	}
	require.NoError(t, cw.Flush())

	recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(buf.String(), string(utf8BOM)))).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 4)
	assert.Equal(t, []string{"开始时间", "结束时间", "设备 SN", "测量对象", "上行吞吐", "下行包数"}, recs[0])
	byDevice := map[string][]string{}
	for _, row := range recs[1:] {
		byDevice[row[2]] = row
	}
	assert.Equal(t, "Cellid=111,PLMN=46068", byDevice["LTE-SN"][3])
	assert.Equal(t, "Type=Cell,Mode=SA,gNBID=123,NrCGI=46068123456", byDevice["NR-SN"][3])
	assert.Equal(t, "-", byDevice["GSM-SN"][3])
}

func TestWideCSVWriter_FormatsTimeInOutputLocation(t *testing.T) {
	var buf bytes.Buffer
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	cw, err := newWideCSVWriterWithLocation(&buf, "设备 SN", false, true, wideTestCols(), "", shanghai)
	require.NoError(t, err)

	start := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	require.NoError(t, cw.AddRow(ExportRow{
		Device:     "SN1",
		MetricCode: "K001",
		Time:       start,
		StartTime:  start,
		EndTime:    end,
		Value:      1,
	}))
	require.NoError(t, cw.Flush())

	row := nthCSVRow(t, buf.Bytes(), 1)
	assert.Equal(t, "2026-06-04 18:00:00", row[0])
	assert.Equal(t, "2026-06-04 19:00:00", row[1])
}

func TestWideCSVWriter_NilOutputLocationFallsBackToUTC(t *testing.T) {
	var buf bytes.Buffer
	cw, err := newWideCSVWriterWithLocation(&buf, "设备 SN", false, true, wideTestCols(), "", nil)
	require.NoError(t, err)

	start := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{
		Device:     "SN1",
		MetricCode: "K001",
		Time:       start,
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Value:      1,
	}))
	require.NoError(t, cw.Flush())

	row := nthCSVRow(t, buf.Bytes(), 1)
	assert.Equal(t, "2026-06-04 10:00:00", row[0])
	assert.Equal(t, "2026-06-04 11:00:00", row[1])
}

// 某指标在该行键缺值，默认保持 CSV 空单元格。
func TestWideCSVWriter_MissingMetricDefaultEmptyCell(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols())
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{Device: "d2", Time: tm, StartTime: tm, MetricCode: "K001", Value: 9}))
	require.NoError(t, cw.Flush())

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	row := recs[1]
	// 列序：开始时间 | 结束时间 | 设备 SN | Cell ID | PLMN | K001 | C002
	assert.Equal(t, "9.00", row[5]) // K001 有值
	assert.Equal(t, "", row[6])     // C002 缺值 → 空
}

// 配置占位符时，某指标在该行键缺值 → 与页面一致显示 "-"。
func TestWideCSVWriter_MissingMetricPlaceholder(t *testing.T) {
	var buf bytes.Buffer
	cw, err := newWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols(), "-")
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{Device: "d2", Time: tm, StartTime: tm, MetricCode: "K001", Value: 9}))
	require.NoError(t, cw.Flush())

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	row := recs[1]
	// 列序：开始时间 | 结束时间 | 设备 SN | Cell ID | PLMN | K001 | C002
	assert.Equal(t, "9.00", row[5]) // K001 有值
	assert.Equal(t, "-", row[6])    // C002 缺值 → "-"
}

func TestWideCSVWriter_NullMetricValuePlaceholder(t *testing.T) {
	var buf bytes.Buffer
	cw, err := newWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols(), "-")
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	require.NoError(t, cw.AddRow(ExportRow{Device: "d2", Time: tm, StartTime: tm, MetricCode: "K001", Value: math.NaN()}))
	require.NoError(t, cw.Flush())

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	row := recs[1]
	assert.Equal(t, "-", row[5], "DB NULL 缺值应导出为 '-'")
}

// 不同时间 → 时间桶切换，各成一横行。
func TestWideCSVWriter_TimeBucketFlush(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, "设备 SN", false, true, wideTestCols())
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

// 零列（无指标）：只写固定行键列表头（device 口径 5 列：开始时间 结束时间 设备 SN Cell ID PLMN）。
func TestWideCSVWriter_NoColumns(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewWideCSVWriter(&buf, "设备 SN", false, true, nil)
	require.NoError(t, err)
	require.NoError(t, cw.Flush())
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Len(t, recs[0], 5)
}

// 维度列布局：首列表头随维度变；含/不含 Cell ID/PLMN 两态。
func TestWideCSVWriter_DimensionLayout(t *testing.T) {
	cols := []WideColumn{{Code: "C1", Type: "counter", Name: "速率"}}

	// 含小区列（device 口径）：开始时间 | 结束时间 | 设备 SN | Cell ID | PLMN | 速率。
	var buf bytes.Buffer
	w, err := NewWideCSVWriter(&buf, "设备 SN", false, true, cols)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	header := firstCSVRow(t, buf.Bytes())
	assert.Equal(t, []string{"开始时间", "结束时间", "设备 SN", "Cell ID", "PLMN", "速率"}, header)

	// 不含小区列（聚合维度口径）：开始时间 | 结束时间 | 设备组 | 速率。
	buf.Reset()
	w2, err := NewWideCSVWriter(&buf, "设备组", false, false, cols)
	require.NoError(t, err)
	require.NoError(t, w2.Flush())
	header2 := firstCSVRow(t, buf.Bytes())
	assert.Equal(t, []string{"开始时间", "结束时间", "设备组", "速率"}, header2)
}

// 聚合维度行：includeCell=false 时不输出 Cell ID/PLMN 列，CellPLMN 不参与行键、按设备名自然合并。
func TestWideCSVWriter_AggregateNoCellColumn(t *testing.T) {
	cols := []WideColumn{{Code: "C1", Type: "counter", Name: "速率"}}
	var buf bytes.Buffer
	w, err := NewWideCSVWriter(&buf, "产品", false, false, cols)
	require.NoError(t, err)
	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	// CellPLMN 给值也应被忽略（聚合维度无小区列）。
	require.NoError(t, w.AddRow(ExportRow{Device: "BLX", CellPLMN: "应忽略", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour), MetricCode: "C1", Value: 7}))
	require.NoError(t, w.Flush())
	row := nthCSVRow(t, buf.Bytes(), 1)
	// 列：开始时间 | 结束时间 | 产品 | C1 —— 无 Cell ID/PLMN 列。
	assert.Equal(t, "2026-06-04 10:00:00", row[0]) // 开始时间
	assert.Equal(t, "2026-06-04 11:00:00", row[1]) // 结束时间
	assert.Equal(t, "BLX", row[2])
	assert.Equal(t, "7.00", row[3]) // C1 值
	assert.Len(t, row, 4)
}

// 设备组维度（includeTech=true）：表头在对象列后插「制式」列；同组 lte/nr 凭制式区分成两行（B2 修复）。
func TestWideCSVWriter_DeviceGroupTechnologyColumn(t *testing.T) {
	cols := []WideColumn{{Code: "C1", Type: "counter", Name: "速率"}}
	var buf bytes.Buffer
	// 设备组维度：含制式列、不含小区列。
	w, err := NewWideCSVWriter(&buf, "设备组", true, false, cols)
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	// 同组（华东A组）同时间桶，lte / nr 两条 → 必须分成两行（靠制式区分行键）。
	require.NoError(t, w.AddRow(ExportRow{Device: "华东A组", Technology: "NR", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour), MetricCode: "C1", Value: 50}))
	require.NoError(t, w.AddRow(ExportRow{Device: "华东A组", Technology: "LTE", Time: tm, StartTime: tm, EndTime: tm.Add(time.Hour), MetricCode: "C1", Value: 20}))
	require.NoError(t, w.Flush())

	// 表头：开始时间 | 结束时间 | 设备组 | 制式 | 速率。
	header := firstCSVRow(t, buf.Bytes())
	assert.Equal(t, []string{"开始时间", "结束时间", "设备组", "制式", "速率"}, header)

	// 排序后 LTE 先于 NR（同组按制式升序）。
	rowLTE := nthCSVRow(t, buf.Bytes(), 1)
	assert.Equal(t, []string{"2026-06-04 10:00:00", "2026-06-04 11:00:00", "华东A组", "LTE", "20.00"}, rowLTE)
	rowNR := nthCSVRow(t, buf.Bytes(), 2)
	assert.Equal(t, []string{"2026-06-04 10:00:00", "2026-06-04 11:00:00", "华东A组", "NR", "50.00"}, rowNR)
	assert.Equal(t, int64(2), w.RowCount())
}

// firstCSVRow 跳过 UTF-8 BOM 后用 encoding/csv 读首行。
func firstCSVRow(t *testing.T, b []byte) []string {
	t.Helper()
	return nthCSVRow(t, b, 0)
}

// nthCSVRow 跳过 UTF-8 BOM 后用 encoding/csv 读第 n 行（0 基）。
func nthCSVRow(t *testing.T, b []byte, n int) []string {
	t.Helper()
	body := strings.TrimPrefix(string(b), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Greater(t, len(recs), n)
	return recs[n]
}
