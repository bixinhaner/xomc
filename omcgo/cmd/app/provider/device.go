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

	// HeartbeatMonitor
	heartbeatMonitor := device.NewHeartbeatMonitor(c.Redis, deviceRepo, logger)
	heartbeatMonitor.Start()
	c.GS.Register("heartbeat", 1, func(ctx context.Context) error { heartbeatMonitor.Stop(); return nil })

	// Shared infrastructure
	connReqClient := connreq.NewClient(c.Redis, logger)
	stunStore := stun.NewStore(c.Redis, logger)

	// DeviceService
	deviceCache := device.NewDeviceCache(c.Redis, logger)
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, heartbeatMonitor, c.EventBus, logger)
	deviceService.SetDeviceCache(deviceCache)
	deviceService.SetDeviceInfoRepo(deviceInfoRepo)
	deviceService.SetTaskService(c.TaskSvc)
	deviceService.SetCarrierRegistry(c.Carriers) // T-0029: RF control path lookup
	// Phase 6 (设计文档 §4.3): productClass → product 装配件回填 device.model_name。
	// ProductRegistry 在 ModuleGraph 中先于 device 模块初始化（productregistry → device）。
	if c.ProductRegistry != nil {
		deviceService.SetProductMatcher(c.ProductRegistry)
	}
	deviceService.SetConnectionRequester(connReqClient)
	deviceService.SetStunAddressUpdater(stunStore)
	deviceMetrics := device.NewDeviceMetrics(c.MetricsReg)
	deviceService.SetMetrics(deviceMetrics)

	// InfoSyncer
	infoSyncer := device.NewInfoSyncer(deviceInfoRepo, paramRepo, c.Carriers, logger)
	deviceService.SetInfoSyncer(infoSyncer)

	// OfflineDetector
	offlineDetector := device.NewOfflineDetector(deviceRepo, infoSyncer, c.EventBus, logger)
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
		// Phase 6 follow-up: ProductRegistry 注入到 batch path,让 batch flush 也能回填 model_name
		// 与非 batch 路径 device_service.applyProductMetadata 对齐。c.ProductRegistry 已由
		// ModuleGraph 保证早于 device 模块初始化（device Depends "productregistry"）。
		if c.ProductRegistry != nil {
			batchProcessor.SetProductMatcher(c.ProductRegistry)
		}
		// Phase 3 follow-up: InfoSyncer 注入到 batch path,让 batch flush 后异步把
		// device_parameters 投影到 device_info 新列（tac/band/ul_earfcn/mac/transmit_power 等）。
		batchProcessor.SetInfoSyncer(infoSyncer)
		informHandler.SetBatchProcessor(batchProcessor)
	}
	if err := informHandler.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
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
