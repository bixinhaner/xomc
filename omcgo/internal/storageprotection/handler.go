package storageprotection

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// RegisterRoutes mounts admin-only storage protection endpoints. The caller
// supplies the existing /api/v1/admin RBAC group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/storage-protection")
	group.GET("/policies", h.listPolicies)
	group.GET("/targets", h.listTargets)
	group.GET("/events", h.listEvents)
	group.POST("/policies", h.savePolicy)
	group.PUT("/policies/:id", h.savePolicy)
}

func (h *Handler) listTargets(c *gin.Context) {
	targets, err := h.service.ListTargets(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, targets, "查询成功")
}

func (h *Handler) listEvents(c *gin.Context) {
	limit := 5
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			commonerrors.AbortWithError(c, http.StatusBadRequest, errInvalidPolicy("limit must be a positive integer"))
			return
		}
		limit = parsed
	}
	events, err := h.service.ListEvents(c.Request.Context(), TargetType(c.Query("target_type")), c.Query("target_id"), limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, events, "查询成功")
}

func (h *Handler) listPolicies(c *gin.Context) {
	policies, err := h.service.ListPolicies(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, policies, "查询成功")
}

func (h *Handler) savePolicy(c *gin.Context) {
	var policy Policy
	if err := c.ShouldBindJSON(&policy); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if id := c.Param("id"); id != "" {
		policy.ID = id
	}
	saved, err := h.service.SavePolicy(c.Request.Context(), &policy)
	if err != nil {
		status := http.StatusInternalServerError
		if IsValidationError(err) {
			status = http.StatusBadRequest
		}
		commonerrors.AbortWithError(c, status, err)
		return
	}
	response.OKWithMsg(c, saved, "保存成功")
}
