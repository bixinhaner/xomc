package trace

import (
	"context"
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

// taskAllowedSortColumns 防 SQL 注入白名单。
var taskAllowedSortColumns = map[string]bool{
	"created_at": true,
	"start_time": true,
	"expires_at": true,
	"device_sn":  true,
}

var taskColumns = []string{
	"id", "device_sn", "operator_code", "status",
	"start_time", "expires_at", "stopped_at", "purged_at",
	"created_by", "message_count", "created_at", "updated_at",
}

var messageColumns = []string{
	"captured_at", "id", "task_id", "device_sn", "direction",
	"rpc_method", "cwmp_id", "session_id", "http_status",
	"payload_size_bytes", "payload_inline", "payload_object_key",
}

var _ Repository = (*PgRepository)(nil)

// PgRepository PostgreSQL/TimescaleDB 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 构造函数。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// CreateTask 插入新任务（status 必须为 running，partial unique 防同 SN 并发）。
func (r *PgRepository) CreateTask(ctx context.Context, task *Task) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	if task.StartTime.IsZero() {
		task.StartTime = now
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	query, args, err := storage.Psql.Insert("trace_tasks").
		Columns("id", "device_sn", "operator_code", "status",
			"start_time", "expires_at", "created_by",
			"message_count", "created_at", "updated_at").
		Values(task.ID, task.DeviceSN, task.OperatorCode, task.Status,
			task.StartTime, task.ExpiresAt, task.CreatedBy,
			task.MessageCount, task.CreatedAt, task.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trace_task: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert trace_task: %w", err)
	}
	return nil
}

// GetTask 按 ID 查找；未找到返回 ErrNotFound。
func (r *PgRepository) GetTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("trace_tasks").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trace_task: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get trace_task: %w", err)
	}
	return t, nil
}

// GetRunningTaskBySN 查 SN 下当前唯一 running 任务，未找到返回 ErrNotFound。
func (r *PgRepository) GetRunningTaskBySN(ctx context.Context, sn string) (*Task, error) {
	query, args, err := storage.Psql.Select(taskColumns...).From("trace_tasks").
		Where(sq.And{
			sq.Eq{"device_sn": sn},
			sq.Eq{"status": TaskStatusRunning},
		}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get running task: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get running task: %w", err)
	}
	return t, nil
}

// ListTasks 分页查询任务。
func (r *PgRepository) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[Task], error) {
	base := storage.Psql.Select(taskColumns...).From("trace_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("trace_tasks")

	if filter.DeviceSN != "" {
		base = base.Where(sq.Eq{"device_sn": filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.Status != "" {
		base = base.Where(sq.Eq{"status": filter.Status})
		countBase = countBase.Where(sq.Eq{"status": filter.Status})
	}
	if filter.OperatorCode != "" {
		base = base.Where(sq.Eq{"operator_code": filter.OperatorCode})
		countBase = countBase.Where(sq.Eq{"operator_code": filter.OperatorCode})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count trace_tasks: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trace_tasks: %w", err)
	}

	sortBy := "created_at"
	if filter.SortBy != "" && taskAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	base = base.OrderBy(sortBy+" "+sortDir).
		Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trace_tasks: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trace_tasks: %w", err)
	}
	defer rows.Close()

	items := []Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trace_task: %w", err)
		}
		items = append(items, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter trace_tasks: %w", err)
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	return model.NewListResponse(items, total, page, filter.Limit()), nil
}

// UpdateTaskStatus 修改任务状态；status=stopped/purged 时填 stopped_at / purged_at。
func (r *PgRepository) UpdateTaskStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error {
	now := time.Now()
	updater := storage.Psql.Update("trace_tasks").
		Set("status", status).Set("updated_at", now)
	switch status {
	case TaskStatusStopped:
		updater = updater.Set("stopped_at", now)
	case TaskStatusPurged:
		updater = updater.Set("purged_at", now)
	}
	query, args, err := updater.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update trace_task status: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trace_task status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// IncrementMessageCount 报文落库后累加计数（不影响主流程，失败仅日志）。
func (r *PgRepository) IncrementMessageCount(ctx context.Context, id uuid.UUID, delta int) error {
	query := `UPDATE trace_tasks SET message_count = message_count + $1, updated_at = NOW() WHERE id = $2`
	if _, err := r.pool.Exec(ctx, query, delta, id); err != nil {
		return fmt.Errorf("increment trace_task message_count: %w", err)
	}
	return nil
}

// ListRunningSNs 返回 sn → task_id 映射，供 ACS 白名单加载。
func (r *PgRepository) ListRunningSNs(ctx context.Context) (map[string]uuid.UUID, error) {
	query, args, err := storage.Psql.Select("device_sn", "id").From("trace_tasks").
		Where(sq.Eq{"status": TaskStatusRunning}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list running SNs: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list running SNs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]uuid.UUID)
	for rows.Next() {
		var sn string
		var id uuid.UUID
		if err := rows.Scan(&sn, &id); err != nil {
			return nil, fmt.Errorf("scan running SN: %w", err)
		}
		result[sn] = id
	}
	return result, rows.Err()
}

// ListExpired 返回已过期但仍 running 的任务（M2 巡检 worker 使用）。
func (r *PgRepository) ListExpired(ctx context.Context, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 100
	}
	query, args, err := storage.Psql.Select(taskColumns...).From("trace_tasks").
		Where(sq.And{
			sq.Eq{"status": TaskStatusRunning},
			sq.Lt{"expires_at": time.Now()},
		}).OrderBy("expires_at ASC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expired: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list expired: %w", err)
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expired task: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// PurgeTaskMessages 物理删除任务下所有报文。
func (r *PgRepository) PurgeTaskMessages(ctx context.Context, taskID uuid.UUID) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM trace_messages WHERE task_id = $1`, taskID); err != nil {
		return fmt.Errorf("delete trace_messages by task: %w", err)
	}
	return nil
}

// InsertMessage 单条插入。
func (r *PgRepository) InsertMessage(ctx context.Context, msg *Message) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.CapturedAt.IsZero() {
		msg.CapturedAt = time.Now()
	}
	query, args, err := storage.Psql.Insert("trace_messages").
		Columns(messageColumns...).
		Values(msg.CapturedAt, msg.ID, msg.TaskID, msg.DeviceSN, msg.Direction,
			nullableString(msg.RPCMethod), nullableString(msg.CwmpID), nullableString(msg.SessionID),
			nullableInt(msg.HTTPStatus), msg.PayloadSizeBytes,
			nullableString(msg.PayloadInline), nullableString(msg.PayloadObjectKey)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trace_message: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert trace_message: %w", err)
	}
	return nil
}

// InsertMessages 批量插入；空切片直接返回。
func (r *PgRepository) InsertMessages(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}
	builder := storage.Psql.Insert("trace_messages").Columns(messageColumns...)
	for _, m := range msgs {
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		if m.CapturedAt.IsZero() {
			m.CapturedAt = time.Now()
		}
		builder = builder.Values(m.CapturedAt, m.ID, m.TaskID, m.DeviceSN, m.Direction,
			nullableString(m.RPCMethod), nullableString(m.CwmpID), nullableString(m.SessionID),
			nullableInt(m.HTTPStatus), m.PayloadSizeBytes,
			nullableString(m.PayloadInline), nullableString(m.PayloadObjectKey))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build batch insert trace_messages: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("batch insert trace_messages: %w", err)
	}
	return nil
}

// GetMessage 按 (task_id, msg_id) 查单条；hypertable 主键以时序列优先，
// 通过 task_id 索引快速定位 chunk 后再过滤 id。未找到返回 ErrNotFound。
func (r *PgRepository) GetMessage(ctx context.Context, taskID, msgID uuid.UUID) (*Message, error) {
	query, args, err := storage.Psql.Select(messageColumns...).From("trace_messages").
		Where(sq.And{sq.Eq{"task_id": taskID}, sq.Eq{"id": msgID}}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trace_message: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	m, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get trace_message: %w", err)
	}
	return m, nil
}

// ListMessages 分页查询报文，固定按 captured_at 排序（时序列）。
func (r *PgRepository) ListMessages(ctx context.Context, filter MessageFilter) (*model.ListResponse[Message], error) {
	base := storage.Psql.Select(messageColumns...).From("trace_messages")
	countBase := storage.Psql.Select("COUNT(*)").From("trace_messages")

	if filter.TaskID != uuid.Nil {
		base = base.Where(sq.Eq{"task_id": filter.TaskID})
		countBase = countBase.Where(sq.Eq{"task_id": filter.TaskID})
	}
	if filter.DeviceSN != "" {
		base = base.Where(sq.Eq{"device_sn": filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.Direction != "" {
		base = base.Where(sq.Eq{"direction": filter.Direction})
		countBase = countBase.Where(sq.Eq{"direction": filter.Direction})
	}
	if filter.RPCMethod != "" {
		base = base.Where(sq.Eq{"rpc_method": filter.RPCMethod})
		countBase = countBase.Where(sq.Eq{"rpc_method": filter.RPCMethod})
	}
	if filter.StartTime != nil {
		base = base.Where(sq.GtOrEq{"captured_at": *filter.StartTime})
		countBase = countBase.Where(sq.GtOrEq{"captured_at": *filter.StartTime})
	}
	if filter.EndTime != nil {
		base = base.Where(sq.LtOrEq{"captured_at": *filter.EndTime})
		countBase = countBase.Where(sq.LtOrEq{"captured_at": *filter.EndTime})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count trace_messages: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trace_messages: %w", err)
	}

	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	base = base.OrderBy("captured_at " + sortDir).
		Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trace_messages: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trace_messages: %w", err)
	}
	defer rows.Close()

	items := []Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trace_message: %w", err)
		}
		items = append(items, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter trace_messages: %w", err)
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	return model.NewListResponse(items, total, page, filter.Limit()), nil
}

// scanTask 从 row 扫描出 Task。pgx.Row / pgx.Rows 都有 Scan 方法，用同一签名。
func scanTask(row interface{ Scan(dest ...any) error }) (*Task, error) {
	var t Task
	var stoppedAt, purgedAt *time.Time
	if err := row.Scan(
		&t.ID, &t.DeviceSN, &t.OperatorCode, &t.Status,
		&t.StartTime, &t.ExpiresAt, &stoppedAt, &purgedAt,
		&t.CreatedBy, &t.MessageCount, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	t.StoppedAt = stoppedAt
	t.PurgedAt = purgedAt
	return &t, nil
}

func scanMessage(row interface{ Scan(dest ...any) error }) (*Message, error) {
	var m Message
	var rpcMethod, cwmpID, sessionID, payloadInline, payloadObjectKey *string
	var httpStatus *int
	if err := row.Scan(
		&m.CapturedAt, &m.ID, &m.TaskID, &m.DeviceSN, &m.Direction,
		&rpcMethod, &cwmpID, &sessionID, &httpStatus,
		&m.PayloadSizeBytes, &payloadInline, &payloadObjectKey,
	); err != nil {
		return nil, err
	}
	if rpcMethod != nil {
		m.RPCMethod = *rpcMethod
	}
	if cwmpID != nil {
		m.CwmpID = *cwmpID
	}
	if sessionID != nil {
		m.SessionID = *sessionID
	}
	if httpStatus != nil {
		m.HTTPStatus = *httpStatus
	}
	if payloadInline != nil {
		m.PayloadInline = *payloadInline
	}
	if payloadObjectKey != nil {
		m.PayloadObjectKey = *payloadObjectKey
	}
	return &m, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

// ---------- ExportJob ----------

var exportJobColumns = []string{
	"id", "task_id", "requested_by", "status",
	"object_key", "object_bucket", "message_count", "size_bytes",
	"error_message", "started_at", "completed_at",
	"created_at", "updated_at",
}

// CreateExportJob 插入新导出任务（status 默认 queued）。
func (r *PgRepository) CreateExportJob(ctx context.Context, job *ExportJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = ExportJobQueued
	}
	if job.RequestedBy == "" {
		job.RequestedBy = "system"
	}
	query, args, err := storage.Psql.Insert("trace_export_jobs").
		Columns("id", "task_id", "requested_by", "status",
			"message_count", "size_bytes", "created_at", "updated_at").
		Values(job.ID, job.TaskID, job.RequestedBy, job.Status,
			job.MessageCount, job.SizeBytes, job.CreatedAt, job.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert export job: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert export job: %w", err)
	}
	return nil
}

// GetExportJob 按 ID 查；未找到返回 ErrNotFound。
func (r *PgRepository) GetExportJob(ctx context.Context, id uuid.UUID) (*ExportJob, error) {
	query, args, err := storage.Psql.Select(exportJobColumns...).From("trace_export_jobs").
		Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get export job: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	job, err := scanExportJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get export job: %w", err)
	}
	return job, nil
}

// UpdateExportJob 全字段更新（status / object_key / size_bytes / error / 时间戳）。
func (r *PgRepository) UpdateExportJob(ctx context.Context, job *ExportJob) error {
	job.UpdatedAt = time.Now()
	updater := storage.Psql.Update("trace_export_jobs").
		Set("status", job.Status).
		Set("object_key", nullableString(job.ObjectKey)).
		Set("object_bucket", nullableString(job.ObjectBucket)).
		Set("message_count", job.MessageCount).
		Set("size_bytes", job.SizeBytes).
		Set("error_message", nullableString(job.ErrorMessage)).
		Set("started_at", job.StartedAt).
		Set("completed_at", job.CompletedAt).
		Set("updated_at", job.UpdatedAt).
		Where(sq.Eq{"id": job.ID})
	query, args, err := updater.ToSql()
	if err != nil {
		return fmt.Errorf("build update export job: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update export job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func scanExportJob(row interface{ Scan(dest ...any) error }) (*ExportJob, error) {
	var j ExportJob
	var objectKey, objectBucket, errorMsg *string
	var startedAt, completedAt *time.Time
	if err := row.Scan(
		&j.ID, &j.TaskID, &j.RequestedBy, &j.Status,
		&objectKey, &objectBucket, &j.MessageCount, &j.SizeBytes,
		&errorMsg, &startedAt, &completedAt,
		&j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if objectKey != nil {
		j.ObjectKey = *objectKey
	}
	if objectBucket != nil {
		j.ObjectBucket = *objectBucket
	}
	if errorMsg != nil {
		j.ErrorMessage = *errorMsg
	}
	j.StartedAt = startedAt
	j.CompletedAt = completedAt
	return &j, nil
}

// StorageStats 一次查询返回 inline / minio 字节总量。trace_messages 是分区表，
// SUM 走索引扫描；没数据时返回 (0, 0, nil)。
func (r *PgRepository) StorageStats(ctx context.Context) (inlineBytes, minioBytes int64, err error) {
	const q = `SELECT
		COALESCE(SUM(payload_size_bytes) FILTER (WHERE payload_object_key IS NULL OR payload_object_key = ''), 0)::bigint AS inline_bytes,
		COALESCE(SUM(payload_size_bytes) FILTER (WHERE payload_object_key IS NOT NULL AND payload_object_key <> ''), 0)::bigint AS minio_bytes
		FROM trace_messages`
	if err = r.pool.QueryRow(ctx, q).Scan(&inlineBytes, &minioBytes); err != nil {
		return 0, 0, fmt.Errorf("storage stats: %w", err)
	}
	return inlineBytes, minioBytes, nil
}
