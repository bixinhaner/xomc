package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
)

// initPMModule 初始化 F03 性能管理模块。
// 设置: PMCounterRepo, PMKPIRepo
func initPMModule(c *Container) error {
	logger := c.Logger.Named("pm")

	pmCounterRepo := counter.NewPgCounterRepository(c.TsPool)
	pmKPIRepo := kpi.NewPgKPIRepository(c.TsPool)
	pmKPIEngine := kpi.NewKPIEngine(pmCounterRepo, pmKPIRepo, c.Carriers, logger)
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

	// Store deps for route registration
	c.pmHandlerDeps = &pmHandlerDeps{
		pmCounterRepo:    pmCounterRepo,
		pmKPIRepo:        pmKPIRepo,
		pmKPIEngine:      pmKPIEngine,
		pmTaskRepo:       pmTaskRepo,
		pmFileStore:      pmFileStore,
		indicatorHandler: indicatorHandler,
	}

	logger.Info("PM module initialized")
	return nil
}

type pmHandlerDeps struct {
	pmCounterRepo *counter.PgCounterRepository
	pmKPIRepo     *kpi.PgKPIRepository
	pmKPIEngine   *kpi.KPIEngine
	pmTaskRepo    *pm.PgTaskRepository
	pmFileStore   *pm.PgPMFileStore

	// Indicator management handler
	indicatorHandler *indicator.IndicatorHandler
}
