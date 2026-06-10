package main

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

// workerInfra extends components.Infra with worker-specific dependencies.
type workerInfra struct {
	*components.Infra
	Carriers    *carrier.CarrierRegistry
	TaskService *task.TaskService
	TaskRepo    *task.PgTaskRepository
	TaskQueue   *task.RedisTaskQueue
	// TaskMetrics 在 initWorker 内构造并挂到 TaskService，registerSubscribers / reconciler
	// 复用同一实例（避免重复 MustRegister 触发 Prometheus duplicate collector panic）。
	// 在 RestorePendingQueues 之前就绪，确保启动期 recovery 动作能被记到指标。
	TaskMetrics *task.TaskMetrics
}

// initWorker initializes all infrastructure for the background worker.
func initWorker(ctx context.Context, cfg *appconfig.WorkerConfig) (*workerInfra, error) {
	inf, err := components.NewInfra(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := inf.InitTracer(ctx, cfg.Tracer, "omcgo-worker"); err != nil {
		return nil, err
	}

	if err := inf.ConnectPostgres(ctx, cfg.DB); err != nil {
		return nil, err
	}
	if err := inf.ConnectTimescale(ctx, cfg.TSDB); err != nil {
		return nil, err
	}
	if err := inf.ConnectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := inf.ConnectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	if err := inf.ConnectMinIO(ctx, cfg.MinIO); err != nil {
		return nil, err
	}
	inf.CreateEventBus()

	w := &workerInfra{Infra: inf}
	w.registerCarriers()

	// 创建统一任务队列：TaskService（Redis + PG）
	taskQueue := task.NewRedisTaskQueue(inf.Redis)
	taskRepo := task.NewPgTaskRepository(inf.PgPool)
	taskSvc := task.NewTaskService(taskQueue, taskRepo, inf.Logger)
	// 关键：worker 的 ExpiredSweeper 调 TaskService.ExpireTask → notifyCompletion。
	// 没注入 EventBus 时 notifyCompletion 走 fallback 同进程 callbacks（worker 没注册
	// 任何 callback）→ 事件被静默丢弃 → app 端 CompletionEventBridge 永远收不到
	// task.failed → MR / provision 等跨进程订阅者拿不到 expired 通知 → 子任务
	// 进度卡 pending 直到下游 reaper 兜底（甚至永久卡住）。app 那边
	// cmd/app/bootstrap.go:70 已设；这里补上同款注入。
	taskSvc.SetEventBus(inf.EventBus)
	// issue #20：任务指标在这里就构造并注入 —— 必须早于 runWorker 里的
	// RestorePendingQueues，否则启动期 recovery 动作（restore_pending）打点时
	// metrics 还是 nil。reconciler / 双写中断指标复用同一实例。
	taskMetrics := task.NewTaskMetrics(inf.MetricsReg)
	taskSvc.SetMetrics(taskMetrics)
	w.TaskService = taskSvc
	w.TaskRepo = taskRepo
	w.TaskQueue = taskQueue
	w.TaskMetrics = taskMetrics

	return w, nil
}

func (w *workerInfra) registerCarriers() {
	reg := carrier.NewRegistry()
	reg.Register(cmcc.New())
	reg.Register(ctcc.New())
	reg.Register(cucc.New())
	w.Carriers = reg
	w.Logger.Info("carrier registry initialized", zap.Int("carriers", len(reg.All())))
}
