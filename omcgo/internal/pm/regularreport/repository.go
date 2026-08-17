package regularreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

var ErrRunNotFound = errors.New("regularreport: run not found")

type Repository interface {
	SchedulerRepository
	GetRun(ctx context.Context, id uuid.UUID) (*Run, error)
	ListUnsentDeliveries(ctx context.Context, runID uuid.UUID) ([]Delivery, error)
	CountSentDeliveries(ctx context.Context, runID uuid.UUID) (int, error)
	MarkRunProcessing(ctx context.Context, id uuid.UUID) error
	MarkRunResult(ctx context.Context, id uuid.UUID, status RunStatus, subject, lastError string) error
	MarkDeliveryResult(ctx context.Context, id uuid.UUID, sent bool, lastError string) error
	PrepareExportRetry(ctx context.Context, exportTaskID uuid.UUID) error
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

func (r *PgRepository) ListEnabledTemplates(ctx context.Context) ([]Template, error) {
	q, args, err := storage.Psql.Select("id", "name", "creator_id", "payload").
		From("pm_query_templates").
		Where("COALESCE(payload #>> '{regular_report,enabled}', 'false') = 'true'").
		OrderBy("id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list enabled KPI report templates: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list enabled KPI report templates: %w", err)
	}
	defer rows.Close()
	result := make([]Template, 0)
	for rows.Next() {
		var template Template
		if err := rows.Scan(&template.ID, &template.Name, &template.CreatorID, &template.Payload); err != nil {
			return nil, fmt.Errorf("scan KPI report template: %w", err)
		}
		config, err := ParseConfig(template.Payload)
		if err != nil {
			return nil, fmt.Errorf("parse KPI report template %s: %w", template.ID, err)
		}
		template.Config = config
		result = append(result, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate KPI report templates: %w", err)
	}
	return result, nil
}

func (r *PgRepository) EnsureRun(ctx context.Context, request EnsureRunRequest) (bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin KPI report schedule transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	exportTaskID := uuid.New()
	asyncJobID := uuid.New()
	runID := uuid.New()
	exportName := fmt.Sprintf("KPI_%s_%s_%s", safeName(request.Template.Name), request.Period, request.WindowEnd.Format("20060102_1504"))

	q, args, err := storage.Psql.Insert("pm_kpi_export_tasks").
		Columns("id", "task_name", "source_type", "params", "format", "status", "create_user").
		Values(exportTaskID, exportName, "kpi_query", request.ExportParams, "csv", "pending", request.Template.CreatorID.String()).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build KPI report export insert: %w", err)
	}
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return false, fmt.Errorf("insert KPI report export task: %w", err)
	}

	payload, err := json.Marshal(JobPayload{RunID: runID.String()})
	if err != nil {
		return false, fmt.Errorf("build KPI report job payload: %w", err)
	}
	q, args, err = storage.Psql.Insert("async_jobs").
		Columns("id", "job_type", "status", "scheduled_at", "payload", "max_attempts").
		Values(asyncJobID, JobType, "pending", time.Now(), payload, 3).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build KPI report job insert: %w", err)
	}
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return false, fmt.Errorf("insert KPI report async job: %w", err)
	}

	q, args, err = storage.Psql.Insert("pm_kpi_report_runs").
		Columns("id", "template_id", "template_name", "creator_id", "period", "scheduled_at", "window_start", "window_end", "export_task_id", "async_job_id", "status").
		Values(runID, request.Template.ID, request.Template.Name, request.Template.CreatorID, string(request.Period), request.ScheduledAt, request.WindowStart, request.WindowEnd, exportTaskID, asyncJobID, string(RunPending)).
		Suffix("ON CONFLICT (template_id, scheduled_at, period) DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build KPI report run insert: %w", err)
	}
	var insertedID uuid.UUID
	if err := tx.QueryRow(ctx, q, args...).Scan(&insertedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("insert KPI report run: %w", err)
	}

	for _, recipient := range request.Recipients {
		q, args, err = storage.Psql.Insert("pm_kpi_report_deliveries").
			Columns("run_id", "recipient", "status").
			Values(runID, recipient, "pending").
			ToSql()
		if err != nil {
			return false, fmt.Errorf("build KPI report delivery insert: %w", err)
		}
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return false, fmt.Errorf("insert KPI report delivery: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit KPI report schedule transaction: %w", err)
	}
	return true, nil
}

func (r *PgRepository) GetRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	q, args, err := storage.Psql.Select(
		"id", "template_id", "template_name", "creator_id", "period", "scheduled_at", "window_start", "window_end",
		"export_task_id", "status", "COALESCE(subject, '')", "COALESCE(last_error, '')",
	).From("pm_kpi_report_runs").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get KPI report run: %w", err)
	}
	var run Run
	var period, status string
	if err := r.pool.QueryRow(ctx, q, args...).Scan(
		&run.ID, &run.TemplateID, &run.TemplateName, &run.CreatorID, &period, &run.ScheduledAt, &run.WindowStart, &run.WindowEnd,
		&run.ExportTaskID, &status, &run.Subject, &run.LastError,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("get KPI report run: %w", err)
	}
	run.Period = Period(period)
	run.Status = RunStatus(status)
	return &run, nil
}

func (r *PgRepository) ListUnsentDeliveries(ctx context.Context, runID uuid.UUID) ([]Delivery, error) {
	q, args, err := storage.Psql.Select("id", "run_id", "recipient", "status", "attempt").
		From("pm_kpi_report_deliveries").
		Where(sq.Eq{"run_id": runID}).
		Where(sq.NotEq{"status": "sent"}).
		OrderBy("created_at", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list KPI report deliveries: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list KPI report deliveries: %w", err)
	}
	defer rows.Close()
	result := make([]Delivery, 0)
	for rows.Next() {
		var delivery Delivery
		if err := rows.Scan(&delivery.ID, &delivery.RunID, &delivery.Recipient, &delivery.Status, &delivery.Attempt); err != nil {
			return nil, fmt.Errorf("scan KPI report delivery: %w", err)
		}
		result = append(result, delivery)
	}
	return result, rows.Err()
}

func (r *PgRepository) CountSentDeliveries(ctx context.Context, runID uuid.UUID) (int, error) {
	q, args, err := storage.Psql.Select("COUNT(*)").
		From("pm_kpi_report_deliveries").
		Where(sq.Eq{"run_id": runID, "status": "sent"}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count sent KPI report deliveries: %w", err)
	}
	var count int
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sent KPI report deliveries: %w", err)
	}
	return count, nil
}

func (r *PgRepository) MarkRunProcessing(ctx context.Context, id uuid.UUID) error {
	return r.updateRun(ctx, id, map[string]any{"status": string(RunProcessing), "last_error": ""})
}

func (r *PgRepository) MarkRunResult(ctx context.Context, id uuid.UUID, status RunStatus, subject, lastError string) error {
	return r.updateRun(ctx, id, map[string]any{"status": string(status), "subject": subject, "last_error": lastError})
}

func (r *PgRepository) updateRun(ctx context.Context, id uuid.UUID, values map[string]any) error {
	builder := storage.Psql.Update("pm_kpi_report_runs").Where(sq.Eq{"id": id}).Set("updated_at", sq.Expr("NOW()"))
	for key, value := range values {
		builder = builder.Set(key, value)
	}
	q, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update KPI report run: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update KPI report run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRunNotFound
	}
	return nil
}

func (r *PgRepository) MarkDeliveryResult(ctx context.Context, id uuid.UUID, sent bool, lastError string) error {
	status := "failed"
	var sentAt any
	if sent {
		status = "sent"
		sentAt = sq.Expr("NOW()")
	}
	builder := storage.Psql.Update("pm_kpi_report_deliveries").
		Set("status", status).
		Set("attempt", sq.Expr("attempt + 1")).
		Set("last_error", lastError).
		Set("sent_at", sentAt).
		Set("updated_at", sq.Expr("NOW()"))
	q, args, err := builder.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update KPI report delivery: %w", err)
	}
	if _, err := r.pool.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("update KPI report delivery: %w", err)
	}
	return nil
}

func (r *PgRepository) PrepareExportRetry(ctx context.Context, exportTaskID uuid.UUID) error {
	q, args, err := storage.Psql.Update("pm_kpi_export_tasks").
		Set("status", "pending").
		Set("started_at", nil).
		Set("finished_at", nil).
		Set("error", "").
		Where(sq.Eq{"id": exportTaskID, "status": []string{"running", "failed"}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reset KPI report export task: %w", err)
	}
	if _, err := r.pool.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("reset KPI report export task: %w", err)
	}
	return nil
}

func safeName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(value)
	if value == "" {
		return "report"
	}
	return value
}
