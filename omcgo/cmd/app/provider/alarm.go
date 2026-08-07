package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/notification"
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
	alarmMetrics := alarm.NewAlarmMetrics(c.MetricsReg)
	alarmEngine.SetMetrics(alarmMetrics)
	if c.Cfg.Alarm.LifecycleRelayEnabled() {
		outboxRelay := alarm.NewAlarmOutboxRelay(
			alarm.NewAlarmOutboxRepository(c.PgPool), c.EventBus, logger,
		).SetMetrics(alarmMetrics)
		outboxRelay.Start(context.Background())
		c.GS.Register("alarm-outbox-relay", 1, func(context.Context) error {
			outboxRelay.Stop()
			return nil
		})

		historyProjector := alarm.NewHistoryProjector(
			alarmPgStore, c.EventBus, c.Cfg.Alarm.LifecycleStartSequence,
		)
		historyProjector.SetShadow(c.Cfg.Alarm.LifecycleMode == string(alarm.LifecycleModeShadow))
		if err := historyProjector.Subscribe(); err != nil {
			outboxRelay.Stop()
			return fmt.Errorf("subscribe alarm history projector: %w", err)
		}
		c.GS.Register("alarm-history-projector", 1, func(context.Context) error {
			return historyProjector.Close()
		})
		if c.Cfg.Alarm.LifecycleMode == string(alarm.LifecycleModeCanonical) {
			readyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			readyErr := historyProjector.Ready(readyCtx)
			cancel()
			if readyErr != nil {
				_ = historyProjector.Close()
				outboxRelay.Stop()
				return fmt.Errorf("enable canonical alarm lifecycle: %w", readyErr)
			}
		}
		logger.Info("alarm lifecycle Relay and history projector started",
			zap.String("lifecycle_mode", c.Cfg.Alarm.LifecycleMode),
			zap.Uint64("lifecycle_start_sequence", c.Cfg.Alarm.LifecycleStartSequence))
	}
	initialLifecycleMode := alarm.LifecycleMode(c.Cfg.Alarm.LifecycleMode)
	if initialLifecycleMode == alarm.LifecycleModeCanonical {
		// Canonical is activated by initNorthboundModule only after its fixed
		// durable is subscribed and caught up. Remaining legacy during module
		// bootstrap avoids a history/northbound partial cutover.
		initialLifecycleMode = alarm.LifecycleModeLegacy
	}
	if err := alarmEngine.SetLifecycleMode(initialLifecycleMode); err != nil {
		return fmt.Errorf("configure alarm lifecycle mode: %w", err)
	}

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

	// W2 T-0011: webhook 派发器（含 retry / HMAC / dead-letter）+ 死信仓储 + 过滤引擎
	webhookMetrics := alarm.NewWebhookMetrics(c.MetricsReg)
	webhookDispatcher := alarm.NewHTTPWebhookDispatcher(logger.Named("webhook"), webhookMetrics)
	deadLetterRepo := alarm.NewPgDeadLetterRepository(c.PgPool)
	filterEngine := alarm.NewFilterEngine(alarmFilterRuleRepo, alarmPgStore, webhookDispatcher, deadLetterRepo, webhookMetrics, logger.Named("filter"))
	filterEngine.SetDeviceGroupResolver(alarm.NewPgDeviceGroupResolver(c.PgPool))

	// 迁移期 legacy notify_email 复用通知中心的逐收件人 SMTP transport。
	// Task 12 完成规则 barrier 切换后再删除这条兼容入口。
	emailMetrics := alarm.NewEmailMetrics(c.MetricsReg)
	emailDispatcher := alarm.NewSharedEmailDispatcher(sharedNotificationEmailSender(c), logger.Named("email"), emailMetrics)
	filterEngine.SetEmailDispatcher(emailDispatcher)

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
		dataPermissionChecker: dataPermissionChecker,
		alarmSyncService:      alarmSyncService,
	}

	logger.Info("alarm module initialized")
	return nil
}

type alarmHandlerDeps struct {
	alarmPgStore          *alarm.PgAlarmStore
	alarmFilterRuleRepo   *alarm.PgAlarmFilterRuleRepository
	dataPermissionChecker *alarm.DataPermissionChecker
	alarmSyncService      *alarm.AlarmSyncService
}

func sharedNotificationEmailSender(c *Container) *notification.EmailSender {
	if c.NotificationEmailSender == nil {
		c.NotificationEmailSender = notification.NewEmailSender(notification.SMTPOptions{
			Enabled: c.Cfg.Notification.SMTP.Enabled, Host: c.Cfg.Notification.SMTP.Host,
			Port: c.Cfg.Notification.SMTP.Port, Username: c.Cfg.Notification.SMTP.Username,
			Password: c.Cfg.Notification.SMTP.Password, From: c.Cfg.Notification.SMTP.From,
			TLSMode: c.Cfg.Notification.SMTP.TLSMode, StartTLS: c.Cfg.Notification.SMTP.StartTLS,
			Timeout: c.Cfg.Notification.SMTP.Timeout,
		}, c.Logger)
	}
	return c.NotificationEmailSender
}
