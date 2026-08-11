package notification

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

	"github.com/omcgo/omcgo/internal/core/storage"
)

type ProtectedStatusSummaryRecipient struct {
	Ciphertext  []byte
	KeyVersion  int
	Fingerprint []byte
}

type StatusSummaryRun struct {
	ID          uuid.UUID
	RunKey      string
	Snapshot    ZedStatusSummarySnapshot
	GeneratedAt time.Time
	TimeZone    string
	State       string
	Attempt     int
	NextAttempt time.Time
	LastError   string
}

type StatusSummaryRunRecipient struct {
	ID          uuid.UUID
	Ciphertext  []byte
	KeyVersion  int
	Fingerprint []byte
	State       string
	Attempt     int
}

type StatusSummaryRecipientAttempt struct {
	ID          uuid.UUID
	RecipientID uuid.UUID
	AttemptNo   int
	StartedAt   time.Time
}

type StatusSummaryRunRepository interface {
	HasRun(context.Context, string) (bool, error)
	EnsureRun(context.Context, string, ZedStatusSummarySnapshot, []ProtectedStatusSummaryRecipient) (*StatusSummaryRun, bool, error)
	ClaimRuns(context.Context, time.Time, int) ([]StatusSummaryRun, error)
	ListRecipients(context.Context, uuid.UUID) ([]StatusSummaryRunRecipient, error)
	RequeueStaleRecipients(context.Context, uuid.UUID, time.Time) error
	ClaimRecipient(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	StartRecipientAttempt(context.Context, uuid.UUID, time.Time) (*StatusSummaryRecipientAttempt, error)
	FinishRecipientAttempt(context.Context, uuid.UUID, time.Time, string, *string, *string) error
	MarkRecipientSucceeded(context.Context, uuid.UUID) error
	MarkRecipientFailed(context.Context, uuid.UUID, string) error
	MarkRecipientUnknown(context.Context, uuid.UUID, string) error
	MarkRunRetry(context.Context, uuid.UUID, string, time.Time) error
	MarkRunFailed(context.Context, uuid.UUID, string) error
	MarkRunSucceeded(context.Context, uuid.UUID) error
}

type PgStatusSummaryRunRepository struct{ db storage.DB }

func NewPgStatusSummaryRunRepository(pool *pgxpool.Pool) *PgStatusSummaryRunRepository {
	return &PgStatusSummaryRunRepository{db: storage.NewPoolDB(pool)}
}

func (r *PgStatusSummaryRunRepository) HasRun(ctx context.Context, runKey string) (bool, error) {
	query, args, err := storage.Psql.Select("1").From("notification_status_summary_runs").Where(sq.Eq{"run_key": runKey}).Limit(1).ToSql()
	if err != nil {
		return false, fmt.Errorf("build status summary run existence query: %w", err)
	}
	var marker int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&marker); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("check status summary run existence: %w", err)
	}
	return marker == 1, nil
}

func (r *PgStatusSummaryRunRepository) EnsureRun(
	ctx context.Context,
	runKey string,
	snapshot ZedStatusSummarySnapshot,
	recipients []ProtectedStatusSummaryRecipient,
) (*StatusSummaryRun, bool, error) {
	if runKey == "" {
		return nil, false, fmt.Errorf("ensure status summary run: run key is required")
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, false, fmt.Errorf("marshal status summary snapshot: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin status summary run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query, args, err := storage.Psql.Insert("notification_status_summary_runs").
		Columns("run_key", "generated_at", "time_zone", "snapshot", "excluded_cpe").
		Values(runKey, snapshot.GeneratedAt.UTC(), snapshot.TimeZone, payload, snapshot.ExcludedCPE).
		Suffix("ON CONFLICT (run_key) DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return nil, false, fmt.Errorf("build status summary run insert: %w", err)
	}
	var runID uuid.UUID
	created := true
	if err := tx.QueryRow(ctx, query, args...).Scan(&runID); errors.Is(err, pgx.ErrNoRows) {
		created = false
		lookup, lookupArgs, buildErr := storage.Psql.Select("id").
			From("notification_status_summary_runs").Where(sq.Eq{"run_key": runKey}).ToSql()
		if buildErr != nil {
			return nil, false, fmt.Errorf("build status summary run lookup: %w", buildErr)
		}
		if err := tx.QueryRow(ctx, lookup, lookupArgs...).Scan(&runID); err != nil {
			return nil, false, fmt.Errorf("lookup status summary run: %w", err)
		}
	} else if err != nil {
		return nil, false, fmt.Errorf("insert status summary run: %w", err)
	}
	if created {
		for _, recipient := range recipients {
			query, args, err := storage.Psql.Insert("notification_status_summary_recipients").
				Columns("run_id", "address_ciphertext", "address_key_version", "recipient_fingerprint").
				Values(runID, recipient.Ciphertext, recipient.KeyVersion, recipient.Fingerprint).
				ToSql()
			if err != nil {
				return nil, false, fmt.Errorf("build status summary recipient insert: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return nil, false, fmt.Errorf("insert status summary recipient: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit status summary run: %w", err)
	}
	return &StatusSummaryRun{ID: runID, RunKey: runKey, Snapshot: snapshot, GeneratedAt: snapshot.GeneratedAt, TimeZone: snapshot.TimeZone}, created, nil
}

func (r *PgStatusSummaryRunRepository) ClaimRuns(ctx context.Context, now time.Time, limit int) ([]StatusSummaryRun, error) {
	if limit <= 0 {
		return []StatusSummaryRun{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin status summary run claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select(
		"id", "run_key", "generated_at", "time_zone", "snapshot", "state", "attempt", "next_attempt_at", "COALESCE(last_error, '')",
	).From("notification_status_summary_runs").Where(sq.Or{
		sq.And{sq.Eq{"state": []string{"pending", "retry_wait"}}, sq.LtOrEq{"next_attempt_at": now}},
		sq.And{sq.Eq{"state": "sending"}, sq.LtOrEq{"updated_at": now.Add(-5 * time.Minute)}},
	}).OrderBy("next_attempt_at", "id").Limit(uint64(limit)).Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build status summary run claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query status summary runs: %w", err)
	}
	defer rows.Close()
	claimed := make([]StatusSummaryRun, 0)
	for rows.Next() {
		var run StatusSummaryRun
		var payload []byte
		if err := rows.Scan(&run.ID, &run.RunKey, &run.GeneratedAt, &run.TimeZone, &payload, &run.State, &run.Attempt, &run.NextAttempt, &run.LastError); err != nil {
			return nil, fmt.Errorf("scan status summary run: %w", err)
		}
		if err := json.Unmarshal(payload, &run.Snapshot); err != nil {
			return nil, fmt.Errorf("decode status summary snapshot: %w", err)
		}
		if _, err := tx.Exec(ctx, "UPDATE notification_status_summary_runs SET state='sending', attempt=attempt+1, updated_at=NOW() WHERE id=$1", run.ID); err != nil {
			return nil, fmt.Errorf("mark status summary run claimed: %w", err)
		}
		claimed = append(claimed, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status summary runs: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit status summary run claim: %w", err)
	}
	return claimed, nil
}

func (r *PgStatusSummaryRunRepository) ListRecipients(ctx context.Context, runID uuid.UUID) ([]StatusSummaryRunRecipient, error) {
	query, args, err := storage.Psql.Select("id", "address_ciphertext", "address_key_version", "recipient_fingerprint", "state", "attempt").
		From("notification_status_summary_recipients").Where(sq.Eq{"run_id": runID}).OrderBy("id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build status summary recipients: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list status summary recipients: %w", err)
	}
	defer rows.Close()
	items := make([]StatusSummaryRunRecipient, 0)
	for rows.Next() {
		var item StatusSummaryRunRecipient
		if err := rows.Scan(&item.ID, &item.Ciphertext, &item.KeyVersion, &item.Fingerprint, &item.State, &item.Attempt); err != nil {
			return nil, fmt.Errorf("scan status summary recipient: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PgStatusSummaryRunRepository) StartRecipientAttempt(ctx context.Context, recipientID uuid.UUID, startedAt time.Time) (*StatusSummaryRecipientAttempt, error) {
	query, args, err := storage.Psql.Insert("notification_status_summary_recipient_attempts").
		Columns("recipient_id", "attempt_no", "started_at", "result").
		Select(sq.Select("id", "attempt", "$2", "'started'").From("notification_status_summary_recipients").Where(sq.Eq{"id": recipientID})).
		Suffix("RETURNING id, recipient_id, attempt_no, started_at").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build status summary recipient attempt insert: %w", err)
	}
	args = append(args, startedAt)
	var attempt StatusSummaryRecipientAttempt
	if err := r.db.QueryRow(ctx, query, args...).Scan(&attempt.ID, &attempt.RecipientID, &attempt.AttemptNo, &attempt.StartedAt); err != nil {
		return nil, fmt.Errorf("start status summary recipient attempt: %w", err)
	}
	return &attempt, nil
}

func (r *PgStatusSummaryRunRepository) FinishRecipientAttempt(ctx context.Context, attemptID uuid.UUID, finishedAt time.Time, result string, category *string, summary *string) error {
	query, args, err := storage.Psql.Update("notification_status_summary_recipient_attempts").
		Set("finished_at", finishedAt).Set("result", result).Set("error_category", category).Set("status_summary", summary).
		Where(sq.Eq{"id": attemptID}).ToSql()
	if err != nil {
		return fmt.Errorf("build status summary recipient attempt finish: %w", err)
	}
	if tag, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("finish status summary recipient attempt: %w", err)
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("status summary recipient attempt %s not found", attemptID)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) RequeueStaleRecipients(ctx context.Context, runID uuid.UUID, now time.Time) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_recipients SET state='retry_wait', updated_at=NOW() WHERE run_id=$1 AND state='sending' AND updated_at <= $2", runID, now.Add(-5*time.Minute))
	if err != nil {
		return fmt.Errorf("requeue stale status summary recipients: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) ClaimRecipient(ctx context.Context, runID, recipientID uuid.UUID) (bool, error) {
	result, err := r.db.Exec(ctx, "UPDATE notification_status_summary_recipients SET state='sending', attempt=attempt+1, updated_at=NOW() WHERE id=$1 AND run_id=$2 AND state IN ('pending','retry_wait')", recipientID, runID)
	if err != nil {
		return false, fmt.Errorf("claim status summary recipient: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (r *PgStatusSummaryRunRepository) MarkRecipientSucceeded(ctx context.Context, recipientID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_recipients SET state='succeeded', sent_at=NOW(), last_error=NULL, updated_at=NOW() WHERE id=$1", recipientID)
	if err != nil {
		return fmt.Errorf("mark status summary recipient succeeded: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) MarkRecipientFailed(ctx context.Context, recipientID uuid.UUID, message string) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_recipients SET state='retry_wait', last_error=$2, updated_at=NOW() WHERE id=$1", recipientID, message)
	if err != nil {
		return fmt.Errorf("mark status summary recipient failed: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) MarkRecipientUnknown(ctx context.Context, recipientID uuid.UUID, message string) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_recipients SET state='unknown', last_error=$2, updated_at=NOW() WHERE id=$1", recipientID, message)
	if err != nil {
		return fmt.Errorf("mark status summary recipient unknown: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) MarkRunRetry(ctx context.Context, runID uuid.UUID, message string, next time.Time) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_runs SET state='retry_wait', next_attempt_at=$2, last_error=$3, updated_at=NOW() WHERE id=$1", runID, next, message)
	if err != nil {
		return fmt.Errorf("mark status summary run retry: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) MarkRunSucceeded(ctx context.Context, runID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_runs SET state='succeeded', last_error=NULL, updated_at=NOW() WHERE id=$1", runID)
	if err != nil {
		return fmt.Errorf("mark status summary run succeeded: %w", err)
	}
	return nil
}

func (r *PgStatusSummaryRunRepository) MarkRunFailed(ctx context.Context, runID uuid.UUID, message string) error {
	_, err := r.db.Exec(ctx, "UPDATE notification_status_summary_runs SET state='failed', last_error=$2, updated_at=NOW() WHERE id=$1", runID, message)
	if err != nil {
		return fmt.Errorf("mark status summary run failed: %w", err)
	}
	return nil
}

var _ StatusSummaryRunRepository = (*PgStatusSummaryRunRepository)(nil)
