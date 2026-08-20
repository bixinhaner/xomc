package querytemplate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/regularreport"
)

const (
	maxTemplateDeviceSNs   = 50
	maxTemplateMetricPaths = 50
)

// Handler 是 T-0174 指标查询模板的 REST 入口。
type Handler struct {
	repo                   Repository
	enabledMetricValidator *EnabledMetricPayloadService
	deviceScopeValidator   *DeviceScopeValidator
	logger                 *zap.Logger
}

// NewHandler 构造 Handler。
func NewHandler(repo Repository, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{repo: repo, logger: logger.Named("pm.querytemplate")}
}

func (h *Handler) WithEnabledMetricPayloadService(svc *EnabledMetricPayloadService) *Handler {
	h.enabledMetricValidator = svc
	return h
}

func (h *Handler) WithDeviceScopeValidator(validator *DeviceScopeValidator) *Handler {
	h.deviceScopeValidator = validator
	return h
}

// RegisterRoutes 挂载到 /api/v1（由调用方决定 group）。
//
// 端点：
//
//	GET    /pm/query-templates          列表（visibility / search / page / page_size）
//	GET    /pm/query-templates/:id      详情
//	POST   /pm/query-templates          创建
//	PATCH  /pm/query-templates/:id      更新（name / description / payload / visibility）
//	DELETE /pm/query-templates/:id      删除
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/pm/query-templates")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// ─────────────────────────────────────────────────────────────
// DTO
// ─────────────────────────────────────────────────────────────

type templateResponseDTO struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Visibility  Visibility      `json:"visibility"`
	CreatorID   uuid.UUID       `json:"creator_id"`
	Description string          `json:"description,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

func toDTO(t *Template) templateResponseDTO {
	payload := json.RawMessage(t.Payload)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	return templateResponseDTO{
		ID:          t.ID,
		Name:        t.Name,
		Visibility:  t.Visibility,
		CreatorID:   t.CreatorID,
		Description: t.Description,
		Payload:     payload,
		CreatedAt:   t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

type createRequestDTO struct {
	Name        string          `json:"name" binding:"required,max=128"`
	Visibility  string          `json:"visibility" binding:"required,oneof=public private"`
	Description string          `json:"description"`
	Payload     json.RawMessage `json:"payload"`
}

type updateRequestDTO struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Payload     json.RawMessage `json:"payload"`
	Visibility  *string         `json:"visibility"`
}

// ─────────────────────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────────────────────

// List GET /pm/query-templates
func (h *Handler) List(c *gin.Context) {
	callerID, isSuperAdmin := callerInfo(c)
	filter := ListFilter{
		CallerID:           callerID,
		CallerIsSuperAdmin: isSuperAdmin,
		Search:             c.Query("search"),
	}
	if v := c.Query("visibility"); v != "" {
		if !ValidVisibility(v) {
			response.Fail(c, http.StatusBadRequest, "invalid visibility")
			return
		}
		vis := Visibility(v)
		filter.Visibility = &vis
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.PageSize = n
		}
	}

	items, total, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	dtos := make([]templateResponseDTO, 0, len(items))
	for i := range items {
		dtos = append(dtos, toDTO(&items[i]))
	}
	response.OK(c, gin.H{"items": dtos, "total": total})
}

// Get GET /pm/query-templates/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "template not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	callerID, isSuperAdmin := callerInfo(c)
	if !canRead(t, callerID, isSuperAdmin) {
		response.Fail(c, http.StatusForbidden, "no permission to read this template")
		return
	}
	response.OK(c, toDTO(t))
}

// Create POST /pm/query-templates
func (h *Handler) Create(c *gin.Context) {
	var dto createRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	callerID, isSuperAdmin := callerInfo(c)
	if callerID == uuid.Nil {
		response.Fail(c, http.StatusUnauthorized, "login required")
		return
	}
	visibility := Visibility(dto.Visibility)
	if visibility == VisibilityPublic && !isSuperAdmin {
		response.Fail(c, http.StatusForbidden, "only super_admin can create public templates")
		return
	}
	payload := []byte(dto.Payload)
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	if err := validatePayloadLimits(payload); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.enabledMetricValidator.ValidatePayload(c.Request.Context(), payload); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	if err := h.validateDeviceScope(c, payload, callerID, isSuperAdmin); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	req := CreateRequest{
		Name:        dto.Name,
		Visibility:  visibility,
		CreatorID:   callerID,
		Description: dto.Description,
		Payload:     payload,
	}
	id, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			response.Fail(c, http.StatusConflict, "name already used by current user")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, toDTO(t))
}

// Update PATCH /pm/query-templates/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var dto updateRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	callerID, isSuperAdmin := callerInfo(c)
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "template not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !canWrite(t, callerID, isSuperAdmin) {
		response.Fail(c, http.StatusForbidden, "no permission to update this template")
		return
	}
	req := UpdateRequest{Name: dto.Name, Description: dto.Description}
	if len(dto.Payload) > 0 {
		if err := validatePayloadLimits(dto.Payload); err != nil {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.enabledMetricValidator.ValidatePayload(c.Request.Context(), dto.Payload); err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return
		}
		if err := h.validateDeviceScope(c, dto.Payload, callerID, isSuperAdmin); err != nil {
			commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
			return
		}
		req.Payload = []byte(dto.Payload)
	}
	if dto.Visibility != nil {
		v := *dto.Visibility
		if !ValidVisibility(v) {
			response.Fail(c, http.StatusBadRequest, "invalid visibility")
			return
		}
		newVis := Visibility(v)
		// 升级到 public 需 super_admin
		if newVis == VisibilityPublic && !isSuperAdmin {
			response.Fail(c, http.StatusForbidden, "only super_admin can change visibility to public")
			return
		}
		req.Visibility = &newVis
	}
	if err := h.repo.Update(c.Request.Context(), id, req); err != nil {
		if errors.Is(err, ErrDuplicate) {
			response.Fail(c, http.StatusConflict, "name already used by current user")
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "template not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	got, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, toDTO(got))
}

func (h *Handler) validateDeviceScope(c *gin.Context, payload []byte, callerID uuid.UUID, isSuperAdmin bool) error {
	if h.deviceScopeValidator == nil {
		return nil
	}
	return h.deviceScopeValidator.ValidatePayload(c.Request.Context(), payload, callerID, isSuperAdmin)
}

// Delete DELETE /pm/query-templates/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	callerID, isSuperAdmin := callerInfo(c)
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "template not found")
			return
		}
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	if !canWrite(t, callerID, isSuperAdmin) {
		response.Fail(c, http.StatusForbidden, "no permission to delete this template")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": id})
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func callerInfo(c *gin.Context) (uuid.UUID, bool) {
	var callerID uuid.UUID
	if v, ok := c.Get(admin.CtxKeyUserID); ok {
		if u, ok := v.(uuid.UUID); ok {
			callerID = u
		}
	}
	var isSuper bool
	if v, ok := c.Get(admin.CtxKeyIsSuperAdmin); ok {
		if b, ok := v.(bool); ok {
			isSuper = b
		}
	}
	return callerID, isSuper
}

func canRead(t *Template, caller uuid.UUID, isSuperAdmin bool) bool {
	if isSuperAdmin {
		return true
	}
	if t.Visibility == VisibilityPublic {
		return true
	}
	return t.CreatorID == caller
}

func canWrite(t *Template, caller uuid.UUID, isSuperAdmin bool) bool {
	if isSuperAdmin {
		return true
	}
	return t.CreatorID == caller
}

func validatePayloadLimits(payload []byte) error {
	count, err := countPayloadStringArray(payload, "device_sns")
	if err != nil {
		return err
	}
	if count > maxTemplateDeviceSNs {
		return fmt.Errorf("device_sns exceeds maximum of %d", maxTemplateDeviceSNs)
	}
	count, err = countPayloadStringArray(payload, "metric_paths")
	if err != nil {
		return err
	}
	if count > maxTemplateMetricPaths {
		return fmt.Errorf("metric_paths exceeds maximum of %d", maxTemplateMetricPaths)
	}
	if err := regularreport.ValidatePayload(payload); err != nil {
		return err
	}
	return nil
}

func countPayloadStringArray(payload []byte, key string) (int, error) {
	values, err := payloadStringArray(payload, key)
	if err != nil {
		return 0, err
	}
	return len(values), nil
}

func payloadStringArray(payload []byte, key string) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}
	field, ok := raw[key]
	if !ok || len(field) == 0 || string(field) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(field, &values); err != nil {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
	return values, nil
}

func payloadString(payload []byte, key string) (string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return "", fmt.Errorf("invalid payload")
	}
	field, ok := raw[key]
	if !ok || len(field) == 0 || string(field) == "null" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(field, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return value, nil
}
