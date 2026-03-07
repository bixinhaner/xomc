package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/carrier/cucc"
	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/internal/common/middleware"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/infra"
	"github.com/omcgo/omcgo/internal/infra/cache"
	"github.com/omcgo/omcgo/internal/infra/db"
	"github.com/omcgo/omcgo/internal/infra/mq"
	"github.com/omcgo/omcgo/internal/infra/storage"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/nedirect"
	"github.com/omcgo/omcgo/internal/northbound"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
	"github.com/omcgo/omcgo/internal/interop"
	"github.com/omcgo/omcgo/internal/interop/cases"
	"github.com/omcgo/omcgo/internal/omcr/admin"
	"github.com/omcgo/omcgo/internal/omcr/dashboard"
	"github.com/omcgo/omcgo/internal/omcr/device"
	"github.com/omcgo/omcgo/internal/omcr/software"
	"github.com/omcgo/omcgo/internal/omcr/syslog"
	"github.com/omcgo/omcgo/internal/omcr/topology"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-app",
		Short: "OMC Main Application",
		Long:  "OMC main application server providing REST API, device management, and all F02-F10 modules",
		RunE:  runApp,
	}

	rootCmd.Flags().String("config", "configs/app.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) error {
	// 1. Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg config.AppConfig
	if err := config.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	logger, err := infra.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("omcgo-app starting", zap.String("config", cfgPath))

	// 3. Graceful shutdown setup
	gs := infra.NewGracefulShutdown(30*time.Second, logger)
	ctx := context.Background()

	// 4. Connect to PostgreSQL
	pgPool, err := db.NewPostgresPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	gs.Register("postgres", 4, func(ctx context.Context) error { pgPool.Close(); return nil })

	// 5. Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	gs.Register("redis", 3, func(ctx context.Context) error { return redisClient.Close() })

	// 5b. Connect to TimescaleDB
	tsPool, err := db.NewTimescalePool(ctx, cfg.TSDB)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	gs.Register("timescale", 4, func(ctx context.Context) error { tsPool.Close(); return nil })

	// 5c. Connect to MinIO
	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}

	// 6. Connect to NATS
	natsClient, err := mq.NewNATSClient(cfg.NATS, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	gs.Register("nats", 2, func(ctx context.Context) error { natsClient.Close(); return nil })

	if err := natsClient.EnsureStreams(ctx); err != nil {
		logger.Warn("ensure NATS streams", zap.Error(err))
	}

	// 7. Create EventBus
	eventBus := event.NewNATSEventBus(natsClient.Conn, natsClient.JS, logger)
	gs.Register("eventbus", 2, func(ctx context.Context) error { return eventBus.Close() })

	// 8. Create repositories
	deviceRepo := device.NewPgDeviceRepository(pgPool)
	paramRepo := device.NewPgDeviceParameterRepository(pgPool)

	// 9. Create HeartbeatMonitor
	heartbeatMonitor := device.NewHeartbeatMonitor(redisClient, deviceRepo, logger)
	heartbeatMonitor.Start()
	gs.Register("heartbeat", 1, func(ctx context.Context) error { heartbeatMonitor.Stop(); return nil })

	// 10. Create carrier registry
	carrierRegistry := carrier.NewRegistry()
	carrierRegistry.Register(cmcc.New())
	carrierRegistry.Register(ctcc.New())
	carrierRegistry.Register(cucc.New())
	logger.Info("carrier registry initialized", zap.Int("carriers", len(carrierRegistry.All())))

	// 11. Create device services
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, heartbeatMonitor, logger)

	// 12. Subscribe InformHandler to events
	informHandler := device.NewInformHandler(deviceService, carrierRegistry, model.CarrierCMCC, logger)
	if err := informHandler.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
	}

	// 13. Create DataModel module (repository + cache + registry + importer)
	dmRepo := datamodel.NewPgDataModelRepository(pgPool)
	importLogRepo := datamodel.NewPgImportLogRepository(pgPool)
	ouiRepo := datamodel.NewPgOUIRepository(pgPool)
	dmCache := datamodel.NewDataModelCache(redisClient)
	dmRegistry := datamodel.NewDataModelRegistry(dmRepo, dmCache, logger)
	dmRegistry.Start()
	gs.Register("datamodel-registry", 1, func(ctx context.Context) error { dmRegistry.Stop(); return nil })
	dmImporter := datamodel.NewDataModelImporter(dmRepo, importLogRepo)
	logger.Info("datamodel registry started")

	// 14. Create ConfigTemplate module
	templateRepo := template.NewPgConfigTemplateRepository(pgPool)
	templateService := template.NewConfigTemplateService(templateRepo, logger)

	// 15. Create command queue (shared with ACS)
	cmdQueue := cmdqueue.NewRedisCommandQueue(redisClient)

	// 16. Create Provisioning module
	provisionRepo := provision.NewPgProvisioningTaskRepository(pgPool)
	provisionEngine := provision.NewProvisioningEngine(
		provisionRepo, deviceService, dmRegistry, templateService,
		carrierRegistry, cmdQueue, eventBus, logger,
	)
	if err := provisionEngine.Subscribe(eventBus); err != nil {
		logger.Warn("subscribe provisioning engine", zap.Error(err))
	}

	// 17. Create Topology module
	groupRepo := topology.NewPgDeviceGroupRepository(pgPool)
	groupService := topology.NewDeviceGroupService(groupRepo, logger)

	// 18. Create Admin/RBAC module
	userRepo := admin.NewPgUserRepository(pgPool)
	roleRepo := admin.NewPgRoleRepository(pgPool)
	auditRepo := admin.NewPgAuditRepository(pgPool)
	jwtService := admin.NewJWTServiceWithTTL(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	adminService := admin.NewAdminService(userRepo, roleRepo, auditRepo, jwtService, logger)
	adminHandler := admin.NewHandler(adminService, logger)
	logger.Info("admin/RBAC module initialized")

	// 19. Create Software Management module
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

	// 20. PM module components (created early for use in routes)
	pmCounterRepo := counter.NewPgCounterRepository(tsPool)
	pmKPIRepo := kpi.NewPgKPIRepository(tsPool)
	pmKPIEngine := kpi.NewKPIEngine(pmCounterRepo, pmKPIRepo, carrierRegistry, logger)

	// 20. Alarm module components
	alarmRedisStore := alarm.NewRedisAlarmStore(redisClient)
	alarmPgStore := alarm.NewPgAlarmStore(pgPool, tsPool)
	alarmEngine := alarm.NewAlarmEngine(alarmPgStore, alarmRedisStore, carrierRegistry, eventBus, logger)

	// 21. MR module components
	mrStore := mr.NewPgMRStore(pgPool, tsPool)

	// 22. Setup Gin router
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	corsOrigins := cfg.CORS.AllowOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: corsOrigins,
	}))
	router.Use(middleware.RequestLogger(logger))

	metricsReg := prometheus.NewRegistry()
	router.Use(middleware.PrometheusMetrics(metricsReg))

	// Health check (public, no auth)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public auth routes (no authentication required)
	publicV1 := router.Group("/api/v1")
	adminHandler.RegisterAuthRoutes(publicV1)

	// Protected API v1 routes (JWT authentication required)
	v1 := router.Group("/api/v1")
	v1.Use(admin.RequireAuth(jwtService))
	v1.Use(admin.RequireCarrier())
	v1.Use(admin.AuditLogger(auditRepo))

	// Protected auth routes (authentication required)
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
	topologyHandler := topology.NewHandler(groupRepo, groupService)
	topologyHandler.RegisterRoutes(v1)

	// PM routes
	pmHandler := pm.NewHandler(pmCounterRepo, pmKPIRepo, pmKPIEngine, logger)
	pmHandler.RegisterRoutes(v1)

	// Alarm routes
	alarmHandler := alarm.NewHandler(alarmEngine, alarmPgStore, logger)
	alarmHandler.RegisterRoutes(v1)

	// Alarm rule routes (Sprint 4)
	alarmRuleRepo := alarm.NewPgAlarmRuleRepository(pgPool)
	alarmRuleHandler := alarm.NewRuleHandler(alarmRuleRepo, logger)
	alarmsGroup := v1.Group("/alarms")
	alarmRuleHandler.RegisterRoutes(alarmsGroup)

	// KPI threshold routes (Sprint 4)
	thresholdRepo := pm.NewPgThresholdRepository(pgPool)
	thresholdHandler := pm.NewThresholdHandler(thresholdRepo, logger)
	pmGroup := v1.Group("/pm")
	thresholdHandler.RegisterRoutes(pmGroup)

	// Dashboard routes (Sprint 4)
	dashboardService := dashboard.NewService(deviceService, alarmPgStore, pmKPIRepo, logger)
	dashboardHandler := dashboard.NewHandler(dashboardService)
	dashboardHandler.RegisterRoutes(v1)

	// Syslog routes (Sprint 4)
	syslogRepo := syslog.NewPgSyslogRepository(pgPool)
	syslogHandler := syslog.NewHandler(syslogRepo, logger)
	syslogHandler.RegisterRoutes(v1)

	// Config sync routes (Sprint 4)
	syncHandler := config.NewSyncHandler(cmdQueue, logger)
	syncHandler.RegisterRoutes(v1)

	// MR routes
	mrHandler := mr.NewHandler(mrStore, minioClient, cfg.MinIO.Buckets.MRFiles, logger)
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

	// 23. Prometheus metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
		Handler: metricsMux,
	}
	gs.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	go func() {
		logger.Info("metrics server starting", zap.Int("port", cfg.Metrics.Port))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	// 24. Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	gs.Register("app-http", 1, func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		logger.Info("app server starting", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 25. Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			logger.Error("app server error", zap.Error(err))
		}
	}

	// 26. Graceful shutdown
	if err := gs.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("omcgo-app stopped")
	return nil
}
