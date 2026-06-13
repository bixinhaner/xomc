package logger

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.opentelemetry.io/otel/trace"
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
					writer = NewLumberjackWriter(path, cfg.Rotation)
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

// NewLumberjackWriter 构造日志轮转写入器，支持两种归档管理模式：
//
//   Compactor 模式（cfg.KeepUncompressed > 0）：
//     - lumberjack 仅做切割（MaxBackups=0 / MaxAge=0 / Compress=false 都禁用）
//     - 后台 compactor goroutine 每分钟扫描归档目录：
//         · mtime > MaxAgeDays → 删
//         · 最新 KeepUncompressed 个 → 保持 .log 形式，rename 到分钟精度
//         · 其余 .log → gzip 为 .log.gz，删原文件
//     - 用于 app/acs/worker 三个主日志 + acs protocol_log
//
//   Legacy 模式（cfg.KeepUncompressed == 0）：
//     - 完全沿用 lumberjack 原生 MaxBackups + MaxAge + Compress 行为
//     - 仅在极少数需保留 lumberjack 原生归档语义的场景下使用
//
// MaxSizeMB 与 RotateInterval 是 OR 关系：任一满足都触发切割。空文件保护防止
// 低流量环境下每个 tick 产生空 .gz 归档。
func NewLumberjackWriter(path string, cfg appconfig.RotationConfig) io.Writer {
	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 50 // default 50MB
	}
	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7 // default 7 days
	}

	useCompactor := cfg.KeepUncompressed > 0

	lj := &lumberjack.Logger{
		Filename:  path,
		MaxSize:   maxSize,
		LocalTime: cfg.LocalTime,
	}
	if !useCompactor {
		// Legacy：lumberjack 自己管 cleanup + compress
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 30
		}
		lj.MaxAge = maxAge
		lj.MaxBackups = maxBackups
		lj.Compress = cfg.Compress
	}
	// Compactor 模式下故意不设 MaxBackups/MaxAge/Compress，让 lumberjack 仅做切割

	if cfg.RotateInterval > 0 {
		startTimedRotation(lj, path, cfg.RotateInterval)
	}
	if useCompactor {
		startCompactor(lj, path, cfg.KeepUncompressed, time.Duration(maxAge)*24*time.Hour)
	}

	return lj
}

// startTimedRotation 启动后台 goroutine 周期性强制切割 lumberjack 日志文件。
//
// goroutine 生命周期与进程一致——OMC 进程没有热替换 logger 的场景，所以不提供 Stop()。
// 空文件保护防止低流量环境下每个 tick 都产生 ~20 字节的空 .gz 归档。
func startTimedRotation(lj *lumberjack.Logger, path string, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			info, err := os.Stat(path)
			if err != nil || info.Size() == 0 {
				continue // 文件不存在或为空 — 跳过这次轮转
			}
			_ = lj.Rotate()
		}
	}()
}

// lumberjackBackupTimeRe 匹配 lumberjack 原生归档名的秒+毫秒部分：
//   acs-2026-05-15T03-55-16.790.log    → 捕获 "-16.790"
//   acs-2026-05-15T03-55-16.790.log.gz → 同上（.gz 后缀走第二个正则）
var (
	lumberjackBackupTimeRe = regexp.MustCompile(`(-\d{4}-\d{2}-\d{2}T\d{2}-\d{2})-\d{2}\.\d{3}(\.log)$`)
	lumberjackBackupTimeGz = regexp.MustCompile(`(-\d{4}-\d{2}-\d{2}T\d{2}-\d{2})-\d{2}\.\d{3}(\.log\.gz)$`)
)

// truncateToMinute 把 lumberjack 秒.毫秒精度的归档名截断到分钟精度。
// 已是分钟精度的文件原样返回（regex 不匹配）。
func truncateToMinute(path string) string {
	if lumberjackBackupTimeRe.MatchString(path) {
		return lumberjackBackupTimeRe.ReplaceAllString(path, "$1$2")
	}
	if lumberjackBackupTimeGz.MatchString(path) {
		return lumberjackBackupTimeGz.ReplaceAllString(path, "$1$2")
	}
	return path
}

// startCompactor 启动后台归档维护 goroutine：每 1 分钟扫描一次。
//   - 删除 mtime 早于 cutoff 的归档
//   - 保留最新 keepUncompressed 个 .log 文件不压缩（rename 到分钟精度供 tail/less 直读）
//   - 其余 .log 归档压缩为 .log.gz 并删除原文件
//
// keepUncompressed/maxAge 是启动期 YAML 值；每个 tick 通过 effectiveRotation 与运行期
// sys_configs override 合并（见 rotation.go），故 log.rotation 改完 ≤1 分钟生效。lj 用于
// max_size override 的 size 切割（enforceMaxSize 调 lumberjack 线程安全 Rotate）。
func startCompactor(lj *lumberjack.Logger, path string, keepUncompressed int, maxAge time.Duration) {
	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		for range t.C {
			keep, age, maxSizeMB := effectiveRotation(keepUncompressed, maxAge)
			if maxSizeMB > 0 {
				enforceMaxSize(lj, path, int64(maxSizeMB)*1024*1024)
			}
			compactOnce(path, keep, age)
		}
	}()
}

type backupEntry struct {
	path  string
	mtime time.Time
	isGz  bool
}

// compactOnce 执行一次归档目录扫描 + 整理。
func compactOnce(path string, keepUncompressed int, maxAge time.Duration) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Glob 所有归档（注意：active 文件 <name><ext> 不匹配，因为模式要求至少一个 '-'）
	rawPattern := filepath.Join(dir, name+"-*"+ext)
	gzPattern := filepath.Join(dir, name+"-*"+ext+".gz")

	raws, _ := filepath.Glob(rawPattern)
	gzs, _ := filepath.Glob(gzPattern)

	all := make([]backupEntry, 0, len(raws)+len(gzs))
	for _, f := range raws {
		// filepath.Glob 用 "*" 会匹配 "-*.log" 也命中 "-*.log.gz" 因为 "*.log" 是字面后缀
		// Go 的 filepath.Glob "*" 不跨 PathSeparator，但确实贪婪——需要排除 .gz 结尾
		if strings.HasSuffix(f, ".gz") {
			continue
		}
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		all = append(all, backupEntry{path: f, mtime: info.ModTime(), isGz: false})
	}
	for _, f := range gzs {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		all = append(all, backupEntry{path: f, mtime: info.ModTime(), isGz: true})
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].mtime.After(all[j].mtime)
	})

	cutoff := time.Now().Add(-maxAge)

	for i := range all {
		e := &all[i]

		// 超龄整删（不管是否压缩）
		if e.mtime.Before(cutoff) {
			if err := os.Remove(e.path); err != nil {
				stdlog.Printf("[logger compactor] remove expired %s: %v", e.path, err)
			}
			continue
		}

		if i < keepUncompressed {
			// 最新 N 个：保持 .log 不压。若名字含秒.毫秒精度则 rename 到分钟。
			if !e.isGz {
				if newPath := truncateToMinute(e.path); newPath != e.path {
					if final, ok := safeRename(e.path, newPath); ok {
						e.path = final
					}
				}
			}
			continue
		}

		// 其余：必须压缩
		if e.isGz {
			continue // 已压缩 — 不动
		}
		// 先 rename 到分钟精度（避免 .gz 名字里仍带秒.毫秒）
		if newPath := truncateToMinute(e.path); newPath != e.path {
			if final, ok := safeRename(e.path, newPath); ok {
				e.path = final
			}
		}
		if err := gzipFile(e.path); err != nil {
			stdlog.Printf("[logger compactor] gzip %s: %v", e.path, err)
		}
	}
}

// safeRename 把 src rename 到 target；若 target 已存在则附加 -N 后缀避免覆盖。
// 返回最终落地路径与是否成功。
func safeRename(src, target string) (string, bool) {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.Rename(src, target); err != nil {
			return src, false
		}
		return target, true
	}
	if src == target {
		return src, true
	}
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 1; i < 100; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			if err := os.Rename(src, candidate); err != nil {
				return src, false
			}
			return candidate, true
		}
	}
	return src, false
}

// gzipFile 把 path 内容 gzip 写到 path+".gz"，成功后删除源文件。
// 中途失败会清理掉残留的 .gz 半成品，保留源文件以便下轮重试。
func gzipFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()

	gzPath := path + ".gz"
	dst, err := os.OpenFile(gzPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}

	gz := gzip.NewWriter(dst)
	if _, err := io.Copy(gz, src); err != nil {
		_ = gz.Close()
		_ = dst.Close()
		_ = os.Remove(gzPath)
		return fmt.Errorf("compress: %w", err)
	}
	if err := gz.Close(); err != nil {
		_ = dst.Close()
		_ = os.Remove(gzPath)
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		_ = os.Remove(gzPath)
		return fmt.Errorf("sync dst: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(gzPath)
		return fmt.Errorf("close dst: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove src: %w", err)
	}
	return nil
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

// L returns a logger with request_id + trace_id/span_id fields populated from
// context. request_id 由 RequestID middleware 写入 context；trace_id / span_id
// 由 OTel Tracing middleware 透过 propagation.TraceContext 注入。
//
// 三件套同时存在时单次 logger.With 调用合并以避免多次 clone（zap 内部对
// .With 链没做去重）。任一字段为空就不写，下游 Loki/Grafana 仍能查询。
//
// trace-to-logs 关联：trace_id 字段写入后，Grafana Tempo 数据源点 span →
// 自动跳 Loki "{service=...} |= \"<traceID>\"" 精确匹配（详见 T-0155 §3）。
func L(ctx context.Context) *zap.Logger {
	// Get the global logger or fallback to nop logger
	logger := zap.L()
	if logger == nil {
		return zap.NewNop()
	}

	fields := make([]zap.Field, 0, 3)
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		fields = append(fields,
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		)
	}
	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields...)
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
