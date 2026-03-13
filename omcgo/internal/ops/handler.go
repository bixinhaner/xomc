package ops

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for ops tools REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new ops Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("ops-handler"),
	}
}

// RegisterRoutes registers ops routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	opsGroup := rg.Group("/ops")

	templates := opsGroup.Group("/templates")
	templates.GET("", h.ListTemplates)
	templates.POST("", h.CreateTemplate)
	templates.GET("/:id", h.GetTemplate)
	templates.PUT("/:id", h.UpdateTemplate)
	templates.DELETE("/:id", h.DeleteTemplate)

	cmdRecords := opsGroup.Group("/command-records")
	cmdRecords.GET("", h.ListCommandRecords)
	cmdRecords.POST("", h.CreateCommandRecord)

	tasks := opsGroup.Group("/tasks")
	tasks.GET("", h.ListTasks)
	tasks.POST("", h.CreateTask)
	tasks.GET("/:id", h.GetTask)
	tasks.POST("/:id/cancel", h.CancelTask)
	tasks.POST("/:id/pause", h.PauseTask)
	tasks.POST("/:id/resume", h.ResumeTask)
}

// ---- Template request types ----

// CreateTemplateRequest defines the request body for creating an ops template.
type CreateTemplateRequest struct {
	TemplateName      string          `json:"template_name" binding:"required"`
	Description       string          `json:"description"`
	Category          string          `json:"category"`
	TargetDeviceTypes json.RawMessage `json:"target_device_types"`
	Steps             json.RawMessage `json:"steps"`
	EstimatedDuration int             `json:"estimated_duration"`
	Creator           string          `json:"creator"`
	Tags              json.RawMessage `json:"tags"`
}

// UpdateTemplateRequest defines the request body for updating an ops template.
type UpdateTemplateRequest struct {
	TemplateName      string          `json:"template_name" binding:"required"`
	Description       string          `json:"description"`
	Category          string          `json:"category"`
	TargetDeviceTypes json.RawMessage `json:"target_device_types"`
	Steps             json.RawMessage `json:"steps"`
	EstimatedDuration int             `json:"estimated_duration"`
	Creator           string          `json:"creator"`
	Tags              json.RawMessage `json:"tags"`
}

// ---- Task request types ----

// CreateTaskRequest defines the request body for creating an ops task.
type CreateTaskRequest struct {
	TaskName   string          `json:"task_name" binding:"required"`
	TemplateID *uuid.UUID      `json:"template_id"`
	DeviceSNs  json.RawMessage `json:"device_sns"`
	TotalSteps int             `json:"total_steps"`
	TotalCount int             `json:"total_count"`
	Creator    string          `json:"creator"`
	Message    string          `json:"message"`
}

// ---- Command Record request types ----

// CreateCommandRecordRequest defines the request body for creating a command record.
type CreateCommandRecordRequest struct {
	CommandText  string `json:"command_text" binding:"required"`
	DeviceSN     string `json:"device_sn" binding:"required"`
	DeviceName   string `json:"device_name"`
	Operator     string `json:"operator"`
	ExecuteTime  string `json:"execute_time"`
	Duration     int    `json:"duration"`
	Success      bool   `json:"success"`
	Output       string `json:"output"`
	ErrorMessage *string `json:"error_message"`
}

// ---- Template handlers ----

// ListTemplates handles GET /api/v1/ops/templates.
func (h *Handler) ListTemplates(c *gin.Context) {
	filter := TemplateFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if category := c.Query("category"); category != "" {
		filter.Category = category
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filter.Keyword = keyword
	}

	result, err := h.service.ListTemplates(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateTemplate handles POST /api/v1/ops/templates.
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tmpl := &OpsTemplate{
		TemplateName:      req.TemplateName,
		Description:       req.Description,
		Category:          req.Category,
		TargetDeviceTypes: req.TargetDeviceTypes,
		Steps:             req.Steps,
		EstimatedDuration: req.EstimatedDuration,
		Creator:           req.Creator,
		Tags:              req.Tags,
	}

	created, err := h.service.CreateTemplate(c.Request.Context(), tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetTemplate handles GET /api/v1/ops/templates/:id.
func (h *Handler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	tmpl, err := h.service.GetTemplate(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, tmpl)
}

// UpdateTemplate handles PUT /api/v1/ops/templates/:id.
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tmpl := &OpsTemplate{
		TemplateName:      req.TemplateName,
		Description:       req.Description,
		Category:          req.Category,
		TargetDeviceTypes: req.TargetDeviceTypes,
		Steps:             req.Steps,
		EstimatedDuration: req.EstimatedDuration,
		Creator:           req.Creator,
		Tags:              req.Tags,
	}

	updated, err := h.service.UpdateTemplate(c.Request.Context(), id, tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteTemplate handles DELETE /api/v1/ops/templates/:id.
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteTemplate(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---- Command Record handlers ----

// ListCommandRecords handles GET /api/v1/ops/command-records.
func (h *Handler) ListCommandRecords(c *gin.Context) {
	filter := CommandRecordFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if deviceSN := c.Query("deviceSn"); deviceSN != "" {
		filter.DeviceSN = deviceSN
	}
	if operator := c.Query("operator"); operator != "" {
		filter.Operator = operator
	}
	if success := c.Query("success"); success != "" {
		b := success == "true"
		filter.Success = &b
	}

	result, err := h.service.ListCommandRecords(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateCommandRecord handles POST /api/v1/ops/command-records.
func (h *Handler) CreateCommandRecord(c *gin.Context) {
	var req CreateCommandRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	record := &OpsCommandRecord{
		CommandText:  req.CommandText,
		DeviceSN:     req.DeviceSN,
		DeviceName:   req.DeviceName,
		Operator:     req.Operator,
		Duration:     req.Duration,
		Success:      req.Success,
		Output:       req.Output,
		ErrorMessage: req.ErrorMessage,
	}

	created, err := h.service.CreateCommandRecord(c.Request.Context(), record)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// ---- Task handlers ----

// ListTasks handles GET /api/v1/ops/tasks.
func (h *Handler) ListTasks(c *gin.Context) {
	filter := TaskFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := OpsTaskStatus(status)
		filter.Status = &s
	}
	if templateID := c.Query("templateId"); templateID != "" {
		id, err := uuid.Parse(templateID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.TemplateID = &id
	}

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateTask handles POST /api/v1/ops/tasks.
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task := &OpsTask{
		TaskName:   req.TaskName,
		TemplateID: req.TemplateID,
		DeviceSNs:  req.DeviceSNs,
		TotalSteps: req.TotalSteps,
		TotalCount: req.TotalCount,
		Creator:    req.Creator,
		Message:    req.Message,
	}

	created, err := h.service.CreateTask(c.Request.Context(), task)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetTask handles GET /api/v1/ops/tasks/:id.
func (h *Handler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, task)
}

// CancelTask handles POST /api/v1/ops/tasks/:id/cancel.
func (h *Handler) CancelTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.CancelTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// PauseTask handles POST /api/v1/ops/tasks/:id/pause.
func (h *Handler) PauseTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.PauseTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "paused"})
}

// ResumeTask handles POST /api/v1/ops/tasks/:id/resume.
func (h *Handler) ResumeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.ResumeTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "running"})
}
