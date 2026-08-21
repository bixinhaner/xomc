package mml

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrInvalidRequest 标识 ExecuteStatements / BuildStatementCommands 检测到的
// 用户输入错误（空 statements / 空 devices / sub_field 缺失 / target_object 缺失 /
// 缺 instance index 等）。Handler 据此把 4xx 与 5xx 区分开：
//
//	errors.Is(err, ErrInvalidRequest) → 400 Bad Request
//	errors.Is(err, ErrCommandNotFound) → 404 Not Found
//	其余 → 500 Internal Server Error
var ErrInvalidRequest = errors.New("mml: invalid request")

// ============================================================
// console_executor.go — T-0123-P1 S3-D2 下
//
// 设计依据：docs/design/mml-restore-old-interaction-plan-20260514.md §N.5 / §N.10 W1
//
// 入口：ConsoleService.ExecuteStatements
// 输入：N 设备 × M statements（已 parse 解析完，含 CommandID / SelectedSubFieldIDs / Values / RmvInstanceIndex）
// 输出：1 个 *MMLTask（status=running 或 pending），Commands 数组长度 = len(Statements)
//
// 每条 statement → 1 个 commands[] entry → fanouter 扇出到 N 设备 × 1 device_task
// 多条 statement → sequentialMode=true，初次仅入队 cmd_idx=0 的 device_task，
//   Sequencer 在前一条完成后追加入队（实现 LST→MOD→ADD 严格序列）
//
// statement → commands[] entry 映射规则：
//   - LST: rpc_method=GetParameterValues
//          SelectedSubFieldIDs → 选中的 sub_field.tr069_path 合成 param_refs
//          SelectedSubFieldIDs 为空 → 默认全选（与老 OMC 默认勾选状态一致）
//   - MOD: rpc_method=SetParameterValues
//          param_refs = Values 命中的 sub_fields（ParamCode=MMLCode 便于 SPV builder 查表）
//          Values 全无命中时保留所有 sub_fields，兼容旧版手工 MML 文本
//          parameters = Values（map[string]string → map[string]interface{}）
//   - ADD: rpc_method=AddObject
//          parameters["object_name"] = command.TargetObject（必含尾点）
//          单 entry 路径（stmt.Values 空）：原 2026-05-20 决策"1 MML=1 RPC 不做复合"
//          仍保留；stmt.Values 透传 parameters 仅为审计。
//          复合路径（stmt.Values 非空，spec §R-4.3，2026-05-21）：buildStatementCommandEntries
//          追加第 2 行 SetParameterValues entry，新实例 path 含字面占位符 ".{NEW}."，
//          Sequencer 在 AddObject task 完成后从 result.instance_number 替换为具体数字。
//          **仍然 1 device_task = 1 RPC**（task 链复用 sequencer 现有按 cmd_idx 推进机制），
//          因此与原 "1 MML=1 RPC" 决策的核心约束（单 task 不并发 / 不复合 RPC）不冲突。
//   - RMV: rpc_method=DeleteObject
//          单实例：parameters["object_name"] = command.TargetObject + "{RmvInstanceIndex}."
//          用户决策 2026-05-20：RMV 严格单实例（R-7 多选已禁用）。
// ============================================================

// ExecuteStatementsRequest 是 POST /mml/execute-statements 的请求体。
type ExecuteStatementsRequest struct {
	Statements  []Statement `json:"statements" binding:"required,min=1"`
	DeviceSNs   []string    `json:"device_sns" binding:"required,min=1"`
	TaskName    string      `json:"task_name"`
	Creator     string      `json:"creator"`
	Executor    string      `json:"executor"`
	ExecuteType ExecuteType `json:"execute_type"` // 空 → ExecuteImmediate
}

// MMLTaskCreator 让 ConsoleService 把 *MMLTask 交给上游 Service 做持久化 +
// fanout 调度，避免 ConsoleService 直接依赖 fanouter / taskRepo。
//
// 消费者驱动小接口：*mml.Service 天然实现 CreateAndFanoutTask（在 service.go 加薄包装）。
type MMLTaskCreator interface {
	CreateAndFanoutTask(ctx context.Context, task *MMLTask, sequential bool) error
}

// ExecuteStatements 把 statements 编译为 MMLTask 并交给 MMLTaskCreator 持久化 + fanout。
//
// 行为：
//   - 多条 statement → sequentialMode=true，仅入队 cmd_idx=0 device_task
//   - 任一 statement 编译失败 → 整体拒绝（避免半成品 task 落库）
//   - taskCreator 未注入 → 返错（service.go 未 wire 完整时单测仍可走纯编译路径，见 BuildStatementCommands）
func (s *ConsoleService) ExecuteStatements(ctx context.Context, req ExecuteStatementsRequest, taskCreator MMLTaskCreator) (*MMLTask, error) {
	if len(req.Statements) == 0 {
		return nil, fmt.Errorf("%w: statements is empty", ErrInvalidRequest)
	}
	if len(req.DeviceSNs) == 0 {
		return nil, fmt.Errorf("%w: device_sns is empty", ErrInvalidRequest)
	}
	if taskCreator == nil {
		return nil, fmt.Errorf("ExecuteStatements: taskCreator not wired")
	}

	commands, err := s.BuildStatementCommands(ctx, req.Statements)
	if err != nil {
		return nil, fmt.Errorf("compile statements: %w", err)
	}

	executeType := req.ExecuteType
	if executeType == "" {
		executeType = ExecuteImmediate
	}

	taskName := req.TaskName
	if taskName == "" {
		taskName = fmt.Sprintf("MML console (%d statements × %d devices)",
			len(req.Statements), len(req.DeviceSNs))
	}

	task := &MMLTask{
		TaskName:     taskName,
		DeviceSNs:    req.DeviceSNs,
		Commands:     commands,
		Status:       TaskPending,
		Results:      []map[string]interface{}{},
		Creator:      req.Creator,
		Executor:     req.Executor,
		ExecuteType:  executeType,
		TotalDevices: len(req.DeviceSNs),
	}

	// 多 commands 走严格序列；单 command 默认并发（fanouter 默认行为）。
	// 按 commands 数量判定（而非 statements 数）— spec §R-4.3 单条 ADD with values
	// statement 会展开为 2 个 commands（AddObject + SetParameterValues），同样需 sequential
	// 让 Sequencer 在 AddObject 完成后才入队 SPV（以读取 result.instance_number 替换 .{NEW}.）。
	sequential := len(commands) > 1

	if err := taskCreator.CreateAndFanoutTask(ctx, task, sequential); err != nil {
		return nil, fmt.Errorf("create+fanout task: %w", err)
	}

	s.logger.Info("mml console executed statements",
		zap.String("task_id", task.ID.String()),
		zap.Int("statement_count", len(req.Statements)),
		zap.Int("device_count", len(req.DeviceSNs)),
		zap.Bool("sequential", sequential),
	)
	return task, nil
}

// BuildStatementCommands 把 statements 编译为 MMLTask.Commands 数组。
// 公开导出便于上层批量入口（脚本运行 / dry-run 预览）复用。
//
// 失败语义：任一 statement 编译失败 → 全体放弃；ParseError 已在 parser 阶段累加，
// 这里遇到的错误属于 lookup 命中后 sub_field / target_object 缺失等运行期问题。
//
// 1 statement → **1 或 2** commands[] entries：
//   - LST / MOD / RMV / ADD without values → 1 entry（与历史行为一致）
//   - ADD with values（spec §R-4.3 复合流程）→ 2 entries：AddObject + SetParameterValues with {NEW}
//     第 2 entry 的 Tr069Path 包含字面占位符 ".{NEW}."；Sequencer 在前一条
//     AddObject task 完成后从 result.instance_number 替换为具体数字再入队 device_task。
func (s *ConsoleService) BuildStatementCommands(ctx context.Context, stmts []Statement) ([]map[string]interface{}, error) {
	commands := make([]map[string]interface{}, 0, len(stmts))
	for i, stmt := range stmts {
		cmd, subFields, err := s.resolveStatement(ctx, stmt)
		if err != nil {
			return nil, fmt.Errorf("statement[%d]: %w", i, err)
		}
		entries, err := buildStatementCommandEntries(stmt, cmd, subFields)
		if err != nil {
			return nil, fmt.Errorf("statement[%d] (%s %s): %w: %s",
				i, stmt.OperationType, stmt.LogicalCode, ErrInvalidRequest, err)
		}
		commands = append(commands, entries...)
	}
	return commands, nil
}

// buildStatementCommandEntries 把单条 statement 编译为 1-2 个 commands[] entries。
//
// 历史 buildStatementCommandEntry 保留（单 entry 返回），新 wrapper 处理 spec §R-4.3
// "ADD 复合流程"：ADD with values 时追加第 2 行 SetParameterValues entry，第 2 行
// 的 sub_field path 中"新实例对应的 {i}"被替换为字面字符串 "{NEW}"，运行时由
// Sequencer 在前一条 AddObject task 完成后读 result.instance_number 替换。
func buildStatementCommandEntries(stmt Statement, cmd *MMLCommand, subFields []MMLCommandSubField) ([]map[string]interface{}, error) {
	base, err := buildStatementCommandEntry(stmt, cmd, subFields)
	if err != nil {
		return nil, err
	}
	op := strings.ToUpper(strings.TrimSpace(stmt.OperationType))
	if op == "" {
		op = strings.ToUpper(cmd.OperationType)
	}
	// R-4.3 复合：仅 ADD with values 触发第 2 行 SPV
	if op == "ADD" && len(stmt.Values) > 0 {
		spv, err := buildADDCompoundSpvEntry(stmt, cmd, subFields)
		if err != nil {
			return nil, fmt.Errorf("ADD compound SPV: %w", err)
		}
		// spv 可能为 nil（防御性：所有 sub_fields 都没在 stmt.Values 里）— 此时不追加
		if spv != nil {
			rollback := buildADDCompensationEntry(stmt, cmd)
			return []map[string]interface{}{base, spv, rollback}, nil
		}
	}
	// #196 复合：MOD with values 后自动追加 LST 回读，核实基站是否真的改成功。
	// SPV 响应按 TR069 规范不含参数值，是"KPI URL 等结果显示空"的根因；追加一条
	// GetParameterValues 把设备实际值读回。Sequencer 在 MOD task 完成后顺序入队本 LST
	// （prev=SPV / next=GPV 不触发 R-4.3 的 .{NEW}. 替换分支，无需 Sequencer 改动）。
	if op == "MOD" && len(stmt.Values) > 0 {
		lst, err := buildMODReadbackLSTEntry(stmt, cmd, subFields)
		if err != nil {
			return nil, fmt.Errorf("MOD readback LST: %w", err)
		}
		// lst 可能为 nil（防御性：stmt.Values 全无命中 sub_field）— 此时不追加
		if lst != nil {
			return []map[string]interface{}{base, lst}, nil
		}
	}
	return []map[string]interface{}{base}, nil
}

func buildADDCompensationEntry(stmt Statement, cmd *MMLCommand) map[string]interface{} {
	targetObject := strings.TrimSpace(cmd.TargetObject)
	if !strings.HasSuffix(targetObject, ".") {
		targetObject += "."
	}
	targetObject, _ = substituteInstanceSelectors(targetObject, stmt.InstanceSelectors)
	return map[string]interface{}{
		"command_code":      cmd.CommandCode,
		"operation_type":    "RMV",
		"command_id":        cmd.ID.String(),
		"logical_code":      stmt.LogicalCode,
		"rpc_method":        "DeleteObject",
		"parameters":        map[string]interface{}{"object_name": targetObject + "{NEW}."},
		"compound_phase":    "rollback_after_add",
		"compensation_only": true,
	}
}

// buildMODReadbackLSTEntry 构造 #196「MOD 后自动 LST 回读核实」的第 2 行 GetParameterValues entry。
//
// 只回读本条 MOD 实际下发（stmt.Values 命中）的 PATH，让前端拿到设备真实值核实是否改成功
// （根治 SPV 不回值导致的"结果显示空"）。与 ADD 复合不同：无新实例，**无 .{i}/.{NEW} 占位**，
// 沿用 MOD 自身的 instance_selectors 把外层 .{i}. 替换为具体实例号。
//
// 返回 nil 表示无可回读 PATH（stmt.Values 全无命中 sub_field）→ 不追加，仅保留 SPV。
func buildMODReadbackLSTEntry(stmt Statement, cmd *MMLCommand, subFields []MMLCommandSubField) (map[string]interface{}, error) {
	// 仅回读本次实际下发（stmt.Values 命中）的 sub_fields
	ids := make([]uuid.UUID, 0, len(stmt.Values))
	for _, sf := range subFields {
		if _, ok := stmt.Values[sf.MMLCode]; ok {
			ids = append(ids, sf.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	refs := buildLSTParamRefs(ids, subFields, cmd.Params)
	if len(refs) == 0 {
		return nil, nil
	}
	if err := applyInstanceSelectorsToRefs(refs, stmt.InstanceSelectors); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"command_code":   cmd.CommandCode,
		"operation_type": "LST", // GPV 语义；回读核实，audit 通过 compound_phase 关联识别
		"command_id":     cmd.ID.String(),
		"logical_code":   stmt.LogicalCode,
		"rpc_method":     "GetParameterValues",
		"param_refs":     refs,
		// 标记复合阶段，便于 Sequencer / 前端 / audit 识别这是 MOD 的回读 LST
		"compound_phase": "lst_after_mod",
	}, nil
}

// buildADDCompoundSpvEntry 构造 §R-4.3 复合流程的第 2 行 SetParameterValues entry。
//
// 路径占位符策略：
//   - sub_field.tr069_path 含 N+1 个 .{i}.（N 由 stmt.InstanceSelectors 提供，最后 1 个是新实例）
//   - applyInstanceSelectorsForADDCompound 把前 N 个替换为具体值、最后 1 个替换为 ".{NEW}."
//   - Sequencer 在前一条 AddObject task 完成后从 result.instance_number 替换 .{NEW}. → ".<n>."
//
// param_refs 仅纳入用户实际填了值的 sub_field（stmt.Values 的 keys），避免 SPV 携带空值。
// 返回 nil 表示 stmt.Values 全无命中 sub_field（不构造 SPV，仅 AddObject）。
func buildADDCompoundSpvEntry(stmt Statement, cmd *MMLCommand, subFields []MMLCommandSubField) (map[string]interface{}, error) {
	// 仅保留 stmt.Values 命中的 sub_fields
	filtered := make([]MMLCommandSubField, 0, len(stmt.Values))
	for _, sf := range subFields {
		if _, ok := stmt.Values[sf.MMLCode]; ok {
			filtered = append(filtered, sf)
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	refs := buildMODParamRefs(filtered, cmd.Params)
	for i := range refs {
		newPath, err := substituteInstanceSelectorsForADDCompound(refs[i].Tr069Path, stmt.InstanceSelectors)
		if err != nil {
			return nil, fmt.Errorf("param_ref %s: %w", refs[i].ParamCode, err)
		}
		refs[i].Tr069Path = newPath
	}

	// parameters 仅含命中的 keys（filtered 子集对应的 mml_code → value）
	params := make(map[string]interface{}, len(filtered))
	for _, sf := range filtered {
		params[sf.MMLCode] = stmt.Values[sf.MMLCode]
	}

	return map[string]interface{}{
		"command_code":   cmd.CommandCode,
		"operation_type": "MOD", // SPV 语义；audit 通过 prev task (AddObject) 关联识别
		"command_id":     cmd.ID.String(),
		"logical_code":   stmt.LogicalCode,
		"rpc_method":     "SetParameterValues",
		"param_refs":     refs,
		"parameters":     params,
		// 标记复合阶段，便于 Sequencer 识别 + audit log 可读
		"compound_phase": "spv_after_add",
	}, nil
}

// substituteInstanceSelectorsForADDCompound 把路径中 N 个外层 .{i} 占位替换为 selectors 值，
// 同时把最后 1 个 .{i} 占位替换为字面占位符 .{NEW}（新实例号占位）。
//
// 约束：path 必须比 selectors 多正好 1 个 .{i} 占位（额外那 1 个 = 新实例）。
// 占位既可能是中间段 .{i}.，也可能是末尾对象段 .{i}。
//
// 示例：
//
//	path = "Device.Services.FAPService.{i}.PLMNList.{i}.PLMNID"
//	selectors = {iα: "1"}     →    path 有 2 个 .{i} 占位，selectors 1 个 → expected
//	result = "Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID"
func substituteInstanceSelectorsForADDCompound(path string, selectors map[string]string) (string, error) {
	placeholderCount := strings.Count(path, ".{i}")
	expected := len(selectors) + 1
	if placeholderCount != expected {
		return "", fmt.Errorf("%w: ADD compound path .{i} placeholder count=%d, expected selectors+1=%d (path=%s)",
			ErrInvalidRequest, placeholderCount, expected, path)
	}

	keys := make([]string, 0, len(selectors))
	for k := range selectors {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := path
	for _, key := range keys {
		val := selectors[key]
		idx := strings.Index(result, ".{i}")
		if idx < 0 {
			break
		}
		markerLength := len(".{i}")
		trailingDot := ""
		if idx+markerLength < len(result) && result[idx+markerLength] == '.' {
			markerLength++
			trailingDot = "."
		}
		result = result[:idx] + "." + val + trailingDot + result[idx+markerLength:]
	}

	// 此时应该正好剩 1 个 .{i}（防御性校验）
	if strings.Count(result, ".{i}") != 1 {
		return "", fmt.Errorf("ADD compound: post-substitution .{i} placeholder count=%d (result=%s)",
			strings.Count(result, ".{i}"), result)
	}
	idx := strings.Index(result, ".{i}")
	markerLength := len(".{i}")
	trailingDot := ""
	if idx+markerLength < len(result) && result[idx+markerLength] == '.' {
		markerLength++
		trailingDot = "."
	}
	return result[:idx] + ".{NEW}" + trailingDot + result[idx+markerLength:], nil
}

// resolveStatement 加载 statement 对应的 MMLCommand + sub_fields。
//   - stmt.CommandID 优先（前端从命令树选定命令时已携带）
//   - 否则按 (op, logical_code) 复用 LookupByLogicalCode
func (s *ConsoleService) resolveStatement(ctx context.Context, stmt Statement) (*MMLCommand, []MMLCommandSubField, error) {
	if stmt.CommandID != nil {
		cmd, err := s.commandRepo.GetByID(ctx, *stmt.CommandID)
		if err != nil {
			return nil, nil, fmt.Errorf("get command %s: %w", *stmt.CommandID, err)
		}
		if cmd == nil {
			return nil, nil, ErrCommandNotFound
		}
		// 见 attachParams 文档：buildLSTParamRefs/buildMODParamRefs 依赖 cmd.Params
		// 提供 Tr069Path；缺失会让 BuildTR069Params 拿到空 path 失败。
		if err := s.attachParams(ctx, cmd); err != nil {
			return nil, nil, err
		}
		subFields, err := s.subFieldRepo.ListByCommand(ctx, cmd.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list sub_fields: %w", err)
		}
		return cmd, subFields, nil
	}
	return s.LookupByLogicalCode(ctx, stmt.OperationType, stmt.LogicalCode)
}

// buildStatementCommandEntries 把单条 statement 编译为 fanouter 可消费的 commands[] entries。
//
// 1 statement → 1 entry（用户决策 2026-05-20：RMV 单实例，1 MML 命令 = 1 RPC）。
//
// entry shape（与 fanout.buildDeviceTaskRequests 期望一致）：
//
//	{
//	  "command_code":    "LST_DEVICE_INFO",
//	  "rpc_method":      "GetParameterValues",
//	  "operation_type":  "LST",
//	  "param_refs":      [...],                  // []MMLParamRef
//	  "parameters":      {...},                  // map[string]interface{}
//	  "command_id":      "uuid",                 // 审计/日志可读性
//	  "logical_code":    "DEVICE_INFO",
//	}
func buildStatementCommandEntry(stmt Statement, cmd *MMLCommand, subFields []MMLCommandSubField) (map[string]interface{}, error) {
	op := strings.ToUpper(strings.TrimSpace(stmt.OperationType))
	if op == "" {
		op = strings.ToUpper(cmd.OperationType)
	}

	entry := map[string]interface{}{
		"command_code":   cmd.CommandCode,
		"operation_type": op,
		"command_id":     cmd.ID.String(),
		"logical_code":   stmt.LogicalCode,
	}

	switch op {
	case "LST":
		refs := buildLSTParamRefs(stmt.SelectedSubFieldIDs, subFields, cmd.Params)
		if len(refs) == 0 {
			return nil, fmt.Errorf("LST: no usable sub_fields (selected=%d, total=%d)",
				len(stmt.SelectedSubFieldIDs), len(subFields))
		}
		// 查询实例留空时，多个叶子会折叠成同一个对象前缀。保留折叠前的标准路径，
		// 供任务历史重建结果列时只展示用户实际勾选的叶子，而不是对象下全部参数。
		selectedStandardPaths := paramRefTR069Paths(refs)
		if len(selectedStandardPaths) > 0 {
			entry["selected_standard_paths"] = selectedStandardPaths
		}
		applyQueryInstanceSelectorsToRefs(refs, stmt.InstanceSelectors)
		entry["rpc_method"] = "GetParameterValues"
		entry["param_refs"] = refs

	case "MOD":
		if len(stmt.Values) == 0 {
			// 历史触发场景：① 前端未勾选任何 sub_field 直接提交；② 调用方按
			// access_type 过滤可写字段时漏写枚举值（如只匹配 'RW' 漏了
			// 'READ_WRITE'）。Hint 提示常见排查点。
			return nil, fmt.Errorf("MOD: empty values — 请确认至少勾选一个 READ_WRITE 字段并填入值；若调用 API 时按 access_type 过滤可写字段，注意使用完整字面值 'READ_WRITE'/'READ_ONLY'，而非缩写 'RW'/'RO'")
		}
		selectedSubFields := make([]MMLCommandSubField, 0, len(stmt.Values))
		for _, sf := range subFields {
			if _, selected := stmt.Values[sf.MMLCode]; selected {
				selectedSubFields = append(selectedSubFields, sf)
			}
		}
		// 旧版手工 MML 文本允许解析未知 mml_code，并通过 UnknownCodes 提示调用方。
		// 当全部 Values 均未知时，保持改动前的 refs 形状；结构化页面请求会更早在
		// StructuredToStatement 以 ErrUnknownPaths 拒绝未知 Path，不会进入此回退。
		if len(selectedSubFields) == 0 {
			selectedSubFields = subFields
		}
		refs := buildMODParamRefs(selectedSubFields, cmd.Params)
		if err := applyInstanceSelectorsToRefs(refs, stmt.InstanceSelectors); err != nil {
			return nil, err
		}
		params := stringMapToInterface(stmt.Values)
		entry["rpc_method"] = "SetParameterValues"
		entry["param_refs"] = refs
		entry["parameters"] = params

	case "ADD":
		targetObject := strings.TrimSpace(cmd.TargetObject)
		if targetObject == "" {
			return nil, fmt.Errorf("ADD: command %s missing target_object", cmd.CommandCode)
		}
		if !strings.HasSuffix(targetObject, ".") {
			targetObject += "."
		}
		// R-4：ADD 的 targetObject 是父级对象路径，可能含上层 `.{i}.` 占位符；
		// instance_selectors 把这些占位符替换为具体实例号，新对象创建在指定层级下。
		substitutedObject, err := substituteInstanceSelectors(targetObject, stmt.InstanceSelectors)
		if err != nil {
			return nil, fmt.Errorf("ADD: target_object: %w", err)
		}
		targetObject = substitutedObject
		params := map[string]interface{}{"object_name": targetObject}
		// SPV 字段值随 parameters 透传保留审计痕迹（mml_audit_logs 记录用户完整意图）。
		// SOAP 层 BuildTR069Params(AddObject) 只用 object_name，其它键不污染 wire。
		for k, v := range stmt.Values {
			if k == "object_name" {
				continue // 避免覆盖
			}
			params[k] = v
		}
		entry["rpc_method"] = "AddObject"
		entry["parameters"] = params

	case "RMV":
		targetObject := strings.TrimSpace(cmd.TargetObject)
		if targetObject == "" {
			return nil, fmt.Errorf("RMV: command %s missing target_object", cmd.CommandCode)
		}
		// R-4：RMV 的 targetObject 是父对象路径；上层 `.{i}.` 由 selectors 替换。
		// 用户决策 2026-05-20：RMV 保持单实例，不支持多选。
		substitutedObject, err := substituteInstanceSelectors(targetObject, stmt.InstanceSelectors)
		if err != nil {
			return nil, fmt.Errorf("RMV: target_object: %w", err)
		}
		targetObject = substitutedObject
		if stmt.RmvInstanceIndex == nil {
			return nil, fmt.Errorf("RMV: missing instance index")
		}
		base := strings.TrimSuffix(targetObject, ".")
		entry["rpc_method"] = "DeleteObject"
		entry["parameters"] = map[string]interface{}{
			"object_name": fmt.Sprintf("%s.%d.", base, *stmt.RmvInstanceIndex),
		}

	default:
		return nil, fmt.Errorf("unsupported operation type %q", op)
	}
	return entry, nil
}

// substituteInstanceSelectors 把路径中 `.{i}.` 占位符按 selectors 替换为具体实例号。
//
// 约定：
//   - selector key 仅作 UX 标签（plan §6.6.1：iα/iβ/iγ）；实际位置由 **字典序左到右**映射
//     到路径的 `.{i}.` 出现顺序
//   - selectors 数量必须严格 == 路径中 `.{i}.` 计数，否则 ErrInvalidRequest
//   - 空 selectors + 路径无 `.{i}.` → 原样返回（兼容老路径无 instance 的场景）
//
// 示例：
//
//	path = "Device.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion"
//	selectors = {iα: "1", iβ: "2"}  → 排序后 keys=[iα, iβ]
//	result = "Device.DeviceInfo.MU.1.Slot.2.3GPPSpecVersion"
func substituteInstanceSelectors(path string, selectors map[string]string) (string, error) {
	placeholderCount := strings.Count(path, ".{i}.")
	if placeholderCount == 0 && len(selectors) == 0 {
		return path, nil
	}
	if placeholderCount != len(selectors) {
		return "", fmt.Errorf("%w: instance_selectors count mismatch (path has %d .{i}. placeholders, selectors=%d)",
			ErrInvalidRequest, placeholderCount, len(selectors))
	}

	keys := make([]string, 0, len(selectors))
	for k := range selectors {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := path
	for _, key := range keys {
		val := selectors[key]
		idx := strings.Index(result, ".{i}.")
		if idx < 0 {
			break // 防御性 — 上面计数已校验
		}
		// `.{i}.` 共 5 字符；replacement 保留外侧的两个 `.`，中间替换为 val
		result = result[:idx] + "." + val + "." + result[idx+5:]
	}
	return result, nil
}

func substituteQueryInstanceSelectors(path string, selectors map[string]string) string {
	if len(selectors) == 0 {
		return path
	}

	// 查询路径既可能把实例占位符放在中间（.{i}.），也可能把对象实例放在末级（.{i}）。
	// 末级对象占位符在最后一层 selector 为空时应归一化为对象集合路径（末尾保留 .）。
	placeholderCount := strings.Count(path, ".{i}")
	layerValues := make([]string, placeholderCount)
	layerBound := make([]bool, placeholderCount)
	legacyKeys := make([]string, 0, len(selectors))
	for key, value := range selectors {
		layer, numbered := queryInstanceSelectorLayer(key)
		if !numbered {
			legacyKeys = append(legacyKeys, key)
			continue
		}
		if layer >= 1 && layer <= placeholderCount {
			layerValues[layer-1] = value
			layerBound[layer-1] = true
		}
	}

	sort.Strings(legacyKeys)
	nextLayer := 0
	for _, key := range legacyKeys {
		for nextLayer < placeholderCount && layerBound[nextLayer] {
			nextLayer++
		}
		if nextLayer >= placeholderCount {
			break
		}
		layerValues[nextLayer] = strings.TrimSpace(selectors[key])
		layerBound[nextLayer] = true
		nextLayer++
	}

	result := path
	for layer := 0; layer < placeholderCount; layer++ {
		idx := strings.Index(result, ".{i}")
		if !layerBound[layer] || strings.TrimSpace(layerValues[layer]) == "" {
			return result[:idx+1]
		}
		hasTrailingDot := idx+4 < len(result) && result[idx+4] == '.'
		markerLength := 4
		trailingDot := ""
		if hasTrailingDot {
			markerLength = 5
			trailingDot = "."
		}
		result = result[:idx] + "." + layerValues[layer] + trailingDot + result[idx+markerLength:]
	}
	return result
}

func queryInstanceSelectorLayer(key string) (int, bool) {
	if len(key) != 3 || key[0] != 'i' || key[1] < '0' || key[1] > '9' || key[2] < '0' || key[2] > '9' {
		return 0, false
	}
	return int(key[1]-'0')*10 + int(key[2]-'0'), true
}

// buildLSTParamRefs 把 LST 选中的 sub_field 合成为 BuildTR069Params 可消费的 param_refs。
//
//   - SelectedSubFieldIDs 非空 → 仅选中那部分
//   - SelectedSubFieldIDs 为空 → 默认全 sub_fields（与老 OMC 默认勾选状态一致）
//
// 每个 sub_field 通过 cmd.Params（mml_command_sub_fields JOIN standard_params）的 ParamID
// 对应一条 MMLParamRef；若 cmd.Params 未挂载（lookup 路径未触发 attachParamRefs），
// 退化为合成最小集（tr069_path 从 enriched 列单独查更彻底，但 P1 范围内 cmd.Params
// 在 ListByCommand 已能拿到 enriched 数据，这里走基本 sub_field.MMLCode → 空 Tr069Path
// 兜底；调用方 BuildTR069Params 会过滤空 path 返 ErrNoUsableParams）。
func buildLSTParamRefs(selected []uuid.UUID, subFields []MMLCommandSubField, cmdParams []MMLParamRef) []MMLParamRef {
	paramByID := indexParamsByID(cmdParams)

	pick := subFields
	if len(selected) > 0 {
		selSet := make(map[uuid.UUID]struct{}, len(selected))
		for _, id := range selected {
			selSet[id] = struct{}{}
		}
		pick = pick[:0:0]
		for _, sf := range subFields {
			if _, ok := selSet[sf.ID]; ok {
				pick = append(pick, sf)
			}
		}
	}

	refs := make([]MMLParamRef, 0, len(pick))
	for _, sf := range pick {
		// 修复 2026-05-22：用 sf.ID 而非 sf.ParamID 查 paramByID。
		// MMLParamRef.ID 来自 csf.id（sub_field.id），sf.ID 同源；旧版误用
		// sf.ParamID（migration 000113 起语义 = standard_params.id）查 → 100% miss。
		if ref, ok := paramByID[sf.ID]; ok {
			refs = append(refs, ref)
			continue
		}
		// 退化兜底：cmd.Params 未挂载或对应 ID 缺失，合成最小 ref
		refs = append(refs, MMLParamRef{
			ID:        sf.ID,
			ParamCode: sf.MMLCode,
		})
	}
	return refs
}

// buildMODParamRefs 为 MOD 命令合成 param_refs：纳入调用方选定的 sub_fields，
// 让 BuildTR069Params.buildParameterValues 能按 ParamCode（=MMLCode）索引 Tr069Path。
//
// ParamCode 改写为 sub_field.MMLCode（命令上下文的 code），与 parser 解出的 Values key 对齐。
func buildMODParamRefs(subFields []MMLCommandSubField, cmdParams []MMLParamRef) []MMLParamRef {
	paramByID := indexParamsByID(cmdParams)
	refs := make([]MMLParamRef, 0, len(subFields))
	for _, sf := range subFields {
		// 修复 2026-05-22：见 buildLSTParamRefs 同名注释，用 sf.ID 而非 sf.ParamID。
		ref, ok := paramByID[sf.ID]
		if !ok {
			ref = MMLParamRef{ID: sf.ID}
		}
		// 用 MMLCode 覆盖 ParamCode，使 buildParameterValues 能按 Values key 找到 ref
		ref.ParamCode = sf.MMLCode
		refs = append(refs, ref)
	}
	return refs
}

func indexParamsByID(refs []MMLParamRef) map[uuid.UUID]MMLParamRef {
	out := make(map[uuid.UUID]MMLParamRef, len(refs))
	for _, ref := range refs {
		out[ref.ID] = ref
	}
	return out
}

func paramRefTR069Paths(refs []MMLParamRef) []string {
	paths := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		path := strings.TrimSpace(ref.Tr069Path)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func stringMapToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// applyInstanceSelectorsToRefs 把 selectors 应用到 LST/MOD param_refs 的 Tr069Path。
//
// 行为：
//   - selectors 为空 → no-op（含路径仍带 `.{i}.` 的场景）
//   - 任一 ref 的路径替换失败 → 整体放弃
//
// 注意：refs 是 slice of struct value（不是指针）；为安全 in-place 更新，遍历 idx 写回。
func applyInstanceSelectorsToRefs(refs []MMLParamRef, selectors map[string]string) error {
	if len(selectors) == 0 {
		return nil
	}
	for i := range refs {
		newPath, err := substituteInstanceSelectors(refs[i].Tr069Path, selectors)
		if err != nil {
			return fmt.Errorf("param_ref %s: %w", refs[i].ParamCode, err)
		}
		refs[i].Tr069Path = newPath
	}
	return nil
}

func applyQueryInstanceSelectorsToRefs(
	refs []MMLParamRef,
	selectors map[string]string,
) {
	if len(selectors) == 0 {
		return
	}
	for i := range refs {
		refs[i].Tr069Path = substituteQueryInstanceSelectors(
			refs[i].Tr069Path,
			selectors,
		)
	}
}

// 用户决策 2026-05-20 二次澄清（仍生效，核心约束）：
//   - 1 device_task = 1 RPC，严格遵守，**单 task 不复合 RPC**
//   - ACS 端零业务编排
//
// spec §R-4.3 复合流程（2026-05-21 实施）：
//   - 仍然 1 device_task = 1 RPC（不破核心约束）
//   - 但 1 MML statement 可以展开为 2 个 commands[] entries（AddObject + SPV）
//     由 Sequencer 在 OnTaskCompleted(AddObject) 后从 result.instance_number 提取
//     新实例号，替换第 2 个 entry 中 path 的 .{NEW}. 占位符，再入队第 2 个 device_task
//   - 实现见 buildStatementCommandEntries（本文件下方）+ Sequencer.OnTaskCompleted
//     新增的 substituteNewInstance 钩子
//   - 等价于用户手写多行 MML "ADD X; MOD X.{NEW}.Foo=bar"，区别仅是占位符与
//     instance_number 之间的运行时替换由 Sequencer 自动完成
//
// 单 entry 路径（ADD with empty values）：保留 stmt.Values audit-transfer 到
// entry["parameters"]，BuildTR069Params(AddObject) 仅消费 object_name，其它键忽略。
