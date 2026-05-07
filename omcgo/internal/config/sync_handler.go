package config

import (
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

// SyncHandler provides HTTP endpoints for configuration parameter sync operations.
type SyncHandler struct {
	taskSvc task.Enqueuer
	logger  *zap.Logger
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(taskSvc task.Enqueuer, logger *zap.Logger) *SyncHandler {
	return &SyncHandler{
		taskSvc: taskSvc,
		logger:  logger.Named("config-sync"),
	}
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
