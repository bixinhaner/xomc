package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// LogHandler provides HTTP endpoints for system log management.
type LogHandler struct {
	repo LogRepository
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(repo LogRepository) *LogHandler {
	return &LogHandler{repo: repo}
}

// RegisterRoutes registers log routes on the given router group.
func (h *LogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/logs")
	{
		logs.GET("/login", h.ListLoginLogs)
		logs.GET("/operation", h.ListOperLogs)
		logs.GET("/task", h.ListTaskLogs)
	}
}

func (h *LogHandler) ListLoginLogs(c *gin.Context) {
	var filter LoginLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	result, err := h.repo.ListLoginLogs(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, result, "查询成功")
}

func (h *LogHandler) ListOperLogs(c *gin.Context) {
	var filter OperLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	result, err := h.repo.ListOperLogs(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, result, "查询成功")
}

func (h *LogHandler) ListTaskLogs(c *gin.Context) {
	var filter TaskLogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	result, err := h.repo.ListTaskLogs(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, result, "查询成功")
}
