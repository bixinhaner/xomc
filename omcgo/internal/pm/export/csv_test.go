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

func TestCSVWriter_BOMAndHeader(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewCSVWriter(&buf)
	require.NoError(t, err)
	require.NoError(t, cw.Flush())

	b := buf.Bytes()
	// 头三字节 UTF-8 BOM。
	require.GreaterOrEqual(t, len(b), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, b[:3])

	// 去 BOM 后第一行是中文表头。
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	firstLine := strings.SplitN(body, "\n", 2)[0]
	assert.Contains(t, firstLine, "设备")
	assert.Contains(t, firstLine, "小区/PLMN")
	assert.Contains(t, firstLine, "统计方式")
}

func TestCSVWriter_RowFormatting(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewCSVWriter(&buf)
	require.NoError(t, err)

	tm := time.Date(2026, 6, 4, 10, 30, 0, 0, time.UTC)
	require.NoError(t, cw.WriteRow(ExportRow{
		Device:      "ABCDEF/SN1",
		CellPLMN:    "Cellid=111,PLMN=46000",
		MetricCode:  "K001",
		MetricName:  "上行吞吐量",
		MetricType:  "kpi",
		Granularity: "hourly",
		Time:        tm,
		StartTime:   tm,
		EndTime:     tm.Add(time.Hour),
		Value:       12.5,
		StatisType:  "", // 空值 → 空串
	}))
	require.NoError(t, cw.Flush())
	assert.Equal(t, int64(1), cw.RowCount())

	// 解析回来核对字段。
	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	r := csv.NewReader(strings.NewReader(body))
	records, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2) // 表头 + 1 行
	row := records[1]
	assert.Equal(t, "ABCDEF/SN1", row[0])
	assert.Equal(t, "Cellid=111,PLMN=46000", row[1])
	assert.Equal(t, "K001", row[2])
	assert.Equal(t, "上行吞吐量", row[3])
	assert.Equal(t, "kpi", row[4])
	assert.Equal(t, "hourly", row[5])
	assert.Equal(t, "2026-06-04 10:30:00", row[6])
	assert.Equal(t, "12.5", row[9])
	assert.Equal(t, "", row[10]) // 统计方式空串
}

func TestCSVWriter_EmptyValues(t *testing.T) {
	var buf bytes.Buffer
	cw, err := NewCSVWriter(&buf)
	require.NoError(t, err)
	// 全空：设备/小区/统计方式空串，零时间空串。
	require.NoError(t, cw.WriteRow(ExportRow{MetricCode: "C1"}))
	require.NoError(t, cw.Flush())

	body := strings.TrimPrefix(buf.String(), string(utf8BOM))
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	row := records[1]
	assert.Equal(t, "", row[0])  // 设备
	assert.Equal(t, "", row[1])  // 小区/PLMN
	assert.Equal(t, "", row[6])  // 时间（零值）
	assert.Equal(t, "0", row[9]) // 值 0
	assert.Equal(t, "", row[10]) // 统计方式
}
