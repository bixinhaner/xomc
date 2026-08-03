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
}

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func staleBarrierRedeliveryUpdate(table string, before time.Time) sq.UpdateBuilder {
	return storage.Psql.Update(table).
		Set("published_at", nil).
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
			"event_id", "source_file_id", "ingest_batch_id",
			"event_window_start", "event_window_end", "payload", "barrier_eligible",
		).
		Values(
			payload.EventID, payload.SourceFileID, payload.IngestBatchID,
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
		Columns("event_window_start", "event_id", "payload", "created_at").
		Values(payload.WindowStart, payload.EventID, json.RawMessage(data), time.Now().UTC()).
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

func (r *OutboxRepository) DeleteReplayBefore(ctx context.Context, before time.Time) error {
	if _, err := r.pool.Exec(
		ctx,
		"SELECT drop_chunks('public.pm_aggregation_replay_sources', older_than => $1::timestamptz)",
		before,
	); err != nil {
		return fmt.Errorf("cleanup PM replay source chunks: %w", err)
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

func (r *OutboxRepository) lockNext(ctx context.Context, tx pgx.Tx) (*OutboxRecord, error) {
	rows, err := r.lockBatch(ctx, tx, 1)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, pgx.ErrNoRows
	}
	return rows[0], nil
}

func (r *OutboxRepository) lockBatch(ctx context.Context, tx pgx.Tx, limit uint64) ([]*OutboxRecord, error) {
	if limit == 0 {
		limit = 1
	}
	query, args, err := pendingOutboxSelect(
		"pm_aggregation_outbox",
		"event_id", "source_file_id", "payload", "created_at",
	).
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
	return records, nil
}

func markOutboxPublished(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_aggregation_outbox").
		Set("published_at", time.Now().UTC()).
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", nil).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark outbox published SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM aggregation outbox published: %w", err)
	}
	return nil
}

func markOutboxFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, publishErr error) error {
	query, args, err := storage.Psql.Update("pm_aggregation_outbox").
		Set("publish_attempts", sq.Expr("publish_attempts + 1")).
		Set("last_error", publishErr.Error()).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark outbox failed SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM aggregation outbox failed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) DeletePublishedBefore(
	ctx context.Context,
	before, legacyBefore time.Time,
) error {
	query, args, err := storage.Psql.Delete("pm_aggregation_outbox").
		Where(sq.Or{
			sq.And{
				sq.Lt{"published_at": before},
				sq.Expr("consumed_at IS NOT NULL"),
				sq.Eq{"barrier_eligible": true},
			},
			sq.And{
				sq.Lt{"published_at": legacyBefore},
				sq.Eq{"barrier_eligible": false},
			},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build cleanup PM aggregation outbox SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("cleanup PM aggregation outbox: %w", err)
	}
	return nil
}

func (r *OutboxRepository) VisitPayloadsForPeriod(
	ctx context.Context,
	start, end time.Time,
	visit func(event.PMAggregationNormalizedPayload) error,
) error {
	query := `
SELECT payload
FROM (
    SELECT event_window_start, event_id, payload
    FROM pm_aggregation_replay_sources
    WHERE event_window_start >= $1 AND event_window_start < $2
    UNION ALL
    SELECT event_window_start, event_id, payload
    FROM pm_aggregation_outbox
    WHERE barrier_eligible = false
      AND event_window_start >= $1 AND event_window_start < $2
) replay
ORDER BY event_window_start, event_id`
	args := []any{start, end}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query durable PM source events: %w", err)
	}
	defer rows.Close()
	return visitReplayPayloadRows(rows, visit)
}

func visitReplayPayloadRows(
	rows pgx.Rows,
	visit func(event.PMAggregationNormalizedPayload) error,
) error {
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
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
