package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

type OutboxRelay struct {
	repo            *OutboxRepository
	bus             event.EventBus
	interval        time.Duration
	logger          *zap.Logger
	metrics         *Metrics
	batch           int
	redeliveryAfter time.Duration
	redeliveryEvery time.Duration
}

func (r *OutboxRelay) SetMetrics(metrics *Metrics) *OutboxRelay {
	r.metrics = metrics
	return r
}

func NewOutboxRelay(repo *OutboxRepository, bus event.EventBus, logger *zap.Logger) *OutboxRelay {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OutboxRelay{
		repo: repo, bus: bus, interval: 200 * time.Millisecond,
		logger: logger, batch: 100,
		redeliveryAfter: 5 * time.Minute, redeliveryEvery: time.Minute,
	}
}

func (r *OutboxRelay) SetBatch(batch int) *OutboxRelay {
	if batch > 0 {
		r.batch = batch
	}
	return r
}

func (r *OutboxRelay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	go runRedeliveryLoop(ctx, r.redeliveryEvery, func() {
		r.requeueStale(ctx)
	})
	for {
		published, err := r.publishBatch(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			if r.metrics != nil {
				r.metrics.OutboxErrorsTotal.Inc()
			}
			r.logger.Warn("publish PM aggregation outbox", zap.Error(err))
		}
		if published {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func runRedeliveryLoop(ctx context.Context, every time.Duration, run func()) {
	if every <= 0 {
		every = time.Minute
	}
	run()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (r *OutboxRelay) requeueStale(ctx context.Context) {
	count, err := r.repo.RequeueStaleUnconsumed(
		ctx, time.Now().UTC().Add(-r.redeliveryAfter),
	)
	if err != nil {
		if r.metrics != nil {
			r.metrics.OutboxErrorsTotal.Inc()
		}
		r.logger.Warn("requeue stale PM aggregation outbox", zap.Error(err))
		return
	}
	if count > 0 {
		r.logger.Warn("requeued stale PM aggregation events after consumer acknowledgement timeout",
			zap.Int64("events", count),
			zap.Duration("ack_timeout", r.redeliveryAfter))
	}
}

func (r *OutboxRelay) publishOne(ctx context.Context) (bool, error) {
	previous := r.batch
	r.batch = 1
	defer func() { r.batch = previous }()
	return r.publishBatch(ctx)
}

// RelayOnce publishes at most one configured batch. It is useful for
// readiness probes, controlled drains and fault verification.
func (r *OutboxRelay) RelayOnce(ctx context.Context) (bool, error) {
	return r.publishBatch(ctx)
}

func (r *OutboxRelay) publishBatch(ctx context.Context) (bool, error) {
	tx, err := r.repo.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin PM aggregation outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := r.repo.lockBatch(ctx, tx, uint64(r.batch))
	if errors.Is(err, pgx.ErrNoRows) || len(rows) == 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var publishErrors []error
	for _, row := range rows {
		envelope, eventErr := event.NewEvent(event.SubjectPMAggregationNormalized, row.Payload)
		if eventErr != nil {
			return false, fmt.Errorf("build PM aggregation event: %w", eventErr)
		}
		envelope.ID = row.EventID.String()
		if publishErr := r.bus.Publish(ctx, event.SubjectPMAggregationNormalized, envelope); publishErr != nil {
			if markErr := markOutboxFailed(ctx, tx, row.EventID, publishErr); markErr != nil {
				return false, errors.Join(publishErr, markErr)
			}
			publishErrors = append(publishErrors, publishErr)
			continue
		}
		if err := markOutboxPublished(ctx, tx, row.EventID); err != nil {
			return false, err
		}
		if r.metrics != nil {
			r.metrics.OutboxPublishedTotal.Inc()
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit PM aggregation outbox publish: %w", err)
	}
	if len(publishErrors) > 0 {
		return true, errors.Join(publishErrors...)
	}
	return true, nil
}
