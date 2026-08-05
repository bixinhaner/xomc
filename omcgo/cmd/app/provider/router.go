package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentruntime"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/bundle"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/components"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mr"
	mrtask "github.com/omcgo/omcgo/internal/mr/task"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/quicksettings"
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
		Name: "device",
		// Phase 6 (设计文档 §4.3)：device.applyProductMetadata 需注入 ProductRegistry
		// 才能回填 devices.model_name；ModuleGraph 必须保证 productregistry 先于
		// device 初始化。漏掉此依赖会让 SetProductMatcher 跳过 → productMatcher=nil
		// → applyProductMetadata 第一行 return → 永远不调 MatchProductClass。
		Depends: []string{"topology", "productregistry", "alarm"},
		Init:    func() error { return initDeviceModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "alarm",
		Depends: []string{"productregistry"},
		Init:    func() error { return initAlarmModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "alarm-retention",
		Depends: []string{"admin", "alarm"},
		Init:    func() error { return initAlarmRetentionModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "pm",
		// T-0164-P1：KPIEngine 现在依赖 ProductRegistry / DeviceRepo / IndicatorRepo
		// 三者构造 KPI Router。productregistry / device 必须先就绪。
		// #241：indicator FileHandler 需 c.DictService(导入后刷 kpi_platform_enb 字典)→ Depends admin。
		Depends: []string{"productregistry", "device", "admin"},
		Init:    func() error { return initPMModule(c) },
	})
	// T-0164-P2 / G2 PM 保留策略层（sys_configs 5 键 + SavedHook reload）
	// 依赖 admin（拿 SysConfigSvc 挂 hook），pm 仅占位保证 PM 主模块先初始化。
	graph.Add(components.ModuleInitializer{
		Name:    "pm-retention",
		Depends: []string{"admin", "pm"},
		Init:    func() error { return initPMRetentionModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "minio-ilm",
		Depends: []string{"admin"},
		Init:    func() error { return initMinIOILMModule(c) },
	})
	// issue #548 切片 2 · D 后端：MinIO public_endpoint 运行时订阅桥。
	// 依赖 admin（拿 SysConfigSvc 挂 SavedHook / Validator）。
	// 消费方（trace / 后续 mml/license/backup）需 Depends 上 minio-presign-bridge
	// 才能保证 c.PresignBridge 已就绪——misc 模块当前装配 trace，于下方追加该依赖。
	graph.Add(components.ModuleInitializer{
		Name:    "minio-presign-bridge",
		Depends: []string{"admin"},
		Init:    func() error { return initMinIOPresignBridgeModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "log-rotation",
		Depends: []string{"admin"},
		Init:    func() error { return initLogRotationModule(c) },
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
		Name:    "paramsync",
		Depends: []string{"device", "task", "paramregistry", "productregistry"},
		Init:    func() error { return initParamSyncModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "provision",
		Depends: []string{"device", "config", "paramsync"},
		Init:    func() error { return initProvisionModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "task",
		Depends: []string{"device"},
		Init:    func() error { return initTaskModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "backup",
		// 依赖 software：FilePathRecorder 装配时要拿 c.miscDeps.softwareService 作为
		// FileLandedNotifier（FAULT_LOG_COLLECT 文件落地即完成的 hook 回调）；不声明依赖
		// 时 backup 跑得比 software 早，softwareService 还是 nil，hook 永远拿不到通知。
		// 依赖 ufte (T-0164)：backup.RestoreService 反向注入到 ufte.Service 作为
		// CONFIG_RESTORE 派发器；ufte 必须先就绪。
		Depends: []string{"software", "ufte"},
		Init:    func() error { return initBackupModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "stationlog",
		Depends: []string{"device"},
		Init:    func() error { return initStationLogModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "eventlog",
		Depends: []string{"device"},
		Init:    func() error { return initEventLogModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "rebootrecord",
		Depends: []string{"eventlog", "stationlog"},
		Init:    func() error { return initRebootRecordModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "dashboard",
		Depends: []string{"device", "alarm", "pm", "topology", "misc"},
		Init:    func() error { return initDashboardModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name: "northbound",
		// 依赖 device + admin：northbound 数据导出要复用 DeviceService（按 SN/ID 校验设备归属）
		// 与 PermissionService（解析用户可见设备组）做多租户隔离，二者必须先就绪。
		Depends: []string{"alarm", "pm", "device", "admin"},
		Init:    func() error { return initNorthboundModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "interop",
		Depends: []string{"device", "config"},
		Init:    func() error { return initInteropModule(c) },
	})
	graph.Add(components.ModuleInitializer{
		Name:    "misc",
		Depends: []string{"task", "admin", "minio-presign-bridge"},
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
		Name: "paramregistry",
		// #241：parammodel Handler 需 c.DictService(导入后刷 param_model_name 字典)→ Depends admin。
		Depends: []string{"dictload", "productregistry", "admin"},
		Init:    func() error { return initParamRegistryModule(c) },
	})
	// T-0098 P3-04：AlarmDefinition Registry + Service + Handler — REST API 入口装配。
	// 依赖 dictload 完成后 alarm_definitions / alarm_severity_levels 表已写入。
	graph.Add(components.ModuleInitializer{
		Name: "alarmdef",
		// #241：alarmdef FileHandler 需 c.DictService(导入后刷 alarm_ne_type 字典)→ Depends admin。
		Depends: []string{"dictload", "admin"},
		Init:    func() error { return initAlarmDefModule(c) },
	})
	// F05 MR Task management (PRD docs/project/prd/F05-mr-task-management.md)
	// 依赖：mr（mrStore 给 cleaner 用）、paramregistry + productregistry（dispatcher 翻译）、
	//      device、task、misc（CompletionRouter）
	graph.Add(components.ModuleInitializer{
		Name:    "mrtask",
		Depends: []string{"mr", "paramregistry", "productregistry", "device", "task", "misc"},
		Init:    func() error { return initMRTaskModule(c) },
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
	if c.SysConfigSvc != nil {
		stopRetry := c.SysConfigSvc.StartApplyRetry(15 * time.Second)
		if c.GS != nil {
			c.GS.Register("sys-config-apply-retry", 1, func(context.Context) error {
				stopRetry()
				return nil
			})
		}
	}

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
	r.Use(middleware.Locale()) // 解析 Accept-Language → ctx，供指标名 / 内置任务名按语言本地化
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
	ad.agentConfigHandler.RegisterPublicRoutes(publicV1)

	// Protected API v1 routes (JWT or API Key authentication required)
	v1 := r.Group("/api/v1")
	v1.Use(admin.RequireAuthWithAPIKey(ad.jwtService, ad.apiKeySvc, ad.userRepo, ad.roleRepo, ad.tokenRevoker))
	// v1.0：admin.RequireCarrier() 已删除（users.carrier 已移除，详见 docs/prd/system/users.md §11.11）
	v1.Use(admin.AuditLogger(ad.auditRepo))
	// #122：OperLogger 写 sys_oper_logs（管理面写操作的可分页运维视图，
	// 与 audit_logs 合规链并存）。fire-and-forget，写失败仅 Warn 不阻塞请求。
	v1.Use(admin.OperLogger(ad.logRepo, c.Logger.Named("oper-log")))

	// Protected auth routes (no permission check)
	v1.GET("/auth/me", ad.adminHandler.Me)
	v1.POST("/auth/switch-role", ad.adminHandler.SwitchRole)
	v1.GET("/auth/menus", ad.adminHandler.GetUserMenusByRole)
	// Authenticated user routes (any authenticated user)
	ad.adminHandler.RegisterAuthenticatedRoutes(v1)

	agentRuntimeHandler := agentruntime.NewHandler(
		ad.agentConfigHandler.Service(),
		c.JWTService,
		http.DefaultClient,
		c.Logger,
		r,
		r,
		agentruntime.NewConversationService(admin.NewPgSysConfigRepository(c.PgPool)),
	)
	ad.agentConfigHandler.RegisterRuntimeRoutes(v1)
	agentRuntimeHandler.RegisterRoutes(v1)

	// Helper: authenticated sub-group.
	//
	// RequireAPIPermission 按标准路由模板 + HTTP 方法执行端点级 Casbin 鉴权。
	// resource 参数保留供现有调用点表达业务归属；实际权限键由路由模板和方法决定。
	permGroup := func(_ string) *gin.RouterGroup {
		g := v1.Group("")
		g.Use(admin.RequireAPIPermission(ad.roleRepo))
		return g
	}

	// ----- Device routes → resource "devices" -----
	dh := c.deviceHandlerDeps
	deviceHandler := device.NewHandler(c.DeviceService)
	deviceHandler.SetPermissionService(c.PermService)
	locationSyncRepo := device.NewPgLocationObservationRepository(c.PgPool)
	deviceHandler.SetLocationSyncService(device.NewLocationSyncService(locationSyncRepo))
	deviceHandler.RegisterRoutes(permGroup("devices"))
	if c.miscDeps.paramSyncHandler != nil {
		c.miscDeps.paramSyncHandler.RegisterRoutes(permGroup("devices"))
	}

	deviceInfoHandler := device.NewDeviceInfoHandler(c.DeviceService)
	deviceInfoHandler.SetPermissionService(c.PermService)
	deviceInfoHandler.RegisterRoutes(permGroup("devices"))

	regHandler := device.NewRegistrationHandler(dh.regService)
	regHandler.SetPermissionService(c.PermService) // #64 设备组可见性数据权限
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
	paramTreeHandler.SetPermissionService(c.PermService)
	paramTreeHandler.RegisterRoutes(permGroup("devices"))

	// T-0179 + T-0183: devsweep HTTP 端点已下线 — omcctl device sweep-paths
	// 改成进程内直连 PG/Redis,在 CLI 进程内调用 devsweep.Service + Applier。
	// 见 cmd/omcctl/device_sweep.go。

	// T-0138：「快速设置」分组元数据（设备运维人员可读）
	if c.QuickSettingsRegistry != nil && c.DeviceService != nil && c.ProductRegistry != nil && c.ProductRepo != nil {
		quickSettingsHandler := quicksettings.NewHandler(
			c.QuickSettingsRegistry,
			c.DeviceService,
			c.ProductRegistry,
			c.ProductRepo,
		)
		quickSettingsHandler.RegisterRoutes(permGroup("devices"))
	}

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
	// issue #126 item 10: 建任务前预检设备存在性，避免 device_id 合法但不存在
	// 时落库孤儿任务（provisioning_tasks 对 device_id 无外键）。
	if c.DeviceService != nil {
		provisionHandler.SetDeviceChecker(c.DeviceService)
	}
	provisionHandler.RegisterRoutes(permGroup("config"))

	// ----- Topology routes → resource "devices" -----
	th := c.topologyHandlerDeps
	topologyHandler := topology.NewHandler(th.groupRepo, th.groupService, th.siteRepo, th.topoNodeRepo, th.topoEdgeRepo, th.syncSvc, th.logger)
	topologyHandler.SetPermissionService(c.PermService)
	topologyHandler.RegisterRoutes(permGroup("devices"))

	// ----- PM routes → resource "pm" -----
	ph := c.pmHandlerDeps
	pmHandler := pm.NewHandler(ph.pmCounterRepo, ph.pmKPIRepo, ph.pmKPIEngine, ph.pmTaskRepo, ph.pmFileStore, c.MinIO, c.Cfg.MinIO.Buckets.PMFiles, c.TsPool, ph.pmIndicatorRepo, c.Logger).
		WithAggregator(ph.pmAggregator).
		WithAsyncJobRepo(ph.pmAsyncJobRepo).
		WithTimezoneProvider(c.SystemTimezone)
	pmHandler.SetMetrics(pm.NewPMMetrics(c.MetricsReg))
	pmHandler.SetPermissionService(c.PermService) // #64 设备组可见性数据权限
	if c.ProductRegistry != nil {
		pmHandler.SetProductPatternResolver(c.ProductRegistry) // #602 按产品名称下拉过滤
	}
	pmHandler.RegisterRoutes(permGroup("pm"))
	// T-0164-P7 / G7：adhoc 自定义聚合任务 REST 路由（同 pm 权限组）
	if ph.pmAdhocHandler != nil {
		ph.pmAdhocHandler.RegisterRoutes(permGroup("pm"))
	}
	// T-0174 阶段 1：指标查询模板 REST 路由（5 CRUD，同 pm 权限组）
	if ph.pmQueryTemplateHandler != nil {
		ph.pmQueryTemplateHandler.RegisterRoutes(permGroup("pm"))
	}
	// KPI-EXPORT T1：KPI 导出 REST 路由（建任务/列任务/列文件/下载/删除，同 pm 权限组）
	if ph.pmExportHandler != nil {
		ph.pmExportHandler.RegisterRoutes(permGroup("pm"))
	}

	// ----- Alarm routes → resource "alarms" -----
	ah := c.alarmHandlerDeps
	alarmHandler := alarm.NewHandler(c.AlarmEngine, ah.alarmPgStore, ah.alarmSyncService, c.Logger)
	alarmHandler.SetPermissionService(c.PermService) // #64 设备组可见性数据权限
	alarmHandler.RegisterRoutes(permGroup("alarms"))

	// T-0098-P5-06：旧 /alarms/alarm-libraries 路由已下线，治理走 /alarms/alarm-definitions（super_admin）。

	// ----- T-0098 P3-05: Super-admin-only group — 仅放行 super_admin 用户。
	// 4 个 P3 治理 handler（产品 / 参数模型 / KPI 库 / 告警库）均挂在此处，
	// admin / operator / viewer 一律 403。
	superAdminGroup := v1.Group("")
	superAdminGroup.Use(admin.RequireSuperAdmin())

	// ----- T-0098 P3-04: Alarm Definitions routes → super_admin only -----
	if c.AlarmDefHandler != nil {
		c.AlarmDefHandler.RegisterReadRoutes(v1)
		c.AlarmDefHandler.RegisterWriteRoutes(superAdminGroup)
	}
	// 告警自定义 XML 上传/删除(严格对标 indicator FileHandler)
	if c.AlarmDefFileHandler != nil {
		c.AlarmDefFileHandler.RegisterRoutes(superAdminGroup)
	}

	// ----- T-0098 P3-02: ParamModel routes → super_admin only -----
	if c.ParamModelHandler != nil {
		c.ParamModelHandler.RegisterReadRoutes(v1)
		c.ParamModelHandler.RegisterWriteRoutes(superAdminGroup)
	}

	// ----- T-0098 P3-01: Product routes → super_admin only -----
	if c.ProductHandler != nil {
		c.ProductHandler.RegisterReadRoutes(v1)
		c.ProductHandler.RegisterWriteRoutes(superAdminGroup)
	}

	// ----- Sprint B Q-V3-7：dictload admin reload 端点（含 mml-standard）-----
	registerDictLoadAdminRoutes(c, superAdminGroup)
	// #41：KPI 路由人工恢复入口，与 indicator reload 复用同一失效器。
	registerKPIRouteAdminRoutes(c.KPIRouteInvalidator, c.Logger.Named("kpi-route-admin"), superAdminGroup)

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
		ph.indicatorRESTHandler.RegisterReadRoutes(v1)
		ph.indicatorRESTHandler.RegisterWriteRoutes(superAdminGroup)
	}

	// ----- T-0180 P1.3: Indicator XML file management (DELETE 守门) → super_admin only -----
	if ph.indicatorFileHandler != nil {
		ph.indicatorFileHandler.RegisterRoutes(superAdminGroup)
	}

	// ----- Dashboard routes → resource "devices" -----
	md.dashboardHandler.SetPermissionService(c.PermService)
	md.dashboardHandler.RegisterRoutes(permGroup("devices"))

	// ----- Syslog routes → resource "devices" -----
	md.syslogHandler.RegisterRoutes(permGroup("devices"))

	// ----- Config sync routes → resource "config" -----
	md.syncHandler.RegisterRoutes(permGroup("config"))

	// ----- MR routes → resource "pm" -----
	mrHandler := mr.NewHandler(md.mrStore, md.mrIndRepo, md.mrMapRepo, c.MinIO, c.Cfg.MinIO.Buckets.MRFiles, c.Logger)
	if c.ProductRegistry != nil {
		mrHandler.SetProductPatternResolver(c.ProductRegistry) // #602 按产品名称下拉过滤
	}
	mrHandler.RegisterRoutes(permGroup("pm"))

	// ----- MR Task management (F05 测量任务) → resource "pm" -----
	// 复用 initMRTaskModule 已构造的 repo（dispatcher / scheduler / heartbeat / cleaner 共享）。
	if md.mrTaskRepo != nil {
		mrTaskSvc := mrtask.NewService(md.mrTaskRepo, c.Logger)
		mrtask.NewHandler(mrTaskSvc, c.Logger).RegisterRoutes(permGroup("pm"))
	}

	// ----- Software routes → resource "firmware" -----
	md.softwareHandler.SetPermissionService(c.PermService) // #59 升级/回退创建逐设备归属校验
	md.softwareHandler.RegisterRoutes(permGroup("firmware"))
	md.ufteHandler.SetPermissionService(c.PermService) // #63 设备组可见性数据权限
	md.ufteHandler.RegisterRoutes(permGroup("firmware"))

	// ----- Task routes → resource "devices" -----
	md.taskHandler.RegisterRoutes(permGroup("devices"))

	// ----- Interop routes → resource "interop" -----
	md.interopHandler.SetPermissionService(c.PermService) // #63 设备组可见性数据权限
	md.interopHandler.RegisterRoutes(permGroup("interop"))

	// ----- Backup routes → resource "devices" -----
	md.backupHandler.RegisterRoutes(permGroup("devices"))

	// ----- 文件管理 4 Tab 批量下载（bundle 模块,同步流式） -----
	// 每个模块 POST /<module>/batch-download 挂在各自资源下,鉴权独立。
	// handler 直接流 zip 到 response writer,浏览器一次下载。
	if md.bundleSvc != nil {
		// #63 设备组可见性：批量下载共用一个 Resolver，handler 内按调用者可见组过滤。
		bundleResolver := authz.NewResolver(c.PermService)
		permGroup("firmware").POST("/firmware/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModuleFirmware, bundleResolver))

		bkGrp := permGroup("devices")
		bkGrp.POST("/backup/config-snapshots/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModuleConfigSnapshot, bundleResolver))
		bkGrp.POST("/backup/device-licenses/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModuleDeviceLicense, bundleResolver))

		permGroup("pm").POST("/mr/files/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModuleMR, bundleResolver))
		// 按 file_id 粒度打包(DeviceFilesDrawer 抽屉用,跟按设备整盘下载语义不同)。
		permGroup("pm").POST("/mr/files/by-id/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModuleMRFiles, bundleResolver))

		// PM 批量下载（按设备 SN / 按 file_id），跟 MR 同构。
		permGroup("pm").POST("/pm/files/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModulePM, bundleResolver))
		permGroup("pm").POST("/pm/files/by-id/batch-download",
			bundle.NewBatchDownloadHandler(md.bundleSvc, bundle.ModulePMFiles, bundleResolver))
	}

	// ----- Station Log routes → resource "devices" -----
	if md.stationlogHandler != nil {
		md.stationlogHandler.SetPermissionService(c.PermService) // #63 设备组可见性数据权限
		md.stationlogHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- EventLog routes → resource "devices" -----
	if md.eventlogHandler != nil {
		md.eventlogHandler.SetPermissionService(c.PermService) // #63 设备组可见性数据权限
		md.eventlogHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- RebootRecord routes（统一重启记录）→ resource "devices" -----
	if md.rebootrecordHandler != nil {
		md.rebootrecordHandler.SetPermissionService(c.PermService) // #63 设备组可见性数据权限
		md.rebootrecordHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- File Manager routes → resource "devices" -----
	md.fileHandler.RegisterRoutes(permGroup("devices"))

	// ----- MML Console routes → resource "devices" -----
	md.mmlHandler.RegisterRoutes(permGroup("devices"))
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

	// ----- T-0137 / M1: TR069 报文跟踪 → resource "devices" -----
	if md.traceHandler != nil {
		md.traceHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- W2.A.4 / T-0043: Notification template + history → resource "alarms" -----
	// 路径前缀 /notifications，handler 内部挂 /templates 和 /history 子路由：
	//   final paths: /api/v1/notifications/templates[...] + /api/v1/notifications/history[...]
	notifGroup := permGroup("alarms").Group("/notifications")
	md.notifTemplateHandler.RegisterReadOnlyRoutes(notifGroup)
	md.notifHistoryHandler.RegisterRoutes(notifGroup)
	// 通知中心管理面使用根级版本化 API。所有写操作同时经过 endpoint RBAC、
	// audit_logs 与 sys_oper_logs；handler 额外执行 If-Match 并发前置条件。
	if md.notifRuleHandler != nil {
		md.notifRuleHandler.RegisterRoutes(permGroup("alarms"))
	}
	if md.notifContactGroupHandler != nil {
		md.notifContactGroupHandler.RegisterRoutes(permGroup("alarms"))
	}
	if md.notifManagedTemplateHandler != nil {
		md.notifManagedTemplateHandler.RegisterRoutes(permGroup("alarms"))
	}
	if md.notifChannelHandler != nil {
		md.notifChannelHandler.RegisterRoutes(permGroup("alarms"))
	}
	if md.notifDeliveryHandler != nil {
		md.notifDeliveryHandler.RegisterRoutes(permGroup("alarms"))
	}

	// ----- T-0152: Alertmanager 告警 webhook → publicV1（无 JWT）-----
	// Alertmanager 无法携带 JWT，故挂在无鉴权的 publicV1 上；可选 Bearer token
	// 校验由 notification.alert_webhook.token 配置。最终路径 /api/v1/alerts/webhook。
	if md.alertWebhookHandler != nil {
		md.alertWebhookHandler.RegisterRoutes(publicV1)
	}

	// ----- Config Baseline routes → resource "config" -----
	md.baselineHandler.RegisterRoutes(permGroup("config"))

	// ----- System License (singleton) routes → resource "devices" -----
	// F06 重构 Step 5：老 /licenses/* multi-license 路由已下线；本路由是唯一入口。
	if md.systemLicenseHandler != nil {
		md.systemLicenseHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- DeviceDetail "License 参数" tab routes → resource "devices" -----
	// GET  /api/v1/devices/:id/license-params          → ListLicenseParams
	// POST /api/v1/devices/:id/license-params/refresh  → 下发 GPV 刷新
	if md.licenseParamHandler != nil {
		md.licenseParamHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- OpsTools routes → resource "devices" -----
	md.opsHandler.RegisterRoutes(permGroup("devices"))
	if md.opsExtHandler != nil {
		md.opsExtHandler.RegisterRoutes(permGroup("devices"))
	}

	// ----- Report routes → resource "pm" -----
	md.reportHandler.RegisterRoutes(permGroup("pm"))

	// ----- System Info endpoint -----
	sysInfoGroup := v1.Group("")
	sysInfoGroup.Use(admin.RequireAPIPermission(ad.roleRepo))
	sysInfoGroup.GET("/system/info", md.sysInfoHandler.GetSystemInfo)

	// ----- Northbound routes → resource "northbound" -----
	md.nbRouter.RegisterRoutes(permGroup("northbound"))

	// ----- API Key management routes (authenticated users) -----
	ad.apiKeyHandler.RegisterRoutes(v1)

	// ----- Admin management routes -----
	// 独立路由组统一执行端点级 API 权限校验。
	adminGroup := v1.Group("/admin")
	adminGroup.Use(admin.RequireAPIPermission(ad.roleRepo))
	ad.adminHandler.RegisterAdminRoutes(adminGroup)

	// ----- Dictionary management routes (require admin permission) -----
	ad.dictHandler.RegisterRoutes(adminGroup)

	// ----- System config management routes (require admin permission) -----
	ad.sysConfigHandler.RegisterRoutes(adminGroup)
	ad.agentConfigHandler.RegisterAdminRoutes(adminGroup)

	// ----- UI 定制化：上传 Logo / 登录背景图（require admin permission）-----
	ad.uiAssetHandler.RegisterRoutes(adminGroup)

	// ----- System log routes (require admin permission) -----
	ad.logHandler.RegisterRoutes(adminGroup)
	if c.StorageProtectionHandler != nil {
		c.StorageProtectionHandler.RegisterRoutes(adminGroup)
	}

	// ----- Dead-letter queue admin routes (T-0012 / R-106) -----
	// adminGroup already enforces RBAC users:admin; dead-letter handler nests
	// under /admin via its own internal /admin/dead-letters group. Mount under
	// v1 directly to avoid double /admin prefix.
	dlqAdmin := v1.Group("")
	dlqAdmin.Use(admin.RequireAPIPermission(ad.roleRepo))
	if md.deadLetterHandler != nil {
		md.deadLetterHandler.RegisterRoutes(dlqAdmin)
	}

	// All optional modules have registered their routes at this point. Build the
	// immutable Agent handbook from this instance's actual route inventory.
	if err := agentRuntimeHandler.PrepareHandbook(); err != nil {
		c.Logger.Warn("agent handbook preparation failed",
			zap.Error(err),
		)
	}

	// Inject gin routes into admin handler for SyncApiEndpoints
	ad.adminHandler.SetGinRoutes(r.Routes())

	// Auto-sync API endpoints from Gin routes at startup
	if err := syncApiEndpoints(c, ad); err != nil {
		return fmt.Errorf("sync API endpoints and built-in permission baseline: %w", err)
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

// syncApiEndpoints auto-syncs Gin routes and the immutable built-in role
// permission baseline before the HTTP server starts. Failure is fatal so the
// process cannot advertise readiness while normal roles would receive 403s.
type apiEndpointSyncer interface {
	SyncApiEndpoints(context.Context, gin.RoutesInfo) (admin.SyncResult, error)
}

func syncApiEndpoints(c *Container, ad *adminHandlerDeps) error {
	timeout := 2 * time.Minute
	if raw := os.Getenv("OMCGO_API_ENDPOINT_SYNC_TIMEOUT_SECONDS"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	routes := ad.adminHandler.GetGinRoutes()
	if len(routes) == 0 {
		return nil
	}

	if ad.apiEndpointService == nil {
		return fmt.Errorf("API endpoint sync service is not configured")
	}

	result, err := ad.apiEndpointService.SyncApiEndpoints(ctx, routes)
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
