package pm

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for PM data.
type Handler struct {
	counterRepo counter.CounterRepository
	kpiRepo     kpi.KPIRepository
	kpiEngine   *kpi.KPIEngine
	taskRepo    TaskRepository
	logger      *zap.Logger
}

// NewHandler creates a new PM handler.
func NewHandler(counterRepo counter.CounterRepository, kpiRepo kpi.KPIRepository, kpiEngine *kpi.KPIEngine, taskRepo TaskRepository, logger *zap.Logger) *Handler {
	return &Handler{counterRepo: counterRepo, kpiRepo: kpiRepo, kpiEngine: kpiEngine, taskRepo: taskRepo, logger: logger}
}

// RegisterRoutes registers PM API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	pm := rg.Group("/pm")
	{
		pm.GET("/counters", h.ListCounters)
		pm.GET("/counters/aggregated", h.ListAggregatedCounters)
		pm.GET("/kpi", h.ListKPIValues)
		pm.GET("/kpi/definitions", h.ListKPIDefinitions)
		pm.POST("/kpi/calculate", h.CalculateKPI)
		pm.GET("/tasks", h.ListTasks)
		pm.POST("/tasks", h.CreateTask)
	}
}

type counterQuery struct {
	DeviceID     string `form:"device_id"`
	CellID       string `form:"cell_id"`
	CounterGroup string `form:"counter_group"`
	CounterName  string `form:"counter_name"`
	StartTime    string `form:"start_time"`
	EndTime      string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) ListCounters(c *gin.Context) {
	var q counterQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := counter.CounterFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
			return
		}
		filter.DeviceID = &id
	}
	if q.CellID != "" {
		filter.CellID = &q.CellID
	}
	if q.CounterGroup != "" {
		filter.CounterGroup = &q.CounterGroup
	}
	if q.CounterName != "" {
		filter.CounterName = &q.CounterName
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = t
		}
	}
	result, err := h.counterRepo.Query(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListAggregatedCounters(c *gin.Context) {
	var q counterQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := counter.CounterFilter{}
	if q.DeviceID != "" {
		id, _ := uuid.Parse(q.DeviceID)
		filter.DeviceID = &id
	}
	if q.CellID != "" {
		filter.CellID = &q.CellID
	}
	if q.CounterGroup != "" {
		filter.CounterGroup = &q.CounterGroup
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = t
		}
	}
	result, err := h.counterRepo.QueryAggregated(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result})
}

type kpiQuery struct {
	DeviceID   string `form:"device_id"`
	CellID     string `form:"cell_id"`
	KPIName    string `form:"kpi_name"`
	Carrier    string `form:"carrier"`
	Technology string `form:"technology"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	model.ListRequest
}

func (h *Handler) ListKPIValues(c *gin.Context) {
	var q kpiQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter := kpi.KPIFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, _ := uuid.Parse(q.DeviceID)
		filter.DeviceID = &id
	}
	if q.CellID != "" {
		filter.CellID = &q.CellID
	}
	if q.KPIName != "" {
		filter.KPIName = &q.KPIName
	}
	if q.Carrier != "" {
		cc := model.CarrierCode(q.Carrier)
		filter.Carrier = &cc
	}
	if q.Technology != "" {
		t := model.Technology(q.Technology)
		filter.Technology = &t
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = t
		}
	}
	result, err := h.kpiRepo.Query(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListKPIDefinitions(c *gin.Context) {
	carrier := c.Query("carrier")
	tech := c.Query("technology")

	// Return from engine's in-memory formulas
	var items []model.KPIDefinition
	for _, f := range h.kpiEngine.Formulas() {
		if carrier != "" && string(f.Carrier) != carrier {
			continue
		}
		if tech != "" && string(f.Technology) != tech {
			continue
		}
		items = append(items, model.KPIDefinition{
			Name:        f.Name,
			DisplayName: f.DisplayName,
			Formula:     f.Parsed.Expression,
			Unit:        f.Unit,
			Carrier:     f.Carrier,
			Technology:  f.Technology,
			Counters:    f.Counters,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

type calculateRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	CellID     string `json:"cell_id"`
	StartTime  string `json:"start_time" binding:"required"`
	EndTime    string `json:"end_time" binding:"required"`
	Carrier    string `json:"carrier" binding:"required"`
	Technology string `json:"technology" binding:"required"`
}

func (h *Handler) CalculateKPI(c *gin.Context) {
	var req calculateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time"})
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time"})
		return
	}

	results, err := h.kpiEngine.CalculateAndStore(
		c.Request.Context(), deviceID, req.CellID, endTime,
		model.CarrierCode(req.Carrier), model.Technology(req.Technology),
	)
	_ = startTime // endTime is used as collectTime
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": results, "total": len(results)})
}

// ---- PM Task handlers ----

type taskQuery struct {
	Status   string `form:"status"`
	TaskType string `form:"task_type"`
	model.ListRequest
}

// ListTasks handles GET /pm/tasks.
func (h *Handler) ListTasks(c *gin.Context) {
	var q taskQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := TaskFilter{ListRequest: q.ListRequest}
	if q.Status != "" {
		s := TaskStatus(q.Status)
		filter.Status = &s
	}
	if q.TaskType != "" {
		t := PMTaskType(q.TaskType)
		filter.TaskType = &t
	}

	result, err := h.taskRepo.List(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// CreateTask handles POST /pm/tasks.
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreatePerformanceTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	task := &PerformanceTask{
		TaskName:    req.TaskName,
		TaskType:    req.TaskType,
		DeviceSNs:   req.DeviceSNs,
		KPICodes:    req.KPICodes,
		Granularity: req.Granularity,
		TimeRange:   req.TimeRange,
		Creator:     req.Creator,
	}

	if err := h.taskRepo.Create(c.Request.Context(), task); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}
