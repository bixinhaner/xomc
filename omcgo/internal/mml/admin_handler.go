package mml

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/response"
)

// ============================================================
// admin_handler.go — T-0123-P0 catalog 管理 HTTP handler 层
//
// 13 端点（§M.4.1）：
//   POST   /admin/groups
//   PATCH  /admin/groups/:id
//   DELETE /admin/groups/:id
//   POST   /admin/commands
//   PATCH  /admin/commands/:id
//   DELETE /admin/commands/:id
//   POST   /admin/commands/:cid/sub-fields
//   PATCH  /admin/commands/:cid/sub-fields/:sid
//   DELETE /admin/commands/:cid/sub-fields/:sid
//   GET    /admin/params
//   POST   /admin/params
//   PATCH  /admin/params/:id
//   DELETE /admin/params/:id
//
// RBAC：路由组上层中间件（Casbin）通过 api_endpoints + role_api_permissions 自动鉴权（§M.4.2）。
// 错误翻译：
//   - ErrCatalogProtected   → 403 Forbidden
//   - IsErrNotFound         → 404 Not Found
//   - ErrParamInUse         → 409 Conflict
//   - ErrGroupNotEmpty      → 409 Conflict
//   - bind 错误              → 400 Bad Request
//   - 其余                   → 500 Internal Server Error
// ============================================================

// commandLookup 是 AdminService.Update/DeleteCommand 所需的命令读取函数签名。
// 由 handler 通过既有 CommandRepository.GetByID 绑入；保持 service 层与 repo 解耦。
type commandLookup func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)

// AdminHandler 提供 catalog 管理的 HTTP 入口。
type AdminHandler struct {
	service       *AdminService
	xmlImport     *XMLImportService
	commandLookup commandLookup
	logger        *zap.Logger
}

// NewAdminHandler 构造 AdminHandler。
// commandReader 任意实现 GetByID 的对象（既有 CommandRepository 满足）。
// xmlImport 可为 nil（T-0132 import 端点 fallback 返 500 — 兼容历史 caller 未传入场景）。
func NewAdminHandler(service *AdminService, commandReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
}, xmlImport *XMLImportService, logger *zap.Logger) *AdminHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AdminHandler{
		service:       service,
		xmlImport:     xmlImport,
		commandLookup: commandReader.GetByID,
		logger:        logger.Named("mml-admin-handler"),
	}
}

// RegisterRoutes registers the 13 admin endpoints under the given router group.
// 父 group 应为 /api/v1 + RBAC middleware（端点级 Casbin 鉴权）。
func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/mml/admin")

	groups := admin.Group("/groups")
	groups.POST("", h.CreateGroup)
	groups.PATCH("/:id", h.UpdateGroup)
	groups.DELETE("/:id", h.DeleteGroup)

	commands := admin.Group("/commands")
	commands.POST("", h.CreateCommand)
	commands.PATCH("/:id", h.UpdateCommand)
	commands.DELETE("/:id", h.DeleteCommand)

	// gin tree 不允许同 prefix 下命名参数同位置不同名 — 上方 commands.PATCH("/:id",...)
	// 已把首段命名为 :id，这里子资源必须沿用 :id 而非 :cid（否则启动 panic）。
	// handler 内 parseUUIDParam 同步改 "id"。
	commands.POST("/:id/sub-fields", h.CreateSubField)
	commands.PATCH("/:id/sub-fields/:sid", h.UpdateSubField)
	commands.DELETE("/:id/sub-fields/:sid", h.DeleteSubField)

	params := admin.Group("/params")
	params.GET("", h.ListParams)
	params.GET("/:id/references", h.ListParamReferences)
	params.POST("", h.CreateParam)
	params.PATCH("/:id", h.UpdateParam)
	params.DELETE("/:id", h.DeleteParam)

	// T-0132 admin Tab 4 XML 导入 — dry-run preview + 写表 apply 两端点。
	imp := admin.Group("/import")
	imp.POST("/preview", h.ImportPreview)
	imp.POST("/apply", h.ImportApply)
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
// Param handlers
// ============================================================

// ListParamsResp 是 GET /admin/params 的响应体。
type ListParamsResp struct {
	Items    []Param `json:"items"`
	Total    int64   `json:"total"`
	PageNum  int     `json:"page_num"`
	PageSize int     `json:"page_size"`
}

// ListParams GET /api/v1/mml/admin/params
//   ?search=&access_type=&is_object=&source=&param_version=&page_num=&page_size=
func (h *AdminHandler) ListParams(c *gin.Context) {
	f := AdminParamFilter{
		PageNum:  parsePositiveInt(c.Query("page_num"), 1),
		PageSize: parsePositiveInt(c.Query("page_size"), 50),
	}
	if v := c.Query("search"); v != "" {
		f.Search = &v
	}
	if v := c.Query("access_type"); v != "" {
		f.AccessType = &v
	}
	if v := c.Query("is_object"); v != "" {
		b := v == "true" || v == "1"
		f.IsObject = &b
	}
	if v := c.Query("source"); v != "" {
		f.Source = &v
	}
	if v := c.Query("param_version"); v != "" {
		f.ParamVersion = &v
	}
	items, total, err := h.service.ListParams(c.Request.Context(), f)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, ListParamsResp{
		Items:    items,
		Total:    total,
		PageNum:  f.PageNum,
		PageSize: f.PageSize,
	})
}

// ListParamReferencesResp 是 GET /admin/params/:id/references 的响应体（T-0131）。
type ListParamReferencesResp struct {
	Items []ParamReference `json:"items"`
}

// ListParamReferences GET /api/v1/mml/admin/params/:id/references
//
//	反向查：返回引用该 param 的命令列表（admin Tab 3 抽屉用，T-0131）。
//	不分页（单 param 的引用集通常 ≤ 50，全量返回足够）。
func (h *AdminHandler) ListParamReferences(c *gin.Context) {
	idStr := c.Param("id")
	paramID, err := uuid.Parse(idStr)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid param id: "+idStr)
		return
	}
	refs, err := h.service.ListParamReferences(c.Request.Context(), paramID)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, ListParamReferencesResp{Items: refs})
}

// CreateParam POST /api/v1/mml/admin/params
func (h *AdminHandler) CreateParam(c *gin.Context) {
	var req CreateParamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p, err := h.service.CreateParam(c.Request.Context(), req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, p)
}

// UpdateParam PATCH /api/v1/mml/admin/params/:id
func (h *AdminHandler) UpdateParam(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateParamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p, err := h.service.UpdateParam(c.Request.Context(), id, req)
	if err != nil {
		h.respondAdminError(c, err)
		return
	}
	response.OK(c, p)
}

// DeleteParam DELETE /api/v1/mml/admin/params/:id
func (h *AdminHandler) DeleteParam(c *gin.Context) {
	id, ok := h.parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteParam(c.Request.Context(), id); err != nil {
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

// respondAdminError 把 admin service / repo 的错误翻译为 HTTP 状态。
func (h *AdminHandler) respondAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCatalogProtected):
		response.Fail(c, http.StatusForbidden, err.Error())
	case IsErrNotFound(err):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrParamInUse) || errors.Is(err, ErrGroupNotEmpty):
		response.Fail(c, http.StatusConflict, err.Error())
	default:
		h.logger.Error("admin handler internal error", zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

func parsePositiveInt(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// ============================================================
// T-0132 XML 导入 handlers
// ============================================================

// maxImportUploadBytes 单次 XML 上传体积上限（防 OOM / DoS）。
// standard-model.xml ~500KB；预留 4x 余量 = 2MB。
const maxImportUploadBytes = 2 << 20 // 2 MiB

// ImportPreview POST /api/v1/mml/admin/import/preview
//
// multipart/form-data:
//   - file: standard-model.xml
//   - version_code: form value (e.g. "STANDARD")
//
// 返 dry-run JSON：summary 三桶（add/modify/skipped）+ 前 200 行 diff 详情 + truncated 标志。
// 不写表，可重复调用。
func (h *AdminHandler) ImportPreview(c *gin.Context) {
	if h.xmlImport == nil {
		response.Fail(c, http.StatusInternalServerError, "xml import service not wired")
		return
	}
	versionCode := c.PostForm("version_code")
	if versionCode == "" {
		response.Fail(c, http.StatusBadRequest, "version_code is required")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "file is required: "+err.Error())
		return
	}
	if fileHeader.Size > maxImportUploadBytes {
		response.Fail(c, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file size %d exceeds limit %d", fileHeader.Size, maxImportUploadBytes))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "open uploaded file: "+err.Error())
		return
	}
	defer f.Close()

	resp, err := h.xmlImport.Preview(c.Request.Context(), f, versionCode)
	if err != nil {
		h.logger.Warn("xml import preview failed",
			zap.String("version_code", versionCode),
			zap.Error(err))
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, resp)
}

// ImportApply POST /api/v1/mml/admin/import/apply
//
// multipart/form-data:
//   - file: standard-model.xml（重传 — 防 server-side session）
//   - version_code: form value
//
// 执行批量 UPSERT；catalog_protected=true 的 standard 行被更新；catalog_protected=false 的
// admin 改过的行被守护跳过（Q2=C 决议）。
func (h *AdminHandler) ImportApply(c *gin.Context) {
	if h.xmlImport == nil {
		response.Fail(c, http.StatusInternalServerError, "xml import service not wired")
		return
	}
	versionCode := c.PostForm("version_code")
	if versionCode == "" {
		response.Fail(c, http.StatusBadRequest, "version_code is required")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "file is required: "+err.Error())
		return
	}
	if fileHeader.Size > maxImportUploadBytes {
		response.Fail(c, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file size %d exceeds limit %d", fileHeader.Size, maxImportUploadBytes))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "open uploaded file: "+err.Error())
		return
	}
	defer f.Close()

	resp, err := h.xmlImport.Apply(c.Request.Context(), f, versionCode)
	if err != nil {
		h.logger.Error("xml import apply failed",
			zap.String("version_code", versionCode),
			zap.Error(err))
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OKWithStatus(c, http.StatusOK, resp)
}
