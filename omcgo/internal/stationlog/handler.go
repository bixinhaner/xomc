package stationlog

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 日志采集文件的 HTTP 处理器
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("stationlog-handler"),
	}
}

// RegisterRoutes 注册路由到指定 RouterGroup。
//
// 挂载于 /api/v1 下：
//
//	GET    /station-logs                              （泛日志：运行日志/故障日志）
//	GET    /station-logs/:id/download
//	DELETE /station-logs/:id
//
//	GET    /device-abnormal-reboots                   （T-0158：异常重启记录列表）
//	GET    /device-abnormal-reboots/:id
//	DELETE /device-abnormal-reboots/:id
//	GET    /device-abnormal-reboots/:id/download
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/station-logs")
	logs.GET("", h.List)
	logs.GET("/:id/download", h.Download)
	logs.DELETE("/:id", h.Delete)

	// T-0158: 异常重启记录专用前缀。语义上属于设备管理域，参数维度与 station-logs
	// 不完全相同（多 record_status / device_type / 时间区间过滤），独立挂载更清晰。
	reb := rg.Group("/device-abnormal-reboots")
	reb.GET("", h.ListAbnormalReboots)
	reb.GET("/:id", h.GetAbnormalReboot)
	reb.DELETE("/:id", h.DeleteAbnormalReboot)
	reb.GET("/:id/download", h.DownloadAbnormalReboot)
}

// List godoc
// @Summary  查询日志文件列表
// @Tags     station-logs
// @Produce  json
// @Param    device_id  query  string  false  "设备 ID（UUID）"
// @Param    log_type   query  string  false  "日志类型: running | fault"
// @Param    page       query  int     false  "页码，默认 1"
// @Param    page_size  query  int     false  "每页数量，默认 20"
// @Success  200 {object} map[string]interface{}
// @Router   /station-logs [get]
func (h *Handler) List(c *gin.Context) {
	filter := LogFileFilter{
		Page:     1,
		PageSize: 20,
	}

	if deviceIDStr := c.Query("device_id"); deviceIDStr != "" {
		id, err := uuid.Parse(deviceIDStr)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DeviceID = &id
	}
	if logTypeStr := c.Query("log_type"); logTypeStr != "" {
		filter.LogType = LogType(logTypeStr)
	}
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			filter.PageSize = s
		}
	}

	items, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list station log files", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"size":  filter.PageSize,
	})
}

// Download godoc
// @Summary  获取日志文件预签名下载 URL（有效期 1 小时）
// @Tags     station-logs
// @Param    id        path   string  true   "日志文件 ID（UUID）"
// @Param    log_type  query  string  false  "日志类型: running（默认）| fault"
// @Param    redirect  query  string  false  "传 true 时 302 跳转到下载地址"
// @Success  200 {object} map[string]interface{}
// @Router   /station-logs/{id}/download [get]
func (h *Handler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	logType := logTypeFromQuery(c, LogTypeRunning)

	url, err := h.svc.DownloadURL(c.Request.Context(), id, logType)
	if err != nil {
		h.logger.Error("generate download url", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if c.Query("redirect") == "true" {
		c.Redirect(http.StatusFound, url)
		return
	}
	response.OK(c, gin.H{"url": url})
}

// Delete godoc
// @Summary  删除日志文件（MinIO 文件 + 标记记录已删除）
// @Tags     station-logs
// @Param    id        path   string  true   "日志文件 ID（UUID）"
// @Param    log_type  query  string  false  "日志类型: running（默认）| fault"
// @Success  200 {object} map[string]interface{}
// @Router   /station-logs/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	logType := logTypeFromQuery(c, LogTypeRunning)

	if err := h.svc.Delete(c.Request.Context(), id, logType); err != nil {
		h.logger.Error("delete station log file", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"status": "deleted"})
}

// logTypeFromQuery 解析 log_type query 参数，不合法时使用 defaultType。
func logTypeFromQuery(c *gin.Context, defaultType LogType) LogType {
	v := c.Query("log_type")
	if v == string(LogTypeRunning) || v == string(LogTypeFault) {
		return LogType(v)
	}
	return defaultType
}

// -----------------------------------------------------------------------------
// T-0158: 异常重启记录专用端点
// -----------------------------------------------------------------------------

// ListAbnormalReboots godoc
// @Summary  查询异常重启记录列表
// @Tags     device-abnormal-reboots
// @Produce  json
// @Param    device_id      query  string  false  "设备 ID（UUID）"
// @Param    device_sn      query  string  false  "设备 SN"
// @Param    record_status  query  string  false  "记录状态: detected | file_received | collection_failed"
// @Param    device_type    query  string  false  "设备类型: eNB | gNB"
// @Param    start_time     query  string  false  "起始时间（RFC3339）"
// @Param    end_time       query  string  false  "结束时间（RFC3339）"
// @Param    page           query  int     false  "页码，默认 1"
// @Param    page_size      query  int     false  "每页数量，默认 20"
// @Success  200 {object} map[string]interface{}
// @Router   /device-abnormal-reboots [get]
func (h *Handler) ListAbnormalReboots(c *gin.Context) {
	filter := LogFileFilter{
		LogType:  LogTypeFault,
		Page:     1,
		PageSize: 20,
	}

	if v := c.Query("device_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DeviceID = &id
	}
	if v := strings.TrimSpace(c.Query("device_sn")); v != "" {
		filter.DeviceSN = v
	}
	if v := strings.TrimSpace(c.Query("record_status")); v != "" {
		filter.RecordStatus = v
	}
	if v := strings.TrimSpace(c.Query("device_type")); v != "" {
		filter.DeviceType = v
	}
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartTime = &t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndTime = &t
		}
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if v := c.Query("page_size"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			filter.PageSize = s
		}
	}

	items, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list abnormal reboot records", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"size":  filter.PageSize,
	})
}

// GetAbnormalReboot godoc
// @Summary  获取异常重启记录详情
// @Tags     device-abnormal-reboots
// @Param    id  path  string  true  "记录 ID（UUID）"
// @Success  200 {object} map[string]interface{}
// @Failure  404 {object} map[string]interface{}
// @Router   /device-abnormal-reboots/{id} [get]
func (h *Handler) GetAbnormalReboot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	item, err := h.svc.GetByID(c.Request.Context(), id, LogTypeFault)
	if err != nil {
		h.logger.Error("get abnormal reboot record", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, item)
}

// DeleteAbnormalReboot godoc
// @Summary  删除异常重启记录（软删 + 清理 MinIO 文件）
// @Tags     device-abnormal-reboots
// @Param    id  path  string  true  "记录 ID（UUID）"
// @Success  200 {object} map[string]interface{}
// @Router   /device-abnormal-reboots/{id} [delete]
func (h *Handler) DeleteAbnormalReboot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// 不论 detected 占位还是 file_received，都允许删除：detected 只做软删 + 标记，
	// file_received 走完整的 MinIO 清理 + 标记。
	if err := h.svc.Delete(c.Request.Context(), id, LogTypeFault); err != nil {
		h.logger.Error("delete abnormal reboot record", zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"status": "deleted"})
}

// DownloadAbnormalReboot godoc
// @Summary  下载异常重启日志文件（仅 file_received 状态有文件）
// @Tags     device-abnormal-reboots
// @Param    id        path   string  true   "记录 ID（UUID）"
// @Param    redirect  query  string  false  "传 true 时 302 跳转到下载地址"
// @Success  200 {object} map[string]interface{}
// @Failure  409 {object} map[string]interface{} "记录处于 detected 状态，尚无文件"
// @Router   /device-abnormal-reboots/{id}/download [get]
func (h *Handler) DownloadAbnormalReboot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// 先取一遍记录，提前拦截 detected 状态（避免落到 minio presign 才报错）
	item, err := h.svc.GetByID(c.Request.Context(), id, LogTypeFault)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if item.RecordStatus != FaultRecordStatusFileReceived || item.FileName == "" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"code":    http.StatusConflict,
			"message": "abnormal reboot record has no file yet",
			"status":  item.RecordStatus,
		})
		return
	}

	url, err := h.svc.DownloadURL(c.Request.Context(), id, LogTypeFault)
	if err != nil {
		h.logger.Error("generate abnormal reboot download url",
			zap.String("id", id.String()), zap.Error(err))
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if c.Query("redirect") == "true" {
		c.Redirect(http.StatusFound, url)
		return
	}
	response.OK(c, gin.H{"url": url})
}
