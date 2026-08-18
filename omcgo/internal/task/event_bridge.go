package task

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// CompletionEventBridge subscribes to task.completed / task.failed NATS subjects
// and dispatches the decoded Task through a CompletionRouter. 使跨进程部署下的
// 聚合器（如 MML ResultAggregator）能收到 ACS / Worker 发来的终态事件。
//
// P1 重构（docs/design/mml-task-flow-design-20260424.md §3.3 C3/C4）：
// 原实现持有 `callbacks []TaskCompletionCallback` 且假定所有订阅者同等对待
// 每个 Task，这对多上游（mml / provision / backup / ...）不友好。现改为
// 持有一个 CompletionRouter，由 router 按 `Task.Source` 分发；装配阶段
// `router.Register(TaskSourceMML, mmlAggregator)` 等完成上下游绑定。
type CompletionEventBridge struct {
	router  *CompletionRouter
	deduper *event.Deduper
	logger  *zap.Logger
}

// NewCompletionEventBridge 构造事件桥接。router 不能为 nil。
// Completion projections must be idempotent themselves because their errors
// are returned to NATS for redelivery; pre-consumption deduplication is unsafe.
func NewCompletionEventBridge(logger *zap.Logger, router *CompletionRouter, deduper *event.Deduper) *CompletionEventBridge {
	return &CompletionEventBridge{
		router:  router,
		deduper: deduper,
		logger:  logger.Named("task-event-bridge"),
	}
}

// Subscribe 注册到 EventBus 的 task.completed / task.failed 主题。
// 使用 QueueSubscribe 保证同组订阅者负载均衡（多 APP 副本场景下每个事件只处理一次）。
func (b *CompletionEventBridge) Subscribe(bus event.EventBus) error {
	if bus == nil {
		return fmt.Errorf("event bus is nil")
	}
	if b.router == nil {
		return fmt.Errorf("completion router is nil")
	}
	// 每个 subject 用独立 queue 名（durable consumer name），避免 NATS Durable
	// Consumer "subject does not match consumer" 拒绝。
	subs := map[string]string{
		event.SubjectTaskCompleted: "task-completion-bridge-completed",
		event.SubjectTaskFailed:    "task-completion-bridge-failed",
	}
	// Completion projections participate in the durable NAK/retry contract.
	// Do not pre-mark them with Deduper.Wrap: that wrapper intentionally skips a
	// redelivery after the first handler failure.
	handler := event.EventHandler(b.handle)
	if b.deduper != nil {
		handler = b.deduper.WrapAfterSuccess("task-completion-bridge", handler)
	}
	for subject, queue := range subs {
		if _, err := bus.QueueSubscribe(subject, queue, handler); err != nil {
			return fmt.Errorf("subscribe %s: %w", subject, err)
		}
	}
	subjects := make([]string, 0, len(subs))
	for s := range subs {
		subjects = append(subjects, s)
	}
	b.logger.Info("task completion event bridge subscribed",
		zap.Strings("subjects", subjects))
	return nil
}

func (b *CompletionEventBridge) handle(ctx context.Context, evt event.Event) error {
	var t Task
	if err := evt.DecodePayload(&t); err != nil {
		b.logger.Warn("decode task event payload",
			zap.String("subject", evt.Subject),
			zap.Error(err))
		return fmt.Errorf("decode task event payload: %w", err)
	}
	if err := b.router.DispatchReliable(ctx, &t); err != nil {
		return fmt.Errorf("dispatch task completion: %w", err)
	}
	return nil
}
