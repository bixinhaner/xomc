package template

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for configuration template REST API.
type Handler struct {
	repo ConfigTemplateRepository
}

// NewHandler creates a new template REST API handler.
func NewHandler(repo ConfigTemplateRepository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers template routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/templates")
	{
		templates.GET("", h.List)
		templates.GET("/:id", h.Get)
		templates.POST("", h.Create)
		templates.PUT("/:id", h.Update)
		templates.DELETE("/:id", h.Delete)
	}
}

// List handles GET /api/v1/templates.
func (h *Handler) List(c *gin.Context) {
	filter := ConfigTemplateFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if carrier := c.Query("carrier"); carrier != "" {
		filter.Carrier = model.CarrierCode(carrier)
	}
	if tech := c.Query("technology"); tech != "" {
		filter.Technology = model.Technology(tech)
	}
	if pc := c.Query("product_class"); pc != "" {
		filter.ProductClass = pc
	}
	if tt := c.Query("template_type"); tt != "" {
		filter.TemplateType = TemplateType(tt)
	}
	if active := c.Query("active"); active == "true" {
		v := true
		filter.Active = &v
	} else if active == "false" {
		v := false
		filter.Active = &v
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Get handles GET /api/v1/templates/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	t, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

type createTemplateRequest struct {
	Name         string          `json:"name" binding:"required"`
	Carrier      string          `json:"carrier" binding:"required"`
	Technology   string          `json:"technology" binding:"required"`
	ProductClass string          `json:"product_class"`
	TemplateType string          `json:"template_type" binding:"required"`
	Parameters   json.RawMessage `json:"parameters" binding:"required"`
	Priority     int             `json:"priority"`
	Description  string          `json:"description"`
}

// Create handles POST /api/v1/templates.
func (h *Handler) Create(c *gin.Context) {
	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t := &ConfigTemplate{
		Name:         req.Name,
		Carrier:      model.CarrierCode(req.Carrier),
		Technology:   model.Technology(req.Technology),
		ProductClass: req.ProductClass,
		TemplateType: TemplateType(req.TemplateType),
		Parameters:   req.Parameters,
		Priority:     req.Priority,
		Version:      1,
		Active:       true,
		Description:  req.Description,
	}

	if err := h.repo.Create(c.Request.Context(), t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

type updateTemplateRequest struct {
	Name         string          `json:"name" binding:"required"`
	Carrier      string          `json:"carrier" binding:"required"`
	Technology   string          `json:"technology" binding:"required"`
	ProductClass string          `json:"product_class"`
	TemplateType string          `json:"template_type" binding:"required"`
	Parameters   json.RawMessage `json:"parameters" binding:"required"`
	Priority     int             `json:"priority"`
	Active       bool            `json:"active"`
	Description  string          `json:"description"`
}

// Update handles PUT /api/v1/templates/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	existing.Name = req.Name
	existing.Carrier = model.CarrierCode(req.Carrier)
	existing.Technology = model.Technology(req.Technology)
	existing.ProductClass = req.ProductClass
	existing.TemplateType = TemplateType(req.TemplateType)
	existing.Parameters = req.Parameters
	existing.Priority = req.Priority
	existing.Active = req.Active
	existing.Description = req.Description

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/templates/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
