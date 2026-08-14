package reportsubscription

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

type Repository interface {
	GetByTemplate(ctx context.Context, templateID uuid.UUID) (*Subscription, error)
	Upsert(ctx context.Context, templateID uuid.UUID, input UpsertInput) (*Subscription, error)
	Delete(ctx context.Context, templateID uuid.UUID) error
	ListRuns(ctx context.Context, templateID uuid.UUID, limit int) ([]Run, error)
	GetRun(ctx context.Context, runID uuid.UUID) (*Run, error)
	MarkRunRunning(ctx context.Context, runID uuid.UUID) error
	MarkRunExportTask(ctx context.Context, runID uuid.UUID, exportTaskID uuid.UUID) error
	MarkRunSent(ctx context.Context, runID uuid.UUID, historyID uuid.UUID, attachmentName string) error
	MarkRunFailed(ctx context.Context, runID uuid.UUID, status string, message string) error
}

func (r *PgRepository) GetRun(ctx context.Context, runID uuid.UUID) (*Run, error) {
	query, args, err := storage.Psql.Select(
		"id", "subscription_id", "query_template_id", "query_template_name", "query_payload",
		"period", "recipients", "window_start", "window_end", "job_id", "export_task_id", "notification_history_id",
		"status", "attachment_name", "error_message", "started_at", "finished_at", "created_at",
	).From("pm_query_report_runs").Where(sq.Eq{"id": runID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get report run SQL: %w", err)
	}
	var item Run
	var period string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.SubscriptionID, &item.QueryTemplateID, &item.QueryTemplateName, &item.QueryPayload,
		&period, &item.Recipients, &item.WindowStart, &item.WindowEnd, &item.JobID, &item.ExportTaskID, &item.NotificationHistory,
		&item.Status, &item.AttachmentName, &item.ErrorMessage, &item.StartedAt, &item.FinishedAt, &item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get report run: %w", err)
	}
	item.Period = Period(period)
	return &item, nil
}

func (r *PgRepository) MarkRunRunning(ctx context.Context, runID uuid.UUID) error {
	return r.updateRunAndSubscription(ctx, runID, map[string]any{
		"status": RunStatusRunning, "started_at": sq.Expr("NOW()"), "finished_at": nil, "error_message": nil,
	}, map[string]any{"last_status": RunStatusRunning, "last_error": nil})
}

func (r *PgRepository) MarkRunExportTask(ctx context.Context, runID uuid.UUID, exportTaskID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_query_report_runs").
		Set("export_task_id", exportTaskID).
		Where(sq.Eq{"id": runID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build link report export task SQL: %w", err)
	}
	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("link report export task: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) MarkRunSent(ctx context.Context, runID uuid.UUID, historyID uuid.UUID, attachmentName string) error {
	return r.updateRunAndSubscription(ctx, runID, map[string]any{
		"status": RunStatusSent, "notification_history_id": historyID, "attachment_name": attachmentName,
		"finished_at": sq.Expr("NOW()"), "error_message": nil,
	}, map[string]any{"last_status": RunStatusSent, "last_error": nil, "last_run_at": sq.Expr("NOW()")})
}

func (r *PgRepository) MarkRunFailed(ctx context.Context, runID uuid.UUID, status string, message string) error {
	if status != RunStatusExportFailed && status != RunStatusDeliveryFailed {
		return fmt.Errorf("invalid report run failure status %q", status)
	}
	return r.updateRunAndSubscription(ctx, runID, map[string]any{
		"status": status, "error_message": message, "finished_at": sq.Expr("NOW()"),
	}, map[string]any{"last_status": status, "last_error": message, "last_run_at": sq.Expr("NOW()")})
}

func (r *PgRepository) updateRunAndSubscription(ctx context.Context, runID uuid.UUID, values, subscriptionValues map[string]any) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin report run status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	builder := storage.Psql.Update("pm_query_report_runs").Where(sq.Eq{"id": runID})
	for column, value := range values {
		builder = builder.Set(column, value)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update report run SQL: %w", err)
	}
	cmd, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update report run: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	subscriptionBuilder := storage.Psql.Update("pm_query_report_subscriptions")
	for column, value := range subscriptionValues {
		subscriptionBuilder = subscriptionBuilder.Set(column, value)
	}
	subscriptionQuery, subscriptionArgs, err := subscriptionBuilder.
		Where("id = (SELECT subscription_id FROM pm_query_report_runs WHERE id = ?)", runID).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update report subscription status SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, subscriptionQuery, subscriptionArgs...); err != nil {
		return fmt.Errorf("update report subscription status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit report run status transaction: %w", err)
	}
	return nil
}

type PgRepository struct{ pool *pgxpool.Pool }

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

var _ Repository = (*PgRepository)(nil)

var subscriptionColumns = []string{
	"id", "query_template_id", "enabled", "period", "send_times::text[]",
	"recipients", "timezone_name", "next_run_at", "last_run_at", "last_status", "last_error",
	"created_by", "created_at", "updated_at",
}

func (r *PgRepository) GetByTemplate(ctx context.Context, templateID uuid.UUID) (*Subscription, error) {
	query, args, err := storage.Psql.Select(subscriptionColumns...).
		From("pm_query_report_subscriptions").
		Where(sq.Eq{"query_template_id": templateID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get report subscription SQL: %w", err)
	}
	var item Subscription
	var period string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.QueryTemplateID, &item.Enabled, &period, &item.SendTimes,
		&item.Recipients, &item.TimezoneName, &item.NextRunAt, &item.LastRunAt, &item.LastStatus, &item.LastError,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get report subscription: %w", err)
	}
	item.Period = Period(period)
	for i := range item.SendTimes {
		item.SendTimes[i] = strings.TrimSuffix(item.SendTimes[i], ":00")
	}
	return &item, nil
}

func (r *PgRepository) Upsert(ctx context.Context, templateID uuid.UUID, input UpsertInput) (*Subscription, error) {
	query, args, err := storage.Psql.Insert("pm_query_report_subscriptions").
		Columns("query_template_id", "enabled", "period", "send_times", "recipients", "timezone_name", "next_run_at", "created_by").
		Values(
			templateID, input.Enabled, string(input.Period),
			sq.Expr("ARRAY(SELECT value::time FROM unnest(?::text[]) AS value)", input.SendTimes),
			input.Recipients, input.TimezoneName, input.NextRunAt, input.CreatedBy,
		).
		Suffix(`ON CONFLICT (query_template_id) DO UPDATE SET
enabled = EXCLUDED.enabled,
period = EXCLUDED.period,
send_times = EXCLUDED.send_times,
recipients = EXCLUDED.recipients,
timezone_name = EXCLUDED.timezone_name,
next_run_at = EXCLUDED.next_run_at,
last_error = NULL,
updated_at = NOW()`).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert report subscription SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("upsert report subscription: %w", err)
	}
	return r.GetByTemplate(ctx, templateID)
}

func (r *PgRepository) Delete(ctx context.Context, templateID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("pm_query_report_subscriptions").
		Where(sq.Eq{"query_template_id": templateID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete report subscription SQL: %w", err)
	}
	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete report subscription: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) ListRuns(ctx context.Context, templateID uuid.UUID, limit int) ([]Run, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	query, args, err := storage.Psql.Select(
		"id", "subscription_id", "query_template_id", "query_template_name", "query_payload",
		"period", "recipients", "window_start", "window_end", "job_id", "export_task_id", "notification_history_id",
		"status", "attachment_name", "error_message", "started_at", "finished_at", "created_at",
	).
		From("pm_query_report_runs").
		Where(sq.Eq{"query_template_id": templateID}).
		OrderBy("created_at DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list report runs SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list report runs: %w", err)
	}
	defer rows.Close()
	items := make([]Run, 0)
	for rows.Next() {
		var item Run
		var period string
		if err := rows.Scan(
			&item.ID, &item.SubscriptionID, &item.QueryTemplateID, &item.QueryTemplateName, &item.QueryPayload,
			&period, &item.Recipients, &item.WindowStart, &item.WindowEnd, &item.JobID, &item.ExportTaskID, &item.NotificationHistory,
			&item.Status, &item.AttachmentName, &item.ErrorMessage, &item.StartedAt, &item.FinishedAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report run: %w", err)
		}
		item.Period = Period(period)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report runs: %w", err)
	}
	return items, nil
}
