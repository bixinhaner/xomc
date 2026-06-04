package export

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 是 pm_kpi_export_tasks 表的持久化接口（便于单测用 stub 实现）。
type Repository interface {
	// Create 插入一行导出任务（status=pending），返回 ID。
	Create(ctx context.Context, req CreateRequest) (uuid.UUID, error)

	// Get 按 ID 取单行（不限状态）；不存在返 ErrNotFound。
	Get(ctx context.Context, id uuid.UUID) (*Task, error)

	// List 按 filter 取分页结果（created_at 倒序）。
	List(ctx context.Context, filter ListFilter) ([]Task, error)

	// Delete 删除一行导出任务记录；不存在返 ErrNotFound。
	Delete(ctx context.Context, id uuid.UUID) error

	// MarkRunning 把任务从 pending 切到 running 并记 started_at（worker 认领时调用）。
	// 任务不在 pending 状态（已被抢 / 已终态）时返 ErrNotRunnable，避免重复处理。
	MarkRunning(ctx context.Context, id uuid.UUID) error

	// MarkSucceeded 回填 bucket/file_path/file_size/row_count，切 succeeded 并记 finished_at。
	MarkSucceeded(ctx context.Context, id uuid.UUID, bucket, filePath string, fileSize, rowCount int64) error

	// MarkFailed 把任务切 failed、落 error 并记 finished_at。
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// Errors
var (
	ErrNotFound    = errors.New("export: task not found")
	ErrNotRunnable = errors.New("export: task not in pending state")
)

// PgRepository 是 Repository 的 pgxpool 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 创建 PgRepository。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

var taskCols = []string{
	"id", "task_name", "source_type", "params", "format", "status",
	"row_count", "bucket", "file_path", "file_size", "error", "create_user",
	"created_at", "started_at", "finished_at", "expire_at",
}

func (r *PgRepository) Create(ctx context.Context, req CreateRequest) (uuid.UUID, error) {
	params := req.Params
	if len(params) == 0 {
		params = []byte("{}")
	}
	q, args, err := storage.Psql.Insert("pm_kpi_export_tasks").
		Columns("task_name", "source_type", "params", "format", "status", "create_user").
		Values(req.TaskName, string(req.SourceType), params, FormatCSV, string(StatusPending), req.CreateUser).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build insert pm_kpi_export_tasks: %w", err)
	}
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("insert pm_kpi_export_tasks: %w", err)
	}
	return id, nil
}

func (r *PgRepository) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	q, args, err := storage.Psql.Select(taskCols...).
		From("pm_kpi_export_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get pm_kpi_export_tasks: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	b := storage.Psql.Select(taskCols...).From("pm_kpi_export_tasks")
	if filter.SourceType != nil {
		b = b.Where(sq.Eq{"source_type": string(*filter.SourceType)})
	}
	if filter.Status != nil {
		b = b.Where(sq.Eq{"status": string(*filter.Status)})
	}
	if filter.OnlyReady {
		// 文件管理 Tab 视图：只列已成功 + 文件就绪。
		b = b.Where(sq.Eq{"status": string(StatusSucceeded)}).
			Where(sq.NotEq{"file_path": ""})
	}
	b = b.OrderBy("created_at DESC")
	if filter.Limit > 0 {
		b = b.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		b = b.Offset(uint64(filter.Offset))
	}
	q, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list pm_kpi_export_tasks: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query pm_kpi_export_tasks: %w", err)
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q, args, err := storage.Psql.Delete("pm_kpi_export_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete pm_kpi_export_tasks: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("delete pm_kpi_export_tasks: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE pm_kpi_export_tasks
SET status = 'running', started_at = NOW()
WHERE id = $1 AND status = 'pending'`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark export task running: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 区分"行不存在"与"行存在但非 pending"：再查一次。
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, ErrNotFound) {
			return ErrNotFound
		}
		return ErrNotRunnable
	}
	return nil
}

func (r *PgRepository) MarkSucceeded(ctx context.Context, id uuid.UUID, bucket, filePath string, fileSize, rowCount int64) error {
	const q = `UPDATE pm_kpi_export_tasks
SET status = 'succeeded', bucket = $2, file_path = $3, file_size = $4, row_count = $5, finished_at = NOW(), error = ''
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, bucket, filePath, fileSize, rowCount)
	if err != nil {
		return fmt.Errorf("mark export task succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `UPDATE pm_kpi_export_tasks
SET status = 'failed', error = $2, finished_at = NOW()
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark export task failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (*Task, error) {
	var t Task
	var sourceType, status string
	var params []byte
	if err := row.Scan(
		&t.ID, &t.TaskName, &sourceType, &params, &t.Format, &status,
		&t.RowCount, &t.Bucket, &t.FilePath, &t.FileSize, &t.Error, &t.CreateUser,
		&t.CreatedAt, &t.StartedAt, &t.FinishedAt, &t.ExpireAt,
	); err != nil {
		return nil, err
	}
	t.SourceType = SourceType(sourceType)
	t.Status = Status(status)
	if len(params) > 0 {
		t.Params = params
	} else {
		t.Params = []byte("{}")
	}
	return &t, nil
}
