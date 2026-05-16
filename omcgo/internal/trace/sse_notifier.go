package trace

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/events"
)

// SSENotifier 订阅 trace.task.* NATS 事件 → MessageHub.PublishGlobal 广播给在线用户。
//
// 设计文档 §4.4：M2 由 RBAC trace:read 过滤接收方；当前 RBAC 未细分该权限点，
// 用 PublishGlobal 给全部在线用户。M3 引入权限过滤后改 Publish(userID, ...)。
//
// 事件命名约定：SSE event 名 = NATS subject（前端按 event 名分发）。
type SSENotifier struct {
	hub    *events.MessageHub
	logger *zap.Logger
}

// NewSSENotifier 构造函数。
func NewSSENotifier(hub *events.MessageHub, logger *zap.Logger) *SSENotifier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SSENotifier{hub: hub, logger: logger.Named("trace-sse-notifier")}
}

// Subscribe 订阅 3 个 trace.task 事件 → 转发到 SSE Hub。
// 返回 cleanup func，进程关闭时调用 Unsubscribe。
func (n *SSENotifier) Subscribe(bus event.EventBus) (func(), error) {
	if bus == nil || n.hub == nil {
		return func() {}, nil
	}
	subs := make([]event.Subscription, 0, 3)
	for _, subject := range []string{
		event.SubjectTraceTaskStarted,
		event.SubjectTraceTaskStopped,
		event.SubjectTraceTaskPurged,
	} {
		sub, err := bus.Subscribe(subject, n.handle)
		if err != nil {
			for _, s := range subs {
				_ = s.Unsubscribe()
			}
			return nil, err
		}
		subs = append(subs, sub)
	}
	n.logger.Info("trace SSE notifier subscribed")
	return func() {
		for _, s := range subs {
			_ = s.Unsubscribe()
		}
	}, nil
}

func (n *SSENotifier) handle(_ context.Context, evt event.Event) error {
	// 直接转发 NATS payload 给前端；前端按 event 名分发
	data, err := json.Marshal(map[string]any{
		"subject": evt.Subject,
		"payload": evt.Payload,
	})
	if err != nil {
		return nil
	}
	msg := &events.SSEMessage{
		ID:    uuid.New().String(),
		Event: evt.Subject,
		Data:  data,
	}
	if pubErr := n.hub.PublishGlobal(msg); pubErr != nil {
		n.logger.Warn("trace SSE publish failed", zap.String("subject", evt.Subject), zap.Error(pubErr))
	}
	return nil
}
