package provider

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/topology"
)

// Setup 初始化所有模块并注册路由。
// 按以下顺序执行：
//  1. 初始化各模块（按依赖顺序）
//  2. 配置 Gin 中间件
//  3. 注册所有路由
func Setup(r *gin.Engine, c *Container) error {
	// ===== Phase 1: 初始化各模块（按依赖顺序）=====

	// F02 数据模型与配置（无跨模块依赖）
	if err := initConfigModule(c); err != nil {
		return err
	}

	// F06 拓扑管理（无跨模块依赖）
	if err := initTopologyModule(c); err != nil {
		return err
	}

	// F06 用户/RBAC（依赖 GroupRepo from TopologyModule）
	if err := initAdminModule(c); err != nil {
		return err
	}

	// F06 设备管理（依赖 GroupRepo from TopologyModule）
	if err := initDeviceModule(c); err != nil {
		return err
	}

	// F04 告警管理（依赖 Carriers, EventBus）
	if err := initAlarmModule(c); err != nil {
		return err
	}

	// F03 性能管理（依赖 Carriers）
	if err := initPMModule(c); err != nil {
		return err
	}

	// F05 测量报告（无跨模块依赖）
	if err := initMRModule(c); err != nil {
		return err
	}

	// F06 固件管理（依赖 DeviceRepo, ConnReqClient from DeviceModule）
	if err := initSoftwareModule(c); err != nil {
		return err
	}

	// F09 自动开站（依赖 DeviceService, DMRegistry from DeviceModule/ConfigModule）
	if err := initProvisionModule(c); err != nil {
		return err
	}

	// F06 任务队列（依赖 DeviceRepo, StunStore from DeviceModule）
	if err := initTaskModule(c); err != nil {
		return err
	}

	// F06 备份（依赖 EventBus）
	if err := initBackupModule(c); err != nil {
		return err
	}

	// F06 仪表盘（依赖 DeviceService, AlarmPgStore, PMKPIRepo, GroupRepo）
	if err := initDashboardModule(c); err != nil {
		return err
	}

	// F08 北向接口（依赖 AlarmPgStore, PMCounterRepo, PMKPIRepo）
	if err := initNorthboundModule(c); err != nil {
		return err
	}

	// F10 互操作测试（依赖 DeviceRepo, DMRegistry）
	if err := initInteropModule(c); err != nil {
		return err
	}

	// 其余小型模块
	if err := initMiscModules(c); err != nil {
		return err
	}

	c.Logger.Info("all modules initialized")

	// ===== Phase 2: 配置 Gin 中间件 =====
	setupMiddleware(r, c)

	// ===== Phase 3: 注册所有路由 =====
	return registerRoutes(r, c)
}

// setupMiddleware 配置 Gin 全局中间件。
func setupMiddleware(r *gin.Engine, c *Container) {
	corsOrigins := c.Cfg.CORS.AllowOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	requestIDPrefix := c.Cfg.RequestIDPrefix
	if requestIDPrefix == "" {
		requestIDPrefix = "app"
	}

	r.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{Prefix: requestIDPrefix}))
	r.Use(middleware.Tracing("omcgo-app"))
	r.Use(middleware.CORS(middleware.CORSConfig{AllowOrigins: corsOrigins}))
	r.Use(middleware.RequestLogger())
	r.Use(middleware.PrometheusMetrics(c.MetricsReg))
	r.Use(middleware.SecurityHeaders())

	r.NoRoute(func(gc *gin.Context) {
		commonerrors.AbortWithError(gc, http.StatusNotFound, commonerrors.ErrNotFound)
	})

	r.GET("/healthz", func(gc *gin.Context) {
		gc.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// registerRoutes 注册所有模块的路由。
func registerRoutes(r *gin.Engine, c *Container) error {
	ad := c.adminHandlerDeps

	// Public auth routes (no authentication required)
	publicV1 := r.Group("/api/v1")
	ad.adminHandler.RegisterAuthRoutes(publicV1)

	// Protected API v1 routes (JWT or API Key authentication required)
	v1 := r.Group("/api/v1")
	v1.Use(admin.RequireAuthWithAPIKey(ad.jwtService, ad.apiKeySvc, ad.userRepo))
	v1.Use(admin.RequireCarrier())
	v1.Use(admin.AuditLogger(ad.auditRepo))

	// Protected auth routes (no permission check)
	v1.GET("/auth/me", ad.adminHandler.Me)

	// Helper: permission-scoped sub-group
	permGroup := func(resource string) *gin.RouterGroup {
		g := v1.Group("")
		g.Use(admin.RequireResourcePermission(ad.roleRepo, resource))
		return g
	}

	// ----- Device routes → resource "devices" -----
	dh := c.deviceHandlerDeps
	deviceHandler := device.NewHandler(c.DeviceService)
	deviceHandler.SetPermissionService(c.PermService)
	deviceHandler.RegisterRoutes(permGroup("devices"))

	deviceInfoHandler := device.NewDeviceInfoHandler(c.DeviceService)
	deviceInfoHandler.RegisterRoutes(permGroup("devices"))

	regHandler := device.NewRegistrationHandler(dh.regService)
	regHandler.RegisterRoutes(permGroup("devices"))

	columnConfigRepo := device.NewPgColumnConfigRepository(c.PgPool)
	columnConfigHandler := device.NewColumnConfigHandler(columnConfigRepo)
	columnConfigHandler.RegisterRoutes(permGroup("devices"))

	exportService := device.NewExportService(c.DeviceInfoRepo, c.Logger)
	exportHandler := device.NewExportHandler(exportService)
	exportHandler.SetPermissionService(c.PermService)
	exportHandler.RegisterRoutes(permGroup("devices"))

	paramTreeHandler := device.NewParameterTreeHandler(c.DeviceService, c.ParamRepo, c.DMRegistry, c.Logger)
	paramTreeHandler.RegisterRoutes(permGroup("devices"))

	// ----- Config routes → resource "datamodels" -----
	ch := c.configHandlerDeps
	dmHandler := datamodel.NewHandler(ch.dmRepo, ch.ouiRepo, ch.dmRegistry, ch.dmImporter)
	dmHandler.RegisterRoutes(permGroup("datamodels"))

	// ----- ConfigTemplate routes → resource "config" -----
	templateHandler := template.NewHandler(ch.templateRepo)
	templateHandler.RegisterRoutes(permGroup("config"))

	// ----- Provisioning routes → resource "config" -----
	md := c.miscDeps
	provisionHandler := provision.NewHandler(md.provisionRepo, md.provisionEngine)
	provisionHandler.RegisterRoutes(permGroup("config"))

	// ----- Topology routes → resource "devices" -----
	th := c.topologyHandlerDeps
	topologyHandler := topology.NewHandler(th.groupRepo, th.groupService, th.siteRepo, th.topoNodeRepo, th.topoEdgeRepo)
	topologyHandler.RegisterRoutes(permGroup("devices"))

	// ----- Device Rules routes → resource "devices" -----
	md.ruleHandler.RegisterRoutes(permGroup("devices"))

	// ----- PM routes → resource "pm" -----
	ph := c.pmHandlerDeps
	pmHandler := pm.NewHandler(ph.pmCounterRepo, ph.pmKPIRepo, ph.pmKPIEngine, ph.pmTaskRepo, ph.pmFileStore, c.MinIO, c.Cfg.MinIO.Buckets.PMFiles, c.Logger)
	pmHandler.SetMetrics(pm.NewPMMetrics(c.MetricsReg))
	pmHandler.RegisterRoutes(permGroup("pm"))

	// ----- Alarm routes → resource "alarms" -----
	ah := c.alarmHandlerDeps
	alarmHandler := alarm.NewHandler(c.AlarmEngine, ah.alarmPgStore, c.Logger)
	alarmHandler.RegisterRoutes(permGroup("alarms"))

	// ----- Alarm rule routes → resource "alarms" -----
	alarmRuleHandler := alarm.NewRuleHandler(md.alarmRuleRepo, c.Logger)
	alarmsGroup := permGroup("alarms").Group("/alarms")
	alarmRuleHandler.RegisterRoutes(alarmsGroup)

	// ----- KPI threshold routes → resource "pm" -----
	thresholdHandler := pm.NewThresholdHandler(md.thresholdRepo, c.Logger)
	pmGroup := permGroup("pm").Group("/pm")
	thresholdHandler.RegisterRoutes(pmGroup)

	// ----- Dashboard routes → resource "devices" -----
	md.dashboardHandler.RegisterRoutes(permGroup("devices"))

	// ----- Syslog routes → resource "devices" -----
	md.syslogHandler.RegisterRoutes(permGroup("devices"))

	// ----- Config sync routes → resource "config" -----
	md.syncHandler.RegisterRoutes(permGroup("config"))

	// ----- MR routes → resource "pm" -----
	mrHandler := mr.NewHandler(md.mrStore, md.mrIndRepo, md.mrMapRepo, c.MinIO, c.Cfg.MinIO.Buckets.MRFiles, c.Logger)
	mrHandler.RegisterRoutes(permGroup("pm"))

	// ----- Software routes → resource "firmware" -----
	md.softwareHandler.RegisterRoutes(permGroup("firmware"))

	// ----- Task routes → resource "devices" -----
	md.taskHandler.RegisterRoutes(permGroup("devices"))

	// ----- Interop routes → resource "interop" -----
	md.interopHandler.RegisterRoutes(permGroup("interop"))

	// ----- Backup routes → resource "devices" -----
	md.backupHandler.RegisterRoutes(permGroup("devices"))

	// ----- File Manager routes → resource "devices" -----
	md.fileHandler.RegisterRoutes(permGroup("devices"))

	// ----- MML Console routes → resource "devices" -----
	md.mmlHandler.RegisterRoutes(permGroup("devices"))

	// ----- Config Baseline routes → resource "config" -----
	md.baselineHandler.RegisterRoutes(permGroup("config"))

	// ----- License routes → resource "devices" -----
	md.licenseHandler.RegisterRoutes(permGroup("devices"))

	// ----- OpsTools routes → resource "devices" -----
	md.opsHandler.RegisterRoutes(permGroup("devices"))

	// ----- Report routes → resource "pm" -----
	md.reportHandler.RegisterRoutes(permGroup("pm"))

	// ----- System Info endpoint → resource "devices", read only -----
	sysInfoGroup := v1.Group("")
	sysInfoGroup.Use(admin.RequirePermission(ad.roleRepo, "devices", "read"))
	sysInfoGroup.GET("/system/info", md.sysInfoHandler.GetSystemInfo)

	// ----- Northbound routes → resource "northbound" -----
	md.nbRouter.RegisterRoutes(permGroup("northbound"))

	// ----- API Key management routes (authenticated users) -----
	ad.apiKeyHandler.RegisterRoutes(v1)

	// ----- Admin management routes (require admin permission) -----
	adminGroup := v1.Group("/admin")
	adminGroup.Use(admin.RequirePermission(ad.roleRepo, "users", "admin"))
	ad.adminHandler.RegisterAdminRoutes(adminGroup)

	c.Logger.Info("all routes registered")
	return nil
}

