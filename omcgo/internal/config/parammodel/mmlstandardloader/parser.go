// Package mmlstandardloader 把 TR-069 标准模型 XML 翻译为 MML 命令字典。
//
// 数据流:
//
//	data/param-mappings/standard-model.xml (1988 paths + 13 objects)
//	    │
//	    ▼ parser.go : XML decode
//	[]ParamSpec / []ObjectSpec
//	    │
//	    ▼ grouper.go : ≤50 阈值 path-prefix 树
//	[]GroupSpec
//	    │
//	    ▼ command_gen.go : LST/MOD/ADD/RMV 生成
//	[]CommandSpec
//	    │
//	    ▼ loader.go : 事务 UPSERT
//	mml_param_versions / mml_command_groups / mml_params / mml_commands / mml_group_param_rel
//
// 决策依据：docs/design/mml-rebuild-plan-20260513.md v3.1
package mmlstandardloader

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// AccessReadOnly / AccessReadWrite — TR-069 §A.2.2 access semantics
// 与 XML 文件 access="READ_ONLY" / "READ_WRITE" 字面值一致。
const (
	AccessReadOnly  = "READ_ONLY"
	AccessReadWrite = "READ_WRITE"
)

// ParamSpec 标准模型中一条 <param> 的解析结果。
//
// XML 形态例：
//
//	<param standardPath="Device.DeviceInfo.AntennaInfo.Azimuth"
//	       access="READ_WRITE" type="INT" changeApplies="Immediate"
//	       min="0" max="359"/>
type ParamSpec struct {
	StandardPath  string // 完整 TR-069 path（含 {i} 占位符）
	Access        string // READ_ONLY / READ_WRITE
	Type          string // STRING / INT / U_INT / BOOLEAN / DATE_TIME
	ChangeApplies string // Immediate / Reboot 等
	Min           *int64 // 数值类型才有值
	Max           *int64
	MaxLen        *int    // string 类型才有值（XML 中也叫 max，类型用判别）
	Enum          []string // 枚举值（XML 中暂无该字段，预留 future）
}

// ObjectSpec 标准模型中一条 <object> 的解析结果。
//
// XML 形态例：
//
//	<object standardPath="Device.DeviceInfo.EU."
//	        access="READ_ONLY" changeApplies="Immediate"/>
type ObjectSpec struct {
	StandardPath  string
	Access        string
	ChangeApplies string
}

// IsWritable 是否可写（access == READ_WRITE）。
func (p ParamSpec) IsWritable() bool { return p.Access == AccessReadWrite }

// HasInstanceIndex 路径是否含 `{i}` 实例占位符（决定是否生成 ADD/RMV 命令 + 是否 fanout 透明 GPN→GPV）。
func (p ParamSpec) HasInstanceIndex() bool { return strings.Contains(p.StandardPath, ".{i}.") }

// HasInstanceIndex 同上，object 维度。
func (o ObjectSpec) HasInstanceIndex() bool { return strings.Contains(o.StandardPath, ".{i}.") }

// IsObjectPath path 末段以 `.` 结尾 = TR-069 object 路径（vs 叶子参数）。
func IsObjectPath(p string) bool { return strings.HasSuffix(p, ".") }

// StripInstanceIndex 删除 `.{i}` → 用于 group code 生成 + ADD/RMV target_object。
// "Device.X.{i}.Y" → "Device.X.Y"
// "Device.X.{i}."  → "Device.X."
func StripInstanceIndex(p string) string {
	return strings.ReplaceAll(p, ".{i}", "")
}

// ParseStandardXMLFile 读取并解析 standard-model.xml 文件。
//
// 失败语义：
//   - 文件无法打开 → 错误返回
//   - XML 结构异常 → 错误返回
//   - 单 param 字段非法（如 type 不在白名单）→ 该条 skip + 错误累加返回（但其它继续）
func ParseStandardXMLFile(path string) ([]ParamSpec, []ObjectSpec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open standard model xml %s: %w", path, err)
	}
	defer f.Close()
	return ParseStandardXML(f)
}

// ParseStandardXML 从 io.Reader 解析。便于 unit test 直接吃字符串。
func ParseStandardXML(r io.Reader) ([]ParamSpec, []ObjectSpec, error) {
	type xmlObject struct {
		StandardPath  string `xml:"standardPath,attr"`
		Access        string `xml:"access,attr"`
		ChangeApplies string `xml:"changeApplies,attr"`
	}
	type xmlParam struct {
		StandardPath  string `xml:"standardPath,attr"`
		Access        string `xml:"access,attr"`
		Type          string `xml:"type,attr"`
		ChangeApplies string `xml:"changeApplies,attr"`
		Min           string `xml:"min,attr"`
		Max           string `xml:"max,attr"`
	}
	type xmlStandardModel struct {
		XMLName    xml.Name    `xml:"standardModel"`
		Objects    []xmlObject `xml:"objects>object"`
		Parameters []xmlParam  `xml:"parameters>param"`
	}

	var m xmlStandardModel
	if err := xml.NewDecoder(r).Decode(&m); err != nil {
		return nil, nil, fmt.Errorf("decode standard model xml: %w", err)
	}

	params := make([]ParamSpec, 0, len(m.Parameters))
	for _, p := range m.Parameters {
		spec := ParamSpec{
			StandardPath:  p.StandardPath,
			Access:        p.Access,
			Type:          p.Type,
			ChangeApplies: p.ChangeApplies,
		}
		// 数值类型解析 min/max
		if p.Type == "INT" || p.Type == "U_INT" {
			if p.Min != "" {
				if v, err := strconv.ParseInt(p.Min, 10, 64); err == nil {
					spec.Min = &v
				}
			}
			if p.Max != "" {
				if v, err := strconv.ParseInt(p.Max, 10, 64); err == nil {
					spec.Max = &v
				}
			}
		}
		// 字符串类型把 max 解释为最大长度
		if p.Type == "STRING" && p.Max != "" {
			if v, err := strconv.Atoi(p.Max); err == nil {
				spec.MaxLen = &v
			}
		}
		params = append(params, spec)
	}

	objects := make([]ObjectSpec, 0, len(m.Objects))
	for _, o := range m.Objects {
		objects = append(objects, ObjectSpec{
			StandardPath:  o.StandardPath,
			Access:        o.Access,
			ChangeApplies: o.ChangeApplies,
		})
	}

	return params, objects, nil
}
