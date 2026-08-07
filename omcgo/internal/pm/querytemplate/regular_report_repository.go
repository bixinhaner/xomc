package querytemplate

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/core/storage"
)

func (r *PgRepository) GetRegularReport(ctx context.Context, templateID uuid.UUID) (*regularReportRecord, error) {
	query, args, err := storage.Psql.Select(
		"id", "report_enabled", "to_char(report_send_time, 'HH24:MI')", "report_period",
		"report_revision", "report_next_run_at",
	).From("pm_query_templates").Where(sq.Eq{"id": templateID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI regular report lookup: %w", err)
	}
	record := &regularReportRecord{}
	var period string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&record.TemplateID, &record.Enabled, &record.SendTime, &period, &record.Revision, &record.NextRunAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan KPI regular report: %w", err)
	}
	record.Period = RegularReportPeriod(period)
	recipients, err := r.listRegularReportRecipients(ctx, r.pool, templateID)
	if err != nil {
		return nil, err
	}
	record.Recipients = recipients
	return record, nil
}

func (r *PgRepository) UpdateRegularReport(
	ctx context.Context,
	templateID uuid.UUID,
	expectedRevision int64,
	input RegularReportInput,
	recipients []ProtectedReportRecipient,
) (*regularReportRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin KPI regular report update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Update("pm_query_templates").
		Set("report_enabled", input.Enabled).
		Set("report_send_time", input.SendTime).
		Set("report_period", string(input.Period)).
		Set("report_revision", expectedRevision+1).
		Set("report_next_run_at", nil).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": templateID, "report_revision": expectedRevision}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI regular report update: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update KPI regular report: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrReportRevisionMismatch
	}
	deleteQuery, deleteArgs, err := storage.Psql.Delete("pm_query_template_report_recipients").
		Where(sq.Eq{"template_id": templateID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI report recipient replacement: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return nil, fmt.Errorf("delete KPI report recipients: %w", err)
	}
	for _, recipient := range recipients {
		insertQuery, insertArgs, err := storage.Psql.Insert("pm_query_template_report_recipients").
			Columns("template_id", "address_ciphertext", "address_key_version", "recipient_fingerprint").
			Values(templateID, recipient.Ciphertext, recipient.KeyVersion, recipient.Fingerprint).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build KPI report recipient insert: %w", err)
		}
		if _, err := tx.Exec(ctx, insertQuery, insertArgs...); err != nil {
			return nil, fmt.Errorf("insert KPI report recipient: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit KPI regular report update: %w", err)
	}
	return r.GetRegularReport(ctx, templateID)
}

func (r *PgRepository) ClaimDueRegularReports(ctx context.Context, now time.Time, location *time.Location, limit int) ([]ClaimedRegularReport, error) {
	if limit <= 0 {
		return []ClaimedRegularReport{}, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin KPI regular report claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select(
		"id", "name", "payload", "report_period", "report_send_time", "report_next_run_at",
	).From("pm_query_templates").
		Where(sq.Eq{"report_enabled": true}).
		Where(sq.Or{sq.Expr("report_next_run_at IS NULL"), sq.LtOrEq{"report_next_run_at": now}}).
		OrderBy("report_next_run_at NULLS FIRST", "id").
		Limit(uint64(limit)).Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI regular report claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query due KPI regular reports: %w", err)
	}
	defer rows.Close()
	claimed := make([]ClaimedRegularReport, 0)
	for rows.Next() {
		var (
			templateID   uuid.UUID
			templateName string
			payload      []byte
			period       string
			sendTime     time.Time
			nextRun      *time.Time
		)
		if err := rows.Scan(&templateID, &templateName, &payload, &period, &sendTime, &nextRun); err != nil {
			return nil, fmt.Errorf("scan due KPI regular report: %w", err)
		}
		next, err := RegularReportNextRun(RegularReportPeriod(period), sendTime.Format("15:04"), now, location)
		if err != nil {
			return nil, fmt.Errorf("calculate next KPI regular report run: %w", err)
		}
		if _, err := tx.Exec(ctx, "UPDATE pm_query_templates SET report_next_run_at=$2, updated_at=NOW() WHERE id=$1", templateID, next); err != nil {
			return nil, fmt.Errorf("advance KPI regular report schedule: %w", err)
		}
		if nextRun == nil {
			continue
		}
		windowStart, windowEnd, err := RegularReportWindow(RegularReportPeriod(period), now, location)
		if err != nil {
			return nil, fmt.Errorf("calculate KPI regular report window: %w", err)
		}
		var runID uuid.UUID
		insert, insertArgs, err := storage.Psql.Insert("pm_query_template_report_runs").
			Columns("template_id", "window_start", "window_end", "template_payload").
			Values(templateID, windowStart.UTC(), windowEnd.UTC(), payload).
			Suffix("ON CONFLICT (template_id, window_start, window_end) DO NOTHING RETURNING id").ToSql()
		if err != nil {
			return nil, fmt.Errorf("build KPI regular report run insert: %w", err)
		}
		if err := tx.QueryRow(ctx, insert, insertArgs...).Scan(&runID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("insert KPI regular report run: %w", err)
		}
		recipients, err := r.listRegularReportRecipients(ctx, tx, templateID)
		if err != nil {
			return nil, err
		}
		for _, recipient := range recipients {
			insert, insertArgs, err := storage.Psql.Insert("pm_query_template_report_run_recipients").
				Columns("run_id", "address_ciphertext", "address_key_version", "recipient_fingerprint").
				Values(runID, recipient.Ciphertext, recipient.KeyVersion, recipient.Fingerprint).ToSql()
			if err != nil {
				return nil, fmt.Errorf("build KPI report run recipient insert: %w", err)
			}
			if _, err := tx.Exec(ctx, insert, insertArgs...); err != nil {
				return nil, fmt.Errorf("insert KPI report run recipient: %w", err)
			}
		}
		claimed = append(claimed, ClaimedRegularReport{RunID: runID, TemplateID: templateID, TemplateName: templateName, Payload: payload, Period: RegularReportPeriod(period), Recipients: recipients, WindowStart: windowStart, WindowEnd: windowEnd})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due KPI regular reports: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit KPI regular report claim: %w", err)
	}
	return claimed, nil
}

func (r *PgRepository) ClaimRegularReportRuns(ctx context.Context, now time.Time, limit int) ([]RegularReportRun, error) {
	if limit <= 0 {
		return []RegularReportRun{}, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin KPI report run claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select("id", "template_id", "window_start", "window_end", "template_payload", "export_task_id", "state").
		From("pm_query_template_report_runs").
		Where(sq.Or{
			sq.And{sq.Eq{"state": []string{"pending", "retry_wait", "failed"}}, sq.LtOrEq{"next_attempt_at": now}},
			sq.And{sq.Eq{"state": []string{"exporting", "sending"}}, sq.LtOrEq{"updated_at": now.Add(-5 * time.Minute)}},
		}).
		OrderBy("next_attempt_at", "id").Limit(uint64(limit)).Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI report run claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query KPI report runs: %w", err)
	}
	defer rows.Close()
	claimed := make([]RegularReportRun, 0)
	for rows.Next() {
		var run RegularReportRun
		var state string
		if err := rows.Scan(&run.ID, &run.TemplateID, &run.WindowStart, &run.WindowEnd, &run.Payload, &run.ExportTaskID, &state); err != nil {
			return nil, fmt.Errorf("scan KPI report run: %w", err)
		}
		run.State = RegularReportRunState(state)
		if _, err := tx.Exec(ctx, "UPDATE pm_query_template_report_runs SET state='exporting', attempt=attempt+1, updated_at=NOW() WHERE id=$1", run.ID); err != nil {
			return nil, fmt.Errorf("mark KPI report run claimed: %w", err)
		}
		claimed = append(claimed, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate KPI report runs: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit KPI report run claim: %w", err)
	}
	return claimed, nil
}

func (r *PgRepository) SetRegularReportExportTask(ctx context.Context, runID, taskID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_runs SET export_task_id=$2, updated_at=NOW() WHERE id=$1", runID, taskID)
	if err != nil {
		return fmt.Errorf("set KPI report export task: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("KPI report run %s not found", runID)
	}
	return nil
}

func (r *PgRepository) MarkRegularReportRunSending(ctx context.Context, runID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_runs SET state='sending', updated_at=NOW() WHERE id=$1", runID)
	if err != nil {
		return fmt.Errorf("mark KPI report run sending: %w", err)
	}
	return nil
}

func (r *PgRepository) ListRegularReportRunRecipients(ctx context.Context, runID uuid.UUID) ([]RegularReportRunRecipient, error) {
	query, args, err := storage.Psql.Select("id", "address_ciphertext", "address_key_version", "recipient_fingerprint", "state").
		From("pm_query_template_report_run_recipients").Where(sq.Eq{"run_id": runID}).OrderBy("id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI report run recipients: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list KPI report run recipients: %w", err)
	}
	defer rows.Close()
	result := make([]RegularReportRunRecipient, 0)
	for rows.Next() {
		var recipient RegularReportRunRecipient
		if err := rows.Scan(&recipient.ID, &recipient.Ciphertext, &recipient.KeyVersion, &recipient.Fingerprint, &recipient.State); err != nil {
			return nil, fmt.Errorf("scan KPI report run recipient: %w", err)
		}
		result = append(result, recipient)
	}
	return result, rows.Err()
}

func (r *PgRepository) ClaimRegularReportRecipient(ctx context.Context, runID, recipientID uuid.UUID) (bool, error) {
	result, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_run_recipients SET state='sending', attempt=attempt+1, updated_at=NOW() WHERE id=$1 AND run_id=$2 AND state IN ('pending','retry_wait','failed')", recipientID, runID)
	if err != nil {
		return false, fmt.Errorf("claim KPI report recipient: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (r *PgRepository) RequeueStaleRegularReportRecipients(ctx context.Context, runID uuid.UUID, now time.Time) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_run_recipients SET state='retry_wait', updated_at=NOW() WHERE run_id=$1 AND state='sending' AND updated_at <= $2", runID, now.Add(-5*time.Minute))
	if err != nil {
		return fmt.Errorf("requeue stale KPI report recipients: %w", err)
	}
	return nil
}

func (r *PgRepository) MarkRegularReportRecipientSucceeded(ctx context.Context, recipientID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_run_recipients SET state='succeeded', sent_at=NOW(), last_error=NULL, updated_at=NOW() WHERE id=$1", recipientID)
	if err != nil {
		return fmt.Errorf("mark KPI report recipient succeeded: %w", err)
	}
	return nil
}

func (r *PgRepository) MarkRegularReportRecipientFailed(ctx context.Context, recipientID uuid.UUID, message string) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_run_recipients SET state='retry_wait', last_error=$2, updated_at=NOW() WHERE id=$1", recipientID, message)
	if err != nil {
		return fmt.Errorf("mark KPI report recipient failed: %w", err)
	}
	return nil
}

func (r *PgRepository) MarkRegularReportRunRetry(ctx context.Context, runID uuid.UUID, message string, next time.Time) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_runs SET state='retry_wait', next_attempt_at=$2, last_error=$3, updated_at=NOW() WHERE id=$1", runID, next, message)
	if err != nil {
		return fmt.Errorf("mark KPI report run retry: %w", err)
	}
	return nil
}

func (r *PgRepository) MarkRegularReportRunSucceeded(ctx context.Context, runID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_runs SET state='succeeded', last_error=NULL, updated_at=NOW() WHERE id=$1", runID)
	if err != nil {
		return fmt.Errorf("mark KPI report run succeeded: %w", err)
	}
	return nil
}

func (r *PgRepository) ClearRegularReportExportTask(ctx context.Context, runID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "UPDATE pm_query_template_report_runs SET export_task_id=NULL, updated_at=NOW() WHERE id=$1", runID)
	if err != nil {
		return fmt.Errorf("clear KPI report export task: %w", err)
	}
	return nil
}

type regularReportQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *PgRepository) listRegularReportRecipients(ctx context.Context, db regularReportQueryer, templateID uuid.UUID) ([]ProtectedReportRecipient, error) {
	query, args, err := storage.Psql.Select("address_ciphertext", "address_key_version", "recipient_fingerprint").
		From("pm_query_template_report_recipients").Where(sq.Eq{"template_id": templateID}).OrderBy("created_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build KPI report recipient lookup: %w", err)
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list KPI report recipients: %w", err)
	}
	defer rows.Close()
	result := make([]ProtectedReportRecipient, 0)
	for rows.Next() {
		var recipient ProtectedReportRecipient
		if err := rows.Scan(&recipient.Ciphertext, &recipient.KeyVersion, &recipient.Fingerprint); err != nil {
			return nil, fmt.Errorf("scan KPI report recipient: %w", err)
		}
		result = append(result, recipient)
	}
	return result, rows.Err()
}
