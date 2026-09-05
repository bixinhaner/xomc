package agentbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/agentassistant"
	"go.uber.org/zap"
)

type AssistantEventSink interface {
	OnEvent(context.Context, agentassistant.Event) error
}
type EventForwarder struct {
	assistants AssistantEventSink
	repo       *OutboxRepository
	client     *Client
	logger     *zap.Logger
	cancel     context.CancelFunc
	done       chan struct{}
	once       sync.Once
}

func NewEventForwarder(repo *OutboxRepository, client *Client, logger *zap.Logger) *EventForwarder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventForwarder{repo: repo, client: client, logger: logger.Named("agent-event-forwarder"), done: make(chan struct{})}
}

func (w *EventForwarder) SetAssistantEventSink(sink AssistantEventSink) { w.assistants = sink }

func (w *EventForwarder) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)
}

func (w *EventForwarder) Close() error {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		<-w.done
	})
	return nil
}

func (w *EventForwarder) run(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.flush(ctx)
		}
	}
}

func (w *EventForwarder) flush(ctx context.Context) {
	items, err := w.repo.Claim(ctx, 20)
	if err != nil {
		w.logger.Error("claim agent event outbox", zap.Error(err))
		return
	}
	for _, item := range items {
		var envelope ConnectorEventEnvelope
		if err := json.Unmarshal(item.Payload, &envelope); err != nil {
			_ = w.repo.MarkFailed(ctx, item.ID, 11, fmt.Errorf("decode agent event outbox payload: %w", err))
			continue
		}
		if w.assistants != nil {
			deviceID := ""
			for _, resource := range envelope.Resources {
				if resource.Type == "device" {
					deviceID = resource.ID
					break
				}
			}
			data := make(map[string]any, len(envelope.Data))
			for key, value := range envelope.Data {
				if key != "apiHandbook" {
					data[key] = value
				}
			}
			if err := w.assistants.OnEvent(ctx, agentassistant.Event{ID: envelope.EventID, Type: envelope.EventType, DeviceID: deviceID, Data: data, OccurredAt: envelope.OccurredAt}); err != nil {
				_ = w.repo.MarkFailed(ctx, item.ID, item.Attempt, err)
				continue
			}
		}
		if err := w.client.PublishEvent(ctx, envelope); err != nil {
			w.logger.Warn("publish agent event", zap.String("event_id", item.EventID), zap.Error(err))
			if markErr := w.repo.MarkFailed(ctx, item.ID, item.Attempt, err); markErr != nil {
				w.logger.Error("mark agent event publish failure", zap.Error(markErr))
			}
			continue
		}
		if err := w.repo.MarkDelivered(ctx, item.ID); err != nil {
			w.logger.Error("mark agent event delivered", zap.String("event_id", item.EventID), zap.Error(err))
		}
	}
}
