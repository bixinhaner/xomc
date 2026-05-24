package specparser

import (
	"encoding/json"
	"time"
)

// CatalogJSONFile 是 data/mml-catalog/cmcc-tdlte-v2.3.json 的顶层结构。
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
type CatalogJSONGroup struct {
	Chapter       string            `json:"chapter"`
	GroupCode     string            `json:"group_code"`
	CommandZhName string            `json:"command_zh_name"`
	HasInstance   bool              `json:"has_instance"`
	Paths         []CatalogJSONPath `json:"paths"`
	Commands      []CatalogJSONCmd  `json:"commands"` // 派生的 LST/MOD/ADD/RMV
}

type CatalogJSONPath struct {
	StandardPath string `json:"standard_path"`
	ParamName    string `json:"param_name"`
	ChineseName  string `json:"chinese_name"`
	Access       string `json:"access"`
	DataType     string `json:"data_type"`
	MinValue     *int64 `json:"min_value,omitempty"`
	MaxValue     *int64 `json:"max_value,omitempty"`
	MaxLength    *int   `json:"max_length,omitempty"`
}

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
