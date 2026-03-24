package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/acs/upload"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "omcgo-acs",
		Short: "OMC TR069 ACS Engine",
		Long:  "TR069/CWMP Auto Configuration Server for small cell management",
		RunE:  runACS,
	}

	rootCmd.Flags().String("config", "cmd/acs/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runACS(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.ACSConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Handle LOG_OUTPUT_PATHS environment variable (comma-separated)
	if outputPaths := os.Getenv("OMCGO_LOG_OUTPUT_PATHS"); outputPaths != "" {
		cfg.Log.OutputPaths = parseStringSlice(outputPaths)
	}

	inf, err := initACS(context.Background(), &cfg)
	if err != nil {
		return err
	}
	defer inf.Logger.Sync()
	inf.Logger.Info("omcgo-acs starting", zap.String("config", cfgPath))

	// Create ACS-specific components
	sessionStore := acs.NewRedisSessionStore(inf.Redis, cfg.Session.Timeout)
	cmdQueue := cmdqueue.NewRedisCommandQueue(inf.Redis)

	// Create TaskService if PostgreSQL is available
	var taskService *task.TaskService
	if inf.PgPool != nil {
		taskQueue := task.NewRedisTaskQueue(inf.Redis)
		taskRepo := task.NewPgTaskRepository(inf.PgPool)
		taskService = task.NewTaskService(taskQueue, taskRepo, inf.Logger)
		inf.Logger.Info("task service initialized")
	}

	// Get request ID prefix from config, default to "acs"
	requestIDPrefix := cfg.RequestIDPrefix
	if requestIDPrefix == "" {
		requestIDPrefix = "acs"
	}

	deps := acs.NewDefaultDeps(
		sessionStore,
		cmdQueue,
		taskService,
		inf.EventBus,
		cfg.Auth.Mode, cfg.Auth.Username, cfg.Auth.Password,
		cfg.RateLimit,
		cfg.Session.MaxConcurrent,
		inf.MetricsReg,
		inf.Logger,
		requestIDPrefix,
		cfg.EnableTestTaskInjection,
	)

	// Setup upload handler for CPE file upload (PM/MR/DataModel files).
	if cfg.Upload.Username != "" && inf.MinIO != nil {
		tokenMgr := upload.NewTokenManager(cfg.Upload.TokenSecret, cfg.Upload.TokenTTL)
		// SessionStore requires *redis.Client; extract from UniversalClient if possible.
		var uploadSessionStore *upload.SessionStore
		if redisClient, ok := inf.Redis.(*redis.Client); ok {
			uploadSessionStore = upload.NewSessionStore(redisClient, cfg.Upload.TokenTTL)
		}
		uploadHandler := upload.NewHandler(
			tokenMgr, uploadSessionStore, inf.MinIO,
			cfg.Upload.MaxFileSize, cfg.MinIO.Buckets,
			cfg.Upload.Username, cfg.Upload.Password,
			inf.EventBus, inf.Logger,
		)
		deps.UploadHandler = uploadHandler
		deps.UploadConfig = &cfg.Upload
		inf.Logger.Info("upload handler enabled",
			zap.String("username", cfg.Upload.Username))
	}

	// Setup STUN store and UDP sender (needed for both STUN server and post-session wake)
	var stunStore *stun.Store
	var udpSender *connreq.UDPSender
	if cfg.STUN.Enabled {
		stunStore = stun.NewStore(inf.Redis, inf.Logger)
		udpSender = connreq.NewUDPSender(stunStore, cfg.STUN.SharedSecret, inf.Logger)
	}

	// Always pass STUN store to handler so it caches UDPConnectionRequestAddress from Inform.
	if stunStore != nil {
		deps.StunStore = stunStore
	}

	// Setup post-session wake: send Connection Request when session ends with remaining commands.
	if cfg.PostSessionWake.Enabled {
		httpClient := connreq.NewClient(inf.Redis, inf.Logger)
		dispatcher := connreq.NewDispatcher(httpClient, udpSender, inf.Logger)
		if inf.MetricsReg != nil {
			dispatcher.SetMetrics(connreq.NewDispatcherMetrics(inf.MetricsReg))
		}
		deps.ConnReqSender = &acsConnReqSender{dispatcher: dispatcher, isENB: true}
		deps.PostSessionWakeCfg = cfg.PostSessionWake
		deps.RedisClient = inf.Redis
		inf.Logger.Info("post-session wake enabled",
			zap.Duration("delay_after", cfg.PostSessionWake.DelayAfter),
			zap.Int("max_continuous", cfg.PostSessionWake.MaxContinuous))
	}

	acsServer := acs.NewACSServer(cfg, deps)
	inf.GS.Register("acs-http", 1, func(ctx context.Context) error { return acsServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		errCh <- acsServer.Start()
	}()

	// Start STUN UDP server for NAT traversal and Connection Request
	if cfg.STUN.Enabled && stunStore != nil {
		stunCfg := stun.Config{
			Enabled:      cfg.STUN.Enabled,
			ListenAddr:   cfg.STUN.ListenAddr,
			WorkerSize:   cfg.STUN.WorkerSize,
			BufferSize:   cfg.STUN.BufferSize,
			CacheTTL:     cfg.STUN.CacheTTL,
			SharedSecret: cfg.STUN.SharedSecret,
		}
		stunServer := stun.NewServer(stunCfg, stunStore, inf.Logger)
		if inf.MetricsReg != nil {
			stunServer.SetMetrics(stun.NewMetrics(inf.MetricsReg))
		}
		inf.GS.Register("stun-udp", 2, func(ctx context.Context) error { return stunServer.Stop() })

		go func() {
			if err := stunServer.Start(context.Background()); err != nil {
				inf.Logger.Error("STUN server error", zap.Error(err))
			}
		}()
		inf.Logger.Info("STUN UDP server enabled", zap.String("addr", cfg.STUN.ListenAddr))
	}

	return inf.WaitAndShutdown(errCh)
}

// acsConnReqSender adapts connreq.Dispatcher to the acs.ConnectionRequester interface.
// For post-session wake, we only use UDP (STUN-based) since the device's HTTP URL
// is not readily available in the ACS handler context.
type acsConnReqSender struct {
	dispatcher *connreq.Dispatcher
	isENB      bool
}

func (s *acsConnReqSender) Send(ctx context.Context, deviceSN, httpURL string) error {
	return s.dispatcher.Send(ctx, deviceSN, httpURL, "", s.isENB)
}

// parseStringSlice parses a comma-separated string into a slice.
func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
