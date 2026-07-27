package indicator

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// RESTHandler 提供 T-0098 P3-03 设计 §2.7 规定的 RESTful KPI 管理 API
// （/api/v1/indicators 系列）。它与既有 IndicatorHandler（/pm/indicatormg 子树）
// 路径互不重叠，是治理（产品管理皮肤）视角的纯 REST 入口。
type RESTHandler struct {
	svc     *IndicatorManagementService
	formula PlatformFormulaRepository
	logger  *zap.Logger
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不强依赖 dictloader。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewRESTHandler 构造 REST handler(indicator 行级 CRUD + groups + enabled)。
//
// 2026-06-03 用户决策:import-directory / cache/refresh 端点已下线(合并进 upload-xml),
// 故不再需要 reloader / fileRepo 注入。
func NewRESTHandler(
	svc *IndicatorManagementService,
	formula PlatformFormulaRepository,
	logger *zap.Logger,
) *RESTHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RESTHandler{svc: svc, formula: formula, logger: logger.Named("indicator.rest")}
}

// RegisterRoutes 挂在 /api/v1 下。
func (h *RESTHandler) RegisterRoutes(rg *gin.RouterGroup) {
	h.RegisterReadRoutes(rg)
	h.RegisterWriteRoutes(rg)
}

// RegisterReadRoutes 挂载指标库的只读接口，供已登录用户读取 KPI 数据。
func (h *RESTHandler) RegisterReadRoutes(rg *gin.RouterGroup) {
	// indicators
	ig := rg.Group("/indicators")
	ig.GET("", h.ListIndicators)
	ig.GET("/platforms", h.ListPlatforms)
	ig.GET("/:id", h.GetIndicator)
	ig.GET("/:id/formulas", h.ListFormulas)
	ig.GET("/:id/formulas/:platform", h.GetFormula)

	// indicator-groups
	gg := rg.Group("/indicator-groups")
	gg.GET("", h.ListGroups)

	// enabled-indicators
	eg := rg.Group("/enabled-indicators")
	eg.GET("", h.GetEnabled)

}

// RegisterWriteRoutes 挂载指标库的写接口，仅供超管管理。
func (h *RESTHandler) RegisterWriteRoutes(rg *gin.RouterGroup) {
	// indicators
	ig := rg.Group("/indicators")
	ig.POST("", h.CreateIndicator)
	ig.PUT("/:id", h.UpdateIndicator)
	ig.DELETE("/:id", h.DeleteIndicator)
	ig.POST("/:id/formulas", h.UpsertFormula)
	ig.DELETE("/:id/formulas/:platform", h.DeleteFormula)

	// indicator-groups
	gg := rg.Group("/indicator-groups")
	gg.POST("", h.CreateGroup)
	gg.PUT("/:id", h.UpdateGroup)
	gg.DELETE("/:id", h.DeleteGroup)

	// enabled-indicators
	eg := rg.Group("/enabled-indicators")
	eg.PUT("", h.SetEnabled)
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
	operatorCode := strings.TrimSpace(c.Query("operatorCode"))
	if operatorCode == "" {
		operatorCode = strings.TrimSpace(c.Query("operator_code"))
	}
	if operatorCode == "" {
		operatorCode = "default"
	}
	filter.OperatorCode = &operatorCode
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
	// issue #67 §4：按请求 locale 把 data_type 受控码映射为 i18n 标签（不改 data_type 码本身）。
	if resp != nil {
		fillDataTypeLabels(resp.Items, appcontext.GetLocale(c.Request.Context()))
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
	// issue #67 §4：详情同样按 locale 填 data_type i18n 标签。
	fillDataTypeLabel(ind, appcontext.GetLocale(c.Request.Context()))
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
		// 2026-05-29:平台过滤 — 与 ListIndicators 的 PlatformName 同语义
		Platform: strings.TrimSpace(c.Query("platform")),
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
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
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
	out, err := h.svc.UpsertPlatformFormula(c.Request.Context(), dt, indicatorID, req.Platform, req.Formula)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, out)
}

func (h *RESTHandler) DeleteFormula(c *gin.Context) {
	dt, err := parseDeviceTypeQuery(c)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	platform := c.Param("platform")
	indicatorID := c.Param("id")
	if err := h.svc.DeletePlatformFormula(c.Request.Context(), dt, indicatorID, platform); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
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
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, gin.H{"updated": len(req.IndicatorIDs), "operator_code": op, "enable": req.Enable})
}
