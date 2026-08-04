package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"go.uber.org/zap"
)

// OutboxWorker polls the outbox table and delivers pending push events.
type OutboxWorker struct {
	engine   *Engine
	repo     OutboxRepository
	logger   *zap.Logger
	pollSize int
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewOutboxWorker creates a new OutboxWorker.
func NewOutboxWorker(engine *Engine, repo OutboxRepository, logger *zap.Logger) *OutboxWorker {
	return &OutboxWorker{
		engine:   engine,
		repo:     repo,
		logger:   logger.Named("outbox-worker"),
		pollSize: 50,
		interval: 5 * time.Second,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the background polling loop.
func (w *OutboxWorker) Start() {
	go w.run()
}

// Close signals the worker to stop and waits for it to finish.
func (w *OutboxWorker) Close() error {
	close(w.stopCh)
	<-w.doneCh
	return nil
}

func (w *OutboxWorker) run() {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *OutboxWorker) poll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := w.repo.FetchPending(ctx, w.pollSize)
	if err != nil {
		w.logger.Error("fetch pending outbox entries", zap.Error(err))
		return
	}

	for _, entry := range entries {
		w.processEntry(ctx, entry)
	}
}

func (w *OutboxWorker) processEntry(ctx context.Context, entry OutboxEntry) {
	if err := w.repo.MarkProcessing(ctx, entry.ID); err != nil {
		w.logger.Error("mark outbox processing", zap.String("id", entry.ID.String()), zap.Error(err))
		return
	}

	target := w.engine.GetTarget(entry.TargetID)
	if target == nil {
		w.logger.Warn("push target not found for outbox entry, moving to dead",
			zap.String("entry_id", entry.ID.String()),
			zap.String("target_id", entry.TargetID))
		if err := w.repo.MarkDead(ctx, entry.ID, "push target not found: "+entry.TargetID); err != nil {
			w.logger.Error("mark dead", zap.Error(err))
		}
		return
	}

	// Check circuit breaker for this target.
	cb := w.engine.GetCircuitBreaker(entry.TargetID)
	if cb != nil {
		if err := cb.Allow(); err != nil {
			// Circuit is open; schedule retry after the reset timeout.
			nextRetry := time.Now().Add(30 * time.Second)
			if markErr := w.repo.MarkFailed(ctx, entry.ID, "circuit breaker open", nextRetry); markErr != nil {
				w.logger.Error("mark failed (circuit open)", zap.Error(markErr))
			}
			return
		}
	}

	evt := event.Event{
		ID:      entry.EventID,
		Subject: entry.Subject,
		Payload: entry.Payload,
	}

	deliverErr := w.engine.deliver(ctx, target, evt)
	if deliverErr == nil {
		if cb != nil {
			cb.RecordSuccess()
		}
		if err := w.repo.MarkDelivered(ctx, entry.ID); err != nil {
			w.logger.Error("mark outbox delivered", zap.Error(err))
		}
		return
	}

	// Delivery failed.
	if cb != nil {
		cb.RecordFailure()
	}

	nextAttempt := entry.Attempts + 1
	if nextAttempt >= entry.MaxAttempts {
		w.logger.Warn("outbox entry exhausted retries, moving to dead letter",
			zap.String("entry_id", entry.ID.String()),
			zap.String("target_id", entry.TargetID),
			zap.Int("attempts", nextAttempt),
			zap.Error(deliverErr))
		if err := w.repo.MarkDead(ctx, entry.ID, deliverErr.Error()); err != nil {
			w.logger.Error("mark dead", zap.Error(err))
		}
		return
	}

	nextRetry := time.Now().Add(calcBackoff(nextAttempt))
	w.logger.Info("outbox delivery failed, scheduling retry",
		zap.String("entry_id", entry.ID.String()),
		zap.Int("attempt", nextAttempt),
		zap.Time("next_retry", nextRetry),
		zap.Error(deliverErr))
	if err := w.repo.MarkFailed(ctx, entry.ID, deliverErr.Error(), nextRetry); err != nil {
		w.logger.Error("mark failed", zap.Error(err))
	}
}

// calcBackoff returns the exponential backoff delay for the given attempt number.
func calcBackoff(attempt int) time.Duration {
	cfg := reliability.DefaultRetryConfig()
	delay := time.Duration(math.Pow(2, float64(attempt-1))) * cfg.BaseDelay
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}
	return delay
}

// EnqueueEvent writes a push event to the outbox for reliable delivery.
// Called by the Engine instead of delivering synchronously.
func (e *Engine) EnqueueEvent(ctx context.Context, evt event.Event) error {
	if e.outboxRepo == nil {
		// Fallback: direct delivery without outbox (backward compat).
		return e.handleEvent(ctx, evt)
	}

	dataType := dataTypeFromSubject(evt.Subject)
	if dataType == "" {
		return nil
	}

	e.mu.RLock()
	targets := make([]*Target, 0)
	for _, t := range e.targets {
		if t.Enabled && matchesDataType(t.DataTypes, dataType) {
			cp := *t
			targets = append(targets, &cp)
		}
	}
	e.mu.RUnlock()

	payload, err := json.Marshal(map[string]interface{}{
		"event_id":  evt.ID,
		"subject":   evt.Subject,
		"payload":   evt.Payload,
		"timestamp": evt.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("marshal northbound outbox event: %w", err)
	}

	var insertErrors []error
	for _, t := range targets {
		entry := &OutboxEntry{
			EventID:     evt.ID,
			Subject:     evt.Subject,
			Payload:     payload,
			TargetID:    t.ID,
			MaxAttempts: t.RetryCount,
			NextRetryAt: time.Now(),
		}
		if entry.MaxAttempts < 1 {
			entry.MaxAttempts = 3
		}
		if err := e.outboxRepo.Insert(ctx, entry); err != nil {
			e.logger.Error("enqueue outbox entry",
				zap.String("target_id", t.ID),
				zap.Error(err))
			insertErrors = append(insertErrors, fmt.Errorf("insert northbound outbox target %s: %w", t.ID, err))
		}
	}

	return errors.Join(insertErrors...)
}
