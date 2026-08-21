package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type OutboxRecord struct {
	EventID      uuid.UUID
	SourceFileID uuid.UUID
	Payload      event.PMAggregationNormalizedPayload
	CreatedAt    time.Time
	ClaimToken   uuid.UUID
}

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func staleBarrierRedeliveryUpdate(table string, before time.Time) sq.UpdateBuilder {
	return storage.Psql.Update(table).
		Set("published_at", nil).
		Set("claim_token", nil).
		Set("claim_expires_at", nil).
		Set("next_attempt_at", sq.Expr("'-infinity'::timestamptz")).
		Set("last_error", nil).
		Where("published_at IS NOT NULL").
		Where("consumed_at IS NULL").
		Where(sq.Eq{"barrier_eligible": true}).
		Where(sq.Lt{"published_at": before})
}

func pendingOutboxSelect(table string, columns ...string) sq.SelectBuilder {
	return storage.Psql.Select(columns...).
		From(table).
		Where("published_at IS NULL").
		Where("consumed_at IS NULL")
}

func outboxClaimDue(now time.Time, prefix string) sq.Sqlizer {
	column := func(name string) string {
		if prefix == "" {
			return name
		}
		return prefix + "." + name
	}
	return sq.And{
		sq.LtOrEq{column("next_attempt_at"): now},
		sq.Or{
			sq.Eq{column("claim_token"): nil},
			sq.Eq{column("claim_expires_at"): nil},
			sq.LtOrEq{column("claim_expires_at"): now},
		},
	}
}

func markOutboxPublishedUpdate(
	table string,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	now time.Time,
) sq.UpdateBuilder {
	return storage.Psql.Update(table).
		Set("published_at", now).
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", nil).
		Set("claim_token", nil).
		Set("claim_expires_at", nil).
		Where(sq.Eq{"event_id": eventID, "claim_token": claimToken}).
		Where(sq.Gt{"claim_expires_at": now}).
		Where("published_at IS NULL").
		Where("consumed_at IS NULL")
}

func markOutboxFailedUpdate(
	table string,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	publishErr error,
	now time.Time,
	retryAfter time.Duration,
) sq.UpdateBuilder {
	nextAttemptAt := now
	if retryAfter > 0 {
		nextAttemptAt = now.Add(retryAfter)
	}
	return storage.Psql.Update(table).
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", publishErr.Error()).
		Set("claim_token", nil).
		Set("claim_expires_at", nil).
		Set("next_attempt_at", nextAttemptAt).
		Where(sq.Eq{"event_id": eventID, "claim_token": claimToken}).
		Where(sq.Gt{"claim_expires_at": now}).
		Where("published_at IS NULL").
		Where("consumed_at IS NULL")
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func InsertOutbox(
	ctx context.Context,
	tx pgx.Tx,
	payload event.PMAggregationNormalizedPayload,
	maxBytes int,
) error {
	data, err := MarshalEvent(payload, maxBytes)
	if err != nil {
		return err
	}
	query, args, err := storage.Psql.Insert("pm_aggregation_outbox").
		Columns(
			"event_id", "source_file_id", "ingest_batch_id", "device_id",
			"event_window_start", "event_window_end", "payload", "barrier_eligible",
		).
		Values(
			payload.EventID, payload.SourceFileID, payload.IngestBatchID, payload.DeviceID,
			payload.WindowStart, payload.WindowEnd, json.RawMessage(data), true,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM aggregation outbox insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert PM aggregation outbox: %w", err)
	}
	replayQuery, replayArgs, err := storage.Psql.Insert("pm_aggregation_replay_sources").
		Columns("event_window_start", "event_id", "device_id", "payload", "created_at").
		Values(payload.WindowStart, payload.EventID, payload.DeviceID, json.RawMessage(data), time.Now().UTC()).
		Suffix("ON CONFLICT (event_window_start, event_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM replay source insert: %w", err)
	}
	if _, err := tx.Exec(ctx, replayQuery, replayArgs...); err != nil {
		return fmt.Errorf("insert PM replay source: %w", err)
	}
	return nil
}

// MarkConsumed advances the durable consumer barrier only after every
// contribution from the event has been committed to its Redis/SQL window.
func (r *OutboxRepository) MarkConsumed(ctx context.Context, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_aggregation_outbox").
		Set("consumed_at", time.Now().UTC()).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM aggregation outbox consumed: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM aggregation outbox consumed: %w", err)
	}
	return nil
}

// RequeueStaleUnconsumed closes the publish/ack crash gap. Publishing and
// marking the durable consumer barrier are separate commits, so a process
// termination between them must not block the event-time watermark forever.
func (r *OutboxRepository) RequeueStaleUnconsumed(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	query, args, err := staleBarrierRedeliveryUpdate(
		"pm_aggregation_outbox", before,
	).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build requeue stale PM aggregation outbox SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("requeue stale PM aggregation outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *OutboxRepository) ClaimBatch(
	ctx context.Context,
	limit uint64,
	now time.Time,
	lease time.Duration,
) ([]*OutboxRecord, error) {
	if limit == 0 {
		limit = 1
	}
	if now.IsZero() {
		return nil, fmt.Errorf("claim PM aggregation outbox batch: current time is required")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("claim PM aggregation outbox batch: lease must be positive")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin PM aggregation outbox claim tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query, args, err := pendingOutboxSelect(
		"pm_aggregation_outbox",
		"event_id", "source_file_id", "payload", "created_at",
	).
		Where(outboxClaimDue(now, "")).
		OrderBy("created_at", "event_id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lock PM aggregation outbox batch SQL: %w", err)
	}
	dbRows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()
	records := make([]*OutboxRecord, 0, limit)
	for dbRows.Next() {
		var row OutboxRecord
		var raw []byte
		if err := dbRows.Scan(&row.EventID, &row.SourceFileID, &raw, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan PM aggregation outbox batch: %w", err)
		}
		if err := json.Unmarshal(raw, &row.Payload); err != nil {
			return nil, fmt.Errorf("decode PM aggregation outbox payload: %w", err)
		}
		records = append(records, &row)
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation outbox batch: %w", err)
	}
	if len(records) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit empty PM aggregation outbox claim: %w", err)
		}
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(records))
	claimToken := uuid.New()
	claimExpiresAt := now.Add(lease)
	for _, record := range records {
		ids = append(ids, record.EventID)
		record.ClaimToken = claimToken
	}
	update, updateArgs, err := storage.Psql.Update("pm_aggregation_outbox").
		Set("claim_token", claimToken).
		Set("claim_expires_at", claimExpiresAt).
		Where(sq.Eq{"event_id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim PM aggregation outbox batch SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, update, updateArgs...)
	if err != nil {
		return nil, fmt.Errorf("claim PM aggregation outbox batch: %w", err)
	}
	if tag.RowsAffected() != int64(len(records)) {
		return nil, fmt.Errorf("claim PM aggregation outbox batch: claimed %d rows, want %d", tag.RowsAffected(), len(records))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit PM aggregation outbox claim: %w", err)
	}
	return records, nil
}

func (r *OutboxRepository) MarkPublished(
	ctx context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	now time.Time,
) (bool, error) {
	query, args, err := markOutboxPublishedUpdate(
		"pm_aggregation_outbox", eventID, claimToken, now,
	).ToSql()
	if err != nil {
		return false, fmt.Errorf("build mark outbox published SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("mark PM aggregation outbox published: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *OutboxRepository) MarkFailed(
	ctx context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	publishErr error,
	now time.Time,
	retryAfter time.Duration,
) (bool, error) {
	query, args, err := markOutboxFailedUpdate(
		"pm_aggregation_outbox", eventID, claimToken, publishErr, now, retryAfter,
	).ToSql()
	if err != nil {
		return false, fmt.Errorf("build mark outbox failed SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("mark PM aggregation outbox failed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func devicePeriodReplaySelect(
	deviceID uuid.UUID,
	start, end time.Time,
) sq.SelectBuilder {
	statement := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	filter := func(table string) sq.SelectBuilder {
		return statement.Select("event_window_start", "event_id", "payload").
			From(table).
			Where(sq.Eq{"device_id": deviceID}).
			Where(sq.GtOrEq{"event_window_start": start}).
			Where(sq.Lt{"event_window_start": end})
	}
	legacy := filter("pm_aggregation_outbox").
		Where(sq.Eq{"barrier_eligible": false})
	return filter("pm_aggregation_replay_sources").
		SuffixExpr(sq.Expr("UNION ALL ?", legacy)).
		Suffix("ORDER BY event_window_start, event_id").
		PlaceholderFormat(sq.Dollar)
}

func (r *OutboxRepository) VisitPayloadsForDevicePeriod(
	ctx context.Context,
	deviceID uuid.UUID,
	start, end time.Time,
	visit func(event.PMAggregationNormalizedPayload) error,
) error {
	query, args, err := devicePeriodReplaySelect(deviceID, start, end).ToSql()
	if err != nil {
		return fmt.Errorf("build durable PM device-period replay query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query durable PM device-period source events: %w", err)
	}
	defer rows.Close()
	return visitReplayPayloadRows(rows, visit)
}

func visitReplayPayloadRows(
	rows pgx.Rows,
	visit func(event.PMAggregationNormalizedPayload) error,
) error {
	for rows.Next() {
		var windowStart time.Time
		var eventID uuid.UUID
		var raw []byte
		if err := rows.Scan(&windowStart, &eventID, &raw); err != nil {
			return fmt.Errorf("scan durable PM source event: %w", err)
		}
		var payload event.PMAggregationNormalizedPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("decode durable PM source event: %w", err)
		}
		if err := visit(payload); err != nil {
			return fmt.Errorf("visit durable PM source event: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate durable PM source events: %w", err)
	}
	return nil
}
