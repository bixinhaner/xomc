package device

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ExportHandler provides HTTP endpoints for device data export.
type ExportHandler struct {
	exportService *ExportService
	permService   VisibleGroupsResolver
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exportService *ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// SetPermissionService sets the data permission service for export filtering.
func (h *ExportHandler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permService = ps
}

// RegisterRoutes registers export routes.
func (h *ExportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/devices/export", h.ExportDevices)
}

// ExportDevices handles GET /devices/export.
func (h *ExportHandler) ExportDevices(c *gin.Context) {
	filter := DeviceFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if carrier := c.Query("carrier"); carrier != "" {
		cc := model.CarrierCode(carrier)
		filter.Carrier = &cc
	}
	if status := c.Query("status"); status != "" {
		s := model.DeviceStatus(status)
		filter.Status = &s
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

	filename := fmt.Sprintf("devices_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	// UTF-8 BOM for Excel compatibility.
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	if err := h.exportService.ExportCSV(c.Request.Context(), filter, c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "export failed: %v", err)
	}
}
