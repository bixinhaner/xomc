package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger creates a new zap logger from configuration.
// Supports both console output and file output with rotation via lumberjack.
func NewLogger(cfg appconfig.LogConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	switch cfg.Format {
	case "console":
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Level, err)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Handle output paths
	if len(cfg.OutputPaths) > 0 {
		// Build custom cores for each output
		cores := make([]zapcore.Core, 0, len(cfg.OutputPaths))

		var encoder zapcore.Encoder
		if cfg.Format == "console" {
			encoder = zapcore.NewConsoleEncoder(zapCfg.EncoderConfig)
		} else {
			encoder = zapcore.NewJSONEncoder(zapCfg.EncoderConfig)
		}

		for _, path := range cfg.OutputPaths {
			var writer io.Writer
			switch path {
			case "stdout":
				writer = os.Stdout
			case "stderr":
				writer = os.Stderr
			default:
				// File output with rotation support
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return nil, fmt.Errorf("create log directory %s: %w", dir, err)
				}

				if cfg.Rotation.Enabled {
					writer = newLumberjackWriter(path, cfg.Rotation)
				} else {
					file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						return nil, fmt.Errorf("open log file %s: %w", path, err)
					}
					writer = file
				}
			}

			core := zapcore.NewCore(encoder, zapcore.AddSync(writer), zapCfg.Level)
			cores = append(cores, core)
		}

		// Combine all cores
		core := zapcore.NewTee(cores...)
		logger := zap.New(core,
			zap.AddCaller(),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		return logger, nil
	}

	// Fallback to default zap config if no output paths specified
	logger, err := zapCfg.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return logger, nil
}

// newLumberjackWriter creates a lumberjack writer for log rotation.
func newLumberjackWriter(path string, cfg appconfig.RotationConfig) io.Writer {
	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 20 // default 20MB
	}

	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7 // default 7 days
	}

	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 100 // default 100 files
	}

	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSize,    // megabytes
		MaxAge:     maxAge,     // days
		MaxBackups: maxBackups, // number of backups
		Compress:   cfg.Compress,
		LocalTime:  cfg.LocalTime,
	}
}

// ParseStringSlice parses a comma-separated string into a slice.
// Useful for environment variables that represent slices.
func ParseStringSlice(s string) []string {
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
