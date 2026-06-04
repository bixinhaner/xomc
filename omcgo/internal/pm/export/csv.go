package export

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

// utf8BOM 是 UTF-8 字节顺序标记，写在 CSV 首部让 Excel 双击直接按 UTF-8 解析、中文不乱码。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// csvHeader 是导出 CSV 的中文长表表头（设计 §5.5）。
var csvHeader = []string{
	"设备", "小区/PLMN", "指标编号", "指标名", "类型", "粒度", "时间", "时窗起", "时窗止", "值", "统计方式",
}

// csvTimeLayout CSV 时间列格式：本地可读（与前端展示口径一致，避免 RFC3339 的 T/Z 噪音）。
const csvTimeLayout = "2006-01-02 15:04:05"

// ExportRow 是一行导出数据点（来源无关的中间表示）。
// dashboard 来源由 aggregator.Row 映射；adhoc 来源由结果表行映射。
type ExportRow struct {
	Device     string  // 设备：OUI/SN 或聚合标识（AGGREGATED / 设备组名 等）
	CellPLMN   string  // 小区/PLMN：object_ldn 原文（空 → 空串）
	MetricCode string  // 指标编号：metric_path（K/C 编号）
	MetricName string  // 指标名：DisplayName（回填的本地化名）
	MetricType string  // 类型：counter / kpi
	Granularity string // 粒度：15min/hourly/...
	Time        time.Time
	StartTime   time.Time
	EndTime     time.Time
	Value       float64
	StatisType  string // 统计方式：sum/avg/max/min/pct，空 → 空串
}

// CSVWriter 流式写 CSV：先写 BOM + 表头，再逐行写数据点，统计写入行数。
// 不缓存全量行——每行 Write 立即 Flush 到底层 io.Writer（通常接 io.Pipe → 对象存储）。
type CSVWriter struct {
	w        *csv.Writer
	rowCount int64
}

// NewCSVWriter 构造 CSVWriter 并立即写出 BOM + 表头。
func NewCSVWriter(out io.Writer) (*CSVWriter, error) {
	if _, err := out.Write(utf8BOM); err != nil {
		return nil, err
	}
	cw := csv.NewWriter(out)
	if err := cw.Write(csvHeader); err != nil {
		return nil, err
	}
	return &CSVWriter{w: cw}, nil
}

// WriteRow 写一行数据点。空值（统计方式 / 小区·PLMN）输出空串。
func (c *CSVWriter) WriteRow(r ExportRow) error {
	rec := []string{
		r.Device,
		r.CellPLMN,
		r.MetricCode,
		r.MetricName,
		r.MetricType,
		r.Granularity,
		formatTime(r.Time),
		formatTime(r.StartTime),
		formatTime(r.EndTime),
		strconv.FormatFloat(r.Value, 'f', -1, 64),
		r.StatisType,
	}
	if err := c.w.Write(rec); err != nil {
		return err
	}
	c.rowCount++
	return nil
}

// Flush 刷新底层缓冲并返回写入过程中的错误。
func (c *CSVWriter) Flush() error {
	c.w.Flush()
	return c.w.Error()
}

// RowCount 返回已写出的数据行数（不含表头）。
func (c *CSVWriter) RowCount() int64 { return c.rowCount }

// formatTime 空时间输出空串，否则本地可读格式。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(csvTimeLayout)
}
