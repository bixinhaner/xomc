package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

// utf8BOM 是 UTF-8 字节顺序标记，写在 CSV 首部让 Excel 双击直接按 UTF-8 解析、中文不乱码。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// csvTimeLayout CSV 时间列格式：本地可读（与前端展示口径一致，避免 RFC3339 的 T/Z 噪音）。
const csvTimeLayout = "2006-01-02 15:04:05"

const missingMetricValuePlaceholder = "-"

// ExportRow 是一行导出数据点（来源无关的中间表示）。
// dashboard 来源由 aggregator.Row 映射；adhoc 来源由结果表行映射。
// 横表模式下按 (Device, CellPLMN, Time) 行键聚合，每个 MetricCode 摊成一列。
type ExportRow struct {
	Device      string // 设备：device 维度为 SN（与页面一致），聚合维度为对象标识（设备组名 / 产品名 / 全网 等）
	Technology  string // 制式：仅 device_group 维度从 object_ldn 解析出（LTE/NR/GSM，大写），其余维度空串
	CellPLMN    string // 测量对象：object_ldn 原文（dashboard/adhoc 可拆成 Cell ID / PLMN；kpi_query 原样输出）
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

// WideColumn 是横表里一个指标列：列名 = 指标友好名（与页面表格表头一致，如「下行数据业务流量」）。
type WideColumn struct {
	Code string // 指标编号 metric_path（仍作单元格取值的列键，不再进表头）
	Type string // counter / kpi（保留供来源映射，不再进表头）
	Name string // 本地化指标名（列发现阶段回填，回退编号本身）
}

// header 返回该指标列的 CSV 列名 = 指标友好名（与页面表格一致），名缺失时回退编号。
func (c WideColumn) header() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Code
}

// wideKey 是横表行键（同一时间桶内唯一标识一行）：设备 + 制式 + 小区/PLMN。
// 粒度 / 时间 / 时窗起止在同一时间桶内恒定，故不入行键。
// 制式入行键：device_group 维度下同组 lte/nr 同名同维度，必须靠制式区分成两行（否则被并成一行）。
type wideKey struct {
	device   string
	tech     string
	cellPLMN string
}

// WideCSVWriter 流式写横表 CSV：先写 BOM + 表头（固定列 + 每个所选指标一列），
// 再按「时间桶」缓冲——累计同一 time 的全部数据点 → 按 (设备,小区) 摊成横行 → time 变了就 flush。
//
// 内存只占一个时间桶（设备 × 小区 × 指标列宽），仍是流式、不破坏无上限导出。
// 前提：上游 RowSource 按 time 连续返回（device/adhoc 走 (time,id) keyset、aggregate 走 time DESC，
// 同一 time 的行天然连续，可能跨多个批次但不交错）。
type WideCSVWriter struct {
	w                             *csv.Writer
	cols                          []WideColumn
	colIdx                        map[string]int // 指标编号 → 列下标
	rowCount                      int64          // 已写出的横行数（= 去重 (设备×小区×时间) 行数）
	firstHeader                   string         // 首列表头（随 adhoc 维度变：设备 / 设备组 / 产品 / 频段 / 全网 / 聚合组）
	includeTech                   bool           // 是否含「制式」列（仅 device_group 维度为 true）
	includeCell                   bool           // 是否含「小区/PLMN」列（仅 device 维度为 true）
	includeMeasurementObject      bool           // 是否把 object_ldn 原样输出为「测量对象」列（指标查询页导出）
	fixedCount                    int            // 固定行键列数（含/不含制式列与小区列各态），用于行容量预分配
	missingMetricValuePlaceholder string         // 指标缺失时的单元格占位符；空串表示沿用 CSV 空单元格。
	outputLocation                *time.Location // CSV 时间列输出时区；nil 入口统一回退 UTC。

	// 当前时间桶状态。
	active              bool
	bTime, bStart, bEnd time.Time
	order               []wideKey            // 行键首现顺序（flush 前再排序求稳定输出）
	cells               map[wideKey][]string // 行键 → 各指标列值（len = len(cols)）
}

// NewWideCSVWriter 构造 WideCSVWriter 并立即写出 BOM + 表头（固定列 + 指标列）。
// 列序与页面表格一致：开始时间 / 结束时间 / 设备(对象) / [制式] / [Cell ID / PLMN] / 指标...
// firstColHeader 为设备(对象)列表头（按维度自适应：设备 SN / 设备组 / 产品 / 频段 / 全网 / 聚合组）；
// includeTech 控制是否在对象列后输出「制式」列（仅 device_group 维度为 true，与页面表格一致）；
// includeCell 控制是否输出「Cell ID / PLMN」两列（仅 device 维度为 true，聚合维度小区已聚掉、不含）。
func NewWideCSVWriter(out io.Writer, firstColHeader string, includeTech, includeCell bool, cols []WideColumn) (*WideCSVWriter, error) {
	return newWideCSVWriter(out, firstColHeader, includeTech, includeCell, cols, "")
}

func newWideCSVWriter(out io.Writer, firstColHeader string, includeTech, includeCell bool, cols []WideColumn, missingPlaceholder string) (*WideCSVWriter, error) {
	return newWideCSVWriterWithLocation(out, firstColHeader, includeTech, includeCell, cols, missingPlaceholder, nil)
}

func newWideCSVWriterWithLocation(out io.Writer, firstColHeader string, includeTech, includeCell bool, cols []WideColumn, missingPlaceholder string, outputLocation *time.Location) (*WideCSVWriter, error) {
	return newWideCSVWriterWithLayout(out, firstColHeader, includeTech, includeCell, false, cols, missingPlaceholder, outputLocation)
}

func newWideCSVWriterWithMeasurementObject(out io.Writer, firstColHeader string, includeTech, includeMeasurementObject bool, cols []WideColumn, missingPlaceholder string, outputLocation *time.Location) (*WideCSVWriter, error) {
	return newWideCSVWriterWithLayout(out, firstColHeader, includeTech, false, includeMeasurementObject, cols, missingPlaceholder, outputLocation)
}

func newWideCSVWriterWithLayout(out io.Writer, firstColHeader string, includeTech, includeCell, includeMeasurementObject bool, cols []WideColumn, missingPlaceholder string, outputLocation *time.Location) (*WideCSVWriter, error) {
	return newWideCSVWriterWithLocale(out, firstColHeader, includeTech, includeCell, includeMeasurementObject, cols, missingPlaceholder, outputLocation, appcontext.LocaleZH)
}

func newWideCSVWriterWithLocale(out io.Writer, firstColHeader string, includeTech, includeCell, includeMeasurementObject bool, cols []WideColumn, missingPlaceholder string, outputLocation *time.Location, loc appcontext.Locale) (*WideCSVWriter, error) {
	if _, err := out.Write(utf8BOM); err != nil {
		return nil, err
	}
	if outputLocation == nil {
		outputLocation = time.UTC
	}
	cw := csv.NewWriter(out)
	headers := localizedCSVHeaders(loc)
	fixed := []string{headers.startTime, headers.endTime, firstColHeader}
	if includeTech {
		fixed = append(fixed, headers.technology)
	}
	if includeCell {
		fixed = append(fixed, "Cell ID", "PLMN")
	}
	if includeMeasurementObject {
		fixed = append(fixed, headers.measurementObject)
	}
	header := make([]string, 0, len(fixed)+len(cols))
	header = append(header, fixed...)
	idx := make(map[string]int, len(cols))
	for i, c := range cols {
		header = append(header, c.header())
		idx[c.Code] = i
	}
	if err := cw.Write(header); err != nil {
		return nil, err
	}
	writer := &WideCSVWriter{
		w: cw, cols: cols, colIdx: idx,
		firstHeader: firstColHeader, includeTech: includeTech, includeCell: includeCell, includeMeasurementObject: includeMeasurementObject, fixedCount: len(fixed),
		missingMetricValuePlaceholder: missingPlaceholder,
		outputLocation:                outputLocation,
	}
	return writer, nil
}

type csvHeaders struct {
	startTime         string
	endTime           string
	technology        string
	measurementObject string
}

func localizedCSVHeaders(loc appcontext.Locale) csvHeaders {
	if loc == appcontext.LocaleEN {
		return csvHeaders{
			startTime:         "Start Time",
			endTime:           "End Time",
			technology:        "Technology",
			measurementObject: "Measurement Object",
		}
	}
	return csvHeaders{
		startTime:         "开始时间",
		endTime:           "结束时间",
		technology:        "制式",
		measurementObject: "测量对象",
	}
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
		c.bTime, c.bStart, c.bEnd = r.Time, r.StartTime, r.EndTime
		c.order = nil
		c.cells = make(map[wideKey][]string)
	}
	k := wideKey{device: r.Device, tech: r.Technology, cellPLMN: r.CellPLMN}
	cells, ok := c.cells[k]
	if !ok {
		cells = make([]string, len(c.cols))
		if c.missingMetricValuePlaceholder != "" {
			for i := range cells {
				cells[i] = c.missingMetricValuePlaceholder
			}
		}
		c.cells[k] = cells
		c.order = append(c.order, k)
	}
	if idx, ok := c.colIdx[r.MetricCode]; ok {
		if !isFiniteMetricCSVValue(r.Value) {
			cells[idx] = c.missingMetricValuePlaceholder
		} else {
			cells[idx] = formatMetricCSVValue(r.Value)
		}
	}
	return nil
}

func isFiniteMetricCSVValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func formatMetricCSVValue(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

// flushBucket 把当前时间桶的所有行键按 (设备,小区) 排序后逐行写出，然后清空桶。
// 列序与表头一致：开始时间 / 结束时间 / 设备(对象) / [Cell ID / PLMN] / 各指标值。
func (c *WideCSVWriter) flushBucket() error {
	sort.Slice(c.order, func(i, j int) bool {
		if c.order[i].device != c.order[j].device {
			return c.order[i].device < c.order[j].device
		}
		if c.order[i].tech != c.order[j].tech {
			return c.order[i].tech < c.order[j].tech
		}
		return c.order[i].cellPLMN < c.order[j].cellPLMN
	})
	// 开始时间取时窗起（与页面"开始时间"同口径）；缺失时回退桶时间。
	startStr := c.formatTime(c.bStart)
	if c.bStart.IsZero() {
		startStr = c.formatTime(c.bTime)
	}
	endStr := c.formatTime(c.bEnd)
	for _, k := range c.order {
		rec := make([]string, 0, c.fixedCount+len(c.cols))
		rec = append(rec, startStr, endStr, k.device)
		if c.includeTech {
			rec = append(rec, k.tech)
		}
		if c.includeCell {
			cellID, plmn := parseObjectLDN(k.cellPLMN)
			rec = append(rec, cellID, plmn)
		}
		if c.includeMeasurementObject {
			objectLDN := k.cellPLMN
			if objectLDN == "" {
				objectLDN = c.missingMetricValuePlaceholder
			}
			rec = append(rec, objectLDN)
		}
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

// parseObjectLDN 从 object_ldn 原文解析出 Cell ID 与 PLMN（与前端 parseObjectLdn 同口径，大小写不敏感）。
// 支持 "Cellid=111172245" / "Cellid=111172245,PLMN=46068" / "PLMN=...,Cellid=..."（顺序无关）；空串 → "","" 。
func parseObjectLDN(ldn string) (cellID, plmn string) {
	if ldn == "" {
		return "", ""
	}
	for _, kv := range strings.Split(ldn, ",") {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[:idx]))
		val := strings.TrimSpace(kv[idx+1:])
		if val == "" {
			continue
		}
		switch key {
		case "cellid":
			cellID = val
		case "plmn":
			plmn = val
		}
	}
	return cellID, plmn
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

// formatTime 空时间输出空串，否则按导出时区输出本地可读格式。
func (c *WideCSVWriter) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	loc := c.outputLocation
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format(csvTimeLayout)
}
