// Package realtime provides non-persistent Core NATS broadcast messaging.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const subscribeFlushTimeout = 5 * time.Second

// Subscription represents one Core NATS subscription.
type Subscription interface {
	Unsubscribe() error
}

// CoreNATS publishes transient payloads and broadcasts them to every ordinary
// subscriber. It reuses its caller-owned connection and never closes it.
type CoreNATS struct {
	publish   func(subject string, data []byte) error
	subscribe func(subject string, handler func([]byte)) (Subscription, error)
	flush     func(context.Context) error
}

// NewCoreNATS adapts a shared NATS connection for transient broadcasts.
func NewCoreNATS(conn *nats.Conn) *CoreNATS {
	return newCoreNATS(
		conn.Publish,
		func(subject string, handler func([]byte)) (Subscription, error) {
			return conn.Subscribe(subject, func(msg *nats.Msg) { handler(msg.Data) })
		},
		conn.FlushWithContext,
	)
}

func newCoreNATS(
	publish func(string, []byte) error,
	subscribe func(string, func([]byte)) (Subscription, error),
	flush func(context.Context) error,
) *CoreNATS {
	return &CoreNATS{publish: publish, subscribe: subscribe, flush: flush}
}

// Publish sends the payload as direct JSON without the durable EventBus envelope.
func (b *CoreNATS) Publish(ctx context.Context, subject string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal realtime payload: %w", err)
	}
	if err := b.publish(subject, data); err != nil {
		return fmt.Errorf("publish realtime subject %s: %w", subject, err)
	}
	return nil
}

// Subscribe registers an ordinary Core NATS subscriber. A flush completes
// before return so a subsequent publisher cannot race ahead of registration.
func (b *CoreNATS) Subscribe(subject string, handler func([]byte)) (Subscription, error) {
	sub, err := b.subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe realtime subject %s: %w", subject, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), subscribeFlushTimeout)
	defer cancel()
	if err := b.flush(ctx); err != nil {
		flushErr := fmt.Errorf("flush realtime subscription %s: %w", subject, err)
		if cleanupErr := sub.Unsubscribe(); cleanupErr != nil {
			return nil, errors.Join(
				flushErr,
				fmt.Errorf("unsubscribe realtime subject %s after flush failure: %w", subject, cleanupErr),
			)
		}
		return nil, flushErr
	}
	return sub, nil
}
