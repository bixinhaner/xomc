package northbound

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// AlarmHandler handles northbound alarm export and sync endpoints.
type AlarmHandler struct {
	svc    *NorthboundService
	logger *zap.Logger
}

// NewAlarmHandler creates a new AlarmHandler.
func NewAlarmHandler(svc *NorthboundService, logger *zap.Logger) *AlarmHandler {
	return &AlarmHandler{
		svc:    svc,
		logger: logger,
	}
}

// ExportAlarms handles alarm data export for northbound consumers.
func (h *AlarmHandler) ExportAlarms(c *gin.Context) {
	var req ExportAlarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := alarm.AlarmFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}
	if req.DeviceSN != "" {
		filter.DeviceSN = &req.DeviceSN
	}
	if req.Severity != "" {
		sev := parseSeverity(req.Severity)
		if sev != 0 {
			filter.Severity = &sev
		}
	}
	if req.Status != "" {
		st := model.AlarmStatus(req.Status)
		filter.Status = &st
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.svc.ExportAlarms(c.Request.Context(), filter)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound alarm export failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "alarm export failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListActiveAlarms returns all active alarms for northbound sync.
func (h *AlarmHandler) ListActiveAlarms(c *gin.Context) {
	listReq := model.DefaultListRequest()
	if err := c.ShouldBindQuery(&listReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.ListActiveAlarms(c.Request.Context(), listReq)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound alarm list failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "alarm list failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func parseSeverity(s string) model.AlarmSeverity {
	switch s {
	case "1":
		return model.AlarmCritical
	case "2":
		return model.AlarmMajor
	case "3":
		return model.AlarmMinor
	case "4":
		return model.AlarmWarning
	default:
		return 0
	}
}
