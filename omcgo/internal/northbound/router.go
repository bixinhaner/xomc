package northbound

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
)

// Router registers northbound/OSS API routes.
type Router struct {
	pmHandler     *PMHandler
	alarmHandler  *AlarmHandler
	configHandler *ConfigHandler
	pushEngine    *push.Engine
	syncService   *nbsync.Service
}

// NewRouter creates a new northbound Router.
func NewRouter(
	pmHandler *PMHandler,
	alarmHandler *AlarmHandler,
	configHandler *ConfigHandler,
	pushEngine *push.Engine,
	syncService *nbsync.Service,
) *Router {
	return &Router{
		pmHandler:     pmHandler,
		alarmHandler:  alarmHandler,
		configHandler: configHandler,
		pushEngine:    pushEngine,
		syncService:   syncService,
	}
}

// RegisterRoutes registers northbound API routes on the given router group.
func (r *Router) RegisterRoutes(rg *gin.RouterGroup) {
	nb := rg.Group("/northbound")
	{
		// Push target management
		nb.GET("/push/targets", r.listTargets)
		nb.POST("/push/targets", r.addTarget)
		nb.DELETE("/push/targets/:id", r.removeTarget)

		// Sync endpoints
		nb.GET("/sync/full", r.fullSync)
		nb.GET("/sync/incremental", r.incrementalSync)

		// Export endpoints
		nb.POST("/export/pm", r.pmHandler.ExportPM)
		nb.POST("/export/alarms", r.alarmHandler.ExportAlarms)
		nb.GET("/export/config/:deviceId", r.configHandler.ExportConfig)
	}
}

func (r *Router) listTargets(c *gin.Context) {
	targets := r.pushEngine.ListTargets()
	c.JSON(http.StatusOK, gin.H{"items": targets, "total": len(targets)})
}

func (r *Router) addTarget(c *gin.Context) {
	var req AddPushTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	target := &push.Target{
		ID:         req.ID,
		URL:        req.URL,
		AuthType:   req.AuthType,
		AuthToken:  req.AuthToken,
		DataTypes:  req.DataTypes,
		Format:     req.Format,
		BatchSize:  req.BatchSize,
		RetryCount: req.RetryCount,
		Enabled:    req.Enabled,
	}

	r.pushEngine.AddTarget(target)
	c.JSON(http.StatusCreated, gin.H{"message": "push target added", "id": req.ID})
}

func (r *Router) removeTarget(c *gin.Context) {
	id := c.Param("id")
	if !r.pushEngine.RemoveTarget(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "push target not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "push target removed", "id": id})
}

func (r *Router) fullSync(c *gin.Context) {
	dataType := c.DefaultQuery("data_type", "device")
	result, err := r.syncService.FullSync(c.Request.Context(), dataType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (r *Router) incrementalSync(c *gin.Context) {
	dataType := c.DefaultQuery("data_type", "alarm")
	sinceStr := c.Query("since")
	if sinceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "since parameter is required (RFC3339 format)"})
		return
	}

	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since format, use RFC3339"})
		return
	}

	result, err := r.syncService.IncrementalSync(c.Request.Context(), dataType, since)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
