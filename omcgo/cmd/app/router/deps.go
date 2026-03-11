package router

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/event"
	"github.com/omcgo/omcgo/internal/appconfig"
	"github.com/omcgo/omcgo/internal/components"
)

// Deps aggregates all infrastructure dependencies initialized by main.go.
type Deps struct {
	PgPool          *pgxpool.Pool
	TsPool          *pgxpool.Pool
	Redis           redis.UniversalClient
	MinIO           *minio.Client
	EventBus        event.EventBus
	CmdQueue        *cmdqueue.RedisCommandQueue
	CarrierRegistry *carrier.CarrierRegistry
	Cfg             *appconfig.AppConfig
	Logger          *zap.Logger
	GS              *components.GracefulShutdown
	MetricsReg      *prometheus.Registry
}
