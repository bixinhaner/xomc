package mml

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/global"
)

// ============================================================
// console_structured.go — R-9.2 结构化执行入口适配层
//
// 方案：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.6.1
//
// 与既有 `POST /mml/execute-statements`（MML 文本 round-trip）的关系：
//   - 旧通道：前端渲染 MML 文本 → 后端 parse → Statement.SelectedSubFieldIDs (id) + Values (key=mml_code)
//   - 新通道（本文件）：前端直接传 standardPath + value，省去 round-trip
//
// 设计取舍（Approach A）：
//   - 新通道在 handler 入口把 StructuredStatement 适配为既有 Statement
//   - 后续 fanout / sequencer / translator / RMV 多实例展开全部复用
//   - 旧通道保留 1 release 兼容期；ScriptTask（多语句脚本）继续走 MML 文本通道
//
// 反向映射核心：
//   - standardPath → sub_field：cmd.Params 提供 (ParamID → Tr069Path)，
//     sub_field.ParamID 提供 (sub_field → ParamID)；两者拼出 (Tr069Path → sub_field)
//   - values key 由 standardPath 翻译回 sub_field.MMLCode（与旧 Statement.Values 键空间对齐）
//   - 任一 path 未命中 sub_field 集 → ErrUnknownPaths 汇总返 422
// ============================================================

// StructuredStatement 是 R-9.2 结构化通道单条 statement 入参。
//
// 与 Statement 的字段差异：
//   - Paths      ：用户直接选 standardPath，无需先去 sub_field id
//   - Values key ：standardPath（旧 Statement.Values key 为 MMLCode）
//   - InstanceSelectors：R-4 多层 {iα/iβ} 上层 `.{i}.` 替换
//   - RmvInstance：RMV 单实例（用户决策 2026-05-20：1 MML 命令 = 1 RPC，禁止多实例）
type StructuredStatement struct {
	CommandID         uuid.UUID         `json:"command_id" binding:"required"`
	OperationType     string            `json:"operation_type" binding:"required,oneof=LST MOD ADD RMV"`
	CommandCode       string            `json:"command_code,omitempty"`
	Paths             []string          `json:"paths,omitempty"`
	Values            map[string]string `json:"values,omitempty"`
	InstanceSelectors map[string]string `json:"instance_selectors,omitempty"`
	RmvInstance       *int              `json:"rmv_instance,omitempty"`
	InstanceIndices   []int             `json:"instance_indices,omitempty"`
}

// StructuredExecuteRequest 是 POST /mml/console/execute-statements-structured 的请求体。
type StructuredExecuteRequest struct {
	Statements  []StructuredStatement `json:"statements" binding:"required,min=1"`
	DeviceSNs   []string              `json:"device_sns" binding:"required,min=1"`
	TaskName    string                `json:"task_name"`
	Creator     string                `json:"creator"`
	Executor    string                `json:"executor"`
	ExecuteType ExecuteType           `json:"execute_type"`
}

// ErrUnknownPaths 由 StructuredToStatement 抛出，handler 据此返 422 含 unknown_paths 列表。
//
// 汇总语义：单次转换中所有未命中 sub_field 的 path（来自 Paths + Values keys）一次性返回，
// 避免前端逐个试错。
type ErrUnknownPaths struct {
	CommandID uuid.UUID
	Paths     []string
}

func (e *ErrUnknownPaths) Error() string {
	return fmt.Sprintf("mml: command %s unknown paths: %v (R-9.2)", e.CommandID, e.Paths)
}

// Code 让 handler 错误码映射器拿到 ErrCodeInvalidStatementPayload。
func (e *ErrUnknownPaths) Code() int { return global.ErrCodeInvalidStatementPayload }

// StructuredToStatement 把单条结构化入参翻译为既有 executor 可消费的 Statement。
//
// 流程：
//  1. 加载 cmd + sub_fields（含 cmd.Params 提供 Tr069Path 反向索引）
//  2. 构造 pathToSubField 索引
//  3. Paths → SelectedSubFieldIDs；未命中累入 unknown
//  4. Values（key=path）→ Values（key=MMLCode）；未命中累入 unknown
//  5. 任一 unknown 非空 → ErrUnknownPaths
//  6. InstanceSelectors 透传到 Statement；R-4 替换在 executor buildStatementCommandEntries 中执行
//
// 注意：LST 模式允许 Paths 为空（既有 executor 默认全选）；其他 op 由 executor 各自校验。
// path 匹配用 template 形（含 `.{i}.`）与 sub_field.Tr069Path 模板形精确比对 —
// **不在 adapter 做替换**，否则 sub_field lookup 会失败。
func (s *ConsoleService) StructuredToStatement(ctx context.Context, ss StructuredStatement) (Statement, error) {
	if ss.CommandID == uuid.Nil {
		return Statement{}, fmt.Errorf("structured statement: command_id required")
	}
	if ss.OperationType == "RMV" && len(ss.InstanceIndices) > 1 {
		return Statement{}, fmt.Errorf("structured RMV requires exactly one instance index, got %d", len(ss.InstanceIndices))
	}
	if ss.RmvInstance != nil && len(ss.InstanceIndices) > 0 {
		return Statement{}, fmt.Errorf("structured RMV instance fields are mutually exclusive")
	}
	rmvInstance := ss.RmvInstance
	if len(ss.InstanceIndices) == 1 {
		idx := ss.InstanceIndices[0]
		rmvInstance = &idx
	}

	cmd, err := s.commandRepo.GetByID(ctx, ss.CommandID)
	if err != nil {
		return Statement{}, fmt.Errorf("get command %s: %w", ss.CommandID, err)
	}
	if cmd == nil {
		return Statement{}, ErrCommandNotFound
	}
	// commandRepo.GetByID 不查 params 列，必须显式 enrichment 才能拿到
	// (ParamID → Tr069Path) 反查表，否则下方 buildPathToSubFieldIndex 全返空 → R-9.2 误报。
	if err := s.attachParams(ctx, cmd); err != nil {
		return Statement{}, err
	}

	subFields, err := s.subFieldRepo.ListByCommand(ctx, ss.CommandID)
	if err != nil {
		return Statement{}, fmt.Errorf("list sub_fields: %w", err)
	}

	pathToSF := buildPathToSubFieldIndex(subFields, cmd.Params)

	unknown := make([]string, 0)
	selectedIDs := make([]uuid.UUID, 0, len(ss.Paths))
	for _, p := range ss.Paths {
		if sf, ok := pathToSF[p]; ok {
			selectedIDs = append(selectedIDs, sf.ID)
		} else {
			unknown = append(unknown, p)
		}
	}

	values := make(map[string]string, len(ss.Values))
	for p, v := range ss.Values {
		if sf, ok := pathToSF[p]; ok {
			values[sf.MMLCode] = v
		} else {
			unknown = append(unknown, p)
		}
	}

	if len(unknown) > 0 {
		unknown = dedupSortedStrings(unknown)
		return Statement{}, &ErrUnknownPaths{CommandID: ss.CommandID, Paths: unknown}
	}

	logicalCode := cmd.LogicalCode
	if logicalCode == "" {
		logicalCode = deriveLogicalCodeFromCommandCode(cmd.CommandCode, cmd.OperationType)
	}

	cmdID := ss.CommandID
	return Statement{
		CommandID:           &cmdID,
		LogicalCode:         logicalCode,
		OperationType:       ss.OperationType,
		SelectedSubFieldIDs: selectedIDs,
		Values:              values,
		RmvInstanceIndex:    rmvInstance,
		InstanceSelectors:   ss.InstanceSelectors,
	}, nil
}

// buildPathToSubFieldIndex 构造 standardPath → sub_field 反向索引。
//
// 关系链（migration 000113 后语义澄清）：
//   - MMLParamRef.ID       == mml_command_sub_fields.id（来自 paramRefSelectExpr.csf.id）
//   - MMLCommandSubField.ID == mml_command_sub_fields.id
//   - MMLCommandSubField.ParamID == standard_params.id（字段名 soft alias，
//     migration 000113 列重命名为 standard_path_id 但结构体保留 ParamID）
//
// 历史 bug（commit pre-2026-05-22）：曾用 paramByID[sf.ParamID] 查 → 把
// standard_params.id 当 sub_field.id 查，索引 100% miss，全部 path 误报 R-9.2。
// 现在用 sub_field.id（sf.ID == MMLParamRef.ID）做 key — 两边都是 csf.id，对齐。
//
// 同一 path 在 command 内由 UNIQUE(command_id, standard_path_id) 保证唯一。
func buildPathToSubFieldIndex(subFields []MMLCommandSubField, params []MMLParamRef) map[string]MMLCommandSubField {
	paramBySubFieldID := make(map[uuid.UUID]MMLParamRef, len(params))
	for _, p := range params {
		paramBySubFieldID[p.ID] = p
	}
	idx := make(map[string]MMLCommandSubField, len(subFields))
	for _, sf := range subFields {
		ref, ok := paramBySubFieldID[sf.ID]
		if !ok || ref.Tr069Path == "" {
			continue
		}
		idx[ref.Tr069Path] = sf
	}
	return idx
}

// dedupSortedStrings 去重 + 升序，让错误体的 Paths 列表稳定可读。
func dedupSortedStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ExecuteStructured 把结构化入参整批转换后委托给 ExecuteStatements。
//
// 转换失败语义：任一 statement 翻译失败 → 整批拒绝（避免部分执行）；
// 错误返回首个失败的 ErrUnknownPaths / ErrInstanceSelectorsNotImplemented，
// 调用方 handler 据此返 422。
func (s *ConsoleService) ExecuteStructured(ctx context.Context, req StructuredExecuteRequest, taskCreator MMLTaskCreator) (*MMLTask, error) {
	stmts := make([]Statement, 0, len(req.Statements))
	for i, ss := range req.Statements {
		stmt, err := s.StructuredToStatement(ctx, ss)
		if err != nil {
			return nil, fmt.Errorf("statement[%d]: %w", i, err)
		}
		stmts = append(stmts, stmt)
	}

	legacy := ExecuteStatementsRequest{
		Statements:  stmts,
		DeviceSNs:   req.DeviceSNs,
		TaskName:    req.TaskName,
		Creator:     req.Creator,
		Executor:    req.Executor,
		ExecuteType: req.ExecuteType,
	}
	return s.ExecuteStatements(ctx, legacy, taskCreator)
}
