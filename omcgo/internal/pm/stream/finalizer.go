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
	location *time.Location
}

type finalizationCoverage struct {
	SourceExpectedSlots  int64
	SourceReceivedSlots  int64
	MissingSlots         int64
	ChildrenComplete     bool
	DataComplete         bool
	VersionExpectedSlots int64
	NaturalSlots         int64
	PeriodComplete       bool
}

func (f *Finalizer) SetSnapshot(snapshot *SnapshotStore) *Finalizer {
	f.snapshot = snapshot
	return f
}

func (f *Finalizer) SetLocation(location *time.Location) *Finalizer {
	if location != nil {
		f.location = location
	}
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
	return f.finalizeUnderLock(ctx, key, reason)
}

func (f *Finalizer) finalizeUnderLock(ctx context.Context, key WindowKey, reason CloseReason) error {
	published, err := f.windows.IsPublished(ctx, key)
	if err != nil {
		return err
	}
	if published {
		return f.store.DeleteState(ctx, key)
	}
	state, err := f.store.Read(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("PM aggregation window state missing")
		}
		return err
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
	if err := f.store.DeleteState(ctx, key); err != nil {
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
	var revision int
	revisionSQL, revisionArgs, err := storage.Psql.Select("revision").
		From("pm_aggregation_windows").
		Where(windowKeyPredicate(key)).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM aggregation revision query: %w", err)
	}
	if err := tx.QueryRow(ctx, revisionSQL, revisionArgs...).Scan(&revision); err != nil {
		return fmt.Errorf("query PM aggregation revision: %w", err)
	}
	var version *TaskVersionSnapshot
	if f.snapshot != nil && f.snapshot.Current() != nil {
		version = f.snapshot.Current().ByVersion[key.TaskVersionID]
	}
	state, coverage := f.finalizationCoverageFor(key, version, state)
	rollups, err := buildRollupPayloads(
		key, reason, state, version, defaultRollupBatchValues,
	)
	if err != nil {
		return err
	}
	finalMetrics, _, err := buildFinalizedMetrics(version, state)
	if err != nil {
		return err
	}
	claimSQL, claimArgs, err := finalizationClaimUpdate(key, reason, state, coverage, version).ToSql()
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
	for _, payload := range rollups {
		if err := insertRollupTx(ctx, tx, payload); err != nil {
			return fmt.Errorf("persist compact PM Counter rollup: %w", err)
		}
	}
	deleteResultsSQL, deleteResultsArgs, err := storage.Psql.Delete("pm_aggregation_results").
		Where(sq.Eq{
			"task_version_id": key.TaskVersionID,
			"granularity":     string(key.Granularity),
			"window_start":    key.Start,
			"dimension_key":   key.EntityKey,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build replace PM aggregation results SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteResultsSQL, deleteResultsArgs...); err != nil {
		return fmt.Errorf("replace PM aggregation results: %w", err)
	}

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
				"revision", "version_effective_from", "version_effective_to",
				"received_slots", "expected_slots", "version_expected_slots",
				"natural_expected_slots", "version_slice_complete", "period_complete",
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
				metric.SampleCount, coverage.PeriodComplete && metric.FormulaComplete, coverage.MissingSlots,
				revision, versionEffectiveFrom(version), versionEffectiveTo(version),
				state.ReceivedSlots, coverage.NaturalSlots, coverage.VersionExpectedSlots, coverage.NaturalSlots,
				coverage.DataComplete && metric.FormulaComplete,
				coverage.PeriodComplete && metric.FormulaComplete,
			)
		}
		query, args, buildErr := builder.Suffix(`
ON CONFLICT (
  task_version_id, granularity, window_start, dimension_key, object_ldn, technology, metric_id
) DO UPDATE SET
  window_end = EXCLUDED.window_end,
  metric_value = EXCLUDED.metric_value,
  sample_count = EXCLUDED.sample_count,
  complete = EXCLUDED.complete,
  missing_slots = EXCLUDED.missing_slots,
  revision = EXCLUDED.revision,
  version_effective_from = EXCLUDED.version_effective_from,
  version_effective_to = EXCLUDED.version_effective_to,
  received_slots = EXCLUDED.received_slots,
  expected_slots = EXCLUDED.expected_slots,
  version_expected_slots = EXCLUDED.version_expected_slots,
  natural_expected_slots = EXCLUDED.natural_expected_slots,
  version_slice_complete = EXCLUDED.version_slice_complete,
  period_complete = EXCLUDED.period_complete`).ToSql()
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

func finalizationCoverageFor(
	key WindowKey,
	version *TaskVersionSnapshot,
	state WindowState,
	location *time.Location,
) (WindowState, finalizationCoverage) {
	state.ExpectedSlots = expectedSlotsForFinalization(key, version, state.ExpectedSlots, location)
	sourceExpected := state.SourceExpectedSlots
	sourceReceived := state.SourceReceivedSlots
	if sourceExpected == 0 {
		sourceExpected = state.ExpectedSlots
		sourceReceived = state.ReceivedSlots
	}
	childrenComplete := state.ReceivedSlots >= state.ExpectedSlots
	missing := max64(0, sourceExpected-sourceReceived)
	dataComplete := childrenComplete && missing == 0 && state.SourceIncompleteSlots == 0
	return state, finalizationCoverage{
		SourceExpectedSlots:  sourceExpected,
		SourceReceivedSlots:  sourceReceived,
		MissingSlots:         missing,
		ChildrenComplete:     childrenComplete,
		DataComplete:         dataComplete,
		VersionExpectedSlots: state.ExpectedSlots,
		NaturalSlots:         naturalExpectedSlots(version, key, state),
		PeriodComplete:       dataComplete && versionCoversNaturalPeriod(version, key),
	}
}

func (f *Finalizer) finalizationLocation() *time.Location {
	if f.location != nil {
		return f.location
	}
	return time.UTC
}

func (f *Finalizer) finalizationCoverageFor(
	key WindowKey,
	version *TaskVersionSnapshot,
	state WindowState,
) (WindowState, finalizationCoverage) {
	return finalizationCoverageFor(key, version, state, f.finalizationLocation())
}

func versionCoversNaturalPeriod(version *TaskVersionSnapshot, key WindowKey) bool {
	if version == nil || version.EffectiveFrom.After(key.Start) {
		return false
	}
	return version.EffectiveTo == nil || !version.EffectiveTo.Before(key.End)
}

func versionEffectiveFrom(version *TaskVersionSnapshot) any {
	if version == nil {
		return nil
	}
	return version.EffectiveFrom
}

func versionEffectiveTo(version *TaskVersionSnapshot) any {
	if version == nil || version.EffectiveTo == nil {
		return nil
	}
	return *version.EffectiveTo
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
