package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/download"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/acs/upload"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	miniocomp "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/health"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
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

	// #2: 启动期凭证 guardrail —— 生产环境下检测到默认/占位/已泄露凭证即拒启。
	// ACS 持有 PG / MinIO / STUN 共享密钥；dev/test 自动跳过。
	if err := appconfig.GuardProductionSecrets(
		appconfig.SecretCheck{Field: "db.dsn(password)", Value: cfg.DB.DSN, IsDSN: true},
		appconfig.SecretCheck{Field: "minio.access_key", Value: cfg.MinIO.AccessKey},
		appconfig.SecretCheck{Field: "minio.secret_key", Value: cfg.MinIO.SecretKey},
		appconfig.SecretCheck{Field: "stun.shared_secret", Value: cfg.STUN.SharedSecret},
	); err != nil {
		return fmt.Errorf("生产凭证校验失败: %w", err)
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

	// T-XXX: ACS 端 standardPath → privatePath 翻译服务（取代 App fanout 翻译）。
	//   - ProductRegistry: productClass → product 路由（regex patterns）
	//   - ParamRegistry: discovered → default 双源映射
	//   - DeviceRepo: 按 SN 查 product_class / firmware_version
	// 任一依赖初始化失败 → PathTranslator 为 nil-safe 透传（设备直收 standardPath，CPE 必返 9005;
	// 与改造前 fanout fallback 行为一致，所以仅 WARN 不阻塞 ACS 启动）。
	pathTranslator := buildACSPathTranslator(context.Background(), inf)

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

	// issue #65（Option B）— ACS 横扩去进程态：把 4 类会话副作用状态接到共享 Redis，
	// 使 ACS 在多实例无亲和（nginx/k8s 无需 sticky session）部署下不漂移/泄漏/失败。
	// ACS 已硬依赖 Redis（会话/任务/STUN 均在 Redis），这里保持一致。
	if inf.Redis != nil {
		// 1) 准入计数 → 全局 Redis sorted set（TTL 自愈丢失的 Release）。
		deps.Admission = acs.NewRedisAdmissionController(inf.Redis, cfg.Session.MaxConcurrent, inf.Logger)
		// 2) 设备当前活跃会话指针 → Redis（跨实例孤儿会话清理）。TTL 取会话超时 + 余量。
		deviceSessionTTL := cfg.Session.Timeout + 5*time.Minute
		deps.DeviceSessionStore = acs.NewRedisDeviceSessionStore(inf.Redis, deviceSessionTTL)
		// 3) ConnectionRequestURL → Redis 共享存储（镜像 STUN store，跨实例 HTTP 唤醒回退）。
		deps.ConnReqURLStore = acs.NewRedisConnReqURLStore(inf.Redis, 0)
		// 4) Digest nonce → Redis（SETEX + GETDEL，跨实例 Challenge/Authenticate 不丢 nonce）。
		deps.Authenticator = auth.NewAuthenticatorWithRedis(cfg.Auth.Mode, cfg.Auth.Username, cfg.Auth.Password, inf.Redis)
		inf.Logger.Info("ACS stateless redis state enabled (issue #65 option B)",
			zap.Int64("max_concurrent", cfg.Session.MaxConcurrent),
			zap.Duration("device_session_ttl", deviceSessionTTL))
	} else {
		inf.Logger.Warn("ACS redis unavailable; falling back to per-instance in-process session state (single-instance only)")
	}

	// 装配 /readyz 依赖检查器：仅勾选实际连接成功的基础设施，避免在精简部署
	// （未配 PG/MinIO）下误报。检查列表与 components.HealthChecker.Register 同源，
	// 但这里独立组装是为了让 ACS HTTP server（非 metrics 端口）也能直接探测。
	deps.ReadinessCheckers = buildACSReadinessCheckers(inf)
	deps.PathTranslator = pathTranslator
	transferPolicy := transfercfg.NewPolicy(
		transfercfg.DefaultsFromACSConfig(cfg),
		newTransferSysConfigLookup(admin.NewPgSysConfigRepository(inf.PgPool)),
	)
	deps.TransferConfigProvider = transferPolicy
	if inf.EventBus != nil {
		sub, err := inf.EventBus.Subscribe(
			event.SubjectSysConfigSaved,
			transfercfg.HandleSysConfigSavedEvent(transferPolicy, inf.Logger.Named("transfercfg")),
		)
		if err != nil {
			inf.Logger.Warn("subscribe sys config saved events for transfer config", zap.Error(err))
		} else {
			inf.GS.Register("transfercfg-sysconfig-sub", 1, func(ctx context.Context) error {
				_ = ctx
				return sub.Unsubscribe()
			})
		}
	}

	deps.RPCDispatcher = rpc.NewDispatcher(rpc.DispatcherConfig{TransferConfigProvider: transferPolicy})

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
		uploadHandler.SetRuntimeProvider(transferPolicy)
		// T-0074: enable streaming compression for FileTypeConfig backup uploads.
		// PolicyGetter pulls live policy from PG; metrics track ratio/duration.
		// Both args are nil-safe — PolicyService.Get always returns DefaultPolicy
		// when the row does not exist, so compression activates only when the
		// operator has explicitly set EnableCompression=true.
		backupPolicyRepo := backup.NewPgPolicyRepository(inf.PgPool)
		backupPolicySvc := backup.NewPolicyService(backupPolicyRepo, inf.Logger)
		backupPolicyMetrics := backup.NewPolicyMetrics(inf.MetricsReg)
		uploadHandler.SetCompression(backupPolicySvc, backupPolicyMetrics)

		// T-0075: optional backup encryption (AES-256-GCM envelope). The KEK
		// comes from OMC_BACKUP_ENCRYPTION_KEY env var (64 hex chars / 32B).
		// If unset, encryption stays disabled (Available()=false) — operators
		// who want encryption must restart with the env var set. An invalid
		// value (bad hex / wrong length) is a startup config error.
		backupKeyProvider, kpErr := backup.NewEnvKeyProvider()
		if kpErr != nil {
			return fmt.Errorf("backup encryption key: %w", kpErr)
		}
		var backupEncryptor backup.Encryptor
		if backupKeyProvider.Available() {
			enc, encErr := backup.NewEncryptor("AES-256-GCM", backupKeyProvider)
			if encErr != nil {
				return fmt.Errorf("backup encryptor: %w", encErr)
			}
			backupEncryptor = enc
			uploadHandler.SetEncryption(backupEncryptor)
			inf.Logger.Info("backup encryption enabled",
				zap.String("algorithm", "AES-256-GCM"))
		} else {
			inf.Logger.Info("backup encryption disabled (OMC_BACKUP_ENCRYPTION_KEY unset)")
		}

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
		downloadHandler.SetRuntimeProvider(transferPolicy)
		// T-0072: enable on-the-fly decompression for compressed backup objects
		// (.gz/.zst/.lz4/.bz2). Mirrors the streaming compression added in T-0074
		// on the upload side. Metrics are nil-safe.
		downloadHandler.SetDecompressMetrics(download.NewDecompressMetrics(inf.MetricsReg))
		// T-0075: enable on-the-fly decryption for `.enc` objects.
		if backupEncryptor != nil {
			downloadHandler.SetEncryption(backupEncryptor, backupPolicyMetrics)
			// T-0089: bound concurrent decrypt buffer allocations
			// (64MB × N memory amplification). Tunable via env var; 0
			// disables (preserves T-0075 unbounded behaviour for
			// deployments that prefer rate-limit middleware alone).
			limit := decryptDefaultLimit(inf.Logger)
			sem := download.NewDecryptSemaphore(limit, decryptDefaultTimeout, backupPolicyMetrics)
			downloadHandler.SetDecryptSemaphore(sem)
			// Review MED-2 fix: surface clamp visibility at startup —
			// previously this was a metric pulse, now it's a structured
			// log so misconfig at boot is auditable without polluting
			// per-acquire rejection telemetry.
			if effective := sem.ClampedLimit(); effective != 0 && effective < limit {
				inf.Logger.Warn("OMC_BACKUP_DECRYPT_CONCURRENCY exceeds hard cap; clamped",
					zap.Int("requested", limit),
					zap.Int("clamped", effective),
					zap.Int("hard_cap", download.DecryptSemaphoreHardCap))
			}
			inf.Logger.Info("backup decrypt semaphore configured",
				zap.Int("limit", limit),
				zap.Int("effective", sem.ClampedLimit()),
				zap.Duration("timeout", decryptDefaultTimeout))
		}
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

	// T-0137 / M2: TR069 报文跟踪 — ACS 端旁路 hook。
	//   - WhitelistCache 启动加载 + 订阅 trace.task.* 实时增删 SN（30s 兜底轮询保活）
	//   - Service 注入 EventBus，EnqueueCapture 改为 publish trace.message.captured 到 JetStream，
	//     worker 群组消费 + 批量落库（ACS 不再写 PG，hot path < 1ms）
	if inf.PgPool != nil {
		traceRepo := trace.NewPgRepository(inf.PgPool)
		traceSvc := trace.NewService(traceRepo, trace.DefaultConfig(), inf.Logger)
		if inf.EventBus != nil {
			traceSvc.SetEventBus(inf.EventBus)
		}
		// M3-01：Prometheus 指标（ACS 端 capture latency histogram + drop 计数）
		traceSvc.SetMetrics(trace.NewMetrics(inf.MetricsReg))
		traceSvc.Start(context.Background())

		wlCfg := trace.DefaultWhitelistConfig()
		if inf.EventBus != nil {
			wlCfg = trace.M2WhitelistConfig()
		}
		traceWL := trace.NewWhitelistCache(traceRepo, wlCfg, inf.Logger)
		if err := traceWL.Start(context.Background()); err != nil {
			inf.Logger.Warn("trace whitelist initial load failed; capture disabled", zap.Error(err))
		} else {
			var unsubTrace func()
			if inf.EventBus != nil {
				if cleanup, subErr := traceWL.Subscribe(inf.EventBus); subErr != nil {
					inf.Logger.Warn("trace whitelist NATS subscribe failed; falling back to poll-only",
						zap.Error(subErr))
				} else {
					unsubTrace = cleanup
				}
			}
			deps.TraceWhitelist = traceWL
			deps.TraceService = traceSvc
			inf.GS.Register("trace-capture", 3, func(ctx context.Context) error {
				if unsubTrace != nil {
					unsubTrace()
				}
				traceWL.Stop()
				traceSvc.Stop()
				return nil
			})
			inf.Logger.Info("TR069 message trace capture enabled (T-0137 M2)",
				zap.Bool("nats", inf.EventBus != nil))
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

// buildACSPathTranslator 构造 ACS 端的 standardPath → privatePath 翻译服务。
//
// 启动期一次性 Refresh ProductRegistry 与 ParamRegistry —— 把 product_class_patterns
// 与字典版本号同步到本进程。运行期通过 ensureFresh 协议感知其他实例的 BumpVersion。
//
// 任一依赖（PG / Redis 缺失）→ 返回 nil，Handler.translateTaskParamsInPlace 退化为透传。
// 仅 WARN 不阻塞启动 —— 与改造前 App fanout 失败 fallback 的行为一致。
func buildACSPathTranslator(ctx context.Context, inf *components.Infra) *acs.PathTranslationService {
	if inf == nil || inf.PgPool == nil {
		inf.Logger.Warn("ACS path translator disabled: PgPool not available")
		return nil
	}
	logger := inf.Logger.Named("acs-path-translator")

	// ProductRegistry — productClass → product 路由（regex patterns）
	productRepo := product.NewPgRepository(inf.PgPool)
	productMetrics := product.NewRegistryMetrics(inf.MetricsReg)
	productCache := product.Cache(product.NopCache{})
	if inf.Redis != nil {
		productCache = product.NewRedisCache(inf.Redis)
	}
	productReg := product.NewRegistry(productRepo, productCache, productMetrics, logger)

	refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := productReg.Refresh(refreshCtx); err != nil {
		logger.Warn("product registry refresh failed; path translation disabled", zap.Error(err))
		return nil
	}

	// ParamRegistry — discovered → default 双源映射（依赖 productReg 反查 ParamModelID）
	paramRepo := parammodel.NewPgRepository(inf.PgPool)
	paramMetrics := parammodel.NewRegistryMetrics(inf.MetricsReg)
	paramCache := parammodel.Cache(parammodel.NopCache{})
	if inf.Redis != nil {
		paramCache = parammodel.NewRedisCache(inf.Redis)
	}
	paramReg := parammodel.NewRegistry(paramRepo, paramCache, productReg, paramMetrics, logger)
	if err := paramReg.Refresh(refreshCtx); err != nil {
		// ParamRegistry Refresh 仅清 L1 + BumpVersion，PG 错误较罕见 — WARN 后继续，
		// 后续 GetByProduct 仍会按需 read-through。
		logger.Warn("param registry refresh failed (non-fatal); continuing", zap.Error(err))
	}

	deviceRepo := device.NewPgDeviceRepository(inf.PgPool)
	return acs.NewPathTranslationService(deviceRepo, productReg, paramReg, inf.Logger)
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
//
// Rotation 走 logger.NewLumberjackWriter 共用入口 — 按 cfg.Rotation.KeepUncompressed
// 自动路由 compactor / legacy 双模式（与主 acs.log 同款）。protocol_log 体积大且
// 含 SOAP 凭据敏感，dev/test 用 compactor 自动 gzip 压缩 + max_age 删旧；prod 默认
// enabled=false 等保合规。
func newProtocolLogger(cfg appconfig.ProtocolLogConfig) (*zap.Logger, error) {
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create protocol log directory %s: %w", dir, err)
	}

	var writer io.Writer
	if cfg.Rotation.Enabled {
		writer = logger.NewLumberjackWriter(cfg.FilePath, cfg.Rotation)
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

func newTransferSysConfigLookup(repo interface {
	GetByKey(context.Context, string, string) (*admin.SysConfig, error)
}) transfercfg.SysConfigLookup {
	return func(ctx context.Context, category, key string) (string, bool) {
		cfg, err := repo.GetByKey(ctx, category, key)
		if err != nil || cfg == nil {
			return "", false
		}
		return cfg.Value, true
	}
}

// decryptDefaultLimit reads OMC_BACKUP_DECRYPT_CONCURRENCY (T-0089). Empty
// returns 8 (sane default for ~512MB peak buffer footprint). Invalid values
// log warn and fall back to 8 — boot must not panic on operator typo. Zero
// or negative disables the semaphore entirely (preserves T-0075 baseline);
// review MED-1 fix emits a loud WARN in that case so an operator who
// typed -1 thinking "no limit" sees the **opposite** semantics in logs
// before deploying.
const decryptDefaultTimeout = 30 * time.Second

func decryptDefaultLimit(logger *zap.Logger) int {
	const fallback = 8
	v := os.Getenv("OMC_BACKUP_DECRYPT_CONCURRENCY")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		logger.Warn("OMC_BACKUP_DECRYPT_CONCURRENCY invalid; using default",
			zap.String("value", v), zap.Int("default", fallback), zap.Error(err))
		return fallback
	}
	if n <= 0 {
		logger.Warn("OMC_BACKUP_DECRYPT_CONCURRENCY <= 0 disables semaphore — backup decrypt 64MB×N memory amplification UNBOUNDED; set positive integer to enable cap",
			zap.Int("value", n))
	}
	return n
}
