package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/alarm"
	"go.uber.org/zap"
)

// initAlarmModule 初始化 F04 告警管理模块。
// 设置: AlarmPgStore, AlarmEngine, AlarmLibraryRepository, AlarmFilterRuleRepository
func initAlarmModule(c *Container) error {
	logger := c.Logger.Named("alarm")

	alarmRedisStore := alarm.NewRedisAlarmStore(c.Redis)
	alarmPgStore := alarm.NewPgAlarmStore(c.PgPool, c.TsPool)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, c.Carriers, c.EventBus, logger)
	alarmEngine.SetMetrics(alarm.NewAlarmMetrics(c.MetricsReg))

	// 告警同步服务
	alarmSyncService := alarm.NewAlarmSyncService(c.TaskSvc, c.Redis, c.EventBus, logger)
	if err := alarmSyncService.Subscribe(); err != nil {
		logger.Warn("subscribe alarm sync service", zap.NamedError("err", err))
	}

	// 告警同步处理器
	alarmSyncProcessor := alarm.NewAlarmSyncProcessor(alarmEngine, alarmPgStore, alarmSyncService, c.EventBus, logger)
	if err := alarmSyncProcessor.Start(context.Background()); err != nil {
		logger.Warn("start alarm sync processor", zap.NamedError("err", err))
	}

	// 告警库仓储
	alarmLibraryRepo := alarm.NewPgAlarmLibraryRepository(c.PgPool)
	alarmLibraryService := alarm.NewLibraryService(alarmLibraryRepo, logger)

	// 过滤规则仓储
	alarmFilterRuleRepo := alarm.NewPgAlarmFilterRuleRepository(c.PgPool)

	// W2 T-0011: webhook 派发器（含 retry / HMAC / dead-letter）+ 死信仓储 + 过滤引擎
	webhookMetrics := alarm.NewWebhookMetrics(c.MetricsReg)
	webhookDispatcher := alarm.NewHTTPWebhookDispatcher(logger.Named("webhook"), webhookMetrics)
	deadLetterRepo := alarm.NewPgDeadLetterRepository(c.PgPool)
	filterEngine := alarm.NewFilterEngine(alarmFilterRuleRepo, alarmPgStore, webhookDispatcher, deadLetterRepo, webhookMetrics, logger.Named("filter"))
	alarmEngine.SetFilterEngine(filterEngine)

	// 数据权限检查器
	dataPermissionChecker := alarm.NewDataPermissionChecker(logger)

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
		alarmPgStore:           alarmPgStore,
		alarmLibraryService:    alarmLibraryService,
		alarmFilterRuleRepo:    alarmFilterRuleRepo,
		dataPermissionChecker:  dataPermissionChecker,
		alarmSyncService:       alarmSyncService,
	}

	logger.Info("alarm module initialized")
	return nil
}

type alarmHandlerDeps struct {
	alarmPgStore          *alarm.PgAlarmStore
	alarmLibraryService   *alarm.LibraryService
	alarmFilterRuleRepo   *alarm.PgAlarmFilterRuleRepository
	dataPermissionChecker *alarm.DataPermissionChecker
	alarmSyncService      *alarm.AlarmSyncService
}
