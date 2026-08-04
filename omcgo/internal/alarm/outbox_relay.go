package alarm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const alarmOutboxRetention = 30 * 24 * time.Hour

type AlarmOutboxRelayOptions struct {
	WorkerID        string
	BatchSize       int
	PollInterval    time.Duration
	LeaseDuration   time.Duration
	MaxAttempts     int
	RetryBase       time.Duration
	RetryMax        time.Duration
	Retention       time.Duration
	MetricsInterval time.Duration
	CleanupInterval time.Duration
	Now             func() time.Time
}

// AlarmOutboxRelay publishes committed alarm lifecycle facts. A successful
// EventBus.Publish means the production JetStream implementation returned a
// synchronous PubAck; only then is the row marked published.
type AlarmOutboxRelay struct {
	repo    alarmOutboxRepository
	bus     event.EventBus
	logger  *zap.Logger
	metrics *AlarmMetrics
	options AlarmOutboxRelayOptions

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewAlarmOutboxRelay(repo alarmOutboxRepository, bus event.EventBus, logger *zap.Logger) *AlarmOutboxRelay {
	if logger == nil {
		logger = zap.NewNop()
	}
	return (&AlarmOutboxRelay{repo: repo, bus: bus, logger: logger.Named("outbox-relay")}).WithOptions(
		AlarmOutboxRelayOptions{},
	)
}

func (r *AlarmOutboxRelay) WithOptions(options AlarmOutboxRelayOptions) *AlarmOutboxRelay {
	if options.WorkerID == "" {
		options.WorkerID = uuid.NewString()
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 200 * time.Millisecond
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = time.Minute
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 20
	}
	if options.RetryBase <= 0 {
		options.RetryBase = time.Second
	}
	if options.RetryMax <= 0 {
		options.RetryMax = 5 * time.Minute
	}
	if options.Retention <= 0 {
		options.Retention = alarmOutboxRetention
	}
	if options.MetricsInterval <= 0 {
		options.MetricsInterval = 30 * time.Second
	}
	if options.CleanupInterval <= 0 {
		options.CleanupInterval = time.Hour
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	r.options = options
	return r
}

func (r *AlarmOutboxRelay) SetMetrics(metrics *AlarmMetrics) *AlarmOutboxRelay {
	r.metrics = metrics
	return r
}

// Start is idempotent and owns exactly one relay loop per constructed process service.
func (r *AlarmOutboxRelay) Start(ctx context.Context) {
	if r == nil || r.repo == nil || r.bus == nil {
		return
	}
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.done = make(chan struct{})
	done := r.done
	r.mu.Unlock()

	go func() {
		defer close(done)
		r.run(workerCtx)
	}()
}

// Stop is idempotent and waits until no publication can update ownership state.
func (r *AlarmOutboxRelay) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

func (r *AlarmOutboxRelay) run(ctx context.Context) {
	pollTicker := time.NewTicker(r.options.PollInterval)
	metricsTicker := time.NewTicker(r.options.MetricsInterval)
	cleanupTicker := time.NewTicker(r.options.CleanupInterval)
	defer pollTicker.Stop()
	defer metricsTicker.Stop()
	defer cleanupTicker.Stop()

	r.observe(ctx)
	for {
		published, err := r.RelayOnce(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			r.logger.Warn("publish alarm lifecycle outbox", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-metricsTicker.C:
			r.observe(ctx)
		case <-cleanupTicker.C:
			r.cleanup(ctx)
		default:
		}
		if published > 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
		case <-metricsTicker.C:
			r.observe(ctx)
		case <-cleanupTicker.C:
			r.cleanup(ctx)
		}
	}
}

func (r *AlarmOutboxRelay) cleanup(ctx context.Context) {
	before := r.options.Now().Add(-r.options.Retention)
	if _, err := r.repo.DeletePublishedBefore(ctx, before); err != nil && !errors.Is(err, context.Canceled) {
		r.logger.Warn("cleanup published alarm lifecycle outbox", zap.Error(err))
	}
}

// RelayOnce claims and publishes at most one configured batch.
func (r *AlarmOutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	now := r.options.Now()
	records, err := r.repo.Claim(ctx, OutboxClaimRequest{
		WorkerID: r.options.WorkerID, Now: now, LeaseDuration: r.options.LeaseDuration, Limit: r.options.BatchSize,
	})
	if err != nil {
		return 0, fmt.Errorf("claim alarm lifecycle outbox: %w", err)
	}

	published := 0
	var publishErrors []error
	for _, record := range records {
		envelope := event.Event{
			ID: record.EventID.String(), Subject: record.Subject, Payload: record.Payload, Timestamp: now,
		}
		if err := r.bus.Publish(ctx, record.Subject, envelope); err != nil {
			attempt := record.AttemptCount + 1
			status, markErr := r.repo.MarkFailed(ctx, OutboxFailure{
				EventID: record.EventID, WorkerID: r.options.WorkerID,
				AttemptCount: attempt, MaxAttempts: r.options.MaxAttempts,
				NextAttemptAt: now.Add(exponentialBackoff(attempt, r.options.RetryBase, r.options.RetryMax)),
				LastError:     "publish_failed", Now: now,
			})
			if markErr != nil {
				publishErrors = append(publishErrors, errors.Join(err, markErr))
				continue
			}
			if r.metrics != nil {
				r.metrics.OutboxRetryTotal.Inc()
				if status == OutboxStatusDead {
					r.metrics.OutboxDeadTotal.Inc()
				}
			}
			publishErrors = append(publishErrors, fmt.Errorf("publish alarm lifecycle event %s: %w", record.EventID, err))
			continue
		}
		if err := r.repo.MarkPublished(ctx, record.EventID, r.options.WorkerID, now); err != nil {
			publishErrors = append(publishErrors, err)
			continue
		}
		published++
		if r.metrics != nil {
			r.metrics.OutboxPublishedTotal.Inc()
		}
	}
	return published, errors.Join(publishErrors...)
}

func (r *AlarmOutboxRelay) Replay(ctx context.Context, eventID uuid.UUID) error {
	if err := r.repo.Replay(ctx, eventID, r.options.Now()); err != nil {
		return fmt.Errorf("replay alarm lifecycle event: %w", err)
	}
	return nil
}

func (r *AlarmOutboxRelay) observe(ctx context.Context) {
	if r.metrics == nil {
		return
	}
	now := r.options.Now()
	stats, err := r.repo.Stats(ctx, now)
	if err != nil {
		r.logger.Warn("observe alarm lifecycle outbox", zap.Error(err))
		return
	}
	r.metrics.OutboxBacklog.Set(float64(stats.Backlog))
	oldestAge := 0.0
	if stats.OldestCreatedAt != nil {
		oldestAge = now.Sub(*stats.OldestCreatedAt).Seconds()
		if oldestAge < 0 {
			oldestAge = 0
		}
	}
	r.metrics.OutboxOldestAgeSeconds.Set(oldestAge)
}

func exponentialBackoff(attempt int, base, maximum time.Duration) time.Duration {
	if base >= maximum {
		return maximum
	}
	if attempt <= 1 {
		return base
	}
	delay := base
	for i := 1; i < attempt; i++ {
		if delay >= maximum/2 {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}
