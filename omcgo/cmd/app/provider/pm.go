package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/dashboard"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/counter"
	pmexport "github.com/omcgo/omcgo/internal/pm/export"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/querytemplate"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

// initPMModule 初始化 F03 性能管理模块。
// 设置: PMCounterRepo, PMKPIRepo
func initPMModule(c *Container) error {
	logger := c.Logger.Named("pm")

	pmCounterRepo := counter.NewPgCounterRepository(c.TsPool)
	pmKPIRepo := kpi.NewPgKPIRepository(c.TsPool)
	pmTaskRepo := pm.NewPgTaskRepository(c.PgPool)
	// KPI/时序库物理分离：pm_files 已迁时序库（与 pm_metrics 同库保 copy_ingest 原子性），文件存储走 TsPool。
	pmFileStore := pm.NewPgPMFileStore(c.TsPool)

	// Set shared services
	c.PMCounterRepo = pmCounterRepo
	c.PMKPIRepo = pmKPIRepo

	// Register module-level health check (TimescaleDB is the PM module's primary dependency)
	c.Health.Register("pm", func(ctx context.Context) error {
		if err := c.TsPool.Ping(ctx); err != nil {
			return fmt.Errorf("pm module tsdb ping: %w", err)
		}
		return nil
	})

	// Initialize indicator management sub-module
	indicatorGroupRepo := indicator.NewPgGroupRepository(c.PgPool)
	indicatorRepo := indicator.NewPgIndicatorRepository(c.PgPool)
	platformFormulaRepo := indicator.NewPgPlatformFormulaRepository(c.PgPool)

	// T-0164-P1：构造 KPI Router 替代旧 carrier-based 公式路由。
	// 依赖：ProductRegistry / DeviceRepo / IndicatorRepository / PlatformFormulaRepository。
	var l2Cache router.L2Cache
	if cache := c.kpiRouteRedisCache(); cache != nil {
		l2Cache = cache
	}
	kpiRouter, err := router.New(
		c.DeviceRepo,
		c.ProductRegistry,
		indicatorRepo,
		platformFormulaRepo,
		router.Options{
			L2Cache: l2Cache,
			Metrics: c.KPIRouteMetrics,
			Logger:  logger,
		},
	)
	if err != nil {
		return fmt.Errorf("build kpi router: %w", err)
	}
	if c.KPIRouteInvalidator == nil {
		return errors.New("KPI route invalidator not initialized")
	}
	c.KPIRouteInvalidator.SetLocalTarget(kpiRouter)

	pmKPIEngine := kpi.NewKPIEngine(pmCounterRepo, pmKPIRepo, kpiRouter, logger)

	// T-0164-P5 / G5：聚合查询入口。复用同一 kpiRouter（KPI 反算所需），
	// app 端只走查询不跑 cron（cron runner 在 worker 端注册）。
	// #532 P2：双池注入——聚合源 + perf_indicators_* 在时序库（TsPool），
	// 启用集表 enabled_pm_indicators_* 只在主库（PgPool）。落库侧全存已启用需读启用集走主库。
	pmAggregator := aggregator.NewWithPools(c.TsPool, c.PgPool, kpiRouter, logger.Named("aggregator"))
	pmAggregator.SetTimezoneProvider(c.SystemTimezone)

	// T-0164 收尾 G5-Gap-2：手动重算入口需 asyncjob.Repository
	pmAsyncJobRepo := asyncjob.NewPgRepository(c.PgPool)

	// T-0164-P7 / G7：adhoc 任务 REST 入口（worker 端跑实际执行）。
	pmAdhocRepo := buildPMAdhocRepo(c.PgPool, c.TsPool, c.EventBus, logger)
	pmAdhocRepo.SetTimezoneProvider(c.SystemTimezone)
	enabledRepo := indicator.NewPgEnabledRepository(c.PgPool)
	pmAdhocHandler := adhoc.NewHandler(pmAdhocRepo, c.TsPool, nil, logger.Named("adhoc")).
		WithEnabledMetricSelectionService(adhoc.NewEnabledMetricSelectionService(enabledRepo)).
		WithTimezoneProvider(c.SystemTimezone)
	var progressService *pmstream.ProgressService
	if c.PMRedis != nil {
		streamCfg := pmstream.ConfigFromEnv()
		progressStore := pmstream.NewRedisWindowStore(c.PMRedis, streamCfg.WindowTTL)
		progressTasks := pmstream.NewPgProgressTaskLoader(c.PgPool)
		progressSnapshot := pmstream.NewSnapshotStore(progressTasks, logger.Named("pm-progress-snapshot"))
		backfillCtx, backfillCancel := context.WithTimeout(context.Background(), 5*time.Second)
		loadErr := progressSnapshot.Reload(backfillCtx)
		backfilled := false
		if loadErr == nil {
			backfilled, loadErr = pmstream.NewWindowRepository(c.TsPool).
				TryBackfillVersionMetadata(backfillCtx, progressSnapshot.Current())
		}
		backfillCancel()
		if loadErr != nil {
			logger.Warn("backfill PM progress version metadata; worker will retry",
				zap.Error(loadErr))
		} else if !backfilled {
			logger.Info("PM progress version metadata backfill already owned by worker")
		}
		progressService = pmstream.NewProgressServiceWithSnapshot(
			c.TsPool, progressStore, progressSnapshot,
		).SetTimezoneProvider(c.SystemTimezone)
		pmAdhocHandler.WithProgressService(progressService)
	}

	// T-0174 阶段 1：指标查询页"查询模板"REST 入口（5 CRUD：list/get/create/update/delete）。
	pmQueryTemplateRepo := querytemplate.NewPgRepository(c.PgPool)
	pmQueryTemplateHandler := querytemplate.NewHandler(pmQueryTemplateRepo, logger.Named("querytemplate")).
		WithEnabledMetricPayloadService(querytemplate.NewEnabledMetricPayloadService(enabledRepo)).
		WithDeviceScopeValidator(querytemplate.NewDeviceScopeValidator(
			c.PermService,
			c.DeviceService,
			device.NewPgDeviceGroupReader(c.PgPool),
		))

	// KPI-EXPORT T1：KPI 数据导出 REST 入口（建任务落表 + 入队 pm_kpi_export job）。
	// 文件管理下载默认走 app 同源流式响应；presign client 仅保留给 ?mode=url 兼容路径。
	//
	// issue #548 切片 4：同时注入 PresignBridge，sys_configs 写入
	// storage.minio_public_endpoint 后下一次 Download 立即用新 endpoint，无需重启容器。
	// 原 NewPresignClient 路径作启动期兜底。
	pmExportRepo := pmexport.NewPgRepository(c.PgPool)
	pmExportSvc := pmexport.NewService(pmExportRepo, pmAsyncJobRepo)
	var exportPresigner pmexport.Presigner
	if presignClient, presignErr := minioinfra.NewPresignClient(c.Cfg.MinIO); presignErr != nil {
		logger.Warn("create MinIO presign client for KPI export failed; download URLs disabled")
	} else {
		exportPresigner = presignClient
	}
	pmExportHandler := pmexport.NewHandler(pmExportSvc, exportPresigner, logger.Named("export"))
	pmExportHandler.SetObjectClient(c.MinIO)
	pmExportHandler.SetAdhocTaskReader(pmAdhocRepo)
	if c.PresignBridge != nil {
		pmExportHandler.SetPresignProvider(c.PresignBridge)
	}

	templateRelRepo := indicator.NewPgTemplateRelRepository(c.PgPool)
	custNameRepo := indicator.NewPgCustNameRepository(c.PgPool)
	thresholdRepo := indicator.NewPgIndicatorThresholdRepository(c.PgPool)

	indicatorSvc := indicator.NewIndicatorManagementService(
		indicatorGroupRepo,
		indicatorRepo,
		platformFormulaRepo,
		enabledRepo,
		templateRelRepo,
		custNameRepo,
		thresholdRepo,
		c.PgPool,
		c.Redis,
		logger.Named("indicator"),
	).WithDashboardLayoutReferenceChecker(
		dashboard.NewKPILayoutReferenceChecker(c.PgPool),
	).WithRouteInvalidator(func(ctx context.Context, trigger indicator.RouteInvalidationTrigger) error {
		_, err := c.KPIRouteInvalidator.Invalidate(ctx, router.InvalidationTrigger(trigger))
		return err
	})
	reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := indicatorSvc.ReconcileEnabledDependencies(reconcileCtx); err != nil {
		reconcileCancel()
		return fmt.Errorf("reconcile enabled PM indicator dependencies: %w", err)
	}
	reconcileCancel()

	indicatorHandler := indicator.NewIndicatorHandler(indicatorSvc, logger.Named("indicator"))

	// T-0098 P3-03：REST 风格 KPI 治理 API（/api/v1/indicators 系列）
	// 复用既有 IndicatorManagementService + PlatformFormulaRepository。
	// 单位定义已迁移到数据字典(seed/000007),units CRUD 下线,不再注入 unit repo。
	indicatorReloader := &indicatorReloader{reg: c.DictLoaderRegistry}
	// fileRepo 由 FileHandler 用于上传后的 destructive 重载(删孤儿)
	indicatorFileRepo := indicator.NewPgFileRepository(c.PgPool, logger)
	// 2026-06-03 用户决策:import-directory / cache/refresh 端点已下线,RESTHandler 只保留行级 CRUD
	indicatorRESTHandler := indicator.NewRESTHandler(
		indicatorSvc, platformFormulaRepo, logger.Named("indicator-rest"),
	)

	// XML 文件粒度管理(DELETE 级联清理 + 上传 + 聚合 + 列表)
	// XMLBaseDir 与 dictloader 共用,确保 loaded_from 相对路径能 join 到正确绝对路径
	// 2026-06-03 用户决策:上传 = 写 builtin 目录 → destructive 重载(删孤儿)→ 刷新缓存
	// reloader 触发 Loader.Reload;cache=indicatorSvc 重载成功后 BumpCacheVersion
	indicatorFileHandler := indicator.NewFileHandler(
		indicatorFileRepo, indicatorReloader, indicatorSvc, c.DictService, // #241: 导入后刷 kpi_platform_enb 绑定字典(pm 模块 Depends "admin")
		c.Cfg.DictLoader.XMLBaseDir, logger.Named("indicator-file"),
	)
	// 启动期幂等 mkdir host bind mount 三制式子目录,首次部署不报错
	if err := indicator.EnsureBaseDir(context.Background(), c.Cfg.DictLoader.XMLBaseDir); err != nil {
		// 不阻塞启动:host bind mount 未挂载是部署问题,handler 后续 Upload/Delete 仍会显式报错
		logger.Warn("ensure indicator custom dir failed; uploads/deletes may fail until host bind mount is ready")
	}

	// Store deps for route registration
	c.pmHandlerDeps = &pmHandlerDeps{
		pmCounterRepo:          pmCounterRepo,
		pmKPIRepo:              pmKPIRepo,
		pmKPIEngine:            pmKPIEngine,
		pmTaskRepo:             pmTaskRepo,
		pmFileStore:            pmFileStore,
		pmIndicatorRepo:        indicatorRepo,
		pmAggregator:           pmAggregator,
		pmAsyncJobRepo:         pmAsyncJobRepo,
		pmProgressService:      progressService,
		pmAdhocHandler:         pmAdhocHandler,
		pmQueryTemplateHandler: pmQueryTemplateHandler,
		pmExportHandler:        pmExportHandler,
		indicatorHandler:       indicatorHandler,
		indicatorRESTHandler:   indicatorRESTHandler,
		indicatorFileHandler:   indicatorFileHandler,
	}

	logger.Info("PM module initialized")
	return nil
}

// buildPMAdhocRepo 将现有任务管理 API 接到新的逻辑任务 + 不可变版本控制面。
// pm_tasks 只承载用户可编辑的展示定义；worker 不再扫描它执行历史窗口。
func buildPMAdhocRepo(
	pgPool, tsPool *pgxpool.Pool,
	bus event.EventBus,
	_ *zap.Logger,
) *adhoc.PgRepository {
	return adhoc.NewPgRepository(pgPool, tsPool).
		SetStreamingRepository(pmstream.NewPgTaskRepository(pgPool, bus))
}

// indicatorReloader 把 dictloader.Registry.ReloadOne(...)(Report, error)
// 适配为 indicator.Reloader 期望的 ReloadOne(ctx, name) error。
type indicatorReloader struct {
	reg *dictloader.Registry
}

func (r *indicatorReloader) ReloadOne(ctx context.Context, name string) error {
	if r.reg == nil {
		return errors.New("dictloader registry not wired")
	}
	_, err := r.reg.ReloadOne(ctx, name)
	return err
}

type pmHandlerDeps struct {
	pmCounterRepo          *counter.PgCounterRepository
	pmKPIRepo              *kpi.PgKPIRepository
	pmKPIEngine            *kpi.KPIEngine
	pmTaskRepo             *pm.PgTaskRepository
	pmFileStore            *pm.PgPMFileStore
	pmIndicatorRepo        indicator.IndicatorRepository // T-0164-P1 ListKPIDefinitions 数据源
	pmAggregator           *aggregator.Aggregator        // T-0164-P5 ListAggregatedMetrics 数据源
	pmAsyncJobRepo         asyncjob.Repository           // T-0164 收尾 G5-Gap-2 手动重算端点
	pmProgressService      *pmstream.ProgressService     // 当前日/周只读预览，adhoc 与首页共用
	pmAdhocHandler         *adhoc.Handler                // T-0164-P7 自定义聚合任务 REST 入口
	pmQueryTemplateHandler *querytemplate.Handler        // T-0174 指标查询模板 REST 入口
	pmExportHandler        *pmexport.Handler             // KPI-EXPORT T1 KPI 导出 REST 入口

	// Indicator management handler
	indicatorHandler     *indicator.IndicatorHandler
	indicatorRESTHandler *indicator.RESTHandler
	indicatorFileHandler *indicator.FileHandler // T-0180 P1.3 XML 文件粒度管理
}
