package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/alarm"
	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/device"
	"go.uber.org/zap"
)

// initAlarmModule 初始化 F04 告警管理模块。
// 设置: AlarmPgStore, AlarmEngine, AlarmFilterRuleRepository
// T-0098-P5-06：旧 alarm_libraries / alarm_library_i18n 已 DROP，改由 alarm_definitions（dictloader）提供。
func initAlarmModule(c *Container) error {
	logger := c.Logger.Named("alarm")

	alarmRedisStore := alarm.NewRedisAlarmStore(c.Redis)
	alarmPgStore := alarm.NewPgAlarmStore(c.PgPool, c.TsPool)

	alarmReconciler := alarm.NewReconciler(c.PgPool, alarmRedisStore, logger)
	alarmReconciler.Start()
	c.GS.Register("alarm-reconciler", 2, func(ctx context.Context) error {
		alarmReconciler.Stop()
		return nil
	})

	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, c.Carriers, c.EventBus, logger)
	alarmEngine.SetMetrics(alarm.NewAlarmMetrics(c.MetricsReg))

	// 告警同步服务
	alarmSyncService := alarm.NewAlarmSyncService(c.TaskSvc, c.Redis, c.EventBus, logger)
	if err := alarmSyncService.Subscribe(); err != nil {
		logger.Warn("subscribe alarm sync service", zap.NamedError("err", err))
	}

	// 告警同步处理器
	alarmDeviceRepo := device.NewPgDeviceRepository(c.PgPool)
	alarmSyncProcessor := alarm.NewAlarmSyncProcessor(alarmEngine, alarmPgStore, alarmSyncService, c.EventBus, logger).WithDeviceReader(alarmDeviceRepo)
	if c.ProductRegistry != nil {
		alarmSyncProcessor = alarmSyncProcessor.WithProductResolver(&alarmdef.ProductRegistryAdapter{Registry: c.ProductRegistry})
	}
	if c.AlarmDefRegistry != nil {
		alarmSyncProcessor = alarmSyncProcessor.WithAlarmDefRegistry(c.AlarmDefRegistry)
	}
	if err := alarmSyncProcessor.Start(context.Background()); err != nil {
		logger.Warn("start alarm sync processor", zap.NamedError("err", err))
	}

	// 过滤规则仓储
	alarmFilterRuleRepo := alarm.NewPgAlarmFilterRuleRepository(c.PgPool)
	alarmEmailRepository := alarm.NewPgAlarmEmailRepository(c.PgPool)

	// W2 T-0011: webhook 派发器（含 retry / HMAC / dead-letter）+ 死信仓储 + 过滤引擎
	webhookMetrics := alarm.NewWebhookMetrics(c.MetricsReg)
	webhookDispatcher := alarm.NewHTTPWebhookDispatcher(logger.Named("webhook"), webhookMetrics)
	deadLetterRepo := alarm.NewPgDeadLetterRepository(c.PgPool)
	filterEngine := alarm.NewFilterEngine(alarmFilterRuleRepo, alarmPgStore, webhookDispatcher, deadLetterRepo, webhookMetrics, logger.Named("filter"))
	filterEngine.SetDeviceGroupResolver(alarm.NewPgDeviceGroupResolver(c.PgPool))

	// issue #67：告警字典在 alarmdef 模块（Depends=dictload）才装配，晚于本模块。
	// 沿用 AlarmSyncProcessor 的后置注入约定——若此刻 Registry 已就绪先注入，
	// 否则交由 initAlarmDefModule 在构造 Registry 后回填（见 alarmdef.go）。
	if c.AlarmDefRegistry != nil {
		filterEngine.SetAlarmDefLookup(c.AlarmDefRegistry)
	}

	alarmEngine.SetFilterEngine(filterEngine)

	// 数据权限检查器
	dataPermissionChecker := alarm.NewDataPermissionChecker(logger)

	// Set shared services
	c.AlarmPgStore = alarmPgStore
	c.AlarmEngine = alarmEngine
	c.AlarmSyncProcessor = alarmSyncProcessor
	c.AlarmFilterEngine = filterEngine

	// Register module-level health check
	c.Health.Register("alarm", func(ctx context.Context) error {
		if err := c.Redis.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("alarm module redis ping: %w", err)
		}
		return nil
	})

	// Store deps for route registration
	c.alarmHandlerDeps = &alarmHandlerDeps{
		alarmPgStore:          alarmPgStore,
		alarmFilterRuleRepo:   alarmFilterRuleRepo,
		alarmEmailRepository:  alarmEmailRepository,
		dataPermissionChecker: dataPermissionChecker,
		alarmSyncService:      alarmSyncService,
	}

	logger.Info("alarm module initialized")
	return nil
}

type alarmHandlerDeps struct {
	alarmPgStore          *alarm.PgAlarmStore
	alarmFilterRuleRepo   *alarm.PgAlarmFilterRuleRepository
	alarmEmailRepository  *alarm.PgAlarmEmailRepository
	dataPermissionChecker *alarm.DataPermissionChecker
	alarmSyncService      *alarm.AlarmSyncService
}
