package pm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/core/compress"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"go.uber.org/zap"
)

const (
	maxAggregatedMetricDeviceSNs  = 50
	maxAggregatedMetricPaths      = 50
	aggregatedMetricsQueryTimeout = 60 * time.Second
)

// Handler provides REST API endpoints for PM data.
type Handler struct {
	counterRepo counter.CounterRepository
	kpiRepo     kpi.KPIRepository
	kpiEngine   *kpi.KPIEngine
	taskRepo    TaskRepository
	fileStore   PMFileStore
	minioClient *minio.Client
	pmBucket    string
	// deviceQuery 收敛 #18：Handler 原先直连 SQL 池（h.pool）反查 devices /
	// 读 pm_metrics，现统一经 DeviceQueryService（handler → service → repository）。
	deviceQuery     DeviceQueryService
	indicatorRepo   indicator.IndicatorRepository // T-0164-P1：/pm/kpi/definitions 数据源（替代旧 carrier KPIDefinitions 列表）
	aggr            aggregatedMetricsBackend      // T-0164-P5 / G5：按粒度路由聚合表查询（hourly/daily/weekly/monthly + device_group）
	asyncJobRepo    asyncjob.Repository           // T-0164 收尾 G5-Gap-2：手动重算入口入队 async_jobs
	resolver        *authz.Resolver               // #64 设备组数据权限：PM 读链路按调用者可见分组过滤
	productResolver ProductPatternResolver        // #602; nil-safe (product_id 过滤参数被忽略)
	metrics         *PMMetrics
	timezone        calendarfilter.TimezoneProvider
	logger          *zap.Logger
}

type aggregatedMetricsBackend interface {
	Query(context.Context, aggregator.QueryRequest) ([]aggregator.Row, error)
	Count(context.Context, aggregator.QueryRequest) (int, error)
	DiscoverObjectLDNs(context.Context, aggregator.QueryRequest) ([]string, error)
	DevicePivotRowKeys(context.Context, aggregator.QueryRequest) ([]aggregator.PivotRowKey, error)
	BackfillDisplayNames(context.Context, []aggregator.Row)
	SetTimezoneProvider(calendarfilter.TimezoneProvider)
}

// ProductPatternResolver 把 product_id 解析为该产品的 product_class 模式字面量集合。
// 由 cmd/app/provider 将 *product.Registry 以接口注入，避免 pm 包直接依赖 product 包。
type ProductPatternResolver interface {
	GetPatternsByProductID(ctx context.Context, productID uuid.UUID) ([]string, error)
}

// SetProductPatternResolver 装配「产品名称下拉」过滤能力。未装配时 product_id 查询参数被忽略。
func (h *Handler) SetProductPatternResolver(r ProductPatternResolver) {
	h.productResolver = r
}

// SetPermissionService 注入数据权限解析器（#64 统一强制层），使 PM 读链路按调用者可见设备组
// 过滤。未注入时退化为不过滤（dev/test），与 alarm / device 模块语义一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// resolveVisibleGroups 解析调用者可见设备组（三态：nil 超管 / [] 无权限 / [g...] 限定）。
// 返回 ok=false 表示解析失败已 abort（403/500），调用方应立即 return。
// h.resolver 为 nil 时 FromContext 走 nil-safe 退化路径，返回 (nil, true) 不过滤。
func (h *Handler) resolveVisibleGroups(c *gin.Context) (groups []uuid.UUID, ok bool) {
	return h.resolver.FromContext(c)
}

// parseCSVInts 解析逗号分隔的整数列表（如 "0,1,2"）。非法值静默跳过；空返回 nil。
func parseCSVInts(raw string) []int {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// requestedDeviceInScope 在请求带 device_id 时预检其是否在可见分组内。
// visibleGroups==nil（超管）放行；[] 直接拒；否则查该设备的分组与可见集合是否有交集。
// 复用 deviceQuery 的连接池查 device_group_members，避免 PM handler 反向 import device。
// 返回 (inScope, ok)：ok=false 表示已 abort（解析/查询失败）；inScope=false 表示越权（调用方应 403/空）。
func (h *Handler) requestedDeviceInScope(c *gin.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) (inScope bool, ok bool) {
	if visibleGroups == nil {
		return true, true // 超管
	}
	if len(visibleGroups) == 0 {
		return false, true // 无任何分组权限
	}
	groupIDs, err := h.deviceQuery.DeviceGroupIDs(c.Request.Context(), deviceID)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
		return false, false
	}
	visible := make(map[uuid.UUID]struct{}, len(visibleGroups))
	for _, g := range visibleGroups {
		visible[g] = struct{}{}
	}
	for _, g := range groupIDs {
		if _, in := visible[g]; in {
			return true, true
		}
	}
	return false, true
}

// groupInVisible 判定显式 device_group_id 是否在可见集合内（三态）：
// visibleGroups==nil（超管）恒真；[] 恒假；否则成员判定。
func groupInVisible(groupID uuid.UUID, visibleGroups []uuid.UUID) bool {
	if visibleGroups == nil {
		return true
	}
	for _, g := range visibleGroups {
		if g == groupID {
			return true
		}
	}
	return false
}

// WithAggregator 注入 G5 聚合查询入口（可选；nil 时退回老路径）。
//
// 调用方：cmd/app/provider/router.go 在初始化 pm.Handler 后调用。
func (h *Handler) WithAggregator(aggr *aggregator.Aggregator) *Handler {
	if aggr == nil {
		h.aggr = nil
		return h
	}
	h.aggr = aggr
	if h.timezone != nil {
		aggr.SetTimezoneProvider(h.timezone)
	}
	return h
}

func (h *Handler) withAggregatedMetricsBackend(backend aggregatedMetricsBackend) *Handler {
	h.aggr = backend
	if h.aggr != nil && h.timezone != nil {
		h.aggr.SetTimezoneProvider(h.timezone)
	}
	return h
}

// WithAsyncJobRepo 注入 asyncjob 仓库（G5-Gap-2 手动重算端点用）。
func (h *Handler) WithAsyncJobRepo(repo asyncjob.Repository) *Handler {
	h.asyncJobRepo = repo
	return h
}

func (h *Handler) WithTimezoneProvider(provider calendarfilter.TimezoneProvider) *Handler {
	h.timezone = provider
	if h.aggr != nil {
		h.aggr.SetTimezoneProvider(provider)
	}
	return h
}

// NewHandler creates a new PM handler.
//
// pool 仅用于构造 DeviceQueryService（#18 收敛后 Handler 不直接持有连接池）；
// 测试可传 nil（不触达 devices / pm_metrics 端点时无需真实池）。
func NewHandler(counterRepo counter.CounterRepository, kpiRepo kpi.KPIRepository, kpiEngine *kpi.KPIEngine, taskRepo TaskRepository, fileStore PMFileStore, minioClient *minio.Client, pmBucket string, pool *pgxpool.Pool, indicatorRepo indicator.IndicatorRepository, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		counterRepo:   counterRepo,
		kpiRepo:       kpiRepo,
		kpiEngine:     kpiEngine,
		taskRepo:      taskRepo,
		fileStore:     fileStore,
		minioClient:   minioClient,
		pmBucket:      pmBucket,
		deviceQuery:   NewDeviceQueryService(pool),
		indicatorRepo: indicatorRepo,
		logger:        logger,
	}
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
	visibleGroups, ok := h.resolveVisibleGroups(c)
	if !ok {
		return
	}
	filter.VisibleGroups = visibleGroups
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		// #64：显式 device_id 越权直接 403，避免泄露"设备存在但无权"信号。
		inScope, ok := h.requestedDeviceInScope(c, id, visibleGroups)
		if !ok {
			return
		}
		if !inScope {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
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
	q.ListRequest = model.DefaultListRequest()
	if err := c.ShouldBindQuery(&q); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	filter := counter.CounterFilter{}
	visibleGroups, ok := h.resolveVisibleGroups(c)
	if !ok {
		return
	}
	filter.VisibleGroups = visibleGroups
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		inScope, ok := h.requestedDeviceInScope(c, id, visibleGroups)
		if !ok {
			return
		}
		if !inScope {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
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
//   - technology / technologies：制式过滤（lte/nr/gsm）
//   - start_time / end_time：RFC3339
//   - limit / offset
//   - page_by=pivot_row：limit / offset 按透视表行 key 分页，total 返回透视表行总数
//   - count_mode=n_plus_one：取 limit+1 判断 truncated，不执行精确 COUNT
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

	// #64 设备组数据权限：解析调用者可见分组，注入聚合查询（各维度在仓库层按 device_sn /
	// device_group_id 三态 fail-closed 收口）。
	visibleGroups, ok := h.resolveVisibleGroups(c)
	if !ok {
		return
	}
	req.VisibleGroups = visibleGroups

	if v := c.Query("device_oui"); v != "" {
		req.DeviceOUIs = []string{v}
	}
	if v := c.Query("device_sn"); v != "" {
		req.DeviceSNs = []string{v}
	}
	if v := c.Query("device_sns"); v != "" {
		sns := splitCSVNonEmpty(v)
		if len(sns) > maxAggregatedMetricDeviceSNs {
			response.Fail(c, http.StatusBadRequest, fmt.Sprintf("device_sns exceeds maximum of %d", maxAggregatedMetricDeviceSNs))
			return
		}
		if len(sns) > 0 {
			req.DeviceSNs = sns
		}
	}
	if v := c.Query("device_group_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_group_id")
			return
		}
		// #64：显式 device_group_id 不在可见集合内直接 403（超管 visibleGroups==nil 放行）。
		if !groupInVisible(id, visibleGroups) {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
			return
		}
		req.DeviceGroupIDs = []uuid.UUID{id}
	}
	if v := c.Query("metric_paths"); v != "" {
		paths := splitCSVNonEmpty(v)
		if len(paths) > maxAggregatedMetricPaths {
			response.Fail(c, http.StatusBadRequest, fmt.Sprintf("metric_paths exceeds maximum of %d", maxAggregatedMetricPaths))
			return
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
	if v := c.Query("technologies"); v != "" {
		req.Technologies = splitCSVNonEmpty(v)
	} else if v := c.Query("technology"); v != "" {
		req.Technologies = splitCSVNonEmpty(v)
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
	clientLimit := 0
	useNPlusOneCount := c.Query("count_mode") == "n_plus_one"
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			clientLimit = n
			req.Limit = n
			if useNPlusOneCount {
				req.Limit = n + 1
			}
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			req.Offset = n
		}
	}
	if c.Query("page_by") == "pivot_row" {
		req.PageByPivotRow = true
	}
	// #599：星期/小时段后端过滤（逗号分隔 int 列表，全选/空 = 不过滤）。
	if v := c.Query("weekdays"); v != "" {
		req.Weekdays = parseCSVInts(v)
	}
	if v := c.Query("hours"); v != "" {
		req.Hours = parseCSVInts(v)
	}
	req.CalendarTimezone = calendarfilter.ProviderName(c.Request.Context(), h.timezone)
	// #619：测量对象（object_ldn）后端过滤。
	// LDN 值自身合法含逗号（如 Cellid=x,PLMN=y），不能 CSV split——
	// 前端以「重复键」形态发 ?object_ldns=a&object_ldns=b（与 #401 修复同模式），后端用 QueryArray 整值取回。
	if vs := c.QueryArray("object_ldns"); len(vs) > 0 {
		ldns := make([]string, 0, len(vs))
		for _, p := range vs {
			if p = strings.TrimSpace(p); p != "" {
				ldns = append(ldns, p)
			}
		}
		if len(ldns) > 0 {
			req.ObjectLDNs = ldns
		}
	}

	queryCtx, queryCancel := context.WithTimeout(c.Request.Context(), aggregatedMetricsQueryTimeout)
	defer queryCancel()

	timing := aggregatedMetricsTiming{startedAt: time.Now()}
	defer func() {
		h.logAggregatedMetricsTiming(c, req, &timing)
	}()

	// fill_empty=true：数据驱动补齐占位行（T-0192d）。只对"已存在真实记录组"里所查
	// 但缺失的指标补一行占位，让透视表能区分"该时段有采样但此指标无值"与"此指标有值"。
	// 没有任何真实行的时间桶/object 永不出现（空时段不凭空造桶）。
	// 仅 device 维度（单 OUI+SN）+ metric_paths 非空时启用。
	fillEmpty := c.Query("fill_empty") == "true"
	objectLDNsDiscovered := false
	if fillEmpty && req.PageByPivotRow && aggregator.CanAutoDiscoverObjectSkeletonRequest(req) {
		stageStarted := time.Now()
		objectLDNs, err := h.aggr.DiscoverObjectLDNs(queryCtx, req)
		timing.discoverObjectLDNs = time.Since(stageStarted)
		timing.discoverObjectLDNRuns++
		timing.discoveredObjects = len(objectLDNs)
		objectLDNsDiscovered = true
		if err != nil {
			h.abortAggregatedMetricsQueryError(c, queryCtx, err)
			return
		}
		req.ObjectLDNs = objectLDNs
	}
	if fillEmpty && req.PageByPivotRow && aggregator.IsExplicitObjectSkeletonRequest(req) {
		stageStarted := time.Now()
		pivotKeys, err := h.aggr.DevicePivotRowKeys(queryCtx, req)
		timing.devicePivotRowKeys = time.Since(stageStarted)
		timing.pivotKeys = len(pivotKeys)
		if err != nil {
			h.abortAggregatedMetricsQueryError(c, queryCtx, err)
			return
		}
		req.PivotRowKeys = pivotKeys
	}

	stageStarted := time.Now()
	rows, err := h.aggr.Query(queryCtx, req)
	timing.query = time.Since(stageStarted)
	timing.queryRows = len(rows)
	if err != nil {
		h.abortAggregatedMetricsQueryError(c, queryCtx, err)
		return
	}
	var truncated bool
	if useNPlusOneCount && clientLimit > 0 {
		rows, truncated = truncateAggregatedRows(rows, clientLimit, req.PageByPivotRow)
		req.Limit = clientLimit
		if len(req.PivotRowKeys) > clientLimit {
			req.PivotRowKeys = req.PivotRowKeys[:clientLimit]
		}
	}
	if fillEmpty {
		if !objectLDNsDiscovered && aggregator.CanAutoDiscoverObjectSkeletonRequest(req) {
			stageStarted := time.Now()
			objectLDNs, err := h.aggr.DiscoverObjectLDNs(queryCtx, req)
			timing.discoverObjectLDNs += time.Since(stageStarted)
			timing.discoverObjectLDNRuns++
			timing.discoveredObjects = len(objectLDNs)
			objectLDNsDiscovered = true
			if err != nil {
				h.abortAggregatedMetricsQueryError(c, queryCtx, err)
				return
			}
			req.ObjectLDNs = objectLDNs
		}
		stageStarted := time.Now()
		rows = fillEmptyBuckets(rows, req)
		timing.fillEmpty = time.Since(stageStarted)
		timing.filledRows = len(rows)
		// 占位行可能因「该指标本次无任何真实行」而 DisplayName 为空（fillEmptyBuckets 的 nameByPath
		// 只从真实行收集）；整体按指标库再回填一次，使占位行与真实行同口径取名，避免透视表列头
		// 退化成裸编号（查不到名的合成计数器仍回退编号本身，行为不变）。
		stageStarted = time.Now()
		h.aggr.BackfillDisplayNames(queryCtx, rows)
		timing.backfillDisplayNames = time.Since(stageStarted)
	}
	total := len(rows)
	if !useNPlusOneCount && req.Limit > 0 {
		countReq := req
		// page_by=pivot_row + fill_empty 的页面语义是「按对象骨架补齐透视行」。
		// 当所选指标本身没有真实行时，按 metric_path count 会得到 0；此时 total 必须按
		// 同设备/对象/时间桶的透视行骨架计数，才能和 UI 补出的行数一致。
		if req.PageByPivotRow && fillEmpty && aggregator.IsExplicitObjectSkeletonRequest(req) {
			countReq.MetricPaths = nil
			countReq.MetricType = nil
		}
		stageStarted := time.Now()
		if n, err := h.aggr.Count(queryCtx, countReq); err == nil {
			total = n
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(queryCtx.Err(), context.DeadlineExceeded) {
			h.abortAggregatedMetricsQueryError(c, queryCtx, err)
			return
		}
		timing.count = time.Since(stageStarted)
	}
	result := gin.H{"items": rows, "total": total, "truncated": truncated}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		win := aggregator.BuildBucketWindow(req)
		result["requested_start_time"] = win.RequestedStartTime
		result["requested_end_time"] = win.RequestedEndTime
		result["actual_start_time"] = nilIfZeroTime(win.ActualStartTime)
		result["actual_end_time"] = nilIfZeroTime(win.ActualEndTime)
		result["granularity"] = win.Granularity
		result["timezone"] = win.Timezone
	}
	response.OK(c, result)
}

func (h *Handler) abortAggregatedMetricsQueryError(c *gin.Context, queryCtx context.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(queryCtx.Err(), context.DeadlineExceeded) {
		timeoutErr := fmt.Errorf("pm aggregated metrics query exceeded %s: %w", aggregatedMetricsQueryTimeout, commonerrors.ErrTimeout)
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(timeoutErr), timeoutErr)
		return
	}
	commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
}

type aggregatedMetricsTiming struct {
	startedAt             time.Time
	discoverObjectLDNs    time.Duration
	discoverObjectLDNRuns int
	devicePivotRowKeys    time.Duration
	query                 time.Duration
	fillEmpty             time.Duration
	backfillDisplayNames  time.Duration
	count                 time.Duration
	discoveredObjects     int
	pivotKeys             int
	queryRows             int
	filledRows            int
}

func (h *Handler) logAggregatedMetricsTiming(c *gin.Context, req aggregator.QueryRequest, timing *aggregatedMetricsTiming) {
	logger := h.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	objectFilter := "none"
	switch {
	case len(req.ObjectLDNs) > 0:
		objectFilter = "explicit"
	case aggregator.CanAutoDiscoverObjectSkeletonRequest(req):
		objectFilter = "auto_discover"
	}
	logger.Info("pm aggregated metrics query timing",
		zap.String("granularity", string(req.Granularity)),
		zap.String("dimension", string(req.Dimension)),
		zap.Int("device_sn_count", len(req.DeviceSNs)),
		zap.Int("metric_path_count", len(req.MetricPaths)),
		zap.String("object_filter", objectFilter),
		zap.Bool("page_by_pivot_row", req.PageByPivotRow),
		zap.Bool("fill_empty", c.Query("fill_empty") == "true"),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
		zap.Int("discovered_object_count", timing.discoveredObjects),
		zap.Int("discover_object_ldns_runs", timing.discoverObjectLDNRuns),
		zap.Int("pivot_key_count", timing.pivotKeys),
		zap.Int("query_row_count", timing.queryRows),
		zap.Int("filled_row_count", timing.filledRows),
		zap.Duration("discover_object_ldns_duration", timing.discoverObjectLDNs),
		zap.Duration("device_pivot_row_keys_duration", timing.devicePivotRowKeys),
		zap.Duration("query_duration", timing.query),
		zap.Duration("fill_empty_duration", timing.fillEmpty),
		zap.Duration("backfill_display_names_duration", timing.backfillDisplayNames),
		zap.Duration("count_duration", timing.count),
		zap.Duration("total_duration", time.Since(timing.startedAt)),
		zap.Int("http_status", c.Writer.Status()),
	)
}

// fillEmptyBuckets 保留 pm 包内测试入口，真实实现收敛在 aggregator 包，供页面接口和导出共用。
func fillEmptyBuckets(rows []aggregator.Row, req aggregator.QueryRequest) []aggregator.Row {
	return aggregator.FillEmptyBuckets(rows, req)
}

func splitCSVNonEmpty(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func nilIfZeroTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func truncateAggregatedRows(rows []aggregator.Row, limit int, pageByPivotRow bool) ([]aggregator.Row, bool) {
	if limit <= 0 {
		return rows, false
	}
	if !pageByPivotRow {
		if len(rows) <= limit {
			return rows, false
		}
		return rows[:limit], true
	}
	seen := make(map[aggregator.PivotRowKey]struct{}, limit+1)
	ordered := make([]aggregator.PivotRowKey, 0, limit+1)
	out := make([]aggregator.Row, 0, len(rows))
	truncated := false
	allowed := make(map[aggregator.PivotRowKey]struct{}, limit)
	for _, row := range rows {
		objectLDN := ""
		if row.ObjectLDN != nil {
			objectLDN = *row.ObjectLDN
		}
		key := aggregator.PivotRowKey{
			DeviceOUI:   row.DeviceOUI,
			DeviceSN:    row.DeviceSN,
			ObjectLDN:   objectLDN,
			Granularity: row.Granularity,
			Time:        row.Time,
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			ordered = append(ordered, key)
			if len(ordered) > limit {
				truncated = true
			} else {
				allowed[key] = struct{}{}
			}
		}
		if _, ok := allowed[key]; ok {
			out = append(out, row)
		}
	}
	return out, truncated
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
	visibleGroups, ok := h.resolveVisibleGroups(c)
	if !ok {
		return
	}
	filter.VisibleGroups = visibleGroups
	if q.DeviceID != "" {
		id, err := uuid.Parse(q.DeviceID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid device_id")
			return
		}
		inScope, ok := h.requestedDeviceInScope(c, id, visibleGroups)
		if !ok {
			return
		}
		if !inScope {
			commonerrors.AbortWithError(c, http.StatusForbidden, commonerrors.ErrForbidden)
			return
		}
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
//   - only_enabled：可选开关（缺省 false）。true 时只返回 operator_code='default' 下已启用指标，
//     供性能查询、设备性能查看、首页 KPI 配置和自定义聚合候选使用。
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
	onlyEnabled := parseBoolQuery(c.Query("only_enabled")) ||
		parseBoolQuery(c.Query("is_enabled")) ||
		parseBoolQuery(c.Query("isEnabled"))

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

	loc := appcontext.GetLocale(c.Request.Context())
	items := make([]kpiDefinitionItem, 0)
	for _, dt := range dts {
		filter := indicator.IndicatorListFilter{
			DeviceType: string(dt),
		}
		if !includeCounters {
			isCounter := "0" // 仅 KPI（默认行为，不变）
			filter.IsCounter = &isCounter
		}
		if onlyEnabled {
			isEnabled := "1"
			operatorCode := "default"
			filter.IsEnabled = &isEnabled
			filter.OperatorCode = &operatorCode
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
				ID:             r.ID,
				IsCounter:      r.IsCounter,
				IndicatorLevel: derefOr(r.IndicatorLevel, ""),
				Name:           r.EnName,
				DisplayName:    localizedIndicatorName(loc, r.EnName, r.CnName, r.ID),
				Formula:        derefOr(r.Arithmetic, ""),
				Unit:           derefOr(r.UnitID, ""),
			})
		}
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

// kpiDefinitionItem 是 ListKPIDefinitions 的 handler 局部响应结构，
// 历史字段（name/display_name/formula/unit）与旧 model.KPIDefinition 的 JSON 形态一致，
// 额外补 id / is_counter。不污染共享的 model.KPIDefinition（KPI 计算引擎在用）。
type kpiDefinitionItem struct {
	ID             string `json:"id"`
	IsCounter      string `json:"is_counter"`
	IndicatorLevel string `json:"indicator_level"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Formula        string `json:"formula"`
	Unit           string `json:"unit"`
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

func localizedIndicatorName(loc appcontext.Locale, en string, cn *string, fallback string) string {
	cnName := derefOr(cn, "")
	if loc == appcontext.LocaleEN {
		if strings.TrimSpace(en) != "" {
			return en
		}
		if strings.TrimSpace(cnName) != "" {
			return cnName
		}
		return fallback
	}
	if strings.TrimSpace(cnName) != "" {
		return cnName
	}
	if strings.TrimSpace(en) != "" {
		return en
	}
	return fallback
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

	oui, sn, err := h.deviceQuery.LookupDeviceOUISN(c.Request.Context(), deviceID)
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
	if raw := strings.TrimSpace(c.Query("product_id")); raw != "" && h.productResolver != nil {
		pid, perr := uuid.Parse(raw)
		if perr != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid product_id: %w", perr))
			return
		}
		patterns, rerr := h.productResolver.GetPatternsByProductID(c.Request.Context(), pid)
		if rerr != nil {
			commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("resolve product patterns: %w", rerr))
			return
		}
		if len(patterns) == 0 {
			response.OK(c, model.NewListResponse[PMFileDeviceAggregate](nil, 0, filter.Page, filter.PageSize))
			return
		}
		filter.ProductClasses = patterns
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

	// issue #321：入库后原始 XML 被 gzip 压缩回写 MinIO 省盘，但对象键 / file_name 仍是 .xml。
	// 直接透传压缩字节会让用户下载到「.xml 实为 gzip」的乱码文件。嗅探 gzip 魔数：是压缩内容
	// 就把下载名补 .gz（让用户拿到可正常解压的 .xml.gz），明文 .xml 原样透传。注意不要设
	// Content-Encoding: gzip——否则浏览器会自动解压，与 .gz 文件名矛盾。Peek 不消耗数据，后续
	// DataFromReader 仍从头读满 stat.Size。
	filename := fileInfo.FileName
	contentType := "application/xml"
	br := bufio.NewReader(obj)
	if head, _ := br.Peek(2); compress.IsGzip(head) {
		if !strings.HasSuffix(strings.ToLower(filename), ".gz") {
			filename += ".gz"
		}
		contentType = "application/gzip"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", contentType)
	c.DataFromReader(http.StatusOK, stat.Size, contentType, br, nil)
}
