package agentbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type EventForwarder struct {
	repo   *OutboxRepository
	client *Client
	logger *zap.Logger
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func NewEventForwarder(repo *OutboxRepository, client *Client, logger *zap.Logger) *EventForwarder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventForwarder{repo: repo, client: client, logger: logger.Named("agent-event-forwarder"), done: make(chan struct{})}
}

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
