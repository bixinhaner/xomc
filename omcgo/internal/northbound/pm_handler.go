package northbound

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
)

// PMHandler handles northbound PM and KPI export endpoints.
type PMHandler struct {
	svc    *NorthboundService
	logger *zap.Logger
}

// NewPMHandler creates a new PMHandler.
func NewPMHandler(svc *NorthboundService, logger *zap.Logger) *PMHandler {
	return &PMHandler{
		svc:    svc,
		logger: logger,
	}
}

// ExportPM handles PM data export for northbound consumers.
func (h *PMHandler) ExportPM(c *gin.Context) {
	var req ExportPMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid start_time format, use RFC3339")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid end_time format, use RFC3339")
		return
	}

	filter := counter.CounterFilter{
		StartTime:   startTime,
		EndTime:     endTime,
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}
	if req.DeviceID != "" {
		id, err := uuid.Parse(req.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if req.CellID != "" {
		filter.CellID = &req.CellID
	}
	if req.CounterGroup != "" {
		filter.CounterGroup = &req.CounterGroup
	}

	result, err := h.svc.ExportPM(c.Request.Context(), filter)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound PM export failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "pm export failed")
		return
	}

	response.OK(c, result)
}

// ExportKPI handles KPI data export for northbound consumers.
func (h *PMHandler) ExportKPI(c *gin.Context) {
	var q struct {
		DeviceID   string `form:"device_id"`
		KPIName    string `form:"kpi_name"`
		Carrier    string `form:"carrier"`
		Technology string `form:"technology"`
		StartTime  string `form:"start_time"`
		EndTime    string `form:"end_time"`
		model.ListRequest
	}
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	filter := kpi.KPIFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if q.KPIName != "" {
		filter.KPIName = &q.KPIName
	}
	if q.Carrier != "" {
		cc := model.CarrierCode(q.Carrier)
		filter.Carrier = &cc
	}
	if q.Technology != "" {
		t := model.Technology(q.Technology)
		filter.Technology = &t
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = t
		}
	}

	result, err := h.svc.ExportKPI(c.Request.Context(), filter)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound KPI export failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "kpi export failed")
		return
	}

	response.OK(c, result)
}
