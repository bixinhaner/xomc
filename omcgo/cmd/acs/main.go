package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/acs/stun"
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

	acsServer := acs.NewACSServer(cfg, deps)
	inf.GS.Register("acs-http", 1, func(ctx context.Context) error { return acsServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		errCh <- acsServer.Start()
	}()

	// Start STUN UDP server for NAT traversal and Connection Request
	if cfg.STUN.Enabled {
		stunStore := stun.NewStore(inf.Redis, inf.Logger)
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
