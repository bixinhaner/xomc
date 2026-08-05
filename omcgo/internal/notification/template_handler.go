package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// TemplateHandler exposes the notification template CRUD endpoints.
type TemplateHandler struct {
	service *TemplateService
	logger  *zap.Logger
}

// NewTemplateHandler creates a TemplateHandler.
func NewTemplateHandler(service *TemplateService, logger *zap.Logger) *TemplateHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TemplateHandler{
		service: service,
		logger:  logger.Named("notification-template-handler"),
	}
}

// RegisterRoutes mounts /templates under the supplied router group.
// Caller is expected to mount the group at /api/v1/notifications, so the final
// paths become /api/v1/notifications/templates[...].
func (h *TemplateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/templates")
	templates.GET("", h.List)
	templates.GET("/:id", h.GetByID)
	templates.POST("", h.Create)
	templates.PUT("/:id", h.Update)
	templates.DELETE("/:id", h.Delete)
}

// RegisterReadOnlyRoutes keeps the legacy template representation available to
// the existing alert webhook without allowing new management writes to bypass
// immutable template versions. New writes use /notification-templates.
func (h *TemplateHandler) RegisterReadOnlyRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/templates")
	templates.GET("", h.List)
	templates.GET("/:id", h.GetByID)
}

// templateListQuery is the binding struct for GET /templates query params.
type templateListQuery struct {
	Channel  string `form:"channel"`
	Language string `form:"language"`
	Enabled  string `form:"enabled"`
	model.ListRequest
}

// List handles GET /templates.
func (h *TemplateHandler) List(c *gin.Context) {
	q := templateListQuery{ListRequest: model.DefaultListRequest()}
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := NotificationTemplateFilter{ListRequest: q.ListRequest}
	if q.Channel != "" {
		ch := q.Channel
		filter.Channel = &ch
	}
	if q.Language != "" {
		lang := q.Language
		filter.Language = &lang
	}
	if q.Enabled == "true" {
		v := true
		filter.Enabled = &v
	} else if q.Enabled == "false" {
		v := false
		filter.Enabled = &v
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

// GetByID handles GET /templates/:id.
func (h *TemplateHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	tpl, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, tpl)
}

// Create handles POST /templates.
func (h *TemplateHandler) Create(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tpl, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, tpl)
}

// Update handles PUT /templates/:id.
func (h *TemplateHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tpl, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, tpl)
}

// Delete handles DELETE /templates/:id.
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, nil)
}
