package topology

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// VisibleGroupsResolver resolves which L2 device groups a user can see.
// nil slice means unrestricted (super admin), empty slice means no visible groups.
type VisibleGroupsResolver interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID, isSuperAdmin bool) ([]uuid.UUID, error)
}

// Handler provides HTTP handlers for topology (device group) REST API.
type Handler struct {
	repo     DeviceGroupRepository
	service  *DeviceGroupService
	siteRepo SiteRepository
	nodeRepo TopoNodeRepository
	edgeRepo TopoEdgeRepository
	syncSvc  *DeviceSyncService
	logger   *zap.Logger
	permSvc  VisibleGroupsResolver
}

// NewHandler creates a new topology REST API handler.
func NewHandler(repo DeviceGroupRepository, service *DeviceGroupService, siteRepo SiteRepository, nodeRepo TopoNodeRepository, edgeRepo TopoEdgeRepository, syncSvc *DeviceSyncService, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, service: service, siteRepo: siteRepo, nodeRepo: nodeRepo, edgeRepo: edgeRepo, syncSvc: syncSvc, logger: logger}
}

// SetPermissionService sets the data permission service for group-tree filtering.
func (h *Handler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permSvc = ps
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
		topo.POST("/nodes", h.CreateTopoNode)
		topo.GET("/nodes/:id", h.GetTopoNode)
		topo.PUT("/nodes/:id", h.UpdateTopoNode)
		topo.DELETE("/nodes/:id", h.DeleteTopoNode)
		topo.GET("/edges", h.ListTopoEdges)
		topo.POST("/edges", h.CreateTopoEdge)
		topo.GET("/edges/:id", h.GetTopoEdge)
		topo.PUT("/edges/:id", h.UpdateTopoEdge)
		topo.DELETE("/edges/:id", h.DeleteTopoEdge)
		topo.POST("/nodes/batch", h.BatchCreateTopoNodes)
		topo.GET("/graph", h.GetTopoGraph)
		topo.GET("/geo", h.GetGeoData)

		topo.POST("/sync", h.SyncDevicesFromTopology)
		topo.GET("/statistics", h.GetTopologyStatistics)
	}
}

func getOperator(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		return v.(string)
	}
	return ""
}

func (h *Handler) resolveVisibleGroups(c *gin.Context) ([]uuid.UUID, bool, error) {
	if h.permSvc == nil {
		return nil, false, nil
	}
	userIDVal, ok := c.Get(admin.CtxKeyUserID)
	if !ok {
		return nil, false, nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return nil, false, nil
	}
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)
	visibleGroups, err := h.permSvc.GetUserVisibleGroupIDs(c.Request.Context(), userID, isSuper)
	if err != nil {
		return nil, false, err
	}
	return visibleGroups, visibleGroups != nil, nil
}

func filterVisibleGroups(tree []DeviceGroup, visibleGroups []uuid.UUID) []DeviceGroup {
	if visibleGroups == nil {
		return tree
	}
	visibleSet := make(map[uuid.UUID]struct{}, len(visibleGroups))
	for _, id := range visibleGroups {
		visibleSet[id] = struct{}{}
	}
	return filterVisibleGroupNodes(tree, visibleSet)
}

func filterVisibleGroupNodes(nodes []DeviceGroup, visibleSet map[uuid.UUID]struct{}) []DeviceGroup {
	filtered := make([]DeviceGroup, 0, len(nodes))
	for _, node := range nodes {
		node.Children = filterVisibleGroupNodes(node.Children, visibleSet)
		_, selfVisible := visibleSet[node.ID]
		if !selfVisible && len(node.Children) == 0 {
			continue
		}
		if node.Level == 1 || len(node.Children) > 0 {
			childCount := 0
			for _, child := range node.Children {
				childCount += child.DeviceCount
			}
			node.DeviceCount = childCount
		}
		filtered = append(filtered, node)
	}
	return filtered
}

func buildVisibleGroupStats(tree []DeviceGroup) *GroupStats {
	stats := &GroupStats{}
	var walk func(nodes []DeviceGroup)
	walk = func(nodes []DeviceGroup) {
		for _, node := range nodes {
			stats.TotalGroups++
			if len(node.Children) == 0 {
				stats.GroupedDevices += node.DeviceCount
				continue
			}
			walk(node.Children)
		}
	}
	walk(tree)
	return stats
}

// --- Device Group handlers ---

// GetTreeWithCounts handles GET /device-groups/tree.
func (h *Handler) GetTreeWithCounts(c *gin.Context) {
	tree, stats, err := h.service.GetTreeWithCounts(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	visibleGroups, shouldFilter, err := h.resolveVisibleGroups(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if shouldFilter {
		tree = filterVisibleGroups(tree, visibleGroups)
		stats = buildVisibleGroupStats(tree)
	}
	response.OK(c, TreeResponse{Items: tree, Stats: stats})
}

// ListTree handles GET /groups (legacy, no counts).
func (h *Handler) ListTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	visibleGroups, shouldFilter, err := h.resolveVisibleGroups(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if shouldFilter {
		tree = filterVisibleGroups(tree, visibleGroups)
	}
	response.OK(c, gin.H{"items": tree})
}

// Get handles GET /device-groups/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	group, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// CreateGroup handles POST /device-groups.
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	group, err := h.service.CreateGroup(c.Request.Context(), req, getOperator(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, group)
}

// UpdateGroup handles PUT /device-groups/:id.
func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	group, err := h.service.UpdateGroup(c.Request.Context(), id, req, getOperator(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, group)
}

// DeleteGroup handles DELETE /device-groups/:id.
func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	if err := h.service.DeleteGroup(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, nil)
}

// CheckDelete handles GET /device-groups/:id/check-delete.
func (h *Handler) CheckDelete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	resp, err := h.service.CheckDelete(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, resp)
}

// GetStats handles GET /device-groups/stats.
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, stats)
}

// BatchSort handles PUT /device-groups/sort.
func (h *Handler) BatchSort(c *gin.Context) {
	var req BatchSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.BatchSort(c.Request.Context(), req.Items); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "sort updated")
}

// BatchAddDevices handles POST /device-groups/:id/devices.
func (h *Handler) BatchAddDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	var req BatchDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Validate target is L2.
	group, err := h.repo.GetByID(c.Request.Context(), groupID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if group.Level != 2 {
		response.Fail(c, http.StatusBadRequest, "devices can only be added to level-2 groups")
		return
	}

	affected, err := h.service.BatchAddDevices(c.Request.Context(), groupID, deviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"affected": affected})
}

// BatchRemoveDevices handles DELETE /device-groups/:id/devices.
func (h *Handler) BatchRemoveDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	var req BatchDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	deviceIDs, err := parseUUIDs(req.DeviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	affected, err := h.service.BatchRemoveDevices(c.Request.Context(), groupID, deviceIDs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"affected": affected})
}

// MoveDevicesHandler handles POST /device-groups/move-devices.
func (h *Handler) MoveDevicesHandler(c *gin.Context) {
	var req MoveDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	affected, err := h.service.MoveDevices(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"affected": affected})
}

// ListDevices handles GET /device-groups/:id/devices.
func (h *Handler) ListDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	ids, err := h.repo.ListDeviceIDs(c.Request.Context(), groupID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"device_ids": ids, "total": len(ids)})
}

// AddDevice handles POST /groups/:id/devices (legacy single device).
func (h *Handler) AddDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return
	}

	if err := h.service.AddDevice(c.Request.Context(), groupID, deviceID); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{"message": "device added to group"})
}

// RemoveDevice handles DELETE /groups/:id/devices/:deviceId (legacy).
func (h *Handler) RemoveDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device ID")
		return
	}

	if err := h.service.RemoveDevice(c.Request.Context(), groupID, deviceID); err != nil {
		response.Fail(c, http.StatusNotFound, "device not in group")
		return
	}
	response.OK(c, nil)
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

	response.OK(c, result)
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

	response.OKWithStatus(c, http.StatusCreated, site)
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

	response.OK(c, site)
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
	if status := c.Query("status"); status != "" {
		s := NodeStatus(status)
		filter.Status = &s
	}

	result, err := h.nodeRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
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

	if status := c.Query("status"); status != "" {
		s := EdgeStatus(status)
		filter.Status = &s
	}

	result, err := h.edgeRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
}

// GetTopoGraph handles GET /api/v1/topology/graph.
// Supports ?layout_type=hierarchy|force|circular to apply layout algorithm.
// Supports ?domain_id to filter by domain.
// Supports ?node_type to filter by node type (eNB, gNB, CPE, etc.).
// Supports ?status to filter by node status (online, offline, alarm, maintenance).
// Supports ?limit=N to cap node count (default: 200, max: 500).
// Recommended limits: 50 (detailed), 100 (standard), 200 (default), 500 (large).
func (h *Handler) GetTopoGraph(c *gin.Context) {
	startTotal := time.Now()

	var domainID *uuid.UUID
	if did := c.Query("domain_id"); did != "" {
		id, err := uuid.Parse(did)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		domainID = &id
	}

	var nodeType *string
	if nt := c.Query("node_type"); nt != "" {
		nodeType = &nt
	}

	var status *string
	if st := c.Query("status"); st != "" {
		// Validate status value
		validStatuses := map[string]bool{
			"online":      true,
			"offline":     true,
			"alarm":       true,
			"maintenance": true,
		}
		if !validStatuses[st] {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		status = &st
	}

	// Tiered limit options: 50, 100, 200 (default), 500 (max)
	limit := 200
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	// Cap at 500 for performance (force layout is O(n²))
	if limit > 500 {
		limit = 500
	}

	// Query nodes
	startDB := time.Now()
	nodes, err := h.nodeRepo.ListAll(c.Request.Context(), domainID, nodeType, status, limit)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	dbDuration := time.Since(startDB)

	// Skip detailed log for healthy queries (reduce logging overhead)
	if dbDuration > 500*time.Millisecond {
		h.logger.Info("topo_graph_db_slow",
			zap.Int("node_count", len(nodes)),
			zap.Duration("db_query_total", dbDuration),
		)
	}

	// Query edges - pre-allocate slice with capacity for better performance
	startEdges := time.Now()
	nodeIDs := make([]uuid.UUID, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID)
	}
	edges, err := h.edgeRepo.ListByNodeIDs(c.Request.Context(), nodeIDs)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	edgesDuration := time.Since(startEdges)

	// Apply layout algorithm if requested
	var layoutDuration time.Duration
	layoutType := c.Query("layout_type")
	if layoutType != "" {
		var cfg LayoutConfig
		// For large datasets, skip layout entirely for performance
		nodeCount := len(nodes)
		if nodeCount > 300 {
			// Skip layout for very large datasets - use coordinates from DB
			h.logger.Debug("skipping layout for large dataset",
				zap.Int("node_count", nodeCount),
				zap.Duration("db_query", dbDuration),
				zap.Duration("edges_query", edgesDuration))
		} else {
			startLayout := time.Now()
			switch {
			case nodeCount > 200:
				cfg = FastLayoutConfig()
				cfg.Type = layoutType
			default:
				cfg = DefaultLayoutConfig()
				cfg.Type = layoutType
			}
			algorithm := NewLayoutAlgorithm(cfg, h.logger)

			nodes, err = algorithm.Apply(c.Request.Context(), nodes, edges)
			if err != nil {
				h.logger.Warn("failed to apply layout", zap.String("type", layoutType), zap.Error(err))
			}
			layoutDuration = time.Since(startLayout)
		}
	}

	// Calculate statistics
	startStats := time.Now()
	statistics := CalculateStatistics(nodes, edges)
	statsDuration := time.Since(startStats)

	totalDuration := time.Since(startTotal)

	// Conditional logging: log detailed metrics only for slow requests (>500ms) or 10% sampling
	// This reduces logging overhead while maintaining visibility into performance issues
	shouldLogDetailed := totalDuration > 500*time.Millisecond || rand.Float64() < 0.1
	if shouldLogDetailed {
		h.logger.Debug("topo_graph_performance",
			zap.Int("node_count", len(nodes)),
			zap.Int("edge_count", len(edges)),
			zap.Duration("db_query", dbDuration),
			zap.Duration("edges_query", edgesDuration),
			zap.Duration("layout", layoutDuration),
			zap.Duration("statistics", statsDuration),
			zap.Duration("total", totalDuration),
			zap.String("layout_type", layoutType),
		)
	}

	response.OK(c, TopoGraph{
		Nodes:      nodes,
		Edges:      edges,
		Statistics: statistics,
	})
}

// GetGeoData handles GET /api/v1/topology/geo.
func (h *Handler) GetGeoData(c *gin.Context) {
	sites, err := h.siteRepo.ListWithCoordinates(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	nodes, err := h.nodeRepo.ListAll(c.Request.Context(), nil, nil, nil, 0) // 0 = no limit for geo data
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, GeoData{
		Sites: sites,
		Nodes: nodes,
	})
}

// ---- Topology Node CRUD handlers ----

// CreateTopoNodeRequest creates a topology node request.
type CreateTopoNodeRequest struct {
	Label    string  `json:"label" binding:"required"`
	NodeType string  `json:"node_type" binding:"required,oneof=eNB gNB CPE eGW domain router switch"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Status   string  `json:"status" binding:"omitempty,oneof=online offline alarm maintenance"`
	DeviceSN string  `json:"device_sn,omitempty"`
	SiteID   string  `json:"site_id,omitempty"`
	DomainID string  `json:"domain_id,omitempty"`
}

// CreateTopoNode handles POST /api/v1/topology/nodes.
func (h *Handler) CreateTopoNode(c *gin.Context) {
	var req CreateTopoNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	node := &TopoNode{
		Label:    req.Label,
		NodeType: req.NodeType,
		X:        req.X,
		Y:        req.Y,
		Status:   NodeStatus(req.Status),
		DeviceSN: req.DeviceSN,
	}

	if req.SiteID != "" {
		siteID, err := uuid.Parse(req.SiteID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		node.SiteID = &siteID
	}

	if req.DomainID != "" {
		domainID, err := uuid.Parse(req.DomainID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		node.DomainID = &domainID
	}

	if err := h.nodeRepo.Create(c.Request.Context(), node); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, node)
}

// GetTopoNode handles GET /api/v1/topology/nodes/:id.
func (h *Handler) GetTopoNode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	node, err := h.nodeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, node)
}

// UpdateTopoNodeRequest updates a topology node request.
type UpdateTopoNodeRequest struct {
	Label    *string  `json:"label"`
	NodeType *string  `json:"node_type"`
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Status   *string  `json:"status"`
	DeviceSN *string  `json:"device_sn"`
	SiteID   *string  `json:"site_id"`
	DomainID *string  `json:"domain_id"`
}

// UpdateTopoNode handles PUT /api/v1/topology/nodes/:id.
func (h *Handler) UpdateTopoNode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateTopoNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	node, err := h.nodeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if req.Label != nil {
		node.Label = *req.Label
	}
	if req.NodeType != nil {
		node.NodeType = *req.NodeType
	}
	if req.X != nil {
		node.X = *req.X
	}
	if req.Y != nil {
		node.Y = *req.Y
	}
	if req.Status != nil {
		node.Status = NodeStatus(*req.Status)
	}
	if req.DeviceSN != nil {
		node.DeviceSN = *req.DeviceSN
	}
	if req.SiteID != nil {
		siteID, err := uuid.Parse(*req.SiteID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		node.SiteID = &siteID
	}
	if req.DomainID != nil {
		domainID, err := uuid.Parse(*req.DomainID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		node.DomainID = &domainID
	}

	if err := h.nodeRepo.Update(c.Request.Context(), node); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, node)
}

// DeleteTopoNode handles DELETE /api/v1/topology/nodes/:id.
func (h *Handler) DeleteTopoNode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.nodeRepo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}

// ---- Topology Edge CRUD handlers ----

// CreateTopoEdgeRequest creates a topology edge request.
type CreateTopoEdgeRequest struct {
	SourceID string `json:"source_id" binding:"required"`
	TargetID string `json:"target_id" binding:"required"`
	Label    string `json:"label,omitempty"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive degraded"`
}

// CreateTopoEdge handles POST /api/v1/topology/edges.
func (h *Handler) CreateTopoEdge(c *gin.Context) {
	var req CreateTopoEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	edge := &TopoEdge{
		SourceID: sourceID,
		TargetID: targetID,
		Label:    req.Label,
		Status:   EdgeStatus(req.Status),
	}

	if err := h.edgeRepo.Create(c.Request.Context(), edge); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, edge)
}

// GetTopoEdge handles GET /api/v1/topology/edges/:id.
func (h *Handler) GetTopoEdge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	edge, err := h.edgeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, edge)
}

// UpdateTopoEdgeRequest updates a topology edge request.
type UpdateTopoEdgeRequest struct {
	SourceID *string `json:"source_id"`
	TargetID *string `json:"target_id"`
	Label    *string `json:"label"`
	Status   *string `json:"status"`
}

// UpdateTopoEdge handles PUT /api/v1/topology/edges/:id.
func (h *Handler) UpdateTopoEdge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateTopoEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	edge, err := h.edgeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	if req.SourceID != nil {
		sourceID, err := uuid.Parse(*req.SourceID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		edge.SourceID = sourceID
	}
	if req.TargetID != nil {
		targetID, err := uuid.Parse(*req.TargetID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		edge.TargetID = targetID
	}
	if req.Label != nil {
		edge.Label = *req.Label
	}
	if req.Status != nil {
		edge.Status = EdgeStatus(*req.Status)
	}

	if err := h.edgeRepo.Update(c.Request.Context(), edge); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, edge)
}

// DeleteTopoEdge handles DELETE /api/v1/topology/edges/:id.
func (h *Handler) DeleteTopoEdge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.edgeRepo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}

// ---- Batch Create Topo Nodes ----

// BatchCreateTopoNodesRequest batch creates topology nodes request.
type BatchCreateTopoNodesRequest struct {
	SiteID    string   `json:"site_id,omitempty"`
	DomainID  string   `json:"domain_id,omitempty"`
	NodeTypes []string `json:"node_types,omitempty"`
	Limit     int      `json:"limit,omitempty"`
}

// BatchCreateTopoNodes handles POST /api/v1/topology/nodes/batch.
func (h *Handler) BatchCreateTopoNodes(c *gin.Context) {
	var req BatchCreateTopoNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	nodes, err := h.service.CreateTopoNodesFromDevices(c.Request.Context(), req.SiteID, req.DomainID, req.NodeTypes, req.Limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, gin.H{"created": len(nodes), "nodes": nodes})
}

// SyncDevicesFromTopology handles POST /api/v1/topology/sync.
// Synchronizes devices from the devices table to topology nodes.
func (h *Handler) SyncDevicesFromTopology(c *gin.Context) {
	var domainID *uuid.UUID
	if did := c.Query("domain_id"); did != "" {
		id, err := uuid.Parse(did)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		domainID = &id
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if h.syncSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("sync service not available"))
		return
	}

	result, err := h.syncSvc.SyncFromDevices(c.Request.Context(), domainID, limit)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
}

// GetTopologyStatistics handles GET /api/v1/topology/statistics.
// Returns topology statistics with optional domain filtering.
func (h *Handler) GetTopologyStatistics(c *gin.Context) {
	var domainID *uuid.UUID
	if did := c.Query("domain_id"); did != "" {
		id, err := uuid.Parse(did)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
			return
		}
		domainID = &id
	}

	nodes, err := h.nodeRepo.ListAll(c.Request.Context(), domainID, nil, nil, 0) // 0 = no limit for statistics
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	edges, err := h.edgeRepo.ListAll(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	statistics := CalculateStatistics(nodes, edges)
	response.OK(c, statistics)
}
