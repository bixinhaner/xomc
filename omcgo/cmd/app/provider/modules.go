package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/dashboard"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/filemanager"
	"github.com/omcgo/omcgo/internal/interop"
	"github.com/omcgo/omcgo/internal/interop/cases"
	"github.com/omcgo/omcgo/internal/license"
	"github.com/omcgo/omcgo/internal/events"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/nedirect"
	"github.com/omcgo/omcgo/internal/northbound"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
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
	softwareHandler := software.NewHandler(softwareService, firmwareRepo, taskRepo, subTaskRepo, logger)

	c.miscDeps.softwareHandler = softwareHandler

	logger.Info("software management module initialized")
	return nil
}

// initProvisionModule 初始化 F09 自动开站模块。
func initProvisionModule(c *Container) error {
	logger := c.Logger.Named("provision")

	provisionRepo := provision.NewPgProvisioningTaskRepository(c.PgPool)
	discoveryLogRepo := provision.NewPgParameterDiscoveryLogRepository(c.PgPool)
	provisionEngine := provision.NewProvisioningEngine(
		provisionRepo, c.DeviceService, c.DMRegistry, c.TemplateService,
		c.Carriers, c.TaskSvc, c.EventBus, c.Cfg.Provision, logger,
	)

	if c.Cfg.Provision.ModelUpload.Enabled {
		modelUploadSvc := provision.NewModelUploadService(
			discoveryLogRepo, c.DMImporter, c.DMRegistry, c.TaskSvc,
			c.MinIO, c.Cfg.Provision.ModelUpload, logger,
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
		)
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

	c.miscDeps.nbRouter = nbRouter

	logger.Info("northbound/OSS module initialized")
	return nil
}

// initInteropModule 初始化 F10 互操作测试模块。
func initInteropModule(c *Container) error {
	logger := c.Logger.Named("interop")

	testRunner := interop.NewConformanceTestRunner(c.DeviceRepo, c.ParamRepo, c.DMRegistry, c.TaskSvc, logger)
	testRunner.RegisterCases(cases.ProtocolCases())
	testRunner.RegisterCases(cases.DataModelCases())
	testRunner.RegisterCases(cases.RPCCases())
	dmValidator := interop.NewDataModelValidator(c.DMRegistry, c.ParamRepo, c.DeviceRepo, logger)
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
			bridge := task.NewCompletionEventBridge(logger, completionRouter)
			if err := bridge.Subscribe(c.EventBus); err != nil {
				logger.Warn("subscribe task completion bridge", zap.Error(err))
			}
		} else {
			// 单进程部署（单测/无 NATS）下退化为同进程回调
			c.miscDeps.taskSvc.AddCompletionCallback(aggregator)
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

	// License module
	licenseRepo := license.NewPgLicenseRepository(c.PgPool)
	licenseSvc := license.NewService(licenseRepo, logger)
	c.miscDeps.licenseHandler = license.NewHandler(licenseSvc, logger)
	logger.Info("license module initialized")

	// OpsTools module
	opsTemplateRepo := ops.NewPgTemplateRepository(c.PgPool)
	opsTaskRepo := ops.NewPgTaskRepository(c.PgPool)
	opsCmdRepo := ops.NewPgCommandRecordRepository(c.PgPool)
	opsSvc := ops.NewService(opsTemplateRepo, opsTaskRepo, opsCmdRepo, logger)
	c.miscDeps.opsHandler = ops.NewHandler(opsSvc, logger)
	logger.Info("ops tools module initialized")

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
	c.miscDeps.ruleHandler = topology.NewRuleHandler(ruleService)

	// System Info endpoint
	c.miscDeps.sysInfoHandler = components.NewSystemInfoHandler(c.PgPool, c.Redis, logger)

	// PM threshold
	c.miscDeps.thresholdRepo = pm.NewPgThresholdRepository(c.PgPool)

	return nil
}

// miscDeps holds handlers and deps from miscellaneous modules.
// These are populated by initMiscModules and consumed during route registration.
type miscDeps struct {
	// MR
	mrStore    *mr.PgMRStore
	mrIndRepo  *mr.PgIndicatorRepository
	mrMapRepo  *mr.PgMappingRepository

	// Software
	softwareHandler *software.Handler

	// Provision
	provisionRepo    *provision.PgProvisioningTaskRepository
	provisionEngine  *provision.ProvisioningEngine

	// Task
	taskHandler *task.Handler
	taskSvc     *task.TaskService

	// Backup
	backupHandler *backup.Handler

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

	// License
	licenseHandler *license.Handler

	// Ops
	opsHandler *ops.Handler

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
