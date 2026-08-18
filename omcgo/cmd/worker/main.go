package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // T-0192 兜底：内嵌 IANA 时区库，万一 base 镜像无 /usr/share/zoneinfo 也不静默回落 UTC

	"golang.org/x/sync/singleflight"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	coreoutbox "github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/core/rawarchive"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/geofence"
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
	pmmetrics "github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
	"github.com/omcgo/omcgo/internal/pm/slothealth"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/omcgo/omcgo/internal/transfer"
	"github.com/omcgo/omcgo/internal/tsdbsync"
	"github.com/redis/go-redis/v9"
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
	aggregationCfg := pmstream.ConfigFromEnv()
	pmConcurrency := effectivePMConsumerConcurrency(cfg.PMConsumerConcurrency)
	finalizeConcurrency := 0
	aggregationConsumerConcurrency := 0
	if aggregationCfg.Enabled {
		if aggregationCfg.FinalizeConcurrency <= 0 ||
			aggregationCfg.ConsumerConcurrency <= 0 ||
			aggregationCfg.FinalizeConcurrency > pmMaxAggregationConcurrency ||
			aggregationCfg.ConsumerConcurrency > pmMaxAggregationConcurrency {
			return fmt.Errorf(
				"PM aggregation concurrency must be within 1..%d (consumer=%d finalize=%d)",
				pmMaxAggregationConcurrency,
				aggregationCfg.ConsumerConcurrency, aggregationCfg.FinalizeConcurrency,
			)
		}
		finalizeConcurrency = aggregationCfg.FinalizeConcurrency
		aggregationConsumerConcurrency = aggregationCfg.ConsumerConcurrency
	}
	requiredTSDBConns := pmTSDBConnectionBudget(
		pmConcurrency,
		finalizeConcurrency,
		aggregationConsumerConcurrency,
	)
	if cfg.TSDB.MaxConns < requiredTSDBConns {
		return fmt.Errorf(
			"tsdb.max_conns=%d is below concurrent PM budget=%d (ingest=%d finalize=%d aggregation=%d*%d reserve=%d)",
			cfg.TSDB.MaxConns, requiredTSDBConns, pmConcurrency, finalizeConcurrency,
			pmAggregationTSDBWriteConsumers, aggregationConsumerConcurrency,
			pmTSDBConnectionReserve,
		)
	}

	// Handle LOG_OUTPUT_PATHS environment variable (comma-separated)
	if outputPaths := os.Getenv("OMCGO_LOG_OUTPUT_PATHS"); outputPaths != "" {
		cfg.Log.OutputPaths = parseStringSlice(outputPaths)
	}

	// #2: 启动期凭证 guardrail —— 生产环境下检测到默认/占位/已泄露凭证即拒启。
	// Worker 持有 PG / TSDB / MinIO 凭证；dev/test 自动跳过。
	if err := appconfig.GuardProductionSecrets(
		appconfig.SecretCheck{Field: "db.dsn(password)", Value: cfg.DB.DSN, IsDSN: true},
		appconfig.SecretCheck{Field: "tsdb.dsn(password)", Value: cfg.TSDB.DSN, IsDSN: true},
		appconfig.SecretCheck{Field: "minio.access_key", Value: cfg.MinIO.AccessKey},
		appconfig.SecretCheck{Field: "minio.secret_key", Value: cfg.MinIO.SecretKey},
	); err != nil {
		return fmt.Errorf("生产凭证校验失败: %w", err)
	}

	ctx := context.Background()
	w, err := initWorker(ctx, &cfg)
	if err != nil {
		return err
	}
	defer w.Logger.Sync()
	w.Logger.Info("omcgo-worker starting", zap.String("config", cfgPath))

	// 大数据增量升级：registerSubscribers 内 PM 聚合的一次性升级整理（版本元数据
	// 回填 / 活跃窗口恢复 / 补建索引）在时序库历史窗口量大时可能以小时计。metrics
	// /healthz 先于此启动 —— 否则 :9092 长时间无人监听，部署健康门禁与监控全盲
	// （线上事故：升级后 install 因 worker /healthz 探测失败而中止）。
	w.StartMetrics()

	// Register all event subscribers. Geofence consumers are part of the
	// acceptance-critical control plane, so a missing JetStream stream must
	// fail startup instead of silently disabling alarms and device control.
	if err := registerSubscribers(w, &cfg); err != nil {
		return err
	}
	if err := startDeviceAccessWorkers(w); err != nil {
		return fmt.Errorf("start device access workers: %w", err)
	}

	// 启动时把 device_tasks 里仍为 pending 的任务重灌进 Redis 设备队列。
	// 大库冷启动时即使有专用索引，恢复也可能受机械盘或 autovacuum 影响；放到后台
	// 执行，避免历史任务积压阻塞 metrics/healthz 和 PM 消费者启动。ZScore 去重保证
	// 多 Worker/多次重启都不会重复入队；sent 任务仍由 CPE 重连恢复路径处理。
	restoreCtx, cancelRestore := context.WithCancel(ctx)
	w.GS.Register("pending-queue-restore", 1, func(context.Context) error {
		cancelRestore()
		return nil
	})
	startPendingQueueRestore(restoreCtx, func(ctx context.Context) (task.RestoreStats, error) {
		return w.TaskService.RestorePendingQueues(ctx, 0)
	}, w.Logger)

	w.Logger.Info("omcgo-worker ready, waiting for events...")
	return w.WaitAndShutdown(nil)
}

// startPendingQueueRestore 异步执行启动期 pending 队列恢复，返回只读完成信号供测试
// 和未来编排使用。恢复失败只降级告警，不影响 worker 健康端点和其它消费者。
func startPendingQueueRestore(
	ctx context.Context,
	restore func(context.Context) (task.RestoreStats, error),
	logger *zap.Logger,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		stats, err := restore(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Warn("restore pending task queues failed", zap.Error(err))
			}
			return
		}
		logger.Info("restore pending task queues completed",
			zap.Int("scanned", stats.Scanned),
			zap.Int("pushed", stats.Pushed),
			zap.Int("skipped", stats.Skipped),
			zap.Int("failed", stats.Failed))
	}()
	return done
}

func registerSubscribers(w *workerInfra, cfg *appconfig.WorkerConfig) error {
	logger := w.Logger

	eventOutboxRepo := coreoutbox.NewPgRelayRepository(storage.NewPoolDB(w.PgPool))
	stopEventOutboxRelay, err := startEventOutboxRelay(
		context.Background(),
		eventOutboxRepo,
		w.EventBus,
		logger,
	)
	if err != nil {
		logger.Fatal("start generic event outbox relay", zap.Error(err))
	}
	w.GS.Register("event-outbox-relay", 1, func(context.Context) error {
		stopEventOutboxRelay()
		return nil
	})

	stopGeofenceCoordinator, err := startGeofenceCoordinator(
		storage.NewPoolDB(w.PgPool),
		w.EventBus,
	)
	if err != nil {
		logger.Fatal("start geofence coordinator", zap.Error(err))
	}
	w.GS.Register("geofence-coordinator", 1, func(context.Context) error {
		return stopGeofenceCoordinator()
	})

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
	w.ProductRegistry = pmProductRegistry
	if err := pmProductRegistry.Refresh(context.Background()); err != nil {
		// 与 alarm-definition fallback 相同的容错策略：refresh 失败仅 WARN，让 KPI Router 跑
		// 在零 patterns 状态（所有设备都会被判 orphan，KPI 跳过 + log warn）。比 worker 整体启动失败更稳。
		logger.Warn("product registry refresh failed in worker; kpi route will be orphan-only",
			zap.Error(err))
	}
	workerParamRegistry := newWorkerParamRegistry(w, pmProductRegistry, logger)
	pmDeviceRepo := device.NewPgDeviceRepository(w.PgPool)
	pmIndicatorRepo := indicator.NewPgIndicatorRepository(w.PgPool)
	pmFormulaRepo := indicator.NewPgPlatformFormulaRepository(w.PgPool)
	pmL2Cache := newWorkerKPIRouteL2(w)
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
	// KPI/时序库物理分离：pm_files 已迁时序库（与 pm_metrics 同库保 copy_ingest 原子性），文件存储走 TsPool。
	pmFileStore := pm.NewPgPMFileStore(w.TsPool)

	// issue #836：原始文件压缩回写器（PM/MR 共用一个实例，单点注册 metric）。新文件入库成功后
	// 对明文 XML 尝试一次 gzip 回写；真机已是 .xml.gz 的对象零成本跳过。开关走 sys_configs
	// raw_archive.compress_after_ingest（默认 true，TTL 缓存）。异步有界并发，不阻塞入库 ack。
	rawArchiveSysCfg := admin.NewPgSysConfigRepository(w.PgPool)
	// 可取消的 base ctx：进程优雅关停时取消，停掉在途压缩 goroutine（对标 backpressure
	// watchdog / tsdb-shadow-sync 的 GS.Register 模式）。
	archiverCtx, archiverCancel := context.WithCancel(context.Background())
	w.GS.Register("raw-archiver", 3, func(ctx context.Context) error {
		_ = ctx
		archiverCancel()
		return nil
	})
	rawArchiver := rawarchive.New(
		archiverCtx,
		rawarchive.NewMinIOStore(w.MinIO),
		func(ctx context.Context, category, key string) (string, bool) {
			row, err := rawArchiveSysCfg.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		},
		rawarchive.NewMetrics(w.MetricsReg),
		logger.Named("raw-archive"),
		4,
	)
	rawArchiver.SetStorageAdmission(w.StorageProtection, cfg.MinIO.Buckets.PMFiles, cfg.MinIO.Buckets.MRFiles)

	pmCollector := collector.NewPMCollector(w.MinIO, cfg.MinIO.Buckets.PMFiles, pmParser, kpiEngine, pmFileStore, w.EventBus, logger)
	pmMetrics := pm.NewPMMetrics(w.MetricsReg)
	pmCollector.SetMetrics(pmMetrics)
	pmSlotCtx, pmSlotCancel := context.WithCancel(context.Background())
	pmSlotObserver := slothealth.NewObserver(
		slothealth.NewRepository(w.PgPool, w.TsPool),
		pmMetrics,
		time.Now(),
		pmstream.ConfigFromEnv().CloseGrace,
		time.Minute,
		logger.Named("pm-slot-health"),
	)
	go pmSlotObserver.Run(pmSlotCtx)
	w.GS.Register("pm-slot-health-observer", 2, func(context.Context) error {
		pmSlotCancel()
		return nil
	})
	// T-0164 G1 真机闭环：acs.upload.Handler 发的瘦 payload 只带 device_sn，
	// 由 collector 用同一个 deviceRepo 反查补齐 UUID / OUI / carrier / technology。
	pmCollector.SetDeviceLookup(pmDeviceRepo)
	// 已完成入库的同名文件重投时，在读取 MinIO 前按文件 marker 幂等短路。
	// 原始 .xml 可能已被归档器改名为 .xml.gz，不能依赖旧对象仍然存在。
	pmCollector.SetFileMarkerLookup(pmFileStore)

	// T-0164 G1 BUG-6 真根因复盘 / 方案 D：collector 用指标库白名单过滤孤儿 counter。
	// 复用 pmKPIRouter 的 LookupByDevice：route.Counters[].ReportKey 即设备所属产品在
	// perf_indicators_{enb,gnb,gsm} 中注册的 counter 上报名全集（PM-P2 按 report_key 建键，
	// 命中后把上报名改写成编号 IndicatorID 落库，并回填 Unit/StatisType 供结果值规范化）。
	// filterByWhitelist 本阶段 fail-open（lookup 失败 / 空集合时跳过过滤，避免误删）；#866
	// normalizeResults 会在写入前要求 Unit/StatisType 齐全，缺失时失败并暴露。
	pmCollector.SetCounterWhitelist(&routerCounterWhitelist{r: pmKPIRouter, log: logger})
	// 全局 report_key 目录与产品路由白名单分离：前者用于识别“指标库已知但当前
	// 产品未绑定”，避免把这类配置状态误报为厂家新上报名。五分钟缓存避免按文件查库。
	pmCollector.SetKnownReportKeyLookup(newKnownReportKeyLookup(pmIndicatorRepo))
	pmCollector.SetEnabledIndicatorLookup(newEnabledIndicatorLookup(
		indicator.NewPgEnabledRepository(w.PgPool), w.Redis,
	))
	pmCollector.SetQuarantineStore(collector.NewPgQuarantineStore(w.TsPool))
	pmResultNormSysCfg := admin.NewPgSysConfigRepository(w.PgPool)
	pmCollector.SetNumberProcessLookup(func(ctx context.Context) (string, error) {
		row, err := pmResultNormSysCfg.GetByKey(ctx, resultnorm.ConfigCategory, resultnorm.ConfigKey)
		if err != nil {
			if errors.Is(err, commonerrors.ErrNotFound) {
				return "", nil
			}
			return "", err
		}
		return row.Value, nil
	})

	// Runner wires retry + DLQ instrumentation around the PM handler.
	// dlqRepo + runnerMetrics are scoped to the worker process; admin handler
	// in app process reads the same dead_letters table directly.
	dlqRepo := dlq.NewPgRepository(w.PgPool)
	runnerMetrics := runner.NewMetrics(w.MetricsReg)
	pmRunner := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, w.EventBus, runnerMetrics, logger)
	pmCollector.SetRunner(pmRunner)

	// PM 入库进程内并发：NATS push 订阅 async 回调单 goroutine 串行（单订阅只用 ~1 核）。
	// 配 N 个订阅共享 durable consumer 吃满 worker 多核。<=0 回退 GOMAXPROCS（容器 CPU 配额）。
	// 上限 32（2026-07-21 压测实测：20000 设备规模下旧上限 16 把并发顶死，worker CPU 却只用了
	// ~30%（远未跑满）；提到 32 后隔离测量消费吞吐从约10/s提升到约20-27/s，稳定验证有效。
	// tsdb 连接池按入库并发 + 小时结算并发 + 后台余量核定，避免两个波峰重叠时池耗尽。
	pmConcurrency := effectivePMConsumerConcurrency(cfg.PMConsumerConcurrency)
	pmCollector.SetConcurrency(pmConcurrency)
	// 服务端 durable consumer 的 MaxAckPending 必须与实际处理能力绑定。默认 1000 会在
	// 机械盘过载时把大量消息同时推到 worker，形成重投和内存/IO 放大。
	if setter, ok := w.EventBus.(interface {
		SetQueueTuning(string, event.QueueTuning)
	}); ok {
		tuning := pmQueueTuning(pmConcurrency)
		setter.SetQueueTuning(event.SubjectPMFileReceived, tuning)
		setter.SetQueueTuning(event.SubjectPMFileDeferred, tuning)
	}

	// PM 指标大批量写异步提交（synchronous_commit=off）：PM 数据可从 MinIO 重建，换写吞吐。
	// 注意：仅作用于 metrics.batchInsertCopy 等旁路；copy-direct 主路径 CopyIngest 刻意忽略它以保证
	// pm_files 幂等锚点 durable 落盘（见 copy_ingest.go）。
	pmmetrics.BulkAsyncCommit = cfg.PMAsyncCommit

	// PM 入库统一走 copy-direct 写模式：单事务 plain COPY 原子入库（pm_files 标记 + counter + 内存
	// 算出的 KPI），幂等下沉到每文件一次 pm_files 唯一约束 + 文件内 last-wins 去重。migration 000042
	// 删 uq_pm_metrics_natural 后无索引可供 ON CONFLICT，upsert 写模式已退役，copy 是唯一写路径。
	// KPI/时序库物理分离：pm_metrics + pm_files 同在时序库（TsPool），单事务 copy 原子入库。
	pmCollector.SetCopyIngestor(pmmetrics.NewPgRepository(w.TsPool))

	// issue #836：入库成功后对原始 PM XML 尝试一次压缩回写省盘。
	pmCollector.SetArchiver(rawArchiver)

	// copy 是唯一写路径，KPI 恒用当前文件内存 counter 计算（CalculateFromCounters）；
	// pm_kpi_window_from_db 的 DB 回读分支已随非 copy 旁路退役。仍配 true 时明确告警，避免运营误以为生效。
	if cfg.PMKPIWindowFromDB {
		logger.Warn("pm_kpi_window_from_db=true is ignored: copy is the sole PM write mode; KPIs are computed from the current file's in-memory counters")
	}

	if err := pmCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe PM collector", zap.Error(err))
	}
	logger.Info("PM collector started with retry+DLQ runner",
		zap.Int("pm_consumer_concurrency", pmConcurrency),
		zap.Bool("pm_async_commit", cfg.PMAsyncCommit),
		zap.String("pm_write_mode", "copy"))

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

	geofenceAlarmMonitor := alarm.NewGeofenceAlarmMonitor(alarmEngine, logger)
	if err := geofenceAlarmMonitor.Subscribe(w.EventBus); err != nil {
		return fmt.Errorf("subscribe geofence alarm monitor: %w", err)
	}
	logger.Info("geofence alarm monitor started")

	geofenceControlMonitor := geofence.NewGeofenceControlMonitor(
		device.NewPgDeviceRepository(w.PgPool),
		w.Carriers,
		w.TaskService,
		logger.Named("geofence-control"),
	)
	geofenceControlMonitor.SetParameterReader(
		device.NewPgDeviceParameterRepository(w.PgPool),
	)
	geofenceControlMonitor.SetMappingReader(
		newWorkerGeofenceMappingReader(pmProductRegistry, workerParamRegistry),
	)
	geofenceControlMonitor.SetTaskHistoryReader(w.TaskRepo)
	geofenceControlMonitor.SetActionRepository(
		geofence.NewPgControlActionRepository(w.PgPool),
	)
	if err := geofenceControlMonitor.Subscribe(w.EventBus); err != nil {
		return fmt.Errorf("subscribe geofence control monitor: %w", err)
	} else {
		verificationCtx, cancelVerification := context.WithCancel(context.Background())
		w.GS.Register("geofence-control-verifier", 1, func(context.Context) error {
			cancelVerification()
			return nil
		})
		go geofenceControlMonitor.RunVerificationLoop(verificationCtx, 5*time.Second)
		logger.Info("geofence control monitor started")
	}

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
		alarmSyncProcessor = alarmSyncProcessor.WithAlarmDefRegistry(alarmDefRegistry)
		// T-0164-P1：复用 KPI Router 已构造的 pmProductRegistry，避免
		// 重复 RegistryMetrics MustRegister 触发 Prometheus duplicate collector panic。
		var productResolver definition.ProductResolver
		alarmReceiver, expeditedReceiver, productResolver = wireUnknownAlarmFallback(alarmReceiver, expeditedReceiver, alarmDefRegistry, pmProductRegistry)
		alarmSyncProcessor = alarmSyncProcessor.WithProductResolver(productResolver)
		logger.Info("alarm-definition fallback enabled",
			zap.Int("definitions_loaded", alarmDefRegistry.Count()))
	}
	scheduleAlarmDefRegistryStartupCatchUp(logger, alarmDefRegistry)
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

	// 设备离线超时告警清理：离线满阈值仍未恢复时，把当前告警归档到历史告警。
	// #358：阈值/周期/批量从 config.offline_alarm_cleanup 注入（<=0 回退 Default* 常量），
	// 与设备离线检测阈值（offline_threshold.go，#203）对齐口径，运营商可把收敛阈值调到分钟级。
	offlineAlarmCleaner := alarm.NewOfflineAlarmCleaner(alarmDeviceRepo, alarmPgStore, alarmEngine, logger)
	if s := cfg.OfflineAlarmCleanup.ThresholdSeconds; s > 0 {
		offlineAlarmCleaner.SetThreshold(time.Duration(s) * time.Second)
	}
	if s := cfg.OfflineAlarmCleanup.IntervalSeconds; s > 0 {
		offlineAlarmCleaner.SetInterval(time.Duration(s) * time.Second)
	}
	if n := cfg.OfflineAlarmCleanup.BatchSize; n > 0 {
		offlineAlarmCleaner.SetBatchSize(n)
	}
	go offlineAlarmCleaner.Run(context.Background())
	logger.Info("offline alarm cleaner started",
		zap.Duration("interval", offlineAlarmCleaner.Interval()),
		zap.Duration("threshold", offlineAlarmCleaner.Threshold()),
		zap.Int("batch_size", offlineAlarmCleaner.BatchSize()))

	// Reboot Task Closer (F01/F06)：M Reboot Inform 兜底收敛未 ACK 的 Reboot/FactoryReset 任务。
	rebootCloser := task.NewRebootCloser(w.TaskRepo, w.TaskService, logger)
	if err := rebootCloser.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe reboot task closer", zap.Error(err))
	}
	logger.Info("reboot task closer started")

	// PROVISION drain consumer（F09 自动开站暂不支持）：provision.Engine 仍会发布
	// provision.started/completed/failed/step.done，但 PROVISION 是 WorkQueue 流——没有
	// 任何 consumer ack 时消息会一直堆到 MaxAge（72h）才过期，期间在 nats jsz 里表现为
	// 持续增长的积压（实测 2 万+条无人消费）。在功能正式落地前，用一个 no-op drain
	// consumer 把 provision.* 全部 ack 掉，保持 workqueue 清空、监控干净。
	// 用单个 `provision.>` 通配 consumer 覆盖整条流（WorkQueue 流不允许同一 filter subject
	// 挂多个 consumer；通配即流本身的 subject，最稳）。功能上线时删掉此处、换成真正的订阅者即可。
	if _, err := w.EventBus.QueueSubscribe("provision.>", "provision-drain",
		func(_ context.Context, evt event.Event) error {
			logger.Debug("provision event drained (auto-provisioning not yet supported)",
				zap.String("subject", evt.Subject), zap.String("event_id", evt.ID))
			return nil
		}); err != nil {
		logger.Warn("subscribe provision drain consumer", zap.Error(err))
	} else {
		logger.Info("provision drain consumer started (no-op ack; auto-provisioning not yet supported)")
	}

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

	// #13: 任务状态对账器 — 周期对账 Redis（运行时真相源）与 PG（持久化权威源）的分叉。
	// 修复 "PG 滞后于 Redis 终态" 的孤儿/陈旧记录（双写 sync 失败被吞 → PG 永远 pending/sent，
	// Redis TTL 过期后成永久孤儿）。TaskMetrics 已在 initWorker 构造并挂到 TaskService
	// （issue #20：早于 RestorePendingQueues 才能记到启动期 recovery 动作），reconciler
	// 复用同一 metric 实例，避免重复 MustRegister 触发 Prometheus duplicate collector panic。
	if cfg.Task.ReconcileIntervalSeconds > 0 {
		reconciler := task.NewReconciler(
			w.TaskRepo, w.TaskQueue, w.TaskRepo, w.TaskMetrics,
			time.Duration(cfg.Task.ReconcileIntervalSeconds)*time.Second,
			0, // grace 默认 60s
			0, // batchSize 默认 100
			logger,
		).WithTransitionPublisher(w.TaskService.PublishTransitionEvent)
		go reconciler.Run(context.Background())
		logger.Info("task reconciler started (#13)",
			zap.Int("interval_seconds", cfg.Task.ReconcileIntervalSeconds))
	} else {
		logger.Info("task reconciler disabled (reconcile_interval_seconds <= 0)")
	}

	// MR Collector（worker 端保留 — 文件 I/O 类，与 PM collector 同进程更合理）
	mrStore := mr.NewPgMRStore(w.TsPool) // mr_files 已迁时序库，单 TsPool
	mrCollector := mrcollector.NewMRCollector(w.MinIO, cfg.MinIO.Buckets.MRFiles, mrStore, w.EventBus, logger)
	// 注入 DeviceLookup：upload handler 直传路径的 mr.file.received payload 不带
	// device_id，由 collector 按 device_sn 反查（参 internal/acs/upload publishMRFileReceivedEvent）。
	mrCollector.SetDeviceLookup(device.NewPgDeviceRepository(w.PgPool))
	// 注入 carrier MR-type 支持判定（#17）：MRE parser 的 "运营商是否采集 MRE" 决策
	// 走 Carrier 适配器，去除 mr/parser 里的 "if carrier == cucc" 硬编码。
	mrCollector.SetMRTypeSupport(w.Carriers)
	// issue #836：入库成功后对原始 MR XML 尝试一次压缩回写省盘（与 PM 共用同一 archiver）。
	mrCollector.SetArchiver(rawArchiver)
	if err := mrCollector.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe MR collector", zap.Error(err))
	}
	logger.Info("MR collector started")

	// issue #836：PM/MR 原始文件压缩只在本次新文件入库成功后尝试一次。
	// 不再启动 rawarchive Sweeper 扫描历史 raw_compressed=false 文件；漏压/失败可接受，
	// 避免开关关闭后仍有后台补压或启用时产生历史 IO 峰值。
	logger.Info("raw-archive sweeper disabled; post-ingest compression is one-shot")

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
	transferBridge.SetStorageAdmission(w.StorageProtection)
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
	// 全新部署的 seed 只落协议开关和 HTTPS 字段；HTTP 地址沿用启动配置作为兜底，避免
	// 未填写页面时 PM 首次自动下发拿不到默认 upload base。
	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	transferPolicy := transfercfg.NewPolicy(
		newWorkerTransferDefaults(cfg.PM),
		newTransferSysConfigLookup(sysConfigRepo),
	)
	// SYS is a WorkQueue stream and ACS owns the existing sys.config.saved
	// consumer. Worker policies therefore use the bounded Policy TTL instead of
	// registering a competing filtered consumer.
	backupExecutor.SetTransferProvider(transferPolicy)
	transferParamRepo := device.NewPgDeviceParameterRepository(w.PgPool)
	backupExecutor.SetUploadAddressResolver(newTransferAddressResolver(
		transferPolicy,
		transferParamRepo,
	))
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
	// KPI/时序库物理分离：trace_messages 已迁时序库（TsPool），retention 超表。
	traceRepo := trace.NewPgRepository(w.TsPool)
	var traceBulk *trace.BulkStore
	if w.MinIO != nil && cfg.MinIO.Buckets.TraceBulk != "" {
		traceBulk = trace.NewBulkStore(w.MinIO, cfg.MinIO.Buckets.TraceBulk)
		traceBulk.SetStorageAdmission(w.StorageProtection)
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
		traceExporter.SetStorageAdmission(w.StorageProtection)
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
	reportGenerator.SetStorageAdmission(w.StorageProtection)
	if err := reportGenerator.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe report generator", zap.Error(err))
	}
	logger.Info("report generator started")

	// M2: TransferCompleteRouter — 订阅 device.inform.transfer_complete，按 CommandKey 回写 backup/restore_tasks 终态
	restoreRepo := backup.NewPgRestoreTaskRepository(w.PgPool)
	tcRouter := backup.NewTransferCompleteRouter(backupTaskRepo, restoreRepo, nil, logger)
	// #70：恢复 Download 成功后走主动完整性校验编排（downloaded → 回读校验 →
	// completed/failed）。verifier 是 device-dependent hook，当前未接 ACS GPV 回读，
	// 故传 nil —— 安全默认（DefaultVerificationConfig）下恢复停在 downloaded 中间态、
	// 不谎报 completed。接入设备回读后只需注入 RestoreVerifier 即可激活完整闭环。
	restoreVerifyOrch := backup.NewRestoreVerificationOrchestrator(
		restoreRepo, nil /*verifier: device-dependent, not yet wired*/, backup.DefaultVerificationConfig(), nil, logger,
	)
	tcRouter.SetRestoreVerificationOrchestrator(restoreVerifyOrch)
	if err := tcRouter.Subscribe(w.EventBus); err != nil {
		logger.Warn("subscribe backup transfer-complete router", zap.Error(err))
	}
	logger.Info("backup transfer-complete router started (active restore verification: downloaded→verify; no device verifier wired → stays downloaded)")

	pmOnlineSub := pm.NewOnlineSubscriber(
		w.TaskService,
		cfg.PM.UploadURLTemplate,
		cfg.PM.EnableValue,
		cfg.PM.PeriodicUploadInterval,
		logger,
	)
	pmOnlineSub.SetAdmissionGate(pm.NewRedisPMSetupAdmissionGate(w.Redis, 0))
	pmParameterRepo := transferParamRepo
	pmOnlineSub.SetUploadAddressResolver(newTransferAddressResolver(
		transferPolicy,
		pmParameterRepo,
	))
	pmOnlineSub.SetParamSyncPMCompensationReaders(pmDeviceRepo, pmParameterRepo)

	// PM 设备上线自动下发 PM 上传配置（KPI 上报参数整理.md 三参数）。
	// device.online/device.registered 仅在 cfg.PM.AutoSetupOnOnline=true 时启用；
	// 参数同步后的 HTTPS 补偿不受该开关影响，避免首次未知能力下发 HTTP 后无法收敛。
	if cfg.PM.AutoSetupOnOnline {
		// 启动期一次性校验 PM 上传 URL 模板的 host：渲染后 host 为空（如生产 .env
		// 漏配 OMC_PUBLIC_HOST，模板渲染成 "http://:7557/..."）则醒目 Error 告警，
		// 把运维漏配从「设备上线时才发现」前移到「部署即可见」。不 fail-fast：
		// worker 还跑 PM 解析/聚合等关键流程，单个配置项不应阻断整个 worker。
		// 真正下发优先使用统一 transfercfg.AddressResolver 决策的 base URL。
		if rendered, verr := pm.ValidateUploadURLTemplate(cfg.PM.UploadURLTemplate); verr != nil {
			logger.Error("PM upload URL template invalid; unified transfer base URL will be used when available (check OMC_PUBLIC_HOST fallback)",
				zap.String("url_template", cfg.PM.UploadURLTemplate),
				zap.String("rendered", rendered),
				zap.Error(verr))
		}
		if err := pmOnlineSub.Subscribe(w.EventBus); err != nil {
			logger.Warn("subscribe pm online subscriber failed", zap.Error(err))
		} else {
			logger.Info("pm online subscriber started (auto SPV on device.registered/online)")
		}
	} else {
		if err := pmOnlineSub.SubscribeParamSyncCompleted(w.EventBus); err != nil {
			logger.Warn("subscribe pm param sync HTTPS compensation failed", zap.Error(err))
		} else {
			logger.Info("pm online subscriber disabled (cfg.pm.auto_setup_on_online=false); param sync HTTPS compensation enabled")
		}
	}

	// T-0164-P5 / G5 + T-0164-P8 / G8：PM 自然桶聚合 + asyncjob 框架接入。
	// 复用上文已构造的 pmKPIRouter（KPI 反算依赖路由）；注册 4 个设备级 runner +
	// sweeper + hourly 触发器（hourly @:05）。daily/weekly/monthly 由上游 bucket 成功后 chain。
	// T-0192：日/周/月桶按业务时区切本地零点。
	// #458：业务时区改读 #456 统一源 sys_configs（category='basic'/key='timezoneCode'），
	// 不再读 YAML PM.Timezone（空/非法 Provider 内回落 UTC）。tzManager 统一管理
	// 窗口计算与 cron 调度的时区；后台轮询 sys_configs（默认 30s）感知管理员改时区，
	// 变更即重排调度、无需重启 worker。不走 sys.config.saved 事件：该 subject 属 NATS SYS
	// WorkQueue 流，其唯一 consumer 已被 ACS transfercfg 占用，worker 不能在同 filter
	// subject 再开第二个 consumer（必报 "filtered consumer not unique on workqueue stream"）。
	pmTz := newTzManager(context.Background(), w.PgPool, logger)
	// KPI 导出文件落地桶：复用报表桶（设计 §5.6）；缺省回退 "reports"。
	exportBucket := cfg.MinIO.Buckets.Reports
	if exportBucket == "" {
		exportBucket = "reports"
	}
	// 可取消的流水线 ctx：进程优雅关停时取消整棵 PM 聚合/各 retention cron/adhoc + 日志轮转
	// watcher 的 goroutine 树（对标上面 raw-archiver 的 GS.Register 模式）。否则这些只 gate 在
	// <-ctx.Done() 的 goroutine 永不退出，关停后仍访问已关闭的 PG/Redis 池产生 "pool closed" 噪声。
	// 优先级 1：早于一切资源关闭（nats/eventbus 2 / redis 3 / postgres·timescale 4），
	// 让取消信号尽早发出，goroutine 在池关闭前就开始退出循环，最大限度减少关停期噪声。
	pipeCtx, pipeCancel := context.WithCancel(context.Background())
	w.GS.Register("pm-pipeline", 1, func(context.Context) error {
		pipeCancel()
		return nil
	})
	// #458：cron 随 pipeCtx 取消统一停；后台轮询 sys_configs 感知改时区后即时重排（不重启）。
	pmTz.shutdownOnCtx(pipeCtx)
	pmTz.startReloadPoller(pipeCtx, defaultReloadPollInterval)
	startPMAggregatorPipeline(pipeCtx, w, pmKPIRouter, pmTz, exportBucket)
	startPMAggregationStream(pipeCtx, w, pmTz)
	startRawObjectCleanup(pipeCtx, w, cfg)

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

	// parammodel XML 备份清理 cron(每天默认 03:00)。三库 XML 导入重构后扫单目录
	// param-mappings/ 下 .deleted.<ts>/.bak.<ts>(> retentionDays 清) + .tmp.<uuid>(> 1h 清)
	// + 孤儿 sidecar(X.xml.custom 而 X.xml 已不在)。
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
	// #779: 回收站自动移入 cron（每天 00:10，与 UI 配置说明对齐）。
	// 读取 sys_configs device:deviceOfflineEnable / deviceOfflineSaveDay，
	// 满足离线天数阈值的设备批量软删除（deleted_by='system'，executor='system:auto_recycle'）。
	startAutoRecycleCron(w, logger)

	// KPI/时序库物理分离：worker 把主库维度表周期刷入时序库影子维度表，
	// 供 PM/告警时序查询本库 JOIN（device_dim / cell_band_dim / product_dim 等），
	// 替代跨库 JOIN。仅 worker 跑同步，app 只读影子表。默认 60s 周期。
	// 用可取消 ctx + GS 钩子接入优雅关机（ctx.Done 时 Run 退出循环）。
	if w.TsPool != nil {
		syncCtx, syncCancel := context.WithCancel(context.Background())
		syncRunner := tsdbsync.NewSyncRunner(w.PgPool, w.TsPool, tsdbsync.DefaultInterval, logger)
		go syncRunner.Run(syncCtx)
		w.GS.Register("tsdb-shadow-dim-sync", 5, func(ctx context.Context) error {
			syncCancel()
			return nil
		})
		logger.Info("tsdb shadow-dim sync started",
			zap.Duration("interval", tsdbsync.DefaultInterval))
	} else {
		logger.Warn("tsdb shadow-dim sync disabled: TsPool not connected")
	}
	return nil
}

func newWorkerKPIRouteL2(w *workerInfra) router.L2Cache {
	if w == nil || w.PMRedis == nil {
		return nil
	}
	return router.NewRedisCache(w.PMRedis)
}

func pmQueueTuning(concurrency int) event.QueueTuning {
	maxAckPending := concurrency * 4
	if maxAckPending < 16 {
		maxAckPending = 16
	}
	return event.QueueTuning{
		AckWait:       2 * time.Minute,
		MaxDeliver:    event.MaxDeliveriesForRetryHorizon(collector.DeviceRegistrationGrace),
		MaxAckPending: maxAckPending,
	}
}

const pmTSDBConnectionReserve = 8
const pmAggregationTSDBWriteConsumers = 3
const pmMaxAggregationConcurrency = 256
const pmMaxTSDBConnectionBudget = int64(1<<31 - 1)

func effectivePMConsumerConcurrency(configured int) int {
	if configured <= 0 {
		configured = runtime.GOMAXPROCS(0)
	}
	if configured > 32 {
		return 32
	}
	return configured
}

// pmTSDBConnectionBudget reserves independent capacity for synchronized PM
// ingestion and hourly finalization. Both paths write TimescaleDB and overlap
// after the 12-minute close grace; sizing only for either path starves the
// other and turns a normal device burst into upload 503s.
func pmTSDBConnectionBudget(
	ingestConcurrency,
	finalizeConcurrency,
	aggregationConsumerConcurrency int,
) int32 {
	if ingestConcurrency < 0 {
		ingestConcurrency = 0
	}
	if finalizeConcurrency < 0 {
		finalizeConcurrency = 0
	}
	if aggregationConsumerConcurrency < 0 {
		aggregationConsumerConcurrency = 0
	}
	for _, concurrency := range []int{
		ingestConcurrency,
		finalizeConcurrency,
		aggregationConsumerConcurrency,
	} {
		if int64(concurrency) > pmMaxTSDBConnectionBudget {
			return int32(pmMaxTSDBConnectionBudget)
		}
	}
	total := int64(ingestConcurrency) +
		int64(finalizeConcurrency) +
		int64(pmAggregationTSDBWriteConsumers)*int64(aggregationConsumerConcurrency) +
		int64(pmTSDBConnectionReserve)
	if total > pmMaxTSDBConnectionBudget {
		return int32(pmMaxTSDBConnectionBudget)
	}
	return int32(total)
}

func startGeofenceCoordinator(
	db storage.DB,
	bus event.EventBus,
) (func() error, error) {
	repository := geofence.NewPgCoordinatorRepository(db)
	coordinator := geofence.NewCoordinator(repository)
	if err := coordinator.Start(bus); err != nil {
		return nil, err
	}
	return coordinator.Stop, nil
}

func startEventOutboxRelay(
	parent context.Context,
	repo coreoutbox.DeliveryRepository,
	publisher coreoutbox.Publisher,
	logger *zap.Logger,
) (func(), error) {
	relay, err := coreoutbox.NewRelay(repo, publisher, coreoutbox.RelayConfig{}, logger)
	if err != nil {
		return nil, err
	}
	relayCtx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := relay.Run(relayCtx); err != nil {
			logger.Error("generic event outbox relay stopped", zap.Error(err))
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			cancel()
			<-done
		})
	}, nil
}

// startAutoRecycleCron 启动 #779 回收站自动移入 cron（每天 00:10）。
//
// 行为：
//   - 注册 cron（固定 "10 0 * * *"，与 UI 标注「每天 00:10 检查设备离线时间」对齐）
//   - 启动期延迟 30s 跑一次 catch-up：防 worker 长期宕机后积压的离线设备未被回收
//   - 读取 sys_configs device:deviceOfflineEnable（总开关）+ deviceOfflineSaveDay（天数阈值）
//   - 开关 false 时直接跳过，不软删任何设备
//   - 单实例假设（单 worker 部署），无锁保护；横扩需加 PG advisory lock
func startAutoRecycleCron(w *workerInfra, logger *zap.Logger) {
	sysConfigRepo := admin.NewPgSysConfigRepository(w.PgPool)
	lookup := device.SysConfigLookupFn(func(ctx context.Context, category, key string) (string, bool) {
		row, err := sysConfigRepo.GetByKey(ctx, category, key)
		if err != nil || row == nil {
			return "", false
		}
		return row.Value, true
	})
	deviceOps := device.NewPgDeviceRepository(w.PgPool)
	job := device.NewAutoRecycleJob(lookup, deviceOps, 0, logger.Named("auto-recycle"))

	// recycleCtx 绑定 worker 生命周期：GS 触发时取消，catch-up goroutine 可感知 shutdown。
	// 对标 archiverCtx / pipeCtx 的 GS.Register 模式。
	recycleCtx, recycleCancel := context.WithCancel(context.Background())
	w.GS.Register("auto-recycle", 4, func(context.Context) error {
		recycleCancel()
		return nil
	})

	c := cron.New()
	if _, err := c.AddFunc(device.DefaultAutoRecycleCron, func() {
		ctx, cancel := context.WithTimeout(recycleCtx, 10*time.Minute)
		defer cancel()
		if _, runErr := job.Run(ctx); runErr != nil {
			logger.Warn("auto recycle run failed", zap.Error(runErr))
		}
	}); err != nil {
		logger.Warn("invalid auto recycle cron; skipping",
			zap.String("cron", device.DefaultAutoRecycleCron), zap.Error(err))
		recycleCancel()
		return
	}
	c.Start()
	logger.Info("auto recycle cron started",
		zap.String("cron", device.DefaultAutoRecycleCron))

	// 启动期延迟 catch-up（给 app + 字典加载 30s 缓冲）。
	// 用 select 替代裸 time.Sleep，SIGTERM 时可立即退出，不污染 graceful shutdown 日志。
	go func() {
		select {
		case <-time.After(30 * time.Second):
		case <-recycleCtx.Done():
			return
		}
		ctx, cancel := context.WithTimeout(recycleCtx, 10*time.Minute)
		defer cancel()
		deleted, err := job.Run(ctx)
		if err != nil {
			logger.Warn("auto recycle startup catch-up failed", zap.Error(err))
			return
		}
		if deleted > 0 {
			logger.Info("auto recycle startup catch-up",
				zap.Int64("deleted", deleted))
		}
	}()
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
	// 三库 XML 导入重构:单目录(param-mappings/),builtin + custom XML 同住,
	// 备份 / 孤儿 sidecar 也都落在此目录。
	dir := filepath.Join(cfg.DictLoader.XMLBaseDir, parammodel.BuiltinDirSubdir)
	cronExpr := pmCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = parammodel.DefaultBackupCleanupCron
	}

	metrics := parammodel.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := parammodel.NewBackupCleanup(dir, pmCfg.BackupRetentionDays, metrics, logger)

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
		zap.String("dir", dir),
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
//   - 备份扫描根 = indicator-library 单目录(BuiltinDirSubdir;单目录 + sidecar 后
//     builtin/custom 同住),Run() 遍历 enb/gsm/gnb 各自的子目录
func startIndicatorBackupCleanup(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	// 解析配置 + 应用默认值
	indCfg := cfg.DictLoader.Indicator
	baseSub := indCfg.BaseDirectory
	if baseSub == "" {
		baseSub = indicator.BuiltinDirSubdir
	}
	baseDir := filepath.Join(cfg.DictLoader.XMLBaseDir, baseSub)
	cronExpr := indCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = indicator.DefaultIndicatorBackupCleanupCron
	}

	metrics := indicator.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := indicator.NewBackupCleanup(baseDir, indCfg.BackupRetentionDays, metrics, logger)

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
		zap.String("base_dir", baseDir),
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

// startAlarmBackupCleanup 启动告警库自定义 XML 备份清理 cron(三库 XML 导入重构单目录)。
//
//   - 注册 cron(默认 "0 3 * * *");无效表达式 fallback 默认值
//   - 启动期延迟 30s 跑一次 catch-up:防 worker 长期宕机后备份堆积
//   - dir 是单目录 (.../alarm-definitions),Run() 扫单层备份 + 孤儿 sidecar
//   - 单实例假设;横扩需加 PG advisory lock(与 indicator 同,P1 不做)
func startAlarmBackupCleanup(w *workerInfra, cfg *appconfig.WorkerConfig, logger *zap.Logger) {
	alarmCfg := cfg.DictLoader.AlarmDefinition
	sub := alarmCfg.Directory
	if sub == "" {
		sub = definition.BuiltinDirSubdir
	}
	dir := filepath.Join(cfg.DictLoader.XMLBaseDir, sub)
	cronExpr := alarmCfg.BackupCleanupCron
	if cronExpr == "" {
		cronExpr = definition.DefaultAlarmBackupCleanupCron
	}

	metrics := definition.NewBackupCleanupMetrics(w.MetricsReg)
	cleanup := definition.NewBackupCleanup(dir, alarmCfg.BackupRetentionDays, metrics, logger)

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
		zap.String("dir", dir),
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

func scheduleAlarmDefRegistryStartupCatchUp(logger *zap.Logger, alarmDefRegistry *definition.Registry) {
	if logger == nil || alarmDefRegistry == nil {
		return
	}
	go func() {
		time.Sleep(30 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := alarmDefRegistry.Refresh(ctx); err != nil {
			logger.Warn("alarm-definition registry startup catch-up failed", zap.Error(err))
			return
		}
		logger.Info("alarm-definition registry startup catch-up",
			zap.Int("definitions_loaded", alarmDefRegistry.Count()))
	}()
}

// parseStringSlice parses a comma-separated string into a slice.
// routerCounterWhitelist 把 *router.Router 包装成 collector.CounterWhitelist 接口。
// 通过 LookupByDevice 拿设备所属产品的 KPIRoute.Counters，转
// report_key→{编号, statis_type, unit} 映射作为白名单。Lookup 失败时把错误透传给
// collector，由 collector 在过滤阶段 fail-open（log warn + 不过滤），再由写入前规范化
// 对缺失元数据 fail-fast。
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
			ReportKey:   c.ReportKey,
			StatisType:  c.StatisType,
			Unit:        c.Unit,
		}
	}
	return out, nil
}

const defaultEnabledIndicatorCacheTTL = 5 * time.Minute

type enabledIndicatorRepository interface {
	ListAll(ctx context.Context, dt indicator.DeviceType) ([]string, error)
}

type enabledIndicatorCacheEntry struct {
	indicators map[string]struct{}
	expiresAt  time.Time
}

type enabledIndicatorLookup struct {
	repo             enabledIndicatorRepository
	ttl              time.Duration
	now              func() time.Time
	readCacheVersion func(context.Context) (string, error)

	mu           sync.RWMutex
	cache        map[indicator.DeviceType]enabledIndicatorCacheEntry
	cacheVersion string
	loads        singleflight.Group
}

type knownReportKeyRepository interface {
	ListCounterReportKeys(ctx context.Context, dt indicator.DeviceType) ([]string, error)
}

type knownReportKeyCacheEntry struct {
	keys      map[string]struct{}
	expiresAt time.Time
}

type knownReportKeyLookup struct {
	repo knownReportKeyRepository
	ttl  time.Duration
	now  func() time.Time

	mu    sync.RWMutex
	cache map[indicator.DeviceType]knownReportKeyCacheEntry
	loads singleflight.Group
}

func newKnownReportKeyLookup(repo knownReportKeyRepository) *knownReportKeyLookup {
	return &knownReportKeyLookup{
		repo:  repo,
		ttl:   defaultEnabledIndicatorCacheTTL,
		now:   time.Now,
		cache: make(map[indicator.DeviceType]knownReportKeyCacheEntry),
	}
}

func (l *knownReportKeyLookup) LookupKnownReportKeys(ctx context.Context, technology string) (map[string]struct{}, error) {
	dt, err := indicatorDeviceTypeFromTechnology(technology)
	if err != nil {
		return nil, err
	}
	now := l.nowTime()
	if cached, ok := l.lookupCache(dt, now); ok {
		return cached, nil
	}

	value, err, _ := l.loads.Do(string(dt), func() (interface{}, error) {
		now := l.nowTime()
		if cached, ok := l.lookupCache(dt, now); ok {
			return cached, nil
		}
		if l.repo == nil {
			return nil, fmt.Errorf("known PM report-key repository is not configured")
		}
		keys, err := l.repo.ListCounterReportKeys(ctx, dt)
		if err != nil {
			return nil, fmt.Errorf("list known PM report keys (%s): %w", dt, err)
		}
		out := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			if key != "" {
				out[key] = struct{}{}
			}
		}
		for _, key := range indicator.KnownUnstoredReportKeys(dt) {
			if key != "" {
				out[key] = struct{}{}
			}
		}
		l.storeCache(dt, out, now.Add(l.ttlDuration()))
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	keys, ok := value.(map[string]struct{})
	if !ok {
		return nil, fmt.Errorf("known PM report-key cache returned unexpected type %T", value)
	}
	return keys, nil
}

func (l *knownReportKeyLookup) lookupCache(dt indicator.DeviceType, now time.Time) (map[string]struct{}, bool) {
	l.mu.RLock()
	entry, ok := l.cache[dt]
	l.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		return nil, false
	}
	return entry.keys, true
}

func (l *knownReportKeyLookup) storeCache(dt indicator.DeviceType, keys map[string]struct{}, expiresAt time.Time) {
	l.mu.Lock()
	if l.cache == nil {
		l.cache = make(map[indicator.DeviceType]knownReportKeyCacheEntry)
	}
	l.cache[dt] = knownReportKeyCacheEntry{
		keys:      keys,
		expiresAt: expiresAt,
	}
	l.mu.Unlock()
}

func (l *knownReportKeyLookup) ttlDuration() time.Duration {
	if l.ttl <= 0 {
		return defaultEnabledIndicatorCacheTTL
	}
	return l.ttl
}

func (l *knownReportKeyLookup) nowTime() time.Time {
	if l.now == nil {
		return time.Now()
	}
	return l.now()
}

func newEnabledIndicatorLookup(
	repo enabledIndicatorRepository,
	rdb redis.UniversalClient,
) *enabledIndicatorLookup {
	lookup := &enabledIndicatorLookup{
		repo:  repo,
		ttl:   defaultEnabledIndicatorCacheTTL,
		now:   time.Now,
		cache: make(map[indicator.DeviceType]enabledIndicatorCacheEntry),
	}
	if rdb != nil {
		lookup.readCacheVersion = func(ctx context.Context) (string, error) {
			version, err := rdb.Get(ctx, "indicator:cache_version").Result()
			if errors.Is(err, redis.Nil) {
				return "0", nil
			}
			if err != nil {
				return "", fmt.Errorf("read indicator cache version: %w", err)
			}
			return version, nil
		}
	}
	return lookup
}

func (l *enabledIndicatorLookup) LookupEnabledIndicators(ctx context.Context, technology string) (map[string]struct{}, error) {
	dt, err := indicatorDeviceTypeFromTechnology(technology)
	if err != nil {
		return nil, err
	}
	for {
		version, err := l.ensureCacheVersion(ctx)
		if err != nil {
			return nil, err
		}
		now := l.nowTime()
		if cached, ok := l.lookupCache(dt, now, version); ok {
			return cached, nil
		}

		value, err, _ := l.loads.Do(string(dt)+"\x1f"+version, func() (interface{}, error) {
			now := l.nowTime()
			if cached, ok := l.lookupCache(dt, now, version); ok {
				return cached, nil
			}
			if l.repo == nil {
				return nil, fmt.Errorf("enabled PM indicator repository is not configured")
			}
			ids, err := l.repo.ListAll(ctx, dt)
			if err != nil {
				return nil, fmt.Errorf("list enabled PM indicators (%s): %w", dt, err)
			}
			out := make(map[string]struct{})
			for _, id := range ids {
				if id != "" {
					out[id] = struct{}{}
				}
			}
			if !l.storeCache(dt, out, now.Add(l.ttlDuration()), version) {
				return nil, errEnabledIndicatorCacheVersionChanged
			}
			return cloneIndicatorSet(out), nil
		})
		if errors.Is(err, errEnabledIndicatorCacheVersionChanged) {
			continue
		}
		if err != nil {
			return nil, err
		}
		indicators, ok := value.(map[string]struct{})
		if !ok {
			return nil, fmt.Errorf("enabled PM indicator cache returned unexpected type %T", value)
		}
		return cloneIndicatorSet(indicators), nil
	}
}

var errEnabledIndicatorCacheVersionChanged = errors.New("enabled PM indicator cache version changed")

func (l *enabledIndicatorLookup) ensureCacheVersion(ctx context.Context) (string, error) {
	if l.readCacheVersion == nil {
		return "", nil
	}
	version, err := l.readCacheVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("synchronize enabled PM indicator cache: %w", err)
	}
	l.mu.Lock()
	if version != l.cacheVersion {
		l.cache = make(map[indicator.DeviceType]enabledIndicatorCacheEntry)
		l.cacheVersion = version
	}
	l.mu.Unlock()
	return version, nil
}

func (l *enabledIndicatorLookup) lookupCache(
	dt indicator.DeviceType,
	now time.Time,
	version string,
) (map[string]struct{}, bool) {
	l.mu.RLock()
	entry, ok := l.cache[dt]
	versionMatches := version == l.cacheVersion
	l.mu.RUnlock()
	if !versionMatches || !ok || now.After(entry.expiresAt) {
		return nil, false
	}
	return cloneIndicatorSet(entry.indicators), true
}

func (l *enabledIndicatorLookup) storeCache(
	dt indicator.DeviceType,
	indicators map[string]struct{},
	expiresAt time.Time,
	version string,
) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if version != l.cacheVersion {
		return false
	}
	if l.cache == nil {
		l.cache = make(map[indicator.DeviceType]enabledIndicatorCacheEntry)
	}
	l.cache[dt] = enabledIndicatorCacheEntry{
		indicators: cloneIndicatorSet(indicators),
		expiresAt:  expiresAt,
	}
	return true
}

func (l *enabledIndicatorLookup) ttlDuration() time.Duration {
	if l.ttl <= 0 {
		return defaultEnabledIndicatorCacheTTL
	}
	return l.ttl
}

func (l *enabledIndicatorLookup) nowTime() time.Time {
	if l.now == nil {
		return time.Now()
	}
	return l.now()
}

func cloneIndicatorSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for id := range in {
		out[id] = struct{}{}
	}
	return out
}

func indicatorDeviceTypeFromTechnology(technology string) (indicator.DeviceType, error) {
	switch model.NormalizeTechnology(technology) {
	case model.TechLTE:
		return indicator.DeviceTypeENB, nil
	case model.TechNR:
		return indicator.DeviceTypeGNB, nil
	case model.TechGSM:
		return indicator.DeviceTypeGSM, nil
	default:
		return "", fmt.Errorf("unsupported PM technology for indicator metadata: %q", technology)
	}
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
