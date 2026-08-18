package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// PgTaskRepository 实现 TaskRepository 接口
type PgTaskRepository struct {
	pool          *pgxpool.Pool
	syncLockSlots chan struct{}
}

// sentTaskExpiryGrace separates the absolute dispatch deadline from an RPC
// already written to a live CWMP session. The ACS idle timeout is 120 seconds;
// after that bound the task is genuinely stale and may be expired normally.
const sentTaskExpiryGrace = 2 * time.Minute

// NewPgTaskRepository 创建 PostgreSQL 任务仓库
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	lockSlots := 1
	if pool != nil {
		lockSlots = int(pool.Config().MaxConns) / 2
		if lockSlots < 1 {
			lockSlots = 1
		}
		if lockSlots > 8 {
			lockSlots = 8
		}
	}
	return &PgTaskRepository{pool: pool, syncLockSlots: make(chan struct{}, lockSlots)}
}

func (r *PgTaskRepository) acquireTaskAdvisoryLock(
	ctx context.Context,
	lockKey string,
	purpose string,
) (func(), error) {
	if r == nil || r.pool == nil {
		return func() {}, nil
	}
	if strings.TrimSpace(lockKey) == "" {
		return nil, fmt.Errorf("acquire %s lock: lock key is required", purpose)
	}
	if r.pool.Config().MaxConns < 2 {
		return nil, fmt.Errorf(
			"acquire %s lock: postgres pool requires at least 2 connections",
			purpose,
		)
	}
	select {
	case r.syncLockSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for %s lock slot: %w", purpose, ctx.Err())
	}
	releaseSlot := func() { <-r.syncLockSlots }
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		releaseSlot()
		return nil, fmt.Errorf("begin %s lock transaction: %w", purpose, err)
	}
	if _, err := tx.Exec(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		lockKey,
	); err != nil {
		_ = tx.Rollback(context.Background())
		releaseSlot()
		return nil, fmt.Errorf("acquire %s lock: %w", purpose, err)
	}
	return func() {
		defer releaseSlot()
		releaseCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()
		_ = tx.Rollback(releaseCtx)
	}, nil
}

// AcquireSyncGPVDeviceLock serializes sync-gpv creation for one device across
// all application instances. The returned release function ends the otherwise
// empty transaction, which releases the transaction-scoped advisory lock.
func (r *PgTaskRepository) AcquireSyncGPVDeviceLock(ctx context.Context, deviceSN string) (func(), error) {
	return r.acquireTaskAdvisoryLock(ctx, deviceSN, "sync-gpv device")
}

// AcquireCommandKeyLock serializes check-and-create for a deterministic task
// command key across all worker instances. The lock is held while the caller
// checks durable history and creates the Redis/PostgreSQL task, closing the
// at-least-once event delivery race without introducing a second task model.
func (r *PgTaskRepository) AcquireCommandKeyLock(
	ctx context.Context,
	commandKey string,
) (func(), error) {
	return r.acquireTaskAdvisoryLock(
		ctx,
		"device-task-command-key:"+commandKey,
		"task command key",
	)
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
			"command_key", "cwmp_id", "status", "retry_count", "max_retries", "retry_interval_seconds",
			"created_at", "sent_at", "completed_at", "expires_at", "next_attempt_at",
			"result", "error_code", "error_message",
			"source", "creator_id", "description",
			"source_id", "command_index", "device_index",
			"has_path_translation_miss", "path_translation_miss_count",
			"path_translation_source", // T-0168
			"admission_class",
		).
		Values(
			task.ID, task.DeviceSN, task.Method, task.Params, task.Priority,
			task.CommandKey, task.CWMPID, task.Status, task.RetryCount, task.MaxRetries, task.RetryIntervalSeconds,
			task.CreatedAt, task.SentAt, task.CompletedAt, task.ExpiresAt, task.NextAttemptAt,
			task.Result, task.ErrorCode, task.ErrorMessage,
			task.Source, task.CreatorID, task.Description,
			nilUUID(task.SourceID), task.CommandIndex, task.DeviceIndex,
			task.HasPathTranslationMiss, task.PathTranslationMissCount,
			nilString(task.PathTranslationSource), // T-0168: 空字符串 → NULL
			admissionClassOrDefault(task.AdmissionClass),
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
	builder := storage.Psql.Update("device_tasks").
		Set("method", task.Method).
		Set("params", task.Params).
		Set("priority", task.Priority).
		Set("command_key", task.CommandKey).
		Set("cwmp_id", task.CWMPID).
		Set("status", task.Status).
		Set("retry_count", task.RetryCount).
		Set("max_retries", task.MaxRetries).
		Set("retry_interval_seconds", task.RetryIntervalSeconds).
		Set("sent_at", task.SentAt).
		Set("completed_at", task.CompletedAt).
		Set("expires_at", task.ExpiresAt).
		Set("next_attempt_at", task.NextAttemptAt).
		Set("result", task.Result).
		Set("error_code", task.ErrorCode).
		Set("error_message", task.ErrorMessage).
		Set("admission_class", admissionClassOrDefault(task.AdmissionClass)).
		Where(sq.Eq{"id": task.ID})
	if task.Status == TaskStatusSent {
		// A fast CPE can complete before the sent-state PG sync returns. The
		// late sent write must not downgrade a terminal row already written by
		// MarkTaskCompleted/Failed.
		builder = builder.Where(sq.NotEq{"status": []TaskStatus{
			TaskStatusCompleted,
			TaskStatusFailed,
			TaskStatusExpired,
			TaskStatusCancelled,
		}})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

// TransitionIfStatus applies a complete task snapshot only when the durable
// row still has the state observed by the Redis transition winner. This is the
// PostgreSQL half of the first-writer-wins fence and prevents a late retry,
// cancellation, expiration, or sent update from reviving a terminal row.
func (r *PgTaskRepository) TransitionIfStatus(
	ctx context.Context,
	task *Task,
	from TaskStatus,
) (bool, error) {
	if task == nil {
		return false, fmt.Errorf("task is nil")
	}
	query, args, err := storage.Psql.Update("device_tasks").
		Set("method", task.Method).
		Set("params", task.Params).
		Set("priority", task.Priority).
		Set("command_key", task.CommandKey).
		Set("cwmp_id", task.CWMPID).
		Set("status", task.Status).
		Set("retry_count", task.RetryCount).
		Set("max_retries", task.MaxRetries).
		Set("retry_interval_seconds", task.RetryIntervalSeconds).
		Set("sent_at", task.SentAt).
		Set("completed_at", task.CompletedAt).
		Set("expires_at", task.ExpiresAt).
		Set("next_attempt_at", task.NextAttemptAt).
		Set("result", task.Result).
		Set("error_code", task.ErrorCode).
		Set("error_message", task.ErrorMessage).
		Where(sq.Eq{"id": task.ID, "status": from}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build conditional task transition: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("execute conditional task transition: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// MarkSentIfPending is the PostgreSQL execution fence used immediately before
// ACS sends an RPC. A cancelled/expired/terminal task can never be revived to
// sent, even when a stale copy is still present in Redis.
func (r *PgTaskRepository) MarkSentIfPending(ctx context.Context, taskID, cwmpID string, sentAt time.Time) (bool, error) {
	const query = `UPDATE device_tasks SET status=$2, cwmp_id=$3, sent_at=$4
WHERE id=$1 AND status='pending' AND (
  expires_at IS NULL OR expires_at > $4
) AND (
  COALESCE(source, '') <> 'param_sync' OR EXISTS (
SELECT 1 FROM parameter_sync_runs r
WHERE r.id=device_tasks.source_id
  AND r.status IN ('planning','enqueuing','waiting_device','executing','processing')
  )
)`
	tag, err := r.pool.Exec(ctx, query, taskID, TaskStatusSent, cwmpID, sentAt)
	if err != nil {
		return false, fmt.Errorf("execute task send fence: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ReleaseSentClaimIfUnwritten compensates only the pre-write failure window in
// which ACS knows the RPC was not placed on the HTTP response. Matching both
// status and CWMP ID prevents an old failure path from reviving a terminal task
// or releasing a newer send claim.
func (r *PgTaskRepository) ReleaseSentClaimIfUnwritten(ctx context.Context, taskID, cwmpID string) (bool, error) {
	const query = `UPDATE device_tasks SET status='pending', cwmp_id=NULL, sent_at=NULL
WHERE id=$1 AND status='sent' AND cwmp_id=$2`
	tag, err := r.pool.Exec(ctx, query, taskID, cwmpID)
	if err != nil {
		return false, fmt.Errorf("release unwritten task send claim: %w", err)
	}
	return tag.RowsAffected() == 1, nil
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

// TaskStatusRow 是 LookupStatusesByIDs 的轻量返回（只取 stale sync 反查需要的字段，
// 不扫描整行 task，避免 N 次 GetByID 全列扫描）。
type TaskStatusRow struct {
	ID           string
	Status       TaskStatus
	ErrorMessage string
}

// LookupStatusesByIDs 批量反查多个 task 的当前状态（#16 消除 notification stale sync 的 N+1）。
// 一条 WHERE id = ANY(...) 查询替代逐 ID 的 GetByID；结果按 id 去重映射，缺失的 id 不在返回 map 中。
// ids 为空时直接返回空 map，不打 DB。
func (r *PgTaskRepository) LookupStatusesByIDs(ctx context.Context, ids []string) (map[string]TaskStatusRow, error) {
	out := make(map[string]TaskStatusRow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	query, args, err := storage.Psql.Select("id", "status", "error_message").
		From("device_tasks").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build lookup statuses query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query task statuses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row TaskStatusRow
		if err := rows.Scan(&row.ID, &row.Status, &row.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan task status: %w", err)
		}
		out[row.ID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task statuses: %w", err)
	}

	return out, nil
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

// CountOpenByDevice 统计设备仍在执行窗口内的 active 任务数量。
//
// Redis 队列是运行时加速层，可能留下已过期或已终态任务 ID；设备同步状态这类用户可见
// 判断应以 PG 中 pending/sent 且未过期的任务为准，避免脏队列把页面长期卡在 syncing。
func (r *PgTaskRepository) CountOpenByDevice(ctx context.Context, deviceSN string, now time.Time) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("device_tasks").
		Where(sq.Eq{
			"device_sn": deviceSN,
			"status":    []TaskStatus{TaskStatusPending, TaskStatusSent},
		}).
		Where(sq.Or{
			sq.Eq{"expires_at": nil},
			sq.Gt{"expires_at": now},
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count open tasks query: %w", err)
	}

	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("query open task count: %w", err)
	}
	return count, nil
}

func (r *PgTaskRepository) HasIncompleteSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error) {
	prefix := fmt.Sprintf("sync-gpv-%s", deviceSN)
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("device_tasks t").
		Where(sq.Eq{"t.device_sn": deviceSN}).
		Where(sq.Eq{"t.method": "GetParameterValues"}).
		Where(sq.Like{"t.command_key": prefix + "%"}).
		Where(sq.Eq{"t.status": []TaskStatus{TaskStatusPending, TaskStatusSent}}).
		Where(sq.Expr("t.created_at > now() - interval '24 hours'")).
		Where(sq.NotEq{"t.expires_at": nil}).
		Where(sq.Expr("t.expires_at > now()")).
		Where(`t.source_id = (
			SELECT latest.source_id
			FROM device_tasks latest
			WHERE latest.device_sn = t.device_sn
			  AND latest.method = 'GetParameterValues'
			  AND latest.command_key LIKE ?
			  AND latest.source_id IS NOT NULL
			ORDER BY latest.created_at DESC
			LIMIT 1
		)`, prefix+"%").
		Where(`NOT EXISTS (
			SELECT 1 FROM device_tasks failed
			WHERE failed.device_sn = t.device_sn
			  AND failed.source_id = t.source_id
			  AND failed.method = 'GetParameterValues'
			  AND failed.command_key LIKE ?
			  AND failed.status IN ('failed', 'expired', 'cancelled')
		)`, prefix+"%").
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

// HasOpenSyncGPVTasksByDevice reports whether any recent sync-gpv task is still pending/sent.
// This is stricter than HasIncompleteSyncGPVTasksByDevice: it intentionally does not ignore a
// source_id just because another task in that source has failed, so callers can use it as a
// backend re-entry guard before creating a new manual sync source.
func (r *PgTaskRepository) HasOpenSyncGPVTasksByDevice(ctx context.Context, deviceSN string) (bool, error) {
	prefix := fmt.Sprintf("sync-gpv-%s", deviceSN)
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("device_tasks t").
		Where(sq.Eq{"t.device_sn": deviceSN}).
		Where(sq.Eq{"t.method": "GetParameterValues"}).
		Where(sq.Like{"t.command_key": prefix + "%"}).
		Where(sq.Eq{"t.status": []TaskStatus{TaskStatusPending, TaskStatusSent}}).
		Where(sq.Expr("t.created_at > now() - interval '24 hours'")).
		Where(sq.NotEq{"t.expires_at": nil}).
		Where(sq.Expr("t.expires_at > now()")).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build open sync-gpv guard query: %w", err)
	}
	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("query open sync-gpv guard: %w", err)
	}
	return count > 0, nil
}

// CountOpenSyncGPVByDevice 统计近 24h 内仍未完成的参数同步 GPV task。
//
// 设备任务队列里还会混有 PM 初始化、MML、无 command_key 的临时 GPV，以及历史遗留 sent
// 行；参数树的 sync-status 只应被当前/近期参数同步本身影响。
func (r *PgTaskRepository) CountOpenSyncGPVByDevice(ctx context.Context, deviceSN string) (int64, error) {
	const query = `
WITH latest AS (
  SELECT source_id
  FROM device_tasks
  WHERE device_sn=$1 AND method='GetParameterValues'
    AND (command_key LIKE 'sync-gpv-%' OR source='param_sync')
    AND source_id IS NOT NULL
  ORDER BY created_at DESC
  LIMIT 1
)
SELECT COUNT(*)
FROM device_tasks t
JOIN latest l ON l.source_id=t.source_id
WHERE t.device_sn=$1 AND t.method='GetParameterValues'
  AND (t.command_key LIKE 'sync-gpv-%' OR t.source='param_sync')
  AND t.status IN ('pending','sent')
  AND t.created_at > now() - interval '24 hours'
  AND t.expires_at IS NOT NULL AND t.expires_at > now()
  AND (t.source<>'param_sync' OR EXISTS (
    SELECT 1 FROM parameter_sync_runs r
    WHERE r.id=t.source_id
      AND r.status IN ('planning','enqueuing','waiting_device','executing','processing')
  ))
  AND (t.source='param_sync' OR NOT EXISTS (
    SELECT 1 FROM device_tasks failed
    WHERE failed.device_sn=t.device_sn AND failed.source_id=t.source_id
      AND failed.method='GetParameterValues' AND failed.command_key LIKE 'sync-gpv-%'
      AND failed.status IN ('failed','expired','cancelled')
  ))`
	var count int64
	if err := r.pool.QueryRow(ctx, query, deviceSN).Scan(&count); err != nil {
		return 0, fmt.Errorf("query open sync-gpv count: %w", err)
	}
	return count, nil
}

func (r *PgTaskRepository) LatestSyncGPVSummaryByDevice(ctx context.Context, deviceSN string) (*SyncGPVSummary, error) {
	const query = `
WITH latest AS (
  SELECT
    source_id,
    command_key AS root_key,
    created_at
  FROM device_tasks
  WHERE device_sn = $1
    AND method = 'GetParameterValues'
    AND (command_key LIKE 'sync-gpv-%' OR source = 'param_sync')
    AND command_key !~ '(-r)+$'
    AND source_id IS NOT NULL
    AND completed_at IS NOT NULL
  ORDER BY created_at DESC, completed_at DESC
  LIMIT 1
),
roots AS (
  SELECT
    t.source_id,
    t.command_key AS root_key,
    t.created_at
  FROM device_tasks t
  JOIN latest l ON t.source_id = l.source_id
  WHERE t.device_sn = $1
    AND t.method = 'GetParameterValues'
    AND (t.command_key LIKE 'sync-gpv-%' OR t.source = 'param_sync')
    AND t.command_key !~ '(-r)+$'
),
anchor AS (
  SELECT
    l.source_id,
    COALESCE((
      SELECT r.created_at
      FROM roots r
      WHERE r.root_key = l.root_key
        AND r.created_at <= l.created_at
      ORDER BY r.created_at DESC
      LIMIT 1
    ), l.created_at) AS created_at
  FROM latest l
),
boundary AS (
  SELECT COALESCE(MAX(r.created_at), '-infinity'::timestamptz) AS prev_created_at
  FROM roots r
  CROSS JOIN anchor a
  WHERE r.created_at < a.created_at - interval '30 seconds'
),
run_roots AS (
  SELECT r.source_id, r.root_key, r.created_at
  FROM roots r
  CROSS JOIN anchor a
  CROSS JOIN boundary b
  WHERE r.created_at > b.prev_created_at
    AND r.created_at <= a.created_at + interval '30 seconds'
),
run_bounds AS (
  SELECT MIN(created_at) AS first_root_created_at
  FROM run_roots
),
run_tasks AS (
  SELECT t.*
  FROM device_tasks t
  JOIN run_roots r
    ON t.source_id = r.source_id
   AND regexp_replace(t.command_key, '(-r)+$', '') = r.root_key
  CROSS JOIN run_bounds rb
  WHERE t.device_sn = $1
    AND t.method = 'GetParameterValues'
    AND t.completed_at IS NOT NULL
    AND t.created_at >= rb.first_root_created_at - interval '30 seconds'
),
requested_paths AS (
  SELECT COUNT(DISTINCT requested.path)::int AS requested_path_count
  FROM device_tasks root
  JOIN run_roots r
    ON root.source_id = r.source_id
   AND root.command_key = r.root_key
  CROSS JOIN LATERAL jsonb_array_elements_text(
    COALESCE(root.params::jsonb->'names', root.params::jsonb->'paths', '[]'::jsonb)
  ) AS requested(path)
  WHERE root.device_sn = $1
    AND root.method = 'GetParameterValues'
),
failed_path_rows AS (
  SELECT jsonb_build_object(
    'path', skipped.path,
    'fault_code', NULLIF(t.result::jsonb->>'fault_code', '')::int,
    'fault_text', '',
    'command_key', t.command_key,
    'status', t.status
  ) AS item
  FROM run_tasks t
  CROSS JOIN LATERAL jsonb_array_elements_text(
    COALESCE(t.result::jsonb->'bad_paths', '[]'::jsonb)
  ) AS skipped(path)
  WHERE t.status = 'completed'
    AND t.result IS NOT NULL
    AND COALESCE((t.result::jsonb->>'recovered')::boolean, false)

  UNION ALL

  SELECT jsonb_build_object(
    'path', t.result::jsonb->>'bad_path',
    'fault_code', NULLIF(t.result::jsonb->>'fault_code', '')::int,
    'fault_text', '',
    'command_key', t.command_key,
    'status', t.status
  ) AS item
  FROM run_tasks t
  WHERE t.status = 'completed'
    AND t.result IS NOT NULL
    AND COALESCE((t.result::jsonb->>'recovered')::boolean, false)
    AND COALESCE(t.result::jsonb->>'bad_path', '') <> ''
    AND jsonb_array_length(COALESCE(t.result::jsonb->'bad_paths', '[]'::jsonb)) = 0

  UNION ALL

  SELECT jsonb_build_object(
    'path', fault->>'parameter_name',
    'fault_code', NULLIF(fault->>'fault_code', '')::int,
    'fault_text', COALESCE(fault->>'fault_string', ''),
    'command_key', t.command_key,
    'status', t.status
  ) AS item
  FROM run_tasks t
  CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result::jsonb->'param_faults', '[]'::jsonb)) AS fault
  WHERE t.status IN ('failed', 'expired')
    AND t.result IS NOT NULL

  UNION ALL

  SELECT jsonb_build_object(
	'path', requested.path,
	'fault_code', t.error_code,
	'fault_text', COALESCE(t.error_message, ''),
	'command_key', t.command_key,
	'status', t.status
  ) AS item
  FROM run_tasks t
  CROSS JOIN LATERAL jsonb_array_elements_text(
	COALESCE(t.params::jsonb->'names', t.params::jsonb->'paths', '[]'::jsonb)
  ) AS requested(path)
  WHERE t.status IN ('failed', 'expired')
	AND jsonb_array_length(COALESCE(t.result::jsonb->'param_faults', '[]'::jsonb)) = 0

  UNION ALL

  SELECT jsonb_build_object(
	'path', '',
	'fault_code', t.error_code,
	'fault_text', COALESCE(t.error_message, ''),
	'command_key', t.command_key,
	'status', t.status
  ) AS item
  FROM run_tasks t
  WHERE t.status IN ('failed', 'expired')
	AND jsonb_array_length(COALESCE(t.result::jsonb->'param_faults', '[]'::jsonb)) = 0
	AND jsonb_array_length(
	  COALESCE(t.params::jsonb->'names', t.params::jsonb->'paths', '[]'::jsonb)
	) = 0
),
failed_paths AS (
  SELECT
    COUNT(*)::int AS failed_path_count,
    COALESCE(jsonb_agg(item ORDER BY item->>'command_key', item->>'path'), '[]'::jsonb) AS failed_paths
  FROM failed_path_rows
),
successful_path_rows AS (
  SELECT DISTINCT successful.path
  FROM run_tasks t
  CROSS JOIN LATERAL jsonb_array_elements_text(
    COALESCE(t.params::jsonb->'names', t.params::jsonb->'paths', '[]'::jsonb)
  ) AS successful(path)
  WHERE t.status = 'completed'
    AND NOT COALESCE((t.result::jsonb->>'recovered')::boolean, false)
),
successful_paths AS (
  SELECT COUNT(*)::int AS successful_path_count
  FROM successful_path_rows
)
SELECT
  rt.source_id::text,
  COUNT(*)::int,
  COUNT(*) FILTER (WHERE rt.status = 'completed')::int,
  COUNT(*) FILTER (WHERE rt.status IN ('failed', 'expired', 'cancelled'))::int,
  COALESCE(rp.requested_path_count, 0)::int,
  COALESCE(sp.successful_path_count, 0)::int,
  COALESCE(fp.failed_path_count, 0)::int,
  COALESCE(fp.failed_paths, '[]'::jsonb)::text,
  MIN(rt.created_at),
  MAX(rt.completed_at),
  EXTRACT(EPOCH FROM (MAX(rt.completed_at) - MIN(rt.created_at)))::float8
FROM run_tasks rt
CROSS JOIN failed_paths fp
CROSS JOIN successful_paths sp
CROSS JOIN requested_paths rp
GROUP BY rt.source_id, fp.failed_path_count, fp.failed_paths, sp.successful_path_count, rp.requested_path_count`

	var summary SyncGPVSummary
	var completed sql.NullTime
	var failedPathsJSON string
	if err := r.pool.QueryRow(ctx, query, deviceSN).Scan(
		&summary.SourceID,
		&summary.TaskCount,
		&summary.SuccessfulCommands,
		&summary.FailedCommands,
		&summary.RequestedPathCount,
		&summary.SuccessfulPathCount,
		&summary.FailedPathCount,
		&failedPathsJSON,
		&summary.FirstCreatedAt,
		&completed,
		&summary.WallClockSeconds,
	); err != nil {
		if err == pgx.ErrNoRows || err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest sync-gpv summary: %w", err)
	}
	if completed.Valid {
		summary.LastCompletedAt = &completed.Time
	}
	if failedPathsJSON != "" {
		if err := json.Unmarshal([]byte(failedPathsJSON), &summary.FailedPaths); err != nil {
			return nil, fmt.Errorf("decode latest sync-gpv failure paths: %w", err)
		}
	}
	return &summary, nil
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

// LatestOpenByDeviceMethodDescription 精确查询某类未完成任务，避免调用方先加载设备
// 的全部同方法任务再在内存中过滤。
func (r *PgTaskRepository) LatestOpenByDeviceMethodDescription(
	ctx context.Context,
	deviceSN, method, description string,
) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{
			"device_sn":   deviceSN,
			"method":      method,
			"description": description,
			"status":      []TaskStatus{TaskStatusPending, TaskStatusSent},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest open task query: %w", err)
	}
	taskItem, err := r.scanTaskRow(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest open task: %w", err)
	}
	return taskItem, nil
}

// LatestCompletedByDeviceCommandKey returns the newest successful execution of
// one idempotent system command.
func (r *PgTaskRepository) LatestCompletedByDeviceCommandKey(
	ctx context.Context,
	deviceSN, commandKey string,
) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{
			"device_sn":   deviceSN,
			"command_key": commandKey,
			"status":      TaskStatusCompleted,
		}).
		OrderBy("completed_at DESC", "created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build latest completed task query: %w", err)
	}
	taskItem, err := r.scanTaskRow(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest completed task: %w", err)
	}
	return taskItem, nil
}

func (r *PgTaskRepository) GetByCommandKey(
	ctx context.Context,
	commandKey string,
) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"command_key": commandKey}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build geofence command task query: %w", err)
	}
	item, err := r.scanTaskRow(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query geofence command task: %w", err)
	}
	return item, nil
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
// 条件：
//   - pending：expires_at < now()，绝对截止后不可再下发；
//   - sent：expires_at < now() 且 sent_at 已超过有界在途保护窗口。
//
// 这样 TTL 仍然限制排队/重试总时长，但不会把截止前刚写入 CWMP 会话、正在等待
// 设备响应的 RPC 抢先标成 expired。
// 排序：expires_at ASC（最早过期的先处理）。limit > 0 时限制单批数量防 worker 长事务。
//
// 调用方（worker/task_sweeper）拿到后调 task.MarkExpired + Update + publish task.expired。
func (r *PgTaskRepository) ListExpiredCandidates(ctx context.Context, now time.Time, limit int) ([]*Task, error) {
	sentCutoff := now.Add(-sentTaskExpiryGrace)
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.NotEq{"expires_at": nil}).
		Where(sq.Lt{"expires_at": now}).
		Where(sq.Or{
			sq.Eq{"status": TaskStatusPending},
			sq.And{
				sq.Eq{"status": TaskStatusSent},
				sq.Or{
					sq.Eq{"sent_at": nil},
					sq.LtOrEq{"sent_at": sentCutoff},
				},
			},
		}).
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

// ListSentByDeviceBefore 查找指定设备已下发但尚未结束的任务。
//
// PopTask 会把任务从 Redis 队列 ZSET 移除，再把状态标成 sent；如果 CWMP 会话中断，
// 这些任务不会留在 Redis 队列里。因此 CPE 新 Inform 时的恢复必须以 PG 的 sent 记录为准。
func (r *PgTaskRepository) ListSentByDeviceBefore(ctx context.Context, deviceSN string, sentBefore time.Time, limit int) ([]*Task, error) {
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"device_sn": deviceSN, "status": TaskStatusSent}).
		Where(sq.NotEq{"sent_at": nil}).
		Where(sq.LtOrEq{"sent_at": sentBefore}).
		OrderBy("sent_at ASC")
	if limit > 0 {
		q = q.Limit(uint64(limit))
	}
	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list sent by device query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sent tasks by device: %w", err)
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
		return nil, fmt.Errorf("iterate sent tasks by device: %w", err)
	}
	return tasks, nil
}

// ListActiveTasks 列出所有仍处于活跃态(pending/sent)且创建时间早于 olderThan 的任务，
// 用于 Reconciler 检测 PG 与 Redis 的状态分叉（#13）。
//
// olderThan 是一个"宽限阈值"：只挑选创建已足够久的任务，避开正在双写途中的在飞任务
// （CreateTask / MarkTaskCompleted 等的 PG sync 可能尚未落地），从而不误判正常时序差为分叉。
// limit > 0 时限制单批数量防 worker 长查询；剩余项下一轮处理。
func (r *PgTaskRepository) ListActiveTasks(ctx context.Context, olderThan time.Time, limit int) ([]*Task, error) {
	return r.ListActiveTasksAfter(ctx, olderThan, nil, limit)
}

// ActiveTaskCursor 是活跃任务对账的稳定键集游标。created_at 可能相同，必须以 id
// 作为第二排序键，避免跨轮重复或遗漏。
type ActiveTaskCursor struct {
	CreatedAt time.Time
	ID        string
}

func buildListActiveTasksSQL(
	olderThan time.Time,
	after *ActiveTaskCursor,
	limit int,
) (string, []any, error) {
	q := storage.Psql.Select(taskColumns()...).
		From("device_tasks").
		Where(sq.Eq{"status": []TaskStatus{TaskStatusPending, TaskStatusSent}}).
		Where(sq.Lt{"created_at": olderThan}).
		OrderBy("created_at ASC", "id ASC")
	if after != nil {
		q = q.Where(sq.Expr(
			"(created_at, id) > (?, ?)",
			after.CreatedAt,
			after.ID,
		))
	}
	if limit > 0 {
		q = q.Limit(uint64(limit))
	}
	return q.ToSql()
}

// ListActiveTasksAfter 返回游标之后的一个有界活跃任务批次。
func (r *PgTaskRepository) ListActiveTasksAfter(
	ctx context.Context,
	olderThan time.Time,
	after *ActiveTaskCursor,
	limit int,
) ([]*Task, error) {
	query, args, err := buildListActiveTasksSQL(olderThan, after, limit)
	if err != nil {
		return nil, fmt.Errorf("build list active query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active tasks: %w", err)
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
		"command_key", "cwmp_id", "status", "retry_count", "max_retries", "retry_interval_seconds",
		"created_at", "sent_at", "completed_at", "expires_at", "next_attempt_at",
		"result", "error_code", "error_message",
		"source", "creator_id", "description",
		"source_id", "command_index", "device_index",
		"has_path_translation_miss", "path_translation_miss_count",
		"path_translation_source", // T-0168
		"admission_class",
	}

	insertBuilder := storage.Psql.Insert("device_tasks").Columns(columns...)

	for _, task := range tasks {
		insertBuilder = insertBuilder.Values(
			task.ID, task.DeviceSN, task.Method, task.Params, task.Priority,
			task.CommandKey, task.CWMPID, task.Status, task.RetryCount, task.MaxRetries, task.RetryIntervalSeconds,
			task.CreatedAt, task.SentAt, task.CompletedAt, task.ExpiresAt, task.NextAttemptAt,
			task.Result, task.ErrorCode, task.ErrorMessage,
			task.Source, task.CreatorID, task.Description,
			task.SourceID, task.CommandIndex, task.DeviceIndex,
			task.HasPathTranslationMiss, task.PathTranslationMissCount,
			nilString(task.PathTranslationSource), // T-0168
			admissionClassOrDefault(task.AdmissionClass),
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

func admissionClassOrDefault(class AdmissionClass) AdmissionClass {
	if class == "" {
		return AdmissionClassNormal
	}
	return class
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
	Method       string
	Params       json.RawMessage
	CommandKey   string
	CWMPID       string
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
SELECT id, device_sn, method, params, COALESCE(command_key, ''), COALESCE(cwmp_id, ''), status,
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
		var params, raw []byte
		if err := rows.Scan(
			&row.ID, &row.DeviceSN, &row.Method, &params, &row.CommandKey, &row.CWMPID, &row.Status,
			&row.ErrorCode, &row.ErrorMessage,
			&raw, &row.SentAt, &row.CompletedAt, &row.CreatedAt,
			&row.CommandIndex, &row.DeviceIndex,
		); err != nil {
			return nil, 0, fmt.Errorf("scan device task row: %w", err)
		}
		if len(params) > 0 {
			row.Params = json.RawMessage(params)
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
		"COALESCE(command_key, '')", "COALESCE(cwmp_id, '')", "status", "retry_count", "max_retries", "retry_interval_seconds",
		"created_at", "sent_at", "completed_at", "expires_at", "next_attempt_at",
		"result", "COALESCE(error_code, 0)", "COALESCE(error_message, '')",
		"COALESCE(source, '')", "COALESCE(creator_id, '')", "COALESCE(description, '')",
		"source_id", "command_index", "device_index",
		"has_path_translation_miss", "path_translation_miss_count",
		"path_translation_source", // T-0168
		"admission_class",
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
		&task.CommandKey, &task.CWMPID, &task.Status, &task.RetryCount, &task.MaxRetries, &task.RetryIntervalSeconds,
		&task.CreatedAt, &task.SentAt, &task.CompletedAt, &task.ExpiresAt, &task.NextAttemptAt,
		&result, &task.ErrorCode, &task.ErrorMessage,
		&task.Source, &task.CreatorID, &task.Description,
		&sourceID, &task.CommandIndex, &task.DeviceIndex,
		&task.HasPathTranslationMiss, &task.PathTranslationMissCount,
		&pathTranslationSource, &task.AdmissionClass,
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
	if task.AdmissionClass == "" {
		task.AdmissionClass = AdmissionClassNormal
	}

	return &task, nil
}
