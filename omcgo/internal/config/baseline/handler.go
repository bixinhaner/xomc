package baseline

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// Handler provides HTTP handlers for config baseline management REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new config baseline Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("config-baseline-handler"),
	}
}

// RegisterRoutes registers config baseline routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	config := rg.Group("/config")

	baselines := config.Group("/baselines")
	baselines.GET("", h.ListBaselines)
	baselines.POST("", h.CreateBaseline)
	baselines.GET("/:id", h.GetBaseline)
	baselines.PUT("/:id", h.UpdateBaseline)
	baselines.DELETE("/:id", h.DeleteBaseline)

	tasks := config.Group("/tasks")
	tasks.GET("", h.ListConfigTasks)
	tasks.POST("", h.CreateConfigTask)

	neighbors := config.Group("/neighbors")
	neighbors.GET("", h.ListNeighbors)
}

// ---- Baseline request types ----

// CreateBaselineRequest defines the request body for creating a baseline config.
type CreateBaselineRequest struct {
	BaselineName string          `json:"baseline_name" binding:"required"`
	Description  *string         `json:"description"`
	DeviceType   *string         `json:"device_type"`
	Version      *string         `json:"version"`
	Params       json.RawMessage `json:"params"`
	Creator      *string         `json:"creator"`
	Status       BaselineStatus  `json:"status"`
}

// UpdateBaselineRequest defines the request body for updating a baseline config.
type UpdateBaselineRequest struct {
	BaselineName string          `json:"baseline_name" binding:"required"`
	Description  *string         `json:"description"`
	DeviceType   *string         `json:"device_type"`
	Version      *string         `json:"version"`
	Params       json.RawMessage `json:"params"`
	Creator      *string         `json:"creator"`
	Status       BaselineStatus  `json:"status"`
}

// CreateConfigTaskRequest defines the request body for creating a config task.
type CreateConfigTaskRequest struct {
	TaskName   string          `json:"task_name" binding:"required"`
	TaskType   ConfigTaskType  `json:"task_type" binding:"required"`
	DeviceSns  json.RawMessage `json:"device_sns"`
	TemplateID *uuid.UUID      `json:"template_id"`
	BaselineID *uuid.UUID      `json:"baseline_id"`
	Params     json.RawMessage `json:"params"`
	Creator    *string         `json:"creator"`
	Message    *string         `json:"message"`
	TotalCount int             `json:"total_count"`
}

// ---- Baseline handlers ----

// ListBaselines handles GET /api/v1/config/baselines.
func (h *Handler) ListBaselines(c *gin.Context) {
	filter := BaselineFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if deviceType := c.Query("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}
	if status := c.Query("status"); status != "" {
		s := BaselineStatus(status)
		filter.Status = &s
	}

	result, err := h.service.ListBaselines(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateBaseline handles POST /api/v1/config/baselines.
func (h *Handler) CreateBaseline(c *gin.Context) {
	var req CreateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	baseline := &BaselineConfig{
		BaselineName: req.BaselineName,
		Description:  req.Description,
		DeviceType:   req.DeviceType,
		Version:      req.Version,
		Params:       req.Params,
		Creator:      req.Creator,
		Status:       req.Status,
	}

	created, err := h.service.CreateBaseline(c.Request.Context(), baseline)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetBaseline handles GET /api/v1/config/baselines/:id.
func (h *Handler) GetBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	baseline, err := h.service.GetBaseline(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, baseline)
}

// UpdateBaseline handles PUT /api/v1/config/baselines/:id.
func (h *Handler) UpdateBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	baseline := &BaselineConfig{
		BaselineName: req.BaselineName,
		Description:  req.Description,
		DeviceType:   req.DeviceType,
		Version:      req.Version,
		Params:       req.Params,
		Creator:      req.Creator,
		Status:       req.Status,
	}

	updated, err := h.service.UpdateBaseline(c.Request.Context(), id, baseline)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteBaseline handles DELETE /api/v1/config/baselines/:id.
func (h *Handler) DeleteBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteBaseline(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---- Config Task handlers ----

// ListConfigTasks handles GET /api/v1/config/tasks.
func (h *Handler) ListConfigTasks(c *gin.Context) {
	filter := ConfigTaskFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := ConfigTaskStatus(status)
		filter.Status = &s
	}

	result, err := h.service.ListConfigTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateConfigTask handles POST /api/v1/config/tasks.
func (h *Handler) CreateConfigTask(c *gin.Context) {
	var req CreateConfigTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task := &ConfigTask{
		TaskName:   req.TaskName,
		TaskType:   req.TaskType,
		DeviceSns:  req.DeviceSns,
		TemplateID: req.TemplateID,
		BaselineID: req.BaselineID,
		Params:     req.Params,
		Creator:    req.Creator,
		Message:    req.Message,
		TotalCount: req.TotalCount,
	}

	created, err := h.service.CreateConfigTask(c.Request.Context(), task)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// ---- Neighbor handlers ----

// ListNeighbors handles GET /api/v1/config/neighbors.
func (h *Handler) ListNeighbors(c *gin.Context) {
	filter := NeighborFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if sourceCellID := c.Query("source_cell_id"); sourceCellID != "" {
		filter.SourceCellID = &sourceCellID
	}

	result, err := h.service.ListNeighbors(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}
