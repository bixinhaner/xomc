package mmlstandardloader

// command_derivator.go — 把 seed 中的 SeedCommand 翻译为 mml_commands 行（LST/MOD/ADD/RMV）。
//
// 派生规则（与任务 #3 验证标准对齐）：
//   - LST 总生成（前提：command 含 ≥1 个 param）
//   - MOD 仅当 command 含 ≥1 个 access="RW" 参数
//   - ADD/RMV 同时满足：
//       1) object_path 含 ".{i}." 占位符（动态实例对象）
//       2) 至少 1 个 RW 参数
//       3) command 自身 non_creatable=false
//       4) object_path 不在 SeedRoot.NonCreatableObjects 中
//
// command_code 格式（任务 #3 §4 明确）：
//   "<OP> <LOGICAL_CODE>"，空格分隔。如 "LST DEVICE_INFO" / "MOD MU_SW_UPGRADE"。
//
// logical_code 派生（同 chapter / 全表 UNIQUE，写入 mml_commands.logical_code）：
//   1. object_path 去 ".{i}"，trim 前缀 "Device."、trim 末尾 "."
//   2. 末段在 shortQualifierSet（LTE/EPC/RAN/MAC/PHY/MBSFN/SwUpgrade …）→ 取末 2 段
//      否则取末 1 段
//   3. 每段 camelCase → SNAKE_UPPER，"_" 拼接
//   4. 若多个 object_path 派生出同一 logical_code → 全部冲突项 depth+1（再向 root 追一段）
//      循环直到无冲突或耗尽 segs（最差兜底全路径）

import (
	"strings"
	"unicode"
)

// shortQualifierSet — 末段是这些"短限定符"时强制取末 2 段，避免大量 "MAC"/"PHY"/"EPC" 同名。
//
// 来源：观察 cmcc_tdlte_v23.json 71 个 object_path 后的高频末段聚类。
var shortQualifierSet = map[string]bool{
	// TR-181 RAT/协议层缩写
	"LTE": true, "NR": true, "GSM": true, "UMTS": true,
	"EUTRA": true, "IRAT": true, "UTRA": true, "GERAN": true,
	// LTE 内部子层
	"EPC": true, "RAN": true, "MAC": true, "PHY": true, "MBSFN": true,
	// 强语义末段（短得自带歧义）
	"SwUpgrade":    true,
	"Capabilities": true,
	"GPS":          true,
	"VoLTE":        true,
	"SCTP":         true,
}

// DerivedCommand 是 SeedCommand 派生出的 1-4 条 mml_commands 之一。
type DerivedCommand struct {
	OperationType string      // OpLST / OpMOD / OpADD / OpRMV
	LogicalCode   string      // DEVICE_INFO 等；同 source 全表 UNIQUE（生成时保证）
	CommandCode   string      // "LST DEVICE_INFO" — 即 "<op> <logical_code>"
	NameZh        string      // "列出 设备基本信息"
	NameEn        string      // "List Device Basic Info"（拷贝中文 — 暂无英文 seed）
	LogicalNameZh string      // "设备基本信息" — 命令树叶子标签来源
	LogicalNameEn string      // 同上 en（暂用 ASCII 化的 logical_code）
	GroupCode     string      // SeedGroup.Code（SA…SR）— 用于 mml_command_groups 关联
	ObjectPath    string      // 原 seed object_path（含 .{i}.），用于 standard_params object 行 + sub_field 关联
	RPCMethod     string      // GetParameterValues / SetParameterValues / AddObject / DeleteObject
	TargetObject  string      // ADD/RMV 时 = strip{i}(ObjectPath)；LST/MOD 时空
	SubFields     []SeedParam // LST 取全部；MOD 取全部 RW；ADD 仅取当前实例层 RW；RMV 为 nil
}

// HasSubFields 标识需要写入命令参数绑定的操作。
func (d DerivedCommand) HasSubFields() bool {
	return d.OperationType == OpLST || d.OperationType == OpMOD || d.OperationType == OpADD
}

// DeriveCommandsFromSeed 走全 seed 派生命令，并解决 logical_code 冲突。
//
// 返回顺序与 seed 内顺序一致（chapter 主序，chapter 内 object_path 主序，
// 同 object_path 内 LST/MOD/ADD/RMV 主序）— 便于排查 + display_order 直接用 index。
func DeriveCommandsFromSeed(seed *SeedRoot) []DerivedCommand {
	nonCreatable := seed.NonCreatableSet()

	// 第一遍：每个 SeedCommand 派生出 0-4 条原子命令（logical_code 用初值，未消歧）。
	// 同时收集所有 object_path → 集中跑 logical_code 派生。
	type slot struct {
		group     *SeedGroup
		cmd       *SeedCommand
		ops       []string // ["LST","MOD","ADD","RMV"] 子集
		rwParams  []SeedParam
		addParams []SeedParam
		allParams []SeedParam
	}
	var slots []slot
	objectPaths := make([]string, 0, 96)
	pathSeen := make(map[string]bool, 96)

	for gi := range seed.Groups {
		g := &seed.Groups[gi]
		for ci := range g.Commands {
			c := &g.Commands[ci]
			if len(c.Params) == 0 {
				// 任务 #3：LST 至少要有参数；空 command 跳过
				continue
			}

			// 收集 RW 集合 + 全集。ADD 只能携带当前新实例层的字段；更深一层
			// 的 {i} 属于子对象，必须通过子对象自己的 ADD 命令创建。
			var rw []SeedParam
			var addRW []SeedParam
			objectInstanceDepth := strings.Count(c.ObjectPath, "{i}")
			for _, p := range c.Params {
				if p.IsWritable() {
					rw = append(rw, p)
					if strings.Count(p.Path, "{i}") == objectInstanceDepth {
						addRW = append(addRW, p)
					}
				}
			}
			s := slot{group: g, cmd: c, allParams: c.Params, rwParams: rw, addParams: addRW}

			// 操作集决策
			s.ops = append(s.ops, OpLST) // LST 总加
			if len(rw) > 0 {
				s.ops = append(s.ops, OpMOD)
			}
			if !c.NonCreatable && len(addRW) > 0 &&
				strings.Contains(c.ObjectPath, ".{i}.") &&
				!nonCreatable[c.ObjectPath] {
				s.ops = append(s.ops, OpADD, OpRMV)
			}

			slots = append(slots, s)
			if !pathSeen[c.ObjectPath] {
				pathSeen[c.ObjectPath] = true
				objectPaths = append(objectPaths, c.ObjectPath)
			}
		}
	}

	logicalCodes := DeriveLogicalCodes(objectPaths)

	// 第二遍：把派生命令展开
	var out []DerivedCommand
	for _, s := range slots {
		lc := logicalCodes[s.cmd.ObjectPath]
		baseNameZh := s.cmd.Name
		nameEnAscii := logicalCodeToTitle(lc) // 占位英文（无 seed）
		targetObj := stripInstanceIndex(s.cmd.ObjectPath)
		if !strings.HasSuffix(targetObj, ".") {
			targetObj += "."
		}
		for _, op := range s.ops {
			d := DerivedCommand{
				OperationType: op,
				LogicalCode:   lc,
				CommandCode:   op + " " + lc,
				LogicalNameZh: baseNameZh,
				LogicalNameEn: nameEnAscii,
				GroupCode:     s.group.Code,
				ObjectPath:    s.cmd.ObjectPath,
			}
			switch op {
			case OpLST:
				d.NameZh = "列出 " + baseNameZh
				d.NameEn = "List " + nameEnAscii
				d.RPCMethod = RPCGetParameterValues
				d.SubFields = s.allParams
			case OpMOD:
				d.NameZh = "修改 " + baseNameZh
				d.NameEn = "Modify " + nameEnAscii
				d.RPCMethod = RPCSetParameterValues
				d.SubFields = s.rwParams
			case OpADD:
				d.NameZh = "新增 " + baseNameZh
				d.NameEn = "Add " + nameEnAscii
				d.RPCMethod = RPCAddObject
				d.TargetObject = targetObj
				d.SubFields = s.addParams
			case OpRMV:
				d.NameZh = "删除 " + baseNameZh
				d.NameEn = "Remove " + nameEnAscii
				d.RPCMethod = RPCDeleteObject
				d.TargetObject = targetObj
			}
			out = append(out, d)
		}
	}
	return out
}

// DeriveLogicalCodes 给一组 object_path 计算唯一的 logical_code。
//
// 算法：
//  1. 每个 path 取初始 depth（短限定符末段时 depth=2，否则 depth=1）
//  2. 按 depth 渲染候选码；若有冲突，所有冲突项 depth+1
//  3. 重复直到稳定（无冲突 / 无法继续追加 → segs 全部用尽时停）
//
// 在 71 个 cmcc_tdlte_v23 object_path 上的实测：1-3 轮内全部解决；
// 最深 5 层（DeviceInfo.MU.Slot.EU.RU.RFChannel）也只到 depth=2。
func DeriveLogicalCodes(objectPaths []string) map[string]string {
	entries := make(map[string]*entry, len(objectPaths))
	for _, op := range objectPaths {
		s := segsForLogical(op)
		entries[op] = &entry{segs: s, depth: initialDepth(s)}
	}

	for iter := 0; iter < 16; iter++ { // 安全上限 — 任何 path 都 ≤16 段
		// 按当前候选分组
		groups := make(map[string][]string, len(entries))
		for op, e := range entries {
			groups[renderEntry(e)] = append(groups[renderEntry(e)], op)
		}
		anyConflict := false
		for _, ops := range groups {
			if len(ops) <= 1 {
				continue
			}
			// 把全部冲突项 depth+1（如果还能加深）
			bumpedAny := false
			for _, op := range ops {
				e := entries[op]
				if e.depth < len(e.segs) {
					e.depth++
					bumpedAny = true
				}
			}
			if bumpedAny {
				anyConflict = true
			}
			// 否则已经吃满全路径，无法再消歧（极端情况，seed 设计错误才出现）
		}
		if !anyConflict {
			break
		}
	}

	out := make(map[string]string, len(entries))
	for op, e := range entries {
		out[op] = renderEntry(e)
	}
	return out
}

// segsForLogical 把 object_path 切成 logical_code 派生用的有序段：
//   - 去 ".{i}" 占位
//   - 去 "Device." 前缀（顶层冗余）
//   - 去末尾 "."
//
// "Device.Services.FAPService.{i}.CellConfig.LTE.EPC."
// → ["Services","FAPService","CellConfig","LTE","EPC"]
func segsForLogical(objectPath string) []string {
	s := stripInstanceIndex(objectPath)
	s = strings.TrimPrefix(s, "Device.")
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

// stripInstanceIndex 移除 ".{i}" 占位符。
//
//	"Device.X.{i}.Y." → "Device.X.Y."
//	"Device.X.{i}."  → "Device.X."
func stripInstanceIndex(p string) string {
	return strings.ReplaceAll(p, ".{i}", "")
}

// initialDepth 末段是 shortQualifier → 2，否则 1。
func initialDepth(segs []string) int {
	n := len(segs)
	if n == 0 {
		return 0
	}
	if n >= 2 && shortQualifierSet[segs[n-1]] {
		return 2
	}
	return 1
}

// renderEntry 把 entry.segs[末 depth 段] 拼成 SNAKE_UPPER logical_code。
func renderEntry(e *entry) string {
	n := len(e.segs)
	if n == 0 {
		return "ROOT"
	}
	d := e.depth
	if d > n {
		d = n
	}
	if d < 1 {
		d = 1
	}
	parts := e.segs[n-d:]
	chunks := make([]string, len(parts))
	for i, p := range parts {
		chunks[i] = camelToSnakeUpper(p)
	}
	return strings.Join(chunks, "_")
}

// entry 私有 — 仅 DeriveLogicalCodes 用。
type entry struct {
	segs  []string
	depth int
}

// camelToSnakeUpper 把 PascalCase / camelCase / 已含下划线 标识符转 SNAKE_UPPER。
//
// 规则（边界检测）：
//   - 当前是大写，且前一字符是小写/数字           → 前置 _（"DeviceInfo" → "DEVICE_INFO"）
//   - 当前是大写，前一是大写但下一是小写         → 前置 _（"FAPService" → "FAP_SERVICE"）
//   - 已有的 '_' 直通；连续 _ 折叠为 1 个
//   - 数字、小写、其它字符直接拷贝（大写化）
//
// 测试：
//
//	"DeviceInfo"    → "DEVICE_INFO"
//	"FAPService"    → "FAP_SERVICE"
//	"MU"            → "MU"
//	"VlanInterface" → "VLAN_INTERFACE"
//	"X_COM"         → "X_COM"
//	"IPv4Address"   → "I_PV4_ADDRESS"  // 边角；TR-181 暂无此名
func camelToSnakeUpper(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if r == '_' {
			if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
				b.WriteByte('_')
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			var nextLower bool
			if i+1 < len(runes) {
				nextLower = unicode.IsLower(runes[i+1])
			}
			needSep := unicode.IsLower(prev) || unicode.IsDigit(prev) ||
				(unicode.IsUpper(prev) && nextLower)
			if needSep && b.Len() > 0 && b.String()[b.Len()-1] != '_' {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// logicalCodeToTitle 把 "DEVICE_INFO" 反向变 "Device Info"（暂无英文 seed 时用）。
func logicalCodeToTitle(lc string) string {
	parts := strings.Split(lc, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		lower := strings.ToLower(p)
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}
