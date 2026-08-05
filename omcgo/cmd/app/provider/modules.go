package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/buildinfo"
	"github.com/omcgo/omcgo/internal/bundle"
	"github.com/omcgo/omcgo/internal/config"
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/components"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/ratelimit"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/dlq"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/dashboard"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/eventlog"
	"github.com/omcgo/omcgo/internal/events"
	"github.com/omcgo/omcgo/internal/filemanager"
	"github.com/omcgo/omcgo/internal/interop"
	"github.com/omcgo/omcgo/internal/interop/cases"
	"github.com/omcgo/omcgo/internal/license"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/mr"
	mrtask "github.com/omcgo/omcgo/internal/mr/task"
	"github.com/omcgo/omcgo/internal/nedirect"
	"github.com/omcgo/omcgo/internal/northbound"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/omcgo/omcgo/internal/ops"
	"github.com/omcgo/omcgo/internal/paramsync"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/rebootrecord"
	"github.com/omcgo/omcgo/internal/report"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/omcgo/omcgo/internal/stationlog"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/internal/syslog"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
	"github.com/omcgo/omcgo/internal/trace"
	transferrepo "github.com/omcgo/omcgo/internal/transfer/repo"
	"github.com/omcgo/omcgo/internal/ufte"
)

// initMRModule 初始化 F05 测量报告模块。
func initMRModule(c *Container) error {
	logger := c.Logger.Named("mr")

	mrStore := mr.NewPgMRStore(c.TsPool) // mr_files 已迁时序库，单 TsPool
	mrIndRepo := mr.NewPgIndicatorRepository(c.PgPool)
	mrMapRepo := mr.NewPgMappingRepository(c.PgPool)

	c.miscDeps.mrStore = mrStore
	c.miscDeps.mrIndRepo = mrIndRepo
	c.miscDeps.mrMapRepo = mrMapRepo

	logger.Info("MR module initialized")
	return nil
}

// initMRTaskModule 装配 F05 MR 测量任务的调度链路：
//   - Dispatcher：standardPath → ToPrivate → SPV 派发
//   - Scheduler：@every 30s 触发开/关/心跳巡检（Redis SETNX 锁防并发）
//   - HeartbeatSubscriber：订阅 mr.file.uploaded 写 Redis + PG 心跳
//   - Cleaner：@daily 删超期 MR 文件
//
// 与 misc 模块的 CompletionRouter 注册解耦：mrtask CompletionCallback 在
// initMiscModules 内部跟 MML aggregator 一起注册（共用 task.TaskSourceSystem）。
//
// 依赖项（ModuleGraph 强制 Depends）：
//   - "paramregistry"：dispatcher 通过 TranslatorResolver 调 c.ParamRegistry
//   - "productregistry"：同上，参与 productClass → product.id 匹配
//   - "device"：c.DeviceRepo 已初始化
//   - "task"：c.TaskSvc 已初始化（dispatcher 的 Enqueuer）
//   - "misc"：CompletionRouter 已就绪（虽然回调注册在 misc 内部，本模块装配
//     时只需要 TaskService / Repo，不直接读 CompletionRouter）
func initMRTaskModule(c *Container) error {
	logger := c.Logger.Named("mr-task")

	mrCfg := c.Cfg.MR.Defaults()
	// mr.url_base 为空时回退到 sys_configs.acs_transfer.uploadBaseURL（运行时配置，
	// 跟 FAULT_LOG_COLLECT / CONFIG_RESTORE 等其它 ACS 上传链路共用同一个真值源）。
	// YAML 里的 acs_upload_base_url 是 deploy-time 兜底，sys_configs 表才是
	// "系统管理 → ACS 传输"页面修改后的真实运行时值。
	// 原 dispatcher 在 base="" 时直接下发相对 path `/smallcell/...`，设备拿到
	// 拒收 / 拼错 host → MR 文件 0 上传，progress 推到 openSuccess 但 health 永远 abnormal。
	if mrCfg.URLBase == "" {
		sysCfg := admin.NewPgSysConfigRepository(c.PgPool)
		mrPolicy := transfercfg.NewPolicy(transfercfg.Snapshot{},
			func(ctx context.Context, category, key string) (string, bool) {
				row, err := sysCfg.GetByKey(ctx, category, key)
				if err != nil || row == nil {
					return "", false
				}
				return row.Value, true
			})
		if snap := mrPolicy.Snapshot(context.Background()); snap.Upload.BaseURL != "" {
			mrCfg.URLBase = snap.Upload.BaseURL
			logger.Info("mr.url_base empty; resolved via sys_configs.acs_transfer.uploadBaseURL",
				zap.String("base", mrCfg.URLBase))
		} else if c.Cfg.Upgrade.ACSUploadBaseURL != "" {
			mrCfg.URLBase = c.Cfg.Upgrade.ACSUploadBaseURL
			logger.Info("mr.url_base empty; sys_configs empty too, fell back to YAML acs_upload_base_url",
				zap.String("base", mrCfg.URLBase))
		} else {
			logger.Warn("mr.url_base empty AND no fallback found; MrUrl will be path-only and devices will reject")
		}
	}
	taskRepo := mrtask.NewPgRepository(c.PgPool)
	// Prometheus 指标（4 + 2 共 6 个）；c.MetricsReg 为 nil 时所有 IncXxx 退化为 no-op。
	metrics := mrtask.NewMetrics(c.MetricsReg)

	// Dispatcher：负责把 mrtask 操作转成 SPV，路径走 ParamRegistry 翻译。
	dispatcher := mrtask.NewDispatcher(
		mrCfg,
		c.TaskSvc,
		NewMRDeviceContext(c.DeviceRepo),
		NewMRTranslatorResolver(c.ProductRegistry, c.ParamRegistry, logger),
		taskRepo,
		logger,
	)
	dispatcher.SetMetrics(metrics)
	// 平台判断走"productClass → ProductRegistry → param_model.name → 允许列表"。
	// c.ParamModelRepo 由 paramregistry 模块初始化（PgRepository）。
	dispatcher.SetPlatformResolver(NewMRPlatformResolver(c.ProductRegistry, c.ParamModelRepo, logger))

	// Scheduler：3 个 cron entry（open / close / heartbeat），Redis SETNX 互斥
	scheduler := mrtask.NewScheduler(taskRepo, dispatcher, c.Redis,
		mrtask.SchedulerConfig{
			IntervalSeconds:        mrCfg.SchedulerIntervalSeconds,
			HeartbeatMissThreshold: mrCfg.HeartbeatMissThreshold,
		}, logger)
	scheduler.SetMetrics(metrics)
	// 简化模型：scheduler 在开启任务时从 mr_device_mappings 动态枚举 enabled cell
	if c.miscDeps.mrMapRepo != nil {
		scheduler.SetMappings(c.miscDeps.mrMapRepo)
	}
	if err := scheduler.Start(context.Background()); err != nil {
		// Start 失败一般是 cron 注册错误，属配置类错误 — 升级为 error
		// 避免静默漂移（与 worker 端原行为对齐）。
		logger.Error("mr scheduler start failed", zap.Error(err))
	} else {
		logger.Info("mr scheduler started", zap.Int("interval_sec", mrCfg.SchedulerIntervalSeconds))
	}

	// HeartbeatSubscriber：订阅 mr.file.uploaded → Redis SET + PG TouchHeartbeat
	heartbeat := mrtask.NewHeartbeatSubscriber(taskRepo, c.Redis, c.EventBus, c.Deduper, logger)
	heartbeat.SetMetrics(metrics)
	if err := heartbeat.Subscribe(); err != nil {
		logger.Warn("mr heartbeat subscribe failed", zap.Error(err))
	} else {
		logger.Info("mr heartbeat subscriber started")
	}

	// 暴露 repo 给 router.go 注册 REST handler（initMRTaskModule 也接管原 router.go
	// 里直接 NewPgRepository 的那行 — 避免双实例）。
	c.miscDeps.mrTaskRepo = taskRepo

	// 注册 SPV 完成回调到 misc 模块的 CompletionRouter。
	// 不能在 misc 模块 init 时注册（misc → mrtask 依赖图意味着 misc 先 init，
	// 那时 mrTaskRepo 还是 nil → if 分支不进 → 回调永远不挂 → SPV fault 事件
	// 拿不到，progress_status 卡 pending 让 reaper 兜底）。CompletionRouter.Register
	// 加了 RWMutex，pre-traffic 阶段后挂 handler 无 race（cmd/app/provider 设计原则）。
	if c.miscDeps.completionRouter != nil {
		c.miscDeps.completionRouter.Register(task.TaskSourceSystem,
			mrtask.NewCompletionCallback(taskRepo, logger))
		logger.Info("mr completion callback registered to TaskSourceSystem")
	} else {
		logger.Warn("completion router not wired; MR SPV fault → openFailure routing disabled")
	}
	return nil
}

func startMMLScheduler(
	c *Container,
	mmlService *mml.Service,
	taskRepo mml.ScheduledTaskRepository,
	logger *zap.Logger,
) *mml.Scheduler {
	scheduler := mml.NewScheduler(mmlService, taskRepo, nil, 0, logger)
	scheduler.Start(context.Background())

	if c != nil {
		c.miscDeps.mmlScheduler = scheduler
		if c.GS != nil {
			c.GS.Register("mml-scheduler", 1, func(context.Context) error {
				scheduler.Stop()
				scheduler.Wait()
				return nil
			})
		}
	}

	logger.Info("mml scheduler started")
	return scheduler
}

// initSoftwareModule 初始化 F06 固件管理模块。
func initSoftwareModule(c *Container) error {
	logger := c.Logger.Named("software")

	firmwareRepo := software.NewPgFirmwareRepository(c.PgPool)
	rawTaskRepo := software.NewPgTaskRepository(c.PgPool)
	rawSubTaskRepo := software.NewPgSubTaskRepository(c.PgPool)

	// 文件传输任务按业务拆表（T-0162 后续 / docs/design/task-tables-split-by-business-20260521.md）：
	// 4 类新业务（配置备份 / 配置下发 / 运行日志 / 异常日志）各占一对物理表；升级 / 回退
	// 继续用旧 upgrade_tasks / upgrade_sub_tasks。RoutingTaskRepository / RoutingSubTaskRepository
	// 实现 software.TaskRepository / SubTaskRepository 接口，对 software / executor 透明：
	//   · Create：从 ctx WithRouteHint 取 fileType 选目标表（BatchCollect 入口注入）。
	//   · 读 / Update / Delete：先 fallback 旧表，未命中再 fan-out 4 张新表（命中即返回）。
	// 数据迁移待 S5 阶段补；当前线上 16 行 task_type=10 行将继续在旧表（routing 兼容）。
	configBackupTaskRepo := transferrepo.NewPgTaskRepo(c.PgPool, transferrepo.TableConfigBackupTasks)
	configBackupSubRepo := transferrepo.NewPgSubTaskRepo(c.PgPool, transferrepo.TableConfigBackupSubTasks, transferrepo.TableConfigBackupTasks)
	configRestoreTaskRepo := transferrepo.NewPgTaskRepo(c.PgPool, transferrepo.TableConfigRestoreTasks)
	configRestoreSubRepo := transferrepo.NewPgSubTaskRepo(c.PgPool, transferrepo.TableConfigRestoreSubTasks, transferrepo.TableConfigRestoreTasks)
	runtimeLogTaskRepo := transferrepo.NewPgTaskRepo(c.PgPool, transferrepo.TableRuntimeLogCollectTasks)
	runtimeLogSubRepo := transferrepo.NewPgSubTaskRepo(c.PgPool, transferrepo.TableRuntimeLogCollectSubTasks, transferrepo.TableRuntimeLogCollectTasks)
	faultLogTaskRepo := transferrepo.NewPgTaskRepo(c.PgPool, transferrepo.TableFaultLogCollectTasks)
	faultLogSubRepo := transferrepo.NewPgSubTaskRepo(c.PgPool, transferrepo.TableFaultLogCollectSubTasks, transferrepo.TableFaultLogCollectTasks)
	deviceLockRepo := transferrepo.NewPgDeviceLockRepo(c.PgPool)
	_ = deviceLockRepo // S4 阶段接入 reaper / executor 时启用

	transferRouter := &software.TransferRepoRouter{
		Default: software.TransferRepoSet{
			Task: rawTaskRepo, SubTask: rawSubTaskRepo,
			SubTaskTable: transferrepo.TableUpgradeSubTasks,
			BusinessType: transferrepo.BusinessUpgrade,
		},
		ConfigBackup: software.TransferRepoSet{
			Task: configBackupTaskRepo, SubTask: configBackupSubRepo,
			SubTaskTable: transferrepo.TableConfigBackupSubTasks,
			BusinessType: transferrepo.BusinessConfigBackup,
		},
		ConfigRestore: software.TransferRepoSet{
			Task: configRestoreTaskRepo, SubTask: configRestoreSubRepo,
			SubTaskTable: transferrepo.TableConfigRestoreSubTasks,
			BusinessType: transferrepo.BusinessConfigRestore,
		},
		RuntimeLogCollect: software.TransferRepoSet{
			Task: runtimeLogTaskRepo, SubTask: runtimeLogSubRepo,
			SubTaskTable: transferrepo.TableRuntimeLogCollectSubTasks,
			BusinessType: transferrepo.BusinessRuntimeLogCollect,
		},
		FaultLogCollect: software.TransferRepoSet{
			Task: faultLogTaskRepo, SubTask: faultLogSubRepo,
			SubTaskTable: transferrepo.TableFaultLogCollectSubTasks,
			BusinessType: transferrepo.BusinessFaultLogCollect,
		},
	}
	taskRepo := software.NewRoutingTaskRepository(transferRouter, rawTaskRepo)
	subTaskRepo := software.NewRoutingSubTaskRepository(transferRouter, rawSubTaskRepo)

	softwareService := software.NewSoftwareService(
		firmwareRepo, taskRepo, subTaskRepo, c.DeviceRepo, c.TaskSvc, c.ConnReqClient,
		c.MinIO, c.Cfg.MinIO.Buckets.Firmware, c.EventBus, c.Redis, logger,
	)
	if err := softwareService.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe software service", zap.Error(err))
	}
	softwareService.RestorePendingUpgrades(context.Background())
	softwareService.StartTaskReaper()

	// 定时任务调度器：周期扫所有 5 张 task 表的"pending+timing"行，到期触发。
	// LogCollect 类任务的触发回调由 UFTE 模块在 initUFTEModule 中通过
	// taskScheduler.SetCollectTrigger 注入。
	taskScheduler := software.NewTaskScheduler(
		softwareService,
		[]software.ScheduledTaskSource{
			rawTaskRepo,
			configBackupTaskRepo,
			configRestoreTaskRepo,
			runtimeLogTaskRepo,
			faultLogTaskRepo,
		},
		0, // 默认 30s
		logger,
	)
	taskScheduler.Start(context.Background())
	c.miscDeps.taskScheduler = taskScheduler
	// 注入 ACS 上传基础 URL，供日志采集 Upload RPC 构造目标 URL（YAML 静态兜底）。
	if c.Cfg.Upgrade.ACSUploadBaseURL != "" {
		softwareService.SetUploadConfig(c.Cfg.Upgrade.ACSUploadBaseURL)
	}
	// 注入运行时 ACS 传输配置：从 sys_config 'acs_transfer' 类别读 BaseURL / Username /
	// Password / Path。前端"系统管理 → ACS 传输"页面修改后 30 秒内自动生效，
	// 优先级高于 YAML 静态兜底。worker 进程 / acs 进程也各自起一份 Policy（详见
	// cmd/worker/main.go / cmd/acs/main.go），共享同一张 sys_configs 表。
	softwareSysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	softwareTransferPolicy := transfercfg.NewPolicy(
		transfercfg.Snapshot{},
		func(ctx context.Context, category, key string) (string, bool) {
			row, err := softwareSysConfigRepo.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		},
	)
	softwareService.SetTransferProvider(softwareTransferPolicy)

	// Canary monitor + metrics (T-0018 / R-101)
	canaryMetrics := software.NewCanaryMetrics(c.MetricsReg)
	softwareService.SetCanaryMetrics(canaryMetrics)
	// Rollback metrics (T-0021 / R-101): registered before canary monitor wiring
	// so the auto-rollback path emits counters on first fire.
	rollbackMetrics := software.NewRollbackMetrics(c.MetricsReg)
	softwareService.SetRollbackMetrics(rollbackMetrics)

	// 固件下发前完整性 / 签名校验（issue #8）。完整性校验（SHA-256，存量回退 MD5）始终
	// 强制；签名校验为脚手架：仅对带签名固件验签，是否对未签名固件硬拒由
	// upgrade.require_firmware_signature 控制（默认 false，不破坏现网未签名固件 happy-path）。
	// verifyFn 暂传 nil（厂商 PKI / 公钥装载 / 密钥轮换 / 吊销是独立子系统，尚未落地）——
	// 此情形下带签名固件会被拒绝下发（不假装验过），未签名固件按上述开关处理。
	firmwareMetrics := software.NewFirmwareMetrics(c.MetricsReg)
	softwareService.SetFirmwareMetrics(firmwareMetrics)
	softwareService.SetFirmwareVerifier(
		software.NewSignatureVerifier(software.NewHashOnlyVerifier(), c.Cfg.Upgrade.RequireFirmwareSignature, nil),
	)
	canaryMonitor := software.NewCanaryMonitor(taskRepo, canaryMetrics, logger)
	// Wire SoftwareService as the auto-rollback trigger. Default RollbackOnFailure
	// is false; the trigger only fires when a canary task explicitly opted in.
	canaryMonitor.SetRollbackTrigger(softwareService)
	// #59 Problem 3：装配阈值越限止血回调——越限时真正 SuspendUpgrade（取消执行
	// context + 翻 sub_task 状态），让待派发 / 在飞设备停止收 Download，而非仅标 StageStatus。
	canaryMonitor.SetSuspender(softwareService)
	if err := canaryMonitor.Start(context.Background()); err != nil {
		logger.Warn("start canary monitor", zap.Error(err))
	}
	c.miscDeps.canaryMonitor = canaryMonitor

	// #59 Problem 4：注入设备组读取器，供升级 / 回退创建链路逐设备归属校验
	// （handler 经 SetPermissionService 解析 visibleGroups，service 用此 reader 判定）。
	softwareService.SetDeviceGroupReader(device.NewPgDeviceGroupReader(c.PgPool))

	softwareHandler := software.NewHandler(softwareService, logger)

	c.miscDeps.softwareHandler = softwareHandler
	c.miscDeps.softwareService = softwareService
	c.miscDeps.softwareTaskRepo = taskRepo
	c.miscDeps.softwareSubTaskRepo = subTaskRepo

	logger.Info("software management module initialized with canary monitor")
	return nil
}

func initUFTEModule(c *Container) error {
	logger := c.Logger.Named("ufte")
	taskTypeRepo := ufte.NewPgTaskTypeRepository(c.PgPool)
	service := ufte.NewService(
		c.miscDeps.softwareService,
		taskTypeRepo,
		c.miscDeps.softwareTaskRepo,
		c.miscDeps.softwareSubTaskRepo,
		c.DeviceRepo,
		logger,
	)
	// #63 租户隔离：注入设备组归属读取器，CreateTask / 各操作端点按设备可见性强制。
	service.SetGroupReader(device.NewPgDeviceGroupReader(c.PgPool))
	// 设备候选过滤的 productClass → tech 精确识别走 ProductRegistry，避免
	// FAP/BSC7041C243 这种"非 5G/GNB 关键字命名但实际是 5G 产品"被关键字模糊
	// 匹配漏选。Registry 缺失 → matchesTaskTypeScope 关键字兜底。
	if c.ProductRegistry != nil {
		service.SetProductTechLookup(func(ctx context.Context, productClass string) (string, bool) {
			res, err := c.ProductRegistry.MatchProductClass(ctx, productClass)
			if err != nil || res == nil || res.Product == nil {
				return "", false
			}
			return res.Product.Tech, true
		})
		// #492：模板 product_scope（适用产品名列表）非空时，设备候选按产品英文名精确匹配。
		service.SetProductNameLookup(func(ctx context.Context, productClass string) (string, bool) {
			res, err := c.ProductRegistry.MatchProductClass(ctx, productClass)
			if err != nil || res == nil || res.Product == nil {
				return "", false
			}
			return res.Product.Name, true
		})
	}
	inserted, err := service.EnsureBuiltInTaskTypes(context.Background())
	if err != nil {
		return err
	}

	// 装配 LogCollect 类（备份 / 日志采集）"已完成"子任务的下载链接回调。
	// backup.PgFileRepository 是无状态适配器（仅持有 PgPool），不依赖 initBackupModule
	// 是否运行；MinIO 也是进程级共享。任一空则 lookup 返回 ("", nil)，DownloadURL 留空，
	// 不阻断设备列表。详见 docs/project/backup-display-fix-20260520.md B5。
	backupFileRepo := backup.NewPgFileRepository(c.PgPool)
	// 用 PresignClient 而不是内部 MinIO client：内部 client 的 endpoint 是
	// `minio:9000`（docker 服务名 / k8s ClusterIP），签出来的 presigned URL 浏览器
	// 解析不了；NewPresignClient 在 minio.public_endpoint 非空时换成对外可达 host
	// 重签（如 dev 配置 localhost:9000），空时退化到内部 endpoint。
	//
	// issue #548 切片 4：默认 signer 优先用 c.PresignBridge.Get()（sys_configs 热改
	// storage.minio_public_endpoint 后下一次签 URL 立即用新 endpoint）；桥未装配
	// （dev/test 路径）回退旧 NewPresignClient。注意 line 451 的"按 ACS uploadBaseURL
	// 动态重签"是 UFTE 独立业务诉求，保留不动。
	minioClient := c.MinIO
	if c.PresignBridge != nil {
		if pc := c.PresignBridge.Get(); pc != nil {
			minioClient = pc
		}
	} else if presignClient, presignErr := minioinfra.NewPresignClient(c.Cfg.MinIO); presignErr != nil {
		logger.Warn("create MinIO presign client failed; download URLs will use internal endpoint",
			zap.Error(presignErr))
	} else {
		minioClient = presignClient
	}
	// 复用 software 模块同款 transferPolicy：sys_configs 的 acs_transfer.uploadBaseURL
	// 是运维在 OMC 后台维护的"对外可达 host"，复用它作为 MinIO 下载 URL 的 host 来源，
	// 比 yaml minio.public_endpoint 更动态（改配置不用重启）。host 提取后保持 :9000
	// 端口（MinIO 固定）。30 秒缓存内置在 transferPolicy 里，无额外查库开销。
	ufteSysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	ufteTransferPolicy := transfercfg.NewPolicy(
		transfercfg.Snapshot{},
		func(ctx context.Context, category, key string) (string, bool) {
			row, err := ufteSysConfigRepo.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		},
	)
	service.SetDownloadURLLookup(func(ctx context.Context, sn, fileName string) (string, error) {
		if minioClient == nil || sn == "" || fileName == "" {
			return "", nil
		}
		files, lookupErr := backupFileRepo.ListBySerial(ctx, sn)
		if lookupErr != nil {
			return "", lookupErr
		}
		for i := range files {
			if files[i].FileName != fileName || files[i].ObjectPath == "" {
				continue
			}
			if files[i].IsDeleted {
				continue
			}
			bucket, objectPath, splitErr := backup.SplitBucketAndPath(files[i].ObjectPath)
			if splitErr != nil {
				return "", splitErr
			}
			// 动态 host 重新签：presigned URL 的签名 V4 算法包含 host header，
			// 不能签后改 host（破坏签名 → SignatureDoesNotMatch）。从 sys_configs
			// 拿 uploadBaseURL.host，结合 MinIO 9000 端口当 PublicEndpoint，临时
			// 创建 PresignClient 签一次。minio.New 本身不发网络请求（Region 已显式
			// 指定，跳过 GetBucketLocation 探测），开销可忽略。
			signer := minioClient
			snap := ufteTransferPolicy.Snapshot(ctx)
			if snap.Upload.BaseURL != "" {
				if parsed, perr := url.Parse(snap.Upload.BaseURL); perr == nil && parsed.Hostname() != "" {
					customCfg := c.Cfg.MinIO
					customCfg.PublicEndpoint = parsed.Hostname() + ":9000"
					if pc, nerr := minioinfra.NewPresignClient(customCfg); nerr == nil {
						signer = pc
					}
				}
			}
			u, presignErr := signer.PresignedGetObject(ctx, bucket, objectPath, time.Hour, url.Values{})
			if presignErr != nil {
				return "", presignErr
			}
			return u.String(), nil
		}
		return "", nil // metadata not yet populated → caller leaves URL blank
	})

	// 注入"backup_restore_file 反查"回调——按 (sn, mainTaskID) 精确反查。
	// 从 migrations/000133 起 backup_restore_file 加了 task_id 字段（UFTE 主任务 UUID），
	// 唯一键 (sn, task_id, file_name) 保证不同任务隔离。这里直接按 task_id 精确反查
	// 当前任务的落地记录，返回设备真实文件名。匹配不到（设备未上报 / 旧数据无 task_id）
	// 时返回 ("", false)，DeviceItem.TargetFile 留空。
	service.SetFileLandedLookup(func(ctx context.Context, sn, mainTaskID string) (string, bool, error) {
		if sn == "" || mainTaskID == "" {
			return "", false, nil
		}
		files, lookupErr := backupFileRepo.ListBySerial(ctx, sn)
		if lookupErr != nil {
			return "", false, lookupErr
		}
		for i := range files {
			f := &files[i]
			if f.ObjectPath == "" {
				continue
			}
			if f.TaskID == nil || *f.TaskID != mainTaskID {
				continue
			}
			return f.FileName, true, nil
		}
		return "", false, nil
	})

	// 注入"日志文件元数据是否已标记删除"回调。UFTE 的文件名/下载列来自
	// backup_restore_file；故障日志文件数配额清理也记录在 backup_restore_file.is_deleted。
	// 按 (sn, task_id, file_name) 精确命中当前 UFTE 任务，避免同设备重复上传同名厂商日志时误判。
	service.SetFileDeletedLookup(func(ctx context.Context, sn, mainTaskID, fileName string) (bool, error) {
		if sn == "" || mainTaskID == "" || fileName == "" {
			return false, nil
		}
		files, lookupErr := backupFileRepo.ListBySerial(ctx, sn)
		if lookupErr != nil {
			return false, lookupErr
		}
		for i := range files {
			f := &files[i]
			if f.FileName != fileName || f.ObjectPath == "" {
				continue
			}
			if f.TaskID == nil || *f.TaskID != mainTaskID {
				continue
			}
			return f.IsDeleted, nil
		}
		return false, nil
	})

	// 注入"设备上线即重试"回调：让 software.HandleDeviceOnline 在 LogCollect 类
	// 子任务被唤醒时能复用 UFTE catalog 解析 transport_path 后重启 Upload RPC。
	// 不装配则用户挂起→开始时若设备恰好离线，子任务会永远停在 suspended。
	// 详见 docs/project/backup-display-fix-20260520.md F6-F9。
	if c.miscDeps.softwareService != nil {
		c.miscDeps.softwareService.SetLogCollectResumer(service)
	}

	// 注入"任务删除时清理 MinIO + backup_restore_file"的回调：UFTE 文件传输
	// 任务（CONFIG_BACKUP_*/RUNTIME_LOG_COLLECT/FAULT_LOG_COLLECT）的产物
	// 落在 config_backup / logs 等 bucket，DeleteUpgrade 默认只删 DB 任务行，
	// MinIO 上的备份/日志文件会成为孤儿。这里按 task_id 回收：
	//   1) 反查 backup_restore_file 拿到 (bucket, objectPath) 列表（同步）
	//   2) 删 backup_restore_file 元数据（同步，让接口尽快返回）
	//   3) **后台 goroutine** 逐个 RemoveObject（异步，best-effort）
	// 元数据先删 → UI 立即看到任务消失；MinIO 慢请求不阻塞响应。MinIO 失败
	// 也不影响 DB 一致性（最坏只是孤儿对象，由运维定期 bucket lifecycle 兜底）。
	// config_snapshots 不在此清理——它是"每设备最新一份"的辅助索引，独立 bucket，
	// 由 promote 链路覆盖式维护。
	// 注意：清理用 c.MinIO（内部客户端，连 docker 服务名 minio:9000），不能用
	// 上面那个 minioClient（被覆盖成 PresignClient，public_endpoint=localhost:9000，
	// 容器内连不通）。PresignClient 只该用来签 presigned URL，不该跑真实 API。
	internalMinIO := c.MinIO
	if c.miscDeps.softwareService != nil && internalMinIO != nil {
		c.miscDeps.softwareService.SetTaskFileCleaner(func(ctx context.Context, taskID uuid.UUID) error {
			taskIDStr := taskID.String()
			files, listErr := backupFileRepo.ListByTaskID(ctx, taskIDStr)
			if listErr != nil {
				return fmt.Errorf("list backup_restore_file by task %s: %w", taskIDStr, listErr)
			}
			if delErr := backupFileRepo.DeleteByTaskID(ctx, taskIDStr); delErr != nil {
				return fmt.Errorf("delete backup_restore_file by task %s: %w", taskIDStr, delErr)
			}
			if len(files) == 0 {
				return nil
			}
			// 异步清理 MinIO 对象 — 不阻塞 HTTP 响应。用独立 ctx（不继承请求 ctx）
			// 避免响应返回后 ctx 被 cancel 导致 RemoveObject 中断。2 分钟超时兜底
			// 防止 MinIO 卡死时 goroutine 永远跑。
			go func(toRemove []backup.BackupRestoreFile) {
				asyncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				removed, failed := 0, 0
				for i := range toRemove {
					if toRemove[i].ObjectPath == "" {
						continue
					}
					bucket, objectPath, splitErr := backup.SplitBucketAndPath(toRemove[i].ObjectPath)
					if splitErr != nil || bucket == "" || objectPath == "" {
						logger.Warn("skip MinIO removal: bad object_path",
							zap.String("task_id", taskIDStr),
							zap.String("object_path", toRemove[i].ObjectPath))
						failed++
						continue
					}
					if rmErr := internalMinIO.RemoveObject(asyncCtx, bucket, objectPath, minio.RemoveObjectOptions{}); rmErr != nil {
						logger.Warn("async remove MinIO object failed",
							zap.String("task_id", taskIDStr),
							zap.String("bucket", bucket),
							zap.String("path", objectPath),
							zap.Error(rmErr))
						failed++
						continue
					}
					removed++
				}
				logger.Info("async MinIO cleanup done",
					zap.String("task_id", taskIDStr),
					zap.Int("removed", removed),
					zap.Int("failed", failed))
			}(files)
			return nil
		})
	}

	c.miscDeps.ufteHandler = ufte.NewHandler(service, logger)
	c.miscDeps.ufteService = service

	// 把"定时任务调度器触发 LogCollect 类任务"的回调指向 ufte.Service.StartTask。
	// StartTask 内部已经按 TaskType / TypeCode 分流到 ResumeCollect 或 startDirectDispatchTask，
	// scheduler 不用关心子类型。LogCollect 之外的（升级 / 回退）走 SoftwareService.ResumeUpgrade，
	// 不经过本回调。
	if c.miscDeps.taskScheduler != nil {
		c.miscDeps.taskScheduler.SetCollectTrigger(func(ctx context.Context, taskID uuid.UUID) error {
			// scheduler 是进程内可信触发（任务创建时已校验过设备归属），传 nil 等价
			// 超管不再二次按租户限制（#63 设备组可见性只约束 HTTP 暴露面）。
			return service.StartTask(ctx, taskID, nil)
		})
		logger.Info("task scheduler collect trigger wired to ufte.Service.StartTask")
	}

	logger.Info("UFTE adapter module initialized", zap.Int("built_in_task_types_inserted", inserted))
	return nil
}

// initProvisionModule 初始化 F09 自动开站模块。
// T-0098 P5-01：dmRegistry / dmImporter 已删除，改由 paramRegistry / productRegistry / intersectService 接管。
func initProvisionModule(c *Container) error {
	logger := c.Logger.Named("provision")

	provisionRepo := provision.NewPgProvisioningTaskRepository(c.PgPool)
	plugAndPlayRepo := provision.NewPgPlugAndPlayRepository(c.PgPool)
	discoveryLogRepo := provision.NewPgParameterDiscoveryLogRepository(c.PgPool)
	provisionEngine := provision.NewProvisioningEngine(
		provisionRepo, c.DeviceService, c.TemplateService,
		c.Carriers, c.TaskSvc, c.EventBus, c.Cfg.Provision, logger,
	)
	provisionEngine.SetDeduper(c.Deduper)
	provisionEngine.SetParamSyncRoutingMode(c.Cfg.ParamSync.RoutingMode)
	provisionEngine.SetActivationStateReader(c.DeviceInfoRepo)
	provisionEngine.SetActivationStateRefresher(device.NewInfoSyncer(
		c.DeviceInfoRepo,
		c.ParamRepo,
		device.NewPgDeviceRepository(c.PgPool),
		c.Carriers,
		logger,
		device.NewPgLocationObservationRepository(c.PgPool),
	))
	if c.miscDeps.paramSyncStarter != nil {
		provisionEngine.SetRegisteredDeviceSyncStarter(c.miscDeps.paramSyncStarter)
		provisionEngine.SetDeviceOnlineFullSyncSubmitter(c.miscDeps.paramSyncStarter)
	}
	// HIGH-27 / MEDIUM-19：provisioning 指标（discovery_log 状态写库失败、Redis 节流失败）。
	provisionMetrics := provision.NewMetrics(c.MetricsReg)
	provisionEngine.SetMetrics(provisionMetrics)
	// T-0123: 注入 Redis 客户端供 device.online 节流（provision:online_sync:{deviceID} TTL=60s）+
	// Path B 同步 reason 标签（provision:syncreason:{deviceID} TTL=10min）。
	provisionEngine.SetRedisClient(c.Redis)
	// B1：identify 阶段路由产品并回写 product_id/param_model_id。
	if c.ProductRegistry != nil && c.ProductRepo != nil {
		provisionEngine.SetProductBinder(c.ProductRegistry, c.ProductRepo)
	}
	// T-0176-PR-D：bindDeviceProductIfNeeded 在 DeviceOnline / FirmwareChanged /
	// FileType11 三个非 Bootstrap 入口懒补绑定，写库成功后失效 SN cache。
	if c.DeviceCache != nil {
		provisionEngine.SetDeviceCache(c.DeviceCache)
	}

	if c.Cfg.Provision.ModelUpload.Enabled {
		modelUploadSvc := provision.NewModelUploadService(
			discoveryLogRepo, c.TaskSvc, c.MinIO,
			c.ProductRegistry, c.ParamIntersect,
			c.Cfg.Provision.ModelUpload, logger,
		)
		modelUploadSvc.SetMetrics(provisionMetrics)
		if c.miscDeps.paramSyncStarter != nil {
			modelUploadSvc.SetParamSyncSubmitter(c.miscDeps.paramSyncStarter)
		}
		provisionEngine.SetModelUploadService(modelUploadSvc)
		logger.Info("model upload service enabled",
			zap.String("upload_url", c.Cfg.Provision.ModelUpload.UploadURL))
	}
	var periodicSyncStarter provision.PathBSyncStarter
	if c.Cfg.Provision.AutoSync.Enabled {
		planStore := provision.NewSyncPlanStore(c.Redis)
		syncSvc := provision.NewSyncService(
			c.ParamRepo, discoveryLogRepo, c.TaskSvc, planStore,
			c.Cfg.Provision.AutoSync, c.Cfg.Provision.AutoSync.GPVBatchSize, logger,
		).WithParamRegistry(c.ParamRegistry, c.ProductRegistry, true).
			SetRedisClient(c.Redis).
			SetParamSyncWriter(c.DeviceRepo).
			SetPathBSyncTaskReader(task.NewPgTaskRepository(c.PgPool)).
			SetDeviceInfoRefresher(device.NewInfoSyncer(c.DeviceInfoRepo, c.ParamRepo, device.NewPgDeviceRepository(c.PgPool), c.Carriers, logger, device.NewPgLocationObservationRepository(c.PgPool))) // Path B 参数落库后立即刷新 device_info 快照

		// Issue #758: 设备名称同步钩子装配
		// Path B 同步完成后比对 LMT 设备名与网管名，按配置方向自动同步或标记待确认。
		nameSyncCfgRepo := admin.NewPgSysConfigRepository(c.PgPool)
		nameSyncConfigLookup := provision.NameSyncConfigLookup(func(ctx context.Context, cat, key string) (string, bool) {
			cfg, err := nameSyncCfgRepo.GetByKey(ctx, cat, key)
			if err != nil || cfg == nil {
				return "", false
			}
			return cfg.Value, true
		})
		deviceNameSyncHook := provision.NewDeviceNameSyncHook(
			nameSyncConfigLookup, c.ParamRepo, c.DeviceInfoRepo, c.DeviceInfoRepo, logger,
		).SetSiteUpdater(device.NewPgDeviceRepository(c.PgPool)).
			SetSPVSender(provision.NewDeviceNameTaskSender(c.TaskSvc))
		syncSvc.SetDeviceNameSyncHook(deviceNameSyncHook)
		logger.Info("device name sync hook enabled (Issue #758)")

		c.SyncSvc = syncSvc
		periodicSyncStarter = syncSvc
		if c.miscDeps.paramSyncStarter != nil {
			syncSvc.SetDurableStarter(c.miscDeps.paramSyncStarter)
		}
		provisionEngine.SetSyncService(syncSvc)
		// T-0126: 注入 ParamSyncStarter 让手动同步先走 durable parameter_sync_*。
		// 旧 sync-gpv Path B 仅作为临时兜底，待 param_sync_running 稳定后删除。
		if c.DeviceService != nil {
			c.DeviceService.SetParamSyncRoutingMode(c.Cfg.ParamSync.RoutingMode)
			c.DeviceService.SetParamSyncManualOfflineMode(c.Cfg.ParamSync.ManualOfflineMode)
			if c.miscDeps.paramSyncStarter != nil {
				c.miscDeps.paramSyncStarter.SetLegacy(syncSvc)
				c.DeviceService.SetParamSyncStarter(c.miscDeps.paramSyncStarter)
			} else {
				c.DeviceService.SetParamSyncStarter(syncSvc)
			}
		}
		logger.Info("auto-sync service enabled")

		// License Params Tab 后端装配（DeviceDetail "License 参数" tab）。
		// 刷新直接提交到 durable paramsync 数据面，不经过 provision.SyncService
		// 或旧 Path B 调度器。
		if c.DeviceRepo != nil && c.ParamRepo != nil &&
			c.ProductRegistry != nil && c.ParamRegistry != nil &&
			c.miscDeps.paramSyncStarter != nil {
			licenseParamSvc := device.NewLicenseParamService(
				c.DeviceRepo, c.ParamRepo,
				c.ProductRegistry, c.ParamRegistry,
				c.miscDeps.paramSyncStarter, c.Redis, logger,
			)
			c.miscDeps.licenseParamHandler = device.NewLicenseParamHandler(licenseParamSvc, logger)
			logger.Info("device license params handler initialized")
		}

	}

	releaseCampaignID, hasReleaseIdentity := buildinfo.ReleaseCampaignID()
	releaseSyncReady := hasReleaseIdentity &&
		c.miscDeps.paramSyncStarter != nil &&
		c.Cfg.ParamSync.RoutingMode == "durable"
	if periodicSyncStarter != nil || releaseSyncReady {
		// T-0124 周期同步与 Issue #148 发布同步共用同一个 scheduler、
		// PG leader、批次、并发和 stagger 参数。
		sysCfgRepo := admin.NewPgSysConfigRepository(c.PgPool)
		periodicSyncLookup := provision.SysConfigLookup(func(ctx context.Context, cat, key string) (string, bool) {
			cfg, err := sysCfgRepo.GetByKey(ctx, cat, key)
			if err != nil || cfg == nil {
				return "", false
			}
			return cfg.Value, true
		})
		periodicSyncPolicy := provision.NewPeriodicSyncPolicy(periodicSyncLookup, logger)
		// FE 保存"设备设置"页后立即让 PeriodicSyncPolicy 30s 缓存失效，
		// scheduler 下次 tick（≤ 1 分钟）就读到最新配置；无 hook 时要等 ≤90s 才生效。
		if c.SysConfigSvc != nil {
			c.SysConfigSvc.RegisterSavedHook(func(_ context.Context, category string) {
				if category == "device" {
					periodicSyncPolicy.InvalidateCache()
				}
			})
		}
		leader := provision.NewPGAdvisoryLeaderElector(c.PgPool, "periodic_param_syncer", logger)
		periodicSyncer := provision.NewPeriodicSyncer(
			c.DeviceRepo, periodicSyncStarter, leader,
			periodicSyncPolicy, logger,
		)
		periodicSyncer.SetParamSyncRoutingMode(c.Cfg.ParamSync.RoutingMode)
		if releaseSyncReady {
			periodicSyncer.SetReleaseSync(
				paramsync.NewPGRepository(c.PgPool),
				c.miscDeps.paramSyncStarter,
				releaseCampaignID,
			)
		}
		go func() {
			if err := periodicSyncer.Start(context.Background()); err != nil && err != context.Canceled {
				logger.Warn("periodic syncer exited with error", zap.Error(err))
			}
		}()
		logger.Info("parameter sync scheduler started",
			zap.Bool("periodic_enabled", periodicSyncStarter != nil),
			zap.Bool("release_sync_enabled", releaseSyncReady))
	}

	if err := provisionEngine.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe provisioning engine", zap.Error(err))
	}
	provisionEngine.StartTaskReaper()
	logger.Info("provisioning engine started",
		zap.Bool("auto_configure", c.Cfg.Provision.AutoConfigure),
		zap.Bool("model_upload", c.Cfg.Provision.ModelUpload.Enabled),
		zap.Bool("auto_sync", c.Cfg.Provision.AutoSync.Enabled),
	)

	c.miscDeps.provisionRepo = provisionRepo
	c.miscDeps.provisionEngine = provisionEngine
	c.miscDeps.plugAndPlayRepo = plugAndPlayRepo
	registerProvisionCompletionHandler(
		c.miscDeps.completionRouter, c.miscDeps.taskSvc, provisionEngine, logger,
	)

	return nil
}

// registerProvisionCompletionHandler wires terminal device-task results only
// after ProvisioningEngine exists. The misc module creates CompletionRouter
// before the provision module is initialized, so attempting this registration
// from initMiscModules silently skips it and leaves PnP Download SOAP faults to
// be reported later as generic provisioning timeouts.
func registerProvisionCompletionHandler(
	router *task.CompletionRouter,
	taskSvc *task.TaskService,
	handler task.TaskCompletionCallback,
	logger *zap.Logger,
) {
	if handler == nil {
		return
	}
	if router != nil {
		router.Register(task.TaskSourceSystem, handler)
		logger.Info("provision completion callback registered to TaskSourceSystem")
		return
	}
	if taskSvc != nil {
		taskSvc.AddCompletionCallback(handler)
		logger.Info("provision completion callback registered to TaskService")
		return
	}
	logger.Warn("task completion routing unavailable; provisioning device-task failures will not propagate")
}

// initTaskModule 初始化 F06 任务队列模块。
// TaskService 核心已在 bootstrap 中创建（供 BridgeQueue 使用），
// 此处仅添加运行时增强（指标、Connection Request）并注册 handler。
func initTaskModule(c *Container) error {
	logger := c.Logger.Named("task")

	taskService := c.TaskSvc
	taskMetrics := task.NewTaskMetrics(c.MetricsReg)
	taskService.SetMetrics(taskMetrics)
	redisQueueObserver := task.NewRedisQueueObserver(c.Redis, taskMetrics, 30*time.Second, logger)
	redisQueueObserver.Start(context.Background())
	if c.GS != nil {
		c.GS.Register("redis-task-queue-observer", 1, func(context.Context) error {
			redisQueueObserver.Stop()
			return nil
		})
	}
	persistentQueueMetrics := components.NewPersistentQueueMetrics(c.MetricsReg)
	persistentQueueObserver := components.NewPersistentQueueObserver(c.PgPool, persistentQueueMetrics, 30*time.Second, logger)
	persistentQueueObserver.Start(context.Background())
	if c.GS != nil {
		c.GS.Register("persistent-queue-observer", 1, func(context.Context) error {
			persistentQueueObserver.Stop()
			return nil
		})
	}

	// Wire Connection Request into TaskService
	udpSender := connreq.NewUDPSender(c.StunStore, c.Cfg.ConnReq.SharedSecret, c.Logger)
	crDispatcher := connreq.NewDispatcher(c.ConnReqClient, udpSender, c.Logger)
	crDispatcher.SetMetrics(connreq.NewDispatcherMetrics(c.MetricsReg))
	crDispatcher.SetLANPortLookup(newSTUNServerPortLookup(c.DeviceRepo, c.ParamRepo, c.Logger))
	taskService.SetConnectionRequester(
		&taskDeviceLookup{repo: c.DeviceRepo},
		&taskCRSender{dispatcher: crDispatcher, serverAddr: c.Cfg.ConnReq.ServerAddr},
	)

	c.miscDeps.taskHandler = task.NewHandler(taskService)
	c.miscDeps.taskSvc = taskService

	logger.Info("task queue module initialized")
	return nil
}

// initBackupModule 初始化 F06 备份模块。
func initBackupModule(c *Container) error {
	logger := c.Logger.Named("backup")

	backupTaskRepo := backup.NewPgTaskRepository(c.PgPool)
	backupScheduleRepo := backup.NewPgScheduleRepository(c.PgPool)
	ftpRepo := backup.NewPgFTPConfigRepository(c.PgPool)
	backupService := backup.NewService(backupTaskRepo, backupScheduleRepo, c.EventBus, logger)
	backupHandler := backup.NewHandler(backupService, ftpRepo, logger)
	if c.ProductRegistry != nil {
		backupHandler.SetProductPatternResolver(c.ProductRegistry) // #602 配置快照/许可按产品名称下拉过滤
	}

	// T-0071 / R-102 followup: singleton BackupPolicy persistence.
	policyRepo := backup.NewPgPolicyRepository(c.PgPool)
	policyService := backup.NewPolicyService(policyRepo, logger)
	// T-0075: wire KeyProvider for PUT-time validation. The same env var is
	// read on both ACS and App processes — keep them in sync via deployment
	// config (systemd EnvironmentFile= or k8s ConfigMap).
	// 捕获到函数作用域：除 policyService 外，#61 的快照 promote 解码管线也需要它
	// 构造 AES-256-GCM 解密器（解密源备份对象还原明文快照）。
	var backupKeyProvider backup.KeyProvider
	if kp, kpErr := backup.NewEnvKeyProvider(); kpErr != nil {
		logger.Error("backup encryption key invalid; PUT /backup/policy will block AES-256-GCM",
			zap.Error(kpErr))
	} else {
		backupKeyProvider = kp
		policyService.SetKeyProvider(kp)
	}
	backupHandler.SetPolicyService(policyService)

	// T-0073 Phase 1: BackupPolicy enforcement Monitor (cleanup cron) +
	// metrics. Failure-alarm publishing is wired on the worker side
	// (executor) — see cmd/worker/main.go.
	policyMetrics := backup.NewPolicyMetrics(c.MetricsReg)
	backupPolicyMonitor := backup.NewPolicyMonitor(policyService, backupTaskRepo, policyMetrics, logger)
	// T-0076 Phase 2: enable physical MinIO object deletion alongside DB
	// cleanup. minio is optional (nil-safe); when wired, RunCleanupOnce
	// calls RemoveObject for each deleted backup_task's file_path.
	backupPolicyMonitor.SetMinIO(c.MinIO)
	// T-0082: enable hourly bucket-usage poll + edge-trigger alarm.raised/
	// cleared. Both BucketLister and EventBus must be set; either nil
	// disables the storage check (preserves T-0073/T-0076 behaviour).
	backupPolicyMonitor.SetBucketLister(c.MinIO)
	backupPolicyMonitor.SetEventBus(c.EventBus)
	// T-0083: enable weekly multi-device orphan reaper. PgTaskRepository's
	// ListAllTaskIDPrefixes satisfies the narrow TaskIDLister contract;
	// reaper additionally requires SetMinIO + SetBucketLister wired above.
	backupPolicyMonitor.SetTaskIDLister(backupTaskRepo)
	if err := backupPolicyMonitor.Start(context.Background()); err != nil {
		logger.Warn("start backup policy monitor", zap.Error(err))
	}
	c.miscDeps.backupPolicyMonitor = backupPolicyMonitor

	// T-0072 / R-102 followup: restore endpoint + restore_tasks tracking.
	restoreRepo := backup.NewPgRestoreTaskRepository(c.PgPool)
	restoreMetrics := backup.NewRestoreMetrics(c.MetricsReg)
	restoreService := backup.NewRestoreService(
		restoreRepo, c.DeviceRepo, c.TaskSvc, c.MinIO, restoreMetrics, logger,
	)
	// 配置文件恢复在下发时读 MinIO 文件流现算 Download MD5（Download 报文必填字段）。
	restoreService.SetObjectReader(backup.NewMinIOObjectReader(c.MinIO))
	// #70 task 3：跨版本检查（快照来源版本 vs 目标设备当前固件版本）。默认 warn+audit
	// 策略，不阻断；audit sink 暂传 nil（仅 warn 日志），后续可注入 ops_audit_logs 适配器。
	restoreService.SetCrossVersionChecker(
		backup.NewCrossVersionChecker(backup.CrossVersionWarnAudit, nil, logger),
	)
	// T-0079: enable by-task-id restore mode + subscribe FilePathRecorder to
	// `backup.file.received` events so backup_tasks.file_path is populated
	// after CPE finishes uploading. Both wire onto the same RestoreMetrics.
	restoreService.SetBackupTaskFinder(backupTaskRepo)
	filePathRecorder := backup.NewFilePathRecorder(backupTaskRepo, restoreMetrics, logger)
	backupRestoreFileRepo := backup.NewPgFileRepository(c.PgPool)
	// M1 of backup-restore-alignment-plan: 同步落库 backup_restore_file 元数据
	// （SN/file_name/md5/size/operator_code/update_time），支撑后续查询与导出。
	filePathRecorder.SetFileRepository(backupRestoreFileRepo)
	logFileRetentionSysCfg := admin.NewPgSysConfigRepository(c.PgPool)
	logFileRetentionPolicy := stationlog.NewRetentionPolicy(
		func(ctx context.Context, category, key string) (string, bool) {
			row, err := logFileRetentionSysCfg.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		},
		logger,
	)
	if c.MinIO != nil {
		filePathRecorder.SetLogFileQuota(backupRestoreFileRepo, logFileRetentionPolicy, c.MinIO)
	} else {
		filePathRecorder.SetLogFileQuota(backupRestoreFileRepo, logFileRetentionPolicy, nil)
	}
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterSavedHook(func(_ context.Context, category string) {
			if category == stationlog.RetentionCategory {
				logFileRetentionPolicy.InvalidateCache()
				go func() {
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()
					filePathRecorder.EnforceAllLogFileQuotas(cleanupCtx)
				}()
			}
		})
	}
	// FAULT_LOG_COLLECT / RUNTIME_LOG_COLLECT 的"文件落地即任务成功"hook：
	// BACKUP stream 是 WorkQueuePolicy，软件包不能再订一份 backup.file.received，
	// 走同进程 hook 让 FilePathRecorder 处理完元数据后回调 software 推进 sub_task。
	if c.miscDeps.softwareService != nil {
		filePathRecorder.SetFileLandedNotifier(c.miscDeps.softwareService)
	}
	// T-0164 B6: ConfigSnapshot 子系统装配（独立 bucket / 表 / Service / Handler）。
	// 历史数据不回填（用户确认 2026-05-22），表上线后从首次新备份或导入开始累积。
	snapshotRepo := backup.NewPgSnapshotRepository(c.PgPool)
	snapshotFileLookup := backup.NewPgFileRepository(c.PgPool) // 复用 backup_restore_file repo
	snapshotService := backup.NewSnapshotService(
		snapshotRepo, c.MinIO, snapshotFileLookup, c.DeviceRepo,
		backup.SnapshotBucketDefault, logger,
	)
	c.miscDeps.snapshotService = snapshotService
	// #61: promote 解码管线 —— 读源备份对象 + AES-256-GCM 解密器，把(压缩/加密的)
	// 备份还原成明文写快照（替代会产生不可用快照的 server-side CopyObject）。
	// 源读取器在 MinIO 注入时才装；解密器仅在 KEK 可用时装（与上传加密算法一致），
	// 不可用时只能 promote 未加密备份、遇加密源拒绝（不写坏快照）。
	if c.MinIO != nil {
		var snapshotDecryptor backup.Encryptor
		if backupKeyProvider != nil {
			if enc, encErr := backup.NewEncryptor("AES-256-GCM", backupKeyProvider); encErr != nil {
				logger.Warn("snapshot promote decryptor unavailable; encrypted backups will not promote",
					zap.Error(encErr))
			} else {
				snapshotDecryptor = enc
			}
		}
		snapshotService.SetPromoteDecoder(
			backup.NewMinIOSnapshotSourceReader(c.MinIO), snapshotDecryptor,
		)
	}
	// promote hook：备份成功后自动 promote 到 config_snapshots。
	filePathRecorder.SetSnapshotPromoter(snapshotService)
	// by-snapshot 恢复模式：每设备取自己最新快照。
	restoreService.SetSnapshotLookup(snapshotService)
	backupHandler.SetSnapshotService(snapshotService)
	// T-0164: 让 UFTE CONFIG_RESTORE 走 backup.RestoreService.CreateBySnapshot 链路。
	// 反向注入：ufte init 早于 backup，必须等 restoreService 装配完成才能 wire。
	if c.miscDeps.ufteService != nil {
		c.miscDeps.ufteService.SetSnapshotConfigRestoreDispatcher(restoreService)
	}
	// 启动期幂等创建 bucket（MinIOClient 注入存在才执行）。
	if c.MinIO != nil {
		if err := ensureSnapshotBucket(context.Background(), c.MinIO, backup.SnapshotBucketDefault); err != nil {
			// 失败不阻塞模块启动 —— Service 调用时 MinIO 会返回更明确的错误，
			// 比此处启动期 panic 更便于排查。
			logger.Warn("ensure config-snapshots bucket failed; service may 5xx until fixed",
				zap.String("bucket", backup.SnapshotBucketDefault), zap.Error(err))
		}
	}

	// T-0165: DeviceLicense 子系统装配 — 与 SnapshotService 同款，但只支持手动导入，
	// 派发 LICENSE_UPGRADE 任务时直接构造 device_tasks（FileType="License File"）。
	licenseRepo := backup.NewPgLicenseRepository(c.PgPool)
	licenseService := backup.NewLicenseService(
		licenseRepo, c.MinIO, c.DeviceRepo,
		c.miscDeps.taskSvc, backup.LicenseBucketDefault, logger,
	)
	backupHandler.SetLicenseService(licenseService)
	licensePreinstallSubscriber := backup.NewLicensePreinstallSubscriber(licenseService, logger)
	if err := licensePreinstallSubscriber.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe license preinstall dispatcher", zap.Error(err))
	}
	if c.miscDeps.ufteService != nil {
		c.miscDeps.ufteService.SetLicenseUpgradeDispatcher(licenseService)
	}
	if c.MinIO != nil {
		if err := ensureSnapshotBucket(context.Background(), c.MinIO, backup.LicenseBucketDefault); err != nil {
			logger.Warn("ensure device-licenses bucket failed; service may 5xx until fixed",
				zap.String("bucket", backup.LicenseBucketDefault), zap.Error(err))
		}
	}

	if err := filePathRecorder.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe backup file path recorder", zap.Error(err))
	}
	backupHandler.SetRestoreService(restoreService)

	// ── 文件管理 4 Tab 批量下载（bundle 模块） ─────────────────────────────
	// 同步流式: handler 直接打 zip 到 response writer,浏览器一次下载。
	// 不存中间产物 / 不签 presigned URL / 不轮询 — 比异步方案干净得多。
	if c.MinIO != nil {
		bundleSvc := bundle.NewService(c.MinIO, logger)
		// #63 设备组可见性：批量下载按调用者可见设备组过滤（SN-keyed 模块入口预过滤，
		// 文件 ID 模块按 BundleFile.DeviceSN 后置剔除）。
		bundleSvc.SetSNVisibilityReader(bundle.NewPgSNVisibilityReader(c.PgPool))

		// Source #1 firmware: targetIDs = firmware_versions.id (UUID 字符串)。
		// 重建一个轻量 repo 避免依赖 initSoftwareModule 内的局部变量(那里
		// firmwareRepo 不在 miscDeps 暴露)。Pg constructor 只是 &Pg{pool}, 无副作用。
		firmwareBucket := c.Cfg.MinIO.Buckets.Firmware
		firmwareRepoForBundle := software.NewPgFirmwareRepository(c.PgPool)
		bundleSvc.Register(bundle.ModuleFirmware, func(ctx context.Context, ids []string) ([]bundle.BundleFile, error) {
			out := make([]bundle.BundleFile, 0, len(ids))
			for _, id := range ids {
				uid, err := uuid.Parse(id)
				if err != nil {
					continue
				}
				fw, err := firmwareRepoForBundle.GetByID(ctx, uid)
				if err != nil || fw == nil {
					continue
				}
				out = append(out, bundle.BundleFile{
					Bucket:     firmwareBucket,
					ObjectPath: fw.MinIOPath,
					EntryName:  fw.FileName,
				})
			}
			return out, nil
		})

		// Source #2 config_snapshot: targetIDs = serial_number 列表
		bundleSvc.Register(bundle.ModuleConfigSnapshot, func(ctx context.Context, sns []string) ([]bundle.BundleFile, error) {
			res, err := snapshotRepo.BatchGetBySerialNumbers(ctx, sns)
			if err != nil {
				return nil, err
			}
			out := make([]bundle.BundleFile, 0, len(res))
			for _, snap := range res {
				out = append(out, bundle.BundleFile{
					Bucket:     snap.ObjectBucket,
					ObjectPath: snap.ObjectPath,
					EntryName:  snap.FileName,
				})
			}
			return out, nil
		})

		// Source #3 device_license: targetIDs = serial_number 列表
		bundleSvc.Register(bundle.ModuleDeviceLicense, func(ctx context.Context, sns []string) ([]bundle.BundleFile, error) {
			res, err := licenseRepo.BatchGetBySerialNumbers(ctx, sns)
			if err != nil {
				return nil, err
			}
			out := make([]bundle.BundleFile, 0, len(res))
			for _, lic := range res {
				out = append(out, bundle.BundleFile{
					Bucket:     lic.ObjectBucket,
					ObjectPath: lic.ObjectPath,
					EntryName:  lic.FileName,
				})
			}
			return out, nil
		})

		// Source #4 mr: targetIDs = serial_number 列表,zip 里按
		// {prefix}/{SN}/{filename} 三层嵌套。最外层 prefix=mr-bundle-<时间>
		// 跟 zip 文件名一致,避免解压后直接撒一堆 SN 目录到当前文件夹。
		mrBucket := c.Cfg.MinIO.Buckets.MRFiles
		bundleSvc.Register(bundle.ModuleMR, func(ctx context.Context, sns []string) ([]bundle.BundleFile, error) {
			prefix := fmt.Sprintf("mr-bundle-%s", time.Now().Format("20060102-150405"))
			out := make([]bundle.BundleFile, 0, len(sns)*5)
			for _, sn := range sns {
				snCopy := sn
				files, ferr := c.miscDeps.mrStore.ListFiles(ctx, mr.MRFileFilter{
					DeviceSN:    &snCopy,
					ListRequest: model.ListRequest{Page: 1, PageSize: 10000},
				})
				if ferr != nil {
					logger.Warn("mr batch download: list files failed",
						zap.String("sn", sn), zap.Error(ferr))
					continue
				}
				for _, f := range files.Items {
					out = append(out, bundle.BundleFile{
						Bucket:     mrBucket,
						ObjectPath: f.MinioPath,
						EntryName:  prefix + "/" + sn + "/" + f.FileName,
					})
				}
			}
			return out, nil
		})

		// Source #5 mr_files: targetIDs = mr_files.id 列表,用户在 DeviceFilesDrawer
		// 勾选具体若干个文件下载。EntryName 跟 ModuleMR 同结构 {prefix}/{SN}/{filename},
		// 解压体验一致。GetFileByID N 次串行调用 — 一次 batch 通常 < 50 个文件,
		// 网络往返可接受;后续若性能瓶颈再加 BatchGetByIDs。
		bundleSvc.Register(bundle.ModuleMRFiles, func(ctx context.Context, ids []string) ([]bundle.BundleFile, error) {
			prefix := fmt.Sprintf("mr-bundle-%s", time.Now().Format("20060102-150405"))
			out := make([]bundle.BundleFile, 0, len(ids))
			for _, idStr := range ids {
				fid, perr := uuid.Parse(idStr)
				if perr != nil {
					logger.Warn("mr_files batch download: invalid uuid",
						zap.String("id", idStr), zap.Error(perr))
					continue
				}
				f, gerr := c.miscDeps.mrStore.GetFileByID(ctx, fid)
				if gerr != nil || f == nil {
					logger.Warn("mr_files batch download: get file failed",
						zap.String("id", idStr), zap.Error(gerr))
					continue
				}
				out = append(out, bundle.BundleFile{
					Bucket:     mrBucket,
					ObjectPath: f.MinioPath,
					EntryName:  prefix + "/" + f.DeviceSN + "/" + f.FileName,
					DeviceSN:   f.DeviceSN, // #63 供 WriteZipTo 按可见性后置剔除
				})
			}
			return out, nil
		})

		// Source #6 pm: targetIDs = serial_number 列表,zip 里按
		// {prefix}/{SN}/{filename} 三层嵌套,语义跟 ModuleMR 同构。
		// 用独立的 PMFileStore（NewPgPMFileStore 是无状态构造器，可与 ph.pmFileStore
		// 并存，不会产生竞争）。
		pmBucket := c.Cfg.MinIO.Buckets.PMFiles
		// KPI/时序库物理分离：pm_files 已迁时序库（TsPool）。
		pmFileStoreForBundle := pm.NewPgPMFileStore(c.TsPool)
		bundleSvc.Register(bundle.ModulePM, func(ctx context.Context, sns []string) ([]bundle.BundleFile, error) {
			prefix := fmt.Sprintf("pm-bundle-%s", time.Now().Format("20060102-150405"))
			out := make([]bundle.BundleFile, 0, len(sns)*5)
			for _, sn := range sns {
				snCopy := sn
				files, ferr := pmFileStoreForBundle.ListFiles(ctx, pm.PMFileFilter{
					DeviceSN:    &snCopy,
					ListRequest: model.ListRequest{Page: 1, PageSize: 10000},
				})
				if ferr != nil {
					logger.Warn("pm batch download: list files failed",
						zap.String("sn", sn), zap.Error(ferr))
					continue
				}
				for _, f := range files.Items {
					out = append(out, bundle.BundleFile{
						Bucket:     pmBucket,
						ObjectPath: f.MinioPath,
						EntryName:  prefix + "/" + sn + "/" + f.FileName,
					})
				}
			}
			return out, nil
		})

		// Source #7 pm_files: targetIDs = pm_files.id 列表（DeviceFilesDrawer 勾选）。
		bundleSvc.Register(bundle.ModulePMFiles, func(ctx context.Context, ids []string) ([]bundle.BundleFile, error) {
			prefix := fmt.Sprintf("pm-bundle-%s", time.Now().Format("20060102-150405"))
			out := make([]bundle.BundleFile, 0, len(ids))
			for _, idStr := range ids {
				fid, perr := uuid.Parse(idStr)
				if perr != nil {
					logger.Warn("pm_files batch download: invalid uuid",
						zap.String("id", idStr), zap.Error(perr))
					continue
				}
				f, gerr := pmFileStoreForBundle.GetFileByID(ctx, fid)
				if gerr != nil || f == nil {
					logger.Warn("pm_files batch download: get file failed",
						zap.String("id", idStr), zap.Error(gerr))
					continue
				}
				out = append(out, bundle.BundleFile{
					Bucket:     pmBucket,
					ObjectPath: f.MinioPath,
					EntryName:  prefix + "/" + f.DeviceSN + "/" + f.FileName,
					DeviceSN:   f.DeviceSN, // #63 供 WriteZipTo 按可见性后置剔除
				})
			}
			return out, nil
		})

		c.miscDeps.bundleSvc = bundleSvc
		logger.Info("bundle service wired (sync streaming)",
			zap.Int("sources_registered", 7))
	}

	// T-0032 + T-0093: FTP/SFTP/FTPS connection-test service with
	// default 5s timeout. FTP path uses stdlib net/textproto; SFTP
	// uses x/crypto/ssh password auth; FTPS uses crypto/tls implicit
	// (port 990 style) handshake followed by FTP USER/PASS over TLS.
	backupHandler.SetFTPTester(backup.NewFTPConnectionTester(nil, 0, logger))

	// M4: ExportFile 依赖（按 SN 列文件 + 生成 MinIO presigned URL）
	backupHandler.SetFileRepository(backup.NewPgFileRepository(c.PgPool))
	if c.MinIO != nil {
		backupHandler.SetObjectClient(c.MinIO)
		// 注入 PresignClient 而不是内部 client —— 内部 client 的 endpoint 是
		// `minio:9000`（docker 服务名），签出来的 presigned URL 浏览器解析不了
		// 会报 ERR_NAME_NOT_RESOLVED。PresignClient 用 minio.public_endpoint
		// 重签为对外可达 host（如 localhost:9000）。
		// 与 UFTE 模块 SetDownloadURLLookup 同一套逻辑（见上方 line 207-213）。
		//
		// issue #548 切片 4：同时注入 PresignBridge 作运行时 provider，sys_configs
		// 写入 storage.minio_public_endpoint 后下一次 M4 ExportFile 签名 URL 立即用新 endpoint，
		// 无需重启容器。License / 快照下载已改为同源流式返回，走上面的 objectClient。
		presignSigner := c.MinIO
		// 传入 logger：public_endpoint 为空回退内部 endpoint 时会打 Warn（qa-614 #377），
		// 让运维知道 ExportFile 预签名下载 URL 浏览器可能解析失败。
		if pc, perr := minioinfra.NewPresignClient(c.Cfg.MinIO, logger); perr != nil {
			logger.Warn("create MinIO presign client failed for backupHandler; download URLs may be unreachable",
				zap.Error(perr))
		} else {
			presignSigner = pc
		}
		backupHandler.SetMinioClient(presignSigner)
		if c.PresignBridge != nil {
			backupHandler.SetPresignProvider(c.PresignBridge)
		}
	}
	// M4: device 相关端点（QueryCellInfos / QueryTaskDeviceList / GetProductType）
	backupHandler.SetDeviceReader(c.DeviceRepo)

	c.miscDeps.backupHandler = backupHandler

	logger.Info("backup module initialized")
	return nil
}

// ensureSnapshotBucket 启动期幂等创建 config-snapshots bucket（T-0164 B6）。
// 与 core/components/minio.EnsureBuckets 的差异：那是配置驱动的统一批量创建，
// 此处是 backup 模块私有的额外 bucket，不进 BucketConfig（避免污染配置 schema）。
func ensureSnapshotBucket(ctx context.Context, client *minio.Client, bucket string) error {
	if bucket == "" {
		return nil
	}
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", bucket, err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket %s: %w", bucket, err)
		}
	}
	return nil
}

// initStationLogModule 初始化基站日志采集模块（运行日志 + 异常重启记录）。
func initStationLogModule(c *Container) error {
	logger := c.Logger.Named("stationlog")

	runningRepo := stationlog.NewPgRunningRepository(c.PgPool)
	faultRepo := stationlog.NewPgFaultRepository(c.PgPool)
	svc := stationlog.NewService(runningRepo, faultRepo, c.DeviceService, c.MinIO, c.Cfg.MinIO.Buckets, logger)
	// #63 租户隔离：注入设备组归属读取器，按记录归属设备校验下载 / 删除 / 详情。
	svc.SetGroupReader(device.NewPgDeviceGroupReader(c.PgPool))

	// #320：注入保留策略，使故障日志文件数配额可经 sys_configs 配置（stationlog.retention.
	// max_file_count，默认 20，0=禁用）。按时间保留（60 天）由 worker cron 执行，二者并存。
	slSysCfg := admin.NewPgSysConfigRepository(c.PgPool)
	retentionPolicy := stationlog.NewRetentionPolicy(
		func(ctx context.Context, category, key string) (string, bool) {
			row, err := slSysCfg.GetByKey(ctx, category, key)
			if err != nil || row == nil {
				return "", false
			}
			return row.Value, true
		},
		logger,
	)
	svc.SetRetentionPolicy(retentionPolicy)
	if c.SysConfigSvc != nil {
		c.SysConfigSvc.RegisterSavedHook(func(_ context.Context, category string) {
			if category == stationlog.RetentionCategory {
				retentionPolicy.InvalidateCache()
			}
		})
	}

	// 订阅 SubjectLogFileReceived 事件；运行日志入库，故障日志上传由文件传输任务链路维护。
	if c.EventBus != nil {
		if _, err := c.EventBus.Subscribe(event.SubjectLogFileReceived, func(ctx context.Context, evt event.Event) error {
			return svc.HandleLogFileReceived(ctx, evt)
		}); err != nil {
			logger.Warn("subscribe log.file.received", zap.Error(err))
		}
	}

	c.miscDeps.stationlogHandler = stationlog.NewHandler(svc, logger)

	// T-0158: 把 stationlog.Service 作为 AbnormalRebootRecorder 注入回 DeviceService，
	// 让"1 BOOT + HaltReason 非空"识别即落库。
	if c.DeviceService != nil {
		c.DeviceService.SetAbnormalRebootRecorder(svc)
	}

	logger.Info("stationlog module initialized")
	return nil
}

// initEventLogModule 初始化事件日志模块（设备活动审计流水）。
// 1 BOOT 无 HaltReason 时由 device.RecordBootFromInform 调用此 service 写入 event_logs。
func initEventLogModule(c *Container) error {
	logger := c.Logger.Named("eventlog")

	repo := eventlog.NewPgRepository(c.PgPool)
	svc := eventlog.NewService(repo, logger)
	// #63 租户隔离：注入设备组归属读取器，按记录归属设备校验详情读取。
	svc.SetGroupReader(device.NewPgDeviceGroupReader(c.PgPool))

	c.miscDeps.eventlogHandler = eventlog.NewHandler(svc, logger)

	if c.DeviceService != nil {
		c.DeviceService.SetBootEventRecorder(svc)
	}

	logger.Info("eventlog module initialized")
	return nil
}

// initRebootRecordModule 初始化统一重启记录模块（只读 UNION event_logs + station_fault_logs）。
// 仅供「启动记录」页面单列表展示，不写入；写入仍由 device.RecordBootFromInform 分流到两表。
func initRebootRecordModule(c *Container) error {
	logger := c.Logger.Named("rebootrecord")

	repo := rebootrecord.NewPgRepository(c.PgPool)
	svc := rebootrecord.NewService(repo, logger)
	c.miscDeps.rebootrecordHandler = rebootrecord.NewHandler(svc, logger)

	logger.Info("rebootrecord module initialized")
	return nil
}

// initDashboardModule 初始化 F06 仪表盘模块。
func initDashboardModule(c *Container) error {
	logger := c.Logger.Named("dashboard")

	// KPI/时序库物理分离：alarms_history / alarm_efficiency_metrics matview 在时序库（TsPool），新增 tsPool 入参。
	// issue #213 Phase1：注入 PM 的 indicator 仓库（perf_indicators_{enb,gsm,gnb}），
	// 供 GetKPIDefinitions 按别名表 K 编号反查 cnName / unit。dashboard 依赖 pm，pmHandlerDeps 此时已就绪。
	var dashIndicatorRepo indicator.IndicatorRepository
	if c.pmHandlerDeps != nil {
		dashIndicatorRepo = c.pmHandlerDeps.pmIndicatorRepo
	}
	dashboardCfg := c.Cfg.Dashboard.Defaults()
	dashboardMetrics := dashboard.NewMetrics(c.MetricsReg)
	networkRollups := dashboard.NewNetworkRollupRepository(c.TsPool, dashboardCfg.StatementTimeout)
	queryGuard := dashboard.NewKPIQueryGuard(dashboard.KPIQueryGuardConfig{
		QueryTimeout:  dashboardCfg.QueryTimeout,
		MaxConcurrent: dashboardCfg.MaxConcurrent,
		QueueTimeout:  dashboardCfg.QueueTimeout,
		FreshTTL:      dashboardCfg.FreshCacheTTL,
		StaleTTL:      dashboardCfg.StaleTTL,
	}, dashboardMetrics)
	dashboardService := dashboard.NewService(c.DeviceService, c.AlarmPgStore, c.PMKPIRepo, c.PgPool, c.TsPool, c.GroupRepo, dashIndicatorRepo, networkRollups, logger)
	// 首页 KPI 与 Counter 都只读内置全网任务已发布的 network 预聚合结果。
	// Counter 不得回退通用 Aggregator 做全网现场汇总，否则 hourly 兼容视图会在大数据量下超时。
	dashboardService.SetCounterSeriesReader(networkRollups)
	dashboardService.SetKPIQueryGuard(queryGuard)
	dashboardService.SetMetrics(dashboardMetrics)
	if c.pmHandlerDeps != nil && c.pmHandlerDeps.pmProgressService != nil {
		dashboardService.SetNetworkProgressReader(c.pmHandlerDeps.pmProgressService)
	}
	// issue #213 S1：存全局布局接口在 handler 层再校验管理员身份，注入 RoleRepo 作 Casbin 权限检查器
	// （super_admin 旁路 + 端点级权限点）。RoleRepo 实现 admin.PermissionChecker；为 nil 时只认 super_admin。
	var dashPermChecker admin.PermissionChecker
	if c.RoleRepo != nil {
		dashPermChecker = c.RoleRepo
	}
	dashboardHandler := dashboard.NewHandler(dashboardService, dashPermChecker)

	c.miscDeps.dashboardHandler = dashboardHandler

	logger.Info("dashboard module initialized")
	return nil
}

// initNorthboundModule 初始化 F08 北向/OSS 接口模块。
func initNorthboundModule(c *Container) error {
	logger := c.Logger.Named("northbound")

	pushEngine := push.NewEngine(c.Cfg.Northbound.PushTargets, logger)

	// Outbox 投递模式：构造 PG outbox repo 并注入 Engine（必须在 Subscribe 之前——
	// Subscribe 按 outboxRepo 是否存在选择 EnqueueEvent / 直发 handleEvent）。
	outboxRepo := push.NewPgOutboxRepository(c.PgPool, logger)
	pushEngine.SetOutboxRepo(outboxRepo)

	if err := pushEngine.Subscribe(c.EventBus); err != nil {
		logger.Warn("subscribe push engine", zap.Error(err))
	}
	c.GS.Register("push-engine", 1, func(ctx context.Context) error { return pushEngine.Close() })

	// Outbox 后台投递 worker：轮询 northbound_outbox 待投递条目，按 target 重试，
	// 耗尽重试落 dead 状态，供死信队列端点（GET/POST /push/deadletter*）消费。
	outboxWorker := push.NewOutboxWorker(pushEngine, outboxRepo, logger)
	outboxWorker.Start()
	c.GS.Register("northbound-outbox-worker", 1, func(ctx context.Context) error { return outboxWorker.Close() })

	syncService := nbsync.NewService(c.DeviceRepo, c.AlarmPgStore, c.PMCounterRepo, c.PMKPIRepo, c.ParamRepo, logger)
	nbService := northbound.NewNorthboundService(c.AlarmPgStore, c.PMCounterRepo, c.PMKPIRepo, c.ParamRepo, pushEngine, syncService, logger)
	nbRouter := northbound.NewRouter(nbService)
	// 死信队列端点（GET /push/deadletter、POST /push/deadletter/:id/replay）
	// 依赖 outbox repo；不注入则恒 503 "outbox not configured"。
	nbRouter.SetOutboxRepo(outboxRepo)

	// 主备 OSS 服务器配置 + 切换（system/config 北向设置页消费）。
	// 配置面与数据面分离：nbService 管数据导出 / 推送 / 同步；ServerService 管
	// active 组配置 (primary <-> standby)。
	// T-0099: ServerService 注入 EventBus，SetActive/Update 后发
	// SubjectNorthboundServerChanged → push engine 订阅 reload 关闭"切换即生效"环。
	nbServerRepo := northbound.NewPgServerRepository(c.PgPool)
	nbServerSvc := northbound.NewServerService(nbServerRepo, c.EventBus, logger)
	nbRouter.SetServerService(nbServerSvc)

	// T-0099: push engine 接入 active server provider + 启动期 refresh 兜底。
	// 失败仅 warn（dev 环境 northbound_servers 表可能空），不阻塞启动。
	pushEngine.SetActiveServerProvider(nbServerSvc)
	if err := pushEngine.RefreshActiveTarget(context.Background()); err != nil {
		logger.Warn("initial active target refresh failed", zap.Error(err))
	}

	// 多租户数据隔离 + per-endpoint 限流（审计缺口收口）。
	// PermService / DeviceService 由 admin / device 模块先行装配（见 ModuleGraph 依赖）。
	if c.PermService != nil && c.DeviceService != nil {
		scoper := northbound.NewScoper(
			c.PermService,
			c.DeviceService,
			northboundSNResolver{svc: c.DeviceService},
		)
		nbRouter.SetScoper(scoper)
		logger.Info("northbound multi-tenant scoper wired")
	} else {
		logger.Warn("northbound scoper not wired (perm/device service missing); exports run unscoped")
	}
	// per-endpoint 固定窗口限流（每分钟阈值，缺省 300）。
	nbRouter.SetRateLimiter(ratelimit.NewFixedWindowLimiter(
		c.Redis, "nb:endpoint", int64(c.Cfg.Northbound.EndpointRateLimit()), time.Minute))

	c.miscDeps.nbRouter = nbRouter

	logger.Info("northbound/OSS module initialized")
	return nil
}

// northboundSNResolver 把 device.DeviceService.GetBySerialNumber 适配成
// northbound.DeviceSNResolver（SN → deviceID）。
type northboundSNResolver struct {
	svc *device.DeviceService
}

func (r northboundSNResolver) ResolveDeviceID(ctx context.Context, sn string) (uuid.UUID, bool, error) {
	dev, err := r.svc.GetBySerialNumber(ctx, sn)
	if err != nil {
		return uuid.Nil, false, err
	}
	if dev == nil {
		return uuid.Nil, false, nil
	}
	return dev.ID, true, nil
}

// initInteropModule 初始化 F10 互操作测试模块。
// T-0098 P5-01：dataModelReg 已删除，直接注入 ParamRegistry / ProductRegistry。
func initInteropModule(c *Container) error {
	logger := c.Logger.Named("interop")

	testRunner := interop.NewConformanceTestRunner(c.DeviceRepo, c.ParamRepo, c.ParamRegistry, c.ProductRegistry, c.TaskSvc, logger)
	testRunner.RegisterCases(cases.ProtocolCases())
	testRunner.RegisterCases(cases.DataModelCases())
	testRunner.RegisterCases(cases.RPCCases())
	testRunner.RegisterCases(cases.InformCases())
	testRunner.RegisterCases(cases.FaultInjectCases())
	dmValidator := interop.NewDataModelValidator(c.ParamRegistry, c.ProductRegistry, c.ParamRepo, c.DeviceRepo, logger)
	// #63 租户隔离：注入设备组归属读取器，run / validate 执行（可能含破坏性 RPC）前按设备校验。
	interopGroupReader := device.NewPgDeviceGroupReader(c.PgPool)
	testRunner.SetGroupReader(interopGroupReader)
	dmValidator.SetGroupReader(interopGroupReader)
	interopHandler := interop.NewHandler(testRunner, dmValidator, logger)

	c.miscDeps.interopHandler = interopHandler

	logger.Info("interop testing module initialized")
	return nil
}

// initMiscModules 初始化其余小型模块。
func initMiscModules(c *Container) error {
	logger := c.Logger
	if c.StorageProtection == nil {
		prometheusURL := os.Getenv("OMCGO_SYSTEM_INFO_PROMETHEUS_URL")
		if prometheusURL == "" {
			prometheusURL = "http://prometheus:9090"
		}
		storageCollector := components.NewPrometheusStorageCollector(prometheusURL, 2*time.Second, time.Minute, nil)
		storageProtection := storageprotection.NewService(
			storageprotection.NewPgRepository(c.PgPool),
			storageprotection.NewCollectorUsageProvider(storageCollector),
			storageprotection.NewMetrics(c.MetricsReg),
			logger,
		)
		storageProtection.SetLogAdmissionController(c.LogGate)
		storageProtection.Start(context.Background(), 30*time.Second)
		c.StorageCollector = storageCollector
		c.StorageProtection = storageProtection
		c.StorageProtectionHandler = storageprotection.NewHandler(storageProtection)
		if c.GS != nil {
			c.GS.Register("storage-protection", 1, func(context.Context) error {
				storageProtection.Stop()
				return nil
			})
		}
	}

	// Syslog module
	syslogRepo := syslog.NewPgSyslogRepository(c.PgPool)
	c.miscDeps.syslogHandler = syslog.NewHandler(syslogRepo, logger)

	// Config sync —— PullConfig 优先走 SyncSvc 拿批次拆分 + Fault 自愈；AutoSync 未启用时 SyncSvc=nil，handler 自动降级到单 task 路径。
	var gpvBatcher config.GPVBatcher
	if c.SyncSvc != nil {
		gpvBatcher = c.SyncSvc
	}
	// 设备存在性预检：push/pull 入队前按 SN 查 device 表，不存在返回 404，避免孤儿任务（issue #126 第3项）。
	var syncDeviceChk config.DeviceExistenceChecker
	if c.DeviceRepo != nil {
		syncDeviceChk = &syncDeviceChecker{repo: c.DeviceRepo}
	}
	c.miscDeps.syncHandler = config.NewSyncHandler(c.TaskSvc, gpvBatcher, syncDeviceChk, logger)
	if c.miscDeps.paramSyncStarter != nil {
		c.miscDeps.syncHandler.WithDurablePullSubmitter(c.miscDeps.paramSyncStarter)
	}

	// File Manager module
	fileRepo := filemanager.NewPgFileRepository(c.PgPool)
	fileService := filemanager.NewFileService(fileRepo, c.MinIO, c.Cfg.MinIO.Buckets.ConfigBackup, c.TaskSvc, logger)
	fileService.SetStorageAdmission(c.StorageProtection)
	c.miscDeps.fileHandler = filemanager.NewHandler(fileService, logger)
	logger.Info("file manager module initialized")

	// SSE / Events module (must init before MML so hub is available)
	eventStore := events.NewRedisMessageStore(c.Redis, time.Hour)
	messageHub := events.NewMessageHub(eventStore, logger)
	sseHandler := events.NewSSEHandler(messageHub, c.JWTService, logger)
	c.miscDeps.sseHandler = sseHandler
	c.miscDeps.messageHub = messageHub
	logger.Info("SSE events module initialized")

	// Notification module
	notifRepo := notification.NewPgRepository(c.PgPool)
	notifService := notification.NewService(notifRepo, messageHub, logger)
	// T-0157 stale sync: 注入 task lookup 让 SyncStaleByUser 能反查 task 状态修正卡死消息。
	// 用 task.PgTaskRepository (复用同一 PgPool；适配器避免 notification → task 直接依赖)。
	notifService.SetStaleTaskLookup(task.NewStaleNotificationLookup(task.NewPgTaskRepository(c.PgPool)))
	c.miscDeps.notificationHandler = notification.NewHandler(notifService, logger)

	// W2.A.4 / T-0043: Notification template + history submodules
	templateRepo := notification.NewPgTemplateRepository(c.PgPool)
	templateService := notification.NewTemplateService(templateRepo, logger)
	c.miscDeps.notifTemplateHandler = notification.NewTemplateHandler(templateService, logger)

	historyRepo := notification.NewPgHistoryRepository(c.PgPool)
	historyService := notification.NewHistoryService(historyRepo, logger)
	c.miscDeps.notifHistoryHandler = notification.NewHistoryHandler(historyService, logger)

	// T-0152: SMTP 邮件发送器 + Mailer（模板渲染→发送→历史）+ Alertmanager
	// 告警 webhook 入口。SMTP 默认 disabled，配好邮件服务器后由配置启用。
	emailSender := notification.NewEmailSender(notification.SMTPOptions{
		Enabled:  c.Cfg.Notification.SMTP.Enabled,
		Host:     c.Cfg.Notification.SMTP.Host,
		Port:     c.Cfg.Notification.SMTP.Port,
		Username: c.Cfg.Notification.SMTP.Username,
		Password: c.Cfg.Notification.SMTP.Password,
		From:     c.Cfg.Notification.SMTP.From,
		StartTLS: c.Cfg.Notification.SMTP.StartTLS,
		Timeout:  c.Cfg.Notification.SMTP.Timeout,
	}, logger)
	mailer := notification.NewMailer(templateService, historyService, emailSender, logger)
	c.miscDeps.alertWebhookHandler = notification.NewAlertWebhookHandler(mailer, notification.AlertWebhookOptions{
		Token:      c.Cfg.Notification.AlertWebhook.Token,
		Recipients: c.Cfg.Notification.AlertWebhook.Recipients,
	}, logger)

	logger.Info("notification module initialized",
		zap.Bool("smtp_enabled", c.Cfg.Notification.SMTP.Enabled))

	// MML Console module
	mmlCmdRepo := mml.NewPgCommandRepository(c.PgPool)
	mmlScriptRepo := mml.NewPgScriptRepository(c.PgPool)
	mmlTaskRepo := mml.NewPgTaskRepository(c.PgPool).WithLogger(logger)
	mmlCustomCmdRepo := mml.NewPgCustomCommandRepository(c.PgPool)
	mmlAuditRepo := mml.NewPgAuditRepository(c.PgPool)
	mmlCmdParamRepo := mml.NewPgCommandParamRepository(c.PgPool)
	mmlService := mml.NewService(mmlCmdRepo, mmlScriptRepo, mmlTaskRepo, mmlCustomCmdRepo, messageHub, logger)
	// MML TXT import is a two-phase flow: validation snapshots are held in
	// Redis for 15 minutes, while command/device authority is loaded from PG.
	// The service is injected before the handler is registered below so every
	// route observes the same repository and session-store instances.
	importSessions := mml.NewRedisImportSessionStore(c.Redis, 15*time.Minute)
	importValidator := mml.NewScriptImportValidator(mml.NewPgScriptValidationRepository(c.PgPool))
	mmlService.SetScriptImportService(mml.NewScriptImportService(mmlScriptRepo, importValidator, importSessions, logger))
	mmlService.SetScriptExecutionValidator(importValidator)
	mmlService.SetAuditRepo(mmlAuditRepo)
	mmlService.SetCmdParamRepo(mmlCmdParamRepo)
	// issue #115 调整3（A1）：自定义命令 PATH 关联表仓库。
	mmlService.SetCustomCommandPathRepo(mml.NewPgCustomCommandPathRepository(c.PgPool))
	// CSV 导出「参数名称」列：优先使用命令 sub-field 的请求语言标签；
	// 中文缺失时回退 standard_params.description，英文缺失时回退 standardPath，避免中文泄漏到英文导出。
	mmlService.SetPathNameResolver(func(ctx context.Context, paths []string) (map[string]string, error) {
		if len(paths) == 0 {
			return nil, nil
		}
		locale := string(appcontext.GetLocale(ctx))
		rows, err := c.PgPool.Query(ctx,
			`SELECT sp.standard_path,
				COALESCE(
					NULLIF((SELECT csf.label_i18n->>$1
						FROM mml_command_sub_fields csf
						WHERE csf.standard_path_id = sp.id
						ORDER BY csf.sort_order, csf.id
						LIMIT 1), ''),
					NULLIF((SELECT csf.label_i18n->>'en-US'
						FROM mml_command_sub_fields csf
						WHERE csf.standard_path_id = sp.id
						ORDER BY csf.sort_order, csf.id
						LIMIT 1), ''),
					CASE WHEN $1 = 'zh-CN' THEN COALESCE(sp.description, '') ELSE '' END
				) AS display_name
			 FROM standard_params sp
			 WHERE sp.standard_path = ANY($2)`, locale, paths)
		if err != nil {
			return nil, fmt.Errorf("query standard_params descriptions: %w", err)
		}
		defer rows.Close()
		m := make(map[string]string, len(paths))
		for rows.Next() {
			var p, d string
			if err := rows.Scan(&p, &d); err != nil {
				return nil, fmt.Errorf("scan standard parameter display name: %w", err)
			}
			if d != "" {
				m[p] = d
			}
		}
		return m, rows.Err()
	})
	// 结果 CSV 导出：上传用内部 client（c.MinIO，连 docker 内网 minio:9000），
	// 下载 URL 用 PresignClient（public_endpoint，浏览器可达）；落 reports bucket 的
	// mml-results/ 独立目录。任一缺失 → 导出端点返回 503。
	//
	// issue #548 切片 4：同时注入 PresignBridge 给 Exporter，sys_configs 写入
	// storage.minio_public_endpoint 后下一次 presign 立即用新 endpoint，无需重启容器。
	// 原 exportSignClient（NewPresignClient）作启动期兜底。
	if c.MinIO != nil {
		exportSignClient := c.MinIO
		if pc, perr := minioinfra.NewPresignClient(c.Cfg.MinIO); perr != nil {
			logger.Warn("create MinIO presign client for mml export failed; using internal endpoint", zap.Error(perr))
		} else {
			exportSignClient = pc
		}
		mmlExporter := mml.NewExporter(c.MinIO, exportSignClient, c.Cfg.MinIO.Buckets.Reports, logger)
		if c.PresignBridge != nil {
			mmlExporter.SetSignProvider(c.PresignBridge)
		}
		mmlService.SetExporter(mmlExporter)
	}
	// T-0090-c：注入 admin RoleRepo 作 RBAC group 派生器，让 ListCustomCommands
	// 走 group-share 路径（同组管理员可见对方 private 命令）。c.RoleRepo 由 admin
	// 模块初始化时（router.go misc Depends admin）填入，此处必非 nil。
	if c.RoleRepo != nil {
		mmlService.SetRoleQuerier(c.RoleRepo)
	}
	// R-8.4 + R-9.3（方案 §6.4 / §6.6）：注入 DeviceLookup + PathTranslator 适配器，
	// 启用 CreateAndFanoutTask 入口的 product_class 一致性校验 + standardPath 翻译。
	if c.DeviceService != nil {
		mmlService.SetDeviceLookup(NewMMLDeviceLookup(c.DeviceService))
	}
	// T-0168: 提取 FanoutMetrics 同实例，供 PathTranslator 适配器 + Fanouter 共享，
	// 让 mml_path_translation_orphan_total 与 _miss_total 注册到同一 Registry，
	// 避免后续 fanouter.SetMetrics 重复 NewFanoutMetrics 导致 Prometheus 重复注册 panic。
	mmlFanoutMetrics := mml.NewFanoutMetrics(c.MetricsReg)
	if c.ProductRegistry != nil && c.ParamRegistry != nil {
		mmlService.SetPathTranslator(NewMMLPathTranslator(c.ProductRegistry, c.ParamRegistry, mmlFanoutMetrics, logger))
		// 同源装配 software 端：SPV 触发的日志采集（如 FAULT_LOG_COLLECT）下发前
		// 用同款 productClass → Registry → Translator 链路把 standardPath 翻译成
		// privatePath。CLAUDE.md §5.3 要求；未注入时 ExecuteOneSetParamCollect
		// 退化为 passthrough。
		if c.miscDeps.softwareService != nil {
			c.miscDeps.softwareService.SetParamPathTranslator(
				NewSoftwareParamPathTranslator(c.ProductRegistry, c.ParamRegistry, logger),
			)
		}
	}
	c.miscDeps.mmlHandler = mml.NewHandler(mmlService, logger)
	c.miscDeps.mmlService = mmlService

	// T-0123-P0：catalog 管理 admin 端点（mml_admin api_group）。
	// catalog_protected 守护 / sentinel error → HTTP 403/404/409 翻译。
	// mml_params 表已下线，AdminParamRepository / XMLImportService 已随之移除。
	mmlSubFieldRepo := mml.NewPgSubFieldRepository(c.PgPool).WithLogger(logger)
	mmlAdminGroupRepo := mml.NewPgAdminGroupRepository(c.PgPool)
	mmlAdminCmdRepo := mml.NewPgAdminCommandRepository(c.PgPool)
	mmlStandardParamRepo := mml.NewPgStandardParamRepository(c.PgPool)
	mmlAdminService := mml.NewAdminService(
		mmlAdminGroupRepo, mmlAdminCmdRepo, mmlSubFieldRepo,
		nil, // AuditWriter — TODO: wire internal/admin/auditlog when admin module exposes interface
		logger,
	)
	// T-Mml-Admin: 注入 admin 列表 / standard_params 路径下拉所需读 Repo。
	mmlAdminService.SetCommandReader(mmlCmdRepo)
	mmlAdminService.SetStandardParamRepo(mmlStandardParamRepo)
	c.miscDeps.mmlAdminHandler = mml.NewAdminHandler(mmlAdminService, mmlCmdRepo, logger)

	// T-0123-P1：Console 5 端点（group-tree / sub-fields / render / parse / execute-statements）。
	// 复用 T-0123-P0 的 SubFieldRepo + 既有 CommandRepo；新增 GroupTreeRepo（带 ltree JOIN 子树）。
	// BuildTree 始终走 v2 视图（chapter:* 顶层 + DB 真实嵌套，spec v2.3 §R-1）。
	mmlGroupTreeRepo := mml.NewPgGroupTreeRepository(c.PgPool)
	mmlConsoleSvc := mml.NewConsoleService(mmlGroupTreeRepo, mmlSubFieldRepo, mmlCmdRepo, logger)
	// Task #4: 装配扁平命令树仓储（GET /mml/group-tree?format=flat）
	// 依赖同一个 PgPool，与现有 GroupTreeRepository 只读同表不冲突。
	mmlConsoleSvc.SetFlatTreeRepo(mml.NewPgFlatGroupTreeRepository(c.PgPool))
	// Bundle C: 装配命令搜索仓储（GET /mml/commands/search）。
	mmlConsoleSvc.SetSearchRepo(mml.NewPgSearchRepository(c.PgPool))
	// 修复 2026-05-22：装配 cmdParamRepo 让 cmd.Params 在 GetByID 后被 enrichment
	// 填上（旧版漏装 → execute-statements-structured 全部 path 误报 R-9.2 unknown）。
	mmlConsoleSvc.SetCmdParamRepo(mmlCmdParamRepo)
	// T-0170 / T-0176-PR-C: 注入 device → paramModel 反查闭包，让 GET
	// /mml/commands/:id/sub-fields?device_id= 端点能按 paramModel 过滤 sub_field。
	//
	// 重构（PR-C）：抛弃原"devices LEFT JOIN products"单条 SQL，改走 cache-aware
	// 分层链路：
	//   1) deviceKey 解析为设备 (UUID 首先尝试 GetByID 不命中再退 SN)
	//   2) DeviceService.GetBySerialNumber → 享受 DeviceCache L1 命中
	//   3) 路由到 product（product_id 优先，product_class 兜底）：
	//      3a) device.ProductID 非空 → ProductRegistry.GetProductByID → product
	//          （设备已绑定的产品为权威来源，避免 product_class 正则未覆盖时误判孤儿）
	//      3b) product_id 为空 / 指向的产品不存在 → device.ProductClass →
	//          ProductRegistry.MatchProductClass → product（享受 T-0173 productClass 缓存）
	//   4) product.ParamModelID
	//
	// 旧 SQL 的 device.param_model_id 字段已不在 Go model 上（T-0098 后字段下线，
	// 仅 DB 列残留）；本路径不读该列，统一经 product 装配件取 ParamModelID，
	// 单一真值源。
	//
	// silent skip 语义（返回 (nil, nil)）：
	//   - 设备不存在
	//   - product_id 与 productClass 均无法路由到 product（孤儿）
	//   - product 装配件未挂 paramModel
	// 上层（ConsoleService）见 nil 即返回空的设备过滤结果，避免设备未解析时放行
	// 未过滤的全集 sub_field。
	mmlConsoleSvc.SetParamModelByDeviceResolver(func(ctx context.Context, deviceKey string) (*uuid.UUID, error) {
		dev, err := resolveDeviceByKey(ctx, c, deviceKey)
		if err != nil {
			return nil, fmt.Errorf("resolve device %q: %w", deviceKey, err)
		}
		if dev == nil {
			return nil, nil
		}
		// 3a) 优先按设备已绑定的 product_id 直接取产品 → paramModel。
		if dev.ProductID != nil {
			p, err := c.ProductRegistry.GetProductByID(ctx, *dev.ProductID)
			if err != nil {
				return nil, fmt.Errorf("get product %s for device %q: %w", dev.ProductID, deviceKey, err)
			}
			if p != nil {
				return p.ParamModelID, nil
			}
			// product_id 指向的产品已不存在（脏数据）→ 落到 3b productClass 兜底。
		}
		// 3b) product_id 为空 / 失效 → productClass 正则路由兜底。
		if dev.ProductClass == "" {
			return nil, nil
		}
		mr, err := c.ProductRegistry.MatchProductClass(ctx, dev.ProductClass)
		if errors.Is(err, product.ErrOrphan) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("match product_class %q for device %q: %w", dev.ProductClass, deviceKey, err)
		}
		if mr == nil || mr.Product == nil {
			return nil, nil
		}
		return mr.Product.ParamModelID, nil
	})
	mmlConsoleSvc.SetDeviceSupportedPathsResolver(func(ctx context.Context, deviceKey string) (*mml.SupportedSet, error) {
		dev, err := resolveDeviceByKey(ctx, c, deviceKey)
		if err != nil {
			return nil, fmt.Errorf("resolve device %q: %w", deviceKey, err)
		}
		if dev == nil {
			return &mml.SupportedSet{ProductResolved: false, Paths: map[string]struct{}{}}, nil
		}

		var productID uuid.UUID
		var productTech string
		var paramModelID *uuid.UUID
		if dev.ProductID != nil {
			p, err := c.ProductRegistry.GetProductByID(ctx, *dev.ProductID)
			if err != nil {
				return nil, fmt.Errorf("get product %s for device %q: %w", dev.ProductID, deviceKey, err)
			}
			if p != nil {
				productID = p.ID
				productTech = p.Tech
				paramModelID = p.ParamModelID
			}
		}
		if productID == uuid.Nil {
			if dev.ProductClass == "" {
				return &mml.SupportedSet{ProductResolved: false, Paths: map[string]struct{}{}}, nil
			}
			mr, err := c.ProductRegistry.MatchProductClass(ctx, dev.ProductClass)
			if errors.Is(err, product.ErrOrphan) {
				return &mml.SupportedSet{
					ProductClass:    dev.ProductClass,
					ProductResolved: false,
					Paths:           map[string]struct{}{},
				}, nil
			}
			if errors.Is(err, product.ErrInactiveParamModel) {
				if mr != nil && mr.Product != nil {
					productID = mr.Product.ID
					productTech = mr.Product.Tech
					paramModelID = mr.Product.ParamModelID
				}
				var pidPtr *uuid.UUID
				if productID != uuid.Nil {
					pid := productID
					pidPtr = &pid
				}
				return &mml.SupportedSet{
					ProductClass:    dev.ProductClass,
					ProductID:       pidPtr,
					ProductTech:     productTech,
					ParamModelID:    paramModelID,
					ProductResolved: true,
					Paths:           map[string]struct{}{},
				}, nil
			}
			if err != nil {
				return nil, fmt.Errorf("match product_class %q for device %q: %w", dev.ProductClass, deviceKey, err)
			}
			if mr == nil || mr.Product == nil {
				return &mml.SupportedSet{ProductResolved: false, Paths: map[string]struct{}{}}, nil
			}
			productID = mr.Product.ID
			productTech = mr.Product.Tech
			paramModelID = mr.Product.ParamModelID
		}
		if paramModelID == nil {
			pid := productID
			return &mml.SupportedSet{
				ProductClass:    dev.ProductClass,
				ProductID:       &pid,
				ProductTech:     productTech,
				ProductResolved: false,
				Paths:           map[string]struct{}{},
			}, nil
		}

		set, err := c.ParamRegistry.GetByProduct(ctx, productID, dev.FirmwareVersion)
		if err != nil {
			if errors.Is(err, parammodel.ErrNoMapping) ||
				errors.Is(err, parammodel.ErrNoParamModel) ||
				errors.Is(err, parammodel.ErrInactiveParamModel) {
				pid := productID
				pmID := *paramModelID
				return &mml.SupportedSet{
					ProductClass:    dev.ProductClass,
					ProductID:       &pid,
					ProductTech:     productTech,
					ParamModelID:    &pmID,
					ProductResolved: true,
					Paths:           map[string]struct{}{},
				}, nil
			}
			return nil, fmt.Errorf("get product %s paths for device %q @ firmware %q: %w", productID, deviceKey, dev.FirmwareVersion, err)
		}

		paths := make(map[string]struct{}, len(set.Mappings))
		for _, m := range set.Mappings {
			if m.StandardPath != "" && m.IsActive && m.IsSupported {
				paths[m.StandardPath] = struct{}{}
			}
		}
		rows, err := c.PgPool.Query(ctx, `
SELECT DISTINCT regexp_replace(parameter_path::text, '\.[0-9]+\.', '.{i}.', 'g') AS standard_path
  FROM device_parameters
 WHERE device_id = $1`, dev.ID)
		if err != nil {
			return nil, fmt.Errorf("list observed parameter paths for device %q: %w", deviceKey, err)
		}
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan observed parameter path for device %q: %w", deviceKey, err)
			}
			if path != "" {
				paths[path] = struct{}{}
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate observed parameter paths for device %q: %w", deviceKey, err)
		}
		rows.Close()
		pid := productID
		pmID := set.ParamModelID
		return &mml.SupportedSet{
			ProductClass:    dev.ProductClass,
			ProductID:       &pid,
			ProductTech:     productTech,
			ParamModelID:    &pmID,
			ProductResolved: true,
			Paths:           paths,
		}, nil
	})

	// 产品不支持 path 自学习表（migration 000029）：MML path 不支持类故障时按 product_id 录入，
	// 前端「选择命令 / 配置参数」据此按读/写过滤。deviceSN → product_id 复用 resolveDeviceByKey + ProductRegistry。
	unsupportedPathRepo := mml.NewPgProductUnsupportedPathRepository(c.PgPool)
	productIDByDevice := func(ctx context.Context, deviceKey string) (*uuid.UUID, error) {
		dev, err := resolveDeviceByKey(ctx, c, deviceKey)
		if err != nil {
			return nil, fmt.Errorf("resolve device %q: %w", deviceKey, err)
		}
		if dev == nil {
			return nil, nil
		}
		// 优先用设备已绑定的 product_id；为空再按 product_class 正则路由兜底。
		if dev.ProductID != nil {
			pid := *dev.ProductID
			return &pid, nil
		}
		if dev.ProductClass == "" {
			return nil, nil
		}
		mr, err := c.ProductRegistry.MatchProductClass(ctx, dev.ProductClass)
		if errors.Is(err, product.ErrOrphan) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("match product_class %q for device %q: %w", dev.ProductClass, deviceKey, err)
		}
		if mr == nil || mr.Product == nil {
			return nil, nil
		}
		pid := mr.Product.ID
		return &pid, nil
	}
	// 控制台公有/私有自定义命令也必须使用当前产品参数模型的支持集合；否则
	// 模板中的跨产品 path 会在树中显示，直到选中后才被过滤。
	if c.ProductRegistry != nil && c.ParamRegistry != nil {
		mmlService.SetCustomCommandSupportedPathsResolver(func(ctx context.Context, productID uuid.UUID) (map[string]struct{}, error) {
			p, err := c.ProductRegistry.GetProductByID(ctx, productID)
			if err != nil {
				return nil, fmt.Errorf("get product %s: %w", productID, err)
			}
			if p == nil || p.ParamModelID == nil {
				return map[string]struct{}{}, nil
			}
			set, err := c.ParamRegistry.GetByParamModel(ctx, *p.ParamModelID)
			if err != nil {
				if errors.Is(err, parammodel.ErrNoMapping) || errors.Is(err, parammodel.ErrInactiveParamModel) {
					return map[string]struct{}{}, nil
				}
				return nil, fmt.Errorf("get param_model %s: %w", p.ParamModelID, err)
			}
			paths := make(map[string]struct{}, len(set.Mappings))
			for _, mapping := range set.Mappings {
				if mapping.StandardPath != "" && mapping.IsActive && mapping.IsSupported {
					paths[mapping.StandardPath] = struct{}{}
				}
			}
			return paths, nil
		})
	}
	mmlConsoleSvc.SetUnsupportedPathsProvider(unsupportedPathRepo, productIDByDevice)
	if c.SyncSvc != nil {
		c.SyncSvc.SetUnsupportedPathRepo(unsupportedPathRepo)
	}

	// T-0172: 注入 productClass → SupportedSet 反查（catalog 按产品过滤命令树）。
	// 方案 X：supported set 仅取 default param_mappings（不掺杂 discovered）；
	// 与 per-device A1 (T-0170) 区分开。详见 mml/device_supported_paths.go 注释。
	mmlConsoleSvc.SetSupportedPathsRepository(mml.SupportedPathsResolverFunc(
		func(ctx context.Context, productClass string) (*mml.SupportedSet, error) {
			mr, err := c.ProductRegistry.MatchProductClass(ctx, productClass)
			if errors.Is(err, product.ErrInactiveParamModel) {
				set := &mml.SupportedSet{
					ProductClass:    productClass,
					ProductResolved: true,
					Paths:           map[string]struct{}{},
				}
				if mr != nil && mr.Product != nil {
					productID := mr.Product.ID
					set.ProductID = &productID
					set.ProductTech = mr.Product.Tech
					if mr.Product.ParamModelID != nil {
						paramModelID := *mr.Product.ParamModelID
						set.ParamModelID = &paramModelID
					}
				}
				return set, nil
			}
			if errors.Is(err, product.ErrOrphan) {
				// 孤儿产品：保留 productResolved=false，由控制台返回空过滤结果。
				return &mml.SupportedSet{
					ProductClass:    productClass,
					ProductResolved: false,
					Paths:           map[string]struct{}{},
				}, nil
			}
			if err != nil {
				return nil, fmt.Errorf("match product_class %q: %w", productClass, err)
			}
			if mr == nil || mr.Product == nil || mr.Product.ParamModelID == nil {
				// 匹配到 product 但无 paramModel — 类同孤儿处理（无法构造 supported set）
				return &mml.SupportedSet{
					ProductClass:    productClass,
					ProductResolved: false,
					Paths:           map[string]struct{}{},
				}, nil
			}
			set, err := c.ParamRegistry.GetByParamModel(ctx, *mr.Product.ParamModelID)
			if err != nil {
				if errors.Is(err, parammodel.ErrInactiveParamModel) {
					productID := mr.Product.ID
					paramModelID := *mr.Product.ParamModelID
					return &mml.SupportedSet{
						ProductClass:    productClass,
						ProductID:       &productID,
						ProductTech:     mr.Product.Tech,
						ParamModelID:    &paramModelID,
						ProductResolved: true,
						Paths:           map[string]struct{}{},
					}, nil
				}
				return nil, fmt.Errorf("get param_model %s mappings: %w", mr.Product.ParamModelID, err)
			}
			paths := make(map[string]struct{}, len(set.Mappings))
			for _, m := range set.Mappings {
				if m.StandardPath != "" && m.IsActive && m.IsSupported {
					paths[m.StandardPath] = struct{}{}
				}
			}
			productID := mr.Product.ID
			paramModelID := *mr.Product.ParamModelID
			return &mml.SupportedSet{
				ProductClass:    productClass,
				ProductID:       &productID,
				ProductTech:     mr.Product.Tech,
				ParamModelID:    &paramModelID,
				ProductResolved: true,
				Paths:           paths,
			}, nil
		}))
	// deviceKey 分支：按 paramModelID 从 ParamRegistry（Redis L1→L2→DB）取 supported paths。
	mmlConsoleSvc.SetParamModelPathsResolver(func(ctx context.Context, pmID uuid.UUID) (map[string]struct{}, error) {
		set, err := c.ParamRegistry.GetByParamModel(ctx, pmID)
		if err != nil {
			if errors.Is(err, parammodel.ErrNoMapping) || errors.Is(err, parammodel.ErrInactiveParamModel) {
				return map[string]struct{}{}, nil
			}
			return nil, fmt.Errorf("get param_model %s paths: %w", pmID, err)
		}
		paths := make(map[string]struct{}, len(set.Mappings))
		for _, m := range set.Mappings {
			if m.StandardPath != "" && m.IsActive && m.IsSupported {
				paths[m.StandardPath] = struct{}{}
			}
		}
		return paths, nil
	})
	// R-8.5: 独立 CompatibilityService（不耦合 ConsoleService 签名 / 测试）。
	// T-0177：走 ProductRegistry.MatchProductClass（全局正则路由）取代旧
	// 直查 devices LEFT JOIN products 的 raw SQL — 不再依赖 devices.product_id /
	// param_model_id denorm 列填充状态，享受 PR-B (T-0173) productClass L1+L2
	// 缓存；与 group_tree (PR-C) / supported_paths (T-0172) resolver 模式统一。
	mmlCmdPathRepo := mml.NewPgCommandPathRepository(c.PgPool)
	mmlCompatibility := mml.NewCompatibilityService(c.ProductRegistry, c.ParamRegistry, mmlCmdPathRepo, logger)
	c.miscDeps.mmlConsoleHandler = mml.NewConsoleHandler(mmlConsoleSvc, mmlService, mmlCompatibility, logger)

	// Stage 3（T-0123 v5）：装配 mml.Service 的 device_tasks 路径翻译 miss 聚合
	// 接口；适配器 mmlPathMissAdapter（本文件末尾）把 task 包的
	// PathTranslationMissStats 转换为 mml 包的视图类型，让 GET /mml/tasks/:id
	// 返回 path_translation_warning 字段供前端任务详情警告标签使用。
	if c.miscDeps.taskSvc != nil {
		mmlService.SetDeviceTaskPathMissAggregator(&mmlPathMissAdapter{svc: c.miscDeps.taskSvc})
		// 2026-05-23 修：装配 device_tasks → mml task results 适配器，让任务记录
		// 页"查看"modal 拿到真实设备级执行结果（原 mml_tasks.results JSONB 永远空）。
		mmlService.SetDeviceTaskResultLister(&mmlDeviceTaskResultAdapter{svc: c.miscDeps.taskSvc})
	}

	// Stage 2（T-0123 v5）：上行 RPC 响应订阅者 — ACS 进程发 EventBus 后，
	// 本进程订阅 command.get_parameters.response，把 privatePath 翻译为
	// standardPath 再写 device_parameters。
	if c.EventBus != nil && c.DeviceService != nil && c.ParamRepo != nil {
		rpcRespSub := device.NewRPCResponseSubscriber(
			c.EventBus,
			c.ProductRegistry,
			c.ParamRegistry,
			c.DeviceService,
			c.ParamRepo,
			device.NewInfoSyncer(c.DeviceInfoRepo, c.ParamRepo, device.NewPgDeviceRepository(c.PgPool), c.Carriers, logger, device.NewPgLocationObservationRepository(c.PgPool)),
			c.DeviceRepo, // migration 000146: 写 last_param_sync_failed_at + error
			c.miscDeps.taskSvc,
			c.Cfg.Provision.GPVResponse,
			logger,
		)
		if err := rpcRespSub.Start(); err != nil {
			logger.Warn("start rpc response subscriber failed; uplink path translation disabled",
				zap.Error(err))
		}
	}

	// Wire MML fan-out to device tasks. misc 模块在 router.go 声明 Depends=["task"]，
	// 保证此处 c.miscDeps.taskSvc 一定已就绪。
	//
	// Stage 1 (T-0123 v5)：fanout 阶段 standardPath → privatePath 翻译需要
	// ProductRegistry / ParamRegistry / DeviceService 三方协作；任一未注入
	// 则 fanouter 走 fallback（standardPath 直接下发）。生产部署必须三者齐全。
	if c.miscDeps.taskSvc != nil {
		fanouter := mml.NewFanouter(
			c.miscDeps.taskSvc,
			c.ProductRegistry,
			c.ParamRegistry,
			c.DeviceService,
			logger,
		)
		// Sprint B Q-V3-3：脚本多行严格序列 — 初次 fanout 仅入队 cmd_idx=0；
		// Sequencer 通过 completion callback 链式入队后续行。
		fanouter.SetSequentialMode(true)
		// Stage 1 整改方案 §3：累加 mml_path_translation_miss_total 供
		// `deployments/monitoring/alerts/omc-rules.yml MMLPathTranslationMissSustained` 告警。
		// T-0168: 复用上面已构造的 mmlFanoutMetrics 实例，避免重复注册。
		fanouter.SetMetrics(mmlFanoutMetrics)
		mmlService.SetFanouter(fanouter)
		startMMLScheduler(c, mmlService, mmlTaskRepo, logger)

		// Sequencer：与 ResultAggregator 并行挂到 MML completion 通路，
		// 在 device_task 进入终态后追加入队下一行命令。
		sequencer := mml.NewSequencer(mmlTaskRepo, c.miscDeps.taskSvc, fanouter, logger)

		// device_tasks 终态通过 NATS 跨进程事件投递到聚合器：
		// ACS 在 MarkTaskCompleted/Failed 后发布 task.completed/task.failed，
		// 本进程的 bridge 订阅后驱动 ResultAggregator 更新 mml_tasks 统计并推送 SSE。
		// T-0174: 注入 mmlSubFieldRepo 让 aggregator 能在收到 SetParameterValues 9005
		// 时自动 mark is_supported=false（auto-learn 不支持 path）。
		// T-0176-PR-E: auto-learn 真值源迁到 param_mappings 后必须先解析 paramModelID；
		// 这里复用与 ConsoleService.SetParamModelByDeviceResolver 相同的 devices LEFT
		// JOIN products SQL —— 两闭包是 cosmetic duplication，PR-C / PR-E 合主分支
		// 后可抽出 helper（独立清理任务，留 TODO 在此）。
		aggregator := mml.NewResultAggregator(mmlTaskRepo, mmlScriptRepo, mmlSubFieldRepo, messageHub, logger)
		aggregator.SetParamModelResolver(func(ctx context.Context, deviceKey string) (*uuid.UUID, error) {
			// deviceKey 可以是 SN 或 UUID。SQL 用 OR 兼容；ACS 上报的 dt.DeviceSN 是 SN，
			// admin 工具可能传 UUID — 同 SQL 通用。
			// TODO(T-0176 cleanup): 与 mmlConsoleSvc.SetParamModelByDeviceResolver 闭包
			// 抽出共享 helper（如 provider.resolveParamModelByDeviceKey）。
			const q = `
SELECT COALESCE(d.param_model_id, p.param_model_id) AS effective_param_model_id
  FROM devices d
  LEFT JOIN products p ON p.id = d.product_id
 WHERE d.serial_number = $1 OR d.id::text = $1
 LIMIT 1`
			var pmID *uuid.UUID
			if err := c.PgPool.QueryRow(ctx, q, deviceKey).Scan(&pmID); err != nil {
				return nil, fmt.Errorf("resolve param_model by device %q: %w", deviceKey, err)
			}
			return pmID, nil
		})
		// path 不支持类故障时按 product_id 录入 product_unsupported_paths（与 console 端查询同源 repo / resolver）。
		aggregator.SetUnsupportedPathRecorder(unsupportedPathRepo, productIDByDevice)
		// 让聚合器以"实际 device_tasks 全部终态"判定完成，修复理论 total 因 fanout/
		// sequencer 跳过而永远到不了导致的任务卡 running（见 ResultAggregator.finalizeIfComplete）。
		aggregator.SetDeviceTaskStatsReader(task.NewPgTaskRepository(c.PgPool))
		if c.EventBus != nil {
			// 注：completionRouter 通过 miscDeps 持有，方便 ops 模块（在本块之后初始化）
			// 也注册自己的 TaskSourceOps 聚合器。CompletionRouter.Register 是 mutex-safe，
			// 允许 bridge.Subscribe 之后再追加 handler — 启动序无 race（pre-traffic 阶段）。
			c.miscDeps.completionRouter = task.NewCompletionRouter(logger)
			c.miscDeps.completionRouter.Register(task.TaskSourceParamSync, paramSyncCompletionHandled{})
			// #122：source 无关的终态观察者，把每个终态任务写入 sys_task_logs。
			// 必须在 per-source handler 之前用 RegisterObserver 注册，覆盖全部 source。
			if c.adminHandlerDeps != nil && c.adminHandlerDeps.logRepo != nil {
				c.miscDeps.completionRouter.RegisterObserver(
					newTaskLogObserver(c.adminHandlerDeps.logRepo, logger))
			}
			// 顺序关键：Sequencer 必须先注册——某行完成时它先把下一行 device_task 入队，
			// 聚合器随后判定"无在途"才不会在顺序链中途误判完成（见 finalizeIfComplete 注释）。
			c.miscDeps.completionRouter.Register(task.TaskSourceMML, sequencer) // Sprint B Q-V3-3
			c.miscDeps.completionRouter.Register(task.TaskSourceMML, aggregator)
			// F05 MR 任务完成回调改由 initMRTaskModule 注册（依赖图 misc → mrtask，
			// 这里 c.miscDeps.mrTaskRepo 还是 nil，原版 if 永远进不来）。CompletionRouter
			// mutex-safe 支持后挂 handler。

			bridge := task.NewCompletionEventBridge(logger, c.miscDeps.completionRouter, c.Deduper)
			if err := bridge.Subscribe(c.EventBus); err != nil {
				logger.Warn("subscribe task completion bridge", zap.Error(err))
			}
		} else {
			// 单进程部署（单测/无 NATS）下退化为同进程回调。
			// #122：任务日志观察者同样以回调形式兜底，覆盖单进程终态。
			if c.adminHandlerDeps != nil && c.adminHandlerDeps.logRepo != nil {
				c.miscDeps.taskSvc.AddCompletionCallback(
					newTaskLogObserver(c.adminHandlerDeps.logRepo, logger))
			}
			// 顺序同上：Sequencer 先注册，聚合器后注册（见 finalizeIfComplete 注释）。
			c.miscDeps.taskSvc.AddCompletionCallback(sequencer) // Sprint B Q-V3-3
			c.miscDeps.taskSvc.AddCompletionCallback(aggregator)
		}

		logger.Info("MML fan-out bridge enabled")
	}

	// Parameter Library module 已下线（mml_params 表 + 4 个 /mml/param-versions/* 端点删除）。
	// v2.3 sub_field 元数据现由 standard_params 直供，无需独立参数库 API。
	logger.Info("MML console module initialized")

	// Config Baseline module
	baselineRepo := baseline.NewPgBaselineRepository(c.PgPool)
	configTaskRepo := baseline.NewPgConfigTaskRepository(c.PgPool)
	neighborRepo := baseline.NewPgNeighborRepository(c.PgPool)
	baselineSvc := baseline.NewService(baselineRepo, configTaskRepo, neighborRepo, logger)
	c.BaselineSvc = baselineSvc
	c.miscDeps.baselineHandler = baseline.NewHandler(baselineSvc, logger)
	logger.Info("config baseline module initialized")

	// F06 System License 重构 Step 5 起：老 multi-license 模型整体下线。
	// 现在的 license 模块只剩 singleton system_license 链路：
	//   PgSystemLicenseRepository → SystemLicenseService → SystemLicenseHandler
	//   PgSystemLicenseRepository + PgDeviceCounter → Enforcer / Monitor
	systemLicenseRepo := license.NewPgSystemLicenseRepository(c.PgPool)
	deviceCounter := license.NewPgDeviceCounter(c.PgPool)
	licenseMetrics := license.NewEnforcementMetrics(c.MetricsReg)
	licenseEnforcer := license.NewEnforcer(systemLicenseRepo, deviceCounter, logger, licenseMetrics)
	licenseMonitor := license.NewMonitor(systemLicenseRepo, deviceCounter, license.NoopAlertSink{}, licenseMetrics, logger)

	// OEM 公钥加载 + 注入 SignatureVerifier。dev 默认 strict=false + 空
	// PublicKeyDir → 退化为 unverified 放过；prod 推荐配置 OEM 公钥目录 +
	// strict=true 收紧。
	licenseVerifier := license.NewSignatureVerifier(c.Cfg.License.Signing.Strict)
	if dir := c.Cfg.License.Signing.PublicKeyDir; dir != "" {
		if loadErr := licenseVerifier.LoadKeysFromDir(dir); loadErr != nil {
			logger.Warn("license OEM public key load reported errors (non-fatal)",
				zap.String("dir", dir), zap.Error(loadErr))
		}
		logger.Info("license signature verifier loaded",
			zap.String("dir", dir),
			zap.Int("key_count", licenseVerifier.KeyCount()),
			zap.Bool("strict", c.Cfg.License.Signing.Strict))
	}
	// strict=true 但实际 0 keys 启动是高风险静默失败 — 所有 Update 都会被拒，
	// 运维不知原因。Fatal 阻止启动让运维立刻定位 license.signing.public_key_dir 误配。
	if c.Cfg.License.Signing.Strict && licenseVerifier.KeyCount() == 0 {
		logger.Fatal("license strict mode requires at least 1 OEM public key but none loaded; check license.signing.public_key_dir",
			zap.String("dir", c.Cfg.License.Signing.PublicKeyDir))
	}

	c.miscDeps.licenseEnforcer = licenseEnforcer
	c.miscDeps.licenseMonitor = licenseMonitor

	// SystemLicense service/handler — 复用 verifier + enforcer（Update 后调
	// Invalidate 让 enforcer 立即拉新 license，避开 5min cache TTL）。
	systemLicenseSvc := license.NewSystemLicenseService(systemLicenseRepo, logger)
	systemLicenseSvc.SetSignatureVerifier(licenseVerifier)
	systemLicenseSvc.SetEnforcer(licenseEnforcer)
	c.miscDeps.systemLicenseHandler = license.NewSystemLicenseHandler(systemLicenseSvc, logger)

	// Wire enforcer into DeviceService so device.create / future write ops
	// gate on capacity + expiry. Read-only operations are unaffected (D1).
	if c.DeviceService != nil {
		c.DeviceService.SetLicenseEnforcer(licenseEnforcer)
	}

	// Start the cron monitor. ctx-derived timeout per check ensures a stuck
	// scrape can't cascade-fail subsequent ticks.
	if err := c.miscDeps.licenseMonitor.Start(context.Background()); err != nil {
		logger.Warn("license monitor start failed", zap.Error(err))
	}
	logger.Info("license module initialized with enforcer + cron monitor")

	// OpsTools module
	opsTemplateRepo := ops.NewPgTemplateRepository(c.PgPool)
	opsTaskRepo := ops.NewPgTaskRepository(c.PgPool)
	opsCmdRepo := ops.NewPgCommandRecordRepository(c.PgPool)
	opsSvc := ops.NewService(opsTemplateRepo, opsTaskRepo, opsCmdRepo, logger)
	c.miscDeps.opsHandler = ops.NewHandler(opsSvc, logger)

	// F06 运维管理扩展（T-0101..T-0112）— 7 个新子系统
	opsExecRepo := ops.NewPgTaskExecutionRepository(c.PgPool)
	opsDiagRepo := ops.NewPgDiagnosticRepository(c.PgPool)
	opsDLRepo := ops.NewPgDownloadRepository(c.PgPool)
	opsAuditRepo := ops.NewPgAuditLogRepository(c.PgPool)
	opsMWRepo := ops.NewPgMaintenanceWindowRepository(c.PgPool)
	opsPBRepo := ops.NewPgPlaybookRepository(c.PgPool)

	opsAuditSvc := ops.NewAuditLogService(opsAuditRepo, logger)
	opsApprovalSvc := ops.NewApprovalService(opsTaskRepo, opsAuditSvc, logger)
	opsDiagSvc := ops.NewDiagnosticService(opsDiagRepo, opsAuditSvc, logger)
	opsDLSvc := ops.NewDownloadService(opsDLRepo, opsAuditSvc, logger)
	opsMWSvc := ops.NewMaintenanceWindowService(opsMWRepo, opsAuditSvc, logger)
	opsPBSvc := ops.NewPlaybookService(opsPBRepo, logger)
	opsExecutor := ops.NewTaskExecutor(opsTaskRepo, opsExecRepo, opsAuditSvc, logger)
	// T-0102-c: wire device-task enqueuer + SSE hub so inline RPC tasks
	// fan out to internal/task (Redis + PG) and emit per-device events.
	opsExecutor.SetEnqueuer(c.TaskSvc)
	// T-0102-e: wire per-device rate limiter so a single device can't be
	// flooded by one batch task; also enables retry-with-backoff inside
	// dispatchInlineRPC. Defaults: 5 RPC/sec/device + 3 enqueue retries.
	opsExecutor.SetLimiter(ops.NewConcurrencyLimiter(0, 0))
	opsBGSvc := ops.NewBreakGlassService(opsAuditSvc, logger)
	opsInspectionSvc := ops.NewInspectionService(opsDiagSvc, opsAuditSvc, logger)
	opsSSEHub := ops.NewSSEHub()
	opsExecutor.SetSSEHub(opsSSEHub)

	c.miscDeps.opsExtHandler = ops.NewExtHandler(
		opsDiagSvc, opsDLSvc, opsAuditSvc, opsMWSvc, opsPBSvc,
		opsExecutor, opsApprovalSvc, opsBGSvc, opsInspectionSvc,
		opsSvc, opsSSEHub, logger,
	)

	// T-0102 残债 2 / B′：completion ACS→ops 桥。
	// 当 device_task (source=ops) 进入终态时，更新 ops_task_executions 行 +
	// 发 SSE command.completed / command.failed 事件，闭合入队侧 SSE 通路
	// (publishDispatchEvent → publishCompletionEvent)。
	// 与 MML 模块的 ResultAggregator 共享同一个 CompletionRouter，
	// router.Register 是 mutex-safe 允许 bridge.Subscribe 之后再追加。
	if c.miscDeps.taskSvc != nil {
		opsAggregator := ops.NewResultAggregator(opsExecRepo, opsSSEHub, logger)
		if c.miscDeps.completionRouter != nil {
			// 多进程：NATS bridge 路由
			c.miscDeps.completionRouter.Register(task.TaskSourceOps, opsAggregator)
			logger.Info("ops completion aggregator registered to CompletionRouter (multi-process)")
		} else {
			// 单进程：直接挂到 taskSvc completion callback
			c.miscDeps.taskSvc.AddCompletionCallback(opsAggregator)
			logger.Info("ops completion aggregator registered to taskSvc (single-process)")
		}
	}

	logger.Info("ops tools module initialized (incl. F06 ext T-0101..T-0112)")

	// Report module
	reportDefRepo := report.NewPgDefinitionRepository(c.PgPool)
	reportRecordRepo := report.NewPgRecordRepository(c.PgPool)
	reportService := report.NewService(reportDefRepo, reportRecordRepo, c.EventBus, logger)
	c.miscDeps.reportHandler = report.NewHandler(reportService, c.MinIO, c.Cfg.MinIO.Buckets.Reports, logger)
	logger.Info("report module initialized")

	// NE Direct module (conditional)
	if c.Cfg.NEDirect.Enabled {
		neSessionRepo := nedirect.NewPgSessionRepository(c.PgPool)
		neCommandRepo := nedirect.NewPgCommandRepository(c.PgPool)
		neService := nedirect.NewService(neSessionRepo, neCommandRepo, c.DeviceService, c.AlarmEngine, c.EventBus, logger)
		// 安全中间件：强制 JWT/API-Key 认证 + per-endpoint/per-device Redis 限流 + 审计日志。
		neMiddleware := buildNEDirectMiddleware(c, logger)
		neHandler := nedirect.NewHandler(neService, neMiddleware, logger)
		neServer := nedirect.NewServer(c.Cfg.NEDirect, neHandler, logger)
		if err := neServer.Start(); err != nil {
			logger.Error("ne-direct server start failed", zap.Error(err))
		} else {
			c.GS.Register("ne-direct", 1, func(ctx context.Context) error { return neServer.Shutdown(ctx) })
			logger.Info("ne-direct server started",
				zap.String("host", c.Cfg.NEDirect.Host),
				zap.Int("port", c.Cfg.NEDirect.Port))
		}
	}

	// 设备自动归组统一使用 GroupMatchEngine：
	//   - 新设备注册和分组属性变化由领域事件触发
	//   - 规则变化和每小时全量重评估负责兜底
	// （历史的 device_rules 独立引擎已彻底下线，相关表/handler/seed 一并删除）
	matcher := topology.NewDeviceMatcher(c.GroupRepo, c.PgPool, logger)
	deviceLister := topology.NewPgDeviceLister(c.PgPool, logger)

	// 设备分组自动匹配引擎（GroupMatchEngine）：消费 L2 分组自带的匹配规则，
	// 触发时机 = 分组新增/编辑 + 新设备注册 + 心跳 inform + cron @hourly。
	groupMatchEngine := topology.NewGroupMatchEngine(
		matcher, deviceLister, c.GroupRepo, logger)
	groupMatchEngine.SetEventBus(c.EventBus)
	if err := groupMatchEngine.Start(context.Background()); err != nil {
		logger.Error("group match engine Start failed", zap.Error(err))
	}
	if c.GroupService != nil {
		c.GroupService.SetGroupMatchEngine(groupMatchEngine)
	}
	c.GS.Register("topology-group-match", 1, func(_ context.Context) error {
		groupMatchEngine.Stop()
		return nil
	})

	// System Info endpoint
	c.miscDeps.sysInfoHandler = components.NewSystemInfoHandler(c.PgPool, c.Redis, logger)
	if c.StorageCollector != nil {
		c.miscDeps.sysInfoHandler.SetStorageCollector(c.StorageCollector)
	}

	// PM threshold
	c.miscDeps.thresholdRepo = pm.NewPgThresholdRepository(c.PgPool)

	// Dead-letter admin handler (T-0012 / R-106).
	// Worker process owns the runner that writes to dead_letters; the app
	// process exposes read/delete/replay over /admin/dead-letters. Replay uses
	// a per-module runner.Runner (sharing the EventBus publisher) so the app
	// can re-publish events back to the bus — worker subscribers will pick
	// them up like any normal event.
	dlqRepo := dlq.NewPgRepository(c.PgPool)
	dlqMetrics := runner.NewMetrics(c.MetricsReg)
	dlqHandler := admin.NewDeadLetterHandler(dlqRepo, logger)
	if c.EventBus != nil {
		pmReplayer := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, c.EventBus, dlqMetrics, logger)
		dlqHandler.SetReplayer("pm", pmReplayer)
	}
	c.miscDeps.deadLetterHandler = dlqHandler
	logger.Info("dead-letter admin handler initialized")

	// T-0137 / M2: TR069 报文跟踪 — app 端管理面。
	//   - 任务 CRUD + 查询接口
	//   - CreateTask/StopTask 通过 EventBus 发 trace.task.{started,stopped,purged} 事件，
	//     ACS 实例订阅后实时增删 SN 白名单；worker 订阅 purged 异步清理报文。
	//   - 注入 BulkStore：handler 的 GetMessagePayload 端点遇到 external 报文时按需拉 MinIO
	// KPI/时序库物理分离：trace_messages 已迁时序库（TsPool），retention 超表。
	traceRepo := trace.NewPgRepository(c.TsPool)
	traceSvc := trace.NewService(traceRepo, trace.DefaultConfig(), logger)
	if c.EventBus != nil {
		traceSvc.SetEventBus(c.EventBus)
	}
	if c.MinIO != nil && c.Cfg.MinIO.Buckets.TraceBulk != "" {
		traceSvc.SetPayloadFetcher(trace.NewBulkStore(c.MinIO, c.Cfg.MinIO.Buckets.TraceBulk))
	}
	// M3-01：Prometheus 指标（app 端 publish 失败 drop 计数 + 兜底）
	traceSvc.SetMetrics(trace.NewMetrics(c.MetricsReg))
	c.miscDeps.traceService = traceSvc
	traceHandler := trace.NewHandler(traceSvc, logger)
	if c.PresignBridge != nil {
		// issue #548 切片 2：trace 模块作为 PoC 接入 PresignBridge——sys_configs 改
		// storage.minio_public_endpoint 后无需重启，下一次 GetExportJob 立即用新 endpoint
		// 重签下载 URL（minio-presign-bridge 模块已注册 SavedHook 触发 SetPublicEndpoint）。
		traceHandler.SetPresignProvider(c.PresignBridge)
	} else if c.MinIO != nil {
		// 兜底（dev/test 路径没装 bridge）：保留旧 NewPresignClient 路径。
		// L-8：预签名 URL 必须用 PublicEndpoint 签出来浏览器才能直接打开。
		presignClient, err := minioinfra.NewPresignClient(c.Cfg.MinIO)
		if err != nil {
			logger.Warn("create MinIO presign client failed, falling back to internal client",
				zap.Error(err))
			traceHandler.SetMinIO(c.MinIO)
		} else {
			traceHandler.SetMinIO(presignClient)
		}
	}
	c.miscDeps.traceHandler = traceHandler
	// SSE 通知：订阅 trace.task.* 事件转发给在线用户
	if c.EventBus != nil && c.miscDeps.messageHub != nil {
		traceNotifier := trace.NewSSENotifier(c.miscDeps.messageHub, logger)
		if _, err := traceNotifier.Subscribe(c.EventBus); err != nil {
			logger.Warn("trace SSE notifier subscribe failed", zap.Error(err))
		} else {
			logger.Info("trace SSE notifier subscribed (T-0137 M2-07)")
		}
	}
	logger.Info("trace module initialized (T-0137 M2)")

	return nil
}

// miscDeps holds handlers and deps from miscellaneous modules.
// These are populated by initMiscModules and consumed during route registration.
type miscDeps struct {
	// MR
	mrStore    *mr.PgMRStore
	mrIndRepo  *mr.PgIndicatorRepository
	mrMapRepo  *mr.PgMappingRepository
	mrTaskRepo *mrtask.PgRepository // F05 任务管理（initMRTaskModule 装配）

	// 文件管理 4 Tab 批量下载（firmware / config_snapshot / device_license / mr）
	bundleSvc *bundle.Service

	// Software
	softwareHandler     *software.Handler
	softwareService     *software.SoftwareService
	softwareTaskRepo    software.TaskRepository
	softwareSubTaskRepo software.SubTaskRepository
	canaryMonitor       *software.CanaryMonitor
	taskScheduler       *software.TaskScheduler // 定时任务调度器（000181）
	ufteHandler         *ufte.Handler
	ufteService         *ufte.Service

	// Provision
	provisionRepo     *provision.PgProvisioningTaskRepository
	provisionEngine   *provision.ProvisioningEngine
	plugAndPlayRepo   *provision.PgPlugAndPlayRepository
	paramSyncHandler  *paramsync.Handler
	paramSyncStarter  *paramSyncStarter
	paramSyncConsumer *paramsync.ResultConsumer

	// Task
	taskHandler      *task.Handler
	taskSvc          *task.TaskService
	completionRouter *task.CompletionRouter // 由 MML 模块创建；ops 模块在自己 init 时也注册

	// Backup
	backupHandler       *backup.Handler
	backupPolicyMonitor *backup.PolicyMonitor // T-0073 Phase 1
	snapshotService     *backup.SnapshotService

	// StationLog
	stationlogHandler *stationlog.Handler

	// EventLog（设备活动审计流水）
	eventlogHandler *eventlog.Handler

	// RebootRecord（统一重启记录只读视图）
	rebootrecordHandler *rebootrecord.Handler

	// Dashboard
	dashboardHandler *dashboard.Handler

	// Northbound
	nbRouter *northbound.Router

	// Interop
	interopHandler *interop.Handler

	// Syslog
	syslogHandler *syslog.Handler

	// Config sync
	syncHandler *config.SyncHandler

	// File Manager
	fileHandler *filemanager.Handler

	// MML
	mmlHandler        *mml.Handler
	mmlService        *mml.Service
	mmlScheduler      *mml.Scheduler
	mmlAdminHandler   *mml.AdminHandler   // T-0123-P0 catalog 管理 13 端点
	mmlConsoleHandler *mml.ConsoleHandler // T-0123-P1 Console 5 端点（group-tree / sub-fields / render / parse / execute-statements）

	// Baseline
	baselineHandler *baseline.Handler

	// License + enforcement (F06 重构 Step 5：multi-license 模型已下线，
	// 仅剩 singleton system_license 链路)
	licenseEnforcer *license.EnforcerImpl
	licenseMonitor  *license.Monitor

	// F06 System License 重构（PRD F06-system-license-redesign Step 5）：
	// singleton 模型 handler，唯一的 license REST 入口。
	systemLicenseHandler *license.SystemLicenseHandler

	// DeviceDetail "License 参数" tab 后端（device 模块 license_params.go）
	licenseParamHandler *device.LicenseParamHandler

	// Ops
	opsHandler    *ops.Handler
	opsExtHandler *ops.ExtHandler

	// Report
	reportHandler *report.Handler

	// System Info
	sysInfoHandler *components.SystemInfoHandler

	// PM Threshold
	thresholdRepo *pm.PgThresholdRepository

	// SSE / Events
	sseHandler          *events.SSEHandler
	notificationHandler *notification.Handler
	messageHub          *events.MessageHub

	// W2.A.4 / T-0043: Notification template + history
	notifTemplateHandler *notification.TemplateHandler
	notifHistoryHandler  *notification.HistoryHandler

	// T-0152: Alertmanager 告警 webhook 入口（SMTP 邮件发送链）
	alertWebhookHandler *notification.AlertWebhookHandler

	// T-0012 / R-106: worker retry + dead-letter queue admin
	deadLetterHandler *admin.DeadLetterHandler

	// T-0137 / M1: TR069 报文跟踪 handler（app 侧仅管 CRUD，capture flusher 在 ACS 侧）
	traceHandler *trace.Handler
	traceService *trace.Service
}

// syncDeviceChecker adapts device.DeviceReader to config.DeviceExistenceChecker.
// GetBySerialNumber 返回 (nil,nil) 表示设备不存在；据此映射 push/pull 入队前的 404 预检。
type syncDeviceChecker struct {
	repo device.DeviceReader
}

func (a *syncDeviceChecker) ExistsBySerialNumber(ctx context.Context, sn string) (bool, error) {
	dev, err := a.repo.GetBySerialNumber(ctx, sn)
	if err != nil {
		return false, fmt.Errorf("lookup device %s: %w", sn, err)
	}
	return dev != nil, nil
}

// taskDeviceLookup adapts device.DeviceReader to task.DeviceLookup.
type taskDeviceLookup struct {
	repo device.DeviceReader
}

func (a *taskDeviceLookup) GetConnectionRequestURL(ctx context.Context, deviceSN string) (string, error) {
	dev, err := a.repo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return "", fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}
	if dev == nil {
		return "", nil
	}
	return dev.ConnectionRequestURL, nil
}

// taskCRSender adapts connreq.Dispatcher to task.ConnectionRequestSender.
type taskCRSender struct {
	dispatcher *connreq.Dispatcher
	serverAddr string
}

func (a *taskCRSender) Send(ctx context.Context, deviceSN, httpURL string) error {
	return a.dispatcher.Send(ctx, deviceSN, httpURL, a.serverAddr, true)
}

// stunServerPortParamPath is the TR-069 parameter that, when present in
// device_parameters, overrides the hard-coded UDP LAN CR fallback port (3478).
// Annex G lets vendors pick this freely; for example the empirically tested
// BAICELLS BSC7041C243 and Dengyo BSC7079B243 both report 3478, but other
// firmware can ship different values that would silently break LAN-direct wake.
const stunServerPortParamPath = "Device.ManagementServer.STUNServerPort"

// newSTUNServerPortLookup returns a connreq.LANPortLookup that reads
// Device.ManagementServer.STUNServerPort for the given SN. It logs and
// returns (0, false) on any lookup miss / parse failure so the dispatcher
// falls back to the UDPSender's hard-coded default port.
//
// Resolution path: SN → device row (DeviceReader.GetBySerialNumber) →
// device_parameters[STUNServerPort] (DeviceParameterRepository.GetByPath).
// Both repos are already constructed for other purposes — no new schema.
func newSTUNServerPortLookup(
	repo device.DeviceReader,
	paramRepo device.DeviceParameterRepository,
	logger *zap.Logger,
) connreq.LANPortLookup {
	if repo == nil || paramRepo == nil {
		return nil
	}
	return func(ctx context.Context, deviceSN string) (int, bool) {
		dev, err := repo.GetBySerialNumber(ctx, deviceSN)
		if err != nil {
			logger.Debug("stun server port lookup: device fetch failed",
				zap.String("device_sn", deviceSN), zap.Error(err))
			return 0, false
		}
		if dev == nil {
			return 0, false
		}
		param, err := paramRepo.GetByPath(ctx, dev.ID, stunServerPortParamPath)
		if err != nil {
			logger.Debug("stun server port lookup: param fetch failed",
				zap.String("device_sn", deviceSN), zap.Error(err))
			return 0, false
		}
		if param == nil || param.ParameterValue == "" {
			return 0, false
		}
		port, err := strconv.Atoi(param.ParameterValue)
		if err != nil || port <= 0 || port > 65535 {
			logger.Debug("stun server port lookup: invalid value",
				zap.String("device_sn", deviceSN),
				zap.String("raw", param.ParameterValue))
			return 0, false
		}
		return port, true
	}
}

// mmlPathMissAdapter 把 task.TaskService 的 AggregatePathTranslationMissBySourceID
// 适配为 mml.DeviceTaskPathMissAggregator 接口。task 包与 mml 包分别定义统计
// struct（消费者驱动接口），适配器在装配层完成字段映射，避免 mml 包反向依赖
// task 包的具体类型。
type mmlPathMissAdapter struct {
	svc *task.TaskService
}

func (a *mmlPathMissAdapter) AggregatePathTranslationMissBySourceID(
	ctx context.Context, sourceID string,
) (mml.PathTranslationMissStatsView, error) {
	stats, err := a.svc.AggregatePathTranslationMissBySourceID(ctx, sourceID)
	if err != nil {
		return mml.PathTranslationMissStatsView{}, err
	}
	return mml.PathTranslationMissStatsView{
		DeviceCount: stats.DeviceCount,
		PathCount:   stats.PathCount,
		AnyMiss:     stats.AnyMiss,
	}, nil
}

// mmlDeviceTaskResultAdapter 把 task.TaskService.ListResultsBySourceID 适配为
// mml.DeviceTaskResultLister。同消费者驱动模式：task 包返回 task.DeviceTaskResultRow，
// 这里逐字段拷贝到 mml.DeviceTaskResultRowView，让 mml 包不反向 import task 包。
type mmlDeviceTaskResultAdapter struct {
	svc *task.TaskService
}

func (a *mmlDeviceTaskResultAdapter) ListResultsBySourceID(
	ctx context.Context, sourceID string, page, pageSize int,
) ([]mml.DeviceTaskResultRowView, int64, error) {
	rows, total, err := a.svc.ListResultsBySourceID(ctx, sourceID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]mml.DeviceTaskResultRowView, 0, len(rows))
	for _, r := range rows {
		out = append(out, mml.DeviceTaskResultRowView{
			DeviceTaskID: r.ID,
			DeviceSN:     r.DeviceSN,
			Method:       r.Method,
			Params:       r.Params,
			CommandKey:   r.CommandKey,
			CWMPID:       r.CWMPID,
			Status:       r.Status,
			ErrorCode:    r.ErrorCode,
			ErrorMessage: r.ErrorMessage,
			Result:       r.Result,
			SentAt:       r.SentAt,
			CompletedAt:  r.CompletedAt,
			CreatedAt:    r.CreatedAt,
			CommandIndex: r.CommandIndex,
			DeviceIndex:  r.DeviceIndex,
		})
	}
	return out, total, nil
}
