package mml

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ============================================================
// mml_parser.go — T-0123-P1 MML 字符串解析
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.4
//
// 老 OMC MML 语法（playwright 2026-05-14 实测）:
//   LST <CODE>:lstId={CODE1,CODE2,...};
//   MOD <CODE>:Field1=Value1,Field2=Value2;
//   ADD <CODE>:Field1=Value1;
//   RMV <CODE>:Index=N;
//   多条 ; 分隔，末尾 ;
//
// 容错策略：
//   - 大小写：op 忽略大小写（lst / LST 等价）
//   - 空格：op/code/key/value 周围空格容忍
//   - 双引号：值含空格/逗号/分号时用 "..." 包裹，内部 \" 转义
//   - 连续 ;;：跳过空段
//   - 缺末尾 ;：补一个解析
//
// 错误策略：单条 statement 失败累入 ParseError 不中断整体；返回的 statements
// 是成功解析的子集，调用方根据 ParseError 列表决定是否拒绝整体提交。
// ============================================================

// CommandLookup 由 service 注入，用于把 (op, logical_code) 映射到 mml_commands
// 实体 + 该命令的 sub_fields。解耦 parser 与 DB 层。
//
// 行为约定：
//   - 命中：返 (*MMLCommand, []MMLCommandSubField, nil)
//   - 未命中：返 (nil, nil, ErrCommandNotFound) 或包装该错误
//   - DB 故障：返 (nil, nil, err) 其他错误
type CommandLookup interface {
	LookupByLogicalCode(ctx context.Context, op, logicalCode string) (*MMLCommand, []MMLCommandSubField, error)
}

// ErrAmbiguousCommand 由 CommandLookup 在同 (op, logical_code) 多命中时返回。
// 见 §N.10 W2 决议：(op, logical_code) 应唯一；多命中是字典数据不一致信号。
// 注：ErrCommandNotFound 复用 admin_repository.go 中的同名 sentinel（已在 T-0123-P0 定义）。
var ErrAmbiguousCommand = errors.New("mml: ambiguous (op, logical_code) — multiple commands matched")

// ParseMMLString 把 MML 字符串解析为 statements。
//
// 参数：
//   - lookup 可为 nil（仅做语法解析，不验证 command_id / sub_field_ids）
//   - 提供 lookup 时按 (op, logical_code) 查 command + sub_fields，命中后填充
//     stmt.CommandID + 解析 LST.SelectedSubFieldIDs / 校验 MOD/ADD.Values keys
//
// 返回：
//   - statements: 所有成功解析的 statement（含 lookup 失败但语法正确的）
//   - errors: 失败条目（不中断整体解析）
func ParseMMLString(ctx context.Context, s string, lookup CommandLookup) ([]Statement, []ParseError) {
	stmts := make([]Statement, 0)
	errs := make([]ParseError, 0)

	parts := splitStatements(s)
	for idx, raw := range parts {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		stmt, err := parseSingleStatement(trimmed)
		if err != nil {
			errs = append(errs, ParseError{
				StatementIndex: idx,
				Raw:            trimmed,
				Reason:         err.Error(),
			})
			continue
		}

		if lookup != nil {
			cmd, subFields, lkErr := lookup.LookupByLogicalCode(ctx, stmt.OperationType, stmt.LogicalCode)
			if lkErr != nil {
				errs = append(errs, ParseError{
					StatementIndex: idx,
					Raw:            trimmed,
					Reason:         fmt.Sprintf("lookup (%s,%s): %v", stmt.OperationType, stmt.LogicalCode, lkErr),
				})
				// lookup 失败时仍保留 stmt 语法解析结果（前端可显原文 + badge）
				stmts = append(stmts, stmt)
				continue
			}
			stmt.CommandID = &cmd.ID

			switch stmt.OperationType {
			case "LST":
				resolveLSTSubFieldIDs(&stmt, subFields)
			case "MOD", "ADD":
				checkValueKeys(&stmt, subFields)
			}
		}

		stmts = append(stmts, stmt)
	}

	return stmts, errs
}

// ============================================================
// 分段：按 ; 切割 statements（双引号内 ; 不切割）
// ============================================================

func splitStatements(s string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	escape := false

	for _, r := range s {
		switch {
		case escape:
			cur.WriteRune(r)
			escape = false
		case r == '\\' && inQuote:
			cur.WriteRune(r)
			escape = true
		case r == '"' && !inQuote:
			inQuote = true
			cur.WriteRune(r)
		case r == '"' && inQuote:
			inQuote = false
			cur.WriteRune(r)
		case r == ';' && !inQuote:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// ============================================================
// 单条 statement 解析
// ============================================================

func parseSingleStatement(raw string) (Statement, error) {
	// 找 OP 与 rest 的分界（第一个空白字符）
	spaceIdx := -1
	for i, r := range raw {
		if r == ' ' || r == '\t' {
			spaceIdx = i
			break
		}
	}
	if spaceIdx < 0 {
		return Statement{}, fmt.Errorf("缺少操作类型与命令编码之间的空格: %q", raw)
	}

	op := strings.ToUpper(strings.TrimSpace(raw[:spaceIdx]))
	if !isValidOp(op) {
		return Statement{}, fmt.Errorf("非法操作类型 %q（应为 LST/MOD/ADD/RMV）", op)
	}

	rest := strings.TrimSpace(raw[spaceIdx+1:])
	if rest == "" {
		return Statement{}, fmt.Errorf("缺少命令编码: %q", raw)
	}

	// 按 ':' 分 code 与 params（仅切第一个 :，因为 value 可能含 :）
	var code, paramsRaw string
	colonIdx := findFirstColonOutsideQuote(rest)
	if colonIdx < 0 {
		code = strings.TrimSpace(rest)
		paramsRaw = ""
	} else {
		code = strings.TrimSpace(rest[:colonIdx])
		paramsRaw = strings.TrimSpace(rest[colonIdx+1:])
	}

	if code == "" {
		return Statement{}, fmt.Errorf("命令编码为空: %q", raw)
	}

	stmt := Statement{
		OperationType: op,
		LogicalCode:   code,
	}

	if paramsRaw == "" {
		return stmt, nil
	}

	switch op {
	case "LST":
		codes, err := parseLSTParams(paramsRaw)
		if err != nil {
			return Statement{}, err
		}
		stmt.SelectedMMLCodes = codes
	case "MOD", "ADD":
		values, err := parseKeyValueParams(paramsRaw)
		if err != nil {
			return Statement{}, err
		}
		stmt.Values = values
	case "RMV":
		idx, err := parseRMVParams(paramsRaw)
		if err != nil {
			return Statement{}, err
		}
		stmt.RmvInstanceIndex = idx
	}

	return stmt, nil
}

func findFirstColonOutsideQuote(s string) int {
	inQuote := false
	escape := false
	for i, r := range s {
		switch {
		case escape:
			escape = false
		case r == '\\' && inQuote:
			escape = true
		case r == '"':
			inQuote = !inQuote
		case r == ':' && !inQuote:
			return i
		}
	}
	return -1
}

// parseLSTParams 解析 "lstId={CODE1,CODE2,...}" 形式。
// 大小写不敏感（lstId / lstid / LSTID 都接受）。
func parseLSTParams(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	const prefix = "lstid"
	if len(s) < len(prefix) || strings.ToLower(s[:len(prefix)]) != prefix {
		return nil, fmt.Errorf("LST 参数应以 lstId=... 开头: %q", s)
	}
	s = strings.TrimSpace(s[len(prefix):])
	if !strings.HasPrefix(s, "=") {
		return nil, fmt.Errorf("LST 参数 lstId 后缺少 =: %q", s)
	}
	s = strings.TrimSpace(s[1:])
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("LST 参数应用 {...} 包裹: %q", s)
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return []string{}, nil
	}
	rawCodes := strings.Split(inner, ",")
	codes := make([]string, 0, len(rawCodes))
	for _, c := range rawCodes {
		c = strings.TrimSpace(c)
		if c != "" {
			codes = append(codes, c)
		}
	}
	return codes, nil
}

// parseKeyValueParams 解析 "K1=V1,K2=V2,..." 形式。
// 支持双引号包裹值。
func parseKeyValueParams(s string) (map[string]string, error) {
	pairs, err := splitOnUnquotedComma(s)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		eqIdx := findFirstEqOutsideQuote(p)
		if eqIdx < 0 {
			return nil, fmt.Errorf("键值对缺少 =: %q", p)
		}
		k := strings.TrimSpace(p[:eqIdx])
		v := strings.TrimSpace(p[eqIdx+1:])
		v = unquoteIfNeeded(v)
		if k == "" {
			return nil, fmt.Errorf("键名为空: %q", p)
		}
		out[k] = v
	}
	return out, nil
}

func splitOnUnquotedComma(s string) ([]string, error) {
	var parts []string
	var cur strings.Builder
	inQuote := false
	escape := false
	for _, r := range s {
		switch {
		case escape:
			cur.WriteRune(r)
			escape = false
		case r == '\\' && inQuote:
			cur.WriteRune(r)
			escape = true
		case r == '"' && !inQuote:
			inQuote = true
			cur.WriteRune(r)
		case r == '"' && inQuote:
			inQuote = false
			cur.WriteRune(r)
		case r == ',' && !inQuote:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	if inQuote {
		return nil, fmt.Errorf("未闭合的双引号")
	}
	return parts, nil
}

func findFirstEqOutsideQuote(s string) int {
	inQuote := false
	escape := false
	for i, r := range s {
		switch {
		case escape:
			escape = false
		case r == '\\' && inQuote:
			escape = true
		case r == '"':
			inQuote = !inQuote
		case r == '=' && !inQuote:
			return i
		}
	}
	return -1
}

// unquoteIfNeeded 若值用 "..." 包裹则剥引号并 unescape 内部 \"。
func unquoteIfNeeded(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	inner := s[1 : len(s)-1]
	return strings.ReplaceAll(inner, `\"`, `"`)
}

// parseRMVParams 解析 "Index=N" 形式。
func parseRMVParams(s string) (*int, error) {
	s = strings.TrimSpace(s)
	const prefix = "index"
	if len(s) < len(prefix) || strings.ToLower(s[:len(prefix)]) != prefix {
		return nil, fmt.Errorf("RMV 参数应为 Index=N: %q", s)
	}
	s = strings.TrimSpace(s[len(prefix):])
	if !strings.HasPrefix(s, "=") {
		return nil, fmt.Errorf("RMV 参数 Index 后缺少 =: %q", s)
	}
	s = strings.TrimSpace(s[1:])
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("RMV Index 必须为整数: %w", err)
	}
	return &n, nil
}

// ============================================================
// lookup 命中后的 sub_field id 解析（LST）+ unknown_codes 累加（MOD/ADD）
// ============================================================

// resolveLSTSubFieldIDs 把 stmt.SelectedMMLCodes 按 lookup 命中的 sub_field 映射成
// stmt.SelectedSubFieldIDs，未命中 code 累入 stmt.UnknownCodes。
//
// 顺序：**保持用户输入顺序**（textbox 与 UI 勾选状态 round-trip 同源），不按 sort_order 重排。
func resolveLSTSubFieldIDs(stmt *Statement, subFields []MMLCommandSubField) {
	if len(stmt.SelectedMMLCodes) == 0 {
		return
	}
	codeToSF := make(map[string]MMLCommandSubField, len(subFields))
	for _, sf := range subFields {
		codeToSF[sf.MMLCode] = sf
	}
	stmt.SelectedSubFieldIDs = stmt.SelectedSubFieldIDs[:0]
	unknown := make([]string, 0)
	for _, code := range stmt.SelectedMMLCodes {
		if sf, ok := codeToSF[code]; ok {
			stmt.SelectedSubFieldIDs = append(stmt.SelectedSubFieldIDs, sf.ID)
		} else {
			unknown = append(unknown, code)
		}
	}
	stmt.UnknownCodes = unknown
}

func checkValueKeys(stmt *Statement, subFields []MMLCommandSubField) {
	if len(stmt.Values) == 0 {
		return
	}
	known := make(map[string]struct{}, len(subFields))
	for _, sf := range subFields {
		known[sf.MMLCode] = struct{}{}
	}
	unknown := make([]string, 0)
	for k := range stmt.Values {
		if _, ok := known[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	stmt.UnknownCodes = unknown
}
