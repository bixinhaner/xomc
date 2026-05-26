package task

import (
	gerr "errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 暴露 MR 任务管理 REST API。
//
// 路由（注册在 /api/v1 下，资源组 "pm"，与 internal/mr/handler.go 一致）：
//
//	POST   /mr/tasks
//	GET    /mr/tasks
//	GET    /mr/tasks/:id
//	POST   /mr/tasks/:id/stop
//	DELETE /mr/tasks/:id
//	GET    /mr/tasks/:id/progress
type Handler struct {
	svc    Service
	logger *zap.Logger
}

// NewHandler 创建 Handler 实例。
func NewHandler(svc Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 挂载路由到给定 router group（一般是 permGroup("pm")）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	grp := rg.Group("/mr/tasks")
	{
		grp.POST("", h.Create)
		grp.GET("", h.List)
		grp.GET("/:id", h.Get)
		grp.POST("/:id/stop", h.Stop)
		grp.DELETE("/:id", h.Delete)
		grp.GET("/:id/progress", h.ListProgress)
	}
}

// ---------- 请求/响应 DTO ----------

// createRequest 是 POST /mr/tasks 的 body。
// 时间字段使用 RFC3339 字符串（前端 dayjs.toISOString()）。
//
// 简化版（2026-05-25）：去除 operator_code / creator / note / targets。
// creator 由 handler 从 auth ctx 自动填；targets 由 scheduler 在开启时
// 从 mr_device_mappings 动态枚举。
type createRequest struct {
	TaskName        string   `json:"task_name"        binding:"required,max=128"`
	MRType          string   `json:"mr_type"`
	StatisPeriod    string   `json:"statis_period"`
	ReportPeriod    string   `json:"report_period"`
	StartTime       string   `json:"start_time"       binding:"required"`
	EndTime         string   `json:"end_time"`
	TargetDeviceSNs []string `json:"target_device_sns" binding:"required,min=1"`
}

// ---------- handler 实现 ----------

func (h *Handler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid start_time: must be RFC3339")
		return
	}
	var endTimePtr *time.Time
	if strings.TrimSpace(req.EndTime) != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid end_time: must be RFC3339")
			return
		}
		endTimePtr = &endTime
	}

	// creator 从 gin Context 取（admin/auth middleware 在 c 上挂 "username"）。
	// 取不到（未鉴权场景）退化到 "system"，service 层会兜底。
	creator := "system"
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			creator = s
		}
	}

	input := CreateTaskInput{
		TaskName:        req.TaskName,
		MRType:          req.MRType,
		StatisPeriod:    req.StatisPeriod,
		ReportPeriod:    req.ReportPeriod,
		StartTime:       startTime,
		EndTime:         endTimePtr,
		Creator:         creator,
		TargetDeviceSNs: req.TargetDeviceSNs,
	}

	task, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

type listQuery struct {
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	SortBy   string `form:"sort_by"`
	SortDir  string `form:"sort_dir"`
}

func (h *Handler) List(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	filter := TaskListFilter{
		Page:     q.Page,
		PageSize: q.PageSize,
		SortBy:   q.SortBy,
		SortDir:  q.SortDir,
	}
	if q.Status != "" {
		s := TaskStatus(q.Status)
		filter.Status = &s
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	resp, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

func (h *Handler) Get(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task id")
		return
	}
	t, err := h.svc.Get(c.Request.Context(), taskID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	response.OK(c, t)
}

// Stop 不再要求 body — 仅 path 参数。
func (h *Handler) Stop(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := h.svc.Stop(c.Request.Context(), taskID); err != nil {
		h.writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"task_id": taskID.String(), "status": StatusTermination})
}

func (h *Handler) Delete(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), taskID); err != nil {
		h.writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"task_id": taskID.String()})
}

type progressQuery struct {
	Status   string `form:"status"`
	Health   string `form:"health"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=50"`
}

func (h *Handler) ListProgress(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task id")
		return
	}
	var q progressQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	filter := ProgressListFilter{
		TaskID:   taskID,
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	if q.Status != "" {
		s := ProgressStatus(q.Status)
		filter.Status = &s
	}
	if q.Health != "" {
		s := HealthStatus(q.Health)
		filter.Health = &s
	}
	resp, err := h.svc.ListProgress(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

// writeServiceError 把 service 层错误映射到合适的 HTTP 状态：
//   - 入参校验 → 400
//   - 状态机不允许 → 409
//   - NotFound → 404
//   - 其它 → 500
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case IsValidationError(err):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case IsStateError(err):
		response.Fail(c, http.StatusConflict, err.Error())
	case gerr.Is(err, commonerrors.ErrNotFound):
		response.Fail(c, http.StatusNotFound, "mr task not found")
	default:
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
	}
}
