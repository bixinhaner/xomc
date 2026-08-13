package provider

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/alarm"
	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/components"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/realtime"
	"github.com/omcgo/omcgo/internal/core/systimezone"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/geofence"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	kpirouter "github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/retention"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/provision"
	"github.com/omcgo/omcgo/internal/quicksettings"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/topology"
)

// Container 聚合所有基础设施依赖和跨模块共享服务。
//
// 分为两层：
//   - 基础设施层：由 bootstrap 初始化（DB、Redis、MinIO 等）
//   - 共享服务层：由各模块 Init 函数设置（DeviceService、DMRegistry 等）
//
// 各模块 Init 函数按依赖顺序执行，先初始化的模块将共享服务写入 Container，
// 后初始化的模块从 Container 读取所需依赖。
type Container struct {
	// ===== 基础设施（由 bootstrap 设置）=====
	PgPool                   *pgxpool.Pool
	TsPool                   *pgxpool.Pool
	Redis                    redis.UniversalClient
	PMRedis                  redis.UniversalClient
	MinIO                    *minio.Client
	EventBus                 event.EventBus
	Realtime                 *realtime.CoreNATS
	Deduper                  *event.Deduper
	TaskSvc                  *task.TaskService
	Carriers                 *carrier.CarrierRegistry
	Cfg                      *appconfig.AppConfig
	Logger                   *zap.Logger
	LogGate                  storageprotection.LogAdmissionController
	GS                       *components.GracefulShutdown
	MetricsReg               *prometheus.Registry
	Health                   *components.HealthChecker
	StorageProtection        *storageprotection.Service
	StorageProtectionHandler *storageprotection.Handler
	StorageCollector         components.StorageCollector

	// ===== 共享服务（由各模块 Init 设置）=====

	// DictLoad 模块设置（T-0098 P1-06）
	DictLoaderRegistry *dictloader.Registry

	// ProductRegistry 模块设置（T-0098 P2-01）
	ProductRegistry *product.Registry

	// AdminService（initAdminModule 设置；misc/license 模块注入 license feature gate）
	AdminService *admin.AdminService

	// QuickSettingsRegistry 模块设置（T-0138）
	QuickSettingsRegistry *quicksettings.Registry

	// ParamRegistry 模块设置（T-0098 P2-02）
	ParamRegistry *parammodel.Registry

	// ParamIntersect 模块设置（T-0098 P2-03）
	ParamIntersect *parammodel.IntersectService

	// AlarmDefModule 设置（T-0098 P3-04）
	AlarmDefRegistry    *alarmdef.Registry
	AlarmDefHandler     *alarmdef.Handler
	AlarmDefFileHandler *alarmdef.FileHandler

	// ParamModelHandler 设置（T-0098 P3-02）
	ParamModelRepo    *parammodel.PgRepository
	ParamModelHandler *parammodel.Handler

	// ProductHandler 设置（T-0098 P3-01）
	ProductRepo    *product.PgRepository
	ProductHandler *product.Handler

	// ConfigModule 设置（T-0098 P5-01：旧 DMRegistry / DMImporter 已删除）
	TemplateService *template.ConfigTemplateService

	// 兼容参数同步服务（AutoSync.Enabled 时由 provisionModule.Init 创建，否则为 nil）。
	// durable 请求入口由 paramSyncStarter 独立装配，不受 AutoSync 开关影响。
	// 用途：让 config.SyncHandler.PullConfig 也能拿到统一的批次拆分能力，从而触发
	// ACS handler.tryRecoverGPVFault 的 Fault 自愈循环。
	SyncSvc *provision.SyncService

	// TopologyModule 设置
	GroupRepo    *topology.PgDeviceGroupRepository
	GroupService *topology.DeviceGroupService

	// AdminModule 设置
	JWTService     *admin.JWTService
	APIKeySvc      *admin.APIKeyService
	UserRepo       *admin.PgUserRepository
	RoleRepo       *admin.PgRoleRepository
	AuditRepo      *admin.PgAuditRepository
	PermService    *admin.PermissionService
	SysConfigSvc   *admin.SysConfigService  // 提供 RegisterSavedHook 给其他模块挂 cache invalidate
	SecurityPolicy *admin.SecurityPolicy    // 让其他模块可注册 InvalidateCache hook
	DictService    *admin.DictionaryService // #241：三库导入 XML 后按 source_table 刷新绑定字典（依赖 admin 模块先初始化）
	SystemTimezone *systimezone.Provider    // 系统时区统一源（basic/timezoneCode）

	// MinIO 预签名 client 运行时订阅桥（issue #548 切片 2 D 后端）。
	// 由 minio-presign-bridge 模块装配；sys_configs 写入 storage.minio_public_endpoint
	// 时原子替换内部 client，trace / 后续 mml/license/backup 通过 .Get() 拿当前 client。
	// 可为 nil（dev/test 路径无 MinIO 部署时），消费方需 nil-safe（如 trace.Handler 的 presignProvider 检查）。
	PresignBridge *minioinfra.PresignBridge

	// DeviceModule 设置
	DeviceService  *device.DeviceService
	DeviceRepo     device.DeviceRepository
	ParamRepo      device.DeviceParameterRepository
	DeviceInfoRepo device.DeviceInfoRepository
	ConnReqClient  *connreq.Client
	StunStore      *stun.Store
	// DeviceCache 用于其他模块（如 provision lazy bind / product orphan bind）
	// 在写库后失效 SN 缓存（T-0176-PR-D）。可为 nil（仅 dev/test 路径），消费方需 nil-safe。
	DeviceCache *device.DeviceCache
	// InformHandler 暴露给后续模块（如 topology matcher）注入"心跳自动分组"钩子。
	// 在 initDeviceModule 创建时尚无 matcher，由 initMiscModules 创建 matcher 后
	// 反向注入 SetGroupAssigner — 避免循环依赖（device → topology）。
	InformHandler *device.InformHandler

	// GeofenceModule 设置。Phase 1 仅提供配置、版本和绑定能力，不连接设备控制。
	GeofenceService *geofence.Service
	GeofenceHandler *geofence.Handler

	// AlarmModule 设置
	AlarmPgStore             *alarm.PgAlarmStore
	AlarmEngine              *alarm.AlarmEngine
	AlarmSyncProcessor       *alarm.AlarmSyncProcessor
	AlarmFilterEngine        *alarm.FilterEngine
	AlarmHistoryRetentionSvc *alarm.HistoryRetentionService

	// PMModule 设置
	PMCounterRepo *counter.PgCounterRepository
	PMKPIRepo     *kpi.PgKPIRepository
	// KPIRouteInvalidator 由 dictload 创建、PM Router 初始化后绑定本进程 L1，
	// 统一供指标 reload、人工恢复及后续写后热更新调用。
	KPIRouteInvalidator *kpirouter.Invalidator
	KPIRouteMetrics     *kpirouter.Metrics

	// PM retention 策略层（T-0164-P2 / G2）
	// sys_configs (category='pm.retention') 5 键的运行时缓存 + RegisterSavedHook 监听 reload
	// G3/G5 实施时通过 retention.Service.RegisterListener 挂 alter_compression_policy / alter_retention_policy 触发器
	PMRetentionSvc *retention.Service

	// BaselineModule 设置
	BaselineSvc *baseline.Service

	// ===== 内部 deps（各模块 handler/路由注册使用）=====
	configHandlerDeps   *configHandlerDeps
	topologyHandlerDeps *topologyHandlerDeps
	adminHandlerDeps    *adminHandlerDeps
	deviceHandlerDeps   *deviceHandlerDeps
	alarmHandlerDeps    *alarmHandlerDeps
	pmHandlerDeps       *pmHandlerDeps
	miscDeps            miscDeps
}

func (c *Container) kpiRouteRedisCache() *kpirouter.RedisCache {
	if c == nil || c.PMRedis == nil {
		return nil
	}
	return kpirouter.NewRedisCache(c.PMRedis)
}
