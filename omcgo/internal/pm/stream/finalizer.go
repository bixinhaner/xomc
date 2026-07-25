package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type CloseReason string

const (
	CloseComplete        CloseReason = "complete"
	CloseTimeout         CloseReason = "timeout"
	finalResultBatchSize             = 1000
)

type Finalizer struct {
	windows  *WindowRepository
	store    *RedisWindowStore
	logger   *zap.Logger
	metrics  *Metrics
	slots    chan struct{}
	snapshot *SnapshotStore
	promote  func(context.Context, WindowKey, CloseReason, WindowState) error
}

func (f *Finalizer) SetSnapshot(snapshot *SnapshotStore) *Finalizer {
	f.snapshot = snapshot
	return f
}

func (f *Finalizer) SetRollupPromoter(
	promote func(context.Context, WindowKey, CloseReason, WindowState) error,
) *Finalizer {
	f.promote = promote
	return f
}

func NewFinalizer(
	windows *WindowRepository,
	store *RedisWindowStore,
	logger *zap.Logger,
) *Finalizer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Finalizer{windows: windows, store: store, logger: logger}
}

func (f *Finalizer) SetConcurrency(concurrency int) *Finalizer {
	if concurrency > 0 {
		f.slots = make(chan struct{}, concurrency)
	}
	return f
}

func (f *Finalizer) SetMetrics(metrics *Metrics) *Finalizer {
	f.metrics = metrics
	return f
}

func (f *Finalizer) Finalize(ctx context.Context, key WindowKey, reason CloseReason) error {
	if f.slots != nil {
		select {
		case f.slots <- struct{}{}:
			defer func() { <-f.slots }()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	started := time.Now()
	if f.metrics != nil {
		defer func() { f.metrics.FinalizeDuration.Observe(time.Since(started).Seconds()) }()
	}
	lock, err := f.store.TryFinalizeLock(ctx, key, 2*time.Minute)
	if err != nil {
		return err
	}
	if lock == nil {
		return nil
	}
	defer func() {
		if err := lock.Release(context.Background()); err != nil {
			f.logger.Warn("release PM aggregation finalize lock", zap.Error(err))
		}
	}()

	published, err := f.windows.IsPublished(ctx, key)
	if err != nil {
		return err
	}
	if published {
		return f.store.Delete(ctx, key)
	}
	state, err := f.store.Read(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("PM aggregation window state missing")
		}
		return err
	}
	if f.promote != nil &&
		(key.Granularity == GranularityHourly || key.Granularity == GranularityDaily) {
		if err := f.promote(ctx, key, reason, state); err != nil {
			promoteErr := fmt.Errorf("promote PM aggregation counter state: %w", err)
			if f.metrics != nil {
				f.metrics.FinalizeErrorsTotal.Inc()
			}
			_ = f.windows.MarkFailed(ctx, key, promoteErr)
			return promoteErr
		}
	}
	if err := f.writeFinal(ctx, key, reason, state); err != nil {
		if f.metrics != nil {
			f.metrics.FinalizeErrorsTotal.Inc()
		}
		_ = f.windows.MarkFailed(ctx, key, err)
		return err
	}
	if f.metrics != nil {
		f.metrics.WindowsFinalizedTotal.WithLabelValues(string(reason)).Inc()
	}
	if err := f.store.Delete(ctx, key); err != nil {
		f.logger.Warn("delete published PM aggregation Redis window",
			zap.String("task_version_id", key.TaskVersionID.String()),
			zap.Time("window_start", key.Start),
			zap.Error(err))
	}
	return nil
}

func (f *Finalizer) writeFinal(
	ctx context.Context,
	key WindowKey,
	reason CloseReason,
	state WindowState,
) error {
	tx, err := f.windows.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin finalize PM aggregation window: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var version *TaskVersionSnapshot
	if f.snapshot != nil && f.snapshot.Current() != nil {
		version = f.snapshot.Current().ByVersion[key.TaskVersionID]
	}
	finalMetrics, formulaIncomplete, err := buildFinalizedMetrics(version, state)
	if err != nil {
		return err
	}
	sourceExpected := state.SourceExpectedSlots
	sourceReceived := state.SourceReceivedSlots
	if sourceExpected == 0 {
		sourceExpected = state.ExpectedSlots
		sourceReceived = state.ReceivedSlots
	}
	childrenComplete := state.ReceivedSlots >= state.ExpectedSlots
	missing := max64(0, sourceExpected-sourceReceived)
	dataComplete := childrenComplete && missing == 0 &&
		state.SourceIncompleteSlots == 0 && !formulaIncomplete

	claimSQL, claimArgs, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "finalizing").
		Set("close_reason", string(reason)).
		Set("received_slots", state.ReceivedSlots).
		Set("source_expected_slots", sourceExpected).
		Set("source_received_slots", sourceReceived).
		Set("missing_slots", missing).
		Set("children_complete", childrenComplete).
		Set("source_incomplete_slots", state.SourceIncompleteSlots).
		Set("data_complete", dataComplete).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": []string{"open", "failed", "finalizing"}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build claim PM aggregation window SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, claimSQL, claimArgs...)
	if err != nil {
		return fmt.Errorf("claim PM aggregation window: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}

	complete := reason == CloseComplete && dataComplete
	resultCount := 0
	for start := 0; start < len(finalMetrics); start += finalResultBatchSize {
		end := start + finalResultBatchSize
		if end > len(finalMetrics) {
			end = len(finalMetrics)
		}
		builder := storage.Psql.Insert("pm_aggregation_results").
			Columns(
				"window_start", "window_end", "task_id", "task_version_id",
				"granularity", "dimension", "dimension_key", "dimension_name",
				"object_ldn", "device_oui", "device_sn", "technology",
				"metric_id", "metric_path", "metric_type",
				"aggregation_op", "metric_value", "sample_count", "complete", "missing_slots",
			)
		for _, metric := range finalMetrics[start:end] {
			definition := metric.Definition
			builder = builder.Values(
				key.Start, key.End, key.TaskID, key.TaskVersionID,
				string(key.Granularity), string(definition.Dimension),
				definition.DimensionKey, definition.DimensionName,
				definition.ObjectLDN, definition.DeviceOUI, definition.DeviceSN,
				definition.Technology, metric.MetricID, definition.MetricPath,
				metric.MetricType, string(metric.Operation), metric.Value,
				metric.SampleCount, complete, missing,
			)
		}
		query, args, buildErr := builder.Suffix(`
ON CONFLICT (
  task_version_id, granularity, window_start, dimension_key, object_ldn, technology, metric_id
) DO NOTHING`).ToSql()
		if buildErr != nil {
			return fmt.Errorf("build insert PM aggregation result SQL: %w", buildErr)
		}
		tag, execErr := tx.Exec(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("insert PM aggregation result: %w", execErr)
		}
		resultCount += int(tag.RowsAffected())
	}
	publishSQL, publishArgs, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "published").
		Set("result_count", resultCount).
		Set("published_at", time.Now().UTC()).
		Set("updated_at", time.Now().UTC()).
		Set("last_error", nil).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": "finalizing"}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build publish PM aggregation window SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, publishSQL, publishArgs...); err != nil {
		return fmt.Errorf("publish PM aggregation window: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit PM aggregation window: %w", err)
	}
	return nil
}

func accumulatorValue(accumulator Accumulator) float64 {
	switch accumulator.Definition.Operation {
	case AggregationAvg:
		return accumulator.Sum / float64(accumulator.Count)
	case AggregationMin:
		return accumulator.Min
	case AggregationMax:
		return accumulator.Max
	default:
		return accumulator.Sum
	}
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
