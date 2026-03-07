package datamodel

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// Handler provides HTTP handlers for data model REST API.
type Handler struct {
	repo     DataModelRepository
	ouiRepo  OUIRepository
	registry *DataModelRegistry
	importer *DataModelImporter
}

// NewHandler creates a new data model REST API handler.
func NewHandler(repo DataModelRepository, ouiRepo OUIRepository, registry *DataModelRegistry, importer *DataModelImporter) *Handler {
	return &Handler{
		repo:     repo,
		ouiRepo:  ouiRepo,
		registry: registry,
		importer: importer,
	}
}

// RegisterRoutes registers data model routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dm := rg.Group("/datamodels")
	{
		dm.GET("", h.List)
		dm.GET("/resolve", h.Resolve)
		dm.GET("/statistics", h.Statistics)
		dm.POST("", h.Create)
		dm.POST("/import", h.Import)
		dm.POST("/cache/refresh", h.RefreshCache)
		dm.GET("/:id", h.Get)
		dm.GET("/:id/export", h.Export)
		dm.PUT("/:id", h.Update)
		dm.DELETE("/:id", h.Delete)
		dm.POST("/:id/activate", h.Activate)
		dm.POST("/:id/deprecate", h.Deprecate)
	}

	oui := rg.Group("/oui")
	{
		oui.GET("", h.ListOUI)
		oui.POST("", h.CreateOUI)
	}
}

// List handles GET /api/v1/datamodels.
func (h *Handler) List(c *gin.Context) {
	filter := DataModelFilter{
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
	if oui := c.Query("oui"); oui != "" {
		filter.OUI = oui
	}
	if pc := c.Query("product_class"); pc != "" {
		filter.ProductClass = pc
	}
	if scope := c.Query("scope"); scope != "" {
		filter.Scope = model.DataModelScope(scope)
	}
	if status := c.Query("status"); status != "" {
		filter.Status = DataModelStatus(status)
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Get handles GET /api/v1/datamodels/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	dm, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "data model not found"})
		return
	}
	c.JSON(http.StatusOK, dm)
}

type createDataModelRequest struct {
	Carrier         string          `json:"carrier" binding:"required"`
	Technology      string          `json:"technology" binding:"required"`
	Version         string          `json:"version" binding:"required"`
	OUI             string          `json:"oui"`
	ProductClass    string          `json:"product_class"`
	RootObject      string          `json:"root_object"`
	ParameterTree   json.RawMessage `json:"parameter_tree" binding:"required"`
	Source          string          `json:"source"`
	SpecDocumentRef string          `json:"spec_document_ref"`
	Description     string          `json:"description"`
}

// Create handles POST /api/v1/datamodels.
func (h *Handler) Create(c *gin.Context) {
	var req createDataModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scope := determineScope(req.OUI, req.ProductClass)
	rootObject := req.RootObject
	if rootObject == "" {
		rootObject = "Device."
	}

	dm := &DataModel{
		Carrier:         model.CarrierCode(req.Carrier),
		Technology:      model.Technology(req.Technology),
		Version:         req.Version,
		OUI:             req.OUI,
		ProductClass:    req.ProductClass,
		Scope:           scope,
		Status:          StatusDraft,
		IsActive:        false,
		RootObject:      rootObject,
		ParameterTree:   req.ParameterTree,
		Source:          req.Source,
		SpecDocumentRef: req.SpecDocumentRef,
		Description:     req.Description,
	}

	if err := h.repo.Create(c.Request.Context(), dm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dm)
}

type updateDataModelRequest struct {
	Version         string          `json:"version" binding:"required"`
	RootObject      string          `json:"root_object"`
	ParameterTree   json.RawMessage `json:"parameter_tree" binding:"required"`
	Source          string          `json:"source"`
	SpecDocumentRef string          `json:"spec_document_ref"`
	Description     string          `json:"description"`
}

// Update handles PUT /api/v1/datamodels/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	var req updateDataModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "data model not found"})
		return
	}

	existing.Version = req.Version
	if req.RootObject != "" {
		existing.RootObject = req.RootObject
	}
	existing.ParameterTree = req.ParameterTree
	existing.Source = req.Source
	existing.SpecDocumentRef = req.SpecDocumentRef
	existing.Description = req.Description

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/datamodels/:id (only draft models).
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// Activate handles POST /api/v1/datamodels/:id/activate.
func (h *Handler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	if err := h.repo.Activate(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Invalidate caches after activation.
	dm, err := h.repo.GetByID(c.Request.Context(), id)
	if err == nil && h.registry != nil {
		_ = h.registry.InvalidateCache(c.Request.Context(), dm)
	}

	c.JSON(http.StatusOK, gin.H{"message": "data model activated"})
}

// Deprecate handles POST /api/v1/datamodels/:id/deprecate.
func (h *Handler) Deprecate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	// Get model before deprecating for cache invalidation.
	dm, _ := h.repo.GetByID(c.Request.Context(), id)

	if err := h.repo.Deprecate(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dm != nil && h.registry != nil {
		_ = h.registry.InvalidateCache(c.Request.Context(), dm)
	}

	c.JSON(http.StatusOK, gin.H{"message": "data model deprecated"})
}

// Import handles POST /api/v1/datamodels/import.
func (h *Handler) Import(c *gin.Context) {
	dm, err := h.importer.ImportFromJSON(c.Request.Context(), c.Request.Body, "api")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dm)
}

// Export handles GET /api/v1/datamodels/:id/export.
func (h *Handler) Export(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data model ID"})
		return
	}

	data, err := h.importer.ExportToJSON(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// Resolve handles GET /api/v1/datamodels/resolve?carrier=&technology=&oui=&product_class=.
func (h *Handler) Resolve(c *gin.Context) {
	carrier := model.CarrierCode(c.Query("carrier"))
	tech := model.Technology(c.Query("technology"))
	oui := c.Query("oui")
	productClass := c.Query("product_class")

	if carrier == "" || tech == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "carrier and technology are required"})
		return
	}

	dm, err := h.registry.Resolve(c.Request.Context(), carrier, tech, oui, productClass)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dm == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no matching data model found"})
		return
	}
	c.JSON(http.StatusOK, dm)
}

// Statistics handles GET /api/v1/datamodels/statistics.
func (h *Handler) Statistics(c *gin.Context) {
	stats, err := h.repo.Statistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// RefreshCache handles POST /api/v1/datamodels/cache/refresh.
func (h *Handler) RefreshCache(c *gin.Context) {
	if err := h.registry.InvalidateAll(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cache refreshed"})
}

// ListOUI handles GET /api/v1/oui.
func (h *Handler) ListOUI(c *gin.Context) {
	entries, err := h.ouiRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries, "total": len(entries)})
}

type createOUIRequest struct {
	OUI          string `json:"oui" binding:"required"`
	Manufacturer string `json:"manufacturer" binding:"required"`
	ShortName    string `json:"short_name" binding:"required"`
	Country      string `json:"country"`
}

// CreateOUI handles POST /api/v1/oui.
func (h *Handler) CreateOUI(c *gin.Context) {
	var req createOUIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := &OUIEntry{
		OUI:          req.OUI,
		Manufacturer: req.Manufacturer,
		ShortName:    req.ShortName,
		Country:      req.Country,
	}

	if err := h.ouiRepo.Create(c.Request.Context(), entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entry)
}

