package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/admin"
)

// Handler provides REST API endpoints for dashboard.
type Handler struct {
	service *Service
}

// NewHandler creates a new dashboard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers dashboard routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dashboard := rg.Group("/dashboard")
	{
		dashboard.GET("/summary", h.GetSummary)
		dashboard.GET("/alarm-trend", h.GetAlarmTrend)
		dashboard.GET("/device-status", h.GetDeviceStatus)
		dashboard.GET("/kpi-trend", h.GetKPITrend)
		dashboard.GET("/region-stats", h.GetRegionStats)
		dashboard.GET("/widgets", h.GetWidgets)
		dashboard.PUT("/widgets", h.SaveWidgets)
		dashboard.GET("/alarm-type-pie", h.GetAlarmTypePie)
		dashboard.GET("/kpi-time-series", h.GetKPITimeSeries)
	}
}

// GetSummary handles GET /api/v1/dashboard/summary.
func (h *Handler) GetSummary(c *gin.Context) {
	summary, err := h.service.GetSummary(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetAlarmTrend handles GET /api/v1/dashboard/alarm-trend?days=7.
func (h *Handler) GetAlarmTrend(c *gin.Context) {
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: %s", daysStr))
			return
		}
		days = parsed
	}

	entries, err := h.service.GetAlarmTrend(c.Request.Context(), days)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

// GetDeviceStatus handles GET /api/v1/dashboard/device-status.
func (h *Handler) GetDeviceStatus(c *gin.Context) {
	counts, err := h.service.GetDeviceStatus(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

// GetKPITrend handles GET /api/v1/dashboard/kpi-trend?kpi_name=...&days=7.
func (h *Handler) GetKPITrend(c *gin.Context) {
	kpiName := c.Query("kpi_name")
	if kpiName == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("kpi_name is required"))
		return
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || parsed < 1 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid days parameter: %s", daysStr))
			return
		}
		days = parsed
	}

	entries, err := h.service.GetKPITrend(c.Request.Context(), kpiName, days)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

// GetRegionStats handles GET /api/v1/dashboard/region-stats.
func (h *Handler) GetRegionStats(c *gin.Context) {
	entries, err := h.service.GetRegionStats(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

// GetWidgets handles GET /api/v1/dashboard/widgets.
func (h *Handler) GetWidgets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, err)
		return
	}

	layout, err := h.service.GetWidgetLayout(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, layout)
}

// SaveWidgets handles PUT /api/v1/dashboard/widgets.
func (h *Handler) SaveWidgets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, err)
		return
	}

	var body struct {
		Layout json.RawMessage `json:"layout" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid request body: %w", err))
		return
	}

	layout, err := h.service.SaveWidgetLayout(c.Request.Context(), userID, body.Layout)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, layout)
}

// GetAlarmTypePie handles GET /api/v1/dashboard/alarm-type-pie.
func (h *Handler) GetAlarmTypePie(c *gin.Context) {
	entries, err := h.service.GetAlarmTypePie(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

// GetKPITimeSeries handles GET /api/v1/dashboard/kpi-time-series?kpi_names=...&start_time=...&end_time=...
func (h *Handler) GetKPITimeSeries(c *gin.Context) {
	kpiNamesRaw := c.Query("kpi_names")
	kpiNames := parseKPINames(kpiNamesRaw)
	if len(kpiNames) == 0 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("kpi_names is required (comma-separated)"))
		return
	}

	// Default to last 7 days if not specified
	now := time.Now()
	startTime := now.AddDate(0, 0, -7)
	endTime := now

	if st := c.Query("start_time"); st != "" {
		parsed, err := time.Parse(time.RFC3339, st)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid start_time (use RFC3339 format): %w", err))
			return
		}
		startTime = parsed
	}

	if et := c.Query("end_time"); et != "" {
		parsed, err := time.Parse(time.RFC3339, et)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("invalid end_time (use RFC3339 format): %w", err))
			return
		}
		endTime = parsed
	}

	result, err := h.service.GetKPITimeSeries(c.Request.Context(), kpiNames, startTime, endTime)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// getUserID extracts the authenticated user's UUID from the Gin context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get(admin.CtxKeyUserID)
	if !exists {
		return uuid.UUID{}, fmt.Errorf("user not authenticated")
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("invalid user ID in context")
	}
	return userID, nil
}
