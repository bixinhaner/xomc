package device

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/global"
)

// RegistrationHandler provides HTTP endpoints for device pre-registration.
type RegistrationHandler struct {
	service     *RegistrationService
	permService VisibleGroupsResolver // #64 设备组数据权限：预注册列表按调用者可见分组过滤
}

// NewRegistrationHandler creates a new RegistrationHandler.
func NewRegistrationHandler(service *RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{service: service}
}

// SetPermissionService 注入数据权限解析器（#64 统一强制层）。未注入时退化为不过滤
// （dev/test），与 device / alarm 模块语义一致。
func (h *RegistrationHandler) SetPermissionService(ps VisibleGroupsResolver) {
	h.permService = ps
}

// resolveVisibleGroups 解析调用者可见设备组（三态：nil 超管 / [] 无权限 / [g...] 限定）。
// 返回 ok=false 表示解析失败已 abort（403/500），调用方应立即 return。
// permService 为 nil（dev/test）→ 返回 (nil, true) 不过滤。
func (h *RegistrationHandler) resolveVisibleGroups(c *gin.Context) (groups []uuid.UUID, ok bool) {
	if h.permService == nil {
		return nil, true
	}
	userID, _ := c.Get(admin.CtxKeyUserID)
	uid, isUUID := userID.(uuid.UUID)
	if !isUUID {
		commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
		return nil, false
	}
	isSuperVal, _ := c.Get(admin.CtxKeyIsSuperAdmin)
	isSuper, _ := isSuperVal.(bool)
	visibleGroups, err := h.permService.GetUserVisibleGroupIDs(c.Request.Context(), uid, isSuper)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return nil, false
	}
	return visibleGroups, true
}

// RegisterRoutes registers device registration routes.
func (h *RegistrationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	reg := rg.Group("/device-registrations")
	{
		reg.POST("", h.CreateRegistration)
		reg.GET("", h.ListRegistrations)
		reg.DELETE("/:id", h.DeleteRegistration)
	}
}

func getOperator(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		return v.(string)
	}
	return ""
}

// CreateRegistration handles POST /device-registrations.
func (h *RegistrationHandler) CreateRegistration(c *gin.Context) {
	var req CreateRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	reg, err := h.service.Register(c.Request.Context(), req, getOperator(c))
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, reg)
}

// ListRegistrations handles GET /device-registrations.
func (h *RegistrationHandler) ListRegistrations(c *gin.Context) {
	filter := RegistrationFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := global.RegistrationStatus(status)
		filter.Status = &s
	}
	if sn := c.Query("serial_number"); sn != "" {
		filter.SerialNumber = &sn
	}
	if carrier := c.Query("carrier"); carrier != "" {
		cc := model.CarrierCode(carrier)
		filter.Carrier = &cc
	}

	// #64 设备组数据权限：预注册列表只露调用者可见分组内的条目（未分组对非超管不可见）。
	visibleGroups, ok := h.resolveVisibleGroups(c)
	if !ok {
		return
	}
	filter.VisibleGroups = visibleGroups

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
}

// DeleteRegistration handles DELETE /device-registrations/:id.
func (h *RegistrationHandler) DeleteRegistration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
}
