package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgTaskRepository 实现 TaskRepository 接口
type PgTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgTaskRepository 创建 PostgreSQL 任务仓库
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	return &PgTaskRepository{pool: pool}
}

// nilUUID converts an empty string to nil for nullable UUID columns.
func nilUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Create 创建任务记录
func (r *PgTaskRepository) Create(ctx context.Context, task *Task) error {
	query, args, err := storage.Psql.Insert("device_tasks").
		Columns(
			"id", "device_sn", "method", "params", "priority",
			"command_key", "cwmp_id", "status", "retry_count", "max_retries",
			"created_at", "sent_at", "completed_at", "expires_at",
			"result", "error_code", "error_message",
			"source", "creator_id", "description",
			"source_id", "command_index", "device_index",
			"has_path_translation_miss", "path_translation_miss_count",
		).
		Values(
			task.ID, task.DeviceSN, task.Method, task.Params, task.Priority,
			task.CommandKey, task.CWMPID, task.Status, task.RetryCount, task.MaxRetries,
			task.CreatedAt, task.SentAt, task.CompletedAt, task.ExpiresAt,
			task.Result, task.ErrorCode, task.ErrorMessage,
			task.Source, task.CreatorID, task.Description,
			nilUUID(task.SourceID), task.CommandIndex, task.DeviceIndex,
			task.HasPathTranslationMiss, task.PathTranslationMissCount,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// Update 更新任务记录
func (r *PgTaskRepository) Update(ctx context.Context, task *Task) error {
	query, args, err := storage.Psql.Update("device_tasks").
		Set("method", task.Method).
		Set("params", task.Params).
		Set("priority", task.Priority).
		Set("command_key", task.CommandKey).
		Set("cwmp_id", task.CWMPID).
		Set("status", task.Status).
		Set("retry_count", task.RetryCount).
		Set("max_retries", task.MaxRetries).
		Set("sent_at", task.SentAt).
		Set("completed_at", task.CompletedAt).
		Set("expires_at", task.ExpiresAt).
		Set("result", task.Result).
		Set("error_code", task.ErrorCode).
		Set("error_message", task.ErrorMessage).
		Where(sq.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取任务
func (r *PgTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	return r.scanTask(ctx, query, args...)
}

// GetByCWMPID 根据 CWMP ID 获取任务
func (r *PgTaskRepository) GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"cwmp_id": cwmpID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	return r.scanTask(ctx, query, args...)
}

// GetHistory 获取设备任务历史
func (r *PgTaskRepository) GetHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error) {
	if opts == nil {
		opts = &TaskHistoryOptions{Page: 1, PageSize: 20}
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 100 {
		opts.PageSize = 20
	}

	whereClause := sq.Eq{"device_sn": deviceSN}
	if opts.Status != "" {
		whereClause["status"] = opts.Status
	}

	var timeConditions []sq.Sqlizer
	if opts.Start != nil {
		timeConditions = append(timeConditions, sq.GtOrEq{"created_at": opts.Start})
	}
	if opts.End != nil {
		timeConditions = append(timeConditions, sq.LtOrEq{"created_at": opts.End})
	}

	baseQuery := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(whereClause)

	for _, cond := range timeConditions {
		baseQuery = baseQuery.Where(cond)
	}

	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("device_tasks").
		Where(whereClause).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int64
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	offset := (opts.Page - 1) * opts.PageSize
	query, args, err := baseQuery.
		OrderBy("created_at DESC").
		Limit(uint64(opts.PageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := r.scanTaskRow(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// GetPendingByDevice 获取设备的所有 pending 状态任务
func (r *PgTaskRepository) GetPendingByDevice(ctx context.Context, deviceSN string) ([]*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{
			"device_sn": deviceSN,
			"status":    TaskStatusPending,
		}).
		OrderBy("priority ASC, created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := r.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListOpenByDeviceAndMethods 列出指定设备的 pending/sent 状态任务（用于 RebootCloser）。
func (r *PgTaskRepository) ListOpenByDeviceAndMethods(ctx context.Context, deviceSN string, methods []string) ([]*Task, error) {
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.And{
			sq.Eq{"device_sn": deviceSN},
			sq.Eq{"method": methods},
			sq.Eq{"status": []TaskStatus{TaskStatusPending, TaskStatusSent}},
		}).
		OrderBy("created_at ASC")
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list open tasks query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query open tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t, err := r.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ListPendingAllDevices 列出所有 pending 状态的任务（用于 RestorePendingQueues）。
func (r *PgTaskRepository) ListPendingAllDevices(ctx context.Context, limit int) ([]*Task, error) {
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"status": TaskStatusPending}).
		OrderBy("created_at ASC")
	if limit > 0 {
		q = q.Limit(uint64(limit))
	}
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list pending query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t, err := r.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// Delete 删除任务
func (r *PgTaskRepository) Delete(ctx context.Context, id string) error {
	query, args, err := storage.Psql.Delete("device_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

// BatchCreate 批量创建任务
func (r *PgTaskRepository) BatchCreate(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	columns := []string{
		"id", "device_sn", "method", "params", "priority",
		"command_key", "cwmp_id", "status", "retry_count", "max_retries",
		"created_at", "sent_at", "completed_at", "expires_at",
		"result", "error_code", "error_message",
		"source", "creator_id", "description",
		"source_id", "command_index", "device_index",
		"has_path_translation_miss", "path_translation_miss_count",
	}

	insertBuilder := storage.Psql.Insert("device_tasks").Columns(columns...)

	for _, task := range tasks {
		insertBuilder = insertBuilder.Values(
			task.ID, task.DeviceSN, task.Method, task.Params, task.Priority,
			task.CommandKey, task.CWMPID, task.Status, task.RetryCount, task.MaxRetries,
			task.CreatedAt, task.SentAt, task.CompletedAt, task.ExpiresAt,
			task.Result, task.ErrorCode, task.ErrorMessage,
			task.Source, task.CreatorID, task.Description,
			task.SourceID, task.CommandIndex, task.DeviceIndex,
			task.HasPathTranslationMiss, task.PathTranslationMissCount,
		)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch insert tasks: %w", err)
	}

	return nil
}

// PathTranslationMissStats 是按 source_id 聚合的 device_tasks 路径翻译 miss 统计。
//
// 用于 MML 任务详情接口（GET /api/v1/mml/tasks/:id）—前端任务详情页据此显示
// "路径翻译警告" 标签 + 详细计数（被影响 device 数 / miss path 总数）。
type PathTranslationMissStats struct {
	DeviceCount int   `json:"device_count"`
	PathCount   int64 `json:"path_count"`
	AnyMiss     bool  `json:"any_miss"`
}

// AggregatePathTranslationMissBySourceID 聚合特定 source_id (mml_task.id) 下
// 所有 device_tasks 的 has_path_translation_miss / path_translation_miss_count，
// 用于 MML 任务详情聚合显示（Stage 3 — 整改方案 UI 警告标签）。
func (r *PgTaskRepository) AggregatePathTranslationMissBySourceID(
	ctx context.Context, sourceID string,
) (PathTranslationMissStats, error) {
	const q = `
SELECT
    COUNT(*) FILTER (WHERE has_path_translation_miss) AS device_count,
    COALESCE(SUM(path_translation_miss_count), 0)::bigint AS path_count
FROM device_tasks
WHERE source = 'mml' AND source_id = $1`
	var stats PathTranslationMissStats
	if err := r.pool.QueryRow(ctx, q, sourceID).Scan(&stats.DeviceCount, &stats.PathCount); err != nil {
		return PathTranslationMissStats{}, fmt.Errorf("aggregate path translation miss: %w", err)
	}
	stats.AnyMiss = stats.DeviceCount > 0 || stats.PathCount > 0
	return stats, nil
}

// CountByStatus 统计各状态任务数量
func (r *PgTaskRepository) CountByStatus(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
	query, args, err := storage.Psql.Select("status", "COUNT(*) as count").
		From("device_tasks").
		Where(sq.Eq{"device_sn": deviceSN}).
		GroupBy("status").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[TaskStatus]int64)
	for rows.Next() {
		var status TaskStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[status] = count
	}

	return counts, nil
}

// PurgeOldTasks 清理指定时间之前的已完成任务
func (r *PgTaskRepository) PurgeOldTasks(ctx context.Context, before string) (int64, error) {
	beforeTime, err := time.Parse(time.RFC3339, before)
	if err != nil {
		return 0, fmt.Errorf("parse time: %w", err)
	}

	query, args, err := storage.Psql.Delete("device_tasks").
		Where(sq.Eq{"status": []TaskStatus{TaskStatusCompleted, TaskStatusFailed, TaskStatusExpired, TaskStatusCancelled}}).
		Where(sq.Lt{"completed_at": beforeTime}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("purge tasks: %w", err)
	}

	return result.RowsAffected(), nil
}

// taskColumns 返回任务表的列名
func taskColumns() []string {
	return []string{
		"id", "device_sn", "method", "params", "priority",
		"command_key", "cwmp_id", "status", "retry_count", "max_retries",
		"created_at", "sent_at", "completed_at", "expires_at",
		"result", "error_code", "error_message",
		"source", "creator_id", "description",
		"source_id", "command_index", "device_index",
		"has_path_translation_miss", "path_translation_miss_count",
	}
}

// scanTask 扫描单个任务
func (r *PgTaskRepository) scanTask(ctx context.Context, query string, args ...any) (*Task, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanTaskRow(row)
}

// scanTaskRow 扫描任务行
func (r *PgTaskRepository) scanTaskRow(row pgx.Row) (*Task, error) {
	var task Task
	var params, result []byte
	// source_id 列允许 NULL（pre-existing 任务可能没 source_id）；
	// pgx 直接 Scan NULL 到 *string 会失败，用 NullString 中介。
	var sourceID sql.NullString

	err := row.Scan(
		&task.ID, &task.DeviceSN, &task.Method, &params, &task.Priority,
		&task.CommandKey, &task.CWMPID, &task.Status, &task.RetryCount, &task.MaxRetries,
		&task.CreatedAt, &task.SentAt, &task.CompletedAt, &task.ExpiresAt,
		&result, &task.ErrorCode, &task.ErrorMessage,
		&task.Source, &task.CreatorID, &task.Description,
		&sourceID, &task.CommandIndex, &task.DeviceIndex,
		&task.HasPathTranslationMiss, &task.PathTranslationMissCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	if len(params) > 0 {
		task.Params = json.RawMessage(params)
	}
	if len(result) > 0 {
		task.Result = json.RawMessage(result)
	}
	if sourceID.Valid {
		task.SourceID = sourceID.String
	}

	return &task, nil
}
