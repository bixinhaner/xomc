package license

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// Handler provides HTTP handlers for license management REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new license Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("license-handler"),
	}
}

// RegisterRoutes registers license routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	licenses := rg.Group("/licenses")

	// Register specific routes BEFORE the :id route to avoid conflicts.
	licenses.GET("/summary", h.GetSummary)
	licenses.POST("/activate", h.Activate)
	licenses.POST("/import", h.Import)

	licenses.GET("", h.List)
	licenses.GET("/:id", h.GetByID)
	licenses.POST("/:id/revoke", h.Revoke)
}

// ---- Request types ----

// ActivateRequest defines the request body for activating a license.
type ActivateRequest struct {
	LicenseCode string `json:"license_code" binding:"required"`
}

// ImportRequest defines the request body for importing a new license.
type ImportRequest struct {
	LicenseName string          `json:"license_name" binding:"required"`
	LicenseCode string          `json:"license_code" binding:"required"`
	ProductName string          `json:"product_name" binding:"required"`
	LicenseType LicenseType     `json:"license_type"`
	Status      LicenseStatus   `json:"status"`
	MaxDevices  int             `json:"max_devices"`
	UsedDevices int             `json:"used_devices"`
	Features    json.RawMessage `json:"features"`
	IssueDate   time.Time       `json:"issue_date" binding:"required"`
	ExpiryDate  *time.Time      `json:"expiry_date"`
	Licensor    *string         `json:"licensor"`
	DeviceType  *string         `json:"device_type"`
	Region      *string         `json:"region"`
	Notes       *string         `json:"notes"`
}

// ---- Handlers ----

// List handles GET /api/v1/licenses.
func (h *Handler) List(c *gin.Context) {
	filter := LicenseFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := LicenseStatus(status)
		filter.Status = &s
	}
	if licenseType := c.Query("license_type"); licenseType != "" {
		t := LicenseType(licenseType)
		filter.LicenseType = &t
	}
	if deviceType := c.Query("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByID handles GET /api/v1/licenses/:id.
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	lic, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, lic)
}

// GetSummary handles GET /api/v1/licenses/summary.
func (h *Handler) GetSummary(c *gin.Context) {
	summary, err := h.service.Summary(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// Activate handles POST /api/v1/licenses/activate.
func (h *Handler) Activate(c *gin.Context) {
	var req ActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	lic, err := h.service.Activate(c.Request.Context(), req.LicenseCode)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, lic)
}

// Revoke handles POST /api/v1/licenses/:id/revoke.
func (h *Handler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.Revoke(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

// Import handles POST /api/v1/licenses/import.
func (h *Handler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	lic := &License{
		LicenseName: req.LicenseName,
		LicenseCode: req.LicenseCode,
		ProductName: req.ProductName,
		LicenseType: req.LicenseType,
		Status:      req.Status,
		MaxDevices:  req.MaxDevices,
		UsedDevices: req.UsedDevices,
		Features:    req.Features,
		IssueDate:   req.IssueDate,
		ExpiryDate:  req.ExpiryDate,
		Licensor:    req.Licensor,
		DeviceType:  req.DeviceType,
		Region:      req.Region,
		Notes:       req.Notes,
	}

	if lic.LicenseType == "" {
		lic.LicenseType = TypeSubscription
	}

	created, err := h.service.Import(c.Request.Context(), lic)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}
