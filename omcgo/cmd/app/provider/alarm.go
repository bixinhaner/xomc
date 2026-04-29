package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

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

	// W2.A.1 / T-0007 整合: SMTP 邮件派发器（实现 EmailDispatcher 接口）。
	// 配置从环境变量读取（OMC_SMTP_HOST/PORT/USERNAME/PASSWORD/FROM/USE_TLS/USE_STARTTLS）。
	// 配置缺失时仍创建 dispatcher（dispatch 时会因空 host 拨号失败，进 metric=failure），
	// 这样 filter_engine 永远走 SMTPEmailDispatcher 而非 noop，保证生产可观测性。
	// 后续 task 把 SMTP 配置接入 appconfig YAML（替换本处 env 读取）。
	emailMetrics := alarm.NewEmailMetrics(c.MetricsReg)
	emailCfg := loadEmailConfigFromEnv()
	emailDispatcher := alarm.NewSMTPEmailDispatcher(emailCfg, logger.Named("email"), emailMetrics)
	filterEngine.SetEmailDispatcher(emailDispatcher)

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
		alarmPgStore:          alarmPgStore,
		alarmLibraryService:   alarmLibraryService,
		alarmFilterRuleRepo:   alarmFilterRuleRepo,
		dataPermissionChecker: dataPermissionChecker,
		alarmSyncService:      alarmSyncService,
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
