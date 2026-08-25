package attention

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Reader interface {
	Get(context.Context, Scope, int, int) (*Response, error)
	GetPage(context.Context, Scope, Section, int, int) (*Page, error)
}

type VisibleGroupsResolver interface {
	ResolveFromContext(*gin.Context) ([]uuid.UUID, error)
}

type Handler struct {
	service  Reader
	resolver VisibleGroupsResolver
}

func NewHandler(service Reader, resolver VisibleGroupsResolver) *Handler {
	return &Handler{service: service, resolver: resolver}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dashboard := rg.Group("/dashboard")
	dashboard.GET("/attention", h.Get)
	dashboard.GET("/attention/abnormalities", h.GetAbnormalities)
	dashboard.GET("/attention/todos", h.GetTodos)
}

func (h *Handler) Get(c *gin.Context) {
	abnormalLimit, ok := boundedQuery(c, "abnormal_limit", 1, 1, 5)
	if !ok {
		return
	}
	todoLimit, ok := boundedQuery(c, "todo_limit", 1, 1, 5)
	if !ok {
		return
	}
	scope, ok := h.scope(c)
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), scope, abnormalLimit, todoLimit)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) GetAbnormalities(c *gin.Context) {
	h.getPage(c, SectionAbnormalities)
}

func (h *Handler) GetTodos(c *gin.Context) {
	h.getPage(c, SectionTodos)
}

func (h *Handler) getPage(c *gin.Context, section Section) {
	page, ok := boundedQuery(c, "page", 1, 1, int(^uint(0)>>1))
	if !ok {
		return
	}
	pageSize, ok := boundedQuery(c, "page_size", 20, 1, 50)
	if !ok {
		return
	}
	scope, ok := h.scope(c)
	if !ok {
		return
	}
	result, err := h.service.GetPage(c.Request.Context(), scope, section, page, pageSize)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) scope(c *gin.Context) (Scope, bool) {
	userID, ok := admin.UserIDFromCtx(c)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return Scope{}, false
	}
	visibleGroups := []uuid.UUID(nil)
	if h.resolver != nil {
		groups, err := h.resolver.ResolveFromContext(c)
		if err != nil {
			status := commonerrors.HTTPStatusFromError(err)
			commonerrors.AbortWithError(c, status, err)
			return Scope{}, false
		}
		visibleGroups = groups
	}
	isSuperAdmin, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperAdmin.(bool)
	usernameValue, _ := c.Get(admin.CtxKeyUsername)
	username, _ := usernameValue.(string)
	return Scope{UserID: userID, Username: username, IsSuperAdmin: isSuper, VisibleGroups: visibleGroups}, true
}

func boundedQuery(c *gin.Context, key string, fallback, minimum, maximum int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid %s: %w", key, commonerrors.ErrInvalidInput))
		return 0, false
	}
	return value, true
}
