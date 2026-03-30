package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceInfoHandler provides HTTP handlers for device extended info management.
type DeviceInfoHandler struct {
	service *DeviceService
}

// NewDeviceInfoHandler creates a new device info REST API handler.
func NewDeviceInfoHandler(service *DeviceService) *DeviceInfoHandler {
	return &DeviceInfoHandler{service: service}
}

// RegisterRoutes registers device info routes on the given router group.
func (h *DeviceInfoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("/enums", h.GetEnums)
		devices.GET("/:id/info", h.GetDeviceInfo)
		devices.PUT("/:id/info", h.UpdateDeviceInfo)
		devices.PUT("/:id/activate", h.ActivateDevice)
		devices.PUT("/:id/deactivate", h.DeactivateDevice)
	}
}

// GetDeviceInfo handles GET /api/v1/devices/:id/info.
func (h *DeviceInfoHandler) GetDeviceInfo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	info, err := h.service.GetDeviceInfo(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if info == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, info)
}

// UpdateDeviceInfo handles PUT /api/v1/devices/:id/info.
func (h *DeviceInfoHandler) UpdateDeviceInfo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateDeviceInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Extract username from JWT context (set by auth middleware)
	updater, _ := c.Get("username")
	updaterStr, _ := updater.(string)

	if err := h.service.UpdateDeviceInfo(c.Request.Context(), id, req, updaterStr); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// ActivateDevice handles PUT /api/v1/devices/:id/activate.
func (h *DeviceInfoHandler) ActivateDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.TransitionStatus(c.Request.Context(), id, model.DeviceActive); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device activated"})
}

// DeactivateDevice handles PUT /api/v1/devices/:id/deactivate.
func (h *DeviceInfoHandler) DeactivateDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.TransitionStatus(c.Request.Context(), id, model.DeviceMaintenance); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device deactivated"})
}

// enumItem represents a single enum option.
type enumItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// GetEnums handles GET /api/v1/devices/enums.
func (h *DeviceInfoHandler) GetEnums(c *gin.Context) {
	enums := map[string][]enumItem{
		"carrier": {
			{Value: "cmcc", Label: "中国移动"},
			{Value: "ctcc", Label: "中国电信"},
			{Value: "cucc", Label: "中国联通"},
		},
		"technology": {
			{Value: "lte", Label: "LTE (4G)"},
			{Value: "nr", Label: "NR (5G)"},
		},
		"status": {
			{Value: "discovered", Label: "已发现"},
			{Value: "registered", Label: "已注册"},
			{Value: "provisioning", Label: "开通中"},
			{Value: "active", Label: "在线"},
			{Value: "offline", Label: "离线"},
			{Value: "maintenance", Label: "维护中"},
			{Value: "decommissioned", Label: "已退服"},
		},
		"rf_status": {
			{Value: "on", Label: "开启"},
			{Value: "off", Label: "关闭"},
			{Value: "error", Label: "异常"},
		},
		"cell_status": {
			{Value: "normal", Label: "正常"},
			{Value: "fault", Label: "故障"},
			{Value: "unconfigured", Label: "未配置"},
			{Value: "decommissioned", Label: "退服"},
		},
		"mme_status": {
			{Value: "normal", Label: "正常"},
			{Value: "error", Label: "异常"},
			{Value: "disconnected", Label: "未连接"},
		},
		"sync_status": {
			{Value: "gps", Label: "GPS同步"},
			{Value: "beidou", Label: "北斗同步"},
			{Value: "ntp", Label: "NTP同步"},
			{Value: "error", Label: "异常"},
		},
		"project_status": {
			{Value: "building", Label: "在建"},
			{Value: "delivered", Label: "已交付"},
			{Value: "operating", Label: "运维中"},
			{Value: "deactivated", Label: "停用"},
		},
	}

	c.JSON(http.StatusOK, enums)
}
