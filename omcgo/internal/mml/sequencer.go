package mml

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// Sequencer 实现脚本多行严格序列执行（Sprint B Q-V3-3 决议）。
//
// 工作原理：
//
//  1. Fanout 阶段只 enqueue **每个设备的 command_index=0** 的 device_task
//     （未启用 Sequencer 时仍 enqueue 全部，保持向后兼容）
//  2. 当 device_task 完成（terminal status），Sequencer 收到
//     CompletionRouter dispatch 的 OnTaskCompleted 回调
//  3. Sequencer 查 mml_task.commands[command_index+1]，构造下一行的 device_task
//     enqueue 到该设备的队列
//  4. 直到 command_index >= len(commands)-1，本设备链结束
//
// 跨设备并行：设备 A 与设备 B 各自跑各自的 commands 链，互不阻塞。
// 但同一设备内部 line N+1 严格等 line N 终态后才发，符合 §6.5 要求。
//
// 这是 "fanout 内 channel 串行" Q-V3-2 决议的轻量实现 — 不动 internal/task
// schema（无 depends_on 列），完全靠 completion callback 驱动。
type Sequencer struct {
	taskRepo TaskRepository // 查 MMLTask.commands 数组
	enqueuer task.Enqueuer  // 入队下一行 device_task
	fanouter *Fanouter      // 复用 buildDeviceTaskRequests 单条命令翻译逻辑
	logger   *zap.Logger
}

// NewSequencer 构造 Sequencer。所有依赖必须非 nil。
func NewSequencer(taskRepo TaskRepository, enqueuer task.Enqueuer, fanouter *Fanouter, logger *zap.Logger) *Sequencer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Sequencer{
		taskRepo: taskRepo,
		enqueuer: enqueuer,
		fanouter: fanouter,
		logger:   logger.Named("mml-sequencer"),
	}
}

// OnTaskCompleted 实现 task.TaskCompletionCallback。
//
// 触发时机：device_task 进入终态（completed / failed / expired / cancelled）。
// 行为：查上一行 source_id 对应的 mml_task，找下一行 command 入队。
//
// 关键不变量：
//   - 失败也驱动下一行 — Q-V3-3 "严格序列"语义保留：即便上一步失败，
//     下一步仍要跑（用户视角的"我把脚本跑完，每条结果都看到"）。
//     如要"失败即停"语义，下一版加 `mml_scripts.fail_fast` 开关再说。
//   - cancelled 不驱动 — 用户主动取消脚本时不应继续推后续命令。
func (s *Sequencer) OnTaskCompleted(ctx context.Context, t *task.Task) {
	if t == nil || t.Source != task.TaskSourceMML {
		return
	}
	if t.Status == task.TaskStatusCancelled {
		s.logger.Debug("sequencer skip: task cancelled",
			zap.String("device_task_id", t.ID))
		return
	}

	mmlTaskID, err := uuid.Parse(t.SourceID)
	if err != nil {
		// 老 device_task source_id 可能不是 UUID（如 ProvisioningTask）；
		// MML 任务的 source_id 必为 mml_task.ID，解析失败说明该任务不归 Sequencer 管。
		return
	}

	mmlTask, err := s.taskRepo.GetByID(ctx, mmlTaskID)
	if err != nil || mmlTask == nil {
		s.logger.Warn("sequencer: load mml_task failed",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.Error(err))
		return
	}

	nextIdx := nextCommandIndexForDevice(mmlTask, t.DeviceSN, t.CommandIndex)
	if nextIdx < 0 {
		s.logger.Debug("sequencer: device chain finished",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("completed_cmd_idx", t.CommandIndex),
			zap.Int("total_commands", len(mmlTask.Commands)),
		)
		return
	}

	if nextIdx >= len(mmlTask.Commands) {
		s.logger.Debug("sequencer: device chain finished",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("completed_cmd_idx", t.CommandIndex),
			zap.Int("total_commands", len(mmlTask.Commands)),
		)
		return
	}

	// ADD compound compensation is conditional: a successful SPV must not
	// delete the newly configured instance. Mark the compensation command as a
	// virtual success so a later statement in the same script can still run.
	if isSpvMethod(t.Method) && isAddRollbackEntry(mmlTask.Commands[nextIdx]) && t.Status == task.TaskStatusCompleted {
		s.continueAfterCompensationSkip(ctx, mmlTask, t, nextIdx)
		return
	}

	// 构造下一行 device_task — 复用 fanouter 的单 command 翻译能力。
	// R-4.3 复合：prevTask 传入，buildNextRequest 据此识别 AddObject→SPV 链路并替换 .{NEW}.
	nextReq, err := s.buildNextRequest(ctx, mmlTask, t.DeviceSN, t.DeviceIndex, nextIdx, t)
	if err != nil {
		s.logger.Warn("sequencer: build next request failed",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("next_cmd_idx", nextIdx),
			zap.Error(err))
		s.failAndContinueSkippedCommand(ctx, mmlTask, t, nextIdx, err.Error())
		return
	}
	if nextReq == nil {
		// build 跳过本命令（如 BuildTR069Params 失败 / 全路径不合规）
		// 视为该 line 失败，继续推后续 line（递归触发自身）
		s.logger.Info("sequencer: next command skipped (build returned nil); recursing",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("skipped_cmd_idx", nextIdx),
		)
		s.failAndContinueSkippedCommand(ctx, mmlTask, t, nextIdx, "MML command could not be converted to a device task")
		return
	}

	created, err := s.enqueuer.CreateTask(ctx, nextReq)
	if err != nil {
		s.logger.Error("sequencer: enqueue next command failed",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("next_cmd_idx", nextIdx),
			zap.Error(err))
		return
	}
	s.logger.Info("sequencer: next command enqueued",
		zap.String("mml_task_id", mmlTaskID.String()),
		zap.String("device_sn", t.DeviceSN),
		zap.Int("next_cmd_idx", nextIdx),
		zap.String("next_device_task_id", created.ID),
		zap.String("method", nextReq.Method),
	)
}

func (s *Sequencer) failAndContinueSkippedCommand(ctx context.Context, mmlTask *MMLTask, prev *task.Task, skippedIdx int, reason string) {
	req := s.failedCommandRequest(mmlTask, prev, skippedIdx, reason)
	created, err := s.enqueuer.CreateTask(ctx, req)
	if err != nil {
		s.logger.Error("sequencer: record skipped command failure failed; falling back to virtual continuation",
			zap.String("mml_task_id", req.SourceID),
			zap.String("device_sn", req.DeviceSN),
			zap.Int("skipped_cmd_idx", skippedIdx),
			zap.Error(err),
		)
		s.continueAfterSkippedCommand(ctx, mmlTask, prev, skippedIdx)
		return
	}
	s.logger.Info("sequencer: skipped command recorded as failed",
		zap.String("mml_task_id", req.SourceID),
		zap.String("device_sn", req.DeviceSN),
		zap.Int("skipped_cmd_idx", skippedIdx),
		zap.String("device_task_id", created.ID),
		zap.String("method", req.Method),
	)
}

func (s *Sequencer) failedCommandRequest(mmlTask *MMLTask, prev *task.Task, cmdIdx int, reason string) *task.CreateTaskRequest {
	cmd := map[string]interface{}{}
	if mmlTask != nil && cmdIdx >= 0 && cmdIdx < len(mmlTask.Commands) {
		cmd = mmlTask.Commands[cmdIdx]
	}
	method := commandString(cmd, "rpc_method")
	if method == "" {
		method = "InvalidMMLCommand"
	}
	commandCode := commandString(cmd, "command_code")
	description := "MML command failed before enqueue"
	if commandCode != "" {
		description = fmt.Sprintf("MML %s", commandCode)
	}
	if mmlTask != nil && mmlTask.TaskName != "" {
		description = fmt.Sprintf("%s: %s", description, mmlTask.TaskName)
	}
	sourceID := prev.SourceID
	if sourceID == "" && mmlTask != nil {
		sourceID = mmlTask.ID.String()
	}
	creator := ""
	if mmlTask != nil {
		creator = mmlTask.Creator
	}
	maxRetries := 0
	payload, _ := json.Marshal(map[string]interface{}{
		"command":     cmd,
		"skip_reason": reason,
		"skipped":     true,
	})
	return &task.CreateTaskRequest{
		DeviceSN:    prev.DeviceSN,
		Method:      method,
		Params:      payload,
		Priority:    10,
		Source:      task.TaskSourceMML,
		CreatorID:   creator,
		Description: description,
		MaxRetries:  &maxRetries,

		SourceID:     sourceID,
		CommandIndex: cmdIdx,
		DeviceIndex:  prev.DeviceIndex,

		FailImmediately: true,
		FailReason:      reason,
	}
}

func (s *Sequencer) continueAfterSkippedCommand(ctx context.Context, mmlTask *MMLTask, prev *task.Task, skippedIdx int) {
	virtual := &task.Task{
		Source:       task.TaskSourceMML,
		SourceID:     prev.SourceID,
		DeviceSN:     prev.DeviceSN,
		DeviceIndex:  prev.DeviceIndex,
		CommandIndex: skippedIdx,
		Status:       task.TaskStatusFailed,
	}
	if skippedIdx >= 0 && skippedIdx < len(mmlTask.Commands) {
		virtual.Method = commandString(mmlTask.Commands[skippedIdx], "rpc_method")
	}
	// 模拟 skippedIdx 的"虚拟完成"事件，递归找 skippedIdx+1。
	s.OnTaskCompleted(ctx, virtual)
}

func (s *Sequencer) continueAfterCompensationSkip(ctx context.Context, mmlTask *MMLTask, prev *task.Task, skippedIdx int) {
	virtual := &task.Task{
		Source:       task.TaskSourceMML,
		SourceID:     prev.SourceID,
		DeviceSN:     prev.DeviceSN,
		DeviceIndex:  prev.DeviceIndex,
		CommandIndex: skippedIdx,
		Method:       "DeleteObject",
		Status:       task.TaskStatusCompleted,
	}
	s.OnTaskCompleted(ctx, virtual)
}

// buildNextRequest 用 fanouter 的逻辑构造单条 device_task 请求。
// 单命令单设备 → 返回单 request；不合规 / 翻译失败 → 返 nil（caller 跳过）。
//
// prevTask 可为 nil（递归虚拟完成事件路径）；非 nil 时用于 R-4.3 ADD 复合检测：
// prev=AddObject + next=SetParameterValues → 从 prev.Result.instance_number 提取
// 新实例号，把 next 命令中 param_refs[].Tr069Path 的 .{NEW}. 字面替换为 ".<n>."
// 再交给 fanouter；若 instance_number 缺失则 chain break（返回 err，caller 处理）。
func (s *Sequencer) buildNextRequest(ctx context.Context, mmlTask *MMLTask, deviceSN string, deviceIdx, cmdIdx int, prevTask *task.Task) (*task.CreateTaskRequest, error) {
	if cmdIdx < 0 || cmdIdx >= len(mmlTask.Commands) {
		return nil, fmt.Errorf("cmd_idx %d out of range [0, %d)", cmdIdx, len(mmlTask.Commands))
	}

	cmdEntry := mmlTask.Commands[cmdIdx]

	// R-4.3 复合流程：prev=AddObject + next=SetParameterValues → 替换 .{NEW}.
	if prevTask != nil && isAddObjectMethod(prevTask.Method) && isSpvCmdEntry(cmdEntry) {
		instanceNumber, ok := extractInstanceNumber(prevTask.Result)
		if !ok {
			return nil, fmt.Errorf("R-4.3 chain break: prev AddObject task %s missing instance_number in result", prevTask.ID)
		}
		cmdEntry = substituteNewInstance(cmdEntry, instanceNumber)
		s.logger.Info("sequencer: R-4.3 substituted .{NEW}. with instance_number",
			zap.String("mml_task_id", mmlTask.ID.String()),
			zap.String("device_sn", deviceSN),
			zap.Int("cmd_idx", cmdIdx),
			zap.Int("instance_number", instanceNumber),
		)
	}
	if prevTask != nil && isSpvMethod(prevTask.Method) && isAddRollbackEntry(cmdEntry) {
		objectName, ok := rollbackObjectName(prevTask.Params)
		if !ok {
			return nil, fmt.Errorf("ADD compensation chain break: SPV task %s missing rollback object", prevTask.ID)
		}
		cmdEntry = setObjectName(cmdEntry, objectName)
	}

	// 临时构造一个只含单 device + 单 command 的 MMLTask 视图，
	// 复用 fanouter.buildDeviceTaskRequests 的 BuildTR069Params 链路。
	view := &MMLTask{
		ID:                    mmlTask.ID,
		TaskName:              mmlTask.TaskName,
		Creator:               mmlTask.Creator,
		DeviceSNs:             []string{deviceSN},
		Commands:              []map[string]interface{}{cmdEntry},
		OfflineRetry:          mmlTask.OfflineRetry,
		OfflineRetryWait:      mmlTask.OfflineRetryWait,
		FailedRetry:           mmlTask.FailedRetry,
		FailedRetryCount:      mmlTask.FailedRetryCount,
		FailedRetryInterval:   mmlTask.FailedRetryInterval,
		PathTranslationSource: mmlTask.PathTranslationSource,
	}
	reqs := s.fanouter.buildDeviceTaskRequests(ctx, view)
	if len(reqs) == 0 {
		return nil, nil
	}
	// fix 索引：buildDeviceTaskRequests 把单视图当作 cmd_idx=0 / dev_idx=0；
	// 还原成原 mmlTask 中的真实索引，保证 result_aggregator / 后续 Sequencer 调用链正确。
	reqs[0].CommandIndex = cmdIdx
	reqs[0].DeviceIndex = deviceIdx
	if prevTask != nil && isAddObjectMethod(prevTask.Method) && isSpvCmdEntry(cmdEntry) {
		instanceNumber, _ := extractInstanceNumber(prevTask.Result)
		if objectName, ok := addInstanceObjectName(prevTask.Params, instanceNumber); ok {
			reqs[0].Params = withRollbackObjectName(reqs[0].Params, objectName)
		}
	}
	return reqs[0], nil
}

func nextCommandIndexForDevice(mmlTask *MMLTask, deviceSN string, currentIdx int) int {
	if mmlTask == nil {
		return -1
	}
	if mmlTask.ExecuteMode != TaskExecuteModeDeviceBound {
		nextIdx := currentIdx + 1
		if nextIdx >= len(mmlTask.Commands) {
			return -1
		}
		return nextIdx
	}

	currentOrder := planOrderForCommand(mmlTask, currentIdx)
	bestIdx := -1
	bestOrder := 0
	for idx := range mmlTask.Commands {
		if idx == currentIdx {
			continue
		}
		if planDeviceSNForCommand(mmlTask, idx) != deviceSN {
			continue
		}
		order := planOrderForCommand(mmlTask, idx)
		if order <= currentOrder {
			continue
		}
		if bestIdx == -1 || order < bestOrder || (order == bestOrder && idx < bestIdx) {
			bestIdx = idx
			bestOrder = order
		}
	}
	return bestIdx
}

func planDeviceSNForCommand(mmlTask *MMLTask, cmdIdx int) string {
	if cmdIdx >= 0 && cmdIdx < len(mmlTask.Commands) {
		if sn := commandString(mmlTask.Commands[cmdIdx], "plan_device_sn"); sn != "" {
			return sn
		}
	}
	if cmdIdx >= 0 && cmdIdx < len(mmlTask.PlanItems) {
		return strings.TrimSpace(mmlTask.PlanItems[cmdIdx].DeviceSN)
	}
	return ""
}

// isAddObjectMethod 判断 task.Method 是否为 AddObject（防御性，处理 SOAP 命名差异）。
func isAddObjectMethod(method string) bool {
	// CWMP 标准是 "AddObject"；fanouter / SOAP 层可能写成 "addObject" 之类
	return strings.EqualFold(strings.TrimSpace(method), "AddObject")
}

// isSpvCmdEntry 判断 cmd entry 是否为 SetParameterValues。
func isSpvCmdEntry(entry map[string]interface{}) bool {
	v, ok := entry["rpc_method"]
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(s), "SetParameterValues")
}

func isSpvMethod(method string) bool {
	return strings.EqualFold(strings.TrimSpace(method), "SetParameterValues")
}

func isAddRollbackEntry(entry map[string]interface{}) bool {
	return commandString(entry, "compound_phase") == "rollback_after_add"
}

const rollbackObjectNameKey = "_mml_rollback_object_name"

func addInstanceObjectName(params json.RawMessage, instanceNumber int) (string, bool) {
	if instanceNumber <= 0 {
		return "", false
	}
	var payload map[string]interface{}
	if json.Unmarshal(params, &payload) != nil {
		return "", false
	}
	base, _ := payload["object_name"].(string)
	base = strings.TrimSpace(base)
	if base == "" {
		return "", false
	}
	return strings.TrimSuffix(base, ".") + "." + strconv.Itoa(instanceNumber) + ".", true
}

func withRollbackObjectName(params json.RawMessage, objectName string) json.RawMessage {
	var payload map[string]interface{}
	if json.Unmarshal(params, &payload) != nil {
		return params
	}
	payload[rollbackObjectNameKey] = objectName
	out, err := json.Marshal(payload)
	if err != nil {
		return params
	}
	return out
}

func rollbackObjectName(params json.RawMessage) (string, bool) {
	var payload map[string]interface{}
	if json.Unmarshal(params, &payload) != nil {
		return "", false
	}
	name, _ := payload[rollbackObjectNameKey].(string)
	return name, strings.TrimSpace(name) != ""
}

func setObjectName(cmd map[string]interface{}, objectName string) map[string]interface{} {
	out := make(map[string]interface{}, len(cmd))
	for k, v := range cmd {
		out[k] = v
	}
	out["parameters"] = map[string]interface{}{"object_name": objectName}
	return out
}

// extractInstanceNumber 从 device_task.result JSON 解析 instance_number 整数。
//
// 期望形态（由 acs/handler.go MarkTaskCompleted 写入）：
//
//	{ "method": "AddObjectResponse", "raw_response": "...", "instance_number": 5 }
//
// 缺字段 / 非 JSON / 非整数 → (0, false)，调用方据此决定 chain break。
func extractInstanceNumber(resultJSON json.RawMessage) (int, bool) {
	if len(resultJSON) == 0 {
		return 0, false
	}
	var m map[string]interface{}
	if err := json.Unmarshal(resultJSON, &m); err != nil {
		return 0, false
	}
	raw, ok := m["instance_number"]
	if !ok {
		return 0, false
	}
	// JSON 数字默认解析为 float64
	switch n := raw.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case string:
		// 防御性：某些路径可能存为字符串
		if v, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return v, true
		}
	}
	return 0, false
}

// substituteNewInstance 深拷贝 cmd entry 并把 param_refs 中所有 Tr069Path 的
// 字面占位符 ".{NEW}." 替换为 "."+<n>+"."。
//
// 处理两种 param_refs 形态（兼容 in-memory 与 JSON-roundtrip）：
//   - []MMLParamRef                              (直接 in-memory，单测路径)
//   - []interface{} of map[string]interface{}    (DB JSONB 反序列化，生产路径)
//
// 其它字段浅拷贝。parameters 字段不动（spec §R-4.3 keys 是 mml_code，不含路径）。
func substituteNewInstance(cmd map[string]interface{}, instanceNumber int) map[string]interface{} {
	newToken := "." + strconv.Itoa(instanceNumber) + "."
	const placeholder = ".{NEW}."

	out := make(map[string]interface{}, len(cmd))
	for k, v := range cmd {
		if k != "param_refs" {
			out[k] = v
			continue
		}
		// 形态 1：[]MMLParamRef（in-memory）
		if typed, ok := v.([]MMLParamRef); ok {
			rewritten := make([]MMLParamRef, len(typed))
			for i, r := range typed {
				r.Tr069Path = strings.ReplaceAll(r.Tr069Path, placeholder, newToken)
				rewritten[i] = r
			}
			out[k] = rewritten
			continue
		}
		// 形态 2：[]interface{} of maps（JSON 反序列化后）
		if list, ok := v.([]interface{}); ok {
			rewritten := make([]interface{}, len(list))
			for i, item := range list {
				m, ok := item.(map[string]interface{})
				if !ok {
					rewritten[i] = item
					continue
				}
				cp := make(map[string]interface{}, len(m))
				for mk, mv := range m {
					if mk == "tr069_path" {
						if s, ok := mv.(string); ok {
							cp[mk] = strings.ReplaceAll(s, placeholder, newToken)
							continue
						}
					}
					cp[mk] = mv
				}
				rewritten[i] = cp
			}
			out[k] = rewritten
			continue
		}
		// 未知形态 — 原样保留（防御性）
		out[k] = v
	}
	return out
}
