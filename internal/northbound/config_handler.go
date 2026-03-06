package northbound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/omcr/device"
	"go.uber.org/zap"
)

// ConfigHandler wraps DeviceParameterRepository for northbound config snapshot export.
type ConfigHandler struct {
	paramRepo device.DeviceParameterRepository
	logger    *zap.Logger
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(paramRepo device.DeviceParameterRepository, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{
		paramRepo: paramRepo,
		logger:    logger,
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

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), deviceID)
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
