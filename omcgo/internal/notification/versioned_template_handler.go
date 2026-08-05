package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type TemplateManagementHandler struct{ service *TemplateManagementService }

func NewTemplateManagementHandler(service *TemplateManagementService) *TemplateManagementHandler {
	return &TemplateManagementHandler{service: service}
}

func (h *TemplateManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/notification-templates")
	templates.GET("", h.List)
	templates.POST("", h.Create)
	templates.GET("/:id", h.Get)
	templates.PATCH("/:id", h.UpdateDraft)
	templates.POST("/:id/publish", h.Publish)
	templates.POST("/:id/preview", h.Preview)
}

func (h *TemplateManagementHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *TemplateManagementHandler) Get(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}

func (h *TemplateManagementHandler) Create(c *gin.Context) {
	var input ManagedTemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(c.Request.Context(), input, requestActor(c))
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OKWithStatus(c, http.StatusCreated, item)
}

func (h *TemplateManagementHandler) UpdateDraft(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	var input ManagedTemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.UpdateDraft(c.Request.Context(), id, revision, input, requestActor(c))
	writeTemplateMutation(c, item, err)
}

func (h *TemplateManagementHandler) Publish(c *gin.Context) {
	id, revision, ok := parseRuleMutation(c)
	if !ok {
		return
	}
	item, err := h.service.Publish(c.Request.Context(), id, revision, requestActor(c))
	writeTemplateMutation(c, item, err)
}

func (h *TemplateManagementHandler) Preview(c *gin.Context) {
	id, ok := parseRuleID(c)
	if !ok {
		return
	}
	var request struct {
		Template *ManagedTemplateInput `json:"template,omitempty"`
		Values   map[string]string     `json:"values"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	input := request.Template
	if input == nil {
		item, err := h.service.Get(c.Request.Context(), id)
		if err != nil {
			abortRuleError(c, err)
			return
		}
		if item.Draft == nil {
			abortRuleError(c, ErrDraftUnavailable)
			return
		}
		input = &ManagedTemplateInput{
			Name: item.Name, Channel: item.Draft.Channel, Language: item.Draft.Language,
			Subject: item.Draft.Subject, TextBody: item.Draft.TextBody, HTMLBody: item.Draft.HTMLBody,
			Variables: item.Draft.Variables, ChangeReason: item.Draft.ChangeReason,
		}
	}
	preview, err := h.service.Preview(*input, request.Values)
	if err != nil {
		abortRuleError(c, err)
		return
	}
	response.OK(c, preview)
}

func writeTemplateMutation(c *gin.Context, item *ManagedTemplate, err error) {
	if err != nil {
		abortRuleError(c, err)
		return
	}
	setRevisionETag(c, item.Revision)
	response.OK(c, item)
}
