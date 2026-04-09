package provider

import (
	"context"

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
	deviceService.SetCommandQueue(c.CmdQueue)
	deviceService.SetConnectionRequester(connReqClient)
	deviceService.SetStunAddressUpdater(stunStore)
	deviceService.SetMetrics(device.NewDeviceMetrics(c.MetricsReg))

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
			stunStore, device.NewDeviceMetrics(c.MetricsReg), logger,
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
		informHandler.SetBatchProcessor(batchProcessor)
	}
	if err := informHandler.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe inform handler", zap.Error(err))
	}

	// Set shared services
	c.DeviceService = deviceService
	c.DeviceRepo = deviceRepo
	c.ParamRepo = paramRepo
	c.DeviceInfoRepo = deviceInfoRepo
	c.ConnReqClient = connReqClient
	c.StunStore = stunStore

	// Store deps for route registration
	c.deviceHandlerDeps = &deviceHandlerDeps{
		regRepo:       regRepo,
		regService:    regService,
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
