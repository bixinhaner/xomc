package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/download"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/acs/upload"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components"
	miniocomp "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/health"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	// TaskService（必需）：统一任务队列，Redis + PostgreSQL 双写
	if inf.PgPool == nil {
		return fmt.Errorf("PostgreSQL connection required for ACS task service")
	}
	taskQueue := task.NewRedisTaskQueue(inf.Redis)
	taskRepo := task.NewPgTaskRepository(inf.PgPool)
	taskService := task.NewTaskService(taskQueue, taskRepo, inf.Logger)
	// Broadcast terminal task states so APP/Worker subscribers (MML ResultAggregator)
	// can update mml_tasks without being in the ACS process.
	if inf.EventBus != nil {
		taskService.SetEventBus(inf.EventBus)
	}
	inf.Logger.Info("task service initialized")

	// Get request ID prefix from config, default to "acs"
	requestIDPrefix := cfg.RequestIDPrefix
	if requestIDPrefix == "" {
		requestIDPrefix = "acs"
	}

	deps := acs.NewDefaultDeps(
		sessionStore,
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

	// 装配 /readyz 依赖检查器：仅勾选实际连接成功的基础设施，避免在精简部署
	// （未配 PG/MinIO）下误报。检查列表与 components.HealthChecker.Register 同源，
	// 但这里独立组装是为了让 ACS HTTP server（非 metrics 端口）也能直接探测。
	deps.ReadinessCheckers = buildACSReadinessCheckers(inf)

	// Override dispatcher with download config for MinIO path → HTTP URL translation.
	if cfg.Download.BaseURL != "" {
		deps.RPCDispatcher = rpc.NewDispatcher(rpc.DispatcherConfig{
			DownloadBaseURL: cfg.Download.BaseURL,
			DownloadPath:    cfg.Download.Path,
			DownloadUser:    cfg.Download.Username,
			DownloadPass:    cfg.Download.Password,
		})
	}

	// Setup upload handler for CPE file upload (PM/MR/DataModel files).
	if inf.MinIO != nil {
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
		// T-0074: enable streaming compression for FileTypeConfig backup uploads.
		// PolicyGetter pulls live policy from PG; metrics track ratio/duration.
		// Both args are nil-safe — PolicyService.Get always returns DefaultPolicy
		// when the row does not exist, so compression activates only when the
		// operator has explicitly set EnableCompression=true.
		backupPolicyRepo := backup.NewPgPolicyRepository(inf.PgPool)
		backupPolicySvc := backup.NewPolicyService(backupPolicyRepo, inf.Logger)
		backupPolicyMetrics := backup.NewPolicyMetrics(inf.MetricsReg)
		uploadHandler.SetCompression(backupPolicySvc, backupPolicyMetrics)
		deps.UploadHandler = uploadHandler
		deps.UploadConfig = &cfg.Upload
		inf.Logger.Info("upload handler enabled with backup compression",
			zap.String("username", cfg.Upload.Username))

		// Setup download handler for CPE file download (MinIO -> CPE proxy).
		downloadHandler := download.NewHandler(
			inf.MinIO,
			cfg.Download.Username, cfg.Download.Password,
			inf.Logger,
		)
		// T-0072: enable on-the-fly decompression for compressed backup objects
		// (.gz/.zst/.lz4/.bz2). Mirrors the streaming compression added in T-0074
		// on the upload side. Metrics are nil-safe.
		downloadHandler.SetDecompressMetrics(download.NewDecompressMetrics(inf.MetricsReg))
		deps.DownloadHandler = downloadHandler
		deps.DownloadConfig = &cfg.Download
		inf.Logger.Info("download handler enabled with backup decompression",
			zap.String("username", cfg.Download.Username))
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

	// 协议交互日志：独立的 zap logger 写入专用文件，记录完整 XML
	if cfg.ProtocolLog.Enabled && cfg.ProtocolLog.FilePath != "" {
		protocolLogger, err := newProtocolLogger(cfg.ProtocolLog)
		if err != nil {
			inf.Logger.Error("failed to create protocol logger", zap.Error(err))
		} else {
			deps.ProtocolLogger = protocolLogger
			deps.MaxBodySize = cfg.ProtocolLog.MaxBodySize
			inf.Logger.Info("protocol logging enabled",
				zap.String("file_path", cfg.ProtocolLog.FilePath),
				zap.Int("max_body_size", cfg.ProtocolLog.MaxBodySize))
		}
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

// buildACSReadinessCheckers 收集 ACS 进程实际依赖的基础设施，构造 /readyz 检查列表。
// 任一依赖未配置（如未连 MinIO）即不加入，避免在精简部署下误报 503。
func buildACSReadinessCheckers(inf *components.Infra) []health.Checker {
	var checkers []health.Checker
	if inf.Redis != nil {
		client := inf.Redis
		checkers = append(checkers, health.NewChecker("redis", func(ctx context.Context) error {
			return client.Ping(ctx).Err()
		}))
	}
	if inf.PgPool != nil {
		pool := inf.PgPool
		checkers = append(checkers, health.NewChecker("postgres", func(ctx context.Context) error {
			return pool.Ping(ctx)
		}))
	}
	if inf.NATS != nil {
		nc := inf.NATS
		checkers = append(checkers, health.NewChecker("nats", func(ctx context.Context) error {
			return nc.HealthCheck()
		}))
	}
	if inf.MinIO != nil {
		client := inf.MinIO
		checkers = append(checkers, health.NewChecker("minio", func(ctx context.Context) error {
			return miniocomp.MinIOHealthCheck(ctx, client)
		}))
	}
	return checkers
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

// newProtocolLogger creates a dedicated zap logger for ACS protocol interaction logging.
// It writes structured JSON to a separate file with its own rotation settings.
func newProtocolLogger(cfg appconfig.ProtocolLogConfig) (*zap.Logger, error) {
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create protocol log directory %s: %w", dir, err)
	}

	var writer io.Writer
	if cfg.Rotation.Enabled {
		maxSize := cfg.Rotation.MaxSizeMB
		if maxSize <= 0 {
			maxSize = 50
		}
		maxAge := cfg.Rotation.MaxAgeDays
		if maxAge <= 0 {
			maxAge = 7
		}
		maxBackups := cfg.Rotation.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 5
		}
		writer = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			MaxBackups: maxBackups,
			Compress:   cfg.Rotation.Compress,
			LocalTime:  cfg.Rotation.LocalTime,
		}
	} else {
		f, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open protocol log file %s: %w", cfg.FilePath, err)
		}
		writer = f
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(writer),
		zap.InfoLevel,
	)

	return zap.New(core), nil
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
