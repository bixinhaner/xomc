package components

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/response"
)

var (
	startTime = time.Now()
	version   = "dev"
	buildDate = "unknown"
)

const defaultAppLogPath = "/app/logs"

// SystemInfo 表示 GET /api/v1/system/info 接口返回的运行时信息。
// 用于管理单页明小板展示服务基本状态，不需要登录。
type SystemInfo struct {
	Version     string          `json:"version"`
	BuildDate   string          `json:"build_date"`
	ServerTime  string          `json:"server_time"`
	UptimeHours float64         `json:"uptime_hours"`
	DBStatus    string          `json:"db_status"`
	CacheStatus string          `json:"cache_status"`
	Storage     []StorageMetric `json:"storage"`
}

type filesystemStats struct {
	blockSize       uint64
	blocks          uint64
	availableBlocks uint64
	files           uint64
	availableFiles  uint64
}

type filesystemStatfs func(path string) (filesystemStats, error)

// SystemInfoHandler 提供 GET /api/v1/system/info 接口。
// 返回服务版本、运行时长、DB 和 Redis 连接状态，不需要认证，与 /healthz 互补。
// DBStatus/CacheStatus 返回 "normal" 或 "error"，3 秒超时自动展示 error。
type SystemInfoHandler struct {
	pgPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	logger      *zap.Logger
	diskPaths   []string
	statfs      filesystemStatfs
	now         func() time.Time
	storage     StorageCollector
}

// SetStorageCollector adds source-specific storage metrics. A nil collector is
// valid and keeps the endpoint usable when monitoring is not configured.
func (h *SystemInfoHandler) SetStorageCollector(collector StorageCollector) {
	h.storage = collector
}

// NewSystemInfoHandler creates a new SystemInfoHandler.
func NewSystemInfoHandler(pgPool *pgxpool.Pool, redisClient redis.UniversalClient, logger *zap.Logger) *SystemInfoHandler {
	return &SystemInfoHandler{
		pgPool:      pgPool,
		redisClient: redisClient,
		logger:      logger.Named("sysinfo"),
		diskPaths: appVisibleFilesystemPaths(configuredAppLogPath(), func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.IsDir()
		}),
		statfs: systemStatfs,
		now:    time.Now,
	}
}

// GetSystemInfo handles GET /api/v1/system/info.
func (h *SystemInfoHandler) GetSystemInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "normal"
	if h.pgPool == nil || h.pgPool.Ping(ctx) != nil {
		dbStatus = "error"
	}

	cacheStatus := "normal"
	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			cacheStatus = "error"
		}
	}

	info := SystemInfo{
		Version:     version,
		BuildDate:   buildDate,
		ServerTime:  time.Now().Format(time.RFC3339),
		UptimeHours: time.Since(startTime).Hours(),
		DBStatus:    dbStatus,
		CacheStatus: cacheStatus,
		Storage:     h.collectStorageMetrics(ctx),
	}

	response.OK(c, info)
}

func (h *SystemInfoHandler) collectStorageMetrics(ctx context.Context) []StorageMetric {
	statfs := h.statfs
	if statfs == nil {
		statfs = systemStatfs
	}
	now := time.Now
	if h.now != nil {
		now = h.now
	}

	metrics := make([]StorageMetric, 0, len(h.diskPaths)+3)
	for _, mountPath := range h.diskPaths {
		metrics = append(metrics, collectAppFilesystemMetric(mountPath, now(), statfs))
	}
	if h.storage != nil {
		metrics = append(metrics, h.storage.Collect(ctx)...)
	}
	return metrics
}

func collectAppFilesystemMetric(mountPath string, collectedAt time.Time, statfs filesystemStatfs) StorageMetric {
	metric := StorageMetric{
		ID: filesystemID(mountPath), Kind: "app_filesystem", Label: mountPath,
		Source: "statfs/app", MountPath: mountPath, Mountpoint: mountPath,
		TargetType: "application", TargetID: filesystemID(mountPath), Status: "unavailable",
	}

	stats, err := statfs(mountPath)
	if err != nil {
		metric.Error = err.Error()
		return metric
	}
	if stats.blockSize == 0 {
		metric.Error = "filesystem reported a zero block size"
		return metric
	}

	total := stats.blocks * stats.blockSize
	available := stats.availableBlocks * stats.blockSize
	if total == 0 {
		metric.Error = "filesystem reported zero total capacity"
		return metric
	}
	if available > total {
		metric.Error = "filesystem reported available capacity greater than total capacity"
		return metric
	}
	// Bavail is what the unprivileged application can still write. Treat
	// filesystem-reserved blocks as occupied so this matches node_exporter's
	// operational capacity formula: (size - avail) / size.
	used := total - available
	percent := math.Round(float64(used)*10000/float64(total)) / 100
	metric.TotalBytes, metric.UsedBytes, metric.AvailableBytes = &total, &used, &available
	if stats.files > 0 && stats.availableFiles <= stats.files {
		usedFiles := stats.files - stats.availableFiles
		inodePercent := math.Round(float64(usedFiles)*10000/float64(stats.files)) / 100
		metric.TotalInodes = &stats.files
		metric.UsedInodes = &usedFiles
		metric.AvailableInodes = &stats.availableFiles
		metric.UsedInodePercent = &inodePercent
	}
	metric.UsedPercent, metric.CollectedAt, metric.Status = &percent, &collectedAt, "available"
	return metric
}

func appVisibleFilesystemPaths(logPath string, pathExists func(path string) bool) []string {
	paths := []string{"/"}
	if logPath == "" {
		logPath = defaultAppLogPath
	}
	logPath = filepath.Clean(logPath)
	if logPath != "/" && pathExists(logPath) {
		paths = append(paths, logPath)
	}
	return paths
}

func configuredAppLogPath() string {
	return os.Getenv("OMC_LOG_DIR")
}

func filesystemID(mountPath string) string {
	if mountPath == "/" {
		return "root"
	}
	return strings.ReplaceAll(strings.Trim(strings.TrimSpace(mountPath), "/"), "/", "-")
}

func systemStatfs(path string) (filesystemStats, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return filesystemStats{}, fmt.Errorf("statfs %q: %w", path, err)
	}
	if stat.Bsize <= 0 {
		return filesystemStats{}, fmt.Errorf("statfs %q: invalid block size %d", path, stat.Bsize)
	}
	return filesystemStats{
		blockSize:       uint64(stat.Bsize),
		blocks:          stat.Blocks,
		availableBlocks: stat.Bavail,
		files:           stat.Files,
		availableFiles:  stat.Ffree,
	}, nil
}
