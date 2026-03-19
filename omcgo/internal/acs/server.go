package acs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

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
	CommandQueue  cmdqueue.CommandQueue
	EventBus      event.EventBus
	Authenticator auth.DeviceAuthenticator
	RPCDispatcher *rpc.Dispatcher
	RateLimiter   *DeviceRateLimiter
	Admission     *AdmissionController
	Metrics       *ACSMetrics
	Logger        *zap.Logger
}

// NewACSServer creates a new ACS server with all dependencies wired.
func NewACSServer(cfg appconfig.ACSConfig, deps ServerDeps) *ACSServer {
	h := &Handler{
		sessionStore:  deps.SessionStore,
		commandQueue:  deps.CommandQueue,
		eventBus:      deps.EventBus,
		authenticator: deps.Authenticator,
		rpcDispatcher: deps.RPCDispatcher,
		rateLimiter:   deps.RateLimiter,
		admission:     deps.Admission,
		metrics:       deps.Metrics,
		logger:        deps.Logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("", h.ServeHTTP)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Start background session reaper to clean up stale connSessions entries
	// from dropped TCP connections. Scans every 30s, cleans entries older than 5min.
	h.startSessionReaper(30*time.Second, 5*time.Minute)

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

	return &ACSServer{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      mux,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
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
	cmdQueue cmdqueue.CommandQueue,
	eventBus event.EventBus,
	authMode, authUser, authPass string,
	rateCfg appconfig.RateLimitConfig,
	maxSessions int64,
	metricsReg prometheus.Registerer,
	logger *zap.Logger,
) ServerDeps {
	return ServerDeps{
		SessionStore:  sessionStore,
		CommandQueue:  cmdQueue,
		EventBus:      eventBus,
		Authenticator: auth.NewAuthenticator(authMode, authUser, authPass),
		RPCDispatcher: rpc.NewDispatcher(),
		RateLimiter:   NewDeviceRateLimiter(rateCfg.PerDevice, rateCfg.Burst, rateCfg.MaxDevices, logger),
		Admission:     NewAdmissionController(maxSessions),
		Metrics:       NewACSMetrics(metricsReg),
		Logger:        logger,
	}
}
