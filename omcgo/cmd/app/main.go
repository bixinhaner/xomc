package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/cmd/app/router"
	"github.com/omcgo/omcgo/internal/core/appconfig"
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

	app, err := initApp(context.Background(), &cfg)
	if err != nil {
		return err
	}
	defer app.Logger.Sync()
	app.Logger.Info("omcgo-app starting", zap.String("config", cfgPath))

	// Setup Gin engine
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	if err := router.Setup(engine, &router.Deps{
		PgPool:          app.PgPool,
		TsPool:          app.TsPool,
		Redis:           app.Redis,
		MinIO:           app.MinIO,
		EventBus:        app.EventBus,
		CmdQueue:        app.CmdQueue,
		CarrierRegistry: app.Carriers,
		Cfg:             &cfg,
		Logger:          app.Logger,
		GS:              app.GS,
		MetricsReg:      app.MetricsReg,
	}); err != nil {
		return fmt.Errorf("setup routes: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return app.ListenAndServe(engine, addr)
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
