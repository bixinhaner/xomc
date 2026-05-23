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
		adhoc.GET("/tasks/:id/progress", h.Progress) // SSE
	}
}

// ── 请求/响应 DTO ─────────────────────────────────────────────────────────

type createRequestDTO struct {
	Name          string    `json:"name" binding:"required"`
	Mode          string    `json:"mode" binding:"required,oneof=oneshot continuous"`
	CronExpr      string    `json:"cron_expr"`
	DeviceSNs     []string  `json:"device_sns" binding:"required,min=1"`
	MetricPaths   []string  `json:"metric_paths" binding:"required,min=1"`
	Granularities []string  `json:"granularities" binding:"required,min=1"`
	WindowStart   time.Time `json:"window_start" binding:"required"`
	WindowEnd     time.Time `json:"window_end" binding:"required"`
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
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Creator       string    `json:"creator"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func taskToDTO(t *Task) taskResponseDTO {
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
	if !req.WindowEnd.After(req.WindowStart) {
		response.Fail(c, http.StatusBadRequest, "window_end must be after window_start")
		return
	}
	var cronPtr *string
	if req.CronExpr != "" {
		cronPtr = &req.CronExpr
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
		Creator:       creator,
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{"id": id.String()})
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
	currentUser := extractCreator(c)
	all := c.Query("all") == "true"
	switch {
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

// Results GET /pm/adhoc/tasks/:id/results?device_sn=&metric_path=&granularity=&limit=&offset=
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
	q := `
SELECT id, task_id, device_oui, device_sn, metric_path, metric_type, metric_value,
       statis_type, granularity, time, start_time, end_time, ingest_time, object_ldn, extra
FROM pm_adhoc_aggregation_results
WHERE task_id = $1`
	args := []any{id}
	pos := 2
	if v := c.Query("device_sn"); v != "" {
		q += fmt.Sprintf(" AND device_sn = $%d", pos)
		args = append(args, v)
		pos++
	}
	if v := c.Query("metric_path"); v != "" {
		q += fmt.Sprintf(" AND metric_path = $%d", pos)
		args = append(args, v)
		pos++
	}
	if v := c.Query("granularity"); v != "" {
		q += fmt.Sprintf(" AND granularity = $%d", pos)
		args = append(args, v)
		pos++
	}
	q += fmt.Sprintf(" ORDER BY time DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)

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
		MetricPath  string    `json:"metric_path"`
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
		var extraBytes []byte
		if err := rows.Scan(
			&resultID, &taskID, &dto.DeviceOUI, &dto.DeviceSN, &dto.MetricPath,
			&dto.MetricType, &dto.MetricValue, &dto.StatisType, &dto.Granularity,
			&dto.Time, &dto.StartTime, &dto.EndTime, &dto.IngestTime, &dto.ObjectLDN, &extraBytes,
		); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		dto.ID = resultID.String()
		dto.TaskID = taskID.String()
		items = append(items, dto)
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
