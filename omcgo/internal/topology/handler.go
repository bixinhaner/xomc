package topology

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for topology (device group) REST API.
type Handler struct {
	repo     DeviceGroupRepository
	service  *DeviceGroupService
	siteRepo SiteRepository
	nodeRepo TopoNodeRepository
	edgeRepo TopoEdgeRepository
}

// NewHandler creates a new topology REST API handler.
func NewHandler(repo DeviceGroupRepository, service *DeviceGroupService, siteRepo SiteRepository, nodeRepo TopoNodeRepository, edgeRepo TopoEdgeRepository) *Handler {
	return &Handler{repo: repo, service: service, siteRepo: siteRepo, nodeRepo: nodeRepo, edgeRepo: edgeRepo}
}

// RegisterRoutes registers topology routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	groups := rg.Group("/groups")
	{
		groups.GET("", h.ListTree)
		groups.POST("", h.Create)
		groups.GET("/:id", h.Get)
		groups.PUT("/:id", h.Update)
		groups.DELETE("/:id", h.Delete)
		groups.POST("/:id/devices", h.AddDevice)
		groups.DELETE("/:id/devices/:deviceId", h.RemoveDevice)
		groups.GET("/:id/devices", h.ListDevices)
	}

	// Site routes
	sites := rg.Group("/sites")
	{
		sites.GET("", h.ListSites)
		sites.POST("", h.CreateSite)
		sites.GET("/:id", h.GetSite)
	}

	// Topology graph routes
	topo := rg.Group("/topology")
	{
		topo.GET("/nodes", h.ListTopoNodes)
		topo.GET("/edges", h.ListTopoEdges)
		topo.GET("/graph", h.GetTopoGraph)
		topo.GET("/geo", h.GetGeoData)
	}
}

// ListTree handles GET /api/v1/groups (returns tree structure).
func (h *Handler) ListTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tree})
}

// Get handles GET /api/v1/groups/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

type createGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	ParentID    string `json:"parent_id"`
	Carrier     string `json:"carrier"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// Create handles POST /api/v1/groups.
func (h *Handler) Create(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := &DeviceGroup{
		Name:        req.Name,
		Carrier:     model.CarrierCode(req.Carrier),
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		group.ParentID = &pid
	}

	if err := h.repo.Create(c.Request.Context(), group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, group)
}

type updateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	ParentID    string `json:"parent_id"`
	Carrier     string `json:"carrier"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// Update handles PUT /api/v1/groups/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	existing.Name = req.Name
	existing.Carrier = model.CarrierCode(req.Carrier)
	existing.Description = req.Description
	existing.SortOrder = req.SortOrder

	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		existing.ParentID = &pid
	} else {
		existing.ParentID = nil
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/groups/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

type addDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// AddDevice handles POST /api/v1/groups/:id/devices.
func (h *Handler) AddDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req addDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}

	if err := h.repo.AddDevice(c.Request.Context(), groupID, deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "device added to group"})
}

// RemoveDevice handles DELETE /api/v1/groups/:id/devices/:deviceId.
func (h *Handler) RemoveDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	if err := h.repo.RemoveDevice(c.Request.Context(), groupID, deviceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not in group"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ListDevices handles GET /api/v1/groups/:id/devices.
func (h *Handler) ListDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	ids, err := h.repo.ListDeviceIDs(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_ids": ids, "total": len(ids)})
}

// ---- Site handlers ----

// ListSites handles GET /api/v1/sites.
func (h *Handler) ListSites(c *gin.Context) {
	filter := SiteFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if domainID := c.Query("domain_id"); domainID != "" {
		id, err := uuid.Parse(domainID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DomainID = &id
	}
	if status := c.Query("status"); status != "" {
		s := SiteStatus(status)
		filter.Status = &s
	}

	result, err := h.siteRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type createSiteRequest struct {
	Name        string  `json:"name" binding:"required"`
	DomainID    string  `json:"domain_id"`
	Address     string  `json:"address"`
	Longitude   *float64 `json:"longitude"`
	Latitude    *float64 `json:"latitude"`
	DeviceCount int     `json:"device_count"`
	Status      string  `json:"status"`
}

// CreateSite handles POST /api/v1/sites.
func (h *Handler) CreateSite(c *gin.Context) {
	var req createSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	site := &Site{
		Name:        req.Name,
		Address:     req.Address,
		Longitude:   req.Longitude,
		Latitude:    req.Latitude,
		DeviceCount: req.DeviceCount,
		Status:      SiteStatus(req.Status),
	}

	if req.DomainID != "" {
		did, err := uuid.Parse(req.DomainID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		site.DomainID = &did
	}

	if err := h.siteRepo.Create(c.Request.Context(), site); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, site)
}

// GetSite handles GET /api/v1/sites/:id.
func (h *Handler) GetSite(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	site, err := h.siteRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, site)
}

// ---- Topology graph handlers ----

// ListTopoNodes handles GET /api/v1/topology/nodes.
func (h *Handler) ListTopoNodes(c *gin.Context) {
	filter := TopoNodeFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if domainID := c.Query("domain_id"); domainID != "" {
		id, err := uuid.Parse(domainID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		filter.DomainID = &id
	}
	if nodeType := c.Query("node_type"); nodeType != "" {
		filter.NodeType = &nodeType
	}

	result, err := h.nodeRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListTopoEdges handles GET /api/v1/topology/edges.
func (h *Handler) ListTopoEdges(c *gin.Context) {
	filter := TopoEdgeFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.edgeRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTopoGraph handles GET /api/v1/topology/graph.
func (h *Handler) GetTopoGraph(c *gin.Context) {
	var domainID *uuid.UUID
	if did := c.Query("domain_id"); did != "" {
		id, err := uuid.Parse(did)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		domainID = &id
	}

	nodes, err := h.nodeRepo.ListAll(c.Request.Context(), domainID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	edges, err := h.edgeRepo.ListAll(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, TopoGraph{
		Nodes: nodes,
		Edges: edges,
	})
}

// GetGeoData handles GET /api/v1/topology/geo.
func (h *Handler) GetGeoData(c *gin.Context) {
	sites, err := h.siteRepo.ListWithCoordinates(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	// Return nodes that have coordinates via their sites
	nodes, err := h.nodeRepo.ListAll(c.Request.Context(), nil)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, GeoData{
		Sites: sites,
		Nodes: nodes,
	})
}
