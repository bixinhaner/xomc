package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
)

// initConfigModule 初始化 F02 数据模型与配置模板模块。
// 设置: DMRegistry, DMImporter, TemplateService
func initConfigModule(c *Container) error {
	logger := c.Logger.Named("config")

	// DataModel module
	dmRepo := datamodel.NewPgDataModelRepository(c.PgPool)
	importLogRepo := datamodel.NewPgImportLogRepository(c.PgPool)
	ouiRepo := datamodel.NewPgOUIRepository(c.PgPool)
	dmCache := datamodel.NewDataModelCache(c.Redis)

	dmRegistry := datamodel.NewDataModelRegistry(dmRepo, dmCache, logger)
	dmRegistry.Start()
	c.GS.Register("datamodel-registry", 1, func(ctx context.Context) error { dmRegistry.Stop(); return nil })

	dmImporter := datamodel.NewDataModelImporter(dmRepo, importLogRepo)

	// DataModel expiry cleaner
	expiryConf := c.Cfg.DataModelExpiry
	if expiryConf.AutoMaxIdleDays <= 0 {
		expiryConf.AutoMaxIdleDays = 15
	}
	if expiryConf.ManualMaxIdleDays <= 0 {
		expiryConf.ManualMaxIdleDays = 60
	}
	if expiryConf.CleanupCron == "" {
		expiryConf.CleanupCron = "0 3 * * *"
	}
	dmCleaner := datamodel.NewDataModelCleaner(dmRepo, dmCache, expiryConf.AutoMaxIdleDays, expiryConf.ManualMaxIdleDays, logger)
	if err := dmCleaner.Start(expiryConf.CleanupCron); err != nil {
		return fmt.Errorf("start datamodel cleaner: %w", err)
	}
	c.GS.Register("datamodel-cleaner", 1, func(ctx context.Context) error { dmCleaner.Stop(); return nil })

	// ConfigTemplate module
	templateRepo := template.NewPgConfigTemplateRepository(c.PgPool)
	templateService := template.NewConfigTemplateService(templateRepo, logger)

	// Set shared services
	c.DMRegistry = dmRegistry
	c.DMImporter = dmImporter
	c.TemplateService = templateService

	// Store repos for handler creation
	c.configHandlerDeps = &configHandlerDeps{
		dmRepo:     dmRepo,
		ouiRepo:    ouiRepo,
		dmRegistry: dmRegistry,
		dmImporter: dmImporter,
		templateRepo: templateRepo,
	}

	logger.Info("config module initialized")
	return nil
}

// configHandlerDeps holds repos needed for handler creation.
type configHandlerDeps struct {
	dmRepo       *datamodel.PgDataModelRepository
	ouiRepo      *datamodel.PgOUIRepository
	dmRegistry   *datamodel.DataModelRegistry
	dmImporter   *datamodel.DataModelImporter
	templateRepo *template.PgConfigTemplateRepository
}
