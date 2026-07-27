package mmlstandardloader

// json_parser.go — 解析 cmcc_tdlte_v23.json seed 文件为内存结构。
//
// JSON 形态：
//
//	{
//	  "version": "2.3",
//	  "carrier": "cmcc",
//	  "network_type": "td-lte",
//	  "groups": [
//	    {
//	      "code": "SA",
//	      "name": "设备信息参数管理",
//	      "commands": [
//	        {
//	          "object_path": "Device.DeviceInfo.",
//	          "name": "设备基本信息",
//	          "non_creatable": false,
//	          "params": [
//	            {"path": "Device.DeviceInfo.UserLabel", "name": "用户友好名",
//	             "access": "RW", "type": "string", "constraints": {}}
//	          ]
//	        }
//	      ]
//	    }
//	  ],
//	  "non_creatable_objects": ["Device.Services.FAPService.{i}.", ...]
//	}
//
// 关键字段含义：
//   - version/carrier/network_type — 拼成 mml_param_versions.version_code（"cmcc-tdlte-v2.3"）
//   - groups[].code — 18 个一级分组 code（SA…SR），落 mml_command_groups.chapter_code/group_code
//   - groups[].commands[].object_path — TR-181 容器路径，包含 .{i}. 占位符；派生命令的锚点
//   - groups[].commands[].non_creatable — 命令级 ADD/RMV 抑制位
//   - groups[].commands[].params[].access — "R" / "RW"（=> standard_params.access "READ_ONLY"/"READ_WRITE"
//     与 mml_command_sub_fields.access_type "RO"/"RW"）
//   - non_creatable_objects[] — 全局 object_path 集合，包含此 path 的 command 不生成 ADD/RMV

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SeedRoot 是 cmcc_tdlte_v23.json 顶层结构。
type SeedRoot struct {
	Version             string      `json:"version"`
	Carrier             string      `json:"carrier"`
	NetworkType         string      `json:"network_type"`
	Groups              []SeedGroup `json:"groups"`
	NonCreatableObjects []string    `json:"non_creatable_objects"`
}

// SeedGroup 是 SA…SR 一级分组。
type SeedGroup struct {
	Code     string        `json:"code"`     // SA / SB / … / SR（18 个）
	Name     string        `json:"name"`     // 中文 chapter 标题（如 "设备信息参数管理"）
	Commands []SeedCommand `json:"commands"` // 该 chapter 下的所有 object_path 定义
}

// SeedCommand 是单个 TR-181 object 容器维度的命令源。
//
// 一个 SeedCommand 派生最多 4 条 mml_commands（LST/MOD/ADD/RMV），具体见 command_derivator.go。
type SeedCommand struct {
	ObjectPath   string      `json:"object_path"`   // 含 .{i}. 占位符的 TR-181 路径
	Name         string      `json:"name"`          // 中文命令名（如 "设备基本信息"）
	NonCreatable bool        `json:"non_creatable"` // true ⇒ 不生成 ADD/RMV（即使 path 含 {i}）
	Params       []SeedParam `json:"params"`
}

// SeedParam 是一条标准 path 元数据。
type SeedParam struct {
	Path        string          `json:"path"`        // 完整 TR-181 path（含 .{i}.）
	Name        string          `json:"name"`        // 中文显示名
	Access      string          `json:"access"`      // "R" 只读 / "RW" 可写
	Type        string          `json:"type"`        // string / unsignedInt / int / boolean / dateTime / unsignedLong
	Constraints SeedConstraints `json:"constraints"` // 类型约束
}

// SeedConstraints 数值/字符串约束。
type SeedConstraints struct {
	MaxLength *int   `json:"max_length,omitempty"` // string 类型最大长度
	Min       *int64 `json:"min,omitempty"`        // 数值类型下限
	Max       *int64 `json:"max,omitempty"`        // 数值类型上限
}

// IsWritable 是否可写（"RW"）。
func (p SeedParam) IsWritable() bool { return p.Access == AccessRW }

// IsReadOnly 是否只读（"R"）。
func (p SeedParam) IsReadOnly() bool { return p.Access == AccessR }

// VersionCode 拼出 mml_param_versions.version_code（如 "cmcc-tdlte-v2.3"）。
func (r *SeedRoot) VersionCode() string {
	return fmt.Sprintf("%s-%s-v%s", r.Carrier, r.NetworkType, r.Version)
}

// VersionName 拼出 mml_param_versions.version_name（人读用，如 "CMCC TD-LTE v2.3"）。
func (r *SeedRoot) VersionName() string {
	return fmt.Sprintf("%s %s v%s", upperASCII(r.Carrier), upperASCII(r.NetworkType), r.Version)
}

// NonCreatableSet 把 NonCreatableObjects 转 set，方便 O(1) 查询。
func (r *SeedRoot) NonCreatableSet() map[string]bool {
	m := make(map[string]bool, len(r.NonCreatableObjects))
	for _, op := range r.NonCreatableObjects {
		m[op] = true
	}
	return m
}

// AccessR / AccessRW — 与 JSON access 字段字面值一致。
const (
	AccessR  = "R"
	AccessRW = "RW"
)

// ParseSeedFile 从磁盘读取并解析 seed JSON。
//
// 失败语义：
//   - 文件不存在或不可读 → error
//   - JSON 解码失败 → error
//   - 未知字段不报错（DisallowUnknownFields=false）— seed 演进时向前兼容
func ParseSeedFile(path string) (*SeedRoot, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("open seed %s: %w", path, err)
	}
	defer f.Close()
	return ParseSeed(f)
}

// ParseSeed 从 io.Reader 解析；便于单元测试直接吃字符串。
func ParseSeed(r io.Reader) (*SeedRoot, error) {
	var root SeedRoot
	dec := json.NewDecoder(r)
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode seed json: %w", err)
	}
	if root.Version == "" || root.Carrier == "" || root.NetworkType == "" {
		return nil, fmt.Errorf("seed json missing required version/carrier/network_type")
	}
	if len(root.Groups) == 0 {
		return nil, fmt.Errorf("seed json contains no groups")
	}
	return &root, nil
}

// CountCommands 返回 seed 中的 SeedCommand 总数（即原始 object_path 条数）。
//
// 验证用：cmcc_tdlte_v23.json = 71。
func (r *SeedRoot) CountCommands() int {
	n := 0
	for _, g := range r.Groups {
		n += len(g.Commands)
	}
	return n
}

// CountParams 返回 seed 中所有 leaf path 总数。
//
// 验证用：cmcc_tdlte_v23.json = 616。
func (r *SeedRoot) CountParams() int {
	n := 0
	for _, g := range r.Groups {
		for _, c := range g.Commands {
			n += len(c.Params)
		}
	}
	return n
}

// upperASCII 简单 ASCII 大写化（仅用于 VersionName 显示）。
func upperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
