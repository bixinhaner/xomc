package adhoc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 是 G7 adhoc 任务的 REST 入口。
type Handler struct {
	repo   Repository
	pool   *pgxpool.Pool // results 查询 + SSE backplane（直接 SQL，避免再加一层 repository）
	bus    event.EventBus
	logger *zap.Logger
}

// NewHandler 构造 Handler。
func NewHandler(repo Repository, pool *pgxpool.Pool, bus event.EventBus, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{repo: repo, pool: pool, bus: bus, logger: logger.Named("pm.adhoc.handler")}
}

// RegisterRoutes 把 6 个 REST 端点挂到 router group（不带 /pm 前缀，由调用方决定 group）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	adhoc := rg.Group("/pm/adhoc")
	{
		adhoc.POST("/tasks", h.Create)
		adhoc.GET("/tasks", h.List)
		adhoc.GET("/tasks/:id", h.Get)
		adhoc.DELETE("/tasks/:id", h.Cancel)
		adhoc.GET("/tasks/:id/results", h.Results)
		adhoc.GET("/tasks/:id/runs", h.Runs) // T-0186：运行历史
		adhoc.GET("/tasks/:id/progress", h.Progress) // SSE
	}
}

// ── 请求/响应 DTO ─────────────────────────────────────────────────────────

type createRequestDTO struct {
	Name     string `json:"name" binding:"required"`
	Mode     string `json:"mode" binding:"required,oneof=oneshot continuous"`
	CronExpr string `json:"cron_expr"`
	// T-0185：device_sns 仅在 device/aggregate_group 维度必填（向导期放宽）；
	// network/product/band/device_group 维度按制式全量聚合，不限设备，device_sns 可空。
	DeviceSNs     []string  `json:"device_sns"`
	MetricPaths   []string  `json:"metric_paths" binding:"required,min=1"`
	Granularities []string  `json:"granularities" binding:"required,min=1"`
	// T-0185：window 仅 oneshot 必填；continuous 不填 → 存 NULL 开窗滚动聚合（与内置任务同语义）。
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	// 维度：'device' (默认) / 'aggregate_group' (N 个 SN 临时组) / 'product' (按产品) /
	//       'band' (按频段，T-0183) / 'network' (全网，T-0184) / 'device_group' (设备组，T-0184)
	Dimension string `json:"dimension" binding:"omitempty,oneof=device aggregate_group product band network device_group"`
	// 制式：lte/nr/gsm，空=不限；建后不可改（T-0182）
	Technology string `json:"technology" binding:"omitempty,oneof=lte nr gsm"`
	// 内置任务标记（T-0182，由内置任务预置流程使用；普通用户建任务忽略）
	IsBuiltin bool `json:"is_builtin"`
	// 非持续型过期天数（T-0182，默认 60）
	ExpireDays int `json:"expire_days" binding:"omitempty,min=1"`
}

type taskResponseDTO struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Mode          string    `json:"mode"`
	CronExpr      *string   `json:"cron_expr,omitempty"`
	DeviceSNs     []string  `json:"device_sns"`
	MetricPaths   []string  `json:"metric_paths"`
	Granularities []string  `json:"granularities"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
	Dimension     string    `json:"dimension"`
	Technology    string    `json:"technology,omitempty"`
	IsBuiltin     bool      `json:"is_builtin"`
	ExpireDays    int       `json:"expire_days"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Creator       string    `json:"creator"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func taskToDTO(t *Task) taskResponseDTO {
	dim := string(t.Dimension)
	if dim == "" {
		dim = string(DimensionDevice)
	}
	return taskResponseDTO{
		ID:            t.ID.String(),
		Name:          t.Name,
		Mode:          string(t.Mode),
		CronExpr:      t.CronExpr,
		DeviceSNs:     t.DeviceSNs,
		MetricPaths:   t.MetricPaths,
		Granularities: t.Granularities,
		WindowStart:   t.WindowStart,
		WindowEnd:     t.WindowEnd,
		Dimension:     dim,
		Technology:    t.Technology,
		IsBuiltin:     t.IsBuiltin,
		ExpireDays:    t.ExpireDays,
		Status:        string(t.Status),
		Progress:      t.Progress,
		Creator:       t.Creator,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────

// Create POST /pm/adhoc/tasks
func (h *Handler) Create(c *gin.Context) {
	var req createRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// 单粒度（设计 §2.4/§2.6 单选一个；要别的粒度另建任务）。保留数组结构不动 executor 循环。
	if len(req.Granularities) != 1 {
		response.Fail(c, http.StatusBadRequest, "granularities must contain exactly one value (single granularity per task)")
		return
	}
	dim := Dimension(req.Dimension)
	if dim == "" {
		dim = DimensionDevice
	}
	// T-0185：device_sns 仅 device/aggregate_group（自选设备）维度必填；其余维度按制式全量聚合。
	if (dim == DimensionDevice || dim == DimensionAggregateGroup) && len(req.DeviceSNs) == 0 {
		response.Fail(c, http.StatusBadRequest, "device_sns is required for device/aggregate_group dimension")
		return
	}
	// T-0185：oneshot 必须给有效时间窗（end > start）；continuous 留空 → NULL 开窗滚动聚合。
	if Mode(req.Mode) == ModeOneshot {
		if !req.WindowEnd.After(req.WindowStart) {
			response.Fail(c, http.StatusBadRequest, "window_end must be after window_start for oneshot task")
			return
		}
	} else {
		// continuous：忽略传入窗口，强制开窗（与内置任务一致，每次滚动聚合最新可用桶）。
		req.WindowStart = time.Time{}
		req.WindowEnd = time.Time{}
	}
	// 制式过滤：建任务拒跨制式 —— 选定制式后，范围内的设备必须全部属于该制式（设计 §2.5）。
	if req.Technology != "" && len(req.DeviceSNs) > 0 {
		if err := h.rejectCrossTechnology(c.Request.Context(), req.Technology, req.DeviceSNs); err != nil {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	// T-0185：continuous 任务的 cron 由粒度自动派生（向导不暴露 cron 字段）；显式传 cron 则尊重。
	cronExpr := req.CronExpr
	if Mode(req.Mode) == ModeContinuous && cronExpr == "" {
		cronExpr = cronForGranularity(req.Granularities[0])
	}
	var cronPtr *string
	if cronExpr != "" {
		cronPtr = &cronExpr
	}
	creator := extractCreator(c)
	id, err := h.repo.Create(c.Request.Context(), CreateRequest{
		Name:          req.Name,
		Mode:          Mode(req.Mode),
		CronExpr:      cronPtr,
		DeviceSNs:     req.DeviceSNs,
		MetricPaths:   req.MetricPaths,
		Granularities: req.Granularities,
		WindowStart:   req.WindowStart,
		WindowEnd:     req.WindowEnd,
		Dimension:     dim,
		Technology:    req.Technology,
		IsBuiltin:     req.IsBuiltin,
		ExpireDays:    req.ExpireDays,
		Creator:       creator,
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{"id": id.String()})
}

// cronForGranularity 把粒度映射成 continuous 任务的滚动触发 cron（5 字段：m h dom mon dow）。
//
// 各档都在桶边界之后留几分钟，等下级数据落齐再聚合（hourly 对齐内置任务的 '5 * * * *'）。
// 未知粒度兜底按小时滚动。
func cronForGranularity(g string) string {
	switch g {
	case "15min":
		return "5,20,35,50 * * * *" // 每刻钟过 5 分
	case "hourly":
		return "5 * * * *" // 每小时第 5 分
	case "daily":
		return "10 0 * * *" // 每天 00:10
	case "weekly":
		return "15 0 * * 1" // 每周一 00:15
	case "monthly":
		return "20 0 1 * *" // 每月 1 号 00:20
	default:
		return "5 * * * *"
	}
}

// rejectCrossTechnology 校验 deviceSNs 全部属于指定制式 tech（lte/nr/gsm）。
//
// 任一设备制式不符（或在 devices 表查不到 → 无法确认制式）即返错，实现"建任务拒跨制式"（设计 §2.5）。
// devices.technology 为小写 lte/nr/gsm。
func (h *Handler) rejectCrossTechnology(ctx context.Context, tech string, deviceSNs []string) error {
	const q = `
SELECT serial_number, technology
FROM devices
WHERE serial_number = ANY($1)`
	rows, err := h.pool.Query(ctx, q, deviceSNs)
	if err != nil {
		return fmt.Errorf("verify device technology: %w", err)
	}
	defer rows.Close()
	found := make(map[string]string, len(deviceSNs))
	for rows.Next() {
		var sn string
		var t *string
		if err := rows.Scan(&sn, &t); err != nil {
			return fmt.Errorf("verify device technology: %w", err)
		}
		if t != nil {
			found[sn] = *t
		} else {
			found[sn] = ""
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("verify device technology: %w", err)
	}
	for _, sn := range deviceSNs {
		dt, ok := found[sn]
		if !ok {
			return fmt.Errorf("device %s not found; cannot create %s task with unknown-technology device", sn, tech)
		}
		if dt != tech {
			return fmt.Errorf("device %s is technology %q, mismatch task technology %q (cross-technology not allowed)", sn, dt, tech)
		}
	}
	return nil
}

// List GET /pm/adhoc/tasks?mode=&status=&limit=&offset=&all=true
//
// T-0164 收尾 G7-Gap-7：默认按 creator=current_user 过滤（"我的任务"），
// admin 角色传 ?all=true 可看全部任务（运维 / 审计场景）。
func (h *Handler) List(c *gin.Context) {
	filter := ListFilter{Limit: 50}
	if v := c.Query("mode"); v != "" {
		m := Mode(v)
		filter.Mode = &m
	}
	if v := c.Query("status"); v != "" {
		s := Status(v)
		filter.Status = &s
	}

	// T-0164 收尾 G7-Gap-7：creator 过滤
	// - 默认按当前用户过滤（"我的任务"）
	// - admin 角色传 ?all=true 可看全部
	// - 显式传 ?creator=xxx 时尊重（向后兼容老 client + 运维筛查特定用户场景）
	// T-0184：内置任务过滤（前端分"内置区"/"自建区"）。
	//   ?is_builtin=true  → 只看内置 12 个预置任务（全用户可见，不按 creator 过滤）
	//   ?is_builtin=false → 只看自建任务（仍按 creator 默认过滤）
	var builtinOnly bool
	if v := c.Query("is_builtin"); v != "" {
		b := v == "true"
		filter.IsBuiltin = &b
		builtinOnly = b
	}

	currentUser := extractCreator(c)
	all := c.Query("all") == "true"
	switch {
	case builtinOnly:
		// 内置任务无 per-user 归属，全用户共享可见 → 不按 creator 过滤
		filter.Creator = ""
	case c.Query("creator") != "":
		filter.Creator = c.Query("creator")
	case all && isAdmin(c):
		// admin + 显式 ?all=true → 不过滤
		filter.Creator = ""
	default:
		// 默认按当前用户过滤
		filter.Creator = currentUser
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			filter.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			filter.Offset = n
		}
	}
	tasks, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	dtos := make([]taskResponseDTO, 0, len(tasks))
	for i := range tasks {
		dtos = append(dtos, taskToDTO(&tasks[i]))
	}
	response.OK(c, gin.H{"items": dtos, "total": len(dtos)})
}

// Get GET /pm/adhoc/tasks/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, taskToDTO(t))
}

// Cancel DELETE /pm/adhoc/tasks/:id
func (h *Handler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Cancel(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Fail(c, http.StatusNotFound, "not found")
		case errors.Is(err, ErrTerminalState):
			response.Fail(c, http.StatusConflict, "task already in terminal state")
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, gin.H{"id": id.String(), "status": string(StatusCanceled)})
}

// resultsFilter 是 Results 端点的可选过滤项（均为原始 query 字符串，空串=不过滤）。
type resultsFilter struct {
	DeviceSN    string
	MetricPath  string
	Granularity string
	StartTime   string // RFC3339；非法/空则忽略
	EndTime     string // RFC3339；非法/空则忽略
}

// buildResultsQuery 纯函数：拼 adhoc results 查询 SQL + 占位参数。
// 抽出来便于单测（带/不带大时间段两路）；时间段非法值容错忽略而非报错。
func buildResultsQuery(taskID uuid.UUID, f resultsFilter, limit, offset int) (string, []any) {
	q := `
SELECT id, task_id, device_oui, device_sn, product_id, metric_path, metric_type, metric_value,
       statis_type, granularity, time, start_time, end_time, ingest_time, object_ldn, extra
FROM pm_adhoc_aggregation_results
WHERE task_id = $1`
	args := []any{taskID}
	pos := 2
	if f.DeviceSN != "" {
		q += fmt.Sprintf(" AND device_sn = $%d", pos)
		args = append(args, f.DeviceSN)
		pos++
	}
	if f.MetricPath != "" {
		q += fmt.Sprintf(" AND metric_path = $%d", pos)
		args = append(args, f.MetricPath)
		pos++
	}
	if f.Granularity != "" {
		q += fmt.Sprintf(" AND granularity = $%d", pos)
		args = append(args, f.Granularity)
		pos++
	}
	// 可选大时间段过滤（页签1 仪表盘大时间段驱动取数）：start_time/end_time 用 RFC3339 解析，
	// 命中则按 time 列窗口过滤，与现有 ORDER BY time DESC 同列；非法值忽略（容错而非 400）。
	if f.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, f.StartTime); err == nil {
			q += fmt.Sprintf(" AND time >= $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	if f.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, f.EndTime); err == nil {
			q += fmt.Sprintf(" AND time <= $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	q += fmt.Sprintf(" ORDER BY time DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)
	return q, args
}

// Results GET /pm/adhoc/tasks/:id/results?device_sn=&metric_path=&granularity=&start_time=&end_time=&limit=&offset=
func (h *Handler) Results(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 任务存在校验
	if _, err := h.repo.Get(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10000 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	// 直接 SQL 查 — adhoc results 只读用例，不值得再拆 repo
	q, args := buildResultsQuery(id, resultsFilter{
		DeviceSN:    c.Query("device_sn"),
		MetricPath:  c.Query("metric_path"),
		Granularity: c.Query("granularity"),
		StartTime:   c.Query("start_time"),
		EndTime:     c.Query("end_time"),
	}, limit, offset)

	rows, err := h.pool.Query(c.Request.Context(), q, args...)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type resultDTO struct {
		ID          string    `json:"id"`
		TaskID      string    `json:"task_id"`
		DeviceOUI   string    `json:"device_oui"`
		DeviceSN    string    `json:"device_sn"`
		// product 维度结果的分组键（T-0182-fix）；device/aggregate_group 维度为空。
		ProductID   string    `json:"product_id,omitempty"`
		MetricPath  string    `json:"metric_path"`
		// KPI 行 metric_path 是 K 编号；display_name 为按编号回填的友好名（PLMN 级带标记）。counter 行 = metric_path。
		DisplayName string    `json:"display_name,omitempty"`
		MetricType  string    `json:"metric_type"`
		MetricValue float64   `json:"metric_value"`
		StatisType  *string   `json:"statis_type,omitempty"`
		Granularity string    `json:"granularity"`
		Time        time.Time `json:"time"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
		IngestTime  time.Time `json:"ingest_time"`
		ObjectLDN   *string   `json:"object_ldn,omitempty"`
	}
	items := make([]resultDTO, 0)
	for rows.Next() {
		var dto resultDTO
		var resultID, taskID uuid.UUID
		var productID *uuid.UUID // product_id 列可空（仅 product 维度有值）
		var extraBytes []byte
		if err := rows.Scan(
			&resultID, &taskID, &dto.DeviceOUI, &dto.DeviceSN, &productID, &dto.MetricPath,
			&dto.MetricType, &dto.MetricValue, &dto.StatisType, &dto.Granularity,
			&dto.Time, &dto.StartTime, &dto.EndTime, &dto.IngestTime, &dto.ObjectLDN, &extraBytes,
		); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		dto.ID = resultID.String()
		dto.TaskID = taskID.String()
		if productID != nil && *productID != uuid.Nil {
			dto.ProductID = productID.String()
		}
		items = append(items, dto)
	}

	// 回填 display_name：KPI 行 metric_path 是 K 编号，按编号查指标库取友好名；counter 行 = metric_path。
	codeSet := make(map[string]struct{})
	for i := range items {
		if items[i].MetricType == "kpi" && items[i].MetricPath != "" {
			codeSet[items[i].MetricPath] = struct{}{}
		} else {
			items[i].DisplayName = items[i].MetricPath
		}
	}
	if len(codeSet) > 0 {
		codes := make([]string, 0, len(codeSet))
		for code := range codeSet {
			codes = append(codes, code)
		}
		nameByCode := h.lookupIndicatorNames(c.Request.Context(), codes)
		for i := range items {
			if items[i].MetricType != "kpi" {
				continue
			}
			if name, ok := nameByCode[items[i].MetricPath]; ok && name != "" {
				items[i].DisplayName = name
			} else {
				items[i].DisplayName = items[i].MetricPath
			}
		}
	}

	response.OK(c, gin.H{"items": items, "total": len(items)})
}

// lookupIndicatorNames 按编号集合一次性查三张指标表，返回 code → cn_name（缺则 en_name）。
// K 编号在 perf_indicators_{enb,gnb,gsm} 三表全局唯一，一次 UNION 即可覆盖（与 aggregator 查询层一致）。
func (h *Handler) lookupIndicatorNames(ctx context.Context, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	const tmpl = `
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_enb  WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_gnb  WHERE id = ANY($1)
UNION ALL
SELECT id, COALESCE(NULLIF(cn_name, ''), en_name) AS display_name FROM perf_indicators_gsm  WHERE id = ANY($1)`
	rows, err := h.pool.Query(ctx, tmpl, codes)
	if err != nil {
		h.logger.Warn("backfill adhoc display names query failed; fall back to codes", zap.Error(err))
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			h.logger.Warn("backfill adhoc display names scan failed", zap.Error(err))
			return out
		}
		out[id] = name
	}
	return out
}

// taskRunDTO 是 pm_adhoc_task_runs 一行的响应体（T-0186，snake_case 对齐前端 mapper）。
type taskRunDTO struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	RunSeq      int        `json:"run_seq"`
	Granularity string     `json:"granularity"`
	Dimension   string     `json:"dimension"`
	WindowStart *time.Time `json:"window_start,omitempty"`
	WindowEnd   *time.Time `json:"window_end,omitempty"`
	Status      string     `json:"status"`
	QueuedAt    *time.Time `json:"queued_at,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	RowsTotal   int        `json:"rows_total"`
}

func taskRunToDTO(r *TaskRun) taskRunDTO {
	return taskRunDTO{
		ID:          r.ID.String(),
		TaskID:      r.TaskID.String(),
		RunSeq:      r.RunSeq,
		Granularity: r.Granularity,
		Dimension:   r.Dimension,
		WindowStart: r.WindowStart,
		WindowEnd:   r.WindowEnd,
		Status:      string(r.Status),
		QueuedAt:    r.QueuedAt,
		StartedAt:   r.StartedAt,
		FinishedAt:  r.FinishedAt,
		Error:       r.Error,
		RowsTotal:   r.RowsTotal,
	}
}

// Runs GET /pm/adhoc/tasks/:id/runs?limit=&offset=
//
// 返回该任务运行历史，倒序 started_at，支持分页（T-0186）。
func (h *Handler) Runs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 任务存在校验
	if _, err := h.repo.Get(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	runs, err := h.repo.ListRuns(c.Request.Context(), id, limit, offset)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	items := make([]taskRunDTO, 0, len(runs))
	for i := range runs {
		items = append(items, taskRunToDTO(&runs[i]))
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

// Progress GET /pm/adhoc/tasks/:id/progress（SSE）
//
// 订阅 pm.adhoc.progress 与 pm.adhoc.completed 主题，过滤匹配 task_id 的事件流给客户端。
// 客户端 EventSource 'progress'/'completed' 事件名分别接收。
func (h *Handler) Progress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if h.bus == nil {
		response.Fail(c, http.StatusServiceUnavailable, "event bus not wired")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx 不缓冲

	taskID := id.String()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	makeHandler := func(eventName string) event.EventHandler {
		return func(ctx context.Context, evt event.Event) error {
			var payload map[string]any
			if err := evt.DecodePayload(&payload); err != nil {
				return nil // 忽略解析错误，不阻塞订阅链
			}
			tid, _ := payload["task_id"].(string)
			if tid != taskID {
				return nil
			}
			data, _ := json.Marshal(payload)
			h.writeSSE(c.Writer.(io.Writer), eventName, data)
			flusher.Flush()
			return nil
		}
	}

	subProgress, err := h.bus.Subscribe(SubjectProgress, makeHandler("progress"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer subProgress.Unsubscribe()

	subCompleted, err := h.bus.Subscribe(SubjectCompleted, makeHandler("completed"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer subCompleted.Unsubscribe()

	// 等客户端断开。SSE 心跳每 30s 发个 comment 防代理超时。
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			h.writeSSE(c.Writer.(io.Writer), "", []byte(": keep-alive"))
			flusher.Flush()
		}
	}
}

func (h *Handler) writeSSE(w io.Writer, eventName string, data []byte) {
	if eventName != "" {
		_, _ = w.Write([]byte("event: " + eventName + "\n"))
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
}

// extractCreator 从 gin context 取登录用户名（如有 middleware 注入）。否则用 "anonymous"。
func extractCreator(c *gin.Context) string {
	// 中间件可能在 c.Set("user", User{...}) 或 c.Set("username", "xxx")
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "anonymous"
}

// isAdmin 判断当前用户是否 admin / super_admin（T-0164 收尾 G7-Gap-7 用，决定 ?all=true 是否生效）。
//
// admin.AuthMiddleware 注入的 context key（roles / is_super_admin / user role）；
// 任一为真即视为有权限看全部任务。
func isAdmin(c *gin.Context) bool {
	if v, ok := c.Get("is_super_admin"); ok {
		if b, ok := v.(bool); ok && b {
			return true
		}
	}
	if v, ok := c.Get("roles"); ok {
		if roles, ok := v.([]string); ok {
			for _, r := range roles {
				if r == "admin" || r == "super_admin" {
					return true
				}
			}
		}
	}
	return false
}
