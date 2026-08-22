package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/cmd/app/provider"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/core/realtime"
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

	rootCmd.Flags().String("config", "cmd/app/etc/config.dev.yaml", "configuration file path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runApp(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg appconfig.AppConfig
	if err := appconfig.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Handle LOG_OUTPUT_PATHS environment variable (comma-separated)
	if outputPaths := os.Getenv("OMCGO_LOG_OUTPUT_PATHS"); outputPaths != "" {
		cfg.Log.OutputPaths = parseStringSlice(outputPaths)
	}

	// P1-8: Validate JWT secret in production mode.
	// In non-dev/test environments, reject weak or default secrets.
	if err := validateJWTSecret(cfg.JWT.Secret); err != nil {
		return fmt.Errorf("JWT secret validation failed: %w", err)
	}

	// #2: 启动期凭证 guardrail —— 生产环境下检测到默认/占位/已泄露凭证即拒启，
	// 防止 omcgo123 / minioadmin / dps / REPLACE_ME 等随配置入 git 的密钥被直接
	// 部署到生产。dev/test 自动跳过。
	if err := appconfig.GuardProductionSecrets(
		appconfig.SecretCheck{Field: "db.dsn(password)", Value: cfg.DB.DSN, IsDSN: true},
		appconfig.SecretCheck{Field: "tsdb.dsn(password)", Value: cfg.TSDB.DSN, IsDSN: true},
		appconfig.SecretCheck{Field: "minio.access_key", Value: cfg.MinIO.AccessKey},
		appconfig.SecretCheck{Field: "minio.secret_key", Value: cfg.MinIO.SecretKey},
		appconfig.SecretCheck{Field: "jwt.secret", Value: cfg.JWT.Secret},
		appconfig.SecretCheck{Field: "conn_req.shared_secret", Value: cfg.ConnReq.SharedSecret},
	); err != nil {
		return fmt.Errorf("生产凭证校验失败: %w", err)
	}

	app, err := initApp(context.Background(), &cfg)
	if err != nil {
		return err
	}
	defer app.Logger.Sync()
	// provider.Setup may run heavyweight one-shot dictionary/model reconciliation
	// before the main HTTP server is ready. Expose liveness/metrics first so
	// deploy health gates and monitoring do not see a long blind startup window.
	app.StartMetrics()
	app.Logger.Info("omcgo-app starting", zap.String("config", cfgPath))

	// Setup Gin engine
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	// T-0178 S5 安全审计:multipart 上传内存上限 4 MiB(Gin 默认 32 MiB 太宽,
	// 配 N 并发上传时易耗内存)。单文件硬上限 1 MiB 在 parammodel.handler.go
	// UploadXML 内额外卡;此处 engine 层卡总 multipart 大小(N 个 form 字段 + 文件)。
	engine.MaxMultipartMemory = 4 << 20 // 4 MiB
	// 自定义 recovery 中间件取代 gin.Recovery()：panic 走 zap.Error 而非
	// stdlib log，确保 panic 同时进入 stdout（docker logs）和 zap output_paths
	// 配置的 app.log 文件，运维只在 docker logs 看不到 app.log 的体验消除。
	engine.Use(middleware.Recovery(app.Logger))

	if err := provider.Setup(engine, &provider.Container{
		PgPool:     app.PgPool,
		TsPool:     app.TsPool,
		Redis:      app.Redis,
		PMRedis:    app.PMRedis,
		MinIO:      app.MinIO,
		EventBus:   app.EventBus,
		Realtime:   realtime.NewCoreNATS(app.NATS.Conn),
		Deduper:    event.NewDeduper(app.Redis, 24*time.Hour, app.Logger),
		TaskSvc:    app.TaskSvc,
		Carriers:   app.Carriers,
		Cfg:        &cfg,
		Logger:     app.Logger,
		LogGate:    app.LogGate,
		GS:         app.GS,
		MetricsReg: app.MetricsReg,
		Health:     app.Health,
	}); err != nil {
		return fmt.Errorf("setup routes: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return app.ListenAndServe(engine, addr, cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout)
}

// defaultJWTSecret is the placeholder secret shipped in dev/test config files.
const defaultJWTSecret = "change-me-in-production-minimum-32-characters!!"

// validateJWTSecret checks that the JWT secret is strong enough for production use.
// In development or test mode (OMCGO_ENV=dev|test or GIN_MODE=debug), validation is skipped.
func validateJWTSecret(secret string) error {
	env := os.Getenv("OMCGO_ENV")
	ginMode := os.Getenv("GIN_MODE")

	// Skip validation for dev/test environments.
	// Empty OMCGO_ENV is treated as dev (consistent with entrypoint.sh default).
	if env == "" || env == "dev" || env == "test" || env == "development" || ginMode == "debug" || ginMode == "test" {
		return nil
	}

	if secret == "" {
		return fmt.Errorf("JWT secret must not be empty")
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters, got %d", len(secret))
	}
	if secret == defaultJWTSecret {
		return fmt.Errorf("JWT secret must not be the default placeholder value — set a unique secret via OMCGO_JWT_SECRET environment variable")
	}
	return nil
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
