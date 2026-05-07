package topology

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// RuleHandler 设备规则 HTTP 处理器
type RuleHandler struct {
	service *DeviceRuleService
}

// NewRuleHandler 创建设备规则处理器
func NewRuleHandler(service *DeviceRuleService) *RuleHandler {
	return &RuleHandler{service: service}
}

// RegisterRoutes 注册设备规则路由
func (h *RuleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rules := rg.Group("/device-rules")
	{
		rules.GET("", h.ListRules)
		rules.GET("/next-priority", h.GetNextPriority)
		rules.POST("", h.CreateRule)
		rules.PUT("/batch-sort", h.BatchSortRules)
		rules.GET("/:id", h.GetRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
		rules.PATCH("/:id/toggle", h.ToggleRule)
		rules.POST("/:id/apply", h.ApplyRule)
		rules.GET("/:id/tasks", h.ListTasks)
		rules.GET("/:id/tasks/:taskId", h.GetTask)
	}
}

// getOperator 从上下文获取操作者用户名
func getRuleOperator(c *gin.Context) string {
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		return v.(string)
	}
	return ""
}

// ListRules 获取规则列表
// GET /api/v1/device-rules
func (h *RuleHandler) ListRules(c *gin.Context) {
	var req RuleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	resp, err := h.service.ListRules(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, resp)
}

// GetRule 获取规则详情
// GET /api/v1/device-rules/:id
func (h *RuleHandler) GetRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	rule, err := h.service.GetRule(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, rule)
}

// CreateRule 创建规则
// POST /api/v1/device-rules
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operator := getRuleOperator(c)
	rule, err := h.service.CreateRule(c.Request.Context(), req, operator)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, rule)
}

// UpdateRule 更新规则
// PUT /api/v1/device-rules/:id
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operator := getRuleOperator(c)
	rule, err := h.service.UpdateRule(c.Request.Context(), id, req, operator)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, rule)
}

// DeleteRule 删除规则
// DELETE /api/v1/device-rules/:id
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteRule(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithMsg(c, nil, "deleted")
}

// ToggleRule 切换规则启用状态
// PATCH /api/v1/device-rules/:id/toggle
func (h *RuleHandler) ToggleRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operator := getRuleOperator(c)
	rule, err := h.service.ToggleRule(c.Request.Context(), id, req.Enabled, operator)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, rule)
}

// BatchSortRules 批量调整规则优先级
// PUT /api/v1/device-rules/batch-sort
func (h *RuleHandler) BatchSortRules(c *gin.Context) {
	var req BatchSortRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	operator := getRuleOperator(c)
	if err := h.service.BatchSortRules(c.Request.Context(), req.Items, operator); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithMsg(c, nil, "sorted")
}

// ApplyRule 应用规则
// POST /api/v1/device-rules/:id/apply
func (h *RuleHandler) ApplyRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req ApplyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体
		req = ApplyRuleRequest{}
	}

	operator := getRuleOperator(c)
	task, err := h.service.ApplyRule(c.Request.Context(), id, req, operator)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusAccepted, task)
}

// GetTask 获取任务详情
// GET /api/v1/device-rules/:id/tasks/:taskId
func (h *RuleHandler) GetTask(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), ruleID, taskID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, task)
}

// ListTasks 获取规则的任务列表
// GET /api/v1/device-rules/:id/tasks
func (h *RuleHandler) ListTasks(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	tasks, err := h.service.ListTasks(c.Request.Context(), ruleID, limit)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, RuleTaskListResponse{
		Items: tasks,
		Total: int64(len(tasks)),
	})
}

// GetNextPriority 获取下一个可用优先级
// GET /api/v1/device-rules/next-priority
func (h *RuleHandler) GetNextPriority(c *gin.Context) {
	// 使用 repo 的 GetNextPriority 方法
	repo, ok := h.service.repo.(*PgDeviceRuleRepository)
	if !ok {
		response.OK(c, gin.H{"priority": 1})
		return
	}

	priority, err := repo.GetNextPriority(c.Request.Context())
	if err != nil {
		response.OK(c, gin.H{"priority": 1})
		return
	}

	response.OK(c, gin.H{"priority": priority})
}

// parseIntParam 解析整数参数
func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
