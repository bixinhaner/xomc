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
	"github.com/omcgo/omcgo/internal/config/datamodel"
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
// T-0098 P2-07：双栈期 dmRegistry 与 paramRegistry 共存。当 paramRegistryEnabled
// 为 true 且 productRegistry/paramRegistry 均注入、且基站 ProductClass 能命中
// product 时，校验走 MappingValidator；否则降级到既有 dmRegistry 路径。
//
// 切换原则与 P2-02 verify §6 / 设计 §1.7 一致：feature flag 不写入 Registry 自身，
// 由各消费者按需读取——本 handler 通过 paramRegistryEnabled 显式控制。
type ParameterTreeHandler struct {
	deviceService        *DeviceService
	paramRepo            DeviceParameterRepository
	dmRegistry           *datamodel.DataModelRegistry
	paramRegistry        *parammodel.Registry
	productRegistry      *product.Registry
	paramRegistryEnabled bool
	logger               *zap.Logger
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

// WithParamRegistry 启用 T-0098 P2-07 dual-stack 模式。
//
// 调用方典型用法（provider/router.go）：
//
//	h := device.NewParameterTreeHandler(svc, repo, dmReg, logger).
//	    WithParamRegistry(c.ParamRegistry, c.ProductRegistry, c.Cfg.ParamRegistry.UseNew)
//
// enabled=false 或 paramRegistry/productRegistry 为 nil 时，handler 退化到
// 既有 dmRegistry 路径（与未调用本方法等价）。
func (h *ParameterTreeHandler) WithParamRegistry(paramReg *parammodel.Registry, prodReg *product.Registry, enabled bool) *ParameterTreeHandler {
	h.paramRegistry = paramReg
	h.productRegistry = prodReg
	h.paramRegistryEnabled = enabled && paramReg != nil && prodReg != nil
	return h
}

// resolveMappingValidator 在 dual-stack 启用时尝试构造 MappingValidator。
//
// 返回 nil 时调用方需走旧 dmRegistry 路径。任意一步失败均静默 fallthrough，
// 调试可通过 zap debug 字段排查。
func (h *ParameterTreeHandler) resolveMappingValidator(ctx context.Context, dev *model.Device) *parammodel.MappingValidator {
	if !h.paramRegistryEnabled || h.paramRegistry == nil || h.productRegistry == nil || dev == nil {
		return nil
	}
	if dev.ProductClass == "" {
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

// mappingValidationToLegacy 把 MappingValidationError 转成 datamodel.ValidationError，
// 保持响应 JSON 形态稳定（path/rule/message）。Code → Rule 字段名映射。
func mappingValidationToLegacy(ve *parammodel.MappingValidationError) datamodel.ValidationError {
	return datamodel.ValidationError{
		Path:    ve.Path,
		Rule:    ve.Code,
		Message: ve.Message,
	}
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
		devices.POST("/:id/parameters/sync", h.TriggerSync)
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
		response.OK(c, gin.H{"items": params, "total": len(params)})
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

	// objects_only=true 时只返回对象/文件夹节点，去掉所有叶子节点。
	// 用于左侧树面板，将 ~5925 节点缩减为 ~200 个文件夹节点。
	if c.DefaultQuery("objects_only", "false") == "true" {
		tree = stripLeafNodes(tree)
	}

	response.OK(c, gin.H{"tree": tree, "total": len(params)})
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

	response.OK(c, gin.H{"items": matched, "total": len(matched)})
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

	// Model-based validation — T-0098 P2-07：先尝试新 MappingValidator，
	// 落空再降级到 dmRegistry。两条路径只走其一。
	var rebootRequired bool
	if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
		var validationErrors []datamodel.ValidationError
		for _, item := range req.Parameters {
			if ve := mv.ValidateValue(item.Path, item.Value); ve != nil {
				validationErrors = append(validationErrors, mappingValidationToLegacy(ve))
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
	} else if h.dmRegistry != nil {
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
					response.FailWithData(c, http.StatusBadRequest,
						"parameter validation failed",
						gin.H{"validation_errors": validationErrors})
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

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
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
		response.OKWithMsg(c, nil, "no parameters to sync")
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

	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	if _, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "GetParameterValues",
		Params:     gpvParams,
		Priority:   5,
		CommandKey: fmt.Sprintf("manual-sync-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
	}); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
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
// Triggers an Upload RPC with filetype=11 (Configuration File) to sync device config.
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

	// 检查任务服务是否可用
	if h.deviceService.GetTaskService() == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	// 创建 Upload RPC 命令，filetype=11 (Configuration File / Data Model)
	// URL 将由 ACS 的 Upload handler 根据 upload server 配置自动填充
	uploadParams, _ := json.Marshal(map[string]interface{}{
		"file_type":       "11", // Configuration File / Data Model
		"url":             "",   // 由 ACS 根据 upload server 配置自动填充
		"username":        "",
		"password":        "",
		"delay_seconds":   0,
		"no_more_requests": 1, // 这是最后一个请求
	})

	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

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
// 同时过滤掉数据模型模板占位符（如 {i}），这些是多实例对象的模板路径，不应展示。
func stripLeafNodes(nodes []*ParameterTreeNode) []*ParameterTreeNode {
	var result []*ParameterTreeNode
	for _, node := range nodes {
		if node.IsLeaf {
			continue
		}
		// 跳过数据模型模板占位符节点（如 {i}、{i+1} 等）
		if strings.Contains(node.Name, "{") {
			continue
		}
		// 递归处理子节点
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
	ParameterPath  string                 `json:"parameter_path"`
	ParameterValue string                 `json:"parameter_value"`
	ParameterType  string                 `json:"parameter_type"`
	Writable       bool                   `json:"writable"`
	LastUpdatedAt  string                 `json:"last_updated_at"`
	Description    string                 `json:"description,omitempty"`
	DefaultValue   string                 `json:"default_value,omitempty"`
	ChangeApplies  string                 `json:"change_applies,omitempty"`
	Constraints    *datamodel.Constraints `json:"constraints,omitempty"`
}

// SubObjectSummary 子对象摘要，包含名称和后代参数数量。
type SubObjectSummary struct {
	Name       string `json:"name"`
	FullPath   string `json:"full_path"`
	ChildCount int    `json:"child_count"`
}

// GetDirectChildren handles GET /api/v1/devices/:id/parameters/children.
// 返回指定 path_prefix 下的直接叶子参数（分页）和直接子对象列表。
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

	// 1. 获取直接叶子参数（分页）
	leaves, total, err := h.paramRepo.GetDirectChildLeaves(c.Request.Context(), id, pathPrefix, pageSize, offset)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// 2. 计算直接子对象：获取该前缀下所有更深层级的参数，提取第一级子路径段
	allDescendants, err := h.paramRepo.GetByPathPrefix(c.Request.Context(), id, pathPrefix)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	subObjectCounts := make(map[string]int) // segment name -> 后代参数计数
	for _, p := range allDescendants {
		suffix := strings.TrimPrefix(p.ParameterPath, pathPrefix)
		if suffix == "" {
			continue
		}
		// 如果 suffix 不包含 "."，是直接叶子（已通过 SQL 查询获取），跳过
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
	// 按名称排序
	sort.Slice(subObjects, func(i, j int) bool {
		return subObjects[i].Name < subObjects[j].Name
	})

	// 3. 构建富化叶子参数（含数据模型元数据）
	items := make([]ChildParameterItem, 0, len(leaves))
	var validator *datamodel.ParameterValidator
	if h.dmRegistry != nil {
		dev, devErr := h.deviceService.GetDevice(c.Request.Context(), id)
		if devErr == nil && dev != nil {
			dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
			if dm != nil {
				validator, _ = datamodel.NewParameterValidator(dm)
			}
		}
	}

	for _, p := range leaves {
		item := ChildParameterItem{
			ParameterPath:  p.ParameterPath,
			ParameterValue: p.ParameterValue,
			ParameterType:  string(p.ParameterType),
			Writable:       p.Writable,
			LastUpdatedAt:  p.LastUpdatedAt.Format(time.RFC3339),
		}
		if validator != nil {
			if def := validator.LookupParam(p.ParameterPath); def != nil {
				if def.Writable {
					item.Writable = true
				}
				if def.Type != "" {
					item.ParameterType = def.Type
				}
				item.Description = def.Description
				item.DefaultValue = def.DefaultValue
				item.ChangeApplies = def.ChangeApplies
				item.Constraints = def.Constraints
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

	response.OK(c, gin.H{
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
				node.Writable = def.Writable
				node.Type = def.Type
				node.Description = def.Description
				node.ChangeApplies = def.ChangeApplies
				node.DefaultValue = def.DefaultValue
				node.Constraints = def.Constraints
			}
		} else {
			// 判断是否为多实例容器：子节点中有数字命名的实例（如 1, 2, 3）。
			// 不能依赖 LookupObject 返回的模板定义，因为实例节点（如 Interface.1）
			// 也会匹配到模板（如 Interface.{12}），导致误标记。
			instanceCount := 0
			for _, child := range node.Children {
				if !child.IsLeaf && isNumericName(child.Name) {
					instanceCount++
				}
			}
			if instanceCount > 0 {
				node.MultiInstance = true
				node.InstanceCount = instanceCount
				// 通过第一个实例子节点反查数据模型模板定义，获取约束信息。
				for _, child := range node.Children {
					if !child.IsLeaf && isNumericName(child.Name) {
						templatePath := child.FullPath + "."
						if obj := v.LookupObject(templatePath); obj != nil {
							node.MaxInstances = obj.MaxInstances
							node.MinInstances = datamodel.GetMinInstances(*obj)
							node.CanAdd = obj.Access == "READ_WRITE" &&
								(obj.MaxInstances == 0 || instanceCount < obj.MaxInstances)
							node.CanDelete = obj.Access == "READ_WRITE" &&
								instanceCount > node.MinInstances
						}
						break
					}
				}
			}
		}
		enrichTreeWithModel(node.Children, v)
	}
}

// isNumericName 判断节点名是否为纯数字（多实例对象的实例编号）。
func isNumericName(name string) bool {
	_, err := strconv.Atoi(name)
	return err == nil
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

	// Validate using data model — T-0098 P2-07：MappingValidator 优先，dmRegistry 兜底。
	if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
		currentCount, _ := h.countInstances(c.Request.Context(), id, req.ObjectPath)
		if ve := mv.ValidateAddObject(req.ObjectPath, currentCount); ve != nil {
			response.FailWithData(c, http.StatusBadRequest,
				"add object validation failed", gin.H{"details": ve})
			return
		}
	} else if h.dmRegistry != nil {
		dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
		if dm != nil {
			validator, _ := datamodel.NewParameterValidator(dm)
			if validator != nil {
				currentCount, _ := h.countInstances(c.Request.Context(), id, req.ObjectPath)
				if ve := validator.ValidateAddObject(req.ObjectPath, currentCount); ve != nil {
					response.FailWithData(c, http.StatusBadRequest,
						"add object validation failed", gin.H{"details": ve})
					return
				}
			}
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

	// Validate using data model — T-0098 P2-07：MappingValidator 优先，dmRegistry 兜底。
	if parentPath != "" {
		if mv := h.resolveMappingValidator(c.Request.Context(), dev); mv != nil {
			currentCount, _ := h.countInstances(c.Request.Context(), id, parentPath)
			if ve := mv.ValidateDeleteObject(parentPath, currentCount); ve != nil {
				response.FailWithData(c, http.StatusBadRequest,
					"delete object validation failed", gin.H{"details": ve})
				return
			}
		} else if h.dmRegistry != nil {
			dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
			if dm != nil {
				validator, _ := datamodel.NewParameterValidator(dm)
				if validator != nil {
					currentCount, _ := h.countInstances(c.Request.Context(), id, parentPath)
					if ve := validator.ValidateDeleteObject(parentPath, currentCount); ve != nil {
						response.FailWithData(c, http.StatusBadRequest,
							"delete object validation failed", gin.H{"details": ve})
						return
					}
				}
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
