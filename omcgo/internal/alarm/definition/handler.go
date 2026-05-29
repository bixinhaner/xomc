package definition

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 暴露 /api/v1/alarm-definitions/* 与 /api/v1/alarm-severity-levels（设计 §3.5）。
//
// 写路径走 Service（DB + Registry refresh）。读路径直接走 repo（保持 ListAll 等
// 既有方法的语义）。XML 导入复用 dictloader.Loader 的 Reload — 通过 Reloader
// 接口注入，避免 handler 直接依赖 dictloader 包。
type Handler struct {
	service  *Service
	reloader Reloader
	logger   *zap.Logger
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader 包。
//
// 任一实现失败时 handler 返回 500 + 详细 message；调用方不会破坏数据库。
// 返回值与 dictloader.Registry.ReloadOne 同：(Report-like, error)；handler 仅
// 关心 error，返回的第一个值由 provider 适配为 nil/任意类型即可。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewHandler 构造 Handler。reloader 为 nil 时 import-directory 端点返回 503。
func NewHandler(service *Service, reloader Reloader, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{service: service, reloader: reloader, logger: logger.Named("alarmdef.handler")}
}

// RegisterRoutes 注册到给定 RouterGroup。
//
// 调用方约定挂在 /api/v1 之下；本 handler 内部使用绝对子路径
// /alarm-definitions 与 /alarm-severity-levels。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/alarm-definitions")
	g.GET("", h.List)
	g.GET("/ne-types", h.NeTypes)
	g.GET("/unknown-stats", h.UnknownStats)
	g.POST("/import-directory", h.ImportDirectory)
	g.POST("/cache/refresh", h.CacheRefresh)
	g.GET("/:identifier", h.Get)
	g.POST("", h.Create)
	g.PUT("/:identifier", h.Update)
	g.DELETE("/:identifier", h.Delete)

	rg.GET("/alarm-severity-levels", h.SeverityLevels)
}

// ── DTO ─────────────────────────────────────────────────────────────

type listQuery struct {
	NeType       string `form:"ne_type"`
	SeverityCode int    `form:"severity_code"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type createReq struct {
	Identifier      string `json:"identifier" binding:"required"`
	NeType          string `json:"ne_type" binding:"required"`
	CnName          string `json:"cn_name"`
	EnName          string `json:"en_name"`
	SeverityCode    int    `json:"severity_code" binding:"required"`
	EventType       *int   `json:"event_type"`
	CnProbableCause string `json:"cn_probable_cause"`
	EnProbableCause string `json:"en_probable_cause"`
	CnSuggestion    string `json:"cn_suggestion"`
	EnSuggestion    string `json:"en_suggestion"`
	IsShow          *bool  `json:"is_show"`
}

type updateReq struct {
	NeType          *string `json:"ne_type"`
	CnName          *string `json:"cn_name"`
	EnName          *string `json:"en_name"`
	SeverityCode    *int    `json:"severity_code"`
	EventType       *int    `json:"event_type"`
	CnProbableCause *string `json:"cn_probable_cause"`
	EnProbableCause *string `json:"en_probable_cause"`
	CnSuggestion    *string `json:"cn_suggestion"`
	EnSuggestion    *string `json:"en_suggestion"`
	IsShow          *bool   `json:"is_show"`
}

type defView struct {
	ID              uuid.UUID `json:"id"`
	Identifier      string    `json:"identifier"`
	NeType          string    `json:"ne_type"`
	CnName          string    `json:"cn_name"`
	EnName          string    `json:"en_name"`
	SeverityID      uuid.UUID `json:"severity_id"`
	SeverityCode    int       `json:"severity_code"`
	SeverityName    string    `json:"severity_name"`
	EventType       *int      `json:"event_type,omitempty"`
	CnProbableCause string    `json:"cn_probable_cause"`
	EnProbableCause string    `json:"en_probable_cause"`
	CnSuggestion    string    `json:"cn_suggestion"`
	EnSuggestion    string    `json:"en_suggestion"`
	IsShow          bool      `json:"is_show"`
}

func toView(rd *ResolvedDefinition) defView {
	return defView{
		ID:              rd.ID,
		Identifier:      rd.Identifier,
		NeType:          rd.NeType,
		CnName:          rd.CnName,
		EnName:          rd.EnName,
		SeverityID:      rd.SeverityID,
		SeverityCode:    rd.SeverityCode,
		SeverityName:    rd.SeverityName,
		EventType:       rd.EventType,
		CnProbableCause: rd.CnProbableCause,
		EnProbableCause: rd.EnProbableCause,
		CnSuggestion:    rd.CnSuggestion,
		EnSuggestion:    rd.EnSuggestion,
		IsShow:          rd.IsShow,
	}
}

// ── Endpoint impls ──────────────────────────────────────────────────

// List GET /api/v1/alarm-definitions
func (h *Handler) List(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	f := ListFilter{Page: q.Page, PageSize: q.PageSize}
	if strings.TrimSpace(q.NeType) != "" {
		s := strings.TrimSpace(q.NeType)
		f.NeType = &s
	}
	if q.SeverityCode != 0 {
		v := q.SeverityCode
		f.SeverityCode = &v
	}
	if strings.TrimSpace(q.Keyword) != "" {
		s := strings.TrimSpace(q.Keyword)
		f.Keyword = &s
	}

	defs, total, err := h.service.List(c.Request.Context(), f)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	views := make([]defView, 0, len(defs))
	for i := range defs {
		views = append(views, toView(&defs[i]))
	}
	response.OK(c, gin.H{
		"items":     views,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

// Get GET /api/v1/alarm-definitions/:identifier
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("identifier")
	rd, err := h.service.Get(c.Request.Context(), id)
	if errors.Is(err, ErrUnknownIdentifier) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, toView(rd))
}

// Create POST /api/v1/alarm-definitions
func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	in := CreateInput{
		Identifier:      strings.TrimSpace(req.Identifier),
		NeType:          strings.TrimSpace(req.NeType),
		CnName:          req.CnName,
		EnName:          req.EnName,
		SeverityCode:    req.SeverityCode,
		EventType:       req.EventType,
		CnProbableCause: req.CnProbableCause,
		EnProbableCause: req.EnProbableCause,
		CnSuggestion:    req.CnSuggestion,
		EnSuggestion:    req.EnSuggestion,
		IsShow:          true,
	}
	if req.IsShow != nil {
		in.IsShow = *req.IsShow
	}
	rd, err := h.service.Create(c.Request.Context(), in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, toView(rd))
}

// Update PUT /api/v1/alarm-definitions/:identifier
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("identifier")
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	rd, err := h.service.Update(c.Request.Context(), id, UpdateInput{
		NeType:          req.NeType,
		CnName:          req.CnName,
		EnName:          req.EnName,
		SeverityCode:    req.SeverityCode,
		EventType:       req.EventType,
		CnProbableCause: req.CnProbableCause,
		EnProbableCause: req.EnProbableCause,
		CnSuggestion:    req.CnSuggestion,
		EnSuggestion:    req.EnSuggestion,
		IsShow:          req.IsShow,
	})
	if errors.Is(err, ErrUnknownIdentifier) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, toView(rd))
}

// Delete DELETE /api/v1/alarm-definitions/:identifier
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("identifier")
	ok, err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"deleted": true, "identifier": id})
}

// SeverityLevels GET /api/v1/alarm-severity-levels — 只读 4 行种子。
// 返回包装 {items: [...]} 与 list-style 端点(/alarm-definitions, /ne-types)对齐;
// 前端 severityLevels() 读 data.items,原裸数组返回时 data.items=undefined → 空下拉
// (2026-05-29 用户报告)。
func (h *Handler) SeverityLevels(c *gin.Context) {
	levels, err := h.service.ListSeverityLevels(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": levels})
}

// UnknownStats GET /api/v1/alarm-definitions/unknown-stats?productId=&days=7
func (h *Handler) UnknownStats(c *gin.Context) {
	var pid *uuid.UUID
	if v := strings.TrimSpace(c.Query("productId")); v != "" {
		u, err := uuid.Parse(v)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("productId not a uuid"))
			return
		}
		pid = &u
	}
	days := 7
	if v := c.Query("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}
	stats, err := h.service.UnknownStats(c.Request.Context(), pid, days)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{
		"items": stats,
		"days":  days,
	})
}

// NeTypes GET /api/v1/alarm-definitions/ne-types
//
// T-0179 drill-down 一级视图聚合端点:按 (ne_type, loaded_from) 双键统计行数 +
// 4 个严重级计数。前端 alarm-library 一级表格直接渲染本接口返回的 items 数组。
func (h *Handler) NeTypes(c *gin.Context) {
	stats, err := h.service.ListNeTypes(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": stats})
}

// ImportDirectory POST /api/v1/alarm-definitions/import-directory
//
// 调用 dictloader.Reload(LoaderName)：扫描 XML 目录 → upsert alarm_definitions →
// Registry.Refresh。等价于"重新跑启动期 LoadOnce"。
func (h *Handler) ImportDirectory(c *gin.Context) {
	if h.reloader == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("dictloader registry not wired"))
		return
	}
	if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if err := h.service.RefreshCache(c.Request.Context()); err != nil {
		// 不阻断：DB 已 upsert 成功，缓存最多落后一拍
		h.logger.Warn("post-reload registry refresh failed", zap.Error(err))
	}
	response.OK(c, gin.H{"reloaded": LoaderName})
}

// CacheRefresh POST /api/v1/alarm-definitions/cache/refresh
func (h *Handler) CacheRefresh(c *gin.Context) {
	if err := h.service.RefreshCache(c.Request.Context()); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"refreshed": true})
}
