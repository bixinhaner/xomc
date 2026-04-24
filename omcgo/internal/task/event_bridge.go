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
	router *CompletionRouter
	logger *zap.Logger
}

// NewCompletionEventBridge 构造事件桥接。router 不能为 nil。
func NewCompletionEventBridge(logger *zap.Logger, router *CompletionRouter) *CompletionEventBridge {
	return &CompletionEventBridge{
		router: router,
		logger: logger.Named("task-event-bridge"),
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
	subs := []string{event.SubjectTaskCompleted, event.SubjectTaskFailed}
	for _, subject := range subs {
		sub := subject
		if _, err := bus.QueueSubscribe(sub, "task-completion-bridge", b.handle); err != nil {
			return fmt.Errorf("subscribe %s: %w", sub, err)
		}
	}
	b.logger.Info("task completion event bridge subscribed",
		zap.Strings("subjects", subs))
	return nil
}

func (b *CompletionEventBridge) handle(ctx context.Context, evt event.Event) error {
	var t Task
	if err := evt.DecodePayload(&t); err != nil {
		b.logger.Warn("decode task event payload",
			zap.String("subject", evt.Subject),
			zap.Error(err))
		return nil
	}
	b.router.Dispatch(ctx, &t)
	return nil
}
