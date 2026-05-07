package mml

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// ParamHandler provides HTTP handlers for the parameter library API.
type ParamHandler struct {
	service *ParamService
	logger  *zap.Logger
}

// NewParamHandler creates a new ParamHandler.
func NewParamHandler(service *ParamService, logger *zap.Logger) *ParamHandler {
	return &ParamHandler{
		service: service,
		logger:  logger.Named("param-handler"),
	}
}

// RegisterRoutes registers parameter library routes on the given router group.
func (h *ParamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	params := rg.Group("/mml/param-versions")
	params.GET("", h.ListVersions)
	params.GET("/:version/groups", h.GetGroupTree)
	params.GET("/:version/groups/:groupId/params", h.GetGroupParams)
	params.GET("/:version/params", h.SearchParams)
}

// ListVersions returns all parameter library versions.
func (h *ParamHandler) ListVersions(c *gin.Context) {
	versions, err := h.service.ListVersions(c.Request.Context())
	if err != nil {
		h.logger.Error("list param versions failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if versions == nil {
		versions = []ParamVersion{}
	}
	response.OK(c, gin.H{"items": versions})
}

// GetGroupTree returns the hierarchical group tree for a version.
func (h *ParamHandler) GetGroupTree(c *gin.Context) {
	versionCode := c.Param("version")
	if versionCode == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	groups, err := h.service.GetGroupTree(c.Request.Context(), versionCode)
	if err != nil {
		h.logger.Error("get group tree failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if groups == nil {
		groups = []ParamGroup{}
	}
	response.OK(c, gin.H{"items": groups})
}

// GetGroupParams returns parameters for a specific group.
func (h *ParamHandler) GetGroupParams(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	params, err := h.service.GetGroupParams(c.Request.Context(), groupID)
	if err != nil {
		h.logger.Error("get group params failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if params == nil {
		params = []Param{}
	}
	response.OK(c, gin.H{"items": params})
}

// SearchParams searches parameters within a version.
func (h *ParamHandler) SearchParams(c *gin.Context) {
	versionCode := c.Param("version")
	if versionCode == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	filter := ParamFilter{VersionCode: versionCode}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}
	if tr069Path := c.Query("tr069_path"); tr069Path != "" {
		filter.Tr069Path = &tr069Path
	}

	params, err := h.service.SearchParams(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("search params failed", zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if params == nil {
		params = []Param{}
	}
	response.OK(c, gin.H{"items": params})
}
