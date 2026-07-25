package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	aggregationRawStreamName    = "PM_AGG_15M"
	aggregationHourlyStreamName = "PM_AGG_HOURLY"
	aggregationDailyStreamName  = "PM_AGG_DAILY"
)

type Recovery struct {
	js       nats.JetStreamContext
	windows  *WindowRepository
	store    *RedisWindowStore
	snapshot *SnapshotStore
	matcher  *Matcher
	rollups  *RollupOutboxRepository
	logger   *zap.Logger
}

func NewRecovery(
	js nats.JetStreamContext,
	windows *WindowRepository,
	store *RedisWindowStore,
	snapshot *SnapshotStore,
	matcher *Matcher,
	logger *zap.Logger,
) *Recovery {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Recovery{
		js: js, windows: windows, store: store, snapshot: snapshot,
		matcher: matcher, rollups: NewRollupOutboxRepository(windows.pool), logger: logger,
	}
}

func (r *Recovery) RestoreActiveWindows(ctx context.Context) error {
	active, err := r.windows.ListActive(ctx, 10000)
	if err != nil {
		return err
	}
	for _, record := range active {
		if _, err := r.store.Read(ctx, record.Key); err == nil {
			continue
		} else if !errors.Is(err, redis.Nil) {
			return err
		}
		if err := r.ReplayWindow(ctx, record.Key); err != nil {
			_ = r.windows.MarkFailed(ctx, record.Key, err)
			r.logger.Error("restore PM aggregation window from NATS",
				zap.String("task_version_id", record.Key.TaskVersionID.String()),
				zap.Time("window_start", record.Key.Start),
				zap.Error(err))
		}
	}
	return nil
}

func (r *Recovery) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.RestoreActiveWindows(ctx); err != nil && ctx.Err() == nil {
				r.logger.Error("periodic PM aggregation recovery", zap.Error(err))
			}
		}
	}
}

func (r *Recovery) ReplayWindow(ctx context.Context, key WindowKey) error {
	if r.js == nil {
		return errors.New("PM aggregation recovery JetStream is nil")
	}
	if key.Granularity == GranularityHourly {
		return r.replayRawWindow(ctx, key)
	}
	return r.replayRollupWindow(ctx, key)
}

func (r *Recovery) replayRawWindow(ctx context.Context, key WindowKey) error {
	subscription, err := r.js.PullSubscribe(
		event.SubjectPMAggregationNormalized,
		"",
		nats.BindStream(aggregationRawStreamName),
		nats.StartTime(key.Start),
		nats.AckNone(),
	)
	if err != nil {
		return fmt.Errorf("create PM aggregation replay consumer: %w", err)
	}
	defer func() { _ = subscription.Unsubscribe() }()

	matched := 0
	for {
		messages, fetchErr := subscription.Fetch(256, nats.MaxWait(time.Second))
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			return fmt.Errorf("fetch PM aggregation replay messages: %w", fetchErr)
		}
		for _, message := range messages {
			var envelope event.Event
			if err := json.Unmarshal(message.Data, &envelope); err != nil {
				return fmt.Errorf("decode replay PM aggregation envelope: %w", err)
			}
			var payload event.PMAggregationNormalizedPayload
			if err := envelope.DecodePayload(&payload); err != nil {
				return fmt.Errorf("decode replay PM aggregation payload: %w", err)
			}
			contributions, err := r.matcher.MatchGranularity(
				payload, r.snapshot.Current(), GranularityHourly,
			)
			if err != nil {
				return err
			}
			for _, contribution := range contributions {
				if !sameWindowKey(contribution.Key, key) {
					continue
				}
				if _, err := r.store.Accumulate(ctx, contribution); err != nil {
					return err
				}
				matched++
			}
		}
		if errors.Is(fetchErr, nats.ErrTimeout) || len(messages) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if matched == 0 {
		return fmt.Errorf("no retained 15-minute PM events matched active hourly window")
	}
	return nil
}

func (r *Recovery) replayRollupWindow(ctx context.Context, key WindowKey) error {
	streamName := aggregationHourlyStreamName
	subject := event.SubjectPMAggregationHourlyRollup
	if key.Granularity == GranularityWeekly || key.Granularity == GranularityMonthly {
		streamName = aggregationDailyStreamName
		subject = event.SubjectPMAggregationDailyRollup
	}
	subscription, err := r.js.PullSubscribe(
		subject,
		"",
		nats.BindStream(streamName),
		nats.StartTime(key.Start),
		nats.AckNone(),
	)
	if err != nil {
		return fmt.Errorf("create PM compact rollup replay consumer: %w", err)
	}
	defer func() { _ = subscription.Unsubscribe() }()

	matched := 0
	for {
		messages, fetchErr := subscription.Fetch(256, nats.MaxWait(time.Second))
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			return fmt.Errorf("fetch PM compact rollup replay messages: %w", fetchErr)
		}
		for _, message := range messages {
			var envelope event.Event
			if err := json.Unmarshal(message.Data, &envelope); err != nil {
				return fmt.Errorf("decode compact rollup replay envelope: %w", err)
			}
			var payload RollupPayload
			if err := envelope.DecodePayload(&payload); err != nil {
				return fmt.Errorf("decode compact rollup replay payload: %w", err)
			}
			current := r.snapshot.Current()
			if current == nil {
				return fmt.Errorf("PM aggregation task snapshot missing")
			}
			contributions, err := rollupContributions(
				payload, current.ByVersion[payload.TaskVersionID], r.matcher.location,
			)
			if err != nil {
				return err
			}
			for _, contribution := range contributions {
				if !sameWindowKey(contribution.Key, key) {
					continue
				}
				if _, err := r.store.Accumulate(ctx, contribution); err != nil {
					return err
				}
				matched++
			}
		}
		if errors.Is(fetchErr, nats.ErrTimeout) || len(messages) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if matched == 0 {
		source := GranularityHourly
		if key.Granularity == GranularityWeekly || key.Granularity == GranularityMonthly {
			source = GranularityDaily
		}
		snapshots, err := r.rollups.ListSnapshots(
			ctx, key.TaskVersionID, source, key.Start, key.End,
		)
		if err != nil {
			return err
		}
		current := r.snapshot.Current()
		for _, payload := range snapshots {
			contributions, err := rollupContributions(
				payload, current.ByVersion[payload.TaskVersionID], r.matcher.location,
			)
			if err != nil {
				return err
			}
			for _, contribution := range contributions {
				if !sameWindowKey(contribution.Key, key) {
					continue
				}
				if _, err := r.store.Accumulate(ctx, contribution); err != nil {
					return err
				}
				matched++
			}
		}
	}
	if matched == 0 {
		return fmt.Errorf("no compact PM rollups matched active %s window", key.Granularity)
	}
	return nil
}

func sameWindowKey(left, right WindowKey) bool {
	return left.TaskVersionID == right.TaskVersionID &&
		left.Granularity == right.Granularity &&
		left.Start.Equal(right.Start)
}
