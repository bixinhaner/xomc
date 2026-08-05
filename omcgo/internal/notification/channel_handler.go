package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type ChannelHandler struct{ service *ChannelService }

func NewChannelHandler(service *ChannelService) *ChannelHandler {
	return &ChannelHandler{service: service}
}

func (h *ChannelHandler) RegisterRoutes(rg *gin.RouterGroup) {
	channels := rg.Group("/notification-channels")
	channels.GET("", h.List)
	channels.PATCH("/:id", h.Update)
	channels.POST("/:id/verify", h.Verify)
	channels.GET("/:id/health", h.Health)
}

func (h *ChannelHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *ChannelHandler) Update(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var input ChannelConfigUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, revision, input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *ChannelHandler) Verify(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	if err := h.service.Verify(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrChannelVerificationUnavailable) {
			commonerrors.AbortWithError(c, http.StatusServiceUnavailable, err)
			return
		}
		abortRuleError(c, err)
		return
	}
	response.OK(c, gin.H{"verified": true})
}

func (h *ChannelHandler) Health(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	health, err := h.service.Health(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, health)
}
