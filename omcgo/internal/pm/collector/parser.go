package collector

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// parseReadBufferSize 是包在传入 io.Reader 外的 bufio 缓冲上限（64 KiB）。
//
// PM 文件经 MinIO 对象流（GetObject 返回的 *minio.Object）逐块到达。xml.Decoder 本身是
// 流式的（按 token 增量解析，不会先把整个文件读进内存），但底层 reader 若每次只回几字节
// 会放大 syscall 次数。固定 64 KiB bufio 把读放大到顺序大块、内存占用恒定（与文件大小无关），
// 既保住流式解析的低峰值内存，又减少 IO 往返。改成 io.Reader 入参后大文件不再整文件驻留。
const parseReadBufferSize = 64 * 1024

// PMFileContent holds the parsed content of a PM XML file.
//
// T-0164-P4 / G4 加 FileBeginTime / FileEndTime / IngestTime 三时间字段：
//   - FileBeginTime — 基站采集窗口起点（fileHeader/measCollec/@beginTime，基站时钟）
//   - FileEndTime   — 基站采集窗口止点（fileFooter/measCollec/@endTime，基站时钟）
//   - IngestTime    — OMC 入库时刻（Parse 完成时 time.Now()，OMC 时钟）
//
// 用途：上报延迟监控、时钟漂移排查、补传识别、唯一性维度扩展。
// 旧字段 CollectTime 沿用 granPeriod.endTime 作为兼容（指向 measInfo 的窗口止点）。
//
// 缺失场景（fallback）：
//   - 无 fileHeader/measCollec/@beginTime → FileBeginTime = FileEndTime - granPeriod.duration（间接推算）
//   - 无 fileFooter/measCollec/@endTime   → FileEndTime = granPeriod.endTime（measInfo 级窗口止点）
//   - 两段都无                            → FileBeginTime / FileEndTime 均为 zero time.Time{}（消费方按 zero 判定）
//
// G3 合表为 pm_metrics 后，三字段将写入新表 start_time / end_time / ingest_time 列（NOT NULL）。
type PMFileContent struct {
	DeviceSN      string
	CollectTime   time.Time // 兼容：沿用 granPeriod.endTime
	FileBeginTime time.Time // G4: fileHeader/measCollec/@beginTime
	FileEndTime   time.Time // G4: fileFooter/measCollec/@endTime
	IngestTime    time.Time // G4: Parse 完成时刻（OMC 时钟）
	Granularity   int       // minutes
	Counters      []model.PMCounter
}

// PMXMLParser parses 3GPP 32.435 format PM XML files using streaming XML decoder.
type PMXMLParser struct{}

// NewPMXMLParser creates a new PM XML parser.
func NewPMXMLParser() *PMXMLParser {
	return &PMXMLParser{}
}

// XML structures for 3GPP 32.435 PM file format.
type xmlMeasInfo struct {
	MeasInfoId string         `xml:"measInfoId,attr"`
	GranPeriod xmlGranPeriod  `xml:"granPeriod"`
	MeasTypes  []xmlMeasType  `xml:"measType"`
	MeasValues []xmlMeasValue `xml:"measValue"`
}

type xmlGranPeriod struct {
	Duration string `xml:"duration,attr"`
	EndTime  string `xml:"endTime,attr"`
}

type xmlMeasType struct {
	P    string `xml:"p,attr"`
	Name string `xml:",chardata"`
}

type xmlMeasValue struct {
	MeasObjLdn string `xml:"measObjLdn,attr"`
	Results    []xmlR `xml:"r"`
}

type xmlR struct {
	P     string `xml:"p,attr"`
	Value string `xml:",chardata"`
}

type xmlManagedElement struct {
	LocalDn   string `xml:"localDn,attr"`
	SwVersion string `xml:"swVersion,attr"`
}

// xmlFileHeaderSection 解析 <fileHeader><measCollec beginTime="..."/></fileHeader> 段。
// 老格式 PM 文件可能 fileHeader 仅含 dnPrefix/vendorName 等属性，无 measCollec 子元素 —
// 此时 BeginTime 为空字符串，调用方按 zero time 处理。
type xmlFileHeaderSection struct {
	MeasCollec xmlCollecAttrs `xml:"measCollec"`
}

// xmlFileFooterSection 解析 <fileFooter><measCollec endTime="..."/></fileFooter> 段。
type xmlFileFooterSection struct {
	MeasCollec xmlCollecAttrs `xml:"measCollec"`
}

// xmlCollecAttrs 是 fileHeader/measCollec 与 fileFooter/measCollec 共用的属性结构。
type xmlCollecAttrs struct {
	BeginTime string `xml:"beginTime,attr"`
	EndTime   string `xml:"endTime,attr"`
}

// Parse parses a PM XML file from the given reader.
//
// 流式解析（issue #14）：用 xml.Decoder 按 token 增量消费 r，绝不把整个文件读进内存；
// 外面再包一层固定 64 KiB bufio 让底层 reader（MinIO 对象流）的小块读聚成顺序大块，
// 峰值内存与文件大小解耦。只有最终产出的 Counters 切片随文件内容线性增长（BatchInsert
// 契约要求一次拿到全部 counter），这是必要的而非可避免的驻留。
func (p *PMXMLParser) Parse(r io.Reader, deviceID uuid.UUID) (*PMFileContent, error) {
	decoder := xml.NewDecoder(bufio.NewReaderSize(r, parseReadBufferSize))

	content := &PMFileContent{}
	var inMeasData bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode pm xml: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "fileHeader":
			// G4: 读 fileHeader/measCollec/@beginTime。老格式无 measCollec 子元素时 BeginTime 为空，FileBeginTime 保持 zero。
			var fh xmlFileHeaderSection
			if err := decoder.DecodeElement(&fh, &se); err != nil {
				return nil, fmt.Errorf("decode fileHeader: %w", err)
			}
			if fh.MeasCollec.BeginTime != "" {
				if t, perr := time.Parse(time.RFC3339, fh.MeasCollec.BeginTime); perr == nil {
					content.FileBeginTime = t
				}
			}

		case "fileFooter":
			// G4: 读 fileFooter/measCollec/@endTime。老格式无此段时 FileEndTime 保持 zero。
			var ff xmlFileFooterSection
			if err := decoder.DecodeElement(&ff, &se); err != nil {
				return nil, fmt.Errorf("decode fileFooter: %w", err)
			}
			if ff.MeasCollec.EndTime != "" {
				if t, perr := time.Parse(time.RFC3339, ff.MeasCollec.EndTime); perr == nil {
					content.FileEndTime = t
				}
			}

		case "managedElement":
			var me xmlManagedElement
			if err := decoder.DecodeElement(&me, &se); err != nil {
				return nil, fmt.Errorf("decode managedElement: %w", err)
			}
			content.DeviceSN = extractDeviceSN(me.LocalDn)
			inMeasData = true

		case "measInfo":
			if !inMeasData {
				continue
			}
			var mi xmlMeasInfo
			if err := decoder.DecodeElement(&mi, &se); err != nil {
				return nil, fmt.Errorf("decode measInfo: %w", err)
			}

			// Parse granularity and end time
			granSeconds := parseDuration(mi.GranPeriod.Duration)
			content.Granularity = granSeconds / 60
			if content.Granularity == 0 {
				content.Granularity = 15
			}

			collectTime, timeErr := time.Parse(time.RFC3339, mi.GranPeriod.EndTime)
			if timeErr != nil {
				collectTime = time.Now()
			}
			content.CollectTime = collectTime

			// Build counter name index
			typeIndex := make(map[string]string, len(mi.MeasTypes))
			for _, mt := range mi.MeasTypes {
				typeIndex[mt.P] = strings.TrimSpace(mt.Name)
			}

			// Extract counters from each measValue
			for _, mv := range mi.MeasValues {
				cellID := extractCellID(mv.MeasObjLdn)
				counterGroup := mi.MeasInfoId

				for _, r := range mv.Results {
					counterName, ok := typeIndex[r.P]
					if !ok {
						continue
					}
					value, parseErr := strconv.ParseFloat(strings.TrimSpace(r.Value), 64)
					if parseErr != nil {
						continue
					}
					content.Counters = append(content.Counters, model.PMCounter{
						Time:     collectTime,
						DeviceID: deviceID,
						DeviceSN: content.DeviceSN, // T-0164-P3: TR-069 SN（parser 从 managedElement.LocalDn 解析）
						// OUI 由 collector 在 BatchInsert 前从 payload 统一填充（parser 不解析 OUI）
						CellID:       cellID,
						CounterGroup: counterGroup,
						CounterName:  counterName,
						CounterValue: value,
						Granularity:  content.Granularity,
					})
				}
			}
		}
	}

	if len(content.Counters) == 0 {
		return nil, fmt.Errorf("no counters found in pm xml")
	}

	// G4: fallback 推断 + 设 IngestTime。
	// 如果 fileFooter/measCollec/@endTime 缺失，FileEndTime 走 collectTime（measInfo 级窗口止点）兜底。
	// 如果 fileHeader/measCollec/@beginTime 缺失，FileBeginTime 走 collectTime - granularity 兜底。
	// 单 measInfo 文件这两个兜底足够；多 measInfo 文件可能丢失精度（首个 measInfo 的 collectTime），
	// 调用方需自行评估是否接受。
	if content.FileEndTime.IsZero() && !content.CollectTime.IsZero() {
		content.FileEndTime = content.CollectTime
	}
	if content.FileBeginTime.IsZero() && !content.FileEndTime.IsZero() && content.Granularity > 0 {
		content.FileBeginTime = content.FileEndTime.Add(-time.Duration(content.Granularity) * time.Minute)
	}
	content.IngestTime = time.Now()

	return content, nil
}

// extractDeviceSN extracts the device serial number from a localDn string.
func extractDeviceSN(localDn string) string {
	parts := strings.Split(localDn, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "MeContext" {
			return kv[1]
		}
	}
	if len(parts) > 0 {
		kv := strings.SplitN(parts[len(parts)-1], "=", 2)
		if len(kv) == 2 {
			return kv[1]
		}
	}
	return localDn
}

// extractCellID extracts the cell ID from a measObjLdn string.
func extractCellID(measObjLdn string) string {
	parts := strings.Split(measObjLdn, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && (kv[0] == "CellId" || kv[0] == "NRCellDU" || kv[0] == "NRCellCU") {
			return kv[1]
		}
	}
	return measObjLdn
}

// parseDuration parses an ISO 8601 duration like "PT900S" to seconds.
func parseDuration(d string) int {
	d = strings.TrimPrefix(d, "PT")
	d = strings.TrimPrefix(d, "pt")
	if strings.HasSuffix(d, "S") || strings.HasSuffix(d, "s") {
		d = strings.TrimSuffix(d, "S")
		d = strings.TrimSuffix(d, "s")
		v, err := strconv.Atoi(d)
		if err != nil {
			return 900
		}
		return v
	}
	if strings.HasSuffix(d, "M") || strings.HasSuffix(d, "m") {
		d = strings.TrimSuffix(d, "M")
		d = strings.TrimSuffix(d, "m")
		v, err := strconv.Atoi(d)
		if err != nil {
			return 900
		}
		return v * 60
	}
	return 900
}
