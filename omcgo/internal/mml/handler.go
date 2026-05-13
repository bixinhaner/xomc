package mml

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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

	// 脚本任务创建走 /mml/tasks（to-do-list #7）。与 /mml/execute 共享 service
	// 层逻辑，但 URL 更语义化，便于前端区分"新建脚本任务"与"临时执行命令"。
	mml.POST("/tasks", h.CreateTask)

	scripts := mml.Group("/scripts")
	scripts.GET("", h.ListScripts)
	scripts.POST("", h.CreateScript)
	scripts.GET("/:id", h.GetScript)
	scripts.PUT("/:id", h.UpdateScript)
	scripts.DELETE("/:id", h.DeleteScript)
	scripts.POST("/:id/start", h.StartScript)
	// P4 C11: 历史执行列表，用于脚本详情页"历史执行"tab。
	scripts.GET("/:id/runs", h.ListScriptRuns)
	scripts.POST("/:id/pause", h.PauseScript)
	scripts.POST("/:id/cancel", h.CancelScript)

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

	// Sprint B Q-V3-1：mml_param_groups 批量执行。group 下的全部命令
	// 一键展开为一个 mml_task，fanout + sequencer 自动串行下发。
	groups := mml.Group("/groups")
	groups.POST("/:id/execute", h.ExecuteGroup)
}

// ---- Request types ----

// ExecuteHTTPRequest defines the request body for POST /api/v1/mml/execute
// （临时执行命令）。command_code、script_id、commands 三者必须至少有一个，
// 校验逻辑见 Execute handler；不能用 binding 标签强制 command_code，
// 否则 commands[] 形式的请求会被误拒。
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
	ParamValues   []string `json:"param_values"`
	OperationType string   `json:"operation_type"`
}

// CreateTaskHTTPRequest defines the request body for POST /api/v1/mml/tasks
// （脚本任务登记）。command_code 可选，但必须提供 script_id / commands / command_code
// 三者至少其一，否则没有任何命令可下发。字段集与 ExecuteHTTPRequest 完全一致，
// 仅 binding 规则不同；CreateTask handler 通过显式类型转换复用 runExecute。
type CreateTaskHTTPRequest struct {
	CommandCode string                 `json:"command_code"`
	DeviceSNs   []string               `json:"device_sns" binding:"required"`
	Parameters  map[string]interface{} `json:"parameters"`
	TaskName    string                 `json:"task_name"`

	ScriptID string                   `json:"script_id"`
	Commands []map[string]interface{} `json:"commands"`

	ExecuteType string `json:"execute_type"`
	ScheduledAt string `json:"scheduled_at"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	PeriodTime  string `json:"period_time"`

	OfflineRetry        bool `json:"offline_retry"`
	OfflineRetryWait    int  `json:"offline_retry_wait"`
	FailedRetry         bool `json:"failed_retry"`
	FailedRetryCount    int  `json:"failed_retry_count"`
	FailedRetryInterval int  `json:"failed_retry_interval"`

	ParamPaths    []string `json:"param_paths"`
	ParamValues   []string `json:"param_values"`
	OperationType string   `json:"operation_type"`
}

// CreateScriptRequest defines the request body for creating an MML script.
type CreateScriptRequest struct {
	ScriptName  string   `json:"script_name" binding:"required"`
	Description string   `json:"description"`
	Content     string   `json:"content" binding:"required"`
	Tags        []string `json:"tags"`
}

// UpdateScriptRequest defines the request body for updating an MML script.
type UpdateScriptRequest struct {
	ScriptName  string   `json:"script_name" binding:"required"`
	Description string   `json:"description"`
	Content     string   `json:"content" binding:"required"`
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

	response.OK(c, result)
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

	response.OK(c, cmd)
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

	response.OK(c, result)
}

// ---- Execute / Task creation handlers ----

// Execute handles POST /api/v1/mml/execute.
// 「临时执行命令」入口；command_code、script_id、commands 三者至少其一，
// 都缺则 400 + warn 日志。命令源全空意味着 mml_tasks 没有任何东西可下发。
func (h *Handler) Execute(c *gin.Context) {
	var req ExecuteHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("mml execute request validation failed",
			zap.String("client_ip", c.ClientIP()),
			zap.Error(err),
		)
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if req.CommandCode == "" && req.ScriptID == "" && len(req.Commands) == 0 && len(req.ParamPaths) == 0 {
		h.logger.Warn("mml execute rejected: no command source provided",
			zap.String("client_ip", c.ClientIP()),
			zap.String("task_name", req.TaskName),
			zap.Strings("device_sns", req.DeviceSNs),
		)
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("one of command_code, script_id, commands, param_paths is required"))
		return
	}
	h.runExecute(c, req)
}

// CreateTask handles POST /api/v1/mml/tasks（脚本任务登记）。
// command_code 不强制，但 script_id / commands / command_code 三者至少其一；
// 否则任务没有任何命令可下发，device_tasks 无法派生。
func (h *Handler) CreateTask(c *gin.Context) {
	var raw CreateTaskHTTPRequest
	if err := c.ShouldBindJSON(&raw); err != nil {
		h.logger.Warn("mml task create request validation failed",
			zap.String("client_ip", c.ClientIP()),
			zap.Error(err),
		)
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if raw.CommandCode == "" && raw.ScriptID == "" && len(raw.Commands) == 0 && len(raw.ParamPaths) == 0 {
		h.logger.Warn("mml task create rejected: no command source provided",
			zap.String("client_ip", c.ClientIP()),
			zap.String("task_name", raw.TaskName),
		)
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("one of script_id, commands, command_code, param_paths is required"))
		return
	}
	// 两个结构体字段序列与类型一致（仅 binding 标签不同），
	// Go 规范允许在「忽略 tag」意义下显式转换。
	h.runExecute(c, ExecuteHTTPRequest(raw))
}

// runExecute 封装 Execute / CreateTask 共用的归一化、映射到 service 层
// ExecuteRequest 以及下发流程。
func (h *Handler) runExecute(c *gin.Context, req ExecuteHTTPRequest) {
	// 归一化 parameters：前端未传或传 null 时，Gin 反序列化得到 nil map。
	// 置为空 map，避免下游 json.Marshal(nil) 产出 "null" 写进 device_tasks.params。
	if req.Parameters == nil {
		req.Parameters = map[string]interface{}{}
	}

	// Extract creator from context (set by auth middleware); fallback to empty.
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	// 前端不传 execute_type 时默认「立即执行」，避免 ExecuteType 为空串导致
	// service 层扇出条件判断失败，device_tasks 无法派生（to-do-list.md 报告场景）。
	executeType := req.ExecuteType
	if executeType == "" {
		executeType = string(ExecuteImmediate)
	}

	execReq := ExecuteRequest{
		CommandCode:         req.CommandCode,
		DeviceSNs:           req.DeviceSNs,
		Parameters:          req.Parameters,
		TaskName:            req.TaskName,
		Creator:             creatorStr,
		Executor:            creatorStr,
		Commands:            req.Commands,
		ExecuteType:         ExecuteType(executeType),
		OfflineRetry:        req.OfflineRetry,
		OfflineRetryWait:    req.OfflineRetryWait,
		FailedRetry:         req.FailedRetry,
		FailedRetryCount:    req.FailedRetryCount,
		FailedRetryInterval: req.FailedRetryInterval,
		ParamPaths:          req.ParamPaths,
		ParamValues:         req.ParamValues,
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

	response.OKWithStatus(c, http.StatusCreated, task)
}

// ExecuteGroupHTTPRequest defines the request body for
// POST /api/v1/mml/groups/:id/execute（Sprint B Q-V3-1）。
// GroupID 取自 URL path 参数，因此 body 里不暴露该字段。
type ExecuteGroupHTTPRequest struct {
	DeviceSNs       []string               `json:"device_sns" binding:"required"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	TaskName        string                 `json:"task_name,omitempty"`
	OperationFilter []string               `json:"operation_filter,omitempty"`
	ExecuteType     string                 `json:"execute_type,omitempty"`
	ScheduledAt     string                 `json:"scheduled_at,omitempty"`
}

// ExecuteGroup handles POST /api/v1/mml/groups/:id/execute.
// 把指定 mml_param_group 下的全部命令一次性下发（操作类型过滤可选）。
// device_sns 必填；其他字段缺省走 ExecuteImmediate。
func (h *Handler) ExecuteGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req ExecuteGroupHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("mml group execute request validation failed",
			zap.String("client_ip", c.ClientIP()),
			zap.String("group_id", groupID.String()),
			zap.Error(err),
		)
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	executeType := req.ExecuteType
	if executeType == "" {
		executeType = string(ExecuteImmediate)
	}

	serviceReq := ExecuteGroupRequest{
		GroupID:         groupID,
		DeviceSNs:       req.DeviceSNs,
		Parameters:      req.Parameters,
		TaskName:        req.TaskName,
		Creator:         creatorStr,
		Executor:        creatorStr,
		OperationFilter: req.OperationFilter,
		ExecuteType:     ExecuteType(executeType),
	}
	if req.ScheduledAt != "" {
		serviceReq.ScheduledAt = &req.ScheduledAt
	}

	task, err := h.service.ExecuteGroup(c.Request.Context(), serviceReq)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, task)
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

	response.OK(c, result)
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

	response.OK(c, script)
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
		Creator:     creatorStr,
		Tags:        req.Tags,
	}

	created, err := h.service.CreateScript(c.Request.Context(), script)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, created)
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
		Tags:        req.Tags,
	}

	updated, err := h.service.UpdateScript(c.Request.Context(), id, script)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, updated)
}

// ListScriptRuns handles GET /api/v1/mml/scripts/:id/runs.
// 返回该脚本关联的全部 mml_tasks（模板 + periodic 子实例），倒序分页。
// 前端脚本详情页"历史执行"tab 消费该接口（P4 C11）。
func (h *Handler) ListScriptRuns(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	req := model.DefaultListRequest()
	if err := c.ShouldBindQuery(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	resp, err := h.service.ListRunsByScript(c.Request.Context(), id, req)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, resp)
}

// StartScript handles POST /api/v1/mml/scripts/:id/start.
func (h *Handler) StartScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	script, err := h.service.StartScript(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, script)
}

// PauseScript handles POST /api/v1/mml/scripts/:id/pause.
func (h *Handler) PauseScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	script, err := h.service.PauseScript(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, script)
}

// CancelScript handles POST /api/v1/mml/scripts/:id/cancel.
func (h *Handler) CancelScript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}
	script, err := h.service.CancelScript(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, script)
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

	response.OK(c, nil)
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
	if taskName := c.Query("task_name"); taskName != "" {
		filter.TaskName = &taskName
	}

	result, err := h.service.ListTasks(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
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

	response.OK(c, task)
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

	response.OK(c, task)
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

	response.OK(c, task)
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

	response.OK(c, task)
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

	response.OK(c, nil)
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

	response.OK(c, gin.H{
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

	response.OK(c, result)
}

// ---- Template handlers ----

// ListTemplates handles GET /api/v1/mml/templates.
func (h *Handler) ListTemplates(c *gin.Context) {
	filter := CustomCommandFilter{
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
	if commandScope := c.Query("command_scope"); commandScope != "" {
		filter.CommandScope = &commandScope
	}
	if categoryGroup := c.Query("category_group"); categoryGroup != "" {
		filter.CategoryGroup = &categoryGroup
	}

	// T-0090-c：传入当前用户 username + user_id 双凭据让 service 层做 RBAC 可见性派生。
	//   - Creator (username) 用于私有命令 self-fallback（永远能看到自己创建的）
	//   - UserID (uuid) 用于 service 调 RoleQuerier.GetUserVisibleGroupIDs 派生
	//     可见的 device group IDs → repo 走 group-share 路径
	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)
	filter.Creator = &creatorStr
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok && uid != uuid.Nil {
			filter.UserID = &uid
		}
	}

	result, err := h.service.ListCustomCommands(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, result)
}

// GetTemplate handles GET /api/v1/mml/templates/:id.
func (h *Handler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	tmpl, err := h.service.GetCustomCommand(c.Request.Context(), id)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, tmpl)
}

// CreateCustomCommandRequest defines the request body for creating a custom command.
type CreateCustomCommandRequest struct {
	CommandName   string                 `json:"command_name" binding:"required"`
	CommandCode   string                 `json:"command_code" binding:"required"`
	OperationType string                 `json:"operation_type" binding:"required"`
	CommandScope  string                 `json:"command_scope" binding:"required"`
	CategoryGroup string                 `json:"category_group"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
}

// CreateTemplate handles POST /api/v1/mml/templates.
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req CreateCustomCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	creator, _ := c.Get("username")
	creatorStr, _ := creator.(string)

	tmpl := &MMLCustomCommand{
		CommandName:   req.CommandName,
		CommandCode:   req.CommandCode,
		OperationType: req.OperationType,
		CommandScope:  req.CommandScope,
		CategoryGroup: req.CategoryGroup,
		Parameters:    req.Parameters,
		ParamPaths:    req.ParamPaths,
		Description:   req.Description,
		Creator:       creatorStr,
	}

	created, err := h.service.CreateCustomCommand(c.Request.Context(), tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, created)
}

// UpdateCustomCommandRequest defines the request body for updating a custom command.
type UpdateCustomCommandRequest struct {
	CommandName   string                 `json:"command_name" binding:"required"`
	CommandCode   string                 `json:"command_code" binding:"required"`
	OperationType string                 `json:"operation_type" binding:"required"`
	CommandScope  string                 `json:"command_scope" binding:"required"`
	CategoryGroup string                 `json:"category_group"`
	Parameters    map[string]interface{} `json:"parameters"`
	ParamPaths    []string               `json:"param_paths"`
	Description   string                 `json:"description"`
}

// UpdateTemplate handles PUT /api/v1/mml/templates/:id.
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return
	}

	var req UpdateCustomCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	tmpl := &MMLCustomCommand{
		CommandName:   req.CommandName,
		CommandCode:   req.CommandCode,
		OperationType: req.OperationType,
		CommandScope:  req.CommandScope,
		CategoryGroup: req.CategoryGroup,
		Parameters:    req.Parameters,
		ParamPaths:    req.ParamPaths,
		Description:   req.Description,
	}

	updated, err := h.service.UpdateCustomCommand(c.Request.Context(), id, tmpl)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, updated)
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

	if err := h.service.DeleteCustomCommand(c.Request.Context(), id, creatorStr); err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OK(c, nil)
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

	cloned, err := h.service.CloneCustomCommand(c.Request.Context(), id, creatorStr)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, cloned)
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
