package router

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/interop"
	"github.com/omcgo/omcgo/internal/interop/cases"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/nedirect"
	"github.com/omcgo/omcgo/internal/northbound"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/dashboard"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/filemanager"
	"github.com/omcgo/omcgo/internal/license"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/ops"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/omcgo/omcgo/internal/syslog"
	"github.com/omcgo/omcgo/internal/topology"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/provision"
)

// Setup creates all repository/service/handler instances and registers routes.
func Setup(r *gin.Engine, deps *Deps) {
	pgPool := deps.PgPool
	tsPool := deps.TsPool
	redisClient := deps.Redis
	minioClient := deps.MinIO
	eventBus := deps.EventBus
	cmdQueue := deps.CmdQueue
	carrierRegistry := deps.CarrierRegistry
	cfg := deps.Cfg
	logger := deps.Logger
	gs := deps.GS
	metricsReg := deps.MetricsReg

	// Device repositories
	deviceRepo := device.NewPgDeviceRepository(pgPool)
	paramRepo := device.NewPgDeviceParameterRepository(pgPool)

	// HeartbeatMonitor
	heartbeatMonitor := device.NewHeartbeatMonitor(redisClient, deviceRepo, logger)
	heartbeatMonitor.Start()
	gs.Register("heartbeat", 1, func(ctx context.Context) error { heartbeatMonitor.Stop(); return nil })

	// Device services
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, heartbeatMonitor, eventBus, logger)

	// Subscribe InformHandler to events
	informHandler := device.NewInformHandler(deviceService, carrierRegistry, model.CarrierCMCC, logger)
	if err := informHandler.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
	}

	// DataModel module (repository + cache + registry + importer)
	dmRepo := datamodel.NewPgDataModelRepository(pgPool)
	importLogRepo := datamodel.NewPgImportLogRepository(pgPool)
	ouiRepo := datamodel.NewPgOUIRepository(pgPool)
	dmCache := datamodel.NewDataModelCache(redisClient)
	dmRegistry := datamodel.NewDataModelRegistry(dmRepo, dmCache, logger)
	dmRegistry.Start()
	gs.Register("datamodel-registry", 1, func(ctx context.Context) error { dmRegistry.Stop(); return nil })
	dmImporter := datamodel.NewDataModelImporter(dmRepo, importLogRepo)
	logger.Info("datamodel registry started")

	// ConfigTemplate module
	templateRepo := template.NewPgConfigTemplateRepository(pgPool)
	templateService := template.NewConfigTemplateService(templateRepo, logger)

	// Provisioning module
	provisionRepo := provision.NewPgProvisioningTaskRepository(pgPool)
	provisionEngine := provision.NewProvisioningEngine(
		provisionRepo, deviceService, dmRegistry, templateService,
		carrierRegistry, cmdQueue, eventBus, logger,
	)
	if err := provisionEngine.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe provisioning engine", zap.Error(err))
	}

	// Topology module
	groupRepo := topology.NewPgDeviceGroupRepository(pgPool)
	groupService := topology.NewDeviceGroupService(groupRepo, logger)
	siteRepo := topology.NewPgSiteRepository(pgPool)
	topoNodeRepo := topology.NewPgTopoNodeRepository(pgPool)
	topoEdgeRepo := topology.NewPgTopoEdgeRepository(pgPool)

	// Admin/RBAC module
	userRepo := admin.NewPgUserRepository(pgPool)
	roleRepo := admin.NewPgRoleRepository(pgPool)
	auditRepo := admin.NewPgAuditRepository(pgPool)
	jwtService := admin.NewJWTServiceWithTTL(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	adminService := admin.NewAdminService(userRepo, roleRepo, auditRepo, jwtService, logger)
	adminHandler := admin.NewHandler(adminService, logger)
	logger.Info("admin/RBAC module initialized")

	// Software Management module
	firmwareRepo := software.NewPgFirmwareRepository(pgPool)
	upgradeRepo := software.NewPgUpgradeTaskRepository(pgPool)
	connReqClient := connreq.NewClient(redisClient, logger)
	softwareService := software.NewSoftwareService(
		firmwareRepo, upgradeRepo, deviceRepo, cmdQueue, connReqClient,
		minioClient, cfg.MinIO.Buckets.Firmware, eventBus, logger,
	)
	if err := softwareService.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe software service", zap.Error(err))
	}
	softwareHandler := software.NewHandler(softwareService, firmwareRepo, upgradeRepo, logger)
	logger.Info("software management module initialized")

	// PM module components
	pmCounterRepo := counter.NewPgCounterRepository(tsPool)
	pmKPIRepo := kpi.NewPgKPIRepository(tsPool)
	pmKPIEngine := kpi.NewKPIEngine(pmCounterRepo, pmKPIRepo, carrierRegistry, logger)
	pmTaskRepo := pm.NewPgTaskRepository(pgPool)
	pmFileStore := pm.NewPgPMFileStore(pgPool)

	// Alarm module components
	alarmRedisStore := alarm.NewRedisAlarmStore(redisClient)
	alarmPgStore := alarm.NewPgAlarmStore(pgPool, tsPool)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, carrierRegistry, eventBus, logger)

	// MR module components
	mrStore := mr.NewPgMRStore(pgPool, tsPool)
	mrIndRepo := mr.NewPgIndicatorRepository(pgPool)
	mrMapRepo := mr.NewPgMappingRepository(pgPool)

	// Setup Gin middleware
	corsOrigins := cfg.CORS.AllowOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	// Setup Request ID middleware with configurable prefix
	requestIDPrefix := cfg.RequestIDPrefix
	if requestIDPrefix == "" {
		requestIDPrefix = "app" // default prefix for app service
	}
	r.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{Prefix: requestIDPrefix}))

	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: corsOrigins,
	}))
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.PrometheusMetrics(metricsReg))

	// Unified JSON 404 for unmatched routes
	r.NoRoute(func(c *gin.Context) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
	})

	// Health check (public, no auth)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public auth routes (no authentication required)
	publicV1 := r.Group("/api/v1")
	adminHandler.RegisterAuthRoutes(publicV1)

	// Protected API v1 routes (JWT authentication required)
	v1 := r.Group("/api/v1")
	v1.Use(admin.RequireAuth(jwtService))
	v1.Use(admin.RequireCarrier())
	v1.Use(admin.AuditLogger(auditRepo))

	// Protected auth routes
	v1.GET("/auth/me", adminHandler.Me)

	// Device routes
	deviceHandler := device.NewHandler(deviceService)
	deviceHandler.RegisterRoutes(v1)

	// DataModel routes
	dmHandler := datamodel.NewHandler(dmRepo, ouiRepo, dmRegistry, dmImporter)
	dmHandler.RegisterRoutes(v1)

	// ConfigTemplate routes
	templateHandler := template.NewHandler(templateRepo)
	templateHandler.RegisterRoutes(v1)

	// Provisioning routes
	provisionHandler := provision.NewHandler(provisionRepo, provisionEngine)
	provisionHandler.RegisterRoutes(v1)

	// Topology routes
	topologyHandler := topology.NewHandler(groupRepo, groupService, siteRepo, topoNodeRepo, topoEdgeRepo)
	topologyHandler.RegisterRoutes(v1)

	// PM routes
	pmHandler := pm.NewHandler(pmCounterRepo, pmKPIRepo, pmKPIEngine, pmTaskRepo, pmFileStore, minioClient, cfg.MinIO.Buckets.PMFiles, logger)
	pmHandler.RegisterRoutes(v1)

	// Alarm routes
	alarmHandler := alarm.NewHandler(alarmEngine, alarmPgStore, logger)
	alarmHandler.RegisterRoutes(v1)

	// Alarm rule routes
	alarmRuleRepo := alarm.NewPgAlarmRuleRepository(pgPool)
	alarmRuleHandler := alarm.NewRuleHandler(alarmRuleRepo, logger)
	alarmsGroup := v1.Group("/alarms")
	alarmRuleHandler.RegisterRoutes(alarmsGroup)

	// KPI threshold routes
	thresholdRepo := pm.NewPgThresholdRepository(pgPool)
	thresholdHandler := pm.NewThresholdHandler(thresholdRepo, logger)
	pmGroup := v1.Group("/pm")
	thresholdHandler.RegisterRoutes(pmGroup)

	// Dashboard routes
	dashboardService := dashboard.NewService(deviceService, alarmPgStore, pmKPIRepo, pgPool, groupRepo, logger)
	dashboardHandler := dashboard.NewHandler(dashboardService)
	dashboardHandler.RegisterRoutes(v1)

	// Syslog routes
	syslogRepo := syslog.NewPgSyslogRepository(pgPool)
	syslogHandler := syslog.NewHandler(syslogRepo, logger)
	syslogHandler.RegisterRoutes(v1)

	// Config sync routes
	syncHandler := config.NewSyncHandler(cmdQueue, logger)
	syncHandler.RegisterRoutes(v1)

	// MR routes
	mrHandler := mr.NewHandler(mrStore, mrIndRepo, mrMapRepo, minioClient, cfg.MinIO.Buckets.MRFiles, logger)
	mrHandler.RegisterRoutes(v1)

	// Software routes
	softwareHandler.RegisterRoutes(v1)

	// Interop Testing module (F10)
	testRunner := interop.NewConformanceTestRunner(deviceRepo, paramRepo, dmRegistry, cmdQueue, logger)
	testRunner.RegisterCases(cases.ProtocolCases())
	testRunner.RegisterCases(cases.DataModelCases())
	testRunner.RegisterCases(cases.RPCCases())
	dmValidator := interop.NewDataModelValidator(dmRegistry, paramRepo, deviceRepo, logger)
	interopHandler := interop.NewHandler(testRunner, dmValidator, logger)
	interopHandler.RegisterRoutes(v1)
	logger.Info("interop testing module initialized")

	// Backup module
	backupTaskRepo := backup.NewPgTaskRepository(pgPool)
	backupScheduleRepo := backup.NewPgScheduleRepository(pgPool)
	ftpRepo := backup.NewPgFTPConfigRepository(pgPool)
	backupService := backup.NewService(backupTaskRepo, backupScheduleRepo, eventBus, logger)
	backupHandler := backup.NewHandler(backupService, ftpRepo, logger)
	backupHandler.RegisterRoutes(v1)
	logger.Info("backup module initialized")

	// File Manager module
	fileRepo := filemanager.NewPgFileRepository(pgPool)
	fileHandler := filemanager.NewHandler(fileRepo, minioClient, cfg.MinIO.Buckets.ConfigBackup, cmdQueue, logger)
	fileHandler.RegisterRoutes(v1)
	logger.Info("file manager module initialized")

	// MML Console module
	mmlCmdRepo := mml.NewPgCommandRepository(pgPool)
	mmlScriptRepo := mml.NewPgScriptRepository(pgPool)
	mmlTaskRepo := mml.NewPgTaskRepository(pgPool)
	mmlService := mml.NewService(mmlCmdRepo, mmlScriptRepo, mmlTaskRepo, logger)
	mmlHandler := mml.NewHandler(mmlService, logger)
	mmlHandler.RegisterRoutes(v1)
	logger.Info("MML console module initialized")

	// Config Baseline module
	baselineRepo := baseline.NewPgBaselineRepository(pgPool)
	configTaskRepo := baseline.NewPgConfigTaskRepository(pgPool)
	neighborRepo := baseline.NewPgNeighborRepository(pgPool)
	baselineSvc := baseline.NewService(baselineRepo, configTaskRepo, neighborRepo, logger)
	baselineHandler := baseline.NewHandler(baselineSvc, logger)
	baselineHandler.RegisterRoutes(v1)
	logger.Info("config baseline module initialized")

	// License module
	licenseRepo := license.NewPgLicenseRepository(pgPool)
	licenseSvc := license.NewService(licenseRepo, logger)
	licenseHandler := license.NewHandler(licenseSvc, logger)
	licenseHandler.RegisterRoutes(v1)
	logger.Info("license module initialized")

	// OpsTools module
	opsTemplateRepo := ops.NewPgTemplateRepository(pgPool)
	opsTaskRepo := ops.NewPgTaskRepository(pgPool)
	opsCmdRepo := ops.NewPgCommandRecordRepository(pgPool)
	opsSvc := ops.NewService(opsTemplateRepo, opsTaskRepo, opsCmdRepo, logger)
	opsHandler := ops.NewHandler(opsSvc, logger)
	opsHandler.RegisterRoutes(v1)
	logger.Info("ops tools module initialized")

	// Report module
	reportDefRepo := report.NewPgDefinitionRepository(pgPool)
	reportRecordRepo := report.NewPgRecordRepository(pgPool)
	reportService := report.NewService(reportDefRepo, reportRecordRepo, eventBus, logger)
	reportHandler := report.NewHandler(reportService, minioClient, cfg.MinIO.Buckets.Reports, logger)
	reportHandler.RegisterRoutes(v1)
	logger.Info("report module initialized")

	// System Info endpoint
	sysInfoHandler := components.NewSystemInfoHandler(pgPool, redisClient, logger)
	v1.GET("/system/info", sysInfoHandler.GetSystemInfo)
	logger.Info("system info endpoint initialized")

	// Northbound/OSS module
	nbPMHandler := northbound.NewPMHandler(pmCounterRepo, pmKPIRepo, logger)
	nbAlarmHandler := northbound.NewAlarmHandler(alarmPgStore, logger)
	nbConfigHandler := northbound.NewConfigHandler(paramRepo, logger)
	pushEngine := push.NewEngine(cfg.Northbound.PushTargets, logger)
	if err := pushEngine.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe push engine", zap.Error(err))
	}
	gs.Register("push-engine", 1, func(ctx context.Context) error { return pushEngine.Close() })
	syncService := nbsync.NewService(deviceRepo, alarmPgStore, pmCounterRepo, pmKPIRepo, paramRepo, logger)
	nbRouter := northbound.NewRouter(nbPMHandler, nbAlarmHandler, nbConfigHandler, pushEngine, syncService)
	nbRouter.RegisterRoutes(v1)
	logger.Info("northbound/OSS module initialized")

	// NE Direct module (CMCC only)
	if cfg.NEDirect.Enabled {
		neHandler := nedirect.NewHandler(deviceService, alarmEngine, eventBus, logger)
		neServer := nedirect.NewServer(cfg.NEDirect, neHandler, logger)
		if err := neServer.Start(); err != nil {
			logger.Error("ne-direct server start failed", zap.Error(err))
		} else {
			gs.Register("ne-direct", 1, func(ctx context.Context) error { return neServer.Shutdown(ctx) })
			logger.Info("ne-direct server started",
				zap.String("host", cfg.NEDirect.Host),
				zap.Int("port", cfg.NEDirect.Port))
		}
	}

	// Admin management routes (require admin permission)
	adminGroup := v1.Group("/admin")
	adminGroup.Use(admin.RequirePermission(roleRepo, "users", "admin"))
	adminHandler.RegisterAdminRoutes(adminGroup)
}
