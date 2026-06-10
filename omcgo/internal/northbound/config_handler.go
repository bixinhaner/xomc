package northbound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// ConfigHandler handles northbound config snapshot export endpoints.
type ConfigHandler struct {
	svc    *NorthboundService
	logger *zap.Logger
	scoper *Scoper // 多租户隔离；nil 时退化为不隔离（由 Router.SetScoper 注入）
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
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return
	}

	// IDOR / 跨租户守卫：非超管只能导出其可见设备组内设备的配置；
	// 越权（如 CTCC 用户访问 CMCC 设备）→ 403，已写响应直接返回。
	if h.scoper != nil && !h.scoper.AuthorizeDevice(c, deviceID) {
		return
	}

	params, err := h.svc.ExportConfig(c.Request.Context(), deviceID)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound config export failed",
			zap.String("device_id", deviceIDStr),
			zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "config export failed")
		return
	}

	response.OK(c, gin.H{
		"device_id":  deviceIDStr,
		"parameters": params,
		"total":      len(params),
	})
}
