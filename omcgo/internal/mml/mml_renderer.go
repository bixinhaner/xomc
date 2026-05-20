package mml

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// ============================================================
// mml_renderer.go — T-0123-P1 MML 字符串渲染
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.3
//
// 老 OMC 实测语法（playwright 2026-05-14 BSC Configuration > Basic Info）:
//   LST <CODE>:lstId={CODE1,CODE2,...};
//   MOD <CODE>:Field1=Value1,Field2=Value2;
//   ADD <CODE>:Field1=Value1;
//   RMV <CODE>:Index=N;
//   多条 ; 分隔，末尾 ;
//
// 渲染原则：
//   - LST/MOD/ADD 空字段时返裸 op (e.g., "LST DEVICE_INFO")，不输出 :lstId={} / : 后缀
//   - 多 sub_field 按 sort_order 排序，保证渲染稳定（前后端一致 + commit diff 友好）
//   - Values 含特殊字符（, ; : = { } 空格 双引号）时双引号包裹 + 转义内部双引号
// ============================================================

// Statement 是 MML 字符串中的单条语句的抽象。
// 共享给 renderer / parser / executor 三方使用。
//
// 字段说明：
//   - LogicalCode 是 MML 字符串中出现的 code（不带 op 前缀），即 mml_commands.logical_code
//   - CommandID 由 parser 通过 CommandLookup 填充；renderer 不依赖此字段
//   - SelectedSubFieldIDs 由 parser 通过 lookup 从 SelectedMMLCodes 解析填充；renderer 用此字段查 mml_code
//   - SelectedMMLCodes 是 parser 解出的原始 LST 字段 code 列表；renderer 不用
//   - Values 是 MOD/ADD 的 mml_code → 值 映射
//   - RmvInstanceIndex 是 RMV 单实例编号（用户决策 2026-05-20：RMV 保持单实例；
//     R-7 多选已禁用 — 1 MML 命令 = 1 RPC 在协议层不可分批）
//   - InstanceSelectors 是 R-4 多层 {i} 实例选择器。catalog 一律用通用 `.{i}.`
//     占位符（不区分层级），selector key 仅作 UX 标签（如 iα/iβ/iγ），实际按 key
//     字典序左到右映射到路径中的 `.{i}.` 位置。Executor 编译 commands[] entry 时
//     调用 substituteInstanceSelectors 替换为具体实例号。
//     数量不匹配（路径 `.{i}.` 计数 ≠ selector 数）→ ErrInvalidRequest。
//   - UnknownCodes 是 parser 发现的未在 lookup 命中的 mml_code（前端 toast 提示）
type Statement struct {
	CommandID           *uuid.UUID        `json:"command_id,omitempty"`
	LogicalCode         string            `json:"logical_code"`
	OperationType       string            `json:"operation_type"`
	SelectedSubFieldIDs []uuid.UUID       `json:"selected_sub_field_ids,omitempty"`
	SelectedMMLCodes    []string          `json:"selected_mml_codes,omitempty"`
	Values              map[string]string `json:"values,omitempty"`
	RmvInstanceIndex    *int              `json:"rmv_instance_index,omitempty"`
	InstanceSelectors   map[string]string `json:"instance_selectors,omitempty"`
	UnknownCodes        []string          `json:"unknown_codes,omitempty"`
}

// StatementWithSubFields 把 Statement 与该命令的 sub_fields 元数据绑在一起，
// 供 RenderStatements 批量渲染时复用单条函数。
type StatementWithSubFields struct {
	Statement Statement
	SubFields []MMLCommandSubField
}

// ParseError 是 parser 累加的非致命错误。整个字符串解析不因单条错而中断；
// 调用方根据 ParseError 列表判断是否拒绝整体提交。
type ParseError struct {
	StatementIndex int    `json:"statement_index"` // 0-based 的 statement 序号
	Raw            string `json:"raw"`             // 原始字符串片段
	Reason         string `json:"reason"`          // 中文错误描述
}

// RenderStatement 把单条 statement 渲染为 MML 字符串片段（不含末尾分号）。
//
// 行为：
//   - LST: 用 SelectedSubFieldIDs 在 subFields 中查找对应 mml_code，按 sort_order 排序
//   - MOD/ADD: 用 Values 在 subFields 查 mml_code 顺序，按 sort_order 排序输出
//   - RMV: 输出 Index=N（无 index 时返裸 op）
//   - 空 LST/MOD/ADD 返裸 op（"LST DEVICE_INFO" 而非 "LST DEVICE_INFO:lstId={}"）
func RenderStatement(stmt Statement, subFields []MMLCommandSubField) (string, error) {
	op := strings.ToUpper(strings.TrimSpace(stmt.OperationType))
	code := strings.TrimSpace(stmt.LogicalCode)
	if code == "" {
		return "", fmt.Errorf("RenderStatement: empty logical_code")
	}
	if !isValidOp(op) {
		return "", fmt.Errorf("RenderStatement: unsupported operation type %q", stmt.OperationType)
	}

	switch op {
	case "LST":
		codes := collectSelectedMMLCodes(stmt.SelectedSubFieldIDs, subFields)
		if len(codes) == 0 {
			return "LST " + code, nil
		}
		return fmt.Sprintf("LST %s:lstId={%s}", code, strings.Join(codes, ",")), nil

	case "MOD", "ADD":
		kvs := collectKeyValuePairs(stmt.Values, subFields)
		if len(kvs) == 0 {
			return op + " " + code, nil
		}
		return fmt.Sprintf("%s %s:%s", op, code, strings.Join(kvs, ",")), nil

	case "RMV":
		if stmt.RmvInstanceIndex != nil {
			return fmt.Sprintf("RMV %s:Index=%d", code, *stmt.RmvInstanceIndex), nil
		}
		return "RMV " + code, nil
	}
	return "", fmt.Errorf("RenderStatement: unreachable op %q", op)
}

// RenderStatements 把多条 statement 渲染为完整 MML 字符串（; 分隔 + 末尾 ;）。
// 任一条渲染失败立即返错（不部分渲染）。
func RenderStatements(stmts []StatementWithSubFields) (string, error) {
	if len(stmts) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(stmts))
	for i, sm := range stmts {
		part, err := RenderStatement(sm.Statement, sm.SubFields)
		if err != nil {
			return "", fmt.Errorf("statement[%d]: %w", i, err)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ";") + ";", nil
}

// ============================================================
// 内部 helper
// ============================================================

// collectSelectedMMLCodes 把 selectedIDs 映射到 mml_code 列表，按 subFields 的
// sort_order 排序。selectedIDs 中未在 subFields 命中的 ID 被忽略。
//
// 用 sort.SliceStable 保证同 sort_order 内按 mml_code 字典序稳定排序（diff 友好）。
func collectSelectedMMLCodes(selectedIDs []uuid.UUID, subFields []MMLCommandSubField) []string {
	if len(selectedIDs) == 0 || len(subFields) == 0 {
		return nil
	}
	selSet := make(map[uuid.UUID]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		selSet[id] = struct{}{}
	}

	// 过滤命中的 sub_field 副本（不破坏外部入参）
	hits := make([]MMLCommandSubField, 0, len(selectedIDs))
	for _, sf := range subFields {
		if _, ok := selSet[sf.ID]; ok {
			hits = append(hits, sf)
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].SortOrder != hits[j].SortOrder {
			return hits[i].SortOrder < hits[j].SortOrder
		}
		return hits[i].MMLCode < hits[j].MMLCode
	})

	out := make([]string, len(hits))
	for i, sf := range hits {
		out[i] = sf.MMLCode
	}
	return out
}

// collectKeyValuePairs 把 Values map 按 subFields 的 sort_order 排序为
// `Key=Value` 切片；Values 中未在 subFields 命中的 key 按字典序追加在末尾。
//
// 值含特殊字符时双引号包裹。
func collectKeyValuePairs(values map[string]string, subFields []MMLCommandSubField) []string {
	if len(values) == 0 {
		return nil
	}

	// 1. subFields 顺序优先
	pairs := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	// subFields 已经在 repo 层按 sort_order 排序；这里复刻一次保证幂等
	sorted := make([]MMLCommandSubField, len(subFields))
	copy(sorted, subFields)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].MMLCode < sorted[j].MMLCode
	})

	for _, sf := range sorted {
		if v, ok := values[sf.MMLCode]; ok {
			pairs = append(pairs, sf.MMLCode+"="+quoteValueIfNeeded(v))
			seen[sf.MMLCode] = struct{}{}
		}
	}

	// 2. Values 中未命中 subFields 的 key（unknown_codes）按字典序追加
	unknownKeys := make([]string, 0)
	for k := range values {
		if _, ok := seen[k]; !ok {
			unknownKeys = append(unknownKeys, k)
		}
	}
	sort.Strings(unknownKeys)
	for _, k := range unknownKeys {
		pairs = append(pairs, k+"="+quoteValueIfNeeded(values[k]))
	}

	return pairs
}

// quoteValueIfNeeded 值含 MML 特殊字符（, ; : = { } 空格 双引号）时用双引号包裹。
// 内部双引号转义为 \"。
func quoteValueIfNeeded(v string) string {
	if !strings.ContainsAny(v, `,;:={} "`) {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
}

// isValidOp 校验 op 是否在 4 类 MML 操作中。
func isValidOp(op string) bool {
	switch op {
	case "LST", "MOD", "ADD", "RMV":
		return true
	}
	return false
}
