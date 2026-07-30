package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

type RollupOutboxRecord struct {
	EventID uuid.UUID
	Subject string
	Payload RollupPayload
}

var maxRollupEventBytes = DefaultConfig().MaxEventBytes

func SetMaxRollupEventBytes(limit int) {
	if limit > 0 {
		maxRollupEventBytes = limit
	}
}

type RollupOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewRollupOutboxRepository(pool *pgxpool.Pool) *RollupOutboxRepository {
	return &RollupOutboxRepository{pool: pool}
}

func insertRollupTx(ctx context.Context, tx pgx.Tx, payload RollupPayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal PM compact rollup: %w", err)
	}
	if len(data) > maxRollupEventBytes {
		return fmt.Errorf(
			"PM compact rollup is %d bytes, exceeds limit %d",
			len(data), maxRollupEventBytes,
		)
	}
	subject := event.SubjectPMAggregationHourlyRollup
	if payload.SourceGranularity == GranularityDaily {
		subject = event.SubjectPMAggregationDailyRollup
	}
	snapshotSQL, snapshotArgs, err := storage.Psql.Insert("pm_aggregation_counter_rollups").
		Columns(
			"event_id", "task_id", "task_version_id", "entity_key", "granularity",
			"window_start", "window_end", "chunk_index", "chunk_count",
			"complete", "payload",
		).
		Values(
			payload.EventID, payload.TaskID, payload.TaskVersionID, payload.EntityKey,
			string(payload.SourceGranularity), payload.WindowStart, payload.WindowEnd,
			payload.ChunkIndex, payload.ChunkCount, payload.Complete, json.RawMessage(data),
		).
		Suffix(`ON CONFLICT (event_id) DO UPDATE SET
			payload = EXCLUDED.payload,
			complete = EXCLUDED.complete,
			created_at = now()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM Counter rollup snapshot SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, snapshotSQL, snapshotArgs...); err != nil {
		return fmt.Errorf("insert PM Counter rollup snapshot: %w", err)
	}
	outboxSQL, outboxArgs, err := storage.Psql.Insert("pm_aggregation_rollup_outbox").
		Columns(
			"event_id", "subject", "granularity", "window_start", "payload",
			"barrier_eligible",
		).
		Values(
			payload.EventID, subject, string(payload.SourceGranularity),
			payload.WindowStart, json.RawMessage(data), true,
		).
		Suffix(`ON CONFLICT (event_id) DO UPDATE SET
			payload = EXCLUDED.payload,
			published_at = NULL,
			consumed_at = NULL,
			barrier_eligible = true,
			publish_attempts = 0,
			last_error = NULL,
			created_at = now()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM rollup outbox SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, outboxSQL, outboxArgs...); err != nil {
		return fmt.Errorf("insert PM rollup outbox: %w", err)
	}
	return nil
}

func (r *RollupOutboxRepository) MarkConsumed(ctx context.Context, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("consumed_at", time.Now().UTC()).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM rollup outbox consumed: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM rollup outbox consumed: %w", err)
	}
	return nil
}

func (r *RollupOutboxRepository) RequeueStaleUnconsumed(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	query, args, err := staleBarrierRedeliveryUpdate(
		"pm_aggregation_rollup_outbox", before,
	).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build requeue stale PM rollup outbox SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("requeue stale PM rollup outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *RollupOutboxRepository) lockBatch(
	ctx context.Context,
	tx pgx.Tx,
	limit uint64,
) ([]RollupOutboxRecord, error) {
	if limit == 0 {
		limit = 1
	}
	query, args, err := pendingOutboxSelect(
		"pm_aggregation_rollup_outbox", "event_id", "subject", "payload",
	).
		OrderBy("created_at", "event_id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock PM rollup outbox SQL: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("lock PM rollup outbox: %w", err)
	}
	defer rows.Close()
	out := make([]RollupOutboxRecord, 0, limit)
	for rows.Next() {
		var record RollupOutboxRecord
		var raw []byte
		if err := rows.Scan(&record.EventID, &record.Subject, &raw); err != nil {
			return nil, fmt.Errorf("scan PM rollup outbox: %w", err)
		}
		if err := json.Unmarshal(raw, &record.Payload); err != nil {
			return nil, fmt.Errorf("decode PM rollup outbox: %w", err)
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func markRollupPublished(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("published_at", time.Now().UTC()).
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", nil).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, query, args...)
	return err
}

func markRollupFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, publishErr error) error {
	query, args, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", publishErr.Error()).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, query, args...)
	return err
}

func (r *RollupOutboxRepository) DeletePublishedBefore(ctx context.Context, before time.Time) error {
	query, args, err := storage.Psql.Delete("pm_aggregation_rollup_outbox").
		Where(sq.Lt{"published_at": before}).
		Where("consumed_at IS NOT NULL").
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *RollupOutboxRepository) ListSnapshots(
	ctx context.Context,
	taskVersionID uuid.UUID,
	entityKey string,
	granularity Granularity,
	start, end time.Time,
) ([]RollupPayload, error) {
	query, args, err := storage.Psql.Select("payload").
		From("pm_aggregation_counter_rollups").
		Where(sq.Eq{
			"task_version_id": taskVersionID,
			"entity_key":      entityKey,
			"granularity":     string(granularity),
		}).
		Where(sq.GtOrEq{"window_start": start}).
		Where(sq.Lt{"window_start": end}).
		OrderBy("window_start", "chunk_index").
		ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM Counter rollup snapshots: %w", err)
	}
	defer rows.Close()
	var out []RollupPayload
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var payload RollupPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode PM Counter rollup snapshot: %w", err)
		}
		out = append(out, payload)
	}
	return out, rows.Err()
}

func counterRollupPeriodSelect(
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
) sq.SelectBuilder {
	return storage.Psql.Select("payload").
		From("pm_aggregation_counter_rollups").
		Where(sq.Eq{
			"task_version_id": taskVersionIDs,
			"granularity":     string(granularity),
		}).
		Where(sq.GtOrEq{"window_start": start}).
		Where(sq.Lt{"window_start": end}).
		OrderBy("window_start", "task_version_id", "entity_key", "chunk_index")
}

func (r *RollupOutboxRepository) VisitSnapshotsForPeriod(
	ctx context.Context,
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
	visit func(RollupPayload) error,
) error {
	if len(taskVersionIDs) == 0 {
		return fmt.Errorf("PM Counter rollup replay task versions are empty")
	}
	query, args, err := counterRollupPeriodSelect(
		taskVersionIDs, granularity, start, end,
	).ToSql()
	if err != nil {
		return err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query PM Counter rollup period snapshots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var payload RollupPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("decode PM Counter rollup period snapshot: %w", err)
		}
		if err := visit(payload); err != nil {
			return err
		}
	}
	return rows.Err()
}

type rebuildSnapshotSource interface {
	VisitSnapshotsForPeriod(
		context.Context,
		[]uuid.UUID,
		Granularity,
		time.Time,
		time.Time,
		func(RollupPayload) error,
	) error
}

func visitSnapshotsForRebuildBatch(
	ctx context.Context,
	source rebuildSnapshotSource,
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
	dispatch func(RollupPayload) error,
) (int, error) {
	rows := 0
	err := source.VisitSnapshotsForPeriod(
		ctx, taskVersionIDs, granularity, start, end,
		func(payload RollupPayload) error {
			rows++
			return dispatch(payload)
		},
	)
	return rows, err
}

type RollupOutboxRelay struct {
	repo            *RollupOutboxRepository
	bus             event.EventBus
	logger          *zap.Logger
	metrics         *Metrics
	batch           int
	interval        time.Duration
	redeliveryAfter time.Duration
	redeliveryEvery time.Duration
}

func NewRollupOutboxRelay(
	repo *RollupOutboxRepository,
	bus event.EventBus,
	logger *zap.Logger,
) *RollupOutboxRelay {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RollupOutboxRelay{
		repo: repo, bus: bus, logger: logger, batch: 100, interval: 200 * time.Millisecond,
		redeliveryAfter: 5 * time.Minute, redeliveryEvery: time.Minute,
	}
}

func (r *RollupOutboxRelay) SetBatch(batch int) *RollupOutboxRelay {
	if batch > 0 {
		r.batch = batch
	}
	return r
}

func (r *RollupOutboxRelay) SetMetrics(metrics *Metrics) *RollupOutboxRelay {
	r.metrics = metrics
	return r
}

func (r *RollupOutboxRelay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	cleanup := time.NewTicker(time.Hour)
	defer ticker.Stop()
	defer cleanup.Stop()
	go runRedeliveryLoop(ctx, r.redeliveryEvery, func() {
		r.requeueStale(ctx)
	})
	for {
		published, err := r.publishBatch(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			if r.metrics != nil {
				r.metrics.RollupOutboxErrorsTotal.Inc()
			}
			r.logger.Warn("publish PM rollup outbox", zap.Error(err))
		}
		if published {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cleanup.C:
			if err := r.repo.DeletePublishedBefore(ctx, time.Now().UTC().Add(-24*time.Hour)); err != nil {
				r.logger.Warn("cleanup PM rollup outbox", zap.Error(err))
			}
		case <-ticker.C:
		}
	}
}

func (r *RollupOutboxRelay) requeueStale(ctx context.Context) {
	count, err := r.repo.RequeueStaleUnconsumed(
		ctx, time.Now().UTC().Add(-r.redeliveryAfter),
	)
	if err != nil {
		if r.metrics != nil {
			r.metrics.RollupOutboxErrorsTotal.Inc()
		}
		r.logger.Warn("requeue stale PM rollup outbox", zap.Error(err))
		return
	}
	if count > 0 {
		r.logger.Warn("requeued stale PM rollup events after consumer acknowledgement timeout",
			zap.Int64("events", count),
			zap.Duration("ack_timeout", r.redeliveryAfter))
	}
}

func (r *RollupOutboxRelay) publishBatch(ctx context.Context) (bool, error) {
	tx, err := r.repo.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin PM rollup outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := r.repo.lockBatch(ctx, tx, uint64(r.batch))
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	var publishErrors []error
	publishedCount := 0
	for _, row := range rows {
		envelope, eventErr := event.NewEvent(row.Subject, row.Payload)
		if eventErr != nil {
			return false, eventErr
		}
		envelope.ID = row.EventID.String()
		if publishErr := r.bus.Publish(ctx, row.Subject, envelope); publishErr != nil {
			if markErr := markRollupFailed(ctx, tx, row.EventID, publishErr); markErr != nil {
				return false, errors.Join(publishErr, markErr)
			}
			publishErrors = append(publishErrors, publishErr)
			continue
		}
		if err := markRollupPublished(ctx, tx, row.EventID); err != nil {
			return false, err
		}
		publishedCount++
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	if r.metrics != nil {
		r.metrics.RollupOutboxPublishedTotal.Add(float64(publishedCount))
	}
	if len(publishErrors) > 0 {
		return true, errors.Join(publishErrors...)
	}
	return true, nil
}
