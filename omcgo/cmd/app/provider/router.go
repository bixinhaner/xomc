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
		Name:    "ufte",
		Depends: []string{"software", "device"},
		Init:    func() error { return initUFTEModule(c) },
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
		Name:    "misc",
		Depends: []string{"task", "admin"},
		Init:    func() error { return initMiscModules(c) },
	})
	// T-0098 P1-06：字典加载（4 域 paramModel/indicator/alarm-definition/product）
	// 无业务依赖，但 P1 schema 必须已迁移（migrate up 跑过 000057-000059）
	graph.Add(components.ModuleInitializer{
		Name: "dictload",
		Init: func() error { return initDictLoadModule(c) },
	})
	// T-0098 P2-01：ProductRegistry（productClass 全局正则路由 + L1+L2 缓存 + 三引用校验）
	// 依赖 dictload 完成后 products / alarm_definitions / perf_indicators_* 表已写入。
	graph.Add(components.ModuleInitializer{
		Name:    "productregistry",
		Depends: []string{"dictload"},
		Init:    func() error { return initProductRegistryModule(c) },
	})
	// T-0098 P2-02：ParamRegistry（按 productId/paramModelId 取映射 + Translator 双向翻译）
	// 依赖 productregistry 注入为 productGetter（反查 product.ParamModelID 供 default 降级）。
	graph.Add(components.ModuleInitializer{
		Name:    "paramregistry",
		Depends: []string{"dictload", "productregistry"},
		Init:    func() error { return initParamRegistryModule(c) },
	})
	// T-0098 P3-04：AlarmDefinition Registry + Service + Handler — REST API 入口装配。
	// 依赖 dictload 完成后 alarm_definitions / alarm_severity_levels 表已写入。
	graph.Add(components.ModuleInitializer{
		Name:    "alarmdef",
		Depends: []string{"dictload"},
		Init:    func() error { return initAlarmDefModule(c) },
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
	// RateLimit: per-IP token bucket，防止单 IP 洪泛拖垮后端。
	// 默认 100 req/s, burst 200；放行 /healthz、/readyz、/metrics 探针。
	// 未来如需配置化可在 AppConfig 中新增 ratelimit 段位。
	r.Use(middleware.RateLimit(middleware.RateLimitConfig{
		RatePerSecond: 100,
		Burst:         200,
		Registerer:    c.MetricsReg,
		Skipper: func(gc *gin.Context) bool {
			p := gc.Request.URL.Path
			return p == "/healthz" || p == "/readyz" || p == "/metrics"
		},
	}))
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
	ad.adminHandler.RegisterAuthRoutes(publicV1, ad.pubKeyHandler)

	// 公开端点：登录页拉取品牌化资产 + 公开配置项（is_public=true）
	// 详见 docs/prd/system/ui-customization.md §6
	ad.sysConfigHandler.RegisterPublicRoutes(publicV1)
	ad.uiAssetHandler.RegisterPublicRoutes(publicV1)

	// Protected API v1 routes (JWT or API Key authentication required)
	v1 := r.Group("/api/v1")
	v1.Use(admin.RequireAuthWithAPIKey(ad.jwtService, ad.apiKeySvc, ad.userRepo, ad.roleRepo, ad.tokenRevoker))
	// v1.0：admin.RequireCarrier() 已删除（users.carrier 已移除，详见 docs/prd/system/users.md §11.11）
	v1.Use(admin.AuditLogger(ad.auditRepo))

	// Protected auth routes (no permission check)
	v1.GET("/auth/me", ad.adminHandler.Me)
	v1.POST("/auth/switch-role", ad.adminHandler.SwitchRole)
	v1.GET("/auth/menus", ad.adminHandler.GetUserMenusByRole)
	// Authenticated user routes (any authenticated user)
	ad.adminHandler.RegisterAuthenticatedRoutes(v1)

	// Helper: permission-scoped sub-group.
	//
	// B3-Phase2（参 docs/prd/system/menu-dynamic-loading.md §4.2.4）：
	// 内部从粗粒度 RequireResourcePermission(resource) 切换为端点级
	// RequireAPIPermission（按 c.Request.URL.Path + Method 鉴权，对齐 GVA 风格）。
	// resource 参数保留供 30+ 调用点签名兼容，新版被忽略；helper 名留作"受保护
	// 路由组"语义提示。
	permGroup := func(_ string) *gin.RouterGroup {
		g := v1.Group("")
		g.Use(admin.RequireAPIPermission(ad.roleRepo))
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

	// T-0098 P5-01：dmRegistry 已删除，直接注入 ParamRegistry / ProductRegistry。
	paramTreeHandler := device.NewParameterTreeHandler(c.DeviceService, c.ParamRepo, c.ParamRegistry, c.ProductRegistry, c.Logger)
	paramTreeHandler.RegisterRoutes(permGroup("devices"))

	// T-0098-P5-01：旧 /api/v1/datamodels CRUD 已下线，治理走 /api/v1/products + /api/v1/param-models（super_admin）。

	// ----- ConfigTemplate routes → resource "config" -----
	ch := c.configHandlerDeps
	md := c.miscDeps
	templateHandler := template.NewHandler(ch.templateRepo)
	// T-0120: 接 provisioning engine 给 POST /:id/dispatch 提供 Path A 显式下发能力。
	templateHandler.SetDispatcher(md.provisionEngine)
	templateHandler.RegisterRoutes(permGroup("config"))

	// ----- Provisioning routes → resource "config" -----
	provisionHandler := provision.NewHandler(md.provisionRepo, md.provisionEngine)
	provisionHandler.RegisterRoutes(permGroup("config"))

	// ----- Topology routes → resource "devices" -----
	th := c.topologyHandlerDeps
	topologyHandler := topology.NewHandler(th.groupRepo, th.groupService, th.siteRepo, th.topoNodeRepo, th.topoEdgeRepo, th.syncSvc, th.logger)
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
	alarmHandler := alarm.NewHandler(c.AlarmEngine, ah.alarmPgStore, ah.alarmSyncService, c.Logger)
	alarmHandler.RegisterRoutes(permGroup("alarms"))

	// T-0098-P5-06：旧 /alarms/alarm-libraries 路由已下线，治理走 /alarms/alarm-definitions（super_admin）。

	// ----- T-0098 P3-05: Super-admin-only group — 仅放行 super_admin 用户。
	// 4 个 P3 治理 handler（产品 / 参数模型 / KPI 库 / 告警库）均挂在此处，
	// admin / operator / viewer 一律 403。
	superAdminGroup := v1.Group("")
	superAdminGroup.Use(admin.RequireSuperAdmin())

	// ----- T-0098 P3-04: Alarm Definitions routes → super_admin only -----
	if c.AlarmDefHandler != nil {
		c.AlarmDefHandler.RegisterRoutes(superAdminGroup)
	}

	// ----- T-0098 P3-02: ParamModel routes → super_admin only -----
	if c.ParamModelHandler != nil {
		c.ParamModelHandler.RegisterRoutes(superAdminGroup)
	}

	// ----- T-0098 P3-01: Product routes → super_admin only -----
	if c.ProductHandler != nil {
		c.ProductHandler.RegisterRoutes(superAdminGroup)
	}

	// ----- Sprint B Q-V3-7：dictload admin reload 端点（含 mml-standard）-----
	registerDictLoadAdminRoutes(c, superAdminGroup)

	// ----- Alarm filter rule routes → resource "alarms" -----
	alarmFilterHandler := alarm.NewFilterHandler(ah.alarmFilterRuleRepo, c.Logger)
	alarmFilterHandler.RegisterRoutes(permGroup("alarms").Group("/alarms/alarm-filters"))

	// ----- KPI threshold routes → resource "pm" -----
	thresholdHandler := pm.NewThresholdHandler(md.thresholdRepo, c.Logger)
	pmGroup := permGroup("pm").Group("/pm")
	thresholdHandler.RegisterRoutes(pmGroup)

	// ----- Indicator management routes → resource "pm" -----
	ph.indicatorHandler.RegisterRoutes(permGroup("pm"))

	// ----- T-0098 P3-03: Indicator REST routes → super_admin only -----
	if ph.indicatorRESTHandler != nil {
		ph.indicatorRESTHandler.RegisterRoutes(superAdminGroup)
	}

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
	md.ufteHandler.RegisterRoutes(permGroup("firmware"))

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
	md.paramHandler.RegisterRoutes(permGroup("devices"))
	// T-0123-P0：MML Catalog Admin (13 endpoints) → resource "devices"（端点级 Casbin 鉴权进一步细分）
	if md.mmlAdminHandler != nil {
		md.mmlAdminHandler.RegisterRoutes(permGroup("devices"))
	}
	// T-0123-P1：MML Console 5 端点（group-tree / sub-fields / render / parse / execute-statements）
	if md.mmlConsoleHandler != nil {
		md.mmlConsoleHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- SSE Stream endpoint (authenticated users, no permission check) -----
	md.sseHandler.RegisterRoutes(v1)

	// ----- Notifications → resource "devices" -----
	md.notificationHandler.RegisterRoutes(permGroup("devices"))

	// ----- W2.A.4 / T-0043: Notification template + history → resource "alarms" -----
	// 路径前缀 /notifications，handler 内部挂 /templates 和 /history 子路由：
	//   final paths: /api/v1/notifications/templates[...] + /api/v1/notifications/history[...]
	notifGroup := permGroup("alarms").Group("/notifications")
	md.notifTemplateHandler.RegisterRoutes(notifGroup)
	md.notifHistoryHandler.RegisterRoutes(notifGroup)

	// ----- Config Baseline routes → resource "config" -----
	md.baselineHandler.RegisterRoutes(permGroup("config"))

	// ----- License routes → resource "devices" -----
	md.licenseHandler.RegisterRoutes(permGroup("devices"))

	// ----- OpsTools routes → resource "devices" -----
	md.opsHandler.RegisterRoutes(permGroup("devices"))
	if md.opsExtHandler != nil {
		md.opsExtHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- Report routes → resource "pm" -----
	md.reportHandler.RegisterRoutes(permGroup("pm"))

	// ----- System Info endpoint -----
	// B3-Phase2：原 RequirePermission("devices","read") 改为端点级 RequireAPIPermission。
	sysInfoGroup := v1.Group("")
	sysInfoGroup.Use(admin.RequireAPIPermission(ad.roleRepo))
	sysInfoGroup.GET("/system/info", md.sysInfoHandler.GetSystemInfo)

	// ----- Northbound routes → resource "northbound" -----
	md.nbRouter.RegisterRoutes(permGroup("northbound"))

	// ----- API Key management routes (authenticated users) -----
	ad.apiKeyHandler.RegisterRoutes(v1)

	// ----- Admin management routes -----
	// B3-Phase2：原 RequirePermission("users","admin") 改为端点级 RequireAPIPermission。
	// 受保护粒度：/admin/* 下每个具体端点 path+method 单独鉴权。
	adminGroup := v1.Group("/admin")
	adminGroup.Use(admin.RequireAPIPermission(ad.roleRepo))
	ad.adminHandler.RegisterAdminRoutes(adminGroup)

	// ----- Dictionary management routes (require admin permission) -----
	ad.dictHandler.RegisterRoutes(adminGroup)

	// ----- System config management routes (require admin permission) -----
	ad.sysConfigHandler.RegisterRoutes(adminGroup)

	// ----- UI 定制化：上传 Logo / 登录背景图（require admin permission）-----
	ad.uiAssetHandler.RegisterRoutes(adminGroup)

	// ----- System log routes (require admin permission) -----
	ad.logHandler.RegisterRoutes(adminGroup)

	// ----- Dead-letter queue admin routes (T-0012 / R-106) -----
	// adminGroup already enforces RBAC users:admin; dead-letter handler nests
	// under /admin via its own internal /admin/dead-letters group. Mount under
	// v1 directly with the same admin permission to avoid double /admin prefix.
	// B3-Phase2：dlqAdmin 同 adminGroup 切换为端点级。
	dlqAdmin := v1.Group("")
	dlqAdmin.Use(admin.RequireAPIPermission(ad.roleRepo))
	if md.deadLetterHandler != nil {
		md.deadLetterHandler.RegisterRoutes(dlqAdmin)
	}

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
