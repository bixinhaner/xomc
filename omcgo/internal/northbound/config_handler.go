package northbound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConfigHandler handles northbound config snapshot export endpoints.
type ConfigHandler struct {
	svc    *NorthboundService
	logger *zap.Logger
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(svc *NorthboundService, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{
		svc:    svc,
		logger: logger,
	}
}

// ExportConfig handles config snapshot export for a specific device.
func (h *ConfigHandler) ExportConfig(c *gin.Context) {
	deviceIDStr := c.Param("deviceId")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}

	params, err := h.svc.ExportConfig(c.Request.Context(), deviceID)
	if err != nil {
		h.logger.Error("northbound config export failed",
			zap.String("device_id", deviceIDStr),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "config export failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":  deviceIDStr,
		"parameters": params,
		"total":      len(params),
	})
}
