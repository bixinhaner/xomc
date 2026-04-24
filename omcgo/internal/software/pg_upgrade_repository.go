package software

import (
	"context"
	"database/sql"
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

var subTaskColumns = []string{
	"id", "task_id", "device_id", "firmware_id", "status",
	"error_message", "retry_count", "max_retries",
	"device_sn", "ori_version", "dest_version",
	"command_key", "failure_reason", "pre_suspend_status",
	"started_at", "completed_at", "created_at", "updated_at",
}

var _ SubTaskRepository = (*PgSubTaskRepository)(nil)

// PgSubTaskRepository is a PostgreSQL implementation of SubTaskRepository.
type PgSubTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgSubTaskRepository creates a new PgSubTaskRepository.
func NewPgSubTaskRepository(pool *pgxpool.Pool) *PgSubTaskRepository {
	return &PgSubTaskRepository{pool: pool}
}

func scanSubTask(row pgx.Row) (*UpgradeSubTask, error) {
	var t UpgradeSubTask
	var firmwareID sql.NullString
	var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend sql.NullString
	var startedAt, completedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
		&errorMsg, &t.RetryCount, &t.MaxRetries,
		&deviceSN, &oriVer, &destVer,
		&cmdKey, &failReason, &preSuspend,
		&startedAt, &completedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		t.FirmwareID = &fid
	}
	if errorMsg.Valid {
		t.ErrorMessage = errorMsg.String
	}
	if deviceSN.Valid {
		t.DeviceSN = deviceSN.String
	}
	if oriVer.Valid {
		t.OriVersion = oriVer.String
	}
	if destVer.Valid {
		t.DestVersion = destVer.String
	}
	if cmdKey.Valid {
		t.CommandKey = cmdKey.String
	}
	if failReason.Valid {
		t.FailureReason = failReason.String
	}
	if preSuspend.Valid {
		t.PreSuspendStatus = preSuspend.String
	}
	if startedAt.Valid {
		jt := JSONTime(startedAt.Time)
		t.StartedAt = &jt
	}
	if completedAt.Valid {
		jt := JSONTime(completedAt.Time)
		t.CompletedAt = &jt
	}
	t.CreatedAt = JSONTime(createdAt)
	t.UpdatedAt = JSONTime(updatedAt)
	return &t, nil
}

func scanSubTaskRow(rows pgx.Rows) (*UpgradeSubTask, error) {
	var t UpgradeSubTask
	var firmwareID sql.NullString
	var errorMsg, deviceSN, oriVer, destVer, cmdKey, failReason, preSuspend sql.NullString
	var startedAt, completedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&t.ID, &t.TaskID, &t.DeviceID, &firmwareID, &t.Status,
		&errorMsg, &t.RetryCount, &t.MaxRetries,
		&deviceSN, &oriVer, &destVer,
		&cmdKey, &failReason, &preSuspend,
		&startedAt, &completedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		t.FirmwareID = &fid
	}
	if errorMsg.Valid {
		t.ErrorMessage = errorMsg.String
	}
	if deviceSN.Valid {
		t.DeviceSN = deviceSN.String
	}
	if oriVer.Valid {
		t.OriVersion = oriVer.String
	}
	if destVer.Valid {
		t.DestVersion = destVer.String
	}
	if cmdKey.Valid {
		t.CommandKey = cmdKey.String
	}
	if failReason.Valid {
		t.FailureReason = failReason.String
	}
	if preSuspend.Valid {
		t.PreSuspendStatus = preSuspend.String
	}
	if startedAt.Valid {
		jt := JSONTime(startedAt.Time)
		t.StartedAt = &jt
	}
	if completedAt.Valid {
		jt := JSONTime(completedAt.Time)
		t.CompletedAt = &jt
	}
	t.CreatedAt = JSONTime(createdAt)
	t.UpdatedAt = JSONTime(updatedAt)
	return &t, nil
}

func (r *PgSubTaskRepository) Create(ctx context.Context, task *UpgradeSubTask) error {
	builder := storage.Psql.Insert("upgrade_sub_tasks").
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version").
		Values(task.TaskID, task.DeviceID, task.FirmwareID, task.Status, task.MaxRetries,
			task.DeviceSN, task.OriVersion, task.DestVersion).
		Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert sub-task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanSubTask(row)
	if err != nil {
		return fmt.Errorf("create sub-task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgSubTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get sub-task SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get sub-task: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error {
	builder := storage.Psql.Update("upgrade_sub_tasks").
		Set("status", status).
		Where(sq.Eq{"id": id})

	if errorMsg != "" {
		builder = builder.Set("error_message", errorMsg)
	}
	if status == UpgradeDownloading {
		builder = builder.Set("started_at", time.Now())
	}
	if status == UpgradeCompleted || status == UpgradeFailed || status == UpgradeTerminated {
		builder = builder.Set("completed_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update sub-task status SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update sub-task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepository) Update(ctx context.Context, task *UpgradeSubTask) error {
	builder := storage.Psql.Update("upgrade_sub_tasks").
		Set("status", task.Status).
		Set("error_message", task.ErrorMessage).
		Set("retry_count", task.RetryCount).
		Set("command_key", task.CommandKey).
		Set("failure_reason", task.FailureReason).
		Set("pre_suspend_status", task.PreSuspendStatus).
		Where(sq.Eq{"id": task.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update sub-task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update sub-task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgSubTaskRepository) List(ctx context.Context, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTask], error) {
	return r.ListByTaskID(ctx, filter.TaskID, filter)
}

func (r *PgSubTaskRepository) ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTask], error) {
	base := storage.Psql.Select(subTaskColumns...).From("upgrade_sub_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("upgrade_sub_tasks")

	base = base.Where(sq.Eq{"task_id": taskID})
	countBase = countBase.Where(sq.Eq{"task_id": taskID})

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count sub-tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sub-tasks: %w", err)
	}

	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, args, err := base.
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list sub-tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sub-tasks: %w", err)
	}
	defer rows.Close()

	var items []UpgradeSubTask
	for rows.Next() {
		t, err := scanSubTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sub-task row: %w", err)
		}
		items = append(items, *t)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[UpgradeSubTask]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgSubTaskRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.And{
			sq.Eq{"device_id": deviceID},
			sq.NotEq{"status": []UpgradeState{UpgradeCompleted, UpgradeFailed, UpgradeTerminated}},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active sub-task SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get active sub-task: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error) {
	query, args, err := storage.Psql.Select(subTaskColumns...).
		From("upgrade_sub_tasks").
		Where(sq.Eq{"command_key": commandKey}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get by command key SQL: %w", err)
	}

	task, err := scanSubTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get sub-task by command key: %w", err)
	}
	return task, nil
}

func (r *PgSubTaskRepository) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
	if len(tasks) == 0 {
		return nil
	}

	builder := storage.Psql.Insert("upgrade_sub_tasks").
		Columns("task_id", "device_id", "firmware_id", "status", "max_retries",
			"device_sn", "ori_version", "dest_version")

	for _, t := range tasks {
		builder = builder.Values(
			t.TaskID, t.DeviceID, t.FirmwareID, t.Status, t.MaxRetries,
			t.DeviceSN, t.OriVersion, t.DestVersion,
		)
	}

	builder = builder.Suffix("RETURNING " + joinColumns(subTaskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert sub-tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch create sub-tasks: %w", err)
	}
	defer rows.Close()

	idx := 0
	for rows.Next() {
		created, err := scanSubTaskRow(rows)
		if err != nil {
			return fmt.Errorf("scan batch created sub-task: %w", err)
		}
		if idx < len(tasks) {
			*tasks[idx] = *created
		}
		idx++
	}
	return nil
}

func (r *PgSubTaskRepository) FailStale(ctx context.Context, cutoff time.Time) (int64, error) {
	query, args, err := storage.Psql.Update("upgrade_sub_tasks").
		Set("status", UpgradeFailed).
		Set("error_message", "task timeout").
		Set("completed_at", time.Now()).
		Where(sq.And{
			sq.NotEq{"status": []UpgradeState{UpgradeCompleted, UpgradeFailed, UpgradeTerminated}},
			sq.Lt{"updated_at": cutoff},
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build fail stale sub-tasks SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("fail stale sub-tasks: %w", err)
	}
	return result.RowsAffected(), nil
}
