package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/dashboard"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/events"
	"github.com/omcgo/omcgo/internal/filemanager"
	"github.com/omcgo/omcgo/internal/interop"
	"github.com/omcgo/omcgo/internal/interop/cases"
	"github.com/omcgo/omcgo/internal/license"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/nedirect"
	"github.com/omcgo/omcgo/internal/northbound"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/ops"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/omcgo/omcgo/internal/syslog"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
)

// initMRModule 初始化 F05 测量报告模块。
func initMRModule(c *Container) error {
	logger := c.Logger.Named("mr")

	mrStore := mr.NewPgMRStore(c.PgPool, c.TsPool)
	mrIndRepo := mr.NewPgIndicatorRepository(c.PgPool)
	mrMapRepo := mr.NewPgMappingRepository(c.PgPool)

	c.miscDeps.mrStore = mrStore
	c.miscDeps.mrIndRepo = mrIndRepo
	c.miscDeps.mrMapRepo = mrMapRepo

	logger.Info("MR module initialized")
	return nil
}

// initSoftwareModule 初始化 F06 固件管理模块。
func initSoftwareModule(c *Container) error {
	logger := c.Logger.Named("software")

	firmwareRepo := software.NewPgFirmwareRepository(c.PgPool)
	taskRepo := software.NewPgTaskRepository(c.PgPool)
	subTaskRepo := software.NewPgSubTaskRepository(c.PgPool)
	softwareService := software.NewSoftwareService(
		firmwareRepo, taskRepo, subTaskRepo, c.DeviceRepo, c.TaskSvc, c.ConnReqClient,
		c.MinIO, c.Cfg.MinIO.Buckets.Firmware, c.EventBus, c.Redis, logger,
	)
	if err := softwareService.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe software service", zap.Error(err))
	}
	softwareService.RestorePendingUpgrades(context.Background())
	softwareService.StartTaskReaper()

	// Canary monitor + metrics (T-0018 / R-101)
	canaryMetrics := software.NewCanaryMetrics(c.MetricsReg)
	softwareService.SetCanaryMetrics(canaryMetrics)
	// Rollback metrics (T-0021 / R-101): registered before canary monitor wiring
	// so the auto-rollback path emits counters on first fire.
	rollbackMetrics := software.NewRollbackMetrics(c.MetricsReg)
	softwareService.SetRollbackMetrics(rollbackMetrics)
	canaryMonitor := software.NewCanaryMonitor(taskRepo, canaryMetrics, logger)
	// Wire SoftwareService as the auto-rollback trigger. Default RollbackOnFailure
	// is false; the trigger only fires when a canary task explicitly opted in.
	canaryMonitor.SetRollbackTrigger(softwareService)
	if err := canaryMonitor.Start(context.Background()); err != nil {
		logger.Warn("start canary monitor", zap.Error(err))
	}
	c.miscDeps.canaryMonitor = canaryMonitor

	softwareHandler := software.NewHandler(softwareService, firmwareRepo, taskRepo, subTaskRepo, logger)

	c.miscDeps.softwareHandler = softwareHandler

	logger.Info("software management module initialized with canary monitor")
	return nil
}

// initProvisionModule 初始化 F09 自动开站模块。
// T-0098 P5-01：dmRegistry / dmImporter 已删除，改由 paramRegistry / productRegistry / intersectService 接管。
func initProvisionModule(c *Container) error {
	logger := c.Logger.Named("provision")

	provisionRepo := provision.NewPgProvisioningTaskRepository(c.PgPool)
	discoveryLogRepo := provision.NewPgParameterDiscoveryLogRepository(c.PgPool)
	provisionEngine := provision.NewProvisioningEngine(
		provisionRepo, c.DeviceService, c.TemplateService,
		c.Carriers, c.TaskSvc, c.EventBus, c.Cfg.Provision, logger,
	)
	provisionEngine.SetDeduper(c.Deduper)
	// B1：identify 阶段路由产品并回写 product_id/param_model_id。
	if c.ProductRegistry != nil && c.ProductRepo != nil {
		provisionEngine.SetProductBinder(c.ProductRegistry, c.ProductRepo)
	}

	if c.Cfg.Provision.ModelUpload.Enabled {
		modelUploadSvc := provision.NewModelUploadService(
			discoveryLogRepo, c.TaskSvc, c.MinIO,
			c.ProductRegistry, c.ParamIntersect,
			c.Cfg.Provision.ModelUpload, logger,
		)
		provisionEngine.SetModelUploadService(modelUploadSvc)
		logger.Info("model upload service enabled",
			zap.String("upload_url", c.Cfg.Provision.ModelUpload.UploadURL))
	}
	if c.Cfg.Provision.AutoSync.Enabled {
		planStore := provision.NewSyncPlanStore(c.Redis)
		syncSvc := provision.NewSyncService(
			c.ParamRepo, discoveryLogRepo, c.TaskSvc, planStore,
			c.Cfg.Provision.AutoSync, c.Cfg.Provision.AutoSync.GPVBatchSize, logger,
		).WithParamRegistry(c.ParamRegistry, c.ProductRegistry, true)
		provisionEngine.SetSyncService(syncSvc)
		logger.Info("auto-sync service enabled")
	}

	if err := provisionEngine.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe provisioning engine", zap.Error(err))
	}
	provisionEngine.StartTaskReaper()
	logger.Info("provisioning engine started",
		zap.Bool("auto_configure", c.Cfg.Provision.AutoConfigure),
		zap.Bool("model_upload", c.Cfg.Provision.ModelUpload.Enabled),
		zap.Bool("auto_sync", c.Cfg.Provision.AutoSync.Enabled),
	)

	c.miscDeps.provisionRepo = provisionRepo
	c.miscDeps.provisionEngine = provisionEngine

	return nil
}

// initTaskModule 初始化 F06 任务队列模块。
// TaskService 核心已在 bootstrap 中创建（供 BridgeQueue 使用），
// 此处仅添加运行时增强（指标、Connection Request）并注册 handler。
func initTaskModule(c *Container) error {
	logger := c.Logger.Named("task")

	taskService := c.TaskSvc
	taskService.SetMetrics(task.NewTaskMetrics(c.MetricsReg))

	// Wire Connection Request into TaskService
	udpSender := connreq.NewUDPSender(c.StunStore, c.Cfg.ConnReq.SharedSecret, c.Logger)
	crDispatcher := connreq.NewDispatcher(c.ConnReqClient, udpSender, c.Logger)
	crDispatcher.SetMetrics(connreq.NewDispatcherMetrics(c.MetricsReg))
	taskService.SetConnectionRequester(
		&taskDeviceLookup{repo: c.DeviceRepo},
		&taskCRSender{dispatcher: crDispatcher, serverAddr: c.Cfg.ConnReq.ServerAddr},
	)

	c.miscDeps.taskHandler = task.NewHandler(taskService)
	c.miscDeps.taskSvc = taskService

	logger.Info("task queue module initialized")
	return nil
}

// initBackupModule 初始化 F06 备份模块。
func initBackupModule(c *Container) error {
	logger := c.Logger.Named("backup")

	backupTaskRepo := backup.NewPgTaskRepository(c.PgPool)
	backupScheduleRepo := backup.NewPgScheduleRepository(c.PgPool)
	ftpRepo := backup.NewPgFTPConfigRepository(c.PgPool)
	backupService := backup.NewService(backupTaskRepo, backupScheduleRepo, c.EventBus, logger)
	backupHandler := backup.NewHandler(backupService, ftpRepo, logger)

	// T-0071 / R-102 followup: singleton BackupPolicy persistence.
	policyRepo := backup.NewPgPolicyRepository(c.PgPool)
	policyService := backup.NewPolicyService(policyRepo, logger)
	// T-0075: wire KeyProvider for PUT-time validation. The same env var is
	// read on both ACS and App processes — keep them in sync via deployment
	// config (systemd EnvironmentFile= or k8s ConfigMap).
	if kp, kpErr := backup.NewEnvKeyProvider(); kpErr != nil {
		logger.Error("backup encryption key invalid; PUT /backup/policy will block AES-256-GCM",
			zap.Error(kpErr))
	} else {
		policyService.SetKeyProvider(kp)
	}
	backupHandler.SetPolicyService(policyService)

	// T-0073 Phase 1: BackupPolicy enforcement Monitor (cleanup cron) +
	// metrics. Failure-alarm publishing is wired on the worker side
	// (executor) — see cmd/worker/main.go.
	policyMetrics := backup.NewPolicyMetrics(c.MetricsReg)
	backupPolicyMonitor := backup.NewPolicyMonitor(policyService, backupTaskRepo, policyMetrics, logger)
	// T-0076 Phase 2: enable physical MinIO object deletion alongside DB
	// cleanup. minio is optional (nil-safe); when wired, RunCleanupOnce
	// calls RemoveObject for each deleted backup_task's file_path.
	backupPolicyMonitor.SetMinIO(c.MinIO)
	// T-0082: enable hourly bucket-usage poll + edge-trigger alarm.raised/
	// cleared. Both BucketLister and EventBus must be set; either nil
	// disables the storage check (preserves T-0073/T-0076 behaviour).
	backupPolicyMonitor.SetBucketLister(c.MinIO)
	backupPolicyMonitor.SetEventBus(c.EventBus)
	// T-0083: enable weekly multi-device orphan reaper. PgTaskRepository's
	// ListAllTaskIDPrefixes satisfies the narrow TaskIDLister contract;
	// reaper additionally requires SetMinIO + SetBucketLister wired above.
	backupPolicyMonitor.SetTaskIDLister(backupTaskRepo)
	if err := backupPolicyMonitor.Start(context.Background()); err != nil {
		logger.Warn("start backup policy monitor", zap.Error(err))
	}
	c.miscDeps.backupPolicyMonitor = backupPolicyMonitor

	// T-0072 / R-102 followup: restore endpoint + restore_tasks tracking.
	restoreRepo := backup.NewPgRestoreTaskRepository(c.PgPool)
	restoreMetrics := backup.NewRestoreMetrics(c.MetricsReg)
	restoreService := backup.NewRestoreService(
		restoreRepo, c.DeviceRepo, c.TaskSvc, c.MinIO, restoreMetrics, logger,
	)
	// T-0079: enable by-task-id restore mode + subscribe FilePathRecorder to
	// `backup.file.received` events so backup_tasks.file_path is populated
	// after CPE finishes uploading. Both wire onto the same RestoreMetrics.
	restoreService.SetBackupTaskFinder(backupTaskRepo)
	filePathRecorder := backup.NewFilePathRecorder(backupTaskRepo, restoreMetrics, logger)
	if err := filePathRecorder.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe backup file path recorder", zap.Error(err))
	}
	backupHandler.SetRestoreService(restoreService)

	// T-0032 + T-0093: FTP/SFTP/FTPS connection-test service with
	// default 5s timeout. FTP path uses stdlib net/textproto; SFTP
	// uses x/crypto/ssh password auth; FTPS uses crypto/tls implicit
	// (port 990 style) handshake followed by FTP USER/PASS over TLS.
	backupHandler.SetFTPTester(backup.NewFTPConnectionTester(nil, 0, logger))

	c.miscDeps.backupHandler = backupHandler

	logger.Info("backup module initialized")
	return nil
}

// initDashboardModule 初始化 F06 仪表盘模块。
func initDashboardModule(c *Container) error {
	logger := c.Logger.Named("dashboard")

	dashboardService := dashboard.NewService(c.DeviceService, c.AlarmPgStore, c.PMKPIRepo, c.PgPool, c.GroupRepo, logger)
	dashboardHandler := dashboard.NewHandler(dashboardService)

	c.miscDeps.dashboardHandler = dashboardHandler

	logger.Info("dashboard module initialized")
	return nil
}

// initNorthboundModule 初始化 F08 北向/OSS 接口模块。
func initNorthboundModule(c *Container) error {
	logger := c.Logger.Named("northbound")

	pushEngine := push.NewEngine(c.Cfg.Northbound.PushTargets, logger)
	if err := pushEngine.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe push engine", zap.Error(err))
	}
	c.GS.Register("push-engine", 1, func(ctx context.Context) error { return pushEngine.Close() })

	syncService := nbsync.NewService(c.DeviceRepo, c.AlarmPgStore, c.PMCounterRepo, c.PMKPIRepo, c.ParamRepo, logger)
	nbService := northbound.NewNorthboundService(c.AlarmPgStore, c.PMCounterRepo, c.PMKPIRepo, c.ParamRepo, pushEngine, syncService, logger)
	nbRouter := northbound.NewRouter(nbService)

	// 主备 OSS 服务器配置 + 切换（system/config 北向设置页消费）。
	// 配置面与数据面分离：nbService 管数据导出 / 推送 / 同步；ServerService 管
	// active 组配置 (primary <-> standby)。
	// T-0099: ServerService 注入 EventBus，SetActive/Update 后发
	// SubjectNorthboundServerChanged → push engine 订阅 reload 关闭"切换即生效"环。
	nbServerRepo := northbound.NewPgServerRepository(c.PgPool)
	nbServerSvc := northbound.NewServerService(nbServerRepo, c.EventBus, logger)
	nbRouter.SetServerService(nbServerSvc)

	// T-0099: push engine 接入 active server provider + 启动期 refresh 兜底。
	// 失败仅 warn（dev 环境 northbound_servers 表可能空），不阻塞启动。
	pushEngine.SetActiveServerProvider(nbServerSvc)
	if err := pushEngine.RefreshActiveTarget(context.Background()); err != nil {
		logger.Warn("initial active target refresh failed", zap.Error(err))
	}

	c.miscDeps.nbRouter = nbRouter

	logger.Info("northbound/OSS module initialized")
	return nil
}

// initInteropModule 初始化 F10 互操作测试模块。
// T-0098 P5-01：dataModelReg 已删除，直接注入 ParamRegistry / ProductRegistry。
func initInteropModule(c *Container) error {
	logger := c.Logger.Named("interop")

	testRunner := interop.NewConformanceTestRunner(c.DeviceRepo, c.ParamRepo, c.ParamRegistry, c.ProductRegistry, c.TaskSvc, logger)
	testRunner.RegisterCases(cases.ProtocolCases())
	testRunner.RegisterCases(cases.DataModelCases())
	testRunner.RegisterCases(cases.RPCCases())
	testRunner.RegisterCases(cases.InformCases())
	testRunner.RegisterCases(cases.FaultInjectCases())
	dmValidator := interop.NewDataModelValidator(c.ParamRegistry, c.ProductRegistry, c.ParamRepo, c.DeviceRepo, logger)
	interopHandler := interop.NewHandler(testRunner, dmValidator, logger)

	c.miscDeps.interopHandler = interopHandler

	logger.Info("interop testing module initialized")
	return nil
}

// initMiscModules 初始化其余小型模块。
func initMiscModules(c *Container) error {
	logger := c.Logger

	// Syslog module
	syslogRepo := syslog.NewPgSyslogRepository(c.PgPool)
	c.miscDeps.syslogHandler = syslog.NewHandler(syslogRepo, logger)

	// Config sync
	c.miscDeps.syncHandler = config.NewSyncHandler(c.TaskSvc, logger)

	// File Manager module
	fileRepo := filemanager.NewPgFileRepository(c.PgPool)
	fileService := filemanager.NewFileService(fileRepo, c.MinIO, c.Cfg.MinIO.Buckets.ConfigBackup, c.TaskSvc, logger)
	c.miscDeps.fileHandler = filemanager.NewHandler(fileService, logger)
	logger.Info("file manager module initialized")

	// SSE / Events module (must init before MML so hub is available)
	eventStore := events.NewRedisMessageStore(c.Redis, time.Hour)
	messageHub := events.NewMessageHub(eventStore, logger)
	sseHandler := events.NewSSEHandler(messageHub, c.JWTService, logger)
	c.miscDeps.sseHandler = sseHandler
	c.miscDeps.messageHub = messageHub
	logger.Info("SSE events module initialized")

	// Notification module
	notifRepo := notification.NewPgRepository(c.PgPool)
	notifService := notification.NewService(notifRepo, messageHub, logger)
	c.miscDeps.notificationHandler = notification.NewHandler(notifService, logger)

	// W2.A.4 / T-0043: Notification template + history submodules
	templateRepo := notification.NewPgTemplateRepository(c.PgPool)
	templateService := notification.NewTemplateService(templateRepo, logger)
	c.miscDeps.notifTemplateHandler = notification.NewTemplateHandler(templateService, logger)

	historyRepo := notification.NewPgHistoryRepository(c.PgPool)
	historyService := notification.NewHistoryService(historyRepo, logger)
	c.miscDeps.notifHistoryHandler = notification.NewHistoryHandler(historyService, logger)

	logger.Info("notification module initialized")

	// MML Console module
	mmlCmdRepo := mml.NewPgCommandRepository(c.PgPool)
	mmlScriptRepo := mml.NewPgScriptRepository(c.PgPool)
	mmlTaskRepo := mml.NewPgTaskRepository(c.PgPool)
	mmlCustomCmdRepo := mml.NewPgCustomCommandRepository(c.PgPool)
	mmlAuditRepo := mml.NewPgAuditRepository(c.PgPool)
	mmlCmdParamRepo := mml.NewPgCommandParamRepository(c.PgPool)
	mmlService := mml.NewService(mmlCmdRepo, mmlScriptRepo, mmlTaskRepo, mmlCustomCmdRepo, messageHub, logger)
	mmlService.SetAuditRepo(mmlAuditRepo)
	mmlService.SetCmdParamRepo(mmlCmdParamRepo)
	// T-0090-c：注入 admin RoleRepo 作 RBAC group 派生器，让 ListCustomCommands
	// 走 group-share 路径（同组管理员可见对方 private 命令）。c.RoleRepo 由 admin
	// 模块初始化时（router.go misc Depends admin）填入，此处必非 nil。
	if c.RoleRepo != nil {
		mmlService.SetRoleQuerier(c.RoleRepo)
	}
	c.miscDeps.mmlHandler = mml.NewHandler(mmlService, logger)
	c.miscDeps.mmlService = mmlService

	// Wire MML fan-out to device tasks. misc 模块在 router.go 声明 Depends=["task"]，
	// 保证此处 c.miscDeps.taskSvc 一定已就绪。
	if c.miscDeps.taskSvc != nil {
		fanouter := mml.NewFanouter(c.miscDeps.taskSvc, logger)
		mmlService.SetFanouter(fanouter)

		// device_tasks 终态通过 NATS 跨进程事件投递到聚合器：
		// ACS 在 MarkTaskCompleted/Failed 后发布 task.completed/task.failed，
		// 本进程的 bridge 订阅后驱动 ResultAggregator 更新 mml_tasks 统计并推送 SSE。
		aggregator := mml.NewResultAggregator(mmlTaskRepo, mmlScriptRepo, messageHub, logger)
		if c.EventBus != nil {
			completionRouter := task.NewCompletionRouter(logger)
			completionRouter.Register(task.TaskSourceMML, aggregator)
			// D2 修复：provision 创建的 device_task（GPV / Upload / SPV / Reboot）source=system，
			// 失败时由 ProvisioningEngine 回查 source_id（=ProvisioningTask.id）联动 fail。
			if c.miscDeps.provisionEngine != nil {
				completionRouter.Register(task.TaskSourceSystem, c.miscDeps.provisionEngine)
			}
			bridge := task.NewCompletionEventBridge(logger, completionRouter, c.Deduper)
			if err := bridge.Subscribe(c.EventBus); err != nil {
				logger.Warn("subscribe task completion bridge", zap.Error(err))
			}
		} else {
			// 单进程部署（单测/无 NATS）下退化为同进程回调
			c.miscDeps.taskSvc.AddCompletionCallback(aggregator)
			if c.miscDeps.provisionEngine != nil {
				c.miscDeps.taskSvc.AddCompletionCallback(c.miscDeps.provisionEngine)
			}
		}

		logger.Info("MML fan-out bridge enabled")
	}

	// Parameter Library module
	paramRepo := mml.NewPgParamRepository(c.PgPool)
	paramService := mml.NewParamService(paramRepo)
	c.miscDeps.paramHandler = mml.NewParamHandler(paramService, logger)
	logger.Info("MML console module initialized")

	// Config Baseline module
	baselineRepo := baseline.NewPgBaselineRepository(c.PgPool)
	configTaskRepo := baseline.NewPgConfigTaskRepository(c.PgPool)
	neighborRepo := baseline.NewPgNeighborRepository(c.PgPool)
	baselineSvc := baseline.NewService(baselineRepo, configTaskRepo, neighborRepo, logger)
	c.BaselineSvc = baselineSvc
	c.miscDeps.baselineHandler = baseline.NewHandler(baselineSvc, logger)
	logger.Info("config baseline module initialized")

	// License module + Enforcer (T-0015 / R-103).
	// Enforcer is registered with the global Prometheus registry so the
	// 6 license metrics are scraped without further wiring. Enforcer is
	// wired into Service for cache invalidation and also exposed via
	// Container so DeviceService can pick it up via SetLicenseEnforcer.
	licenseRepo := license.NewPgLicenseRepository(c.PgPool)
	licenseSvc := license.NewService(licenseRepo, logger)
	licenseMetrics := license.NewEnforcementMetrics(c.MetricsReg)
	licenseEnforcer := license.NewEnforcer(licenseRepo, logger, licenseMetrics)
	licenseSvc.SetEnforcer(licenseEnforcer)
	licenseHandler := license.NewHandler(licenseSvc, logger)
	licenseMonitor := license.NewMonitor(licenseRepo, license.NoopAlertSink{}, licenseMetrics, logger)

	// T-0100-P0：审计日志（license_logs 表）。enforcer / monitor / handler 三处写入点
	// 通过 SetLogWriter 注入；NewLogWriter 内部失败降级 warn 不阻断主业务（详见
	// internal/license/log_writer.go）。
	licenseLogRepo := license.NewPgLicenseLogRepository(c.PgPool)
	licenseLogWriter := license.NewLogWriter(licenseLogRepo, logger)
	licenseEnforcer.SetLogWriter(licenseLogWriter)
	licenseMonitor.SetLogWriter(licenseLogWriter)
	licenseHandler.SetLogWriter(licenseLogWriter)
	licenseHandler.SetLogRepo(licenseLogRepo)  // T-0100-P1：让 GET /licenses/logs 走真实 repo
	licenseSvc.SetLogRepo(licenseLogRepo)      // T-0100-P2：让 Summary 卡 enforcement_hits_7d 走真实 count
	c.miscDeps.licenseLogRepo = licenseLogRepo

	// T-0100-P4-C：OEM 公钥加载 + 注入 SignatureVerifier。dev 默认 strict=false
	// + 空 PublicKeyDir → 等价于 P3 stub（unverified 放过）；prod 推荐配置
	// configs/oem_public_keys/*.pem + strict=true 收紧。
	licenseVerifier := license.NewSignatureVerifier(c.Cfg.License.Signing.Strict)
	if dir := c.Cfg.License.Signing.PublicKeyDir; dir != "" {
		if loadErr := licenseVerifier.LoadKeysFromDir(dir); loadErr != nil {
			logger.Warn("license OEM public key load reported errors (non-fatal)",
				zap.String("dir", dir), zap.Error(loadErr))
		}
		logger.Info("license signature verifier loaded",
			zap.String("dir", dir),
			zap.Int("key_count", licenseVerifier.KeyCount()),
			zap.Bool("strict", c.Cfg.License.Signing.Strict))
	}
	// T-0100-P5-a W4：strict=true 但实际 0 keys 启动是高风险静默失败（所有
	// import 都会被拒，运维不知原因）。strict 模式必须有至少 1 个公钥，否则
	// Fatal 阻止启动让运维立刻定位（config.yaml license.signing.public_key_dir
	// 误配 / 公钥文件缺失 等场景）。
	if c.Cfg.License.Signing.Strict && licenseVerifier.KeyCount() == 0 {
		logger.Fatal("license strict mode requires at least 1 OEM public key but none loaded; check license.signing.public_key_dir",
			zap.String("dir", c.Cfg.License.Signing.PublicKeyDir))
	}
	licenseHandler.SetSignatureVerifier(licenseVerifier)

	// T-0100-P4-B：周级 license_logs 归档 cron。MinIO bucket 默认走 logs；
	// retention=0 / nil minio / 空 bucket → 静默禁用归档（dev 友好）。
	archiveBucket := c.Cfg.License.LogArchive.MinIOBucket
	if archiveBucket == "" {
		archiveBucket = c.Cfg.MinIO.Buckets.Logs
	}
	if c.Cfg.License.LogArchive.RetentionMonths > 0 && archiveBucket != "" && c.MinIO != nil {
		licenseArchiver := license.NewLogArchiver(
			licenseLogRepo, c.MinIO, archiveBucket,
			c.Cfg.License.LogArchive.RetentionMonths, logger,
		)
		licenseMonitor.SetArchiver(licenseArchiver, c.Cfg.License.LogArchive.Schedule)
		logger.Info("license log archive cron configured",
			zap.Int("retention_months", c.Cfg.License.LogArchive.RetentionMonths),
			zap.String("bucket", archiveBucket),
			zap.String("schedule", c.Cfg.License.LogArchive.Schedule))
	}

	c.miscDeps.licenseHandler = licenseHandler
	c.miscDeps.licenseEnforcer = licenseEnforcer
	c.miscDeps.licenseMonitor = licenseMonitor

	// Wire enforcer into DeviceService so device.create / future write ops
	// gate on capacity + expiry. Read-only operations are unaffected (D1).
	if c.DeviceService != nil {
		c.DeviceService.SetLicenseEnforcer(licenseEnforcer)
	}

	// Start the cron monitor. ctx-derived timeout per check ensures a stuck
	// scrape can't cascade-fail subsequent ticks.
	if err := c.miscDeps.licenseMonitor.Start(context.Background()); err != nil {
		logger.Warn("license monitor start failed", zap.Error(err))
	}
	logger.Info("license module initialized with enforcer + cron monitor")

	// OpsTools module
	opsTemplateRepo := ops.NewPgTemplateRepository(c.PgPool)
	opsTaskRepo := ops.NewPgTaskRepository(c.PgPool)
	opsCmdRepo := ops.NewPgCommandRecordRepository(c.PgPool)
	opsSvc := ops.NewService(opsTemplateRepo, opsTaskRepo, opsCmdRepo, logger)
	c.miscDeps.opsHandler = ops.NewHandler(opsSvc, logger)

	// F06 运维管理扩展（T-0101..T-0112）— 7 个新子系统
	opsExecRepo := ops.NewPgTaskExecutionRepository(c.PgPool)
	opsDiagRepo := ops.NewPgDiagnosticRepository(c.PgPool)
	opsDLRepo := ops.NewPgDownloadRepository(c.PgPool)
	opsAuditRepo := ops.NewPgAuditLogRepository(c.PgPool)
	opsMWRepo := ops.NewPgMaintenanceWindowRepository(c.PgPool)
	opsPBRepo := ops.NewPgPlaybookRepository(c.PgPool)

	opsAuditSvc := ops.NewAuditLogService(opsAuditRepo, logger)
	opsApprovalSvc := ops.NewApprovalService(opsTaskRepo, opsAuditSvc, logger)
	opsDiagSvc := ops.NewDiagnosticService(opsDiagRepo, opsAuditSvc, logger)
	opsDLSvc := ops.NewDownloadService(opsDLRepo, opsAuditSvc, logger)
	opsMWSvc := ops.NewMaintenanceWindowService(opsMWRepo, opsAuditSvc, logger)
	opsPBSvc := ops.NewPlaybookService(opsPBRepo, logger)
	opsExecutor := ops.NewTaskExecutor(opsTaskRepo, opsExecRepo, opsAuditSvc, logger)
	// T-0102-c: wire device-task enqueuer + SSE hub so inline RPC tasks
	// fan out to internal/task (Redis + PG) and emit per-device events.
	opsExecutor.SetEnqueuer(c.TaskSvc)
	opsBGSvc := ops.NewBreakGlassService(opsAuditSvc, logger)
	opsInspectionSvc := ops.NewInspectionService(opsDiagSvc, opsAuditSvc, logger)
	opsSSEHub := ops.NewSSEHub()
	opsExecutor.SetSSEHub(opsSSEHub)

	c.miscDeps.opsExtHandler = ops.NewExtHandler(
		opsDiagSvc, opsDLSvc, opsAuditSvc, opsMWSvc, opsPBSvc,
		opsExecutor, opsApprovalSvc, opsBGSvc, opsInspectionSvc,
		opsSvc, opsSSEHub, logger,
	)
	logger.Info("ops tools module initialized (incl. F06 ext T-0101..T-0112)")

	// Report module
	reportDefRepo := report.NewPgDefinitionRepository(c.PgPool)
	reportRecordRepo := report.NewPgRecordRepository(c.PgPool)
	reportService := report.NewService(reportDefRepo, reportRecordRepo, c.EventBus, logger)
	c.miscDeps.reportHandler = report.NewHandler(reportService, c.MinIO, c.Cfg.MinIO.Buckets.Reports, logger)
	logger.Info("report module initialized")

	// NE Direct module (conditional)
	if c.Cfg.NEDirect.Enabled {
		neSessionRepo := nedirect.NewPgSessionRepository(c.PgPool)
		neCommandRepo := nedirect.NewPgCommandRepository(c.PgPool)
		neService := nedirect.NewService(neSessionRepo, neCommandRepo, c.DeviceService, c.AlarmEngine, c.EventBus, logger)
		neHandler := nedirect.NewHandler(neService, logger)
		neServer := nedirect.NewServer(c.Cfg.NEDirect, neHandler, logger)
		if err := neServer.Start(); err != nil {
			logger.Error("ne-direct server start failed", zap.Error(err))
		} else {
			c.GS.Register("ne-direct", 1, func(ctx context.Context) error { return neServer.Shutdown(ctx) })
			logger.Info("ne-direct server started",
				zap.String("host", c.Cfg.NEDirect.Host),
				zap.Int("port", c.Cfg.NEDirect.Port))
		}
	}

	// Device Rules (topology rules)
	ruleRepo := topology.NewPgDeviceRuleRepository(c.PgPool)
	ruleTaskRepo := topology.NewPgRuleTaskRepository(c.PgPool)
	matcher := topology.NewDeviceMatcher(c.GroupRepo, c.PgPool, logger)
	ruleService := topology.NewDeviceRuleService(ruleRepo, ruleTaskRepo, c.GroupRepo, matcher, c.PgPool, 4, logger)
	// T-0027 S3 Day 4：注入 PgDeviceLister 替换 getAllDevices stub
	// 见 prd/F06-topology-auto-grouping.md §12.1，让 ApplyRule 能扫描真实设备
	ruleService.SetDeviceLister(topology.NewPgDeviceLister(c.PgPool, logger))
	// T-0027 S3 Day 7：注入 EventBus 让 Start 装配 device.registered 订阅
	// PRD §12.7 corrected：项目模式订 device.registered 而非早稿 device.inform.bootstrap
	// 避免 InformHandler race（与 ProvisioningEngine 同模式，commit 2026-03-18）
	ruleService.SetEventBus(c.EventBus)
	// T-0027 S3 Day 8：注入 RuleMetrics（PRD §12.5 — 6 metric / 7 log key）
	ruleService.SetMetrics(topology.NewRuleMetrics(c.MetricsReg))
	// T-0027 S3 Day 6：启动 cron @hourly reEvaluateAll
	// PRD §12.1 §D2；A4 manual 守护已在 AddDeviceWithSource SQL 层强制
	// ctx=Background — cron.Cron 自带 goroutine 生命周期，进程退出随之结束
	if err := ruleService.Start(context.Background()); err != nil {
		logger.Error("topology rule service Start failed", zap.Error(err))
	}
	c.miscDeps.ruleHandler = topology.NewRuleHandler(ruleService)

	// System Info endpoint
	c.miscDeps.sysInfoHandler = components.NewSystemInfoHandler(c.PgPool, c.Redis, logger)

	// PM threshold
	c.miscDeps.thresholdRepo = pm.NewPgThresholdRepository(c.PgPool)

	// Dead-letter admin handler (T-0012 / R-106).
	// Worker process owns the runner that writes to dead_letters; the app
	// process exposes read/delete/replay over /admin/dead-letters. Replay uses
	// a per-module runner.Runner (sharing the EventBus publisher) so the app
	// can re-publish events back to the bus — worker subscribers will pick
	// them up like any normal event.
	dlqRepo := dlq.NewPgRepository(c.PgPool)
	dlqMetrics := runner.NewMetrics(c.MetricsReg)
	dlqHandler := admin.NewDeadLetterHandler(dlqRepo, logger)
	if c.EventBus != nil {
		pmReplayer := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, c.EventBus, dlqMetrics, logger)
		dlqHandler.SetReplayer("pm", pmReplayer)
	}
	c.miscDeps.deadLetterHandler = dlqHandler
	logger.Info("dead-letter admin handler initialized")

	return nil
}

// miscDeps holds handlers and deps from miscellaneous modules.
// These are populated by initMiscModules and consumed during route registration.
type miscDeps struct {
	// MR
	mrStore   *mr.PgMRStore
	mrIndRepo *mr.PgIndicatorRepository
	mrMapRepo *mr.PgMappingRepository

	// Software
	softwareHandler *software.Handler
	canaryMonitor   *software.CanaryMonitor

	// Provision
	provisionRepo   *provision.PgProvisioningTaskRepository
	provisionEngine *provision.ProvisioningEngine

	// Task
	taskHandler *task.Handler
	taskSvc     *task.TaskService

	// Backup
	backupHandler       *backup.Handler
	backupPolicyMonitor *backup.PolicyMonitor // T-0073 Phase 1

	// Dashboard
	dashboardHandler *dashboard.Handler

	// Northbound
	nbRouter *northbound.Router

	// Interop
	interopHandler *interop.Handler

	// Syslog
	syslogHandler *syslog.Handler

	// Config sync
	syncHandler *config.SyncHandler

	// File Manager
	fileHandler *filemanager.Handler

	// MML
	mmlHandler *mml.Handler
	mmlService *mml.Service

	// Param Library
	paramHandler *mml.ParamHandler

	// Baseline
	baselineHandler *baseline.Handler

	// License + enforcement (T-0015 / R-103)
	licenseHandler  *license.Handler
	licenseEnforcer *license.EnforcerImpl
	licenseMonitor  *license.Monitor
	// T-0100-P0：审计日志 repo（暴露给 P1 GET /licenses/logs handler 复用）
	licenseLogRepo license.LicenseLogRepository

	// Ops
	opsHandler    *ops.Handler
	opsExtHandler *ops.ExtHandler

	// Report
	reportHandler *report.Handler

	// Device Rules
	ruleHandler *topology.RuleHandler

	// System Info
	sysInfoHandler *components.SystemInfoHandler

	// PM Threshold
	thresholdRepo *pm.PgThresholdRepository

	// SSE / Events
	sseHandler          *events.SSEHandler
	notificationHandler *notification.Handler
	messageHub          *events.MessageHub

	// W2.A.4 / T-0043: Notification template + history
	notifTemplateHandler *notification.TemplateHandler
	notifHistoryHandler  *notification.HistoryHandler

	// T-0012 / R-106: worker retry + dead-letter queue admin
	deadLetterHandler *admin.DeadLetterHandler
}

// taskDeviceLookup adapts device.DeviceReader to task.DeviceLookup.
type taskDeviceLookup struct {
	repo device.DeviceReader
}

func (a *taskDeviceLookup) GetConnectionRequestURL(ctx context.Context, deviceSN string) (string, error) {
	dev, err := a.repo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return "", fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}
	if dev == nil {
		return "", nil
	}
	return dev.ConnectionRequestURL, nil
}

// taskCRSender adapts connreq.Dispatcher to task.ConnectionRequestSender.
type taskCRSender struct {
	dispatcher *connreq.Dispatcher
	serverAddr string
}

func (a *taskCRSender) Send(ctx context.Context, deviceSN, httpURL string) error {
	return a.dispatcher.Send(ctx, deviceSN, httpURL, a.serverAddr, true)
}
