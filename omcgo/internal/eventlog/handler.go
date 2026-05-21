package eventlog

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

type Handler struct {
	svc    *Service
	logger *zap.Logger
}

func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger.Named("eventlog-handler")}
}

// RegisterRoutes 挂载于 /api/v1 下：
//
//	GET /event-logs                 列表 + 过滤
//	GET /event-logs/:id             详情
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/event-logs")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}

// List godoc
// @Summary 查询事件日志列表
// @Param device_sn   query string false "设备 SN（模糊匹配 ILIKE）"
// @Param event_type  query string false "事件类型：boot / ..."
// @Param start_time  query string false "起始时间 RFC3339"
// @Param end_time    query string false "结束时间 RFC3339"
// @Param page        query int    false "页码 默认 1"
// @Param page_size   query int    false "每页 默认 20"
func (h *Handler) List(c *gin.Context) {
	filter := Filter{Page: 1, PageSize: 20}

	if v := strings.TrimSpace(c.Query("device_sn")); v != "" {
		filter.DeviceSN = v
	}
	if v := strings.TrimSpace(c.Query("event_type")); v != "" {
		filter.EventType = v
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
		h.logger.Error("list event logs", zap.Error(err))
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

// Get 详情
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	item, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, item)
}
