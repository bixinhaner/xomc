package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/calendarfilter"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/storageprotection"
)

// taskRepo 是 Runner 操作导出任务表的契约（载任务 + 三态切换），便于单测 stub。
type taskRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*Task, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	MarkSucceeded(ctx context.Context, id uuid.UUID, bucket, filePath string, fileSize, rowCount int64) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// TimezoneProvider 是系统时区读取的最小契约。
// Provider 或 Location 为 nil 时，Runner 在导出展示层回退 UTC。
type TimezoneProvider interface {
	Location(ctx context.Context) *time.Location
}

// Runner 是 pm_kpi_export 的 asyncjob.JobRunner 实现（T2 真生成）。
//
// 流程：载任务 → MarkRunning → 按 source_type 取数 → 流式写 CSV 直传对象存储 →
// MarkSucceeded 回填 file_path/file_size/row_count；任意环节出错 → MarkFailed + error 落库。
type Runner struct {
	repo     taskRepo
	aggr     *aggregator.Aggregator // dashboard 聚合维度取数 + KPI 反算
	metricDB PgQuerier              // 时序库（TsPool）：dashboard device 维度直查 pm_metrics + 指标名解析
	adhocDB  PgQuerier              // 时序库（TsPool）：pm_adhoc_aggregation_results 直查 + 指标名解析
	// taskMetaDB 主库（PgPool）：pm_tasks 元数据读（loadAdhocDimension 取 adhoc 任务维度/设备数）。
	// KPI/时序库物理分离后 pm_adhoc_aggregation_results 在 TsPool、pm_tasks 在 PgPool，二者拆池。
	taskMetaDB PgQuerier
	uploader   Uploader // 对象存储上传（流式）
	bucket     string   // 导出文件落地桶
	logger     *zap.Logger
	timezone   TimezoneProvider
	admission  storageprotection.WriteAdmission

	// buildSourceFn 取数源构造入口；默认 r.buildSource，单测可注入 stub 源绕过 DB。
	// 返回取数源 + 横表指标列集（列名已解析）+ CSV 列布局（首列表头 / 是否含小区列），
	// 列发现 / 名解析 / 维度判定在构造期一次性完成。
	buildSourceFn func(ctx context.Context, task *Task) (RowSource, []WideColumn, csvLayout, error)
}

// RunnerDeps 是构造 T2 Runner 的依赖集合。
//
// 池路由（KPI/时序库物理分离）：
//   - MetricDB / AdhocDB = 时序库（TsPool）：dashboard device 维度查 pm_metrics、
//     adhoc 查 pm_adhoc_aggregation_results（两表均在时序库）。
//   - TaskMetaDB = 主库（PgPool）：loadAdhocDimension 读 pm_tasks（任务元数据留主库）。
type RunnerDeps struct {
	Repo             taskRepo
	Aggr             *aggregator.Aggregator
	MetricDB         PgQuerier
	AdhocDB          PgQuerier
	TaskMetaDB       PgQuerier
	Uploader         Uploader
	Bucket           string
	Logger           *zap.Logger
	TimezoneProvider TimezoneProvider
	StorageAdmission storageprotection.WriteAdmission
}

// NewRunner 构造 T2 Runner。
func NewRunner(d RunnerDeps) *Runner {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	r := &Runner{
		repo:       d.Repo,
		aggr:       d.Aggr,
		metricDB:   d.MetricDB,
		adhocDB:    d.AdhocDB,
		taskMetaDB: d.TaskMetaDB,
		uploader:   d.Uploader,
		bucket:     d.Bucket,
		logger:     logger.Named("pm.export.runner"),
		timezone:   d.TimezoneProvider,
		admission:  d.StorageAdmission,
	}
	r.buildSourceFn = r.buildSource
	return r
}

var _ asyncjob.JobRunner = (*Runner)(nil)

// JobType 返回 asyncjob 注册主键。
func (r *Runner) JobType() string { return JobType }

// Run 实现 asyncjob.JobRunner.Run（T2 真生成）。
//
// 注意：MarkRunning 之后的任何失败都会落 MarkFailed（任务进 failed + error 落库），
// 然后 Run 返回 nil——让 asyncjob 框架把 job 记成已处理、不再重试（避免反复失败卡队列）。
// MarkRunning 之前的失败（payload 坏 / 任务不存在 / 已被抢）返回 error 走框架常规路径。
func (r *Runner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	var p JobPayload
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return nil, fmt.Errorf("export runner: unmarshal payload: %w", err)
		}
	}
	if p.ExportTaskID == "" {
		return nil, fmt.Errorf("export runner: payload missing export_task_id")
	}
	taskID, err := uuid.Parse(p.ExportTaskID)
	if err != nil {
		return nil, fmt.Errorf("export runner: invalid export_task_id %q: %w", p.ExportTaskID, err)
	}

	task, err := r.repo.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("export runner: load task: %w", err)
	}

	if err := r.repo.MarkRunning(ctx, taskID); err != nil {
		return nil, fmt.Errorf("export runner: mark running: %w", err)
	}

	// 自此进入"已 running"区：失败统一落 MarkFailed 并吞掉 job（返回 nil 不重试）。
	res, genErr := r.generate(ctx, task)
	if genErr != nil {
		r.logger.Warn("kpi export generate failed; mark task failed",
			zap.String("export_task_id", taskID.String()), zap.Error(genErr))
		if mfErr := r.repo.MarkFailed(ctx, taskID, genErr.Error()); mfErr != nil {
			// 落 failed 都失败：返回 error 让框架重试（DB 可能瞬时不可用）。
			return nil, fmt.Errorf("export runner: mark failed after generate error %v: %w", genErr, mfErr)
		}
		out, _ := json.Marshal(map[string]any{"export_task_id": taskID.String(), "status": "failed"})
		return out, nil
	}

	if err := r.repo.MarkSucceeded(ctx, taskID, r.bucket, res.filePath, res.fileSize, res.rowCount); err != nil {
		return nil, fmt.Errorf("export runner: mark succeeded: %w", err)
	}
	r.logger.Info("kpi export succeeded",
		zap.String("export_task_id", taskID.String()),
		zap.String("file_path", res.filePath),
		zap.Int64("row_count", res.rowCount),
		zap.Int64("file_size", res.fileSize))
	out, _ := json.Marshal(map[string]any{
		"export_task_id": taskID.String(),
		"status":         "succeeded",
		"row_count":      res.rowCount,
		"file_size":      res.fileSize,
	})
	return out, nil
}

type genResult struct {
	filePath string
	fileSize int64
	rowCount int64
}

// generate 按 source_type 选源 → 流式生成 CSV 上传 → 返回回填值。
func (r *Runner) generate(ctx context.Context, task *Task) (genResult, error) {
	if r.uploader == nil {
		return genResult{}, fmt.Errorf("object storage uploader not wired")
	}
	if r.bucket == "" {
		return genResult{}, fmt.Errorf("export bucket not configured")
	}
	if r.admission != nil {
		decision, err := r.admission.Check(ctx, storageprotection.TargetFilesystem, storageprotection.UnifiedStorageTargetID, storageprotection.WriteScopePM)
		if err != nil {
			return genResult{}, fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return genResult{}, fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}

	src, cols, layout, err := r.buildSourceFn(ctx, task)
	if err != nil {
		return genResult{}, err
	}

	object := objectPath(task)
	out, err := streamCSVToObject(ctx, r.uploader, r.bucket, object, src, cols, layout, r.outputLocation(ctx))
	if err != nil {
		return genResult{}, err
	}
	return genResult{filePath: object, fileSize: out.FileSize, rowCount: out.RowCount}, nil
}

func (r *Runner) outputLocation(ctx context.Context) *time.Location {
	if r.timezone == nil {
		return time.UTC
	}
	loc := r.timezone.Location(ctx)
	if loc == nil {
		return time.UTC
	}
	return loc
}

// buildSource 按 source_type 构造取数源 + 横表指标列集（含已解析列名）。
//
// 横表列集在此一次性发现（DISTINCT metric_path/metric_type）+ 解析名（三张指标表 UNION 按 locale），
// 表头开头即知；流式阶段只摊行不再查名。
func (r *Runner) buildSource(ctx context.Context, task *Task) (RowSource, []WideColumn, csvLayout, error) {
	loc := exportLocale(task.Params)
	deviceHeader := "设备 SN"
	if loc == appcontext.LocaleEN {
		deviceHeader = "Device SN"
	}
	// dashboard 路径恒为 device 维度：首列「设备 SN」+ 含 Cell ID/PLMN 列（保持仪表盘既有导出口径）。
	// 缺值与 adhoc 导出保持一致写 "-"，避免 CSV 空单元格被误读为未导出。
	dashboardLayout := csvLayout{
		FirstColHeader:                deviceHeader,
		Locale:                        loc,
		IncludeCell:                   true,
		MissingMetricValuePlaceholder: missingMetricValuePlaceholder,
	}
	deviceViewLayout := csvLayout{
		FirstColHeader:                deviceHeader,
		Locale:                        loc,
		IncludeMeasurementObject:      true,
		MissingMetricValuePlaceholder: missingMetricValuePlaceholder,
	}
	kpiQueryLayout := csvLayout{
		FirstColHeader:                deviceHeader,
		Locale:                        loc,
		IncludeMeasurementObject:      true,
		MissingMetricValuePlaceholder: missingMetricValuePlaceholder,
	}
	switch task.SourceType {
	case SourceDashboard:
		return r.buildDashboardLikeSource(ctx, task, loc, dashboardLayout)

	case SourceDeviceView:
		return r.buildDashboardLikeSource(ctx, task, loc, deviceViewLayout)

	case SourceKpiQuery:
		return r.buildDashboardLikeSource(ctx, task, loc, kpiQueryLayout)

	case SourcePMDashboard, SourceAdhocResult:
		return r.buildAdhocResultSource(ctx, task, loc)

	default:
		return nil, nil, csvLayout{}, fmt.Errorf("export runner: unsupported source_type %q", task.SourceType)
	}
}

func (r *Runner) buildAdhocResultSource(ctx context.Context, task *Task, loc appcontext.Locale) (RowSource, []WideColumn, csvLayout, error) {
	filter, err := parseAdhocParams(task.Params)
	if err != nil {
		return nil, nil, csvLayout{}, err
	}
	filter.CalendarTimezone = calendarfilter.ProviderName(ctx, r.timezone)
	// 先查任务聚合维度、圈选设备数和配置指标集：决定首列表头 / 对象名解析口径，
	// 并确保全量落库模式下导出仍只包含任务配置的 N 个指标。
	// pm_tasks 在主库（PgPool），用 taskMetaDB 读；adhoc 结果表查询走 adhocDB（TsPool）。
	meta, derr := loadAdhocTaskMeta(ctx, r.taskMetaDB, filter.TaskID)
	if derr != nil {
		return nil, nil, csvLayout{}, derr
	}
	// 列发现与真实行查询必须共享同一规范化指标集，避免表头与数据 SQL 因空格/重复/全空白分叉。
	metricPaths := normalizeMetricPaths(meta.metricPaths)
	keys, kerr := discoverAdhocColumns(ctx, r.adhocDB, filter.TaskID, metricPaths, filter)
	if kerr != nil {
		return nil, nil, csvLayout{}, kerr
	}
	cols := newNameResolver(r.adhocDB, loc).resolveColumns(ctx, keys)
	layout := csvLayout{
		FirstColHeader:                adhocFirstColHeader(meta.dimension, loc),
		Locale:                        loc,
		IncludeTechnology:             meta.dimension == "device_group", // 设备组维度按制式分行，导出补「制式」列（与页面表格一致）
		IncludeCell:                   adhocIncludesCell(meta.dimension),
		MissingMetricValuePlaceholder: missingMetricValuePlaceholder,
	}
	return newAdhocSource(r.adhocDB, filter.TaskID, metricPaths, filter, meta.dimension, meta.deviceCount, loc), cols, layout, nil
}

func (r *Runner) buildDashboardLikeSource(ctx context.Context, task *Task, loc appcontext.Locale, layout csvLayout) (RowSource, []WideColumn, csvLayout, error) {
	req, objectLDNs, err := parseDashboardParams(task.Params)
	if err != nil {
		return nil, nil, csvLayout{}, err
	}
	req.CalendarTimezone = calendarfilter.ProviderName(ctx, r.timezone)
	req = normalizeStoredResultExportRequest(req)
	dim := req.Dimension
	if dim == "" {
		dim = aggregator.DimensionDevice
	}
	table, terr := aggregator.SelectTable(req.Granularity, dim)
	if terr != nil {
		return nil, nil, csvLayout{}, terr
	}
	if shouldAutoDiscoverExportSkeleton(task.SourceType, req) {
		if r.aggr == nil {
			return nil, nil, csvLayout{}, fmt.Errorf("aggregator not wired for kpi query skeleton export")
		}
		discovered, aerr := r.aggr.DiscoverObjectLDNs(ctx, req)
		if aerr != nil {
			return nil, nil, csvLayout{}, aerr
		}
		req.ObjectLDNs = discovered
		objectLDNs = discovered
	}
	// 发现列集（编号+类型，与设备/小区无关）→ 解析本地化列名。
	keys, derr := discoverMetricColumns(ctx, r.metricDB, table, req)
	if derr != nil {
		return nil, nil, csvLayout{}, derr
	}
	cols := newNameResolver(r.metricDB, loc).resolveColumns(ctx, keys)

	// 普通 device 导出只读取已经落库的 KPI/counter 结果，不进入 aggregator.Query 的
	// KPI 现场重算路径。含行级 id 的表用 (time,id) keyset；无 id 的日/周/月表用稳定排序 offset。
	if dim == aggregator.DimensionDevice {
		var src RowSource
		if tableHasIDColumn(table) {
			src = newDashboardDeviceSource(r.metricDB, table, req, objectLDNs)
		} else {
			src = newDashboardDeviceOffsetSource(r.metricDB, table, req, objectLDNs)
		}
		if shouldFillExportSkeleton(task.SourceType, req) {
			src = newFillEmptySource(src, req)
		}
		return src, cols, layout, nil
	}
	if r.aggr == nil {
		return nil, nil, csvLayout{}, fmt.Errorf("aggregator not wired for dashboard aggregate export")
	}
	src := RowSource(newDashboardAggregateSource(r.aggr, req, objectLDNs))
	if shouldFillExportSkeleton(task.SourceType, req) {
		src = newFillEmptySource(src, req)
	}
	return src, cols, layout, nil
}

func shouldAutoDiscoverExportSkeleton(source SourceType, req aggregator.QueryRequest) bool {
	// Hourly KPI query export must match the query page table. The page exports
	// the submitted filter result rows, not a synthetic full time/object skeleton.
	return source == SourceKpiQuery &&
		req.Granularity != metrics.GranularityHourly &&
		aggregator.CanAutoDiscoverObjectSkeletonRequest(req)
}

func shouldFillExportSkeleton(source SourceType, req aggregator.QueryRequest) bool {
	// Keep hourly export row counts aligned with the visible KPI query result.
	return source == SourceKpiQuery &&
		req.Granularity != metrics.GranularityHourly &&
		aggregator.IsExplicitObjectSkeletonRequest(req)
}

type adhocTaskMeta struct {
	dimension   string
	deviceCount int
	metricPaths []string
}

// loadAdhocTaskMeta 读 adhoc 任务的聚合维度、圈选设备数与配置指标集。
// pm_tasks.id 是主键，按 id 直查不依赖子类型过滤。dimension 空 → 退化 "device"；
// 任务查不到 → 同样退化 "device"（容错，不让导出整体失败）。
func loadAdhocTaskMeta(ctx context.Context, db PgQuerier, taskID uuid.UUID) (adhocTaskMeta, error) {
	const q = `SELECT COALESCE(dimension, ''), COALESCE(jsonb_array_length(device_sns), 0), COALESCE(metric_paths, '{}'::text[]) FROM pm_tasks WHERE id = $1`
	var meta adhocTaskMeta
	if err := db.QueryRow(ctx, q, taskID).Scan(&meta.dimension, &meta.deviceCount, &meta.metricPaths); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adhocTaskMeta{dimension: "device"}, nil
		}
		return adhocTaskMeta{}, fmt.Errorf("export adhoc load task metadata: %w", err)
	}
	if meta.dimension == "" {
		meta.dimension = "device"
	}
	return meta, nil
}

// objectPath 约定 object path：kpi-export/{task_id}/kpi_{source}_{timestamp}.csv（设计 §5.6）。
func objectPath(task *Task) string {
	return fmt.Sprintf("kpi-export/%s/kpi_%s_%s.csv",
		task.ID.String(), task.SourceType, time.Now().Format("20060102_150405"))
}

// tableHasIDColumn 标识哪些 device 维度表含行级 id 列（可做 (time,id) keyset）。
// pm_metrics（15min raw）、pm_metrics_hourly 含 id；daily/weekly/monthly 为复合 PK 无 id。
func tableHasIDColumn(table string) bool {
	return table == "pm_metrics" || table == "pm_metrics_hourly"
}
