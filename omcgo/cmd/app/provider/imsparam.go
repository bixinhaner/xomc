package provider

import (
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/imsparam"
)

// initImsParamModule 初始化核心网参数文件库模块（docs/design/imscore-file-transfer.md）。
//
// 装配内容：
//   - imsparam.Service：参数文件库（导入/列表/删除/下载）+ IMS_PARAM_DISTRIBUTE 派发器
//   - imsparam.Handler：/imsparam/param-types、/imsparam/param-files/*
//   - 反向注入 ufte.Service.SetImsParamDispatcher（与 backup → ufte 同款模式）
//
// 依赖 software（BuildDirectDispatchCommandKey）、ufte（派发器注入目标）、
// admin（sys_configs Policy 失效钩子）。
func initImsParamModule(c *Container) error {
	logger := c.Logger.Named("imsparam")

	repo := imsparam.NewPgRepository(c.PgPool)
	service := imsparam.NewService(
		repo,
		c.MinIO,
		c.DeviceRepo,
		c.TaskSvc,
		appconfig.NormalizeConfigBackupBucket(c.Cfg.MinIO.Buckets.ConfigBackup),
		logger,
	)
	service.SetStorageAdmission(c.StorageProtection)
	c.miscDeps.imsParamService = service

	// 运行时 ACS 下载配置（sys_configs 'acs_transfer'，前端"系统管理 → ACS 传输"维护），
	// 与 software / backup 模块同款 Policy + 地址决策 resolver。
	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	transferPolicy := transfercfg.NewPolicy(
		newSoftwareTransferDefaults(c.Cfg.Upgrade),
		newTransferSysConfigLookup(sysConfigRepo),
	)
	if c.SysConfigSvc != nil {
		registerTransferPolicyInvalidation(c.SysConfigSvc, transferPolicy)
	}
	service.SetTransferProvider(transferPolicy)
	paramRepo := c.ParamRepo
	if paramRepo == nil {
		paramRepo = device.NewPgDeviceParameterRepository(c.PgPool)
	}
	service.SetDownloadAddressResolver(newTransferAddressResolver(transferPolicy, paramRepo))

	handler := imsparam.NewHandler(service, c.MinIO)
	c.miscDeps.imsParamHandler = handler

	// 反向注入 UFTE：IMS_PARAM_DISTRIBUTE 任务创建 / Start 走本模块派发器。
	if c.miscDeps.ufteService != nil {
		c.miscDeps.ufteService.SetImsParamDispatcher(service)
	} else {
		logger.Warn("ufte service not ready; IMS_PARAM_DISTRIBUTE dispatch disabled until restart")
	}
	logger.Info("imsparam module initialized")
	return nil
}
