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
	"github.com/omcgo/omcgo/internal/config/parammodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

// ParameterTreeHandler provides REST API handlers for the device parameter tree.
//
// T-0098 P5-01：旧 dmRegistry / datamodel.ParameterValidator 路径已删除。
// 现仅走 parammodel.MappingValidator（来自 paramRegistry + productRegistry 的
// 双向映射）。元属性范围因此收敛到 ParamMapping 的 5 项（access / data_type /
// change_applies / min_value / max_value）；description / default_value /
// max_instances 等字段在 mapping 表外，本 handler 不再注入。
type ParameterTreeHandler struct {
	deviceService   *DeviceService
	paramRepo       DeviceParameterRepository
	paramRegistry   *parammodel.Registry
	productRegistry *product.Registry
	logger          *zap.Logger
}

// NewParameterTreeHandler creates a new parameter tree handler.
func NewParameterTreeHandler(
	deviceService *DeviceService,
	paramRepo DeviceParameterRepository,
	paramReg *parammodel.Registry,
	prodReg *product.Registry,
	logger *zap.Logger,
) *ParameterTreeHandler {
	return &ParameterTreeHandler{
		deviceService:   deviceService,
		paramRepo:       paramRepo,
		paramRegistry:   paramReg,
		productRegistry: prodReg,
		logger:          logger,
	}
}

// resolveMappingValidator 尝试构造当前设备的 MappingValidator。
// 任意一步失败返回 nil；调用方需做空检查后跳过校验/富化逻辑。
func (h *ParameterTreeHandler) resolveMappingValidator(ctx context.Context, dev *model.Device) *parammodel.MappingValidator {
	if h.paramRegistry == nil || h.productRegistry == nil || dev == nil || dev.ProductClass == "" {
		return nil
	}
	match, err := h.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil
	}
	set, err := h.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil
	}
	return parammodel.NewMappingValidator(set)
}

// Constraints 是参数取值范围的简化表示，对齐前端字段（路径上 JSON key=constraints）。
// 仅承载数值上下限；字符串长度/枚举/正则不在 ParamMapping 范畴内。
type Constraints struct {
	MinValue *int64 `json:"min_value,omitempty"`
	MaxValue *int64 `json:"max_value,omitempty"`
}

// constraintsFromMapping 把 ParamMapping 的 MinValue/MaxValue 收集为 Constraints；
// 二者全空返回 nil（避免返回无意义对象）。
func constraintsFromMapping(m *parammodel.ParamMapping) *Constraints {
	if m == nil || (m.MinValue == nil && m.MaxValue == nil) {
		return nil
	}
	return &Constraints{MinValue: m.MinValue, MaxValue: m.MaxValue}
}

// RegisterRoutes registers parameter tree routes.
func (h *ParameterTreeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	devices := rg.Group("/devices")
	{
		devices.GET("/:id/parameters/tree", h.GetParameterTree)
		devices.GET("/:id/parameters/children", h.GetDirectChildren)
		devices.GET("/:id/parameters/search", h.SearchParameters)
		devices.GET("/:id/parameters/schema", h.GetParameterSchema)
		devices.PUT("/:id/parameters", h.SetParameterValues)
		// T-0126: 旧 /parameters/sync (Path A) 已下线，替换为 /sync-params (Path B + reason="manual")
		// 由 device_handler.go 注册；本处仅保留 discover/sync-status（discovery flow 与 Path B 全量同步并存）
		devices.POST("/:id/parameters/discover", h.TriggerDiscover)
		devices.GET("/:id/parameters/sync-status", h.GetSyncStatus)
		devices.POST("/:id/objects/add", h.AddObject)
		devices.POST("/:id/objects/delete", h.DeleteObject)
		// 配置文件同步 - 创建 filetype=11 的 Upload RPC 任务
		devices.POST("/:id/config-file/sync", h.SyncConfigFile)
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
	// Model metadata (enriched from MappingValidator)
	Description   string       `json:"description,omitempty"`
	MultiInstance bool         `json:"multi_instance,omitempty"`
	MaxInstances  int          `json:"max_instances,omitempty"`
	MinInstances  int          `json:"min_instances,omitempty"`
	InstanceCount int          `json:"instance_count,omitempty"`
	CanAdd        bool         `json:"can_add,omitempty"`
	CanDelete     bool         `json:"can_delete,omitempty"`
	ChangeApplies string       `json:"change_applies,omitempty"`
	DefaultValue  string       `json:"default_value,omitempty"`
	Constraints   *Constraints `json:"constraints,omitempty"`
}

// ParameterSchemaItem represents a parameter with schema and current value info.
type ParameterSchemaItem struct {
	Path          string       `json:"path"`
	Type          string       `json:"type"`
	Writable      bool         `json:"writable"`
	Description   string       `json:"description,omitempty"`
	DefaultValue  string       `json:"default_value,omitempty"`
	Notify        string       `json:"notify,omitempty"`
	ForcedInform  bool         `json:"forced_inform,omitempty"`
	ChangeApplies string       `json:"change_applies,omitempty"`
	Category      string       `json:"category,omitempty"`
	IsList        bool         `json:"is_list,omitempty"`
	Constraints   *Constraints `json:"constraints,omitempty"`
	CurrentValue  *string      `json:"current_value"`
	LastSyncedAt  *time.Time   `json:"last_synced_at,omitempty"`
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
		response.OK(c, gin.H{"items": params, "total": len(params)})
		return
	}

	tree := buildTree(params)

	// Enrich tree with mapping metadata if available.
	dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
	if devErr == nil && dev != nil {
		if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
			enrichTreeWithMapping(tree, mv)
		}
	}

	// objects_only=true 时只返回对象/文件夹节点，去掉所有叶子节点。
	if c.DefaultQuery("objects_only", "false") == "true" {
		tree = stripLeafNodes(tree)
	}

	response.OK(c, gin.H{"tree": tree, "total": len(params)})
}

// SearchParameters handles GET /api/v1/devices/:id/parameter-tree/search.
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

	response.OK(c, gin.H{"items": matched, "total": len(matched)})
}

// SetParameterValuesRequest defines the request body for setting parameter values.
type SetParameterValuesRequest struct {
	Parameters []ParameterValueItem `json:"parameters" binding:"required,min=1"`
}

// SetParameterValues handles PUT /api/v1/devices/:id/parameter-tree.
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

	// Mapping-based validation（T-0098 P5-01：已无 dmRegistry 兜底，未命中即跳过）。
	var rebootRequired bool
	if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
		var validationErrors []*parammodel.MappingValidationError
		for _, item := range req.Parameters {
			if ve := mv.ValidateValue(item.Path, item.Value); ve != nil {
				validationErrors = append(validationErrors, ve)
			}
		}
		if len(validationErrors) > 0 {
			response.FailWithData(c, http.StatusBadRequest,
				"parameter validation failed",
				gin.H{"validation_errors": validationErrors})
			return
		}
		for _, item := range req.Parameters {
			if def := mv.LookupParam(item.Path); def != nil {
				if def.ChangeApplies == "RebootRequired" || def.ChangeApplies == "NotifyRequired" {
					rebootRequired = true
					break
				}
			}
		}
	}

	if err := h.deviceService.SetParameters(c.Request.Context(), id, req.Parameters); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"message":         "set parameter values command queued",
		"parameters":      len(req.Parameters),
		"reboot_required": rebootRequired,
	})
}

// T-0126: TriggerSync (POST /api/v1/devices/:id/parameters/sync) 已下线。
// 替换为 device_handler.go 的 SyncDeviceParams (POST /devices/:id/sync-params) 走 Path B。
// 旧实现是 Path A —— 查 device_parameters 现有路径列表 + 单 batch GPV，
// 不走 Translator / 不写 last_param_sync_at / 不发差异日志。

// TriggerDiscover handles POST /api/v1/devices/:id/parameters/discover.
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

	gpnParams, _ := json.Marshal(map[string]interface{}{
		"path":       "Device.",
		"next_level": false,
	})

	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	if _, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "GetParameterNames",
		Params:     gpnParams,
		Priority:   1,
		CommandKey: fmt.Sprintf("manual-discover-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
	}); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{"message": "parameter discovery command queued"})
}

// GetSyncStatus handles GET /api/v1/devices/:id/parameters/sync-status.
func (h *ParameterTreeHandler) GetSyncStatus(c *gin.Context) {
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

	var pendingCommands int64
	if taskSvc := h.deviceService.GetTaskService(); taskSvc != nil {
		dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
		if devErr == nil && dev != nil {
			pendingCommands, _ = taskSvc.GetQueueLength(c.Request.Context(), dev.SerialNumber)
		}
	}

	status := "idle"
	if pendingCommands > 0 {
		status = "syncing"
	}

	response.OK(c, gin.H{
		"status":           status,
		"total_parameters": len(params),
		"pending_commands": pendingCommands,
	})
}

// SyncConfigFile handles POST /api/v1/devices/:id/config-file/sync.
func (h *ParameterTreeHandler) SyncConfigFile(c *gin.Context) {
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

	if h.deviceService.GetTaskService() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	uploadParams, _ := json.Marshal(map[string]interface{}{
		"file_type":        "11",
		"url":              "",
		"username":         "",
		"password":         "",
		"delay_seconds":    0,
		"no_more_requests": 1,
	})

	taskSvc := h.deviceService.GetTaskService()
	created, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "Upload",
		Params:     uploadParams,
		Priority:   5,
		CommandKey: fmt.Sprintf("config-sync-%s-%d", dev.SerialNumber, time.Now().Unix()),
		Source:     task.TaskSourceAPI,
	})
	if err != nil {
		h.logger.Error("failed to queue config sync command",
			zap.String("device_id", id.String()),
			zap.String("serial_number", dev.SerialNumber),
			zap.Error(err))
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	h.logger.Info("config sync command queued",
		zap.String("device_id", id.String()),
		zap.String("serial_number", dev.SerialNumber),
		zap.String("command_id", created.ID))

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"message":    "configuration file sync command queued",
		"command_id": created.ID,
		"file_type":  "11",
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

// stripLeafNodes 递归移除树中的所有叶子节点，只保留对象/文件夹节点。
// 同时过滤掉数据模型模板占位符（如 {i}），这些是多实例对象的模板路径。
func stripLeafNodes(nodes []*ParameterTreeNode) []*ParameterTreeNode {
	var result []*ParameterTreeNode
	for _, node := range nodes {
		if node.IsLeaf {
			continue
		}
		if strings.Contains(node.Name, "{") {
			continue
		}
		node.Children = stripLeafNodes(node.Children)
		result = append(result, node)
	}
	return result
}

// DirectChildrenResponse 返回指定前缀下的直接子项（富化叶子参数 + 子对象摘要）。
type DirectChildrenResponse struct {
	Items      []ChildParameterItem `json:"items"`
	SubObjects []SubObjectSummary   `json:"sub_objects"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
}

// ChildParameterItem 富化叶子参数，包含数据模型元数据。
type ChildParameterItem struct {
	ParameterPath  string       `json:"parameter_path"`
	ParameterValue string       `json:"parameter_value"`
	ParameterType  string       `json:"parameter_type"`
	Writable       bool         `json:"writable"`
	LastUpdatedAt  string       `json:"last_updated_at"`
	Description    string       `json:"description,omitempty"`
	DefaultValue   string       `json:"default_value,omitempty"`
	ChangeApplies  string       `json:"change_applies,omitempty"`
	Constraints    *Constraints `json:"constraints,omitempty"`
}

// SubObjectSummary 子对象摘要，包含名称和后代参数数量。
type SubObjectSummary struct {
	Name       string `json:"name"`
	FullPath   string `json:"full_path"`
	ChildCount int    `json:"child_count"`
}

// GetDirectChildren handles GET /api/v1/devices/:id/parameters/children.
func (h *ParameterTreeHandler) GetDirectChildren(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	pathPrefix := c.Query("path_prefix")
	if pathPrefix == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("path_prefix is required"))
		return
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 50
	if ps := c.Query("page_size"); ps != "" {
		if parsed, parseErr := strconv.Atoi(ps); parseErr == nil && parsed > 0 && parsed <= 200 {
			pageSize = parsed
		}
	}
	offset := (page - 1) * pageSize

	leaves, total, err := h.paramRepo.GetDirectChildLeaves(c.Request.Context(), id, pathPrefix, pageSize, offset)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	allDescendants, err := h.paramRepo.GetByPathPrefix(c.Request.Context(), id, pathPrefix)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	subObjectCounts := make(map[string]int)
	for _, p := range allDescendants {
		suffix := strings.TrimPrefix(p.ParameterPath, pathPrefix)
		if suffix == "" {
			continue
		}
		dotIdx := strings.Index(suffix, ".")
		if dotIdx < 0 {
			continue
		}
		segment := suffix[:dotIdx]
		subObjectCounts[segment]++
	}

	subObjects := make([]SubObjectSummary, 0, len(subObjectCounts))
	for segment, count := range subObjectCounts {
		subObjects = append(subObjects, SubObjectSummary{
			Name:       segment,
			FullPath:   pathPrefix + segment,
			ChildCount: count,
		})
	}
	sort.Slice(subObjects, func(i, j int) bool {
		return subObjects[i].Name < subObjects[j].Name
	})

	items := make([]ChildParameterItem, 0, len(leaves))
	var mv *parammodel.MappingValidator
	dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
	if devErr == nil && dev != nil {
		mv = h.resolveMappingValidator(c.Request.Context(), dev)
	}

	for _, p := range leaves {
		item := ChildParameterItem{
			ParameterPath:  p.ParameterPath,
			ParameterValue: p.ParameterValue,
			ParameterType:  string(p.ParameterType),
			Writable:       p.Writable,
			LastUpdatedAt:  p.LastUpdatedAt.Format(time.RFC3339),
		}
		if mv != nil {
			if def := mv.LookupParam(p.ParameterPath); def != nil {
				if parammodel.IsAccessWritable(def.Access) {
					item.Writable = true
				}
				if def.DataType != "" {
					item.ParameterType = def.DataType
				}
				item.ChangeApplies = def.ChangeApplies
				item.Constraints = constraintsFromMapping(def)
			}
		}
		items = append(items, item)
	}

	response.OK(c, DirectChildrenResponse{
		Items:      items,
		SubObjects: subObjects,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	})
}

// GetParameterSchema handles GET /api/v1/devices/:id/parameters/schema.
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

	var schemaItems []ParameterSchemaItem
	var objectItems []ObjectSchemaItem

	if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
		schemaItems = mergeSchemaWithValues(mv, params, pathPrefix)
		objectItems = buildObjectSchema(mv, params, pathPrefix)
	}

	if schemaItems == nil {
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

	response.OK(c, gin.H{
		"parameters": schemaItems,
		"objects":    objectItems,
		"total":      len(schemaItems),
	})
}

// mergeSchemaWithValues 用 MappingValidator 富化参数 schema。
// MappingValidator 不承载 description / default / notify / forced_inform / category / is_list；
// 这些字段在响应里会留空（向前兼容）。
func mergeSchemaWithValues(mv *parammodel.MappingValidator, params []model.DeviceParameter, pathPrefix string) []ParameterSchemaItem {
	items := make([]ParameterSchemaItem, 0, len(params))

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

		if def := mv.LookupParam(p.ParameterPath); def != nil {
			item.Type = def.DataType
			item.Writable = parammodel.IsAccessWritable(def.Access)
			item.ChangeApplies = def.ChangeApplies
			item.Constraints = constraintsFromMapping(def)
		} else {
			item.Type = string(p.ParameterType)
			item.Writable = p.Writable
		}

		items = append(items, item)
	}
	return items
}

// buildObjectSchema 从实际参数路径推断多实例对象 + 用 MappingValidator 补 access / writable。
// MappingValidator 不存 max/min instances；本响应字段设 0，前端做空值处理。
func buildObjectSchema(mv *parammodel.MappingValidator, params []model.DeviceParameter, pathPrefix string) []ObjectSchemaItem {
	objectInstances := make(map[string]map[int]bool)

	for _, p := range params {
		refs := extractInstanceRefs(p.ParameterPath)
		path := p.ParameterPath
		for _, ref := range refs {
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

	items := make([]ObjectSchemaItem, 0, len(objectInstances))
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

		if obj := mv.LookupObject(objPrefix); obj != nil {
			item.Access = obj.Access
			canWrite := parammodel.IsAccessWritable(obj.Access)
			item.CanAdd = canWrite
			item.CanDeleteAny = canWrite && len(instances) > 0
		}

		items = append(items, item)
	}

	return items
}

// enrichTreeWithMapping 用 MappingValidator 给 ParameterTreeNode 补元属性。
// 多实例容器节点的 MaxInstances/MinInstances/CanAdd/CanDelete 因 mapping 不存这些字段，
// 退化为：仅在 access 含写权限时允许 CanAdd / CanDelete。
func enrichTreeWithMapping(nodes []*ParameterTreeNode, mv *parammodel.MappingValidator) {
	for _, node := range nodes {
		if node.IsLeaf {
			if def := mv.LookupParam(node.FullPath); def != nil {
				node.Writable = parammodel.IsAccessWritable(def.Access)
				node.Type = def.DataType
				node.ChangeApplies = def.ChangeApplies
				node.Constraints = constraintsFromMapping(def)
			}
		} else {
			instanceCount := 0
			for _, child := range node.Children {
				if !child.IsLeaf && isNumericName(child.Name) {
					instanceCount++
				}
			}
			if instanceCount > 0 {
				node.MultiInstance = true
				node.InstanceCount = instanceCount
				for _, child := range node.Children {
					if !child.IsLeaf && isNumericName(child.Name) {
						templatePath := child.FullPath + "."
						if obj := mv.LookupObject(templatePath); obj != nil {
							canWrite := parammodel.IsAccessWritable(obj.Access)
							node.CanAdd = canWrite
							node.CanDelete = canWrite && instanceCount > 0
						}
						break
					}
				}
			}
		}
		enrichTreeWithMapping(node.Children, mv)
	}
}

// isNumericName 判断节点名是否为纯数字（多实例对象的实例编号）。
func isNumericName(name string) bool {
	_, err := strconv.Atoi(name)
	return err == nil
}

// instanceRef 表示一条多实例引用（e.g. "Foo.7" → segment="Foo" instance=7）。
type instanceRef struct {
	Segment  string
	Instance int
}

// extractInstanceRefs 从参数路径里抽出 ".<segment>.<num>." 形态的多实例引用。
// 等价于旧 datamodel.ExtractInstanceNumbers。
func extractInstanceRefs(path string) []instanceRef {
	parts := strings.Split(path, ".")
	var refs []instanceRef
	for i := 1; i < len(parts); i++ {
		if isNumericName(parts[i]) && parts[i-1] != "" {
			n, _ := strconv.Atoi(parts[i])
			refs = append(refs, instanceRef{Segment: parts[i-1], Instance: n})
		}
	}
	return refs
}

// AddObjectRequest defines the request body for adding a multi-instance object.
type AddObjectRequest struct {
	ObjectPath string `json:"object_path" binding:"required"`
}

// AddObject handles POST /api/v1/devices/:id/objects/add.
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

	if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
		currentCount, _ := h.countInstances(c.Request.Context(), id, req.ObjectPath)
		if ve := mv.ValidateAddObject(req.ObjectPath, currentCount); ve != nil {
			response.FailWithData(c, http.StatusBadRequest,
				"add object validation failed", gin.H{"details": ve})
			return
		}
	}

	addParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	if _, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "AddObject",
		Params:     addParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("add-object-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
	}); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{"message": "add object command queued"})
}

// DeleteObjectRequest defines the request body for deleting a multi-instance object.
type DeleteObjectRequest struct {
	ObjectPath string `json:"object_path" binding:"required"`
}

// DeleteObject handles POST /api/v1/devices/:id/objects/delete.
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

	parentPath := extractParentObjectPath(req.ObjectPath)

	if parentPath != "" {
		if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
			currentCount, _ := h.countInstances(c.Request.Context(), id, parentPath)
			if ve := mv.ValidateDeleteObject(parentPath, currentCount); ve != nil {
				response.FailWithData(c, http.StatusBadRequest,
					"delete object validation failed", gin.H{"details": ve})
				return
			}
		}
	}

	delParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	if _, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "DeleteObject",
		Params:     delParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("del-object-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
	}); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{"message": "delete object command queued"})
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
func extractParentObjectPath(instancePath string) string {
	path := strings.TrimSuffix(instancePath, ".")
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}
	if _, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
		return strings.Join(parts[:len(parts)-1], ".") + "."
	}
	return ""
}
