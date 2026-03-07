package backup

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// Handler provides HTTP handlers for backup management REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new backup Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("backup-handler"),
	}
}

// RegisterRoutes registers backup routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	backup := rg.Group("/backup")

	tasks := backup.Group("/tasks")
	tasks.GET("", h.ListTasks)
	tasks.POST("", h.CreateTask)
	tasks.GET("/:id", h.GetTask)
	tasks.DELETE("/:id", h.DeleteTask)
	tasks.POST("/:id/cancel", h.CancelTask)

	schedules := backup.Group("/schedules")
	schedules.GET("", h.ListSchedules)
	schedules.POST("", h.CreateSchedule)
	schedules.PUT("/:id", h.UpdateSchedule)
	schedules.DELETE("/:id", h.DeleteSchedule)
}

// ---- Task request types ----

// CreateTaskRequest defines the request body for creating a backup task.
type CreateTaskRequest struct {
	TaskType   TaskType `json:"task_type" binding:"required"`
	TargetType string   `json:"target_type" binding:"required"`
	TargetIDs  []string `json:"target_ids"`
}

// ---- Schedule request types ----

// CreateScheduleRequest defines the request body for creating a backup schedule.
type CreateScheduleRequest struct {
	Name       string   `json:"name" binding:"required"`
	CronExpr   string   `json:"cron_expr" binding:"required"`
	Enabled    bool     `json:"enabled"`
	TaskType   TaskType `json:"task_type" binding:"required"`
	TargetType string   `json:"target_type"`
	TargetIDs  []string `json:"target_ids"`
}

// UpdateScheduleRequest defines the request body for updating a backup schedule.
type UpdateScheduleRequest struct {
	Name       string   `json:"name" binding:"required"`
	CronExpr   string   `json:"cron_expr" binding:"required"`
	Enabled    bool     `json:"enabled"`
	TaskType   TaskType `json:"task_type" binding:"required"`
	TargetType string   `json:"target_type"`
	TargetIDs  []string `json:"target_ids"`
}

// ---- Task handlers ----

// ListTasks handles GET /api/v1/backup/tasks.
func (h *Handler) ListTasks(c *gin.Context) {
	filter := TaskFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := TaskStatus(status)
		filter.Status = &s
	}
	if taskType := c.Query("task_type"); taskType != "" {
		t := TaskType(taskType)
		filter.TaskType = &t
	}

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateTask handles POST /api/v1/backup/tasks.
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task := &BackupTask{
		TaskType:   req.TaskType,
		TargetType: req.TargetType,
		TargetIDs:  req.TargetIDs,
	}

	created, err := h.service.CreateTask(c.Request.Context(), task)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetTask handles GET /api/v1/backup/tasks/:id.
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

// DeleteTask handles DELETE /api/v1/backup/tasks/:id.
func (h *Handler) DeleteTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CancelTask handles POST /api/v1/backup/tasks/:id/cancel.
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

// ---- Schedule handlers ----

// ListSchedules handles GET /api/v1/backup/schedules.
func (h *Handler) ListSchedules(c *gin.Context) {
	filter := ScheduleFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if enabled := c.Query("enabled"); enabled != "" {
		b := enabled == "true"
		filter.Enabled = &b
	}

	result, err := h.service.ListSchedules(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateSchedule handles POST /api/v1/backup/schedules.
func (h *Handler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	schedule := &BackupSchedule{
		Name:     req.Name,
		CronExpr: req.CronExpr,
		Enabled:  req.Enabled,
		TaskType: req.TaskType,
		TargetIDs: req.TargetIDs,
	}
	if req.TargetType != "" {
		schedule.TargetType = &req.TargetType
	}

	created, err := h.service.CreateSchedule(c.Request.Context(), schedule)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateSchedule handles PUT /api/v1/backup/schedules/:id.
func (h *Handler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	schedule := &BackupSchedule{
		Name:     req.Name,
		CronExpr: req.CronExpr,
		Enabled:  req.Enabled,
		TaskType: req.TaskType,
		TargetIDs: req.TargetIDs,
	}
	if req.TargetType != "" {
		schedule.TargetType = &req.TargetType
	}

	updated, err := h.service.UpdateSchedule(c.Request.Context(), id, schedule)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteSchedule handles DELETE /api/v1/backup/schedules/:id.
func (h *Handler) DeleteSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteSchedule(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}
