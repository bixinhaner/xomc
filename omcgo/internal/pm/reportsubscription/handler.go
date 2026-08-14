package reportsubscription

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/pm/query-templates/:id/report-subscription")
	g.GET("", h.Get)
	g.PUT("", h.Upsert)
	g.DELETE("", h.Delete)
	g.GET("/runs", h.ListRuns)
}

type upsertDTO struct {
	Enabled    bool     `json:"enabled"`
	Period     Period   `json:"period" binding:"required"`
	SendTimes  []string `json:"send_times" binding:"required"`
	Recipients []string `json:"recipients" binding:"required"`
}

func (h *Handler) Get(c *gin.Context) {
	templateID, ok := parseTemplateID(c)
	if !ok {
		return
	}
	callerID, superAdmin := reportCallerInfo(c)
	item, err := h.service.Get(c.Request.Context(), templateID, callerID, superAdmin)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) Upsert(c *gin.Context) {
	templateID, ok := parseTemplateID(c)
	if !ok {
		return
	}
	var dto upsertDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	callerID, superAdmin := reportCallerInfo(c)
	if callerID == uuid.Nil {
		response.Fail(c, http.StatusUnauthorized, "login required")
		return
	}
	item, err := h.service.Upsert(c.Request.Context(), templateID, callerID, superAdmin, UpsertInput{
		Enabled: dto.Enabled, Period: dto.Period, SendTimes: dto.SendTimes, Recipients: dto.Recipients,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	templateID, ok := parseTemplateID(c)
	if !ok {
		return
	}
	callerID, superAdmin := reportCallerInfo(c)
	if err := h.service.Delete(c.Request.Context(), templateID, callerID, superAdmin); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) ListRuns(c *gin.Context) {
	templateID, ok := parseTemplateID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	callerID, superAdmin := reportCallerInfo(c)
	items, err := h.service.ListRuns(c.Request.Context(), templateID, callerID, superAdmin, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func parseTemplateID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid template id")
		return uuid.Nil, false
	}
	return id, true
}

func reportCallerInfo(c *gin.Context) (uuid.UUID, bool) {
	callerID, _ := c.Get(admin.CtxKeyUserID)
	id, _ := callerID.(uuid.UUID)
	superAdmin, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuperAdmin, _ := superAdmin.(bool)
	return id, isSuperAdmin
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "report subscription not found")
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, "no permission for this query template")
	case errors.Is(err, ErrInvalid):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
	}
}
