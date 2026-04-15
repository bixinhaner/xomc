package alarm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// LibraryHandler 告警库 HTTP 处理器。
type LibraryHandler struct {
	service *LibraryService
	logger  *zap.Logger
}

// NewLibraryHandler 创建告警库 Handler。
func NewLibraryHandler(service *LibraryService, logger *zap.Logger) *LibraryHandler {
	return &LibraryHandler{service: service, logger: logger}
}

// RegisterRoutes 注册告警库路由。
func (h *LibraryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.GetByID)
	rg.GET("/:id/i18n", h.ListI18n)
	rg.POST("", h.Create)
	rg.POST("/:id/i18n", h.CreateI18n)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.DELETE("/:id/i18n/:i18nId", h.DeleteI18n)
}

type libraryQuery struct {
	AlarmCode   string `form:"alarm_code"`
	AlarmSource string `form:"alarm_source"`
	Severity    int    `form:"severity"`
	Enabled     string `form:"enabled"`
	Carrier     string `form:"carrier"`
	EventType   string `form:"event_type"`
	Keyword    string `form:"keyword"`
	model.ListRequest
}

func (h *LibraryHandler) List(c *gin.Context) {
	var q libraryQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := AlarmLibraryFilter{ListRequest: q.ListRequest}
	if q.AlarmCode != "" {
		filter.AlarmCode = &q.AlarmCode
	}
	if q.AlarmSource != "" {
		filter.AlarmSource = &q.AlarmSource
	}
	if q.Severity != 0 {
		filter.Severity = &q.Severity
	}
	if q.Enabled != "" {
		enabled := q.Enabled == "true"
		filter.Enabled = &enabled
	}
	if q.Carrier != "" {
		filter.Carrier = &q.Carrier
	}
	if q.EventType != "" {
		filter.EventType = &q.EventType
	}
	if q.Keyword != "" {
		filter.Keyword = &q.Keyword
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *LibraryHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	lib, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	c.JSON(http.StatusOK, lib)
}

func (h *LibraryHandler) Create(c *gin.Context) {
	var req CreateAlarmLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	lib, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, lib)
}

func (h *LibraryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var req UpdateAlarmLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	lib, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, lib)
}

func (h *LibraryHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alarm library deleted"})
}

func (h *LibraryHandler) ListI18n(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	items, err := h.service.ListI18n(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if items == nil {
		items = []AlarmLibraryI18n{}
	}
	c.JSON(http.StatusOK, items)
}

func (h *LibraryHandler) CreateI18n(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var req CreateAlarmLibraryI18nRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	i18n, err := h.service.CreateI18n(c.Request.Context(), id, &req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, i18n)
}

func (h *LibraryHandler) DeleteI18n(c *gin.Context) {
	i18nID, err := uuid.Parse(c.Param("i18nId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.service.DeleteI18n(c.Request.Context(), i18nID); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "i18n deleted"})
}
