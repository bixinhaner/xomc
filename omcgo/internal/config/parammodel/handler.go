package parammodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// optInt64 在 JSON 解码时同时接受 number、numeric string、null、""。
//   - 字段缺省                → *optInt64 为 nil(语义:不提供)
//   - JSON `null` / `""`      → 非 nil 指针但 Valid=false(语义:显式清空)
//   - JSON number / 数字字符串 → 非 nil 指针且 Valid=true,Value=对应 int64
//
// 用于 min_value / max_value 这类 BIGINT 列:前端 Antd <Input> 总是吐字符串,
// 不能直接绑 *int64,否则 JSON 解码报 "cannot unmarshal string into ... int64"。
type optInt64 struct {
	Value int64
	Valid bool
}

func (o *optInt64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		o.Value, o.Valid = n, true
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected number or numeric string, got %s", string(data))
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parse int64 from %q: %w", s, err)
	}
	o.Value, o.Valid = parsed, true
	return nil
}

// Ptr 把 *optInt64 转 *int64:nil 或 Valid=false 都返回 nil,否则返回拷贝指针。
func (o *optInt64) Ptr() *int64 {
	if o == nil || !o.Valid {
		return nil
	}
	v := o.Value
	return &v
}

// Handler 暴露 /api/v1/param-models/* 与 standard_params CRUD（设计 §1.13）。
//
// 写路径会自动调用 Registry.Refresh 同步内存映射缓存；XML 重载通过
// Reloader 接口注入（dictloader.Registry.ReloadOne 适配器）。
type Handler struct {
	repo     *PgRepository
	registry *Registry
	reloader Reloader
	logger   *zap.Logger
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader 包。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewHandler 构造 Handler；reloader 可为 nil（import-directory 端点会返回 503）。
func NewHandler(repo *PgRepository, registry *Registry, reloader Reloader, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{repo: repo, registry: registry, reloader: reloader, logger: logger.Named("parammodel.handler")}
}

// RegisterRoutes 挂在 /api/v1 下；内部使用 /param-models 子路径。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/param-models")
	// 集中操作（无 :name）
	g.GET("", h.ListModels)
	g.POST("/import-directory", h.ImportDirectory)
	g.POST("/cache/refresh", h.CacheRefresh)
	g.POST("/translate", h.Translate)
	// 标准参数树（位于 /param-models/standard 子路径）
	g.GET("/standard", h.ListStandard)
	g.GET("/standard/:path", h.GetStandard)
	g.POST("/standard", h.UpsertStandard)
	g.PUT("/standard/:path", h.UpdateStandard)
	g.DELETE("/standard/:path", h.DeleteStandard)
	// 单 paramModel
	g.GET("/:name", h.GetModel)
	g.PUT("/:name", h.UpdateModel)
	g.DELETE("/:name", h.DeleteModel)
	// mappings 子资源
	g.GET("/:name/mappings", h.ListMappings)
	g.POST("/:name/mappings", h.CreateMapping)
	g.PUT("/:name/mappings/:id", h.UpdateMapping)
	g.DELETE("/:name/mappings/:id", h.DeleteMapping)

	// discovered 视图（按 product 隔离，挂在 products 命名空间下）
	prod := rg.Group("/products")
	prod.GET("/:id/discovered", h.ListDiscovered)
	prod.GET("/:id/discovered/versions", h.ListDiscoveredVersions)
	prod.DELETE("/:id/discovered/versions/:swVersion", h.DeleteDiscoveredVersion)
}

// ── ParamModel ──────────────────────────────────────────────────────

type modelView struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	TotalEntries int       `json:"total_entries"`
	TotalObjects int       `json:"total_objects"`
	TotalParams  int       `json:"total_params"`
	Description  string    `json:"description"`
	IsActive     bool      `json:"is_active"`
	LoadedFrom   string    `json:"loaded_from"`
}

func toModelView(m *ParamModel) modelView {
	return modelView{
		ID: m.ID, Name: m.Name,
		TotalEntries: m.TotalEntries, TotalObjects: m.TotalObjects, TotalParams: m.TotalParams,
		Description: m.Description, IsActive: m.IsActive, LoadedFrom: m.LoadedFrom,
	}
}

func (h *Handler) ListModels(c *gin.Context) {
	models, err := h.repo.ListParamModels(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]modelView, 0, len(models))
	for i := range models {
		views = append(views, toModelView(&models[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views)})
}

func (h *Handler) GetModel(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, toModelView(m))
}

type updateModelReq struct {
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func (h *Handler) UpdateModel(c *gin.Context) {
	name := c.Param("name")
	var req updateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	m, err := h.repo.UpdateParamModelMeta(c.Request.Context(), name, req.Description, req.IsActive)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "update-model")
	response.OK(c, toModelView(m))
}

func (h *Handler) DeleteModel(c *gin.Context) {
	name := c.Param("name")
	ok, err := h.repo.DeleteParamModel(c.Request.Context(), name)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	h.refreshAsync(c.Request.Context(), "delete-model")
	response.OK(c, gin.H{"deleted": true, "name": name})
}

// ── Mappings ────────────────────────────────────────────────────────

type mappingView struct {
	ID            uuid.UUID `json:"id"`
	ParamModelID  uuid.UUID `json:"param_model_id"`
	StandardPath  string    `json:"standard_path"`
	PrivatePath   string    `json:"private_path"`
	EntryType     string    `json:"entry_type"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *int64    `json:"min_value,omitempty"`
	MaxValue      *int64    `json:"max_value,omitempty"`
	IsStorable    bool      `json:"is_storable"`
	IsActive      bool      `json:"is_active"`
	IsSupported   bool      `json:"is_supported"`
	SoftwareVer   *string   `json:"software_version,omitempty"`
}

func toMappingView(m *ParamMapping) mappingView {
	return mappingView{
		ID: m.ID, ParamModelID: m.ParamModelID,
		StandardPath: m.StandardPath, PrivatePath: m.PrivatePath, EntryType: m.EntryType,
		Access: m.Access, DataType: m.DataType, ChangeApplies: m.ChangeApplies,
		MinValue: m.MinValue, MaxValue: m.MaxValue,
		IsStorable: m.IsStorable, IsActive: m.IsActive, IsSupported: m.IsSupported, SoftwareVer: m.SoftwareVersion,
	}
}

func (h *Handler) ListMappings(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	mappings, err := h.repo.ListMappingsByParamModel(c.Request.Context(), m.ID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]mappingView, 0, len(mappings))
	for i := range mappings {
		views = append(views, toMappingView(&mappings[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views), "param_model": toModelView(m)})
}

type createMappingReq struct {
	StandardPath  string    `json:"standard_path" binding:"required"`
	PrivatePath   string    `json:"private_path" binding:"required"`
	EntryType     string    `json:"entry_type" binding:"required"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
	IsStorable    *bool     `json:"is_storable"`
}

func (h *Handler) CreateMapping(c *gin.Context) {
	name := c.Param("name")
	m, err := h.repo.GetParamModelByName(c.Request.Context(), name)
	if errors.Is(err, ErrNoParamModel) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	var req createMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	in := CreateMappingInput{
		StandardPath:  strings.TrimSpace(req.StandardPath),
		PrivatePath:   strings.TrimSpace(req.PrivatePath),
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
		IsStorable:    true,
	}
	if req.IsStorable != nil {
		in.IsStorable = *req.IsStorable
	}
	created, err := h.repo.CreateMapping(c.Request.Context(), m.ID, in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.invalidateModel(c.Request.Context(), m.ID, "create-mapping")
	response.OKWithStatus(c, http.StatusCreated, toMappingView(created))
}

type updateMappingReq struct {
	PrivatePath   *string   `json:"private_path"`
	Access        *string   `json:"access"`
	DataType      *string   `json:"data_type"`
	ChangeApplies *string   `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
	IsStorable    *bool     `json:"is_storable"`
	IsActive      *bool     `json:"is_active"`
}

func (h *Handler) UpdateMapping(c *gin.Context) {
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("mapping id not a uuid"))
		return
	}
	var req updateMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := h.repo.UpdateMapping(c.Request.Context(), mappingID, UpdateMappingInput{
		PrivatePath:   req.PrivatePath,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
		IsStorable:    req.IsStorable,
		IsActive:      req.IsActive,
	})
	if errors.Is(err, ErrNoMapping) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.invalidateModel(c.Request.Context(), updated.ParamModelID, "update-mapping")
	response.OK(c, toMappingView(updated))
}

func (h *Handler) DeleteMapping(c *gin.Context) {
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("mapping id not a uuid"))
		return
	}
	// 删前查 paramModelID 用于失效
	mapping, getErr := h.repo.getMappingByID(c.Request.Context(), mappingID)
	ok, err := h.repo.DeleteMapping(c.Request.Context(), mappingID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if getErr == nil && mapping != nil {
		h.invalidateModel(c.Request.Context(), mapping.ParamModelID, "delete-mapping")
	}
	response.OK(c, gin.H{"deleted": true, "id": mappingID})
}

// ── Discovered ──────────────────────────────────────────────────────

func (h *Handler) ListDiscovered(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	swVersion := strings.TrimSpace(c.Query("swVersion"))
	if swVersion == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("swVersion query parameter required"))
		return
	}
	mappings, err := h.repo.ListDiscoveredMappings(c.Request.Context(), productID, swVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]mappingView, 0, len(mappings))
	for i := range mappings {
		views = append(views, toMappingView(&mappings[i]))
	}
	response.OK(c, gin.H{
		"items":            views,
		"total":            len(views),
		"product_id":       productID,
		"software_version": swVersion,
	})
}

func (h *Handler) ListDiscoveredVersions(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	versions, err := h.repo.ListDiscoveredVersions(c.Request.Context(), productID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": versions, "total": len(versions), "product_id": productID})
}

func (h *Handler) DeleteDiscoveredVersion(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product id not a uuid"))
		return
	}
	swVersion := strings.TrimSpace(c.Param("swVersion"))
	if swVersion == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("swVersion path parameter required"))
		return
	}
	deleted, err := h.repo.DeleteDiscoveredVersion(c.Request.Context(), productID, swVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if h.registry != nil {
		_ = h.registry.InvalidateProduct(c.Request.Context(), productID, swVersion)
	}
	response.OK(c, gin.H{
		"deleted":          deleted,
		"product_id":       productID,
		"software_version": swVersion,
	})
}

// ── Standard params ─────────────────────────────────────────────────

type standardView struct {
	StandardPath  string `json:"standard_path"`
	EntryType     string `json:"entry_type"`
	Access        string `json:"access"`
	DataType      string `json:"data_type"`
	ChangeApplies string `json:"change_applies"`
	MinValue      *int64 `json:"min_value,omitempty"`
	MaxValue      *int64 `json:"max_value,omitempty"`
}

func toStandardView(sp *StandardParam) standardView {
	return standardView{
		StandardPath: sp.StandardPath, EntryType: sp.EntryType,
		Access: sp.Access, DataType: sp.DataType, ChangeApplies: sp.ChangeApplies,
		MinValue: sp.MinValue, MaxValue: sp.MaxValue,
	}
}

func (h *Handler) ListStandard(c *gin.Context) {
	keyword := c.Query("keyword")
	entryType := c.Query("entry_type")
	items, err := h.repo.ListStandardParams(c.Request.Context(), keyword, entryType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]standardView, 0, len(items))
	for i := range items {
		views = append(views, toStandardView(&items[i]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views)})
}

func (h *Handler) GetStandard(c *gin.Context) {
	standardPath := c.Param("path")
	sp, err := h.repo.GetStandardParam(c.Request.Context(), standardPath)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, toStandardView(sp))
}

type upsertStandardReq struct {
	StandardPath  string    `json:"standard_path" binding:"required"`
	EntryType     string    `json:"entry_type" binding:"required"`
	Access        string    `json:"access"`
	DataType      string    `json:"data_type"`
	ChangeApplies string    `json:"change_applies"`
	MinValue      *optInt64 `json:"min_value"`
	MaxValue      *optInt64 `json:"max_value"`
}

func (h *Handler) UpsertStandard(c *gin.Context) {
	var req upsertStandardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	sp, err := h.repo.UpsertStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, toStandardView(sp))
}

func (h *Handler) UpdateStandard(c *gin.Context) {
	standardPath := c.Param("path")
	var req upsertStandardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// 路径覆盖：URL 上的 path 优先；body 没填或不一致都以 URL 为准
	if req.StandardPath == "" {
		req.StandardPath = standardPath
	} else if req.StandardPath != standardPath {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("standard_path in body (%q) differs from URL (%q)", req.StandardPath, standardPath))
		return
	}
	sp, err := h.repo.UpsertStandardParam(c.Request.Context(), UpsertStandardParamInput{
		StandardPath:  req.StandardPath,
		EntryType:     req.EntryType,
		Access:        req.Access,
		DataType:      req.DataType,
		ChangeApplies: req.ChangeApplies,
		MinValue:      req.MinValue.Ptr(),
		MaxValue:      req.MaxValue.Ptr(),
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, toStandardView(sp))
}

func (h *Handler) DeleteStandard(c *gin.Context) {
	standardPath := c.Param("path")
	ok, err := h.repo.DeleteStandardParam(c.Request.Context(), standardPath)
	if err != nil {
		// 引用拒绝错误用 409 Conflict
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"deleted": true, "standard_path": standardPath})
}

// ── Translate ───────────────────────────────────────────────────────

type translateReq struct {
	ProductID       string   `json:"productId" binding:"required"`
	SoftwareVersion string   `json:"softwareVersion"`
	Direction       string   `json:"direction" binding:"required"` // "to_private" | "to_standard"
	Paths           []string `json:"paths" binding:"required"`
}

type translateItem struct {
	Original   string `json:"original"`
	Translated string `json:"translated"`
	Found      bool   `json:"found"`
	Source     string `json:"source"`
}

func (h *Handler) Translate(c *gin.Context) {
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, fmt.Errorf("param registry not wired"))
		return
	}
	var req translateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("productId not a uuid"))
		return
	}
	if req.Direction != "to_private" && req.Direction != "to_standard" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("direction must be to_private or to_standard"))
		return
	}

	tr, err := h.registry.Translator(c.Request.Context(), productID, req.SoftwareVersion)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	src := string(tr.Source())
	out := make([]translateItem, 0, len(req.Paths))
	for _, p := range req.Paths {
		var tres TranslationResult
		if req.Direction == "to_private" {
			tres = tr.ToPrivate(p)
		} else {
			tres = tr.ToStandard(p)
		}
		out = append(out, translateItem{
			Original:   tres.Original,
			Translated: tres.Translated,
			Found:      tres.Found,
			Source:     src,
		})
	}
	response.OK(c, gin.H{
		"results":          out,
		"product_id":       productID,
		"software_version": req.SoftwareVersion,
		"direction":        req.Direction,
		"source":           src,
	})
}

// ── Cache + Import ──────────────────────────────────────────────────

// ImportDirectory 从 datamodels/ 目录加载 XML。
//
// Query params:
//   - mode=import (默认): 加法 UPSERT —— 仅写入/更新现有 XML 中的模型，
//     不删除 DB 中不在 XML 文件里的孤儿模型（手工 UI 添加项保留）。
//   - mode=reload: destructive 全量重载 —— 完成 UPSERT 后，
//     删除 DB 中所有未被本次加载触达的 param_models（孤儿模型）；
//     param_mappings CASCADE 删除；products.param_model_id SET NULL。
func (h *Handler) ImportDirectory(c *gin.Context) {
	if h.reloader == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("dictloader registry not wired"))
		return
	}
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	if mode == "" {
		mode = "import"
	}
	if mode != "import" && mode != "reload" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("invalid mode %q (expected import|reload)", mode))
		return
	}

	// destructive 模式：记录开始时间，便于事后按 updated_at 识别孤儿
	startedAt := time.Now()

	if err := h.reloader.ReloadOne(c.Request.Context(), "param-model"); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var orphansDeleted int64
	if mode == "reload" {
		var err error
		orphansDeleted, err = h.repo.DeleteOrphansSince(c.Request.Context(), startedAt)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError,
				fmt.Errorf("cleanup orphan param_models: %w", err))
			return
		}
	}

	if h.registry != nil {
		if err := h.registry.Refresh(c.Request.Context()); err != nil {
			h.logger.Warn("post-reload param registry refresh failed", zap.Error(err))
		}
	}
	response.OK(c, gin.H{
		"reloaded":         "param-model",
		"mode":             mode,
		"orphans_deleted":  orphansDeleted,
	})
}

func (h *Handler) CacheRefresh(c *gin.Context) {
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("param registry not wired"))
		return
	}
	if err := h.registry.Refresh(c.Request.Context()); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"refreshed": true})
}

// ── helpers ─────────────────────────────────────────────────────────

func (h *Handler) refreshAsync(ctx context.Context, op string) {
	if h.registry == nil {
		return
	}
	if err := h.registry.Refresh(ctx); err != nil {
		h.logger.Warn("param registry refresh after write failed", zap.String("op", op), zap.Error(err))
	}
}

func (h *Handler) invalidateModel(ctx context.Context, paramModelID uuid.UUID, op string) {
	if h.registry == nil {
		return
	}
	if err := h.registry.InvalidateParamModel(ctx, paramModelID); err != nil {
		h.logger.Warn("param registry invalidate after write failed",
			zap.String("op", op),
			zap.String("param_model_id", paramModelID.String()),
			zap.Error(err))
	}
}
