// Package event 已在 types.go 中声明包注释。
package event

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// ChannelEventBus 是基于 Go Channel 的内存事件总线，仅适用于单进程内。
// 生产环境使用 NATSEventBus；单元测试和开发自测时使用此实现，不依赖 NATS。
// 支持 NATS 风格的主题通配符（* 匹配单级，> 匹配多级）。
// QueueSubscribe 在同一队列组内负载均衡事件，Subscribe 给每个订阅者独立分发。
type ChannelEventBus struct {
	subscribers map[string][]*channelSubscription
	mu          sync.RWMutex
	bufferSize  int
	closed      atomic.Bool
	logger      *zap.Logger
}

type channelSubscription struct {
	ch      chan Event
	handler EventHandler
	queue   string
	cancel  context.CancelFunc
	done    chan struct{}
}

func (s *channelSubscription) Unsubscribe() error {
	s.cancel()
	<-s.done
	return nil
}

// NewChannelEventBus creates an in-process event bus.
func NewChannelEventBus(bufferSize int, logger *zap.Logger) *ChannelEventBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &ChannelEventBus{
		subscribers: make(map[string][]*channelSubscription),
		bufferSize:  bufferSize,
		logger:      logger,
	}
}

func (b *ChannelEventBus) Publish(ctx context.Context, subject string, evt Event) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	evt.Subject = subject

	b.mu.RLock()
	defer b.mu.RUnlock()

	for pattern, subs := range b.subscribers {
		if !matchSubject(pattern, subject) {
			continue
		}

		// For queue groups: pick one subscriber per group using round-robin
		queues := make(map[string][]*channelSubscription)
		var broadcast []*channelSubscription

		for _, sub := range subs {
			if sub.queue != "" {
				queues[sub.queue] = append(queues[sub.queue], sub)
			} else {
				broadcast = append(broadcast, sub)
			}
		}

		// Broadcast to non-queue subscribers
		for _, sub := range broadcast {
			select {
			case sub.ch <- evt:
			default:
				b.logger.Warn("event bus: channel full, dropping event",
					zap.String("subject", subject))
			}
		}

		// Load-balance within each queue group
		for _, queueSubs := range queues {
			if len(queueSubs) > 0 {
				// Simple: pick the first subscriber that can receive
				sent := false
				for _, sub := range queueSubs {
					select {
					case sub.ch <- evt:
						sent = true
					default:
					}
					if sent {
						break
					}
				}
			}
		}
	}

	return nil
}

func (b *ChannelEventBus) Subscribe(subject string, handler EventHandler) (Subscription, error) {
	return b.subscribe(subject, "", handler)
}

func (b *ChannelEventBus) QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	return b.subscribe(subject, queue, handler)
}

func (b *ChannelEventBus) PullSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	return b.subscribe(subject, queue, handler)
}

func (b *ChannelEventBus) subscribe(subject, queue string, handler EventHandler) (Subscription, error) {
	if b.closed.Load() {
		return nil, ErrBusClosed
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := &channelSubscription{
		ch:      make(chan Event, b.bufferSize),
		handler: handler,
		queue:   queue,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	// Start consumer goroutine
	go func() {
		defer close(sub.done)
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-sub.ch:
				if err := handler(ctx, evt); err != nil {
					b.logger.Error("event handler error",
						zap.String("subject", evt.Subject),
						zap.Error(err))
				}
			}
		}
	}()

	b.mu.Lock()
	b.subscribers[subject] = append(b.subscribers[subject], sub)
	b.mu.Unlock()

	return sub, nil
}

func (b *ChannelEventBus) Close() error {
	if b.closed.Swap(true) {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, subs := range b.subscribers {
		for _, sub := range subs {
			sub.cancel()
			<-sub.done
		}
	}
	b.subscribers = make(map[string][]*channelSubscription)
	return nil
}

// matchSubject checks if a NATS-style subject pattern matches a subject.
// Supports `*` (single token) and `>` (one or more trailing tokens).
func matchSubject(pattern, subject string) bool {
	if pattern == subject {
		return true
	}

	patternParts := strings.Split(pattern, ".")
	subjectParts := strings.Split(subject, ".")

	for i, pp := range patternParts {
		if pp == ">" {
			return i < len(subjectParts)
		}
		if i >= len(subjectParts) {
			return false
		}
		if pp != "*" && pp != subjectParts[i] {
			return false
		}
	}

	return len(patternParts) == len(subjectParts)
}

// ErrBusClosed 在对已关闭的 EventBus 进行操作时返回。
// 如 Publish 或 Subscribe 调用在 Close 之后进入时会返回此错误。
var ErrBusClosed = &busClosedError{}

type busClosedError struct{}

func (e *busClosedError) Error() string { return "event bus is closed" }
