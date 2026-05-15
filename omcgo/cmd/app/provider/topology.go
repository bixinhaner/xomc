package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/topology"
	"go.uber.org/zap"
)

// initTopologyModule 初始化 F06 拓扑管理模块。
// 设置: GroupRepo, GroupService, DeviceSyncService
func initTopologyModule(c *Container) error {
	logger := c.Logger.Named("topology")

	groupRepo := topology.NewPgDeviceGroupRepository(c.PgPool)
	siteRepo := topology.NewPgSiteRepository(c.PgPool)
	topoNodeRepo := topology.NewPgTopoNodeRepository(c.PgPool)
	topoEdgeRepo := topology.NewPgTopoEdgeRepository(c.PgPool)
	groupService := topology.NewDeviceGroupService(groupRepo, topoNodeRepo, c.PgPool, logger)

	// Create device sync service with EventBus
	syncSvc := topology.NewDeviceSyncService(c.PgPool, topoNodeRepo, topoEdgeRepo, c.EventBus, logger)

	// Set shared services
	c.GroupRepo = groupRepo
	c.GroupService = groupService

	// Register module-level health check
	c.Health.Register("topology", func(ctx context.Context) error {
		if err := c.PgPool.Ping(ctx); err != nil {
			return fmt.Errorf("topology module db ping: %w", err)
		}
		return nil
	})

	// Store repos for handler/route creation
	c.topologyHandlerDeps = &topologyHandlerDeps{
		groupRepo:    groupRepo,
		groupService: groupService,
		siteRepo:     siteRepo,
		topoNodeRepo: topoNodeRepo,
		topoEdgeRepo: topoEdgeRepo,
		syncSvc:      syncSvc,
		logger:       logger,
	}

	// Configure and start device sync service
	syncConfig := topology.DeviceSyncServiceConfig{
		Enabled:          c.Cfg.Topology.DeviceSync.Enabled,
		InitialSync:      c.Cfg.Topology.DeviceSync.InitialSync,
		FallbackInterval: c.Cfg.Topology.DeviceSync.FallbackInterval,
		InitialSyncDelay: c.Cfg.Topology.DeviceSync.InitialSyncDelay,
		BatchSize:        c.Cfg.Topology.DeviceSync.BatchSize,
	}
	// Apply defaults if not set
	if syncConfig.InitialSyncDelay == 0 {
		syncConfig.InitialSyncDelay = 10 * time.Second
	}
	if syncConfig.BatchSize == 0 {
		syncConfig.BatchSize = 100
	}
	if syncConfig.FallbackInterval == 0 {
		syncConfig.FallbackInterval = 1 * time.Hour // Default fallback interval
	}

	syncSvc.SetConfig(syncConfig)

	if err := syncSvc.Start(context.Background()); err != nil {
		return fmt.Errorf("start device sync service: %w", err)
	}

	// Register shutdown hook for sync service
	c.GS.Register("topology-device-sync", 50, func(ctx context.Context) error {
		syncSvc.Stop()
		return nil
	})

	logger.Info("topology module initialized",
		zap.Bool("sync_enabled", syncConfig.Enabled),
		zap.Bool("initial_sync", syncConfig.InitialSync),
		zap.Duration("fallback_interval", syncConfig.FallbackInterval))

	return nil
}

type topologyHandlerDeps struct {
	groupRepo    *topology.PgDeviceGroupRepository
	groupService *topology.DeviceGroupService
	siteRepo     *topology.PgSiteRepository
	topoNodeRepo *topology.PgTopoNodeRepository
	topoEdgeRepo *topology.PgTopoEdgeRepository
	syncSvc      *topology.DeviceSyncService
	logger       *zap.Logger
}
