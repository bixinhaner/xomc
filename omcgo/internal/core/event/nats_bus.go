// Package event 已在 types.go 中声明包注释。
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const maxDeliveries = 5

const (
	defaultPendingMsgLimit   = 65536
	defaultPendingBytesLimit = 64 * 1024 * 1024
	gpvPendingMsgLimit       = 200000
	gpvPendingBytesLimit     = 256 * 1024 * 1024
)

// NATSEventBus 是基于 NATS JetStream 的生产级事件总线实现。
// 每个事件通过 JetStream 持久化存储，支持 At-Least-Once 交付语义。
// QueueSubscribe 使用 Durable Consumer，各实例彺负载均衡，适用于多实例水平扩展。
// 事件处理失败后指数退退重新投递（最多 5 次），超出后终止该消息防止无限重试。
type NATSEventBus struct {
	conn    *nats.Conn
	js      nats.JetStreamContext
	subs    []*nats.Subscription
	mu      sync.Mutex
	logger  *zap.Logger
	metrics *EventBusMetrics // issue #20：投递结果指标；nil 时（单进程/单测）静默 no-op。
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewNATSEventBus creates an EventBus backed by NATS JetStream.
func NewNATSEventBus(conn *nats.Conn, js nats.JetStreamContext, logger *zap.Logger) *NATSEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &NATSEventBus{
		conn:   conn,
		js:     js,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetMetrics attaches delivery-outcome metrics (issue #20). Call before any
// Subscribe so wrapHandler observes them; nil-safe — when unset, the bus keeps
// the legacy log-only behaviour.
func (b *NATSEventBus) SetMetrics(m *EventBusMetrics) {
	b.metrics = m
}

func (b *NATSEventBus) Publish(ctx context.Context, subject string, evt Event) error {
	evt.Subject = subject
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = b.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("publish to NATS: %w", err)
	}
	return nil
}

func (b *NATSEventBus) Subscribe(subject string, handler EventHandler) (Subscription, error) {
	sub, err := b.js.Subscribe(subject, b.wrapHandler(handler),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	return &natsSubscription{sub: sub}, nil
}

func (b *NATSEventBus) QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error) {
	sub, err := b.js.QueueSubscribe(subject, queue, b.wrapHandler(handler),
		nats.Durable(queue),
		nats.AckExplicit(),
	)
	if err != nil {
		return nil, fmt.Errorf("queue subscribe to %s (queue=%s): %w", subject, queue, err)
	}

	msgLimit, bytesLimit := pendingLimitsForSubject(subject)
	if err := sub.SetPendingLimits(msgLimit, bytesLimit); err != nil {
		_ = sub.Unsubscribe()
		return nil, fmt.Errorf("set pending limits for %s (queue=%s): %w", subject, queue, err)
	}
	b.logger.Info("queue subscription pending limits applied",
		zap.String("subject", subject),
		zap.String("queue", queue),
		zap.Int("pending_msgs", msgLimit),
		zap.Int("pending_bytes", bytesLimit),
	)

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	return &natsSubscription{sub: sub}, nil
}

func pendingLimitsForSubject(subject string) (msgLimit int, bytesLimit int) {
	if subject == SubjectCommandGetParamsResponse {
		return gpvPendingMsgLimit, gpvPendingBytesLimit
	}
	return defaultPendingMsgLimit, defaultPendingBytesLimit
}

func (b *NATSEventBus) Close() error {
	// Cancel the shared context so in-flight event handlers are notified.
	b.cancel()

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subs {
		if err := sub.Unsubscribe(); err != nil {
			b.logger.Warn("unsubscribe error", zap.Error(err))
		}
	}
	b.subs = nil
	return nil
}

func (b *NATSEventBus) wrapHandler(handler EventHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		evt, parseErr := decodeEventBytes(msg.Data)
		if parseErr != nil {
			b.logger.Error("unmarshal event", zap.Error(parseErr))
			// Permanent parse error — terminate to avoid infinite retry.
			// subject 用 msg.Subject（已解出 evt 之前），保证 dropped 指标有 subject 维度。
			b.metrics.inc(msg.Subject, deliveryOutcomeDropped)
			_ = msg.Term()
			return
		}

		handlerErr := handler(b.ctx, evt)
		deliveries := uint64(1)
		if meta, mErr := msg.Metadata(); mErr == nil && meta != nil {
			deliveries = meta.NumDelivered
		}

		decision := decideAck(handlerErr, deliveries, maxDeliveries)
		switch decision.action {
		case ackActionAck:
			b.metrics.inc(evt.Subject, deliveryOutcomeAck)
			_ = msg.Ack()
		case ackActionTerm:
			// 达到 maxDeliveries 后 Term 终止：消息被永久丢弃。此前只有 ERROR 日志，
			// 无指标 → max-retries 终止"静默"。补 terminated 指标使其可告警。
			b.metrics.inc(evt.Subject, deliveryOutcomeTerminated)
			b.logger.Error("handle event (terminating)",
				zap.String("subject", evt.Subject),
				zap.Uint64("delivery", deliveries),
				zap.Error(handlerErr))
			_ = msg.Term()
		case ackActionNak:
			b.metrics.inc(evt.Subject, deliveryOutcomeNak)
			b.logger.Error("handle event (retrying)",
				zap.String("subject", evt.Subject),
				zap.Uint64("delivery", deliveries),
				zap.Duration("backoff", decision.backoff),
				zap.Error(handlerErr))
			_ = msg.NakWithDelay(decision.backoff)
		}
	}
}

// ackAction 表示对一条 NATS 消息的处置动作。
type ackAction int

const (
	ackActionAck ackAction = iota
	ackActionNak
	ackActionTerm
)

// ackDecision 描述 wrapHandler 在执行业务 handler 后对 nats.Msg 的处置。
// 把决策逻辑独立成纯函数便于单元测试，避免依赖真实 NATS server。
type ackDecision struct {
	action  ackAction
	backoff time.Duration
}

// decideAck 决定对已投递 deliveries 次的消息采取何种动作：
//   - handler 无错 → Ack
//   - handler 出错且未达 maxDelivery → Nak with exponential backoff
//   - handler 出错且达到 maxDelivery → Term（避免无限重试）
//
// 指数退避序列：1s, 2s, 4s, 8s ...（基于 deliveries 已投递次数）。
// 该函数无副作用，可独立测试。
func decideAck(handlerErr error, deliveries, maxDelivery uint64) ackDecision {
	if handlerErr == nil {
		return ackDecision{action: ackActionAck}
	}
	if deliveries >= maxDelivery {
		return ackDecision{action: ackActionTerm}
	}
	if deliveries == 0 {
		deliveries = 1
	}
	// shift 上限保护，防止极端 deliveries 触发位移溢出。
	shift := deliveries - 1
	if shift > 30 {
		shift = 30
	}
	return ackDecision{
		action:  ackActionNak,
		backoff: time.Duration(1<<shift) * time.Second,
	}
}

// decodeEventBytes 把 JetStream 投递的 raw bytes 解析成 Event。
// 单独提取便于测试 unmarshal 路径与错误处理。
func decodeEventBytes(data []byte) (Event, error) {
	var evt Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return Event{}, fmt.Errorf("unmarshal event: %w", err)
	}
	return evt, nil
}

type natsSubscription struct {
	sub *nats.Subscription
}

func (s *natsSubscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}
