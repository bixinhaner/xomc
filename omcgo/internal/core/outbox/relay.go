package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const (
	defaultRelayPollInterval = time.Second
	defaultRelayClaimLease   = 30 * time.Second
	defaultRelayBatchSize    = 100
	defaultRelayMaxAttempts  = 10
	defaultRelayBaseBackoff  = time.Second
	defaultRelayMaxBackoff   = 5 * time.Minute
)

// Publisher is the narrow EventBus capability needed by the relay.
type Publisher interface {
	Publish(context.Context, string, event.Event) error
}

type RelayConfig struct {
	PollInterval time.Duration
	ClaimLease   time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

// Relay publishes claimed durable events and finalizes their delivery state.
type Relay struct {
	repo      DeliveryRepository
	publisher Publisher
	cfg       RelayConfig
	logger    *zap.Logger
	now       func() time.Time
	wait      func(context.Context, time.Duration) bool
}

func NewRelay(
	repo DeliveryRepository,
	publisher Publisher,
	cfg RelayConfig,
	logger *zap.Logger,
) (*Relay, error) {
	if repo == nil {
		return nil, fmt.Errorf("create event outbox relay: repository is required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("create event outbox relay: publisher is required")
	}
	var err error
	cfg, err = normalizeRelayConfig(cfg)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Relay{
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		logger:    logger.Named("event-outbox-relay"),
		now:       func() time.Time { return time.Now().UTC() },
		wait:      waitForRelayInterval,
	}, nil
}

func (r *Relay) ProcessOnce(ctx context.Context) (int, error) {
	now := r.now()
	entries, err := r.repo.ClaimDue(ctx, ClaimOptions{
		Now:         now,
		Lease:       r.cfg.ClaimLease,
		Limit:       r.cfg.BatchSize,
		MaxAttempts: r.cfg.MaxAttempts,
	})
	if err != nil {
		return 0, fmt.Errorf("claim event outbox delivery batch: %w", err)
	}

	processed := 0
	for _, entry := range entries {
		evt := event.Event{
			ID:        entry.ID.String(),
			Subject:   entry.Subject,
			Payload:   entry.Payload,
			Timestamp: entry.CreatedAt,
		}
		publishErr := r.publisher.Publish(ctx, entry.Subject, evt)
		finalizedAt := r.now()
		if publishErr == nil {
			ok, err := r.repo.MarkPublished(
				ctx,
				entry.ID,
				entry.ClaimToken,
				finalizedAt,
			)
			if err != nil {
				return processed, fmt.Errorf(
					"finalize published event outbox row %s: %w",
					entry.ID,
					err,
				)
			}
			if !ok {
				return processed, fmt.Errorf(
					"finalize published event outbox row %s: claim ownership lost",
					entry.ID,
				)
			}
			processed++
			continue
		}

		if entry.Attempts >= r.cfg.MaxAttempts {
			ok, err := r.repo.MarkDead(
				ctx,
				entry.ID,
				entry.ClaimToken,
				publishErr.Error(),
				finalizedAt,
			)
			if err != nil {
				return processed, fmt.Errorf(
					"finalize dead event outbox row %s: %w",
					entry.ID,
					err,
				)
			}
			if !ok {
				return processed, fmt.Errorf(
					"finalize dead event outbox row %s: claim ownership lost",
					entry.ID,
				)
			}
			r.logger.Error(
				"event outbox delivery exhausted",
				zap.String("event_id", entry.ID.String()),
				zap.String("subject", entry.Subject),
				zap.Int("attempt", entry.Attempts),
				zap.Error(publishErr),
			)
			processed++
			continue
		}

		nextAttemptAt := finalizedAt.Add(relayBackoff(
			entry.Attempts,
			r.cfg.BaseBackoff,
			r.cfg.MaxBackoff,
		))
		ok, err := r.repo.MarkFailed(
			ctx,
			entry.ID,
			entry.ClaimToken,
			publishErr.Error(),
			nextAttemptAt,
		)
		if err != nil {
			return processed, fmt.Errorf(
				"schedule failed event outbox row %s: %w",
				entry.ID,
				err,
			)
		}
		if !ok {
			return processed, fmt.Errorf(
				"schedule failed event outbox row %s: claim ownership lost",
				entry.ID,
			)
		}
		r.logger.Warn(
			"event outbox delivery failed; retry scheduled",
			zap.String("event_id", entry.ID.String()),
			zap.String("subject", entry.Subject),
			zap.Int("attempt", entry.Attempts),
			zap.Time("next_attempt_at", nextAttemptAt),
			zap.Error(publishErr),
		)
		processed++
	}
	return processed, nil
}

// Run processes immediately, then polls until the context is cancelled.
// Repository failures are logged and retried on the next interval.
func (r *Relay) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		if _, err := r.ProcessOnce(ctx); err != nil {
			r.logger.Error("process event outbox batch", zap.Error(err))
		}
		if !r.wait(ctx, r.cfg.PollInterval) {
			return nil
		}
	}
}

func normalizeRelayConfig(cfg RelayConfig) (RelayConfig, error) {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = defaultRelayPollInterval
	}
	if cfg.ClaimLease == 0 {
		cfg.ClaimLease = defaultRelayClaimLease
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = defaultRelayBatchSize
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = defaultRelayMaxAttempts
	}
	if cfg.BaseBackoff == 0 {
		cfg.BaseBackoff = defaultRelayBaseBackoff
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = defaultRelayMaxBackoff
	}
	switch {
	case cfg.PollInterval < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: poll interval must be positive")
	case cfg.ClaimLease < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: claim lease must be positive")
	case cfg.BatchSize < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: batch size must be positive")
	case cfg.MaxAttempts < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: maximum attempts must be positive")
	case cfg.BaseBackoff < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: base backoff must be positive")
	case cfg.MaxBackoff < 0:
		return RelayConfig{}, fmt.Errorf("create event outbox relay: maximum backoff must be positive")
	case cfg.MaxBackoff < cfg.BaseBackoff:
		return RelayConfig{}, fmt.Errorf(
			"create event outbox relay: maximum backoff must not be below base backoff",
		)
	default:
		return cfg, nil
	}
}

func relayBackoff(attempt int, base, maximum time.Duration) time.Duration {
	if attempt <= 1 {
		return base
	}
	delay := base
	for current := 1; current < attempt; current++ {
		if delay >= maximum || delay > maximum/2 {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

func waitForRelayInterval(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

var _ Publisher = (event.EventBus)(nil)
