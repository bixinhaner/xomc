package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
)

// initPMModule 初始化 F03 性能管理模块。
// 设置: PMCounterRepo, PMKPIRepo
func initPMModule(c *Container) error {
	logger := c.Logger.Named("pm")

	pmCounterRepo := counter.NewPgCounterRepository(c.TsPool)
	pmKPIRepo := kpi.NewPgKPIRepository(c.TsPool)
	pmTaskRepo := pm.NewPgTaskRepository(c.PgPool)
	pmFileStore := pm.NewPgPMFileStore(c.PgPool)

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
	if c.Redis != nil {
		l2Cache = router.NewRedisCache(c.Redis)
	}
	kpiRouter, err := router.New(
		c.DeviceRepo,
		c.ProductRegistry,
		indicatorRepo,
		platformFormulaRepo,
		router.Options{
			L2Cache: l2Cache,
			Metrics: router.NewMetrics(c.MetricsReg),
			Logger:  logger,
		},
	)
	if err != nil {
		return fmt.Errorf("build kpi router: %w", err)
	}

	pmKPIEngine := kpi.NewKPIEngine(pmCounterRepo, pmKPIRepo, kpiRouter, logger)

	// T-0164-P5 / G5：聚合查询入口。复用同一 kpiRouter（KPI 反算所需），
	// app 端只走查询不跑 cron（cron runner 在 worker 端注册）。
	pmAggregator := aggregator.NewWithPool(c.TsPool, kpiRouter, logger.Named("aggregator"))

	// T-0164-P7 / G7：adhoc 任务 REST 入口（worker 端跑实际执行）。
	pmAdhocRepo := adhoc.NewPgRepository(c.PgPool)
	pmAdhocHandler := adhoc.NewHandler(pmAdhocRepo, c.TsPool, c.EventBus, logger.Named("adhoc"))

	enabledRepo := indicator.NewPgEnabledRepository(c.PgPool)
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
	)

	indicatorHandler := indicator.NewIndicatorHandler(indicatorSvc, logger.Named("indicator"))

	// T-0098 P3-03：REST 风格 KPI 治理 API（/api/v1/indicators 系列）
	// 复用既有 IndicatorManagementService + PlatformFormulaRepository；
	// 新增 PgUnitRepository 承担 indicator_unit CRUD。
	indicatorUnitRepo := indicator.NewPgUnitRepository(c.PgPool)
	indicatorReloader := &indicatorReloader{reg: c.DictLoaderRegistry}
	indicatorRESTHandler := indicator.NewRESTHandler(
		indicatorSvc, platformFormulaRepo, indicatorUnitRepo, indicatorReloader, logger.Named("indicator-rest"),
	)

	// Store deps for route registration
	c.pmHandlerDeps = &pmHandlerDeps{
		pmCounterRepo:        pmCounterRepo,
		pmKPIRepo:            pmKPIRepo,
		pmKPIEngine:          pmKPIEngine,
		pmTaskRepo:           pmTaskRepo,
		pmFileStore:          pmFileStore,
		pmIndicatorRepo:      indicatorRepo,
		pmAggregator:         pmAggregator,
		pmAdhocHandler:       pmAdhocHandler,
		indicatorHandler:     indicatorHandler,
		indicatorRESTHandler: indicatorRESTHandler,
	}

	logger.Info("PM module initialized")
	return nil
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
	pmCounterRepo   *counter.PgCounterRepository
	pmKPIRepo       *kpi.PgKPIRepository
	pmKPIEngine     *kpi.KPIEngine
	pmTaskRepo      *pm.PgTaskRepository
	pmFileStore     *pm.PgPMFileStore
	pmIndicatorRepo indicator.IndicatorRepository // T-0164-P1 ListKPIDefinitions 数据源
	pmAggregator    *aggregator.Aggregator        // T-0164-P5 ListAggregatedMetrics 数据源
	pmAdhocHandler  *adhoc.Handler                // T-0164-P7 自定义聚合任务 REST 入口

	// Indicator management handler
	indicatorHandler     *indicator.IndicatorHandler
	indicatorRESTHandler *indicator.RESTHandler
}
