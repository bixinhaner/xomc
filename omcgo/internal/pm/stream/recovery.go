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

const aggregationStreamName = "PM_AGGREGATION"

type Recovery struct {
	js       nats.JetStreamContext
	windows  *WindowRepository
	store    *RedisWindowStore
	snapshot *SnapshotStore
	matcher  *Matcher
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
		matcher: matcher, logger: logger,
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
	subscription, err := r.js.PullSubscribe(
		event.SubjectPMAggregationNormalized,
		"",
		nats.BindStream(aggregationStreamName),
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
			contributions, err := r.matcher.Match(payload, r.snapshot.Current())
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
		return fmt.Errorf("no retained PM aggregation events matched active window")
	}
	return nil
}

func sameWindowKey(left, right WindowKey) bool {
	return left.TaskVersionID == right.TaskVersionID &&
		left.Granularity == right.Granularity &&
		left.Start.Equal(right.Start)
}
