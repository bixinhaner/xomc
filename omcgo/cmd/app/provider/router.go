package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/components"
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
//  1. 使用依赖图解析并初始化各模块
//  2. 配置 Gin 中间件
//  3. 注册所有路由
func Setup(r *gin.Engine, c *Container) error {
	// ===== Phase 1: 使用依赖图初始化各模块 =====

	graph := components.NewModuleGraph()

	// Register all modules with their dependency declarations.
	// The graph resolves correct initialization order automatically.
	graph.Add(components.ModuleInitializer{
		Name: "config",
		Init: func() error { return initConfigModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "topology",
		Init: func() error { return initTopologyModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "admin",
		Depends: []string{"topology"},
		Init:    func() error { return initAdminModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "device",
		Depends: []string{"topology"},
		Init:    func() error { return initDeviceModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "alarm",
		Init: func() error { return initAlarmModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "pm",
		Init: func() error { return initPMModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "mr",
		Init: func() error { return initMRModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "software",
		Depends: []string{"device"},
		Init:    func() error { return initSoftwareModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "provision",
		Depends: []string{"device", "config"},
		Init:    func() error { return initProvisionModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "task",
		Depends: []string{"device"},
		Init:    func() error { return initTaskModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "backup",
		Init: func() error { return initBackupModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "dashboard",
		Depends: []string{"device", "alarm", "pm", "topology"},
		Init:    func() error { return initDashboardModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "northbound",
		Depends: []string{"alarm", "pm"},
		Init:    func() error { return initNorthboundModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "interop",
		Depends: []string{"device", "config"},
		Init:    func() error { return initInteropModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "misc",
		Init: func() error { return initMiscModules(c) },
	})

	totalStart := time.Now()

	groups, err := graph.ParallelGroups()
	if err != nil {
		return err
	}

	moduleCount := 0
	for i, group := range groups {
		groupStart := time.Now()
		for _, name := range group {
			modStart := time.Now()
			m := graph.Module(name)
			if initErr := m.Init(); initErr != nil {
				c.Logger.Error("module init failed",
					zap.String("module", name),
					zap.Duration("duration", time.Since(modStart)),
					zap.Error(initErr),
				)
				return fmt.Errorf("init module %q: %w", name, initErr)
			}
			c.Logger.Info("module initialized",
				zap.String("module", name),
				zap.Duration("duration", time.Since(modStart)),
			)
			moduleCount++
		}
		c.Logger.Info("module group initialized",
			zap.Int("group", i+1),
			zap.Strings("modules", group),
			zap.Duration("duration", time.Since(groupStart)),
		)
	}
	c.Logger.Info("all modules initialized",
		zap.Int("module_count", moduleCount),
		zap.Duration("total_duration", time.Since(totalStart)),
	)

	// ===== Phase 2: 配置 Gin 中间件 =====
	setupMiddleware(r, c)

	// ===== Phase 3: 注册所有路由 =====
	routeStart := time.Now()
	if err := registerRoutes(r, c); err != nil {
		return err
	}
	c.Logger.Info("routes registered",
		zap.Duration("duration", time.Since(routeStart)),
	)
	return nil
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

	// /healthz — liveness probe: quick Ping check with timeout
	r.GET("/healthz", func(gc *gin.Context) {
		if c.Health != nil {
			ctx, cancel := context.WithTimeout(gc.Request.Context(), 2*time.Second)
			defer cancel()
			if !c.Health.IsHealthy(ctx) {
				gc.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
				return
			}
		}
		gc.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// /readyz — readiness probe: detailed component health with latency info
	r.GET("/readyz", func(gc *gin.Context) {
		if c.Health == nil {
			gc.JSON(http.StatusOK, gin.H{"status": "ok", "components": nil})
			return
		}
		ctx, cancel := context.WithTimeout(gc.Request.Context(), 5*time.Second)
		defer cancel()
		results := c.Health.CheckAll(ctx)
		gc.JSON(c.Health.StatusCode(ctx), gin.H{
			"status":     statusFromResults(results),
			"components": results,
		})
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
	v1.Use(admin.RequireAuthWithAPIKey(ad.jwtService, ad.apiKeySvc, ad.userRepo, ad.roleRepo))
	v1.Use(admin.RequireCarrier())
	v1.Use(admin.AuditLogger(ad.auditRepo))

	// Protected auth routes (no permission check)
	v1.GET("/auth/me", ad.adminHandler.Me)
	v1.POST("/auth/switch-role", ad.adminHandler.SwitchRole)
	v1.GET("/auth/menus", ad.adminHandler.GetUserMenusByRole)
	// Authenticated user routes (any authenticated user)
	ad.adminHandler.RegisterAuthenticatedRoutes(v1)

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

	// ----- Dictionary management routes (require admin permission) -----
	ad.dictHandler.RegisterRoutes(adminGroup)

	// ----- System config management routes (require admin permission) -----
	ad.sysConfigHandler.RegisterRoutes(adminGroup)

	// ----- System log routes (require admin permission) -----
	ad.logHandler.RegisterRoutes(adminGroup)

	// Inject gin routes into admin handler for SyncApiEndpoints
	ad.adminHandler.SetGinRoutes(r.Routes())

	// Auto-sync API endpoints from Gin routes at startup
	if err := syncApiEndpoints(c, ad); err != nil {
		c.Logger.Warn("api endpoint auto-sync failed, manual sync required",
			zap.Error(err),
		)
	}

	return nil
}

// statusFromResults derives overall status from component health results.
func statusFromResults(results []components.ComponentHealth) string {
	for _, r := range results {
		if r.Status != "healthy" {
			return "unhealthy"
		}
	}
	return "ok"
}

// syncApiEndpoints auto-syncs Gin routes into the api_endpoints table at startup.
// Only runs if api_endpointService is available; failures are logged but non-fatal.
func syncApiEndpoints(c *Container, ad *adminHandlerDeps) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	routes := ad.adminHandler.GetGinRoutes()
	if len(routes) == 0 {
		return nil
	}

	svc := admin.NewApiEndpointService(
		admin.NewPgApiEndpointRepository(c.PgPool),
		c.Logger,
	)
	result, err := svc.SyncApiEndpoints(ctx, routes)
	if err != nil {
		return err
	}

	c.Logger.Info("api endpoints auto-synced at startup",
		zap.Int("total", result.Total),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
	)
	return nil
}
