package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // T-0192 兜底：内嵌 IANA 时区库，万一 base 镜像无 /usr/share/zoneinfo 也不静默回落 UTC

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mr"
	mrcollector "github.com/omcgo/omcgo/internal/mr/collector"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/omcgo/omcgo/internal/transfer"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-worker",
		Short: "OMC Background Worker",
		Long:  "Background worker process for PM/MR file processing and KPI calculation",
		RunE:  runWorker,
	}

	rootCmd.Flags().String("config", "cmd/worker/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWorker(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.WorkerConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Handle LOG_OUTPUT_PATHS environment variable (comma-separated)
	if outputPaths := os.Getenv("OMCGO_LOG_OUTPUT_PATHS"); outputPaths != "" {
		cfg.Log.OutputPaths = parseStringSlice(outputPaths)
	}

	ctx := context.Background()
	w, err := initWorker(ctx, &cfg)
	if err != nil {
		return err
	}
	defer w.Logger.Sync()
	w.Logger.Info("omcgo-worker starting", zap.String("config", cfgPath))

	// 启动时把 device_tasks 里仍为 pending 的任务重灌进 Redis 设备队列。
	// ZScore 去重保证多 Worker/多次重启都不会重复入队；sent 任务不在此路径，
	// 由 CPE 重连时的 RecoverPendingTasks 走陈旧阈值恢复。
	if _, err := w.TaskService.RestorePendingQueues(ctx, 0); err != nil {
		w.Logger.Warn("restore pending task queues failed", zap.Error(err))
	}

	// Register all event subscribers
	registerSubscribers(w, &cfg)

	w.Logger.Info("omcgo-worker ready, waiting for events...")
	return w.WaitAndShutdown(nil)
}

func registerSubscribers(w *workerInfra, cfg *appconfig.WorkerConfig) {
	logger := w.Logger

	// L-10：worker 端也注入 audit sink，让 Sweeper 自动 stop / Exporter 异步导出
	// 等系统级操作能写 audit_logs（actor=system，与 handler 的 actor=username 区分）。
	auditRepo := admin.NewPgAuditRepository(w.PgPool)
	audit.SetDefault(admin.NewAuditSink(auditRepo))
	audit.SetFallbackLogger(logger.Named("audit"))

	// PM Collector — wraps handler with retry+DLQ runner (T-0012 / R-106).
	counterRepo := counter.NewPgCounterRepository(w.TsPool)
	kpiRepo := kpi.NewPgKPIRepository(w.TsPool)

	// T-0164-P1：worker 端的 KPIEngine 同样走 KPI Router（按 device → product → 平台公式路由），
	// 不再依赖 carrier-based 硬编码 KPIDefinitions。这里独立构造 ProductRegistry，
	// 与后面 alarm-definition fallback 复用同一份 PgRepository / Redis cache。
	pmProductRepo := product.NewPgRepository(w.PgPool)
	pmProductMetrics := product.NewRegistryMetrics(w.MetricsReg)
	pmProductCache := product.Cache(product.NopCache{})
	if w.Redis != nil {
		pmProductCache = product.NewRedisCache(w.Redis)
	}
	pmProductRegistry := product.NewRegistry(pmProductRepo, pmProductCache, pmProductMetrics, logger)
	if err := pmProductRegistry.Refresh(context.Background()); err != nil {
		// 与 alarm-definition fallback 相同的容错策略：refresh 失败仅 WARN，让 KPI Router 跑
		// 在零 patterns 状态（所有设备都会被判 orphan，KPI 跳过 + log warn）。比 worker 整体启动失败更稳。
		logger.Warn("product registry refresh failed in worker; kpi route will be orphan-only",
			zap.Error(err))
	}
	pmDeviceRepo := device.NewPgDeviceRepository(w.PgPool)
	pmIndicatorRepo := indicator.NewPgIndicatorRepository(w.PgPool)
	pmFormulaRepo := indicator.NewPgPlatformFormulaRepository(w.PgPool)
	var pmL2Cache router.L2Cache
	if w.Redis != nil {
		pmL2Cache = router.NewRedisCache(w.Redis)
	}
	pmKPIRouter, err := router.New(
		pmDeviceRepo, pmProductRegistry, pmIndicatorRepo, pmFormulaRepo,
		router.Options{
			L2Cache: pmL2Cache,
			Metrics: router.NewMetrics(w.MetricsReg),
			Logger:  logger,
		},
	)
	if err != nil {
		// 仅在依赖为 nil 时返错（编程错误）— worker 启动期阻塞性失败，及时暴露。
		logger.Fatal("build kpi router failed", zap.Error(err))
	}

	kpiEngine := kpi.NewKPIEngine(counterRepo, kpiRepo, pmKPIRouter, logger)
	pmParser := collector.NewPMXMLParser()
	pmFileStore := pm.NewPgPMFileStore(w.PgPool)
	pmCollector := collector.NewPMCollector(w.MinIO, cfg.MinIO.Buckets.PMFiles, pmParser, counterRepo, kpiEngine, pmFileStore, w.EventBus, logger)
	pmMetrics := pm.NewPMMetrics(w.MetricsReg)
	pmCollector.SetMetrics(pmMetrics)
	// T-0164 G1 真机闭环：acs.upload.Handler 发的瘦 payload 只带 device_sn，
	// 由 collector 用同一个 deviceRepo 反查补齐 UUID / OUI / carrier / technology。
	pmCollector.SetDeviceLookup(pmDeviceRepo)

	// T-0164 G1 BUG-6 真根因复盘 / 方案 D：collector 用指标库白名单过滤孤儿 counter。
	// 复用 pmKPIRouter 的 LookupByDevice：route.Counters[].ReportKey 即设备所属产品在
	// perf_indicators_{enb,gnb,gsm} 中注册的 counter 上报名全集（PM-P2 按 report_key 建键，
	// 命中后把上报名改写成编号 IndicatorID 落库）。fail-open（lookup 失败 / 空集合时跳过
	// 过滤，保留全量入库），细节见 collector.filterByWhitelist。
	pmCollector.SetCounterWhitelist(&routerCounterWhitelist{r: pmKPIRouter, log: logger})

	// Runner wires retry + DLQ instrumentation around the PM handler.
	// dlqRepo + runnerMetrics are scoped to the worker process; admin handler
	// in app process reads the same dead_letters table directly.
	dlqRepo := dlq.NewPgRepository(w.PgPool)
	runnerMetrics := runner.NewMetrics(w.MetricsReg)
	pmRunner := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, w.EventBus, runnerMetrics, logger)
	pmCollector.SetRunner(pmRunner)

	if err := pmCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe PM collector", zap.Error(err))
	}
	logger.Info("PM collector started with retry+DLQ runner")

	// Alarm Receiver + Sync
	alarmPgStore := alarm.NewPgAlarmStore(w.PgPool, w.TsPool)
	alarmRedisStore := alarm.NewRedisAlarmStore(w.Redis)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, w.Carriers, w.EventBus, logger)
	alarmMetrics := alarm.NewAlarmMetrics(w.MetricsReg)
	alarmEngine.SetMetrics(alarmMetrics)
	alarmFilterRuleRepo := alarm.NewPgAlarmFilterRuleRepository(w.PgPool)
	webhookMetrics := alarm.NewWebhookMetrics(w.MetricsReg)
	webhookDispatcher := alarm.NewHTTPWebhookDispatcher(logger.Named("webhook"), webhookMetrics)
	deadLetterRepo := alarm.NewPgDeadLetterRepository(w.PgPool)
	filterEngine := alarm.NewFilterEngine(alarmFilterRuleRepo, alarmPgStore, webhookDispatcher, deadLetterRepo, webhookMetrics, logger)
	filterEngine.SetDeviceGroupResolver(alarm.NewPgDeviceGroupResolver(w.PgPool))
	emailMetrics := alarm.NewEmailMetrics(w.MetricsReg)
	emailCfg := loadEmailConfigFromEnv()
	emailDispatcher := alarm.NewSMTPEmailDispatcher(emailCfg, logger.Named("email"), emailMetrics)
	filterEngine.SetEmailDispatcher(emailDispatcher)
	alarmEngine.SetFilterEngine(filterEngine)

	// Alarm Sync Service (creates GPV tasks to query device alarms)
	alarmSyncService := alarm.NewAlarmSyncService(w.TaskService, w.Redis, w.EventBus, logger)
	if err := alarmSyncService.Subscribe(); err != nil {
		logger.Warn("subscribe alarm sync service", zap.Error(err))
	}

	// Alarm Sync Processor (handles GPV responses, applies diff)
	alarmDeviceRepo := device.NewPgDeviceRepository(w.PgPool)
	alarmSyncProcessor := alarm.NewAlarmSyncProcessor(alarmEngine, alarmPgStore, alarmSyncService, w.EventBus, logger).WithDeviceReader(alarmDeviceRepo)
	if err := alarmSyncProcessor.Start(context.Background()); err != nil {
		logger.Warn("start alarm sync processor", zap.Error(err))
	}

	// T-0098 P2-10：构造 AlarmDefinition Registry + ProductRegistry adapter，启用 fallback 决策。
	// 任何一步失败都仅记 WARN 后退化到旧路径（不阻塞 worker 启动）。
	alarmReceiver := alarm.NewAlarmReceiver(alarmEngine, w.EventBus, logger).WithDeviceReader(alarmDeviceRepo)
	expeditedDeviceRepo := device.NewPgDeviceRepository(w.PgPool)
	expeditedReceiver := alarm.NewExpeditedEventReceiver(alarmEngine, expeditedDeviceRepo, w.EventBus, logger)
	alarmDefRepo := definition.NewPgRepository(w.PgPool)
	alarmDefMetrics := definition.NewRegistryMetrics(w.MetricsReg)
	alarmDefRegistry := definition.NewRegistry(alarmDefRepo, alarmDefMetrics, logger)
	if err := alarmDefRegistry.Refresh(context.Background()); err != nil {
		logger.Warn("alarm-definition registry refresh failed; fallback disabled", zap.Error(err))
	} else {
		// T-0164-P1：复用 KPI Router 已构造的 pmProductRegistry，避免
		// 重复 RegistryMetrics MustRegister 触发 Prometheus duplicate collector panic。
		alarmReceiver, expeditedReceiver, _ = wireUnknownAlarmFallback(alarmReceiver, expeditedReceiver, alarmDefRegistry, pmProductRegistry)
		logger.Info("alarm-definition fallback enabled",
			zap.Int("definitions_loaded", alarmDefRegistry.Count()))
	}
	if err := alarmReceiver.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe alarm receiver", zap.Error(err))
	}
	logger.Info("alarm receiver + sync started")

	// Expedited Alarm Receiver (real-time alarm notifications via VALUE CHANGE ExpeditedEvent)
	if err := expeditedReceiver.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe expedited alarm receiver", zap.Error(err))
	}
	logger.Info("expedited alarm receiver started")

	// Frequent abnormal reboot monitor (F04)：滑动窗口内异常重启 >=阈值抬升告警。
	rebootMonitor := alarm.NewRebootMonitor(alarmEngine, w.Redis, logger)
	if err := rebootMonitor.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe reboot monitor", zap.Error(err))
	}
	logger.Info("reboot monitor started")

	// 设备离线超时告警清理：离线满 1 小时仍未恢复时，把当前告警归档到历史告警。
	offlineAlarmCleaner := alarm.NewOfflineAlarmCleaner(alarmDeviceRepo, alarmPgStore, alarmEngine, logger)
	go offlineAlarmCleaner.Run(context.Background())
	logger.Info("offline alarm cleaner started",
		zap.Duration("interval", alarm.DefaultOfflineAlarmCleanupInterval),
		zap.Duration("threshold", alarm.DefaultOfflineAlarmCleanupThreshold))

	// Reboot Task Closer (F01/F06)：M Reboot Inform 兜底收敛未 ACK 的 Reboot/FactoryReset 任务。
	rebootCloser := task.NewRebootCloser(w.TaskRepo, w.TaskService, logger)
	if err := rebootCloser.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe reboot task closer", zap.Error(err))
	}
	logger.Info("reboot task closer started")

	// T-0157 C5: 消息中心 task 订阅器 — 监听 task.created/completed/failed 事件，按 user_id
	// 隔离写 notifications 表，dedup_key=task.ID 保证同 task 多次状态变更 upsert 同一行。
	// 依赖 task.created 主题（service.CreateTask 末尾 publish）+ 已有 task.completed/failed。
	notifRepo := notification.NewPgRepository(w.PgPool)
	notifSvc := notification.NewService(notifRepo, nil, logger)
	taskSubscriber := notification.NewTaskSubscriber(notifSvc, logger)
	if err := taskSubscriber.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe notification task subscriber", zap.Error(err))
	} else {
		logger.Info("notification task subscriber started (T-0157 C5)")
	}

	// T-0157 C2: 任务过期扫描器 — 周期把 expires_at < now 且仍 pending/sent 的任务标记为 expired，
	// 并 publish 终态事件供消息中心订阅器消费（事件复用 task.failed 主题，订阅器按 status 区分）。
	// SweepInterval <= 0 时跳过启动（生产配置见 cmd/app/etc/config.*.yaml task 段）。
	if cfg.Task.SweepIntervalSeconds > 0 {
		taskSweeper := task.NewExpiredSweeper(
			w.TaskRepo, w.TaskService,
			time.Duration(cfg.Task.SweepIntervalSeconds)*time.Second,
			0, // batchSize 默认 100
			logger,
		)
		go taskSweeper.Run(context.Background())
		logger.Info("task expired sweeper started (T-0157 C2)",
			zap.Int("interval_seconds", cfg.Task.SweepIntervalSeconds))
	} else {
		logger.Info("task expired sweeper disabled (sweep_interval_seconds <= 0)")
	}

	// MR Collector（worker 端保留 — 文件 I/O 类，与 PM collector 同进程更合理）
	mrStore := mr.NewPgMRStore(w.PgPool, w.TsPool)
	mrCollector := mrcollector.NewMRCollector(w.MinIO, cfg.MinIO.Buckets.MRFiles, mrStore, w.EventBus, logger)
	// 注入 DeviceLookup：upload handler 直传路径的 mr.file.received payload 不带
	// device_id，由 collector 按 device_sn 反查（参 internal/acs/upload publishMRFileReceivedEvent）。
	mrCollector.SetDeviceLookup(device.NewPgDeviceRepository(w.PgPool))
	if err := mrCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe MR collector", zap.Error(err))
	}
	logger.Info("MR collector started")

	// F05 MR 任务管理（scheduler / heartbeat / cleaner / completion）已迁到 app 进程，
	// 详见 cmd/app/provider/modules.go 中的 mrtask 装配段。
	// 原因：dispatcher 需要 ParamRegistry + ProductRegistry 做 standardPath → privatePath 翻译，
	// 这两个 Registry 当前仅在 app 进程加载。worker 端不重复加载，避免字典加载放大。

	// Transfer Bridge
	deviceRepo := device.NewPgDeviceRepository(w.PgPool)
	transferDeduper := event.NewDeduper(w.Redis, 24*time.Hour, logger)
	transferBridge := transfer.NewTransferBridge(
		deviceRepo, w.MinIO,
		cfg.MinIO.Buckets,
		w.EventBus, transferDeduper, logger,
	)
	if err := transferBridge.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe transfer bridge", zap.Error(err))
	}
	logger.Info("transfer bridge started")

	// Backup Executor
	backupTaskRepo := backup.NewPgTaskRepository(w.PgPool)
	connReqClient := connreq.NewClient(w.Redis, logger)
	backupExecutor := backup.NewBackupExecutor(
		backupTaskRepo, deviceRepo, w.TaskService, connReqClient,
		w.EventBus, logger,
	)
	// 从 sys_configs 读取 ACS 传输配置（界面「系统管理 → ACS 传输」可配置，运行时生效）。
	// YAML 不再提供默认值，配置全部源自 DB。
	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	backupTransferPolicy := transfercfg.NewPolicy(
		transfercfg.Snapshot{},
		func(ctx context.Context, category, key string) (string, bool) {
			cfg, err := sysConfigRepo.GetByKey(ctx, category, key)
			if err != nil || cfg == nil {
				return "", false
			}
			return cfg.Value, true
		},
	)
	backupExecutor.SetTransferProvider(backupTransferPolicy)
	// T-0073 Phase 1: opt-in backup-failure alarm publish via PolicyService.
	// Worker shares the same backup_policies table as app; reads policy on each
	// failure to honour latest AlertOnFailure flag.
	backupPolicyRepo := backup.NewPgPolicyRepository(w.PgPool)
	backupPolicyService := backup.NewPolicyService(backupPolicyRepo, logger)
	backupPolicyMetrics := backup.NewPolicyMetrics(w.MetricsReg)
	backupFTPConfigRepo := backup.NewPgFTPConfigRepository(w.PgPool)
	backupExecutor.SetFTPConfigRepository(backupFTPConfigRepo)
	backupExecutor.SetPolicyEnforcement(backupPolicyService, backupPolicyMetrics)
	if err := backupExecutor.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe backup executor", zap.Error(err))
	}
	logger.Info("backup executor started (with T-0073 failure-alarm enforcement)")

	// T-0137 / M2: TR069 报文跟踪 capture 消费者 + 超时巡检 + purge 处理。
	// JetStream WorkQueuePolicy 群组消费 trace.message.captured，批量去抖动后写 trace_messages。
	// 多 worker 实例靠 QueueSubscribe 自动负载均衡互不重复。
	traceRepo := trace.NewPgRepository(w.PgPool)
	var traceBulk *trace.BulkStore
	if w.MinIO != nil && cfg.MinIO.Buckets.TraceBulk != "" {
		traceBulk = trace.NewBulkStore(w.MinIO, cfg.MinIO.Buckets.TraceBulk)
		logger.Info("trace bulk store enabled",
			zap.String("bucket", cfg.MinIO.Buckets.TraceBulk),
			zap.Int("inline_max_bytes", trace.MaxInlinePayloadBytes))
	}
	traceConsumer := trace.NewCaptureConsumer(traceRepo, trace.DefaultCaptureConsumerConfig(), logger)
	if traceBulk != nil {
		traceConsumer.SetBulkStore(traceBulk)
	}
	// M3-01：Prometheus 指标 — worker 端是落库主力，captured/dropped 计数都从这里出
	traceMetrics := trace.NewMetrics(w.MetricsReg)
	traceConsumer.SetMetrics(traceMetrics)
	if _, err := traceConsumer.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe trace capture consumer", zap.Error(err))
	} else {
		logger.Info("trace capture consumer started (T-0137 M2)")
	}
	// 巡检：60s 周期把 expires_at<now 的 running 任务转 stopped + 发事件
	// 订阅 trace.task.purged：worker 异步 DELETE PG + 删 MinIO 对象（注入 traceBulk 后启用）
	var sweeperBulk trace.BulkObjectDeleter
	if traceBulk != nil {
		sweeperBulk = traceBulk
	}
	traceSweeper := trace.NewSweeper(traceRepo, w.EventBus, sweeperBulk, trace.DefaultSweeperConfig(), logger)
	traceSweeper.SetMetrics(traceMetrics)
	if _, err := traceSweeper.Start(context.Background()); err != nil {
		logger.Warn("start trace sweeper", zap.Error(err))
	} else {
		logger.Info("trace sweeper started (T-0137 M2)")
	}
	// 异步导出：订阅 trace.export.requested → 生成 XML 写 MinIO exchange
	if w.MinIO != nil && cfg.MinIO.Buckets.Exchange != "" {
		traceExporter := trace.NewExporter(traceRepo, traceBulk, w.MinIO,
			trace.DefaultExporterConfig(cfg.MinIO.Buckets.Exchange), logger)
		if _, err := traceExporter.Subscribe(w.EventBus); err != nil {
			logger.Warn("subscribe trace exporter", zap.Error(err))
		} else {
			logger.Info("trace exporter subscribed (T-0137 M2-08)",
				zap.String("exchange_bucket", cfg.MinIO.Buckets.Exchange))
		}
	}

	// Report Generator
	reportDefRepo := report.NewPgDefinitionRepository(w.PgPool)
	reportRecordRepo := report.NewPgRecordRepository(w.PgPool)
	reportBucket := cfg.MinIO.Buckets.Reports
	if reportBucket == "" {
		reportBucket = "reports"
	}
	reportGenerator := report.NewReportGenerator(
		reportRecordRepo, reportDefRepo, kpiRepo, alarmPgStore,
		w.MinIO, reportBucket,
		w.EventBus, logger,
	)
	if err := reportGenerator.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe report generator", zap.Error(err))
	}
	logger.Info("report generator started")

	// M2: TransferCompleteRouter — 订阅 device.inform.transfer_complete，按 CommandKey 回写 backup/restore_tasks 终态
	restoreRepo := backup.NewPgRestoreTaskRepository(w.PgPool)
	tcRouter := backup.NewTransferCompleteRouter(backupTaskRepo, restoreRepo, nil, logger)
	if err := tcRouter.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe backup transfer-complete router", zap.Error(err))
	}
	logger.Info("backup transfer-complete router started")

	// PM 设备上线自动下发 PM 上传配置（KPI 上报参数整理.md 三参数）
	// 仅在 cfg.PM.AutoSetupOnOnline=true 时启用；test 环境默认关闭防止干扰压测
	if cfg.PM.AutoSetupOnOnline {
		pmOnlineSub := pm.NewOnlineSubscriber(
			w.TaskService,
			cfg.PM.UploadURLTemplate,
			cfg.PM.EnableValue,
			cfg.PM.PeriodicUploadInterval,
			logger,
		)
		if err := pmOnlineSub.Subscribe(w.EventBus); err != nil {
			logger.Warn("subscribe pm online subscriber failed", zap.Error(err))
		} else {
			logger.Info("pm online subscriber started (auto SPV on device.online)")
		}
	} else {
		logger.Info("pm online subscriber disabled (cfg.pm.auto_setup_on_online=false)")
	}

	// T-0164-P5 / G5 + T-0164-P8 / G8：PM 自然桶聚合 cron + asyncjob 框架接入。
	// 复用上文已构造的 pmKPIRouter（KPI 反算依赖路由）；新开 4 个 cron runner +
	// sweeper + 触发器（hourly @:05 / daily 00:05 / weekly 周一 00:10 / monthly 1日 00:15）。
	// T-0192：日/周/月桶按业务时区切本地零点，loc 同时穿入窗口计算与 cron 调度。
	pmLoc := resolvePMTimezone(cfg.PM.Timezone, logger)
	// KPI 导出文件落地桶：复用报表桶（设计 §5.6）；缺省回退 "reports"。
	exportBucket := cfg.MinIO.Buckets.Reports
	if exportBucket == "" {
		exportBucket = "reports"
	}
	startPMAggregatorPipeline(context.Background(), w, pmKPIRouter, pmLoc, exportBucket)

	// T-0164-P7 / G7：自定义聚合任务（oneshot + continuous）。
	// 复用同一 kpiRouter；4 个 worker 抢 pm_tasks 中 task_subtype='adhoc_aggregation' 的 pending 行；
	// continuous scheduler 单 goroutine 每分钟扫 scheduled 任务切回 pending。
	startPMAdhocPipeline(context.Background(), w, pmKPIRouter, cfg)

	// M3: 周期备份调度器 + 任务 reaper（event-loss 兜底恢复）
	backupScheduleRepo := backup.NewPgScheduleRepository(w.PgPool)
	backupService := backup.NewService(backupTaskRepo, backupScheduleRepo, w.EventBus, logger)
	schedulerMetrics := backup.NewSchedulerMetrics(w.MetricsReg)
	periodScheduler := backup.NewPeriodScheduler(backupScheduleRepo, backupService, schedulerMetrics, logger)
	if err := periodScheduler.Start(context.Background()); err != nil {
		logger.Warn("start backup period scheduler", zap.Error(err))
	} else {
		logger.Info("backup period scheduler started")
	}
	// Reload 通道：app 进程 schedule CRUD 后发布 SubjectBackupScheduleChanged
	if err := periodScheduler.SubscribeReload(w.EventBus); err != nil {
		logger.Warn("subscribe backup schedule reload", zap.Error(err))
	}
	taskReaper := backup.NewTaskReaper(backupTaskRepo, w.EventBus, schedulerMetrics, 0, 0, logger)
	taskReaper.Start()
	logger.Info("backup task reaper started")

	// T-0178 P2: parammodel 自定义 XML 备份清理 cron(每天默认 03:00)。
	// 扫 customDir 下 .deleted.<ts>/.bak.<ts>(> retentionDays 清) + .tmp.<uuid>(> 1h 清)。
	startParamModelBackupCleanup(w, cfg, logger)

	// T-0180 P2: indicator 自定义 XML 备份清理 cron(对标 T-0178,同样 03:00 默认)。
	// 扫 customDir/<enb,gsm,gnb>/ 三制式子目录下 .deleted/.bak/.tmp 残留。
	startIndicatorBackupCleanup(w, cfg, logger)

	// 告警库自定义 XML 备份清理 cron(对标 indicator,扁平 customDir,同 03:00 默认)。
	startAlarmBackupCleanup(w, cfg, logger)

	// T-0182 P2: 数据字典数据源每日同步 cron(每天 02:00 BJT)。
	// 遍历所有 source_table != NULL AND status=true 的字典,SyncEngine 拉源表
	// distinct 行 → upsert/delete auto 项。失败发 dictionary.refresh.failed 事件。
	startDictSourceDailySync(w, cfg, logger)

	// T-0184: adhoc 过期任务定义清理 cron(每天 03:00)。
	// 删 mode='oneshot' 且 is_builtin=false 且 created_at < now()-expire_days 天 的任务定义行；
	// 只删 pm_tasks 行,绝不碰结果表 pm_adhoc_aggregation_results(两层口径分离,设计 §2.3)。
	startPMAdhocExpireCleanup(w, logger)
}

// startPMAdhocExpireCleanup 启动 T-0184 adhoc 过期任务定义清理 cron。
//
// 行为(对标备份清理 cron):
//   - 注册 cron(默认 "0 3 * * *");无效表达式 fallback 默认值(此处固定默认,无配置项)
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后过期任务堆积
//   - 单实例假设(单 worker 部署),无锁保护;横扩需加 PG advisory lock
func startPMAdhocExpireCleanup(w *workerInfra, logger *zap.Logger) {
	cleanup := adhoc.NewExpireCleanup(w.PgPool, logger)

	c := cron.New()
	if _, err := c.AddFunc(adhoc.DefaultExpireCleanupCron, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, runErr := cleanup.Run(ctx); runErr != nil {
			logger.Warn("adhoc expire cleanup run failed", zap.Error(runErr))
		}
	}); err != nil {
		logger.Warn("invalid adhoc expire cleanup cron; skipping",
			zap.String("cron", adhoc.DefaultExpireCleanupCron), zap.Error(err))
		return
	}
	c.Start()
	logger.Info("adhoc expire cleanup cron started",
		zap.String("cron", adhoc.DefaultExpireCleanupCron))

	// 启动期延迟 catch-up(给 app + 字典加载 30s 缓冲)
	go func() {
		time.Sleep(30 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		deleted, err := cleanup.Run(ctx)
		if err != nil {
			logger.Warn("adhoc expire cleanup startup catch-up failed", zap.Error(err))
			return
		}
		if deleted > 0 {
			logger.Info("adhoc expire cleanup startup catch-up",
				zap.Int64("deleted", deleted))
		}
	}()
}

// startParamModelBackupCleanup 启动 T-0178 自定义 XML 备份清理 cron。
//
// 行为:
//   - 注册 cron(默认 "0 3 * * *");无效表达式 fallback 默认值
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后备份堆积
//   - 单实例(单 worker 部署)假设,无锁保护;横扩需加 PG advisory lock(P1 不做)
func startParamModelBackupCleanup(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	// 解析配置 + 应用默认值
	pmCfg := cfg.DictLoader.ParamModel
	customSub := pmCfg.CustomDirectory
	if customSub == "" {
		customSub = parammodel.CustomDirSubdir
	}
	customDir := filepath.Join(cfg.DictLoader.XMLBaseDir, customSub)
	cronExpr := pmCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = parammodel.DefaultBackupCleanupCron
	}

	metrics := parammodel.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := parammodel.NewBackupCleanup(customDir, pmCfg.BackupRetentionDays, metrics, logger)

	c := cron.New()
	if _, err := c.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, runErr := cleanup.Run(ctx); runErr != nil {
			logger.Warn("parammodel backup cleanup run failed", zap.Error(runErr))
		}
	}); err != nil {
		logger.Warn("invalid parammodel backup cleanup cron; using default",
			zap.String("cron", cronExpr), zap.Error(err))
		_, _ = c.AddFunc(parammodel.DefaultBackupCleanupCron, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			_, _ = cleanup.Run(ctx)
		})
	}
	c.Start()
	logger.Info("parammodel backup cleanup cron started",
		zap.String("custom_dir", customDir),
		zap.String("cron", cronExpr),
		zap.Int("retention_days", pmCfg.BackupRetentionDays))

	// 启动期延迟 catch-up(给 app + 字典加载 30s 缓冲)
	go func() {
		time.Sleep(30 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		swept, err := cleanup.Run(ctx)
		if err != nil {
			logger.Warn("parammodel backup cleanup startup catch-up failed", zap.Error(err))
			return
		}
		if swept > 0 {
			logger.Info("parammodel backup cleanup startup catch-up",
				zap.Int("swept", swept))
		}
	}()
}

// startIndicatorBackupCleanup 启动 T-0180 自定义 indicator XML 备份清理 cron。
//
// 行为(对标 T-0178 startParamModelBackupCleanup):
//   - 注册 cron(默认 "0 3 * * *");无效表达式 fallback 默认值
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后备份堆积
//   - 单实例假设;横扩需加 PG advisory lock(P1 不做)
//   - customDir 是三制式子目录的根 (.../indicator-library-custom),
//     Run() 遍历 enb/gsm/gnb 各自的子目录
func startIndicatorBackupCleanup(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	// 解析配置 + 应用默认值
	indCfg := cfg.DictLoader.Indicator
	customSub := indCfg.CustomBaseDirectory
	if customSub == "" {
		customSub = indicator.CustomDirSubdir
	}
	customDir := filepath.Join(cfg.DictLoader.XMLBaseDir, customSub)
	cronExpr := indCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = indicator.DefaultIndicatorBackupCleanupCron
	}

	metrics := indicator.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := indicator.NewBackupCleanup(customDir, indCfg.BackupRetentionDays, metrics, logger)

	c := cron.New()
	if _, err := c.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, runErr := cleanup.Run(ctx); runErr != nil {
			logger.Warn("indicator backup cleanup run failed", zap.Error(runErr))
		}
	}); err != nil {
		logger.Warn("invalid indicator backup cleanup cron; using default",
			zap.String("cron", cronExpr), zap.Error(err))
		_, _ = c.AddFunc(indicator.DefaultIndicatorBackupCleanupCron, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			_, _ = cleanup.Run(ctx)
		})
	}
	c.Start()
	logger.Info("indicator backup cleanup cron started",
		zap.String("custom_dir", customDir),
		zap.String("cron", cronExpr),
		zap.Int("retention_days", indCfg.BackupRetentionDays))

	// 启动期延迟 catch-up(给 app + 字典加载 30s 缓冲)
	go func() {
		time.Sleep(30 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		swept, err := cleanup.Run(ctx)
		if err != nil {
			logger.Warn("indicator backup cleanup startup catch-up failed", zap.Error(err))
			return
		}
		if swept > 0 {
			logger.Info("indicator backup cleanup startup catch-up",
				zap.Int("swept", swept))
		}
	}()
}

// startAlarmBackupCleanup 启动告警库自定义 XML 备份清理 cron(对标 indicator)。
//
//   - 注册 cron(默认 "0 3 * * *");无效表达式 fallback 默认值
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后备份堆积
//   - customDir 是扁平目录 (.../alarm-definitions-custom),Run() 直接扫单层
//   - 单实例假设;横扩需加 PG advisory lock(与 indicator 同,P1 不做)
func startAlarmBackupCleanup(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	alarmCfg := cfg.DictLoader.AlarmDefinition
	customSub := alarmCfg.CustomDirectory
	if customSub == "" {
		customSub = definition.CustomDirSubdir
	}
	customDir := filepath.Join(cfg.DictLoader.XMLBaseDir, customSub)
	cronExpr := alarmCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = definition.DefaultAlarmBackupCleanupCron
	}

	metrics := definition.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := definition.NewBackupCleanup(customDir, alarmCfg.BackupRetentionDays, metrics, logger)

	c := cron.New()
	if _, err := c.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, runErr := cleanup.Run(ctx); runErr != nil {
			logger.Warn("alarm backup cleanup run failed", zap.Error(runErr))
		}
	}); err != nil {
		logger.Warn("invalid alarm backup cleanup cron; using default",
			zap.String("cron", cronExpr), zap.Error(err))
		_, _ = c.AddFunc(definition.DefaultAlarmBackupCleanupCron, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			_, _ = cleanup.Run(ctx)
		})
	}
	c.Start()
	logger.Info("alarm backup cleanup cron started",
		zap.String("custom_dir", customDir),
		zap.String("cron", cronExpr),
		zap.Int("retention_days", alarmCfg.BackupRetentionDays))

	go func() {
		time.Sleep(30 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		swept, err := cleanup.Run(ctx)
		if err != nil {
			logger.Warn("alarm backup cleanup startup catch-up failed", zap.Error(err))
			return
		}
		if swept > 0 {
			logger.Info("alarm backup cleanup startup catch-up", zap.Int("swept", swept))
		}
	}()
}

// startDictSourceDailySync 启动 T-0182 数据字典数据源每日同步 cron。
//
// 行为:
//   - 注册 cron(默认 "0 0 2 * * *",每日 02:00 BJT);无效表达式 fallback 默认值
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后字典数据严重落后
//   - 单字典 5s 超时,任一失败不中断其它字典(SyncEngine 内部已实现)
//   - 失败的字典 SyncEngine 已写 last_refresh_error 列;此处额外通过 EventBus
//     发 dictionary.refresh.failed 事件预留给 notification 通道(P3+/F04)
//   - 单实例假设;横扩前要么用 Redis SETNX 选 leader,要么按"app 进程直接调"
//     而不是 worker 多实例同时跑 — v1 接受单 worker 部署
func startDictSourceDailySync(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	// v1 用包内常量,不入 appconfig(P3+ 真有用户改需求再迁配置项)。
	cronExpr := admin.DefaultDictSourceDailyCron
	_ = cfg // 预留

	// 构造 admin DictionaryService(worker 自有,与 app 进程独立)。
	// 仅注入 SyncAll 路径需要的最小依赖:dict repo + detail repo + Registry + SyncEngine。
	dictRepo := admin.NewPgDictionaryRepository(w.PgPool)
	dictDetailRepo := admin.NewPgDictionaryDetailRepository(w.PgPool)
	dictService := admin.NewDictionaryService(dictRepo, dictDetailRepo)
	reg, err := admin.LoadDictSourceRegistry()
	if err != nil {
		logger.Warn("dict_source_daily_skipped_registry_failed", zap.Error(err))
		return
	}
	engine := admin.NewDictSyncEngine(reg, w.PgPool, dictRepo, dictDetailRepo, logger.Named("dict_source"))
	engine.SetMetrics(admin.NewDictSourceMetrics(w.MetricsReg))
	dictService.SetSourceWiring(reg, engine, logger.Named("dict_source"))

	runOnce := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		ok, failed := dictService.SyncSourceBoundAll(ctx)
		logger.Info("dict_source_daily_finished",
			zap.Int("ok", ok), zap.Int("failed", failed))
		// 失败时发 EventBus 事件预留给 notification(payload 由订阅方按需消费;
		// v1 不主动遍历失败明细 — last_refresh_error 列已存,P3 通知中心如需逐条
		// 推送可订阅本事件 + 查询 DB 拿明细)。
		if failed > 0 && w.EventBus != nil {
			evt, evtErr := event.NewEvent(event.SubjectDictionaryRefreshFailed, map[string]any{
				"ok":     ok,
				"failed": failed,
			})
			if evtErr != nil {
				logger.Warn("dict_source_build_failed_event_failed", zap.Error(evtErr))
			} else if pubErr := w.EventBus.Publish(context.Background(),
				event.SubjectDictionaryRefreshFailed, evt); pubErr != nil {
				logger.Warn("dict_source_publish_failed_event_failed", zap.Error(pubErr))
			}
		}
	}

	c := cron.New()
	if _, err := c.AddFunc(cronExpr, runOnce); err != nil {
		logger.Warn("invalid dict_source daily cron; using default",
			zap.String("cron", cronExpr), zap.Error(err))
		_, _ = c.AddFunc(admin.DefaultDictSourceDailyCron, runOnce)
	}
	c.Start()
	logger.Info("dict_source daily cron started", zap.String("cron", cronExpr))

	// 启动期延迟 catch-up(给 app + 字典加载 30s 缓冲)。
	go func() {
		time.Sleep(30 * time.Second)
		runOnce()
	}()
}

func wireUnknownAlarmFallback(
	alarmReceiver *alarm.AlarmReceiver,
	expeditedReceiver *alarm.ExpeditedEventReceiver,
	alarmDefRegistry *definition.Registry,
	productRegistry *product.Registry,
) (*alarm.AlarmReceiver, *alarm.ExpeditedEventReceiver, definition.ProductResolver) {
	if alarmReceiver == nil || expeditedReceiver == nil || alarmDefRegistry == nil || productRegistry == nil {
		return alarmReceiver, expeditedReceiver, nil
	}
	productResolver := &definition.ProductRegistryAdapter{Registry: productRegistry}
	alarmReceiver = alarmReceiver.WithAlarmDefRegistry(alarmDefRegistry, productResolver)
	expeditedReceiver = expeditedReceiver.WithAlarmDefRegistry(alarmDefRegistry, productResolver)
	return alarmReceiver, expeditedReceiver, productResolver
}

// parseStringSlice parses a comma-separated string into a slice.
// routerCounterWhitelist 把 *router.Router 包装成 collector.CounterWhitelist 接口。
// 通过 LookupByDevice 拿设备所属产品的 KPIRoute.Counters，转
// report_key→{编号, statis_type} 映射作为白名单。Lookup 失败时把错误透传给
// collector，由 collector 决定 fail-open（log warn + 不过滤）。
//
// PM-P2：建键锚点改为 report_key（上报名、入库不可改的解析契约），value 带指标
// 编号（IndicatorID），collector.filterByWhitelist 命中后把上报名改写成编号落库
// （落库即编号化）。statis_type 一并回填驱动 G5 自然桶聚合（T-0164-G6 BUG-A）。
type routerCounterWhitelist struct {
	r   *router.Router
	log *zap.Logger
}

func (a *routerCounterWhitelist) LookupCounters(ctx context.Context, deviceSN string) (map[string]collector.CounterMeta, error) {
	route, err := a.r.LookupByDevice(ctx, deviceSN)
	if err != nil {
		// 注意：ErrProductNotMatched / ErrInvalidProductMetadata 是业务上的"空白名单"信号，
		// 不是技术错误。返回 (nil, err) 让 collector log warn 后 fail-open（不过滤 = 入全量）。
		// 与其他真技术错误（DB 故障）的处理一致。
		return nil, err
	}
	if route == nil || len(route.Counters) == 0 {
		return nil, nil
	}
	out := make(map[string]collector.CounterMeta, len(route.Counters))
	for _, c := range route.Counters {
		// 按 report_key 建键（PM-P2）。report_key 缺省的 counter 不入白名单——
		// 没有上报名锚点就无从匹配 PM 文件里的上报计数器。
		if c.ReportKey == "" {
			continue
		}
		out[c.ReportKey] = collector.CounterMeta{
			IndicatorID: c.IndicatorID,
			StatisType:  c.StatisType,
		}
	}
	return out, nil
}

func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func loadEmailConfigFromEnv() alarm.EmailConfig {
	port, _ := strconv.Atoi(os.Getenv("OMC_SMTP_PORT"))
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
