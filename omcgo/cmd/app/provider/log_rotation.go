package provider

import (
	"context"

	"github.com/omcgo/omcgo/internal/admin"
	corelogger "github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/logretention"
)

// 日志轮转与 OMC 服务日志有效期 wiring（category=log.rotation.service_days）。
//
// 让 app 进程自身日志文件（app.log）的「大小/间隔/个数/过期」可在系统配置页里调：启动一个轮询
// watcher 从 sys_configs 读取轮转配置和统一服务日志有效期，热更新进程级 logger override（后台
// 轮转/归档清理 ≤1 分钟生效）。acs/worker 各自在自己的 main 里起同样的 watcher（管各自日志文件）。
func initLogRotationModule(c *Container) error {
	repo := admin.NewPgSysConfigRepository(c.PgPool)
	lookup := func(ctx context.Context, category, key string) (string, bool) {
		row, err := repo.GetByKey(ctx, category, key)
		if err != nil || row == nil {
			return "", false
		}
		return row.Value, true
	}
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterValidator(corelogger.RotationCategory, corelogger.KeyServiceDays,
			corelogger.ValidateServiceLogDays)
		// 兼容已经存在的旧配置行；新页面不再写入该分类。
		c.SysConfigSvc.RegisterValidator(corelogger.RetentionCategory, corelogger.KeyServiceDays,
			corelogger.ValidateServiceLogDays)
		c.SysConfigSvc.RegisterValidator(logretention.Category, logretention.KeyDatabaseDays,
			corelogger.ValidateDatabaseLogDays)
	}

	ctx, cancel := context.WithCancel(context.Background())
	corelogger.StartRotationConfigWatcher(ctx, lookup, c.Logger.Named("log-rotation"))
	// GS 在 app 容器装配时必非 nil（main.go Container{GS: app.GS}）；注册 cancel 优雅关停 watcher。
	c.GS.Register("log-rotation-watcher", 1, func(context.Context) error {
		cancel()
		return nil
	})
	return nil
}
