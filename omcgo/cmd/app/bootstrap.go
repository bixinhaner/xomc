package main

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/task"
	"go.uber.org/zap"
)

// appInfra extends components.Infra with app-specific dependencies.
type appInfra struct {
	*components.Infra
	Carriers *carrier.CarrierRegistry
	TaskSvc  *task.TaskService
	NotifSvc *notification.Service // T-0157 stale sync: handler 需复用同一实例（已注入 task lookup）
}

// initApp initializes all infrastructure for the main application.
func initApp(ctx context.Context, cfg *appconfig.AppConfig) (*appInfra, error) {
	inf, err := components.NewInfra(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := inf.InitTracer(ctx, cfg.Tracer, "omcgo-app"); err != nil {
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

	app := &appInfra{Infra: inf}
	app.registerCarriers()

	// 创建统一任务队列：TaskService（Redis + PG 双写）。
	// 业务模块直连 TaskService.CreateTask；旧的 cmdqueue 兼容适配层已下线。
	taskQueue := task.NewRedisTaskQueue(inf.Redis)
	taskRepo := task.NewPgTaskRepository(inf.PgPool)
	app.TaskSvc = task.NewTaskService(taskQueue, taskRepo, inf.Logger)
	// T-0157 C1: 注入默认超时兜底（详见 appconfig.TaskConfig 注释）
	app.TaskSvc.SetDefaultExpiresIn(cfg.Task.DefaultExpiresInSeconds)
	// T-0157 C5: 注入 EventBus —— 没有它 CreateTask 末尾 publish task.created
	// 会因 eventBus==nil 静默跳过，notification subscriber 永远收不到事件。
	app.TaskSvc.SetEventBus(inf.EventBus)
	// T-0157 C6: 注入入队失败兜底通知（avoid"点了没反应"，覆盖所有 CreateTask 调用点）
	notifRepo := notification.NewPgRepository(inf.PgPool)
	notifSvc := notification.NewService(notifRepo, nil, inf.Logger)
	app.TaskSvc.SetCreateFailureNotifier(notification.NewCreateFailureNotifier(notifSvc, inf.Logger))
	// T-0157 stale sync: 注入 task lookup 让 SyncStaleByUser 能反查 task 状态修正卡死消息。
	notifSvc.SetStaleTaskLookup(task.NewStaleNotificationLookup(taskRepo))
	app.NotifSvc = notifSvc

	return app, nil
}

func (a *appInfra) registerCarriers() {
	reg := carrier.NewRegistry()
	reg.Register(cmcc.New())
	reg.Register(ctcc.New())
	reg.Register(cucc.New())
	a.Carriers = reg
	a.Logger.Info("carrier registry initialized", zap.Int("carriers", len(reg.All())))
}
