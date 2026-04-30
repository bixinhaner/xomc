package backup

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for backup management REST API.
type Handler struct {
	service        *Service
	ftpRepo        FTPConfigRepository
	policyService  *PolicyService       // T-0071; nil-safe (UpdatePolicy/GetPolicy return 503 if unset)
	restoreService *RestoreService      // T-0072; nil-safe (restore endpoints return 503 if unset)
	ftpTester      *FTPConnectionTester // T-0032; nil-safe (TestFTPConnection returns stub when unset)
	logger         *zap.Logger
}

// NewHandler creates a new backup Handler.
// policyService is optional during incremental rollout; pass nil to disable
// the /backup/policy endpoint group, or use SetPolicyService after construction.
func NewHandler(service *Service, ftpRepo FTPConfigRepository, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		ftpRepo: ftpRepo,
		logger:  logger.Named("backup-handler"),
	}
}

// SetPolicyService wires the BackupPolicy CRUD service post-construction.
// Used by DI to avoid changing the existing NewHandler signature.
func (h *Handler) SetPolicyService(s *PolicyService) {
	h.policyService = s
}

// SetFTPTester wires the FTP connection-test service (T-0032). When nil,
// the TestFTPConnection endpoint reports a clear "not wired" message
// instead of running a real probe.
func (h *Handler) SetFTPTester(t *FTPConnectionTester) {
	h.ftpTester = t
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

	ftp := rg.Group("/backup/ftp-configs")
	ftp.GET("", h.ListFTPConfigs)
	ftp.POST("", h.CreateFTPConfig)
	ftp.PUT("/:id", h.UpdateFTPConfig)
	ftp.DELETE("/:id", h.DeleteFTPConfig)
	ftp.POST("/:id/test", h.TestFTPConnection)

	// T-0071 / R-102 followup: singleton policy endpoint.
	policy := rg.Group("/backup/policy")
	policy.GET("", h.GetPolicy)
	policy.PUT("", h.UpdatePolicy)

	// T-0072 / R-102 followup: restore endpoint group + restore_tasks listing.
	// Routes are always registered — handlers themselves return 503 if the
	// restore service is not wired (mirrors policy nil-safety).
	restore := rg.Group("/backup")
	restore.POST("/restore", h.CreateRestore)
	// T-0079: restore_by_task_id mode — same fan-out, but resolves bucket +
	// object_path automatically from backup_tasks.file_path linkage.
	restore.POST("/restore/by-task-id", h.CreateRestoreByTaskID)
	restore.GET("/restore-tasks", h.ListRestoreTasks)
	restore.GET("/restore-tasks/:id", h.GetRestoreTask)
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
		Name:      req.Name,
		CronExpr:  req.CronExpr,
		Enabled:   req.Enabled,
		TaskType:  req.TaskType,
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
		Name:      req.Name,
		CronExpr:  req.CronExpr,
		Enabled:   req.Enabled,
		TaskType:  req.TaskType,
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

// ---- FTP Config request types ----

// CreateFTPConfigRequest defines the request body for creating an FTP config.
type CreateFTPConfigRequest struct {
	ConfigName        string  `json:"config_name" binding:"required"`
	Host              string  `json:"host" binding:"required"`
	Port              int     `json:"port"`
	Username          string  `json:"username" binding:"required"`
	PasswordEncrypted *string `json:"password_encrypted"`
	Protocol          string  `json:"protocol"`
	RemotePath        string  `json:"remote_path"`
	Passive           bool    `json:"passive"`
	Enabled           bool    `json:"enabled"`
}

// UpdateFTPConfigRequest defines the request body for updating an FTP config.
type UpdateFTPConfigRequest struct {
	ConfigName        string  `json:"config_name" binding:"required"`
	Host              string  `json:"host" binding:"required"`
	Port              int     `json:"port"`
	Username          string  `json:"username" binding:"required"`
	PasswordEncrypted *string `json:"password_encrypted"`
	Protocol          string  `json:"protocol"`
	RemotePath        string  `json:"remote_path"`
	Passive           bool    `json:"passive"`
	Enabled           bool    `json:"enabled"`
}

// ---- FTP Config handlers ----

// ListFTPConfigs handles GET /api/v1/backup/ftp-configs.
func (h *Handler) ListFTPConfigs(c *gin.Context) {
	filter := FTPConfigFilter{
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

	result, err := h.ftpRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateFTPConfig handles POST /api/v1/backup/ftp-configs.
func (h *Handler) CreateFTPConfig(c *gin.Context) {
	var req CreateFTPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	config := &FTPConfig{
		ConfigName:        req.ConfigName,
		Host:              req.Host,
		Port:              req.Port,
		Username:          req.Username,
		PasswordEncrypted: req.PasswordEncrypted,
		Protocol:          req.Protocol,
		RemotePath:        req.RemotePath,
		Passive:           req.Passive,
		Enabled:           req.Enabled,
	}
	if config.Port == 0 {
		config.Port = 21
	}
	if config.Protocol == "" {
		config.Protocol = "FTP"
	}
	if config.RemotePath == "" {
		config.RemotePath = "/"
	}

	if err := h.ftpRepo.Create(c.Request.Context(), config); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateFTPConfig handles PUT /api/v1/backup/ftp-configs/:id.
func (h *Handler) UpdateFTPConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateFTPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	existing, err := h.ftpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	existing.ConfigName = req.ConfigName
	existing.Host = req.Host
	existing.Port = req.Port
	existing.Username = req.Username
	existing.PasswordEncrypted = req.PasswordEncrypted
	existing.Protocol = req.Protocol
	existing.RemotePath = req.RemotePath
	existing.Passive = req.Passive
	existing.Enabled = req.Enabled

	if existing.Port == 0 {
		existing.Port = 21
	}
	if existing.Protocol == "" {
		existing.Protocol = "FTP"
	}
	if existing.RemotePath == "" {
		existing.RemotePath = "/"
	}

	if err := h.ftpRepo.Update(c.Request.Context(), existing); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteFTPConfig handles DELETE /api/v1/backup/ftp-configs/:id.
func (h *Handler) DeleteFTPConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.ftpRepo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// TestFTPConnection handles POST /api/v1/backup/ftp-configs/:id/test.
//
// T-0032: when FTPConnectionTester is wired (SetFTPTester), runs a
// two-tier probe (TCP reachability + protocol-specific auth) and
// returns FTPTestResult JSON. When unwired, returns a clear "not
// wired" stub so existing callers still get HTTP 200.
func (h *Handler) TestFTPConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	cfg, err := h.ftpRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if h.ftpTester == nil {
		c.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "FTP connection tester not wired (DI gap; see T-0032)",
			"protocol": cfg.Protocol,
			"host":     cfg.Host,
			"port":     cfg.Port,
		})
		return
	}

	result := h.ftpTester.Test(c.Request.Context(), cfg)
	c.JSON(http.StatusOK, result)
}
