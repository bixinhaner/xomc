// Package specparser 解析 MML 南向数据模型规范 markdown 文档，与现有 DB diff
// 后生成 goose seed SQL 和 catalog JSON。
//
// 设计文档：docs/project/prd/F06-mml-catalog-spec-parser.md
//
// 由 omcctl mml import-spec-md 子命令驱动；仅 CMCC TD-LTE v2.3 实施，多运营商
// 扩展通过 --carrier flag 切 spec 路径。
package specparser

// SpecCatalog 是解析 spec markdown 的完整结果集。
//
// 数据流：spec.md → SpecCatalog → Differ → DiffReport → SQL / JSON。
type SpecCatalog struct {
	// Version 写入 mml_param_versions.version_code，如 "cmcc-td-lte-v2.3"。
	Version string

	// SourceMD 是 spec md 的绝对路径，写入生成的 SQL 文件头注释。
	SourceMD string

	// SourceHash 是 spec md 的 sha256（前 16 位），用于变更追溯。
	SourceHash string

	// Groups 按 §R-2.4 表顺序（SA→SR）排列；v2.3 共 71 条。
	Groups []*SpecGroup

	// Blacklist 是 §R-3.2「非可创建对象清单」中的 group_code 列表。
	// 即使 group_code 含 {i} 占位符，命中本清单也不派生 ADD/RMV。
	Blacklist []string
}

// SpecGroup 是 §R-2.4 表中一行 + §SA-SR 详情中对应章节的合并视图。
type SpecGroup struct {
	// Chapter 是 spec 章节标识，"SA" / "SB" / ... / "SR"，对应 mml_command_groups.group_code = "chapter:<Chapter>"。
	Chapter string

	// GroupCode 是 §R-2.4 表第三列 group_code（TR-181 归一化 path），
	// 如 "Device.DeviceInfo.*" / "Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*"。
	GroupCode string

	// CommandZhName 是 §R-2.4 表「命令中文名」列，全中文（含行业缩写如 LTE/MME/PHY），
	// 71 条互不相同（spec §R-2.4 唯一性禁令）。
	CommandZhName string

	// CommandEnName 是命令对象英文名。规范源暂未稳定提供时可为空；SQL 生成器会从
	// LogicalCode 派生可读英文兜底，禁止继续把 CommandZhName 写入 en-US。
	CommandEnName string

	// HasInstance true 表示 GroupCode 含 {i} 占位符 → 候选 ADD/RMV 派生
	// （是否真生成 ADD/RMV 还需叠加 Blacklist 判断）。
	HasInstance bool

	// Paths 是从 §SA-SR 详情段下 "#### 命令: <path>" H4 表格抓取的 standardPath 集合。
	// 每条含 access / data_type / range 元数据，供 standard_params INSERT 用。
	Paths []*SpecPath
}

// SpecPath 是单条 standardPath 的完整元属性（来自 spec H4 表的一行）。
type SpecPath struct {
	// StandardPath 是 TR-181 形式 path 字符串（spec H4 表第 3 列「TR-181 路径」）。
	StandardPath string

	// ParamName 是参数英文名（spec H4 表第 4 列）。
	ParamName string

	// ChineseName 是参数中文名（spec H4 表第 5 列），写入 mml_command_sub_fields.label_i18n.zh-CN。
	ChineseName string

	// Access 是权限：READ_ONLY / READ_WRITE / WRITE_ONLY。
	// 从 spec H4 表第 6 列 "📖 R" → READ_ONLY / "📝 RW" → READ_WRITE 映射。
	Access string

	// DataType 是类型简写：string / unsignedInt / int / boolean / dateTime / hexBinary 等。
	// 从 spec H4 表第 7 列剥离 [min:max] 与 (length) 后得到。
	DataType string

	// MinValue / MaxValue 仅对数值类型有效，从 "unsignedInt[1:5]" 这种类型字符串提取。
	MinValue *int64
	MaxValue *int64

	// MaxLength 仅对 string 类型有效，从 "string(64)" 提取 64。
	MaxLength *int
}

// =============================================================================
// DBSnapshot 与 DiffReport — Differ 输入输出
// =============================================================================

// DBSnapshot 是 catalog 相关表的现状快照。Differ 仅读，不写。
type DBSnapshot struct {
	// StandardParams 索引 standard_path → row。
	StandardParams map[string]*StandardParamRow

	// Commands 索引 command_code → row（source='standard' 行优先；admin 行不影响 diff）。
	Commands map[string]*MMLCommandRow

	// CommandSubFields 索引 command_code → standard_path → row。
	CommandSubFields map[string]map[string]*SubFieldRow
}

// StandardParamRow 复刻 standard_params 表关键列（不含 created_at/updated_at）。
type StandardParamRow struct {
	StandardPath  string
	EntryType     string // "parameter" / "object"
	Access        string
	DataType      string
	ChangeApplies string
	MinValue      *int64
	MaxValue      *int64
}

// MMLCommandRow 复刻 mml_commands 关键列。
type MMLCommandRow struct {
	CommandCode   string
	OperationType string // LST / MOD / ADD / RMV
	LogicalCode   string
	TargetPaths   []string // 反序列化自 target_paths JSONB
	Source        string   // "standard" / "admin"
}

// SubFieldRow 复刻 mml_command_sub_fields 关键列。
type SubFieldRow struct {
	CommandCode  string
	StandardPath string
	MmlCode      string
	SortOrder    int
}

// DiffReport 是 Spec vs DB 三方比对结果，由 SQL/JSON 生成器消费。
type DiffReport struct {
	// NewStandardParams 是 spec 有但 DB 无的 standardPath（需 INSERT standard_params）。
	NewStandardParams []*SpecPath

	// NewCommands 是 spec 派生但 DB 无的命令叶子（需 INSERT mml_commands）。
	NewCommands []*SpecCommand

	// UpdatedCommands 是 DB 已有但 target_paths 与 spec 派生不一致（需 UPDATE mml_commands）。
	UpdatedCommands []*SpecCommand

	// NewSubFieldLinks 是 spec 要求但 DB 中 (command, standard_path) 无关联（需 INSERT mml_command_sub_fields）。
	NewSubFieldLinks []*SubFieldLink

	// OrphanCommands 是 DB source='standard' 但 spec 未覆盖（仅标记，不删除）。
	OrphanCommands []string

	// OrphanLinks 是 DB 已有但 spec 未派生的 sub_field 关联 command_code 清单（仅标记）。
	OrphanLinks []string

	// CrossCheckWarnings 是 §R-2.4 71 条权威表与 §SA-SR H4 标题集不一致的告警（非阻塞）。
	CrossCheckWarnings []string

	// Summary 是供 stderr 输出的统计摘要。
	Summary DiffSummary
}

// DiffSummary 统计 DiffReport 各类条目数量。
type DiffSummary struct {
	NewStandardParams int
	NewCommands       int
	UpdatedCommands   int
	NewSubFieldLinks  int
	OrphanCommands    int
	OrphanLinks       int
	CrossWarnings     int
}

// SpecCommand 是 SpecGroup 按 §R-3 派生出的命令叶子（差异 SQL 的主体）。
type SpecCommand struct {
	GroupCode     string   // 来源 SpecGroup.GroupCode
	Chapter       string   // 来源 SpecGroup.Chapter ("SA"…"SR")
	OperationType string   // LST / MOD / ADD / RMV
	CommandCode   string   // "<OP> <logical_code>"，如 "LST X2_IP_ADDR_MAP"
	LogicalCode   string   // 从 group_code 派生，去前缀+去{i}+大写下划线
	CommandZhName string   // 来源 SpecGroup.CommandZhName
	CommandEnName string   // 来源 SpecGroup.CommandEnName；为空时由 LogicalCode 派生英文兜底
	TargetPaths   []string // LST=全 path; MOD=仅 RW path; ADD/RMV=[object_name]
	RPCMethod     string   // GetParameterValues / SetParameterValues / AddObject / DeleteObject
	TargetObject  string   // ADD/RMV 用：group_code 去 "{i}.*" 后的父对象路径
}

// SubFieldLink 描述一条 (command, standard_path) 关联。
type SubFieldLink struct {
	CommandCode  string
	StandardPath string
	MmlCode      string // 老系统 MML 字符串内部 code（命令上下文相关，从 ParamName 大写下划线派生）
	SortOrder    int
	ChineseName  string // 来源 SpecPath.ChineseName
	ParamName    string // 来源 SpecPath.ParamName
}

// =============================================================================
// 派生常量（与 DB schema 字符串对齐，避免拼写漂移）
// =============================================================================

const (
	AccessReadOnly  = "READ_ONLY"
	AccessReadWrite = "READ_WRITE"
	AccessWriteOnly = "WRITE_ONLY"

	OpLST = "LST"
	OpMOD = "MOD"
	OpADD = "ADD"
	OpRMV = "RMV"

	RPCGetParameterValues = "GetParameterValues"
	RPCSetParameterValues = "SetParameterValues"
	RPCAddObject          = "AddObject"
	RPCDeleteObject       = "DeleteObject"

	EntryTypeParameter = "parameter"
	EntryTypeObject    = "object"

	ChangeAppliesImmediate = "Immediate"
	ChangeAppliesOnReboot  = "OnReboot"

	SourceStandard = "standard"
	SourceAdmin    = "admin"
)
