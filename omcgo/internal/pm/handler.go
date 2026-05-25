package pm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"go.uber.org/zap"
)

// Handler provides REST API endpoints for PM data.
type Handler struct {
	counterRepo   counter.CounterRepository
	kpiRepo       kpi.KPIRepository
	kpiEngine     *kpi.KPIEngine
	taskRepo      TaskRepository
	fileStore     PMFileStore
	minioClient   *minio.Client
	pmBucket      string
	pool          *pgxpool.Pool // T-0164-P3 fix: 反查 devices 拿 (oui, sn) 给 KPIEngine
	indicatorRepo indicator.IndicatorRepository // T-0164-P1：/pm/kpi/definitions 数据源（替代旧 carrier KPIDefinitions 列表）
	aggr          *aggregator.Aggregator        // T-0164-P5 / G5：按粒度路由聚合表查询（hourly/daily/weekly/monthly + device_group）
	asyncJobRepo  asyncjob.Repository           // T-0164 收尾 G5-Gap-2：手动重算入口入队 async_jobs
	metrics       *PMMetrics
	logger        *zap.Logger
}

// WithAggregator 注入 G5 聚合查询入口（可选；nil 时退回老路径）。
//
// 调用方：cmd/app/provider/router.go 在初始化 pm.Handler 后调用。
func (h *Handler) WithAggregator(aggr *aggregator.Aggregator) *Handler {
	h.aggr = aggr
	return h
}

// WithAsyncJobRepo 注入 asyncjob 仓库（G5-Gap-2 手动重算端点用）。
func (h *Handler) WithAsyncJobRepo(repo asyncjob.Repository) *Handler {
	h.asyncJobRepo = repo
	return h
}

// NewHandler creates a new PM handler.
func NewHandler(counterRepo counter.CounterRepository, kpiRepo kpi.KPIRepository, kpiEngine *kpi.KPIEngine, taskRepo TaskRepository, fileStore PMFileStore, minioClient *minio.Client, pmBucket string, pool *pgxpool.Pool, indicatorRepo indicator.IndicatorRepository, logger *zap.Logger) *Handler {
	return &Handler{
		counterRepo:   counterRepo,
		kpiRepo:       kpiRepo,
		kpiEngine:     kpiEngine,
		taskRepo:      taskRepo,
		fileStore:     fileStore,
		minioClient:   minioClient,
		pmBucket:      pmBucket,
		pool:          pool,
		indicatorRepo: indicatorRepo,
		logger:        logger,
	}
}

// lookupDeviceOUISN 反查 devices 表的 (oui, serial_number) 双键（T-0164-P3 fix）。
func (h *Handler) lookupDeviceOUISN(ctx context.Context, deviceID uuid.UUID) (string, string, error) {
	var oui, sn string
	err := h.pool.QueryRow(ctx,
		`SELECT oui, serial_number FROM devices WHERE id = $1`, deviceID,
	).Scan(&oui, &sn)
	if err != nil {
		return "", "", fmt.Errorf("lookup device oui+sn: %w", err)
	}
	return oui, sn, nil
}

// SetMetrics attaches Prometheus metrics to the handler.
func (h *Handler) SetMetrics(m *PMMetrics) {
	h.metrics = m
}

// RegisterRoutes registers PM API routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	pm := rg.Group("/pm")
	{
		pm.GET("/counters", h.ListCounters)
		pm.GET("/counters/aggregated", h.ListAggregatedCounters)
		// T-0164-P5 / G5：按粒度路由的聚合查询接口（pm_metrics_hourly / daily / weekly / monthly
		// + pm_group_metrics_*）。15min 粒度直查 pm_metrics；其余粒度走聚合表。
		pm.GET("/metrics/aggregated", h.ListAggregatedMetrics)
		// T-0164 收尾 G5-Gap-2：手动重算入口（晚到数据 / 补传场景运维触发）。
		pm.POST("/aggregation/recompute", h.RecomputeAggregation)
		pm.GET("/kpi", h.ListKPIValues)
		pm.GET("/kpi/definitions", h.ListKPIDefinitions)
		pm.POST("/kpi/calculate", h.CalculateKPI)
		pm.GET("/tasks", h.ListTasks)
		pm.POST("/tasks", h.CreateTask)
		pm.GET("/files", h.ListPMFiles)
		pm.GET("/files/:id/download", h.DownloadPMFile)
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
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := counter.CounterFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
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
	// Default time range guard: prevent full-table scans on compressed hypertable.
	// If no time range specified, default to last 24 hours.
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.StartTime.IsZero() {
		// If only end_time given, look back 24 hours from it
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}
	result, err := h.counterRepo.Query(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListAggregatedCounters(c *gin.Context) {
	var q counterQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
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
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": result})
}

// ListAggregatedMetrics 走 G5 aggregator.Query 按粒度路由聚合表（hourly+ 直查物化表，
// 15min 退回 pm_metrics 原表）。device_group 维度可选。
//
// Query params：
//   - granularity（必填）: 15min / hourly / daily / weekly / monthly
//   - dimension（可选）: device（默认）/ device_group
//   - device_oui+device_sn / device_group_id：维度过滤（与 dimension 配套）
//   - metric_path：单 metric 过滤（兼容 v1）；metric_paths：逗号分隔的多 metric 过滤（v2，PmDashboard panel 用）
//   - metric_type：counter / kpi
//   - start_time / end_time：RFC3339
//   - limit / offset
//
// 没注入 aggregator（兼容老部署）时返 503。
func (h *Handler) ListAggregatedMetrics(c *gin.Context) {
	if h.aggr == nil {
		response.Fail(c, http.StatusServiceUnavailable, "aggregator not wired; see plan T-0164-P5")
		return
	}
	gran := metrics.Granularity(c.Query("granularity"))
	if gran == "" {
		response.Fail(c, http.StatusBadRequest, "granularity is required")
		return
	}
	dim := aggregator.Dimension(c.Query("dimension"))
	req := aggregator.QueryRequest{Granularity: gran, Dimension: dim}

	if v := c.Query("device_oui"); v != "" {
		req.DeviceOUIs = []string{v}
	}
	if v := c.Query("device_sn"); v != "" {
		req.DeviceSNs = []string{v}
	}
	if v := c.Query("device_group_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_group_id")
			return
		}
		req.DeviceGroupIDs = []uuid.UUID{id}
	}
	if v := c.Query("metric_paths"); v != "" {
		parts := strings.Split(v, ",")
		paths := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			req.MetricPaths = paths
		}
	} else if v := c.Query("metric_path"); v != "" {
		req.MetricPaths = []string{v}
	}
	if v := c.Query("metric_type"); v != "" {
		mt := metrics.MetricType(v)
		req.MetricType = &mt
	}
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.StartTime = t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.EndTime = t
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			req.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			req.Offset = n
		}
	}

	rows, err := h.aggr.Query(c.Request.Context(), req)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": rows, "total": len(rows)})
}

// RecomputeAggregation POST /pm/aggregation/recompute（T-0164 收尾 G5-Gap-2）
//
// 入参：{granularity, dimension, start, end}
//   - granularity: 'hourly' / 'daily' / 'weekly' / 'monthly'
//   - dimension:   'device'（默认）/ 'device_group'
//   - start/end:   桶起止（RFC3339）
//
// 行为：直接 INSERT async_jobs 入队对应 job_type，worker 抢到后跑一次。
// 用于"晚到数据 / 补传场景"运维补算或自动化脚本调度，不走 cron。
type recomputeRequest struct {
	Granularity string    `json:"granularity" binding:"required,oneof=hourly daily weekly monthly"`
	Dimension   string    `json:"dimension"`
	Start       time.Time `json:"start" binding:"required"`
	End         time.Time `json:"end" binding:"required"`
}

func (h *Handler) RecomputeAggregation(c *gin.Context) {
	if h.asyncJobRepo == nil {
		response.Fail(c, http.StatusServiceUnavailable, "async job repo not wired; see G5-Gap-2")
		return
	}
	var req recomputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if !req.End.After(req.Start) {
		response.Fail(c, http.StatusBadRequest, "end must be after start")
		return
	}

	dim := req.Dimension
	if dim == "" {
		dim = "device"
	}
	// 路由 granularity + dimension → job_type
	jobType := ""
	switch req.Granularity {
	case "hourly":
		jobType = aggregator.JobTypeHourly
		if dim == "device_group" {
			jobType = aggregator.JobTypeHourlyGroup
		}
	case "daily":
		jobType = aggregator.JobTypeDaily
		if dim == "device_group" {
			jobType = aggregator.JobTypeDailyGroup
		}
	case "weekly":
		jobType = aggregator.JobTypeWeekly
		if dim == "device_group" {
			jobType = aggregator.JobTypeWeeklyGroup
		}
	case "monthly":
		jobType = aggregator.JobTypeMonthly
		if dim == "device_group" {
			jobType = aggregator.JobTypeMonthlyGroup
		}
	default:
		response.Fail(c, http.StatusBadRequest, "unsupported granularity")
		return
	}

	payload, err := aggregator.BuildPayload(req.Start, req.End)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	jobID, err := h.asyncJobRepo.Insert(c.Request.Context(), asyncjob.InsertRequest{
		JobType:     jobType,
		ScheduledAt: time.Now(),
		Payload:     payload,
	})
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OKWithStatus(c, http.StatusAccepted, gin.H{
		"job_id":   jobID.String(),
		"job_type": jobType,
		"start":    req.Start,
		"end":      req.End,
	})
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
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
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
	// Default time range guard: prevent full-table scans on compressed hypertable.
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}
	result, err := h.kpiRepo.Query(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// ListKPIDefinitions 返回 KPI 元数据目录，供前端"性能管理→指标列表"等界面消费。
//
// T-0164-P1 改造前：数据源是 carrier 适配器代码里硬编码的 KPIDefinitions 列表。
// 现在改为枚举 perf_indicators_{enb,gsm,gnb}（is_counter='0' 的行），
// platform_name 维度的具体公式在 KPIEngine 计算路径上由 router 路由解析，
// 这里只暴露"系统支持哪些 KPI 指标"。
//
// 查询参数（向后兼容老前端的 carrier 模糊搜索）：
//   - keyword：name/cn_name/id 模糊匹配
//   - device_type：ENB / GSM / GNB（不传 → 三表合并枚举）
func (h *Handler) ListKPIDefinitions(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		// 老前端把 keyword 塞在 carrier 参数里（见 omcmb/.../usePerformance）— 兼容一下。
		keyword = c.Query("carrier")
	}
	dtParam := c.Query("device_type")

	dts := []indicator.DeviceType{indicator.DeviceTypeENB, indicator.DeviceTypeGSM, indicator.DeviceTypeGNB}
	if dtParam != "" {
		dt, err := indicator.ParseDeviceType(dtParam)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_type")
			return
		}
		dts = []indicator.DeviceType{dt}
	}

	if h.indicatorRepo == nil {
		// 测试 / 退化场景：返空集合，调用方按 total=0 处理。
		response.OK(c, gin.H{"items": []model.KPIDefinition{}, "total": 0})
		return
	}

	isCounter := "0" // KPI 而非 counter
	var items []model.KPIDefinition
	for _, dt := range dts {
		filter := indicator.IndicatorListFilter{
			DeviceType: string(dt),
			IsCounter:  &isCounter,
		}
		if keyword != "" {
			kw := keyword
			filter.Keyword = &kw
		}
		rows, err := h.indicatorRepo.ListAll(c.Request.Context(), filter)
		if err != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			items = append(items, model.KPIDefinition{
				Name:        r.EnName,
				DisplayName: derefOr(r.CnName, r.EnName),
				Formula:     derefOr(r.Arithmetic, ""),
				Unit:        derefOr(r.UnitID, ""),
				// Carrier / Technology 在新模型下不再是 KPI 维度（按平台 + 设备类型路由），
				// 留空以保持 wire 兼容。
				Counters: nil,
			})
		}
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
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
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid device_id")
		return
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid start_time")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid end_time")
		return
	}

	oui, sn, err := h.lookupDeviceOUISN(c.Request.Context(), deviceID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	results, err := h.kpiEngine.CalculateAndStore(
		c.Request.Context(), deviceID, oui, sn, req.CellID, endTime,
		model.CarrierCode(req.Carrier), model.Technology(req.Technology),
	)
	_ = startTime // endTime is used as collectTime
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, gin.H{"items": results, "total": len(results)})
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
	response.OK(c, result)
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
	response.OKWithStatus(c, http.StatusCreated, task)
}

// ---- PM File handlers ----

type pmFileQuery struct {
	DeviceID  string `form:"device_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	model.ListRequest
}

// ListPMFiles handles GET /pm/files.
func (h *Handler) ListPMFiles(c *gin.Context) {
	var q pmFileQuery
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	filter := PMFileFilter{ListRequest: q.ListRequest}
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		filter.DeviceID = &id
	}
	if q.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, q.StartTime); err == nil {
			filter.StartTime = &t
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, q.EndTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.fileStore.ListFiles(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// DownloadPMFile handles GET /pm/files/:id/download.
func (h *Handler) DownloadPMFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid file id")
		return
	}

	fileInfo, err := h.fileStore.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "lookup failed")
		return
	}
	if fileInfo == nil {
		response.Fail(c, http.StatusNotFound, "file not found")
		return
	}

	obj, err := h.minioClient.GetObject(c.Request.Context(), h.pmBucket, fileInfo.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "download failed")
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "get file info failed")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.FileName))
	c.Header("Content-Type", "application/xml")
	c.DataFromReader(http.StatusOK, stat.Size, "application/xml", obj, nil)
}
