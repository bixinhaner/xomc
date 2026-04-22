package task

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// CompletionEventBridge subscribes to task.completed / task.failed NATS subjects
// and dispatches the decoded Task to registered TaskCompletionCallback handlers.
// 使跨进程部署下的聚合器（如 MML ResultAggregator）能收到 ACS/Worker 发来的终态。
type CompletionEventBridge struct {
	callbacks []TaskCompletionCallback
	logger    *zap.Logger
}

// NewCompletionEventBridge 构造事件桥接。
func NewCompletionEventBridge(logger *zap.Logger, callbacks ...TaskCompletionCallback) *CompletionEventBridge {
	return &CompletionEventBridge{
		callbacks: callbacks,
		logger:    logger.Named("task-event-bridge"),
	}
}

// AddCallback 注册额外回调。
func (b *CompletionEventBridge) AddCallback(cb TaskCompletionCallback) {
	b.callbacks = append(b.callbacks, cb)
}

// Subscribe 注册到 EventBus 的 task.completed / task.failed 主题。
// 使用 QueueSubscribe 保证同组订阅者负载均衡（多 APP 副本场景下每个事件只处理一次）。
func (b *CompletionEventBridge) Subscribe(bus event.EventBus) error {
	if bus == nil {
		return fmt.Errorf("event bus is nil")
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
	for _, cb := range b.callbacks {
		cb.OnTaskCompleted(ctx, &t)
	}
	return nil
}
