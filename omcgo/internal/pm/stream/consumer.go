package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const aggregationConsumer = "pm-aggregation-workers"

type Consumer struct {
	bus       event.EventBus
	snapshot  *SnapshotStore
	matcher   *Matcher
	windows   *WindowRepository
	store     *RedisWindowStore
	finalizer *Finalizer
	logger    *zap.Logger
	metrics   *Metrics
}

func (c *Consumer) SetMetrics(metrics *Metrics) *Consumer {
	c.metrics = metrics
	return c
}

func NewConsumer(
	bus event.EventBus,
	snapshot *SnapshotStore,
	matcher *Matcher,
	windows *WindowRepository,
	store *RedisWindowStore,
	finalizer *Finalizer,
	logger *zap.Logger,
) *Consumer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Consumer{
		bus: bus, snapshot: snapshot, matcher: matcher, windows: windows,
		store: store, finalizer: finalizer, logger: logger,
	}
}

func (c *Consumer) Subscribe() ([]event.Subscription, error) {
	dataSub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationNormalized,
		aggregationConsumer,
		c.handle,
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe PM aggregation data stream: %w", err)
	}
	controlSub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationTaskVersionChanged,
		"pm-aggregation-control",
		func(ctx context.Context, _ event.Event) error {
			return c.snapshot.Reload(ctx)
		},
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM aggregation task changes: %w", err)
	}
	return []event.Subscription{dataSub, controlSub}, nil
}

func (c *Consumer) handle(ctx context.Context, envelope event.Event) error {
	if err := c.process(ctx, envelope); err != nil {
		if c.metrics != nil {
			c.metrics.EventsFailedTotal.Inc()
		}
		return err
	}
	if c.metrics != nil {
		c.metrics.EventsProcessedTotal.Inc()
	}
	return nil
}

func (c *Consumer) process(ctx context.Context, envelope event.Event) error {
	var payload event.PMAggregationNormalizedPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode PM aggregation event: %w", err)
	}
	if err := ValidateEvent(payload); err != nil {
		return err
	}
	contributions, err := c.matcher.Match(payload, c.snapshot.Current())
	if err != nil {
		return fmt.Errorf("match PM aggregation event: %w", err)
	}
	for _, contribution := range contributions {
		published, err := c.windows.IsPublished(ctx, contribution.Key)
		if err != nil {
			return err
		}
		if published {
			if c.metrics != nil {
				c.metrics.LateEventsTotal.Inc()
			}
			c.logger.Info("ignore late PM aggregation event",
				zap.String("source_file_id", contribution.SourceFileID),
				zap.String("task_version_id", contribution.Key.TaskVersionID.String()),
				zap.Time("window_start", contribution.Key.Start))
			continue
		}
		if err := c.windows.EnsureOpen(ctx, contribution); err != nil {
			return err
		}
		result, err := c.store.Accumulate(ctx, contribution)
		if err != nil {
			return err
		}
		if result.Duplicate && c.metrics != nil {
			c.metrics.DuplicateEventsTotal.Inc()
		}
		if err := c.windows.ObserveReceived(ctx, contribution.Key, result.ReceivedSlots); err != nil {
			return err
		}
		if result.Complete {
			if err := c.finalizer.Finalize(ctx, contribution.Key, CloseComplete); err != nil {
				return err
			}
		}
	}
	return nil
}

type TimeoutScanner struct {
	windows   *WindowRepository
	finalizer *Finalizer
	grace     time.Duration
	logger    *zap.Logger
}

func NewTimeoutScanner(
	windows *WindowRepository,
	finalizer *Finalizer,
	grace time.Duration,
	logger *zap.Logger,
) *TimeoutScanner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TimeoutScanner{windows: windows, finalizer: finalizer, grace: grace, logger: logger}
}

func (s *TimeoutScanner) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.runOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Warn("scan timed out PM aggregation windows", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *TimeoutScanner) runOnce(ctx context.Context) error {
	windows, err := s.windows.ListDue(ctx, time.Now().UTC(), s.grace, 100)
	if err != nil {
		return err
	}
	for _, window := range windows {
		if err := s.finalizer.Finalize(ctx, window.Key, CloseTimeout); err != nil {
			s.logger.Warn("finalize timed out PM aggregation window",
				zap.String("task_version_id", window.Key.TaskVersionID.String()),
				zap.Time("window_start", window.Key.Start),
				zap.Error(err))
		}
	}
	return nil
}
