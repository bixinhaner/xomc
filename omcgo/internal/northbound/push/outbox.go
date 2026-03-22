package push

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// OutboxStatus represents the processing status of an outbox entry.
type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusDelivered  OutboxStatus = "delivered"
	OutboxStatusDead       OutboxStatus = "dead"
)

// OutboxEntry represents a single row in the northbound_outbox table.
type OutboxEntry struct {
	ID          uuid.UUID       `json:"id"`
	EventID     string          `json:"event_id"`
	Subject     string          `json:"subject"`
	Payload     json.RawMessage `json:"payload"`
	TargetID    string          `json:"target_id"`
	Status      OutboxStatus    `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	LastError   string          `json:"last_error,omitempty"`
	NextRetryAt time.Time       `json:"next_retry_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// OutboxRepository provides persistence operations for the northbound outbox.
type OutboxRepository interface {
	// Insert adds a new outbox entry. Ignores duplicates (same event_id + target_id).
	Insert(ctx context.Context, entry *OutboxEntry) error
	// FetchPending retrieves up to limit entries that are ready for processing.
	FetchPending(ctx context.Context, limit int) ([]OutboxEntry, error)
	// MarkProcessing transitions an entry to processing status.
	MarkProcessing(ctx context.Context, id uuid.UUID) error
	// MarkDelivered transitions an entry to delivered status.
	MarkDelivered(ctx context.Context, id uuid.UUID) error
	// MarkFailed records a failed attempt and schedules the next retry.
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, nextRetry time.Time) error
	// MarkDead transitions an entry to dead letter status.
	MarkDead(ctx context.Context, id uuid.UUID, errMsg string) error
	// ListDead returns dead letter entries for manual inspection.
	ListDead(ctx context.Context, limit, offset int) ([]OutboxEntry, int, error)
	// Replay moves a dead entry back to pending for re-delivery.
	Replay(ctx context.Context, id uuid.UUID) error
}

// PgOutboxRepository implements OutboxRepository using PostgreSQL.
type PgOutboxRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	sb     squirrel.StatementBuilderType
}

// NewPgOutboxRepository creates a new PgOutboxRepository.
func NewPgOutboxRepository(pool *pgxpool.Pool, logger *zap.Logger) *PgOutboxRepository {
	return &PgOutboxRepository{
		pool:   pool,
		logger: logger,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *PgOutboxRepository) Insert(ctx context.Context, entry *OutboxEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	query, args, err := r.sb.
		Insert("northbound_outbox").
		Columns("id", "event_id", "subject", "payload", "target_id", "status", "max_attempts", "next_retry_at").
		Values(entry.ID, entry.EventID, entry.Subject, entry.Payload, entry.TargetID, OutboxStatusPending, entry.MaxAttempts, entry.NextRetryAt).
		Suffix("ON CONFLICT (event_id, target_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build outbox insert: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec outbox insert: %w", err)
	}
	return nil
}

func (r *PgOutboxRepository) FetchPending(ctx context.Context, limit int) ([]OutboxEntry, error) {
	query, args, err := r.sb.
		Select("id", "event_id", "subject", "payload", "target_id", "status", "attempts", "max_attempts", "last_error", "next_retry_at", "created_at", "updated_at").
		From("northbound_outbox").
		Where(squirrel.And{
			squirrel.Eq{"status": []OutboxStatus{OutboxStatusPending, OutboxStatusProcessing}},
			squirrel.LtOrEq{"next_retry_at": time.Now()},
		}).
		OrderBy("next_retry_at ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build outbox fetch: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query outbox: %w", err)
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var lastError *string
		if err := rows.Scan(&e.ID, &e.EventID, &e.Subject, &e.Payload, &e.TargetID,
			&e.Status, &e.Attempts, &e.MaxAttempts, &lastError,
			&e.NextRetryAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		if lastError != nil {
			e.LastError = *lastError
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PgOutboxRepository) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.sb.
		Update("northbound_outbox").
		Set("status", OutboxStatusProcessing).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark processing: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec mark processing: %w", err)
	}
	return nil
}

func (r *PgOutboxRepository) MarkDelivered(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.sb.
		Update("northbound_outbox").
		Set("status", OutboxStatusDelivered).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark delivered: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec mark delivered: %w", err)
	}
	return nil
}

func (r *PgOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, nextRetry time.Time) error {
	query, args, err := r.sb.
		Update("northbound_outbox").
		Set("status", OutboxStatusPending).
		Set("attempts", squirrel.Expr("attempts + 1")).
		Set("last_error", errMsg).
		Set("next_retry_at", nextRetry).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark failed: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec mark failed: %w", err)
	}
	return nil
}

func (r *PgOutboxRepository) MarkDead(ctx context.Context, id uuid.UUID, errMsg string) error {
	query, args, err := r.sb.
		Update("northbound_outbox").
		Set("status", OutboxStatusDead).
		Set("attempts", squirrel.Expr("attempts + 1")).
		Set("last_error", errMsg).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark dead: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec mark dead: %w", err)
	}
	return nil
}

func (r *PgOutboxRepository) ListDead(ctx context.Context, limit, offset int) ([]OutboxEntry, int, error) {
	// Count total dead entries.
	countQuery, countArgs, err := r.sb.
		Select("COUNT(*)").
		From("northbound_outbox").
		Where(squirrel.Eq{"status": OutboxStatusDead}).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build dead count: %w", err)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("query dead count: %w", err)
	}

	// Fetch dead entries.
	query, args, err := r.sb.
		Select("id", "event_id", "subject", "payload", "target_id", "status", "attempts", "max_attempts", "last_error", "next_retry_at", "created_at", "updated_at").
		From("northbound_outbox").
		Where(squirrel.Eq{"status": OutboxStatusDead}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build dead list: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query dead list: %w", err)
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var lastError *string
		if err := rows.Scan(&e.ID, &e.EventID, &e.Subject, &e.Payload, &e.TargetID,
			&e.Status, &e.Attempts, &e.MaxAttempts, &lastError,
			&e.NextRetryAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan dead row: %w", err)
		}
		if lastError != nil {
			e.LastError = *lastError
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (r *PgOutboxRepository) Replay(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.sb.
		Update("northbound_outbox").
		Set("status", OutboxStatusPending).
		Set("attempts", 0).
		Set("last_error", nil).
		Set("next_retry_at", time.Now()).
		Set("updated_at", time.Now()).
		Where(squirrel.And{
			squirrel.Eq{"id": id},
			squirrel.Eq{"status": OutboxStatusDead},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build replay: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec replay: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("outbox entry %s not found or not in dead state", id)
	}
	return nil
}
