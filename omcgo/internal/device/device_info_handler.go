package device

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// DeviceInfoHandler provides HTTP handlers for device extended info management.
type DeviceInfoHandler struct {
	service     *DeviceService
	permService VisibleGroupsResolver
}

// NewDeviceInfoHandler creates a new device info REST API handler.
func NewDeviceInfoHandler(service *DeviceService) *DeviceInfoHandler {
	return &DeviceInfoHandler{service: service}
}

// SetPermissionService wires the data permission service for per-device
// group-membership (IDOR) checks on by-ID read endpoints.
func (h *DeviceInfoHandler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permService = ps
}

// RegisterRoutes registers device info routes on the given router group.
func (h *DeviceInfoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("/enums", h.GetEnums)
		devices.GET("/:id/info", h.GetDeviceInfo)
		devices.GET("/:id/detail", h.GetDeviceDetail)
		devices.GET("/:id/control-actions", h.GetDeviceControlActions)
		devices.GET("/:id/antenna-sectors", h.GetAntennaSectors)
		devices.PUT("/:id/antenna-sectors/:sectorNo", h.UpdateAntennaSectorPlan)
		devices.PUT("/:id/info", h.UpdateDeviceInfo)
		devices.PUT("/:id/activate", h.ActivateDevice)
		devices.PUT("/:id/deactivate", h.DeactivateDevice)
	}
}

// GetDeviceControlActions returns the durable OMC-control evidence for one device.
func (h *DeviceInfoHandler) GetDeviceControlActions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	result, err := h.service.ListDeviceControlHistory(c.Request.Context(), id, page, pageSize)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// UpdateAntennaSectorPlan saves OMC-local planning values for one sector.
func (h *DeviceInfoHandler) UpdateAntennaSectorPlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	sectorNumber, err := strconv.Atoi(c.Param("sectorNo"))
	if err != nil || sectorNumber < 1 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}
	var req UpdateAntennaSectorPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	sector, err := h.service.UpdateAntennaSectorPlan(c.Request.Context(), id, sectorNumber, req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, sector)
}

// GetDeviceInfo handles GET /api/v1/devices/:id/info.
func (h *DeviceInfoHandler) GetDeviceInfo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
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

	response.OK(c, info)
}

// GetDeviceDetail handles GET /api/v1/devices/:id/detail.
// Returns a composite view aggregating device, device_info, and device_parameters data.
func (h *DeviceInfoHandler) GetDeviceDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}

	composite, err := h.service.GetDeviceDetailComposite(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if composite == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	response.OK(c, composite)
}

// GetAntennaSectors handles GET /api/v1/devices/:id/antenna-sectors.
func (h *DeviceInfoHandler) GetAntennaSectors(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}

	sectors, err := h.service.GetAntennaSectors(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if sectors == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	response.OK(c, sectors)
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
		// 设备不存在 / 无 device_info 行 → repo 返 ErrNotFound → 映射 404；
		// 其余内部错误仍走 500（issue #145 A 项）。
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithMsg(c, nil, "updated")
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

	response.OKWithMsg(c, nil, "device activated")
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

	response.OKWithMsg(c, nil, "device deactivated")
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
			{Value: "inactive", Label: "未激活"},
			{Value: "fault", Label: "故障"},
			{Value: "decommissioned", Label: "退服"},
		},
		"mme_status": {
			{Value: "connected", Label: "已连接"},
			{Value: "partial", Label: "部分连接"},
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
		"gps_status": {
			{Value: "normal", Label: "正常"},
			{Value: "abnormal", Label: "异常"},
			{Value: "no_signal", Label: "无信号"},
		},
		"alarm_severity": {
			{Value: "Critical", Label: "紧急"},
			{Value: "Major", Label: "重要"},
			{Value: "Minor", Label: "次要"},
			{Value: "Warning", Label: "告警"},
		},
		"license_status": {
			{Value: "active", Label: "有效"},
			{Value: "expiring", Label: "即将过期"},
			{Value: "expired", Label: "已过期"},
		},
	}

	response.OK(c, enums)
}
