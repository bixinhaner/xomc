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
// 失败且 dt.Result.param_faults[].fault_code == 9005 (Invalid parameter name) 时，
// 自动把对应 standardPath 标 is_supported=false —— UI 后续永久隐藏该 path，避免
// 下一个用户继续触发同样的 9003 失败。subFieldRepo 为 nil 时跳过 auto-learn，
// 保持向后兼容（单测 / 无 mml 依赖的场景）。
//
// T-0176-PR-E：auto-learn 的写真值源从 mml_command_sub_fields 迁到 param_mappings，
// 因此需要先把 dt.DeviceSN → paramModelID 解析出来再调 MarkUnsupportedByStandardPath。
// paramModelResolver 闭包由 provider 装配（与 ConsoleService.SetParamModelByDeviceResolver
// 同源逻辑：devices LEFT JOIN products → effective param_model_id）；resolver 未注入
// 或返 nil/err 时整个 auto-learn 跳过，孤儿设备 / 无 paramModel 不污染数据。
type ResultAggregator struct {
	taskRepo     TaskRepository
	scriptRepo   ScriptRepository
	subFieldRepo SubFieldRepository
	hub          SSEPublisher
	logger       *zap.Logger

	// T-0176-PR-E: deviceSN/UUID → paramModelID 反查闭包。注入后
	// autoLearnUnsupportedPaths 在写 param_mappings 之前解析 paramModelID。
	// nil → silent skip 整个 auto-learn。
	paramModelResolver func(ctx context.Context, deviceKey string) (*uuid.UUID, error)

	// unsupportedRepo + productIDResolver：按设备 product_id 记录「path 不支持」到
	// product_unsupported_paths（自学习表，读/写分别标记），前端「选择命令 / 配置参数」据此过滤。
	// 与 autoLearnUnsupportedPaths（param_mappings.is_supported，按 param_model）互补——
	// 本表按 product_id 维度，避免多产品共用 param_model 时互相污染。任一为 nil → 跳过。
	unsupportedRepo   ProductUnsupportedPathRepository
	productIDResolver func(ctx context.Context, deviceSN string) (*uuid.UUID, error)

	// deviceStats 注入后，finalizeIfComplete 以"该任务实际派发的 device_tasks
	// 全部进入终态"为完成判据（而非理论 total=设备数×命令数）。nil（单测）时退回
	// 旧的理论 total 计数判据，保持向后兼容。详见 finalizeIfComplete。
	deviceStats DeviceTaskStatsReader
}

// DeviceTaskStatsReader 按 (source, source_id) 聚合 device_tasks 终态。
// 由 *task.PgTaskRepository 实现（消费者侧小接口，避免 mml 硬依赖其构造器）。
type DeviceTaskStatsReader interface {
	AggregateStatusBySourceID(ctx context.Context, source task.TaskSource, sourceID string) (task.DeviceTaskSourceStats, error)
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

// SetParamModelResolver 注入 deviceSN/UUID → paramModelID 反查闭包（T-0176-PR-E）。
// 不注入时 autoLearnUnsupportedPaths silent skip。注入风格与 PR-C
// ConsoleService.SetParamModelByDeviceResolver 对齐——不改 NewResultAggregator
// 构造签名避免现有测试大面积调整。
func (a *ResultAggregator) SetParamModelResolver(
	fn func(ctx context.Context, deviceKey string) (*uuid.UUID, error),
) {
	a.paramModelResolver = fn
}

// SetUnsupportedPathRecorder 注入「按 product_id 记录 path 不支持」的仓库 + deviceSN→product_id
// 解析闭包。两者均注入后，OnTaskCompleted 在 path 不支持类故障时写入 product_unsupported_paths。
func (a *ResultAggregator) SetUnsupportedPathRecorder(
	repo ProductUnsupportedPathRepository,
	productIDResolver func(ctx context.Context, deviceSN string) (*uuid.UUID, error),
) {
	a.unsupportedRepo = repo
	a.productIDResolver = productIDResolver
}

// SetDeviceTaskStatsReader 注入 device_tasks 终态聚合器（生产由 *task.PgTaskRepository
// 提供）。注入后 finalizeIfComplete 以实际 device_tasks 终态为完成判据；不注入则退回
// 理论 total 计数判据（单测向后兼容）。
func (a *ResultAggregator) SetDeviceTaskStatsReader(r DeviceTaskStatsReader) {
	a.deviceStats = r
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
		a.recordProductUnsupportedPaths(ctx, dt)
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
// 调 subFieldRepo.MarkUnsupportedByStandardPath 把 param_mappings 行标
// is_supported=false。subFieldRepo 为 nil（未注入）、paramModelResolver 未注入、
// dt.Result 不是 SPV fault 形态、resolver 返 nil/err 时均 silent skip。
//
// T-0176-PR-E：写入真值源已从 mml_command_sub_fields 迁到 param_mappings，因此
// 在调 MarkUnsupportedByStandardPath 之前先用 dt.DeviceSN 解析 paramModelID。
func (a *ResultAggregator) autoLearnUnsupportedPaths(ctx context.Context, dt *task.Task) {
	if a.subFieldRepo == nil || a.paramModelResolver == nil {
		return
	}
	if dt.Method != "SetParameterValues" || len(dt.Result) == 0 {
		return
	}

	// T-0176-PR-E: 先解析 paramModelID；孤儿设备 / 无 paramModel / err → silent skip。
	pmID, err := a.paramModelResolver(ctx, dt.DeviceSN)
	if err != nil {
		a.logger.Debug("skip auto-learn: cannot resolve paramModel",
			zap.String("device_sn", dt.DeviceSN), zap.Error(err))
		return
	}
	if pmID == nil {
		a.logger.Debug("skip auto-learn: device has no paramModel",
			zap.String("device_sn", dt.DeviceSN))
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
		affected, err := a.subFieldRepo.MarkUnsupportedByStandardPath(ctx, *pmID, f.ParameterName)
		if err != nil {
			a.logger.Warn("auto-learn mark param_mapping unsupported failed",
				zap.String("device_sn", dt.DeviceSN),
				zap.String("param_model_id", pmID.String()),
				zap.String("path", f.ParameterName),
				zap.Error(err))
			continue
		}
		if affected > 0 {
			learned++
			a.logger.Info("auto-learned unsupported path",
				zap.String("device_sn", dt.DeviceSN),
				zap.String("param_model_id", pmID.String()),
				zap.String("device_task_id", dt.ID),
				zap.String("path", f.ParameterName),
				zap.Int("fault_code", f.FaultCode),
				zap.Int64("mappings_updated", affected),
			)
		}
	}
	if learned > 0 {
		a.logger.Info("auto-learn pass done",
			zap.String("device_task_id", dt.ID),
			zap.String("param_model_id", pmID.String()),
			zap.Int("paths_learned", learned),
		)
	}
}

// recordProductUnsupportedPaths 分析 dt.Result.param_faults 的「path 不支持」类故障，按设备
// product_id 写入 product_unsupported_paths（自学习表，读/写分别标记）。与 autoLearnUnsupportedPaths
// 不同：覆盖所有方法（GPV/LST + SPV/MOD）、按 product_id 维度（不污染共用 param_model 的其它产品）。
//
// 故障 → 读/写标记（用户决策：读写权限控制）：
//   - 9005 Invalid Parameter Name（path 不存在）   → read_unsupported=true AND write_unsupported=true
//   - 9008 Non-writable Parameter（参数只读）       → write_unsupported=true（读仍可用）
//   - 其它 fault code（9006 类型错 / 9007 值越界等） → 非「不支持」语义，跳过
//
// unsupportedRepo / productIDResolver 任一未注入、dt.Result 非 param_faults 形态、解析不到
// product_id 时均 silent skip。
func (a *ResultAggregator) recordProductUnsupportedPaths(ctx context.Context, dt *task.Task) {
	if a.unsupportedRepo == nil || a.productIDResolver == nil || len(dt.Result) == 0 {
		return
	}

	var payload spvFaultPayload
	if err := json.Unmarshal(dt.Result, &payload); err != nil || len(payload.ParamFaults) == 0 {
		return
	}
	// 先扫一遍是否含「不支持」类故障，避免无谓的 product_id 解析查询。
	hasUnsupported := false
	for _, f := range payload.ParamFaults {
		if f.ParameterName != "" && faultMarksUnsupported(f.FaultCode) {
			hasUnsupported = true
			break
		}
	}
	if !hasUnsupported {
		return
	}

	productID, err := a.productIDResolver(ctx, dt.DeviceSN)
	if err != nil || productID == nil {
		a.logger.Debug("skip record unsupported path: cannot resolve product_id",
			zap.String("device_sn", dt.DeviceSN), zap.Error(err))
		return
	}

	recorded := 0
	for _, f := range payload.ParamFaults {
		if f.ParameterName == "" {
			continue
		}
		readUnsup, writeUnsup := faultToReadWriteUnsupported(f.FaultCode)
		if !readUnsup && !writeUnsup {
			continue
		}
		if err := a.unsupportedRepo.Record(ctx, *productID, f.ParameterName, readUnsup, writeUnsup, f.FaultCode, dt.DeviceSN); err != nil {
			a.logger.Warn("record product unsupported path failed",
				zap.String("product_id", productID.String()),
				zap.String("path", f.ParameterName),
				zap.Error(err))
			continue
		}
		recorded++
	}
	if recorded > 0 {
		a.logger.Info("recorded product unsupported paths",
			zap.String("device_sn", dt.DeviceSN),
			zap.String("product_id", productID.String()),
			zap.String("device_task_id", dt.ID),
			zap.Int("paths_recorded", recorded),
		)
	}
}

// cwmpFaultNonWritableParameter 9008 = "Attempt to set a non-writable parameter"（参数只读）。
// 仅影响写（MOD/SPV），读（LST/GPV）仍可用。
const cwmpFaultNonWritableParameter = 9008

// faultMarksUnsupported 判断 fault code 是否表示「path 不支持」（9005 不存在 / 9008 只读）。
func faultMarksUnsupported(faultCode int) bool {
	return faultCode == cwmpFaultInvalidParameterName || faultCode == cwmpFaultNonWritableParameter
}

// faultToReadWriteUnsupported 把 fault code 映射为读/写不支持标记。
func faultToReadWriteUnsupported(faultCode int) (readUnsupported, writeUnsupported bool) {
	switch faultCode {
	case cwmpFaultInvalidParameterName: // 9005 path 不存在 → 读写都不支持
		return true, true
	case cwmpFaultNonWritableParameter: // 9008 只读 → 仅写不支持
		return false, true
	default:
		return false, false
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
	if dt.SentAt != nil {
		payload["sent_at"] = dt.SentAt // RPC 下发时间，供前端结果行「下发时间」展示
	}
	data, err := json.Marshal(payload)
	if err != nil {
		a.logger.Warn("marshal mml device frame", zap.Error(err))
		return
	}
	a.hub.PublishSimple(mmlTask.Executor, "mml_device_frame", data)
}

// finalizeIfComplete transitions the MML task to completed/failed when all sub-tasks finish.
//
// 完成判据（deviceStats 注入时，生产路径）：以该任务【实际派发的 device_tasks 全部
// 进入终态】为准，而非理论 total=设备数×命令数。后者在 fanout（BuildTR069Params 失败
// continue 跳过某设备）或 sequencer（buildNextRequest 返回 nil 时仅虚拟推进、不真正派发）
// 跳过任意 (设备,命令) 时会让 done 永远到不了 total，导致任务永久卡在 running —— 即
// "查看到执行结果成功、状态却一直执行中"的根因。
//
// 顺序模式正确性依赖回调顺序：Sequencer 必须在本聚合器【之前】注册（见 provider
// modules.go），这样某行完成时 Sequencer 先把下一行 device_task 入队（Active>0），
// 本函数再判定时才不会在链路中途误判为已完成。
//
// deviceStats 未注入（单测）时退回旧的理论 total 计数判据，保持向后兼容。
func (a *ResultAggregator) finalizeIfComplete(ctx context.Context, mmlID uuid.UUID) {
	mmlTask, err := a.taskRepo.GetByID(ctx, mmlID)
	if err != nil || mmlTask == nil {
		return
	}
	// 已是终态：幂等返回，避免重复 finalize 覆盖已写结果。
	switch mmlTask.Status {
	case TaskCompleted, TaskFailed, TaskCancelled:
		return
	}

	var successCnt, failedCnt int
	if a.deviceStats != nil {
		stats, err := a.deviceStats.AggregateStatusBySourceID(ctx, task.TaskSourceMML, mmlID.String())
		if err != nil {
			a.logger.Error("aggregate device_task stats for finalize", zap.Error(err))
			return
		}
		// 还没派发任何 device_task（fanout 进行中）或仍有在途 → 等下次完成事件。
		if stats.Total == 0 || stats.Active > 0 {
			return
		}
		successCnt, failedCnt = stats.Completed, stats.Failed
	} else {
		// 向后兼容：未注入 deviceStats 时沿用理论 total 计数判据（保持单测行为）。
		total := mmlTask.TotalDevices * len(mmlTask.Commands)
		done := mmlTask.SuccessCount + mmlTask.FailedCount
		if done < total {
			return
		}
		successCnt, failedCnt = mmlTask.SuccessCount, mmlTask.FailedCount
	}

	var finalStatus TaskStatus
	var finalResult TaskResult
	if failedCnt == 0 {
		finalStatus = TaskCompleted
		finalResult = ResultSuccess
	} else if successCnt == 0 {
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
	// 用真实 device_tasks 统计覆盖计数器列，消除事件丢失/重复导致的漂移。
	mmlTask.SuccessCount = successCnt
	mmlTask.FailedCount = failedCnt
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
