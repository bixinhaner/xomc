package provider

import (
	"github.com/omcgo/omcgo/internal/config/template"
)

// initConfigModule 初始化 F02 配置模板模块。
// T-0098 P5-01：旧 datamodel 注册表 / 缓存 / cleaner / OUI / importer 全部下线，
// 数据字典平台化（参数 / KPI / 告警）改由 dictloader + 4 个 Registry 提供。
func initConfigModule(c *Container) error {
	logger := c.Logger.Named("config")

	templateRepo := template.NewPgConfigTemplateRepository(c.PgPool)
	templateService := template.NewConfigTemplateService(templateRepo, logger)

	c.TemplateService = templateService
	c.configHandlerDeps = &configHandlerDeps{
		templateRepo: templateRepo,
	}

	logger.Info("config module initialized")
	return nil
}

// configHandlerDeps holds repos needed for handler creation.
type configHandlerDeps struct {
	templateRepo *template.PgConfigTemplateRepository
}
