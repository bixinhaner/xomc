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
	service *RegistrationService
}

// NewRegistrationHandler creates a new RegistrationHandler.
func NewRegistrationHandler(service *RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{service: service}
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
