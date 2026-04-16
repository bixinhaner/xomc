package mml

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Handler provides HTTP handlers for the MML console REST API.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new MML Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.Named("mml-handler"),
	}
}

// RegisterRoutes registers MML routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mml := rg.Group("/mml")

	commands := mml.Group("/commands")
	commands.GET("", h.ListCommands)
	commands.GET("/:id", h.GetCommand)
	commands.GET("/:id/param-paths", h.GetCommandParamPaths)

	mml.POST("/execute", h.Execute)
	mml.GET("/dangerous-check", h.DangerousCheck)

	scripts := mml.Group("/scripts")
	scripts.GET("", h.ListScripts)
	scripts.POST("", h.CreateScript)
	scripts.GET("/:id", h.GetScript)
	scripts.PUT("/:id", h.UpdateScript)
	scripts.DELETE("/:id", h.DeleteScript)

	tasks := mml.Group("/tasks")
	tasks.GET("", h.ListTasks)
	tasks.GET("/:id", h.GetTask)
	tasks.GET("/:id/results", h.GetTaskResults)
	tasks.POST("/:id/start", h.StartTask)
	tasks.POST("/:id/pause", h.PauseTask)
	tasks.POST("/:id/cancel", h.CancelTask)
	tasks.DELETE("/:id", h.DeleteTask)

	templates := mml.Group("/templates")
	templates.GET("", h.ListTemplates)
	templates.POST("", h.CreateTemplate)
	templates.GET("/:id", h.GetTemplate)
	templates.PUT("/:id", h.UpdateTemplate)
	templates.DELETE("/:id", h.DeleteTemplate)
	templates.POST("/:id/clone", h.CloneTemplate)
}

// ---- Request types ----

// ExecuteHTTPRequest defines the request body for executing an MML command.
type ExecuteHTTPRequest struct {
	CommandCode string                 `json:"command_code"`
	DeviceSNs   []string               `json:"device_sns" binding:"required"`
	Parameters  map[string]interface{} `json:"parameters"`
	TaskName    string                 `json:"task_name"`

	// Script execution support
	ScriptID string                   `json:"script_id"`
	Commands []map[string]interface{} `json:"commands"`

	// Scheduling
	ExecuteType string `json:"execute_type"`
	ScheduledAt string `json:"scheduled_at"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	PeriodTime  string `json:"period_time"`

	// Retry strategy
	OfflineRetry        bool `json:"offline_retry"`
	OfflineRetryWait    int  `json:"offline_retry_wait"`
	FailedRetry         bool `json:"failed_retry"`
	FailedRetryCount    int  `json:"failed_retry_count"`
	FailedRetryInterval int  `json:"failed_retry_interval"`

	// Parameter path command support
	ParamPaths    []string `json:"param_paths"`
	OperationType string   `json:"operation_type"`
}

// CreateScriptRequest defines the request body for creating an MML script.
type CreateScriptRequest struct {
	ScriptName  string   `json:"script_name" binding:"required"`
	Description string   `json:"description"`
	Content     string   `json:"content" binding:"required"`
	DeviceType  string   `json:"device_type"`
	Tags        []string `json:"tags"`
}

// UpdateScriptRequest defines the request body for updating an MML script.
type UpdateScriptRequest struct {
	ScriptName  string   `json:"script_name" binding:"required"`
	Description string   `json:"description"`
	Content     string   `json:"content" binding:"required"`
	DeviceType  string   `json:"device_type"`
	Tags        []string `json:"tags"`
}

// ---- Command handlers ----

// ListCommands handles GET /api/v1/mml/commands.
func (h *Handler) ListCommands(c *gin.Context) {
	filter := CommandFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if category := c.Query("category"); category != "" {
		filter.Category = &category
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	result, err := h.service.ListCommands(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetCommand handles GET /api/v1/mml/commands/:id.
func (h *Handler) GetCommand(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	cmd, err := h.service.GetCommand(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, cmd)
}

// GetCommandParamPaths handles GET /api/v1/mml/commands/:id/param-paths.
func (h *Handler) GetCommandParamPaths(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	result, err := h.service.GetCommandParamPaths(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// ---- Execute handler ----

// Execute handles POST /api/v1/mml/execute.
func (h *Handler) Execute(c *gin.Context) {
	var req ExecuteHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Extract creator from context (set by auth middleware); fallback to empty.
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	execReq := ExecuteRequest{
		CommandCode:         req.CommandCode,
		DeviceSNs:           req.DeviceSNs,
		Parameters:          req.Parameters,
		TaskName:            req.TaskName,
		Creator:             creatorStr,
		Executor:            creatorStr,
		Commands:            req.Commands,
		ExecuteType:         ExecuteType(req.ExecuteType),
		OfflineRetry:        req.OfflineRetry,
		OfflineRetryWait:    req.OfflineRetryWait,
		FailedRetry:         req.FailedRetry,
		FailedRetryCount:    req.FailedRetryCount,
		FailedRetryInterval: req.FailedRetryInterval,
		ParamPaths:          req.ParamPaths,
		OperationType:       req.OperationType,
	}
	if req.ScriptID != "" {
		execReq.ScriptID = &req.ScriptID
	}
	if req.ScheduledAt != "" {
		execReq.ScheduledAt = &req.ScheduledAt
	}
	if req.PeriodStart != "" {
		execReq.PeriodStart = &req.PeriodStart
	}
	if req.PeriodEnd != "" {
		execReq.PeriodEnd = &req.PeriodEnd
	}
	execReq.PeriodTime = req.PeriodTime

	task, err := h.service.ExecuteCommand(c.Request.Context(), execReq)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, task)
}

// ---- Script handlers ----

// ListScripts handles GET /api/v1/mml/scripts.
func (h *Handler) ListScripts(c *gin.Context) {
	filter := ScriptFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if deviceType := c.Query("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}
	if creator := c.Query("creator"); creator != "" {
		filter.Creator = &creator
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	result, err := h.service.ListScripts(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetScript handles GET /api/v1/mml/scripts/:id.
func (h *Handler) GetScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	script, err := h.service.GetScript(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, script)
}

// CreateScript handles POST /api/v1/mml/scripts.
func (h *Handler) CreateScript(c *gin.Context) {
	var req CreateScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Extract creator from context (set by auth middleware); fallback to empty.
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	script := &MMLScript{
		ScriptName:  req.ScriptName,
		Description: req.Description,
		Content:     req.Content,
		DeviceType:  req.DeviceType,
		Creator:     creatorStr,
		Tags:        req.Tags,
	}

	created, err := h.service.CreateScript(c.Request.Context(), script)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateScript handles PUT /api/v1/mml/scripts/:id.
func (h *Handler) UpdateScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	script := &MMLScript{
		ScriptName:  req.ScriptName,
		Description: req.Description,
		Content:     req.Content,
		DeviceType:  req.DeviceType,
		Tags:        req.Tags,
	}

	updated, err := h.service.UpdateScript(c.Request.Context(), id, script)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteScript handles DELETE /api/v1/mml/scripts/:id.
func (h *Handler) DeleteScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteScript(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---- Task handlers ----

// ListTasks handles GET /api/v1/mml/tasks.
func (h *Handler) ListTasks(c *gin.Context) {
	filter := TaskFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if status := c.Query("status"); status != "" {
		s := TaskStatus(status)
		filter.Status = &s
	}
	if executeType := c.Query("execute_type"); executeType != "" {
		et := ExecuteType(executeType)
		filter.ExecuteType = &et
	}
	if result := c.Query("result"); result != "" {
		r := TaskResult(result)
		filter.Result = &r
	}

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTask handles GET /api/v1/mml/tasks/:id.
func (h *Handler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, task)
}

// ---- Task control handlers (Phase 2) ----

// StartTask handles POST /api/v1/mml/tasks/:id/start.
func (h *Handler) StartTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.StartTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, task)
}

// PauseTask handles POST /api/v1/mml/tasks/:id/pause.
func (h *Handler) PauseTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.PauseTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, task)
}

// CancelTask handles POST /api/v1/mml/tasks/:id/cancel.
func (h *Handler) CancelTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	task, err := h.service.CancelTask(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask handles DELETE /api/v1/mml/tasks/:id.
func (h *Handler) DeleteTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ---- Dangerous command check ----

// DangerousCheck handles GET /api/v1/mml/dangerous-check.
func (h *Handler) DangerousCheck(c *gin.Context) {
	commandCode := c.Query("command_code")
	if commandCode == "" {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	dc, err := h.service.IsDangerousCommand(c.Request.Context(), commandCode)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dangerous": dc != nil,
		"info":      dc,
	})
}

// ---- Task result handler ----

// GetTaskResults handles GET /api/v1/mml/tasks/:id/results.
func (h *Handler) GetTaskResults(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, e := parseInt(p); e == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, e := parseInt(ps); e == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	result, err := h.service.GetTaskResults(c.Request.Context(), id, page, pageSize)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ---- Template handlers ----

// ListTemplates handles GET /api/v1/mml/templates.
func (h *Handler) ListTemplates(c *gin.Context) {
	filter := TemplateFilter{
		ListRequest: model.DefaultListRequest(),
	}

	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	if commandCode := c.Query("command_code"); commandCode != "" {
		filter.CommandCode = &commandCode
	}
	if operationType := c.Query("operation_type"); operationType != "" {
		filter.OperationType = &operationType
	}
	if templateScope := c.Query("template_scope"); templateScope != "" {
		filter.TemplateScope = &templateScope
	}
	if categoryGroup := c.Query("category_group"); categoryGroup != "" {
		filter.CategoryGroup = &categoryGroup
	}

	// Pass current user for private template filtering
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)
	filter.Creator = &creatorStr

	result, err := h.service.ListTemplates(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTemplate handles GET /api/v1/mml/templates/:id.
func (h *Handler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	tmpl, err := h.service.GetTemplate(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, tmpl)
}

// CreateTemplateRequest defines the request body for creating an MML template.
type CreateTemplateRequest struct {
	TemplateName  string                 `json:"template_name" binding:"required"`
	CommandCode   string                 `json:"command_code" binding:"required"`
	OperationType string                 `json:"operation_type" binding:"required"`
	TemplateScope string                 `json:"template_scope" binding:"required"`
	CategoryGroup string                 `json:"category_group"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
	ProductTypes  []string               `json:"product_types"`
}

// CreateTemplate handles POST /api/v1/mml/templates.
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	tmpl := &MMLTemplate{
		TemplateName:  req.TemplateName,
		CommandCode:   req.CommandCode,
		OperationType: req.OperationType,
		TemplateScope: req.TemplateScope,
		CategoryGroup: req.CategoryGroup,
		Parameters:    req.Parameters,
		ParamPaths:    req.ParamPaths,
		Description:   req.Description,
		ProductTypes:  req.ProductTypes,
		Creator:       creatorStr,
	}

	created, err := h.service.CreateTemplate(c.Request.Context(), tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateTemplateRequest defines the request body for updating an MML template.
type UpdateTemplateRequest struct {
	TemplateName  string                 `json:"template_name" binding:"required"`
	CommandCode   string                 `json:"command_code" binding:"required"`
	OperationType string                 `json:"operation_type" binding:"required"`
	TemplateScope string                 `json:"template_scope" binding:"required"`
	CategoryGroup string                 `json:"category_group"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
	ProductTypes  []string               `json:"product_types"`
}

// UpdateTemplate handles PUT /api/v1/mml/templates/:id.
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tmpl := &MMLTemplate{
		TemplateName:  req.TemplateName,
		CommandCode:   req.CommandCode,
		OperationType: req.OperationType,
		TemplateScope: req.TemplateScope,
		CategoryGroup: req.CategoryGroup,
		Parameters:    req.Parameters,
		ParamPaths:    req.ParamPaths,
		Description:   req.Description,
		ProductTypes:  req.ProductTypes,
	}

	updated, err := h.service.UpdateTemplate(c.Request.Context(), id, tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteTemplate handles DELETE /api/v1/mml/templates/:id.
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	if err := h.service.DeleteTemplate(c.Request.Context(), id, creatorStr); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CloneTemplate handles POST /api/v1/mml/templates/:id/clone.
func (h *Handler) CloneTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	cloned, err := h.service.CloneTemplate(c.Request.Context(), id, creatorStr)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	c.JSON(http.StatusCreated, cloned)
}

// parseInt is a helper to parse an int from a string.
func parseInt(s string) (int, error) {
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		v = v*10 + int(c-'0')
	}
	return v, nil
}
