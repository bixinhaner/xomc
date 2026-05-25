package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

// initDeviceModule 初始化 F06 设备管理模块。
// 设置: DeviceService, DeviceRepo, ParamRepo, DeviceInfoRepo, ConnReqClient, StunStore
func initDeviceModule(c *Container) error {
	logger := c.Logger.Named("device")

	// Repositories
	deviceRepo := device.NewPgDeviceRepository(c.PgPool)
	paramRepo := device.NewPgDeviceParameterRepository(c.PgPool)
	deviceInfoRepo := device.NewPgDeviceInfoRepository(c.PgPool)

	// Per-device Sequencer：与 DeviceService / RebootResponseOfflineMarker /
	// OfflineDetector / HeartbeatMonitor 共享同一实例，保证同一 device 的 is_online /
	// lifecycle_state 写操作按 EventBus publish 顺序 FIFO 串行处理，从根因消除
	// PERIODIC 异步覆盖 RebootResponse 写 false 的竞态（真机验证 2026-05-25）。
	// 单进程方案；多 worker 实例时需换 Redis 分布式锁 / NATS partition（v2）。
	deviceSeq := device.NewSequencer()

	// DeviceCache (proactively constructed to share across HeartbeatMonitor / OfflineDetector /
	// marker / DeviceService — they all need to invalidate after writing is_online=false).
	deviceCache := device.NewDeviceCache(c.Redis, logger)

	// HeartbeatMonitor
	heartbeatMonitor := device.NewHeartbeatMonitor(c.Redis, deviceRepo, logger)
	heartbeatMonitor.SetSequencer(deviceSeq)
	heartbeatMonitor.SetCache(deviceCache)
	heartbeatMonitor.Start()
	c.GS.Register("heartbeat", 1, func(ctx context.Context) error { heartbeatMonitor.Stop(); return nil })

	// Shared infrastructure
	connReqClient := connreq.NewClient(c.Redis, logger)
	stunStore := stun.NewStore(c.Redis, logger)

	// DeviceService
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, heartbeatMonitor, c.EventBus, logger)
	deviceService.SetDeviceCache(deviceCache)
	deviceService.SetDeviceInfoRepo(deviceInfoRepo)
	deviceService.SetTaskService(c.TaskSvc)
	deviceService.SetCarrierRegistry(c.Carriers) // T-0029: RF control path lookup
	deviceService.SetConnectionRequester(connReqClient)
	deviceService.SetStunAddressUpdater(stunStore)
	deviceService.SetSequencer(deviceSeq)
	deviceMetrics := device.NewDeviceMetrics(c.MetricsReg)
	deviceService.SetMetrics(deviceMetrics)

	// InfoSyncer
	infoSyncer := device.NewInfoSyncer(deviceInfoRepo, paramRepo, c.Carriers, logger)
	deviceService.SetInfoSyncer(infoSyncer)

	// OfflineDetector
	offlineDetector := device.NewOfflineDetector(deviceRepo, infoSyncer, c.EventBus, logger)
	offlineDetector.SetSequencer(deviceSeq)
	offlineDetector.SetCache(deviceCache)
	go func() {
		if err := offlineDetector.Start(context.Background()); err != nil {
			logger.Error("offline detector stopped with error", zap.Error(err))
		}
	}()
	c.GS.Register("offline-detector", 2, func(ctx context.Context) error { return nil })

	// Registration module
	regRepo := device.NewPgRegistrationRepository(c.PgPool)
	regService := device.NewRegistrationService(regRepo, logger)
	deviceService.SetRegistrationRepo(regRepo)

	// Group assigner (from topology module)
	deviceService.SetGroupAssigner(c.GroupRepo)

	// BatchInformProcessor（可选）
	var batchProcessor *device.BatchInformProcessor
	if c.Cfg.BatchProcessor.Enabled {
		batchProcessor = device.NewBatchInformProcessor(
			c.Cfg.BatchProcessor,
			c.PgPool, c.Redis, heartbeatMonitor, deviceCache,
			stunStore, deviceMetrics, logger,
		)
		batchProcessor.SetSequencer(deviceSeq)
		batchProcessor.Start()
		c.GS.Register("batch-processor", 2, func(ctx context.Context) error {
			batchProcessor.Stop()
			return nil
		})
		logger.Info("batch inform processor enabled",
			zap.Int("workers", c.Cfg.BatchProcessor.Workers))
	}

	// InformHandler (subscribes to events)
	informHandler := device.NewInformHandler(deviceService, c.Carriers, model.CarrierCMCC, logger)
	if batchProcessor != nil {
		// T-0123 / T-0125 batch path：注入 transition publisher，让 batch flush 完成后能
		// 触发 device.online / device.firmware.changed 事件（与 UpdateFromInform 非 batch
		// 路径对齐）。08754f89 收官 PR 漏挂导致这两个触发源在 batch 模式下全失效。
		batchProcessor.SetTransitionPublisher(deviceService)
		informHandler.SetBatchProcessor(batchProcessor)
	}
	if err := informHandler.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
	}

	// RebootResponseOfflineMarker (F01/F06)：ACS 收到 RebootResponse 时立即把设备置离线。
	// 让 BOOT Inform 回连时 device_service 检测到 offline→active 转换，发 device.online
	// → pm.OnlineSubscriber 入队 PM SPV 自动重发 PM 上传配置（真机验证 2026-05-25 暴露的
	// G1 缺口）。注册在 app 进程，与 DeviceService.UpdateFromInform 共享 deviceSeq，
	// in-memory 锁保证不与 PERIODIC 异步覆盖。
	rebootOfflineMarker := device.NewRebootResponseOfflineMarker(deviceRepo, deviceSeq, logger)
	rebootOfflineMarker.SetCache(deviceCache)
	if err := rebootOfflineMarker.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe reboot response offline marker", zap.Error(err))
	}

	// D 方案：PeriodicOnlineMarker — PERIODIC / VALUE_CHANGE Inform 即时把 is_online 写 true。
	// 与 batch 路径解耦（batch UPDATE 不再写 is_online 列），消除 batch buffer 累积期间
	// marker 写的 is_online=false 被 batch flush 覆盖的 lost update 问题。
	periodicOnlineMarker := device.NewPeriodicOnlineMarker(deviceRepo, deviceSeq, deviceCache, logger)
	if err := periodicOnlineMarker.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe periodic online marker", zap.Error(err))
	}

	// Set shared services
	c.DeviceService = deviceService
	c.InformHandler = informHandler // 后续 initMiscModules 注入 GroupAssigner（matcher 适配器）
	c.DeviceRepo = deviceRepo
	c.ParamRepo = paramRepo
	c.DeviceInfoRepo = deviceInfoRepo
	c.ConnReqClient = connReqClient
	c.StunStore = stunStore

	// Register module-level health check
	c.Health.Register("device", func(ctx context.Context) error {
		if err := c.PgPool.Ping(ctx); err != nil {
			return fmt.Errorf("device module db ping: %w", err)
		}
		if err := c.Redis.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("device module redis ping: %w", err)
		}
		return nil
	})

	// Store deps for route registration
	c.deviceHandlerDeps = &deviceHandlerDeps{
		regRepo:        regRepo,
		regService:     regService,
		batchProcessor: batchProcessor,
	}

	logger.Info("device module initialized")
	return nil
}

type deviceHandlerDeps struct {
	regRepo        *device.PgRegistrationRepository
	regService     *device.RegistrationService
	batchProcessor *device.BatchInformProcessor
}
