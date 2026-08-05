package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

const (
	aggregationRawStreamName    = "PM_AGG_15M"
	aggregationHourlyStreamName = "PM_AGG_HOURLY"
	aggregationDailyStreamName  = "PM_AGG_DAILY"
)

type recoverySource string

const (
	recoveryRaw15m      recoverySource = "raw_15m"
	recoveryDeviceHours recoverySource = "device_hours"
	recoveryHourly      recoverySource = "hourly"
	recoveryDaily       recoverySource = "daily"
)

type Recovery struct {
	js       nats.JetStreamContext
	windows  *WindowRepository
	store    *RedisWindowStore
	snapshot *SnapshotStore
	matcher  *Matcher
	rollups  *RollupOutboxRepository
	outbox   *OutboxRepository
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
		matcher: matcher, rollups: NewRollupOutboxRepository(windows.pool),
		outbox: NewOutboxRepository(windows.pool), logger: logger,
	}
}

func (r *Recovery) RestoreActiveWindows(ctx context.Context) error {
	var restoreErrors []error
	var cursor *WindowKey
	missing := make(map[recoverySource][]WindowKey)
	const pageSize = uint64(1000)
	for {
		active, err := r.windows.ListActiveAfter(ctx, cursor, pageSize)
		if err != nil {
			return errors.Join(append(restoreErrors, err)...)
		}
		for _, record := range active {
			exists, existsErr := r.store.Exists(ctx, record.Key)
			if existsErr != nil {
				restoreErrors = append(restoreErrors, existsErr)
				r.logger.Error("check PM aggregation window before recovery",
					zap.String("task_version_id", record.Key.TaskVersionID.String()),
					zap.String("entity_key", record.Key.EntityKey),
					zap.Time("window_start", record.Key.Start),
					zap.Error(existsErr))
				continue
			}
			if exists {
				continue
			}
			source, sourceErr := r.recoverySourceFor(record.Key)
			if sourceErr != nil {
				restoreErrors = append(restoreErrors, sourceErr)
				_ = r.windows.MarkFailed(ctx, record.Key, sourceErr)
				continue
			}
			missing[source] = append(missing[source], record.Key)
		}
		if len(active) < int(pageSize) {
			break
		}
		last := active[len(active)-1].Key
		cursor = &last
	}
	for source, keys := range missing {
		matched, replayErr := r.replayWindowBatch(ctx, source, keys)
		if replayErr != nil {
			restoreErrors = append(restoreErrors, replayErr)
		}
		for _, key := range keys {
			if _, ok := matched[windowRecoveryID(key)]; ok {
				continue
			}
			missingErr := fmt.Errorf(
				"no retained %s PM events matched active %s window %s",
				source, key.Granularity, key.EntityKey,
			)
			if replayErr != nil {
				missingErr = fmt.Errorf("%w: %v", missingErr, replayErr)
			}
			_ = r.windows.MarkFailed(ctx, key, missingErr)
			r.logger.Error("restore PM aggregation window",
				zap.String("task_version_id", key.TaskVersionID.String()),
				zap.String("entity_key", key.EntityKey),
				zap.Time("window_start", key.Start),
				zap.Error(missingErr))
		}
	}
	return errors.Join(restoreErrors...)
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
	source, err := r.recoverySourceFor(key)
	if err != nil {
		return err
	}
	matched, err := r.replayWindowBatch(ctx, source, []WindowKey{key})
	if err != nil {
		return err
	}
	if _, ok := matched[windowRecoveryID(key)]; !ok {
		return fmt.Errorf("no retained %s PM events matched active window", source)
	}
	return nil
}

// ReplayDurableDeviceHour idempotently replays retained normalized PM events
// into an incomplete synthetic device hour immediately before publication.
// Existing source_file_id values are ignored by Redis, so this can only fill
// missing slots; it cannot double-count the slots that were already accepted.
func (r *Recovery) ReplayDurableDeviceHour(
	ctx context.Context,
	key WindowKey,
) (bool, error) {
	if r == nil || r.outbox == nil || r.store == nil ||
		r.snapshot == nil || r.matcher == nil {
		return false, fmt.Errorf("durable PM device-hour replay is not configured")
	}
	snapshot := r.snapshot.Current()
	var version *TaskVersionSnapshot
	if snapshot != nil {
		version = snapshot.ByVersion[key.TaskVersionID]
	}
	if version == nil || !version.DevicePipeline || key.Granularity != GranularityHourly {
		return false, nil
	}

	_, complete, matched, err := replayDurableDeviceHourSources(
		ctx,
		key,
		snapshot,
		r.matcher,
		r.outbox.VisitPayloadsForPeriod,
		r.store.Accumulate,
	)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}
	return complete, nil
}

type durablePayloadVisitor func(
	context.Context,
	time.Time,
	time.Time,
	func(event.PMAggregationNormalizedPayload) error,
) error

type contributionAccumulatorFunc func(
	context.Context,
	Contribution,
) (AccumulateResult, error)

func replayDurableDeviceHourSources(
	ctx context.Context,
	key WindowKey,
	snapshot *TaskSnapshot,
	matcher *Matcher,
	visit durablePayloadVisitor,
	accumulate contributionAccumulatorFunc,
) (received int64, complete bool, matched bool, err error) {
	if snapshot == nil || matcher == nil || visit == nil || accumulate == nil {
		return 0, false, false, fmt.Errorf("durable PM device-hour replay dependencies are incomplete")
	}
	err = visit(ctx, key.Start, key.End, func(payload event.PMAggregationNormalizedPayload) error {
		if payload.DeviceID.String() != key.EntityKey {
			return nil
		}
		contributions, matchErr := matcher.MatchGranularity(
			payload, snapshot, GranularityHourly,
		)
		if matchErr != nil {
			return fmt.Errorf("match durable PM device-hour source: %w", matchErr)
		}
		for _, contribution := range contributions {
			if !sameWindowKey(contribution.Key, key) {
				continue
			}
			result, accumulateErr := accumulate(ctx, contribution)
			if accumulateErr != nil {
				return fmt.Errorf("accumulate durable PM device-hour source: %w", accumulateErr)
			}
			matched = true
			if result.ReceivedSlots > received {
				received = result.ReceivedSlots
			}
			complete = complete || result.Complete
		}
		return nil
	})
	if err != nil {
		return 0, false, matched, fmt.Errorf("replay durable PM device-hour sources: %w", err)
	}
	return received, complete, matched, nil
}

func (r *Recovery) recoverySourceFor(key WindowKey) (recoverySource, error) {
	current := r.snapshot.Current()
	if current == nil {
		return "", fmt.Errorf("PM aggregation task snapshot missing")
	}
	version := current.ByVersion[key.TaskVersionID]
	switch key.Granularity {
	case GranularityHourly:
		if version != nil && version.DevicePipeline {
			return recoveryRaw15m, nil
		}
		return recoveryDeviceHours, nil
	case GranularityDaily:
		return recoveryHourly, nil
	case GranularityWeekly, GranularityMonthly:
		return recoveryDaily, nil
	default:
		return "", fmt.Errorf("unsupported PM aggregation recovery granularity %q", key.Granularity)
	}
}

func (r *Recovery) replayWindowBatch(
	ctx context.Context,
	source recoverySource,
	keys []WindowKey,
) (map[string]struct{}, error) {
	matched := make(map[string]struct{}, len(keys))
	if len(keys) == 0 {
		return matched, nil
	}
	if r.js == nil {
		return matched, errors.New("PM aggregation recovery JetStream is nil")
	}
	wanted := make(map[string]struct{}, len(keys))
	start := keys[0].Start
	for _, key := range keys {
		wanted[windowRecoveryID(key)] = struct{}{}
		if key.Start.Before(start) {
			start = key.Start
		}
	}
	streamName, subject := aggregationRawStreamName, event.SubjectPMAggregationNormalized
	if source == recoveryDeviceHours || source == recoveryHourly {
		streamName, subject = aggregationHourlyStreamName, event.SubjectPMAggregationHourlyRollup
	} else if source == recoveryDaily {
		streamName, subject = aggregationDailyStreamName, event.SubjectPMAggregationDailyRollup
	}
	subscription, err := r.js.PullSubscribe(
		subject, "", nats.BindStream(streamName), nats.StartTime(start),
		nats.AckNone(),
	)
	if err != nil {
		return matched, fmt.Errorf("create PM aggregation batch replay consumer: %w", err)
	}
	defer func() { _ = subscription.Unsubscribe() }()

	for {
		messages, fetchErr := subscription.Fetch(256, nats.MaxWait(time.Second))
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			return matched, fmt.Errorf("fetch PM aggregation batch replay messages: %w", fetchErr)
		}
		for _, message := range messages {
			var envelope event.Event
			if err := json.Unmarshal(message.Data, &envelope); err != nil {
				return matched, fmt.Errorf("decode replay PM aggregation envelope: %w", err)
			}
			contributions, decodeErr := r.recoveryContributions(source, envelope)
			if decodeErr != nil {
				return matched, decodeErr
			}
			for _, contribution := range contributions {
				id := windowRecoveryID(contribution.Key)
				if _, ok := wanted[id]; !ok {
					continue
				}
				if _, err := r.store.Accumulate(ctx, contribution); err != nil {
					return matched, err
				}
				matched[id] = struct{}{}
			}
		}
		if errors.Is(fetchErr, nats.ErrTimeout) || len(messages) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return matched, ctx.Err()
		default:
		}
	}
	return matched, nil
}

func (r *Recovery) recoveryContributions(
	source recoverySource,
	envelope event.Event,
) ([]Contribution, error) {
	current := r.snapshot.Current()
	if current == nil {
		return nil, fmt.Errorf("PM aggregation task snapshot missing")
	}
	if source == recoveryRaw15m {
		var payload event.PMAggregationNormalizedPayload
		if err := envelope.DecodePayload(&payload); err != nil {
			return nil, fmt.Errorf("decode replay PM aggregation payload: %w", err)
		}
		return r.matcher.MatchGranularity(payload, current, GranularityHourly)
	}
	var payload RollupPayload
	if err := envelope.DecodePayload(&payload); err != nil {
		return nil, fmt.Errorf("decode compact rollup replay payload: %w", err)
	}
	if source == recoveryDeviceHours {
		if !isDeviceHourPayload(payload, current) {
			return nil, nil
		}
		return matchDeviceHourRules(payload, current, r.matcher.Location())
	}
	return rollupContributions(
		payload, current.ByVersion[payload.TaskVersionID], r.matcher.Location(),
	)
}

func windowRecoveryID(key WindowKey) string {
	return fmt.Sprintf(
		"%s\x1f%s\x1f%s\x1f%d",
		key.TaskVersionID, key.EntityKey, key.Granularity, key.Start.UTC().UnixNano(),
	)
}

func sameWindowKey(left, right WindowKey) bool {
	return left.TaskVersionID == right.TaskVersionID &&
		left.EntityKey == right.EntityKey &&
		left.Granularity == right.Granularity &&
		left.Start.Equal(right.Start)
}
