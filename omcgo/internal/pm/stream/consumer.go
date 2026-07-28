package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const (
	aggregationConsumer        = "pm-aggregation-workers"
	hourlyRollupConsumer       = "pm-aggregation-hourly-rollup"
	dailyRollupConsumer        = "pm-aggregation-daily-rollup"
	aggregationControlConsumer = "pm-aggregation-control"
)

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
	hourlySub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationHourlyRollup,
		hourlyRollupConsumer,
		c.handleRollup,
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM hourly rollup stream: %w", err)
	}
	dailySub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationDailyRollup,
		dailyRollupConsumer,
		c.handleRollup,
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		_ = hourlySub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM daily rollup stream: %w", err)
	}
	controlSub, err := c.bus.PullSubscribe(
		event.SubjectPMAggregationTaskVersionChanged,
		aggregationControlConsumer,
		func(ctx context.Context, _ event.Event) error {
			return c.snapshot.Reload(ctx)
		},
	)
	if err != nil {
		_ = dataSub.Unsubscribe()
		_ = hourlySub.Unsubscribe()
		_ = dailySub.Unsubscribe()
		return nil, fmt.Errorf("subscribe PM aggregation task changes: %w", err)
	}
	return []event.Subscription{dataSub, hourlySub, dailySub, controlSub}, nil
}

func (c *Consumer) handleRollup(ctx context.Context, envelope event.Event) error {
	var payload RollupPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode PM compact rollup event: %w", err)
	}
	current := c.snapshot.Current()
	if current == nil {
		return fmt.Errorf("PM aggregation task snapshot missing")
	}
	version := current.ByVersion[payload.TaskVersionID]
	contributions, err := rollupContributions(payload, version, c.matcher.location)
	if err != nil {
		return fmt.Errorf("build PM parent rollup contributions: %w", err)
	}
	if isDeviceHourPayload(payload, current) {
		ruleContributions, matchErr := matchDeviceHourRules(
			payload, current, c.matcher.location,
		)
		if matchErr != nil {
			return fmt.Errorf("match device-hour aggregation rules: %w", matchErr)
		}
		contributions = append(contributions, ruleContributions...)
	}
	if err := c.processContributions(ctx, contributions); err != nil {
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
	return c.processContributions(ctx, contributions)
}

func (c *Consumer) processContributions(ctx context.Context, contributions []Contribution) error {
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
	windows            *WindowRepository
	finalizer          *Finalizer
	grace              time.Duration
	graceByGranularity map[Granularity]time.Duration
	logger             *zap.Logger
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
	return &TimeoutScanner{
		windows: windows, finalizer: finalizer, grace: grace, logger: logger,
		graceByGranularity: map[Granularity]time.Duration{
			GranularityHourly:  grace,
			GranularityDaily:   15 * time.Minute,
			GranularityWeekly:  30 * time.Minute,
			GranularityMonthly: 30 * time.Minute,
		},
	}
}

func (s *TimeoutScanner) SetGranularityGrace(
	daily, weekly, monthly time.Duration,
) *TimeoutScanner {
	if daily > 0 {
		s.graceByGranularity[GranularityDaily] = daily
	}
	if weekly > 0 {
		s.graceByGranularity[GranularityWeekly] = weekly
	}
	if monthly > 0 {
		s.graceByGranularity[GranularityMonthly] = monthly
	}
	return s
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
	const pageSize = 200
	now := time.Now().UTC()
	var after *WindowKey
	var finalizeErrors []error
	for {
		windows, err := s.windows.ListDueByGranularityAfter(
			ctx, now, s.graceByGranularity, s.grace, after, pageSize,
		)
		if err != nil {
			return err
		}
		if len(windows) == 0 {
			break
		}
		var wg sync.WaitGroup
		var errorMu sync.Mutex
		for _, window := range windows {
			window := window
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.finalizer.Finalize(ctx, window.Key, CloseTimeout); err != nil {
					s.logger.Warn("finalize timed out PM aggregation window",
						zap.String("task_version_id", window.Key.TaskVersionID.String()),
						zap.String("entity_key", window.Key.EntityKey),
						zap.Time("window_start", window.Key.Start),
						zap.Error(err))
					errorMu.Lock()
					finalizeErrors = append(finalizeErrors, err)
					errorMu.Unlock()
				}
			}()
		}
		wg.Wait()
		last := windows[len(windows)-1].Key
		after = &last
		if len(windows) < pageSize {
			break
		}
	}
	return errors.Join(finalizeErrors...)
}
