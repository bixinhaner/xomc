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
	"github.com/omcgo/omcgo/internal/admin"
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
	permissionSvc   VisibleGroupsResolver
	logger          *zap.Logger
}

type syncGPVSummaryReader interface {
	LatestSyncGPVSummaryByDevice(ctx context.Context, deviceSN string) (*task.SyncGPVSummary, error)
}

type syncGPVOpenCounter interface {
	CountOpenSyncGPVByDevice(ctx context.Context, deviceSN string) (int64, error)
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

// SetPermissionService wires the data permission service for route-level injection.
// Parameter tree endpoints currently resolve a single device by ID/SN and keep
// behavior unchanged when the service is not consulted.
func (h *ParameterTreeHandler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permissionSvc = ps
}

// resolveMappingValidator 尝试构造当前设备的 MappingValidator。
// 任意一步失败返回 nil；调用方需做空检查后跳过校验/富化逻辑。
func (h *ParameterTreeHandler) resolveMappingValidator(ctx context.Context, dev *model.Device) *parammodel.MappingValidator {
	writeModel := h.resolveParameterWriteModel(ctx, dev)
	if writeModel == nil {
		return nil
	}
	return writeModel.validator
}

// resolveDefaultMappingValidator returns the product-bound default param model only.
// Device-discovered mappings are intentionally excluded for UI tree/list rendering.
func (h *ParameterTreeHandler) resolveDefaultMappingValidator(ctx context.Context, dev *model.Device) *parammodel.MappingValidator {
	if h.paramRegistry == nil || h.productRegistry == nil || dev == nil || dev.ProductClass == "" {
		return nil
	}
	match, err := h.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil || match.Product.ParamModelID == nil {
		return nil
	}
	set, err := h.paramRegistry.GetByParamModel(ctx, *match.Product.ParamModelID)
	if err != nil || set == nil {
		return nil
	}
	return parammodel.NewMappingValidator(set)
}

// resolveDisplayMappingValidator 优先返回设备当前命中的 discovered 映射；
// 若设备尚未学习到 discovered，则退回产品默认映射。
// UI 展示链路需要既看到默认标准化路径，也要尽可能匹配设备真实上报路径。
func (h *ParameterTreeHandler) resolveDisplayMappingValidator(ctx context.Context, dev *model.Device) *parammodel.MappingValidator {
	if mv := h.resolveMappingValidator(ctx, dev); mv != nil {
		return mv
	}
	return h.resolveDefaultMappingValidator(ctx, dev)
}

// Constraints 是参数取值范围的简化表示，对齐前端字段（路径上 JSON key=constraints）。
// 承载数值上下限 + 枚举 (T-0158)；字符串长度复用 min_value/max_value 由前端按 type 解释；
// 正则 / 显式 minLength/maxLength 暂不支持（§11 L-04）。
type Constraints struct {
	MinValue   *int64   `json:"min_value,omitempty"`
	MaxValue   *int64   `json:"max_value,omitempty"`
	EnumValues []string `json:"enum_values,omitempty"` // T-0158: 下发设备的实际值列表
	EnumLabels []string `json:"enum_labels,omitempty"` // T-0158: UI 显示标签，与 EnumValues 一一对应
	// T-0159: 交叉镜像目标 standardPath（含 {i}）。前端 quicksettings 渲染该字段时，
	// 任一端 onChange 应同步写另一端（典型场景 TDD 上下行带宽必须相等）。
	MirrorWith *string `json:"mirror_with,omitempty"`
}

// constraintsFromMapping 把 ParamMapping 的 MinValue/MaxValue/EnumValues/EnumLabels/MirrorWith
// 收集为 Constraints；全空返回 nil（避免无意义对象）。
func constraintsFromMapping(m *parammodel.ParamMapping) *Constraints {
	if m == nil {
		return nil
	}
	hasRange := m.MinValue != nil || m.MaxValue != nil
	hasEnum := m.EnumValues != nil && *m.EnumValues != ""
	hasMirror := m.MirrorWith != nil && *m.MirrorWith != ""
	if !hasRange && !hasEnum && !hasMirror {
		return nil
	}
	c := &Constraints{MinValue: m.MinValue, MaxValue: m.MaxValue}
	if hasEnum {
		c.EnumValues = splitCSV(*m.EnumValues)
		if m.EnumLabels != nil && *m.EnumLabels != "" {
			c.EnumLabels = splitCSV(*m.EnumLabels)
		}
	}
	if hasMirror {
		c.MirrorWith = m.MirrorWith
	}
	return c
}

// splitCSV 把 "a,b,c" 切成 []string，剔除首尾空白；空串返回 nil。
func splitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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

	if !authorizeDeviceAccess(c, h.deviceService, h.permissionSvc, id) {
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

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	displayMV := h.resolveDisplayMappingValidator(c.Request.Context(), dev)
	displayParams := buildDisplayParameters(params, displayMV)

	// Check if client wants flat or tree format.
	format := c.DefaultQuery("format", "tree")
	if format == "flat" {
		response.OK(c, gin.H{"items": displayParams, "total": len(displayParams)})
		return
	}

	tree := buildTree(displayParams)

	// Enrich tree with mapping metadata if available.
	if displayMV != nil {
		enrichTreeWithMapping(tree, displayMV)
	}

	// objects_only=true 时只返回对象/文件夹节点，去掉所有叶子节点。
	if c.DefaultQuery("objects_only", "false") == "true" {
		tree = stripLeafNodes(tree)
	}

	response.OK(c, gin.H{"tree": tree, "total": len(displayParams)})
}

// SearchParameters handles GET /api/v1/devices/:id/parameter-tree/search.
func (h *ParameterTreeHandler) SearchParameters(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if !authorizeDeviceAccess(c, h.deviceService, h.permissionSvc, id) {
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

	// T-0148:移除 device_parameters.writable 预检 — 与 XML schema 双真值源冲突。
	// 原检查:GetByDevice 拉本设备所有路径 → 拒掉 writable=false 的路径。
	// 冲突:device_parameters.writable 由 CPE 早期 GetParameterNames 写入,可能反映运行态锁(如 cell 工作时 PCI 锁定),
	//      而 XML 字典(MappingValidator)说 READ_WRITE → 前端"快速设置"展示可编辑,后端却 400 阻断,UX 撕裂。
	// 新策略:① XML schema (MappingValidator.access) 作唯一真值源 ② CPE 运行态拒绝由 SetParameterValuesResponse
	//        Status≠0 在 T-0146 状态机里显示"应答失败"。

	// Mapping-based validation（T-0098 P5-01：已无 dmRegistry 兜底，未命中即跳过）。
	var rebootRequired bool
	var rebootTarget int
	if writeModel := h.resolveParameterWriteModel(c.Request.Context(), dev); writeModel != nil {
		validation := validateParameterWrites(writeModel, req.Parameters)
		if len(validation.Errors) > 0 {
			response.FailWithData(c, http.StatusBadRequest,
				"parameter validation failed",
				gin.H{"validation_errors": validation.Errors})
			return
		}
		rebootRequired = validation.RebootRequired
		rebootTarget = validation.RebootTarget
	}

	taskID, err := h.deviceService.SetParameters(c.Request.Context(), id, req.Parameters, admin.UserIDStringFromCtx(c))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"message":         "set parameter values command queued",
		"parameters":      len(req.Parameters),
		"reboot_required": rebootRequired,
		"reboot_target":   rebootTarget,
		"task_id":         taskID, // T-0146:前端用 useTaskStatus 轮询真实 CPE 应答状态
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

	if !authorizeDeviceAccess(c, h.deviceService, h.permissionSvc, id) {
		return
	}

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	dev, _ := h.deviceService.GetDevice(c.Request.Context(), id)
	var pendingCommands int64
	if taskSvc := h.deviceService.GetTaskService(); taskSvc != nil && dev != nil {
		if counter, ok := taskSvc.(syncGPVOpenCounter); ok {
			count, err := counter.CountOpenSyncGPVByDevice(c.Request.Context(), dev.SerialNumber)
			if err == nil {
				pendingCommands = count
			} else {
				if h.logger != nil {
					h.logger.Warn("count open sync-gpv tasks failed, falling back to queue length",
						zap.String("device_sn", dev.SerialNumber),
						zap.Error(err))
				}
				pendingCommands, _ = taskSvc.GetQueueLength(c.Request.Context(), dev.SerialNumber)
			}
		} else {
			pendingCommands, _ = taskSvc.GetQueueLength(c.Request.Context(), dev.SerialNumber)
		}
	}

	// 二态: syncing(队列有 pending task) / idle(其余)。"上次同步时间"由
	// last_param_sync_at 字段单独承载,前端据此渲染"上次同步:X 时间前"持久信息;
	// 不再用 completed 过渡态(原因:同步状态属于设备级共享状态,跨会话/跨用户都
	// 应该看到一致的"进行中/上次同步时间",而非"刚完成"这种会话级反馈)。
	status := "idle"
	if pendingCommands > 0 {
		status = "syncing"
	}

	resp := gin.H{
		"status":           status,
		"total_parameters": len(params),
		"pending_commands": pendingCommands,
	}
	if taskSvc := h.deviceService.GetTaskService(); taskSvc != nil && dev != nil {
		if summaryReader, ok := taskSvc.(syncGPVSummaryReader); ok {
			if summary, err := summaryReader.LatestSyncGPVSummaryByDevice(c.Request.Context(), dev.SerialNumber); err == nil && summary != nil {
				lastSync := gin.H{
					"source_id":             summary.SourceID,
					"task_count":            summary.TaskCount,
					"successful_commands":   summary.SuccessfulCommands,
					"failed_commands":       summary.FailedCommands,
					"requested_path_count":  summary.RequestedPathCount,
					"successful_path_count": summary.SuccessfulPathCount,
					"failed_path_count":     summary.FailedPathCount,
					"failed_paths":          summary.FailedPaths,
					"first_created_at":      summary.FirstCreatedAt,
					"wall_clock_seconds":    summary.WallClockSeconds,
				}
				if summary.LastCompletedAt != nil {
					lastSync["last_completed_at"] = summary.LastCompletedAt
				}
				resp["last_sync_gpv"] = lastSync
			}
		}
	}
	// migration 000146 互斥语义:last_param_sync_at / last_param_sync_failed_at 任一非空,
	// 前端据此判定"上次成功"还是"上次失败"(失败时一并展示 error 文案)。
	if dev != nil && dev.LastParamSyncAt != nil {
		resp["last_param_sync_at"] = dev.LastParamSyncAt
	}
	if dev != nil && dev.LastParamSyncFailedAt != nil {
		resp["last_param_sync_failed_at"] = dev.LastParamSyncFailedAt
		if dev.LastParamSyncError != nil {
			resp["last_param_sync_error"] = *dev.LastParamSyncError
		}
	}
	response.OK(c, resp)
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

func buildDisplayParameters(params []model.DeviceParameter, mv *parammodel.MappingValidator) []model.DeviceParameter {
	if mv == nil {
		return params
	}

	actualByPath := make(map[string]model.DeviceParameter, len(params))
	actualByTemplate := make(map[string][]model.DeviceParameter, len(params))
	for _, p := range params {
		actualByPath[p.ParameterPath] = p
		normalized := normalizeDisplayPath(p.ParameterPath)
		actualByTemplate[normalized] = append(actualByTemplate[normalized], p)
	}

	display := make([]model.DeviceParameter, 0, len(mv.Mappings()))
	seen := make(map[string]struct{}, len(mv.Mappings()))

	for _, mapping := range mv.Mappings() {
		if mapping.EntryType != "parameter" {
			continue
		}

		paths := []string{mapping.PrivatePath}
		if strings.Contains(mapping.PrivatePath, "{i}") || strings.Contains(mapping.StandardPath, "{i}") {
			matched := actualByTemplate[normalizeDisplayPath(mapping.PrivatePath)]
			if len(matched) == 0 && mapping.StandardPath != "" {
				matched = actualByTemplate[normalizeDisplayPath(mapping.StandardPath)]
			}
			if len(matched) == 0 {
				continue
			}
			paths = paths[:0]
			for _, actual := range matched {
				paths = append(paths, actual.ParameterPath)
			}
		}

		for _, path := range paths {
			if _, ok := seen[path]; ok {
				continue
			}
			if actual, ok := actualByPath[path]; ok {
				display = append(display, actual)
				seen[path] = struct{}{}
				continue
			}
			if mapping.StandardPath != "" {
				if actual, ok := actualByPath[mapping.StandardPath]; ok {
					display = append(display, actual)
					seen[path] = struct{}{}
					continue
				}
			}
			display = append(display, model.DeviceParameter{
				ParameterPath: path,
				ParameterType: model.ParameterType(mapping.DataType),
				Writable:      parammodel.IsAccessWritable(mapping.Access),
			})
			seen[path] = struct{}{}
		}
	}

	sort.Slice(display, func(i, j int) bool {
		return display[i].ParameterPath < display[j].ParameterPath
	})
	return display
}

func normalizeDisplayPath(path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if isNumericName(part) {
			parts[i] = "{i}"
		}
	}
	return strings.Join(parts, ".")
}

func getDirectChildLeaves(params []model.DeviceParameter, pathPrefix string, pageSize, offset int) ([]model.DeviceParameter, int) {
	if pathPrefix == "" {
		return nil, 0
	}

	filtered := make([]model.DeviceParameter, 0)
	for _, p := range params {
		if !strings.HasPrefix(p.ParameterPath, pathPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(p.ParameterPath, pathPrefix)
		if suffix == "" || strings.Contains(suffix, ".") {
			continue
		}
		filtered = append(filtered, p)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ParameterPath < filtered[j].ParameterPath
	})

	total := len(filtered)
	if offset >= total {
		return []model.DeviceParameter{}, total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return filtered[offset:end], total
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

	if !authorizeDeviceAccess(c, h.deviceService, h.permissionSvc, id) {
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

	dev, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if dev == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}

	params, err := h.paramRepo.GetByDevice(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	displayMV := h.resolveDisplayMappingValidator(c.Request.Context(), dev)
	displayParams := buildDisplayParameters(params, displayMV)

	leaves, total := getDirectChildLeaves(displayParams, pathPrefix, pageSize, offset)

	subObjectCounts := make(map[string]int)
	for _, p := range displayParams {
		if !strings.HasPrefix(p.ParameterPath, pathPrefix) {
			continue
		}
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
	mv := displayMV

	for _, p := range leaves {
		item := ChildParameterItem{
			ParameterPath:  p.ParameterPath,
			ParameterValue: p.ParameterValue,
			ParameterType:  string(p.ParameterType),
			Writable:       p.Writable,
		}
		if !p.LastUpdatedAt.IsZero() {
			item.LastUpdatedAt = p.LastUpdatedAt.Format(time.RFC3339)
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

	if !authorizeDeviceAccess(c, h.deviceService, h.permissionSvc, id) {
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

	schemaMV := h.resolveDisplayMappingValidator(c.Request.Context(), dev)
	var params []model.DeviceParameter
	if pathPrefix != "" && schemaMV == nil {
		params, err = h.paramRepo.GetByPathPrefix(c.Request.Context(), id, pathPrefix)
	} else {
		// With a mapping validator, CPE values may be persisted under standardPath while
		// quicksettings queries by privatePath. Keep the full device value set so schema
		// merge can attach aliased standard values back to the requested private paths.
		params, err = h.paramRepo.GetByDevice(c.Request.Context(), id)
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var schemaItems []ParameterSchemaItem
	var objectItems []ObjectSchemaItem

	if schemaMV != nil {
		schemaItems = mergeSchemaWithValues(schemaMV, params, pathPrefix)
		objectItems = buildObjectSchema(schemaMV, params, pathPrefix)
	}

	if schemaItems == nil {
		schemaItems = make([]ParameterSchemaItem, 0, len(params))
		for _, p := range params {
			if pathPrefix != "" && !strings.HasPrefix(p.ParameterPath, pathPrefix) {
				continue
			}
			val := p.ParameterValue
			item := ParameterSchemaItem{
				Path:         p.ParameterPath,
				Type:         string(p.ParameterType),
				Writable:     p.Writable,
				CurrentValue: &val,
			}
			if !p.LastUpdatedAt.IsZero() {
				item.LastSyncedAt = &p.LastUpdatedAt
			}
			schemaItems = append(schemaItems, item)
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
	seen := make(map[string]struct{}, len(params))
	valuesByPath := make(map[string]model.DeviceParameter, len(params))
	for _, p := range params {
		valuesByPath[p.ParameterPath] = p
	}

	for _, p := range params {
		if pathPrefix != "" && !strings.HasPrefix(p.ParameterPath, pathPrefix) {
			continue
		}
		val := p.ParameterValue
		item := ParameterSchemaItem{
			Path:         p.ParameterPath,
			CurrentValue: &val,
		}
		if !p.LastUpdatedAt.IsZero() {
			item.LastSyncedAt = &p.LastUpdatedAt
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
		seen[item.Path] = struct{}{}
	}

	if mv == nil || pathPrefix == "" {
		return items
	}

	for _, def := range mv.Mappings() {
		if def.EntryType != "parameter" {
			continue
		}
		for _, value := range schemaValuesForPrefix(def, pathPrefix, valuesByPath) {
			if _, exists := seen[value.privatePath]; exists {
				continue
			}
			items = append(items, ParameterSchemaItem{
				Path:          value.privatePath,
				Type:          def.DataType,
				Writable:      parammodel.IsAccessWritable(def.Access),
				ChangeApplies: def.ChangeApplies,
				Constraints:   constraintsFromMapping(&def),
				CurrentValue:  value.currentValue,
				LastSyncedAt:  value.lastSyncedAt,
			})
			seen[value.privatePath] = struct{}{}
		}
	}
	return items
}

type schemaValueForPrefix struct {
	privatePath  string
	currentValue *string
	lastSyncedAt *time.Time
}

func schemaValuesForPrefix(
	def parammodel.ParamMapping,
	pathPrefix string,
	valuesByPath map[string]model.DeviceParameter,
) []schemaValueForPrefix {
	privatePath, ok := instantiatePrivatePathForPrefix(def.PrivatePath, pathPrefix)
	if ok && strings.HasPrefix(privatePath, pathPrefix) {
		return []schemaValueForPrefix{schemaValueFromPaths(privatePath, privatePath, def.StandardPath, pathPrefix, valuesByPath)}
	}

	prefixInstances := extractNumericSegments(pathPrefix)
	if countPlaceholders(def.PrivatePath) != len(prefixInstances)+1 {
		return nil
	}

	byPrivatePath := make(map[string]schemaValueForPrefix)
	for _, p := range valuesByPath {
		inst, matched := matchTemplateWithPrefixInstances(def.PrivatePath, p.ParameterPath, prefixInstances)
		if !matched {
			inst, matched = matchTemplateWithPrefixInstances(def.StandardPath, p.ParameterPath, prefixInstances)
		}
		if !matched {
			continue
		}
		privatePath, ok := instantiatePathWithInstances(def.PrivatePath, append(prefixInstances, inst))
		if !ok || !strings.HasPrefix(privatePath, pathPrefix) {
			continue
		}
		val := p.ParameterValue
		item := schemaValueForPrefix{
			privatePath:  privatePath,
			currentValue: &val,
		}
		if !p.LastUpdatedAt.IsZero() {
			item.lastSyncedAt = &p.LastUpdatedAt
		}
		byPrivatePath[privatePath] = item
	}
	if len(byPrivatePath) == 0 {
		privatePath, ok := instantiatePathWithInstances(def.PrivatePath, append(prefixInstances, "{i}"))
		if ok && strings.HasPrefix(privatePath, pathPrefix) {
			return []schemaValueForPrefix{{privatePath: privatePath}}
		}
	}

	paths := make([]string, 0, len(byPrivatePath))
	for path := range byPrivatePath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]schemaValueForPrefix, 0, len(paths))
	for _, path := range paths {
		out = append(out, byPrivatePath[path])
	}
	return out
}

func schemaValueFromPaths(
	privatePath string,
	privateValuePath string,
	standardTemplate string,
	pathPrefix string,
	valuesByPath map[string]model.DeviceParameter,
) schemaValueForPrefix {
	item := schemaValueForPrefix{privatePath: privatePath}
	if p, ok := valuesByPath[privateValuePath]; ok {
		val := p.ParameterValue
		item.currentValue = &val
		if !p.LastUpdatedAt.IsZero() {
			item.lastSyncedAt = &p.LastUpdatedAt
		}
		return item
	}
	if standardPath, ok := instantiatePrivatePathForPrefix(standardTemplate, pathPrefix); ok {
		if p, exists := valuesByPath[standardPath]; exists {
			val := p.ParameterValue
			item.currentValue = &val
			if !p.LastUpdatedAt.IsZero() {
				item.lastSyncedAt = &p.LastUpdatedAt
			}
		}
	}
	return item
}

func instantiatePrivatePathForPrefix(template, pathPrefix string) (string, bool) {
	if template == "" {
		return "", false
	}
	numbers := extractNumericSegments(pathPrefix)
	parts := strings.Split(template, ".")
	placeholderCount := 0
	for _, part := range parts {
		if part == "{i}" {
			placeholderCount++
		}
	}
	if placeholderCount == 0 {
		return template, true
	}
	if len(numbers) != placeholderCount {
		return "", false
	}
	idx := 0
	for i, part := range parts {
		if part == "{i}" {
			parts[i] = numbers[idx]
			idx++
		}
	}
	return strings.Join(parts, "."), true
}

func instantiatePathWithInstances(template string, instances []string) (string, bool) {
	if template == "" {
		return "", false
	}
	parts := strings.Split(template, ".")
	idx := 0
	for i, part := range parts {
		if part != "{i}" {
			continue
		}
		if idx >= len(instances) {
			return "", false
		}
		parts[i] = instances[idx]
		idx++
	}
	if idx != len(instances) {
		return "", false
	}
	return strings.Join(parts, "."), true
}

func matchTemplateWithPrefixInstances(template, path string, prefixInstances []string) (string, bool) {
	if template == "" || path == "" {
		return "", false
	}
	templateParts := strings.Split(strings.Trim(template, "."), ".")
	pathParts := strings.Split(strings.Trim(path, "."), ".")
	if len(templateParts) != len(pathParts) {
		return "", false
	}
	placeholderIdx := 0
	captured := ""
	for i, part := range templateParts {
		if part != "{i}" {
			if part != pathParts[i] {
				return "", false
			}
			continue
		}
		if placeholderIdx < len(prefixInstances) {
			if pathParts[i] != prefixInstances[placeholderIdx] {
				return "", false
			}
		} else {
			if captured != "" || !isNumericName(pathParts[i]) {
				return "", false
			}
			captured = pathParts[i]
		}
		placeholderIdx++
	}
	if placeholderIdx != len(prefixInstances)+1 || captured == "" {
		return "", false
	}
	return captured, true
}

func countPlaceholders(template string) int {
	count := 0
	for _, part := range strings.Split(template, ".") {
		if part == "{i}" {
			count++
		}
	}
	return count
}

func extractNumericSegments(path string) []string {
	parts := strings.Split(path, ".")
	segments := make([]string, 0, 4)
	for _, part := range parts {
		if part == "" {
			continue
		}
		if isNumericName(part) {
			segments = append(segments, part)
		}
	}
	return segments
}

// buildObjectSchema 从实际参数路径推断多实例对象 + 用 MappingValidator 补 access / writable。
// MappingValidator 不存 max/min instances；本响应字段设 0，前端做空值处理。
//
// Phantom-instance 过滤：某些 CPE 固件会对未真正配置的 cell 槽位也回伪数据，导致 UI 伪实例。
// 规则：实例下 writable 参数数=0 → phantom（覆盖 NR `CellConfig.{2,3,4}` 仅回 Tx0VswrStatus 等只读计数器）。
//
// 单标量 phantom（如 BAIBLQ `FAPService.{3..12}` 仅回 1 个 NumOfCells writable 标量）由前端
// 结合 `LTE.RAN.CA.PARAMS.NumOfCells` + 各 FAP 的 `FAPControl.LTE.InUse` 在 UI 层精确过滤，
// 不在此处做启发式 5x 落差判断，避免误伤真实但配置极少的实例。
func buildObjectSchema(mv *parammodel.MappingValidator, params []model.DeviceParameter, pathPrefix string) []ObjectSchemaItem {
	// objectWritableCount[objPrefix][instance] = 该实例下 writable 参数计数（用于全只读 phantom 过滤）
	objectWritableCount := make(map[string]map[int]int)

	for _, p := range params {
		if p.LastUpdatedAt.IsZero() {
			continue
		}
		// 判断该叶子是否 writable：mapping access 优先，回退到 DB 标记
		paramWritable := p.Writable
		if def := mv.LookupParam(p.ParameterPath); def != nil {
			paramWritable = parammodel.IsAccessWritable(def.Access)
		}

		refs := extractInstanceRefs(p.ParameterPath)
		path := p.ParameterPath
		for _, ref := range refs {
			search := fmt.Sprintf("%s.%d.", ref.Segment, ref.Instance)
			idx := strings.Index(path, search)
			if idx >= 0 {
				objPrefix := path[:idx+len(ref.Segment)+1]
				if _, ok := objectWritableCount[objPrefix]; !ok {
					objectWritableCount[objPrefix] = make(map[int]int)
				}
				if _, exists := objectWritableCount[objPrefix][ref.Instance]; !exists {
					objectWritableCount[objPrefix][ref.Instance] = 0
				}
				if paramWritable {
					objectWritableCount[objPrefix][ref.Instance]++
				}
			}
		}
	}

	items := make([]ObjectSchemaItem, 0, len(objectWritableCount))
	for objPrefix, instCounts := range objectWritableCount {
		if pathPrefix != "" && !strings.HasPrefix(objPrefix, pathPrefix) && !strings.HasPrefix(pathPrefix, objPrefix) {
			continue
		}

		// 全只读对象兜底：BSC 的 DeviceGSM.Bts.{i}.*、TR-181 只读容器等场景，
		// mapping seed 把所有叶子标 READ_ONLY，每个实例的 wcount 必然 = 0；
		// 若仍套用 "wcount==0 → phantom 过滤" 会把整个对象的真实实例全删空，
		// 前端选择器永远 0 实例（实际现网 18-06 BTS=254 的根因之一）。
		// 判定：对象内所有实例 wcount=0 → 对象是 "全只读" → 不过滤；
		// 对象内至少有一个实例 wcount>0 → 用老 phantom 规则剔除 wcount=0 的伪槽位。
		allReadOnly := true
		for _, w := range instCounts {
			if w > 0 {
				allReadOnly = false
				break
			}
		}

		instances := make([]int, 0, len(instCounts))
		for inst, wcount := range instCounts {
			// mv==nil（部分单元测试）退化为不过滤；生产路径总有 mv。
			// allReadOnly=true 时跳过 phantom 过滤（保留全只读对象的全部实例）。
			if mv != nil && !allReadOnly && wcount == 0 {
				continue
			}
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
	rebootRequired, rebootTarget := objectRebootRequirement(h.resolveMappingValidator(c.Request.Context(), dev), req.ObjectPath)

	addParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	createdTask, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "AddObject",
		Params:     addParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("add-object-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
		CreatorID:  admin.UserIDStringFromCtx(c), // T-0157 C5/C7: 消息中心 user_id
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// T-0157 C7: 返回 task_id 让前端 useAddObject 走 useDeviceTaskStatus 状态机
	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"task_id":         createdTask.ID,
		"message":         "add object command queued",
		"reboot_required": rebootRequired,
		"reboot_target":   rebootTarget,
	})
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
	rebootRequired, rebootTarget := objectRebootRequirement(h.resolveMappingValidator(c.Request.Context(), dev), parentPath)

	delParams, _ := json.Marshal(map[string]interface{}{
		"object_name": req.ObjectPath,
	})
	taskSvc := h.deviceService.GetTaskService()
	if taskSvc == nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("task service not configured"))
		return
	}

	createdTask, err := taskSvc.CreateTask(c.Request.Context(), &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     "DeleteObject",
		Params:     delParams,
		Priority:   3,
		CommandKey: fmt.Sprintf("del-object-%s", uuid.New().String()[:8]),
		Source:     task.TaskSourceAPI,
		CreatorID:  admin.UserIDStringFromCtx(c), // T-0157 C5/C7
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"task_id":         createdTask.ID,
		"message":         "delete object command queued",
		"reboot_required": rebootRequired,
		"reboot_target":   rebootTarget,
	})
}

func objectRebootRequirement(validator *parammodel.MappingValidator, path string) (bool, int) {
	if validator == nil {
		return false, 0
	}
	mapping := validator.LookupObject(path)
	if mapping == nil || (mapping.ChangeApplies != "RebootRequired" && mapping.ChangeApplies != "NotifyRequired") {
		return false, 0
	}
	return true, rebootTargetForPath(path)
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
