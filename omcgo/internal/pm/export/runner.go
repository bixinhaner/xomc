package export

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
)

// taskRepo 是 Runner 操作导出任务表的契约（载任务 + 三态切换），便于单测 stub。
type taskRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*Task, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	MarkSucceeded(ctx context.Context, id uuid.UUID, bucket, filePath string, fileSize, rowCount int64) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}

// Runner 是 pm_kpi_export 的 asyncjob.JobRunner 实现（T2 真生成）。
//
// 流程：载任务 → MarkRunning → 按 source_type 取数 → 流式写 CSV 直传对象存储 →
// MarkSucceeded 回填 file_path/file_size/row_count；任意环节出错 → MarkFailed + error 落库。
type Runner struct {
	repo     taskRepo
	aggr     *aggregator.Aggregator // dashboard 聚合维度取数 + KPI 反算
	metricDB PgQuerier              // PM 指标表（TimescaleDB）：dashboard device 维度直查 + 指标名解析
	adhocDB  PgQuerier              // 业务库：pm_adhoc_aggregation_results 直查
	uploader Uploader               // 对象存储上传（流式）
	bucket   string                 // 导出文件落地桶
	logger   *zap.Logger

	// buildSourceFn 取数源构造入口；默认 r.buildSource，单测可注入 stub 源绕过 DB。
	// 返回取数源 + 横表指标列集（列名已解析），列发现 / 名解析在构造期一次性完成。
	buildSourceFn func(ctx context.Context, task *Task) (RowSource, []WideColumn, error)
}

// RunnerDeps 是构造 T2 Runner 的依赖集合。
type RunnerDeps struct {
	Repo     taskRepo
	Aggr     *aggregator.Aggregator
	MetricDB PgQuerier
	AdhocDB  PgQuerier
	Uploader Uploader
	Bucket   string
	Logger   *zap.Logger
}

// NewRunner 构造 T2 Runner。
func NewRunner(d RunnerDeps) *Runner {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	r := &Runner{
		repo:     d.Repo,
		aggr:     d.Aggr,
		metricDB: d.MetricDB,
		adhocDB:  d.AdhocDB,
		uploader: d.Uploader,
		bucket:   d.Bucket,
		logger:   logger.Named("pm.export.runner"),
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

	src, cols, err := r.buildSourceFn(ctx, task)
	if err != nil {
		return genResult{}, err
	}

	object := objectPath(task)
	out, err := streamCSVToObject(ctx, r.uploader, r.bucket, object, src, cols)
	if err != nil {
		return genResult{}, err
	}
	return genResult{filePath: object, fileSize: out.FileSize, rowCount: out.RowCount}, nil
}

// buildSource 按 source_type 构造取数源 + 横表指标列集（含已解析列名）。
//
// 横表列集在此一次性发现（DISTINCT metric_path/metric_type）+ 解析名（三张指标表 UNION 按 locale），
// 表头开头即知；流式阶段只摊行不再查名。
func (r *Runner) buildSource(ctx context.Context, task *Task) (RowSource, []WideColumn, error) {
	loc := appcontext.GetLocale(ctx)
	switch task.SourceType {
	case SourceDashboard:
		req, objectLDNs, err := parseDashboardParams(task.Params)
		if err != nil {
			return nil, nil, err
		}
		dim := req.Dimension
		if dim == "" {
			dim = aggregator.DimensionDevice
		}
		table, terr := aggregator.SelectTable(req.Granularity, dim)
		if terr != nil {
			return nil, nil, terr
		}
		// 发现列集（编号+类型，与设备/小区无关）→ 解析本地化列名。
		keys, derr := discoverMetricColumns(ctx, r.metricDB, table, req.MetricPaths, req.StartTime, req.EndTime)
		if derr != nil {
			return nil, nil, derr
		}
		cols := newNameResolver(r.metricDB, loc).resolveColumns(ctx, keys)

		// device 维度且表含行级 id → (time,id) keyset 直查；否则（聚合维度 / 无 id 的 device 表）走聚合批次游标。
		if dim == aggregator.DimensionDevice && tableHasIDColumn(table) {
			return newDashboardDeviceSource(r.metricDB, table, req, objectLDNs), cols, nil
		}
		if r.aggr == nil {
			return nil, nil, fmt.Errorf("aggregator not wired for dashboard aggregate export")
		}
		return newDashboardAggregateSource(r.aggr, req, objectLDNs), cols, nil

	case SourceAdhoc:
		taskID, startTime, endTime, err := parseAdhocParams(task.Params)
		if err != nil {
			return nil, nil, err
		}
		keys, derr := discoverAdhocColumns(ctx, r.adhocDB, taskID, startTime, endTime)
		if derr != nil {
			return nil, nil, derr
		}
		cols := newNameResolver(r.adhocDB, loc).resolveColumns(ctx, keys)
		return newAdhocSource(r.adhocDB, taskID, startTime, endTime), cols, nil

	default:
		return nil, nil, fmt.Errorf("export runner: unsupported source_type %q", task.SourceType)
	}
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
