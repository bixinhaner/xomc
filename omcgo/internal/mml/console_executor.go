package mml

import (
	"context"
	"errors"
	"fmt"
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
//          param_refs = 所有 sub_fields（ParamCode=MMLCode 便于 SPV builder 查表）
//          parameters = Values（map[string]string → map[string]interface{}）
//   - ADD: rpc_method=AddObject
//          parameters["object_name"] = command.TargetObject（必含尾点）
//          W1 决议（A）：单 device_task 入口；SPV 字段值随 parameters 透传保留审计痕迹，
//          AddObject 完成后的 SPV 链由后续迭代补齐（P1 不实现）
//   - RMV: rpc_method=DeleteObject
//          parameters["object_name"] = command.TargetObject + "{RmvInstanceIndex}."
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

	// 多 statement 走严格序列；单 statement 默认并发（fanouter 默认行为）。
	sequential := len(req.Statements) > 1

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
func (s *ConsoleService) BuildStatementCommands(ctx context.Context, stmts []Statement) ([]map[string]interface{}, error) {
	commands := make([]map[string]interface{}, 0, len(stmts))
	for i, stmt := range stmts {
		cmd, subFields, err := s.resolveStatement(ctx, stmt)
		if err != nil {
			return nil, fmt.Errorf("statement[%d]: %w", i, err)
		}
		entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
		if err != nil {
			return nil, fmt.Errorf("statement[%d] (%s %s): %w: %s",
				i, stmt.OperationType, stmt.LogicalCode, ErrInvalidRequest, err)
		}
		commands = append(commands, entry)
	}
	return commands, nil
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
		subFields, err := s.subFieldRepo.ListByCommand(ctx, cmd.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list sub_fields: %w", err)
		}
		return cmd, subFields, nil
	}
	return s.LookupByLogicalCode(ctx, stmt.OperationType, stmt.LogicalCode)
}

// buildStatementCommandEntry 把单条 statement 编译为 fanouter 可消费的 commands[] entry。
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
		entry["rpc_method"] = "GetParameterValues"
		entry["param_refs"] = refs

	case "MOD":
		if len(stmt.Values) == 0 {
			return nil, fmt.Errorf("MOD: empty values")
		}
		refs := buildMODParamRefs(subFields, cmd.Params)
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
		params := map[string]interface{}{"object_name": targetObject}
		// W1 决议（A）：AddObject 协议层只用 object_name；SPV 字段值此处随 parameters
		// 透传，目的是让 service.writeAuditLogs 在 mml_audit_logs 留下用户**完整意图**
		//（哪些字段想设到新实例上），便于运维追溯。Fanout 调
		// BuildTR069Params(AddObject) → buildObjectName 时这些 key 会被忽略，
		// 不会污染 SOAP body。SPV 链（AddObject 完成 → 取新实例号 → SPV）
		// 由后续迭代补齐（P1 范围外）。
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
		if stmt.RmvInstanceIndex == nil {
			return nil, fmt.Errorf("RMV: missing instance index")
		}
		base := strings.TrimSuffix(targetObject, ".")
		instance := fmt.Sprintf("%s.%d.", base, *stmt.RmvInstanceIndex)
		entry["rpc_method"] = "DeleteObject"
		entry["parameters"] = map[string]interface{}{"object_name": instance}

	default:
		return nil, fmt.Errorf("unsupported operation type %q", op)
	}

	return entry, nil
}

// buildLSTParamRefs 把 LST 选中的 sub_field 合成为 BuildTR069Params 可消费的 param_refs。
//
//   - SelectedSubFieldIDs 非空 → 仅选中那部分
//   - SelectedSubFieldIDs 为空 → 默认全 sub_fields（与老 OMC 默认勾选状态一致）
//
// 每个 sub_field 通过 cmd.Params（mml_command_param_refs JOIN mml_params）的 ParamID
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
		if ref, ok := paramByID[sf.ParamID]; ok {
			refs = append(refs, ref)
			continue
		}
		// 退化兜底：cmd.Params 未挂载或对应 ParamID 缺失，合成最小 ref
		refs = append(refs, MMLParamRef{
			ID:        sf.ParamID,
			ParamCode: sf.MMLCode,
		})
	}
	return refs
}

// buildMODParamRefs 为 MOD 命令合成 param_refs：所有 sub_fields 都纳入，
// 让 BuildTR069Params.buildParameterValues 能按 ParamCode（=MMLCode）索引 Tr069Path。
//
// ParamCode 改写为 sub_field.MMLCode（命令上下文的 code），与 parser 解出的 Values key 对齐。
func buildMODParamRefs(subFields []MMLCommandSubField, cmdParams []MMLParamRef) []MMLParamRef {
	paramByID := indexParamsByID(cmdParams)
	refs := make([]MMLParamRef, 0, len(subFields))
	for _, sf := range subFields {
		ref, ok := paramByID[sf.ParamID]
		if !ok {
			ref = MMLParamRef{ID: sf.ParamID}
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

func stringMapToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
