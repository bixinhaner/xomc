package asyncjob

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

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Repository 是 async_jobs 表的持久化接口（便于单测用 stub 实现）。
type Repository interface {
	// Insert 写一行 status='pending' 的新任务，返回 job_id。
	Insert(ctx context.Context, req InsertRequest) (uuid.UUID, error)

	// GetByID 按 id 取一行（status 不限）。
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)

	// LockNextPending 用 SELECT ... FOR UPDATE SKIP LOCKED 抢一个可用任务，
	// 同时把 status pending→running、heartbeat_at=NOW()、lock_owner 标本进程。
	// 返回 ErrNoPendingJob 表示当前无可用任务。
	LockNextPending(ctx context.Context, jobType, lockOwner string) (*Job, error)

	// UpdateHeartbeat 在 running 期间每 30s 调用，刷新 heartbeat_at 到 NOW()。
	// 任务已不在 running 状态时返回 ErrJobNotRunning。
	UpdateHeartbeat(ctx context.Context, id uuid.UUID) error

	// MarkSucceeded 任务成功收尾：status=succeeded, finished_at=NOW(), result=...
	MarkSucceeded(ctx context.Context, id uuid.UUID, result json.RawMessage) error

	// MarkFailed 任务失败收尾：status=failed, finished_at=NOW(), error_message=...
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error

	// ListZombies 找 status=running 且 heartbeat_at 过 threshold 的任务。
	ListZombies(ctx context.Context, threshold time.Duration) ([]Job, error)

	// ResetZombie 把僵尸任务重置回 pending（attempt+1），便于其他 worker 抢；
	// 若 attempt >= max_attempts 则直接置 failed 返回 ErrAttemptsExhausted。
	ResetZombie(ctx context.Context, id uuid.UUID) error
}

// PgRepository 是 Repository 的 pgxpool 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 创建 Repository 的 PG 实现。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

func (r *PgRepository) Insert(ctx context.Context, req InsertRequest) (uuid.UUID, error) {
	q, args, err := buildInsertSQL(req)
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("insert async_jobs: %w", err)
	}
	return id, nil
}

// RequeueSucceededBucket makes a previously completed natural bucket runnable
// again. It is intentionally separate from Insert so normal cron/catch-up
// idempotency never replays successful work; late PM data is the explicit caller.
func (r *PgRepository) RequeueSucceededBucket(
	ctx context.Context,
	jobType string,
	start, end time.Time,
	payload json.RawMessage,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		UPDATE async_jobs
		   SET status='pending', scheduled_at=now(), payload=$4,
		       started_at=NULL, finished_at=NULL, heartbeat_at=NULL,
		       lock_owner=NULL, attempt=1, result=NULL, error_message=NULL
		 WHERE job_type=$1 AND bucket_start=$2 AND bucket_end=$3
		   AND status='succeeded'
		RETURNING id`, jobType, start, end, payload).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("requeue succeeded async bucket: %w", err)
	}
	bucketStart, bucketEnd := start, end
	return r.Insert(ctx, InsertRequest{
		JobType: jobType, Payload: payload,
		BucketStart: &bucketStart, BucketEnd: &bucketEnd,
	})
}

func buildInsertSQL(req InsertRequest) (string, []any, error) {
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	q, args, err := storage.Psql.Insert("async_jobs").
		Columns("job_type", "status", "schedule_expr", "scheduled_at", "bucket_start", "bucket_end", "payload", "max_attempts").
		Values(req.JobType, string(StatusPending), nullableString(req.ScheduleExpr), req.ScheduledAt, req.BucketStart, req.BucketEnd, req.Payload, maxAttempts).
		Suffix(`
ON CONFLICT (job_type, bucket_start, bucket_end)
WHERE bucket_start IS NOT NULL AND bucket_end IS NOT NULL
DO UPDATE SET
    status = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN 'pending'
        ELSE async_jobs.status
    END,
    scheduled_at = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN EXCLUDED.scheduled_at
        ELSE async_jobs.scheduled_at
    END,
    payload = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN EXCLUDED.payload
        ELSE async_jobs.payload
    END,
    max_attempts = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN EXCLUDED.max_attempts
        ELSE async_jobs.max_attempts
    END,
    started_at = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.started_at
    END,
    finished_at = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.finished_at
    END,
    heartbeat_at = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.heartbeat_at
    END,
    lock_owner = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.lock_owner
    END,
    attempt = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN 1
        ELSE async_jobs.attempt
    END,
    result = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.result
    END,
    error_message = CASE
        WHEN async_jobs.status IN ('failed', 'canceled', 'zombie') THEN NULL
        ELSE async_jobs.error_message
    END`).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build insert async_jobs: %w", err)
	}
	return q, args, nil
}

func (r *PgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	q, args, err := storage.Psql.Select(jobCols...).
		From("async_jobs").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get async_jobs: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	return scanJob(row)
}

// LockNextPending 用 CTE + FOR UPDATE SKIP LOCKED 在单条 SQL 内原子完成
// "找 pending → 改 running"。比"先 SELECT，再 UPDATE"少一次 round-trip 且避免竞态。
func (r *PgRepository) LockNextPending(ctx context.Context, jobType, lockOwner string) (*Job, error) {
	const q = `
WITH next AS (
    SELECT id
    FROM async_jobs
    WHERE job_type = $1 AND status = 'pending' AND scheduled_at <= NOW()
    ORDER BY scheduled_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE async_jobs aj
SET status = 'running',
    started_at = NOW(),
    heartbeat_at = NOW(),
    lock_owner = $2
FROM next
WHERE aj.id = next.id
RETURNING aj.id, aj.job_type, aj.status, aj.schedule_expr, aj.scheduled_at, aj.started_at, aj.finished_at,
          aj.bucket_start, aj.bucket_end, aj.heartbeat_at, aj.lock_owner, aj.attempt, aj.max_attempts, aj.payload, aj.result, aj.error_message,
          aj.created_at, aj.updated_at
`
	row := r.pool.QueryRow(ctx, q, jobType, lockOwner)
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoPendingJob
		}
		return nil, err
	}
	return job, nil
}

func (r *PgRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE async_jobs SET heartbeat_at = NOW() WHERE id = $1 AND status = 'running'`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotRunning
	}
	return nil
}

func (r *PgRepository) MarkSucceeded(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	const q = `UPDATE async_jobs SET status = 'succeeded', finished_at = NOW(), result = $2 WHERE id = $1 AND status = 'running'`
	tag, err := r.pool.Exec(ctx, q, id, result)
	if err != nil {
		return fmt.Errorf("mark succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotRunning
	}
	return nil
}

func (r *PgRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `
UPDATE async_jobs SET
    status = CASE WHEN attempt + 1 > max_attempts THEN 'failed' ELSE 'pending' END,
    attempt = CASE WHEN attempt + 1 > max_attempts THEN attempt ELSE attempt + 1 END,
    scheduled_at = CASE WHEN attempt + 1 > max_attempts THEN scheduled_at ELSE NOW() END,
    started_at = NULL,
    heartbeat_at = NULL,
    lock_owner = NULL,
    finished_at = CASE WHEN attempt + 1 > max_attempts THEN NOW() ELSE NULL END,
    result = NULL,
    error_message = $2
WHERE id = $1 AND status = 'running'`
	tag, err := r.pool.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotRunning
	}
	return nil
}

func (r *PgRepository) ListZombies(ctx context.Context, threshold time.Duration) ([]Job, error) {
	q := fmt.Sprintf(`
SELECT %s FROM async_jobs
WHERE status = 'running' AND heartbeat_at < NOW() - $1::interval
ORDER BY heartbeat_at
LIMIT 100`, joinJobCols())
	rows, err := r.pool.Query(ctx, q, threshold.String())
	if err != nil {
		return nil, fmt.Errorf("list zombies: %w", err)
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *j)
	}
	return jobs, rows.Err()
}

// CountByJobTypeAndStatus 扫 async_jobs 表统计 (job_type, status) 维度的行数；
// 仅返活动状态（pending / running），其余状态视为完结不再产生队列压力。
// G8-Gap-4：QueueDepthSampler 周期调用，更新 QueueDepth gauge。
func (r *PgRepository) CountByJobTypeAndStatus(ctx context.Context) (map[string]map[string]int, error) {
	const q = `
SELECT job_type, status, COUNT(*)
FROM async_jobs
WHERE status IN ('pending', 'running')
GROUP BY job_type, status`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("count by job_type and status: %w", err)
	}
	defer rows.Close()
	out := make(map[string]map[string]int)
	for rows.Next() {
		var jobType, status string
		var n int
		if err := rows.Scan(&jobType, &status, &n); err != nil {
			return nil, err
		}
		if out[jobType] == nil {
			out[jobType] = make(map[string]int)
		}
		out[jobType][status] = n
	}
	return out, rows.Err()
}

// OldestPendingAgeSeconds returns queue age independently of queue depth so a
// steady-size but permanently stalled PM queue remains observable.
func (r *PgRepository) OldestPendingAgeSeconds(ctx context.Context) (map[string]float64, error) {
	const q = `
SELECT job_type, EXTRACT(EPOCH FROM (now()-MIN(scheduled_at)))::float8
FROM async_jobs
WHERE status='pending' AND scheduled_at <= now()
GROUP BY job_type`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("oldest pending age: %w", err)
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var jobType string
		var seconds float64
		if err := rows.Scan(&jobType, &seconds); err != nil {
			return nil, err
		}
		out[jobType] = seconds
	}
	return out, rows.Err()
}

// ResetZombie 单 SQL 原子做"attempt 检查 + 重置"，避免读改写竞态。
// 若 attempt < max_attempts → status pending, attempt+1, heartbeat_at=NULL；
// 若 attempt >= max_attempts → status failed, finished_at=NOW(), error_message='attempts exhausted (zombie)'；
// 用 RETURNING 区分两种结局。
func (r *PgRepository) ResetZombie(ctx context.Context, id uuid.UUID) error {
	const q = `
UPDATE async_jobs SET
    status = CASE WHEN attempt + 1 > max_attempts THEN 'failed' ELSE 'pending' END,
    attempt = attempt + 1,
    heartbeat_at = NULL,
    lock_owner = NULL,
    finished_at = CASE WHEN attempt + 1 > max_attempts THEN NOW() ELSE NULL END,
    error_message = CASE WHEN attempt + 1 > max_attempts THEN 'attempts exhausted (zombie)' ELSE NULL END
WHERE id = $1 AND status = 'running'
RETURNING status
`
	var newStatus string
	err := r.pool.QueryRow(ctx, q, id).Scan(&newStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrJobNotRunning
		}
		return fmt.Errorf("reset zombie: %w", err)
	}
	if newStatus == string(StatusFailed) {
		return ErrAttemptsExhausted
	}
	return nil
}

// jobCols / joinJobCols / scanJob — 内部 helper

var jobCols = []string{
	"id", "job_type", "status", "schedule_expr", "scheduled_at", "started_at", "finished_at",
	"bucket_start", "bucket_end", "heartbeat_at", "lock_owner", "attempt", "max_attempts", "payload", "result", "error_message",
	"created_at", "updated_at",
}

func joinJobCols() string {
	out := ""
	for i, c := range jobCols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (*Job, error) {
	var j Job
	var scheduleExpr, lockOwner, errMsg *string
	var payload, result []byte
	if err := row.Scan(
		&j.ID, &j.JobType, &j.Status, &scheduleExpr, &j.ScheduledAt, &j.StartedAt, &j.FinishedAt,
		&j.BucketStart, &j.BucketEnd, &j.HeartbeatAt, &lockOwner, &j.Attempt, &j.MaxAttempts, &payload, &result, &errMsg,
		&j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if scheduleExpr != nil {
		j.ScheduleExpr = *scheduleExpr
	}
	if lockOwner != nil {
		j.LockOwner = *lockOwner
	}
	if errMsg != nil {
		j.ErrorMessage = *errMsg
	}
	if len(payload) > 0 {
		j.Payload = json.RawMessage(payload)
	}
	if len(result) > 0 {
		j.Result = json.RawMessage(result)
	}
	return &j, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
