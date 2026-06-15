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

	// storeAllMetrics 控制落库范围（T-0182，全局配置 pm.storage.store_all_metrics）：
	//   - true（默认）：聚合结果全部指标都落库，便于事后改任务指标集时无需重算
	//   - false（仅存所选）：落库前按 task.MetricPaths 过滤，只存任务定义的 N 个指标
	// 注意：此开关只影响"落哪些指标"，不影响"查看/导出限 N 个"那条收口规则（设计 §2.4）。
	storeAllMetrics bool
}

// NewExecutor 构造 Executor。publisher 可为 nil（不上报进度事件）。
// storeAllMetrics 默认 true（见字段说明），调用方可用 SetStoreAllMetrics 覆盖。
func NewExecutor(aggr AggregatorQuerier, repo Repository, publisher ProgressPublisher, logger *zap.Logger) *Executor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Executor{
		aggr:            aggr,
		repo:            repo,
		publisher:       publisher,
		logger:          logger.Named("pm.adhoc.executor"),
		storeAllMetrics: true,
	}
}

// SetStoreAllMetrics 配置落库范围开关（worker 启动期按 pm.storage.store_all_metrics 注入）。
// 返回自身便于链式调用。
func (e *Executor) SetStoreAllMetrics(v bool) *Executor {
	e.storeAllMetrics = v
	return e
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
		// 存储范围开关：仅存所选时落库前按 task.MetricPaths 过滤（设计 §2.4）。
		if !e.storeAllMetrics {
			rows = filterByMetricPaths(rows, task.MetricPaths)
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
	// 默认 device 维度（兼容老任务）；其余维度按 task.Dimension 映射到 aggregator 侧枚举。
	dim := aggregator.DimensionDevice
	switch task.Dimension {
	case DimensionAggregateGroup:
		dim = aggregator.DimensionAggregateGroup
	case DimensionProduct:
		dim = aggregator.DimensionProduct
	case DimensionBand:
		dim = aggregator.DimensionBand
	case DimensionNetwork:
		// T-0184：全网维度，现场汇总成一条总线（仅制式过滤）。
		dim = aggregator.DimensionNetwork
	case DimensionDeviceGroup:
		// T-0184：设备组维度，复用 G5 设备组预聚合（pm_group_metrics_*）。
		// 设备组制式治本（B 方案）：快表按「组 × 制式」拆行 → 每组每制式一条；
		// DeviceGroupIDs 留空 = 按全部组分组；制式过滤在查询侧按 technology 列直接筛（不再走设备编号子查询）。
		dim = aggregator.DimensionDeviceGroup
	}
	var techs []string
	if task.Technology != "" {
		techs = []string{task.Technology}
	}
	req := aggregator.QueryRequest{
		Granularity:  g,
		Dimension:    dim,
		DeviceSNs:    task.DeviceSNs,
		MetricPaths:  task.MetricPaths,
		Technologies: techs,
		StartTime:    task.WindowStart,
		EndTime:      task.WindowEnd,
		Limit:        100000,
	}
	// KPI-ALL-IND：全网维度且指标列表为空（全聚到全库）时，让聚合层连派生 KPI 一起重算落库，
	// 使首页读现成全网预聚合表时 KPI 也有线（否则全聚只产 counter、首页 KPI 面板空线）。
	// 仅 network 维度（设备组/产品/频段仍按各自精选列表，不受影响）。
	if dim == aggregator.DimensionNetwork && len(task.MetricPaths) == 0 {
		req.RecomputeAllKPIs = true
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
		ldn := r.ObjectLDN
		// T-0184：device_group 维度结果以组 id 为身份键。结果表无独立 device_group_id 列，
		// 复用 object_ldn 承载（形如 'DeviceGroup=<uuid>'，与 band 维度 'Band=<值>' 同范式），
		// 不新增迁移列。其它维度透传 aggregator 原 ObjectLDN（device/band 等）。
		//
		// 设备组制式治本（B 方案）：设备组结果按「组 × 制式」拆行，把制式也编进 object_ldn
		// → 'DeviceGroup=<uuid>,Tech=<lte|nr|gsm>'。结果表业务唯一键里 object_ldn 已是分组身份的
		// 一部分 → 同组不同制式天然落不同行不互相覆盖，结果表零迁移、业务键零改动。
		if task.Dimension == DimensionDeviceGroup && r.DeviceGroupID != uuid.Nil {
			s := "DeviceGroup=" + r.DeviceGroupID.String()
			if r.Technology != "" {
				s += ",Tech=" + r.Technology
			}
			ldn = &s
		}
		out = append(out, ResultRow{
			TaskID:      task.ID,
			DeviceOUI:   r.DeviceOUI,
			DeviceSN:    r.DeviceSN,
			ProductID:   r.ProductID, // T-0182-fix: product 维度分组键透传（device/aggregate_group 维度为 Nil）
			MetricPath:  r.MetricPath,
			MetricType:  string(r.MetricType),
			MetricValue: float64(r.MetricValue),
			StatisType:  statisStr,
			Granularity: string(r.Granularity),
			Time:        r.Time,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
			ObjectLDN:   ldn,
			Extra:       r.Extra,
		})
	}
	return out, nil
}

// filterByMetricPaths 只保留 metric_path 在 allowed 集合里的结果行（"仅存所选"模式用）。
// allowed 为空时不过滤（视为不限制）。
func filterByMetricPaths(rows []ResultRow, allowed []string) []ResultRow {
	if len(allowed) == 0 {
		return rows
	}
	set := make(map[string]struct{}, len(allowed))
	for _, p := range allowed {
		set[p] = struct{}{}
	}
	out := rows[:0]
	for _, r := range rows {
		if _, ok := set[r.MetricPath]; ok {
			out = append(out, r)
		}
	}
	return out
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
