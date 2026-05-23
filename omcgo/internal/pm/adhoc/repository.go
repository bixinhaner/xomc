package adhoc

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

// Repository 是 G7 adhoc 任务的持久化接口。
//
// 实现共用 pm_tasks 表（task_subtype='adhoc_aggregation' 行）+ pm_adhoc_aggregation_results。
// 与 pm.PgTaskRepository 同表但独立 Repository，避免破坏老接口。
type Repository interface {
	// Create 插入一行 adhoc 任务（pending 状态），返回 ID。
	Create(ctx context.Context, req CreateRequest) (uuid.UUID, error)

	// Get 按 ID 取单行（不限状态）。
	Get(ctx context.Context, id uuid.UUID) (*Task, error)

	// List 按 filter 取分页结果。
	List(ctx context.Context, filter ListFilter) ([]Task, error)

	// Cancel 把 running/scheduled/pending 行改为 canceled。
	// 已 succeeded/failed 行调用返 ErrTerminalState。
	Cancel(ctx context.Context, id uuid.UUID) error

	// LockNextPending worker 抢任务（pending → running + 占 lock_owner）。
	// 无可用任务返 ErrNoPendingTask。
	LockNextPending(ctx context.Context, lockOwner string) (*Task, error)

	// UpdateStatus 切状态（含可选 progress 更新）。
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status, progress *int, errMsg string) error

	// InsertResults 批量写 pm_adhoc_aggregation_results。
	InsertResults(ctx context.Context, rows []ResultRow) error
}

// Errors

var (
	ErrNoPendingTask = errors.New("adhoc: no pending task available")
	ErrTerminalState = errors.New("adhoc: task already in terminal state")
	ErrNotFound      = errors.New("adhoc: task not found")
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

// pm_tasks 列清单（G7 视角，按本包 Task 模型映射）
// 老列也读：task_name/granularity/time_range/kpi_codes/device_sns(JSONB) — 这些列对 adhoc 无意义，写 NULL 或默认值。
// 新列：task_subtype/mode/cron_expr/metric_paths/granularities/window_start/window_end
var taskCols = []string{
	"id", "task_name", "task_subtype", "mode", "cron_expr",
	"device_sns", "metric_paths", "granularities",
	"window_start", "window_end", "status", "progress",
	"creator", "created_at", "updated_at",
}

func (r *PgRepository) Create(ctx context.Context, req CreateRequest) (uuid.UUID, error) {
	if req.Mode == ModeContinuous && (req.CronExpr == nil || *req.CronExpr == "") {
		return uuid.Nil, errors.New("adhoc.Create: continuous mode requires cron_expr")
	}
	// device_sns 走 JSONB 与老列对齐；metric_paths/granularities 走 TEXT[] 新列。
	deviceSNsJSON, err := json.Marshal(req.DeviceSNs)
	if err != nil {
		return uuid.Nil, fmt.Errorf("adhoc.Create: marshal device_sns: %w", err)
	}
	q, args, err := storage.Psql.Insert("pm_tasks").
		Columns(
			"task_name", "task_type", "task_subtype", "mode", "cron_expr",
			"device_sns", "metric_paths", "granularities",
			"window_start", "window_end", "status", "progress", "creator",
		).
		Values(
			req.Name, "extraction", TaskSubtype, string(req.Mode), nullableString(req.CronExpr),
			deviceSNsJSON, req.MetricPaths, req.Granularities,
			req.WindowStart, req.WindowEnd, string(StatusPending), 0, req.Creator,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("adhoc.Create: build SQL: %w", err)
	}
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("adhoc.Create: insert: %w", err)
	}
	return id, nil
}

func (r *PgRepository) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	q, args, err := storage.Psql.Select(taskCols...).
		From("pm_tasks").
		Where(sq.Eq{"id": id, "task_subtype": TaskSubtype}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("adhoc.Get: build SQL: %w", err)
	}
	row := r.pool.QueryRow(ctx, q, args...)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("adhoc.Get: %w", err)
	}
	return t, nil
}

func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	qb := storage.Psql.Select(taskCols...).
		From("pm_tasks").
		Where(sq.Eq{"task_subtype": TaskSubtype}).
		OrderBy("created_at DESC")
	if filter.Mode != nil {
		qb = qb.Where(sq.Eq{"mode": string(*filter.Mode)})
	}
	if filter.Status != nil {
		qb = qb.Where(sq.Eq{"status": string(*filter.Status)})
	}
	if filter.Creator != "" {
		qb = qb.Where(sq.Eq{"creator": filter.Creator})
	}
	if filter.Limit > 0 {
		qb = qb.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(uint64(filter.Offset))
	}
	q, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("adhoc.List: build SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("adhoc.List: query: %w", err)
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

func (r *PgRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	// 终态行不可 cancel；用 RETURNING 区分行不存在 vs 状态不对。
	const q = `
UPDATE pm_tasks
SET status = 'canceled', updated_at = NOW()
WHERE id = $1
  AND task_subtype = $2
  AND status IN ('pending','running','scheduled')
RETURNING status`
	var newStatus string
	err := r.pool.QueryRow(ctx, q, id, TaskSubtype).Scan(&newStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 行不存在 or 已 succeeded/failed/canceled
			check, _ := r.Get(ctx, id)
			if check == nil {
				return ErrNotFound
			}
			return ErrTerminalState
		}
		return fmt.Errorf("adhoc.Cancel: %w", err)
	}
	return nil
}

// LockNextPending 用 CTE + FOR UPDATE SKIP LOCKED 单 SQL 抢任务。
func (r *PgRepository) LockNextPending(ctx context.Context, lockOwner string) (*Task, error) {
	q := fmt.Sprintf(`
WITH next AS (
    SELECT id FROM pm_tasks
    WHERE task_subtype = $1
      AND status = 'pending'
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE pm_tasks t
SET status = 'running',
    updated_at = NOW()
FROM next
WHERE t.id = next.id
RETURNING %s`, joinCols(taskCols, "t"))
	row := r.pool.QueryRow(ctx, q, TaskSubtype)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoPendingTask
		}
		return nil, fmt.Errorf("adhoc.LockNextPending: %w", err)
	}
	// 单独 UPDATE 一次写 lock_owner（pm_tasks 老 schema 没 lock_owner 列；放在
	// task_name 列后缀 / 单独不存。这里仅 in-memory 设置返回给 worker，便于 log。
	// 多 worker 跨进程靠 SKIP LOCKED 保证一行只被一 worker 抢，lock_owner 仅 log 用。
	owner := lockOwner
	t.LockOwner = &owner
	return t, nil
}

func (r *PgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status, progress *int, errMsg string) error {
	qb := storage.Psql.Update("pm_tasks").
		Set("status", string(status)).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id, "task_subtype": TaskSubtype})
	if progress != nil {
		qb = qb.Set("progress", *progress)
	}
	// 终态时不存 errMsg 列（pm_tasks 没 error_message 列）；走 zap log 即可。
	_ = errMsg
	q, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("adhoc.UpdateStatus: build SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("adhoc.UpdateStatus: exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertResults 批量写 pm_adhoc_aggregation_results。
func (r *PgRepository) InsertResults(ctx context.Context, rows []ResultRow) error {
	if len(rows) == 0 {
		return nil
	}
	ib := storage.Psql.Insert("pm_adhoc_aggregation_results").Columns(
		"task_id", "device_oui", "device_sn", "metric_path", "metric_type", "metric_value",
		"statis_type", "granularity", "time", "start_time", "end_time", "object_ldn", "extra",
	)
	for _, row := range rows {
		var stype, ldn any
		if row.StatisType != nil {
			stype = *row.StatisType
		}
		if row.ObjectLDN != nil {
			ldn = *row.ObjectLDN
		}
		var extra any
		if len(row.Extra) > 0 {
			b, err := json.Marshal(row.Extra)
			if err != nil {
				return fmt.Errorf("adhoc.InsertResults: marshal extra: %w", err)
			}
			extra = b
		}
		t := row.Time
		if t.IsZero() {
			t = row.EndTime
		}
		ib = ib.Values(
			row.TaskID, row.DeviceOUI, row.DeviceSN, row.MetricPath, row.MetricType, row.MetricValue,
			stype, row.Granularity, t, row.StartTime, row.EndTime, ldn, extra,
		)
	}
	q, args, err := ib.ToSql()
	if err != nil {
		return fmt.Errorf("adhoc.InsertResults: build SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("adhoc.InsertResults: exec: %w", err)
	}
	return nil
}

// ── scan helper ───────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (*Task, error) {
	var t Task
	var subtype, mode, cronExpr *string
	var deviceSNsJSON []byte
	var metricPaths, granularities []string
	var windowStart, windowEnd *time.Time
	var status string
	var creator *string

	err := row.Scan(
		&t.ID, &t.Name, &subtype, &mode, &cronExpr,
		&deviceSNsJSON, &metricPaths, &granularities,
		&windowStart, &windowEnd, &status, &t.Progress,
		&creator, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if mode != nil {
		t.Mode = Mode(*mode)
	}
	t.CronExpr = cronExpr
	if len(deviceSNsJSON) > 0 {
		if err := json.Unmarshal(deviceSNsJSON, &t.DeviceSNs); err != nil {
			return nil, fmt.Errorf("scan task: unmarshal device_sns: %w", err)
		}
	}
	t.MetricPaths = metricPaths
	t.Granularities = granularities
	if windowStart != nil {
		t.WindowStart = *windowStart
	}
	if windowEnd != nil {
		t.WindowEnd = *windowEnd
	}
	t.Status = Status(status)
	if creator != nil {
		t.Creator = *creator
	}
	return &t, nil
}

func joinCols(cols []string, alias string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += alias + "." + c
	}
	return out
}

func nullableString(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

// ── ContinuousScheduler 用 SQL 适配器 ────────────────────────────────────

// PgContinuousRepository 适配 PgRepository 给 ContinuousScheduler 用。
type PgContinuousRepository struct {
	pool *pgxpool.Pool
}

// NewPgContinuousRepository 创建 PgContinuousRepository。
func NewPgContinuousRepository(pool *pgxpool.Pool) *PgContinuousRepository {
	return &PgContinuousRepository{pool: pool}
}

var _ ContinuousRepository = (*PgContinuousRepository)(nil)

func (r *PgContinuousRepository) ListReschedulable(ctx context.Context) ([]ContinuousTask, error) {
	const q = `
SELECT id::text, cron_expr, updated_at
FROM pm_tasks
WHERE task_subtype = $1
  AND mode = 'continuous'
  AND status = 'scheduled'
  AND cron_expr IS NOT NULL
ORDER BY updated_at ASC
LIMIT 1000`
	rows, err := r.pool.Query(ctx, q, TaskSubtype)
	if err != nil {
		return nil, fmt.Errorf("PgContinuousRepository.List: %w", err)
	}
	defer rows.Close()
	var out []ContinuousTask
	for rows.Next() {
		var ct ContinuousTask
		if err := rows.Scan(&ct.ID, &ct.CronExpr, &ct.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

func (r *PgContinuousRepository) MarkPending(ctx context.Context, id ContinuousTaskID) error {
	const q = `
UPDATE pm_tasks SET status = 'pending', updated_at = NOW()
WHERE id = $1::uuid AND task_subtype = $2 AND status = 'scheduled'`
	tag, err := r.pool.Exec(ctx, q, id, TaskSubtype)
	if err != nil {
		return fmt.Errorf("PgContinuousRepository.MarkPending: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 已被其它 worker 抢，无害
		return nil
	}
	return nil
}
