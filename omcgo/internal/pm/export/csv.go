package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// utf8BOM 是 UTF-8 字节顺序标记，写在 CSV 首部让 Excel 双击直接按 UTF-8 解析、中文不乱码。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// wideFixedHeader 是横表的固定行键列（设计 §5.5 横表版）：每行一个 (设备 × 小区/PLMN × 时间) 行键，
// 其后跟所选指标各一列。去掉了长表的「指标编号 / 指标名 / 类型 / 值 / 统计方式」——指标改摊成列。
var wideFixedHeader = []string{"设备", "小区/PLMN", "粒度", "时间", "时窗起", "时窗止"}

// csvTimeLayout CSV 时间列格式：本地可读（与前端展示口径一致，避免 RFC3339 的 T/Z 噪音）。
const csvTimeLayout = "2006-01-02 15:04:05"

// ExportRow 是一行导出数据点（来源无关的中间表示）。
// dashboard 来源由 aggregator.Row 映射；adhoc 来源由结果表行映射。
// 横表模式下按 (Device, CellPLMN, Time) 行键聚合，每个 MetricCode 摊成一列。
type ExportRow struct {
	Device      string // 设备：OUI/SN 或聚合标识（AGGREGATED / 设备组名 等）
	CellPLMN    string // 小区/PLMN：object_ldn 原文（空 → 空串）
	MetricCode  string // 指标编号：metric_path（K/C 编号）
	MetricName  string // 指标名：DisplayName（横表里不进单元格，列名解析另走列发现）
	MetricType  string // 类型：counter / kpi
	Granularity string // 粒度：15min/hourly/...
	Time        time.Time
	StartTime   time.Time
	EndTime     time.Time
	Value       float64
	StatisType  string // 统计方式：横表已去除此列，仅保留字段兼容来源映射
}

// WideColumn 是横表里一个指标列：列名 = 「指标编号(指标名·类型)」（如 K900010015(下行数据业务流量·kpi)）。
type WideColumn struct {
	Code string // 指标编号 metric_path
	Type string // counter / kpi
	Name string // 本地化指标名（列发现阶段回填，回退编号本身）
}

// header 返回该指标列的 CSV 列名「编号(名·类型)」。
func (c WideColumn) header() string {
	return fmt.Sprintf("%s(%s·%s)", c.Code, c.Name, c.Type)
}

// wideKey 是横表行键（同一时间桶内唯一标识一行）：设备 + 小区/PLMN。
// 粒度 / 时间 / 时窗起止在同一时间桶内恒定，故不入行键。
type wideKey struct {
	device   string
	cellPLMN string
}

// WideCSVWriter 流式写横表 CSV：先写 BOM + 表头（固定列 + 每个所选指标一列），
// 再按「时间桶」缓冲——累计同一 time 的全部数据点 → 按 (设备,小区) 摊成横行 → time 变了就 flush。
//
// 内存只占一个时间桶（设备 × 小区 × 指标列宽），仍是流式、不破坏无上限导出。
// 前提：上游 RowSource 按 time 连续返回（device/adhoc 走 (time,id) keyset、aggregate 走 time DESC，
// 同一 time 的行天然连续，可能跨多个批次但不交错）。
type WideCSVWriter struct {
	w        *csv.Writer
	cols     []WideColumn
	colIdx   map[string]int // 指标编号 → 列下标
	rowCount int64          // 已写出的横行数（= 去重 (设备×小区×时间) 行数）

	// 当前时间桶状态。
	active                bool
	bTime, bStart, bEnd   time.Time
	bGran                 string
	order                 []wideKey            // 行键首现顺序（flush 前再排序求稳定输出）
	cells                 map[wideKey][]string // 行键 → 各指标列值（len = len(cols)）
}

// NewWideCSVWriter 构造 WideCSVWriter 并立即写出 BOM + 表头（固定列 + 指标列）。
func NewWideCSVWriter(out io.Writer, cols []WideColumn) (*WideCSVWriter, error) {
	if _, err := out.Write(utf8BOM); err != nil {
		return nil, err
	}
	cw := csv.NewWriter(out)
	header := make([]string, 0, len(wideFixedHeader)+len(cols))
	header = append(header, wideFixedHeader...)
	idx := make(map[string]int, len(cols))
	for i, c := range cols {
		header = append(header, c.header())
		idx[c.Code] = i
	}
	if err := cw.Write(header); err != nil {
		return nil, err
	}
	return &WideCSVWriter{w: cw, cols: cols, colIdx: idx}, nil
}

// AddRow 把一个数据点喂进当前时间桶；time 变化时先 flush 上一桶。
func (c *WideCSVWriter) AddRow(r ExportRow) error {
	if c.active && !r.Time.Equal(c.bTime) {
		if err := c.flushBucket(); err != nil {
			return err
		}
	}
	if !c.active {
		c.active = true
		c.bTime, c.bStart, c.bEnd, c.bGran = r.Time, r.StartTime, r.EndTime, r.Granularity
		c.order = nil
		c.cells = make(map[wideKey][]string)
	}
	k := wideKey{device: r.Device, cellPLMN: r.CellPLMN}
	cells, ok := c.cells[k]
	if !ok {
		cells = make([]string, len(c.cols))
		c.cells[k] = cells
		c.order = append(c.order, k)
	}
	if idx, ok := c.colIdx[r.MetricCode]; ok {
		cells[idx] = strconv.FormatFloat(r.Value, 'f', -1, 64)
	}
	return nil
}

// flushBucket 把当前时间桶的所有行键按 (设备,小区) 排序后逐行写出，然后清空桶。
func (c *WideCSVWriter) flushBucket() error {
	sort.Slice(c.order, func(i, j int) bool {
		if c.order[i].device != c.order[j].device {
			return c.order[i].device < c.order[j].device
		}
		return c.order[i].cellPLMN < c.order[j].cellPLMN
	})
	timeStr := formatTime(c.bTime)
	startStr := formatTime(c.bStart)
	endStr := formatTime(c.bEnd)
	for _, k := range c.order {
		rec := make([]string, 0, len(wideFixedHeader)+len(c.cols))
		rec = append(rec, k.device, k.cellPLMN, c.bGran, timeStr, startStr, endStr)
		rec = append(rec, c.cells[k]...)
		if err := c.w.Write(rec); err != nil {
			return err
		}
		c.rowCount++
	}
	c.active = false
	c.order = nil
	c.cells = nil
	return nil
}

// Flush 刷出最后一个时间桶 + 底层缓冲，返回写入过程中的错误。
func (c *WideCSVWriter) Flush() error {
	if c.active {
		if err := c.flushBucket(); err != nil {
			return err
		}
	}
	c.w.Flush()
	return c.w.Error()
}

// RowCount 返回已写出的横行数（不含表头）。
func (c *WideCSVWriter) RowCount() int64 { return c.rowCount }

// formatTime 空时间输出空串，否则本地可读格式。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(csvTimeLayout)
}
