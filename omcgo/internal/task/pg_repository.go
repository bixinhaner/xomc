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

// nilString converts an empty string to nil for nullable VARCHAR/TEXT columns（T-0168）。
func nilString(s string) any {
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
			"path_translation_source", // T-0168
		).
		Values(
			task.ID, task.DeviceSN, task.Method, task.Params, task.Priority,
			task.CommandKey, task.CWMPID, task.Status, task.RetryCount, task.MaxRetries,
			task.CreatedAt, task.SentAt, task.CompletedAt, task.ExpiresAt,
			task.Result, task.ErrorCode, task.ErrorMessage,
			task.Source, task.CreatorID, task.Description,
			nilUUID(task.SourceID), task.CommandIndex, task.DeviceIndex,
			task.HasPathTranslationMiss, task.PathTranslationMissCount,
			nilString(task.PathTranslationSource), // T-0168: 空字符串 → NULL
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

func (r *PgTaskRepository) HasIncompleteSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error) {
	prefix := fmt.Sprintf("sync-gpv-%s", deviceSN)
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("device_tasks").
		Where(sq.Eq{"device_sn": deviceSN}).
		Where(sq.Like{"command_key": prefix + "%"}).
		Where(sq.Eq{"status": []TaskStatus{TaskStatusPending, TaskStatusSent}}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build incomplete sync-gpv query: %w", err)
	}
	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("query incomplete sync-gpv tasks: %w", err)
	}
	return count > 0, nil
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

// defaultPendingBatchLimit 是 ListPendingAllDevices 在调用方未给上界（limit<=0）时
// 的兜底批大小。百万设备下 device_tasks 的 pending 行可能极多，一次性 SELECT 全量
// 入内存会 OOM（#11）。RestorePendingQueues 走 ListPendingPage 流式分批恢复，本兜底
// 仅防御直连本方法且不传上界的旧调用。
const defaultPendingBatchLimit = 1000

// PendingCursor 是 pending 任务列表的 keyset 游标。零值（CreatedAt 零 + ID 空）
// 表示从头开始。按 (created_at, id) 升序推进，id 作 created_at 相同的 tiebreaker，
// 保证翻页不漏不重（OFFSET 在大表上随页深线性变慢，keyset 恒定代价）。
type PendingCursor struct {
	CreatedAt time.Time
	ID        string
}

// IsZero 报告游标是否为初始（从头扫）。
func (c PendingCursor) IsZero() bool {
	return c.ID == "" && c.CreatedAt.IsZero()
}

// ListPendingAllDevices 列出所有 pending 状态的任务（用于 RestorePendingQueues）。
//
// limit<=0 时不再隐式拉全表，而是落 defaultPendingBatchLimit 兜底上界防 OOM（#11）。
// 大规模恢复请走 ListPendingPage 流式分批，避免整批驻留内存。
func (r *PgTaskRepository) ListPendingAllDevices(ctx context.Context, limit int) ([]*Task, error) {
	if limit <= 0 {
		limit = defaultPendingBatchLimit
	}
	return r.ListPendingPage(ctx, PendingCursor{}, limit)
}

// buildPendingPageSQL 构造 keyset 分页查询 SQL。抽出供单测验证游标谓词，
// 运行期由 ListPendingPage 调用。
func buildPendingPageSQL(after PendingCursor, batchSize int) (string, []any, error) {
	if batchSize <= 0 {
		batchSize = defaultPendingBatchLimit
	}
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"status": TaskStatusPending})
	if !after.IsZero() {
		// keyset：(created_at, id) > (cursor.created_at, cursor.id)，行级元组比较。
		q = q.Where(sq.Expr("(created_at, id) > (?, ?)", after.CreatedAt, after.ID))
	}
	q = q.OrderBy("created_at ASC", "id ASC").Limit(uint64(batchSize))
	return q.ToSql()
}

// ListPendingPage 按 keyset 游标返回一页 pending 任务（按 (created_at, id) 升序）。
// 返回行数 < batchSize 表示已到末页。调用方用每页最后一条的 (CreatedAt, ID) 构造下一页
// 游标继续推进，从而流式恢复而不必一次性把全量 pending 任务读入内存（#11）。
func (r *PgTaskRepository) ListPendingPage(ctx context.Context, after PendingCursor, batchSize int) ([]*Task, error) {
	query, args, err := buildPendingPageSQL(after, batchSize)
	if err != nil {
		return nil, fmt.Errorf("build list pending page query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending tasks page: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending tasks page: %w", err)
	}
	return tasks, nil
}

// ListExpiredCandidates 查找已过期但仍处于活跃态(pending/sent)的任务（T-0157 C2）。
//
// 条件：expires_at IS NOT NULL AND expires_at < now() AND status IN (pending, sent)。
// 排序：expires_at ASC（最早过期的先处理）。limit > 0 时限制单批数量防 worker 长事务。
//
// 调用方（worker/task_sweeper）拿到后调 task.MarkExpired + Update + publish task.expired。
func (r *PgTaskRepository) ListExpiredCandidates(ctx context.Context, now time.Time, limit int) ([]*Task, error) {
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.NotEq{"expires_at": nil}).
		Where(sq.Lt{"expires_at": now}).
		Where(sq.Eq{"status": []TaskStatus{TaskStatusPending, TaskStatusSent}}).
		OrderBy("expires_at ASC")
	if limit > 0 {
		q = q.Limit(uint64(limit))
	}
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expired query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query expired tasks: %w", err)
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
		"path_translation_source", // T-0168
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
			nilString(task.PathTranslationSource), // T-0168
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

// DeviceTaskResultRow 表示按 source_id（mml_task.id）聚出来的单条 device_task
// 执行结果行，用于 GET /api/v1/mml/tasks/:id/results 在前端展示设备级输出。
//
// 历史问题（2026-05-23 修复）：原来 mml.GetTaskResults 读 mml_tasks.results
// JSONB 列，但执行结果从未由 ACS 回写到 mml_tasks，真实数据全在 device_tasks。
// 改为直接按 source_id 查 device_tasks，可一步把"列表里看见成功/失败但点查
// 看永远空"修好。
type DeviceTaskResultRow struct {
	ID           string // device_tasks.id（CSV 导出「子任务ID」、区分整体/逐 PATH 报文归属）
	DeviceSN     string
	Status       string
	ErrorCode    int
	ErrorMessage string
	Result       json.RawMessage // JSONB（device_tasks.result），含 method / raw_response
	SentAt       *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	CommandIndex int
	DeviceIndex  int
}

// ListResultsBySourceID 返回某个 source_id（典型为 mml_task.id）下所有 device_tasks
// 的执行结果，按 (command_index, device_index, created_at) 升序稳定排序。
//
// total 取自全量 COUNT；items 分页。pageSize <= 0 时退化为 20，page <= 0 退化为 1。
func (r *PgTaskRepository) ListResultsBySourceID(
	ctx context.Context, sourceID string, page, pageSize int,
) ([]DeviceTaskResultRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	const countQ = `
SELECT COUNT(*) FROM device_tasks
WHERE source = 'mml' AND source_id = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQ, sourceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count device tasks by source: %w", err)
	}
	if total == 0 {
		return []DeviceTaskResultRow{}, 0, nil
	}

	const listQ = `
SELECT id, device_sn, status,
       COALESCE(error_code, 0), COALESCE(error_message, ''),
       result, sent_at, completed_at, created_at,
       COALESCE(command_index, 0), COALESCE(device_index, 0)
FROM device_tasks
WHERE source = 'mml' AND source_id = $1
ORDER BY command_index ASC, device_index ASC, created_at ASC
LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, listQ, sourceID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query device tasks by source: %w", err)
	}
	defer rows.Close()

	items := make([]DeviceTaskResultRow, 0, pageSize)
	for rows.Next() {
		var row DeviceTaskResultRow
		var raw []byte
		if err := rows.Scan(
			&row.ID, &row.DeviceSN, &row.Status,
			&row.ErrorCode, &row.ErrorMessage,
			&raw, &row.SentAt, &row.CompletedAt, &row.CreatedAt,
			&row.CommandIndex, &row.DeviceIndex,
		); err != nil {
			return nil, 0, fmt.Errorf("scan device task row: %w", err)
		}
		if len(raw) > 0 {
			row.Result = json.RawMessage(raw)
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate device task rows: %w", err)
	}
	return items, total, nil
}

// DeviceTaskSourceStats 是按 source_id 聚合的 device_tasks 终态统计，
// 供上层（如 MML ResultAggregator）以"实际派发的 device_tasks 全部进入终态"
// 作为来源任务的完成判据 —— 替代"理论 total = 设备数 × 命令数"那种在 fanout /
// sequencer 跳过任意 (设备,命令) 时永远到不了的脆弱判据。
type DeviceTaskSourceStats struct {
	Total     int // 该 source_id 下所有 device_tasks
	Completed int // status='completed'
	Failed    int // status IN ('failed','expired')
	Active    int // 非终态（Total-Completed-Failed），即仍在 pending/在途
}

// AggregateStatusBySourceID 统计某 (source, source_id) 下 device_tasks 的终态分布。
// 终态 = completed / failed / expired；其余（pending 等）计入 Active。
func (r *PgTaskRepository) AggregateStatusBySourceID(
	ctx context.Context, source TaskSource, sourceID string,
) (DeviceTaskSourceStats, error) {
	const q = `
SELECT
  COUNT(*)                                                AS total,
  COUNT(*) FILTER (WHERE status = 'completed')            AS completed,
  COUNT(*) FILTER (WHERE status IN ('failed','expired'))  AS failed
FROM device_tasks
WHERE source = $1 AND source_id = $2`
	var s DeviceTaskSourceStats
	if err := r.pool.QueryRow(ctx, q, string(source), sourceID).
		Scan(&s.Total, &s.Completed, &s.Failed); err != nil {
		return DeviceTaskSourceStats{}, fmt.Errorf("aggregate device_tasks by source: %w", err)
	}
	s.Active = s.Total - s.Completed - s.Failed
	return s, nil
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
		"path_translation_source", // T-0168
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
	// T-0168: path_translation_source 列允许 NULL（非 MML 来源任务）。
	var pathTranslationSource sql.NullString

	err := row.Scan(
		&task.ID, &task.DeviceSN, &task.Method, &params, &task.Priority,
		&task.CommandKey, &task.CWMPID, &task.Status, &task.RetryCount, &task.MaxRetries,
		&task.CreatedAt, &task.SentAt, &task.CompletedAt, &task.ExpiresAt,
		&result, &task.ErrorCode, &task.ErrorMessage,
		&task.Source, &task.CreatorID, &task.Description,
		&sourceID, &task.CommandIndex, &task.DeviceIndex,
		&task.HasPathTranslationMiss, &task.PathTranslationMissCount,
		&pathTranslationSource,
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
	if pathTranslationSource.Valid {
		task.PathTranslationSource = pathTranslationSource.String
	}

	return &task, nil
}
