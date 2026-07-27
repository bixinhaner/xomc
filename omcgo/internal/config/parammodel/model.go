// Package parammodel 提供 T-0098 参数模型字典（设计 §1）。
//
// P1-06 阶段交付：
//   - model.go    XML 解析结构 + 领域类型
//   - loader.go   实现 dictloader.Loader 接口 — 启动期 9 个 paramModel XML + 1 个 standard XML 加载
//
// Phase 2 接力：Registry / Translator / Intersect / Cache（设计 §1.4-§1.10）。
package parammodel

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// ── Domain types（直接对应 §1.2 表 schema）─────────────────────────────────

// ParamModel 对应 param_models 表（设计 §1.2.1）。
type ParamModel struct {
	ID           uuid.UUID
	Name         string
	TotalEntries int
	TotalObjects int
	TotalParams  int
	Description  string
	IsActive     bool
	LoadedFrom   string
}

// ParamMapping 对应 param_mappings 与 discovered_param_mappings 两张表（设计 §1.4 合并 struct）。
//
// 区分依据 SoftwareVersion：
//   - SoftwareVersion == nil → 来自 param_mappings（默认映射）
//   - SoftwareVersion != nil → 来自 discovered_param_mappings（设备发现映射）
//
// 注意：discovered 表无 param_model_id 列；从 discovered 读出的 ParamMapping
// 其 ParamModelID 为零值，调用方若需要可从 product.ParamModelID 反查。
type ParamMapping struct {
	ID                uuid.UUID
	ParamModelID      uuid.UUID
	StandardPath      string
	PrivatePath       string
	EntryType         string // "object" | "parameter"
	Access            string
	DataType          string
	ChangeApplies     string
	MinValue          *int64
	MaxValue          *int64
	DefaultValue      *string
	ValidationPattern *string
	// T-0158: 枚举值列表（CSV 字符串如 "0,1" / "25,50,75,100"），nil 表示非枚举类型。
	// 类型按 type 字段解释：int / unsignedInt 用作下发值；string 也可枚举。
	EnumValues *string
	// T-0158: 枚举显示标签（与 EnumValues 一一对应 CSV，如 "Macro,home" / "CELL_BW_25(5M),..."），
	// nil 时前端用 EnumValues 作 label。
	EnumLabels *string
	// T-0159: 交叉镜像约束 — 完整 standardPath（含 {i}），指向被镜像字段；
	// nil 表示无约束。语义对称：两端均填对方路径。典型用途 TDD 上下行带宽必须相等。
	MirrorWith      *string
	IsStorable      bool
	IsActive        bool
	IsSupported     bool    // T-0103 XML supported="false" → false；path-b sync 据此过滤
	SoftwareVersion *string // 仅 discovered 映射非 nil
	// T-PMSRC: 行级来源。"builtin"=XML 加载(不可删,重载会重建);"custom"=管理员经 UI
	// 新增或对 builtin 编辑后转化的覆盖项(可删,重载不覆盖)。
	Source string
}

// MappingSource 标记 MappingSet 来源，区分精确的 discovered 与降级的 default（设计 §1.6）。
type MappingSource string

// 取值。
const (
	MappingSourceDiscovered MappingSource = "discovered"
	MappingSourceDefault    MappingSource = "default"
)

// MappingSet 是 Registry 一次取出的映射集合，承载双向翻译所需的全部数据。
type MappingSet struct {
	ProductID       uuid.UUID // 仅 discovered 来源时有意义；default 时为零值
	ParamModelID    uuid.UUID // 默认映射的所属模型；discovered 来源会从 product 反查回填
	SoftwareVersion string    // discovered 来源的 swVersion；default 时为空串
	Source          MappingSource
	Mappings        []ParamMapping // 原始顺序保留（供 P2-04 sync.go 去重前缀使用）
}

// TranslationResult 是 Translator.ToPrivate / ToStandard 的统一返回。
//
// Found=false 表示在当前 MappingSet 中未找到对应路径；Mapping 为 nil。
// Translated 在 Found=false 时也设回 Original，便于调用侧无脑回退。
type TranslationResult struct {
	Original   string
	Translated string
	Found      bool
	Mapping    *ParamMapping
}

// StandardParam 对应 standard_params 表（设计 §1.2.4）。
type StandardParam struct {
	StandardPath  string
	EntryType     string
	Access        string
	DataType      string
	ChangeApplies string
	MinValue      *int64
	MaxValue      *int64
	UpdatedAt     time.Time
	UpdatedFields []string
}

// ── XML 解析结构（设计 §1.7 格式 A 与 D）────────────────────────────────

// xmlParameterModel 解析 paramModel.xml（格式 A）。
type xmlParameterModel struct {
	XMLName      xml.Name        `xml:"parameterModel"`
	ParamModel   string          `xml:"paramModel,attr"`
	TotalEntries int             `xml:"totalEntries,attr"`
	Objects      []xmlParamEntry `xml:"objects>object"`
	Params       []xmlParamEntry `xml:"parameters>param"`
}

// xmlParamEntry 同时复用于 <object> 与 <param>（字段集合一致，仅元素名不同）。
// 注意：XML 中 <object> 用 name 属性、<param> 也用 name 属性；store 仅 <param> 出现。
type xmlParamEntry struct {
	Name              string `xml:"name,attr"`
	StandardPath      string `xml:"standardPath,attr"`
	Access            string `xml:"access,attr"`
	DataType          string `xml:"type,attr"`
	ChangeApplies     string `xml:"changeApplies,attr"`
	Min               string `xml:"min,attr"`
	Max               string `xml:"max,attr"`
	DefaultValue      string `xml:"defaultValue,attr"`
	ValidationPattern string `xml:"validationPattern,attr"`
	// T-0158: 枚举值 / 标签（CSV，如 "0,1" / "Macro,home"）。
	// labels 与 values 一一对应；空 labels 时 UI 直接用 values 显示。
	EnumValues string `xml:"enumValues,attr"`
	EnumLabels string `xml:"enumLabels,attr"`
	// T-0159: 交叉镜像 — 完整 standardPath（含 {i}），指向被镜像字段。两端对称填写。
	MirrorWith string `xml:"mirrorWith,attr"`
	Store      string `xml:"store,attr"`     // "true" | "false" | ""（缺省视 true）
	Supported  string `xml:"supported,attr"` // T-0103 "false" → 标记设备不支持；缺省/其它视 true
}

// xmlStandardModel 解析 standard-model.xml（格式 D）。
type xmlStandardModel struct {
	XMLName    xml.Name           `xml:"standardModel"`
	TotalPaths int                `xml:"totalPaths,attr"`
	Objects    []xmlStandardEntry `xml:"objects>object"`
	Params     []xmlStandardEntry `xml:"parameters>param"`
}

type xmlStandardEntry struct {
	StandardPath  string `xml:"standardPath,attr"`
	Access        string `xml:"access,attr"`
	DataType      string `xml:"type,attr"`
	ChangeApplies string `xml:"changeApplies,attr"`
	Min           string `xml:"min,attr"`
	Max           string `xml:"max,attr"`
}
