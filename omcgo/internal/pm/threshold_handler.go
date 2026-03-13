package pm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// ThresholdHandler provides REST API endpoints for KPI threshold management.
type ThresholdHandler struct {
	repo   ThresholdRepository
	logger *zap.Logger
}

// NewThresholdHandler creates a new KPI threshold handler.
func NewThresholdHandler(repo ThresholdRepository, logger *zap.Logger) *ThresholdHandler {
	return &ThresholdHandler{repo: repo, logger: logger}
}

// RegisterRoutes registers KPI threshold API routes under the given router group.
func (h *ThresholdHandler) RegisterRoutes(rg *gin.RouterGroup) {
	thresholds := rg.Group("/thresholds")
	{
		thresholds.GET("", h.ListThresholds)
		thresholds.GET("/:id", h.GetThreshold)
		thresholds.POST("", h.CreateThreshold)
		thresholds.PUT("/:id", h.UpdateThreshold)
		thresholds.DELETE("/:id", h.DeleteThreshold)
	}
}

type thresholdQuery struct {
	KPIName    string `form:"kpi_name"`
	Carrier    string `form:"carrier"`
	Technology string `form:"technology"`
	Enabled    string `form:"enabled"`
	model.ListRequest
}

// ListThresholds handles GET /thresholds.
func (h *ThresholdHandler) ListThresholds(c *gin.Context) {
	var q thresholdQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := KPIThresholdFilter{ListRequest: q.ListRequest}
	if q.KPIName != "" {
		filter.KPIName = &q.KPIName
	}
	if q.Carrier != "" {
		filter.Carrier = &q.Carrier
	}
	if q.Technology != "" {
		filter.Technology = &q.Technology
	}
	if q.Enabled != "" {
		enabled := q.Enabled == "true"
		filter.Enabled = &enabled
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetThreshold handles GET /thresholds/:id.
func (h *ThresholdHandler) GetThreshold(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	threshold, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, threshold)
}

// CreateThreshold handles POST /thresholds.
func (h *ThresholdHandler) CreateThreshold(c *gin.Context) {
	var req CreateKPIThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	threshold := &KPIThreshold{
		KPIName:           req.KPIName,
		Carrier:           req.Carrier,
		Technology:        req.Technology,
		WarningThreshold:  req.WarningThreshold,
		MinorThreshold:    req.MinorThreshold,
		MajorThreshold:    req.MajorThreshold,
		CriticalThreshold: req.CriticalThreshold,
		Comparison:        req.Comparison,
		Enabled:           true,
		Description:       req.Description,
	}

	if threshold.Comparison == "" {
		threshold.Comparison = "gt"
	}
	if req.Enabled != nil {
		threshold.Enabled = *req.Enabled
	}

	if err := h.repo.Create(c.Request.Context(), threshold); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, threshold)
}

// UpdateThreshold handles PUT /thresholds/:id.
func (h *ThresholdHandler) UpdateThreshold(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	var req UpdateKPIThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if req.KPIName != nil {
		existing.KPIName = *req.KPIName
	}
	if req.Carrier != nil {
		existing.Carrier = *req.Carrier
	}
	if req.Technology != nil {
		existing.Technology = *req.Technology
	}
	if req.WarningThreshold != nil {
		existing.WarningThreshold = req.WarningThreshold
	}
	if req.MinorThreshold != nil {
		existing.MinorThreshold = req.MinorThreshold
	}
	if req.MajorThreshold != nil {
		existing.MajorThreshold = req.MajorThreshold
	}
	if req.CriticalThreshold != nil {
		existing.CriticalThreshold = req.CriticalThreshold
	}
	if req.Comparison != nil {
		existing.Comparison = *req.Comparison
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, existing)
}

// DeleteThreshold handles DELETE /thresholds/:id.
func (h *ThresholdHandler) DeleteThreshold(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "threshold deleted"})
}
