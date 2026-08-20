package adhoc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

// Repository 是 G7 adhoc 任务的持久化接口。
//
// 实现共用 pm_tasks 表（task_subtype='adhoc_aggregation' 行）+ PM 聚合结果。
// 与 pm.PgTaskRepository 同表但独立 Repository，避免破坏老接口。
type Repository interface {
	// Create 插入一行 adhoc 任务（pending 状态），返回 ID。
	Create(ctx context.Context, req CreateRequest) (uuid.UUID, error)

	// Update 更新一行 adhoc 任务定义（T-0194）。
	//   - 自建任务（is_builtin=false）：更新 name/device_sns/metric_paths/granularities/object_ldns/window_start/window_end。
	//   - 内置任务（is_builtin=true）：只更新 metric_paths，其余字段保持原值（服务端守门）。
	// mode/technology/dimension/is_builtin/expire_days 一律不动。
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) error

	// Get 按 ID 取单行（不限状态）。
	Get(ctx context.Context, id uuid.UUID) (*Task, error)

	// List 按 filter 取分页结果。
	List(ctx context.Context, filter ListFilter) ([]Task, error)

	// Cancel 把 running/scheduled/pending 行改为 canceled。
	// 已 succeeded/failed 行调用返 ErrTerminalState。
	Cancel(ctx context.Context, id uuid.UUID) error

	// Resume 把 canceled 行恢复为可执行状态（#674）。
	//   - continuous → scheduled（让 ContinuousScheduler 下次 sweep 推 pending）
	//   - oneshot → pending（让 worker 直接捞）
	// 非 canceled 行调用返 ErrNotCanceled。
	Resume(ctx context.Context, id uuid.UUID) (Status, error)

	// Delete 硬删一行 adhoc 任务【定义行】（#392）。
	//   - 仅删终态（succeeded/failed/canceled）+ 自建（is_builtin=false）+ task_subtype='adhoc_aggregation' 行。
	//   - 复用过期清理的删除语义（只删 pm_tasks 定义行，绝不触碰结果表，结果交 TimescaleDB retention 自然过期），
	//     但绕过过期天数判断，按 id 即时删。
	//   - 非终态（pending/running/scheduled）行返 ErrTerminalState（语义：仍活跃，应走 Cancel）。
	//   - 内置任务（is_builtin=true）返 ErrBuiltinNotDeletable。
	//   - 行不存在返 ErrNotFound。
	Delete(ctx context.Context, id uuid.UUID) error

	// LockNextPending worker 抢任务（pending → running + 占 lock_owner）。
	// 无可用任务返 ErrNoPendingTask。
	LockNextPending(ctx context.Context, lockOwner string) (*Task, error)

	// UpdateStatus 切状态（含可选 progress 更新）。
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status, progress *int, errMsg string) error

	// InsertResults 批量写 pm_adhoc_aggregation_results。
	InsertResults(ctx context.Context, rows []ResultRow) error

	// NextRunSeq 返回该任务下一个运行编号（已有 run 数 + 1）。
	NextRunSeq(ctx context.Context, taskID uuid.UUID) (int, error)

	// InsertRun 写一行运行记录（status=running，跑前调用），返回该 run 的 ID。
	InsertRun(ctx context.Context, run TaskRun) (uuid.UUID, error)

	// FinishRun 更新运行记录为终态（status/finished_at/rows_total/error，跑后调用）。
	FinishRun(ctx context.Context, runID uuid.UUID, status Status, rowsTotal int, errMsg string) error

	// ListRuns 按 task_id 取运行历史，倒序 started_at，支持 limit/offset。
	ListRuns(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]TaskRun, error)

	// ListResultMetricPaths 按结果过滤条件从当前稳定聚合结果中发现真实出现过的指标。
	ListResultMetricPaths(ctx context.Context, taskID uuid.UUID, filter resultsFilter) ([]string, error)
}

// Errors

var (
	ErrNoPendingTask = errors.New("adhoc: no pending task available")
	ErrTerminalState = errors.New("adhoc: task already in terminal state")
	ErrNotFound      = errors.New("adhoc: task not found")
	// ErrNotTerminal #392：删除端点对非终态（pending/running/scheduled）任务返回——仍活跃，应走 Cancel。
	ErrNotTerminal = errors.New("adhoc: task not in terminal state, cannot delete")
	// ErrBuiltinNotDeletable #392：删除端点对内置任务返回——内置任务由 seed 维护，永不可删。
	ErrBuiltinNotDeletable = errors.New("adhoc: builtin task cannot be deleted")
	// ErrNotCanceled #674：Resume 端点对非 canceled 任务返回——只有已取消的任务才能恢复。
	ErrNotCanceled = errors.New("adhoc: task is not canceled, cannot resume")
)

// PgRepository 是 Repository 的 pgxpool 实现。
//
// 双池（KPI/时序库物理分离）：
//   - pool（主库 PgPool）：pm_tasks / pm_adhoc_task_runs 任务生命周期表（Create/Update/Get/List/
//     Cancel/LockNextPending/UpdateStatus/InsertRun/FinishRun/ListRuns/NextRunSeq）。
//   - tsPool（时序库 TsPool）：adhoc legacy 写入表 + streaming 聚合结果表。
type PgRepository struct {
	pool   *pgxpool.Pool // 主库：任务生命周期表
	tsPool *pgxpool.Pool // 时序库：adhoc legacy 写入表 + streaming 聚合结果表
	// watermarks 读上游「完成水位」（#528 P3）。新建持续任务时把初始游标 last_fire_at
	// 置为「建任务时刻当前对应水位桶起点」——从「现在」起算、不回扫历史、结果表不冒出史前空格。
	// nil 安全：不注入则 last_fire_at 留 NULL（退化到 created_at），行为不回归。
	watermarks WatermarkReader
	loc        func() *time.Location // #528 P3：初始游标桶对齐用业务时区
	timezone   calendarfilter.TimezoneProvider
	streamRepo streamingTaskRepository
}

type streamingTaskRepository interface {
	Save(context.Context, pmstream.SaveTaskRequest) (*pmstream.TaskVersionSnapshot, error)
	Delete(context.Context, uuid.UUID) error
	PurgeObsoleteBuiltinDeviceTasks(context.Context) (int, error)
	RetireMissingSourceTasks(context.Context, string, string, uint64) (int, error)
}

// NewPgRepository 创建 PgRepository。
//
// pgPool=主库（pm_tasks/pm_adhoc_task_runs），tsPool=时序库（PM 结果表）。
func NewPgRepository(pgPool, tsPool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pgPool, tsPool: tsPool, loc: func() *time.Location { return time.UTC }}
}

// SetStreamingRepository 把现有任务 CRUD 接到新的不可变版本控制面。
func (r *PgRepository) SetStreamingRepository(repo streamingTaskRepository) *PgRepository {
	r.streamRepo = repo
	return r
}

func (r *PgRepository) HasStreamingRepository() bool {
	return r.streamRepo != nil
}

func (r *PgRepository) ListResultMetricPaths(ctx context.Context, taskID uuid.UUID, filter resultsFilter) ([]string, error) {
	if r.tsPool == nil {
		return nil, errors.New("adhoc: tsdb pool is not configured")
	}
	if filter.CalendarTimezone == "" {
		filter.CalendarTimezone = calendarfilter.ProviderName(ctx, r.timezone)
	}
	query, args, err := buildResultMetricScopeQuery(taskID, filter)
	if err != nil {
		return nil, fmt.Errorf("build adhoc result metric scope query: %w", err)
	}
	rows, err := r.tsPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query adhoc result metric scope: %w", err)
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan adhoc result metric scope: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adhoc result metric scope: %w", err)
	}
	return paths, nil
}

func buildResultMetricScopeQuery(taskID uuid.UUID, f resultsFilter) (string, []any, error) {
	where, cteWhere, filterArgs, _ := buildAdhocAggregationResultFilters(f, false, 2)
	args := append([]any{taskID}, filterArgs...)
	q := adhocCurrentVersionsCTE(cteWhere) + `
SELECT DISTINCT r.metric_path
FROM pm_aggregation_results r
JOIN current_versions cv
  ON cv.task_id = r.task_id
 AND cv.granularity = r.granularity
 AND cv.window_start = r.window_start
 AND cv.task_version_id = r.task_version_id
JOIN pm_aggregation_publications published_window
  ON published_window.task_id = r.task_id
 AND published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
WHERE r.task_id = $1` + where + `
ORDER BY r.metric_path`
	return q, args, nil
}

func buildAdhocAggregationResultFilters(f resultsFilter, includeTaskMetricPaths bool, startPos int) (whereSQL, cteWhereSQL string, args []any, nextPos int) {
	pos := startPos
	add := func(format string, arg any) {
		whereSQL += fmt.Sprintf(format, pos)
		args = append(args, arg)
		pos++
	}
	addShared := func(whereFormat, cteFormat string, arg any) {
		whereSQL += fmt.Sprintf(whereFormat, pos)
		cteWhereSQL += fmt.Sprintf(cteFormat, pos)
		args = append(args, arg)
		pos++
	}
	objectExpr := "CASE WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '') WHEN r.dimension = 'network' THEN 'Network' ELSE r.dimension_key END"
	deviceSNExpr := "CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END"
	if predicate := dimensionPredicate(f.Dimension); predicate != "" {
		whereSQL += " AND " + predicate
	}
	if f.DeviceSN != "" {
		add(" AND "+deviceSNExpr+" = $%d", f.DeviceSN)
	}
	if f.MetricPath != "" {
		add(" AND r.metric_path = $%d", f.MetricPath)
	}
	if f.Granularity != "" {
		addShared(" AND r.granularity = $%d", " AND granularity = $%d", f.Granularity)
	}
	if f.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, f.StartTime); err == nil {
			addShared(" AND r.window_start >= $%d", " AND window_start >= $%d", t)
		}
	}
	if f.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, f.EndTime); err == nil {
			addShared(" AND r.window_start < $%d", " AND window_start < $%d", t)
		}
	}
	if len(f.ObjectLDNs) > 0 {
		add(" AND "+objectExpr+" = ANY($%d)", f.ObjectLDNs)
	}
	if len(f.ProductIDs) > 0 {
		add(" AND r.dimension = 'product' AND r.dimension_key = ANY($%d)", f.ProductIDs)
	}
	if len(f.SubsetLDNs) > 0 {
		add(" AND "+objectExpr+" = ANY($%d)", f.SubsetLDNs)
	}
	if includeTaskMetricPaths && len(f.TaskMetricPaths) > 0 {
		add(" AND r.metric_path = ANY($%d)", f.TaskMetricPaths)
	}
	if len(f.Weekdays) > 0 && len(f.Weekdays) < 7 {
		whereSQL += fmt.Sprintf(" AND EXTRACT(dow FROM (r.window_start AT TIME ZONE $%d))::int = ANY($%d)", pos, pos+1)
		args = append(args, calendarfilter.NormalizeName(f.CalendarTimezone), f.Weekdays)
		pos += 2
	}
	if len(f.Hours) > 0 && len(f.Hours) < 24 {
		whereSQL += fmt.Sprintf(" AND EXTRACT(hour FROM (r.window_start AT TIME ZONE $%d))::int = ANY($%d)", pos, pos+1)
		args = append(args, calendarfilter.NormalizeName(f.CalendarTimezone), f.Hours)
		pos += 2
	}
	return whereSQL, cteWhereSQL, args, pos
}

func dimensionPredicate(dim Dimension) string {
	switch dim {
	case DimensionDevice:
		return "r.dimension = 'device'"
	case DimensionAggregateGroup:
		return "r.dimension = 'aggregate_group'"
	case DimensionProduct:
		return "r.dimension = 'product'"
	case DimensionBand:
		return "r.dimension = 'band'"
	case DimensionNetwork:
		return "r.dimension = 'network'"
	case DimensionDeviceGroup:
		return "r.dimension = 'device_group'"
	default:
		return ""
	}
}

func adhocCurrentVersionsCTE(cteWhere string) string {
	return `WITH current_versions AS MATERIALIZED (
    SELECT DISTINCT ON (candidate.task_id, candidate.granularity, candidate.window_start)
        candidate.task_id,
        candidate.granularity,
        candidate.window_start,
        candidate.task_version_id
    FROM (
        SELECT p.task_id, p.granularity, p.window_start, p.task_version_id,
               (SELECT MAX(w.version_effective_from)
                  FROM pm_aggregation_windows w
                 WHERE w.task_version_id = p.task_version_id
                   AND w.granularity = p.granularity
                   AND w.window_start = p.window_start) AS version_effective_from,
               p.revision AS max_revision,
               p.published_at AS max_published_at,
               p.updated_at AS max_updated_at
        FROM pm_aggregation_publications p
        WHERE p.task_id = $1
          AND p.status = 'published'` + cteWhere + `
    ) candidate
    WHERE NOT EXISTS (
        SELECT 1
        FROM pm_aggregation_windows active_window
        WHERE active_window.task_id = candidate.task_id
          AND active_window.granularity = candidate.granularity
          AND active_window.window_start = candidate.window_start
          AND active_window.task_version_id <> candidate.task_version_id
          AND active_window.status IN ('open', 'finalizing', 'prepared', 'rebuilding', 'failed')
          AND active_window.version_effective_from IS NOT NULL
          AND (
              candidate.version_effective_from IS NULL
              OR active_window.version_effective_from > candidate.version_effective_from
          )
    )
    ORDER BY candidate.task_id, candidate.granularity, candidate.window_start,
             candidate.version_effective_from DESC NULLS LAST,
             candidate.max_revision DESC,
             candidate.max_published_at DESC NULLS LAST,
             candidate.max_updated_at DESC,
             candidate.task_version_id DESC
)`
}

// buildResultsQuery 纯函数：拼 adhoc results 查询 SQL + 占位参数。
// 抽出来便于单测（带/不带大时间段两路）；时间段非法值容错忽略而非报错。
func buildResultsQuery(taskID uuid.UUID, f resultsFilter, limit, offset int) (string, []any) {
	// PM-线名解析：LEFT JOIN 在读时把分组键 ID 解析成可读名 —— product 维度按 product_id 取
	// product_dim.product_name；device_group 维度按 'DeviceGroup='||id 比对 object_ldn 取 device_group_dim.name。
	// 两 JOIN 都是 LEFT，互不影响（product 任务时组名 NULL、组任务时产品名 NULL）；名缺失（脏数据/已删）也返 NULL，前端回退 id 前 8 位。
	// 查询跑在 TsPool；结果直接读 pm_aggregation_results，products/device_groups 改读本库影子表。
	where, cteWhere, filterArgs, pos := buildAdhocAggregationResultFilters(f, true, 2)
	args := append([]any{taskID}, filterArgs...)
	q := adhocCurrentVersionsCTE(cteWhere) + `
SELECT r.id, r.task_id, r.device_oui,
       CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END AS device_sn,
       CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
       r.metric_path, r.metric_type, r.metric_value,
       r.aggregation_op::text AS statis_type, r.granularity::text,
       r.window_start AS "time", r.window_start AS start_time, r.window_end AS end_time,
       r.created_at AS ingest_time,
       CASE WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '') WHEN r.dimension = 'network' THEN 'Network' ELSE r.dimension_key END AS object_ldn,
       jsonb_build_object(
           'task_version_id', r.task_version_id,
           'complete', r.period_complete,
           'missing_slots', r.missing_slots,
           'dimension', r.dimension,
           'revision', r.revision,
           'version_effective_from', r.version_effective_from,
           'version_effective_to', r.version_effective_to,
           'received_slots', r.received_slots,
           'expected_slots', r.expected_slots,
           'version_expected_slots', r.version_expected_slots,
           'natural_expected_slots', r.natural_expected_slots,
           'version_slice_complete', r.version_slice_complete,
           'period_complete', r.period_complete,
           'partial', false
       ) AS extra,
       p.product_name, g.name AS device_group_name
FROM pm_aggregation_results r
JOIN current_versions cv
  ON cv.task_id = r.task_id
 AND cv.granularity = r.granularity
 AND cv.window_start = r.window_start
 AND cv.task_version_id = r.task_version_id
JOIN pm_aggregation_publications published_window
  ON published_window.task_id = r.task_id
 AND published_window.task_version_id = r.task_version_id
 AND published_window.granularity = r.granularity
 AND published_window.window_start = r.window_start
 AND published_window.status = 'published'
 AND published_window.revision = r.revision
LEFT JOIN product_dim p ON r.dimension = 'product' AND p.id::text = r.dimension_key
LEFT JOIN device_group_dim g ON r.dimension = 'device_group' AND ('DeviceGroup=' || g.id::text) = split_part(r.dimension_key, ',', 1)
WHERE r.task_id = $1` + where
	q += fmt.Sprintf(" ORDER BY r.window_start DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)
	return q, args
}

// SetWatermarkReader 注入「上游完成水位」读取器（#528 P3，新建持续任务初始游标用）。
func (r *PgRepository) SetWatermarkReader(reader WatermarkReader) *PgRepository {
	r.watermarks = reader
	return r
}

// HasWatermarkReader 报告本 repo 是否已注入「上游完成水位」读取器（#528 P3）。
//
// 仅供装配回归测试用：建持续任务的唯一入口是 app 进程的 POST /pm/adhoc，其 repo 必须注入
// 水位读取器，否则新建持续任务初始游标退化为 NULL（落回 created_at），结果表会冒出史前空格。
// 检查方曾发现「水位读取器误注入到 worker 进程（从不建任务）」的装配缺口——本 getter 让该缺口
// 能在不连真库的单测里被钉死。
func (r *PgRepository) HasWatermarkReader() bool {
	return r.watermarks != nil
}

// SetLocationFunc 注入业务时区取值器（#528 P3，初始游标桶对齐用）。
func (r *PgRepository) SetLocationFunc(fn func() *time.Location) *PgRepository {
	if fn != nil {
		r.loc = fn
	}
	return r
}

func (r *PgRepository) SetTimezoneProvider(provider calendarfilter.TimezoneProvider) *PgRepository {
	r.timezone = provider
	return r
}

// initialCursorForContinuous 求新建持续任务的初始游标 last_fire_at（#528 P3）。
//
// = 建任务时刻「当前对应 (粒度,层级) 完成水位的桶起点」。下次 sweep 从该游标算 cron.Next，
// 加上 P3 的水位 gate（只追 ≤ 水位的格），新任务从当前水位起算、绝不回扫历史空格。
// 无水位读取器 / 无粒度 / 上游尚未卷完任何格 / 读取出错 → 返回 ok=false（last_fire_at 留 NULL，
// 退化到 created_at；新环境下水位 gate 仍会挡住史前格，安全）。
func (r *PgRepository) initialCursorForContinuous(ctx context.Context, req CreateRequest) (time.Time, bool) {
	if r.watermarks == nil || len(req.Granularities) == 0 {
		return time.Time{}, false
	}
	g := metrics.Granularity(req.Granularities[0])
	dim := req.Dimension
	if dim == "" {
		dim = DimensionDevice
	}
	level := watermarkLevelForDimension(dim)
	wm, err := r.watermarks.Get(ctx, g, level)
	if err != nil {
		return time.Time{}, false
	}
	loc := time.UTC
	if r.loc != nil {
		if l := r.loc(); l != nil {
			loc = l
		}
	}
	return truncateBucketStart(g, wm.CompletedBucketStart, loc), true
}

var _ Repository = (*PgRepository)(nil)

// pm_tasks 列清单（G7 视角，按本包 Task 模型映射）
// 老列也读：task_name/granularity/time_range/kpi_codes/device_sns(JSONB) — 这些列对 adhoc 无意义，写 NULL 或默认值。
// 新列：task_subtype/mode/cron_expr/metric_paths/granularities/window_start/window_end
var taskCols = []string{
	"id", "task_name", "task_subtype", "mode", "cron_expr",
	"device_sns", "metric_paths", "granularities",
	"window_start", "window_end", "dimension", "technology", "is_builtin", "expire_days",
	"planned_end_at", "visibility", "status", "progress",
	"creator", "created_at", "updated_at",
	"object_ldns", // T-0193：小区/PLMN 白名单（TEXT[]，NULL=不过滤）
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
	dim := req.Dimension
	if dim == "" {
		dim = DimensionDevice
	}
	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = 60 // 默认 60 天（约束任务定义层，与结果数据 PM 保留期分离）
	}
	visibility := normalizeVisibility(req.Visibility)
	// #528 P3：持续任务初始游标 = 建任务时刻当前对应水位桶起点（从「现在」起算，不回扫历史）。
	// 取不到水位（上游尚未卷完 / 未注入读取器）→ last_fire_at 留 NULL，退化到 created_at，
	// 水位 gate 仍兜底挡史前格，安全。oneshot 任务无 cron 调度，初始游标无意义。
	var initialFire interface{}
	if req.Mode == ModeContinuous {
		if bucket, ok := r.initialCursorForContinuous(ctx, req); ok {
			initialFire = bucket
		}
	}
	plannedEndAt := plannedEndValue(req)
	q, args, err := storage.Psql.Insert("pm_tasks").
		Columns(
			"task_name", "task_type", "task_subtype", "mode", "cron_expr",
			"device_sns", "metric_paths", "granularities",
			"window_start", "window_end", "dimension", "technology", "is_builtin", "expire_days",
			"planned_end_at", "visibility", "status", "progress", "creator", "object_ldns", "last_fire_at",
		).
		Values(
			req.Name, "extraction", TaskSubtype, string(req.Mode), nullableString(req.CronExpr),
			deviceSNsJSON, req.MetricPaths, req.Granularities,
			nullableTime(req.WindowStart), nullableTime(req.WindowEnd), string(dim), nullableTech(req.Technology), req.IsBuiltin, expireDays,
			plannedEndAt, string(visibility), string(StatusScheduled), 0, req.Creator, nullableStrSlice(req.ObjectLDNs), initialFire,
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
	if r.streamRepo != nil {
		task, loadErr := r.Get(ctx, id)
		if loadErr != nil {
			return uuid.Nil, loadErr
		}
		if syncErr := r.syncStreamingTask(ctx, task, true); syncErr != nil {
			_, _ = r.pool.Exec(ctx, "DELETE FROM pm_tasks WHERE id=$1", id)
			return uuid.Nil, syncErr
		}
	}
	return id, nil
}

// Update 更新一行 adhoc 任务定义（T-0194）。
//
// 内置守门：req.IsBuiltin=true 时只 SET metric_paths（其余字段保持原值）；
// 自建任务（false）SET 全部可编辑字段。mode/technology/dimension/is_builtin/expire_days 永不进 SET。
// 按 id + task_subtype='adhoc_aggregation' 限定；行不存在返 ErrNotFound。
func (r *PgRepository) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) error {
	if !req.IsBuiltin && req.Mode == ModeContinuous && req.ResetCursor {
		if bucket, ok := r.initialCursorForContinuous(ctx, CreateRequest{
			Mode:          ModeContinuous,
			Granularities: req.Granularities,
			Dimension:     req.Dimension,
		}); ok {
			req.LastFireAt = bucket
		} else {
			req.LastFireAt = time.Now().UTC()
		}
	}
	var originalTask *Task
	if r.streamRepo != nil {
		task, loadErr := r.Get(ctx, id)
		if loadErr != nil {
			return loadErr
		}
		originalTask = task
		candidate := applyUpdateToTask(task, req)
		if syncErr := r.syncStreamingTask(ctx, candidate, candidate.Status != StatusCanceled); syncErr != nil {
			return syncErr
		}
	}
	q, args, err := buildUpdateSQL(id, req)
	if err != nil {
		return fmt.Errorf("adhoc.Update: build SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		if originalTask != nil {
			_ = r.syncStreamingTask(ctx, originalTask, originalTask.Status != StatusCanceled)
		}
		return fmt.Errorf("adhoc.Update: exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if originalTask != nil {
			_ = r.syncStreamingTask(ctx, originalTask, originalTask.Status != StatusCanceled)
		}
		return ErrNotFound
	}
	return nil
}

func applyUpdateToTask(task *Task, req UpdateRequest) *Task {
	if task == nil {
		return nil
	}
	updated := *task
	updated.MetricPaths = append([]string(nil), req.MetricPaths...)
	if req.IsBuiltin {
		return &updated
	}
	updated.Name = req.Name
	updated.CronExpr = req.CronExpr
	updated.DeviceSNs = append([]string(nil), req.DeviceSNs...)
	updated.Granularities = append([]string(nil), req.Granularities...)
	updated.ObjectLDNs = append([]string(nil), req.ObjectLDNs...)
	updated.WindowStart = req.WindowStart
	updated.WindowEnd = req.WindowEnd
	updated.Visibility = normalizeVisibility(req.Visibility)
	if req.PlannedEndAtSet {
		updated.PlannedEndAt = req.PlannedEndAt
	}
	return &updated
}

// buildUpdateSQL 构建编辑任务的 UPDATE SQL（T-0194）。抽出便于单测断言守门口径（哪些列进 SET）。
//
// 内置（IsBuiltin=true）：只 SET metric_paths + updated_at。
// 自建（false）：额外 SET task_name/device_sns(JSONB)/granularities/cron_expr/object_ldns/window_start/window_end。
// 自建 oneshot 的执行输入变化时，已执行完成的 succeeded/failed 任务重新排队为 pending；pending/running 保持原状态。
// mode/technology/dimension/is_builtin/expire_days 永不进 SET。
func buildUpdateSQL(id uuid.UUID, req UpdateRequest) (string, []any, error) {
	qb := storage.Psql.Update("pm_tasks").
		Set("metric_paths", req.MetricPaths).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id, "task_subtype": TaskSubtype})
	if !req.IsBuiltin {
		deviceSNsJSON, err := json.Marshal(req.DeviceSNs)
		if err != nil {
			return "", nil, fmt.Errorf("marshal device_sns: %w", err)
		}
		qb = qb.
			Set("task_name", req.Name).
			Set("device_sns", deviceSNsJSON).
			Set("granularities", req.Granularities).
			Set("cron_expr", nullableString(req.CronExpr)).
			Set("visibility", string(normalizeVisibility(req.Visibility))).
			Set("object_ldns", nullableStrSlice(req.ObjectLDNs)).
			Set("window_start", nullableTime(req.WindowStart)).
			Set("window_end", nullableTime(req.WindowEnd))
		if req.PlannedEndAtSet {
			qb = qb.Set("planned_end_at", nullablePtrTime(req.PlannedEndAt))
		}
		if req.Mode == ModeOneshot && req.RequeueTerminal {
			qb = qb.
				Set("status", sq.Expr("CASE WHEN status IN ('succeeded','failed') THEN 'pending' ELSE status END")).
				Set("progress", sq.Expr("CASE WHEN status IN ('succeeded','failed') THEN 0 ELSE progress END"))
		}
		if req.ResetCursor {
			qb = qb.Set("last_fire_at", nullableTime(req.LastFireAt))
		}
	}
	return qb.ToSql()
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
	q, args, err := buildListSQL(filter)
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

func buildListSQL(filter ListFilter) (string, []any, error) {
	qb := storage.Psql.Select(taskCols...).
		From("pm_tasks").
		Where(sq.Eq{"task_subtype": TaskSubtype}).
		OrderBy("task_name ASC")
	if filter.Mode != nil {
		qb = qb.Where(sq.Eq{"mode": string(*filter.Mode)})
	}
	if filter.Status != nil {
		qb = qb.Where(sq.Eq{"status": string(*filter.Status)})
	}
	if filter.Creator != "" {
		qb = qb.Where(sq.Eq{"creator": filter.Creator})
	}
	if filter.IsBuiltin != nil {
		qb = qb.Where(sq.Eq{"is_builtin": *filter.IsBuiltin})
	}
	if !filter.IncludeAll {
		qb = qb.Where(visibleTaskExpr(filter.CurrentUser))
	}
	if filter.Limit > 0 {
		qb = qb.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(uint64(filter.Offset))
	}
	return qb.ToSql()
}

func visibleTaskExpr(currentUser string) sq.Sqlizer {
	return sq.Or{
		sq.Eq{"is_builtin": true},
		sq.Eq{"visibility": string(VisibilityPublic)},
		sq.Eq{"creator": currentUser},
	}
}

func (r *PgRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	// #188：planned_end_at 独立承载计划结束时间，取消任务不再覆盖 window_end/planned_end_at。
	const q = `
UPDATE pm_tasks
SET status = 'canceled',
    updated_at = NOW()
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
	if r.streamRepo != nil {
		task, loadErr := r.Get(ctx, id)
		if loadErr != nil {
			return loadErr
		}
		if syncErr := r.syncStreamingTask(ctx, task, false); syncErr != nil {
			return syncErr
		}
	}
	return nil
}

// Resume 把 canceled 行恢复为可执行状态（#674）。
//
// continuous → scheduled（让 ContinuousScheduler 下次 sweep 推 pending）；
// oneshot → pending（让 worker 直接捞）。
// 同时清 last_fire_at（避免恢复后狂追历史）；planned_end_at 保留，继续约束恢复后的后续调度。
func (r *PgRepository) Resume(ctx context.Context, id uuid.UUID) (Status, error) {
	const q = `
UPDATE pm_tasks
SET status       = CASE WHEN mode = 'continuous' THEN 'scheduled' ELSE 'pending' END,
    last_fire_at = NULL,
    updated_at   = NOW()
WHERE id = $1
  AND task_subtype = $2
  AND status = 'canceled'
RETURNING status`
	var newStatus string
	err := r.pool.QueryRow(ctx, q, id, TaskSubtype).Scan(&newStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			check, _ := r.Get(ctx, id)
			if check == nil {
				return "", ErrNotFound
			}
			return "", ErrNotCanceled
		}
		return "", fmt.Errorf("adhoc.Resume: %w", err)
	}
	if r.streamRepo != nil {
		task, loadErr := r.Get(ctx, id)
		if loadErr != nil {
			return "", loadErr
		}
		if syncErr := r.syncStreamingTask(ctx, task, true); syncErr != nil {
			return "", syncErr
		}
	}
	return Status(newStatus), nil
}

// Delete 硬删一行终态自建 adhoc 任务定义行（#392）。
//
// 复用 ExpireCleanup 的删除语义（只删 pm_tasks 定义行，绝不触碰结果表 pm_adhoc_aggregation_results——
// 结果是 TimescaleDB 超表，由 add_retention_policy 365 天自动 drop_chunks），但绕过过期天数判断按 id 即时删。
//
// 删除条件全部 AND：
//   - id = $1
//   - task_subtype = 'adhoc_aggregation'（只动 adhoc 行）
//   - is_builtin = false（内置任务排除）
//   - status IN ('succeeded','failed','canceled')（仅终态可删）
//
// 删 0 行时回查行状态区分原因：不存在→ErrNotFound、内置→ErrBuiltinNotDeletable、非终态→ErrNotTerminal。
func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	check, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if check.IsBuiltin {
		return ErrBuiltinNotDeletable
	}
	if check.Status != StatusSucceeded && check.Status != StatusFailed && check.Status != StatusCanceled {
		return ErrNotTerminal
	}
	if r.streamRepo != nil {
		if err := r.streamRepo.Delete(ctx, id); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	const q = `
DELETE FROM pm_tasks
WHERE id = $1
  AND task_subtype = $2
  AND is_builtin = false
  AND status IN ('succeeded','failed','canceled')`
	tag, err := r.pool.Exec(ctx, q, id, TaskSubtype)
	if err != nil {
		return fmt.Errorf("adhoc.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
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
      AND (is_builtin = true OR planned_end_at IS NULL OR planned_end_at > NOW())
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
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id, "task_subtype": TaskSubtype})
	if status == StatusScheduled {
		qb = qb.Set("status", sq.Expr(`
CASE
  WHEN mode = 'continuous'
   AND is_builtin = false
   AND planned_end_at IS NOT NULL
   AND planned_end_at <= NOW()
  THEN 'canceled'
  ELSE ?
END`, string(status)))
	} else {
		qb = qb.Set("status", string(status))
	}
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

// StopExpiredPlannedContinuous 把到达计划结束时间的自建 continuous 任务停止，并同步关闭流式任务。
//
// 只处理自建任务（is_builtin=false），内置任务不受 planned_end_at 影响。结果数据不回填、不清理。
func (r *PgRepository) StopExpiredPlannedContinuous(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 1000
	}
	const activeOnlySQL = `
WITH expired AS (
    SELECT t.id
    FROM pm_tasks t
    WHERE t.task_subtype = $1
      AND t.mode = 'continuous'
      AND t.is_builtin = false
      AND t.planned_end_at IS NOT NULL
      AND t.planned_end_at <= $2
      AND t.status IN ('pending','running','scheduled')
    ORDER BY t.planned_end_at ASC
    LIMIT $3
    FOR UPDATE OF t SKIP LOCKED
)
UPDATE pm_tasks t
SET status = 'canceled',
    updated_at = NOW()
FROM expired
WHERE t.id = expired.id
RETURNING t.id`
	const activeOrUnsyncedSQL = `
WITH expired AS (
    SELECT t.id
    FROM pm_tasks t
    WHERE t.task_subtype = $1
      AND t.mode = 'continuous'
      AND t.is_builtin = false
      AND t.planned_end_at IS NOT NULL
      AND t.planned_end_at <= $2
      AND (t.status IN ('pending','running','scheduled') OR (
        t.status = 'canceled'
        AND EXISTS (
            SELECT 1 FROM pm_aggregation_tasks st
            WHERE st.id = t.id AND st.enabled = true
        )
      ))
    ORDER BY t.planned_end_at ASC
    LIMIT $3
    FOR UPDATE OF t SKIP LOCKED
)
UPDATE pm_tasks t
SET status = 'canceled',
    updated_at = NOW()
FROM expired
WHERE t.id = expired.id
RETURNING t.id`
	q := activeOnlySQL
	if r.streamRepo != nil {
		q = activeOrUnsyncedSQL
	}
	rows, err := r.pool.Query(ctx, q, TaskSubtype, now, limit)
	if err != nil {
		return 0, fmt.Errorf("adhoc.StopExpiredPlannedContinuous: %w", err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if r.streamRepo != nil {
		for _, id := range ids {
			task, loadErr := r.Get(ctx, id)
			if loadErr != nil {
				return len(ids), loadErr
			}
			if syncErr := r.syncStreamingTask(ctx, task, false); syncErr != nil {
				return len(ids), syncErr
			}
		}
	}
	return len(ids), nil
}

// resultBusinessKey 是结果行的业务唯一键（与 migrations/000018 的唯一索引 8 列一致）。
// 可空列统一兜空串，对齐索引里的 COALESCE(...,”)，保证「两条空值行」也判同键。
type resultBusinessKey struct {
	TaskID      uuid.UUID
	Granularity string
	MetricPath  string
	DeviceOUI   string
	DeviceSN    string
	ProductID   string // product_id::text，Nil → ""
	ObjectLDN   string
	Time        time.Time
}

func businessKeyOf(row ResultRow) resultBusinessKey {
	productID := ""
	if row.ProductID != uuid.Nil {
		productID = row.ProductID.String()
	}
	ldn := ""
	if row.ObjectLDN != nil {
		ldn = *row.ObjectLDN
	}
	t := row.Time
	if t.IsZero() {
		t = row.EndTime
	}
	return resultBusinessKey{
		TaskID:      row.TaskID,
		Granularity: row.Granularity,
		MetricPath:  row.MetricPath,
		DeviceOUI:   row.DeviceOUI,
		DeviceSN:    row.DeviceSN,
		ProductID:   productID,
		ObjectLDN:   ldn,
		Time:        t,
	}
}

// dedupResultRows 按业务唯一键对入参做同批去重（后者覆盖前者，保留最后一条），
// 保持首次出现顺序。防止同一次 rows 切片含重复业务键时 ON CONFLICT DO UPDATE 报
// 「command cannot affect row a second time」。纯函数，便于单测。
func dedupResultRows(rows []ResultRow) []ResultRow {
	if len(rows) <= 1 {
		return rows
	}
	idxByKey := make(map[resultBusinessKey]int, len(rows))
	out := make([]ResultRow, 0, len(rows))
	for _, row := range rows {
		key := businessKeyOf(row)
		if i, ok := idxByKey[key]; ok {
			out[i] = row // 后者覆盖前者（保留最后一条），位置不变
			continue
		}
		idxByKey[key] = len(out)
		out = append(out, row)
	}
	return out
}

// onConflictResultsBusiness 是 ON CONFLICT 的冲突目标表达式，必须与
// migrations/000018 的 uq_pm_adhoc_results_business 索引表达式逐字一致（PG 表达式索引推断要求）。
// 冲突命中时用 EXCLUDED 覆盖值类列（time 是冲突键不更新）。
const onConflictResultsBusiness = `ON CONFLICT (task_id, granularity, metric_path, ` +
	`COALESCE(device_oui, ''), COALESCE(device_sn, ''), ` +
	`COALESCE(product_id::text, ''), COALESCE(object_ldn, ''), "time") ` +
	`DO UPDATE SET metric_value = EXCLUDED.metric_value, ` +
	`metric_type = EXCLUDED.metric_type, statis_type = EXCLUDED.statis_type, ` +
	`start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, extra = EXCLUDED.extra`

// resultInsertCols 是结果表 INSERT 的列清单（14 列）。
// 单独抽出供分批 SQL 构建与单测复用；列序必须与 buildInsertResultsSQL 的 Values 一致。
var resultInsertCols = []string{
	"task_id", "device_oui", "device_sn", "product_id", "metric_path", "metric_type", "metric_value",
	"statis_type", "granularity", "time", "start_time", "end_time", "object_ldn", "extra",
}

// resultInsertBatchSize 是单条多行 INSERT 的最大行数（issue #393）。
//
// pgx 扩展协议单条语句绑定参数上限 65535；结果表每行 14 个参数 →
// 理论上限 65535/14 ≈ 4681 行。取 4000 留余量，避免越界被驱动拒绝回滚。
// 「内置-全网-LTE」等无指标过滤的全网级聚合行数随数据增长会越过旧 ~4600 行上限，
// 不分批则整条 INSERT 失败、本轮结果丢失、任务标记失败。
const resultInsertBatchSize = 4000

// buildInsertResultsSQL 为一批结果行构建带 ON CONFLICT 的多行 INSERT SQL。
// 抽出便于单测断言列序/占位符数量；调用方保证 rows 已全局去重且非空。
func buildInsertResultsSQL(rows []ResultRow) (string, []any, error) {
	ib := storage.Psql.Insert("pm_adhoc_aggregation_results").Columns(resultInsertCols...)
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
				return "", nil, fmt.Errorf("marshal extra: %w", err)
			}
			extra = b
		}
		t := row.Time
		if t.IsZero() {
			t = row.EndTime
		}
		ib = ib.Values(
			row.TaskID, row.DeviceOUI, row.DeviceSN, nullableUUID(row.ProductID), row.MetricPath, row.MetricType, nullableMetricValue(row.MetricValue),
			stype, row.Granularity, t, row.StartTime, row.EndTime, ldn, extra,
		)
	}
	return ib.Suffix(onConflictResultsBusiness).ToSql()
}

func nullableMetricValue(value float64) any {
	if math.IsNaN(value) {
		return nil
	}
	return value
}

// InsertResults 批量写 pm_adhoc_aggregation_results。
// T-0194：改 ON CONFLICT DO UPDATE（值以最新一次聚合为准），入口按业务键同批去重防重复键报错。
// issue #393：分批写入，单批 ≤ resultInsertBatchSize 行（避免越过 pgx 65535 参数上限）；
// 全批同一事务，保证全部落库或全部回滚（不因分批产生部分写入）。
func (r *PgRepository) InsertResults(ctx context.Context, rows []ResultRow) error {
	if len(rows) == 0 {
		return nil
	}
	// 先全局去重：同一调用内同业务键只留最后一条。去重在分批前做，保证任意两个批次
	// 之间不会共享业务键（否则跨批次会触发 ON CONFLICT DO UPDATE 重复命中同行）。
	rows = dedupResultRows(rows)

	// pm_adhoc_aggregation_results 在时序库（TsPool），用 tsPool 写。
	// 全批分片落在同一事务：任一批失败整体回滚，杜绝部分落库。
	tx, err := r.tsPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("adhoc.InsertResults: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 提交成功后 rollback 为 no-op

	for start := 0; start < len(rows); start += resultInsertBatchSize {
		end := start + resultInsertBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		q, args, err := buildInsertResultsSQL(rows[start:end])
		if err != nil {
			return fmt.Errorf("adhoc.InsertResults: build SQL: %w", err)
		}
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return fmt.Errorf("adhoc.InsertResults: exec batch [%d,%d): %w", start, end, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("adhoc.InsertResults: commit: %w", err)
	}
	return nil
}

// ── 运行历史（pm_adhoc_task_runs，T-0186）─────────────────────────────────

var taskRunCols = []string{
	"id", "task_id", "run_seq", "granularity", "dimension",
	"window_start", "window_end", "status", "queued_at",
	"started_at", "finished_at", "error", "rows_total", "created_at",
}

func (r *PgRepository) NextRunSeq(ctx context.Context, taskID uuid.UUID) (int, error) {
	const q = `SELECT COALESCE(MAX(run_seq), 0) + 1 FROM pm_adhoc_task_runs WHERE task_id = $1`
	var seq int
	if err := r.pool.QueryRow(ctx, q, taskID).Scan(&seq); err != nil {
		return 0, fmt.Errorf("adhoc.NextRunSeq: %w", err)
	}
	return seq, nil
}

func (r *PgRepository) InsertRun(ctx context.Context, run TaskRun) (uuid.UUID, error) {
	q, args, err := storage.Psql.Insert("pm_adhoc_task_runs").
		Columns(
			"task_id", "run_seq", "granularity", "dimension",
			"window_start", "window_end", "status", "queued_at", "started_at",
		).
		Values(
			run.TaskID, run.RunSeq, run.Granularity, run.Dimension,
			nullablePtrTime(run.WindowStart), nullablePtrTime(run.WindowEnd),
			string(run.Status), nullablePtrTime(run.QueuedAt), run.StartedAt,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("adhoc.InsertRun: build SQL: %w", err)
	}
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("adhoc.InsertRun: insert: %w", err)
	}
	return id, nil
}

func (r *PgRepository) FinishRun(ctx context.Context, runID uuid.UUID, status Status, rowsTotal int, errMsg string) error {
	q, args, err := storage.Psql.Update("pm_adhoc_task_runs").
		Set("status", string(status)).
		Set("finished_at", time.Now()).
		Set("rows_total", rowsTotal).
		Set("error", errMsg).
		Where(sq.Eq{"id": runID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("adhoc.FinishRun: build SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("adhoc.FinishRun: exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// buildListRunsSQL 构建运行历史查询（倒序 started_at + 分页）。抽出便于单测断言排序/分页。
func buildListRunsSQL(taskID uuid.UUID, limit, offset int) (string, []any, error) {
	qb := storage.Psql.Select(taskRunCols...).
		From("pm_adhoc_task_runs").
		Where(sq.Eq{"task_id": taskID}).
		OrderBy("started_at DESC")
	if limit > 0 {
		qb = qb.Limit(uint64(limit))
	}
	if offset > 0 {
		qb = qb.Offset(uint64(offset))
	}
	return qb.ToSql()
}

func (r *PgRepository) ListRuns(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]TaskRun, error) {
	q, args, err := buildListRunsSQL(taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("adhoc.ListRuns: build SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("adhoc.ListRuns: query: %w", err)
	}
	defer rows.Close()
	out := make([]TaskRun, 0)
	for rows.Next() {
		var run TaskRun
		var status string
		if err := rows.Scan(
			&run.ID, &run.TaskID, &run.RunSeq, &run.Granularity, &run.Dimension,
			&run.WindowStart, &run.WindowEnd, &status, &run.QueuedAt,
			&run.StartedAt, &run.FinishedAt, &run.Error, &run.RowsTotal, &run.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("adhoc.ListRuns: scan: %w", err)
		}
		run.Status = Status(status)
		out = append(out, run)
	}
	return out, rows.Err()
}

// nullablePtrTime 把 nil 或零值 *time.Time 映射为 SQL NULL。
func nullablePtrTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return *t
}

func plannedEndValue(req CreateRequest) any {
	if req.Mode != ModeContinuous || req.IsBuiltin {
		return nil
	}
	if req.PlannedEndAtSet {
		return nullablePtrTime(req.PlannedEndAt)
	}
	if req.PlannedEndAt != nil && !req.PlannedEndAt.IsZero() {
		return *req.PlannedEndAt
	}
	return sq.Expr("NOW() + INTERVAL '30 days'")
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
	var windowStart, windowEnd, plannedEndAt *time.Time
	var dimension string
	var technology *string
	var isBuiltin bool
	var expireDays int
	var visibility string
	var status string
	var creator *string
	var objectLDNs []string // T-0193：白名单列，NULL → nil（不过滤）

	err := row.Scan(
		&t.ID, &t.Name, &subtype, &mode, &cronExpr,
		&deviceSNsJSON, &metricPaths, &granularities,
		&windowStart, &windowEnd, &dimension, &technology, &isBuiltin, &expireDays, &plannedEndAt, &visibility, &status, &t.Progress,
		&creator, &t.CreatedAt, &t.UpdatedAt, &objectLDNs,
	)
	if err != nil {
		return nil, err
	}
	t.ObjectLDNs = objectLDNs
	if technology != nil {
		t.Technology = *technology
	}
	t.IsBuiltin = isBuiltin
	t.ExpireDays = expireDays
	t.Visibility = normalizeVisibility(Visibility(visibility))
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
	t.PlannedEndAt = plannedEndAt
	if dimension != "" {
		t.Dimension = Dimension(dimension)
	} else {
		t.Dimension = DimensionDevice
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

func normalizeVisibility(v Visibility) Visibility {
	if v == VisibilityPublic {
		return VisibilityPublic
	}
	return VisibilityPrivate
}

// nullableStrSlice 把 nil / 空切片映射为 SQL NULL（T-0193 object_ldns 列）。
// 空数组与 NULL 都表"不过滤=全小区"，统一落 NULL 保持语义单一、便于 scanTask 读回 nil。
func nullableStrSlice(s []string) any {
	if len(s) == 0 {
		return nil
	}
	return s
}

// nullableTech 把空制式串映射为 SQL NULL（technology 列可空，约束 NULL or lte/nr/gsm）。
func nullableTech(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableTime 把零值 time 映射为 SQL NULL（T-0185）。
//
// continuous 任务不带固定窗口（handler 已清零 window_start/window_end），存 NULL 让
// aggregator 不加 time>=/time<= 边界、每次滚动捕获最新数据；若直插零值 time 会落成
// 0001-01-01（非 NULL），破坏开窗滚动语义。
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// nullableUUID 把 uuid.Nil 映射为 SQL NULL（product_id 列可空，T-0182-fix）。
// device / aggregate_group 维度结果无 product_id，落 NULL；product 维度落真实分组键。
func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
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
	// G7-Gap-9: 用 COALESCE(last_fire_at, created_at) 作"上次触发时刻"，
	// 老行 last_fire_at IS NULL 时退化到 created_at（首次启动会立即追到当前 cron 窗口）。
	// #528 P3：连带取 granularities/dimension，让调度器能读对应 (粒度,层级) 水位做追平上界。
	const q = `
SELECT id::text, cron_expr, COALESCE(last_fire_at, created_at),
       COALESCE(granularities, '{}'::text[]), COALESCE(dimension, '')
FROM pm_tasks
WHERE task_subtype = $1
  AND mode = 'continuous'
  AND status = 'scheduled'
  AND cron_expr IS NOT NULL
ORDER BY COALESCE(last_fire_at, created_at) ASC
LIMIT 1000`
	rows, err := r.pool.Query(ctx, q, TaskSubtype)
	if err != nil {
		return nil, fmt.Errorf("PgContinuousRepository.List: %w", err)
	}
	defer rows.Close()
	var out []ContinuousTask
	for rows.Next() {
		var ct ContinuousTask
		var dim string
		if err := rows.Scan(&ct.ID, &ct.CronExpr, &ct.LastFireAt, &ct.Granularities, &dim); err != nil {
			return nil, err
		}
		ct.Dimension = Dimension(dim)
		out = append(out, ct)
	}
	return out, rows.Err()
}

func (r *PgContinuousRepository) MarkPending(ctx context.Context, id ContinuousTaskID, fireAt time.Time) error {
	// G7-Gap-9: last_fire_at 推进到本次 cron 触发时间（不是 NOW）。Worker 跑完后回 scheduled，
	// 下一 sweep 会从该 fireAt 算 Next，若仍 < NOW 则继续追下一格，直至追平。
	const q = `
UPDATE pm_tasks
SET status = 'pending',
    updated_at = NOW(),
    last_fire_at = $3
WHERE id = $1::uuid
  AND task_subtype = $2
  AND status = 'scheduled'
  AND (is_builtin = true OR planned_end_at IS NULL OR planned_end_at > NOW())`
	tag, err := r.pool.Exec(ctx, q, id, TaskSubtype, fireAt)
	if err != nil {
		return fmt.Errorf("PgContinuousRepository.MarkPending: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 已被其它 worker 抢，无害
		return nil
	}
	return nil
}

// ── ContinuousScheduler 水位 gate 适配器（#528 P3）────────────────────────

// WatermarkGateAdapter 把 aggregator.WatermarkRepository 适配成调度器的 WatermarkGate。
// 调度器只需「给定 (粒度,层级) 当前水位桶起点 + 是否存在」，不暴露 aggregator 细节。
type WatermarkGateAdapter struct {
	reader WatermarkReader
}

// NewWatermarkGateAdapter 用一个 WatermarkReader（真实为 aggregator.WatermarkRepository）构造 gate。
func NewWatermarkGateAdapter(reader WatermarkReader) *WatermarkGateAdapter {
	return &WatermarkGateAdapter{reader: reader}
}

var _ WatermarkGate = (*WatermarkGateAdapter)(nil)

// CompletedBucketStart 返回 (粒度,层级) 当前完成水位桶起点；无水位 / 出错返 ok=false。
func (a *WatermarkGateAdapter) CompletedBucketStart(ctx context.Context, gran metrics.Granularity, level aggregator.WatermarkLevel) (time.Time, bool) {
	if a == nil || a.reader == nil {
		return time.Time{}, false
	}
	wm, err := a.reader.Get(ctx, gran, level)
	if err != nil {
		// ErrWatermarkNotFound 是正常状态（上游尚未卷完该粒度任何格）；其余错误也保守不放行。
		return time.Time{}, false
	}
	return wm.CompletedBucketStart, true
}
