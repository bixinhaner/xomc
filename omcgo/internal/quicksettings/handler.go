package quicksettings

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// DeviceLookup 抽象 device 查询(满足 *device.DeviceService.GetDevice)。
type DeviceLookup interface {
	GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error)
}

// ProductClassMatcher 抽象 productClass → product 路由(满足 *product.Registry.MatchProductClass)。
type ProductClassMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// ParamModelNameLookup 抽象 paramModelID → name 反查(满足 *product.PgRepository.LookupParamModelNameByID)。
type ParamModelNameLookup interface {
	LookupParamModelNameByID(ctx context.Context, id uuid.UUID) (string, error)
}

// Handler 暴露「快速设置」分组元数据 REST 端点(T-0138)。
//
// 解析链路:device_id → DeviceLookup.GetDevice → productRegistry.MatchProductClass
//
//	→ product.ParamModelID → productRepo.LookupParamModelNameByID
//	→ quicksettings.Registry.GetByParamModel。
type Handler struct {
	registry        *Registry
	deviceSvc       DeviceLookup
	productRegistry ProductClassMatcher
	productRepo     ParamModelNameLookup
}

// NewHandler 构造 Handler。任一依赖为 nil 时,路由层应跳过 Register(避免运行时 panic)。
func NewHandler(
	registry *Registry,
	deviceSvc DeviceLookup,
	productRegistry ProductClassMatcher,
	productRepo ParamModelNameLookup,
) *Handler {
	return &Handler{
		registry:        registry,
		deviceSvc:       deviceSvc,
		productRegistry: productRegistry,
		productRepo:     productRepo,
	}
}

// RegisterRoutes 注册路由到给定的路由组(已含 /api/v1 前缀)。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/quicksettings/groups", h.GetGroups)
}

// GetGroups handles GET /api/v1/quicksettings/groups?device_id={uuid}.
//
// 响应:{"param_model": "<name>", "groups": [Group, ...]}
// 错误状态:
//
//	400 — 缺/非法 device_id
//	404 — 设备不存在 / productClass 未匹配任何 product
//	422 — 设备 product 未配置 paramModel
//	500 — 内部反查失败
func (h *Handler) GetGroups(c *gin.Context) {
	if paramModelName := strings.TrimSpace(c.Query("param_model")); paramModelName != "" {
		c.JSON(http.StatusOK, gin.H{
			"param_model": paramModelName,
			"groups":      h.registry.GetByParamModel(paramModelName),
		})
		return
	}
	deviceIDStr := c.Query("device_id")
	if deviceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameter: device_id"})
		return
	}
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id, expected uuid", "device_id": deviceIDStr})
		return
	}

	device, err := h.deviceSvc.GetDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found", "device_id": deviceIDStr})
		return
	}
	// 设备查询带「排除已删除」过滤,软删/不存在时返回 (nil, nil)。
	// 不判空直接解引用 device.ProductClass 会触发空指针 panic → 兜成 500,前端拿 500 即空白。
	// 按本接口注释承诺的约定,查到空设备返回 404。
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found", "device_id": deviceIDStr})
		return
	}

	mr, err := h.productRegistry.MatchProductClass(c.Request.Context(), device.ProductClass)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "product_class not matched to any product",
			"product_class": device.ProductClass,
		})
		return
	}
	if mr.Product.ParamModelID == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "device's product has no param_model assigned",
			"product": device.ProductClass,
		})
		return
	}

	paramModelName, err := h.productRepo.LookupParamModelNameByID(c.Request.Context(), *mr.Product.ParamModelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "resolve param_model name: " + err.Error()})
		return
	}

	groups := h.registry.GetByParamModel(paramModelName)
	c.JSON(http.StatusOK, gin.H{
		"param_model": paramModelName,
		"groups":      groups,
	})
}
