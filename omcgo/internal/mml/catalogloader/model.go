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
//
// 仅 v2 schema（spec §R-1/§R-2/§R-2.4/§R-2.5/§R-3.1/§R-3.2/§R-4.1.1）：
//   - 一级分组为 18 个 SA-SR 章节（groupCode = "chapter:SA"）
//   - command 显示名 = §R-2.4 全中文命名权威表（71 条，跨章节唯一）
//   - command.treeNodeRefs 引用 standard_params.standardPath
//   - command.instanceRangeMeta 携带每层 {i} 范围 metadata（§R-4.1.1）
//   - Catalog.params 内嵌 standardPath 元属性表（§R-4.2.1 类型 / 范围 / 提示文字）
//   - Catalog.mergeLog 记录同 groupCode 跨章节合并事件（§R-3.1）
//   - Catalog.nonCreatableHits 列出被 §R-3.2 过滤掉 ADD/RMV 的对象
package catalogloader

import "time"

// Catalog 是一份 JSON 数据文件的内存表示，对应 §5.2 schema。
type Catalog struct {
	SpecVersion     string  `json:"specVersion"`     // 多版本 namespace（如 "cmcc-tdlte-v2.3"）
	Carrier         string  `json:"carrier"`         // cmcc / ctcc / cucc
	Tech            string  `json:"tech"`            // lte / nr
	GeneratedAt     string  `json:"generatedAt"`     // ISO8601
	SourceDocSha256 string  `json:"sourceDocSha256"` // 用于 CI 校验 MD 与 JSON 同步
	SchemaVersion   string  `json:"schemaVersion,omitempty"` // 信息字段，固定 "v2"
	Groups          []Group `json:"groups"`

	Stats            *CatalogStats     `json:"stats,omitempty"`
	Params           []ParamRow        `json:"params,omitempty"`
	MergeLog         []MergeLogEntry   `json:"mergeLog,omitempty"`
	NonCreatableHits []NonCreatableHit `json:"nonCreatableHits,omitempty"`
}

// Group 是命令树的一级分组（chapter 维度），对应 mml_command_groups 一行。
// GroupCode = "chapter:SA" / "chapter:SF" ...（spec §R-1 18 SA-SR 章节）。
type Group struct {
	GroupCode    string            `json:"groupCode"`    // "chapter:<SA-SR>"
	NameI18n     map[string]string `json:"nameI18n"`     // {"zh-CN": "设备信息参数管理"}
	ChapterCode  string            `json:"chapterCode"`  // SA…SR 元数据
	DisplayOrder int               `json:"displayOrder"` // SA=1…SR=18 自然顺序
	Commands     []Command         `json:"commands"`
}

// Command 是 mml_commands 的一行（per-operation）。
// 同一 group 内 (GroupCodeObject, OperationType) 唯一。
type Command struct {
	CommandCode       string            `json:"commandCode"`                 // 稳定标识 "<OP>:<groupCodeObject>"
	OperationType     string            `json:"operationType"`               // LST | MOD | ADD | RMV
	RPCMethod         string            `json:"rpcMethod,omitempty"`         // GetParameterValues / SetParameterValues / AddObject / DeleteObject
	LogicalNameI18n   map[string]string `json:"logicalNameI18n,omitempty"`   // §R-2.4 命令中文名 {"zh-CN": "设备基本信息"}
	GroupCodeObject   string            `json:"groupCodeObject,omitempty"`   // 归一化 TR-181 路径模板
	InstanceArity     int               `json:"instanceArity,omitempty"`     // {i} 占位符层数
	InstanceLevels    []string          `json:"instanceLevels,omitempty"`    // 每层对象段名（"MU"/"Slot"/...）
	InstanceRangeMeta []InstanceRange   `json:"instanceRangeMeta,omitempty"` // §R-4.1.1 每层取值范围
	TreeNodeRefs      []string          `json:"treeNodeRefs,omitempty"`      // §R-2.5 引用 standard_params.standardPath
}

// InstanceRange 是 §R-4.1.1 一条 {i} 占位符的取值范围元数据。
// 多层 {i} 时按 layer 顺序排列在 InstanceRangeMeta slice 内。
type InstanceRange struct {
	Layer       int    `json:"layer"`                 // 1-based 层级，外层 → 内层
	RangeExpr   string `json:"rangeExpr"`             // 原始 expr，如 "0~N" / "1~3" / "1~24"
	RangeMin    *int   `json:"rangeMin"`              // nil 表示未指定
	RangeMax    *int   `json:"rangeMax"`              // nil 表示动态（运行时由设备 NumberOfEntries 决定）
	Dynamic     bool   `json:"dynamic"`               // 上限是否动态依赖另一参数
	NSource     string `json:"nSource,omitempty"`     // 动态上限来源参数名（如 MaxCurrentAlarmEntries）
	Description string `json:"description,omitempty"` // 原始注释行的描述文本
}

// ParamRow 是 Catalog.params 数组中一行，对应 standard_params 表一行。
// catalog Loader 启动期可用本数据集和 standard_params 做一致性校验（spec 列出但
// standard_params 缺失 → §R-2.5.2 link health 失败清单）。
type ParamRow struct {
	StandardPath   string            `json:"standardPath"`
	ParamName      string            `json:"paramName"`
	LabelI18n      map[string]string `json:"labelI18n"`
	AccessType     string            `json:"accessType"`     // RO / RW
	ValueType      string            `json:"valueType"`      // string / unsignedInt / boolean / dateTime / hexBinary / enum
	Constraint     string            `json:"constraint"`     // 原始"类型"列字符串
	ConstraintHint string            `json:"constraintHint"` // §R-4.2.1 派生的中文文字提示
	LengthMax      *int              `json:"lengthMax"`
	RangeMin       *int              `json:"rangeMin"`
	RangeMax       *int              `json:"rangeMax"`
}

// MergeLogEntry 是 §R-3.1 同 groupCode 跨章节合并的一条审计记录。
// catalog Loader 可把本数组按需写入运行时日志或 admin API 透出。
type MergeLogEntry struct {
	GroupCode      string `json:"groupCode"`
	OwnerChapter   string `json:"ownerChapter"`
	MergedChapter  string `json:"mergedChapter"`
	AddedPathCount int    `json:"addedPathCount"`
}

// NonCreatableHit 是 §R-3.2 非可创建对象清单的一条命中记录。
type NonCreatableHit struct {
	GroupCodeObject string   `json:"groupCodeObject"`
	FilteredOps     []string `json:"filteredOps"` // 通常是 ["ADD", "RMV"]
}

// CatalogStats 是 v2 catalog 的派生统计信息（offline parser 写入，runtime 仅读）。
type CatalogStats struct {
	Chapters             int `json:"chapters,omitempty"`
	ObjectsTotal         int `json:"objectsTotal,omitempty"`
	CommandLeavesTotal   int `json:"commandLeavesTotal,omitempty"`
	UniqueParams         int `json:"uniqueParams,omitempty"`
	MergeEvents          int `json:"mergeEvents,omitempty"`
	NonCreatableFiltered int `json:"nonCreatableFiltered,omitempty"`
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

	// §R-2.5.2 本次 Loader 检测到的标准参数树未命中数
	LinkHealthFailures int `json:"link_health_failures,omitempty"`
	// §R-2.5.2 本次 Loader 把之前失败项标为 resolved 的数量
	LinkHealthResolved int `json:"link_health_resolved,omitempty"`
}
