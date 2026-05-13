package template

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// TemplateDispatcher 是 Path A 显式模板下发协作方，由 provisioning engine 实现。
// 接口在 template 包定义（消费者驱动）；engine 通过签名匹配隐式满足，不引入反向依赖。
//
// 显式 dispatch 语义与 HandleBootstrap 的 template.Match 不同：
//   - 调用方明确指定 tmpl + deviceID，不走 carrier+tech+productClass 匹配规则；
//   - 不受 config.provision.auto_configure 守门（T-0120 解锁路径）；
//   - 不走 A→B→C selector，强制走 Path A。
type TemplateDispatcher interface {
	DispatchTemplate(ctx context.Context, tmpl *ConfigTemplate, deviceID uuid.UUID) (uuid.UUID, error)
}

// Handler provides HTTP handlers for configuration template REST API.
type Handler struct {
	repo       ConfigTemplateRepository
	dispatcher TemplateDispatcher
}

// NewHandler creates a new template REST API handler.
func NewHandler(repo ConfigTemplateRepository) *Handler {
	return &Handler{repo: repo}
}

// SetDispatcher 注入显式模板下发协作方（POST /:id/dispatch 路径必需）。
// nil 时 dispatch endpoint 返 503，CRUD 五端点不受影响。
func (h *Handler) SetDispatcher(d TemplateDispatcher) { h.dispatcher = d }

// RegisterRoutes registers template routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/templates")
	{
		templates.GET("", h.List)
		templates.GET("/:id", h.Get)
		templates.POST("", h.Create)
		templates.PUT("/:id", h.Update)
		templates.DELETE("/:id", h.Delete)
		templates.POST("/:id/dispatch", h.Dispatch)
	}
}

// List handles GET /api/v1/templates.
func (h *Handler) List(c *gin.Context) {
	filter := ConfigTemplateFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if carrier := c.Query("carrier"); carrier != "" {
		filter.Carrier = model.CarrierCode(carrier)
	}
	if tech := c.Query("technology"); tech != "" {
		filter.Technology = model.Technology(tech)
	}
	if pc := c.Query("product_class"); pc != "" {
		filter.ProductClass = pc
	}
	if tt := c.Query("template_type"); tt != "" {
		filter.TemplateType = TemplateType(tt)
	}
	if active := c.Query("active"); active == "true" {
		v := true
		filter.Active = &v
	} else if active == "false" {
		v := false
		filter.Active = &v
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// Get handles GET /api/v1/templates/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid template ID")
		return
	}

	t, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "template not found")
		return
	}
	response.OK(c, t)
}

type createTemplateRequest struct {
	Name         string          `json:"name" binding:"required"`
	Carrier      string          `json:"carrier" binding:"required"`
	Technology   string          `json:"technology" binding:"required"`
	ProductClass string          `json:"product_class"`
	TemplateType string          `json:"template_type" binding:"required"`
	Parameters   json.RawMessage `json:"parameters" binding:"required"`
	Priority     int             `json:"priority"`
	Description  string          `json:"description"`
}

// Create handles POST /api/v1/templates.
func (h *Handler) Create(c *gin.Context) {
	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	t := &ConfigTemplate{
		Name:         req.Name,
		Carrier:      model.CarrierCode(req.Carrier),
		Technology:   model.Technology(req.Technology),
		ProductClass: req.ProductClass,
		TemplateType: TemplateType(req.TemplateType),
		Parameters:   req.Parameters,
		Priority:     req.Priority,
		Version:      1,
		Active:       true,
		Description:  req.Description,
	}

	if err := h.repo.Create(c.Request.Context(), t); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, t)
}

type updateTemplateRequest struct {
	Name         string          `json:"name" binding:"required"`
	Carrier      string          `json:"carrier" binding:"required"`
	Technology   string          `json:"technology" binding:"required"`
	ProductClass string          `json:"product_class"`
	TemplateType string          `json:"template_type" binding:"required"`
	Parameters   json.RawMessage `json:"parameters" binding:"required"`
	Priority     int             `json:"priority"`
	Active       bool            `json:"active"`
	Description  string          `json:"description"`
}

// Update handles PUT /api/v1/templates/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "template not found")
		return
	}

	existing.Name = req.Name
	existing.Carrier = model.CarrierCode(req.Carrier)
	existing.Technology = model.Technology(req.Technology)
	existing.ProductClass = req.ProductClass
	existing.TemplateType = TemplateType(req.TemplateType)
	existing.Parameters = req.Parameters
	existing.Priority = req.Priority
	existing.Active = req.Active
	existing.Description = req.Description

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, existing)
}

type dispatchTemplateRequest struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
}

type dispatchTemplateResult struct {
	DeviceID string `json:"device_id"`
	TaskID   string `json:"task_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

type dispatchTemplateResponse struct {
	TemplateID   string                   `json:"template_id"`
	Dispatched   []dispatchTemplateResult `json:"dispatched"`
	Failed       []dispatchTemplateResult `json:"failed"`
	TotalDevices int                      `json:"total_devices"`
}

// Dispatch handles POST /api/v1/templates/:id/dispatch (T-0120).
// 显式选模板 + 选设备 → Path A 强制下发；每设备独立成败（与 T-0102-c 同 pattern）。
func (h *Handler) Dispatch(c *gin.Context) {
	if h.dispatcher == nil {
		response.Fail(c, http.StatusServiceUnavailable, "template dispatcher not configured")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req dispatchTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if len(req.DeviceIDs) == 0 {
		response.Fail(c, http.StatusBadRequest, "device_ids must not be empty")
		return
	}

	tmpl, err := h.repo.GetByID(c.Request.Context(), templateID)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "template not found")
		return
	}

	resp := dispatchTemplateResponse{
		TemplateID:   templateID.String(),
		TotalDevices: len(req.DeviceIDs),
		Dispatched:   make([]dispatchTemplateResult, 0, len(req.DeviceIDs)),
		Failed:       make([]dispatchTemplateResult, 0),
	}

	for _, raw := range req.DeviceIDs {
		deviceID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			resp.Failed = append(resp.Failed, dispatchTemplateResult{
				DeviceID: raw,
				Error:    "invalid device_id format",
			})
			continue
		}
		taskID, dispErr := h.dispatcher.DispatchTemplate(c.Request.Context(), tmpl, deviceID)
		if dispErr != nil {
			resp.Failed = append(resp.Failed, dispatchTemplateResult{
				DeviceID: raw,
				Error:    dispErr.Error(),
			})
			continue
		}
		resp.Dispatched = append(resp.Dispatched, dispatchTemplateResult{
			DeviceID: raw,
			TaskID:   taskID.String(),
		})
	}

	status := http.StatusOK
	if len(resp.Dispatched) == 0 {
		status = http.StatusUnprocessableEntity
	} else if len(resp.Failed) > 0 {
		status = http.StatusMultiStatus
	}
	response.OKWithStatus(c, status, resp)
}

// Delete handles DELETE /api/v1/templates/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusNotFound, "template not found")
		return
	}
	response.OK(c, nil)
}
