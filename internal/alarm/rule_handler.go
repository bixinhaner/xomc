package alarm

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
	"go.uber.org/zap"
)

// RuleHandler provides REST API endpoints for alarm rule management.
type RuleHandler struct {
	repo   AlarmRuleRepository
	logger *zap.Logger
}

// NewRuleHandler creates a new alarm rule handler.
func NewRuleHandler(repo AlarmRuleRepository, logger *zap.Logger) *RuleHandler {
	return &RuleHandler{repo: repo, logger: logger}
}

// RegisterRoutes registers alarm rule API routes under the given router group.
func (h *RuleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rules := rg.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.GET("/:id", h.GetRule)
		rules.POST("", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}
}

type ruleQuery struct {
	Carrier       string `form:"carrier"`
	Technology    string `form:"technology"`
	Enabled       string `form:"enabled"`
	AlarmCode     string `form:"alarm_code"`
	ConditionType string `form:"condition_type"`
	model.ListRequest
}

// ListRules handles GET /rules.
func (h *RuleHandler) ListRules(c *gin.Context) {
	var q ruleQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := AlarmRuleFilter{ListRequest: q.ListRequest}
	if q.Carrier != "" {
		filter.Carrier = &q.Carrier
	}
	if q.Technology != "" {
		filter.Technology = &q.Technology
	}
	if q.Enabled != "" {
		enabled := q.Enabled == "true"
		filter.Enabled = &enabled
	}
	if q.AlarmCode != "" {
		filter.AlarmCode = &q.AlarmCode
	}
	if q.ConditionType != "" {
		filter.ConditionType = &q.ConditionType
	}

	result, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetRule handles GET /rules/:id.
func (h *RuleHandler) GetRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	rule, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, rule)
}

// CreateRule handles POST /rules.
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req CreateAlarmRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	rule := &AlarmRule{
		Name:            req.Name,
		Description:     req.Description,
		AlarmCode:       req.AlarmCode,
		Severity:        req.Severity,
		ConditionType:   req.ConditionType,
		ConditionConfig: req.ConditionConfig,
		ActionType:      req.ActionType,
		ActionConfig:    req.ActionConfig,
		Carrier:         req.Carrier,
		Technology:      req.Technology,
		Enabled:         true,
	}

	if rule.Severity == 0 {
		rule.Severity = 4
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if rule.ConditionConfig == nil {
		rule.ConditionConfig = json.RawMessage("{}")
	}
	if rule.ActionConfig == nil {
		rule.ActionConfig = json.RawMessage("{}")
	}

	if err := h.repo.Create(c.Request.Context(), rule); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// UpdateRule handles PUT /rules/:id.
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	var req UpdateAlarmRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.AlarmCode != nil {
		existing.AlarmCode = *req.AlarmCode
	}
	if req.Severity != nil {
		existing.Severity = *req.Severity
	}
	if req.ConditionType != nil {
		existing.ConditionType = *req.ConditionType
	}
	if req.ConditionConfig != nil {
		existing.ConditionConfig = *req.ConditionConfig
	}
	if req.ActionType != nil {
		existing.ActionType = *req.ActionType
	}
	if req.ActionConfig != nil {
		existing.ActionConfig = *req.ActionConfig
	}
	if req.Carrier != nil {
		existing.Carrier = *req.Carrier
	}
	if req.Technology != nil {
		existing.Technology = *req.Technology
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, existing)
}

// DeleteRule handles DELETE /rules/:id.
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alarm rule deleted"})
}
