package ufte

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("ufte-handler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	ufte := rg.Group("/ufte")
	ufte.GET("/overview", h.GetOverview)
	ufte.GET("/task-types", h.ListTaskTypes)
	ufte.POST("/task-types", h.CreateTaskType)
	ufte.PUT("/task-types/:typeCode", h.UpdateTaskType)
	ufte.DELETE("/task-types/:typeCode", h.DeleteTaskType)
	ufte.GET("/tasks", h.ListTasks)
	ufte.POST("/tasks", h.CreateTask)
	ufte.PUT("/tasks/:id/start", h.StartTask)
	ufte.PUT("/tasks/:id/suspend", h.SuspendTask)
	ufte.PUT("/tasks/:id/terminate", h.TerminateTask)
	ufte.DELETE("/tasks/:id", h.DeleteTask)
	// 批量删除：body 走 POST 避免 DELETE+body 在某些代理 / WAF 下被吞
	ufte.POST("/tasks/batch-delete", h.BatchDeleteTasks)
	ufte.POST("/tasks/:id/retry", h.RetryTask)
	ufte.GET("/devices", h.ListDevices)
	ufte.GET("/device-candidates", h.ListDeviceCandidates)
}

func (h *Handler) GetOverview(c *gin.Context) {
	result, err := h.service.GetOverview(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListTaskTypes(c *gin.Context) {
	items, err := h.service.GetTaskTypes(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) CreateTaskType(c *gin.Context) {
	var req TaskTypeWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	editor := currentUsername(c)
	item, err := h.service.CreateTaskType(c.Request.Context(), req, editor)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, item)
}

func (h *Handler) UpdateTaskType(c *gin.Context) {
	var req TaskTypeWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.UpdateTaskType(c.Request.Context(), c.Param("typeCode"), req, currentUsername(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) DeleteTaskType(c *gin.Context) {
	if err := h.service.DeleteTaskType(c.Request.Context(), c.Param("typeCode")); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"typeCode": c.Param("typeCode")})
}

func (h *Handler) ListTasks(c *gin.Context) {
	var filter TaskListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	creator := currentUsername(c)
	task, err := h.service.CreateTask(c.Request.Context(), req, creator)

	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionUpgrade
	entry.ResourceType = audit.ResourceUpgradeTask
	entry.Success = err == nil
	if task != nil {
		entry.ResourceID = task.ID
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.LogAsync(entry)

	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

func (h *Handler) ListDevices(c *gin.Context) {
	var filter DeviceListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.ListDevices(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) parseTaskID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) StartTask(c *gin.Context) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	if err := h.service.StartTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) SuspendTask(c *gin.Context) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	if err := h.service.SuspendTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) TerminateTask(c *gin.Context) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	if err := h.service.TerminateTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

// BatchDeleteTasksRequest 批量删除 body。
type BatchDeleteTasksRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required,min=1"`
}

// BatchDeleteTasksResponse 批量删除响应。
type BatchDeleteTasksResponse struct {
	Succeeded []string                     `json:"succeeded"`
	Failed    []BatchDeleteTaskFailureItem `json:"failed"`
}

// BatchDeleteTaskFailureItem 失败明细。
type BatchDeleteTaskFailureItem struct {
	TaskID string `json:"task_id"`
	Error  string `json:"error"`
}

// BatchDeleteTasks 批量删除任务；单条失败不影响其他。
func (h *Handler) BatchDeleteTasks(c *gin.Context) {
	var req BatchDeleteTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	ids := make([]uuid.UUID, 0, len(req.TaskIDs))
	invalidIDs := make([]BatchDeleteTaskFailureItem, 0)
	for _, raw := range req.TaskIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			invalidIDs = append(invalidIDs, BatchDeleteTaskFailureItem{
				TaskID: raw, Error: "invalid uuid: " + err.Error(),
			})
			continue
		}
		ids = append(ids, id)
	}
	results := h.service.BatchDeleteTasks(c.Request.Context(), ids)
	resp := BatchDeleteTasksResponse{
		Succeeded: make([]string, 0, len(results)),
		Failed:    invalidIDs, // 先把 uuid 解析失败的塞进去
	}
	for _, r := range results {
		if r.Success {
			resp.Succeeded = append(resp.Succeeded, r.TaskID.String())
		} else {
			resp.Failed = append(resp.Failed, BatchDeleteTaskFailureItem{
				TaskID: r.TaskID.String(), Error: r.Error,
			})
		}
	}
	response.OK(c, resp)
}

func (h *Handler) RetryTask(c *gin.Context) {
	id, ok := h.parseTaskID(c)
	if !ok {
		return
	}
	if err := h.service.RetryTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) ListDeviceCandidates(c *gin.Context) {
	var filter DeviceCandidateFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.ListDeviceCandidates(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func currentUsername(c *gin.Context) string {
	if raw, ok := c.Get("username"); ok {
		if name, ok := raw.(string); ok && name != "" {
			return name
		}
	}
	return "system"
}
