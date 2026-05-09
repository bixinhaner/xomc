// Package appconfig 定义所有微服务的配置结构体和加载函数。
// 配置通过 YAML 文件加载（Load/LoadWithEnvOverride），支持环境变量覆盖（前缀 OMCGO_）。
// 三个主要入口配置：
//   - ACSConfig：ACS 服务（cmd/acs），负责 CPE TR-069 会话处理
//   - AppConfig：App 服务（cmd/app），负责北向 API 和业务逻辑
//   - WorkerConfig：Worker 服务（cmd/worker），负责 PM/MR/Alarm 等后台任务
package appconfig

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ACSConfig 是 ACS 服务的完整配置。
// 由 cmd/acs/main.go 读取 config.yaml 后初始化，包含 CPE 接入、会话控制、
// 文件上传下载、事件总线、存储和可观测性配置。
type ACSConfig struct {
	Server                  ACSServerConfig       `mapstructure:"server"`
	Session                 SessionConfig         `mapstructure:"session"`
	RateLimit               RateLimitConfig       `mapstructure:"rate_limit"`
	Auth                    AuthConfig            `mapstructure:"auth"`
	STUN                    STUNConfig            `mapstructure:"stun"`
	PostSessionWake         PostSessionWakeConfig `mapstructure:"post_session_wake"`
	Redis                   RedisConfig           `mapstructure:"redis"`
	NATS                    NATSConfig            `mapstructure:"nats"`
	DB                      PostgresConfig        `mapstructure:"db"`
	MinIO                   MinIOConfig           `mapstructure:"minio"`
	Upload                  UploadConfig          `mapstructure:"upload"`
	Download                DownloadConfig        `mapstructure:"download"`
	Metrics                 MetricsConfig         `mapstructure:"metrics"`
	Tracer                  TracerConfig          `mapstructure:"tracer"`
	Log                     LogConfig             `mapstructure:"log"`
	ProtocolLog             ProtocolLogConfig     `mapstructure:"protocol_log"`               // ACS 协议交互日志（独立文件记录原始 XML）
	RequestIDPrefix         string                `mapstructure:"request_id_prefix"`          // 请求 ID 前缀，如 "acs"
	EnableTestTaskInjection bool                  `mapstructure:"enable_test_task_injection"` // 启用随机测试任务注入（仅用于测试）
}

// STUNConfig 配置 STUN UDP 服务器，用于 NAT 穿透和 Connection Request 触发。
// 当 CPE 处于 NAT 后时，ACS 无法直接访问 CPE HTTP 地址，改通过 UDP 发送 Connection Request。
// CPE 在 Inform 中携带 UDPConnectionRequestAddress 参数通告自己的公网 IP:Port，
// ACS 通过此地址发送 UDP-CR 唤醒 CPE 发起新会话。
type STUNConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	ListenAddr   string        `mapstructure:"listen_addr"`   // UDP listen address, e.g. ":3478"
	WorkerSize   int           `mapstructure:"worker_size"`   // number of reader goroutines
	BufferSize   int           `mapstructure:"buffer_size"`   // UDP read buffer size in bytes
	CacheTTL     time.Duration `mapstructure:"cache_ttl"`     // STUN address cache TTL
	SharedSecret string        `mapstructure:"shared_secret"` // HMAC-SHA1 secret for CPE UDP CR
}

// PostSessionWakeConfig 配置会话结束后自动续唤。
// 当 TR-069 会话正常结束时，若设备命令队列仍有待执行任务，
// ACS 会延迟发送 Connection Request 立刻唤醒 CPE 发起下一次会话，
// 而不是等待下次 Periodic Inform（可能长达数分钟）。
// When a TR069 session ends with remaining commands in the queue,
// the ACS can immediately send a Connection Request to trigger a new session,
// instead of waiting for the device's next periodic Inform.
type PostSessionWakeConfig struct {
	Enabled       bool          `mapstructure:"enabled"`        // 是否启用会话结束续唤
	DelayAfter    time.Duration `mapstructure:"delay_after"`    // 会话结束后延迟多久发 CR（给 CPE 喘息时间）
	MaxContinuous int           `mapstructure:"max_continuous"` // 单设备最大连续续唤次数（防止无限循环）
	CooldownTTL   time.Duration `mapstructure:"cooldown_ttl"`   // 连续续唤冷却 TTL（过期后重置计数）
}

// UploadConfig 配置 ACS 的文件上传接收服务。
// CPE 通过 Upload RPC 将配置备份、日志、PM/MR 文件等上传到 ACS，
// ACS 接收后存入 MinIO 对应的 Bucket。
// BaseURL 对外暴露给 CPE 使用（需要 CPE 可达），Token 用于鉴权。
type UploadConfig struct {
	BaseURL     string        `mapstructure:"base_url"`      // Upload server base URL, e.g. http://acs:7547
	Path        string        `mapstructure:"path"`          // Upload path prefix, default /upload
	Username    string        `mapstructure:"username"`      // HTTP Basic Auth username for CPE upload
	Password    string        `mapstructure:"password"`      // HTTP Basic Auth password for CPE upload
	TokenSecret string        `mapstructure:"token_secret"`  // JWT signing secret (optional)
	TokenTTL    time.Duration `mapstructure:"token_ttl"`     // Token validity duration
	MaxFileSize int64         `mapstructure:"max_file_size"` // Max file size in bytes
}

// DownloadConfig 配置 ACS 的文件下载分发服务。
// ACS 通过 Download RPC 向 CPE 下发固件、配置等文件。
// 实际文件存储在 MinIO，ACS 将 MinIO 文件转换为可供 CPE 访问的 HTTP 下载链接。
// BaseURL 对外暴露，必须 CPE 可达。
type DownloadConfig struct {
	BaseURL  string `mapstructure:"base_url"` // Download server base URL (gateway), e.g. http://localhost:8080
	Path     string `mapstructure:"path"`     // Download path prefix, default /smallcell/FileDownloadService
	Username string `mapstructure:"username"` // HTTP Basic Auth username for CPE download
	Password string `mapstructure:"password"` // HTTP Basic Auth password for CPE download
}

// CORSConfig 配置 HTTP API 的 CORS 跨域策略。
// 由 App 服务的 middleware.CORS 中间件使用，
// AllowOrigins 填写前端部署域名（如 https://omc.example.com）。
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// ConnReqConfig 配置 App 服务发起 Connection Request 的地址信息。
// App 服务在下发 RPC 任务时会主动唤醒 CPE（通过 HTTP CR 或 UDP CR），
// ServerAddr 是 ACS 对外暴露的地址，写入 CPE 的 ConnectionRequestURL 字段。
type ConnReqConfig struct {
	// ServerAddr is the ACS server's externally reachable address for CPE UDP CR URL field.
	// Example: "acs.example.com:7547" or "10.0.0.1:7547"
	ServerAddr   string `mapstructure:"server_addr"`
	SharedSecret string `mapstructure:"shared_secret"` // HMAC-SHA1 secret for CPE UDP CR
}

// BatchProcessorConfig 配置周期性 Inform 的批量处理器。
// 在高并发场景下，大量设备同时发 Periodic Inform 会造成数据库写压力，
// 批量处理器将单条写入改为批量聚合后一次写入，显著降低 DB IOPS。
// 仅影响 Periodic/ValueChange 类型 Inform，Bootstrap 仍走逐条逻辑。
// When enabled, periodic Inform events are buffered and batch-flushed to DB
// at configurable intervals, reducing per-device DB write pressure.
type BatchProcessorConfig struct {
	Enabled         bool          `mapstructure:"enabled"`          // 是否启用批量处理（false 走原有逐条逻辑）
	Workers         int           `mapstructure:"workers"`          // 工作协程数（建议 = CPU 核数 / 2）
	FlushInterval   time.Duration `mapstructure:"flush_interval"`   // 批量刷新间隔（如 10s）
	MaxBatchSize    int           `mapstructure:"max_batch_size"`   // 单次批量上限
	InputBuffer     int           `mapstructure:"input_buffer"`     // 每个 worker 输入通道缓冲大小
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"` // 优雅关闭超时
}

// DictLoaderConfig 配置共享字典装载基础设施（T-0098 P1-01）。
// 由参数模型 / KPI 指标库 / 告警库 / 产品 四类启动期 Loader 共用：
//   - XMLBaseDir 是各域 Loader.Directory() 的根（相对工作目录或绝对路径）
//   - AutoLoadOnStartup=false 时跳过启动期 LoadAll，仅由管理 API 触发 Reload
//   - CacheVersionPollInterval 控制 Redis 计数器轮询周期（每实例独立 goroutine）
//   - LoadConcurrency 限制 LoadAll 内 errgroup 并发数
//
// 各域独立配置（如参数模型的 discovered_cache_ttl）仍放在各自的
// ParamModelConfig / IndicatorConfig / AlarmDefinitionConfig 中。
type DictLoaderConfig struct {
	XMLBaseDir               string        `mapstructure:"xml_base_dir"`
	AutoLoadOnStartup        bool          `mapstructure:"auto_load_on_startup"`
	CacheVersionPollInterval time.Duration `mapstructure:"cache_version_poll_interval"`
	LoadConcurrency          int           `mapstructure:"load_concurrency"`

	// T-0098 P1-06：4 域 Loader 子配置。
	// 默认值由各 Loader 包构造期 fallback（避免 yaml 缺省即崩）。
	ParamModel       ParamModelLoaderConfig       `mapstructure:"param_model"`
	Indicator        IndicatorLoaderConfig        `mapstructure:"indicator"`
	AlarmDefinition  AlarmDefinitionLoaderConfig  `mapstructure:"alarm_definition"`
	Product          ProductLoaderConfig          `mapstructure:"product"`
}

// ParamModelLoaderConfig 控制参数模型 Loader 行为（T-0098 P1-06）。
// 扫描 {XMLBaseDir}/{Directory}/，按 ParamModelFiles 白名单加载 9 个 paramModel XML，
// StandardModelFile 单文件加载 standard-model.xml。
type ParamModelLoaderConfig struct {
	Directory         string   `mapstructure:"directory"`           // 默认 "param-mappings"
	ParamModelFiles   []string `mapstructure:"param_model_files"`   // 9 个白名单（空即扫除已知 routing/products 之外）
	StandardModelFile string   `mapstructure:"standard_model_file"` // 默认 "standard-model.xml"
}

// IndicatorLoaderConfig 控制 KPI 指标库 Loader 行为（T-0098 P1-06）。
// EnbDirectory 内的所有 *.xml 视为 ENB 平台文件；GsmFile / GnbFile 单文件加载。
type IndicatorLoaderConfig struct {
	BaseDirectory string `mapstructure:"base_directory"` // 默认 "indicator-library"
	EnbSubdir     string `mapstructure:"enb_subdir"`     // 默认 "enb"
	GsmFile       string `mapstructure:"gsm_file"`       // 默认 "GSM.xml"
	GnbFile       string `mapstructure:"gnb_file"`       // 默认 "GNB.xml"
}

// AlarmDefinitionLoaderConfig 控制告警库 Loader 行为（T-0098 P1-06）。
// 扫描 {XMLBaseDir}/{Directory}/，按文件名推断 ne_type（ENB.xml → "ENB"）。
type AlarmDefinitionLoaderConfig struct {
	Directory string `mapstructure:"directory"` // 默认 "alarm-definitions"
}

// ProductLoaderConfig 控制产品装配件 Loader 行为（T-0098 P1-06）。
// File 单文件加载 products.xml；启动期校验三引用（paramModel / indicator platform / alarm ne_type）。
type ProductLoaderConfig struct {
	Directory string `mapstructure:"directory"` // 默认 "param-mappings"（与 paramModel 同目录）
	File      string `mapstructure:"file"`      // 默认 "products.xml"
}

// ParamRegistryConfig 控制 ParamRegistry 运行期行为（T-0098 P2-02）。
//
// UseNew 是 dual-stack 切换 flag：
//   - false（默认）→ 消费者（provision/sync、orchestrator、device handler、interop）继续走旧 datamodel.ParamRegistry
//   - true        → 消费者走新 parammodel.Registry + Translator（P2-02..P2-08 全部合入后才能切）
//
// 本 flag **不被 parammodel.Registry 自身消费**——新 Registry 始终可启动可测；
// flag 仅在 P2-04..08 各消费者改造时按需读取，决定走哪个栈。
//
// DefaultTTL / DiscoveredTTL 控制 L2 Redis 缓存 TTL；≤ 0 时由 RedisCache 退化为 24h。
type ParamRegistryConfig struct {
	UseNew        bool          `mapstructure:"use_new"`
	DefaultTTL    time.Duration `mapstructure:"default_ttl"`
	DiscoveredTTL time.Duration `mapstructure:"discovered_ttl"`
}

// AppConfig 是 App 服务的完整配置。
// 由 cmd/app/main.go 读取 config.yaml 后初始化，包含 REST API、认证、
// 北向接口、开站引擎、数据模型过期策略和可观测性配置。
type AppConfig struct {
	Server          AppServerConfig       `mapstructure:"server"`
	DB              PostgresConfig        `mapstructure:"db"`
	TSDB            PostgresConfig        `mapstructure:"tsdb"`
	Redis           RedisConfig           `mapstructure:"redis"`
	NATS            NATSConfig            `mapstructure:"nats"`
	MinIO           MinIOConfig           `mapstructure:"minio"`
	JWT             JWTConfig             `mapstructure:"jwt"`
	LoginCrypto     LoginCryptoConfig     `mapstructure:"login_crypto"`
	CORS            CORSConfig            `mapstructure:"cors"`
	ConnReq         ConnReqConfig         `mapstructure:"conn_req"`
	Northbound      NorthboundConfig      `mapstructure:"northbound"`
	NEDirect        NEDirectConfig        `mapstructure:"ne_direct"`
	Provision       ProvisionConfig       `mapstructure:"provision"`
	Upgrade         UpgradeConfig         `mapstructure:"upgrade"`
	DataModelExpiry DataModelExpiryConfig `mapstructure:"datamodel_expiry"`
	DictLoader      DictLoaderConfig      `mapstructure:"dict_loader"`
	ParamRegistry   ParamRegistryConfig   `mapstructure:"param_registry"`
	BatchProcessor  BatchProcessorConfig  `mapstructure:"batch_processor"`
	License         LicenseConfig         `mapstructure:"license"`
	Metrics         MetricsConfig         `mapstructure:"metrics"`
	Tracer          TracerConfig          `mapstructure:"tracer"`
	Log             LogConfig             `mapstructure:"log"`
	RequestIDPrefix string                `mapstructure:"request_id_prefix"` // 请求 ID 前缀，如 "app"
}

// LicenseConfig 配置 license 子系统的可调参数（T-0100 P4）。
//
// Signing：OEM 签名校验（P4-C）。dev 默认 strict=false + 空 PublicKeyDir，
// 等价于 P3 stub 行为；prod 推荐 strict=true + 挂载 OEM 公钥目录。
//
// LogArchive：审计日志归档 cron（P4-B）。空目录 / 0 月禁用归档；prod 推荐
// retention_months=6（PRD §5.4.5 等保 2.0 三级 8.1.4.7 合规要求）。
type LicenseConfig struct {
	Signing    LicenseSigningConfig    `mapstructure:"signing"`
	LogArchive LicenseLogArchiveConfig `mapstructure:"log_archive"`
}

// LicenseSigningConfig — OEM license 数字签名校验配置（T-0100-P4-C）。
//
// PublicKeyDir：含 *.pem 公钥文件的目录（每个文件一/多个 PEM block，仅识别
// PUBLIC KEY / RSA PUBLIC KEY 类型）。空 / 不存在 → 跳过校验，所有 license
// 落 signature_status='unverified'。
//
// Strict：true 时未签名 / 公钥未配置 / 签名无效 三种情况 import 直接 400 拒绝；
// false（默认 dev）放过仅记 warn。
type LicenseSigningConfig struct {
	PublicKeyDir string `mapstructure:"public_key_dir"`
	Strict       bool   `mapstructure:"strict"`
}

// LicenseLogArchiveConfig — license_logs 周级归档 cron 配置（T-0100-P4-B）。
//
// RetentionMonths：DB 保留月数。0 / 负 = 禁用归档；推荐 6（等保 2.0 三级合规
// 要求重要操作日志保留 ≥ 6 个月）。
//
// MinIOBucket：归档对象存储桶；空时复用 minio.buckets.logs。归档对象命名约定
// `license-logs/{YYYY-MM}.jsonl.gz`，单月聚合便于按月归档审计。
//
// Schedule：cron expression；空时默认 "0 3 * * 0"（每周日凌晨 3 点 UTC）。
type LicenseLogArchiveConfig struct {
	RetentionMonths int    `mapstructure:"retention_months"`
	MinIOBucket     string `mapstructure:"minio_bucket"`
	Schedule        string `mapstructure:"schedule"`
}

// JWTConfig 配置 JWT 认证。
// App 服务用于颁发和验证用户 Access Token / Refresh Token，
// middleware.AuthMiddleware 在每个 API 请求中进行验证。
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// LoginCryptoConfig 配置登录类接口（login / change-password / reset-password / create-user）
// 的密码 RSA-OAEP 加密传输。
//
// PrivateKeyPath：RSA 私钥 PEM 文件路径；不存在时进程启动会自动生成 2048 位密钥落盘
// （文件 0600，目录 0700）。生产部署多副本时建议把同一份私钥挂载为 K8s Secret，
// 避免每个副本独立生成导致 keyID 不一致。
type LoginCryptoConfig struct {
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

// NorthboundConfig 配置北向接口（OSS 推送）。
// 支持将 PM/Alarm/设备状态事件推送至多个上层 OSS 系统（如网管），
// 每个 PushTarget 独立配置 URL、鉴权方式、推送格式和数据类型。
type NorthboundConfig struct {
	PushTargets []PushTargetConfig `mapstructure:"push_targets"`
}

// PushTargetConfig 定义单个北向推送目标（OSS 端点）。
// 支持 Bearer/HMAC 签名鉴权，可配置数据类型过滤（pm/alarm/device），
// 支持批量发送和失败重试。
type PushTargetConfig struct {
	ID             string   `mapstructure:"id"`
	URL            string   `mapstructure:"url"`
	AuthType       string   `mapstructure:"auth_type"`
	AuthToken      string   `mapstructure:"auth_token"`
	DataTypes      []string `mapstructure:"data_types"`
	Format         string   `mapstructure:"format"`
	BatchSize      int      `mapstructure:"batch_size"`
	RetryCount     int      `mapstructure:"retry_count"`
	Enabled        bool     `mapstructure:"enabled"`
	SigningEnabled bool     `mapstructure:"signing_enabled"`
	SigningSecret  string   `mapstructure:"signing_secret"`
}

// NEDirectConfig 配置 NE Direct（网元直联）连接，当前仅 CMCC 使用。
// 部分运营商要求网管平台通过专用协议直连基站，绕过 TR-069 通道下发指令。
type NEDirectConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// ProvisionConfig 配置自动开站引擎。
// 开站引擎在 Bootstrap 触发后执行：
//  1. 匹配数据模型 → 发起 ModelUpload（FileType=11）获取 CPE 参数定义
//  2. 匹配参数模板 → 自动下发配置（AutoConfigure=true 时）
//  3. 参数同步（AutoSync）→ GPV 批量读取设备当前值存入 device_parameters 表
type ProvisionConfig struct {
	Enabled       bool              `mapstructure:"enabled"`
	AutoConfigure bool              `mapstructure:"auto_configure"` // Path A: 匹配模版后自动下发配置（需要参数路径映射层）
	TaskTimeout   time.Duration     `mapstructure:"task_timeout"`   // 超时自动 fail 非终态 task（默认 15 分钟）
	ModelUpload   ModelUploadConfig `mapstructure:"model_upload"`
	AutoSync      AutoSyncConfig    `mapstructure:"auto_sync"`
}

// UpgradeConfig 配置固件升级/回退的超时和并发策略。
// 参照 ProvisionConfig 模式，控制升级执行器的超时断线恢复和并发批次大小。
type UpgradeConfig struct {
	TaskTimeout           time.Duration `mapstructure:"task_timeout"`             // 单设备升级超时，默认 30min
	WaitDeviceReconnect   time.Duration `mapstructure:"wait_device_reconnect"`    // 等待离线设备重连，默认 1h
	WaitDownloadComplete  time.Duration `mapstructure:"wait_download_complete"`   // 等待文件下载完成，默认 10min
	WaitTransferComplete  time.Duration `mapstructure:"wait_transfer_complete"`   // 等待 TC，默认 30min
	WaitRebootComplete    time.Duration `mapstructure:"wait_reboot_complete"`     // 回退等待重启，默认 5min
	MaxConcurrentPerBatch int           `mapstructure:"max_concurrent_per_batch"` // 每批最大并发，默认 5
	ReaperInterval        time.Duration `mapstructure:"reaper_interval"`          // 超时扫描间隔，默认 2min
	UpgradeLockTTL        time.Duration `mapstructure:"upgrade_lock_ttl"`         // Redis 升级锁 TTL，默认 1h
}

// ModelUploadConfig 配置数据模型上传流程（FileType=11）。
// 当设备没有匹配的 DataModel 时，开站引擎向 CPE 发送 Upload RPC，
// 指定 FileType=11 让 CPE 将自身参数模型 XML 上传到 UploadURL。
// ACS 收到文件后通过 NATS 事件通知 App 服务解析并写入 data_model_definitions 表。
// When a device has no matching DataModel, the provision engine dispatches an Upload
// command (FileType "11") to have the CPE upload its parameter model XML.
type ModelUploadConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	UploadURL         string        `mapstructure:"upload_url"`          // ACS upload endpoint, e.g. http://acs:7547/smallcell/FileUploadService
	UploadUsername    string        `mapstructure:"upload_username"`     // HTTP Basic Auth username for CPE upload
	UploadPassword    string        `mapstructure:"upload_password"`     // HTTP Basic Auth password for CPE upload
	AutoActivateModel bool          `mapstructure:"auto_activate_model"` // Auto-activate created model
	UploadTimeout     time.Duration `mapstructure:"upload_timeout"`      // Timeout for Upload RPC (default 5min)
}

// AutoSyncConfig 配置自动参数同步策略。
// 在 Bootstrap 或固件变更后，系统自动通过 GetParameterValues RPC 批量读取
// 设备所有参数当前值，并写入 device_parameters 表，供开站对比和监控使用。
// GPVBatchSize 控制单次 GetParameterValues 请求的参数路径数量（建议 50~100）。
type AutoSyncConfig struct {
	Enabled              bool `mapstructure:"enabled"`
	SyncOnBootstrap      bool `mapstructure:"sync_on_bootstrap"`
	SyncOnFirmwareChange bool `mapstructure:"sync_on_firmware_change"`
	MaxConcurrent        int  `mapstructure:"max_concurrent"`
	GPVBatchSize         int  `mapstructure:"gpv_batch_size"`
}

// DataModelExpiryConfig 配置数据模型模板的自动过期清理策略。
// 自动发现的模板（auto_discovered）若长期无设备使用，会自动过期归档，
// 避免废弃模板占用存储和干扰匹配逻辑。CleanupCron 指定清理定时任务的调度表达式。
type DataModelExpiryConfig struct {
	AutoMaxIdleDays   int    `mapstructure:"auto_max_idle_days"`   // Max idle days for auto_discovered templates (default: 15)
	ManualMaxIdleDays int    `mapstructure:"manual_max_idle_days"` // Max idle days for manual templates (default: 60)
	CleanupCron       string `mapstructure:"cleanup_cron"`         // Cron expression for cleanup (default: "0 3 * * *")
}

// WorkerConfig 是 Worker 服务的完整配置。
// 由 cmd/worker/main.go 读取 config.yaml 后初始化。
// Worker 服务负责后台异步任务：PM 文件解析入库、MR 文件处理、
// 告警聚合/OSS 推送、定时 KPI 计算等，不对外提供 HTTP API。
type WorkerConfig struct {
	DB              PostgresConfig `mapstructure:"db"`
	TSDB            PostgresConfig `mapstructure:"tsdb"`
	Redis           RedisConfig    `mapstructure:"redis"`
	NATS            NATSConfig     `mapstructure:"nats"`
	MinIO           MinIOConfig    `mapstructure:"minio"`
	Metrics         MetricsConfig  `mapstructure:"metrics"`
	Tracer          TracerConfig   `mapstructure:"tracer"`
	Log             LogConfig      `mapstructure:"log"`
	RequestIDPrefix string         `mapstructure:"request_id_prefix"` // 请求 ID 前缀，如 "worker"
}

// ACSServerConfig 配置 ACS HTTP/HTTPS 服务器监听参数。
// Port（默认 7557）供 CPE 发送 TR-069 Inform 请求（明文）；
// TLSPort（默认 7558）供支持 HTTPS 的 CPE 使用（可选）。
// 超时参数直接影响 TR-069 会话的 HTTP 层行为，需与 SessionConfig 协调设置。
type ACSServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	TLSPort      int           `mapstructure:"tls_port"`
	TLS          TLSConfig     `mapstructure:"tls"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// AppServerConfig 配置 App 服务的 HTTP/gRPC 监听参数。
// Port 提供 REST API（供前端和第三方集成）；
// GRPCPort 提供 gRPC 接口（如与其他微服务通信）；
// TLSPort 提供 HTTPS REST API（生产环境推荐启用）。
type AppServerConfig struct {
	Host     string    `mapstructure:"host"`
	Port     int       `mapstructure:"port"`
	TLSPort  int       `mapstructure:"tls_port"`
	GRPCPort int       `mapstructure:"grpc_port"`
	TLS      TLSConfig `mapstructure:"tls"`
}

// TLSConfig 配置 TLS 证书文件路径。
// CertFile 和 KeyFile 为 PEM 格式证书和私钥，
// 由 ACS 或 App 服务在启动 HTTPS 监听时加载。
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// SessionConfig 配置 ACS TR-069 会话行为。
// Timeout 是单次会话最大持续时间，超时后强制关闭防止僵死连接。
// MaxConcurrent 限制并发会话数，防止大规模设备同时上线时 OOM。
// MaxRPCPerSession 限制单次会话内 RPC 交互轮数，防止 Session 无限循环。
type SessionConfig struct {
	Timeout          time.Duration `mapstructure:"timeout"`
	MaxConcurrent    int64         `mapstructure:"max_concurrent"`
	MaxRPCPerSession int           `mapstructure:"max_rpc_per_session"` // 单次会话最大 RPC 交互次数（0=不限制，建议 ≤15）
}

// RateLimitConfig 配置 CPE Inform 请求的限流策略。
// 使用令牌桶算法（token bucket）按设备 SN 限流，防止单台设备频繁 Inform 压垮系统。
// PerDevice 是每分钟允许通过的 Inform 数，Burst 是突发容量。
// MaxDevices 限制追踪的最大设备数，超出后旧设备的令牌桶会被 LRU 淘汰。
type RateLimitConfig struct {
	PerDevice       int           `mapstructure:"per_device"`       // 每设备每分钟最大 Inform 数
	Burst           int           `mapstructure:"burst"`            // token bucket 突发容量
	MaxDevices      int           `mapstructure:"max_devices"`      // 限流器追踪的最大设备数
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"` // 清理扫描间隔
	CleanupTimeout  time.Duration `mapstructure:"cleanup_timeout"`  // 设备不活跃淘汰超时
}

// AuthConfig 配置 CPE 鉴权方式。
// Mode 可选 digest（TR-069 标准 HTTP Digest）、basic（HTTP Basic）或 none（不鉴权）。
// 生产环境强烈建议使用 digest，防止中间人攻击。
type AuthConfig struct {
	Mode     string `mapstructure:"mode"` // digest, basic, none
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// PostgresConfig 配置 PostgreSQL 连接池。
// PgPool 为主库（设备/告警/配置），TsPool 为 TimescaleDB（PM 超表/KPI 超表）。
// LogSQL/LogSQLParams 仅用于调试，生产环境建议关闭以避免日志膨胀。
// LogSQLSlowThreshold 单位毫秒，设为 0 禁用慢查询日志。
type PostgresConfig struct {
	DSN                 string        `mapstructure:"dsn"`
	MaxConns            int32         `mapstructure:"max_conns"`
	MinConns            int32         `mapstructure:"min_conns"`
	MaxConnLifetime     time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime     time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	// SQL 日志配置
	LogSQL              bool `mapstructure:"log_sql"`                // 是否记录 SQL
	LogSQLParams        bool `mapstructure:"log_sql_params"`         // 是否记录参数
	LogSQLSlowThreshold int  `mapstructure:"log_sql_slow_threshold"` // 慢查询阈值(ms)
}

// RedisConfig 配置 Redis 连接。
// 用于：设备缓存（Cache-Aside，TTL 10 分钟）、限流令牌桶、
// 会话状态、续唤计数等。
// 支持单机/Cluster 模式（Addrs 填多个地址时自动使用 ClusterClient）。
type RedisConfig struct {
	Addrs    []string `mapstructure:"addrs"`
	Password string   `mapstructure:"password"`
	DB       int      `mapstructure:"db"`
	PoolSize int      `mapstructure:"pool_size"`
}

// NATSConfig 配置 NATS JetStream 连接。
// NATS 是三个微服务（acs/app/worker）之间的事件总线，
// 用于跨服务异步通信（如 ACS 发布 device.inform.bootstrap，App 订阅开站）。
// MaxReconnect=-1 表示无限重连，建议生产环境启用。
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	MaxReconnect  int           `mapstructure:"max_reconnect"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	// AllowStreamRebuild 控制 EnsureStreams 检测到现有 stream 的 Retention
	// 与 DefaultStreams 不一致时是否自动删除重建。生产默认 false（仅 WARN，
	// 由运维手动处理）；dev/test 可设 true，重启即按新 retention 重建。
	AllowStreamRebuild bool `mapstructure:"allow_stream_rebuild"`
}

// MinIOConfig 配置 MinIO 对象存储连接。
// MinIO 用于存储所有大文件：PM/MR 原始文件、固件镜像、
// 配置备份、设备日志、数据模型 XML、北向报表等。
// Buckets 定义不同类型文件使用的 Bucket 名称。
type MinIOConfig struct {
	Endpoint  string       `mapstructure:"endpoint"`
	AccessKey string       `mapstructure:"access_key"`
	SecretKey string       `mapstructure:"secret_key"`
	UseSSL    bool         `mapstructure:"use_ssl"`
	Buckets   BucketConfig `mapstructure:"buckets"`
}

// BucketConfig 定义各类文件在 MinIO 中的 Bucket 分配。
// 各 Bucket 用途：
//   - PMFiles：性能管理原始 XML 文件
//   - MRFiles：Measurement Report 原始文件
//   - Firmware：固件镜像和配置模板下发
//   - ConfigBackup：设备配置备份、SSL 证书
//   - Logs：设备运行日志、安全日志、故障日志、PCAP 抓包
//   - Reports：北向报表导出
//   - Exchange：数据模型 XML、导入导出中间文件
//   - UIAssets：UI 定制化品牌资产（Logo / 登录背景图等，参 docs/prd/system/ui-customization.md）
type BucketConfig struct {
	PMFiles      string `mapstructure:"pm_files"`
	MRFiles      string `mapstructure:"mr_files"`
	Firmware     string `mapstructure:"firmware"`
	ConfigBackup string `mapstructure:"config_backup"`
	Logs         string `mapstructure:"logs"`
	Reports      string `mapstructure:"reports"`
	Exchange     string `mapstructure:"exchange"`
	UIAssets     string `mapstructure:"ui_assets"`
}

// MetricsConfig 配置 Prometheus 指标暴露端口。
// 各服务在此端口提供 /metrics 端点，供 Prometheus 采集。
// 同一端口也提供 /healthz 健康检查接口（由 HealthChecker 驱动）。
type MetricsConfig struct {
	Port int `mapstructure:"port"`
}

// TracerConfig 配置 OpenTelemetry 分布式链路追踪。
// Endpoint 为 OTLP gRPC 接收端（如 Jaeger/Tempo 的 4317 端口）。
// SampleRate 范围 [0.0, 1.0]：1.0=全采样，生产环境建议 0.1～0.3。
// Enabled=false 时使用 no-op Provider，零开销。
type TracerConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

// LogConfig 配置结构化日志输出。
// Format=json 用于生产（机器可读），Format=console 用于开发（人类可读）。
// OutputPaths 支持 stdout 和文件路径混合，如 ["stdout", "/var/log/omcgo/app.log"]。
// Rotation 配置日志文件轮转，避免日志文件无限增长。
type LogConfig struct {
	Level       string         `mapstructure:"level"`
	Format      string         `mapstructure:"format"`       // json, console
	OutputPaths []string       `mapstructure:"output_paths"` // output destinations, e.g. ["stdout", "/var/log/omcgo/app.log"]
	Rotation    RotationConfig `mapstructure:"rotation"`     // log rotation settings
}

// RotationConfig 配置日志文件轮转策略（基于 lumberjack）。
// MaxSizeMB：文件超过此大小触发轮转；MaxAgeDays：保留最近 N 天日志；
// MaxBackups：保留最多 N 个历史文件；Compress：历史文件是否 gzip 压缩。
// RotateInterval 支持按时间强制轮转（如每 5 分钟），适用于高频日志场景。
type RotationConfig struct {
	Enabled        bool          `mapstructure:"enabled"`         // enable log rotation
	MaxSizeMB      int           `mapstructure:"max_size_mb"`     // max size in MB before rotation (default: 20)
	MaxAgeDays     int           `mapstructure:"max_age_days"`    // max days to retain old log files (default: 7)
	MaxBackups     int           `mapstructure:"max_backups"`     // max number of old log files to retain (default: 100)
	Compress       bool          `mapstructure:"compress"`        // compress rotated files
	LocalTime      bool          `mapstructure:"local_time"`      // use local time for rotation
	RotateInterval time.Duration `mapstructure:"rotate_interval"` // time-based rotation interval (e.g., "5m" for 5 minutes)
}

// ProtocolLogConfig 配置 ACS 协议交互原始日志（独立于结构化日志）。
// 启用后将每次 CPE↔ACS 的完整 HTTP 请求/响应 SOAP XML 记录到专用文件，
// 方便协议调试和互操作测试（interop）。生产环境建议按需开启，文件可能很大。
// MaxBodySize 限制单条 XML 记录的最大字节数，超长时截断。
// When enabled, a dedicated log file records the complete raw SOAP/XML
// for every ACS-CPE HTTP request/response exchange.
type ProtocolLogConfig struct {
	Enabled     bool           `mapstructure:"enabled"`       // 总开关
	FilePath    string         `mapstructure:"file_path"`     // 日志文件路径，如 /run/logs/acs/protocol.log
	MaxBodySize int            `mapstructure:"max_body_size"` // XML 截断阈值 bytes，0=不截断
	Rotation    RotationConfig `mapstructure:"rotation"`      // 轮转配置（复用 RotationConfig）
}

// Load 从 YAML 配置文件加载配置，并支持通过环境变量覆盖（前缀 OMCGO_，"."替换为"_"）。
// 例如 OMCGO_SERVER_PORT=8080 会覆盖 server.port 字段。
// 各微服务入口直接调用，如：appconfig.Load("etc/config.dev.yaml", &cfg)
func Load(path string, target interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("OMCGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate if target implements Validatable
	if v, ok := target.(Validatable); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// LoadWithEnvOverride 在 Load 基础上额外支持显式指定的环境变量映射表。
// 适用于 Docker/K8s 部署，通过 envOverrides 将容器环境变量
// 映射到配置路径，如 {"db.dsn": "DATABASE_URL"}。
// 与自动 OMCGO_ 前缀映射互补，可处理第三方惯例的环境变量名称。
// This is useful for Docker deployments where config values need to be set via env vars.
func LoadWithEnvOverride(path string, target interface{}, envOverrides map[string]string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("OMCGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	// Apply environment variable overrides
	for key, envVar := range envOverrides {
		if val := os.Getenv(envVar); val != "" {
			v.Set(key, val)
		}
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate if target implements Validatable
	if v, ok := target.(Validatable); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}
