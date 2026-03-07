package dashboard

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
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
