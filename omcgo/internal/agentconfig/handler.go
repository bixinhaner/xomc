package agentconfig

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/agent/health", h.Health)
}

func (h *Handler) RegisterRuntimeRoutes(rg *gin.RouterGroup) {
	rg.GET("/agent/config", h.GetRuntimeConfig)
}

func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/agent-config")
	{
		group.GET("", h.GetAdminConfig)
		group.POST("", h.Save)
		group.POST("/test", h.Test)
		group.POST("/sync", h.Sync)
	}
}

func (h *Handler) Health(c *gin.Context) {
	response.OK(c, HealthResponse{Status: "ok", CheckedAt: time.Now().UTC()})
}

func (h *Handler) GetAdminConfig(c *gin.Context) {
	cfg, err := h.service.GetAdminConfig(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, cfg)
}

func (h *Handler) GetRuntimeConfig(c *gin.Context) {
	cfg, err := h.service.GetRuntimeConfig(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, cfg)
}

func (h *Handler) Save(c *gin.Context) {
	req, ok := h.bindUpdate(c)
	if !ok {
		return
	}
	cfg, err := h.service.Save(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, cfg)
}

func (h *Handler) Test(c *gin.Context) {
	req, ok := h.bindUpdate(c)
	if !ok {
		return
	}
	result, err := h.service.Test(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Sync(c *gin.Context) {
	req, ok := h.bindUpdate(c)
	if !ok {
		return
	}
	cfg, err := h.service.Sync(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, cfg)
}

func (h *Handler) bindUpdate(c *gin.Context) (UpdateRequest, bool) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return UpdateRequest{}, false
	}
	if strings.TrimSpace(req.OMCPublicBaseURL) == "" {
		req.OMCPublicBaseURL = inferPublicBaseURL(c)
	}
	return req, true
}

func inferPublicBaseURL(c *gin.Context) string {
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		return ""
	}
	return proto + "://" + host
}
