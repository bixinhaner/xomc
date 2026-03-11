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

// NATSEventBus implements EventBus using NATS JetStream.
type NATSEventBus struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	subs   []*nats.Subscription
	mu     sync.Mutex
	logger *zap.Logger
}

// NewNATSEventBus creates an EventBus backed by NATS JetStream.
func NewNATSEventBus(conn *nats.Conn, js nats.JetStreamContext, logger *zap.Logger) *NATSEventBus {
	return &NATSEventBus{
		conn:   conn,
		js:     js,
		logger: logger,
	}
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

	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()

	return &natsSubscription{sub: sub}, nil
}

func (b *NATSEventBus) Close() error {
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
		var evt Event
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			b.logger.Error("unmarshal event", zap.Error(err))
			// Permanent parse error — terminate to avoid infinite retry
			msg.Term()
			return
		}

		if err := handler(context.Background(), evt); err != nil {
			meta, _ := msg.Metadata()
			deliveries := uint64(1)
			if meta != nil {
				deliveries = meta.NumDelivered
			}
			b.logger.Error("handle event",
				zap.String("subject", evt.Subject),
				zap.Uint64("delivery", deliveries),
				zap.Error(err))
			if deliveries >= maxDeliveries {
				b.logger.Warn("max deliveries reached, terminating message",
					zap.String("subject", evt.Subject),
					zap.Uint64("deliveries", deliveries))
				msg.Term()
			} else {
				// Exponential backoff: 1s, 2s, 4s, 8s ...
				msg.NakWithDelay(time.Duration(1<<(deliveries-1)) * time.Second)
			}
			return
		}

		msg.Ack()
	}
}

type natsSubscription struct {
	sub *nats.Subscription
}

func (s *natsSubscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}
