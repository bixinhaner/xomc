package acs

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/download"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/acs/upload"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/health"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// newOnlineIndexFromRedis 在 redis 客户端可用时构造在线索引（issue #397）。
// c 为 nil（dev/test 无 Redis）时返回 nil，Handler.onlineIndex.Mark 走 nil-receiver no-op。
func newOnlineIndexFromRedis(c redis.Cmdable) *redisx.OnlineIndex {
	if c == nil {
		return nil
	}
	return redisx.NewOnlineIndex(c)
}

// ACSServer is the TR069 ACS HTTP server.
type ACSServer struct {
	httpServer *http.Server
	handler    *Handler
	config     appconfig.ACSServerConfig
	logger     *zap.Logger
}

// ServerDeps holds the dependencies for the ACS server.
type ServerDeps struct {
	SessionStore  SessionStore
	TaskService   *task.TaskService
	EventBus      event.EventBus
	Authenticator auth.DeviceAuthenticator
	RPCDispatcher *rpc.Dispatcher
	RateLimiter   *DeviceRateLimiter
	Admission     AdmissionController
	Metrics       *ACSMetrics
	// DeviceSessionStore 跨实例设备→会话指针存储（issue #65 Option B）。nil 时退化为
	// 不做跨实例孤儿会话检测（单实例由 SessionStore TTL + 准入槽位 TTL 兜底）。
	DeviceSessionStore DeviceSessionStore
	// ConnReqURLStore 跨实例 ConnectionRequestURL 共享存储（issue #65 Option B）。
	// nil 时 postSessionWake 的 HTTP 回退拿到空 URL（等价改造前缓存 miss，STUN 设备不受影响）。
	ConnReqURLStore        ConnReqURLStore
	UploadHandler          *upload.Handler                 // CPE file upload handler (supports query params and path-based token)
	UploadConfig           *appconfig.UploadConfig         // upload server configuration for generating upload URLs
	DownloadHandler        *download.Handler               // CPE file download handler (MinIO → CPE proxy)
	DownloadConfig         *appconfig.DownloadConfig       // download server configuration for generating download URLs
	TransferConfigProvider transfercfg.Provider            // runtime-overridable transfer endpoint settings
	ConnReqSender          ConnectionRequester             // post-session wake: send CR when queue not empty
	PostSessionWakeCfg     appconfig.PostSessionWakeConfig // post-session wake configuration
	RedisClient            redis.Cmdable                   // Redis client for continuous wake counter
	AccessSnapshotStore    AccessSnapshotStore             // shared authorization summary for ACS hot path
	StunStore              *stun.Store                     // STUN address cache (shared with STUN server)
	ProtocolLogger         *zap.Logger                     // dedicated logger for protocol XML (nil = disabled)
	MaxBodySize            int                             // XML truncation threshold for protocol log (0 = no truncation)
	// T-0137 / M1: TR069 报文跟踪。两个都为 nil 表示跟踪关闭。
	TraceWhitelist *trace.WhitelistCache
	TraceService   *trace.Service
	// PathTranslator: ACS 端 standardPath → privatePath 翻译服务。
	// 任一底层依赖（ProductRegistry / ParamRegistry / DeviceRepo）未配置时 → nil-safe
	// 退化为透传（task.Params 原样下发，等价于改造前行为）。
	PathTranslator *PathTranslationService
	// #746: 心跳周期自动调整策略。nil 时功能关闭（不影响 Inform 处理）。
	InformPeriodPolicy      *InformPeriodPolicy
	UECountPolicy           *UECountPolicy
	GPVFaultRecoverer       GPVFaultRecoverer
	DurableReadbackEnabled  bool
	Logger                  *zap.Logger
	RequestIDPrefix         string // prefix for request IDs, e.g., "acs"
	EnableTestTaskInjection bool   // enable random test task injection (for testing only)
	// ReadinessCheckers 提供 /readyz 探测时的依赖检查列表（DB/Redis/NATS 等）。
	// 为 nil 时 /readyz 退化为恒 200，但 /healthz 始终独立存在。
	ReadinessCheckers []health.Checker
}

// NewACSServer creates a new ACS server with all dependencies wired.
func NewACSServer(cfg appconfig.ACSConfig, deps ServerDeps) *ACSServer {
	trustedProxyCIDRs := make([]netip.Prefix, 0, len(cfg.Server.TrustedProxyCIDRs))
	for _, raw := range cfg.Server.TrustedProxyCIDRs {
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(raw)); err == nil {
			trustedProxyCIDRs = append(trustedProxyCIDRs, prefix.Masked())
		}
	}
	h := &Handler{
		sessionStore:            deps.SessionStore,
		taskService:             deps.TaskService,
		eventBus:                deps.EventBus,
		authenticator:           deps.Authenticator,
		rpcDispatcher:           deps.RPCDispatcher,
		rateLimiter:             deps.RateLimiter,
		admission:               deps.Admission,
		deviceSessionStore:      deps.DeviceSessionStore,
		connReqURLStore:         deps.ConnReqURLStore,
		metrics:                 deps.Metrics,
		logger:                  deps.Logger,
		requestIDPrefix:         deps.RequestIDPrefix,
		enableTestTaskInjection: deps.EnableTestTaskInjection,
		uploadConfig:            deps.UploadConfig,
		downloadConfig:          deps.DownloadConfig,
		transferConfigProvider:  deps.TransferConfigProvider,
		maxRPCPerSession:        cfg.Session.MaxRPCPerSession,
		connReqSender:           deps.ConnReqSender,
		postSessionWakeCfg:      deps.PostSessionWakeCfg,
		redisClient:             deps.RedisClient,
		accessSnapshots:         deps.AccessSnapshotStore,
		onlineIndex:             newOnlineIndexFromRedis(deps.RedisClient),
		stunStore:               deps.StunStore,
		protocolLogger:          deps.ProtocolLogger,
		maxBodySize:             deps.MaxBodySize,
		maxRequestBodySize:      cfg.Server.MaxRequestBodySize,
		traceWhitelist:          deps.TraceWhitelist,
		traceService:            deps.TraceService,
		pathTranslator:          deps.PathTranslator,
		informPeriodPolicy:      deps.InformPeriodPolicy,
		ueCountPolicy:           deps.UECountPolicy,
		gpvFaultRecoverer:       deps.GPVFaultRecoverer,
		durableReadbackEnabled:  deps.DurableReadbackEnabled,
		trustedProxyCIDRs:       trustedProxyCIDRs,
	}

	mux := http.NewServeMux()
	// TR069 ACS endpoint - /smallcell/AcsService
	mux.HandleFunc("/smallcell/AcsService", h.ServeHTTP)
	// /healthz — liveness 探针：进程存活即 200，不依赖外部组件。
	// /readyz  — readiness 探针：DB/Redis/NATS 任一不可用返 503，否则 200。
	mux.Handle("/healthz", health.LivenessHandler())
	mux.Handle("/readyz", health.ReadinessHandler(5*time.Second, deps.ReadinessCheckers...))

	// CPE file upload handler (CPE -> ACS -> MinIO proxy)
	// Endpoint: POST /smallcell/FileUploadService?fileType=PM&filename=xxx
	// Auth: HTTP Basic Authentication with global credentials
	if deps.UploadHandler != nil {
		mux.Handle("/smallcell/FileUploadService", deps.UploadHandler)
	}

	// CPE file download handler (ACS -> MinIO -> CPE proxy)
	// Endpoint: GET /smallcell/FileDownloadService/{bucket}/{objectPath...}
	// Auth: HTTP Basic Authentication with global credentials
	if deps.DownloadHandler != nil {
		mux.Handle("/smallcell/FileDownloadService/", deps.DownloadHandler)
	}

	// Start background session reaper to clean up stale connSessions entries
	// from dropped TCP connections. Scans every 30s, cleans entries older than 5min.
	h.startSessionReaper(30*time.Second, 5*time.Minute)
	// 全局准入槽位是多 ACS 实例共同的真实会话数；单实例本地 gauge 在跨实例
	// 孤儿清理时只能最终一致，不能用于 30000 全局上限监控。
	h.startGlobalAdmissionMetrics(5 * time.Second)

	// Start rate limiter background cleanup based on config.
	// Defaults: scan every 5 min, evict devices inactive for 10 min.
	cleanupInterval := cfg.RateLimit.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute
	}
	cleanupTimeout := cfg.RateLimit.CleanupTimeout
	if cleanupTimeout <= 0 {
		cleanupTimeout = 10 * time.Minute
	}
	deps.RateLimiter.StartCleanup(cleanupInterval, cleanupTimeout)

	// MaxHeaderBytes 限制 HTTP 头部总大小，挡住超大头部攻击；未配置时回退到
	// stdlib 默认 1MB（http.DefaultMaxHeaderBytes）。
	maxHeaderBytes := cfg.Server.MaxHeaderBytes
	if maxHeaderBytes <= 0 {
		maxHeaderBytes = http.DefaultMaxHeaderBytes
	}

	return &ACSServer{
		httpServer: &http.Server{
			Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:        mux,
			ReadTimeout:    cfg.Server.ReadTimeout,
			WriteTimeout:   cfg.Server.WriteTimeout,
			IdleTimeout:    cfg.Server.IdleTimeout,
			MaxHeaderBytes: maxHeaderBytes,
		},
		handler: h,
		config:  cfg.Server,
		logger:  deps.Logger,
	}
}

// Start begins listening for incoming TR069 requests.
func (s *ACSServer) Start() error {
	s.logger.Info("ACS server starting",
		zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("ACS server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the ACS server and its background goroutines.
func (s *ACSServer) Shutdown(ctx context.Context) error {
	s.logger.Info("ACS server shutting down")
	s.handler.rateLimiter.Stop()
	return s.httpServer.Shutdown(ctx)
}

// RegisterMetrics registers ACS metrics with a custom Prometheus registry.
func RegisterMetrics(reg prometheus.Registerer) *ACSMetrics {
	return NewACSMetrics(reg)
}

// NewDefaultDeps creates ServerDeps with default implementations suitable for
// starting the ACS server with Redis and NATS-backed components.
func NewDefaultDeps(
	sessionStore SessionStore,
	taskSvc *task.TaskService,
	eventBus event.EventBus,
	authMode, authUser, authPass string,
	rateCfg appconfig.RateLimitConfig,
	maxSessions int64,
	metricsReg prometheus.Registerer,
	logger *zap.Logger,
	requestIDPrefix string,
	enableTestTaskInjection bool,
) ServerDeps {
	return ServerDeps{
		SessionStore:            sessionStore,
		TaskService:             taskSvc,
		EventBus:                eventBus,
		Authenticator:           auth.NewAuthenticator(authMode, authUser, authPass),
		RPCDispatcher:           rpc.NewDispatcher(),
		RateLimiter:             NewDeviceRateLimiter(rateCfg.PerDevice, rateCfg.Burst, rateCfg.MaxDevices, logger),
		Admission:               NewAdmissionController(maxSessions),
		Metrics:                 NewACSMetrics(metricsReg),
		Logger:                  logger,
		RequestIDPrefix:         requestIDPrefix,
		EnableTestTaskInjection: enableTestTaskInjection,
	}
}
