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
	Carriers *carrier.CarrierRegistry
	TaskRepo *task.PgTaskRepository
	TaskSvc  *task.TaskService
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

	// 创建统一任务队列：TaskService（Redis + PG 双写）。
	// 旧的 cmdqueue 兼容适配层已下线，业务模块直连 TaskService.CreateTask。
	taskQueue := task.NewRedisTaskQueue(inf.Redis)
	taskRepo := task.NewPgTaskRepository(inf.PgPool)
	taskSvc := task.NewTaskService(taskQueue, taskRepo, inf.Logger)
	// Worker 可能通过 RebootCloser 等路径写终态，与 ACS 保持一致，终态通过 NATS
	// 广播由订阅者聚合，避免和回调重复计数。
	if inf.EventBus != nil {
		taskSvc.SetEventBus(inf.EventBus)
	}
	w.TaskRepo = taskRepo
	w.TaskSvc = taskSvc

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
