package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// ColumnConfigHandler provides HTTP endpoints for user column configuration.
type ColumnConfigHandler struct {
	repo ColumnConfigRepository
}

// NewColumnConfigHandler creates a new ColumnConfigHandler.
func NewColumnConfigHandler(repo ColumnConfigRepository) *ColumnConfigHandler {
	return &ColumnConfigHandler{repo: repo}
}

// RegisterRoutes registers column config routes.
func (h *ColumnConfigHandler) RegisterRoutes(rg *gin.RouterGroup) {
	cc := rg.Group("/column-configs")
	{
		cc.GET("/:pageKey", h.GetColumnConfig)
		cc.PUT("/:pageKey", h.SaveColumnConfig)
	}
}

// GetColumnConfig handles GET /column-configs/:pageKey.
func (h *ColumnConfigHandler) GetColumnConfig(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	pageKey := c.Param("pageKey")
	config, err := h.repo.Get(c.Request.Context(), userID, pageKey)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if config == nil {
		// Return default columns.
		response.OK(c, gin.H{
			"page_key": pageKey,
			"columns":  DefaultDeviceColumns(),
		})
		return
	}

	response.OK(c, config)
}

// SaveColumnConfig handles PUT /column-configs/:pageKey.
func (h *ColumnConfigHandler) SaveColumnConfig(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}

	pageKey := c.Param("pageKey")

	var req struct {
		Columns []ColumnItem `json:"columns" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	config := &ColumnConfig{
		UserID:  userID,
		PageKey: pageKey,
		Columns: req.Columns,
	}

	if err := h.repo.Upsert(c.Request.Context(), config); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, config)
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDVal, exists := c.Get(admin.CtxKeyUserID)
	if !exists {
		return uuid.Nil, commonerrors.ErrUnauthorized
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, commonerrors.ErrUnauthorized
	}
	return userID, nil
}
