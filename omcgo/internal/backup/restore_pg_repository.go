package backup

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

	// FindByIDPrefix returns up to `limit` restore_tasks whose UUID (without
	// dashes) starts with the given hex prefix. Used by TransferCompleteRouter
	// to map the `{taskID8}` segment of a Download CommandKey back to the
	// originating restore_task (M2 of backup-restore-alignment-plan).
	FindByIDPrefix(ctx context.Context, prefix string, limit int) ([]*RestoreTask, error)

	// MarkComplete writes the final status / task_result / completed_at of a
	// restore_task. Used by TransferCompleteRouter after CPE acknowledges the
	// Download. errMsg is appended only on failure (success path keeps any
	// pre-existing skipped-devices note from UpdateErrorMessage intact).
	MarkComplete(ctx context.Context, id uuid.UUID, status RestoreStatus, result int16, completedAt time.Time, errMsg string) error
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
	// M1 alignment columns.
	"task_seq", "task_name", "task_result", "operator_code", "create_user",
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
			"started_at", "completed_at", "created_by",
			"task_name", "task_result", "operator_code", "create_user").
		Values(task.SourceBucket, task.SourceObjectPath, snsJSON,
			task.Status, task.Progress, task.ErrorMessage,
			task.StartedAt, task.CompletedAt, task.CreatedBy,
			task.TaskName, task.TaskResult, task.OperatorCode, task.CreateUser).
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
		&t.TaskSeq, &t.TaskName, &t.TaskResult, &t.OperatorCode, &t.CreateUser,
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

// FindByIDPrefix — see RestoreTaskRepository interface. Mirrors the backup
// repo helper of the same name (M2 of backup-restore-alignment-plan).
func (r *PgRestoreTaskRepository) FindByIDPrefix(ctx context.Context, prefix string, limit int) ([]*RestoreTask, error) {
	if prefix == "" {
		return nil, nil
	}
	if limit < 1 {
		limit = 1
	}
	q := storage.Psql.Select(restoreColumns...).
		From("restore_tasks").
		Where("REPLACE(id::text, '-', '') LIKE ?", prefix+"%").
		OrderBy("created_at DESC").
		Limit(uint64(limit))
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find restore_task by prefix SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query restore_task by prefix: %w", err)
	}
	defer rows.Close()
	out := make([]*RestoreTask, 0, limit)
	for rows.Next() {
		t, scanErr := scanRestoreTask(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan restore_task row: %w", scanErr)
		}
		out = append(out, t)
	}
	return out, nil
}

// MarkComplete — see RestoreTaskRepository interface. Sets terminal status +
// task_result + completed_at on the row. Updates are idempotent (subsequent
// calls re-write the same values); errMsg is only persisted on failure to
// preserve any pre-existing skipped-devices note set by UpdateErrorMessage.
func (r *PgRestoreTaskRepository) MarkComplete(
	ctx context.Context, id uuid.UUID, status RestoreStatus, result int16, completedAt time.Time, errMsg string,
) error {
	upd := storage.Psql.Update("restore_tasks").
		Set("status", status).
		Set("task_result", result).
		Set("completed_at", completedAt).
		Set("progress", 100).
		Where(sq.Eq{"id": id})
	if status == RestoreFailed && errMsg != "" {
		upd = upd.Set("error_message", errMsg)
	}
	query, args, err := upd.ToSql()
	if err != nil {
		return fmt.Errorf("build mark restore_task complete SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark restore_task complete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("restore_task %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}
