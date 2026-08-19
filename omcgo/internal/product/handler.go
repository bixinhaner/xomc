package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// CtxKeyUserID 是 gin context 里 admin middleware 写入用户 UUID 的 key。
// 与 internal/admin/middleware.go CtxKeyUserID 同字面量,在此重声明避免反向 import。
const CtxKeyUserID = "user_id"

// rematchLockTTL 是 rematch 任务 Redis 锁的兜底 TTL。
// goroutine 正常完成会主动 DEL key;异常 crash 时 TTL 自动释放,防止管理员永久卡死。
// 10 分钟足够覆盖 1000 设备 rematch + 一些保险冗余(实测 60ms / 1000 设备)。
const rematchLockTTL = 10 * time.Minute

// rematchLockKey 构造 per-admin rematch 锁的 Redis key。
func rematchLockKey(userID uuid.UUID) string {
	return "product:rematch:lock:" + userID.String()
}

// rematchAllLockKey 是「正则变更后全量重匹配」的全局锁 key（全局唯一，防多次变更并发重匹配；
// 已在跑则跳过本次，当前任务用最新 registry 兜底）。
const rematchAllLockKey = "product:rematch-all:lock"

// Handler 暴露 /api/v1/products/* (设计 §4.4)。
//
// 写路径完成后调用 Registry.Refresh 同步路由缓存；删除产品时 product_class_patterns
// 与 discovered_param_mappings 由 FK CASCADE 自动清理（设计 §4.2）。
//
// DiscoveredCleaner 由 provider 注入 parammodel.PgRepository.DeleteDiscoveredAll 来
// 实现 reset 端点（避免 product 反向 import parammodel 包）。
type Handler struct {
	repo        *PgRepository
	registry    *Registry
	cleaner     DiscoveredCleaner
	rematcher   Rematcher
	reloader    Reloader
	deviceCache DeviceCacheInvalidator // T-0176-PR-D：BindOrphan 后失效 SN cache（nil = 禁用）
	logger      *zap.Logger
	// routeInvalidator 在 product KPI 路由字段实际变化后推进 KPI route version。
	// nil 仅用于未接完整 provider 的单元测试/降级场景。
	routeInvalidator RouteInvalidator
	// 2026-05-28 异步 rematch per-admin 锁(优先 Redis,兜底 sync.Map)。
	//
	// 设计:
	//   - SetRematchRedis 注入 redis.UniversalClient → SET NX EX 跨进程互斥
	//   - Redis 未注入(单测 / fallback) → sync.Map 进程内排重
	//
	// 业务码语义:首次成功 → "accepted"(异步派发) / 已被锁住 → "running"(等刷新)。
	// 兜底 TTL 10 分钟,防止 goroutine panic 后死锁。
	rematchRedis    redis.UniversalClient
	inflightRematch sync.Map
}

// DeviceCacheInvalidator 是 product handler 写设备绑定字段后清 SN 缓存的最小依赖
// （T-0176-PR-D）。生产由 *device.DeviceCache 满足；写在消费者侧避免 product → device
// 反向依赖。
//
// Delete 失败仅记录（实现内部 warn log），不影响 BindOrphan 主返回值 —— stale
// cache 的代价是下次读到旧值，远比 BindOrphan 失败更可接受。
type DeviceCacheInvalidator interface {
	Delete(ctx context.Context, sn string)
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

// SetDeviceCacheInvalidator 注入 device cache 清理器（T-0176-PR-D）。
// nil 表示禁用 — BindOrphan / RematchOrphan 写完 DB 后不清 cache。
func (h *Handler) SetDeviceCacheInvalidator(c DeviceCacheInvalidator) {
	h.deviceCache = c
}

// SetRouteInvalidator 注入 KPI route 失效器。
func (h *Handler) SetRouteInvalidator(invalidator RouteInvalidator) {
	h.routeInvalidator = invalidator
}

// SetRematchRedis 注入 Redis 客户端用于 per-admin rematch 锁(跨进程互斥)。
// nil 时退化为 sync.Map 进程内排重(开发/单测场景)。
func (h *Handler) SetRematchRedis(rc redis.UniversalClient) {
	h.rematchRedis = rc
}

// invalidateDeviceCacheBySN 是 BindOrphan / RematchOrphan 写完 product_id 后清
// SN cache 的统一入口（T-0176-PR-D）。deviceCache 未注入时 noop；sn 为空也 noop。
// 任何错误均 silent — admin 不应因 cache 维护操作失败收到 5xx。
func (h *Handler) invalidateDeviceCacheBySN(ctx context.Context, sn string) {
	if h.deviceCache == nil || sn == "" {
		return
	}
	h.deviceCache.Delete(ctx, sn)
}

// RegisterRoutes 挂载 /api/v1/products/* 到给定 RouterGroup。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	h.RegisterReadRoutes(rg)
	h.RegisterWriteRoutes(rg)
}

// RegisterReadRoutes 挂载产品中心的只读接口，供已登录用户读取基础目录数据。
func (h *Handler) RegisterReadRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/products")
	// 集中操作（无 :id）放最前以避免被 :id 路由吞掉
	g.GET("", h.List)
	g.GET("/match", h.Match)
	g.GET("/match-order", h.MatchOrder)
	g.GET("/orphan-devices", h.ListOrphan)
	// 枚举字典（产品表单下拉框用）
	g.GET("/indicator-platforms", h.ListIndicatorPlatforms)
	g.GET("/alarm-ne-types", h.ListAlarmNeTypes)

	// 单产品
	g.GET("/:id", h.Get)
}

// RegisterWriteRoutes 挂载产品中心的写接口，仅供超管管理。
func (h *Handler) RegisterWriteRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/products")
	g.POST("", h.Create)
	g.POST("/orphan-devices/rematch", h.RematchOrphan)
	g.PUT("/orphan-devices/:deviceId/bind", h.BindOrphan)
	g.POST("/cache/refresh", h.CacheRefresh)
	g.POST("/import-directory", h.ImportDirectory)
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
	ParamModelName      string         `json:"param_model_name,omitempty"` // 反查 param_models.name，列表展示「参数模型库名称」
	IndicatorDeviceType string         `json:"indicator_device_type"`
	IndicatorPlatform   string         `json:"indicator_platform"`
	AlarmNeType         string         `json:"alarm_ne_type"`
	EnableFileType11    bool           `json:"enable_filetype11"`
	DeviceAttrsOverride map[string]any `json:"device_attrs_override"`
	EnableUnknownAlarm  bool           `json:"enable_unknown_alarm"`
	DeviceCount         int            `json:"device_count,omitempty"`
	IsBuiltin           bool           `json:"is_builtin"` // true=products.xml 装配的内置产品，前端禁止删除
	Patterns            []string       `json:"patterns"`   // 该产品 active 正则（按 sort_order 升序），无则空数组
}

func toProductView(p *Product, deviceCount int, patterns []string) productView {
	if patterns == nil {
		patterns = []string{}
	}
	return productView{
		ID: p.ID, Name: p.Name, Vendor: p.Vendor, Tech: p.Tech, RadioModes: p.RadioModes,
		Description: p.Description, ParamModelID: p.ParamModelID,
		IndicatorDeviceType: p.IndicatorDeviceType, IndicatorPlatform: p.IndicatorPlatform,
		AlarmNeType:         p.AlarmNeType,
		EnableFileType11:    p.EnableFileType11,
		DeviceAttrsOverride: p.DeviceAttrsOverride,
		EnableUnknownAlarm:  p.EnableUnknownAlarm,
		DeviceCount:         deviceCount,
		IsBuiltin:           p.IsBuiltin,
		Patterns:            patterns,
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
	patternsByProduct, err := h.repo.ListPatternsAllByProduct(c.Request.Context())
	if err != nil {
		// 非致命：列表照常返回，只是 patterns 缺
		h.logger.Warn("ListPatternsAllByProduct failed", zap.Error(err))
		patternsByProduct = map[uuid.UUID][]string{}
	}
	// 批量反查 param_model 名称（列表展示「参数模型库名称」），一次查全表避免逐行 N+1。
	paramModelNames, err := h.repo.ListParamModelNames(c.Request.Context())
	if err != nil {
		// 非致命：列表照常返回，只是 param_model_name 缺
		h.logger.Warn("ListParamModelNames failed", zap.Error(err))
		paramModelNames = map[uuid.UUID]string{}
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
		view := toProductView(p, counts[p.ID], patternsByProduct[p.ID])
		if p.ParamModelID != nil {
			view.ParamModelName = paramModelNames[*p.ParamModelID]
		}
		views = append(views, view)
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
	pcStrings := make([]string, 0, len(patterns))
	for _, pv := range patterns {
		if pv.IsActive {
			pcStrings = append(pcStrings, pv.ProductClass)
		}
	}
	response.OK(c, gin.H{
		"product":  toProductView(p, counts[id], pcStrings),
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
	// IndicatorDeviceType 非必填：核心网等非无线产品不区分制式、无 KPI 指标库，传空串。
	// 入库前小写归一；CHECK 仅允许 ''/enb/gsm/gnb。空串时 KPI 路由器软返回空路由。
	IndicatorDeviceType string         `json:"indicator_device_type"`
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
	response.OKWithStatus(c, http.StatusCreated, toProductView(p, 0, in.Patterns))
}

type updateProductReq struct {
	Name                *string        `json:"name"`
	Vendor              *string        `json:"vendor"`
	Tech                *string        `json:"tech"`
	RadioModes          *string        `json:"radio_modes"`
	Description         *string        `json:"description"`
	ParamModelName      *string        `json:"param_model_name"`
	ParamModelID        *string        `json:"param_model_id"`
	ClearParamModel     bool           `json:"clear_param_model"`
	IndicatorDeviceType *string        `json:"indicator_device_type"`
	IndicatorPlatform   *string        `json:"indicator_platform"`
	AlarmNeType         *string        `json:"alarm_ne_type"`
	EnableFileType11    *bool          `json:"enable_filetype11"`
	DeviceAttrsOverride map[string]any `json:"device_attrs_override"`
	EnableUnknownAlarm  *bool          `json:"enable_unknown_alarm"`
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
	before, err := h.repo.GetProductByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if before == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	p, err := h.repo.UpdateProduct(c.Request.Context(), id, in)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAsync(c.Request.Context(), "update-product")
	if productRouteFieldsChanged(before, p) {
		h.invalidateRouteCache(c.Request.Context(), RouteInvalidationTriggerProductWrite)
	}
	// Update 不动 patterns，回读一次以便返回值与 List 视图保持一致
	curPatterns, _ := h.repo.ListPatternsByProduct(c.Request.Context(), id)
	pcStrings := make([]string, 0, len(curPatterns))
	for _, pv := range curPatterns {
		if pv.IsActive {
			pcStrings = append(pcStrings, pv.ProductClass)
		}
	}
	response.OK(c, toProductView(p, 0, pcStrings))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("id not a uuid"))
		return
	}
	// 内置产品（products.xml 装配）禁止删除——前端置灰删除按钮，后端兜底守门。
	existing, err := h.repo.GetProductByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if existing == nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	if existing.IsBuiltin {
		commonerrors.AbortWithError(c, http.StatusForbidden, fmt.Errorf("内置产品不允许删除"))
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
	h.refreshAndRematch(c.Request.Context(), "delete-product")
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
		"deleted_rows":  deleted,
		"product_id":    id,
		"bound_devices": counts[id],
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
	h.refreshAndRematch(c.Request.Context(), "create-pattern")
	response.OKWithStatus(c, http.StatusCreated, pv)
}

// guardPatternEditable 在改/删/移正则前校验来源：内置(source='builtin',来自
// products.xml)正则 UI 只读 → 403；不存在 → 404。返回 true 表示可继续(custom 行)。
func (h *Handler) guardPatternEditable(c *gin.Context, patternID uuid.UUID) bool {
	pv, err := h.repo.GetPatternByID(c.Request.Context(), patternID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
			return false
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return false
	}
	if !isCustomPattern(pv.Source) {
		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("内置正则不允许修改/删除/移动(来自 products.xml,只能改自定义正则)[code=%d]",
				global.ErrCodeProductPatternBuiltinReadonly))
		return false
	}
	return true
}

type updatePatternReq struct {
	ProductClass *string `json:"product_class,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
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
	if req.ProductClass == nil && req.IsActive == nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("at least one of product_class / is_active is required"))
		return
	}
	if !h.guardPatternEditable(c, patternID) {
		return
	}
	ctx := c.Request.Context()
	var pv *PatternView
	if req.ProductClass != nil {
		pv, err = h.repo.UpdatePattern(ctx, patternID, *req.ProductClass)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
				return
			}
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
	}
	if req.IsActive != nil {
		pv, err = h.repo.SetPatternActive(ctx, patternID, *req.IsActive)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
				return
			}
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
	}
	h.refreshAndRematch(ctx, "update-pattern")
	response.OK(c, pv)
}

func (h *Handler) DeletePattern(c *gin.Context) {
	patternID, err := uuid.Parse(c.Param("patternId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("patternId not a uuid"))
		return
	}
	if !h.guardPatternEditable(c, patternID) {
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
	h.refreshAndRematch(c.Request.Context(), "delete-pattern")
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
	if !h.guardPatternEditable(c, patternID) {
		return
	}
	pv, err := h.repo.MovePattern(c.Request.Context(), patternID, direction)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	h.refreshAndRematch(c.Request.Context(), "move-pattern")
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
	// ErrOrphan 不是错误：productClass 未命中任何 active pattern 是正常查询结果，
	// 返回 200 + matched=false（与 mr==nil 兜底分支同义）。其余 err 才是真正的 5xx。
	if errors.Is(err, ErrOrphan) {
		response.OK(c, gin.H{
			"matched":       false,
			"product_class": productClass,
		})
		return
	}
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
		"product":         toProductView(mr.Product, 0, nil),
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

// ListOrphan GET /products/orphan-devices?page=&page_size=&search=
//
// 2026-05-28 改造:支持 server-side 分页 + SN 模糊搜索。
//   - page (default 1) / page_size (default 50, max 1000)
//   - search: 对 serial_number / oui / product_class / manufacturer 做 ILIKE '%xxx%' 过滤
//   - 旧 limit 参数兼容:若 page_size 未传但 limit 传了,以 limit 作为 page_size。
func (h *Handler) ListOrphan(c *gin.Context) {
	page := 1
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	pageSize := 50
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	} else if v := c.Query("limit"); v != "" {
		// 兼容老前端传 limit
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	search := c.Query("search")

	orphans, total, err := h.repo.ListOrphanDevices(c.Request.Context(), page, pageSize, search)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{
		"items":     orphans,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// RematchOrphan POST /products/orphan-devices/rematch
//
// 2026-05-28 防止反复刷新:per-admin 锁优先用 Redis SET NX EX(跨进程互斥),
// 未注入 Redis 时回落 sync.Map 进程内排重。统一返 200 + 业务 status 区分:
//   - "accepted" → 锁获取成功,goroutine 已派发,前端 toast "正在后台执行,请稍等"
//   - "running"  → 锁被占用,有任务在跑,前端 toast "正在执行中,请等待刷新完成"
//
// 不使用 HTTP 409 — running 是"业务上正常的等待态",不是错误,axios 也不会进 catch。
func (h *Handler) RematchOrphan(c *gin.Context) {
	userIDVal, exists := c.Get(CtxKeyUserID)
	if !exists {
		commonerrors.AbortWithError(c, http.StatusUnauthorized, commonerrors.ErrUnauthorized)
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		commonerrors.AbortWithError(c, http.StatusInternalServerError,
			fmt.Errorf("user_id in context is not uuid.UUID"))
		return
	}

	acquired, err := h.acquireRematchLock(c.Request.Context(), userID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !acquired {
		response.OK(c, gin.H{
			"status":  "running",
			"message": "rematch already running for this admin",
		})
		return
	}

	// 异步执行 — context.Background 避免 HTTP 关闭后被取消。
	go h.runRematchAsync(userID)

	response.OK(c, gin.H{
		"status":  "accepted",
		"message": "rematch started in background",
	})
}

// acquireRematchLock 尝试为当前 admin 抢一把 rematch 锁。
//
//	返回 (true, nil)  → 拿到锁,可以派发 goroutine
//	返回 (false, nil) → 锁被占用(已有 in-flight 任务,前端应提示等待)
//	返回 (_, err)     → Redis 故障,handler 应 500
//
// 优先 Redis SET NX EX,Redis 未注入(nil)时回落 sync.Map。
// Redis 用 TTL 兜底防 goroutine crash 后死锁;sync.Map 路径由 defer Delete 释放。
func (h *Handler) acquireRematchLock(ctx context.Context, userID uuid.UUID) (bool, error) {
	if h.rematchRedis != nil {
		ok, err := h.rematchRedis.SetNX(ctx, rematchLockKey(userID),
			time.Now().UTC().Format(time.RFC3339), rematchLockTTL).Result()
		if err != nil {
			return false, fmt.Errorf("redis SETNX rematch lock: %w", err)
		}
		return ok, nil
	}
	_, loaded := h.inflightRematch.LoadOrStore(userID, time.Now())
	return !loaded, nil
}

// releaseRematchLock goroutine 完成后释放锁。Redis DEL / sync.Map Delete 均
// 静默失败 — 锁本身有 TTL 兜底,DEL 失败最多让用户多等 10 分钟。
func (h *Handler) releaseRematchLock(userID uuid.UUID) {
	if h.rematchRedis != nil {
		if err := h.rematchRedis.Del(context.Background(), rematchLockKey(userID)).Err(); err != nil {
			h.logger.Warn("redis DEL rematch lock failed (TTL will release)",
				zap.String("admin_user_id", userID.String()),
				zap.Error(err))
		}
		return
	}
	h.inflightRematch.Delete(userID)
}

// runRematchAsync 在独立 goroutine 跑 rematch,完成后释放锁。
// 任何错误只写日志,不向上抛(没有调用方等结果)。
func (h *Handler) runRematchAsync(userID uuid.UUID) {
	startedAt := time.Now()
	defer func() {
		h.releaseRematchLock(userID)
		h.logger.Info("rematch finished",
			zap.String("admin_user_id", userID.String()),
			zap.Duration("duration", time.Since(startedAt)))
	}()
	// recover 任何 panic 避免拖垮整个 app
	defer func() {
		if p := recover(); p != nil {
			h.logger.Error("rematch panic",
				zap.String("admin_user_id", userID.String()),
				zap.Any("panic", p))
		}
	}()

	ctx := context.Background()

	if h.rematcher != nil {
		n, err := h.rematcher.RematchOrphans(ctx)
		if err != nil {
			h.logger.Error("rematch (injected) failed",
				zap.String("admin_user_id", userID.String()),
				zap.Error(err))
			return
		}
		h.logger.Info("rematch (injected) ok",
			zap.String("admin_user_id", userID.String()),
			zap.Int("rebound", n))
		return
	}

	if h.registry == nil {
		h.logger.Warn("rematch skipped: registry not wired",
			zap.String("admin_user_id", userID.String()))
		return
	}

	// 内置 fallback:本端按 Registry.MatchProductClass 重算。
	orphans, _, err := h.repo.ListOrphanDevices(ctx, 1, 1000, "")
	if err != nil {
		h.logger.Error("rematch list orphans failed",
			zap.String("admin_user_id", userID.String()),
			zap.Error(err))
		return
	}
	rebound := 0
	for _, d := range orphans {
		if d.ProductClass == "" {
			continue
		}
		mr, err := h.registry.MatchProductClass(ctx, d.ProductClass)
		if err != nil || mr == nil || mr.Product == nil {
			continue
		}
		if err := h.repo.BindOrphanDevice(ctx, d.ID, mr.Product.ID); err == nil {
			rebound++
			h.invalidateDeviceCacheBySN(ctx, d.SerialNumber)
		}
	}
	h.logger.Info("rematch (fallback) ok",
		zap.String("admin_user_id", userID.String()),
		zap.Int("scanned", len(orphans)),
		zap.Int("rebound", rebound))
}

// ── 正则变更后全量重匹配（异步，避免 API 超时）─────────────────────────────────

// refreshAndRematch 正则写路径统一收口：同步刷新 registry（让后续路由立即用新正则）+ 异步
// 全量重匹配存量设备 product_id（在独立 goroutine 跑，HTTP 立即返回，避免大表 rematch 超时）。
func (h *Handler) refreshAndRematch(ctx context.Context, op string) {
	h.refreshAsync(ctx, op)
	h.triggerRematchAll(ctx, op)
}

// triggerRematchAll 抢全局锁后派发后台全量重匹配 goroutine；锁被占用（已有任务在跑）则跳过。
func (h *Handler) triggerRematchAll(ctx context.Context, op string) {
	if h.registry == nil {
		return
	}
	acquired, err := h.acquireRematchAllLock(ctx)
	if err != nil {
		h.logger.Warn("acquire rematch-all lock failed", zap.String("op", op), zap.Error(err))
		return
	}
	if !acquired {
		h.logger.Info("rematch-all already running, skip dispatch", zap.String("op", op))
		return
	}
	go h.runRematchAllAsync(op)
}

func (h *Handler) acquireRematchAllLock(ctx context.Context) (bool, error) {
	if h.rematchRedis != nil {
		ok, err := h.rematchRedis.SetNX(ctx, rematchAllLockKey,
			time.Now().UTC().Format(time.RFC3339), rematchLockTTL).Result()
		if err != nil {
			return false, fmt.Errorf("redis SETNX rematch-all lock: %w", err)
		}
		return ok, nil
	}
	_, loaded := h.inflightRematch.LoadOrStore(rematchAllLockKey, time.Now())
	return !loaded, nil
}

func (h *Handler) releaseRematchAllLock() {
	if h.rematchRedis != nil {
		if err := h.rematchRedis.Del(context.Background(), rematchAllLockKey).Err(); err != nil {
			h.logger.Warn("redis DEL rematch-all lock failed (TTL will release)", zap.Error(err))
		}
		return
	}
	h.inflightRematch.Delete(rematchAllLockKey)
}

// runRematchAllAsync 全量遍历存量设备，按最新 registry 重算 product_id / param_model_id，
// 仅在变化时回写（含降级为孤儿）。keyset 分页扛大表；任何错误只记日志不上抛。
func (h *Handler) runRematchAllAsync(op string) {
	startedAt := time.Now()
	defer func() {
		h.releaseRematchAllLock()
		h.logger.Info("rematch-all finished",
			zap.String("op", op), zap.Duration("duration", time.Since(startedAt)))
	}()
	defer func() {
		if p := recover(); p != nil {
			h.logger.Error("rematch-all panic", zap.String("op", op), zap.Any("panic", p))
		}
	}()

	ctx := context.Background()
	// 兜底再刷一次，确保用最新 patterns（调用方已 Refresh，这里防御）。
	if err := h.registry.Refresh(ctx); err != nil {
		h.logger.Warn("rematch-all registry refresh failed", zap.Error(err))
	}

	const pageSize = 1000
	const maxPages = 2000 // 200 万设备安全上限，防异常死循环
	after := uuid.Nil
	scanned, changed := 0, 0
	for page := 0; page < maxPages; page++ {
		devices, err := h.repo.ListDeviceClassesAfter(ctx, after, pageSize)
		if err != nil {
			h.logger.Error("rematch-all list devices failed",
				zap.String("op", op), zap.Int("scanned", scanned), zap.Error(err))
			return
		}
		if len(devices) == 0 {
			break
		}
		for _, d := range devices {
			after = d.ID
			scanned++
			var newPID, newPMID *uuid.UUID
			if d.ProductClass != "" {
				if mr, mErr := h.registry.MatchProductClass(ctx, d.ProductClass); mErr == nil && mr != nil && mr.Product != nil {
					pid := mr.Product.ID
					newPID = &pid
					newPMID = mr.Product.ParamModelID
				}
			}
			if sameUUIDPtr(d.ProductID, newPID) {
				continue
			}
			if err := h.repo.SetDeviceProduct(ctx, d.ID, newPID, newPMID); err != nil {
				h.logger.Warn("rematch-all set device product failed",
					zap.String("device_id", d.ID.String()), zap.Error(err))
				continue
			}
			changed++
			h.invalidateDeviceCacheBySN(ctx, d.SerialNumber)
		}
		if len(devices) < pageSize {
			break
		}
	}
	h.logger.Info("rematch-all ok",
		zap.String("op", op), zap.Int("scanned", scanned), zap.Int("changed", changed))
}

// sameUUIDPtr 比较两个 *uuid.UUID：都 nil → true；一 nil 一非 nil → false；都非 nil → 值相等。
func sameUUIDPtr(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
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
	// T-0176-PR-D：写完 product_id 后清 SN cache，避免 DeviceCache 缓存的 product_class
	// 喂给 ProductRegistry 时仍指向旧路由（PR-C 切完 resolver 后这一步关键）。
	// SN 查不到 / Delete 失败均 silent — admin 仍能看到 bind 成功响应。
	if h.deviceCache != nil {
		if sn, snErr := h.repo.GetDeviceSerialByID(c.Request.Context(), deviceID); snErr == nil {
			h.invalidateDeviceCacheBySN(c.Request.Context(), sn)
		} else {
			h.logger.Warn("BindOrphan: lookup SN for cache invalidation failed",
				zap.String("device_id", deviceID.String()), zap.Error(snErr))
		}
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
	before, err := h.repo.ListRouteFieldsByProductName(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if err := h.reloader.ReloadOne(c.Request.Context(), "product"); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	productRegistryRefreshed := false
	if h.registry != nil {
		if err := h.registry.Refresh(c.Request.Context()); err != nil {
			h.logger.Warn("post-reload product registry refresh failed", zap.Error(err))
		} else {
			productRegistryRefreshed = true
		}
	}
	routeFieldsChanged := false
	after, err := h.repo.ListRouteFieldsByProductName(c.Request.Context())
	if err != nil {
		h.logger.Warn("list product route fields after reload failed; skip KPI route invalidation",
			zap.Error(err))
	} else {
		routeFieldsChanged = RouteFieldsMapChanged(before, after)
		if routeFieldsChanged && productRegistryRefreshed {
			h.invalidateRouteCache(c.Request.Context(), RouteInvalidationTriggerProductReload)
		}
	}
	response.OK(c, gin.H{
		"reloaded":                   "product",
		"product_registry_refreshed": productRegistryRefreshed,
		"kpi_route_invalidated":      routeFieldsChanged && productRegistryRefreshed,
	})
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
// 返回 alarm_definitions.ne_type 字典（如 ENB / GSM / GNB / OMC / EPC / EGW / CPE / UPS）。
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

func (h *Handler) invalidateRouteCache(ctx context.Context, trigger RouteInvalidationTrigger) {
	if h.routeInvalidator == nil {
		return
	}
	if err := h.routeInvalidator(ctx, trigger); err != nil {
		logRouteInvalidationFailure(ctx, h.logger, trigger, err)
	}
}

// 占位以避免 unused 警告（部分 helpers 在不同流程下被调用）
var _ = json.Marshal
