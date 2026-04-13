package provider

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/topology"
)

// initTopologyModule 初始化 F06 拓扑管理模块。
// 设置: GroupRepo, GroupService
func initTopologyModule(c *Container) error {
	logger := c.Logger.Named("topology")

	groupRepo := topology.NewPgDeviceGroupRepository(c.PgPool)
	groupService := topology.NewDeviceGroupService(groupRepo, c.PgPool, logger)
	siteRepo := topology.NewPgSiteRepository(c.PgPool)
	topoNodeRepo := topology.NewPgTopoNodeRepository(c.PgPool)
	topoEdgeRepo := topology.NewPgTopoEdgeRepository(c.PgPool)

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
	}

	logger.Info("topology module initialized")
	return nil
}

type topologyHandlerDeps struct {
	groupRepo    *topology.PgDeviceGroupRepository
	groupService *topology.DeviceGroupService
	siteRepo     *topology.PgSiteRepository
	topoNodeRepo *topology.PgTopoNodeRepository
	topoEdgeRepo *topology.PgTopoEdgeRepository
}
