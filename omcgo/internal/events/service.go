package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventService 是 internal/events 子系统的高层门面（facade）。
//
// 设计目的：
//   - 让上层调用方（notification/admin/router 等）只依赖一个稳定接口，
//     而不是直接持有 *MessageHub / MessageStore 两个具体类型。
//   - 便于测试：单元测试用 mock EventService，无需起 Redis。
//   - 让 Publish / Subscribe / 历史回放三件事统一在同一抽象下。
//
// 接口保持 Go 风格的小接口（3 个方法），更复杂的发布场景仍用
// EventServiceImpl 上的具体方法（PublishGlobal、PublishSimple、Unsubscribe）。
type EventService interface {
	// Publish 投递一条 SSE 消息到指定用户。如果对应 user 没有在线连接，
	// 仅持久化以便其重连时回放。
	Publish(ctx context.Context, userID string, msg *SSEMessage) error

	// SubscribeQuery 订阅指定用户的实时事件流，返回只读 channel。
	// 当同一用户已有连接时，旧连接会被踢掉。
	SubscribeQuery(ctx context.Context, userID string) (<-chan *SSEMessage, error)

	// QueryHistory 回放该用户的历史消息（用于断线重连）。
	// afterID 为空时返回最近 limit 条；非空时返回该 ID 之后的消息。
	QueryHistory(ctx context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error)
}

// EventServiceImpl 是 EventService 的默认实现。
// 它薄薄地封装 *MessageHub（实时分发）和 MessageStore（持久化回放），
// 不引入新的状态、不修改原 hub/store 的语义。
type EventServiceImpl struct {
	hub    *MessageHub
	store  MessageStore
	logger *zap.Logger
}

// NewEventService 构造一个 EventServiceImpl。
//
// 参数：
//   - hub：消息分发中枢；不可为 nil。
//   - store：持久化存储；可为 nil（关闭历史回放，但实时分发仍可用）。
//   - logger：日志；不可为 nil。
//
// 返回 *EventServiceImpl（具体类型），调用方按 Go 习惯
//   "Accept interfaces, return structs"。
func NewEventService(hub *MessageHub, store MessageStore, logger *zap.Logger) *EventServiceImpl {
	if hub == nil {
		panic("events: NewEventService requires non-nil MessageHub")
	}
	if logger == nil {
		panic("events: NewEventService requires non-nil logger")
	}
	return &EventServiceImpl{
		hub:    hub,
		store:  store,
		logger: logger.Named("events-service"),
	}
}

// Publish 实现 EventService 接口。
// ctx 当前未在 hub.Publish 内部使用，仍接受是为了：
//  1. 与 EventService 接口签名一致；
//  2. 未来 hub 内部接入 Redis 异步管道时可直接传入。
func (s *EventServiceImpl) Publish(ctx context.Context, userID string, msg *SSEMessage) error {
	if userID == "" {
		return fmt.Errorf("events.Publish: userID is empty")
	}
	if msg == nil {
		return fmt.Errorf("events.Publish: msg is nil")
	}
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	_ = ctx // reserved for future use
	if err := s.hub.Publish(userID, msg); err != nil {
		return fmt.Errorf("events.Publish hub: %w", err)
	}
	return nil
}

// PublishJSON 是便利方法：把任意可 JSON 序列化的对象作为 data 投递。
// 适用于上层模块只想发"事件类型 + 业务对象"而不想自己处理序列化的场景。
func (s *EventServiceImpl) PublishJSON(ctx context.Context, userID, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("events.PublishJSON marshal: %w", err)
	}
	return s.Publish(ctx, userID, &SSEMessage{
		ID:    uuid.New().String(),
		Event: eventType,
		Data:  data,
	})
}

// PublishGlobal 广播一条消息给所有在线连接（不持久化）。
// 用于"全网通知"场景，例如 license 即将到期、运维公告等。
func (s *EventServiceImpl) PublishGlobal(ctx context.Context, msg *SSEMessage) error {
	if msg == nil {
		return fmt.Errorf("events.PublishGlobal: msg is nil")
	}
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	_ = ctx
	if err := s.hub.PublishGlobal(msg); err != nil {
		return fmt.Errorf("events.PublishGlobal hub: %w", err)
	}
	return nil
}

// SubscribeQuery 实现 EventService 接口。
func (s *EventServiceImpl) SubscribeQuery(ctx context.Context, userID string) (<-chan *SSEMessage, error) {
	if userID == "" {
		return nil, fmt.Errorf("events.SubscribeQuery: userID is empty")
	}
	_ = ctx
	ch, err := s.hub.Subscribe(userID)
	if err != nil {
		return nil, fmt.Errorf("events.SubscribeQuery hub: %w", err)
	}
	return ch, nil
}

// Unsubscribe 解除指定用户的订阅。HTTP handler 应在 defer 里调用它。
func (s *EventServiceImpl) Unsubscribe(userID string) {
	if userID == "" {
		return
	}
	s.hub.Unsubscribe(userID)
}

// QueryHistory 实现 EventService 接口。
// 当 store 为 nil（SSE 持久化禁用）时，返回 nil, nil（不报错）。
func (s *EventServiceImpl) QueryHistory(ctx context.Context, userID string, afterID string, limit int) ([]*SSEMessage, error) {
	if userID == "" {
		return nil, fmt.Errorf("events.QueryHistory: userID is empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if s.store == nil {
		return nil, nil
	}
	msgs, err := s.store.GetSince(ctx, userID, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("events.QueryHistory store: %w", err)
	}
	return msgs, nil
}

// Hub 暴露底层 MessageHub，仅给 SSEHandler 等需要直接操作 channel 的内部组件使用。
// 业务模块不应使用此方法，请走 EventService 接口。
func (s *EventServiceImpl) Hub() *MessageHub {
	return s.hub
}

// Store 暴露底层 MessageStore，仅给 SSEHandler 回放使用。
// 业务模块不应使用此方法，请走 QueryHistory。
func (s *EventServiceImpl) Store() MessageStore {
	return s.store
}

// 编译期断言：EventServiceImpl 实现了 EventService。
var _ EventService = (*EventServiceImpl)(nil)
