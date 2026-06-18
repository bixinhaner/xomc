package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
)

// startPMAdhocPipeline wire 起 T-0164-P7 / G7 自定义聚合任务运行环境。
//
// 三个组件：
//  1. Repository（共用 pm_tasks 表，独立 ORM 视图避免破坏 pm.TaskRepository）
//  2. 4 个 Worker goroutine 抢 pending 任务跑 ExecuteOneshot
//  3. ContinuousScheduler 单 goroutine 每分钟扫 scheduled 任务，按 cron_expr 评估
//     是否到下一 tick → 切回 pending 让 worker 再跑
//
// 与 G5 cron 的关系：共享同一 aggregator + kpiRouter；G5 跑产品默认聚合，G7 跑用户定制任务，
// 数据落不同的表（pm_metrics_* vs pm_adhoc_aggregation_results），无写竞争。
//
// 设计 §4.7 锁定：G7 **不**进 G8 async_jobs 框架；走 pm_tasks per-module 表。
func startPMAdhocPipeline(
	ctx context.Context,
	w *workerInfra,
	kpiRouter *router.Router,
	cfg *appconfig.WorkerConfig,
	tz *tzManager,
) {
	logger := w.Logger.Named("pm-adhoc")

	// Repository + Executor
	// KPI/时序库物理分离：pm_tasks 留主库（PgPool），pm_adhoc_aggregation_results 迁时序库（TsPool），双池。
	// #528 P2：持续任务取数口径改为「上游完成水位」驱动（替换固定 −2 格猜测）。
	// 水位表在主库（PgPool），与上游 runner 写水位的库一致。
	watermarks := aggregator.NewWatermarkRepository(w.PgPool)
	// #528 P3：repo 也注入水位读取器 + 业务时区——新建持续任务初始游标 = 建任务时刻当前水位桶起点
	//（从「现在」起算，不回扫历史）。
	repo := adhoc.NewPgRepository(w.PgPool, w.TsPool).
		SetWatermarkReader(watermarks).
		SetLocationFunc(tz.Current)
	aggr := aggregator.NewWithPool(w.TsPool, kpiRouter, logger)
	publisher := &adhoc.EventBusPublisher{Bus: w.EventBus}
	// T-0182：存储范围全局开关（全存默认 / 仅存所选）。
	// ISSUE-398：持续任务「最近一格」窗口的 daily/weekly/monthly 零点对齐用同一 PM 业务时区。
	// #458：executor 经 SetLocationFunc 实时读当前业务时区（与 cron 调度读同一 sys_configs 源），
	// 管理员改时区后持续任务「最近一格」窗口下次即用新时区零点对齐、无需重启。
	executor := adhoc.NewExecutor(aggr, repo, publisher, logger).
		SetStoreAllMetrics(cfg.PM.Storage.StoreAllMetrics).
		SetLocationFunc(tz.Current).
		SetWatermarkReader(watermarks)

	// 4 worker goroutine（共享 repo，LockNextPending SKIP LOCKED 保证不重复抢同一行）
	hostname := buildLockOwner()
	const workerCount = 4
	for i := 0; i < workerCount; i++ {
		owner := fmt.Sprintf("%s-adhoc-%d", hostname, i)
		worker := adhoc.NewWorker(repo, executor, owner, 0, logger)
		go worker.Run(ctx)
	}
	logger.Info("adhoc worker pool started", zap.Int("workers", workerCount))

	// continuous scheduler — 周期把 scheduled cron 任务转 pending
	// #528 P3：注入水位 gate + 业务时区——追平上界由真实水位决定（不越过水位、不空磨史前格、
	// 不读 #479 半成品格），而非旧墙钟上界。
	contRepo := adhoc.NewPgContinuousRepository(w.PgPool)
	scheduler := adhoc.NewContinuousScheduler(contRepo, 0, logger).
		SetWatermarkGate(adhoc.NewWatermarkGateAdapter(watermarks)).
		SetLocationFunc(tz.Current)
	go scheduler.Run(ctx)
	logger.Info("adhoc continuous scheduler started")

	logger.Info("PM adhoc pipeline ready (4 workers + continuous scheduler)")
}
