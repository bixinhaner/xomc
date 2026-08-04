package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

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
			alarmEngine.SetCanonicalLifecycleReady(true)
		}
		logger.Info("alarm lifecycle Relay and history projector started",
			zap.String("lifecycle_mode", c.Cfg.Alarm.LifecycleMode),
			zap.Uint64("lifecycle_start_sequence", c.Cfg.Alarm.LifecycleStartSequence))
	}
	if err := alarmEngine.SetLifecycleMode(alarm.LifecycleMode(c.Cfg.Alarm.LifecycleMode)); err != nil {
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

	// W2.A.1 / T-0007 整合: SMTP 邮件派发器（实现 EmailDispatcher 接口）。
	// 配置从环境变量读取（OMC_SMTP_HOST/PORT/USERNAME/PASSWORD/FROM/USE_TLS/USE_STARTTLS）。
	// 配置缺失时仍创建 dispatcher（dispatch 时会因空 host 拨号失败，进 metric=failure），
	// 这样 filter_engine 永远走 SMTPEmailDispatcher 而非 noop，保证生产可观测性。
	// 后续 task 把 SMTP 配置接入 appconfig YAML（替换本处 env 读取）。
	emailMetrics := alarm.NewEmailMetrics(c.MetricsReg)
	emailCfg := loadEmailConfigFromEnv()
	emailDispatcher := alarm.NewSMTPEmailDispatcher(emailCfg, logger.Named("email"), emailMetrics)
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

// loadEmailConfigFromEnv 从 OMC_SMTP_* 环境变量读 SMTP 配置（W2.A.1/T-0007）。
// 未设置 → 返回零值 EmailConfig（拨号会失败但不 panic，便于 dev / test 环境）。
// 后续把整段读取迁移到 appconfig.yaml 时替换本函数为 cfg.AppConfig.Email 即可。
func loadEmailConfigFromEnv() alarm.EmailConfig {
	port, _ := strconv.Atoi(os.Getenv("OMC_SMTP_PORT")) // 解析失败 → 0，dispatch 时返错
	return alarm.EmailConfig{
		Host:        os.Getenv("OMC_SMTP_HOST"),
		Port:        port,
		Username:    os.Getenv("OMC_SMTP_USERNAME"),
		Password:    os.Getenv("OMC_SMTP_PASSWORD"),
		From:        os.Getenv("OMC_SMTP_FROM"),
		UseTLS:      os.Getenv("OMC_SMTP_USE_TLS") == "true",
		UseSTARTTLS: os.Getenv("OMC_SMTP_USE_STARTTLS") == "true",
	}
}
