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
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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
	pool          *pgxpool.Pool                 // T-0164-P3 fix: 反查 devices 拿 (oui, sn) 给 KPIEngine
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
		// T-0193 自选设备下钻：列设备在 PM 数据里实际出现过的 distinct 小区/PLMN（object_ldn）。
		pm.GET("/metrics/objects", h.ListMetricObjects)
		// T-0164 收尾 G5-Gap-2：手动重算入口（晚到数据 / 补传场景运维触发）。
		pm.POST("/aggregation/recompute", h.RecomputeAggregation)
		pm.GET("/kpi", h.ListKPIValues)
		pm.GET("/kpi/definitions", h.ListKPIDefinitions)
		pm.POST("/kpi/calculate", h.CalculateKPI)
		pm.GET("/tasks", h.ListTasks)
		pm.POST("/tasks", h.CreateTask)
		pm.GET("/files", h.ListPMFiles)
		pm.GET("/files/devices", h.ListPMFileDevices) // 按设备聚合，给 File Management PM Tab 用
		pm.GET("/files/:id/download", h.DownloadPMFile)
		pm.POST("/files/batch-delete", h.BatchDeletePMFiles) // 按 SN 批量删除（PG + MinIO）
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
	// fill_empty=true：数据驱动补齐占位行（T-0192d）。只对"已存在真实记录组"里所查
	// 但缺失的指标补一行占位，让透视表能区分"该时段有采样但此指标无值"与"此指标有值"。
	// 没有任何真实行的时间桶/object 永不出现（空时段不凭空造桶）。
	// 仅 device 维度（单 OUI+SN）+ metric_paths 非空时启用。
	if c.Query("fill_empty") == "true" {
		rows = fillEmptyBuckets(rows, req)
	}
	// 真实总数：跑一次同过滤的 COUNT，让 total 反映命中真实总数而非本页返回行数
	// （T-0194 截断诚实提示）。命中 limit 时 total>len(rows)，前端据此提示「已截断」。
	// 仅在指定了 limit 时才多跑一次（无 limit = 全量返回，total 即 len 无需 COUNT）；
	// COUNT 失败不阻断结果返回，退回本页行数兜底。
	total := len(rows)
	if req.Limit > 0 {
		if n, err := h.aggr.Count(c.Request.Context(), req); err == nil {
			total = n
		}
	}
	response.OK(c, gin.H{"items": rows, "total": total})
}

// fillEmptyBuckets 数据驱动补齐占位行（T-0192d）。
//
// 语义：判断单位 = 一条测量记录身份 = (object_ldn, 时间桶)。只遍历查询已返回的真实行
// （Filled=false），按 (object_ldn 归一, Time) 分组；对每个**已存在**的分组，req.MetricPaths
// 里缺失的指标补一行占位（身份/时段字段直接抄该组代表行、不做任何桶推算，MetricValue=0、
// Filled=true、DisplayName 沿用同 metric_path 真实行的友好名）。没有任何真实行的时间桶/object
// 永不出现 —— 空时段不凭空造桶。
//
// 分组键含 object_ldn（nil = 设备级，归一为固定空键），修旧实现去重键漏 object_ldn 的 bug：
// 多小区/PLMN 同时段各自独立填充、互不串。
//
// 仅 device 维度（单 SN）+ metric_paths 非空时启用；多 SN / 组维度原样返回。
func fillEmptyBuckets(rows []aggregator.Row, req aggregator.QueryRequest) []aggregator.Row {
	if req.Dimension == aggregator.DimensionDeviceGroup || req.Dimension == aggregator.DimensionAggregateGroup {
		return rows
	}
	if len(req.MetricPaths) == 0 {
		return rows
	}
	// 单 SN 过滤（前端 KPIQuery 总是 1:1 拆分发请求）— 多 SN 复合查询不补。
	if len(req.DeviceSNs) != 1 {
		return rows
	}

	// 各 metric_path 的友好名（占位行沿用，KPI 列头不致一半友好名一半 K 编号）。
	nameByPath := make(map[string]string, len(req.MetricPaths))

	// 按 (object_ldn 归一, Time) 分组：记录每组已出现的 metric_path 集合 + 一行代表行。
	type group struct {
		rep  aggregator.Row      // 代表行，占位行抄它的身份/时段字段
		have map[string]struct{} // 已出现的 metric_path 集合
	}
	groups := make(map[string]*group)
	order := make([]string, 0) // 保持分组出现顺序，占位行追加稳定

	for _, r := range rows {
		if r.Filled {
			continue // 只看真实行（防御性：正常此时 rows 全为真实行）
		}
		if r.DisplayName != "" {
			nameByPath[r.MetricPath] = r.DisplayName
		}
		key := groupKey(r.ObjectLDN, r.Time)
		g := groups[key]
		if g == nil {
			g = &group{rep: r, have: make(map[string]struct{})}
			groups[key] = g
			order = append(order, key)
		}
		g.have[r.MetricPath] = struct{}{}
	}

	// 对每个已存在分组，补 req.MetricPaths 里缺的指标。
	for _, key := range order {
		g := groups[key]
		for _, mp := range req.MetricPaths {
			if _, ok := g.have[mp]; ok {
				continue
			}
			rows = append(rows, aggregator.Row{
				DeviceOUI:   g.rep.DeviceOUI,
				DeviceSN:    g.rep.DeviceSN,
				MetricPath:  mp,
				DisplayName: nameByPath[mp],
				Granularity: g.rep.Granularity,
				Time:        g.rep.Time,
				StartTime:   g.rep.StartTime,
				EndTime:     g.rep.EndTime,
				ObjectLDN:   g.rep.ObjectLDN,
				Filled:      true,
			})
		}
	}
	return rows
}

// groupKey 构造 (object_ldn, time) 分组键。object_ldn=nil（设备级）归一为固定空键，
// 与有值的 object_ldn 互不混淆；time 用 UTC RFC3339 规范化。
func groupKey(objectLDN *string, t time.Time) string {
	ldn := "\x00" // nil 哨兵：与任何真实 object_ldn 串值不可能相等
	if objectLDN != nil {
		ldn = *objectLDN
	}
	return ldn + "||" + t.UTC().Format(time.RFC3339Nano)
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
//   - include_counters：可选开关（缺省 false）。false 时维持 is_counter='0'（仅 KPI），
//     行为与历史完全一致；true 时不按 is_counter 过滤，返回 KPI + 计数器，供向导穿梭框消费。
//
// 响应每条在历史字段（name/display_name/formula/unit）之外，补 id（= perf_indicators 行 ID，
// K/C 编号）与 is_counter（"0" KPI / "1" 计数器）。默认调用方只读历史字段，零回归。
func (h *Handler) ListKPIDefinitions(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		// 老前端把 keyword 塞在 carrier 参数里（见 omcmb/.../usePerformance）— 兼容一下。
		keyword = c.Query("carrier")
	}
	dtParam := c.Query("device_type")
	includeCounters := parseBoolQuery(c.Query("include_counters"))

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
		response.OK(c, gin.H{"items": []kpiDefinitionItem{}, "total": 0})
		return
	}

	items := make([]kpiDefinitionItem, 0)
	for _, dt := range dts {
		filter := indicator.IndicatorListFilter{
			DeviceType: string(dt),
		}
		if !includeCounters {
			isCounter := "0" // 仅 KPI（默认行为，不变）
			filter.IsCounter = &isCounter
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
			items = append(items, kpiDefinitionItem{
				ID:          r.ID,
				IsCounter:   r.IsCounter,
				Name:        r.EnName,
				DisplayName: derefOr(r.CnName, r.EnName),
				Formula:     derefOr(r.Arithmetic, ""),
				Unit:        derefOr(r.UnitID, ""),
			})
		}
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

// kpiDefinitionItem 是 ListKPIDefinitions 的 handler 局部响应结构，
// 历史字段（name/display_name/formula/unit）与旧 model.KPIDefinition 的 JSON 形态一致，
// 额外补 id / is_counter。不污染共享的 model.KPIDefinition（KPI 计算引擎在用）。
type kpiDefinitionItem struct {
	ID          string `json:"id"`
	IsCounter   string `json:"is_counter"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Formula     string `json:"formula"`
	Unit        string `json:"unit"`
}

// parseBoolQuery 把 query 字符串解析为 bool，仅 "true"/"1" 视为 true，其余（含空）为 false。
func parseBoolQuery(v string) bool {
	return v == "true" || v == "1"
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
	DeviceSN  string `form:"device_sn"`
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
	if q.DeviceSN != "" {
		filter.DeviceSN = &q.DeviceSN
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

// ListPMFileDevices 给 File Management → PM Tab 主列表用。每行 1 个设备 +
// 该设备 pm_files 起止 collect_time + 文件数 + reporting 标志 + 站名/产品类。
func (h *Handler) ListPMFileDevices(c *gin.Context) {
	filter := PMFileDeviceFilter{
		ListRequest: model.DefaultListRequest(),
	}
	if err := c.ShouldBindQuery(&filter.ListRequest); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	if kw := c.Query("keyword"); kw != "" {
		filter.Keyword = &kw
	}
	if sn := c.Query("site_name"); sn != "" {
		filter.SiteName = &sn
	}
	if pc := c.Query("product_class"); pc != "" {
		filter.ProductClass = &pc
	}
	result, err := h.fileStore.ListFileDeviceAggregates(c.Request.Context(), filter)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return
	}
	response.OK(c, result)
}

// BatchDeletePMFilesRequest body.
type BatchDeletePMFilesRequest struct {
	SerialNumbers []string `json:"serial_numbers" binding:"required,min=1"`
}

// BatchDeletePMFilesResponse 报告每条结果（与 license batch-delete 对齐）。
type BatchDeletePMFilesResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
}

// BatchDeletePMFiles handles POST /pm/files/batch-delete.
// 流程：每个 SN → 列文件 → 删 MinIO 对象 → 删 PG 行。MinIO 删失败也继续删 PG
// （MinIO 残留靠 retention policy 兜底，比保留孤儿 PG 行更干净）。
func (h *Handler) BatchDeletePMFiles(c *gin.Context) {
	var req BatchDeletePMFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	succeeded := make([]string, 0, len(req.SerialNumbers))
	failed := make([]string, 0)
	for _, sn := range req.SerialNumbers {
		files, err := h.fileStore.ListFilesBySN(c.Request.Context(), sn)
		if err != nil {
			h.logger.Warn("batch delete: list files failed",
				zap.String("sn", sn), zap.Error(err))
			failed = append(failed, sn)
			continue
		}
		// 先删 MinIO 对象（删失败仅 warn 不中断，目的是别留孤儿 PG 行）
		for _, f := range files {
			if rmErr := h.minioClient.RemoveObject(c.Request.Context(), h.pmBucket, f.MinioPath, minio.RemoveObjectOptions{}); rmErr != nil {
				h.logger.Warn("batch delete: minio remove failed",
					zap.String("sn", sn), zap.String("path", f.MinioPath), zap.Error(rmErr))
			}
		}
		if _, err := h.fileStore.DeleteFilesBySN(c.Request.Context(), sn); err != nil {
			h.logger.Warn("batch delete: pg delete failed",
				zap.String("sn", sn), zap.Error(err))
			failed = append(failed, sn)
			continue
		}
		succeeded = append(succeeded, sn)
	}
	response.OK(c, BatchDeletePMFilesResponse{Succeeded: succeeded, Failed: failed})
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
