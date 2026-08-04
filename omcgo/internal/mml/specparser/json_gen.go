package specparser

import (
	"encoding/json"
	"time"
)

// CatalogJSONFile 是 data/mml-catalog/cmcc-tdlte-v2.3.json 的顶层结构。
//
// ⚠️ 该文件**仅用于提取分组结构**(chapter / group_code / command_zh_name /
// has_instance / command_code / operation_type),**不是 path 的真值源**。
// 命令真正引用哪些 path 由 `mml_command_sub_fields` 表 → `standard_params`
// 决定。（catalogloader 已下线删除;mml 命令为 seed-once + DB 单一权威源。）
//
// JSON 中 Groups[].Paths[] 与 Commands[].TargetPaths[] 字段是 catalog 生成
// 阶段写出的冗余元数据(便于审阅),运行时 GPV 不读这两个字段。
//
// 用途：同步给前端 admin / catalog 展示，保留 generated_from + spec_md_hash 元数据
// 让 catalog 来源可追溯。
type CatalogJSONFile struct {
	GeneratedFrom string             `json:"generated_from"`
	SpecMDHash    string             `json:"spec_md_hash"`
	GeneratedAt   string             `json:"generated_at"`
	Version       string             `json:"version"`
	Groups        []CatalogJSONGroup `json:"groups"`
	Blacklist     []string           `json:"blacklist,omitempty"`
}

// CatalogJSONGroup 一个 group 的完整描述（含派生命令）。
//
// Paths 字段:catalog 生成阶段从规范 MD 抽取的 path 元数据,运行时 **不**
// 当作 path 真值源消费 —— 真值源是 `standard_params` 表。该字段仅供 catalog
// 文件审阅与离线工具(specparser)使用。
type CatalogJSONGroup struct {
	Chapter       string            `json:"chapter"`
	GroupCode     string            `json:"group_code"`
	CommandZhName string            `json:"command_zh_name"`
	CommandEnName string            `json:"command_en_name,omitempty"`
	HasInstance   bool              `json:"has_instance"`
	Paths         []CatalogJSONPath `json:"paths"`
	Commands      []CatalogJSONCmd  `json:"commands"` // 派生的 LST/MOD/ADD/RMV
}

type CatalogJSONPath struct {
	StandardPath string `json:"standard_path"`
	// ParamName 是参数英文名（spec H4 表第 4 列），写入 label_i18n['en-US']。
	// issue #67 §2（长期 TODO）：部分规范行此列实际仍是中文，导致 en=中文。规范 MD 若提供
	// 明确英文列（如 english_name），应在 CatalogJSONPath 增 `EnglishName string json:"english_name"`，
	// parser 优先用它填 label_i18n['en-US']，缺失再退回 standard_path 叶子（与 seed/000038 一致）。
	ParamName   string `json:"param_name"`
	ChineseName string `json:"chinese_name"`
	Access      string `json:"access"`
	DataType    string `json:"data_type"`
	MinValue    *int64 `json:"min_value,omitempty"`
	MaxValue    *int64 `json:"max_value,omitempty"`
	MaxLength   *int   `json:"max_length,omitempty"`
}

// CatalogJSONCmd 一条派生命令的元数据。
//
// TargetPaths 字段:**只是 catalog 生成时的冗余 path 列表**(从同 group 的
// Paths 抽取),历史上写入 `mml_commands.target_paths` jsonb 列(现由 sub_fields 触发器维护),
// 但 MML executor 拼 GPV 时不读这两份冗余,而是直接走 `mml_command_sub_fields`
// JOIN `standard_params`。因此修改 catalog 的 target_paths 不会影响实际 GPV 行为;
// 增删命令的字段集请改 sub_field 表(或写 seed migration)。
type CatalogJSONCmd struct {
	CommandCode   string   `json:"command_code"`
	LogicalCode   string   `json:"logical_code"`
	OperationType string   `json:"operation_type"`
	RPCMethod     string   `json:"rpc_method"`
	TargetPaths   []string `json:"target_paths"`
	TargetObject  string   `json:"target_object,omitempty"`
}

// GenerateJSON 把 SpecCatalog 渲染为 catalog JSON 字节（带缩进 + UTF-8 不转义）。
func GenerateJSON(cat *SpecCatalog) ([]byte, error) {
	commands := DeriveCommands(cat)
	cmdsByGroup := make(map[string][]CatalogJSONCmd, len(cat.Groups))
	for _, c := range commands {
		cmdsByGroup[c.GroupCode] = append(cmdsByGroup[c.GroupCode], CatalogJSONCmd{
			CommandCode:   c.CommandCode,
			LogicalCode:   c.LogicalCode,
			OperationType: c.OperationType,
			RPCMethod:     c.RPCMethod,
			TargetPaths:   c.TargetPaths,
			TargetObject:  c.TargetObject,
		})
	}

	file := &CatalogJSONFile{
		GeneratedFrom: cat.SourceMD,
		SpecMDHash:    cat.SourceHash,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Version:       cat.Version,
	}
	for _, g := range cat.Groups {
		jg := CatalogJSONGroup{
			Chapter:       g.Chapter,
			GroupCode:     g.GroupCode,
			CommandZhName: g.CommandZhName,
			CommandEnName: g.CommandEnName,
			HasInstance:   g.HasInstance,
			Commands:      cmdsByGroup[g.GroupCode],
		}
		for _, p := range g.Paths {
			jg.Paths = append(jg.Paths, CatalogJSONPath{
				StandardPath: p.StandardPath,
				ParamName:    p.ParamName,
				ChineseName:  p.ChineseName,
				Access:       p.Access,
				DataType:     p.DataType,
				MinValue:     p.MinValue,
				MaxValue:     p.MaxValue,
				MaxLength:    p.MaxLength,
			})
		}
		file.Groups = append(file.Groups, jg)
	}
	if len(cat.Blacklist) > 0 {
		file.Blacklist = cat.Blacklist
	}
	return json.MarshalIndent(file, "", "  ")
}
