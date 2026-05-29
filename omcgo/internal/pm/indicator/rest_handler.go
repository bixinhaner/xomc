package indicator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// RESTHandler 提供 T-0098 P3-03 设计 §2.7 规定的 RESTful KPI 管理 API
// （/api/v1/indicators 系列）。它与既有 IndicatorHandler（/pm/indicatormg 子树）
// 路径互不重叠，是治理（产品管理皮肤）视角的纯 REST 入口。
type RESTHandler struct {
	svc      *IndicatorManagementService
	formula  PlatformFormulaRepository
	units    *PgUnitRepository
	reloader Reloader
	fileRepo FileRepository // T-0180 P1.5: 供 ImportDirectory ?mode=reload 调 DeleteOrphansBefore
	logger   *zap.Logger
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewRESTHandler 构造 P3-03 REST handler。
// fileRepo 可为 nil — 此时 ImportDirectory ?mode=reload 返 503(测试场景常用)。
func NewRESTHandler(
	svc *IndicatorManagementService,
	formula PlatformFormulaRepository,
	units *PgUnitRepository,
	reloader Reloader,
	fileRepo FileRepository,
	logger *zap.Logger,
) *RESTHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RESTHandler{svc: svc, formula: formula, units: units, reloader: reloader, fileRepo: fileRepo, logger: logger.Named("indicator.rest")}
}

// RegisterRoutes 挂在 /api/v1 下。
func (h *RESTHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// indicators
	ig := rg.Group("/indicators")
	ig.GET("", h.ListIndicators)
	ig.GET("/platforms", h.ListPlatforms)
	ig.POST("/import-directory", h.ImportDirectory)
	ig.POST("/cache/refresh", h.CacheRefresh)
	ig.GET("/:id", h.GetIndicator)
	ig.POST("", h.CreateIndicator)
	ig.PUT("/:id", h.UpdateIndicator)
	ig.DELETE("/:id", h.DeleteIndicator)
	ig.GET("/:id/formulas", h.ListFormulas)
	ig.GET("/:id/formulas/:platform", h.GetFormula)
	ig.POST("/:id/formulas", h.UpsertFormula)
	ig.DELETE("/:id/formulas/:platform", h.DeleteFormula)

	// indicator-groups
	gg := rg.Group("/indicator-groups")
	gg.GET("", h.ListGroups)
	gg.POST("", h.CreateGroup)
	gg.PUT("/:id", h.UpdateGroup)
	gg.DELETE("/:id", h.DeleteGroup)

	// enabled-indicators
	eg := rg.Group("/enabled-indicators")
	eg.GET("", h.GetEnabled)
	eg.PUT("", h.SetEnabled)

	// indicator-units
	ug := rg.Group("/indicator-units")
	ug.GET("", h.ListUnits)
	ug.POST("", h.UpsertUnit)
	ug.PUT("/:id", h.UpdateUnit)
	ug.DELETE("/:id", h.DeleteUnit)
}

// ── 公共 helper ─────────────────────────────────────────────────────

func parseDeviceTypeQuery(c *gin.Context) (DeviceType, error) {
	v := strings.TrimSpace(c.Query("deviceType"))
	if v == "" {
		v = strings.TrimSpace(c.Query("device_type"))
	}
	if v == "" {
		return "", fmt.Errorf("deviceType query parameter required")
	}
	dt, err := ParseDeviceType(v)
	if err != nil {
		return "", err
	}
	return dt, nil
}

// ── Indicators ──────────────────────────────────────────────────────

func (h *RESTHandler) ListIndicators(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := IndicatorListFilter{
		DeviceType:  string(dt),
		ListRequest: model.DefaultListRequest(),
	}
	// 前端 axios 自动转 pageSize → page_size、sortField → sort_by；同时兼容 camelCase
	if v := strings.TrimSpace(c.Query("page_size")); v == "" {
		v = strings.TrimSpace(c.Query("pageSize"))
		if v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				filter.PageSize = n
			}
		}
	} else if n, err := strconv.Atoi(v); err == nil && n > 0 {
		filter.PageSize = n
	}
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Page = n
		}
	}
	if v := strings.TrimSpace(c.Query("groupId")); v != "" {
		filter.GroupID = &v
	}
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		filter.Keyword = &v
	}
	if v := strings.TrimSpace(c.Query("operatorCode")); v != "" {
		filter.OperatorCode = &v
	}
	if v := strings.TrimSpace(c.Query("productType")); v != "" {
		filter.ProductType = &v
	}
	if v := strings.TrimSpace(c.Query("indicatorLevel")); v != "" {
		filter.IndicatorLevel = &v
	}
	if v := strings.TrimSpace(c.Query("isEnabled")); v != "" {
		filter.IsEnabled = &v
	}
	if v := strings.TrimSpace(c.Query("isCounter")); v != "" {
		filter.IsCounter = &v
	}
	if v := strings.TrimSpace(c.Query("platformName")); v != "" {
		filter.PlatformName = &v
	}
	resp, err := h.svc.ListIndicators(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, resp)
}

// ListPlatforms 返回某 deviceType 下公式表中出现过的全部平台名。
// 用于 KPI 指标库列表页的平台筛选下拉。
func (h *RESTHandler) ListPlatforms(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	names, err := h.svc.ListPlatformNames(c.Request.Context(), dt)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if names == nil {
		names = []string{}
	}
	response.OK(c, gin.H{"items": names, "total": len(names)})
}

func (h *RESTHandler) GetIndicator(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	ind, err := h.svc.GetIndicatorInfo(c.Request.Context(), dt, c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, ind)
}

func (h *RESTHandler) CreateIndicator(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var req CreateIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(dt)
	ind, err := h.svc.CreateIndicator(c.Request.Context(), &req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, ind)
}

func (h *RESTHandler) UpdateIndicator(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var req UpdateIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateIndicator(c.Request.Context(), dt, c.Param("id"), &req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	ind, err := h.svc.GetIndicatorInfo(c.Request.Context(), dt, c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, ind)
}

func (h *RESTHandler) DeleteIndicator(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.DeleteIndicator(c.Request.Context(), dt, c.Param("id")); err != nil {
		// 被模板/公式引用 → 409
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": c.Param("id")})
}

// ── Groups ──────────────────────────────────────────────────────────

func (h *RESTHandler) ListGroups(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	groups, err := h.svc.GetGroupTree(c.Request.Context(), IndicatorGroupTreeRequest{
		DeviceType:   string(dt),
		OperatorCode: strings.TrimSpace(c.Query("operatorCode")),
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": groups, "total": len(groups)})
}

func (h *RESTHandler) CreateGroup(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req.DeviceType = string(dt)
	g, err := h.svc.CreateGroup(c.Request.Context(), &req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, g)
}

func (h *RESTHandler) UpdateGroup(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateGroup(c.Request.Context(), dt, c.Param("id"), &req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	g, err := h.svc.GetGroupByID(c.Request.Context(), dt, c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, g)
}

func (h *RESTHandler) DeleteGroup(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.DeleteGroup(c.Request.Context(), dt, c.Param("id")); err != nil {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": c.Param("id")})
}

// ── Platform formulas ───────────────────────────────────────────────

type upsertFormulaReq struct {
	Platform string `json:"platform" binding:"required"`
	Formula  string `json:"formula" binding:"required"`
}

func (h *RESTHandler) ListFormulas(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	items, err := h.formula.ListByIndicatorID(c.Request.Context(), dt, c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *RESTHandler) GetFormula(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	platform := c.Param("platform")
	items, err := h.formula.ListByIndicatorID(c.Request.Context(), dt, c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	for _, f := range items {
		if f.PlatformName == platform {
			response.OK(c, f)
			return
		}
	}
	commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
}

func (h *RESTHandler) UpsertFormula(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var req upsertFormulaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	indicatorID := c.Param("id")
	// 简化的 upsert：先删除该 (indicator, platform) 行，再 BatchCreate 单条。
	// PlatformFormulaRepository 没有 Upsert/单条删除接口，借助 BatchCreate +
	// 单 platform 维度的全删（DeleteByIndicatorID 会清掉所有 platform，副作用大）。
	// 这里为单 platform 场景：调用方可在 service 层后续提供 UpsertOne。
	// 折中做法：先 ListByIndicatorID → 移除冲突行后 + 新条目重写。
	existing, err := h.formula.ListByIndicatorID(c.Request.Context(), dt, indicatorID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	merged := make([]*PlatformFormula, 0, len(existing)+1)
	for _, f := range existing {
		if f.PlatformName != req.Platform {
			merged = append(merged, f)
		}
	}
	merged = append(merged, &PlatformFormula{
		PlatformName: req.Platform,
		IndicatorID:  indicatorID,
		Formula:      req.Formula,
	})
	if err := h.formula.DeleteByIndicatorID(c.Request.Context(), dt, indicatorID, nil); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if err := h.formula.BatchCreate(c.Request.Context(), dt, merged, nil); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, gin.H{
		"indicator_id": indicatorID,
		"platform":     req.Platform,
		"formula":      req.Formula,
	})
}

func (h *RESTHandler) DeleteFormula(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	platform := c.Param("platform")
	indicatorID := c.Param("id")
	existing, err := h.formula.ListByIndicatorID(c.Request.Context(), dt, indicatorID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	kept := make([]*PlatformFormula, 0, len(existing))
	deleted := false
	for _, f := range existing {
		if f.PlatformName == platform {
			deleted = true
			continue
		}
		kept = append(kept, f)
	}
	if !deleted {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if err := h.formula.DeleteByIndicatorID(c.Request.Context(), dt, indicatorID, nil); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if len(kept) > 0 {
		if err := h.formula.BatchCreate(c.Request.Context(), dt, kept, nil); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
	}
	response.OK(c, gin.H{"deleted": true, "indicator_id": indicatorID, "platform": platform})
}

// ── Enabled-indicators ──────────────────────────────────────────────

type setEnabledReq struct {
	IndicatorIDs []string `json:"indicator_ids" binding:"required"`
	Enable       bool     `json:"enable"`
}

func (h *RESTHandler) GetEnabled(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	op := strings.TrimSpace(c.Query("operatorCode"))
	if op == "" {
		op = "default"
	}
	ids, err := h.svc.GetEnabledIndicatorIDs(c.Request.Context(), dt, op)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": ids, "total": len(ids), "operator_code": op})
}

func (h *RESTHandler) SetEnabled(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	op := strings.TrimSpace(c.Query("operatorCode"))
	if op == "" {
		op = "default"
	}
	var req setEnabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	r := &EnableIndicatorsRequest{
		DeviceType:   string(dt),
		OperatorCode: op,
		IndicatorIDs: req.IndicatorIDs,
		Enable:       req.Enable,
	}
	if req.Enable {
		err = h.svc.EnableIndicators(c.Request.Context(), r)
	} else {
		err = h.svc.DisableIndicators(c.Request.Context(), r)
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, gin.H{"updated": len(req.IndicatorIDs), "operator_code": op, "enable": req.Enable})
}

// ── Indicator units ─────────────────────────────────────────────────

type upsertUnitReq struct {
	ID     string `json:"id" binding:"required"`
	EnName string `json:"en_name"`
	CnName string `json:"cn_name"`
}

func (h *RESTHandler) ListUnits(c *gin.Context) {
	items, err := h.units.List(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *RESTHandler) UpsertUnit(c *gin.Context) {
	var req upsertUnitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	u, err := h.units.Upsert(c.Request.Context(), strings.TrimSpace(req.ID), req.EnName, req.CnName)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, u)
}

func (h *RESTHandler) UpdateUnit(c *gin.Context) {
	id := c.Param("id")
	var req upsertUnitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	u, err := h.units.Upsert(c.Request.Context(), id, req.EnName, req.CnName)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	response.OK(c, u)
}

func (h *RESTHandler) DeleteUnit(c *gin.Context) {
	id := c.Param("id")
	ok, err := h.units.Delete(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusConflict, err)
		return
	}
	if !ok {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": id})
}

// ── Cache + Import ──────────────────────────────────────────────────

// ImportDirectory POST /api/v1/indicators/import-directory?mode=import|reload
//
// T-0180 P1.5: 解析 ?mode= 路由两种语义:
//   - import(默认,向后兼容)— 仅 Loader.Reload(加法 UPSERT);不删孤儿
//   - reload — Loader.Reload + 三制式 DeleteOrphansBefore(destructive 全量重载)
//     依赖 perf_indicators_<tech> BEFORE UPDATE trigger 自动刷 updated_at
func (h *RESTHandler) ImportDirectory(c *gin.Context) {
	if h.reloader == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, errors.New("dictloader registry not wired"))
		return
	}

	mode, err := ParseReloadMode(c.Query("mode"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	switch mode {
	case ReloadModeImport:
		// 老行为:仅 UPSERT,不删孤儿
		if err := h.reloader.ReloadOne(c.Request.Context(), LoaderName); err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		response.OK(c, gin.H{
			"reloaded": LoaderName,
			"mode":     string(ReloadModeImport),
		})

	case ReloadModeReload:
		// destructive 模式需 fileRepo 才能删孤儿
		if h.fileRepo == nil {
			commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
				errors.New("file repository not wired; reload mode unavailable"))
			return
		}
		result, err := PerformReloadWithOrphans(c.Request.Context(), h.fileRepo, h.reloader, h.logger)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		response.OK(c, gin.H{
			"reloaded": LoaderName,
			"mode":     string(result.Mode),
			"orphans":  result.Orphans,
		})
	}
}

func (h *RESTHandler) CacheRefresh(c *gin.Context) {
	// 触发跨实例失效：递增 indicator:cache_version（设计 §2.8 + dictloader §5.4 协议）。
	// service 自身用 pull-through 缓存，本端实例下次查询从 DB 重读；其他实例 30s 内
	// 轮询到 cache_version 变化后清空 L1 sync.Map。
	h.svc.BumpCacheVersion(c.Request.Context())
	response.OK(c, gin.H{"refreshed": true})
}
