package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omcgo/omcgo/internal/acs"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/bootstrap"
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

	app, err := bootstrap.InitForACS(context.Background(), &cfg)
	if err != nil {
		return err
	}
	defer app.Logger.Sync()
	app.Logger.Info("omcgo-acs starting", zap.String("config", cfgPath))

	// Create ACS-specific components
	sessionStore := acs.NewRedisSessionStore(app.Redis, cfg.Session.Timeout)
	cmdQueue := cmdqueue.NewRedisCommandQueue(app.Redis)

	deps := acs.NewDefaultDeps(
		sessionStore,
		cmdQueue,
		app.EventBus,
		cfg.Auth.Mode, cfg.Auth.Username, cfg.Auth.Password,
		cfg.RateLimit,
		cfg.Session.MaxConcurrent,
		app.MetricsReg,
		app.Logger,
	)

	acsServer := acs.NewACSServer(cfg, deps)
	app.GS.Register("acs-http", 1, func(ctx context.Context) error { return acsServer.Shutdown(ctx) })

	errCh := make(chan error, 1)
	go func() {
		errCh <- acsServer.Start()
	}()

	return app.WaitAndShutdown(errCh)
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
