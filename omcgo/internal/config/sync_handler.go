package config

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/task"
)

// GPVBatcher 是 PullConfig 拿到批次拆分能力的窄接口（避免 config → provision 循环 import）。
// 由 provision.SyncService 实现，DI 时按接口注入。
type GPVBatcher interface {
	// EnqueueGPVBatches 按统一 batchSize 拆分 paths 并入队 GetParameterValues 任务。
	// 返回入队的 task ID 列表。
	EnqueueGPVBatches(ctx context.Context, deviceSN string, paramPaths []string, sourceID string) ([]string, error)
}

// DeviceExistenceChecker 是 push/pull 入队前做设备存在性预检的窄接口
// （避免 config → device 强耦合，DI 时按接口注入 device.DeviceRepository）。
// ExistsBySerialNumber 返回 (true,nil) 表示设备存在；(false,nil) 表示不存在；
// err != nil 表示查库失败（与"不存在"区分，前者映射 500、后者 404）。
type DeviceExistenceChecker interface {
	ExistsBySerialNumber(ctx context.Context, sn string) (bool, error)
}

// SyncHandler provides HTTP endpoints for configuration parameter sync operations.
type SyncHandler struct {
	taskSvc    task.Enqueuer
	gpvBatcher GPVBatcher
	deviceChk  DeviceExistenceChecker
	logger     *zap.Logger
}

// NewSyncHandler creates a new SyncHandler.
// gpvBatcher 可为 nil（AutoSync 未启用时）；nil 时 PullConfig 降级为单 task 不拆批。
// deviceChk 可为 nil（向后兼容/未注入时跳过存在性预检）；非 nil 时 push/pull 入队前校验设备存在。
func NewSyncHandler(taskSvc task.Enqueuer, gpvBatcher GPVBatcher, deviceChk DeviceExistenceChecker, logger *zap.Logger) *SyncHandler {
	return &SyncHandler{
		taskSvc:    taskSvc,
		gpvBatcher: gpvBatcher,
		deviceChk:  deviceChk,
		logger:     logger.Named("config-sync"),
	}
}

// ensureDeviceExists 在入队前预检设备存在性（按 SN）。
// 返回 true 表示放行；返回 false 表示已写出错误响应，调用方应直接 return。
// deviceChk 未注入时放行（向后兼容，不阻断既有调用方）。
func (h *SyncHandler) ensureDeviceExists(c *gin.Context, deviceSN string) bool {
	if h.deviceChk == nil {
		return true
	}
	exists, err := h.deviceChk.ExistsBySerialNumber(c.Request.Context(), deviceSN)
	if err != nil {
		logger.L(c.Request.Context()).Error("check device existence",
			zap.String("device_id", deviceSN), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, commonerrors.ErrInternal)
		return false
	}
	if !exists {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return false
	}
	return true
}

// PushConfigRequest is the input for pushing configuration parameters to a device.
type PushConfigRequest struct {
	Parameters []ParameterValue `json:"parameters" binding:"required,min=1"`
}

// ParameterValue represents a TR069 parameter name-value pair.
type ParameterValue struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
	Type  string `json:"type"` // string, int, boolean, etc.
}

// PullConfigRequest is the input for pulling configuration parameters from a device.
type PullConfigRequest struct {
	ParameterNames []string `json:"parameter_names" binding:"required,min=1"`
}

// SyncStatusResponse contains the sync status for a device.
type SyncStatusResponse struct {
	DeviceID     string `json:"device_id"`
	PendingCount int64  `json:"pending_count"`
}

// RegisterRoutes registers the config sync routes on the given router group.
func (h *SyncHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sync := rg.Group("/config/sync")
	{
		sync.POST("/push/:deviceId", h.PushConfig)
		sync.POST("/pull/:deviceId", h.PullConfig)
		sync.GET("/status/:deviceId", h.GetSyncStatus)
	}
}

// PushConfig queues SetParameterValues commands for a device.
func (h *SyncHandler) PushConfig(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req PushConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 设备存在性预检：不存在则 404，避免合法 body 入队产生孤儿任务（issue #126 第3项）。
	if !h.ensureDeviceExists(c, deviceID) {
		return
	}

	params, err := json.Marshal(req.Parameters)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	created, err := h.taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN: deviceID,
		Method:   "SetParameterValues",
		Params:   params,
		Source:   task.TaskSourceAPI,
	})

	// Cross-module audit: ActionConfig / category 2 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionConfig
	entry.ResourceType = audit.ResourceConfig
	entry.ResourceID = deviceID
	entry.Details = map[string]interface{}{
		"method":    "SetParameterValues",
		"param_cnt": len(req.Parameters),
	}
	entry.Success = err == nil
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.LogAsync(entry)

	if err != nil {
		logger.L(c.Request.Context()).Error("push config command", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, gin.H{
		"device_id":  deviceID,
		"command_id": created.ID,
	}, "configuration push queued")
}

// PullConfig queues GetParameterValues commands for a device.
//
// 走 GPVBatcher：按统一 batchSize 拆批，commandKey 用 "sync-gpv-{sn}-{i}" 前缀，
// 即可享受 ACS handler.tryRecoverGPVFault 的 Fault 自愈循环（剔除坏 path 续查）。
func (h *SyncHandler) PullConfig(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req PullConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 设备存在性预检：不存在则 404，避免合法 body 入队产生孤儿任务（issue #126 第3项）。
	if !h.ensureDeviceExists(c, deviceID) {
		return
	}

	if h.gpvBatcher != nil {
		taskIDs, err := h.gpvBatcher.EnqueueGPVBatches(c.Request.Context(), deviceID, req.ParameterNames, "")
		if err != nil {
			logger.L(c.Request.Context()).Error("pull config enqueue batches",
				zap.String("device_id", deviceID), zap.Error(err))
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		firstID := ""
		if len(taskIDs) > 0 {
			firstID = taskIDs[0]
		}
		response.OKWithMsg(c, gin.H{
			"device_id":   deviceID,
			"command_id":  firstID,
			"task_ids":    taskIDs,
			"batch_count": len(taskIDs),
		}, "configuration pull queued")
		return
	}

	// 兜底：AutoSync 未启用 → 批次拆分能力不可用，退回旧的单 task 路径。
	params, err := json.Marshal(req.ParameterNames)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	created, err := h.taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN: deviceID,
		Method:   "GetParameterValues",
		Params:   params,
		Source:   task.TaskSourceAPI,
	})
	if err != nil {
		logger.L(c.Request.Context()).Error("pull config command", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, gin.H{
		"device_id":  deviceID,
		"command_id": created.ID,
	}, "configuration pull queued")
}

// GetSyncStatus returns the number of pending commands for a device.
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	count, err := h.taskSvc.GetQueueLength(c.Request.Context(), deviceID)
	if err != nil {
		logger.L(c.Request.Context()).Error("get sync status", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, SyncStatusResponse{
		DeviceID:     deviceID,
		PendingCount: count,
	})
}
