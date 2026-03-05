package topology

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// Handler provides HTTP handlers for topology (device group) REST API.
type Handler struct {
	repo    DeviceGroupRepository
	service *DeviceGroupService
}

// NewHandler creates a new topology REST API handler.
func NewHandler(repo DeviceGroupRepository, service *DeviceGroupService) *Handler {
	return &Handler{repo: repo, service: service}
}

// RegisterRoutes registers topology routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	groups := rg.Group("/groups")
	{
		groups.GET("", h.ListTree)
		groups.POST("", h.Create)
		groups.GET("/:id", h.Get)
		groups.PUT("/:id", h.Update)
		groups.DELETE("/:id", h.Delete)
		groups.POST("/:id/devices", h.AddDevice)
		groups.DELETE("/:id/devices/:deviceId", h.RemoveDevice)
		groups.GET("/:id/devices", h.ListDevices)
	}
}

// ListTree handles GET /api/v1/groups (returns tree structure).
func (h *Handler) ListTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tree})
}

// Get handles GET /api/v1/groups/:id.
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

type createGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	ParentID    string `json:"parent_id"`
	Carrier     string `json:"carrier"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// Create handles POST /api/v1/groups.
func (h *Handler) Create(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := &DeviceGroup{
		Name:        req.Name,
		Carrier:     model.CarrierCode(req.Carrier),
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		group.ParentID = &pid
	}

	if err := h.repo.Create(c.Request.Context(), group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, group)
}

type updateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	ParentID    string `json:"parent_id"`
	Carrier     string `json:"carrier"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// Update handles PUT /api/v1/groups/:id.
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	existing.Name = req.Name
	existing.Carrier = model.CarrierCode(req.Carrier)
	existing.Description = req.Description
	existing.SortOrder = req.SortOrder

	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		existing.ParentID = &pid
	} else {
		existing.ParentID = nil
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/groups/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

type addDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// AddDevice handles POST /api/v1/groups/:id/devices.
func (h *Handler) AddDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req addDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}

	if err := h.repo.AddDevice(c.Request.Context(), groupID, deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "device added to group"})
}

// RemoveDevice handles DELETE /api/v1/groups/:id/devices/:deviceId.
func (h *Handler) RemoveDevice(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	if err := h.repo.RemoveDevice(c.Request.Context(), groupID, deviceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not in group"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ListDevices handles GET /api/v1/groups/:id/devices.
func (h *Handler) ListDevices(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	ids, err := h.repo.ListDeviceIDs(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_ids": ids, "total": len(ids)})
}
