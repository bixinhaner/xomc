package provision

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides HTTP handlers for provisioning REST API.
type Handler struct {
	repo   ProvisioningTaskRepository
	engine *ProvisioningEngine
}

// NewHandler creates a new provisioning REST API handler.
func NewHandler(repo ProvisioningTaskRepository, engine *ProvisioningEngine) *Handler {
	return &Handler{repo: repo, engine: engine}
}

// RegisterRoutes registers provisioning routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	prov := rg.Group("/provisioning/tasks")
	{
		prov.GET("", h.List)
		prov.GET("/:id", h.Get)
		prov.POST("", h.Create)
		prov.POST("/:id/retry", h.Retry)
	}
}

// List handles GET /api/v1/provisioning/tasks.
func (h *Handler) List(c *gin.Context) {
	filter := ProvisioningTaskFilter{
		Page:     1,
		PageSize: 20,
	}
	if status := c.Query("status"); status != "" {
		filter.Status = ProvisioningState(status)
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		id, err := uuid.Parse(deviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}

	items, total, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": items, "total": total})
}

// Get handles GET /api/v1/provisioning/tasks/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}
	response.OK(c, task)
}

type createTaskRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// Create handles POST /api/v1/provisioning/tasks (manual provisioning trigger).
func (h *Handler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return
	}

	task := NewProvisioningTask(deviceID)
	if err := h.repo.Create(c.Request.Context(), task); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, task)
}

// Retry handles POST /api/v1/provisioning/tasks/:id/retry.
func (h *Handler) Retry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}

	if task.Status != StateFailed {
		response.Fail(c, http.StatusBadRequest, "only failed tasks can be retried")
		return
	}

	// Create a new task for retry.
	newTask := NewProvisioningTask(task.DeviceID)
	if err := h.repo.Create(c.Request.Context(), newTask); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}

	response.OKWithStatus(c, http.StatusCreated, newTask)
}
