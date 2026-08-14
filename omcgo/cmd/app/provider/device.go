package provider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
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
	antennaPlanRepo := device.NewPgAntennaSectorPlanRepository(c.PgPool)
	deviceInfoRepo := device.NewPgDeviceInfoRepository(c.PgPool)
	deviceCache := device.NewDeviceCache(c.Redis, logger)

	// T-0173: DeviceStatusReconciler 替代原来的 HeartbeatMonitor + OfflineDetector。
	// 同时承担:
	//   1) RefreshHeartbeat（ACS Inform 路径同步调用,刷 Redis 心跳 key）
	//   2) 后台扫描:每 60s 找 last_inform_at < NOW - max(2×inform_interval, 600s) 的
	//      在线设备,事务性翻 is_online=false + 累加 cumulative_online_duration +
	//      publish device.offline。
	reconciler := device.NewDeviceStatusReconciler(c.Redis, deviceRepo, c.EventBus, logger)
	reconciler.SetOfflineAlarmSink(c.AlarmEngine)
	reconciler.SetDeviceCache(deviceCache)
	// issue #203：离线阈值接 sys_configs (category='device') 实时配置。
	//   - key=enbTimeout → 基站类阈值（秒，默认 100）
	//   - key=cpeTimeout → CPE 类阈值（秒，默认 600）
	// 每轮扫描读最新值；改配置后下一轮扫描即生效（扫描周期 = min(阈值/2, 60s)），无需重启。
	offlineSysCfgRepo := admin.NewPgSysConfigRepository(c.PgPool)
	reconciler.SetThresholdLookup(func(ctx context.Context, category, key string) (string, bool) {
		cfg, err := offlineSysCfgRepo.GetByKey(ctx, category, key)
		if err != nil || cfg == nil {
			return "", false
		}
		return cfg.Value, true
	})
	reconciler.Start()
	c.GS.Register("device-status-reconciler", 1, func(ctx context.Context) error { reconciler.Stop(); return nil })

	// issue #397（加法优先）：在线索引 acs:online ZSET —— ACS 收 Inform 当场同步 ZADD 写入
	// （免 NATS 的存活信号）；这里在 app 单点周期 Prune（控规模）+ 记录在线总数。
	// 本期不改离线判定（reconciler 仍读 last_inform_at），仅铺设索引与计数能力。
	// Redis 为 nil（dev/test）时不构造、不启动。
	var onlineIdx *redisx.OnlineIndex
	if c.Redis != nil {
		onlineIdx = redisx.NewOnlineIndex(c.Redis)
	}
	onlinePruner := device.NewOnlinePruner(onlineIdx, logger)
	onlinePruner.Start()
	c.GS.Register("online-index-pruner", 1, func(ctx context.Context) error { onlinePruner.Stop(); return nil })

	// Shared infrastructure
	connReqClient := connreq.NewClient(c.Redis, logger)
	stunStore := stun.NewStore(c.Redis, logger)
	udpSender := connreq.NewUDPSender(stunStore, c.Cfg.ConnReq.SharedSecret, c.Logger)
	crDispatcher := connreq.NewDispatcher(connReqClient, udpSender, c.Logger)
	crDispatcher.SetLANPortLookup(newSTUNServerPortLookup(deviceRepo, paramRepo, c.Logger))

	// DeviceService
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, reconciler, c.EventBus, logger)
	deviceService.SetAntennaSectorPlanRepository(antennaPlanRepo)
	deviceService.SetRedis(c.Redis)
	deviceService.SetDisconnectedAlarmCleaner(c.AlarmPgStore, c.AlarmEngine)
	deviceService.SetDeviceCache(deviceCache)
	deviceService.SetDeviceInfoRepo(deviceInfoRepo)
	deviceService.SetControlSummaryReader(device.NewPgDeviceControlSummaryReader(c.PgPool))
	deviceService.SetDeviceGroupCountsInvalidator(c.GroupRepo)
	// IDOR 防护：按 ID 直读端点据此判定调用者对设备所属设备组的归属。
	deviceService.SetDeviceGroupReader(device.NewPgDeviceGroupReader(c.PgPool))
	deviceService.SetTaskService(c.TaskSvc)
	deviceService.SetCarrierRegistry(c.Carriers) // T-0029: RF control path lookup
	// Phase 6 (设计文档 §4.3): productClass → product 装配件回填 device.model_name。
	// ProductRegistry 在 ModuleGraph 中先于 device 模块初始化（productregistry → device）。
	if c.ProductRegistry != nil {
		deviceService.SetProductMatcher(c.ProductRegistry)
	}
	// T-0176-PR-D：CreateDevice INSERT 后写回 devices.product_id / param_model_id。
	// ProductRepo 与 ProductRegistry 同在 productregistry 模块创建，依赖关系一致。
	if c.ProductRepo != nil {
		deviceService.SetProductBinder(c.ProductRepo)
	}
	deviceService.SetConnectionRequester(&taskCRSender{dispatcher: crDispatcher, serverAddr: c.Cfg.ConnReq.ServerAddr})
	deviceService.SetStunAddressUpdater(stunStore)
	// license 降容清理：DeviceService.OfflineExcessDevices 委托 deviceRepo 批量置离线。
	deviceService.SetExcessOffliner(deviceRepo)
	deviceMetrics := device.NewDeviceMetrics(c.MetricsReg)
	deviceService.SetMetrics(deviceMetrics)

	// 注入系统配置查询（nameSyncMode 读取，供 RenameDevice 按策略决定行为）
	nameSyncCfgRepo := admin.NewPgSysConfigRepository(c.PgPool)
	deviceService.SetSysConfigLookup(func(ctx context.Context, cat, key string) (string, bool) {
		cfg, err := nameSyncCfgRepo.GetByKey(ctx, cat, key)
		if err != nil || cfg == nil {
			return "", false
		}
		return cfg.Value, true
	})

	// InfoSyncer
	infoSyncer := device.NewInfoSyncer(deviceInfoRepo, paramRepo, deviceRepo, c.Carriers, logger, device.NewPgLocationObservationRepository(c.PgPool))
	deviceService.SetInfoSyncer(infoSyncer)

	// T-0173: OfflineDetector 已合并入 DeviceStatusReconciler（见上方 reconciler 初始化）。

	// Registration module
	regRepo := device.NewPgRegistrationRepository(c.PgPool)
	regService := device.NewRegistrationService(regRepo, logger)
	deviceService.SetRegistrationRepo(regRepo)

	// Group assigner (from topology module)
	// issue #478：走 GroupService 而非 GroupRepo，使"目标 = 未分组内置节点 → 移出分组"
	// 的服务层兜底同样覆盖批量导入 / 预注册分组路径，杜绝写出指向内置节点的归属记录。
	deviceService.SetGroupAssigner(c.GroupService)

	// BatchInformProcessor（可选）
	var batchProcessor *device.BatchInformProcessor
	if c.Cfg.BatchProcessor.Enabled {
		batchProcessor = device.NewBatchInformProcessor(
			c.Cfg.BatchProcessor,
			c.PgPool, c.Redis, reconciler, deviceCache,
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
	// #17: 默认运营商不再硬编码 CMCC，而是经 CarrierRegistry 注入——OUI 解析失败时
	// 用注册表给出的默认运营商兜底（registry 优先 CMCC，无 CMCC 时取确定性首位）。
	// registry 为空（理论上不会发生，启动期已注册 cmcc/ctcc/cucc）时退回 CarrierCMCC。
	defaultCarrier := model.CarrierCMCC
	if c.Carriers != nil {
		if dc := c.Carriers.DefaultCarrier(); dc != "" {
			defaultCarrier = dc
		}
	}
	informHandler := device.NewInformHandler(deviceService, c.Carriers, defaultCarrier, logger)
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
	c.DeviceCache = deviceCache // T-0176-PR-D：供 provision / product 模块写库后失效 SN 缓存

	// T-0176-PR-D：把 DeviceCache 反向注入到 ProductHandler，让 BindOrphan / RematchOrphan
	// 写完 product_id 后立即失效 SN cache（ProductHandler 在 productregistry 模块创建，
	// 早于 device 模块；此处通过 setter 完成迟绑定，避免循环依赖）。
	if c.ProductHandler != nil {
		c.ProductHandler.SetDeviceCacheInvalidator(deviceCache)
	}

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
