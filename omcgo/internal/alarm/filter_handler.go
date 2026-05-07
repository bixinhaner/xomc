package alarm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"go.uber.org/zap"
)

// FilterHandler 告警过滤规则 HTTP 处理器。
type FilterHandler struct {
	repo   AlarmFilterRuleRepository
	logger *zap.Logger
}

// NewFilterHandler 创建过滤规则 Handler。
func NewFilterHandler(repo AlarmFilterRuleRepository, logger *zap.Logger) *FilterHandler {
	return &FilterHandler{repo: repo, logger: logger}
}

// RegisterRoutes 注册过滤规则路由。
func (h *FilterHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.GetByID)
	rg.POST("", h.Create)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/toggle", h.Toggle)
}

type filterRuleQuery struct {
	FilterType string `form:"filter_type"`
	Action     string `form:"action"`
	Enabled    string `form:"enabled"`
	model.ListRequest
}

func (h *FilterHandler) List(c *gin.Context) {
	var q filterRuleQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := AlarmFilterRuleFilter{ListRequest: q.ListRequest}
	if q.FilterType != "" {
		filter.FilterType = &q.FilterType
	}
	if q.Action != "" {
		filter.Action = &q.Action
	}
	if q.Enabled != "" {
		enabled := q.Enabled == "true"
		filter.Enabled = &enabled
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *FilterHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	rule, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	response.OK(c, rule)
}

func (h *FilterHandler) Create(c *gin.Context) {
	var req CreateAlarmFilterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	rule := &AlarmFilterRule{
		ID:              uuid.New(),
		Name:            req.Name,
		FilterType:      req.FilterType,
		AlarmSources:    req.AlarmSources,
		AlarmIdentifiers:      req.AlarmIdentifiers,
		DeviceIDs:       req.DeviceIDs,
		DeviceGroupIDs:  req.DeviceGroupIDs,
		Action:          req.Action,
		AcknowledgeDesc: req.AcknowledgeDesc,
		WebhookURL:      req.WebhookURL,
		WebhookSecret:   req.WebhookSecret,
		Priority:        req.Priority,
		Enabled:         true,
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if rule.AlarmSources == nil {
		rule.AlarmSources = []string{}
	}
	if rule.AlarmIdentifiers == nil {
		rule.AlarmIdentifiers = []string{}
	}
	if rule.DeviceIDs == nil {
		rule.DeviceIDs = []uuid.UUID{}
	}
	if rule.DeviceGroupIDs == nil {
		rule.DeviceGroupIDs = []uuid.UUID{}
	}

	if err := h.repo.Create(c.Request.Context(), rule); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, rule)
}

func (h *FilterHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	rule, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
		return
	}
	var req UpdateAlarmFilterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.FilterType != nil {
		rule.FilterType = *req.FilterType
	}
	if req.AlarmSources != nil {
		rule.AlarmSources = req.AlarmSources
	}
	if req.AlarmIdentifiers != nil {
		rule.AlarmIdentifiers = req.AlarmIdentifiers
	}
	if req.DeviceIDs != nil {
		rule.DeviceIDs = req.DeviceIDs
	}
	if req.DeviceGroupIDs != nil {
		rule.DeviceGroupIDs = req.DeviceGroupIDs
	}
	if req.Action != nil {
		rule.Action = *req.Action
	}
	if req.AcknowledgeDesc != nil {
		rule.AcknowledgeDesc = *req.AcknowledgeDesc
	}
	if req.WebhookURL != nil {
		rule.WebhookURL = req.WebhookURL
	}
	if req.WebhookSecret != nil {
		rule.WebhookSecret = req.WebhookSecret
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.repo.Update(c.Request.Context(), rule); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, rule)
}

func (h *FilterHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, nil, "filter rule deleted")
}

func (h *FilterHandler) Toggle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	if err := h.repo.Toggle(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithMsg(c, nil, "filter rule toggled")
}
