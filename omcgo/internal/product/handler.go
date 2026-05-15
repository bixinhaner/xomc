package product

import (
	"context"
	"encoding/json"
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

// Handler 暴露 /api/v1/products/* (设计 §4.4)。
//
// 写路径完成后调用 Registry.Refresh 同步路由缓存；删除产品时 product_class_patterns
// 与 discovered_param_mappings 由 FK CASCADE 自动清理（设计 §4.2）。
//
// DiscoveredCleaner 由 provider 注入 parammodel.PgRepository.DeleteDiscoveredAll 来
// 实现 reset 端点（避免 product 反向 import parammodel 包）。
type Handler struct {
	repo      *PgRepository
	registry  *Registry
	cleaner   DiscoveredCleaner
	rematcher Rematcher
	reloader  Reloader
	logger    *zap.Logger
}

// DiscoveredCleaner 抽象"清空某 product 所有 swVersion 的 discovered 映射"。
// 实际由 parammodel.PgRepository.DeleteDiscoveredAll 实现，handler 不直接 import。
type DiscoveredCleaner interface {
	DeleteDiscoveredAll(ctx context.Context, productID uuid.UUID) (int64, error)
}

// Rematcher 抽象批量重新匹配孤儿设备的接口；nil 时 rematch 端点退化为
// "本端 SQL 重算 product_id"。生产期可注入 device 模块的实现。
type Rematcher interface {
	RematchOrphans(ctx context.Context) (int, error)
}

// Reloader 抽象 dictloader.Registry.ReloadOne — 让 handler 不依赖 dictloader。
type Reloader interface {
	ReloadOne(ctx context.Context, name string) error
}

// NewHandler 构造 Handler。
func NewHandler(repo *PgRepository, registry *Registry, cleaner DiscoveredCleaner, rematcher Rematcher, reloader Reloader, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		repo:      repo,
		registry:  registry,
		cleaner:   cleaner,
		rematcher: rematcher,
		reloader:  reloader,
		logger:    logger.Named("product.handler"),
	}
}

// RegisterRoutes 挂载 /api/v1/products/* 到给定 RouterGroup。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/products")
	// 集中操作（无 :id）放最前以避免被 :id 路由吞掉
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/match", h.Match)
	g.GET("/match-order", h.MatchOrder)
	g.GET("/orphan-devices", h.ListOrphan)
	g.POST("/orphan-devices/rematch", h.RematchOrphan)
	g.PUT("/orphan-devices/:deviceId/bind", h.BindOrphan)
	g.POST("/cache/refresh", h.CacheRefresh)
	g.POST("/import-directory", h.ImportDirectory)
	// 枚举字典（产品表单下拉框用）
	g.GET("/indicator-platforms", h.ListIndicatorPlatforms)
	g.GET("/alarm-ne-types", h.ListAlarmNeTypes)

	// 单产品
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.DELETE("/:id/discovered", h.ResetDiscovered)
	// patterns
	g.POST("/:id/patterns", h.CreatePattern)
	g.PUT("/:id/patterns/:patternId", h.UpdatePattern)
	g.DELETE("/:id/patterns/:patternId", h.DeletePattern)
	g.PUT("/:id/patterns/:patternId/move", h.MovePattern)
}

// ── DTO ─────────────────────────────────────────────────────────────

type productView struct {
	ID                  uuid.UUID      `json:"id"`
	Name                string         `json:"name"`
	Vendor              string         `json:"vendor"`
	Tech                string         `json:"tech"`
	RadioModes          string         `json:"radio_modes"`
	Description         string         `json:"description"`
	ParamModelID        *uuid.UUID     `json:"param_model_id,omitempty"`
	IndicatorDeviceType string         `json:"indicator_device_type"`
	IndicatorPlatform   string         `json:"indicator_platform"`
	AlarmNeType         string         `json:"alarm_ne_type"`
	EnableFileType11    bool           `json:"enable_filetype11"`
	DeviceAttrsOverride map[string]any `json:"device_attrs_override"`
	EnableUnknownAlarm  bool           `json:"enable_unknown_alarm"`
	DeviceCount         int            `json:"device_count,omitempty"`
}

func toProductView(p *Product, deviceCount int) productView {
	return productView{
		ID: p.ID, Name: p.Name, Vendor: p.Vendor, Tech: p.Tech, RadioModes: p.RadioModes,
		Description: p.Description, ParamModelID: p.ParamModelID,
		IndicatorDeviceType: p.IndicatorDeviceType, IndicatorPlatform: p.IndicatorPlatform,
		AlarmNeType:         p.AlarmNeType,
		EnableFileType11:    p.EnableFileType11,
		DeviceAttrsOverride: p.DeviceAttrsOverride,
		EnableUnknownAlarm:  p.EnableUnknownAlarm,
		DeviceCount:         deviceCount,
	}
}

// ── List/Get ────────────────────────────────────────────────────────

func (h *Handler) List(c *gin.Context) {
	products, err := h.repo.ListProducts(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	counts, err := h.repo.CountDevicesByProduct(c.Request.Context())
	if err != nil {
		// 非致命：列表照常返回，只是 device_count 缺
		h.logger.Warn("CountDevicesByProduct failed", zap.Error(err))
		counts = map[uuid.UUID]int{}
	}
	vendor := strings.TrimSpace(c.Query("vendor"))
	tech := strings.TrimSpace(c.Query("tech"))
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))

	views := make([]productView, 0, len(products))
	for _, p := range products {
		if vendor != "" && !strings.EqualFold(p.Vendor, vendor) {
			continue
		}
		if tech != "" && !strings.EqualFold(p.Tech, tech) {
			continue
		}
		if keyword != "" {
			hay := strings.ToLower(p.Name + " " + p.Vendor + " " + p.Description)
			if !strings.Contains(hay, keyword) {
				continue
			}
		}
		views = append(views, toProductView(p, counts[p.ID]))
	}
	response.OK(c, gin.H{"items": views, "total": len(views)})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	p, err := h.repo.GetProductByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if p == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	patterns, _ := h.repo.ListPatternsByProduct(c.Request.Context(), id)
	counts, _ := h.repo.CountDevicesByProduct(c.Request.Context())
	response.OK(c, gin.H{
		"product":  toProductView(p, counts[id]),
		"patterns": patterns,
	})
}

// ── Create/Update/Delete ────────────────────────────────────────────

type createProductReq struct {
	Name                string         `json:"name" binding:"required"`
	Vendor              string         `json:"vendor"`
	Tech                string         `json:"tech"`
	RadioModes          string         `json:"radio_modes"`
	Description         string         `json:"description"`
	ParamModelName      string         `json:"param_model_name"` // 反查 param_models.id
	ParamModelID        *string        `json:"param_model_id"`   // 直接给 ID 也接受
	IndicatorDeviceType string         `json:"indicator_device_type" binding:"required"`
	IndicatorPlatform   string         `json:"indicator_platform"` // 仅 ENB 必填，由 service 层按 device_type 校验
	AlarmNeType         string         `json:"alarm_ne_type" binding:"required"`
	EnableFileType11    *bool          `json:"enable_filetype11"`
	DeviceAttrsOverride map[string]any `json:"device_attrs_override"`
	EnableUnknownAlarm  *bool          `json:"enable_unknown_alarm"`
	Patterns            []string       `json:"patterns"` // 可选；事务内随 product 一并落库
}

func (h *Handler) Create(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// indicator_device_type 入库前归一化为小写（与 XML loader / 表名 switch case / DB 现状一致）
	req.IndicatorDeviceType = strings.ToLower(strings.TrimSpace(req.IndicatorDeviceType))
	// indicator_platform 仅 ENB 类型必填；GNB / GSM 不收集（前端表单也对应隐藏）
	if req.IndicatorDeviceType == "enb" && strings.TrimSpace(req.IndicatorPlatform) == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("indicator_platform is required when indicator_device_type=enb"))
		return
	}
	in := CreateProductInput{
		Name:                strings.TrimSpace(req.Name),
		Vendor:              req.Vendor,
		Tech:                req.Tech,
		RadioModes:          req.RadioModes,
		Description:         req.Description,
		IndicatorDeviceType: req.IndicatorDeviceType,
		IndicatorPlatform:   req.IndicatorPlatform,
		AlarmNeType:         req.AlarmNeType,
		EnableFileType11:    true,
		DeviceAttrsOverride: req.DeviceAttrsOverride,
		EnableUnknownAlarm:  false,
		Patterns:            req.Patterns,
	}
	if req.EnableFileType11 != nil {
		in.EnableFileType11 = *req.EnableFileType11
	}
	if req.EnableUnknownAlarm != nil {
		in.EnableUnknownAlarm = *req.EnableUnknownAlarm
	}
	if in.DeviceAttrsOverride == nil {
		in.DeviceAttrsOverride = map[string]any{}
	}
	// param_model 解析：name 优先 → ID 备用
	if name := strings.TrimSpace(req.ParamModelName); name != "" {
		id, err := h.repo.LookupParamModelIDByName(c.Request.Context(), name)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		in.ParamModelID = &id
	} else if req.ParamModelID != nil && *req.ParamModelID != "" {
		id, err := uuid.Parse(*req.ParamModelID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("param_model_id not a uuid"))
			return
		}
		in.ParamModelID = &id
	}
	p, err := h.repo.CreateProduct(c.Request.Context(), in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "create-product")
	response.OKWithStatus(c, http.StatusCreated, toProductView(p, 0))
}

type updateProductReq struct {
	Name                *string         `json:"name"`
	Vendor              *string         `json:"vendor"`
	Tech                *string         `json:"tech"`
	RadioModes          *string         `json:"radio_modes"`
	Description         *string         `json:"description"`
	ParamModelName      *string         `json:"param_model_name"`
	ParamModelID        *string         `json:"param_model_id"`
	ClearParamModel     bool            `json:"clear_param_model"`
	IndicatorDeviceType *string         `json:"indicator_device_type"`
	IndicatorPlatform   *string         `json:"indicator_platform"`
	AlarmNeType         *string         `json:"alarm_ne_type"`
	EnableFileType11    *bool           `json:"enable_filetype11"`
	DeviceAttrsOverride map[string]any  `json:"device_attrs_override"`
	EnableUnknownAlarm  *bool           `json:"enable_unknown_alarm"`
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	// indicator_device_type 入库前归一化为小写
	if req.IndicatorDeviceType != nil {
		v := strings.ToLower(strings.TrimSpace(*req.IndicatorDeviceType))
		req.IndicatorDeviceType = &v
	}
	// 若本次更新把 device_type 改为 enb，必须同时提供非空 indicator_platform
	if req.IndicatorDeviceType != nil && *req.IndicatorDeviceType == "enb" &&
		(req.IndicatorPlatform == nil || strings.TrimSpace(*req.IndicatorPlatform) == "") {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("indicator_platform is required when indicator_device_type=enb"))
		return
	}
	in := UpdateProductInput{
		Name:                req.Name,
		Vendor:              req.Vendor,
		Tech:                req.Tech,
		RadioModes:          req.RadioModes,
		Description:         req.Description,
		IndicatorDeviceType: req.IndicatorDeviceType,
		IndicatorPlatform:   req.IndicatorPlatform,
		AlarmNeType:         req.AlarmNeType,
		EnableFileType11:    req.EnableFileType11,
		DeviceAttrsOverride: req.DeviceAttrsOverride,
		EnableUnknownAlarm:  req.EnableUnknownAlarm,
	}
	if req.ClearParamModel {
		in.clearParamModelID = true
	}
	if req.ParamModelName != nil && *req.ParamModelName != "" {
		pmid, err := h.repo.LookupParamModelIDByName(c.Request.Context(), *req.ParamModelName)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		in.ParamModelID = &pmid
	} else if req.ParamModelID != nil && *req.ParamModelID != "" {
		pmid, err := uuid.Parse(*req.ParamModelID)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("param_model_id not a uuid"))
			return
		}
		in.ParamModelID = &pmid
	}
	p, err := h.repo.UpdateProduct(c.Request.Context(), id, in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "update-product")
	response.OK(c, toProductView(p, 0))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	if err := h.repo.DeleteProduct(c.Request.Context(), id); err != nil {
		// 设备引用 → 409，其他 → 500
		if strings.Contains(err.Error(), "referenced by") {
			commonerrors.AbortWithError(c, http.StatusConflict, err)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "delete-product")
	response.OK(c, gin.H{"deleted": true, "id": id})
}

// ResetDiscovered DELETE /api/v1/products/:id/discovered — 清空所有 swVersion。
func (h *Handler) ResetDiscovered(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	if h.cleaner == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("discovered cleaner not wired"))
		return
	}
	deleted, err := h.cleaner.DeleteDiscoveredAll(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// 设备数（统计当前绑定该产品的设备数；下次 Bootstrap 各自重新触发）
	counts, _ := h.repo.CountDevicesByProduct(c.Request.Context())
	response.OK(c, gin.H{
		"deleted_rows":   deleted,
		"product_id":     id,
		"bound_devices":  counts[id],
	})
}

// ── Patterns ────────────────────────────────────────────────────────

type createPatternReq struct {
	ProductClass string `json:"product_class" binding:"required"`
}

func (h *Handler) CreatePattern(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	var req createPatternReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	pv, err := h.repo.CreatePattern(c.Request.Context(), productID, req.ProductClass)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "create-pattern")
	response.OKWithStatus(c, http.StatusCreated, pv)
}

type updatePatternReq struct {
	ProductClass string `json:"product_class" binding:"required"`
}

func (h *Handler) UpdatePattern(c *gin.Context) {
	patternID, err := uuid.Parse(c.Param("patternId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("patternId not a uuid"))
		return
	}
	var req updatePatternReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	pv, err := h.repo.UpdatePattern(c.Request.Context(), patternID, req.ProductClass)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "update-pattern")
	response.OK(c, pv)
}

func (h *Handler) DeletePattern(c *gin.Context) {
	patternID, err := uuid.Parse(c.Param("patternId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("patternId not a uuid"))
		return
	}
	if err := h.repo.DeletePattern(c.Request.Context(), patternID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "delete-pattern")
	response.OK(c, gin.H{"deleted": true, "id": patternID})
}

type movePatternReq struct {
	Direction string `json:"direction" binding:"required"` // up | down
}

func (h *Handler) MovePattern(c *gin.Context) {
	patternID, err := uuid.Parse(c.Param("patternId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("patternId not a uuid"))
		return
	}
	// 支持 query string 与 body 两种形式
	direction := c.Query("direction")
	if direction == "" {
		var req movePatternReq
		if err := c.ShouldBindJSON(&req); err == nil {
			direction = req.Direction
		}
	}
	if direction != "up" && direction != "down" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("direction must be up|down"))
		return
	}
	pv, err := h.repo.MovePattern(c.Request.Context(), patternID, direction)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "move-pattern")
	response.OK(c, pv)
}

// ── Match / MatchOrder ──────────────────────────────────────────────

func (h *Handler) Match(c *gin.Context) {
	productClass := strings.TrimSpace(c.Query("productClass"))
	if productClass == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("productClass query parameter required"))
		return
	}
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, fmt.Errorf("product registry not wired"))
		return
	}
	mr, err := h.registry.MatchProductClass(c.Request.Context(), productClass)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if mr == nil {
		response.OK(c, gin.H{
			"matched":       false,
			"product_class": productClass,
		})
		return
	}
	response.OK(c, gin.H{
		"matched":         true,
		"product":         toProductView(mr.Product, 0),
		"matched_pattern": mr.MatchedPattern,
		"global_order":    mr.GlobalOrder,
		"product_class":   productClass,
	})
}

func (h *Handler) MatchOrder(c *gin.Context) {
	rows, err := h.repo.ListMatchOrder(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": rows, "total": len(rows)})
}

// ── Orphan devices ──────────────────────────────────────────────────

func (h *Handler) ListOrphan(c *gin.Context) {
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	orphans, err := h.repo.ListOrphanDevices(c.Request.Context(), limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": orphans, "total": len(orphans)})
}

func (h *Handler) RematchOrphan(c *gin.Context) {
	if h.rematcher != nil {
		n, err := h.rematcher.RematchOrphans(c.Request.Context())
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		response.OK(c, gin.H{"rebound": n})
		return
	}
	// 内置 fallback：本端按 Registry.MatchProductClass 重算
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, fmt.Errorf("registry not wired"))
		return
	}
	orphans, err := h.repo.ListOrphanDevices(c.Request.Context(), 1000)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	rebound := 0
	for _, d := range orphans {
		if d.ProductClass == "" {
			continue
		}
		mr, err := h.registry.MatchProductClass(c.Request.Context(), d.ProductClass)
		if err != nil || mr == nil || mr.Product == nil {
			continue
		}
		if err := h.repo.BindOrphanDevice(c.Request.Context(), d.ID, mr.Product.ID); err == nil {
			rebound++
		}
	}
	response.OK(c, gin.H{"rebound": rebound, "scanned": len(orphans)})
}

func (h *Handler) BindOrphan(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("deviceId not a uuid"))
		return
	}
	var body struct {
		ProductID string `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	productID, err := uuid.Parse(body.ProductID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("product_id not a uuid"))
		return
	}
	// 校验目标 product 存在
	p, err := h.repo.GetProductByID(c.Request.Context(), productID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if p == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, fmt.Errorf("target product %s not found", productID))
		return
	}
	if err := h.repo.BindOrphanDevice(c.Request.Context(), deviceID, productID); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"device_id": deviceID, "product_id": productID})
}

// ── Cache + Import ──────────────────────────────────────────────────

func (h *Handler) CacheRefresh(c *gin.Context) {
	if h.registry == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable,
			fmt.Errorf("product registry not wired"))
		return
	}
	if err := h.registry.Refresh(c.Request.Context()); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"refreshed": true})
}

func (h *Handler) ImportDirectory(c *gin.Context) {
	if h.reloader == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, errors.New("dictloader registry not wired"))
		return
	}
	if err := h.reloader.ReloadOne(c.Request.Context(), "product"); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if h.registry != nil {
		if err := h.registry.Refresh(c.Request.Context()); err != nil {
			h.logger.Warn("post-reload product registry refresh failed", zap.Error(err))
		}
	}
	response.OK(c, gin.H{"reloaded": "product"})
}

// ── Enums (form dropdowns) ──────────────────────────────────────────

// ListIndicatorPlatforms GET /api/v1/products/indicator-platforms?deviceType=ENB|GSM|GNB
// 仅 ENB 当前有多平台（default / comba / ...），其他类型可能返回空集；前端据此控制可见性。
func (h *Handler) ListIndicatorPlatforms(c *gin.Context) {
	deviceType := strings.TrimSpace(c.Query("deviceType"))
	if deviceType == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("deviceType query parameter required"))
		return
	}
	items, err := h.repo.ListIndicatorPlatforms(c.Request.Context(), deviceType)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{
		"device_type": deviceType,
		"items":       items,
		"total":       len(items),
	})
}

// ListAlarmNeTypes GET /api/v1/products/alarm-ne-types
// 返回 alarm_definitions.ne_type 字典（如 ENB / GNB / OMC / EPC / EGW / CPE / UPS）。
func (h *Handler) ListAlarmNeTypes(c *gin.Context) {
	items, err := h.repo.ListAlarmNeTypes(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

// ── helpers ─────────────────────────────────────────────────────────

func (h *Handler) refreshAsync(ctx context.Context, op string) {
	if h.registry == nil {
		return
	}
	if err := h.registry.Refresh(ctx); err != nil {
		h.logger.Warn("product registry refresh after write failed",
			zap.String("op", op), zap.Error(err))
	}
}

// 占位以避免 unused 警告（部分 helpers 在不同流程下被调用）
var _ = json.Marshal
