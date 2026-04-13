package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/alarm"
)

// initAlarmModule 初始化 F04 告警管理模块。
// 设置: AlarmPgStore, AlarmEngine
func initAlarmModule(c *Container) error {
	logger := c.Logger.Named("alarm")

	alarmRedisStore := alarm.NewRedisAlarmStore(c.Redis)
	alarmPgStore := alarm.NewPgAlarmStore(c.PgPool, c.TsPool)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, c.Carriers, c.EventBus, logger)
	alarmEngine.SetMetrics(alarm.NewAlarmMetrics(c.MetricsReg))

	// Set shared services
	c.AlarmPgStore = alarmPgStore
	c.AlarmEngine = alarmEngine

	// Register module-level health check
	c.Health.Register("alarm", func(ctx context.Context) error {
		if err := c.Redis.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("alarm module redis ping: %w", err)
		}
		return nil
	})

	// Store deps for route registration
	c.alarmHandlerDeps = &alarmHandlerDeps{
		alarmPgStore: alarmPgStore,
	}

	logger.Info("alarm module initialized")
	return nil
}

type alarmHandlerDeps struct {
	alarmPgStore *alarm.PgAlarmStore
}
