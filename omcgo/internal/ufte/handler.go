package ufte

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	ufte.GET("/tasks", h.ListTasks)
	ufte.POST("/tasks", h.CreateTask)
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
