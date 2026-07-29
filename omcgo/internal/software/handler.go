package software

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides REST API endpoints for software/firmware management.
//
// #18 分层收敛：Handler 只依赖 Service，不再持有/直连 Repository。所有读写
// 都经 SoftwareService（handler → service → repository），便于在 Service 层
// 集中挂权限检查 / 缓存策略。
type Handler struct {
	service  *SoftwareService
	resolver *authz.Resolver // #59 Problem 4：从 gin ctx 解析调用者可见设备组
	logger   *zap.Logger
}

// NewHandler creates a new software Handler.
func NewHandler(service *SoftwareService, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("software-handler"),
	}
}

// SetPermissionService 注入数据权限解析器（#59 Problem 4 / #64 统一强制层），使升级 /
// 回退创建端点按调用者可见设备组对请求体里的 DeviceIDs 做逐设备归属校验。未注入时
// h.resolver 为 nil，FromContext 走 nil-safe 退化（不强制），与 device / alarm 一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// authorizeDeviceIDs 解析调用者可见设备组并对 deviceIDs 逐个做归属校验（#59 Problem 4）。
// 返回 false 表示已 abort（403/500），调用方应立即 return。h.resolver 为 nil 时
// FromContext 返回 (nil, true)，AuthorizeDevicesAccess 据三态契约对 nil 放行（dev/test 退化）。
func (h *Handler) authorizeDeviceIDs(c *gin.Context, deviceIDs []uuid.UUID) bool {
	groups, ok := h.resolver.FromContext(c)
	if !ok {
		return false
	}
	if err := h.service.AuthorizeDevicesAccess(c.Request.Context(), groups, deviceIDs...); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return false
	}
	return true
}

// auditUpgradeDenied 记一条「越权被拒」的失败审计（#59 Problem 4）。在 authorizeDeviceIDs
// 返回 false 后调用，确保跨租户写尝试留痕。action / resourceType 由调用方传入区分升级 / 回退。
func (h *Handler) auditUpgradeDenied(c *gin.Context, action, resourceType string) {
	entry := admin.AuditContextFromGin(c)
	entry.Action = action
	entry.ResourceType = resourceType
	entry.Success = false
	entry.ErrorMessage = "forbidden: device out of caller's visible groups"
	audit.LogAsync(entry)
}

// RegisterRoutes registers firmware and upgrade task routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	fw := rg.Group("/firmware")
	fw.GET("", h.ListFirmware)
	fw.POST("", h.UploadFirmware)
	fw.GET("/:id", h.GetFirmware)
	fw.DELETE("/:id", h.DeleteFirmware)
	fw.PUT("/:id/recommend", h.ToggleFirmwareRecommend)
	fw.GET("/:id/download", h.DownloadFirmware)
	fw.PUT("/:id", h.UpdateFirmware)

	upgrade := rg.Group("/upgrade-tasks")
	upgrade.GET("", h.ListUpgradeTasks)
	upgrade.GET("/:id", h.GetUpgradeTask)
	upgrade.POST("", h.CreateUpgradeTask)
	upgrade.PUT("/:id/suspend", h.SuspendUpgradeTask)
	upgrade.PUT("/:id/resume", h.ResumeUpgradeTask)
	upgrade.PUT("/:id/terminate", h.TerminateUpgradeTask)
	upgrade.DELETE("/:id", h.DeleteUpgradeTask)
	upgrade.POST("/:id/retry", h.RetryUpgradeTask)
	upgrade.POST("/rollback", h.CreateRollback)
	upgrade.GET("/:id/tasks", h.ListSubTasks)
	// Canary stage transitions (T-0018 / R-101)
	upgrade.POST("/:id/advance", h.AdvanceCanary)
	upgrade.POST("/:id/pause-canary", h.PauseCanary)
	upgrade.POST("/:id/resume-canary", h.ResumeCanary)
	upgrade.POST("/:id/abort-canary", h.AbortCanary)

	subTasks := rg.Group("/upgrade-sub-tasks")
	subTasks.GET("/:id", h.GetSubTask)
	subTasks.GET("", h.ListAllSubTasks)
}

func (h *Handler) ListFirmware(c *gin.Context) {
	var filter FirmwareFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListFirmware(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) UploadFirmware(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	fw := &FirmwareVersion{
		ProductClass: c.PostForm("product_class"),
		Version:      c.PostForm("version"),
		FileName:     header.Filename,
		ReleaseNotes: c.PostForm("release_notes"),
		Uploader:     c.PostForm("uploader"),
		Manufacturer: c.PostForm("manufacturer"),
		Description:  c.PostForm("description"),
	}
	if c.PostForm("recommend") == "true" {
		fw.Recommend = true
	}
	if ft := c.PostForm("file_type"); ft != "" {
		if n, err := strconv.Atoi(ft); err == nil {
			fw.FileType = FileType(n)
		}
	}
	// #492：上传按产品名（前端传 product_id）。产品归属以 product_id 为权威。
	if pid := c.PostForm("product_id"); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			fw.ProductID = &id
		}
	}
	// #638：多产品上传。前端提交 product_ids 为逗号分隔的 UUID 串（FormData
	// 交互最友好）；service 层会根据 ProductIDs[0] 同步 ProductID 作为兼容主产品。
	if pids := c.PostForm("product_ids"); pids != "" {
		fw.ProductIDs = parseProductIDsCSV(pids)
	}

	if fw.Version == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(8002, "version is required", commonerrors.ErrInvalidInput))
		return
	}

	if err := h.service.UploadFirmware(c.Request.Context(), fw, file, header.Size); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, fw)
}

func (h *Handler) GetFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.service.GetFirmware(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, fw)
}

func (h *Handler) DeleteFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteFirmware(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"status": "deleted"})
}

func (h *Handler) ToggleFirmwareRecommend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.service.ToggleFirmwareRecommend(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, fw)
}

func (h *Handler) DownloadFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Fetch firmware metadata for filename
	fw, err := h.service.GetFirmware(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	obj, err := h.service.DownloadFirmware(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	defer obj.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+fw.FileName+"\"")
	c.Header("Content-Type", "application/octet-stream")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj)
}

func (h *Handler) UpdateFirmware(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	fw, err := h.service.GetFirmware(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	var req struct {
		ProductID    string   `json:"product_id"`
		ProductIDs   []string `json:"product_ids"`
		ProductClass string   `json:"product_class"`
		Version      string   `json:"version"`
		Recommend    *bool    `json:"recommend"`
		Description  string   `json:"description"`
		ReleaseNotes string   `json:"release_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// #492：允许改产品归属（产品名 → 前端传 product_id）。
	if req.ProductID != "" {
		if id, err := uuid.Parse(req.ProductID); err == nil {
			fw.ProductID = &id
		}
	}
	// #638：修改适用多产品。前端限制“同时传 product_ids 才走多产品路径”，不传则保留原有数组，
	// 避免老前端只提交 product_id 时误清多产品。service 层会同步 ProductID = ProductIDs[0]。
	if req.ProductIDs != nil {
		fw.ProductIDs = parseProductIDsList(req.ProductIDs)
	}
	if req.ProductClass != "" {
		fw.ProductClass = req.ProductClass
	}
	if req.Version != "" {
		fw.Version = req.Version
	}
	if req.Recommend != nil {
		fw.Recommend = *req.Recommend
	}
	fw.Description = req.Description
	fw.ReleaseNotes = req.ReleaseNotes

	if err := h.service.UpdateFirmwareMetadata(c.Request.Context(), fw); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, fw)
}

func (h *Handler) ListUpgradeTasks(c *gin.Context) {
	var filter UpgradeTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListUpgradeTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) GetUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.service.GetUpgradeTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, task)
}

func (h *Handler) CreateUpgradeTask(c *gin.Context) {
	var req BatchUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.CreateUser = currentUsername(c)

	// #59 Problem 4：逐设备归属校验——低权用户不得升级其可见设备组之外的设备
	// （跨租户写）。任一 DeviceID 越界即整批 403。在调 service 之前拦截，避免越权
	// 任务落库。审计记一条失败的 ActionUpgrade。
	if !h.authorizeDeviceIDs(c, req.DeviceIDs) {
		h.auditUpgradeDenied(c, audit.ActionUpgrade, audit.ResourceUpgradeTask)
		return
	}

	task, err := h.service.BatchUpgrade(c.Request.Context(), req)

	// Cross-module audit: ActionUpgrade / category 3 of 5 (W3.G.2).
	entry := admin.AuditContextFromGin(c)
	entry.Action = audit.ActionUpgrade
	entry.ResourceType = audit.ResourceUpgradeTask
	entry.Success = err == nil
	if task != nil {
		entry.ResourceID = task.ID.String()
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	audit.LogAsync(entry)

	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

func currentUsername(c *gin.Context) string {
	if raw, ok := c.Get("username"); ok {
		if name, ok := raw.(string); ok && name != "" {
			return name
		}
	}
	return "system"
}

func (h *Handler) SuspendUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.SuspendUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "upgrade task suspended")
}

func (h *Handler) ResumeUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.ResumeUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "upgrade task resumed")
}

func (h *Handler) TerminateUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.TerminateUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "upgrade task terminated")
}

func (h *Handler) RetryUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.RetryUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "retry initiated")
}

func (h *Handler) DeleteUpgradeTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteUpgrade(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithMsg(c, nil, "upgrade task deleted")
}

func (h *Handler) CreateRollback(c *gin.Context) {
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// #59 Problem 4：逐设备归属校验——回退是破坏性写（可把设备降级到旧镜像），
	// 同样不得跨可见设备组。任一 DeviceID 越界即整批 403 + 审计。
	if !h.authorizeDeviceIDs(c, req.DeviceIDs) {
		h.auditUpgradeDenied(c, audit.ActionUpgrade, audit.ResourceUpgradeTask)
		return
	}

	task, err := h.service.RollbackDevices(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

func (h *Handler) ListSubTasks(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	var filter SubTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter.TaskID = taskID

	result, err := h.service.ListSubTasksByTaskID(c.Request.Context(), taskID, filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListAllSubTasks(c *gin.Context) {
	var filter AllSubTaskFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.ListAllSubTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) GetSubTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task, err := h.service.GetSubTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, task)
}

// ==========================================================================
// Canary stage transitions (T-0018 / R-101)
// ==========================================================================

// AdvanceCanary handles POST /upgrade-tasks/:id/advance
//
// Promotes a paused/running canary task to its next stage. 404 if task
// not found, 400 if task is not on the canary path or already terminal.
func (h *Handler) AdvanceCanary(c *gin.Context) {
	h.transitionCanary(c, h.service.AdvanceCanaryStage, "advance")
}

// PauseCanary handles POST /upgrade-tasks/:id/pause-canary
//
// Pauses stage progression. In-flight sub-tasks continue. Distinct from
// SuspendUpgradeTask (PUT /upgrade-tasks/:id/suspend) which halts execution.
func (h *Handler) PauseCanary(c *gin.Context) {
	h.transitionCanary(c, h.service.PauseCanaryStage, "pause")
}

// ResumeCanary handles POST /upgrade-tasks/:id/resume-canary
func (h *Handler) ResumeCanary(c *gin.Context) {
	h.transitionCanary(c, h.service.ResumeCanaryStage, "resume")
}

// AbortCanary handles POST /upgrade-tasks/:id/abort-canary
//
// Terminates remaining stages. Already-running sub-tasks are not killed —
// operators must call SuspendUpgradeTask in addition if they want to halt
// in-flight executions.
func (h *Handler) AbortCanary(c *gin.Context) {
	h.transitionCanary(c, h.service.AbortCanary, "abort")
}

func (h *Handler) transitionCanary(c *gin.Context, fn func(ctx context.Context, taskID uuid.UUID) error, op string) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid task id: %w", err))
		return
	}
	if err := fn(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"task_id": id.String(), "operation": op, "result": "ok"})
}

// parseProductIDsCSV 把表单字段里的逗号分隔 UUID 串解析为去重后的 uuid 切片，跳过
// 解析失败项（避免 1 个错误 UUID 导致整批 400）。空串返回 nil（语义=不传）。
// 用于 multipart/form-data 上传场景（#638 升级文件多产品复选）。
func parseProductIDsCSV(s string) []uuid.UUID {
	parts := strings.Split(s, ",")
	out := make([]uuid.UUID, 0, len(parts))
	seen := make(map[uuid.UUID]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseProductIDsList 是 parseProductIDsCSV 的 JSON 字符串数组版本（用于 PUT 接口）。
// 语义与 CSV 版一致：去重、跳过解析失败项、空 → nil。
func parseProductIDsList(in []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	seen := make(map[uuid.UUID]struct{}, len(in))
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
