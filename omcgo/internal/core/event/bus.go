// Package event 已在 types.go 中声明包注释。
package event

import "context"

// EventBus 定义事件发布和订阅的抽象接口。
// 生产环境使用 NATSEventBus，单元测试使用 ChannelEventBus。
// Subscribe 让每个订阅者独立收到所有事件；
// QueueSubscribe 在同一队列组内负载均衡，用于多实例水平扩展。
type EventBus interface {
	// Publish sends an event to all subscribers of the given subject.
	Publish(ctx context.Context, subject string, event Event) error

	// Subscribe registers a handler for events on the given subject.
	// Each subscriber independently receives all matching events.
	Subscribe(subject string, handler EventHandler) (Subscription, error)

	// QueueSubscribe registers a handler in a named queue group.
	// Events are load-balanced across handlers in the same queue group.
	QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)

	// PullSubscribe registers a handler using a pull consumer.
	// The consumer fetches messages on demand instead of receiving push deliveries,
	// providing natural backpressure via MaxAckPending.
	//
	// queue is used as the base for the durable consumer name (appended with "-pull",
	// e.g. "provision-gpv" → durable "provision-gpv-pull"). This differs from
	// QueueSubscribe where queue is used directly as both the group name and durable name.
	PullSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)

	// Close shuts down the event bus and releases resources.
	Close() error
}

// Subscription represents an active event subscription.
type Subscription interface {
	Unsubscribe() error
}
