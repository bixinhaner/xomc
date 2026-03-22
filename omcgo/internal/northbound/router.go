package northbound

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/northbound/push"
)

// Router registers northbound/OSS API routes.
type Router struct {
	svc           *NorthboundService
	pmHandler     *PMHandler
	alarmHandler  *AlarmHandler
	configHandler *ConfigHandler
	outboxRepo    push.OutboxRepository // may be nil if outbox is not configured
}

// NewRouter creates a new northbound Router.
func NewRouter(svc *NorthboundService) *Router {
	logger := svc.logger
	return &Router{
		svc:           svc,
		pmHandler:     NewPMHandler(svc, logger),
		alarmHandler:  NewAlarmHandler(svc, logger),
		configHandler: NewConfigHandler(svc, logger),
	}
}

// SetOutboxRepo sets the outbox repository for dead letter queue endpoints.
func (r *Router) SetOutboxRepo(repo push.OutboxRepository) {
	r.outboxRepo = repo
}

// RegisterRoutes registers northbound API routes on the given router group.
func (r *Router) RegisterRoutes(rg *gin.RouterGroup) {
	nb := rg.Group("/northbound")
	{
		// Push target management
		nb.GET("/push/targets", r.listTargets)
		nb.POST("/push/targets", r.addTarget)
		nb.DELETE("/push/targets/:id", r.removeTarget)

		// Circuit breaker status
		nb.GET("/push/targets/:id/circuit", r.getCircuitBreaker)
		nb.POST("/push/targets/:id/circuit/reset", r.resetCircuitBreaker)

		// Dead letter queue
		nb.GET("/push/deadletter", r.listDeadLetters)
		nb.POST("/push/deadletter/:id/replay", r.replayDeadLetter)

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
	targets := r.svc.PushEngine().ListTargets()
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

	r.svc.PushEngine().AddTarget(target)
	c.JSON(http.StatusCreated, gin.H{"message": "push target added", "id": req.ID})
}

func (r *Router) removeTarget(c *gin.Context) {
	id := c.Param("id")
	if !r.svc.PushEngine().RemoveTarget(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "push target not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "push target removed", "id": id})
}

func (r *Router) getCircuitBreaker(c *gin.Context) {
	id := c.Param("id")
	cb := r.svc.PushEngine().GetCircuitBreaker(id)
	if cb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "push target not found"})
		return
	}

	failureCount, threshold := cb.Counters()
	state := cb.State()
	c.JSON(http.StatusOK, gin.H{
		"target_id":     id,
		"state":         state.String(),
		"failure_count": failureCount,
		"threshold":     threshold,
	})
}

func (r *Router) resetCircuitBreaker(c *gin.Context) {
	id := c.Param("id")
	cb := r.svc.PushEngine().GetCircuitBreaker(id)
	if cb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "push target not found"})
		return
	}

	cb.Reset()
	c.JSON(http.StatusOK, gin.H{"message": "circuit breaker reset", "target_id": id, "state": reliability.StateClosed.String()})
}

func (r *Router) listDeadLetters(c *gin.Context) {
	if r.outboxRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "outbox not configured"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	entries, total, err := r.outboxRepo.ListDead(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries, "total": total})
}

func (r *Router) replayDeadLetter(c *gin.Context) {
	if r.outboxRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "outbox not configured"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := r.outboxRepo.Replay(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "dead letter replayed", "id": idStr})
}

func (r *Router) fullSync(c *gin.Context) {
	dataType := c.DefaultQuery("data_type", "device")
	result, err := r.svc.SyncService().FullSync(c.Request.Context(), dataType)
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

	result, err := r.svc.SyncService().IncrementalSync(c.Request.Context(), dataType, since)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
