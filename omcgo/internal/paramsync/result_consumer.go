package paramsync

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/event"
)

type ResultConsumer struct {
	bus       event.EventBus
	processor ResultProcessor
	sub       event.Subscription
}

func NewResultConsumer(bus event.EventBus, processor ResultProcessor) *ResultConsumer {
	return &ResultConsumer{bus: bus, processor: processor}
}

func (c *ResultConsumer) Start() error {
	if c.bus == nil || c.processor == nil {
		return fmt.Errorf("parameter sync result consumer requires event bus and processor")
	}
	sub, err := c.bus.PullSubscribe(event.SubjectParamSyncTaskResult, "param-sync-results", c.Handle)
	if err != nil {
		return fmt.Errorf("subscribe parameter sync task results: %w", err)
	}
	c.sub = sub
	return nil
}

func (c *ResultConsumer) Handle(ctx context.Context, evt event.Event) error {
	var payload event.ParamSyncTaskResultPayload
	if err := evt.DecodePayload(&payload); err != nil {
		// Malformed payload is permanent: acknowledge it after surfacing a typed
		// error is not possible with the generic EventBus. Keep it observable by
		// returning nil only after the bus-level poison-message limit/DLQ policy.
		return fmt.Errorf("decode parameter sync task result: %w", err)
	}
	if _, err := c.processor.Process(ctx, payload); err != nil {
		// Returning an error causes NATSEventBus to NAK/redeliver. A nil return is
		// therefore strictly after the database transaction committed.
		return err
	}
	return nil
}

func (c *ResultConsumer) Stop() error {
	if c.sub == nil {
		return nil
	}
	return c.sub.Unsubscribe()
}
