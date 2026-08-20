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
	GroupPath     string   // 关联 group path（运行时 lookup mml_command_groups.path → id）
	TargetPaths   []string // LST 为全部参数；MOD/ADD 为可写参数；RMV 为空
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

	// 收集所有 path、writable path，以及 ADD 可在新实例上直接设置的 writable path。
	// ADD 的复合执行只创建当前对象的一层实例；子对象中的另一个 {i} 必须由其
	// 自己的 ADD 命令处理，不能混入父对象的 SetParameterValues。
	allPaths := make([]string, 0, len(g.Params))
	rwPaths := make([]string, 0, len(g.Params))
	addRWPaths := make([]string, 0, len(g.Params))
	addInstanceDepth := strings.Count(g.Path, "{i}") + 1
	hasWritable := false
	for _, p := range g.Params {
		allPaths = append(allPaths, p.StandardPath)
		if p.IsWritable() {
			rwPaths = append(rwPaths, p.StandardPath)
			if strings.Count(p.StandardPath, "{i}") == addInstanceDepth {
				addRWPaths = append(addRWPaths, p.StandardPath)
			}
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
				TargetPaths:   addRWPaths,
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

// groupDisplayName 从 group path 派生短展示名（F-A：解长命令名问题）。
//
// 旧实现只取末段（"AntennaInfo"），信息密度低 — 上下文丢失导致命令列表里
// 一堆同名 "Transport" / "RF" / "X_PARAM"。新实现按 path 前缀缩写 + 中段
// 选取关键词，让 UI 列表既不超长也保留导航上下文。
//
// 缩写规则（按优先级）：
//
//	Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PhyCellID
//	  → zh "小区.LTE.RAN.PhyCellID"     / en "Cell.LTE.RAN.PhyCellID"
//	Device.Services.FAPService.{i}.CellConfig.{i}.NR.Foo.Bar
//	  → zh "小区.NR.Foo.Bar"           / en "Cell.NR.Foo.Bar"
//	Device.DeviceInfo.AntennaInfo
//	  → zh "设备.AntennaInfo"          / en "Device.AntennaInfo"
//	Device.FAP.GPS
//	  → zh "FAP.GPS"                  / en "FAP.GPS"
//	Device.FaultMgmt.CurrentAlarm
//	  → zh "告警.CurrentAlarm"         / en "Fault.CurrentAlarm"
//	DeviceGSM.Bts
//	  → zh "GSM.Bts"                  / en "GSM.Bts"
//
// command_code 字段（DB UNIQUE）维持完整 group_code 不变，保审计追溯。
func groupDisplayName(g GroupSpec) (zh, en string) {
	clean := StripInstanceIndex(g.Path)

	// 1. CellConfig / FAPService 是 standard-model 里参数密度最高的子树 —
	//    多数 528+210 命令都在这两支下，最值得专门缩写。
	if rest, ok := trimPrefix(clean, "Device.Services.FAPService.CellConfig."); ok {
		// "LTE.RAN.PhyCellID" — 把 LTE/NR 当二级根，drop 中间的实例占位
		return "小区." + rest, "Cell." + rest
	}
	if rest, ok := trimPrefix(clean, "Device.Services.FAPService."); ok {
		return "FAP." + rest, "FAP." + rest
	}

	// 2. DeviceInfo / Time / IP / Ethernet / FAP / FaultMgmt / ManagementServer
	//    这些是顶级业务实体，给一个一字中文别名。
	type prefixAlias struct {
		prefix string
		zh     string
		en     string
	}
	aliases := []prefixAlias{
		{"Device.DeviceInfo.", "设备", "Device"},
		{"Device.Time.", "时间", "Time"},
		{"Device.IP.", "网络", "IP"},
		{"Device.Ethernet.", "以太网", "Ethernet"},
		{"Device.FAP.", "FAP", "FAP"},
		{"Device.FaultMgmt.", "告警", "Fault"},
		{"Device.ManagementServer.", "TR069", "TR069"},
		{"Device.KeepalivedMgmt.", "热备", "HA"},
		{"Device.RemoteDeviceList.", "远程设备", "RemoteDev"},
		{"Device.IPsec.", "IPsec", "IPsec"},
		{"DeviceGSM.", "GSM", "GSM"},
	}
	for _, a := range aliases {
		if rest, ok := trimPrefix(clean, a.prefix); ok {
			return a.zh + "." + rest, a.en + "." + rest
		}
	}

	// 3. 兜底：去 "Device." 前缀（顶层 Device. 信息冗余）；再不行原样
	if rest, ok := trimPrefix(clean, "Device."); ok {
		return rest, rest
	}
	return clean, clean
}

// trimPrefix 移除前缀并标记成功；同时清理结果开头的多余 "."（来自连续点）。
// 调 truncateLongSegments 把超长末段（CamelCase 类名通常 30+ char）截断到
// 25 char + "…"，避免 NR/LTE 子树下的 PdschDedicated...List 类全名仍超 80 char。
func trimPrefix(s, prefix string) (string, bool) {
	if !strings.HasPrefix(s, prefix) {
		return s, false
	}
	rest := strings.TrimPrefix(s, prefix)
	rest = strings.TrimLeft(rest, ".")
	return truncateLongSegments(rest), true
}

// truncateLongSegments 对每一段超过 25 char 的 CamelCase 末名做"前缀 + …"截断。
// 标识符末段才长（域名段 "LTE" / "RAN" / "PHY" 都很短），所以单段截断不损失
// 中段导航上下文。
func truncateLongSegments(s string) string {
	const maxSegLen = 25
	parts := strings.Split(s, ".")
	for i, p := range parts {
		if len(p) > maxSegLen {
			parts[i] = p[:maxSegLen] + "…"
		}
	}
	return strings.Join(parts, ".")
}
