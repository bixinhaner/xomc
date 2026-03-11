package northbound

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/model"
	"go.uber.org/zap"
)

// AlarmHandler wraps AlarmStore for northbound alarm sync and export.
type AlarmHandler struct {
	store  alarm.AlarmStore
	logger *zap.Logger
}

// NewAlarmHandler creates a new AlarmHandler.
func NewAlarmHandler(store alarm.AlarmStore, logger *zap.Logger) *AlarmHandler {
	return &AlarmHandler{
		store:  store,
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

	result, err := h.store.ListActive(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("northbound alarm export failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "alarm export failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListActiveAlarms returns all active alarms for northbound sync.
func (h *AlarmHandler) ListActiveAlarms(c *gin.Context) {
	filter := alarm.AlarmFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.store.ListActive(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("northbound alarm list failed", zap.Error(err))
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
