package mml

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
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

	mml.POST("/execute", h.Execute)

	scripts := mml.Group("/scripts")
	scripts.GET("", h.ListScripts)
	scripts.POST("", h.CreateScript)
	scripts.PUT("/:id", h.UpdateScript)
	scripts.DELETE("/:id", h.DeleteScript)

	tasks := mml.Group("/tasks")
	tasks.GET("", h.ListTasks)
	tasks.GET("/:id", h.GetTask)
}

// ---- Request types ----

// ExecuteHTTPRequest defines the request body for executing an MML command.
type ExecuteHTTPRequest struct {
	CommandCode string                 `json:"command_code"`
	DeviceSNs   []string               `json:"device_sns" binding:"required"`
	Parameters  map[string]interface{} `json:"parameters"`
	TaskName    string                 `json:"task_name"`
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
		CommandCode: req.CommandCode,
		DeviceSNs:   req.DeviceSNs,
		Parameters:  req.Parameters,
		TaskName:    req.TaskName,
		Creator:     creatorStr,
	}

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
