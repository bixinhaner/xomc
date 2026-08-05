package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type ContactGroupHandler struct{ service *ContactGroupService }

func NewContactGroupHandler(service *ContactGroupService) *ContactGroupHandler {
	return &ContactGroupHandler{service: service}
}

func (h *ContactGroupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	groups := rg.Group("/notification-contact-groups")
	groups.GET("", h.List)
	groups.POST("", h.Create)
	groups.GET("/:id", h.Get)
	groups.PATCH("/:id", h.Update)
}

func (h *ContactGroupHandler) List(c *gin.Context) {
	groups, err := h.service.List(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, groups)
}

func (h *ContactGroupHandler) Get(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	group, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, group.Revision)
	response.OK(c, group)
}

func (h *ContactGroupHandler) Create(c *gin.Context) {
	var input ContactGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	group, err := h.service.Create(c.Request.Context(), input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, group.Revision)
	response.OKWithStatus(c, http.StatusCreated, group)
}

func (h *ContactGroupHandler) Update(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var input ContactGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	group, err := h.service.Update(c.Request.Context(), id, revision, input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, group.Revision)
	response.OK(c, group)
}
