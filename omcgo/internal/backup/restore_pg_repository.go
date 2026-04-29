package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// RestoreTaskRepository — narrow contract used by RestoreService.
type RestoreTaskRepository interface {
	Create(ctx context.Context, task *RestoreTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*RestoreTask, error)
	List(ctx context.Context, filter RestoreFilter) (*model.ListResponse[RestoreTask], error)
	UpdateErrorMessage(ctx context.Context, id uuid.UUID, msg string) error
}

// RestoreFilter mirrors TaskFilter but for the restore_tasks table.
type RestoreFilter struct {
	Status *RestoreStatus
	model.ListRequest
}

var restoreColumns = []string{
	"id", "source_bucket", "source_object_path", "target_device_sns",
	"status", "progress", "error_message",
	"started_at", "completed_at", "created_at", "updated_at", "created_by",
}

// PgRestoreTaskRepository is the PostgreSQL implementation of RestoreTaskRepository.
type PgRestoreTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgRestoreTaskRepository constructs a PgRestoreTaskRepository.
func NewPgRestoreTaskRepository(pool *pgxpool.Pool) *PgRestoreTaskRepository {
	return &PgRestoreTaskRepository{pool: pool}
}

var _ RestoreTaskRepository = (*PgRestoreTaskRepository)(nil)

func (r *PgRestoreTaskRepository) Create(ctx context.Context, task *RestoreTask) error {
	snsJSON, err := json.Marshal(task.TargetDeviceSNs)
	if err != nil {
		return fmt.Errorf("marshal target_device_sns: %w", err)
	}
	query, args, err := storage.Psql.Insert("restore_tasks").
		Columns("source_bucket", "source_object_path", "target_device_sns",
			"status", "progress", "error_message",
			"started_at", "completed_at", "created_by").
		Values(task.SourceBucket, task.SourceObjectPath, snsJSON,
			task.Status, task.Progress, task.ErrorMessage,
			task.StartedAt, task.CompletedAt, task.CreatedBy).
		Suffix("RETURNING " + joinColumns(restoreColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert restore_task SQL: %w", err)
	}
	created, scanErr := scanRestoreTask(r.pool.QueryRow(ctx, query, args...))
	if scanErr != nil {
		return fmt.Errorf("create restore_task: %w", scanErr)
	}
	*task = *created
	return nil
}

func (r *PgRestoreTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*RestoreTask, error) {
	query, args, err := storage.Psql.Select(restoreColumns...).
		From("restore_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get restore_task SQL: %w", err)
	}
	task, scanErr := scanRestoreTask(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("restore_task %s: %w", id, commonerrors.ErrNotFound)
	}
	if scanErr != nil {
		return nil, fmt.Errorf("get restore_task: %w", scanErr)
	}
	return task, nil
}

func (r *PgRestoreTaskRepository) List(ctx context.Context, filter RestoreFilter) (*model.ListResponse[RestoreTask], error) {
	q := storage.Psql.Select(restoreColumns...).From("restore_tasks")
	cq := storage.Psql.Select("COUNT(*)").From("restore_tasks")
	if filter.Status != nil {
		q = q.Where(sq.Eq{"status": *filter.Status})
		cq = cq.Where(sq.Eq{"status": *filter.Status})
	}
	q = q.OrderBy("created_at DESC").Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	countSQL, countArgs, err := cq.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count restore_tasks SQL: %w", err)
	}
	var total int64
	if scanErr := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); scanErr != nil {
		return nil, fmt.Errorf("count restore_tasks: %w", scanErr)
	}

	listSQL, listArgs, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list restore_tasks SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("query restore_tasks: %w", err)
	}
	defer rows.Close()

	items := make([]RestoreTask, 0, filter.Limit())
	for rows.Next() {
		t, scanErr := scanRestoreTask(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan restore_task: %w", scanErr)
		}
		items = append(items, *t)
	}
	return model.NewListResponse(items, total, filter.Page, filter.Limit()), nil
}

// UpdateErrorMessage persists the skipped-devices note onto a restore_task
// row. Used by RestoreService when fan-out skips devices but the request as
// a whole succeeds (mismatch between API response and DB row would otherwise
// confuse the operator).
func (r *PgRestoreTaskRepository) UpdateErrorMessage(ctx context.Context, id uuid.UUID, msg string) error {
	query, args, err := storage.Psql.Update("restore_tasks").
		Set("error_message", msg).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update restore_task error_message SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update restore_task error_message: %w", err)
	}
	return nil
}

func scanRestoreTask(row pgx.Row) (*RestoreTask, error) {
	var t RestoreTask
	var snsJSON []byte
	if err := row.Scan(
		&t.ID, &t.SourceBucket, &t.SourceObjectPath, &snsJSON,
		&t.Status, &t.Progress, &t.ErrorMessage,
		&t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
	); err != nil {
		return nil, err
	}
	if len(snsJSON) > 0 {
		if err := json.Unmarshal(snsJSON, &t.TargetDeviceSNs); err != nil {
			return nil, fmt.Errorf("unmarshal target_device_sns: %w", err)
		}
	}
	if t.TargetDeviceSNs == nil {
		t.TargetDeviceSNs = []string{}
	}
	return &t, nil
}
