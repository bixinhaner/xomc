package adhoc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Handler 是 G7 adhoc 任务的 REST 入口。
type Handler struct {
	repo                   Repository
	pool                   *pgxpool.Pool // results 查询（直接 SQL，避免再加一层 repository）
	hub                    *ProgressHub
	enabledMetricValidator *EnabledMetricSelectionService
	logger                 *zap.Logger
}

// NewHandler 构造 Handler。
func NewHandler(repo Repository, pool *pgxpool.Pool, hub *ProgressHub, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{repo: repo, pool: pool, hub: hub, logger: logger.Named("pm.adhoc.handler")}
}

func (h *Handler) WithEnabledMetricSelectionService(svc *EnabledMetricSelectionService) *Handler {
	h.enabledMetricValidator = svc
	return h
}

// RegisterRoutes 把 6 个 REST 端点挂到 router group（不带 /pm 前缀，由调用方决定 group）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	adhoc := rg.Group("/pm/adhoc")
	{
		adhoc.POST("/tasks", h.Create)
		adhoc.GET("/tasks", h.List)
		adhoc.GET("/tasks/:id", h.Get)
		adhoc.PATCH("/tasks/:id", h.Update) // T-0194：编辑任务定义
		adhoc.DELETE("/tasks/:id", h.Cancel)
		adhoc.POST("/tasks/:id/resume", h.Resume)       // #674：恢复已取消任务
		adhoc.DELETE("/tasks/:id/definition", h.Delete) // #392：硬删终态自建任务定义行
		adhoc.GET("/tasks/:id/results", h.Results)
		adhoc.GET("/tasks/:id/filter-options", h.FilterOptions) // PM-DASH-DIMFILTER：按维度列出可筛子集选项
		adhoc.GET("/tasks/:id/runs", h.Runs)                    // T-0186：运行历史
		adhoc.GET("/tasks/:id/progress", h.Progress)            // SSE
	}
}

// ── 请求/响应 DTO ─────────────────────────────────────────────────────────

type createRequestDTO struct {
	Name     string `json:"name" binding:"required"`
	Mode     string `json:"mode" binding:"required,oneof=oneshot continuous"`
	CronExpr string `json:"cron_expr"`
	// T-0185：device_sns 仅在 device/aggregate_group 维度必填（向导期放宽）；
	// network/product/band/device_group 维度按制式全量聚合，不限设备，device_sns 可空。
	DeviceSNs     []string `json:"device_sns"`
	MetricPaths   []string `json:"metric_paths" binding:"required,min=1"`
	Granularities []string `json:"granularities" binding:"required,min=1"`
	// T-0193：小区/PLMN 白名单（完整 object_ldn 字符串）。可选，不传/空 = 全小区（向后兼容）。
	// 仅 device/aggregate_group 维度生效；其他维度忽略（不落库、不报错）。
	ObjectLDNs []string `json:"object_ldns"`
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
	// visibility：private（默认，仅创建者/超管可见可操作）/ public（登录用户可见可操作）
	Visibility string `json:"visibility" binding:"omitempty,oneof=private public"`
}

type taskResponseDTO struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Mode          string    `json:"mode"`
	CronExpr      *string   `json:"cron_expr,omitempty"`
	DeviceSNs     []string  `json:"device_sns"`
	MetricPaths   []string  `json:"metric_paths"`
	Granularities []string  `json:"granularities"`
	ObjectLDNs    []string  `json:"object_ldns"` // T-0193：小区/PLMN 白名单回吐（空=全小区）
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
	Dimension     string    `json:"dimension"`
	Technology    string    `json:"technology,omitempty"`
	IsBuiltin     bool      `json:"is_builtin"`
	ExpireDays    int       `json:"expire_days"`
	Visibility    string    `json:"visibility"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Creator       string    `json:"creator"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func taskToDTO(ctx context.Context, t *Task) taskResponseDTO {
	dimVal := t.Dimension
	if dimVal == "" {
		dimVal = DimensionDevice
	}
	dim := string(dimVal)
	// 内置任务名按 locale 本地化（en 组装 Built-in-<维度>-<制式>，zh 用库里原值）；自建任务原样。
	name := localizeTaskName(appcontext.GetLocale(ctx), t.Name, t.IsBuiltin, dimVal, t.Technology)
	return taskResponseDTO{
		ID:            t.ID.String(),
		Name:          name,
		Mode:          string(t.Mode),
		CronExpr:      t.CronExpr,
		DeviceSNs:     t.DeviceSNs,
		MetricPaths:   t.MetricPaths,
		Granularities: t.Granularities,
		ObjectLDNs:    t.ObjectLDNs,
		WindowStart:   t.WindowStart,
		WindowEnd:     t.WindowEnd,
		Dimension:     dim,
		Technology:    t.Technology,
		IsBuiltin:     t.IsBuiltin,
		ExpireDays:    t.ExpireDays,
		Visibility:    string(normalizeVisibility(t.Visibility)),
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
	// #669：粒度前置守门——15min 已整组下线（详见 unsupportedGranularity 注释）。
	// 保留 dim 入参以便日后扩展新的（粒度,维度）限制。
	if msg := unsupportedGranularity(req.Granularities[0], dim); msg != "" {
		response.Fail(c, http.StatusBadRequest, msg)
		return
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
	if err := h.enabledMetricValidator.ValidateTechnology(c.Request.Context(), req.Technology, req.MetricPaths); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
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
	// T-0193：小区/PLMN 白名单仅在 device/aggregate_group（自选设备）维度承载；
	// 其他维度无单设备小区语义，忽略传入值（不落库、不报错，保持简单）。
	objectLDNs := req.ObjectLDNs
	if dim != DimensionDevice && dim != DimensionAggregateGroup {
		objectLDNs = nil
	}
	creator := extractCreator(c)
	id, err := h.repo.Create(c.Request.Context(), CreateRequest{
		Name:          req.Name,
		Mode:          Mode(req.Mode),
		CronExpr:      cronPtr,
		DeviceSNs:     req.DeviceSNs,
		MetricPaths:   req.MetricPaths,
		Granularities: req.Granularities,
		ObjectLDNs:    objectLDNs,
		WindowStart:   req.WindowStart,
		WindowEnd:     req.WindowEnd,
		Dimension:     dim,
		Technology:    req.Technology,
		IsBuiltin:     req.IsBuiltin,
		ExpireDays:    req.ExpireDays,
		Visibility:    Visibility(req.Visibility),
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
//
// #669：15min 档已下线（自定义聚合任务最细粒度限定 hourly，详见 unsupportedGranularity），
// 这里不再保留 15min 分支——上游守门已拒，不可能走到。
func cronForGranularity(g string) string {
	switch g {
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

// unsupportedGranularity 校验粒度是否被自定义聚合任务支持。
//
// #669（取代旧 #363 特例）：15min 档整组下线，无论维度都不再支持。
// 前因：原 #363 只在「device_group + 15min」做了前置拦截（其他维度 + 15min 一律放行）；
// 但实际「continuous + 15min」存在「创建放行、调度器水位闸门永久拒绝放行」的半成品行为
// （pm_completion_watermarks 表只覆盖 hourly/daily/weekly/monthly 四档，无 15min 行）。
// 权衡过「补水位接通」与「缩范围下线」两条路（见 #669），采纳缩范围：自定义聚合任务
// 最细粒度限定 hourly，15min 数据走「指标查询 / 数据提取」即可，不再走流式聚合管线。
//
// 命中返回面向用户的友好错误消息；不命中返回空串。dim 入参保留是为日后再有新组合限制时
// 扩展用（当前所有维度对 15min 一视同仁，dim 未参与判定）。
func unsupportedGranularity(granularity string, _ Dimension) string {
	if granularity == "15min" {
		return "15min granularity is no longer supported for ad-hoc aggregation tasks; use hourly or coarser, or query 15min raw data via metric-query / data-extraction"
	}
	return ""
}

// rejectCrossTechnology 校验 deviceSNs 全部属于指定制式 tech（lte/nr/gsm）。
//
// 任一设备制式不符（或在影子表查不到 → 无法确认制式）即返错，实现"建任务拒跨制式"（设计 §2.5）。
// device_dim.technology 为小写 lte/nr/gsm。
// h.pool 是 TsPool；devices 改读本库影子表 device_dim（跨库分离）。
func (h *Handler) rejectCrossTechnology(ctx context.Context, tech string, deviceSNs []string) error {
	const q = `
SELECT serial_number, technology
FROM device_dim
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

// List GET /pm/adhoc/tasks?mode=&status=&limit=&offset=
//
// 默认返回：内置任务 + 当前用户 private 自定义任务 + 所有 public 自定义任务。
// 超管（source='builtIn'）由后端登录态自动识别，可看全部任务（运维 / 审计场景）。
func (h *Handler) List(c *gin.Context) {
	currentUser := extractCreator(c)
	filter := ListFilter{Limit: 50, CurrentUser: currentUser}
	if v := c.Query("mode"); v != "" {
		m := Mode(v)
		filter.Mode = &m
	}
	if v := c.Query("status"); v != "" {
		s := Status(v)
		filter.Status = &s
	}

	// T-0184：内置任务过滤（前端分"内置区"/"自建区"）。
	var builtinOnly bool
	if v := c.Query("is_builtin"); v != "" {
		b := v == "true"
		filter.IsBuiltin = &b
		builtinOnly = b
	}

	filter.IncludeAll = isAdmin(c)
	if !builtinOnly && c.Query("creator") != "" {
		filter.Creator = c.Query("creator")
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
		dtos = append(dtos, taskToDTO(c.Request.Context(), &tasks[i]))
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
	if !canViewTask(t, extractCreator(c), isAdmin(c)) {
		response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
		return
	}
	response.OK(c, taskToDTO(c.Request.Context(), t))
}

// updateRequestDTO 是 PATCH /pm/adhoc/tasks/:id 的入参（T-0194）。
//
// 字段集与编辑能力对齐：自建任务可改 name/device_sns/metric_paths/granularities/object_ldns/window；
// 内置任务只取 metric_paths（其余字段服务端忽略）。mode/technology/dimension/is_builtin/expire_days 不在此结构体，不可改。
type updateRequestDTO struct {
	Name          string    `json:"name"`
	DeviceSNs     []string  `json:"device_sns"`
	MetricPaths   []string  `json:"metric_paths" binding:"required,min=1"`
	Granularities []string  `json:"granularities"`
	ObjectLDNs    []string  `json:"object_ldns"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
	Visibility    string    `json:"visibility" binding:"omitempty,oneof=private public"`
}

// Update PATCH /pm/adhoc/tasks/:id
//
// 编辑任务定义（T-0194）。先取既有任务确定 is_builtin / mode / technology（结构性字段不可改，
// 用既有值做校验基准）：
//   - 内置任务：只更新 metric_paths，其余传入字段忽略，跳过结构性校验（设备/粒度/时窗不变）。
//   - 自建任务：复用创建校验——单粒度、device/aggregate_group 维度设备非空、oneshot 时窗 end>start、改设备后拒跨制式。
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// 取既有任务：决定守门口径（is_builtin），并以既有结构性字段（mode/technology/dimension）做校验基准。
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// 自建任务权限：private 仅创建者或超管；public 允许其他登录用户编辑。
	if !existing.IsBuiltin {
		if !canOperate(existing, extractCreator(c), isAdmin(c)) {
			response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
			return
		}
	}

	upd := UpdateRequest{
		IsBuiltin:   existing.IsBuiltin,
		Mode:        existing.Mode,
		Dimension:   existing.Dimension,
		MetricPaths: req.MetricPaths,
	}

	if err := h.enabledMetricValidator.ValidateTechnology(c.Request.Context(), existing.Technology, req.MetricPaths); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if !existing.IsBuiltin {
		// 自建任务：复用创建校验。
		// 单粒度（设计 §2.4/§2.6）。
		if len(req.Granularities) != 1 {
			response.Fail(c, http.StatusBadRequest, "granularities must contain exactly one value (single granularity per task)")
			return
		}
		// device_sns 仅 device/aggregate_group 维度必填（沿用既有维度，不可改）。
		dim := existing.Dimension
		if dim == "" {
			dim = DimensionDevice
		}
		// #669：编辑自建任务时同样守门粒度——15min 已整组下线，不允许把任意维度的任务粒度改回 15min。
		if msg := unsupportedGranularity(req.Granularities[0], dim); msg != "" {
			response.Fail(c, http.StatusBadRequest, msg)
			return
		}
		if (dim == DimensionDevice || dim == DimensionAggregateGroup) && len(req.DeviceSNs) == 0 {
			response.Fail(c, http.StatusBadRequest, "device_sns is required for device/aggregate_group dimension")
			return
		}
		// oneshot 时窗 end>start；continuous 强制清窗（开窗滚动，与创建一致）。
		if existing.Mode == ModeOneshot {
			if !req.WindowEnd.After(req.WindowStart) {
				response.Fail(c, http.StatusBadRequest, "window_end must be after window_start for oneshot task")
				return
			}
		} else {
			req.WindowStart = time.Time{}
			req.WindowEnd = time.Time{}
		}
		// 改设备后仍拒跨制式（按既有任务 technology 校验）。
		if existing.Technology != "" && len(req.DeviceSNs) > 0 {
			if err := h.rejectCrossTechnology(c.Request.Context(), existing.Technology, req.DeviceSNs); err != nil {
				response.Fail(c, http.StatusBadRequest, err.Error())
				return
			}
		}
		// object_ldns 仅 device/aggregate_group 维度承载（与创建一致）。
		objectLDNs := req.ObjectLDNs
		if dim != DimensionDevice && dim != DimensionAggregateGroup {
			objectLDNs = nil
		}
		upd.Name = req.Name
		upd.DeviceSNs = req.DeviceSNs
		upd.Granularities = req.Granularities
		upd.Visibility = existing.Visibility
		if req.Visibility != "" {
			upd.Visibility = Visibility(req.Visibility)
		}
		if existing.Mode == ModeContinuous {
			cronExpr := cronForGranularity(req.Granularities[0])
			upd.CronExpr = &cronExpr
			upd.ResetCursor = !sameFirstGranularity(existing.Granularities, req.Granularities)
		}
		upd.ObjectLDNs = objectLDNs
		upd.WindowStart = req.WindowStart
		upd.WindowEnd = req.WindowEnd
		upd.RequeueTerminal = existing.Mode == ModeOneshot && oneshotExecutionInputsChanged(existing, req, objectLDNs)
	}

	if err := h.repo.Update(c.Request.Context(), id, upd); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func sameFirstGranularity(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	return a[0] == b[0]
}

func oneshotExecutionInputsChanged(existing *Task, req updateRequestDTO, objectLDNs []string) bool {
	return !slices.Equal(existing.DeviceSNs, req.DeviceSNs) ||
		!slices.Equal(existing.MetricPaths, req.MetricPaths) ||
		!slices.Equal(existing.Granularities, req.Granularities) ||
		!slices.Equal(existing.ObjectLDNs, objectLDNs) ||
		!existing.WindowStart.Equal(req.WindowStart) ||
		!existing.WindowEnd.Equal(req.WindowEnd)
}

// Cancel DELETE /pm/adhoc/tasks/:id
func (h *Handler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 取消会中断正在执行/排期的任务，仍限定创建者或超管。
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !existing.IsBuiltin {
		if !canCancelTask(existing, extractCreator(c), isAdmin(c)) {
			response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
			return
		}
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

// Resume POST /pm/adhoc/tasks/:id/resume
//
// 恢复已取消的 adhoc 任务（#674）：
//   - continuous → scheduled（让 ContinuousScheduler 下次 sweep 推 pending）
//   - oneshot → pending（让 worker 直接捞）
//   - 非 canceled 状态调用返回 409 Conflict。
func (h *Handler) Resume(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 自建任务权限：private 仅创建者或超管；public 允许其他登录用户恢复。
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !existing.IsBuiltin {
		if !canOperate(existing, extractCreator(c), isAdmin(c)) {
			response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
			return
		}
	}
	newStatus, err := h.repo.Resume(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Fail(c, http.StatusNotFound, "not found")
		case errors.Is(err, ErrNotCanceled):
			response.Fail(c, http.StatusConflict, "task is not canceled, cannot resume")
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, gin.H{"id": id.String(), "status": string(newStatus)})
}

// Delete DELETE /pm/adhoc/tasks/:id/definition
//
// 硬删终态（succeeded/failed/canceled）自建（is_builtin=false）adhoc 任务的定义行（#392）。
// 只删 pm_tasks 定义行，结果数据交 TimescaleDB retention 自然过期（不级联删）。
//   - 非终态（pending/running/scheduled）任务返 409（仍活跃，应走「取消」）。
//   - 内置任务返 403（永不可删）。
//   - 行不存在返 404。
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 自建任务权限：private 仅创建者或超管；public 允许其他登录用户删除。
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// 内置任务不做归属权校验（由 repo.Delete 的 is_builtin 守门拦截）。
	if !existing.IsBuiltin {
		if !canOperate(existing, extractCreator(c), isAdmin(c)) {
			response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
			return
		}
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Fail(c, http.StatusNotFound, "not found")
		case errors.Is(err, ErrBuiltinNotDeletable):
			response.Fail(c, http.StatusForbidden, "builtin task cannot be deleted")
		case errors.Is(err, ErrNotTerminal):
			response.Fail(c, http.StatusConflict, "task not in terminal state, cancel it first")
		default:
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		}
		return
	}
	response.OK(c, gin.H{"id": id.String(), "deleted": true})
}

// resultsFilter 是 Results 端点的可选过滤项（均为原始 query 字符串，空串=不过滤）。
type resultsFilter struct {
	DeviceSN    string
	MetricPath  string
	Granularity string
	StartTime   string // RFC3339；非法/空则忽略
	EndTime     string // RFC3339；非法/空则忽略
	// ObjectLDNs T-0193：任务自带的小区/PLMN 白名单。非空时叠加 object_ldn = ANY(...) 过滤；
	// 空 = 不过滤（全小区）。与"只看 N 指标"同一层查看级收口。
	ObjectLDNs []string
	// ProductIDs PM-DASH-DIMFILTER：product 维度仪表盘按选中产品子集过滤（product_id = ANY，uuid 数组）。
	// 空 = 不过滤。独立于 T-0193 的 ObjectLDNs（那是任务白名单），两者作为独立 WHERE 子句叠加（AND 取交集）。
	ProductIDs []string
	// SubsetLDNs PM-DASH-DIMFILTER：device_group/band 维度仪表盘按选中子集过滤（object_ldn = ANY，text 数组，
	// 值形态 'DeviceGroup=<uuid>' / 'Band=<值>'）。空 = 不过滤。与 ObjectLDNs（任务白名单）各自独立成子句。
	SubsetLDNs []string
	// TaskMetricPaths #532：任务配置的指标集（task.MetricPaths）。显示侧收口——把「配置指标=显示范围」
	// 真正落在显示阶段。非空时叠加 metric_path = ANY(...)，与用户临时选的单指标/子集（MetricPath）各自独立成子句、
	// AND 取交集；空 = 不过滤（历史/边界任务向后兼容）。P2 落库全存已启用指标后，这道闸防止把全部指标铺满仪表盘。
	TaskMetricPaths []string
	// Weekdays #599：星期过滤（0=周日..6=周六，对齐 PostgreSQL EXTRACT(dow)）。
	// 空/全选 = 不过滤。筛的是 start_time 的星期几（与前端 dayjs().day() 同义）。
	Weekdays []int
	// Hours #599：小时段过滤（0..23，对齐 PostgreSQL EXTRACT(hour)）。
	// 空/全选 = 不过滤。筛的是 start_time 的整点小时。
	Hours []int
}

// buildResultsQuery 纯函数：拼 adhoc results 查询 SQL + 占位参数。
// 抽出来便于单测（带/不带大时间段两路）；时间段非法值容错忽略而非报错。
func buildResultsQuery(taskID uuid.UUID, f resultsFilter, limit, offset int) (string, []any) {
	// PM-线名解析：LEFT JOIN 在读时把分组键 ID 解析成可读名 —— product 维度按 product_id 取
	// product_dim.product_name；device_group 维度按 'DeviceGroup='||id 比对 object_ldn 取 device_group_dim.name。
	// 两 JOIN 都是 LEFT，互不影响（product 任务时组名 NULL、组任务时产品名 NULL）；名缺失（脏数据/已删）也返 NULL，前端回退 id 前 8 位。
	// 查询跑在 TsPool（pm_adhoc_aggregation_results 在时序库），products/device_groups 改读本库影子表。
	q := `
SELECT r.id, r.task_id, r.device_oui, r.device_sn, r.product_id, r.metric_path, r.metric_type, r.metric_value,
       r.statis_type, r.granularity, r.time, r.start_time, r.end_time, r.ingest_time, r.object_ldn, r.extra,
       p.product_name, g.name AS device_group_name
FROM pm_adhoc_aggregation_results r
LEFT JOIN product_dim p ON p.id = r.product_id
LEFT JOIN device_group_dim g ON ('DeviceGroup=' || g.id::text) = split_part(r.object_ldn, ',', 1)
WHERE r.task_id = $1`
	args := []any{taskID}
	pos := 2
	if f.DeviceSN != "" {
		q += fmt.Sprintf(" AND r.device_sn = $%d", pos)
		args = append(args, f.DeviceSN)
		pos++
	}
	if f.MetricPath != "" {
		q += fmt.Sprintf(" AND r.metric_path = $%d", pos)
		args = append(args, f.MetricPath)
		pos++
	}
	if f.Granularity != "" {
		q += fmt.Sprintf(" AND r.granularity = $%d", pos)
		args = append(args, f.Granularity)
		pos++
	}
	// 可选大时间段过滤（页签1 仪表盘大时间段驱动取数）：start_time/end_time 用 RFC3339 解析，
	// 命中则按 time 列窗口过滤，与现有 ORDER BY time DESC 同列；非法值忽略（容错而非 400）。
	if f.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, f.StartTime); err == nil {
			q += fmt.Sprintf(" AND r.time >= $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	if f.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, f.EndTime); err == nil {
			q += fmt.Sprintf(" AND r.time < $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	// T-0193：任务小区/PLMN 白名单（查看级收口）。非空时只返回选中 object_ldn 行；
	// 空 = 不过滤（全小区，向后兼容旧任务）。与上面"只看 N 指标"同层。
	if len(f.ObjectLDNs) > 0 {
		q += fmt.Sprintf(" AND r.object_ldn = ANY($%d)", pos)
		args = append(args, f.ObjectLDNs)
		pos++
	}
	// PM-DASH-DIMFILTER：仪表盘维度子集过滤（与上面任务白名单各自独立成子句，AND 取交集）。
	// product 维度按 product_id 子集；device_group/band 维度按 object_ldn 子集。空 = 不过滤（向后兼容）。
	if len(f.ProductIDs) > 0 {
		q += fmt.Sprintf(" AND r.product_id = ANY($%d)", pos)
		args = append(args, f.ProductIDs)
		pos++
	}
	if len(f.SubsetLDNs) > 0 {
		q += fmt.Sprintf(" AND r.object_ldn = ANY($%d)", pos)
		args = append(args, f.SubsetLDNs)
		pos++
	}
	// #532 显示侧收口：按任务配置指标集过滤（与用户临时选的 MetricPath 各自独立成子句、AND 取交集）。
	// 空 = 不过滤（历史/边界任务向后兼容）。须与 buildResultsCountQuery 同口径，否则 count 与数据对不上。
	if len(f.TaskMetricPaths) > 0 {
		q += fmt.Sprintf(" AND r.metric_path = ANY($%d)", pos)
		args = append(args, f.TaskMetricPaths)
		pos++
	}
	// #599：星期/小时段后端过滤（EXTRACT(dow/hour FROM start_time)）。
	// 全选（7 天/24 时）或空 = 不加条件（向后兼容）。
	if len(f.Weekdays) > 0 && len(f.Weekdays) < 7 {
		q += fmt.Sprintf(" AND EXTRACT(dow FROM r.start_time)::int = ANY($%d)", pos)
		args = append(args, f.Weekdays)
		pos++
	}
	if len(f.Hours) > 0 && len(f.Hours) < 24 {
		q += fmt.Sprintf(" AND EXTRACT(hour FROM r.start_time)::int = ANY($%d)", pos)
		args = append(args, f.Hours)
		pos++
	}
	q += fmt.Sprintf(" ORDER BY r.time DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)
	return q, args
}

// buildResultsCountQuery 纯函数：拼 adhoc results 的真实总数 COUNT(*) SQL + 占位参数。
// 复用与 buildResultsQuery 完全相同的 WHERE 过滤（去掉 LEFT JOIN / ORDER BY / LIMIT / OFFSET），
// 让 total 反映命中行真实总数（T-0194 截断诚实提示）。
func buildResultsCountQuery(taskID uuid.UUID, f resultsFilter) (string, []any) {
	q := `SELECT COUNT(*) FROM pm_adhoc_aggregation_results r WHERE r.task_id = $1`
	args := []any{taskID}
	pos := 2
	if f.DeviceSN != "" {
		q += fmt.Sprintf(" AND r.device_sn = $%d", pos)
		args = append(args, f.DeviceSN)
		pos++
	}
	if f.MetricPath != "" {
		q += fmt.Sprintf(" AND r.metric_path = $%d", pos)
		args = append(args, f.MetricPath)
		pos++
	}
	if f.Granularity != "" {
		q += fmt.Sprintf(" AND r.granularity = $%d", pos)
		args = append(args, f.Granularity)
		pos++
	}
	if f.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, f.StartTime); err == nil {
			q += fmt.Sprintf(" AND r.time >= $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	if f.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, f.EndTime); err == nil {
			q += fmt.Sprintf(" AND r.time < $%d", pos)
			args = append(args, t)
			pos++
		}
	}
	if len(f.ObjectLDNs) > 0 {
		q += fmt.Sprintf(" AND r.object_ldn = ANY($%d)", pos)
		args = append(args, f.ObjectLDNs)
		pos++
	}
	// PM-DASH-DIMFILTER：与 buildResultsQuery 同口径——同样的 ProductIDs / SubsetLDNs 子句，
	// 否则 count 与数据对不上（T-0194 踩过）。
	if len(f.ProductIDs) > 0 {
		q += fmt.Sprintf(" AND r.product_id = ANY($%d)", pos)
		args = append(args, f.ProductIDs)
		pos++
	}
	if len(f.SubsetLDNs) > 0 {
		q += fmt.Sprintf(" AND r.object_ldn = ANY($%d)", pos)
		args = append(args, f.SubsetLDNs)
		pos++
	}
	// #532：与 buildResultsQuery 同口径——同样的任务配置指标集子句，否则 count 与数据对不上。
	if len(f.TaskMetricPaths) > 0 {
		q += fmt.Sprintf(" AND r.metric_path = ANY($%d)", pos)
		args = append(args, f.TaskMetricPaths)
		pos++
	}
	// #599：与 buildResultsQuery 同口径——星期/小时段过滤。
	if len(f.Weekdays) > 0 && len(f.Weekdays) < 7 {
		q += fmt.Sprintf(" AND EXTRACT(dow FROM r.start_time)::int = ANY($%d)", pos)
		args = append(args, f.Weekdays)
		pos++
	}
	if len(f.Hours) > 0 && len(f.Hours) < 24 {
		q += fmt.Sprintf(" AND EXTRACT(hour FROM r.start_time)::int = ANY($%d)", pos)
		args = append(args, f.Hours)
		pos++
	}
	return q, args
}

// Results GET /pm/adhoc/tasks/:id/results?device_sn=&metric_path=&granularity=&start_time=&end_time=&limit=&offset=
func (h *Handler) Results(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	// 任务存在校验 + 取出白名单（T-0193：结果查询按任务自带 object_ldns 收口）
	task, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// 结果读权限与任务查看权限一致。
	if !canViewTask(task, extractCreator(c), isAdmin(c)) {
		response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
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
	filter := resultsFilter{
		DeviceSN:    c.Query("device_sn"),
		MetricPath:  c.Query("metric_path"),
		Granularity: c.Query("granularity"),
		StartTime:   c.Query("start_time"),
		EndTime:     c.Query("end_time"),
		ObjectLDNs:  task.ObjectLDNs, // 任务自带白名单（空=全小区）
		// PM-DASH-DIMFILTER：仪表盘维度子集过滤（默认两者都不传 = 不过滤 = 现行行为）。
		// product_ids 是纯 UUID（永不含逗号），可走 CSV 切分兼容单参数多值。
		ProductIDs: parseCSVQuery(c, "product_ids"),
		// object_ldns 的值合法含逗号（设备组维度 'DeviceGroup=<uuid>,Tech=<tech>'），
		// 故不能按逗号切分（issue #401：切分后两段都匹配不上完整存储值 → 0 行）。
		// 改走纯重复参数形态 ?object_ldns=a&object_ldns=b，整值保留不拆。
		SubsetLDNs: parseRepeatedQuery(c, "object_ldns"),
		// #532：任务配置指标集（显示侧收口）。让「配置指标=显示范围」落在显示阶段——
		// 与用户临时选的 metric_path 各自独立成子句、AND 取交集。空（历史/边界任务）= 不过滤（向后兼容）。
		TaskMetricPaths: task.MetricPaths,
		// #599：星期/小时段后端过滤（全选/空 = 不过滤，向后兼容）。
		Weekdays: parseCSVIntQuery(c, "weekdays"),
		Hours:    parseCSVIntQuery(c, "hours"),
	}
	q, args := buildResultsQuery(id, filter, limit, offset)

	rows, err := h.pool.Query(c.Request.Context(), q, args...)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type resultDTO struct {
		ID        string `json:"id"`
		TaskID    string `json:"task_id"`
		DeviceOUI string `json:"device_oui"`
		DeviceSN  string `json:"device_sn"`
		// product 维度结果的分组键（T-0182-fix）；device/aggregate_group 维度为空。
		ProductID string `json:"product_id,omitempty"`
		// PM-线名解析：读时 LEFT JOIN 解析出的可读名。product 任务才有 ProductName，
		// device_group 任务才有 DeviceGroupName；缺失（已删/脏数据）则空，前端回退 id 前 8 位。
		ProductName     string `json:"product_name,omitempty"`
		DeviceGroupName string `json:"device_group_name,omitempty"`
		MetricPath      string `json:"metric_path"`
		// KPI 行 metric_path 是 K 编号；display_name 为按编号回填的友好名（PLMN 级带标记）。counter 行 = metric_path。
		DisplayName string `json:"display_name,omitempty"`
		MetricType  string `json:"metric_type"`
		// MetricValue 用 jsonx.Float（底层 float64）兜底非有限值（NaN/Inf → null），
		// 避免单个 NaN 行致整批 JSON 编码失败、返回空 body（issue #387）。
		MetricValue jsonx.Float `json:"metric_value"`
		StatisType  *string     `json:"statis_type,omitempty"`
		Granularity string      `json:"granularity"`
		Time        time.Time   `json:"time"`
		StartTime   time.Time   `json:"start_time"`
		EndTime     time.Time   `json:"end_time"`
		IngestTime  time.Time   `json:"ingest_time"`
		ObjectLDN   *string     `json:"object_ldn,omitempty"`
	}
	items := make([]resultDTO, 0)
	for rows.Next() {
		var dto resultDTO
		var resultID, taskID uuid.UUID
		var productID *uuid.UUID // product_id 列可空（仅 product 维度有值）
		var extraBytes []byte
		var productName, deviceGroupName *string // LEFT JOIN 命中才有值，未命中（NULL）则空
		if err := rows.Scan(
			&resultID, &taskID, &dto.DeviceOUI, &dto.DeviceSN, &productID, &dto.MetricPath,
			&dto.MetricType, &dto.MetricValue, &dto.StatisType, &dto.Granularity,
			&dto.Time, &dto.StartTime, &dto.EndTime, &dto.IngestTime, &dto.ObjectLDN, &extraBytes,
			&productName, &deviceGroupName,
		); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		dto.ID = resultID.String()
		dto.TaskID = taskID.String()
		if productID != nil && *productID != uuid.Nil {
			dto.ProductID = productID.String()
		}
		if productName != nil {
			dto.ProductName = *productName
		}
		if deviceGroupName != nil {
			dto.DeviceGroupName = *deviceGroupName
		}
		items = append(items, dto)
	}

	// 回填 display_name：按 metric_path 编号查指标库取友好名——counter 与 kpi 同口径
	// （counter 编号 is_counter='1' 同样落在 perf_indicators_* 三表，与导出 name_resolver 一致）。
	// 查不到回退编号本身，保证非空。横表展示列名「编号(名·类型)」依赖此处给出中文名。
	codeSet := make(map[string]struct{})
	for i := range items {
		if items[i].MetricPath != "" {
			codeSet[items[i].MetricPath] = struct{}{}
		}
	}
	var nameByCode map[string]string
	if len(codeSet) > 0 {
		codes := make([]string, 0, len(codeSet))
		for code := range codeSet {
			codes = append(codes, code)
		}
		nameByCode = h.lookupIndicatorNames(c.Request.Context(), codes)
	}
	for i := range items {
		if name, ok := nameByCode[items[i].MetricPath]; ok && name != "" {
			items[i].DisplayName = name
		} else {
			items[i].DisplayName = items[i].MetricPath
		}
	}

	// 真实总数：跑一次同 WHERE 的 COUNT(*)，让 total 反映命中行真实总数而非本页返回行数
	// （T-0194 截断诚实提示）。COUNT 失败不阻断结果返回，退回本页行数作兜底。
	total := len(items)
	cq, cargs := buildResultsCountQuery(id, filter)
	var realTotal int
	if err := h.pool.QueryRow(c.Request.Context(), cq, cargs...).Scan(&realTotal); err == nil {
		total = realTotal
	}

	response.OK(c, gin.H{"items": items, "total": total})
}

// parseCSVQuery 读取一个既支持 CSV（逗号分隔）又支持重复参数（?k=a&k=b）的 query。
// 返回去空白后的非空项切片；无值返回 nil（让 = ANY 子句不进 SQL = 不过滤）。
func parseCSVQuery(c *gin.Context, key string) []string {
	raw := c.QueryArray(key) // 重复参数形态 ?k=a&k=b
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		for _, part := range strings.Split(v, ",") { // 兼容单参数内逗号分隔 ?k=a,b
			p := strings.TrimSpace(part)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseCSVIntQuery 读取逗号分隔的整数列表 query（如 ?weekdays=0,1,2）。
// 非法值静默跳过；无有效值返回 nil（不过滤）。
func parseCSVIntQuery(c *gin.Context, key string) []int {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseRepeatedQuery 读取只走「重复参数」形态（?k=a&k=b）的多值 query，整值保留不按逗号拆分。
// 用于值本身合法含逗号的参数（如 object_ldns 的设备组值 'DeviceGroup=<uuid>,Tech=<tech>'，
// issue #401：若按逗号拆会把单个完整值切成两段，导致 object_ldn = ANY(...) 匹配不上 → 0 行）。
// 返回去空白后的非空项切片；无值返回 nil（让 = ANY 子句不进 SQL = 不过滤）。
func parseRepeatedQuery(c *gin.Context, key string) []string {
	raw := c.QueryArray(key) // 重复参数形态 ?k=a&k=b；每个值整体保留，不拆逗号
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		p := strings.TrimSpace(v)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// filterOptionDTO 是 filter-options 端点单个可筛选项（value=分组键、label=能拿到的最好名）。
type filterOptionDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// buildFilterOptionsQuery 纯函数：按维度拼 filter-options 的 SELECT DISTINCT SQL + 占位参数。
//
// 返回 (sql, args, supported)。supported=false 表示该维度不支持筛选（device/aggregate_group/network），
// 调用方据此直接回空选项数组，不查库。
//   - product：DISTINCT product_id + LEFT JOIN product_dim 取 product_name（跨库分离用影子表）
//   - device_group：DISTINCT object_ldn + LEFT JOIN device_group_dim 取组名（'DeviceGroup='||id 比对）
//   - band：DISTINCT object_ldn（频段无现成名，label 给原值，可读化交前端）
//
// 三条 SQL 均无 LIMIT/OFFSET —— 选项是与结果分页/上限完全解耦的权威全量子集（不被结果上限截断）。
func buildFilterOptionsQuery(dim Dimension, taskID uuid.UUID) (string, []any, bool) {
	switch dim {
	case DimensionProduct:
		// 查询跑在 TsPool；products 改读本库影子表 product_dim。
		return `
SELECT DISTINCT r.product_id, p.product_name
FROM pm_adhoc_aggregation_results r
LEFT JOIN product_dim p ON p.id = r.product_id
WHERE r.task_id = $1 AND r.product_id IS NOT NULL
ORDER BY p.product_name`, []any{taskID}, true
	case DimensionDeviceGroup:
		// 查询跑在 TsPool；device_groups 改读本库影子表 device_group_dim。
		return `
SELECT DISTINCT r.object_ldn, g.name
FROM pm_adhoc_aggregation_results r
LEFT JOIN device_group_dim g ON ('DeviceGroup=' || g.id::text) = split_part(r.object_ldn, ',', 1)
WHERE r.task_id = $1 AND r.object_ldn LIKE 'DeviceGroup=%'
ORDER BY g.name`, []any{taskID}, true
	case DimensionBand:
		return `
SELECT DISTINCT r.object_ldn
FROM pm_adhoc_aggregation_results r
WHERE r.task_id = $1 AND r.object_ldn LIKE 'Band=%'
ORDER BY r.object_ldn`, []any{taskID}, true
	default:
		// device / aggregate_group / network：无可筛子集
		return "", nil, false
	}
}

// FilterOptions GET /pm/adhoc/tasks/:id/filter-options
//
// 按选中任务的聚合维度，列出"该任务实际聚合了哪些产品/设备组/频段"的权威子集清单（走 SELECT DISTINCT，
// 与结果分页/上限解耦，不被截断）。前端据此渲染对应维度的多选筛选框。
// 维度为 device/aggregate_group/network 时返回空 options（前端不渲染筛选框）。
func (h *Handler) FilterOptions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// 筛选选项属于结果数据的衍生视图，与 Results 同口径做读权限校验。
	if !canViewTask(task, extractCreator(c), isAdmin(c)) {
		response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
		return
	}

	dim := task.Dimension
	if dim == "" {
		dim = DimensionDevice
	}

	options := make([]filterOptionDTO, 0)
	q, args, supported := buildFilterOptionsQuery(dim, id)
	if supported {
		rows, err := h.pool.Query(c.Request.Context(), q, args...)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var opt filterOptionDTO
			switch dim {
			case DimensionProduct:
				var productID uuid.UUID
				var productName *string // LEFT JOIN 未命中（已删/脏数据）则 NULL
				if err := rows.Scan(&productID, &productName); err != nil {
					commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
					return
				}
				opt.Value = productID.String()
				if productName != nil && *productName != "" {
					opt.Label = *productName
				} else {
					// 名缺失兜底（产品已删/脏数据）：label 回退 value，保证不为空、不崩（验收 1）
					opt.Label = productID.String()
				}
			case DimensionDeviceGroup:
				var objectLDN string
				var groupName *string
				if err := rows.Scan(&objectLDN, &groupName); err != nil {
					commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
					return
				}
				opt.Value = objectLDN
				if groupName != nil && *groupName != "" {
					opt.Label = *groupName
				} else {
					opt.Label = objectLDN // 组已删/脏数据，回退原值
				}
			case DimensionBand:
				var objectLDN string
				if err := rows.Scan(&objectLDN); err != nil {
					commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
					return
				}
				// 频段无现成名，后端给原值（如 Band=42），可读化交前端
				opt.Value = objectLDN
				opt.Label = objectLDN
			}
			options = append(options, opt)
		}
		if err := rows.Err(); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
	}

	response.OK(c, gin.H{"dimension": string(dim), "options": options})
}

// lookupIndicatorNames 按编号集合一次性查三张指标表，返回 code → 本地化显示名。
// 取名方向按 ctx 中的 locale 决定（中文 cn_name 优先 / 英文 en_name 优先，空则回退另一种）。
// K 编号在 perf_indicators_{enb,gnb,gsm} 三表全局唯一，一次 UNION 即可覆盖（与 aggregator 查询层一致）。
func (h *Handler) lookupIndicatorNames(ctx context.Context, codes []string) map[string]string {
	out := make(map[string]string, len(codes))
	nameExpr := metrics.IndicatorDisplayNameExpr(appcontext.GetLocale(ctx))
	tmpl := fmt.Sprintf(`
SELECT id, %[1]s AS display_name FROM perf_indicators_enb  WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gnb  WHERE id = ANY($1)
UNION ALL
SELECT id, %[1]s AS display_name FROM perf_indicators_gsm  WHERE id = ANY($1)`, nameExpr)
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
	// 任务存在校验 + #652 读权限校验
	task, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !canViewTask(task, extractCreator(c), isAdmin(c)) {
		response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
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
// 认证授权通过后只订阅当前 APP 实例的本地 ProgressHub。上游 Core NATS 订阅
// 由 APP 启动期的 ProgressBridge 统一持有，不随 HTTP 连接数量增长。
func (h *Handler) Progress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "task not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !canViewTask(task, extractCreator(c), isAdmin(c)) {
		response.Fail(c, http.StatusForbidden, "permission denied: not task owner")
		return
	}
	if h.hub == nil {
		response.Fail(c, http.StatusServiceUnavailable, "progress hub not wired")
		return
	}
	if _, ok := c.Writer.(http.Flusher); !ok {
		response.Fail(c, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	events, unsubscribe := h.hub.Subscribe(id.String())
	defer unsubscribe()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx 不缓冲

	// SSE 必须越过 http.Server 的全响应 WriteTimeout，否则默认 30 秒后连接被切断。
	if rc := http.NewResponseController(c.Writer); rc != nil {
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			h.logger.Warn("clear adhoc progress SSE write deadline failed",
				zap.String("task_id", id.String()), zap.Error(err))
		}
	}

	// 立即提交状态和真正的 SSE comment，避免 EventSource/onopen 等到首个业务事件。
	c.Status(http.StatusOK)
	if err := writeSSEString(c.Writer, ":connected\n\n"); err != nil {
		return
	}
	if err := flushSSE(c.Writer); err != nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := h.writeSSE(c.Writer, event.Name, event.Data); err != nil {
				return
			}
			if err := flushSSE(c.Writer); err != nil {
				return
			}
			if event.Name == "completed" {
				return
			}
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			if err := writeSSEString(c.Writer, ":keepalive\n\n"); err != nil {
				return
			}
			if err := flushSSE(c.Writer); err != nil {
				return
			}
		}
	}
}

func flushSSE(w http.ResponseWriter) error {
	return http.NewResponseController(w).Flush()
}

func (h *Handler) writeSSE(w io.Writer, eventName string, data []byte) error {
	if eventName != "" {
		if err := writeSSEString(w, "event: "+eventName+"\n"); err != nil {
			return err
		}
	}
	if err := writeSSEString(w, "data: "); err != nil {
		return err
	}
	if err := writeSSEBytes(w, data); err != nil {
		return err
	}
	return writeSSEString(w, "\n\n")
}

func writeSSEString(w io.Writer, data string) error {
	n, err := io.WriteString(w, data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func writeSSEBytes(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// canOperate 判断当前用户对自建任务有无管理权限（编辑/恢复/删除）。
// 超管可操作任意自建任务；public 自建任务允许登录用户操作；private 仅创建者可操作。
func canOperate(task *Task, currentUser string, admin bool) bool {
	if admin {
		return true
	}
	if normalizeVisibility(task.Visibility) == VisibilityPublic {
		return true
	}
	return task.Creator == currentUser
}

// canCancelTask 判断当前用户能否取消任务。
// 取消会影响正在执行/排期中的任务，只允许创建者或超管执行。
func canCancelTask(task *Task, currentUser string, admin bool) bool {
	if admin {
		return true
	}
	return task.Creator == currentUser
}

// canViewTask 判断当前用户对任务定义、结果和运行信息有无读权限。
// 内置任务全员可读；public 自建任务全员可读；private 自建任务仅创建者或超管可读。
func canViewTask(task *Task, currentUser string, admin bool) bool {
	if task.IsBuiltin {
		return true
	}
	if admin {
		return true
	}
	if normalizeVisibility(task.Visibility) == VisibilityPublic {
		return true
	}
	return task.Creator == currentUser
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

// isAdmin 判断当前用户是否具备全局管理员视角。
// 内置超管（source='builtIn' 派生 is_super_admin）和 admin / super_admin 角色都可查看全部任务。
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
