package mml

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// ResultAggregator updates MML task statistics when device_tasks complete.
// P1 扩展：当 mml_task.script_id != nil 时，把最终 status 回写到 mml_scripts 的
// last_run_status / last_run_at 字段，供脚本列表/详情页展示"最近一次执行态"。
//
// T-0174 扩展：subFieldRepo 注入后，OnTaskCompleted 在收到 SetParameterValues
// 失败且 dt.Result.param_faults[].fault_code == 9005 (AttributeIdNotFound) 时，
// 自动把对应 standardPath 的 sub_field 标 is_supported=false —— UI 后续永久
// 隐藏该 path，避免下一个用户继续触发同样的 9003 失败。subFieldRepo 为 nil 时
// 跳过 auto-learn，保持向后兼容（单测 / 无 mml 依赖的场景）。
type ResultAggregator struct {
	taskRepo     TaskRepository
	scriptRepo   ScriptRepository
	subFieldRepo SubFieldRepository
	hub          SSEPublisher
	logger       *zap.Logger
}

// NewResultAggregator creates a new ResultAggregator. scriptRepo / subFieldRepo 可为 nil——
// 单测或不需要回写 mml_scripts / 不需要 auto-learn 的场景下传 nil，相应逻辑跳过。
func NewResultAggregator(
	taskRepo TaskRepository,
	scriptRepo ScriptRepository,
	subFieldRepo SubFieldRepository,
	hub SSEPublisher,
	logger *zap.Logger,
) *ResultAggregator {
	return &ResultAggregator{
		taskRepo:     taskRepo,
		scriptRepo:   scriptRepo,
		subFieldRepo: subFieldRepo,
		hub:          hub,
		logger:       logger.Named("mml-result-aggregator"),
	}
}

// OnTaskCompleted is called when a device_task reaches a terminal state.
func (a *ResultAggregator) OnTaskCompleted(ctx context.Context, dt *task.Task) {
	sourceID := dt.SourceID
	if sourceID == "" {
		return
	}

	mmlID, err := uuid.Parse(sourceID)
	if err != nil {
		a.logger.Error("parse source_id", zap.Error(err))
		return
	}

	successDelta, failedDelta := 0, 0
	switch dt.Status {
	case task.TaskStatusCompleted:
		successDelta = 1
	case task.TaskStatusFailed, task.TaskStatusExpired:
		failedDelta = 1
	default:
		return
	}

	if err := a.taskRepo.IncrementStats(ctx, mmlID, successDelta, failedDelta); err != nil {
		a.logger.Error("increment mml task stats", zap.Error(err))
		return
	}

	// T-0102-d: emit a per-device "frame" SSE event so the UI sees each
	// device's output as it arrives, not just the aggregate completion.
	// Multi-frame semantics: one mml_device_frame per device_task terminal
	// state; a 50-device task produces 50 frames over time.
	a.publishDeviceFrame(ctx, mmlID, dt)

	// T-0174 auto-learn: SetParameterValues 失败 + dt.Result.param_faults[9005]
	// → 把对应 sub_field 标 is_supported=false。下一次同一命令进 UI 时该 path
	// 被过滤，避免反复触发 atomic SPV 9003 失败把同批次其它 path 一起拖死。
	if dt.Status == task.TaskStatusFailed {
		a.autoLearnUnsupportedPaths(ctx, dt)
	}

	a.finalizeIfComplete(ctx, mmlID)
}

// spvFaultRecord 与 ACS handler.SPVFault 形态对齐（device_tasks.result.param_faults[]）。
type spvFaultRecord struct {
	ParameterName string `json:"parameter_name"`
	FaultCode     int    `json:"fault_code"`
	FaultString   string `json:"fault_string"`
}

type spvFaultPayload struct {
	ParamFaults []spvFaultRecord `json:"param_faults"`
}

// CWMP fault code 9005 = "Invalid parameter name"。Baicells / 部分厂商描述为
// "AttributeIdNotFound"。该 code 才能确定"path 在 CPE 数据模型中不存在"，从而
// 安全地 auto-mark is_supported=false；其它 code（9007 值越界、9008 只读）含义不同
// 不能映射到不支持，跳过即可。
const cwmpFaultInvalidParameterName = 9005

// autoLearnUnsupportedPaths 解析 dt.Result.param_faults，对 code=9005 的 path
// 调 subFieldRepo.MarkUnsupportedByStandardPath。subFieldRepo 为 nil（未注入）
// 或 dt.Result 不是 SPV fault 形态时 silent skip。
func (a *ResultAggregator) autoLearnUnsupportedPaths(ctx context.Context, dt *task.Task) {
	if a.subFieldRepo == nil {
		return
	}
	if dt.Method != "SetParameterValues" || len(dt.Result) == 0 {
		return
	}
	var payload spvFaultPayload
	if err := json.Unmarshal(dt.Result, &payload); err != nil {
		// dt.Result 是 ACS 写入的 {"param_faults":[...]}；解析失败说明 ACS 没写
		// 或写错了形态（如旧版本仍调 MarkTaskFailed 不带 result），不报错跳过。
		return
	}
	if len(payload.ParamFaults) == 0 {
		return
	}
	learned := 0
	for _, f := range payload.ParamFaults {
		if f.FaultCode != cwmpFaultInvalidParameterName || f.ParameterName == "" {
			continue
		}
		affected, err := a.subFieldRepo.MarkUnsupportedByStandardPath(ctx, f.ParameterName)
		if err != nil {
			a.logger.Warn("auto-learn mark sub_field unsupported failed",
				zap.String("device_sn", dt.DeviceSN),
				zap.String("path", f.ParameterName),
				zap.Error(err))
			continue
		}
		if affected > 0 {
			learned++
			a.logger.Info("auto-learned unsupported path",
				zap.String("device_sn", dt.DeviceSN),
				zap.String("device_task_id", dt.ID),
				zap.String("path", f.ParameterName),
				zap.Int("fault_code", f.FaultCode),
				zap.Int64("sub_fields_updated", affected),
			)
		}
	}
	if learned > 0 {
		a.logger.Info("auto-learn pass done",
			zap.String("device_task_id", dt.ID),
			zap.Int("paths_learned", learned),
		)
	}
}

// publishDeviceFrame emits one SSE frame per device_task terminal state.
// The frame carries the device's per-RPC outcome (status + result/error)
// so the UI can render a live "device × step" stream. Failures during
// the lookup/marshal phase are logged but never fail the caller — SSE
// is best-effort fan-out, not a correctness boundary.
func (a *ResultAggregator) publishDeviceFrame(ctx context.Context, mmlID uuid.UUID, dt *task.Task) {
	if a.hub == nil {
		return
	}
	mmlTask, err := a.taskRepo.GetByID(ctx, mmlID)
	if err != nil || mmlTask == nil || mmlTask.Executor == "" {
		return
	}
	payload := map[string]interface{}{
		"task_id":        mmlID.String(),
		"device_task_id": dt.ID,
		"device_sn":      dt.DeviceSN,
		"method":         dt.Method,
		"status":         string(dt.Status),
		"command_index":  dt.CommandIndex,
		"device_index":   dt.DeviceIndex,
	}
	if len(dt.Result) > 0 {
		payload["result"] = json.RawMessage(dt.Result)
	}
	if dt.ErrorMessage != "" {
		payload["error_message"] = dt.ErrorMessage
	}
	if dt.CompletedAt != nil {
		payload["completed_at"] = dt.CompletedAt
	}
	data, err := json.Marshal(payload)
	if err != nil {
		a.logger.Warn("marshal mml device frame", zap.Error(err))
		return
	}
	a.hub.PublishSimple(mmlTask.Executor, "mml_device_frame", data)
}

// finalizeIfComplete transitions the MML task to completed/failed when all sub-tasks finish.
func (a *ResultAggregator) finalizeIfComplete(ctx context.Context, mmlID uuid.UUID) {
	mmlTask, err := a.taskRepo.GetByID(ctx, mmlID)
	if err != nil || mmlTask == nil {
		return
	}

	total := mmlTask.TotalDevices * len(mmlTask.Commands)
	done := mmlTask.SuccessCount + mmlTask.FailedCount
	if done < total {
		return
	}

	var finalStatus TaskStatus
	var finalResult TaskResult
	if mmlTask.FailedCount == 0 {
		finalStatus = TaskCompleted
		finalResult = ResultSuccess
	} else if mmlTask.SuccessCount == 0 {
		finalStatus = TaskFailed
		finalResult = ResultFailed
	} else {
		finalStatus = TaskCompleted
		finalResult = ResultPartial
	}

	now := time.Now()
	mmlTask.Status = finalStatus
	mmlTask.Result = &finalResult
	mmlTask.FinishedAt = &now
	if err := a.taskRepo.Update(ctx, mmlTask); err != nil {
		a.logger.Error("finalize mml task", zap.Error(err))
		return
	}

	a.logger.Info("mml task finalized",
		zap.String("mml_task_id", mmlID.String()),
		zap.String("status", string(finalStatus)),
		zap.String("result", string(finalResult)),
		zap.Int("success", mmlTask.SuccessCount),
		zap.Int("failed", mmlTask.FailedCount),
	)

	// P1 扩展：若本任务关联脚本（script_id != nil），回写
	// mml_scripts.last_run_status / last_run_at。每次执行详情由 mml_tasks 独立
	// 持有；脚本层只存"最近一次"指针，前端脚本详情页用来快速显示。
	a.updateScriptLastRunIfNeeded(ctx, mmlTask, finalResult, now)

	if a.hub != nil && mmlTask.Executor != "" {
		data, _ := json.Marshal(map[string]interface{}{
			"task_id":       mmlTask.ID.String(),
			"status":        string(mmlTask.Status),
			"result":        string(*mmlTask.Result),
			"success_count": mmlTask.SuccessCount,
			"failed_count":  mmlTask.FailedCount,
		})
		a.hub.PublishSimple(mmlTask.Executor, "mml_task_completed", data)
	}
}

// updateScriptLastRunIfNeeded 当 mml_task 来自脚本时，回写 mml_scripts 指针列。
// scriptRepo 为 nil（单测 / 无脚本场景）或 task.ScriptID 为 nil 时 no-op。
func (a *ResultAggregator) updateScriptLastRunIfNeeded(
	ctx context.Context,
	mmlTask *MMLTask,
	result TaskResult,
	at time.Time,
) {
	if a.scriptRepo == nil || mmlTask.ScriptID == nil {
		return
	}
	if err := a.scriptRepo.UpdateLastRun(ctx, *mmlTask.ScriptID, string(result), at); err != nil {
		a.logger.Warn("update mml_script last_run",
			zap.String("script_id", mmlTask.ScriptID.String()),
			zap.String("mml_task_id", mmlTask.ID.String()),
			zap.String("result", string(result)),
			zap.Error(err))
		return
	}
	a.logger.Debug("mml_script last_run updated",
		zap.String("script_id", mmlTask.ScriptID.String()),
		zap.String("result", string(result)))
}
