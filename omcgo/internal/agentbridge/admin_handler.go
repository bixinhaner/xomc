package agentbridge

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct{ client *Client }

func NewAdminHandler(client *Client) *AdminHandler { return &AdminHandler{client: client} }

func (h *AdminHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/agent/admin/overview", h.overview)
	group.PATCH("/agent/admin/scenarios/:scenarioKey", h.updateScenario)
	group.POST("/agent/admin/runs/:runID/cancel", h.cancelRun)
}

func (h *AdminHandler) overview(c *gin.Context) {
	result, err := h.client.GetOverview(c.Request.Context())
	if err != nil {
		h.bridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) updateScenario(c *gin.Context) {
	var request ScenarioUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scenario update"})
		return
	}
	result, err := h.client.UpdateScenario(c.Request.Context(), c.Param("scenarioKey"), request)
	if err != nil {
		h.bridgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) cancelRun(c *gin.Context) {
	if err := h.client.CancelRun(c.Request.Context(), c.Param("runID")); err != nil {
		h.bridgeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) bridgeError(c *gin.Context, err error) {
	status := http.StatusBadGateway
	if errors.Is(err, ErrBridgeNotConnected) {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
