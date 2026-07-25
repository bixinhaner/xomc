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
		Columns("event_id", "source_file_id", "ingest_batch_id", "payload").
		Values(payload.EventID, payload.SourceFileID, payload.IngestBatchID, json.RawMessage(data)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM aggregation outbox insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert PM aggregation outbox: %w", err)
	}
	return nil
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
	query, args, err := storage.Psql.Select(
		"event_id", "source_file_id", "payload", "created_at",
	).From("pm_aggregation_outbox").
		Where("published_at IS NULL").
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

func (r *OutboxRepository) DeletePublishedBefore(ctx context.Context, before time.Time) error {
	query, args, err := storage.Psql.Delete("pm_aggregation_outbox").
		Where(sq.Lt{"published_at": before}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build cleanup PM aggregation outbox SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("cleanup PM aggregation outbox: %w", err)
	}
	return nil
}
