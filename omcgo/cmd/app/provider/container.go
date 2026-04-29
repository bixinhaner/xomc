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
	"github.com/omcgo/omcgo/internal/config/baseline"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
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
	TaskSvc    *task.TaskService
	Carriers   *carrier.CarrierRegistry
	Cfg        *appconfig.AppConfig
	Logger     *zap.Logger
	GS         *components.GracefulShutdown
	MetricsReg *prometheus.Registry
	Health     *components.HealthChecker

	// ===== 共享服务（由各模块 Init 设置）=====

	// ConfigModule 设置
	DMRegistry      *datamodel.DataModelRegistry
	DMImporter      *datamodel.DataModelImporter
	TemplateService *template.ConfigTemplateService

	// TopologyModule 设置
	GroupRepo    *topology.PgDeviceGroupRepository
	GroupService *topology.DeviceGroupService

	// AdminModule 设置
	JWTService  *admin.JWTService
	APIKeySvc   *admin.APIKeyService
	UserRepo    *admin.PgUserRepository
	RoleRepo    *admin.PgRoleRepository
	AuditRepo   *admin.PgAuditRepository
	PermService *admin.PermissionService

	// DeviceModule 设置
	DeviceService  *device.DeviceService
	DeviceRepo     device.DeviceRepository
	ParamRepo      device.DeviceParameterRepository
	DeviceInfoRepo device.DeviceInfoRepository
	ConnReqClient  *connreq.Client
	StunStore      *stun.Store

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
