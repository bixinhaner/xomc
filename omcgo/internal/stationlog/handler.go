package stationlog

import (
	"net/http"
	"strconv"

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
// 挂载于 /api/v1 下，路径为：
//
//	GET    /station-logs
//	GET    /station-logs/:id/download
//	DELETE /station-logs/:id
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/station-logs")
	logs.GET("", h.List)
	logs.GET("/:id/download", h.Download)
	logs.DELETE("/:id", h.Delete)
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
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
