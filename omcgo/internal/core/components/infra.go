package components

import (
	"context"
	"fmt"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	logpkg "github.com/omcgo/omcgo/internal/core/components/logger"
	miniocomp "github.com/omcgo/omcgo/internal/core/components/minio"
	natscomp "github.com/omcgo/omcgo/internal/core/components/nats"
	"github.com/omcgo/omcgo/internal/core/components/postgres"
	redismetrics "github.com/omcgo/omcgo/internal/core/components/redis"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/components/sdnotify"
	"github.com/omcgo/omcgo/internal/core/event"
	healthpkg "github.com/omcgo/omcgo/internal/core/health"
)

// Infra 持有服务启动期间初始化的所有基础设施连接。
// 各微服务入口（cmd/acs、cmd/app、cmd/worker）选择性调用 Connect* 方法按需初始化组件。
// 它同时整合了健康检查、优雅关机、Prometheus 指标和 /healthz 接口，是服务启动的唯一入口。
type Infra struct {
	Logger *zap.Logger
	// LogGate is shared with storage protection so every service logger can
	// stop emitting to both stdout and its mounted log file at the same time.
	LogGate    *logpkg.AdmissionGate
	GS         *GracefulShutdown
	PgPool     *pgxpool.Pool
	TsPool     *pgxpool.Pool
	Redis      redis.UniversalClient
	PMRedis    redis.UniversalClient
	MinIO      *minio.Client
	NATS       *natscomp.NATSClient
	EventBus   event.EventBus
	MetricsReg *prometheus.Registry
	Health     *HealthChecker

	metricsPort int

	// pprof 开关：由 SetPprof 注入（通常来自 cfg.Metrics.Pprof）；
	// 与原始短 env OMCGO_PPROF / OMCGO_PPROF_CONTENTION 取或，便于临时强开。
	pprofEnabled    bool
	pprofContention bool
}

// SetPprof 注入 pprof 配置（配置文件驱动）。必须在 ListenAndServe / WaitAndShutdown
// 之前调用（startMetrics 读取这两个字段）。
func (inf *Infra) SetPprof(enabled, contention bool) {
	inf.pprofEnabled = enabled
	inf.pprofContention = contention
}

// NewInfra creates a base Infra with logger, graceful shutdown, metrics registry, and health checker.
func NewInfra(logCfg appconfig.LogConfig, metricsPort int) (*Infra, error) {
	logGate := logpkg.NewAdmissionGate()
	logger, err := logpkg.NewLoggerWithAdmissionGate(logCfg, logGate)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	// Set as global logger so logger.L(ctx) can access it via zap.L()
	zap.ReplaceGlobals(logger)

	// 指标 registry：注册 Go runtime 采集器（go_goroutines / go_threads /
	// go_memstats_* / go_gc_*）与进程采集器（process_resident_memory_bytes /
	// process_cpu_seconds_total / process_open_fds）。这是「服务资源监控」的运行时
	// 基线 —— 各业务/基础设施模块的指标随后注册到同一 registry，由 /metrics 统一暴露。
	metricsReg := prometheus.NewRegistry()
	metricsReg.MustRegister(collectors.NewGoCollector())
	metricsReg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return &Infra{
		Logger:      logger,
		LogGate:     logGate,
		GS:          NewGracefulShutdown(30*time.Second, logger),
		MetricsReg:  metricsReg,
		Health:      NewHealthChecker(),
		metricsPort: metricsPort,
	}, nil
}

// InitTracer initializes the OpenTelemetry TracerProvider.
// When tracing is disabled, a no-op provider is set (zero overhead).
// The TracerProvider is registered for graceful shutdown.
func (inf *Infra) InitTracer(ctx context.Context, cfg appconfig.TracerConfig, serviceName string) error {
	tp, err := NewTracerProvider(ctx, cfg, serviceName)
	if err != nil {
		return fmt.Errorf("init tracer provider: %w", err)
	}
	inf.GS.Register("tracer", 5, func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})
	if cfg.Enabled {
		inf.Logger.Info("tracer initialized",
			zap.String("endpoint", cfg.Endpoint),
			zap.Float64("sample_rate", cfg.SampleRate))
	} else {
		inf.Logger.Info("tracer disabled (no-op provider)")
	}
	return nil
}

// ConnectPostgres 初始化 PostgreSQL 连接池，并注册健康检查和优雅关机回调。
// 供主库（devices/alarms 等）使用，结果存入 Infra.PgPool。
func (inf *Infra) ConnectPostgres(ctx context.Context, cfg appconfig.PostgresConfig) error {
	mainRegisterer := prometheus.WrapRegistererWith(
		prometheus.Labels{"pool": "main"},
		inf.MetricsReg,
	)
	pool, err := postgres.NewPostgresPoolWithRegisterer(ctx, cfg, inf.Logger, mainRegisterer)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	inf.PgPool = pool
	inf.GS.Register("postgres", 4, func(ctx context.Context) error { pool.Close(); return nil })
	inf.Health.Register("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	// 连接池资源指标（pgxpool_in_use/idle/max/acquire_total），带 pool="main" 常量标签
	// 与时序库（pool="tsdb"）区分，避免同名指标在同一 registry 冲突。
	mainPoolMetrics := postgres.RegisterPoolMetrics(pool, mainRegisterer)
	// 优先级 1：采样器先于连接池关闭（优先级 4）停止，避免采样已关闭的 pool。
	inf.GS.Register("postgres-pool-metrics", 1, func(context.Context) error { mainPoolMetrics.Stop(); return nil })
	return nil
}

// ConnectTimescale 初始化 TimescaleDB 连接池，并注册健康检查和优雅关机回调。
// 供 PM/KPI 超表使用，结果存入 Infra.TsPool。
func (inf *Infra) ConnectTimescale(ctx context.Context, cfg appconfig.PostgresConfig) error {
	tsdbRegisterer := prometheus.WrapRegistererWith(
		prometheus.Labels{"pool": "tsdb"},
		inf.MetricsReg,
	)
	pool, err := postgres.NewTimescalePoolWithRegisterer(ctx, cfg, inf.Logger, tsdbRegisterer)
	if err != nil {
		return fmt.Errorf("connect to TimescaleDB: %w", err)
	}
	inf.TsPool = pool
	inf.GS.Register("timescale", 4, func(ctx context.Context) error { pool.Close(); return nil })
	inf.Health.Register("timescale", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	// 时序库连接池资源指标，带 pool="tsdb" 常量标签与主库（pool="main"）区分。
	tsPoolMetrics := postgres.RegisterPoolMetrics(pool, tsdbRegisterer)
	inf.GS.Register("timescale-pool-metrics", 1, func(context.Context) error { tsPoolMetrics.Stop(); return nil })
	return nil
}

// ConnectRedis 初始化 Redis 客户端，并注册健康检查和优雅关机回调。
// 结果存入 Infra.Redis，封装为 redis.UniversalClient，支持单机和集群模式。
func (inf *Infra) ConnectRedis(cfg appconfig.RedisConfig) error {
	client, err := inf.connectRedis(cfg, "core")
	if err != nil {
		return err
	}
	inf.Redis = client
	return nil
}

// ConnectPMRedis initializes the physically isolated PM/KPI Redis client.
// Callers that intentionally use the compatibility fallback should assign
// Infra.PMRedis = Infra.Redis instead of opening a duplicate pool.
func (inf *Infra) ConnectPMRedis(cfg appconfig.RedisConfig) error {
	client, err := inf.connectRedis(cfg, "pm")
	if err != nil {
		return err
	}
	inf.PMRedis = client
	return nil
}

func (inf *Infra) connectRedis(cfg appconfig.RedisConfig, role string) (redis.UniversalClient, error) {
	client, err := redisx.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to Redis role %s: %w", role, err)
	}
	component := "redis-" + role
	inf.GS.Register(component, 3, func(ctx context.Context) error { return client.Close() })
	inf.Health.Register(component, func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	})
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{"role": role}, inf.MetricsReg)
	redisPoolMetrics := redismetrics.RegisterPoolMetrics(client, cfg.PoolSize, registerer)
	inf.GS.Register(component+"-pool-metrics", 1, func(context.Context) error { redisPoolMetrics.Stop(); return nil })
	return client, nil
}

// ConnectNATS 初始化 NATS JetStream 客户端，并确保流存在。
// 结果存入 Infra.NATS，后续可调用 CreateEventBus 创建事件总线。
func (inf *Infra) ConnectNATS(ctx context.Context, cfg appconfig.NATSConfig) error {
	return inf.ConnectNATSWithAlarmLifecycleMaxBytes(ctx, cfg, 0)
}

// ConnectNATSWithAlarmLifecycleMaxBytes keeps the common infrastructure path
// while allowing app/worker alarm lifecycle deployments to size DOMAIN_ALARM.
func (inf *Infra) ConnectNATSWithAlarmLifecycleMaxBytes(
	ctx context.Context,
	cfg appconfig.NATSConfig,
	maxBytes int64,
) error {
	client, err := natscomp.NewNATSClient(cfg, inf.Logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	inf.NATS = client
	inf.GS.Register("nats", 2, func(ctx context.Context) error { client.Close(); return nil })
	inf.Health.Register("nats", func(ctx context.Context) error {
		return client.HealthCheck()
	})
	// 连接资源指标（nats_conn_status / nats_reconnect_total / nats_msgs_*）及
	// 固定 JetStream stream/consumer 积压指标（omc_nats_*）。RegisterMetrics
	// 使用同一个 NATSClient.JS 启动两类观测，ConnMetrics.Stop 会先停止队列
	// observer，再由 GracefulShutdown 关闭 NATS 连接，避免查询协程泄漏。
	natsConnMetrics := client.RegisterMetrics(inf.MetricsReg)
	inf.GS.Register("nats-conn-metrics", 1, func(context.Context) error { natsConnMetrics.Stop(); return nil })

	if err := client.EnsureStreamsWithAlarmLifecycleMaxBytes(ctx, cfg.AllowStreamRebuild, maxBytes); err != nil {
		return fmt.Errorf("ensure NATS streams: %w", err)
	}
	return nil
}

// ConnectMinIO 初始化 MinIO 客户端，并确保配置的 Bucket 存在（不存在自动创建）。
// 结果存入 Infra.MinIO，不注册优雅关机（MinIO 客户端无状态）。
func (inf *Infra) ConnectMinIO(ctx context.Context, cfg appconfig.MinIOConfig) error {
	client, err := miniocomp.NewMinIOClient(cfg)
	if err != nil {
		return fmt.Errorf("connect to MinIO: %w", err)
	}
	inf.MinIO = client
	inf.Health.Register("minio", func(ctx context.Context) error {
		return miniocomp.MinIOHealthCheck(ctx, client)
	})

	if err := miniocomp.EnsureBuckets(ctx, client, cfg.Buckets); err != nil {
		inf.Logger.Warn("ensure MinIO buckets", zap.Error(err))
	}
	return nil
}

// readinessCheckers 把 HealthChecker 已注册的检查项适配为 health.Checker 列表，
// 供 /readyz 处理器使用。注册行为仍保留在 Connect* 方法内，保证依赖建模的单一来源。
func (inf *Infra) readinessCheckers() []healthpkg.Checker {
	if inf.Health == nil {
		return nil
	}
	inf.Health.mu.RLock()
	defer inf.Health.mu.RUnlock()
	checkers := make([]healthpkg.Checker, 0, len(inf.Health.checks))
	for _, c := range inf.Health.checks {
		// 闭包捕获循环变量；这里复制一份以保证后续 goroutine 看到正确的项。
		name := c.name
		fn := c.fn
		checkers = append(checkers, healthpkg.NewChecker(name, fn))
	}
	return checkers
}

// CreateEventBus 创建基于 NATS JetStream 的 EventBus，并注册优雅关机回调。
// 必须在 ConnectNATS 之后调用，结果存入 Infra.EventBus。
func (inf *Infra) CreateEventBus() {
	bus := event.NewNATSEventBus(inf.NATS.Conn, inf.NATS.JS, inf.Logger)
	// issue #20：注入投递结果指标（ack/nak/terminated/dropped），让 max-retries 终止
	// 与解析丢弃不再静默；所有进程（app/acs/worker）经此统一入口都自动带上。
	bus.SetMetrics(event.NewEventBusMetrics(inf.MetricsReg))
	inf.EventBus = bus
	inf.GS.Register("eventbus", 2, func(ctx context.Context) error { return inf.EventBus.Close() })
}

// --- server lifecycle ---

// ListenAndServe 启动 HTTP 服务器和 Prometheus 指标服务器，
// 阻塞直到收到 SIGINT/SIGTERM 信号，然后依优先级逐步优雅关机。
// 适用于 App/ACS 等需要外露 HTTP 端口的服务。
func (inf *Infra) ListenAndServe(handler http.Handler, addr string, readTimeout, writeTimeout, idleTimeout time.Duration) error {
	inf.startMetrics()
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}
	if idleTimeout <= 0 {
		idleTimeout = 120 * time.Second
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	inf.GS.Register("app-http", 1, func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		inf.Logger.Info("http server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	return inf.waitForShutdown(errCh)
}

// WaitAndShutdown 启动指标服务器，然后阻塞直到收到信号或 errCh 发送错误。
// 适用于 Worker 等无 HTTP 服务器的后台进程，errCh 为 nil 时仅等信号。
func (inf *Infra) WaitAndShutdown(errCh <-chan error) error {
	inf.startMetrics()
	return inf.waitForShutdown(errCh)
}

func (inf *Infra) startMetrics() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(inf.MetricsReg, promhttp.HandlerOpts{}))
	// /healthz — liveness：进程存活即 200，不关心依赖。
	// /readyz  — readiness：聚合 HealthChecker 中已注册的依赖检查，任一失败 503。
	// 两者职责严格分离，与 Kubernetes 探针语义对齐，避免错误重启正在恢复的实例。
	metricsMux.Handle("/healthz", healthpkg.LivenessHandler())
	metricsMux.Handle("/readyz", healthpkg.ReadinessHandler(5*time.Second, inf.readinessCheckers()...))

	// net/http/pprof —— 仅在 OMCGO_PPROF 为真时挂到内网 metrics 端口（与 /metrics 同信任
	// 边界，不暴露在对外业务 API 端口）。默认关闭：商用部署保持干净，排查 CPU/内存时按需
	// 打开后重启进程即可，无需改业务代码。
	// 配置文件(cfg.Metrics.Pprof)优先；原始短 env 作为临时强开的兜底。
	enablePprof := inf.pprofEnabled || isEnvTruthy(os.Getenv("OMCGO_PPROF"))
	enableContention := inf.pprofContention || isEnvTruthy(os.Getenv("OMCGO_PPROF_CONTENTION"))
	if enablePprof {
		registerPprof(metricsMux)
		inf.Logger.Warn("pprof endpoints ENABLED on metrics port (/debug/pprof/*) —— 仅用于线上排查，排查完请关闭",
			zap.Int("port", inf.metricsPort))
		// 竞争剖析（block/mutex）默认仍关：它对热锁有可测开销，会扰动 CPU profile。
		// 仅在显式开启时低速率采样，用于定位锁/连接池争用。
		if enableContention {
			runtime.SetBlockProfileRate(blockProfileRateNs)
			runtime.SetMutexProfileFraction(mutexProfileFraction)
			inf.Logger.Warn("pprof contention profiling ENABLED (block+mutex 采样有开销)",
				zap.Int("block_rate_ns", blockProfileRateNs),
				zap.Int("mutex_fraction", mutexProfileFraction))
		}
	}

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", inf.metricsPort),
		Handler: metricsMux,
	}
	inf.GS.Register("metrics-http", 1, func(ctx context.Context) error { return metricsServer.Shutdown(ctx) })

	go func() {
		inf.Logger.Info("metrics server starting", zap.Int("port", inf.metricsPort))
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			inf.Logger.Error("metrics server error", zap.Error(err))
		}
	}()
}

// pprof 竞争剖析采样参数（仅 OMCGO_PPROF_CONTENTION 开启时生效）。
const (
	blockProfileRateNs   = 1_000_000 // 平均每阻塞 ~1ms 采样一次，开销可控
	mutexProfileFraction = 100       // 1/100 互斥竞争事件采样
)

// registerPprof 把标准 net/http/pprof 处理器挂到给定 mux（CPU/heap/goroutine/allocs/
// block/mutex/trace）。注意：必须显式注册到自定义 mux —— 包的 init 只注册到
// http.DefaultServeMux，而本进程不对外服务 DefaultServeMux。
func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", httppprof.Index) // 含 heap/goroutine/allocs/block/mutex 等命名 profile
	mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", httppprof.Profile) // CPU profile（?seconds=N）
	mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)
}

func isEnvTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (inf *Infra) waitForShutdown(errCh <-chan error) error {
	// systemd 集成（T-0150）：此处所有初始化已完成（HTTP / metrics 监听已启动），
	// 通知 systemd 启动就绪（Type=notify 必需），并启动看门狗喂狗 goroutine。
	// 非 systemd 环境（dev / 裸跑 / 测试）下 sdnotify 全部为 no-op，
	// 同一二进制在各部署形态下行为一致。
	wdCtx, wdCancel := context.WithCancel(context.Background())
	defer wdCancel()
	sdnotify.Ready()
	sdnotify.StartWatchdog(wdCtx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if errCh != nil {
		select {
		case sig := <-sigCh:
			inf.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
		case err := <-errCh:
			if err != nil {
				inf.Logger.Error("server error", zap.Error(err))
			}
		}
	} else {
		sig := <-sigCh
		inf.Logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	}

	// 通知 systemd 进入停止阶段：systemd 随即不再期待看门狗心跳，改用
	// TimeoutStopSec 约束停止耗时；同时停止本进程的喂狗 goroutine。
	sdnotify.Stopping()
	wdCancel()

	if err := inf.GS.Shutdown(context.Background()); err != nil {
		inf.Logger.Error("shutdown error", zap.Error(err))
	}

	inf.Logger.Info("service stopped")
	return nil
}
