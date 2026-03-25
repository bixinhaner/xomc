package config

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// SyncHandler provides HTTP endpoints for configuration parameter sync operations.
type SyncHandler struct {
	cmdQueue cmdqueue.CommandQueue
	logger   *zap.Logger
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(cmdQueue cmdqueue.CommandQueue, logger *zap.Logger) *SyncHandler {
	return &SyncHandler{
		cmdQueue: cmdQueue,
		logger:   logger.Named("config-sync"),
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

	cmd := &cmdqueue.Command{
		Method: "SetParameterValues",
		Params: params,
	}

	if err := h.cmdQueue.Push(c.Request.Context(), deviceID, cmd); err != nil {
		logger.L(c.Request.Context()).Error("push config command", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "configuration push queued",
		"device_id":  deviceID,
		"command_id": cmd.ID,
	})
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

	cmd := &cmdqueue.Command{
		Method: "GetParameterValues",
		Params: params,
	}

	if err := h.cmdQueue.Push(c.Request.Context(), deviceID, cmd); err != nil {
		logger.L(c.Request.Context()).Error("pull config command", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "configuration pull queued",
		"device_id":  deviceID,
		"command_id": cmd.ID,
	})
}

// GetSyncStatus returns the number of pending commands for a device.
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	count, err := h.cmdQueue.Len(c.Request.Context(), deviceID)
	if err != nil {
		logger.L(c.Request.Context()).Error("get sync status", zap.String("device_id", deviceID), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, SyncStatusResponse{
		DeviceID:     deviceID,
		PendingCount: count,
	})
}
