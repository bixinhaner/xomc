package mmlstandardloader

import (
	"strings"
)

// RPC method 字符串（与 TR-069 spec / internal/acs/rpc/dispatcher.go register 名一致）。
const (
	RPCGetParameterValues = "GetParameterValues"
	RPCSetParameterValues = "SetParameterValues"
	RPCAddObject          = "AddObject"
	RPCDeleteObject       = "DeleteObject"
)

// 操作类型与 seed/000004_mml_enhance.sql 的 mml_commands.operation_type CHECK 约束一致。
const (
	OpLST = "LST"
	OpMOD = "MOD"
	OpADD = "ADD"
	OpRMV = "RMV"
)

// CommandSpec 一条 mml_commands 行的生成结果。
type CommandSpec struct {
	Code          string   // 'LST_DEVICE_DEVICEINFO_ANTENNAINFO'
	Name          string   // 命令中文名（FE name_i18n.zh）
	NameEn        string   // 英文名（FE name_i18n.en）
	Category      string   // '1'..'7'
	OperationType string   // LST / MOD / ADD / RMV
	RPCMethod     string   // GetParameterValues 等
	GroupPath     string   // 关联 group path（运行时 lookup mml_param_groups.path → id）
	TargetPaths   []string // LST/MOD 时填；ADD/RMV 时空
	TargetObject  string   // ADD/RMV 时填；LST/MOD 时空
}

// GenerateCommands 把一批 group 翻译为 LST/MOD/ADD/RMV 4 类命令。
// 规则见 docs/design/mml-rebuild-plan-20260513.md §6.3。
//
// 一个 group 最多生成 4 个命令：
//   - LST 总生成
//   - MOD 仅当 group 含 ≥1 个 READ_WRITE param
//   - ADD/RMV 仅当 group path 含 `{i}`（动态实例对象）
func GenerateCommands(groups []GroupSpec) []CommandSpec {
	var cmds []CommandSpec
	for _, g := range groups {
		cmds = append(cmds, genForGroup(g)...)
	}
	return cmds
}

func genForGroup(g GroupSpec) []CommandSpec {
	if len(g.Params) == 0 {
		return nil
	}

	// 收集所有 path 与 writable path
	allPaths := make([]string, 0, len(g.Params))
	rwPaths := make([]string, 0, len(g.Params))
	hasWritable := false
	for _, p := range g.Params {
		allPaths = append(allPaths, p.StandardPath)
		if p.IsWritable() {
			rwPaths = append(rwPaths, p.StandardPath)
			hasWritable = true
		}
	}

	category := CategorizePath(g.Path)
	nameZh, nameEn := groupDisplayName(g)

	var out []CommandSpec

	// LST 总生成
	out = append(out, CommandSpec{
		Code:          OpLST + "_" + g.Code,
		Name:          "列出 " + nameZh,
		NameEn:        "List " + nameEn,
		Category:      category,
		OperationType: OpLST,
		RPCMethod:     RPCGetParameterValues,
		GroupPath:     g.Path,
		TargetPaths:   allPaths,
	})

	// MOD 仅当有 writable
	if hasWritable {
		out = append(out, CommandSpec{
			Code:          OpMOD + "_" + g.Code,
			Name:          "修改 " + nameZh,
			NameEn:        "Modify " + nameEn,
			Category:      category,
			OperationType: OpMOD,
			RPCMethod:     RPCSetParameterValues,
			GroupPath:     g.Path,
			TargetPaths:   rwPaths,
		})
	}

	// ADD/RMV 仅当 group path 含 `{i}`
	if g.HasInstance {
		// target_object = group path 去 `{i}` 后加末尾点（TR-069 object 形态）
		targetObject := StripInstanceIndex(g.Path)
		if !strings.HasSuffix(targetObject, ".") {
			targetObject += "."
		}
		out = append(out,
			CommandSpec{
				Code:          OpADD + "_" + g.Code,
				Name:          "新增 " + nameZh,
				NameEn:        "Add " + nameEn,
				Category:      category,
				OperationType: OpADD,
				RPCMethod:     RPCAddObject,
				GroupPath:     g.Path,
				TargetObject:  targetObject,
			},
			CommandSpec{
				Code:          OpRMV + "_" + g.Code,
				Name:          "删除 " + nameZh,
				NameEn:        "Remove " + nameEn,
				Category:      category,
				OperationType: OpRMV,
				RPCMethod:     RPCDeleteObject,
				GroupPath:     g.Path,
				TargetObject:  targetObject,
			},
		)
	}

	return out
}

// groupDisplayName 从 group path 派生一个简短的展示名。
//
// 简单策略：取末段（去 {i}）作为基础名；后续可加翻译字典覆盖。
//
//	"Device.DeviceInfo.AntennaInfo"          → "AntennaInfo" / "AntennaInfo"
//	"Device.Services.FAPService.{i}.Transport" → "Transport" / "Transport"
//	"DeviceGSM.Bts"                          → "Bts" / "Bts"
//
// FE 实际展示用 sys_dictionaries / mml_commands.command_name_i18n 覆盖；
// 这里 fallback 在没有翻译字典时让 UI 至少不空白。
func groupDisplayName(g GroupSpec) (zh, en string) {
	// 取末段
	parts := strings.Split(StripInstanceIndex(g.Path), ".")
	last := ""
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			last = parts[i]
			break
		}
	}
	return last, last
}
