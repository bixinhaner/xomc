package device

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// VisibleGroupsResolver resolves which device groups a user can see.
type VisibleGroupsResolver interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, carrier *model.CarrierCode) ([]uuid.UUID, error)
}

// Handler provides HTTP handlers for device management REST API.
type Handler struct {
	service     *DeviceService
	permService VisibleGroupsResolver
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
		devices.GET("/stats", h.GetStats)
		devices.GET("/geo", h.ListGeo)           // Map device geo data
		devices.GET("/geo/stats", h.GetGeoStats) // Map device statistics
		devices.GET("/search", h.SearchDevices)  // Search devices for map
		// Batch routes must be registered before /:id to avoid path conflicts
		devices.DELETE("/batch", h.BatchDeleteDevices)
		devices.POST("/batch-reboot", h.BatchRebootDevices)
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
		devices.POST("/:id/param-sync", h.TriggerParamSync)
		devices.PUT("/:id/rf-switch", h.SetRFSwitch)
	}
}

// CreateDeviceRequest defines the request body for creating a device.
type CreateDeviceRequest struct {
	SerialNumber string            `json:"serial_number" binding:"required"`
	OUI          string            `json:"oui" binding:"required"`
	ProductClass string            `json:"product_class"`
	Manufacturer string            `json:"manufacturer"`
	ModelName    string            `json:"model_name"`
	Carrier      model.CarrierCode `json:"carrier" binding:"required"`
	Technology   model.Technology  `json:"technology" binding:"required"`
	SiteName     string            `json:"site_name"`
	SiteID       string            `json:"site_id"`
	Latitude     float64           `json:"latitude"`
	Longitude    float64           `json:"longitude"`
}

// UpdateDeviceRequest defines the request body for updating a device.
type UpdateDeviceRequest struct {
	SiteName  *string             `json:"site_name"`
	SiteID    *string             `json:"site_id"`
	ModelName *string             `json:"model_name"`
	Latitude  *float64            `json:"latitude"`
	Longitude *float64            `json:"longitude"`
	Status    *model.DeviceStatus `json:"status"`
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

	c.JSON(http.StatusCreated, device)
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

	c.JSON(http.StatusOK, device)
}

// DeleteDevice handles DELETE /api/v1/devices/:id.
func (h *Handler) DeleteDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteDevice(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
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
	if status := c.Query("status"); status != "" {
		s := model.DeviceStatus(status)
		filter.Status = &s
	}
	if oui := c.Query("oui"); oui != "" {
		filter.OUI = &oui
	}
	if sn := c.Query("sn"); sn != "" {
		filter.SN = &sn
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}
	if manufacturer := c.Query("manufacturer"); manufacturer != "" {
		filter.Manufacturer = &manufacturer
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
	if h.permService != nil {
		userID, _ := c.Get(admin.CtxKeyUserID)
		carrierVal, _ := c.Get(admin.CtxKeyCarrier)
		if uid, ok := userID.(uuid.UUID); ok {
			var carrier *model.CarrierCode
			if cv, ok := carrierVal.(*model.CarrierCode); ok {
				carrier = cv
			}
			visibleGroups, err := h.permService.GetUserVisibleGroupIDs(c.Request.Context(), uid, carrier)
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

	c.JSON(http.StatusOK, result)
}

// GetDevice handles GET /api/v1/devices/:id.
func (h *Handler) GetDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	device, err := h.service.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if device == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, device)
}

// GetDeviceParameters handles GET /api/v1/devices/:id/parameters.
func (h *Handler) GetDeviceParameters(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	params, err := h.service.GetDeviceParameters(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": params, "total": len(params)})
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

	c.JSON(http.StatusOK, gin.H{"counts": counts})
}

// RebootDevice handles POST /api/v1/devices/:id/reboot.
func (h *Handler) RebootDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.RebootDevice(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "reboot command queued"})
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

	c.JSON(http.StatusOK, gin.H{
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

	c.JSON(http.StatusOK, result)
}

// SearchDevices handles GET /api/v1/devices/search.
// Searches devices by keyword for map display.
func (h *Handler) SearchDevices(c *gin.Context) {
	keyword := c.Query("keyword")
	if len(keyword) < 2 {
		c.JSON(http.StatusOK, gin.H{"items": []GeoDevice{}})
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

	c.JSON(http.StatusOK, gin.H{"items": devices})
}

// TriggerParamSync handles POST /api/v1/devices/:id/param-sync.
func (h *Handler) TriggerParamSync(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.TriggerParamSync(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "parameter sync command queued"})
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

	c.JSON(http.StatusAccepted, gin.H{"message": "RF switch command queued"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size must not exceed 100"})
		return
	}

	// Get username from context for recycle bin tracking
	deletedBy := ""
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		deletedBy = v.(string)
	}

	result := h.service.BatchDeleteDevices(c.Request.Context(), req.IDs, deletedBy)
	c.JSON(http.StatusOK, result)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size must not exceed 50"})
		return
	}

	result := h.service.BatchRebootDevices(c.Request.Context(), req.IDs)
	c.JSON(http.StatusAccepted, result)
}

// ===== Recycle Bin Handlers =====

// RecycleBinFilterQuery binds query parameters for recycle bin list.
type RecycleBinFilterQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by" binding:"omitempty"`
	SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
	Search   string `form:"search" binding:"omitempty"`
	Carrier  string `form:"carrier" binding:"omitempty"`
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
	if query.DeletedBy != "" {
		filter.DeletedBy = &query.DeletedBy
	}

	result, err := h.service.ListRecycleBin(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size must not exceed 100"})
		return
	}

	restored, err := h.service.RestoreDevices(c.Request.Context(), req.IDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"restored": restored,
		"message": "Devices restored successfully",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size must not exceed 100"})
		return
	}

	deleted, err := h.service.PermanentDeleteDevices(c.Request.Context(), req.IDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": deleted,
		"message": "Devices permanently deleted",
	})
}
