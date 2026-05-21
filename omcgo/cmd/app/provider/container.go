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
	"github.com/omcgo/omcgo/internal/core/dictloader"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mml/catalogloader"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/quicksettings"
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
	PgPool     *pgxpool.Pool
	TsPool     *pgxpool.Pool
	Redis      redis.UniversalClient
	MinIO      *minio.Client
	EventBus   event.EventBus
	Deduper    *event.Deduper
	TaskSvc    *task.TaskService
	Carriers   *carrier.CarrierRegistry
	Cfg        *appconfig.AppConfig
	Logger     *zap.Logger
	GS         *components.GracefulShutdown
	MetricsReg *prometheus.Registry
	Health     *components.HealthChecker

	// ===== 共享服务（由各模块 Init 设置）=====

	// DictLoad 模块设置（T-0098 P1-06）
	DictLoaderRegistry *dictloader.Registry

	// MML 控制台 v2.3 catalog Loader（dictloader 模式第 5 个 Loader）
	// 用于 admin REST API /api/v1/mml/catalog/{info,reload} 的引用
	MMLCatalogLoader *catalogloader.Loader

	// MMLV2Schema 当 true 时启用 spec v2.3 (chapter 顶层) 全链路。
	// 由 env MML_V2_SCHEMA=true 驱动，在 dictload provider 初始化时读取。
	// 同一 toggle 同时驱动 catalogloader.WithV2Schema (写链) 与 mml.WithV2Mode
	// (读链)，确保 Loader 与 BuildTree 路径一致。
	MMLV2Schema bool

	// ProductRegistry 模块设置（T-0098 P2-01）
	ProductRegistry *product.Registry

	// QuickSettingsRegistry 模块设置（T-0138）
	QuickSettingsRegistry *quicksettings.Registry

	// ParamRegistry 模块设置（T-0098 P2-02）
	ParamRegistry *parammodel.Registry

	// ParamIntersect 模块设置（T-0098 P2-03）
	ParamIntersect *parammodel.IntersectService

	// AlarmDefModule 设置（T-0098 P3-04）
	AlarmDefRegistry *alarmdef.Registry
	AlarmDefHandler  *alarmdef.Handler

	// ParamModelHandler 设置（T-0098 P3-02）
	ParamModelRepo    *parammodel.PgRepository
	ParamModelHandler *parammodel.Handler

	// ProductHandler 设置（T-0098 P3-01）
	ProductRepo    *product.PgRepository
	ProductHandler *product.Handler

	// ConfigModule 设置（T-0098 P5-01：旧 DMRegistry / DMImporter 已删除）
	TemplateService *template.ConfigTemplateService

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
	SysConfigSvc   *admin.SysConfigService // 提供 RegisterSavedHook 给其他模块挂 cache invalidate
	SecurityPolicy *admin.SecurityPolicy   // 让其他模块可注册 InvalidateCache hook

	// DeviceModule 设置
	DeviceService  *device.DeviceService
	DeviceRepo     device.DeviceRepository
	ParamRepo      device.DeviceParameterRepository
	DeviceInfoRepo device.DeviceInfoRepository
	ConnReqClient  *connreq.Client
	StunStore      *stun.Store
	// InformHandler 暴露给后续模块（如 topology matcher）注入"心跳自动分组"钩子。
	// 在 initDeviceModule 创建时尚无 matcher，由 initMiscModules 创建 matcher 后
	// 反向注入 SetGroupAssigner — 避免循环依赖（device → topology）。
	InformHandler *device.InformHandler

	// AlarmModule 设置
	AlarmPgStore *alarm.PgAlarmStore
	AlarmEngine  *alarm.AlarmEngine

	// PMModule 设置
	PMCounterRepo *counter.PgCounterRepository
	PMKPIRepo     *kpi.PgKPIRepository

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
