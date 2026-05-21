package repo

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
	"github.com/omcgo/omcgo/internal/software"
)

// taskColumns 跟 software.taskColumns 严格保持一致 — 4 张新表 schema 由
// CREATE TABLE LIKE upgrade_tasks 复制而来，列顺序一致才能复用 scan 函数。
// canary 字段（strategy/canary_stages/...）4 类新业务都用不上，故不出现在此 SELECT 中，
// 避免业务方误用。如未来需要支持 canary，在 SELECT 列表加列即可。
var taskColumns = []string{
	"id", "task_name", "task_type", "firmware_id", "download_file_type", "file_name", "file_md5",
	"status", "result", "product_class", "is_keep_config",
	"create_status", "create_user", "total_count", "success_count", "fail_count",
	"max_concurrent", "started_at", "ended_at", "created_at", "updated_at",
	"rollback_reason", "rollback_source", "rollback_target_firmware_id",
}

var _ TaskRepo = (*PgTaskRepo)(nil)

// PgTaskRepo 是 TaskRepo 接口的 PostgreSQL 实现。
// 通过 tableName 字段适配 config_backup_tasks / config_restore_tasks 等不同物理表。
type PgTaskRepo struct {
	pool      *pgxpool.Pool
	tableName string // 必须是 repository.go 中 Table* 常量之一
}

// NewPgTaskRepo 创建 main task 仓储。tableName 必须为下表之一：
//
//	config_backup_tasks / config_restore_tasks / runtime_log_collect_tasks / fault_log_collect_tasks
func NewPgTaskRepo(pool *pgxpool.Pool, tableName string) *PgTaskRepo {
	return &PgTaskRepo{pool: pool, tableName: tableName}
}

// scanTask 跟 software.scanUpgradeTask 字段对齐；放在本包内避免跨包引用 unexported 函数。
func scanTask(row pgx.Row) (*software.UpgradeTask, error) {
	var task software.UpgradeTask
	var firmwareID sql.NullString
	var downloadFileType sql.NullString
	var fileName, fileMD5, result sql.NullString
	var startedAt, endedAt sql.NullTime
	var createdAt, updatedAt time.Time
	var rollbackReason, rollbackSource, rollbackTargetFW sql.NullString

	err := row.Scan(
		&task.ID, &task.TaskName, &task.TaskType, &firmwareID, &downloadFileType, &fileName, &fileMD5,
		&task.Status, &result, &task.ProductClass, &task.IsKeepConfig,
		&task.CreateStatus, &task.CreateUser, &task.TotalCount, &task.SuccessCount, &task.FailCount,
		&task.MaxConcurrent, &startedAt, &endedAt, &createdAt, &updatedAt,
		&rollbackReason, &rollbackSource, &rollbackTargetFW,
	)
	if err != nil {
		return nil, err
	}
	if firmwareID.Valid {
		fid, _ := uuid.Parse(firmwareID.String)
		task.FirmwareID = &fid
	}
	if downloadFileType.Valid {
		task.DownloadFileType = downloadFileType.String
	}
	if fileName.Valid {
		task.FileName = fileName.String
	}
	if fileMD5.Valid {
		task.FileMD5 = fileMD5.String
	}
	if result.Valid {
		task.Result = software.TaskResult(result.String)
	}
	if startedAt.Valid {
		t := software.JSONTime(startedAt.Time)
		task.StartedAt = &t
	}
	if endedAt.Valid {
		t := software.JSONTime(endedAt.Time)
		task.EndedAt = &t
	}
	task.CreatedAt = software.JSONTime(createdAt)
	task.UpdatedAt = software.JSONTime(updatedAt)
	if rollbackReason.Valid {
		task.RollbackReason = rollbackReason.String
	}
	if rollbackSource.Valid {
		task.RollbackSource = rollbackSource.String
	}
	if rollbackTargetFW.Valid {
		if fid, err := uuid.Parse(rollbackTargetFW.String); err == nil {
			task.RollbackTargetFirmwareID = &fid
		}
	}
	return &task, nil
}

func (r *PgTaskRepo) Create(ctx context.Context, task *software.UpgradeTask) error {
	// 跟 software.PgTaskRepository.Create 同款的 rollback_source CHECK 兜底；
	// 备份 / 下发 / 日志业务用不上 rollback_*，但表里有列、CHECK 约束需要满足。
	rollbackSource := task.RollbackSource
	if rollbackSource == "" {
		rollbackSource = software.RollbackSourceManual
	}
	var rollbackReason any
	if task.RollbackReason != "" {
		rollbackReason = task.RollbackReason
	}
	var rollbackTargetFW any
	if task.RollbackTargetFirmwareID != nil {
		rollbackTargetFW = *task.RollbackTargetFirmwareID
	}

	builder := storage.Psql.Insert(r.tableName).
		Columns("task_name", "task_type", "firmware_id", "download_file_type", "file_name", "file_md5",
			"status", "product_class", "is_keep_config",
			"create_status", "create_user", "total_count", "max_concurrent",
			"rollback_reason", "rollback_source", "rollback_target_firmware_id").
		Values(task.TaskName, task.TaskType, task.FirmwareID, task.DownloadFileType, task.FileName, task.FileMD5,
			task.Status, task.ProductClass, task.IsKeepConfig,
			task.CreateStatus, task.CreateUser, task.TotalCount, task.MaxConcurrent,
			rollbackReason, rollbackSource, rollbackTargetFW).
		Suffix("RETURNING " + joinColumns(taskColumns))

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert %s SQL: %w", r.tableName, err)
	}
	created, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return fmt.Errorf("create %s: %w", r.tableName, err)
	}
	*task = *created
	return nil
}

func (r *PgTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*software.UpgradeTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From(r.tableName).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get %s SQL: %w", r.tableName, err)
	}
	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get %s: %w", r.tableName, err)
	}
	return task, nil
}

func (r *PgTaskRepo) Update(ctx context.Context, task *software.UpgradeTask) error {
	builder := storage.Psql.Update(r.tableName).
		Set("task_name", task.TaskName).
		Set("task_type", task.TaskType).
		Set("firmware_id", task.FirmwareID).
		Set("download_file_type", task.DownloadFileType).
		Set("file_name", task.FileName).
		Set("file_md5", task.FileMD5).
		Set("status", task.Status).
		Set("result", task.Result).
		Set("product_class", task.ProductClass).
		Set("is_keep_config", task.IsKeepConfig).
		Set("create_status", task.CreateStatus).
		Set("create_user", task.CreateUser).
		Set("total_count", task.TotalCount).
		Set("success_count", task.SuccessCount).
		Set("fail_count", task.FailCount).
		Set("max_concurrent", task.MaxConcurrent).
		Set("ended_at", task.EndedAt).
		Where(sq.Eq{"id": task.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update %s SQL: %w", r.tableName, err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s: %w", r.tableName, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status software.TaskStatus, result software.TaskResult) error {
	builder := storage.Psql.Update(r.tableName).
		Set("status", status).
		Where(sq.Eq{"id": id})

	if result != "" {
		builder = builder.Set("result", result)
	}
	if status == software.TaskInProgress {
		builder = builder.Set("started_at", time.Now())
	}
	if status == software.TaskEnded {
		builder = builder.Set("ended_at", time.Now())
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update %s status SQL: %w", r.tableName, err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s status: %w", r.tableName, err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepo) List(ctx context.Context, filter software.UpgradeTaskFilter) (*model.ListResponse[software.UpgradeTask], error) {
	base := storage.Psql.Select(taskColumns...).From(r.tableName)
	countBase := storage.Psql.Select("COUNT(*)").From(r.tableName)

	if filter.TaskType != nil {
		base = base.Where(sq.Eq{"task_type": *filter.TaskType})
		countBase = countBase.Where(sq.Eq{"task_type": *filter.TaskType})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.ProductClass != nil {
		base = base.Where(sq.Eq{"product_class": *filter.ProductClass})
		countBase = countBase.Where(sq.Eq{"product_class": *filter.ProductClass})
	}
	if filter.CreateUser != nil {
		base = base.Where(sq.Eq{"create_user": *filter.CreateUser})
		countBase = countBase.Where(sq.Eq{"create_user": *filter.CreateUser})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count %s SQL: %w", r.tableName, err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", r.tableName, err)
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
		return nil, fmt.Errorf("build list %s SQL: %w", r.tableName, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", r.tableName, err)
	}
	defer rows.Close()

	var items []software.UpgradeTask
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s row: %w", r.tableName, err)
		}
		items = append(items, *task)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[software.UpgradeTask]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgTaskRepo) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
	query := fmt.Sprintf(`UPDATE %s
		SET success_count = success_count + $1,
		    fail_count    = fail_count    + $2,
		    updated_at    = NOW()
		WHERE id = $3`, r.tableName)
	_, err := r.pool.Exec(ctx, query, successDelta, failDelta, taskID)
	if err != nil {
		return fmt.Errorf("increment %s counts: %w", r.tableName, err)
	}
	return nil
}

func (r *PgTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete(r.tableName).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s SQL: %w", r.tableName, err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete %s: %w", r.tableName, err)
	}
	return nil
}

// joinColumns 把列名拼成逗号分隔串供 RETURNING 子句使用。
func joinColumns(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}
