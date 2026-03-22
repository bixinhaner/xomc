package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// ParameterTreeHandler provides REST API handlers for the device parameter tree.
type ParameterTreeHandler struct {
	deviceService *DeviceService
	paramRepo     DeviceParameterRepository
	logger        *zap.Logger
}

// NewParameterTreeHandler creates a new parameter tree handler.
func NewParameterTreeHandler(deviceService *DeviceService, paramRepo DeviceParameterRepository, logger *zap.Logger) *ParameterTreeHandler {
	return &ParameterTreeHandler{
		deviceService: deviceService,
		paramRepo:     paramRepo,
		logger:        logger,
	}
}

// RegisterRoutes registers parameter tree routes.
func (h *ParameterTreeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("/:id/parameters/tree", h.GetParameterTree)
		devices.GET("/:id/parameters/search", h.SearchParameters)
		devices.PUT("/:id/parameters", h.SetParameterValues)
		devices.POST("/:id/parameters/sync", h.TriggerSync)
		devices.POST("/:id/parameters/discover", h.TriggerDiscover)
		devices.GET("/:id/parameters/sync-status", h.GetSyncStatus)
	}
}

// ParameterTreeNode represents a node in the parameter tree hierarchy.
type ParameterTreeNode struct {
	Name     string               `json:"name"`
	FullPath string               `json:"full_path"`
	IsLeaf   bool                 `json:"is_leaf"`
	Value    string               `json:"value,omitempty"`
	Type     string               `json:"type,omitempty"`
	Writable bool                 `json:"writable"`
	Children []*ParameterTreeNode `json:"children,omitempty"`
}

// GetParameterTree handles GET /api/v1/devices/:id/parameter-tree.
// Returns device parameters organized as a hierarchical tree structure.
func (h *ParameterTreeHandler) GetParameterTree(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// Check if client wants flat or tree format.
	format := c.DefaultQuery("format", "tree")
	if format == "flat" {
		c.JSON(http.StatusOK, gin.H{"items": params, "total": len(params)})
		return
	}

	tree := buildTree(params)
	c.JSON(http.StatusOK, gin.H{"tree": tree, "total": len(params)})
}

// SearchParameters handles GET /api/v1/devices/:id/parameter-tree/search.
// Searches parameters by path keyword.
func (h *ParameterTreeHandler) SearchParameters(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	keyword := c.Query("q")
	if keyword == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// Filter parameters matching the keyword (case-insensitive).
	lowerKeyword := strings.ToLower(keyword)
	var matched []model.DeviceParameter
	for _, p := range params {
		if strings.Contains(strings.ToLower(p.ParameterPath), lowerKeyword) ||
			strings.Contains(strings.ToLower(p.ParameterValue), lowerKeyword) {
			matched = append(matched, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": matched, "total": len(matched)})
}

// SetParameterValuesRequest defines the request body for setting parameter values.
type SetParameterValuesRequest struct {
	Parameters []ParameterValueItem `json:"parameters" binding:"required,min=1"`
}

// SetParameterValues handles PUT /api/v1/devices/:id/parameter-tree.
// Queues SetParameterValues RPC for the device.
func (h *ParameterTreeHandler) SetParameterValues(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req SetParameterValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Verify device exists.
	dev, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dev == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	// Verify all parameters are writable.
	existingParams, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	writableMap := make(map[string]bool, len(existingParams))
	for _, p := range existingParams {
		writableMap[p.ParameterPath] = p.Writable
	}
	for _, item := range req.Parameters {
		writable, exists := writableMap[item.Path]
		if exists && !writable {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("parameter %s is not writable", item.Path))
			return
		}
	}

	// Queue SPV command via device service.
	if err := h.deviceService.SetParameters(c.Request.Context(), id, req.Parameters); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "set parameter values command queued",
		"parameters": len(req.Parameters),
	})
}

// TriggerSync handles POST /api/v1/devices/:id/parameters/sync.
// Triggers a manual parameter value sync for the device.
func (h *ParameterTreeHandler) TriggerSync(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	dev, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dev == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	// Get existing parameters to sync.
	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if len(params) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "no parameters to sync"})
		return
	}

	// Enqueue GPV for all parameter paths.
	paths := make([]string, 0, len(params))
	for _, p := range params {
		paths = append(paths, p.ParameterPath)
	}

	gpvParams, _ := json.Marshal(map[string]interface{}{
		"names": paths,
	})

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "GetParameterValues",
		Params:     gpvParams,
		Priority:   5,
		CommandKey: fmt.Sprintf("manual-sync-%s", uuid.New().String()[:8]),
	}

	if h.deviceService.GetCommandQueue() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("command queue not configured"))
		return
	}

	if err := h.deviceService.GetCommandQueue().Push(c.Request.Context(), dev.SerialNumber, cmd); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "parameter sync command queued",
		"parameters": len(paths),
	})
}

// TriggerDiscover handles POST /api/v1/devices/:id/parameters/discover.
// Triggers a manual parameter tree discovery for the device.
func (h *ParameterTreeHandler) TriggerDiscover(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	dev, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dev == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	// Enqueue GPN for full tree discovery.
	gpnParams, _ := json.Marshal(map[string]interface{}{
		"path":       "Device.",
		"next_level": false,
	})

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "GetParameterNames",
		Params:     gpnParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("manual-discover-%s", uuid.New().String()[:8]),
	}

	if h.deviceService.GetCommandQueue() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("command queue not configured"))
		return
	}

	if err := h.deviceService.GetCommandQueue().Push(c.Request.Context(), dev.SerialNumber, cmd); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "parameter discovery command queued"})
}

// GetSyncStatus handles GET /api/v1/devices/:id/parameters/sync-status.
// Returns the current sync/discovery status for the device.
func (h *ParameterTreeHandler) GetSyncStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	// Get parameter count and last update time.
	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// Check pending commands in queue.
	var pendingCommands int64
	if h.deviceService.GetCommandQueue() != nil {
		dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
		if devErr == nil && dev != nil {
			if lenQueue, ok := h.deviceService.GetCommandQueue().(interface {
				Len(ctx context.Context, deviceSN string) (int64, error)
			}); ok {
				pendingCommands, _ = lenQueue.Len(c.Request.Context(), dev.SerialNumber)
			}
		}
	}

	status := "idle"
	if pendingCommands > 0 {
		status = "syncing"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           status,
		"total_parameters": len(params),
		"pending_commands": pendingCommands,
	})
}

// buildTree converts a flat list of parameters into a hierarchical tree.
func buildTree(params []model.DeviceParameter) []*ParameterTreeNode {
	root := &ParameterTreeNode{Children: make([]*ParameterTreeNode, 0)}

	for _, p := range params {
		parts := strings.Split(p.ParameterPath, ".")
		current := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			fullPath := strings.Join(parts[:i+1], ".")
			var child *ParameterTreeNode
			for _, c := range current.Children {
				if c.Name == part {
					child = c
					break
				}
			}
			if child == nil {
				child = &ParameterTreeNode{
					Name:     part,
					FullPath: fullPath,
					Children: make([]*ParameterTreeNode, 0),
				}
				current.Children = append(current.Children, child)
			}
			if i == len(parts)-1 || (i == len(parts)-2 && parts[len(parts)-1] == "") {
				child.IsLeaf = true
				child.Value = p.ParameterValue
				child.Type = string(p.ParameterType)
				child.Writable = p.Writable
			}
			current = child
		}
	}

	return root.Children
}
