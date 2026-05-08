package provision

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/omcgo/omcgo/internal/config/parammodel"
)

// T-0098 P5-01：旧 datamodel/xml_parser.go 已删除；本文件接管 CPE FileType=11
// 上传 XML 的解析职责，直接产出 []parammodel.CPEEntry，供 IntersectService 写
// discovered_param_mappings 使用。
//
// XML 结构与原 datamodel.XMLParameterModel 同源（vendor / networkType /
// objects / parameters），仅删去 datamodel.DataModel / Parameter 这层中间表示。

// xmlParameterModel 匹配 CPE 上传 XML 根节点。
type xmlParameterModel struct {
	XMLName    xml.Name      `xml:"parameterModel"`
	Vendor     string        `xml:"vendor,attr"`
	Objects    xmlObjectList `xml:"objects"`
	Parameters xmlParamList  `xml:"parameters"`
}

type xmlObjectList struct {
	Items []xmlObject `xml:"object"`
}

type xmlParamList struct {
	Items []xmlParam `xml:"param"`
}

type xmlObject struct {
	Name   string `xml:"name,attr"`
	Access string `xml:"access,attr"`
}

type xmlParam struct {
	Name          string `xml:"name,attr"`
	Access        string `xml:"access,attr"`
	Type          string `xml:"type,attr"`
	Min           string `xml:"min,attr"`
	Max           string `xml:"max,attr"`
	ChangeApplies string `xml:"changeApplies,attr"`
}

// parseCPEEntries 把 CPE 上传 XML 解析为 CPEEntry 列表。
//
// 解析规则：
//   - parameters 区块 → entry_type="parameter"；min/max 仅对数值类型设置
//   - objects 区块  → entry_type="object"
//   - access / change_applies 透传（IsAccessWritable 由消费者判定）
//   - 不计算 totalEntries 不校验（设计 §1.8 写路径只关心 privatePath 集合）
func parseCPEEntries(reader io.Reader) ([]parammodel.CPEEntry, error) {
	var doc xmlParameterModel
	dec := xml.NewDecoder(reader)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode parameter model XML: %w", err)
	}

	entries := make([]parammodel.CPEEntry, 0, len(doc.Parameters.Items)+len(doc.Objects.Items))
	for _, p := range doc.Parameters.Items {
		entries = append(entries, parammodel.CPEEntry{
			PrivatePath:   p.Name,
			EntryType:     "parameter",
			Access:        p.Access,
			DataType:      normalizeXMLParamType(p.Type),
			ChangeApplies: p.ChangeApplies,
			MinValue:      parseInt64Ptr(p.Min, p.Type),
			MaxValue:      parseInt64Ptr(p.Max, p.Type),
		})
	}
	for _, o := range doc.Objects.Items {
		entries = append(entries, parammodel.CPEEntry{
			PrivatePath: o.Name,
			EntryType:   "object",
			Access:      o.Access,
		})
	}
	return entries, nil
}

// normalizeXMLParamType 把 CPE 厂商 XML 中常见类型名归一到 TR-069 标准命名。
func normalizeXMLParamType(t string) string {
	switch strings.ToUpper(t) {
	case "STRING":
		return "string"
	case "U_INT", "UINT", "UNSIGNED_INT":
		return "unsignedInt"
	case "INT", "INTEGER":
		return "int"
	case "BOOLEAN", "BOOL":
		return "boolean"
	case "DATE_TIME", "DATETIME":
		return "dateTime"
	case "BASE64":
		return "base64"
	case "LONG":
		return "long"
	case "UNSIGNED_LONG", "U_LONG":
		return "unsignedLong"
	default:
		return "string"
	}
}

// parseInt64Ptr 仅对数值型参数解析 min/max；字符串类型忽略（min/max 在 STRING 下表示长度，
// 当前 mapping schema 不承载长度约束，故直接丢弃）。
func parseInt64Ptr(s, xmlType string) *int64 {
	if s == "" {
		return nil
	}
	if strings.ToUpper(xmlType) == "STRING" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
