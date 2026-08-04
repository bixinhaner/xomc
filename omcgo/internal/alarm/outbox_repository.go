package alarm

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// OutboxClaimRequest defines one short, committed ownership lease. Publishing
// is deliberately performed after Claim commits, never while row locks are held.
type OutboxClaimRequest struct {
	WorkerID      string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

// OutboxFailure is the durable outcome of a failed JetStream publication.
type OutboxFailure struct {
	EventID       uuid.UUID
	WorkerID      string
	AttemptCount  int
	MaxAttempts   int
	NextAttemptAt time.Time
	LastError     string
	Now           time.Time
}

// OutboxStats is the bounded operational view used by relay health metrics.
type OutboxStats struct {
	Backlog         int64
	OldestCreatedAt *time.Time
}

type alarmOutboxRepository interface {
	Claim(context.Context, OutboxClaimRequest) ([]OutboxRecord, error)
	MarkPublished(context.Context, uuid.UUID, string, time.Time) error
	MarkFailed(context.Context, OutboxFailure) (OutboxStatus, error)
	Replay(context.Context, uuid.UUID, time.Time) error
	DeletePublishedBefore(context.Context, time.Time) (int64, error)
	Stats(context.Context, time.Time) (OutboxStats, error)
}

// AlarmOutboxRepository persists the publication state machine in PostgreSQL.
type AlarmOutboxRepository struct {
	db storage.DB
}

func NewAlarmOutboxRepository(pool *pgxpool.Pool) *AlarmOutboxRepository {
	return &AlarmOutboxRepository{db: storage.NewPoolDB(pool)}
}

func newAlarmOutboxRepository(db storage.DB) *AlarmOutboxRepository {
	return &AlarmOutboxRepository{db: db}
}

func buildAlarmOutboxClaimSelect(now, leaseCutoff time.Time, limit int) (string, []any, error) {
	if limit <= 0 {
		limit = 1
	}
	return storage.Psql.Select(
		"id", "event_id", "aggregate_type", "aggregate_id", "aggregate_version",
		"subject", "payload", "status", "attempt_count", "next_attempt_at",
		"locked_by", "locked_at", "published_at", "last_error", "created_at", "updated_at",
	).
		From("alarm_event_outbox").
		Where(sq.Or{
			sq.And{
				sq.Eq{"status": []OutboxStatus{OutboxStatusPending, OutboxStatusFailed}},
				sq.LtOrEq{"next_attempt_at": now},
			},
			sq.And{
				sq.Eq{"status": OutboxStatusPublishing},
				sq.Lt{"locked_at": leaseCutoff},
			},
		}).
		OrderBy("created_at", "event_id").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
}

// Claim leases due pending/failed rows and stale publishing rows in a short
// transaction. The committed lease permits multi-replica relays without a
// process-local coordinator.
func (r *AlarmOutboxRepository) Claim(ctx context.Context, request OutboxClaimRequest) ([]OutboxRecord, error) {
	if request.WorkerID == "" {
		return nil, fmt.Errorf("claim alarm outbox: worker ID is required")
	}
	if request.Now.IsZero() {
		request.Now = time.Now().UTC()
	}
	if request.LeaseDuration <= 0 {
		request.LeaseDuration = time.Minute
	}
	if request.Limit <= 0 {
		request.Limit = 1
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin alarm outbox claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query, args, err := buildAlarmOutboxClaimSelect(
		request.Now, request.Now.Add(-request.LeaseDuration), request.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("build alarm outbox claim select: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select claimable alarm outbox rows: %w", err)
	}
	records := make([]OutboxRecord, 0, request.Limit)
	for rows.Next() {
		var record OutboxRecord
		if err := rows.Scan(
			&record.ID, &record.EventID, &record.AggregateType, &record.AggregateID, &record.AggregateVersion,
			&record.Subject, &record.Payload, &record.Status, &record.AttemptCount, &record.NextAttemptAt,
			&record.LockedBy, &record.LockedAt, &record.PublishedAt, &record.LastError,
			&record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan claimable alarm outbox row: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate claimable alarm outbox rows: %w", err)
	}
	rows.Close()

	if len(records) > 0 {
		ids := make([]uuid.UUID, 0, len(records))
		for i := range records {
			ids = append(ids, records[i].EventID)
		}
		updateQuery, updateArgs, buildErr := storage.Psql.Update("alarm_event_outbox").
			Set("status", OutboxStatusPublishing).
			Set("locked_by", request.WorkerID).
			Set("locked_at", request.Now).
			Set("updated_at", request.Now).
			Where(sq.Eq{"event_id": ids}).
			ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build alarm outbox lease update: %w", buildErr)
		}
		if _, err := tx.Exec(ctx, updateQuery, updateArgs...); err != nil {
			return nil, fmt.Errorf("lease alarm outbox rows: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit alarm outbox claim: %w", err)
	}
	return records, nil
}

func (r *AlarmOutboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, workerID string, now time.Time) error {
	query, args, err := buildMarkAlarmOutboxPublished(eventID, workerID, now)
	if err != nil {
		return fmt.Errorf("build mark alarm outbox published: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark alarm outbox published: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("mark alarm outbox published: lease lost for event %s", eventID)
	}
	return nil
}

func buildMarkAlarmOutboxPublished(eventID uuid.UUID, workerID string, now time.Time) (string, []any, error) {
	return storage.Psql.Update("alarm_event_outbox").
		Set("status", OutboxStatusPublished).
		Set("published_at", now).
		Set("locked_by", nil).
		Set("locked_at", nil).
		Set("last_error", nil).
		Set("updated_at", now).
		Where(sq.Eq{"event_id": eventID, "status": OutboxStatusPublishing, "locked_by": workerID}).
		ToSql()
}

func (r *AlarmOutboxRepository) MarkFailed(ctx context.Context, failure OutboxFailure) (OutboxStatus, error) {
	status := OutboxStatusFailed
	if failure.MaxAttempts > 0 && failure.AttemptCount >= failure.MaxAttempts {
		status = OutboxStatusDead
	}
	query, args, err := storage.Psql.Update("alarm_event_outbox").
		Set("status", status).
		Set("attempt_count", failure.AttemptCount).
		Set("next_attempt_at", failure.NextAttemptAt).
		Set("locked_by", nil).
		Set("locked_at", nil).
		Set("last_error", failure.LastError).
		Set("updated_at", failure.Now).
		Where(sq.Eq{
			"event_id": failure.EventID, "status": OutboxStatusPublishing, "locked_by": failure.WorkerID,
		}).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build mark alarm outbox failed: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("mark alarm outbox failed: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", fmt.Errorf("mark alarm outbox failed: lease lost for event %s", failure.EventID)
	}
	return status, nil
}

// Replay resets only the explicitly selected successfully published event.
func (r *AlarmOutboxRepository) Replay(ctx context.Context, eventID uuid.UUID, now time.Time) error {
	query, args, err := buildReplayAlarmOutboxEvent(eventID, now)
	if err != nil {
		return fmt.Errorf("build replay alarm outbox event: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("replay alarm outbox event: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("replay alarm outbox event: published event %s not found", eventID)
	}
	return nil
}

func buildReplayAlarmOutboxEvent(eventID uuid.UUID, now time.Time) (string, []any, error) {
	return storage.Psql.Update("alarm_event_outbox").
		Set("status", OutboxStatusPending).
		Set("attempt_count", 0).
		Set("next_attempt_at", now).
		Set("locked_by", nil).
		Set("locked_at", nil).
		Set("published_at", nil).
		Set("last_error", nil).
		Set("updated_at", now).
		Where(sq.Eq{"event_id": eventID, "status": OutboxStatusPublished}).
		ToSql()
}

func (r *AlarmOutboxRepository) DeletePublishedBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args, err := buildDeletePublishedAlarmOutboxBefore(before)
	if err != nil {
		return 0, fmt.Errorf("build cleanup published alarm outbox: %w", err)
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup published alarm outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func buildDeletePublishedAlarmOutboxBefore(before time.Time) (string, []any, error) {
	return storage.Psql.Delete("alarm_event_outbox").
		Where(sq.Eq{"status": OutboxStatusPublished}).
		Where(sq.Lt{"published_at": before}).
		ToSql()
}

func (r *AlarmOutboxRepository) Stats(ctx context.Context, _ time.Time) (OutboxStats, error) {
	query, args, err := storage.Psql.Select("COUNT(*)", "MIN(created_at)").
		From("alarm_event_outbox").
		Where(sq.Eq{"status": []OutboxStatus{
			OutboxStatusPending, OutboxStatusPublishing, OutboxStatusFailed,
		}}).
		ToSql()
	if err != nil {
		return OutboxStats{}, fmt.Errorf("build alarm outbox stats: %w", err)
	}
	var stats OutboxStats
	if err := r.db.QueryRow(ctx, query, args...).Scan(&stats.Backlog, &stats.OldestCreatedAt); err != nil {
		return OutboxStats{}, fmt.Errorf("query alarm outbox stats: %w", err)
	}
	return stats, nil
}

var _ alarmOutboxRepository = (*AlarmOutboxRepository)(nil)
