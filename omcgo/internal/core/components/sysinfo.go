package components

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	startTime = time.Now()
	version   = "dev"
	buildDate = "unknown"
)

// SystemInfo represents the runtime system information returned by the API.
type SystemInfo struct {
	Version     string  `json:"version"`
	BuildDate   string  `json:"build_date"`
	ServerTime  string  `json:"server_time"`
	UptimeHours float64 `json:"uptime_hours"`
	DBStatus    string  `json:"db_status"`
	CacheStatus string  `json:"cache_status"`
}

// SystemInfoHandler provides the GET /system/info endpoint.
type SystemInfoHandler struct {
	pgPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	logger      *zap.Logger
}

// NewSystemInfoHandler creates a new SystemInfoHandler.
func NewSystemInfoHandler(pgPool *pgxpool.Pool, redisClient redis.UniversalClient, logger *zap.Logger) *SystemInfoHandler {
	return &SystemInfoHandler{
		pgPool:      pgPool,
		redisClient: redisClient,
		logger:      logger.Named("sysinfo"),
	}
}

// GetSystemInfo handles GET /api/v1/system/info.
func (h *SystemInfoHandler) GetSystemInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "normal"
	if err := h.pgPool.Ping(ctx); err != nil {
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
	}

	c.JSON(http.StatusOK, info)
}
