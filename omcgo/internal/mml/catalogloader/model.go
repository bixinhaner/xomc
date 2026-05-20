// Package catalogloader implements the MML console v2.3 catalog dictloader.
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §5 §6.7
//
// 设计原则：
//   - 生产运行时不依赖任何脚本语言：Go binary + JSON 数据文件 + DB
//   - 与现有 4 个 dictloader（param_models / products / indicators / alarm-definition）
//     设计对齐；MML catalog 是该模式的第 5 个 Loader
//   - 启动期 fail-fast：JSON 文件解析失败 → app 启动失败
//   - 软删除：差集中本次未覆盖的 standard 行标 deprecated_at，不动 admin / Customized
//   - sub_field 用户配置保护：通过 mml_command_sub_field_overrides 表（per-user）
package catalogloader

import "time"

// Catalog 是一份 JSON 数据文件的内存表示，对应 §5.2 schema。
type Catalog struct {
	SpecVersion     string  `json:"specVersion"`     // 多版本 namespace（如 "cmcc-tdlte-v2.3"）
	Carrier         string  `json:"carrier"`         // cmcc / ctcc / cucc
	Tech            string  `json:"tech"`            // lte / nr
	GeneratedAt     string  `json:"generatedAt"`     // ISO8601
	SourceDocSha256 string  `json:"sourceDocSha256"` // 用于 CI 校验 MD 与 JSON 同步
	Groups          []Group `json:"groups"`
}

// Group 是命令树的一级 object 分组，对应 mml_param_groups 一行。
type Group struct {
	GroupCode      string            `json:"groupCode"`      // 归一化 TR-181 路径模板（如 "Device.DeviceInfo.*"）
	NameI18n       map[string]string `json:"nameI18n"`       // {"zh-CN": "设备信息", "en-US": "Device Info"}
	ChapterCode    string            `json:"chapterCode"`    // SA…SR 元数据，仅用于排序，不渲染
	DisplayOrder   int               `json:"displayOrder"`   // SA…SR 自然顺序
	InstanceArity  int               `json:"instanceArity"`  // {i} 占位符层数
	InstanceLevels []string          `json:"instanceLevels"` // 多层 {i} 的层级语义名
	Commands       []Command         `json:"commands"`       // per-operation 行（1-4 个）
	SubFields      []SubField        `json:"subFields"`      // group 维度共享（按 op 过滤展示）
}

// Command 是 mml_commands 的一行（per-operation）。
// 同一 group 内 OperationType 唯一。
type Command struct {
	OperationType string   `json:"operationType"` // LST | MOD | ADD | RMV
	TargetPaths   []string `json:"targetPaths"`   // 该 op 的执行路径集
}

// SubField 是 group 维度的子字段定义，对应 mml_command_sub_fields 一行。
// 渲染时按 command.operationType + subField.accessType 过滤（MOD 隐藏 RO 等）。
type SubField struct {
	MMLCode         string            `json:"mmlCode"`
	StandardPath    string            `json:"standardPath"`
	LabelI18n       map[string]string `json:"labelI18n"`
	AccessType      string            `json:"accessType"` // RO / RW
	ValueType       string            `json:"valueType"`
	Constraint      string            `json:"constraint"`
	DefaultSelected bool              `json:"defaultSelected"`
	SortOrder       int               `json:"sortOrder"`
}

// LoadInfo 描述一份已加载 catalog 的元信息，供 admin API
// GET /api/v1/mml/catalog/info 透出。
type LoadInfo struct {
	SpecVersion     string    `json:"spec_version"`
	Carrier         string    `json:"carrier"`
	Tech            string    `json:"tech"`
	SourceDocSha256 string    `json:"source_doc_sha256"`
	GeneratedAt     string    `json:"generated_at"`
	LoadedAt        time.Time `json:"loaded_at"`
	GroupCount      int       `json:"group_count"`
	CommandCount    int       `json:"command_count"`
	SubFieldCount   int       `json:"sub_field_count"`
	DeprecatedCount int       `json:"deprecated_count"`
}
