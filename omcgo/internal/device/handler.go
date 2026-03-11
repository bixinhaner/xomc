package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// Handler provides HTTP handlers for device management REST API.
type Handler struct {
	service *DeviceService
}

// NewHandler creates a new device REST API handler.
func NewHandler(service *DeviceService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers device routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("", h.ListDevices)
		devices.GET("/stats", h.GetStats)
		devices.GET("/:id", h.GetDevice)
		devices.GET("/:id/parameters", h.GetDeviceParameters)
		devices.POST("", h.CreateDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
		devices.POST("/:id/reboot", h.RebootDevice)
	}
}

// CreateDeviceRequest defines the request body for creating a device.
type CreateDeviceRequest struct {
	SerialNumber string           `json:"serial_number" binding:"required"`
	OUI          string           `json:"oui" binding:"required"`
	ProductClass string           `json:"product_class"`
	Manufacturer string           `json:"manufacturer"`
	ModelName    string           `json:"model_name"`
	Carrier      model.CarrierCode `json:"carrier" binding:"required"`
	Technology   model.Technology  `json:"technology" binding:"required"`
	SiteName     string           `json:"site_name"`
	SiteID       string           `json:"site_id"`
	Latitude     float64          `json:"latitude"`
	Longitude    float64          `json:"longitude"`
}

// UpdateDeviceRequest defines the request body for updating a device.
type UpdateDeviceRequest struct {
	SiteName  *string            `json:"site_name"`
	SiteID    *string            `json:"site_id"`
	ModelName *string            `json:"model_name"`
	Latitude  *float64           `json:"latitude"`
	Longitude *float64           `json:"longitude"`
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

	result, err := h.service.ListDevices(c.Request.Context(), filter)
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
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// TODO: Queue reboot command via ACS command queue (requires gRPC call to ACS)
	c.JSON(http.StatusAccepted, gin.H{"message": "reboot command queued"})
}
