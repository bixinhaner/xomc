package mml

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/response"
)

// ============================================================
// admin_handler.go — T-0123-P0 catalog 管理 HTTP handler 层
//
// 端点（§M.4.1，mml_params 下线后剩余）：
//   POST   /admin/groups
//   PATCH  /admin/groups/:id
//   DELETE /admin/groups/:id
//   POST   /admin/commands
//   PATCH  /admin/commands/:id
//   DELETE /admin/commands/:id
//   POST   /admin/commands/:cid/sub-fields
//   PATCH  /admin/commands/:cid/sub-fields/:sid
//   DELETE /admin/commands/:cid/sub-fields/:sid
//
// RBAC：路由组上层中间件（Casbin）通过 api_endpoints + role_api_permissions 自动鉴权（§M.4.2）。
// 错误翻译：
//   - ErrCatalogProtected            → 403 Forbidden
//   - IsErrNotFound                  → 404 Not Found
//   - ErrGroupNotEmpty               → 409 Conflict
//   - ErrGroupParamVersionNotFound   → 422 Unprocessable Entity（param_version FK 不存在）
//   - bind 错误                       → 400 Bad Request
//   - 其余                            → 500 Internal Server Error
// ============================================================

// commandLookup 是 AdminService.Update/DeleteCommand 所需的命令读取函数签名。
// 由 handler 通过既有 CommandRepository.GetByID 绑入；保持 service 层与 repo 解耦。
type commandLookup func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)

// AdminHandler 提供 catalog 管理的 HTTP 入口。
type AdminHandler struct {
	service       *AdminService
	commandLookup commandLookup
	logger        *zap.Logger
}

// NewAdminHandler 构造 AdminHandler。
// commandReader 任意实现 GetByID 的对象（既有 CommandRepository 满足）。
func NewAdminHandler(service *AdminService, commandReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
}, logger *zap.Logger) *AdminHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AdminHandler{
		service:       service,
		commandLookup: commandReader.GetByID,
		logger:        logger.Named("mml-admin-handler"),
	}
}

// RegisterRoutes registers all admin endpoints under the given router group.
// 父 group 应为 /api/v1 + RBAC middleware（端点级 Casbin 鉴权）。
//
// T-Mml-Admin 扩展：在原有 9 个写端点之上新增 admin 读路径（GET），让"系统管理 →
// MML 配置"页面不必复用 console group_tree（chapter 过滤）作为读源，避免数据视角不一致。
func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/mml/admin")

	// standard_params 下拉（T-Mml-Admin 用户规则 #3：path 必须从下拉选）
	admin.GET("/standard-params", h.ListStandardParams)
	admin.GET("/standard-params/:id", h.GetStandardParam)

	groups := admin.Group("/groups")
	groups.GET("", h.ListGroups)
	groups.POST("", h.CreateGroup)
	groups.PATCH("/:id", h.UpdateGroup)
	groups.DELETE("/:id", h.DeleteGroup)

	commands := admin.Group("/commands")
	commands.GET("", h.ListCommands)
	commands.GET("/:id", h.GetCommand)
	commands.POST("", h.CreateCommand)
	commands.PATCH("/:id", h.UpdateCommand)
	commands.DELETE("/:id", h.DeleteCommand)

	// gin tree 不允许同 prefix 下命名参数同位置不同名 — 上方 commands.PATCH("/:id",...)
	// 已把首段命名为 :id，这里子资源必须沿用 :id 而非 :cid（否则启动 panic）。
	// handler 内 parseUUIDParam 同步改 "id"。
	commands.GET("/:id/sub-fields", h.ListSubFields)
	commands.POST("/:id/sub-fields", h.CreateSubField)
	// T-Mml-Admin 用户规则 #4：批量按 path 创建 sub_field，后端按 standard_params
	// 自动派生 mml_code / label。前端在 path 多选完成后一次性提交，免去逐字段表单的体力活。
	commands.POST("/:id/sub-fields/batch", h.BatchCreateSubFields)
	commands.PATCH("/:id/sub-fields/:sid", h.UpdateSubField)
	commands.DELETE("/:id/sub-fields/:sid", h.DeleteSubField)
}

// ============================================================
// Group handlers
// ============================================================

// CreateGroup POST /api/v1/mml/admin/groups
func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	g, err := h.service.CreateGroup(c.Request.Context(), req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, g)
}

// UpdateGroup PATCH /api/v1/mml/admin/groups/:id
func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	g, err := h.service.UpdateGroup(c.Request.Context(), id, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, g)
}

// DeleteGroup DELETE /api/v1/mml/admin/groups/:id
func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteGroup(c.Request.Context(), id); err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, nil)
}

// ============================================================
// Command handlers
// ============================================================

// CreateCommand POST /api/v1/mml/admin/commands
func (h *AdminHandler) CreateCommand(c *gin.Context) {
	var req CreateCommandReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	cmd, err := h.service.CreateCommand(c.Request.Context(), req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, cmd)
}

// UpdateCommand PATCH /api/v1/mml/admin/commands/:id
func (h *AdminHandler) UpdateCommand(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateCommandReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	cmd, err := h.service.UpdateCommand(c.Request.Context(), h.commandLookup, id, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, cmd)
}

// DeleteCommand DELETE /api/v1/mml/admin/commands/:id
func (h *AdminHandler) DeleteCommand(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteCommand(c.Request.Context(), h.commandLookup, id); err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, nil)
}

// ============================================================
// SubField handlers
// ============================================================

// CreateSubField POST /api/v1/mml/admin/commands/:id/sub-fields
func (h *AdminHandler) CreateSubField(c *gin.Context) {
	cid, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req CreateSubFieldReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sf, err := h.service.CreateSubField(c.Request.Context(), cid, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, sf)
}

// UpdateSubField PATCH /api/v1/mml/admin/commands/:id/sub-fields/:sid
func (h *AdminHandler) UpdateSubField(c *gin.Context) {
	sid, ok := h.parseUUIDParam(c, "sid")
	if !ok {
		return
	}
	var req UpdateSubFieldReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sf, err := h.service.UpdateSubField(c.Request.Context(), sid, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, sf)
}

// DeleteSubField DELETE /api/v1/mml/admin/commands/:id/sub-fields/:sid
func (h *AdminHandler) DeleteSubField(c *gin.Context) {
	sid, ok := h.parseUUIDParam(c, "sid")
	if !ok {
		return
	}
	if err := h.service.DeleteSubField(c.Request.Context(), sid); err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, nil)
}

// ============================================================
// Helpers
// ============================================================

// parseUUIDParam 解析 path param 为 UUID；失败 400。
func (h *AdminHandler) parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	raw := c.Param(key)
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid uuid in path param "+key)
		return uuid.Nil, false
	}
	return id, true
}

// _ = (导出占位) admin handler 已注册 16 个端点（9 写 + 6 读 + 1 batch）。

// respondAdminError 把 admin service / repo 的错误翻译为 HTTP 状态。
func (h *AdminHandler) respondAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCatalogProtected):
		response.Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrAdminReaderNotConfigured):
		// 装配缺失 → 503，避免暴露 nil 指针错误细节。
		response.Fail(c, http.StatusServiceUnavailable, err.Error())
	case IsErrNotFound(err):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrGroupNotEmpty):
		response.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrCommandNameDuplicated):
		response.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrGroupParamVersionNotFound):
		// param_version 外键引用不存在（FK violation 已在 service 层翻译）→ 422。
		response.Fail(c, http.StatusUnprocessableEntity, err.Error())
	default:
		h.logger.Error("admin handler internal error", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

// ============================================================
// T-Mml-Admin 新增 GET / batch 端点
// ============================================================

// ListStandardParams GET /admin/standard-params?q=&entry_type=&page=&page_size=&sort_by=&sort_dir=
// 返 standard_params 全量字段（含 description / range），前端 path 下拉 + autofill 用。
func (h *AdminHandler) ListStandardParams(c *gin.Context) {
	var filter StandardParamFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}
	resp, err := h.service.ListStandardParams(c.Request.Context(), filter)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, resp)
}

// GetStandardParam GET /admin/standard-params/:id
func (h *AdminHandler) GetStandardParam(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	sp, err := h.service.GetStandardParam(c.Request.Context(), id)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, sp)
}

// ListGroups GET /admin/groups?source=&param_version=&q=
// admin 视角全集 group（与 console group_tree 区别：不过滤 chapter:%）。
func (h *AdminHandler) ListGroups(c *gin.Context) {
	var req ListGroupsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}
	out, err := h.service.ListGroups(c.Request.Context(), req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, gin.H{"items": out})
}

// ListCommands GET /admin/commands?group_id=&source=&category=&q=&page=&page_size=
func (h *AdminHandler) ListCommands(c *gin.Context) {
	var req ListCommandsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}
	// group_id 是 UUID，单独解析（form tag 不支持 *uuid.UUID 自动绑定）。
	if g := c.Query("group_id"); g != "" {
		gid, err := uuid.Parse(g)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid group_id")
			return
		}
		req.GroupID = &gid
	}
	out, err := h.service.ListCommands(c.Request.Context(), req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, out)
}

// GetCommand GET /admin/commands/:id — 单条命令详情（含完整字段，便于编辑表单 prefill）。
func (h *AdminHandler) GetCommand(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	cmd, err := h.commandLookup(c.Request.Context(), id)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, cmd)
}

// ListSubFields GET /admin/commands/:id/sub-fields
// admin 视角 sub_field 列表（含 is_supported=false 行；与 console 端 GET
// /commands/:id/sub-fields 区别在不过滤）。
func (h *AdminHandler) ListSubFields(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	out, err := h.service.ListSubFields(c.Request.Context(), id)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, gin.H{"items": out})
}

// BatchCreateSubFields POST /admin/commands/:id/sub-fields/batch
// body: { "standard_path_ids": ["uuid", ...] }
// 后端按 standard_params 元数据自动派生 mml_code / label / sort_order。
func (h *AdminHandler) BatchCreateSubFields(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req BatchCreateSubFieldsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	out, err := h.service.BatchCreateSubFields(c.Request.Context(), id, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{"items": out})
}
