package topology

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
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
	// New device-groups routes (primary).
	dg := rg.Group("/device-groups")
	{
		dg.GET("/tree", h.GetTreeWithCounts)
		dg.GET("/stats", h.GetStats)
		dg.POST("", h.CreateGroup)
		dg.GET("/:id", h.Get)
		dg.PUT("/:id", h.UpdateGroup)
		dg.DELETE("/:id", h.DeleteGroup)
		dg.GET("/:id/check-delete", h.CheckDelete)
		dg.PUT("/sort", h.BatchSort)
		dg.POST("/:id/devices", h.BatchAddDevices)
		dg.DELETE("/:id/devices", h.BatchRemoveDevices)
		dg.POST("/move-devices", h.MoveDevicesHandler)
		dg.GET("/:id/devices", h.ListDevices)
	}

	// Legacy /groups routes (backward compat, delegates to same logic).
	groups := rg.Group("/groups")
	{
		groups.GET("", h.ListTree)
		groups.POST("", h.CreateGroup)
		groups.GET("/:id", h.Get)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
		groups.POST("/:id/devices", h.AddDevice)
		groups.DELETE("/:id/devices/:deviceId", h.RemoveDevice)
		groups.GET("/:id/devices", h.ListDevices)
	}

	// Site routes.
	sites := rg.Group("/sites")
	{
		sites.GET("", h.ListSites)
		sites.POST("", h.CreateSite)
		sites.GET("/:id", h.GetSite)
	}

	// Topology graph routes.
	topo := rg.Group("/topology")
	{
		topo.GET("/nodes", h.ListTopoNodes)
		topo.GET("/edges", h.ListTopoEdges)
		topo.GET("/graph", h.GetTopoGraph)
		topo.GET("/geo", h.GetGeoData)
	}
}

func getOperator(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		return v.(string)
	}
	return ""
}

// --- Device Group handlers ---

// GetTreeWithCounts handles GET /device-groups/tree.
func (h *Handler) GetTreeWithCounts(c *gin.Context) {
	tree, stats, err := h.service.GetTreeWithCounts(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, TreeResponse{Items: tree, Stats: stats})
}

// ListTree handles GET /groups (legacy, no counts).
func (h *Handler) ListTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tree})
}

// Get handles GET /device-groups/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, group)
}

// CreateGroup handles POST /device-groups.
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.service.CreateGroup(c.Request.Context(), req, getOperator(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusCreated, group)
}

// UpdateGroup handles PUT /device-groups/:id.
func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.service.UpdateGroup(c.Request.Context(), id, req, getOperator(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, group)
}

// DeleteGroup handles DELETE /device-groups/:id.
func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.service.DeleteGroup(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CheckDelete handles GET /device-groups/:id/check-delete.
func (h *Handler) CheckDelete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	resp, err := h.service.CheckDelete(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetStats handles GET /device-groups/stats.
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// BatchSort handles PUT /device-groups/sort.
func (h *Handler) BatchSort(c *gin.Context) {
	var req BatchSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.BatchSort(c.Request.Context(), req.Items); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sort updated"})
}

// BatchAddDevices handles POST /device-groups/:id/devices.
func (h *Handler) BatchAddDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req BatchDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate target is L2.
	group, err := h.repo.GetByID(c.Request.Context(), groupID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if group.Level != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "devices can only be added to level-2 groups"})
		return
	}

	affected, err := h.repo.BatchAddDevices(c.Request.Context(), groupID, deviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}

// BatchRemoveDevices handles DELETE /device-groups/:id/devices.
func (h *Handler) BatchRemoveDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req BatchDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	affected, err := h.repo.BatchRemoveDevices(c.Request.Context(), groupID, deviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}

// MoveDevicesHandler handles POST /device-groups/move-devices.
func (h *Handler) MoveDevicesHandler(c *gin.Context) {
	var req MoveDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	affected, err := h.service.MoveDevices(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}

// ListDevices handles GET /device-groups/:id/devices.
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

// AddDevice handles POST /groups/:id/devices (legacy single device).
func (h *Handler) AddDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
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

// RemoveDevice handles DELETE /groups/:id/devices/:deviceId (legacy).
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
	c.Status(http.StatusNoContent)
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
	Name        string   `json:"name" binding:"required"`
	DomainID    string   `json:"domain_id"`
	Address     string   `json:"address"`
	Longitude   *float64 `json:"longitude"`
	Latitude    *float64 `json:"latitude"`
	DeviceCount int      `json:"device_count"`
	Status      string   `json:"status"`
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
