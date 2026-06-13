package northbound

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// AlarmHandler handles northbound alarm export and sync endpoints.
type AlarmHandler struct {
	svc    *NorthboundService
	logger *zap.Logger
	scoper *Scoper // 多租户隔离；nil 时退化为不隔离（由 Router.SetScoper 注入）
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
	// 过滤参数白名单：先窥探原始 JSON，拒绝未知字段（fail-closed）。
	raw, ok := peekJSONBody(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if bad := unknownJSONFields(raw, allowedAlarmExportParams); len(bad) > 0 {
		response.Fail(c, http.StatusBadRequest, "unknown filter fields: "+joinFields(bad))
		return
	}

	var req ExportAlarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 多租户隔离：非超管必须把导出限定到一个具体设备（device_sn），并校验其归属。
	if h.scoper != nil {
		if !h.scoper.RequireDeviceScope(c, req.DeviceSN != "") {
			return
		}
		if req.DeviceSN != "" && !h.scoper.AuthorizeDeviceBySN(c, req.DeviceSN) {
			return
		}
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
		response.Fail(c, http.StatusInternalServerError, "alarm export failed")
		return
	}

	response.OK(c, result)
}

// ListActiveAlarms returns all active alarms for northbound sync.
func (h *AlarmHandler) ListActiveAlarms(c *gin.Context) {
	// 全量活跃告警同步无设备维度过滤：非超管不得跨租户拉取全量 → 403。
	if h.scoper != nil && !h.scoper.RequireDeviceScope(c, false) {
		return
	}

	listReq := model.DefaultListRequest()
	if err := c.ShouldBindQuery(&listReq); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.svc.ListActiveAlarms(c.Request.Context(), listReq)
	if err != nil {
		logger.L(c.Request.Context()).Error("northbound alarm list failed", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, "alarm list failed")
		return
	}

	response.OK(c, result)
}

// parseSeverity 把北向级别入参解析成 canonical 告警级别。
// 库内 severity 列存 5 位字典码（31001~31004），系统对外也暴露这套码，故入参必须接受
// 字典码；同时兼容历史 1~4 小编号。返回 canonical 1~4，仓储层 severityAliases 会把它
// 双向展开匹配库内码。只认 1~4 会让字典码入参落默认分支返回 0、级别过滤静默失效（issue
// #306，与 #219/#272 同根）。
func parseSeverity(s string) model.AlarmSeverity {
	switch s {
	case "1", "31001":
		return model.AlarmCritical
	case "2", "31002":
		return model.AlarmMajor
	case "3", "31003":
		return model.AlarmMinor
	case "4", "31004":
		return model.AlarmWarning
	default:
		return 0
	}
}
