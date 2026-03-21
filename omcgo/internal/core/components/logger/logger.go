package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// contextKey for request_id storage in context
type contextKey int

const (
	requestIDKey contextKey = iota
)

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context.
// Returns an empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

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

// newLumberjackWriter creates a writer for log rotation.
// Supports both time-based rotation (RotateInterval) and size-based rotation (MaxSizeMB).
// If RotateInterval is set, uses time-based rotation; otherwise uses size-based rotation.
func newLumberjackWriter(path string, cfg appconfig.RotationConfig) io.Writer {
	// Time-based rotation takes precedence
	if cfg.RotateInterval > 0 {
		return newTimeBasedRotator(path, cfg)
	}

	// Fallback to size-based rotation
	return newSizeBasedRotator(path, cfg)
}

// newTimeBasedRotator creates a time-based log rotator using file-rotatelogs.
func newTimeBasedRotator(path string, cfg appconfig.RotationConfig) io.Writer {
	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7 // default 7 days
	}

	// Generate rotation filename pattern with timestamp
	// e.g., /var/log/omcgo/acs.log -> /var/log/omcgo/acs.%Y%m%d%H%M.log
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	rotationPattern := filepath.Join(dir, name+".%Y%m%d%H%M"+ext)

	opts := []rotatelogs.Option{
		rotatelogs.WithMaxAge(time.Duration(maxAge) * 24 * time.Hour),
		rotatelogs.WithRotationTime(cfg.RotateInterval),
	}

	if cfg.LocalTime {
		opts = append(opts, rotatelogs.WithClock(rotatelogs.Local))
	} else {
		opts = append(opts, rotatelogs.WithClock(rotatelogs.UTC))
	}

	// Create symlink to current log file
	opts = append(opts, rotatelogs.WithLinkName(path))

	writer, err := rotatelogs.New(rotationPattern, opts...)
	if err != nil {
		// Fallback to size-based rotation if time-based fails
		return newSizeBasedRotator(path, cfg)
	}

	return writer
}

// newSizeBasedRotator creates a size-based log rotator using lumberjack.
func newSizeBasedRotator(path string, cfg appconfig.RotationConfig) io.Writer {
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

// L returns a logger with request_id field if present in context.
// This enables automatic request tracing across all log entries.
func L(ctx context.Context) *zap.Logger {
	// Get the global logger or fallback to nop logger
	logger := zap.L()
	if logger == nil {
		return zap.NewNop()
	}

	// Add request_id if present
	if requestID := GetRequestID(ctx); requestID != "" {
		return logger.With(zap.String("request_id", requestID))
	}
	return logger
}

// Info logs at INFO level with context-aware request_id.
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Info(msg, fields...)
}

// Error logs at ERROR level with context-aware request_id.
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Error(msg, fields...)
}

// Debug logs at DEBUG level with context-aware request_id.
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Debug(msg, fields...)
}

// Warn logs at WARN level with context-aware request_id.
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Warn(msg, fields...)
}
