package backup

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/device"
)

// Handler provides HTTP handlers for backup management REST API.
type Handler struct {
	service         *Service
	ftpRepo         FTPConfigRepository
	policyService   *PolicyService       // T-0071; nil-safe (UpdatePolicy/GetPolicy return 503 if unset)
	restoreService  *RestoreService      // T-0072; nil-safe (restore endpoints return 503 if unset)
	snapshotService *SnapshotService     // T-0164; nil-safe (config-snapshot endpoints return 503 if unset)
	licenseService  *LicenseService      // T-0165; nil-safe (device-license endpoints return 503 if unset)
	ftpTester       *FTPConnectionTester // T-0032; nil-safe (TestFTPConnection returns stub when unset)
	// M4: ExportFile presigned URL support
	fileRepo    FileRepository // nil-safe (ExportFile returns 503 if unset)
	minioClient *minio.Client  // nil-safe (ExportFile returns 503 if unset)
	// issue #548 切片 4：sys_configs 写入 storage.minio_public_endpoint 后
	// presignProvider.Get() 返回新 endpoint 对应的 client；优先级高于 minioClient。
	// 由 provider/modules.go 注入 PresignBridge；nil 时回退 minioClient。
	presignProvider PresignClientProvider // nil-safe (M1补丁法：仅在 PresignBridge 未装配时为 nil)
	// M4: device repo used by QueryCellInfos / QueryTaskDeviceList / GetProductType.
	// nil-safe (those endpoints return 503 if unset).
	deviceReader device.DeviceReader
	logger       *zap.Logger
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

// SetFileRepository wires the BackupRestoreFile metadata repo used by
// ExportFile (M4). When nil, ExportFile returns 503.
func (h *Handler) SetFileRepository(r FileRepository) {
	h.fileRepo = r
}

// SetMinioClient wires the MinIO client used to generate presigned GET URLs
// in ExportFile (M4). When nil, ExportFile returns 503.
func (h *Handler) SetMinioClient(c *minio.Client) {
	h.minioClient = c
}

// PresignClientProvider 抽象"按需取当前 MinIO 预签名 client"的能力（issue #548 切片 4）。
// 主线生产实现是 internal/core/components/minio.PresignBridge——sys_configs 写入
// storage.minio_public_endpoint 后原子替换内部 client，下一次 Get() 拿到新端点的签名 client。
// 不要把 Get() 结果缓存跨请求用。
type PresignClientProvider interface {
	Get() *minio.Client
}

// SetPresignProvider wires the runtime-aware MinIO presign client provider
// (issue #548 切片 4). When set, license / snapshot / M4 导出下载 URL 会用
// provider.Get() 返的当前 client 签发，实时响应 sys_configs 热改。nil 时回退
// h.minioClient（启动期注入的静态 client）。
func (h *Handler) SetPresignProvider(p PresignClientProvider) {
	h.presignProvider = p
}

// presignClient 返回当前该用于签发预签名 URL 的 client。优先 provider.Get()，
// 其次 h.minioClient。两者都不可用时返 nil（调用方负责报 503）。
func (h *Handler) presignClient() *minio.Client {
	if h.presignProvider != nil {
		if c := h.presignProvider.Get(); c != nil {
			return c
		}
	}
	return h.minioClient
}

// SetDeviceReader wires the device repository used by M4 device-facing
// endpoints (QueryCellInfos / QueryTaskDeviceList / GetProductType).
// When nil, those endpoints return 503.
func (h *Handler) SetDeviceReader(r device.DeviceReader) {
	h.deviceReader = r
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
	// T-0164 B5: restore_by_snapshot mode — per-device picks latest snapshot.
	restore.POST("/restore/by-snapshot", h.CreateRestoreBySnapshot)
	restore.GET("/restore-tasks", h.ListRestoreTasks)
	restore.GET("/restore-tasks/:id", h.GetRestoreTask)

	// T-0164 B4: config snapshots endpoints (single source of latest config per
	// device). Routes are always registered; handlers 503 if snapshotService nil.
	snap := rg.Group("/backup/config-snapshots")
	snap.GET("", h.ListSnapshots)
	snap.POST("/batch-get", h.BatchGetSnapshots)
	snap.POST("/validate-sns", h.ValidateImportSNs)
	snap.POST("/import", h.ImportSnapshots)
	// 批量删除走 POST /batch-delete 而不是 DELETE with body —— DELETE 携带 body
	// 在某些代理 / WAF 下会被吞，且 axios 在某些 adapter 下也不一定可靠。
	snap.POST("/batch-delete", h.BatchDeleteSnapshots)
	snap.GET("/:sn", h.GetSnapshot)
	snap.GET("/:sn/download", h.DownloadSnapshot)
	snap.DELETE("/:sn", h.DeleteSnapshot)

	// T-0165: device license endpoints (与 config-snapshots 同款 CRUD)。
	// 与 LICENSE_UPGRADE UFTE 任务配套：库里管理 license 文件，任务下发时按 SN 取最新。
	lic := rg.Group("/backup/device-licenses")
	lic.GET("", h.ListLicenses)
	lic.POST("/batch-get", h.BatchGetLicenses)
	lic.POST("/validate-sns", h.ValidateLicenseSNs)
	lic.POST("/import", h.ImportLicenses)
	lic.POST("/batch-delete", h.BatchDeleteLicenses)
	lic.GET("/:sn", h.GetLicense)
	lic.GET("/:sn/download", h.DownloadLicense)
	lic.DELETE("/:sn", h.DeleteLicense)

	// M4: 运营商规范 API 别名 — /task/enb/config/backupRestore/*
	// 保留 /api/v1/backup/* 原路由，此处仅增加别名前缀，不修改处理逻辑。
	alias := rg.Group("/task/enb/config/backupRestore")
	alias.POST("/addBackupRestoreTask", h.CreateTask)
	alias.POST("/queryTaskList", h.ListTasksAlias)
	alias.POST("/terminateTask", h.TerminateTask)
	alias.POST("/single/importFile", h.CreateRestore)
	alias.GET("/single/exportFile", h.ExportFile)
	alias.POST("/queryCellInfos", h.QueryCellInfos)
	alias.POST("/queryTaskDeviceList", h.QueryTaskDeviceList)
	alias.GET("/getProductType", h.GetProductType)
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

	response.OK(c, result)
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

	response.OKWithStatus(c, http.StatusCreated, created)
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

	response.OK(c, task)
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

	response.OK(c, nil)
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

	response.OK(c, gin.H{"status": "cancelled"})
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

	response.OK(c, result)
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

	response.OKWithStatus(c, http.StatusCreated, created)
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

	response.OK(c, updated)
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

	response.OK(c, nil)
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

	response.OK(c, result)
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

	response.OKWithStatus(c, http.StatusCreated, config)
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

	response.OK(c, existing)
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

	response.OK(c, nil)
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
		response.OK(c, gin.H{
			"success":  false,
			"message":  "FTP connection tester not wired (DI gap; see T-0032)",
			"protocol": cfg.Protocol,
			"host":     cfg.Host,
			"port":     cfg.Port,
		})
		return
	}

	result := h.ftpTester.Test(c.Request.Context(), cfg)
	response.OK(c, result)
}
