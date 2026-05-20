// Package backup — TransferComplete 路由器 (M2 of backup-restore-alignment-plan).
//
// CPE 完成 Upload/Download 后会主动上报 TransferComplete SOAP；ACS handler
// 已在 SubjectDeviceTransferComplete 上发布 *tr069.TransferComplete 事件，但
// 历史上没有按 CommandKey 路由到备份/恢复表回写最终状态。本路由器订阅该 subject，
// 按 CommandKey 解析任务类型 (backup / restore) 与短任务 ID，更新对应行的
// status / task_result / completed_at。
//
// 设计原则：
//   - 与 ACS 解耦：仅依赖 EventBus + 两个 TaskRepository，不引用 acs/upload 包。
//   - 幂等：CPE 可能重发 TransferComplete；MarkComplete 多次写同值无副作用。
//   - 兼容：ParseCommandKey 同时识别新格式 (`*_BACKUP_xxxxxxxx`) 与历史前缀
//     (`backup-xxxxxxxx` / `restore-xxxxxxxx`)，灰度切换 1 个 sprint。
//   - 宽容：无法识别的 CommandKey、匹配不到行、payload 损坏一律返回 nil，避免
//     NATS 无谓重投。真正的 DB 错误才返回，由 JetStream 重投兜底。
package backup

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// transferCompletePayload 是 ACS handler 发布的 *tr069.TransferComplete 的
// 子集。仅声明本路由器关心的字段。
type transferCompletePayload struct {
	CommandKey  string `json:"command_key"`
	FaultStruct *struct {
		FaultCode   int    `json:"FaultCode"`
		FaultString string `json:"FaultString"`
	} `json:"fault_struct,omitempty"`
	CompleteTime time.Time `json:"complete_time"`
}

// TransferCompleteRouter 订阅 SubjectDeviceTransferComplete，按 CommandKey
// 把回写分发到 backup_tasks / restore_tasks。
type TransferCompleteRouter struct {
	backupRepo  TaskRepository
	restoreRepo RestoreTaskRepository
	metrics     *RestoreMetrics // 复用，避免再引入新 metrics 类型；nil-safe
	logger      *zap.Logger
}

// NewTransferCompleteRouter 构造路由器。metrics 可为 nil。
func NewTransferCompleteRouter(
	backupRepo TaskRepository,
	restoreRepo RestoreTaskRepository,
	metrics *RestoreMetrics,
	logger *zap.Logger,
) *TransferCompleteRouter {
	return &TransferCompleteRouter{
		backupRepo:  backupRepo,
		restoreRepo: restoreRepo,
		metrics:     metrics,
		logger:      logger.Named("backup-tc-router"),
	}
}

// Subscribe 注册到 EventBus。使用专属 queue 名以免与 software 模块冲突
// (NATS Durable Consumer 同名 + 不同 FilterSubject 会被拒绝)。
func (r *TransferCompleteRouter) Subscribe(bus event.EventBus) error {
	_, err := bus.QueueSubscribe(
		event.SubjectDeviceTransferComplete,
		"backup-tc-router",
		r.handle,
	)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceTransferComplete, err)
	}
	r.logger.Info("backup transfer-complete router subscribed",
		zap.String("subject", event.SubjectDeviceTransferComplete))
	return nil
}

// handle 是 EventBus 回调。
func (r *TransferCompleteRouter) handle(ctx context.Context, evt event.Event) error {
	var p transferCompletePayload
	if err := evt.DecodePayload(&p); err != nil {
		r.recordOutcome("error")
		return fmt.Errorf("decode transfer_complete: %w", err)
	}
	kind, prefix := ParseCommandKey(p.CommandKey)
	if kind == CommandKeyKindUnknown {
		// 既不是备份也不是恢复 (可能是固件升级 / 其他模块的 CommandKey)；no-op。
		r.recordOutcome("skipped_not_ours")
		return nil
	}
	if prefix == "" {
		r.recordOutcome("skipped_no_prefix")
		r.logger.Debug("transfer_complete CommandKey has no task id prefix",
			zap.String("command_key", p.CommandKey))
		return nil
	}

	completedAt := p.CompleteTime
	if completedAt.IsZero() {
		completedAt = time.Now()
	}

	// 派生最终状态：FaultCode==0 视为成功；其它视为失败。
	success := p.FaultStruct == nil || p.FaultStruct.FaultCode == 0
	errMsg := ""
	if !success {
		errMsg = fmt.Sprintf("TransferComplete fault: code=%d, string=%s",
			p.FaultStruct.FaultCode, p.FaultStruct.FaultString)
	}

	switch kind {
	case CommandKeyKindBackup:
		return r.markBackup(ctx, prefix, success, completedAt, errMsg, p.CommandKey)
	case CommandKeyKindRestore:
		return r.markRestore(ctx, prefix, success, completedAt, errMsg, p.CommandKey)
	default:
		return nil
	}
}

func (r *TransferCompleteRouter) markBackup(
	ctx context.Context, prefix string, success bool, completedAt time.Time, errMsg, ck string,
) error {
	matches, err := r.backupRepo.FindByIDPrefix(ctx, prefix, 2)
	if err != nil {
		r.recordOutcome("error")
		return fmt.Errorf("find backup_task by prefix %q: %w", prefix, err)
	}
	if len(matches) == 0 {
		r.recordOutcome("skipped_no_match")
		r.logger.Info("no backup_task matches CommandKey prefix; skipping",
			zap.String("prefix", prefix), zap.String("command_key", ck))
		return nil
	}
	if len(matches) > 1 {
		r.logger.Warn("multiple backup_tasks match CommandKey prefix; using most recent",
			zap.String("prefix", prefix), zap.Int("count", len(matches)))
	}
	target := matches[0]
	// 不重复写已经终态的行 (避免 Reaper / Router 互相覆盖)。
	if target.Status == TaskCompleted || target.Status == TaskFailed || target.Status == TaskCancelled {
		r.recordOutcome("skipped_already_terminal")
		return nil
	}
	status := TaskCompleted
	result := int16(TaskResultSuccess)
	if !success {
		status = TaskFailed
		result = int16(TaskResultFailed)
	}
	if err := r.backupRepo.MarkComplete(ctx, target.ID, status, result, completedAt, errMsg); err != nil {
		r.recordOutcome("error")
		return fmt.Errorf("mark backup_task %s complete: %w", target.ID, err)
	}
	r.recordOutcome("backup_" + string(status))
	r.logger.Info("backup_task TransferComplete writeback",
		zap.String("task_id", target.ID.String()),
		zap.String("status", string(status)),
		zap.Int16("task_result", result))
	return nil
}

func (r *TransferCompleteRouter) markRestore(
	ctx context.Context, prefix string, success bool, completedAt time.Time, errMsg, ck string,
) error {
	matches, err := r.restoreRepo.FindByIDPrefix(ctx, prefix, 2)
	if err != nil {
		r.recordOutcome("error")
		return fmt.Errorf("find restore_task by prefix %q: %w", prefix, err)
	}
	if len(matches) == 0 {
		r.recordOutcome("skipped_no_match")
		r.logger.Info("no restore_task matches CommandKey prefix; skipping",
			zap.String("prefix", prefix), zap.String("command_key", ck))
		return nil
	}
	if len(matches) > 1 {
		r.logger.Warn("multiple restore_tasks match CommandKey prefix; using most recent",
			zap.String("prefix", prefix), zap.Int("count", len(matches)))
	}
	target := matches[0]
	if target.Status == RestoreCompleted || target.Status == RestoreFailed || target.Status == RestoreCancelled {
		r.recordOutcome("skipped_already_terminal")
		return nil
	}
	status := RestoreCompleted
	result := int16(TaskResultSuccess)
	if !success {
		status = RestoreFailed
		result = int16(TaskResultFailed)
	}
	if err := r.restoreRepo.MarkComplete(ctx, target.ID, status, result, completedAt, errMsg); err != nil {
		r.recordOutcome("error")
		return fmt.Errorf("mark restore_task %s complete: %w", target.ID, err)
	}
	r.recordOutcome("restore_" + string(status))
	r.logger.Info("restore_task TransferComplete writeback",
		zap.String("task_id", target.ID.String()),
		zap.String("status", string(status)),
		zap.Int16("task_result", result))
	return nil
}

// recordOutcome 是对 RestoreMetrics 的 nil-safe 包装；复用其 RecordRequest 计数器
// 类别字段做 outcome 维度（不另起新 collector）。
func (r *TransferCompleteRouter) recordOutcome(outcome string) {
	if r.metrics == nil {
		return
	}
	r.metrics.RecordRequest("tc_router_" + outcome)
}
