package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ============================================================
// F06-ops-management 扩展仓储 — 6 张新表 + 既有表新列
// MVP 实现：List + Get + Create + Update（按需），不写 Delete
// ============================================================

// ---- TaskExecution Repository ----

type TaskExecutionRepository interface {
	Create(ctx context.Context, e *OpsTaskExecution) error
	Update(ctx context.Context, e *OpsTaskExecution) error
	List(ctx context.Context, filter TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error)
	// CountByTaskStatus 聚合统计指定 task 的 executions，返
	// `{status: count}` map（T-0101-e 结果聚合）。skipped 是有意区分于 failed
	// 的状态——dispatcher pause/cancel 后未发出的设备步是 skipped 不是 failed。
	CountByTaskStatus(ctx context.Context, taskID uuid.UUID) (map[string]int, error)
}

type PgTaskExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewPgTaskExecutionRepository(pool *pgxpool.Pool) *PgTaskExecutionRepository {
	return &PgTaskExecutionRepository{pool: pool}
}

func (r *PgTaskExecutionRepository) Create(ctx context.Context, e *OpsTaskExecution) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	query, args, err := storage.Psql.Insert("ops_task_executions").
		Columns("id", "task_id", "device_sn", "step_index", "step_name", "step_type",
			"status", "started_at", "completed_at", "duration_ms", "request", "response",
			"error_message", "created_at").
		Values(e.ID, e.TaskID, e.DeviceSN, e.StepIndex, e.StepName, e.StepType,
			e.Status, e.StartedAt, e.CompletedAt, e.DurationMS, nullableRaw(e.Request),
			nullableRaw(e.Response), nullableString(e.ErrorMessage), e.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert task_execution: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert task_execution: %w", err)
	}
	return nil
}

func (r *PgTaskExecutionRepository) Update(ctx context.Context, e *OpsTaskExecution) error {
	query, args, err := storage.Psql.Update("ops_task_executions").
		Set("status", e.Status).
		Set("started_at", e.StartedAt).
		Set("completed_at", e.CompletedAt).
		Set("duration_ms", e.DurationMS).
		Set("response", nullableRaw(e.Response)).
		Set("error_message", nullableString(e.ErrorMessage)).
		Where(sq.Eq{"id": e.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update task_execution: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update task_execution: %w", err)
	}
	return nil
}

func (r *PgTaskExecutionRepository) List(ctx context.Context, filter TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error) {
	q := storage.Psql.Select("id", "task_id", "device_sn", "step_index", "step_name", "step_type",
		"status", "started_at", "completed_at", "duration_ms", "request", "response",
		"error_message", "created_at").From("ops_task_executions")

	if filter.TaskID != nil {
		q = q.Where(sq.Eq{"task_id": *filter.TaskID})
	}
	if filter.DeviceSN != "" {
		q = q.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.Status != "" {
		q = q.Where(sq.Eq{"status": filter.Status})
	}

	return paginateOps[OpsTaskExecution](ctx, r.pool, q, "ops_task_executions", filter.ListRequest, scanTaskExecution)
}

// CountByTaskStatus 聚合统计指定 task 的 executions，返 `{status: count}` map
// （T-0101-e 结果聚合）。单 SQL `SELECT status, COUNT(*) FROM ops_task_executions
// WHERE task_id=$1 GROUP BY status` 避免 N 次 round-trip。
func (r *PgTaskExecutionRepository) CountByTaskStatus(ctx context.Context, taskID uuid.UUID) (map[string]int, error) {
	const rawSQL = `
		SELECT status, COUNT(*) AS cnt
		FROM ops_task_executions
		WHERE task_id = $1
		GROUP BY status`

	rows, err := r.pool.Query(ctx, rawSQL, taskID)
	if err != nil {
		return nil, fmt.Errorf("count task executions by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			return nil, fmt.Errorf("scan execution status count: %w", err)
		}
		counts[status] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution status counts: %w", err)
	}
	return counts, nil
}

// ---- Diagnostic Repository ----

type DiagnosticRepository interface {
	Create(ctx context.Context, d *OpsDiagnostic) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsDiagnostic, error)
	Update(ctx context.Context, d *OpsDiagnostic) error
	List(ctx context.Context, filter DiagnosticFilter) (*model.ListResponse[OpsDiagnostic], error)
}

type PgDiagnosticRepository struct {
	pool *pgxpool.Pool
}

func NewPgDiagnosticRepository(pool *pgxpool.Pool) *PgDiagnosticRepository {
	return &PgDiagnosticRepository{pool: pool}
}

func (r *PgDiagnosticRepository) Create(ctx context.Context, d *OpsDiagnostic) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	if d.StartedAt.IsZero() {
		d.StartedAt = time.Now()
	}
	if d.Status == "" {
		d.Status = DiagPending
	}
	if d.Initiator == "" {
		d.Initiator = "omc"
	}
	if d.Request == nil {
		d.Request = []byte("{}")
	}

	query, args, err := storage.Psql.Insert("ops_diagnostics").
		Columns("id", "device_sn", "diag_type", "initiator", "request", "result",
			"status", "started_at", "completed_at", "duration_ms", "operator",
			"task_id", "file_path", "error_message", "created_at").
		Values(d.ID, d.DeviceSN, d.DiagType, d.Initiator, d.Request, nullableRaw(d.Result),
			d.Status, d.StartedAt, d.CompletedAt, d.DurationMS, nullableString(d.Operator),
			d.TaskID, nullableString(d.FilePath), nullableString(d.ErrorMessage), d.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert diagnostic: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert diagnostic: %w", err)
	}
	return nil
}

func (r *PgDiagnosticRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsDiagnostic, error) {
	q := diagnosticSelect().Where(sq.Eq{"id": id})
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get diagnostic: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	d, err := scanDiagnostic(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan diagnostic: %w", err)
	}
	return &d, nil
}

func (r *PgDiagnosticRepository) Update(ctx context.Context, d *OpsDiagnostic) error {
	query, args, err := storage.Psql.Update("ops_diagnostics").
		Set("status", d.Status).
		Set("result", nullableRaw(d.Result)).
		Set("completed_at", d.CompletedAt).
		Set("duration_ms", d.DurationMS).
		Set("file_path", nullableString(d.FilePath)).
		Set("error_message", nullableString(d.ErrorMessage)).
		Where(sq.Eq{"id": d.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update diagnostic: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update diagnostic: %w", err)
	}
	return nil
}

func (r *PgDiagnosticRepository) List(ctx context.Context, filter DiagnosticFilter) (*model.ListResponse[OpsDiagnostic], error) {
	q := diagnosticSelect()
	if filter.DeviceSN != "" {
		q = q.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.DiagType != "" {
		q = q.Where(sq.Eq{"diag_type": filter.DiagType})
	}
	if filter.Status != nil {
		q = q.Where(sq.Eq{"status": string(*filter.Status)})
	}
	return paginateOps[OpsDiagnostic](ctx, r.pool, q, "ops_diagnostics", filter.ListRequest, scanDiagnostic)
}

func diagnosticSelect() sq.SelectBuilder {
	return storage.Psql.Select("id", "device_sn", "diag_type", "initiator", "request", "result",
		"status", "started_at", "completed_at", "duration_ms", "operator", "task_id",
		"file_path", "error_message", "created_at").From("ops_diagnostics")
}

// ---- Download Repository ----

type DownloadRepository interface {
	Create(ctx context.Context, d *OpsDownload) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsDownload, error)
	Update(ctx context.Context, d *OpsDownload) error
	List(ctx context.Context, filter DownloadFilter) (*model.ListResponse[OpsDownload], error)
}

type PgDownloadRepository struct {
	pool *pgxpool.Pool
}

func NewPgDownloadRepository(pool *pgxpool.Pool) *PgDownloadRepository {
	return &PgDownloadRepository{pool: pool}
}

func (r *PgDownloadRepository) Create(ctx context.Context, d *OpsDownload) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	if d.Status == "" {
		d.Status = DownloadPending
	}
	query, args, err := storage.Psql.Insert("ops_downloads").
		Columns("id", "device_sn", "content_type", "file_path", "file_size", "checksum",
			"status", "operator", "task_id", "expires_at", "created_at").
		Values(d.ID, d.DeviceSN, d.ContentType, d.FilePath, d.FileSize, nullableString(d.Checksum),
			d.Status, nullableString(d.Operator), d.TaskID, d.ExpiresAt, d.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert download: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert download: %w", err)
	}
	return nil
}

func (r *PgDownloadRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsDownload, error) {
	q := downloadSelect().Where(sq.Eq{"id": id})
	query, args, _ := q.ToSql()
	row := r.pool.QueryRow(ctx, query, args...)
	d, err := scanDownload(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan download: %w", err)
	}
	return &d, nil
}

func (r *PgDownloadRepository) Update(ctx context.Context, d *OpsDownload) error {
	query, args, err := storage.Psql.Update("ops_downloads").
		Set("status", d.Status).
		Set("file_path", d.FilePath).
		Set("file_size", d.FileSize).
		Set("checksum", nullableString(d.Checksum)).
		Set("expires_at", d.ExpiresAt).
		Where(sq.Eq{"id": d.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update download: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update download: %w", err)
	}
	return nil
}

func (r *PgDownloadRepository) List(ctx context.Context, filter DownloadFilter) (*model.ListResponse[OpsDownload], error) {
	q := downloadSelect()
	if filter.DeviceSN != "" {
		q = q.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.ContentType != "" {
		q = q.Where(sq.Eq{"content_type": filter.ContentType})
	}
	if filter.Status != nil {
		q = q.Where(sq.Eq{"status": string(*filter.Status)})
	}
	return paginateOps[OpsDownload](ctx, r.pool, q, "ops_downloads", filter.ListRequest, scanDownload)
}

func downloadSelect() sq.SelectBuilder {
	return storage.Psql.Select("id", "device_sn", "content_type", "file_path", "file_size",
		"checksum", "status", "operator", "task_id", "expires_at", "created_at").From("ops_downloads")
}

// ---- AuditLog Repository ----

type AuditLogRepository interface {
	Create(ctx context.Context, l *OpsAuditLog) error
	List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[OpsAuditLog], error)
}

type PgAuditLogRepository struct {
	pool *pgxpool.Pool
}

func NewPgAuditLogRepository(pool *pgxpool.Pool) *PgAuditLogRepository {
	return &PgAuditLogRepository{pool: pool}
}

func (r *PgAuditLogRepository) Create(ctx context.Context, l *OpsAuditLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	if l.RiskLevel == "" {
		l.RiskLevel = RiskSafe
	}
	if l.Result == "" {
		l.Result = "success"
	}
	query, args, err := storage.Psql.Insert("ops_audit_logs").
		Columns("id", "op_type", "target_type", "target_id", "operator_user_id", "operator_name",
			"risk_level", "input", "output_summary", "result", "approver_user_id", "break_glass",
			"client_ip", "user_agent", "created_at").
		Values(l.ID, l.OpType, l.TargetType, l.TargetID, l.OperatorUserID, l.OperatorName,
			string(l.RiskLevel), nullableRaw(l.Input), nullableString(l.OutputSummary), l.Result,
			l.ApproverUserID, l.BreakGlass, nullableString(l.ClientIP), nullableString(l.UserAgent),
			l.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert audit_log: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert audit_log: %w", err)
	}
	return nil
}

func (r *PgAuditLogRepository) List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[OpsAuditLog], error) {
	q := storage.Psql.Select("id", "op_type", "target_type", "target_id", "operator_user_id",
		"operator_name", "risk_level", "input", "output_summary", "result", "approver_user_id",
		"break_glass", "client_ip", "user_agent", "created_at").From("ops_audit_logs")

	if filter.OpType != "" {
		q = q.Where(sq.Eq{"op_type": filter.OpType})
	}
	if filter.TargetType != "" {
		q = q.Where(sq.Eq{"target_type": filter.TargetType})
	}
	if filter.TargetID != "" {
		q = q.Where(sq.Eq{"target_id": filter.TargetID})
	}
	if filter.OperatorUserID != nil {
		q = q.Where(sq.Eq{"operator_user_id": *filter.OperatorUserID})
	}
	if filter.RiskLevel != nil {
		q = q.Where(sq.Eq{"risk_level": string(*filter.RiskLevel)})
	}
	if filter.BreakGlass != nil {
		q = q.Where(sq.Eq{"break_glass": *filter.BreakGlass})
	}
	if filter.FromTime != nil {
		q = q.Where(sq.GtOrEq{"created_at": *filter.FromTime})
	}
	if filter.ToTime != nil {
		q = q.Where(sq.LtOrEq{"created_at": *filter.ToTime})
	}

	return paginateOps[OpsAuditLog](ctx, r.pool, q, "ops_audit_logs", filter.ListRequest, scanAuditLog)
}

// ---- MaintenanceWindow Repository ----

type MaintenanceWindowRepository interface {
	Create(ctx context.Context, w *OpsMaintenanceWindow) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsMaintenanceWindow, error)
	Update(ctx context.Context, w *OpsMaintenanceWindow) error
	List(ctx context.Context, filter MaintenanceWindowFilter) (*model.ListResponse[OpsMaintenanceWindow], error)
	ListActive(ctx context.Context, now time.Time) ([]OpsMaintenanceWindow, error)
}

type PgMaintenanceWindowRepository struct {
	pool *pgxpool.Pool
}

func NewPgMaintenanceWindowRepository(pool *pgxpool.Pool) *PgMaintenanceWindowRepository {
	return &PgMaintenanceWindowRepository{pool: pool}
}

func (r *PgMaintenanceWindowRepository) Create(ctx context.Context, w *OpsMaintenanceWindow) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	now := time.Now()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	w.UpdatedAt = now
	if w.Status == "" {
		w.Status = MWPlanned
	}
	if w.ScopeIDs == nil {
		w.ScopeIDs = []byte("[]")
	}
	query, args, err := storage.Psql.Insert("ops_maintenance_windows").
		Columns("id", "name", "scope_type", "scope_ids", "start_at", "end_at",
			"suppress_alarms", "pause_provision", "allow_dangerous", "reason",
			"creator_user_id", "approver_user_id", "status", "created_at", "updated_at").
		Values(w.ID, w.Name, w.ScopeType, w.ScopeIDs, w.StartAt, w.EndAt,
			w.SuppressAlarms, w.PauseProvision, w.AllowDangerous, nullableString(w.Reason),
			w.CreatorUserID, w.ApproverUserID, string(w.Status), w.CreatedAt, w.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert maintenance_window: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert maintenance_window: %w", err)
	}
	return nil
}

func (r *PgMaintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsMaintenanceWindow, error) {
	q := maintenanceSelect().Where(sq.Eq{"id": id})
	query, args, _ := q.ToSql()
	row := r.pool.QueryRow(ctx, query, args...)
	w, err := scanMaintenance(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan maintenance_window: %w", err)
	}
	return &w, nil
}

func (r *PgMaintenanceWindowRepository) Update(ctx context.Context, w *OpsMaintenanceWindow) error {
	query, args, err := storage.Psql.Update("ops_maintenance_windows").
		Set("name", w.Name).
		Set("scope_type", w.ScopeType).
		Set("scope_ids", w.ScopeIDs).
		Set("start_at", w.StartAt).
		Set("end_at", w.EndAt).
		Set("suppress_alarms", w.SuppressAlarms).
		Set("pause_provision", w.PauseProvision).
		Set("allow_dangerous", w.AllowDangerous).
		Set("reason", nullableString(w.Reason)).
		Set("status", string(w.Status)).
		Set("approver_user_id", w.ApproverUserID).
		Where(sq.Eq{"id": w.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update maintenance_window: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update maintenance_window: %w", err)
	}
	return nil
}

func (r *PgMaintenanceWindowRepository) List(ctx context.Context, filter MaintenanceWindowFilter) (*model.ListResponse[OpsMaintenanceWindow], error) {
	q := maintenanceSelect()
	if filter.Status != nil {
		q = q.Where(sq.Eq{"status": string(*filter.Status)})
	}
	return paginateOps[OpsMaintenanceWindow](ctx, r.pool, q, "ops_maintenance_windows", filter.ListRequest, scanMaintenance)
}

func (r *PgMaintenanceWindowRepository) ListActive(ctx context.Context, now time.Time) ([]OpsMaintenanceWindow, error) {
	query, args, err := activeMaintenanceWindowQuery(now).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active maintenance_windows query: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active maintenance_windows: %w", err)
	}
	defer rows.Close()
	var out []OpsMaintenanceWindow
	for rows.Next() {
		w, err := scanMaintenance(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active maintenance_window: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func activeMaintenanceWindowQuery(now time.Time) sq.SelectBuilder {
	return maintenanceSelect().
		Where(sq.Eq{"status": []string{string(MWApproved), string(MWActive)}}).
		Where(sq.LtOrEq{"start_at": now}).
		Where(sq.GtOrEq{"end_at": now})
}

func maintenanceSelect() sq.SelectBuilder {
	return storage.Psql.Select("id", "name", "scope_type", "scope_ids", "start_at", "end_at",
		"suppress_alarms", "pause_provision", "allow_dangerous", "reason",
		"creator_user_id", "approver_user_id", "status", "created_at", "updated_at").
		From("ops_maintenance_windows")
}

// ---- Playbook Repository ----

type PlaybookRepository interface {
	Create(ctx context.Context, p *OpsPlaybook) error
	GetByID(ctx context.Context, id uuid.UUID) (*OpsPlaybook, error)
	List(ctx context.Context, filter PlaybookFilter) (*model.ListResponse[OpsPlaybook], error)
	MatchByAlarm(ctx context.Context, alarmCode string) ([]OpsPlaybook, error)
}

type PgPlaybookRepository struct {
	pool *pgxpool.Pool
}

func NewPgPlaybookRepository(pool *pgxpool.Pool) *PgPlaybookRepository {
	return &PgPlaybookRepository{pool: pool}
}

func (r *PgPlaybookRepository) Create(ctx context.Context, p *OpsPlaybook) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.AlarmPattern == nil {
		p.AlarmPattern = []byte("{}")
	}
	if p.RecommendedTemplates == nil {
		p.RecommendedTemplates = []byte("[]")
	}
	if p.Tags == nil {
		p.Tags = []byte("[]")
	}
	query, args, err := storage.Psql.Insert("ops_playbooks").
		Columns("id", "alarm_pattern", "recommended_templates", "title", "docs", "tags",
			"success_rate", "use_count", "created_at", "updated_at").
		Values(p.ID, p.AlarmPattern, p.RecommendedTemplates, p.Title, nullableString(p.Docs),
			p.Tags, p.SuccessRate, p.UseCount, p.CreatedAt, p.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert playbook: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert playbook: %w", err)
	}
	return nil
}

func (r *PgPlaybookRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsPlaybook, error) {
	q := playbookSelect().Where(sq.Eq{"id": id})
	query, args, _ := q.ToSql()
	row := r.pool.QueryRow(ctx, query, args...)
	p, err := scanPlaybook(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan playbook: %w", err)
	}
	return &p, nil
}

func (r *PgPlaybookRepository) List(ctx context.Context, filter PlaybookFilter) (*model.ListResponse[OpsPlaybook], error) {
	q := playbookSelect()
	if filter.Keyword != "" {
		q = q.Where(sq.ILike{"title": "%" + filter.Keyword + "%"})
	}
	return paginateOps[OpsPlaybook](ctx, r.pool, q, "ops_playbooks", filter.ListRequest, scanPlaybook)
}

func (r *PgPlaybookRepository) MatchByAlarm(ctx context.Context, alarmCode string) ([]OpsPlaybook, error) {
	// MVP: 简单 JSONB 查询 — alarm_pattern.alarm_code 数组含 alarmCode
	q := playbookSelect().
		Where("alarm_pattern->'alarm_code' @> ?::jsonb", fmt.Sprintf(`["%s"]`, alarmCode)).
		OrderBy("use_count DESC")
	query, args, _ := q.ToSql()
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query playbooks by alarm: %w", err)
	}
	defer rows.Close()
	var out []OpsPlaybook
	for rows.Next() {
		p, err := scanPlaybook(rows)
		if err != nil {
			return nil, fmt.Errorf("scan playbook match: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func playbookSelect() sq.SelectBuilder {
	return storage.Psql.Select("id", "alarm_pattern", "recommended_templates", "title",
		"docs", "tags", "success_rate", "use_count", "created_at", "updated_at").
		From("ops_playbooks")
}

// ============================================================
// Scan helpers
// ============================================================

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTaskExecution(row rowScanner) (OpsTaskExecution, error) {
	var e OpsTaskExecution
	var request, response []byte
	var errMsg *string
	if err := row.Scan(&e.ID, &e.TaskID, &e.DeviceSN, &e.StepIndex, &e.StepName, &e.StepType,
		&e.Status, &e.StartedAt, &e.CompletedAt, &e.DurationMS, &request, &response,
		&errMsg, &e.CreatedAt); err != nil {
		return e, err
	}
	if len(request) > 0 {
		e.Request = json.RawMessage(request)
	}
	if len(response) > 0 {
		e.Response = json.RawMessage(response)
	}
	if errMsg != nil {
		e.ErrorMessage = *errMsg
	}
	return e, nil
}

func scanDiagnostic(row rowScanner) (OpsDiagnostic, error) {
	var d OpsDiagnostic
	var result []byte
	var operator, filePath, errMsg *string
	if err := row.Scan(&d.ID, &d.DeviceSN, &d.DiagType, &d.Initiator, &d.Request, &result,
		&d.Status, &d.StartedAt, &d.CompletedAt, &d.DurationMS, &operator, &d.TaskID,
		&filePath, &errMsg, &d.CreatedAt); err != nil {
		return d, err
	}
	if len(result) > 0 {
		d.Result = json.RawMessage(result)
	}
	if operator != nil {
		d.Operator = *operator
	}
	if filePath != nil {
		d.FilePath = *filePath
	}
	if errMsg != nil {
		d.ErrorMessage = *errMsg
	}
	return d, nil
}

func scanDownload(row rowScanner) (OpsDownload, error) {
	var d OpsDownload
	var checksum, operator *string
	if err := row.Scan(&d.ID, &d.DeviceSN, &d.ContentType, &d.FilePath, &d.FileSize,
		&checksum, &d.Status, &operator, &d.TaskID, &d.ExpiresAt, &d.CreatedAt); err != nil {
		return d, err
	}
	if checksum != nil {
		d.Checksum = *checksum
	}
	if operator != nil {
		d.Operator = *operator
	}
	return d, nil
}

func scanAuditLog(row rowScanner) (OpsAuditLog, error) {
	var l OpsAuditLog
	var input []byte
	var outputSummary, clientIP, userAgent *string
	var riskLevel string
	if err := row.Scan(&l.ID, &l.OpType, &l.TargetType, &l.TargetID, &l.OperatorUserID,
		&l.OperatorName, &riskLevel, &input, &outputSummary, &l.Result,
		&l.ApproverUserID, &l.BreakGlass, &clientIP, &userAgent, &l.CreatedAt); err != nil {
		return l, err
	}
	l.RiskLevel = RiskLevel(riskLevel)
	if len(input) > 0 {
		l.Input = json.RawMessage(input)
	}
	if outputSummary != nil {
		l.OutputSummary = *outputSummary
	}
	if clientIP != nil {
		l.ClientIP = *clientIP
	}
	if userAgent != nil {
		l.UserAgent = *userAgent
	}
	return l, nil
}

func scanMaintenance(row rowScanner) (OpsMaintenanceWindow, error) {
	var w OpsMaintenanceWindow
	var reason *string
	var status string
	if err := row.Scan(&w.ID, &w.Name, &w.ScopeType, &w.ScopeIDs, &w.StartAt, &w.EndAt,
		&w.SuppressAlarms, &w.PauseProvision, &w.AllowDangerous, &reason,
		&w.CreatorUserID, &w.ApproverUserID, &status, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return w, err
	}
	w.Status = MaintenanceWindowStatus(status)
	if reason != nil {
		w.Reason = *reason
	}
	return w, nil
}

func scanPlaybook(row rowScanner) (OpsPlaybook, error) {
	var p OpsPlaybook
	var docs *string
	if err := row.Scan(&p.ID, &p.AlarmPattern, &p.RecommendedTemplates, &p.Title,
		&docs, &p.Tags, &p.SuccessRate, &p.UseCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return p, err
	}
	if docs != nil {
		p.Docs = *docs
	}
	return p, nil
}

// ---- Generic pagination helper ----

func paginateOps[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	base sq.SelectBuilder,
	tableName string,
	req model.ListRequest,
	scanFn func(rowScanner) (T, error),
) (*model.ListResponse[T], error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// Count
	countQ, countArgs, err := base.RemoveColumns().Columns("COUNT(*)").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count %s: %w", tableName, err)
	}
	var total int64
	if err := pool.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("query count %s: %w", tableName, err)
	}

	// Page query
	pageQ := base.OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize))
	query, args, err := pageQ.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s: %w", tableName, err)
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", tableName, err)
	}
	defer rows.Close()
	items := make([]T, 0, pageSize)
	for rows.Next() {
		item, err := scanFn(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", tableName, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", tableName, err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	return &model.ListResponse[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// ---- Nullable helpers ----

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableRaw(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}
