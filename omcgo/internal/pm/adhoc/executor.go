package adhoc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// AggregatorQuerier 是 Executor 对 G5 aggregator 的最小依赖（便于单测 stub）。
type AggregatorQuerier interface {
	Query(ctx context.Context, req aggregator.QueryRequest) ([]aggregator.Row, error)
}

// ProgressPublisher 把进度事件发布到事件总线（SSE handler 订阅）。
// 真实实现是 internal/core/event.EventBus 的 PublishObject 包装。
type ProgressPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// 事件主题。SSE handler 订阅这两个主题。
const (
	SubjectProgress  = "pm.adhoc.progress"
	SubjectCompleted = "pm.adhoc.completed"
)

// Executor 是 G7 任务执行核心。包多设备 × 多 metric × 多粒度的笛卡尔积聚合。
//
// 单次执行（ExecuteOneshot）：
//  1. 遍历所有 granularity
//     a. 调 aggregator.Query 拉源数据（含 device 维度过滤）
//     b. 转 ResultRow 写 pm_adhoc_aggregation_results
//     c. 每完成一个粒度 → UpdateStatus(progress) + Publish 进度事件
//  2. 全部完成后返 nil（caller 决定终态 succeeded/scheduled）
//
// 与 G5 cron 的关键区别：
//   - G5 跑全设备整桶；G7 跑用户指定 N 设备 + 任意时窗
//   - G5 写 pm_metrics_*；G7 写 pm_adhoc_aggregation_results（独立表，独立保留期）
//   - G5 聚合 source→target；G7 直接 SELECT 已聚合行（G5 跑过的）+ 透传到结果表
type Executor struct {
	aggr      AggregatorQuerier
	repo      Repository
	publisher ProgressPublisher
	logger    *zap.Logger
}

// NewExecutor 构造 Executor。publisher 可为 nil（不上报进度事件）。
func NewExecutor(aggr AggregatorQuerier, repo Repository, publisher ProgressPublisher, logger *zap.Logger) *Executor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Executor{aggr: aggr, repo: repo, publisher: publisher, logger: logger.Named("pm.adhoc.executor")}
}

// ExecuteOneshot 单次执行任务（不切换终态，由 caller 根据 mode 决定 succeeded/scheduled）。
//
// 返回写入的 ResultRow 行数（含所有粒度）。
func (e *Executor) ExecuteOneshot(ctx context.Context, task *Task) (int, error) {
	if len(task.Granularities) == 0 {
		return 0, fmt.Errorf("adhoc.ExecuteOneshot: task %s has no granularities", task.ID)
	}

	total := len(task.Granularities)
	totalRows := 0

	for i, gStr := range task.Granularities {
		g := metrics.Granularity(gStr)
		rows, err := e.queryAndConvert(ctx, task, g)
		if err != nil {
			return totalRows, fmt.Errorf("query %s: %w", g, err)
		}
		if len(rows) > 0 {
			if err := e.repo.InsertResults(ctx, rows); err != nil {
				return totalRows, fmt.Errorf("insert results for %s: %w", g, err)
			}
			totalRows += len(rows)
		}

		// 进度 = (已完成粒度 / 总粒度) × 100
		progress := (i + 1) * 100 / total
		if err := e.repo.UpdateStatus(ctx, task.ID, StatusRunning, &progress, ""); err != nil {
			e.logger.Warn("update progress failed",
				zap.String("task_id", task.ID.String()), zap.Error(err))
		}
		e.publishProgress(ctx, task.ID, progress, gStr, len(rows))
	}
	return totalRows, nil
}

// queryAndConvert 调 aggregator.Query 拉 1 个 granularity 的数据，转 ResultRow。
//
// device 维度过滤：DeviceSNs。OUI 未在 task 中存储，aggregator 接受空 OUI + 非空 SN
// 走 `device_sn IN (...)` 路径（参见 aggregator/query.go applyDeviceFilters）。
func (e *Executor) queryAndConvert(ctx context.Context, task *Task, g metrics.Granularity) ([]ResultRow, error) {
	req := aggregator.QueryRequest{
		Granularity: g,
		Dimension:   aggregator.DimensionDevice,
		DeviceSNs:   task.DeviceSNs,
		MetricPaths: task.MetricPaths,
		StartTime:   task.WindowStart,
		EndTime:     task.WindowEnd,
		Limit:       100000,
	}
	rows, err := e.aggr.Query(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]ResultRow, 0, len(rows))
	for _, r := range rows {
		var statisStr *string
		if r.StatisType != nil {
			s := string(*r.StatisType)
			statisStr = &s
		}
		out = append(out, ResultRow{
			TaskID:      task.ID,
			DeviceOUI:   r.DeviceOUI,
			DeviceSN:    r.DeviceSN,
			MetricPath:  r.MetricPath,
			MetricType:  string(r.MetricType),
			MetricValue: r.MetricValue,
			StatisType:  statisStr,
			Granularity: string(r.Granularity),
			Time:        r.Time,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
			ObjectLDN:   r.ObjectLDN,
			Extra:       r.Extra,
		})
	}
	return out, nil
}

func (e *Executor) publishProgress(ctx context.Context, taskID uuid.UUID, progress int, granularity string, rows int) {
	if e.publisher == nil {
		return
	}
	payload := map[string]any{
		"task_id":     taskID.String(),
		"progress":    progress,
		"granularity": granularity,
		"rows":        rows,
	}
	if err := e.publisher.Publish(ctx, SubjectProgress, payload); err != nil {
		e.logger.Warn("publish progress event failed",
			zap.String("task_id", taskID.String()), zap.Error(err))
	}
}

// PublishCompleted 任务完成时上报 (caller / worker 调用)。
func (e *Executor) PublishCompleted(ctx context.Context, taskID uuid.UUID, status Status, rowsTotal int, errMsg string) {
	if e.publisher == nil {
		return
	}
	payload := map[string]any{
		"task_id":    taskID.String(),
		"status":     string(status),
		"rows_total": rowsTotal,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	if err := e.publisher.Publish(ctx, SubjectCompleted, payload); err != nil {
		e.logger.Warn("publish completed event failed",
			zap.String("task_id", taskID.String()), zap.Error(err))
	}
}

// EventBusPublisher 把 internal/core/event.EventBus 适配为 ProgressPublisher。
type EventBusPublisher struct {
	Bus event.EventBus
}

func (p *EventBusPublisher) Publish(ctx context.Context, subject string, payload any) error {
	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		return err
	}
	return p.Bus.Publish(ctx, subject, evt)
}
