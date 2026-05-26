package task

import (
	"context"
	gerr "errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// 列定义与 SELECT/INSERT/scan 顺序严格保持一致。
// 2026-05-25 简化：去除 operator_code / note 列（migration 000188）。
// 2026-05-26 增加 target_device_sns 列（migration 000190）。
var taskColumns = []string{
	"task_id", "task_name", "mr_type", "statis_period", "report_period",
	"start_time", "end_time", "task_status", "task_result",
	"creator", "target_device_sns", "created_at", "updated_at",
}

var progressColumns = []string{
	"id", "task_id", "small_cell_code", "serial_number", "host_name",
	"progress_status", "health_status", "fault_code",
	"last_heartbeat", "missed_heartbeat", "created_at", "updated_at",
}

var _ Repository = (*PgRepository)(nil)

// PgRepository 是 Repository 的 PostgreSQL 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 创建 PgRepository 实例。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// ---------- 任务主表 ----------

// CreateTask 仅写任务主行，不写进度。简化模型 2026-05-25：进度由 scheduler
// 在 task 开启时动态枚举 mr_device_mappings 后调 InsertProgressRows 写入。
func (r *PgRepository) CreateTask(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("create mr task: task is nil")
	}
	insertTask := storage.Psql.Insert("mr_customize_task").
		Columns(
			"task_id", "task_name", "mr_type", "statis_period", "report_period",
			"start_time", "end_time", "task_status", "creator", "target_device_sns",
		).
		Values(
			task.TaskID, task.TaskName, task.MRType, task.StatisPeriod, task.ReportPeriod,
			task.StartTime, task.EndTime, task.TaskStatus, task.Creator,
			pgTextArray(task.TargetDeviceSNs),
		)
	sqlStr, args, err := insertTask.ToSql()
	if err != nil {
		return fmt.Errorf("build insert mr_customize_task SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("insert mr_customize_task: %w", err)
	}
	return nil
}

// InsertProgressRows 批量插入 cell 进度行（pending）。targets 空时 no-op。
// 同 task_id+small_cell_code 已存在则 ON CONFLICT DO NOTHING，
// 保证 scheduler 多次抢锁时的幂等性。
func (r *PgRepository) InsertProgressRows(ctx context.Context, taskID uuid.UUID, targets []CellTarget) error {
	if len(targets) == 0 {
		return nil
	}
	progressInsert := storage.Psql.Insert("mr_customize_task_progress").
		Columns("task_id", "small_cell_code", "serial_number", "host_name", "progress_status", "health_status").
		Suffix("ON CONFLICT (task_id, small_cell_code) DO NOTHING")
	for _, t := range targets {
		progressInsert = progressInsert.Values(taskID, t.SmallCellCode, t.SerialNumber, t.HostName, ProgressPending, HealthUnknown)
	}
	sqlStr, args, err := progressInsert.ToSql()
	if err != nil {
		return fmt.Errorf("build insert progress SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("insert mr_customize_task_progress: %w", err)
	}
	return nil
}

func (r *PgRepository) GetTask(ctx context.Context, taskID uuid.UUID) (*Task, error) {
	q := storage.Psql.Select(taskColumns...).From("mr_customize_task").Where(squirrel.Eq{"task_id": taskID})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mr task SQL: %w", err)
	}
	t, err := scanTask(r.pool.QueryRow(ctx, sqlStr, args...))
	if err != nil {
		if gerr.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mr task not found: %w", commonerrors.ErrNotFound)
		}
		return nil, fmt.Errorf("get mr task: %w", err)
	}
	return t, nil
}

func (r *PgRepository) ListTasks(ctx context.Context, filter TaskListFilter) (*model.ListResponse[Task], error) {
	base := storage.Psql.Select(taskColumns...).From("mr_customize_task")
	countBase := storage.Psql.Select("COUNT(*)").From("mr_customize_task")

	if filter.Status != nil {
		base = base.Where(squirrel.Eq{"task_status": *filter.Status})
		countBase = countBase.Where(squirrel.Eq{"task_status": *filter.Status})
	}
	if filter.Keyword != nil && *filter.Keyword != "" {
		like := "%" + *filter.Keyword + "%"
		cond := squirrel.ILike{"task_name": like}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mr tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr tasks: %w", err)
	}

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := strings.ToLower(filter.SortDir)
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	base = base.OrderBy(sortBy + " " + sortDir).
		Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize))

	sqlStr, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mr tasks SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list mr tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mr task row: %w", err)
		}
		items = append(items, *t)
	}
	return model.NewListResponse(items, total, page, pageSize), nil
}

func (r *PgRepository) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status TaskStatus, result *string) error {
	upd := storage.Psql.Update("mr_customize_task").
		Set("task_status", status).
		Where(squirrel.Eq{"task_id": taskID})
	if result != nil {
		upd = upd.Set("task_result", *result)
	}
	sqlStr, args, err := upd.ToSql()
	if err != nil {
		return fmt.Errorf("build update mr task status SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("update mr task status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mr task not found: %w", commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgRepository) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	del := storage.Psql.Delete("mr_customize_task").Where(squirrel.Eq{"task_id": taskID})
	sqlStr, args, err := del.ToSql()
	if err != nil {
		return fmt.Errorf("build delete mr task SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("delete mr task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mr task not found: %w", commonerrors.ErrNotFound)
	}
	return nil
}

// ---------- 进度表 ----------

func (r *PgRepository) ListProgress(ctx context.Context, filter ProgressListFilter) (*model.ListResponse[Progress], error) {
	base := storage.Psql.Select(progressColumns...).From("mr_customize_task_progress").
		Where(squirrel.Eq{"task_id": filter.TaskID})
	countBase := storage.Psql.Select("COUNT(*)").From("mr_customize_task_progress").
		Where(squirrel.Eq{"task_id": filter.TaskID})

	if filter.Status != nil {
		base = base.Where(squirrel.Eq{"progress_status": *filter.Status})
		countBase = countBase.Where(squirrel.Eq{"progress_status": *filter.Status})
	}
	if filter.Health != nil {
		base = base.Where(squirrel.Eq{"health_status": *filter.Health})
		countBase = countBase.Where(squirrel.Eq{"health_status": *filter.Health})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mr progress SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mr progress: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	base = base.OrderBy("small_cell_code ASC").
		Limit(uint64(pageSize)).
		Offset(uint64((page - 1) * pageSize))

	sqlStr, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mr progress SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list mr progress: %w", err)
	}
	defer rows.Close()

	items := make([]Progress, 0)
	for rows.Next() {
		p, err := scanProgressRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mr progress row: %w", err)
		}
		items = append(items, *p)
	}
	return model.NewListResponse(items, total, page, pageSize), nil
}

func (r *PgRepository) UpdateProgressDispatch(ctx context.Context, taskID uuid.UUID, smallCellCode string, status ProgressStatus, faultCode *string) error {
	upd := storage.Psql.Update("mr_customize_task_progress").
		Set("progress_status", status).
		Set("fault_code", faultCode).
		Where(squirrel.Eq{"task_id": taskID, "small_cell_code": smallCellCode})
	sqlStr, args, err := upd.ToSql()
	if err != nil {
		return fmt.Errorf("build update progress SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("update progress dispatch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mr progress not found: %w", commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgRepository) TouchHeartbeat(ctx context.Context, smallCellCode string) (bool, error) {
	// 只更新当前归属于"已开启任务"的 progress 行（progress_status=openSuccess）。
	// 避免 stale cell（旧任务残留）的心跳被错认。
	sqlStr, args, err := storage.Psql.Update("mr_customize_task_progress p").
		Set("last_heartbeat", time.Now()).
		Set("missed_heartbeat", 0).
		Set("health_status", HealthNormal).
		Where("p.small_cell_code = ?", smallCellCode).
		Where("p.progress_status = ?", ProgressOpenSuccess).
		Where(squirrel.Expr(
			"EXISTS (SELECT 1 FROM mr_customize_task t WHERE t.task_id = p.task_id AND t.task_status = ?)",
			StatusOn,
		)).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build touch heartbeat SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return false, fmt.Errorf("touch heartbeat: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PgRepository) ListActiveTasksByCell(ctx context.Context, smallCellCode string) ([]Task, error) {
	prefixed := make([]string, len(taskColumns))
	for i, c := range taskColumns {
		prefixed[i] = "t." + c
	}
	sqlStr, args, err := storage.Psql.Select(prefixed...).
		From("mr_customize_task t").
		Join("mr_customize_task_progress p ON p.task_id = t.task_id").
		Where(squirrel.Eq{"p.small_cell_code": smallCellCode, "t.task_status": StatusOn}).
		OrderBy("t.start_time DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active tasks by cell SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list active tasks by cell: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		items = append(items, *t)
	}
	return items, nil
}

func (r *PgRepository) ListDueWaitingTasks(ctx context.Context, limit int) ([]Task, error) {
	return r.listDueTasks(ctx, StatusWaiting, "start_time", limit)
}

func (r *PgRepository) ListDueOnTasks(ctx context.Context, limit int) ([]Task, error) {
	return r.listDueTasksWithEndTime(ctx, StatusOn, limit)
}

func (r *PgRepository) listDueTasks(ctx context.Context, status TaskStatus, timeCol string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 100
	}
	sqlStr, args, err := storage.Psql.Select(taskColumns...).
		From("mr_customize_task").
		Where(squirrel.Eq{"task_status": status}).
		Where(squirrel.LtOrEq{timeCol: time.Now()}).
		OrderBy(timeCol + " ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due tasks SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list due tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		items = append(items, *t)
	}
	return items, nil
}

// listDueTasksWithEndTime 仅返回 end_time IS NOT NULL 且已到期的行（区别于
// listDueTasks 用通用 timeCol，本函数对 end_time 列加 NULL 过滤）。
func (r *PgRepository) listDueTasksWithEndTime(ctx context.Context, status TaskStatus, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 100
	}
	sqlStr, args, err := storage.Psql.Select(taskColumns...).
		From("mr_customize_task").
		Where(squirrel.Eq{"task_status": status}).
		Where(squirrel.NotEq{"end_time": nil}).
		Where(squirrel.LtOrEq{"end_time": time.Now()}).
		OrderBy("end_time ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due on tasks SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list due on tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		items = append(items, *t)
	}
	return items, nil
}

// IncrementMissedHeartbeat 在一条 UPDATE ... RETURNING 内完成 missed_heartbeat +1
// 以及条件触发 health_status='abnormal' 的切换。
//
// SQL 设计：
//   - WHERE 联合 task.task_status='on' AND progress.progress_status='openSuccess'
//     避免对已关闭任务 / 未开启 cell 误累加
//   - SET missed_heartbeat = missed_heartbeat + 1
//   - health_status = CASE WHEN missed_heartbeat + 1 >= threshold THEN 'abnormal' ELSE health_status END
//     仅在跨过阈值时切到 abnormal；已经 abnormal 的保持，已经 normal 的若再次 +1 已 >= 阈值
//     也会被推到 abnormal（修正长期 normal 但实际异常的迟到判定）
//   - RETURNING 返回新值，让调用方区分 becameAbnormal 用以指标上报
func (r *PgRepository) IncrementMissedHeartbeat(ctx context.Context, smallCellCode string, abnormalThreshold int) (bool, int, bool, error) {
	if abnormalThreshold <= 0 {
		abnormalThreshold = 2
	}
	const sqlStr = `
UPDATE mr_customize_task_progress p
SET missed_heartbeat = p.missed_heartbeat + 1,
    health_status    = CASE
                         WHEN p.missed_heartbeat + 1 >= $2 THEN 'abnormal'
                         ELSE p.health_status
                       END
FROM mr_customize_task t
WHERE p.task_id = t.task_id
  AND p.small_cell_code = $1
  AND p.progress_status = 'openSuccess'
  AND t.task_status = 'on'
RETURNING p.missed_heartbeat, p.health_status`

	var newMissed int
	var newHealth string
	err := r.pool.QueryRow(ctx, sqlStr, smallCellCode, abnormalThreshold).Scan(&newMissed, &newHealth)
	if err != nil {
		if gerr.Is(err, pgx.ErrNoRows) {
			return false, 0, false, nil
		}
		return false, 0, false, fmt.Errorf("increment missed heartbeat for %s: %w", smallCellCode, err)
	}
	becameAbnormal := newHealth == string(HealthAbnormal) && newMissed == abnormalThreshold
	return true, newMissed, becameAbnormal, nil
}

// pgTextArray 把 Go []string 转成 pgx 能识别的 TEXT[] 参数；nil/空切片传 nil。
// pgx/v5 对 []string 已自动识别为 text[]，本函数只是显式保持空切片不变（避免 NULL 不一致）。
func pgTextArray(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ---------- scan 辅助 ----------

func scanTask(row pgx.Row) (*Task, error) {
	var t Task
	err := row.Scan(
		&t.TaskID, &t.TaskName, &t.MRType, &t.StatisPeriod, &t.ReportPeriod,
		&t.StartTime, &t.EndTime, &t.TaskStatus, &t.TaskResult,
		&t.Creator, &t.TargetDeviceSNs, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTaskRow(rows pgx.Rows) (*Task, error) {
	var t Task
	err := rows.Scan(
		&t.TaskID, &t.TaskName, &t.MRType, &t.StatisPeriod, &t.ReportPeriod,
		&t.StartTime, &t.EndTime, &t.TaskStatus, &t.TaskResult,
		&t.Creator, &t.TargetDeviceSNs, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanProgressRow(rows pgx.Rows) (*Progress, error) {
	var p Progress
	err := rows.Scan(
		&p.ID, &p.TaskID, &p.SmallCellCode, &p.SerialNumber, &p.HostName,
		&p.ProgressStatus, &p.HealthStatus, &p.FaultCode,
		&p.LastHeartbeat, &p.MissedHeartbeat, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
