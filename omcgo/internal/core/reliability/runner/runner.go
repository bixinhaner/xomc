// Package runner 提供 worker 进程级的「retry + DLQ + 指标」装饰器。
//
// 设计目标：业务订阅函数无侵入接入重试与死信能力，新 subscriber 只需几行
// 装配即可。当前 PR 仅 PM Collector 接入，其他 11 个 subscriber 后续 PR
// 扩展（PRD §11）。
//
// 与 retry.go 的关系：本包复用 reliability.Retry()，在其基础上叠加：
//   - 失败耗尽后写 DLQ（dlq.Repository）
//   - Prometheus 指标（worker_retry_attempts_total / worker_dlq_entries_total）
//   - context 取消立即停止（Retry 自身已处理，本包透传）
//
// 与 EventBus 的关系：Wrap 接受 event.EventHandler 并返回 event.EventHandler，
// 业务方在 Subscribe 前用 Wrap 包一层即可。Replay 时通过 Publisher 接口反向发
// 出，EventBus 自然实现该接口。
package runner

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
)

// Publisher 是 Replay 路径上消费的小接口。EventBus 自然满足此契约。
// 定义在 runner 侧（消费者侧）以保持包之间无循环依赖。
type Publisher interface {
	Publish(ctx context.Context, subject string, evt event.Event) error
}

// Wrapper 是 PM Collector / 其他 subscriber 消费的契约接口。
// 业务侧通过 SetRunner(wrapper) 持有该接口，避免反向依赖整个 Runner 结构。
type Wrapper interface {
	Wrap(subject string, fn event.EventHandler) event.EventHandler
}

// Runner 持有重试 + DLQ 装配，每个业务模块创建一个实例。
//
// module 字段与指标的 module label 对应（"pm" / "mr" 等），用于聚合定位。
// metrics / dlq / logger 全允许为 nil（降级到 no-op，便于轻量部署与测试）。
type Runner struct {
	module    string
	retryCfg  reliability.RetryConfig
	dlq       dlq.Repository
	publisher Publisher
	metrics   *Metrics
	logger    *zap.Logger
}

var _ Wrapper = (*Runner)(nil)

// NewRunner 构造一个 Runner。logger 为 nil 时使用 zap.NewNop()。
func NewRunner(
	module string,
	retryCfg reliability.RetryConfig,
	dlqRepo dlq.Repository,
	pub Publisher,
	metrics *Metrics,
	logger *zap.Logger,
) *Runner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Runner{
		module:    module,
		retryCfg:  retryCfg,
		dlq:       dlqRepo,
		publisher: pub,
		metrics:   metrics,
		logger:    logger.Named("runner").With(zap.String("module", module)),
	}
}

// Module 返回 runner 关联的业务模块名（指标聚合用）。
func (r *Runner) Module() string { return r.module }

// Wrap 把业务 handler 套上「retry → DLQ → 指标」装饰，返回 event.EventHandler。
//
// 行为：
//  1. 调用 reliability.Retry 按 retryCfg 重试 fn(ctx, evt)
//  2. 全部失败后：
//     - 调 dlq.Repository.Insert 写一条 DeadLetter（dlq 为 nil 时跳过）
//     - 指标 worker_retry_attempts_total{result="failed"} +1
//     - 指标 worker_dlq_entries_total +1
//     - 返回 wrapped error（让 EventBus 决定 Nak / 丢弃）
//  3. 任意一次成功：worker_retry_attempts_total{result="success"} +1
//  4. ctx.Cancelled / DeadlineExceeded 立即返回，不写 DLQ（避免误判用户主动停机）
//
// 失败 fn 与失败 dlq.Insert 都不会 panic。
func (r *Runner) Wrap(subject string, fn event.EventHandler) event.EventHandler {
	if fn == nil {
		// 防御：业务方误传 nil 时返回一个 no-op，避免 nil dereference。
		return func(ctx context.Context, evt event.Event) error { return nil }
	}
	return func(ctx context.Context, evt event.Event) error {
		err := reliability.Retry(ctx, r.retryCfg, func(c context.Context) error {
			return fn(c, evt)
		})
		if err == nil {
			r.metrics.RecordRetryResult(r.module, subject, ResultSuccess)
			return nil
		}

		// ctx 取消视作非 DLQ 场景：Worker 收到 SIGTERM 不该把
		// 残余任务全部沉淀到 DLQ；让 EventBus 协议自身决定 Nak 重投。
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			r.logger.Info("retry aborted by context cancel",
				zap.String("subject", subject),
				zap.Error(err),
			)
			return err
		}

		r.metrics.RecordRetryResult(r.module, subject, ResultFailed)
		r.recordDeadLetter(ctx, subject, evt, err)
		return err
	}
}

// recordDeadLetter 在 retry 耗尽后落库并记录 dlq metric。
// 失败时仅记日志不传播：DLQ 记录失败不应阻塞其他事件处理。
func (r *Runner) recordDeadLetter(ctx context.Context, subject string, evt event.Event, lastErr error) {
	if r.dlq == nil {
		r.logger.Warn("dlq repository nil, dropping dead-letter",
			zap.String("subject", subject),
			zap.Error(lastErr),
		)
		return
	}
	entry := &dlq.DeadLetter{
		SourceModule:  r.module,
		SourceSubject: subject,
		Payload:       []byte(evt.Payload),
		Error:         lastErr.Error(),
		RetryCount:    r.retryCfg.MaxAttempts,
	}
	// 用单独 ctx：原 ctx 可能已 cancel，DLQ 写库不该跟着取消。
	// 但严格来说，调用方决定是否传新 ctx；这里保守用原 ctx（PRD V6 已经先放
	// ctx 错误，不会到这里）。
	if err := r.dlq.Insert(ctx, entry); err != nil {
		r.logger.Error("insert dead-letter failed",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return
	}
	r.metrics.RecordDLQEntry(r.module, subject)
	r.logger.Warn("event moved to DLQ",
		zap.String("subject", subject),
		zap.Int("retry_count", entry.RetryCount),
		zap.Error(lastErr),
	)
}

// Replay 把一条死信记录重新发布到原 subject。供 admin handler 复用。
// 不删除原死信记录，由运维 DELETE 路径决定何时清理（PRD §8.3）。
//
// publisher 为 nil 直接返回错误。
func (r *Runner) Replay(ctx context.Context, dl *dlq.DeadLetter) error {
	if dl == nil {
		return fmt.Errorf("replay: dead-letter is nil")
	}
	if r.publisher == nil {
		r.metrics.RecordReplayResult(dl.SourceModule, dl.SourceSubject, ReplayFailed)
		return fmt.Errorf("replay: publisher not configured")
	}
	evt := event.Event{
		ID:      dl.ID.String(),
		Subject: dl.SourceSubject,
		Payload: dl.Payload,
	}
	if err := r.publisher.Publish(ctx, dl.SourceSubject, evt); err != nil {
		r.metrics.RecordReplayResult(dl.SourceModule, dl.SourceSubject, ReplayFailed)
		return fmt.Errorf("replay publish: %w", err)
	}
	r.metrics.RecordReplayResult(dl.SourceModule, dl.SourceSubject, ReplayPublished)
	return nil
}
