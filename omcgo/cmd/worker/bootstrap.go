package main

import (
	"context"
	"os"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/deviceaccess"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/prometheus/client_golang/prometheus"
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
	TaskMetrics       *task.TaskMetrics
	StorageProtection *storageprotection.Service
	ProductRegistry   *product.Registry
}

// initWorker initializes all infrastructure for the background worker.
func initWorker(ctx context.Context, cfg *appconfig.WorkerConfig) (*workerInfra, error) {
	inf, err := components.NewInfra(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}
	// 配置文件驱动的 pprof 开关（metrics 端口挂 /debug/pprof/*，默认关）。issue #218：
	// 便于对 worker 做 30s CPU profile 实测确认 PM 入库热点（次因）。
	inf.SetPprof(cfg.Metrics.Pprof.Enabled, cfg.Metrics.Pprof.Contention)

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
	if len(cfg.PMRedis.Addrs) == 0 {
		inf.PMRedis = inf.Redis
		inf.Logger.Warn("pm_redis not configured; using compatibility fallback to redis-core")
	} else if err := inf.ConnectPMRedis(cfg.EffectivePMRedis()); err != nil {
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
	taskQueue := task.NewRedisTaskQueueWithTerminalTTL(inf.Redis, cfg.Task.EffectiveTerminalRedisTTL())
	taskRepo := task.NewPgTaskRepository(inf.PgPool)
	taskSvc := task.NewTaskService(taskQueue, taskRepo, inf.Logger)
	// 关键：worker 的 ExpiredSweeper 调 TaskService.ExpireTask → notifyCompletion。
	// 没注入 EventBus 时 notifyCompletion 走 fallback 同进程 callbacks（worker 没注册
	// 任何 callback）→ 事件被静默丢弃 → app 端 CompletionEventBridge 永远收不到
	// task.failed → MR / provision 等跨进程订阅者拿不到 expired 通知 → 子任务
	// 进度卡 pending 直到下游 reaper 兜底（甚至永久卡住）。app 那边
	// cmd/app/bootstrap.go:70 已设；这里补上同款注入。
	taskSvc.SetEventBus(inf.EventBus)
	accessRepository := deviceaccess.NewPgRepository(inf.PgPool)
	guard := deviceaccess.NewAccessTaskGuard(accessRepository, accessRepository)
	guard.SetRuntimeSettingsReader(deviceaccess.NewPgRuntimeSettingsStore(inf.PgPool))
	taskSvc.SetAdmissionGuard(guard)
	inf.Logger.Info("device access task admission guard initialized")
	// issue #20：任务指标在这里就构造并注入 —— 必须早于 runWorker 里的
	// RestorePendingQueues，否则启动期 recovery 动作（restore_pending）打点时
	// metrics 还是 nil。reconciler / 双写中断指标复用同一实例。
	taskMetrics := newWorkerTaskMetrics(inf.MetricsReg)
	taskSvc.SetMetrics(taskMetrics)
	persistentQueueMetrics := components.NewPersistentQueueMetrics(inf.MetricsReg)
	persistentQueueObserver := components.NewPersistentQueueObserver(inf.PgPool, persistentQueueMetrics, 30*time.Second, inf.Logger)
	persistentQueueObserver.Start(context.Background())
	inf.GS.Register("persistent-queue-observer", 1, func(context.Context) error {
		persistentQueueObserver.Stop()
		return nil
	})
	prometheusURL := os.Getenv("OMCGO_SYSTEM_INFO_PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://prometheus:9090"
	}
	storageCollector := components.NewPrometheusStorageCollector(prometheusURL, 2*time.Second, time.Minute, nil)
	storageProtection := storageprotection.NewService(
		storageprotection.NewPgRepository(inf.PgPool),
		storageprotection.NewCollectorUsageProvider(storageCollector),
		storageprotection.NewMetrics(inf.MetricsReg),
		inf.Logger,
	)
	storageProtection.SetLogAdmissionController(inf.LogGate)
	storageProtection.Start(context.Background(), 30*time.Second)
	inf.GS.Register("storage-protection", 1, func(context.Context) error {
		storageProtection.Stop()
		return nil
	})
	w.TaskService = taskSvc
	w.TaskRepo = taskRepo
	w.TaskQueue = taskQueue
	w.TaskMetrics = taskMetrics
	w.StorageProtection = storageProtection

	return w, nil
}

func newWorkerTaskMetrics(reg prometheus.Registerer) *task.TaskMetrics {
	metrics := task.NewTaskMetrics(reg)
	// The app process owns the Redis keyspace observation. Running the same
	// multi-million-key scan in worker doubles Redis CPU/IO and creates false
	// up=0 series while a long initial scan is still in progress.
	metrics.DisableRedisQueueObservation()
	return metrics
}

func (w *workerInfra) registerCarriers() {
	reg := carrier.NewRegistry()
	reg.Register(cmcc.New())
	reg.Register(ctcc.New())
	reg.Register(cucc.New())
	w.Carriers = reg
	w.Logger.Info("carrier registry initialized", zap.Int("carriers", len(reg.All())))
}
