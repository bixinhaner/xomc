package agentaction

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	jwtService *admin.JWTService
	service    *Service
	resolver   *authz.Resolver
	logger     *zap.Logger
}

func NewHandler(jwtService *admin.JWTService, service *Service, perm authz.VisibleGroupsResolver, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		jwtService: jwtService,
		service:    service,
		resolver:   authz.NewResolver(perm),
		logger:     logger.Named("agentaction"),
	}
}

func (h *Handler) RegisterDelegationRoutes(rg *gin.RouterGroup) {
	rg.POST("/agent/delegation", h.CreateDelegation)
}

func (h *Handler) RegisterActionRoutes(rg *gin.RouterGroup) {
	rg.GET("/actions", h.ListActions)
	rg.POST("/actions/search", h.SearchActions)
	rg.POST("/actions/describe", h.DescribeAction)
	rg.POST("/actions/preview", h.PreviewAction)
	rg.POST("/actions/execute", h.ExecuteAction)
}

func (h *Handler) CreateDelegation(c *gin.Context) {
	if h.jwtService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrUnavailable)
		return
	}
	claimsVal, exists := c.Get(admin.CtxKeyClaims)
	claims, ok := claimsVal.(*admin.Claims)
	if !exists || !ok || claims == nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}
	token, expiresAt, err := h.jwtService.GenerateAgentDelegationToken(claims)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, DelegationTokenResponse{Token: token, ExpiresAt: expiresAt})
}

func (h *Handler) ListActions(c *gin.Context) {
	response.OK(c, h.service.ListActions())
}

func (h *Handler) SearchActions(c *gin.Context) {
	var req SearchActionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, h.service.SearchActions(req.Query))
}

func (h *Handler) DescribeAction(c *gin.Context) {
	var req DescribeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	desc, err := h.service.DescribeAction(req.ActionID)
	if err != nil {
		writeActionError(c, err)
		return
	}
	response.OK(c, desc)
}

func (h *Handler) PreviewAction(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	actor, ok := h.actorContext(c)
	if !ok {
		return
	}
	preview, err := h.service.Preview(c.Request.Context(), req, actor)
	if err != nil {
		writeActionError(c, err)
		return
	}
	response.OK(c, preview)
}

func (h *Handler) ExecuteAction(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	actor, ok := h.actorContext(c)
	if !ok {
		return
	}
	result, err := h.service.Execute(c.Request.Context(), req, actor)
	if err != nil {
		writeActionError(c, err)
		return
	}
	h.logger.Info("agent action executed",
		zap.String("user_id", actor.UserID.String()),
		zap.String("username", actor.Username),
		zap.String("action_id", req.ActionID),
		zap.String("status", result.Status),
	)
	response.OK(c, result)
}

func (h *Handler) actorContext(c *gin.Context) (RequestContext, bool) {
	userIDVal, _ := c.Get(admin.CtxKeyUserID)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return RequestContext{}, false
	}
	username, _ := c.Get(admin.CtxKeyUsername)
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)
	groups, groupsOK := h.resolver.FromContext(c)
	if !groupsOK {
		return RequestContext{}, false
	}
	return RequestContext{
		UserID:        userID,
		Username:      usernameString(username),
		IsSuperAdmin:  isSuper,
		VisibleGroups: groups,
	}, true
}

func usernameString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func writeActionError(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	response.FailWithData(c, status, err.Error(), gin.H{
		"code": actionErrorCode(err),
	})
}
