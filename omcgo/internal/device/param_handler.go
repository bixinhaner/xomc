package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// ParameterTreeHandler provides REST API handlers for the device parameter tree.
type ParameterTreeHandler struct {
	deviceService *DeviceService
	paramRepo     DeviceParameterRepository
	dmRegistry    *datamodel.DataModelRegistry
	logger        *zap.Logger
}

// NewParameterTreeHandler creates a new parameter tree handler.
func NewParameterTreeHandler(deviceService *DeviceService, paramRepo DeviceParameterRepository, dmRegistry *datamodel.DataModelRegistry, logger *zap.Logger) *ParameterTreeHandler {
	return &ParameterTreeHandler{
		deviceService: deviceService,
		paramRepo:     paramRepo,
		dmRegistry:    dmRegistry,
		logger:        logger,
	}
}

// RegisterRoutes registers parameter tree routes.
func (h *ParameterTreeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("/:id/parameters/tree", h.GetParameterTree)
		devices.GET("/:id/parameters/search", h.SearchParameters)
		devices.GET("/:id/parameters/schema", h.GetParameterSchema)
		devices.PUT("/:id/parameters", h.SetParameterValues)
		devices.POST("/:id/parameters/sync", h.TriggerSync)
		devices.POST("/:id/parameters/discover", h.TriggerDiscover)
		devices.GET("/:id/parameters/sync-status", h.GetSyncStatus)
		devices.POST("/:id/objects/add", h.AddObject)
		devices.POST("/:id/objects/delete", h.DeleteObject)
	}
}

// ParameterTreeNode represents a node in the parameter tree hierarchy.
type ParameterTreeNode struct {
	Name          string               `json:"name"`
	FullPath      string               `json:"full_path"`
	IsLeaf        bool                 `json:"is_leaf"`
	Value         string               `json:"value,omitempty"`
	Type          string               `json:"type,omitempty"`
	Writable      bool                 `json:"writable"`
	Children      []*ParameterTreeNode `json:"children,omitempty"`
	// Model metadata (enriched from data model)
	Description   string                     `json:"description,omitempty"`
	MultiInstance bool                       `json:"multi_instance,omitempty"`
	MaxInstances  int                        `json:"max_instances,omitempty"`
	MinInstances  int                        `json:"min_instances,omitempty"`
	InstanceCount int                        `json:"instance_count,omitempty"`
	CanAdd        bool                       `json:"can_add,omitempty"`
	CanDelete     bool                       `json:"can_delete,omitempty"`
	ChangeApplies string                     `json:"change_applies,omitempty"`
	DefaultValue  string                     `json:"default_value,omitempty"`
	Constraints   *datamodel.Constraints     `json:"constraints,omitempty"`
}

// ParameterSchemaItem represents a parameter with schema and current value info.
type ParameterSchemaItem struct {
	Path          string                 `json:"path"`
	Type          string                 `json:"type"`
	Writable      bool                   `json:"writable"`
	Description   string                 `json:"description,omitempty"`
	DefaultValue  string                 `json:"default_value,omitempty"`
	Notify        string                 `json:"notify,omitempty"`
	ForcedInform  bool                   `json:"forced_inform,omitempty"`
	ChangeApplies string                 `json:"change_applies,omitempty"`
	Category      string                 `json:"category,omitempty"`
	IsList        bool                   `json:"is_list,omitempty"`
	Constraints   *datamodel.Constraints `json:"constraints,omitempty"`
	CurrentValue  *string                `json:"current_value"`
	LastSyncedAt  *time.Time             `json:"last_synced_at,omitempty"`
}

// ObjectSchemaItem represents a multi-instance object with schema info.
type ObjectSchemaItem struct {
	Path             string `json:"path"`
	Access           string `json:"access"`
	MaxInstances     int    `json:"max_instances"`
	MinInstances     int    `json:"min_instances"`
	CurrentInstances []int  `json:"current_instances"`
	CanAdd           bool   `json:"can_add"`
	CanDeleteAny     bool   `json:"can_delete_any"`
	IsList           bool   `json:"is_list"`
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

	// Enrich tree with model metadata if available.
	if h.dmRegistry != nil {
		dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
		if devErr == nil && dev != nil {
			dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
			if dm != nil {
				validator, _ := datamodel.NewParameterValidator(dm)
				if validator != nil {
					enrichTreeWithModel(tree, validator)
				}
			}
		}
	}

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

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	matched, err := h.paramRepo.SearchByKeyword(c.Request.Context(), id, keyword, limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
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

	// Model-based validation (if data model is available).
	var rebootRequired bool
	if h.dmRegistry != nil {
		dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
		if dm != nil {
			validator, validatorErr := datamodel.NewParameterValidator(dm)
			if validatorErr == nil {
				var validationErrors []datamodel.ValidationError
				for _, item := range req.Parameters {
					if ve := validator.ValidateValue(item.Path, item.Value); ve != nil {
						validationErrors = append(validationErrors, *ve)
					}
				}
				if len(validationErrors) > 0 {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":             "parameter validation failed",
						"validation_errors": validationErrors,
					})
					return
				}
				// Check if any parameter requires reboot.
				for _, item := range req.Parameters {
					if def := validator.LookupParam(item.Path); def != nil {
						if def.ChangeApplies == "RebootRequired" || def.ChangeApplies == "NotifyRequired" {
							rebootRequired = true
							break
						}
					}
				}
			}
		}
	}

	// Queue SPV command via device service.
	if err := h.deviceService.SetParameters(c.Request.Context(), id, req.Parameters); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":         "set parameter values command queued",
		"parameters":      len(req.Parameters),
		"reboot_required": rebootRequired,
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

// GetParameterSchema handles GET /api/v1/devices/:id/parameters/schema.
// Returns parameters with schema metadata and current values.
func (h *ParameterTreeHandler) GetParameterSchema(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	pathPrefix := c.DefaultQuery("path_prefix", "")

	dev, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dev == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	// Get current parameter values.
	var params []model.DeviceParameter
	if pathPrefix != "" {
		params, err = h.paramRepo.GetByPathPrefix(c.Request.Context(), id, pathPrefix)
	} else {
		params, err = h.paramRepo.GetByDevice(c.Request.Context(), id)
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// Try to get data model for schema enrichment.
	var schemaItems []ParameterSchemaItem
	var objectItems []ObjectSchemaItem

	if h.dmRegistry != nil {
		dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
		if dm != nil {
			validator, _ := datamodel.NewParameterValidator(dm)
			if validator != nil {
				schemaItems = mergeSchemaWithValues(validator, params, pathPrefix)
				objectItems = buildObjectSchema(validator, params, pathPrefix)
			}
		}
	}

	if schemaItems == nil {
		// Fallback: return basic parameter info without model metadata.
		schemaItems = make([]ParameterSchemaItem, 0, len(params))
		for _, p := range params {
			val := p.ParameterValue
			schemaItems = append(schemaItems, ParameterSchemaItem{
				Path:         p.ParameterPath,
				Type:         string(p.ParameterType),
				Writable:     p.Writable,
				CurrentValue: &val,
				LastSyncedAt: &p.LastUpdatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"parameters": schemaItems,
		"objects":    objectItems,
		"total":      len(schemaItems),
	})
}

func mergeSchemaWithValues(validator *datamodel.ParameterValidator, params []model.DeviceParameter, pathPrefix string) []ParameterSchemaItem {
	var items []ParameterSchemaItem

	for _, p := range params {
		if pathPrefix != "" && !strings.HasPrefix(p.ParameterPath, pathPrefix) {
			continue
		}
		val := p.ParameterValue
		item := ParameterSchemaItem{
			Path:         p.ParameterPath,
			CurrentValue: &val,
			LastSyncedAt: &p.LastUpdatedAt,
		}

		if def := validator.LookupParam(p.ParameterPath); def != nil {
			item.Type = def.Type
			item.Writable = def.Writable
			item.Description = def.Description
			item.DefaultValue = def.DefaultValue
			item.Notify = def.Notify
			item.ForcedInform = def.ForcedInform
			item.ChangeApplies = def.ChangeApplies
			item.Category = def.Category
			item.IsList = def.IsList
			item.Constraints = def.Constraints
		} else {
			item.Type = string(p.ParameterType)
			item.Writable = p.Writable
		}

		items = append(items, item)
	}
	return items
}

func buildObjectSchema(validator *datamodel.ParameterValidator, params []model.DeviceParameter, pathPrefix string) []ObjectSchemaItem {
	// Discover multi-instance objects from actual parameter paths.
	objectInstances := make(map[string]map[int]bool) // objectPrefix -> set of instance numbers

	for _, p := range params {
		refs := datamodel.ExtractInstanceNumbers(p.ParameterPath)
		// Build the object path prefix for each instance ref.
		path := p.ParameterPath
		for _, ref := range refs {
			// Find the position of "ref.Segment.ref.Instance." in the path.
			search := fmt.Sprintf("%s.%d.", ref.Segment, ref.Instance)
			idx := strings.Index(path, search)
			if idx >= 0 {
				objPrefix := path[:idx+len(ref.Segment)+1]
				if _, ok := objectInstances[objPrefix]; !ok {
					objectInstances[objPrefix] = make(map[int]bool)
				}
				objectInstances[objPrefix][ref.Instance] = true
			}
		}
	}

	var items []ObjectSchemaItem
	for objPrefix, instSet := range objectInstances {
		if pathPrefix != "" && !strings.HasPrefix(objPrefix, pathPrefix) && !strings.HasPrefix(pathPrefix, objPrefix) {
			continue
		}

		instances := make([]int, 0, len(instSet))
		for inst := range instSet {
			instances = append(instances, inst)
		}
		sort.Ints(instances)

		item := ObjectSchemaItem{
			Path:             objPrefix,
			CurrentInstances: instances,
		}

		if obj := validator.LookupObject(objPrefix); obj != nil {
			item.Access = obj.Access
			item.MaxInstances = obj.MaxInstances
			item.MinInstances = datamodel.GetMinInstances(*obj)
			item.IsList = obj.IsList
			item.CanAdd = obj.Access == "READ_WRITE" && (obj.MaxInstances == 0 || len(instances) < obj.MaxInstances)
			item.CanDeleteAny = obj.Access == "READ_WRITE" && len(instances) > item.MinInstances
		}

		items = append(items, item)
	}

	return items
}

func enrichTreeWithModel(nodes []*ParameterTreeNode, v *datamodel.ParameterValidator) {
	for _, node := range nodes {
		if node.IsLeaf {
			if def := v.LookupParam(node.FullPath); def != nil {
				node.Description = def.Description
				node.ChangeApplies = def.ChangeApplies
				node.DefaultValue = def.DefaultValue
				node.Constraints = def.Constraints
			}
		} else {
			objPath := node.FullPath + "."
			if obj := v.LookupObject(objPath); obj != nil {
				node.MultiInstance = datamodel.ContainsPlaceholder(obj.Name) || obj.MaxInstances > 0
				node.MaxInstances = obj.MaxInstances
				node.MinInstances = datamodel.GetMinInstances(*obj)
				node.InstanceCount = len(node.Children)
				node.CanAdd = obj.Access == "READ_WRITE" &&
					(obj.MaxInstances == 0 || node.InstanceCount < obj.MaxInstances)
				node.CanDelete = obj.Access == "READ_WRITE" &&
					node.InstanceCount > node.MinInstances
			}
		}
		enrichTreeWithModel(node.Children, v)
	}
}

// AddObjectRequest defines the request body for adding a multi-instance object.
type AddObjectRequest struct {
	ObjectPath string `json:"object_path" binding:"required"`
}

// AddObject handles POST /api/v1/devices/:id/objects/add.
// Queues an AddObject RPC for the device.
func (h *ParameterTreeHandler) AddObject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req AddObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
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

	// Validate using data model.
	if h.dmRegistry != nil {
		dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
		if dm != nil {
			validator, _ := datamodel.NewParameterValidator(dm)
			if validator != nil {
				currentCount, _ := h.countInstances(c.Request.Context(), id, req.ObjectPath)
				if ve := validator.ValidateAddObject(req.ObjectPath, currentCount); ve != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "add object validation failed",
						"details": ve,
					})
					return
				}
			}
		}
	}

	addParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "AddObject",
		Params:     addParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("add-object-%s", uuid.New().String()[:8]),
	}

	if h.deviceService.GetCommandQueue() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("command queue not configured"))
		return
	}

	if err := h.deviceService.GetCommandQueue().Push(c.Request.Context(), dev.SerialNumber, cmd); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "add object command queued"})
}

// DeleteObjectRequest defines the request body for deleting a multi-instance object.
type DeleteObjectRequest struct {
	ObjectPath string `json:"object_path" binding:"required"`
}

// DeleteObject handles POST /api/v1/devices/:id/objects/delete.
// Queues a DeleteObject RPC for the device.
func (h *ParameterTreeHandler) DeleteObject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req DeleteObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
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

	// Extract parent object path for validation.
	parentPath := extractParentObjectPath(req.ObjectPath)

	// Validate using data model.
	if h.dmRegistry != nil && parentPath != "" {
		dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
		if dm != nil {
			validator, _ := datamodel.NewParameterValidator(dm)
			if validator != nil {
				currentCount, _ := h.countInstances(c.Request.Context(), id, parentPath)
				if ve := validator.ValidateDeleteObject(parentPath, currentCount); ve != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "delete object validation failed",
						"details": ve,
					})
					return
				}
			}
		}
	}

	delParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     "DeleteObject",
		Params:     delParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("del-object-%s", uuid.New().String()[:8]),
	}

	if h.deviceService.GetCommandQueue() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("command queue not configured"))
		return
	}

	if err := h.deviceService.GetCommandQueue().Push(c.Request.Context(), dev.SerialNumber, cmd); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "delete object command queued"})
}

// countInstances counts the number of distinct instances under an object path prefix.
func (h *ParameterTreeHandler) countInstances(ctx context.Context, deviceID uuid.UUID, objectPrefix string) (int, []int) {
	params, _ := h.paramRepo.GetByPathPrefix(ctx, deviceID, objectPrefix)
	instanceSet := make(map[int]bool)
	for _, p := range params {
		suffix := strings.TrimPrefix(p.ParameterPath, objectPrefix)
		parts := strings.SplitN(suffix, ".", 2)
		if len(parts) > 0 {
			if num, err := strconv.Atoi(parts[0]); err == nil {
				instanceSet[num] = true
			}
		}
	}
	instances := make([]int, 0, len(instanceSet))
	for num := range instanceSet {
		instances = append(instances, num)
	}
	sort.Ints(instances)
	return len(instances), instances
}

// extractParentObjectPath extracts the parent object path from an instance path.
// "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.3." -> "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList."
func extractParentObjectPath(instancePath string) string {
	path := strings.TrimSuffix(instancePath, ".")
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}
	// Check if the last part is a number (instance number).
	if _, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
		return strings.Join(parts[:len(parts)-1], ".") + "."
	}
	return ""
}
