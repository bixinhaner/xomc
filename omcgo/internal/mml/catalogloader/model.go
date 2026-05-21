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
// v2 schema 支持（2026-05-21 增补，对应 spec §R-1/§R-2/§R-2.4/§R-2.5/§R-3.1/§R-3.2/§R-4.1.1）：
//   - JSON 根 `schemaVersion: "v2"` 标识，Parse 自动分派 v1 / v2 校验路径
//   - 一级分组为 18 个 SA-SR 章节（groupCode = "chapter:SA"），不再是 71 个 object 组
//   - command 显示名 = §R-2.4 全中文命名权威表（71 条，跨章节唯一）
//   - command.treeNodeRefs 引用 standard_params.standardPath，替代旧 targetPaths
//   - command.instanceRangeMeta 携带每层 {i} 范围 metadata（§R-4.1.1）
//   - Catalog.params 内嵌 standardPath 元属性表（§R-4.2.1 类型 / 范围 / 提示文字）
//   - Catalog.mergeLog 记录同 groupCode 跨章节合并事件（§R-3.1）
//   - Catalog.nonCreatableHits 列出被 §R-3.2 过滤掉 ADD/RMV 的对象
package catalogloader

import "time"

// Catalog 是一份 JSON 数据文件的内存表示，对应 §5.2 schema。
//
// v1 schema：groups[].commands[] 各自带 targetPaths + subFields；
// v2 schema：schemaVersion="v2"，groups[] 为 18 chapter，commands[] 用 treeNodeRefs，
// Catalog.params[] 集中放参数元数据。两套形态共用同一个 struct，按 `,omitempty`
// 兼容；解析期由 IsV2() 判定走哪条校验路径。
type Catalog struct {
	SpecVersion     string  `json:"specVersion"`     // 多版本 namespace（如 "cmcc-tdlte-v2.3"）
	Carrier         string  `json:"carrier"`         // cmcc / ctcc / cucc
	Tech            string  `json:"tech"`            // lte / nr
	GeneratedAt     string  `json:"generatedAt"`     // ISO8601
	SourceDocSha256 string  `json:"sourceDocSha256"` // 用于 CI 校验 MD 与 JSON 同步
	Groups          []Group `json:"groups"`

	// ── v2 schema 字段（schemaVersion="v2" 时填充；v1 一律空 / nil）────
	SchemaVersion    string            `json:"schemaVersion,omitempty"`
	Stats            *CatalogStats     `json:"stats,omitempty"`
	Params           []ParamRow        `json:"params,omitempty"`
	MergeLog         []MergeLogEntry   `json:"mergeLog,omitempty"`
	NonCreatableHits []NonCreatableHit `json:"nonCreatableHits,omitempty"`
}

// IsV2 检测 catalog 文件 schema 版本：v2 显式带 schemaVersion="v2" 字段。
// v1 文件无此字段 → 返回 false → 走 v1 校验路径（Group 必有 subFields，Command 必有 targetPaths）。
func (c *Catalog) IsV2() bool {
	return c.SchemaVersion == "v2"
}

// Group 是命令树的一级分组，对应 mml_param_groups 一行。
//
// v1 语义：object 维度（GroupCode = TR-181 路径模板，如 "Device.DeviceInfo.*"）；
// v2 语义：chapter 维度（GroupCode = "chapter:SA" / "chapter:SF" ...）。
type Group struct {
	GroupCode      string            `json:"groupCode"`      // v1: 归一化 TR-181 路径模板；v2: "chapter:<SA-SR>"
	NameI18n       map[string]string `json:"nameI18n"`       // v2: {"zh-CN": "设备信息参数管理"}；v1: {"zh-CN": "设备信息"}
	ChapterCode    string            `json:"chapterCode"`    // SA…SR 元数据
	DisplayOrder   int               `json:"displayOrder"`   // SA=1…SR=18 自然顺序
	InstanceArity  int               `json:"instanceArity"`  // v1: {i} 占位符层数；v2: 0（chapter 无实例）
	InstanceLevels []string          `json:"instanceLevels"` // v1: 多层 {i} 层级语义名；v2: 空
	Commands       []Command         `json:"commands"`       // per-operation 行
	SubFields      []SubField        `json:"subFields,omitempty"` // v1: group 维度共享；v2: 移到 Catalog.params 集中管理
}

// Command 是 mml_commands 的一行（per-operation）。
// 同一 group 内 OperationType 唯一。
//
// v1 字段：OperationType + TargetPaths；
// v2 字段：补 CommandCode / RPCMethod / LogicalNameI18n / GroupCodeObject /
//          InstanceArity / InstanceLevels / InstanceRangeMeta / TreeNodeRefs。
// 共存策略：v1 JSON 解析后 v2 字段为零值；v2 JSON 中 TargetPaths 空（用 TreeNodeRefs 替代）。
type Command struct {
	OperationType string   `json:"operationType"`           // LST | MOD | ADD | RMV
	TargetPaths   []string `json:"targetPaths,omitempty"`   // v1: 该 op 的执行路径集；v2: 留空，用 TreeNodeRefs

	// ── v2 schema 字段 ────────────────────────────────────────
	CommandCode       string            `json:"commandCode,omitempty"`       // 稳定标识 "<OP>:<groupCodeObject>"
	RPCMethod         string            `json:"rpcMethod,omitempty"`         // GetParameterValues / SetParameterValues / AddObject / DeleteObject
	LogicalNameI18n   map[string]string `json:"logicalNameI18n,omitempty"`   // §R-2.4 命令中文名 {"zh-CN": "设备基本信息"}
	GroupCodeObject   string            `json:"groupCodeObject,omitempty"`   // 归一化 TR-181 路径模板（旧 v1 group_code 语义）
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

// SubField 是 group 维度的子字段定义，对应 mml_command_sub_fields 一行。
// 渲染时按 command.operationType + subField.accessType 过滤（MOD 隐藏 RO 等）。
//
// v2 schema 不再用本结构：参数元数据集中放 Catalog.Params，Command 通过
// TreeNodeRefs 引用 standard_params。本结构保留是为 v1 兼容期 + admin Customized
// 命令仍走原 SubField 链路。
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

// ParamRow 是 v2 Catalog.params 数组中一行，对应 standard_params 表一行。
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
	GroupCount          int `json:"group_count"`
	CommandCount        int `json:"command_count"`
	SubFieldCount       int `json:"sub_field_count"`
	DeprecatedCount     int `json:"deprecated_count"`
	SkippedCommandCount int `json:"skipped_command_count"` // v2.4 D34 — MOD/ADD/RMV target_paths 为空被跳过的数量
}
