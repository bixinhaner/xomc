package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type StatusSummaryConfigHandler struct {
	service *StatusSummaryConfigService
}

func NewStatusSummaryConfigHandler(service *StatusSummaryConfigService) *StatusSummaryConfigHandler {
	return &StatusSummaryConfigHandler{service: service}
}

func (h *StatusSummaryConfigHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/notification/status-summary-settings")
	g.GET("", h.Get)
	g.PATCH("", h.Update)
}

func (h *StatusSummaryConfigHandler) Get(c *gin.Context) {
	if !requireStatusSummaryAdmin(c) {
		return
	}
	config, err := h.service.Get(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, config.Revision)
	response.OK(c, config)
}

func (h *StatusSummaryConfigHandler) Update(c *gin.Context) {
	if !requireStatusSummaryAdmin(c) {
		return
	}
	revision, err := parseIfMatchRevision(c.GetHeader("If-Match"))
	if errors.Is(err, ErrPreconditionRequired) {
		commonerrors.AbortWithError(c, http.StatusPreconditionRequired, err)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	var input StatusSummaryConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	config, err := h.service.Update(c.Request.Context(), revision, input, requestActor(c))
	if errors.Is(err, ErrStatusSummaryNotReady) {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, config.Revision)
	response.OK(c, config)
}

func requireStatusSummaryAdmin(c *gin.Context) bool {
	value, ok := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuperAdmin, _ := value.(bool)
	if !ok || !isSuperAdmin {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return false
	}
	return true
}
