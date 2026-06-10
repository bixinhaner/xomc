package device

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// VisibleGroupsResolver resolves which device groups a user can see.
// v1.0：第三参数从 carrier 改为 isSuperAdmin（详见 docs/prd/system/users.md §11.11）。
type VisibleGroupsResolver interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error)
}

// Handler provides HTTP handlers for device management REST API.
type Handler struct {
	service     *DeviceService
	permService VisibleGroupsResolver
}

// authorizeDeviceAccess 是所有"按 ID 直读"设备端点共享的越权（IDOR）守卫。
//
// 流程：从 gin ctx 取 userID + isSuperAdmin → permService.GetUserVisibleGroupIDs
// 解析可见设备组 → DeviceService.AuthorizeDeviceGroupAccess 判定该设备是否在
// 可见集合内。返回 true 表示放行；返回 false 表示已 abort（403/500），caller 应直接 return。
//
// nil-safe：permService 为 nil（dev/test 未注入数据权限）→ 放行，保持与
// ListDevices 同样的退化语义（数据权限未配置时不拦截）。
func authorizeDeviceAccess(c *gin.Context, svc *DeviceService, permService VisibleGroupsResolver, deviceID uuid.UUID) bool {
	if permService == nil {
		return true
	}
	userIDVal, _ := c.Get(admin.CtxKeyUserID)
	uid, ok := userIDVal.(uuid.UUID)
	if !ok {
		// 已过鉴权中间件却拿不到 user_id：按拒绝处理，不泄露设备数据。
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return false
	}
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)

	visibleGroups, err := permService.GetUserVisibleGroupIDs(c.Request.Context(), uid, isSuper)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return false
	}
	if authErr := svc.AuthorizeDeviceGroupAccess(c.Request.Context(), deviceID, visibleGroups); authErr != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(authErr), authErr)
		return false
	}
	return true
}

// NewHandler creates a new device REST API handler.
func NewHandler(service *DeviceService) *Handler {
	return &Handler{service: service}
}

// SetPermissionService sets the data permission service for group-based filtering.
func (h *Handler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permService = ps
}

// RegisterRoutes registers device routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("", h.ListDevices)
		devices.GET("/product-classes", h.ListProductClasses)
		devices.GET("/stats", h.GetStats)
		devices.GET("/geo", h.ListGeo)           // Map device geo data
		devices.GET("/geo/stats", h.GetGeoStats) // Map device statistics
		devices.GET("/search", h.SearchDevices)  // Search devices for map
		// Batch routes must be registered before /:id to avoid path conflicts
		devices.DELETE("/batch", h.BatchDeleteDevices)
		devices.POST("/batch-reboot", h.BatchRebootDevices)
		devices.POST("/batch-import", h.BatchImportDevices)
		// Recycle bin routes
		devices.GET("/recycle", h.ListRecycleBin)
		devices.PATCH("/recycle/restore", h.RestoreDevices)
		devices.DELETE("/recycle/permanent", h.PermanentDeleteDevices)
		devices.GET("/:id", h.GetDevice)
		devices.GET("/:id/parameters", h.GetDeviceParameters)
		devices.POST("", h.CreateDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
		devices.POST("/:id/reboot", h.RebootDevice)
		// T-0126: 旧 /param-sync (Path A) 已下线，替换为 /sync-params (Path B + reason="manual")
		devices.POST("/:id/sync-params", h.SyncDeviceParams)
		devices.PUT("/:id/rf-switch", h.SetRFSwitch)
	}
}

// CreateDeviceRequest defines the request body for creating a device.
//
// 用于"手动预登记"场景：在 CPE 通过 TR-069 Bootstrap 自动注册之前，由运维人员
// 在 FE /device/register 页面录入设备。Bootstrap Inform 到达后，
// DeviceService.RegisterFromInform 会根据 serial_number 找到本行 UPDATE 而非
// INSERT（参 device_service.go RegisterFromInform 注释）。
type CreateDeviceRequest struct {
	SerialNumber string `json:"serial_number" binding:"required"`
	OUI          string `json:"oui" binding:"required"`
	ProductClass string `json:"product_class"`
	Manufacturer string `json:"manufacturer"`
	ModelName    string `json:"model_name"`
	// Carrier / Technology 用 binding oneof 让非法值前置成 400，
	// 比插库后才发现外键违反更友好。
	Carrier    model.CarrierCode `json:"carrier" binding:"required,oneof=cmcc ctcc cucc"`
	Technology model.Technology  `json:"technology" binding:"required,oneof=lte nr"`
	IPAddress  string            `json:"ip_address"`
	DeviceName string            `json:"device_name"`
	SiteID     string            `json:"site_id"`
	Latitude   *float64          `json:"latitude"`
	Longitude  *float64          `json:"longitude"`
}

// UpdateDeviceRequest defines the request body for updating a device.
type UpdateDeviceRequest struct {
	DeviceName *string             `json:"device_name"`
	SiteID     *string             `json:"site_id"`
	ModelName  *string             `json:"model_name"`
	Latitude   *float64            `json:"latitude"`
	Longitude  *float64            `json:"longitude"`
	Status     *model.DeviceStatus `json:"status"`
}

// CreateDevice handles POST /api/v1/devices.
func (h *Handler) CreateDevice(c *gin.Context) {
	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	device, err := h.service.CreateDevice(c.Request.Context(), req)
	if err != nil {
		if err == commonerrors.ErrAlreadyExists {
			commonerrors.AbortWithError(c, http.StatusConflict, err)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, device)
}

// UpdateDevice handles PUT /api/v1/devices/:id.
func (h *Handler) UpdateDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	device, err := h.service.UpdateDevice(c.Request.Context(), id, req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if device == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	response.OK(c, device)
}

// DeleteDevice handles DELETE /api/v1/devices/:id.
func (h *Handler) DeleteDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	delErr := h.service.DeleteDevice(c.Request.Context(), id)

	// Cross-module audit: ActionDelete / category 5 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionDelete
	entry.ResourceType = audit.ResourceDevice
	entry.ResourceID = id.String()
	entry.Success = delErr == nil
	if delErr != nil {
		entry.ErrorMessage = delErr.Error()
	}
	audit.LogAsync(entry)

	if delErr != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, delErr)
		return
	}

	response.OK(c, nil)
}

// ListProductClasses handles GET /api/v1/devices/product-classes.
func (h *Handler) ListProductClasses(c *gin.Context) {
	classes, err := h.service.GetProductClasses(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, classes)
}

// ListDevices handles GET /api/v1/devices with pagination and filtering.
func (h *Handler) ListDevices(c *gin.Context) {
	filter := DeviceFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if carrier := c.Query("carrier"); carrier != "" {
		cc := model.CarrierCode(carrier)
		filter.Carrier = &cc
	}
	if tech := c.Query("technology"); tech != "" {
		t := model.Technology(tech)
		filter.Technology = &t
	}
	// T-0162: 老 ?status= 兼容入口，DeviceFilter.Status 会在 Repository 层翻译
	// 为 lifecycle_state + is_online。新前端代码请走 ?lifecycle_state= / ?is_online=。
	if status := c.Query("status"); status != "" {
		s := model.DeviceStatus(status)
		filter.Status = &s
	}
	// T-0162: lifecycle_state 多选 CSV (?lifecycle_state=commissioned,maintenance)
	if lifecycles := c.Query("lifecycle_state"); lifecycles != "" {
		for _, v := range strings.Split(lifecycles, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			filter.LifecycleState = append(filter.LifecycleState, model.DeviceLifecycle(v))
		}
	}
	// T-0162: is_online 布尔
	if isOnlineStr := c.Query("is_online"); isOnlineStr != "" {
		b := isOnlineStr == "true" || isOnlineStr == "1"
		filter.IsOnline = &b
	}
	// T-0162: 新 3 个筛选维度（设备型号 / 软件版本 / 固件版本，字典 device_model
	// / software_version / firmware_version 提供下拉选项；详见
	// docs/design/device-lifecycle-online-status-decouple-20260520.md §3.2）
	if modelName := c.Query("model_name"); modelName != "" {
		filter.ModelName = &modelName
	}
	if softwareVersion := c.Query("software_version"); softwareVersion != "" {
		filter.SoftwareVersion = &softwareVersion
	}
	if firmwareVersion := c.Query("firmware_version"); firmwareVersion != "" {
		filter.FirmwareVersion = &firmwareVersion
	}
	if oui := c.Query("oui"); oui != "" {
		filter.OUI = &oui
	}
	if sn := c.Query("sn"); sn != "" {
		filter.SN = &sn
	}
	if snList := c.Query("sn_list"); snList != "" {
		// 批量输入：CSV 形式的 SN 列表精确过滤（与 product/class/keyword 等其它筛选条件正交）。
		filter.SNList = SplitCSV(snList)
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}
	if manufacturer := c.Query("manufacturer"); manufacturer != "" {
		filter.Manufacturer = &manufacturer
	}
	if productID := c.Query("product_id"); productID != "" {
		pid, err := uuid.Parse(productID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.ProductID = &pid
	}
	if productClass := c.Query("product_class"); productClass != "" {
		filter.ProductClass = &productClass
	}
	if rfStatus := c.Query("rf_status"); rfStatus != "" {
		filter.RFStatus = &rfStatus
	}
	if cellStatus := c.Query("cell_status"); cellStatus != "" {
		filter.CellStatus = &cellStatus
	}
	if projectStatus := c.Query("project_status"); projectStatus != "" {
		filter.ProjectStatus = &projectStatus
	}
	if gpsStatus := c.Query("gps_status"); gpsStatus != "" {
		filter.GPSStatus = &gpsStatus
	}
	if alarmSeverity := c.Query("alarm_severity"); alarmSeverity != "" {
		filter.AlarmSeverity = &alarmSeverity
	}
	if licenseStatus := c.Query("license_status"); licenseStatus != "" {
		filter.LicenseStatus = &licenseStatus
	}
	if opState := c.Query("op_state"); opState != "" {
		filter.OpState = &opState
	}
	if groupID := c.Query("group_id"); groupID != "" {
		gid, err := uuid.Parse(groupID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.GroupID = &gid
	}

	// Inject data permission: restrict to user-visible groups.
	// v1.0：超管判定从 carrier IS NULL 改为 source = 'builtIn'（来自 ctx CtxKeyIsSuperAdmin）。
	if h.permService != nil {
		userID, _ := c.Get(admin.CtxKeyUserID)
		isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
		isSuper, _ := isSuperVal.(bool)
		if uid, ok := userID.(uuid.UUID); ok {
			visibleGroups, err := h.permService.GetUserVisibleGroupIDs(c.Request.Context(), uid, isSuper)
			if err != nil {
				commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
				return
			}
			filter.VisibleGroups = visibleGroups
		}
	}

	result, err := h.service.ListDevicesWithInfo(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, result)
}

// GetDevice handles GET /api/v1/devices/:id.
func (h *Handler) GetDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}

	// 走 GetDeviceWithInfo（JOIN device_info）：详情页与列表的 op_state（激活
	// 状态）及 rf/mme/sync/小区等扩展字段口径一致，避免详情返回裸 Device。
	device, err := h.service.GetDeviceWithInfo(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if device == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	response.OK(c, device)
}

// GetDeviceParameters handles GET /api/v1/devices/:id/parameters.
func (h *Handler) GetDeviceParameters(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.service, h.permService, id) {
		return
	}

	params, err := h.service.GetDeviceParameters(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{"items": params, "total": len(params)})
}

// GetStats handles GET /api/v1/devices/stats.
func (h *Handler) GetStats(c *gin.Context) {
	var carrier *model.CarrierCode
	if cc := c.Query("carrier"); cc != "" {
		v := model.CarrierCode(cc)
		carrier = &v
	}

	counts, err := h.service.CountByStatus(c.Request.Context(), carrier)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{"counts": counts})
}

// RebootDevice handles POST /api/v1/devices/:id/reboot.
func (h *Handler) RebootDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	rebootErr := h.service.RebootDevice(c.Request.Context(), id)

	// Cross-module audit: ActionReboot / category 4 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionReboot
	entry.ResourceType = audit.ResourceDevice
	entry.ResourceID = id.String()
	entry.Success = rebootErr == nil
	if rebootErr != nil {
		entry.ErrorMessage = rebootErr.Error()
	}
	audit.LogAsync(entry)

	if rebootErr != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(rebootErr), rebootErr)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{"message": "reboot command queued"})
}

// ListGeo handles GET /api/v1/devices/geo.
// Returns devices with geographic coordinates for map display.
func (h *Handler) ListGeo(c *gin.Context) {
	var filter GeoDeviceFilter

	// Parse group_ids (comma-separated)
	if groupIDs := c.Query("group_ids"); groupIDs != "" {
		filter.GroupIDs = strings.Split(groupIDs, ",")
	}

	// Parse status (comma-separated)
	// Frontend sends: onlineActive, onlineInactive, offline
	// Backend expects: active, registered, provisioning, offline, maintenance, discovered, decommissioned
	if statusStr := c.Query("status"); statusStr != "" {
		statusList := strings.Split(statusStr, ",")
		filter.Status = make([]model.DeviceStatus, 0)
		for _, s := range statusList {
			// Map frontend status values to backend database status values
			switch s {
			case "onlineActive":
				filter.Status = append(filter.Status, model.DeviceActive)
			case "onlineInactive":
				filter.Status = append(filter.Status, model.DeviceRegistered, model.DeviceProvisioning)
			case "offline":
				filter.Status = append(filter.Status, model.DeviceOffline, model.DeviceMaintenance, model.DeviceDiscovered, model.DeviceDecommissioned)
			default:
				// Pass through any other status values for backward compatibility
				filter.Status = append(filter.Status, model.DeviceStatus(s))
			}
		}
	}

	// Parse keyword
	filter.Keyword = c.Query("keyword")

	// Parse bounds (format: minLng,maxLng,minLat,maxLat)
	if boundsStr := c.Query("bounds"); boundsStr != "" {
		parts := strings.Split(boundsStr, ",")
		if len(parts) == 4 {
			minLng, _ := strconv.ParseFloat(parts[0], 64)
			maxLng, _ := strconv.ParseFloat(parts[1], 64)
			minLat, _ := strconv.ParseFloat(parts[2], 64)
			maxLat, _ := strconv.ParseFloat(parts[3], 64)
			filter.Bounds = &GeoBounds{
				MinLng: minLng,
				MaxLng: maxLng,
				MinLat: minLat,
				MaxLat: maxLat,
			}
		}
	}

	// Parse pagination
	filter.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	filter.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "1000"))
	if filter.PageSize <= 0 {
		filter.PageSize = 1000
	}
	if filter.PageSize > 10000 {
		filter.PageSize = 10000
	}

	devices, total, err := h.service.ListGeo(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{
		"items": devices,
		"total": total,
	})
}

// GetGeoStats handles GET /api/v1/devices/geo/stats.
// Returns device statistics for map display.
func (h *Handler) GetGeoStats(c *gin.Context) {
	var groupIDs []string

	// Parse group_ids (comma-separated)
	if groupIDsStr := c.Query("group_ids"); groupIDsStr != "" {
		groupIDs = strings.Split(groupIDsStr, ",")
	}

	stats, err := h.service.GetGeoStats(c.Request.Context(), groupIDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// Convert status count to frontend expected format (onlineActive/onlineInactive/offline)
	// onlineActive: active - 在线激活
	// onlineInactive: registered + provisioning - 在线未激活
	// offline: offline + maintenance + discovered + decommissioned - 离线
	onlineInactive := stats.StatusCount[model.DeviceRegistered] + stats.StatusCount[model.DeviceProvisioning]
	offline := stats.StatusCount[model.DeviceOffline] +
		stats.StatusCount[model.DeviceMaintenance] +
		stats.StatusCount[model.DeviceDiscovered] +
		stats.StatusCount[model.DeviceDecommissioned]

	result := gin.H{
		"total": stats.Total,
		"status_count": gin.H{
			"onlineActive":   stats.StatusCount[model.DeviceActive],
			"onlineInactive": onlineInactive,
			"offline":        offline,
		},
	}

	// Add center point if available
	if stats.Center != nil {
		result["center"] = gin.H{
			"lat": stats.Center.Latitude,
			"lng": stats.Center.Longitude,
		}
	}

	response.OK(c, result)
}

// SearchDevices handles GET /api/v1/devices/search.
// Searches devices by keyword for map display.
func (h *Handler) SearchDevices(c *gin.Context) {
	keyword := c.Query("keyword")
	if len(keyword) < 2 {
		response.OK(c, gin.H{"items": []GeoDevice{}})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	devices, err := h.service.SearchDevices(c.Request.Context(), keyword, limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{"items": devices})
}

// SyncDeviceParams T-0126: 手动触发设备参数 Path B 全量同步。
//
// POST /api/v1/devices/:id/sync-params
// Body (可选)：{"force": true}（预留供未来节流绕过；当前 manual 端点天然不走 Redis 节流）
// 响应 202：{"status": "queued", "source_id": "manual:UUID", "device_id": "UUID"}
// 响应 404：device 不存在
// 响应 503：Path B 不可用（设备 productClass 未路由 / MappingSet 缺失）
//
// 替代旧 /param-sync 端点（Path A 已下线），完整接入触发链：
// reason="manual" → 差异日志 (T-0127) + last_param_sync_at 回写 (T-0124) + Translator (T-0098)。
func (h *Handler) SyncDeviceParams(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// force 字段可选，当前 no-op 但 log 记录供未来扩展
	var req struct {
		Force bool `json:"force"`
	}
	_ = c.ShouldBindJSON(&req) // 容错：body 为空仍 OK

	// SourceID 写入 device_tasks.source_id（UUID 列），仅做溯源标识；reason="manual"
	// 通过 WithReason option 走 Redis 通道独立传递。响应里保留 manual:<uuid> display 形式
	// 给前端 toast 与 API 契约。
	sourceID := uuid.New().String()
	displaySourceID := fmt.Sprintf("manual:%s", sourceID)
	used, dev, err := h.service.SyncDeviceParamsManual(c.Request.Context(), id, sourceID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if !used {
		// Path B 不可用 — 设备 productClass 未路由到 product / MappingSet 为空
		response.OKWithStatus(c, http.StatusServiceUnavailable, gin.H{
			"status":    "unavailable",
			"message":   "path-b sync unavailable: device product not routed or mapping set missing",
			"device_id": id.String(),
		})
		return
	}

	deviceSN := ""
	if dev != nil {
		deviceSN = dev.SerialNumber
	}
	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"status":        "queued",
		"source_id":     displaySourceID,
		"device_id":     id.String(),
		"serial_number": deviceSN,
		"force":         req.Force,
	})
}

// SetRFSwitch handles PUT /api/v1/devices/:id/rf-switch.
func (h *Handler) SetRFSwitch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.SetRFSwitch(c.Request.Context(), id, req.Enabled); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{"message": "RF switch command queued"})
}

// BatchDeleteDevices handles DELETE /api/v1/devices/batch.
// Accepts a JSON body with a list of device IDs and soft-deletes all of them.
func (h *Handler) BatchDeleteDevices(c *gin.Context) {
	var req BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if len(req.IDs) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if len(req.IDs) > 100 {
		response.Fail(c, http.StatusBadRequest, "batch size must not exceed 100")
		return
	}

	// Get username from context for recycle bin tracking
	deletedBy := ""
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		deletedBy = v.(string)
	}

	result := h.service.BatchDeleteDevices(c.Request.Context(), req.IDs, deletedBy)

	// Cross-module audit: ActionDelete (batch) / category 5 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionDelete
	entry.ResourceType = audit.ResourceDevice
	entry.Details = map[string]interface{}{
		"batch":     true,
		"id_count":  len(req.IDs),
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
	}
	entry.Success = result.Failed == 0
	audit.LogAsync(entry)

	response.OK(c, result)
}

// BatchImportRequest 来自 POST /api/v1/devices/batch-import 的请求体。
// 前端解析 CSV 后以结构化 JSON 数组提交；服务侧逐条调 CreateDevice 落库，
// 单行失败不影响其他行（与"成功/失败按行单独回执"的产品语义一致）。
//
// GroupID（可选）：用户在 /device/group 页面选中的二级分组 ID；后端在所有
// CreateDevice 成功后一次性把这批新设备 BatchAddDevices 进该分组。未传时
// 设备保持"未分组"，由后续 inform 心跳触发的规则或手工编辑归组。
// BatchImportDeviceRow 是 SN 批量导入的单行。
//
// 产品语义：导入【只更新已注册设备】的 名称 / 备注，并把这批 SN 划入当前所选分组；
// 【不新建设备】。原因：设备由 TR-069 inform 自动注册，且 devices.carrier 是 NOT NULL
// 的不可变 LIST 分区键（由 OUI/inform 决定），无法凭 SN 新建一条合法设备。
// SN 不存在的行返回失败回执（设备不存在）。
type BatchImportDeviceRow struct {
	SerialNumber string  `json:"serial_number" binding:"required"`
	DeviceName   *string `json:"device_name"`
	Remark       *string `json:"remark"`
}

type BatchImportRequest struct {
	Devices []BatchImportDeviceRow `json:"devices" binding:"required,min=1,max=1000,dive"`
	GroupID *uuid.UUID             `json:"group_id,omitempty" binding:"omitempty,uuid"`
}

// BatchImportResponse 报告本次批次的成功/失败统计 + 失败明细。
type BatchImportResponse struct {
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Errors    []BatchImportRowError `json:"errors"`
}

// BatchImportRowError 描述单行导入失败的原因（携带行号方便用户定位）。
type BatchImportRowError struct {
	Row    int    `json:"row"`          // 1-based, 与 CSV 用户视角行号一致（不含 header）
	SN     string `json:"sn,omitempty"` // 失败行的设备 SN（即使插库失败，方便用户定位）
	Reason string `json:"reason"`       // 友好的中文/英文错误描述
}

// BatchImportDevices handles POST /api/v1/devices/batch-import.
//
// 接收一组结构化 device 数据，逐条调 service.CreateDevice 落库，跳过失败行继续；
// 返回每行成败明细。前端 CSV 解析失败的行不会到这里——前端做基础校验，
// 后端只在 DB 唯一性 / 模型校验 / license 等运行时层面给出回执。
func (h *Handler) BatchImportDevices(c *gin.Context) {
	var req BatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 操作人（用于 device_info 审计字段）。
	updater := ""
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		updater, _ = v.(string)
	}

	resp := BatchImportResponse{Total: len(req.Devices)}
	matchedIDs := make([]uuid.UUID, 0, len(req.Devices))
	for i, row := range req.Devices {
		// 按 SN 查已注册设备；不存在 → 该行失败（不新建）。
		dev, err := h.service.GetBySerialNumber(c.Request.Context(), row.SerialNumber)
		if err != nil || dev == nil {
			resp.Failed++
			reason := "设备不存在（导入仅更新已注册设备的名称/备注，并归入当前分组）"
			if err != nil && !errors.Is(err, commonerrors.ErrNotFound) {
				reason = err.Error()
			}
			resp.Errors = append(resp.Errors, BatchImportRowError{Row: i + 1, SN: row.SerialNumber, Reason: reason})
			continue
		}

		// 更新设备名称（devices.device_name —— 列表显示名）。
		if row.DeviceName != nil {
			if _, err := h.service.UpdateDevice(c.Request.Context(), dev.ID, UpdateDeviceRequest{DeviceName: row.DeviceName}); err != nil {
				resp.Failed++
				resp.Errors = append(resp.Errors, BatchImportRowError{Row: i + 1, SN: row.SerialNumber, Reason: "更新设备名称失败：" + err.Error()})
				continue
			}
		}
		// 更新备注（device_info.remark）。
		if row.Remark != nil {
			if err := h.service.UpdateDeviceInfo(c.Request.Context(), dev.ID, UpdateDeviceInfoRequest{Remark: row.Remark}, updater); err != nil {
				resp.Failed++
				resp.Errors = append(resp.Errors, BatchImportRowError{Row: i + 1, SN: row.SerialNumber, Reason: "更新备注失败：" + err.Error()})
				continue
			}
		}
		resp.Succeeded++
		matchedIDs = append(matchedIDs, dev.ID)
	}

	// 产品要求：把本次导入命中的所有 SN 一次性划入当前所选分组（device_group_members）。
	// 归组失败不回滚名称/备注更新，给一条 warning row 让用户感知（不默默吞掉）。
	if req.GroupID != nil && len(matchedIDs) > 0 {
		if err := h.service.BatchAssignToGroup(c.Request.Context(), *req.GroupID, matchedIDs); err != nil {
			resp.Errors = append(resp.Errors, BatchImportRowError{
				Row:    0,
				Reason: "名称/备注已更新，但归入指定分组失败：" + err.Error(),
			})
		}
	}

	response.OK(c, resp)
}

// BatchRebootDevices handles POST /api/v1/devices/batch-reboot.
// Accepts a JSON body with a list of device IDs and queues a Reboot command for each.
func (h *Handler) BatchRebootDevices(c *gin.Context) {
	var req BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if len(req.IDs) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if len(req.IDs) > 50 {
		response.Fail(c, http.StatusBadRequest, "batch size must not exceed 50")
		return
	}

	result := h.service.BatchRebootDevices(c.Request.Context(), req.IDs)

	// Cross-module audit: ActionReboot (batch) / category 4 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionReboot
	entry.ResourceType = audit.ResourceDevice
	entry.Details = map[string]interface{}{
		"batch":     true,
		"id_count":  len(req.IDs),
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
	}
	entry.Success = result.Failed == 0
	audit.LogAsync(entry)

	response.OKWithStatus(c, http.StatusAccepted, result)
}

// ===== Recycle Bin Handlers =====

// RecycleBinFilterQuery binds query parameters for recycle bin list.
type RecycleBinFilterQuery struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy    string `form:"sort_by" binding:"omitempty"`
	SortDir   string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
	Search    string `form:"search" binding:"omitempty"`
	Carrier   string `form:"carrier" binding:"omitempty"`
	GroupID   string `form:"group_id" binding:"omitempty,uuid"`
	DeletedBy string `form:"deleted_by" binding:"omitempty"`
}

// ListRecycleBin handles GET /api/v1/devices/recycle.
// Returns a paginated list of soft-deleted devices.
func (h *Handler) ListRecycleBin(c *gin.Context) {
	var query RecycleBinFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := RecycleBinFilter{
		ListRequest: model.ListRequest{
			Page:     query.Page,
			PageSize: query.PageSize,
			SortBy:   query.SortBy,
			SortDir:  query.SortDir,
		},
	}

	if query.Search != "" {
		filter.Search = &query.Search
	}
	if query.Carrier != "" {
		carrier := model.CarrierCode(query.Carrier)
		filter.Carrier = &carrier
	}
	if query.GroupID != "" {
		groupID, err := uuid.Parse(query.GroupID)
		if err == nil {
			filter.GroupID = &groupID
		}
	}
	if query.DeletedBy != "" {
		filter.DeletedBy = &query.DeletedBy
	}

	result, err := h.service.ListRecycleBin(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, result)
}

// RestoreDevicesRequest binds the request body for restore operation.
type RestoreDevicesRequest struct {
	IDs []uuid.UUID `json:"ids" binding:"required"`
}

// RestoreDevices handles PATCH /api/v1/devices/recycle/restore.
// Restores soft-deleted devices.
func (h *Handler) RestoreDevices(c *gin.Context) {
	var req RestoreDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if len(req.IDs) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if len(req.IDs) > 100 {
		response.Fail(c, http.StatusBadRequest, "batch size must not exceed 100")
		return
	}

	restored, err := h.service.RestoreDevices(c.Request.Context(), req.IDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, gin.H{"restored": restored}, "Devices restored successfully")
}

// PermanentDeleteDevices handles DELETE /api/v1/devices/recycle/permanent.
// Permanently removes devices from the database.
func (h *Handler) PermanentDeleteDevices(c *gin.Context) {
	var req RestoreDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if len(req.IDs) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if len(req.IDs) > 100 {
		response.Fail(c, http.StatusBadRequest, "batch size must not exceed 100")
		return
	}

	deleted, err := h.service.PermanentDeleteDevices(c.Request.Context(), req.IDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithMsg(c, gin.H{"deleted": deleted}, "Devices permanently deleted")
}
