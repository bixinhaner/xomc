package mml

import (
	"context"
	"fmt"

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
	taskRepo TaskRepository       // 查 MMLTask.commands 数组
	enqueuer task.Enqueuer        // 入队下一行 device_task
	fanouter *Fanouter            // 复用 buildDeviceTaskRequests 单条命令翻译逻辑
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

	nextIdx := t.CommandIndex + 1

	mmlTask, err := s.taskRepo.GetByID(ctx, mmlTaskID)
	if err != nil || mmlTask == nil {
		s.logger.Warn("sequencer: load mml_task failed",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.Error(err))
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

	// 构造下一行 device_task — 复用 fanouter 的单 command 翻译能力
	nextReq, err := s.buildNextRequest(ctx, mmlTask, t.DeviceSN, t.DeviceIndex, nextIdx)
	if err != nil {
		s.logger.Warn("sequencer: build next request failed",
			zap.String("mml_task_id", mmlTaskID.String()),
			zap.String("device_sn", t.DeviceSN),
			zap.Int("next_cmd_idx", nextIdx),
			zap.Error(err))
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
		// 模拟 nextIdx 的"虚拟完成"事件，递归找 nextIdx+1
		s.OnTaskCompleted(ctx, &task.Task{
			Source:       task.TaskSourceMML,
			SourceID:     t.SourceID,
			DeviceSN:     t.DeviceSN,
			DeviceIndex:  t.DeviceIndex,
			CommandIndex: nextIdx,
			Status:       task.TaskStatusFailed,
		})
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

// buildNextRequest 用 fanouter 的逻辑构造单条 device_task 请求。
// 单命令单设备 → 返回单 request；不合规 / 翻译失败 → 返 nil（caller 跳过）。
func (s *Sequencer) buildNextRequest(ctx context.Context, mmlTask *MMLTask, deviceSN string, deviceIdx, cmdIdx int) (*task.CreateTaskRequest, error) {
	if cmdIdx < 0 || cmdIdx >= len(mmlTask.Commands) {
		return nil, fmt.Errorf("cmd_idx %d out of range [0, %d)", cmdIdx, len(mmlTask.Commands))
	}

	// 临时构造一个只含单 device + 单 command 的 MMLTask 视图，
	// 复用 fanouter.buildDeviceTaskRequests 的 BuildTR069Params 链路。
	view := &MMLTask{
		ID:        mmlTask.ID,
		TaskName:  mmlTask.TaskName,
		Creator:   mmlTask.Creator,
		DeviceSNs: []string{deviceSN},
		Commands:  []map[string]interface{}{mmlTask.Commands[cmdIdx]},
	}
	reqs := s.fanouter.buildDeviceTaskRequests(ctx, view)
	if len(reqs) == 0 {
		return nil, nil
	}
	// fix 索引：buildDeviceTaskRequests 把单视图当作 cmd_idx=0 / dev_idx=0；
	// 还原成原 mmlTask 中的真实索引，保证 result_aggregator / 后续 Sequencer 调用链正确。
	reqs[0].CommandIndex = cmdIdx
	reqs[0].DeviceIndex = deviceIdx
	return reqs[0], nil
}
