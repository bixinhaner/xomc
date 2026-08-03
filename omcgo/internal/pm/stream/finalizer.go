package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type CloseReason string

const (
	CloseComplete CloseReason = "complete"
	CloseTimeout  CloseReason = "timeout"
)

var ErrFinalizeClaimLost = errors.New("PM aggregation finalize claim ownership lost")

type Finalizer struct {
	windows             *WindowRepository
	store               *RedisWindowStore
	logger              *zap.Logger
	metrics             *Metrics
	slots               chan struct{}
	snapshot            *SnapshotStore
	location            LocationProvider
	initialPublications sync.Map
}

type finalizationCoverage struct {
	SourceExpectedSlots               int64
	SourceReceivedSlots               int64
	MissingSlots                      int64
	ChildrenComplete                  bool
	DataComplete                      bool
	VersionExpectedSlots              int64
	NaturalSlots                      int64
	PeriodComplete                    bool
	DailyVersionExpectedSlotsMismatch bool
}

func (f *Finalizer) SetSnapshot(snapshot *SnapshotStore) *Finalizer {
	f.snapshot = snapshot
	return f
}

func (f *Finalizer) SetLocation(location *time.Location) *Finalizer {
	f.location = fixedLocationProvider(location)
	return f
}

func (f *Finalizer) SetLocationProvider(location LocationProvider) *Finalizer {
	f.location = location
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

func (f *Finalizer) Concurrency() int {
	if f.slots == nil {
		return 1
	}
	return cap(f.slots)
}

func (f *Finalizer) SetMetrics(metrics *Metrics) *Finalizer {
	f.metrics = metrics
	return f
}

func (f *Finalizer) Finalize(ctx context.Context, key WindowKey, reason CloseReason) error {
	return f.finalize(ctx, key, reason, nil)
}

func (f *Finalizer) FinalizeClaimed(
	ctx context.Context,
	key WindowKey,
	reason CloseReason,
	leaseOwner uuid.UUID,
) error {
	return f.finalize(ctx, key, reason, &leaseOwner)
}

func (f *Finalizer) finalize(
	ctx context.Context,
	key WindowKey,
	reason CloseReason,
	leaseOwner *uuid.UUID,
) error {
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
	return f.finalizeUnderLockWithClaim(ctx, key, reason, leaseOwner)
}

func (f *Finalizer) finalizeUnderLock(ctx context.Context, key WindowKey, reason CloseReason) error {
	return f.finalizeUnderLockWithClaim(ctx, key, reason, nil)
}

func (f *Finalizer) finalizeUnderLockWithClaim(
	ctx context.Context,
	key WindowKey,
	reason CloseReason,
	leaseOwner *uuid.UUID,
) error {
	status, revisionHint, err := f.windows.StatusRevision(ctx, key)
	if err != nil {
		return err
	}
	if status == "published" {
		return f.store.DeleteState(ctx, key)
	}
	state, err := f.store.Read(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("PM aggregation window state missing")
		}
		return err
	}
	if err := f.writeFinal(ctx, key, reason, state, leaseOwner, revisionHint); err != nil {
		if errors.Is(err, ErrFinalizeClaimLost) {
			return err
		}
		if f.metrics != nil {
			f.metrics.FinalizeErrorsTotal.Inc()
		}
		_ = f.windows.MarkFailed(ctx, key, err)
		return err
	}
	if f.metrics != nil {
		f.metrics.WindowsFinalizedTotal.WithLabelValues(string(reason)).Inc()
	}
	published, err := f.windows.IsPublished(ctx, key)
	if err != nil {
		return err
	}
	if published {
		if err := f.store.DeleteState(ctx, key); err != nil {
			f.logger.Warn("delete published PM aggregation Redis window",
				zap.String("task_version_id", key.TaskVersionID.String()),
				zap.Time("window_start", key.Start),
				zap.Error(err))
		}
	} else if f.metrics != nil && key.Granularity == GranularityHourly {
		f.metrics.WindowsPreparedTotal.Inc()
	}
	return nil
}

func (f *Finalizer) PublishHourlyReady(
	ctx context.Context,
	now time.Time,
	grace time.Duration,
	limit uint64,
) error {
	started := time.Now()
	if f.metrics != nil {
		defer func() { f.metrics.PublicationDuration.Observe(time.Since(started).Seconds()) }()
	}
	if limit == 0 {
		limit = 1
	}
	const maxPublicationDrainBatches = 64
	for batch := 0; batch < maxPublicationDrainBatches; batch++ {
		keys, err := f.windows.PublishHourlyReady(ctx, now, grace, limit)
		if err != nil {
			return err
		}
		if f.metrics != nil {
			f.metrics.WindowsPublishedTotal.Add(float64(len(keys)))
		}
		for _, key := range keys {
			if err := f.store.DeleteState(ctx, key); err != nil {
				f.logger.Warn("delete watermarked PM aggregation Redis window",
					zap.String("task_version_id", key.TaskVersionID.String()),
					zap.String("entity_key", key.EntityKey),
					zap.Time("window_start", key.Start),
					zap.Error(err))
			}
		}
		if uint64(len(keys)) < limit {
			return nil
		}
	}
	return nil
}

func (f *Finalizer) writeFinal(
	ctx context.Context,
	key WindowKey,
	reason CloseReason,
	state WindowState,
	leaseOwner *uuid.UUID,
	revisionHint int,
) error {
	if !requiresPublicationLock(revisionHint) {
		if err := f.ensureInitialPublication(ctx, key); err != nil {
			return err
		}
	}
	tx, err := f.windows.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin finalize PM aggregation window: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if requiresPublicationLock(revisionHint) {
		if err := lockPublicationForRebuild(ctx, tx, key); err != nil {
			return err
		}
	}
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
	if requiresPublicationLock(revision) != requiresPublicationLock(revisionHint) {
		return fmt.Errorf("PM aggregation revision changed while finalizing: hint=%d current=%d", revisionHint, revision)
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
	claimBuilder := finalizationClaimUpdateForOwner(
		key, reason, state, coverage, version, leaseOwner,
	)
	claimSQL, claimArgs, err := claimBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("build claim PM aggregation window SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, claimSQL, claimArgs...)
	if err != nil {
		return fmt.Errorf("claim PM aggregation window: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if leaseOwner != nil {
			return ErrFinalizeClaimLost
		}
		return nil
	}
	status := finalWindowStatus(key.Granularity, revision)
	if requiresPublicationLock(revision) {
		if err := preparePublicationRevision(ctx, tx, key, revision); err != nil {
			return err
		}
	}
	for _, payload := range rollups {
		if err := insertRollupTxWithEligibility(
			ctx, tx, payload, key.TaskVersionID, status == "published", revision,
		); err != nil {
			return fmt.Errorf("persist compact PM Counter rollup: %w", err)
		}
	}
	rollupEventIDs := make([]uuid.UUID, 0, len(rollups))
	for _, payload := range rollups {
		rollupEventIDs = append(rollupEventIDs, rollupEventIDForRevision(payload.EventID, revision))
	}
	if err := deleteStaleRollupRevisionTx(ctx, tx, key, revision, rollupEventIDs); err != nil {
		return err
	}
	resultReplaceStarted := time.Now()
	resultCount, err := ReplaceWindowResults(
		ctx,
		tx,
		key,
		revision,
		finalMetrics,
		resultCompleteness{
			version:       version,
			coverage:      coverage,
			receivedSlots: state.ReceivedSlots,
		},
	)
	if err != nil {
		return err
	}
	if f.metrics != nil {
		f.metrics.ResultReplaceSeconds.Observe(time.Since(resultReplaceStarted).Seconds())
	}
	publishBuilder := storage.Psql.Update("pm_aggregation_windows").
		Set("status", status).
		Set("result_count", resultCount).
		Set("updated_at", time.Now().UTC()).
		Set("last_error", nil).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": "finalizing"})
	if status == "published" {
		publishBuilder = publishBuilder.Set("published_at", time.Now().UTC())
	} else {
		publishBuilder = publishBuilder.Set("published_at", nil)
	}
	publishSQL, publishArgs, err := publishBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("build publish PM aggregation window SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, publishSQL, publishArgs...); err != nil {
		return fmt.Errorf("publish PM aggregation window: %w", err)
	}
	if status == "prepared" && requiresPublicationLock(revision) {
		preparedSQL, preparedArgs, err := markPublicationPreparedQuery(key, revision).ToSql()
		if err != nil {
			return fmt.Errorf("build mark PM publication prepared: %w", err)
		}
		if _, err := tx.Exec(ctx, preparedSQL, preparedArgs...); err != nil {
			return fmt.Errorf("mark PM publication prepared: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit PM aggregation window: %w", err)
	}
	if f.metrics != nil && shouldRecordDailyVersionExpectedSlotsMismatch(revision, coverage) {
		f.metrics.DailyVersionExpectedSlotsMismatchTotal.Inc()
	}
	return nil
}

type initialPublicationKey struct {
	TaskVersionID uuid.UUID
	Granularity   Granularity
	WindowStart   time.Time
}

func (f *Finalizer) ensureInitialPublication(ctx context.Context, key WindowKey) error {
	cacheKey := initialPublicationKey{
		TaskVersionID: key.TaskVersionID,
		Granularity:   key.Granularity,
		WindowStart:   key.Start,
	}
	if _, loaded := f.initialPublications.LoadOrStore(cacheKey, struct{}{}); loaded {
		return nil
	}
	query, args, err := ensureInitialPublicationQuery(key).ToSql()
	if err != nil {
		f.initialPublications.Delete(cacheKey)
		return fmt.Errorf("build initial PM publication SQL: %w", err)
	}
	if _, err := f.windows.pool.Exec(ctx, query, args...); err != nil {
		f.initialPublications.Delete(cacheKey)
		return fmt.Errorf("ensure initial PM publication: %w", err)
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
		DailyVersionExpectedSlotsMismatch: dailyVersionExpectedSlotsMismatch(
			key, version, state.ExpectedSlots, naturalExpectedSlots(version, key, state),
		),
	}
}

func dailyVersionExpectedSlotsMismatch(
	key WindowKey,
	version *TaskVersionSnapshot,
	versionExpectedSlots, naturalSlots int64,
) bool {
	return key.Granularity == GranularityDaily && version != nil && !version.DevicePipeline &&
		versionExpectedSlots != naturalSlots
}

func shouldRecordDailyVersionExpectedSlotsMismatch(revision int, coverage finalizationCoverage) bool {
	return revision == 1 && coverage.DailyVersionExpectedSlotsMismatch
}

func (f *Finalizer) finalizationLocation() *time.Location {
	return currentLocation(f.location)
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
