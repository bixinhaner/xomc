package specparser

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// 派生：SpecGroup → SpecCommand（按 §R-3 规则）
// =============================================================================

// DeriveCommands 把每个 SpecGroup 派生为 1-4 个 SpecCommand。
//
// 派生规则（§R-3）：
//
//	LST: 恒生成（只要 Paths 非空）
//	MOD: Paths 中至少一条 Access ∈ {READ_WRITE, WRITE_ONLY} 时生成
//	ADD: HasInstance && GroupCode 不在 Blacklist 时生成
//	RMV: 同 ADD
//
// 同时填充 CommandCode / LogicalCode / RPCMethod / TargetPaths / TargetObject。
func DeriveCommands(cat *SpecCatalog) []*SpecCommand {
	blacklist := make(map[string]struct{}, len(cat.Blacklist))
	for _, b := range cat.Blacklist {
		blacklist[b] = struct{}{}
	}

	var out []*SpecCommand
	for _, g := range cat.Groups {
		if len(g.Paths) == 0 {
			continue
		}
		logical := groupCodeToLogicalCode(g.GroupCode)
		objectName := groupCodeToObjectName(g.GroupCode)

		// 全部 path 数组（LST 用）
		allPaths := make([]string, 0, len(g.Paths))
		writablePaths := make([]string, 0, len(g.Paths))
		for _, p := range g.Paths {
			allPaths = append(allPaths, p.StandardPath)
			if p.Access == AccessReadWrite || p.Access == AccessWriteOnly {
				writablePaths = append(writablePaths, p.StandardPath)
			}
		}

		// LST
		out = append(out, &SpecCommand{
			GroupCode:     g.GroupCode,
			Chapter:       g.Chapter,
			OperationType: OpLST,
			CommandCode:   OpLST + " " + logical,
			LogicalCode:   logical,
			CommandZhName: g.CommandZhName,
			CommandEnName: g.CommandEnName,
			TargetPaths:   allPaths,
			RPCMethod:     RPCGetParameterValues,
		})

		// MOD：仅当有可写 path
		if len(writablePaths) > 0 {
			out = append(out, &SpecCommand{
				GroupCode:     g.GroupCode,
				Chapter:       g.Chapter,
				OperationType: OpMOD,
				CommandCode:   OpMOD + " " + logical,
				LogicalCode:   logical,
				CommandZhName: g.CommandZhName,
				CommandEnName: g.CommandEnName,
				TargetPaths:   writablePaths,
				RPCMethod:     RPCSetParameterValues,
			})
		}

		// ADD/RMV：HasInstance && 不在 Blacklist
		if g.HasInstance {
			if _, blocked := blacklist[g.GroupCode]; !blocked {
				out = append(out,
					&SpecCommand{
						GroupCode:     g.GroupCode,
						Chapter:       g.Chapter,
						OperationType: OpADD,
						CommandCode:   OpADD + " " + logical,
						LogicalCode:   logical,
						CommandZhName: g.CommandZhName,
						CommandEnName: g.CommandEnName,
						TargetPaths:   []string{objectName},
						RPCMethod:     RPCAddObject,
						TargetObject:  objectName,
					},
					&SpecCommand{
						GroupCode:     g.GroupCode,
						Chapter:       g.Chapter,
						OperationType: OpRMV,
						CommandCode:   OpRMV + " " + logical,
						LogicalCode:   logical,
						CommandZhName: g.CommandZhName,
						CommandEnName: g.CommandEnName,
						TargetPaths:   []string{objectName},
						RPCMethod:     RPCDeleteObject,
						TargetObject:  objectName,
					},
				)
			}
		}
	}
	return out
}

// groupCodeToLogicalCode 把 group_code 转为 mml_commands.logical_code：
//
//	"Device.DeviceInfo.*"                                    → "DEVICE_INFO"
//	"Device.DeviceInfo.SwUpgrade.*"                          → "DEVICE_INFO_SW_UPGRADE"
//	"Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*"       → "X2_IP_ADDR_MAP_INFO"
//	"Device.FAP.GPS.*"                                       → "GPS"  (去 Device.FAP. 前缀)
//
// 规则：
//
//  1. 去 "Device." 前缀
//  2. 去 ".{i}" 段、".*"、末尾 "."
//  3. 把剩余 "." 切段，每段 CamelCase → SCREAMING_SNAKE_CASE
//  4. 段间用 "_" 连接
//  5. 输出最后 1-3 个段（去掉 "Services.FAPControl" / "Services.FAPService.{i}" 等中间层）
//
// 注：实际产生命令码需要避免冲突（X2IpAddrMapInfo vs IpAddrMapInfo 冲突可能），
// 但 v2.3 71 组的 zh_name 已经被 §R-2.4 唯一性约束，logical_code 取尾段即可保持区分度。
// skipContainers 是 group_code 中间层"容器"关键字，logical_code 派生时跳过，
// 让 "Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*" → "X2_IP_ADDR_MAP_INFO"
// 而不是 "SERVICES_FAP_CONTROL_X2_IP_ADDR_MAP_INFO"。
//
// 不能跳过 CellConfig：否则 Capabilities 与 CellConfig.Capabilities 派生撞 logical_code。
var skipContainers = map[string]struct{}{
	"Services":   {},
	"FAPControl": {},
	"FAPService": {},
}

func groupCodeToLogicalCode(gc string) string {
	s := strings.TrimPrefix(gc, "Device.")
	s = strings.TrimSuffix(s, ".*")
	s = strings.TrimSuffix(s, ".")
	// 去 .{i} 段 + 跳容器关键字
	parts := strings.Split(s, ".")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "{i}" {
			continue
		}
		if _, skip := skipContainers[p]; skip {
			continue
		}
		clean = append(clean, p)
	}
	// 截断：保留最后 2 段作为 logical_code（避免命令名过长 + 保留语义层次）。
	// 例外：若仅 1 段则保 1 段；2 段以内不截。
	if len(clean) > 2 {
		clean = clean[len(clean)-2:]
	}
	// CamelCase → SCREAMING_SNAKE_CASE
	out := make([]string, 0, len(clean))
	for _, p := range clean {
		out = append(out, camelToSnakeUpper(p))
	}
	return strings.Join(out, "_")
}

// deriveLogicalCodeFromCommandCode 从 command_code 派生 logical_code（去 op 前缀）。
//
// command_code 格式为 "<OP> <LOGICAL>"（空格分隔），如 "LST DEVICE_INFO"。
// 与 internal/mml 包同名函数保持一致行为（DB 列 logical_code 已 DROP，读时派生）。
func deriveLogicalCodeFromCommandCode(commandCode, op string) string {
	if op != "" {
		prefix := op + " "
		if len(commandCode) > len(prefix) && commandCode[:len(prefix)] == prefix {
			return commandCode[len(prefix):]
		}
	}
	if parts := strings.SplitN(commandCode, " ", 2); len(parts) == 2 {
		return parts[1]
	}
	return commandCode
}

// camelToSnakeUpper 把 "X2IpAddrMapInfo" → "X2_IP_ADDR_MAP_INFO"。
//   - 连续大写视为一个词（"IP" / "X2"），但跟着的小写字母拆出（"IpAddr" → "Ip_Addr"）
//   - 数字粘附前一个字母（"X2" 保留）
//
// 简单实现：在大写字母前（前一个是小写或数字）插下划线，再 ToUpper。
// camelToSnakeUpper 把 CamelCase 转 SCREAMING_SNAKE_CASE。
//
// 规则（按"词边界"识别）：
//
//  1. lower → upper 边界插下划线（"UserLabel" → "USER_LABEL"）
//  2. upper-cluster → 新词（连续大写后跟小写）：在新词首字母前插（"IPAddr" → "IP_ADDR"）
//  3. digit → upper 但下一个是 lower：在 upper 前插（"X2IpAddr" → "X2_IP_ADDR"）
//  4. digit → upper 但下一个仍是 upper：不插（"3GPPSpec" → "3GPP_SPEC"，3 与 GPP 视为同词）
//  5. digit 紧贴前一个字母：不插（"X2" 保留）
func camelToSnakeUpper(s string) string {
	if s == "" {
		return s
	}
	var sb strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && isUpper(r) {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			switch {
			case isLower(prev):
				sb.WriteByte('_') // 规则 1
			case isUpper(prev) && next != 0 && isLower(next):
				sb.WriteByte('_') // 规则 2
			case isDigit(prev) && next != 0 && isLower(next):
				sb.WriteByte('_') // 规则 3：数字 + 新词
			}
		}
		sb.WriteRune(toUpper(r))
	}
	return sb.String()
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}

// groupCodeToObjectName 把 group_code 转为父对象路径，用于 ADD/RMV 的 TargetObject：
//
//	"Device.X.{i}.*"  → "Device.X."
//	"Device.Y.{i}.Z.{i}.*"  → "Device.Y.{i}.Z." (保留外层 {i}, 末层 {i} 转父对象)
//
// 规则：找到最后一个 ".{i}." 或末尾 "{i}.*"，截到它之前 + "."
func groupCodeToObjectName(gc string) string {
	// 去末尾 .*
	s := strings.TrimSuffix(gc, ".*")
	// 末尾若是 .{i} 则去掉 {i}（保留前一 "."）
	s = strings.TrimSuffix(s, ".{i}")
	// 保证末尾有 "."
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

// =============================================================================
// DB Snapshot 加载
// =============================================================================

// LoadDBSnapshot 从 PG 读 catalog 现状。仅读 source='standard' 行（admin 不影响 diff）。
func LoadDBSnapshot(ctx context.Context, pool *pgxpool.Pool, version string) (*DBSnapshot, error) {
	snap := &DBSnapshot{
		StandardParams:   make(map[string]*StandardParamRow),
		Commands:         make(map[string]*MMLCommandRow),
		CommandSubFields: make(map[string]map[string]*SubFieldRow),
	}

	// 1. standard_params 全量
	rows, err := pool.Query(ctx, `
		SELECT standard_path, entry_type, COALESCE(access,''), COALESCE(data_type,''),
		       COALESCE(change_applies,''), min_value, max_value
		  FROM standard_params`)
	if err != nil {
		return nil, fmt.Errorf("query standard_params: %w", err)
	}
	for rows.Next() {
		var r StandardParamRow
		if err := rows.Scan(&r.StandardPath, &r.EntryType, &r.Access, &r.DataType,
			&r.ChangeApplies, &r.MinValue, &r.MaxValue); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan standard_params: %w", err)
		}
		snap.StandardParams[r.StandardPath] = &r
	}
	rows.Close()

	// 2. mml_commands (source='standard' for this version chapter scope)
	// logical_code 不再持久化为 DB 列（已 DROP）；读后从 command_code 派生填充。
	rows, err = pool.Query(ctx, `
		SELECT command_code, COALESCE(operation_type,''),
		       COALESCE(target_paths::text,'[]'), source
		  FROM mml_commands
		 WHERE source = 'standard'`)
	if err != nil {
		return nil, fmt.Errorf("query mml_commands: %w", err)
	}
	for rows.Next() {
		var r MMLCommandRow
		var targetPathsJSON string
		if err := rows.Scan(&r.CommandCode, &r.OperationType, &targetPathsJSON, &r.Source); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan mml_commands: %w", err)
		}
		r.LogicalCode = deriveLogicalCodeFromCommandCode(r.CommandCode, r.OperationType)
		if targetPathsJSON != "" && targetPathsJSON != "null" {
			_ = json.Unmarshal([]byte(targetPathsJSON), &r.TargetPaths)
		}
		snap.Commands[r.CommandCode] = &r
	}
	rows.Close()

	// 3. mml_command_sub_fields JOIN standard_params 拿 standard_path
	rows, err = pool.Query(ctx, `
		SELECT c.command_code, sp.standard_path, sf.mml_code, sf.sort_order
		  FROM mml_command_sub_fields sf
		  JOIN mml_commands  c  ON c.id  = sf.command_id
		  JOIN standard_params sp ON sp.id = sf.standard_path_id
		 WHERE c.source = 'standard'`)
	if err != nil {
		return nil, fmt.Errorf("query mml_command_sub_fields: %w", err)
	}
	for rows.Next() {
		var r SubFieldRow
		if err := rows.Scan(&r.CommandCode, &r.StandardPath, &r.MmlCode, &r.SortOrder); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan sub_fields: %w", err)
		}
		if _, ok := snap.CommandSubFields[r.CommandCode]; !ok {
			snap.CommandSubFields[r.CommandCode] = make(map[string]*SubFieldRow)
		}
		snap.CommandSubFields[r.CommandCode][r.StandardPath] = &r
	}
	rows.Close()
	return snap, nil
}

// =============================================================================
// Diff: SpecCatalog × DBSnapshot → DiffReport
// =============================================================================

// Diff 比对 spec 与 DB 现状，输出 DiffReport。
//
// 算法：
//  1. 派生 SpecCommands (DeriveCommands)
//  2. 对每个 SpecPath：在 DB.StandardParams 查 → 缺则 NewStandardParams
//  3. 对每个 SpecCommand：
//     - DB.Commands 无 → NewCommands
//     - DB.Commands 有但 target_paths 不一致 → UpdatedCommands
//  4. 对每个 (SpecCommand, standardPath) 组合：DB.CommandSubFields 无关联 → NewSubFieldLinks
//  5. 反向：DB.Commands 有但 spec 无 → OrphanCommands
func Diff(cat *SpecCatalog, snap *DBSnapshot) *DiffReport {
	rep := &DiffReport{}
	commands := DeriveCommands(cat)

	// 索引 SpecCommands by code（用于反向 Orphan 检查）
	specByCode := make(map[string]*SpecCommand, len(commands))
	for _, c := range commands {
		specByCode[c.CommandCode] = c
	}

	// 1. NewStandardParams
	seenPath := make(map[string]struct{})
	for _, g := range cat.Groups {
		for _, p := range g.Paths {
			if _, ok := seenPath[p.StandardPath]; ok {
				continue
			}
			seenPath[p.StandardPath] = struct{}{}
			if _, ok := snap.StandardParams[p.StandardPath]; !ok {
				rep.NewStandardParams = append(rep.NewStandardParams, p)
			}
		}
	}

	// 2. NewCommands / UpdatedCommands
	for _, sc := range commands {
		dbCmd, ok := snap.Commands[sc.CommandCode]
		if !ok {
			rep.NewCommands = append(rep.NewCommands, sc)
			continue
		}
		// 比较 target_paths 集合（顺序无关）
		if !pathSetEqual(sc.TargetPaths, dbCmd.TargetPaths) {
			rep.UpdatedCommands = append(rep.UpdatedCommands, sc)
		}
	}

	// 3. NewSubFieldLinks
	for _, sc := range commands {
		// 只对 LST/MOD 派生 sub_field 关联（ADD/RMV 的 target 是父对象，不需 sub_field）
		if sc.OperationType == OpADD || sc.OperationType == OpRMV {
			continue
		}
		dbLinks := snap.CommandSubFields[sc.CommandCode]
		// 找出 spec 派生该 command 对应的 SpecGroup（按 GroupCode）
		var grp *SpecGroup
		for _, g := range cat.Groups {
			if g.GroupCode == sc.GroupCode {
				grp = g
				break
			}
		}
		if grp == nil {
			continue
		}
		// 对该 command 对应的每条 path：LST=所有 path / MOD=可写 path
		var paths []*SpecPath
		switch sc.OperationType {
		case OpLST:
			paths = grp.Paths
		case OpMOD:
			paths = make([]*SpecPath, 0, len(grp.Paths))
			for _, p := range grp.Paths {
				if p.Access == AccessReadWrite || p.Access == AccessWriteOnly {
					paths = append(paths, p)
				}
			}
		}
		for idx, p := range paths {
			if dbLinks != nil {
				if _, ok := dbLinks[p.StandardPath]; ok {
					continue
				}
			}
			rep.NewSubFieldLinks = append(rep.NewSubFieldLinks, &SubFieldLink{
				CommandCode:  sc.CommandCode,
				StandardPath: p.StandardPath,
				MmlCode:      paramNameToMMLCode(p.ParamName),
				SortOrder:    idx + 1,
				ChineseName:  p.ChineseName,
				ParamName:    p.ParamName,
			})
		}
	}

	// 4. OrphanCommands（DB 有 source='standard' 但 spec 不覆盖）
	for code := range snap.Commands {
		if _, ok := specByCode[code]; !ok {
			rep.OrphanCommands = append(rep.OrphanCommands, code)
		}
	}
	sort.Strings(rep.OrphanCommands)

	// Summary
	rep.Summary = DiffSummary{
		NewStandardParams: len(rep.NewStandardParams),
		NewCommands:       len(rep.NewCommands),
		UpdatedCommands:   len(rep.UpdatedCommands),
		NewSubFieldLinks:  len(rep.NewSubFieldLinks),
		OrphanCommands:    len(rep.OrphanCommands),
		OrphanLinks:       len(rep.OrphanLinks),
		CrossWarnings:     len(rep.CrossCheckWarnings),
	}
	return rep
}

// pathSetEqual 比较两个 path 集合是否相等（顺序无关）。
func pathSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, p := range a {
		m[p]++
	}
	for _, p := range b {
		m[p]--
		if m[p] < 0 {
			return false
		}
	}
	return true
}

// paramNameToMMLCode 把 ParamName（CamelCase）转 mml_code（SCREAMING_SNAKE）。
//   - "UserLabel"        → "USER_LABEL"
//   - "ManufacturerOUI"  → "MANUFACTURER_OUI"
//   - "3GPPSpecVersion"  → "3GPP_SPEC_VERSION"
func paramNameToMMLCode(name string) string {
	return camelToSnakeUpper(name)
}
