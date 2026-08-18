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
	"path/filepath"
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
	ParamSync               ParamSyncConfig       `mapstructure:"param_sync"`
	Task                    TaskConfig            `mapstructure:"task"`
	DB                      PostgresConfig        `mapstructure:"db"`
	TSDB                    PostgresConfig        `mapstructure:"tsdb"` // KPI/时序库物理分离：ACS 写 trace_messages（已迁时序库）所需的第二个连接池
	MinIO                   MinIOConfig           `mapstructure:"minio"`
	Upload                  UploadConfig          `mapstructure:"upload"`
	Backpressure            BackpressureConfig    `mapstructure:"backpressure"`
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
	PMDedupTTL  time.Duration `mapstructure:"pm_dedup_ttl"`  // PM upload event deduplication TTL
}

// EffectivePMDedupTTL 返回 PM 上传事件去重窗口。四小时覆盖正常上报重试和短时
// NATS 重投，同时避免每台设备的 24 小时历史键长期占用 Redis。
func (c UploadConfig) EffectivePMDedupTTL() time.Duration {
	if c.PMDedupTTL <= 0 {
		return 4 * time.Hour
	}
	return c.PMDedupTTL
}

// BackpressureConfig supplies deployment defaults for queue-risk admission.
// Runtime sys_configs may override these values without restarting ACS.
type BackpressureConfig struct {
	QueuePendingHigh int           `mapstructure:"queue_pending_high"`
	QueuePendingLow  int           `mapstructure:"queue_pending_low"`
	QueueOldestHigh  time.Duration `mapstructure:"queue_oldest_high"`
	QueueOldestLow   time.Duration `mapstructure:"queue_oldest_low"`
	QueueSlopeWindow time.Duration `mapstructure:"queue_slope_window"`
}

// Defaults preserves safe queue hysteresis when older configuration files do
// not yet contain a backpressure section.
func (c BackpressureConfig) Defaults() BackpressureConfig {
	if c.QueuePendingHigh <= 0 {
		c.QueuePendingHigh = 5000
	}
	if c.QueuePendingLow <= 0 {
		c.QueuePendingLow = 1000
	}
	if c.QueuePendingLow > c.QueuePendingHigh {
		c.QueuePendingLow = c.QueuePendingHigh
	}
	if c.QueueOldestHigh <= 0 {
		c.QueueOldestHigh = 10 * time.Minute
	}
	if c.QueueOldestLow <= 0 {
		c.QueueOldestLow = 2 * time.Minute
	}
	if c.QueueOldestLow > c.QueueOldestHigh {
		c.QueueOldestLow = c.QueueOldestHigh
	}
	if c.QueueSlopeWindow <= 0 {
		c.QueueSlopeWindow = 5 * time.Minute
	}
	return c
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
	ParamModel      ParamModelLoaderConfig      `mapstructure:"param_model"`
	Indicator       IndicatorLoaderConfig       `mapstructure:"indicator"`
	AlarmDefinition AlarmDefinitionLoaderConfig `mapstructure:"alarm_definition"`
	Product         ProductLoaderConfig         `mapstructure:"product"`
	QuickSettings   QuickSettingsLoaderConfig   `mapstructure:"quick_settings"`
}

// ParamModelLoaderConfig 控制参数模型 Loader 行为（T-0098 P1-06）。
// 扫描 {XMLBaseDir}/{Directory}/，按 ParamModelFiles 白名单加载 9 个 paramModel XML，
// StandardModelFile 单文件加载 standard-model.xml。
type ParamModelLoaderConfig struct {
	Directory         string   `mapstructure:"directory"`           // 默认 "param-mappings"
	ParamModelFiles   []string `mapstructure:"param_model_files"`   // 9 个白名单（空即扫除已知 routing/products 之外）
	StandardModelFile string   `mapstructure:"standard_model_file"` // 默认 "standard-model.xml"

	// 三库 XML 导入重构(2026-06-04 D3/D5/D6):取消 builtin/custom 双目录区分,
	// 所有 XML(出厂 + 用户上传)同住 Directory(param-mappings/),来源由同目录 sidecar
	// (X.xml.custom)判定。CustomDirectory / CustomOverrides 已删除。

	// worker BackupCleanup cron(PRD §9.6):
	//   - .deleted.<ts> / .bak.<ts> 文件超过 N 天即清理(默认 30)
	//   - .tmp.<uuid> 文件超过 1 小时即清理(写盘中断残留)
	//   - 孤儿 sidecar(X.xml.custom 而 X.xml 已不存在)即清
	// cron 表达式默认 "0 3 * * *"(每天凌晨 3 点)。
	BackupRetentionDays int    `mapstructure:"backup_retention_days"`
	BackupCleanupCron   string `mapstructure:"backup_cleanup_cron"`
}

// IndicatorLoaderConfig 控制 KPI 指标库 Loader 行为（T-0098 P1-06）。
// EnbSubdir 内的所有 *.xml 视为 ENB 平台文件；GsmFile / GnbFile 根级单文件加载，
// 同制式子目录 gsm/ / gnb/ 内的 *.xml 一并加载。
//
// 三库 XML 导入重构(2026-06-04 D3/D5/D6):取消 builtin/custom 双目录区分,
// 所有 XML(出厂 + 用户上传)同住 BaseDirectory 目录树(enb/gsm/gnb 子目录 +
// GSM.xml / GNB.xml 根级单文件),来源由同目录 sidecar(X.xml.custom)判定。
// Custom* / CustomOverrides 已删除。
type IndicatorLoaderConfig struct {
	BaseDirectory string `mapstructure:"base_directory"` // 默认 "indicator-library"
	EnbSubdir     string `mapstructure:"enb_subdir"`     // 默认 "enb"
	GsmFile       string `mapstructure:"gsm_file"`       // 默认 "GSM.xml"
	GnbFile       string `mapstructure:"gnb_file"`       // 默认 "GNB.xml"

	// worker BackupCleanup cron(PRD §9.6,对标 T-0178 ParamModel):
	//   - .deleted.<ts> / .bak.<ts> 文件超过 N 天即清理(默认 30)
	//   - .tmp.<uuid> 文件超过 1 小时即清理(写盘中断残留)
	//   - 孤儿 sidecar(X.xml.custom 而 X.xml 已不存在)即清
	// cron 表达式默认 "0 3 * * *"(每天凌晨 3 点;与 ParamModel cron 错峰)。
	BackupRetentionDays int    `mapstructure:"backup_retention_days"`
	BackupCleanupCron   string `mapstructure:"backup_cleanup_cron"`
}

// AlarmDefinitionLoaderConfig 控制告警库 Loader 行为（T-0098 P1-06）。
// 扫描 {XMLBaseDir}/{Directory}/，按 ne_type 推断（XML neType 属性优先，缺省回退文件名）。
//
// 三库 XML 导入重构(2026-06-04 D3/D5/D6):取消 builtin/custom 双目录区分，
// 所有 XML(出厂 + 用户上传)同住 Directory(alarm-definitions/，扁平结构，每个 ne_type 一个
// XML 文件)，来源由同目录 sidecar(X.xml.custom)判定。CustomDirectory / CustomOverrides 已删除。
type AlarmDefinitionLoaderConfig struct {
	Directory string `mapstructure:"directory"` // 默认 "alarm-definitions"

	// worker BackupCleanup cron：
	//   - .deleted.<ts> / .bak.<ts> 超过 N 天即清理（默认 30）
	//   - .tmp.<uuid> 超过 1 小时即清理（写盘中断残留）
	//   - 孤儿 sidecar(X.xml.custom 而 X.xml 已不存在)即清
	BackupRetentionDays int    `mapstructure:"backup_retention_days"`
	BackupCleanupCron   string `mapstructure:"backup_cleanup_cron"`
}

// ProductLoaderConfig 控制产品装配件 Loader 行为（T-0098 P1-06）。
// File 单文件加载 products.xml；启动期校验三引用（paramModel / indicator platform / alarm ne_type）。
type ProductLoaderConfig struct {
	Directory string `mapstructure:"directory"` // 默认 "param-mappings"（与 paramModel 同目录）
	File      string `mapstructure:"file"`      // 默认 "products.xml"
}

// QuickSettingsLoaderConfig 控制「快速设置」分组 Loader 行为（T-0138）。
// 扫描 {XMLBaseDir}/{Directory}/{enb.xml,gnb.xml}，加载到进程内存 Registry，
// 不写 DB；REST 端点 GET /api/v1/quicksettings/groups?tech={lte|nr} 直接读 Registry。
type QuickSettingsLoaderConfig struct {
	Directory string `mapstructure:"directory"` // 默认 "quicksettings"
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

// ParamSyncConfig controls the reliable parameter-sync data plane.
type ParamSyncConfig struct {
	ManualOfflineMode             string        `mapstructure:"manual_offline_mode"`
	ResultConsumerShardCount      int           `mapstructure:"result_consumer_shard_count"`
	ResultConsumerQueueDepth      int           `mapstructure:"result_consumer_queue_depth"`
	ResultConsumerPullBatchSize   int           `mapstructure:"result_consumer_pull_batch_size"`
	ResultConsumerPullConcurrency int           `mapstructure:"result_consumer_pull_concurrency"`
	ResultConsumerAckWait         time.Duration `mapstructure:"result_consumer_ack_wait"`
	ResultConsumerMaxAckPending   int           `mapstructure:"result_consumer_max_ack_pending"`
	RecoveryRunLimit              int           `mapstructure:"recovery_run_limit"`
	RecoveryTaskLimitPerRun       int           `mapstructure:"recovery_task_limit_per_run"`
	RecoveryTaskBudget            int           `mapstructure:"recovery_task_budget"`
}

// AppConfig 是 App 服务的完整配置。
// 由 cmd/app/main.go 读取 config.yaml 后初始化，包含 REST API、认证、
// 北向接口、开站引擎、数据模型过期策略和可观测性配置。
type AppConfig struct {
	Server          AppServerConfig       `mapstructure:"server"`
	DB              PostgresConfig        `mapstructure:"db"`
	TSDB            PostgresConfig        `mapstructure:"tsdb"`
	Redis           RedisConfig           `mapstructure:"redis"`
	PMRedis         RedisConfig           `mapstructure:"pm_redis"`
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
	Topology        TopologyConfig        `mapstructure:"topology"`
	DataModelExpiry DataModelExpiryConfig `mapstructure:"datamodel_expiry"`
	DictLoader      DictLoaderConfig      `mapstructure:"dict_loader"`
	ParamRegistry   ParamRegistryConfig   `mapstructure:"param_registry"`
	ParamSync       ParamSyncConfig       `mapstructure:"param_sync"`
	BatchProcessor  BatchProcessorConfig  `mapstructure:"batch_processor"`
	License         LicenseConfig         `mapstructure:"license"`
	Task            TaskConfig            `mapstructure:"task"`
	MR              MRConfig              `mapstructure:"mr"`
	Notification    NotificationConfig    `mapstructure:"notification"`
	Dashboard       DashboardConfig       `mapstructure:"dashboard"`
	Metrics         MetricsConfig         `mapstructure:"metrics"`
	Tracer          TracerConfig          `mapstructure:"tracer"`
	Log             LogConfig             `mapstructure:"log"`
	RequestIDPrefix string                `mapstructure:"request_id_prefix"` // 请求 ID 前缀，如 "app"
}

type DashboardConfig struct {
	QueryTimeout     time.Duration `mapstructure:"query_timeout"`
	StatementTimeout time.Duration `mapstructure:"statement_timeout"`
	MaxConcurrent    int           `mapstructure:"max_concurrent"`
	QueueTimeout     time.Duration `mapstructure:"queue_timeout"`
	FreshCacheTTL    time.Duration `mapstructure:"fresh_cache_ttl"`
	StaleTTL         time.Duration `mapstructure:"stale_ttl"`
}

func (c DashboardConfig) Defaults() DashboardConfig {
	if c.QueryTimeout <= 0 {
		c.QueryTimeout = 3 * time.Second
	}
	if c.StatementTimeout <= 0 {
		c.StatementTimeout = 2500 * time.Millisecond
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 4
	}
	if c.QueueTimeout <= 0 {
		c.QueueTimeout = 100 * time.Millisecond
	}
	if c.FreshCacheTTL <= 0 {
		c.FreshCacheTTL = 4*time.Minute + 30*time.Second
	}
	if c.StaleTTL <= 0 {
		c.StaleTTL = 15 * time.Minute
	}
	return c
}

// TaskConfig 配置 task 子系统的全局默认行为（T-0157 C1 引入）。
//
// DefaultExpiresInSeconds: device task 创建调用方未显式传 ExpiresIn 时使用的默认超时秒数。
//
//	调用方语义:
//	  - req.ExpiresIn > 0  → 直接采用该值
//	  - req.ExpiresIn == 0 → 用本配置默认值兜底；本配置 <= 0 时表示"永不超时"
//	实测 CPE 应答落在 5-60 秒区间，默认 120s 给慢响应留两倍缓冲又不让用户傻等。
//	各业务（ops/alarm）可在 CreateTaskRequest 中显式覆盖（如告警同步 600s）。
//
// SweepIntervalSeconds: worker 进程 task_sweeper 周期扫描过期 task 的间隔（秒）。
//
//	<= 0 时 sweeper 不启动（C2 引入；C1 阶段先建配置项占位）。
//
// WakeConcurrency: wakeDevice 异步 Connection Request 唤醒的并发上界（issue #12）。
//
//	防 10 万 Inform 风暴 / 批量任务下唤醒 goroutine 无界暴涨 → 峰值 OOM。
//	<= 0 时 TaskService 回退到内置安全默认（defaultWakeConcurrency=256）。
//	满载时多余唤醒被背压丢弃（设备下个 periodic inform 自愈），不阻塞 CreateTask。
//
// ReconcileIntervalSeconds: worker 进程 task_reconciler 周期对账 Redis↔PG 状态分叉的间隔（秒，#13）。
//
//	<= 0 时 reconciler 不启动。修复 "PG 滞后于 Redis 终态" 的孤儿/陈旧记录。
//
// TerminalRedisTTL: Redis 终态 task Hash 的短期保留窗口。
//
//	默认 15 分钟，覆盖 ACS 5 分钟会话迟到响应和 worker 60 秒对账宽限；
//	小于 10 分钟会钳到 10 分钟。pending/sent/待重试仍保留 4 小时。
type TaskConfig struct {
	DefaultExpiresInSeconds  int           `mapstructure:"default_expires_in_seconds"`
	SweepIntervalSeconds     int           `mapstructure:"sweep_interval_seconds"`
	WakeConcurrency          int           `mapstructure:"wake_concurrency"`
	ReconcileIntervalSeconds int           `mapstructure:"reconcile_interval_seconds"`
	TerminalRedisTTL         time.Duration `mapstructure:"terminal_redis_ttl"`
}

// EffectiveTerminalRedisTTL 返回终态 task Hash 的安全保留窗口。
func (c TaskConfig) EffectiveTerminalRedisTTL() time.Duration {
	if c.TerminalRedisTTL <= 0 {
		return 15 * time.Minute
	}
	if c.TerminalRedisTTL < 10*time.Minute {
		return 10 * time.Minute
	}
	return c.TerminalRedisTTL
}

// EffectiveTerminalRedisTTLFor additionally enforces the runtime safety
// window required by the ACS session and the task reconciler. It prevents a
// configuration change from expiring a terminal tombstone while the matching
// device session or compensation grace period can still be active.
func (c TaskConfig) EffectiveTerminalRedisTTLFor(
	sessionTimeout, reconcilerGrace time.Duration,
) time.Duration {
	effective := c.EffectiveTerminalRedisTTL()
	required := sessionTimeout + reconcilerGrace
	if required > effective {
		return required
	}
	return effective
}

// OfflineAlarmCleanupConfig 配置离线设备活动告警清理器（worker 进程的 OfflineAlarmCleaner）。
//
// 解决 issue #358：原阈值硬编码 1h（DefaultOfflineAlarmCleanupThreshold），运营商无法把告警
// 收敛阈值调到分钟级，与已可配的设备离线检测阈值（offline_threshold.go，issue #203）口径不一致。
// 三个字段 <=0 时各自回退到 alarm 包内的 Default* 常量（threshold 1h / interval 5min / batch 200），
// 保持向后兼容；yaml 显式赋值即生效，worker 启动期 SetThreshold/SetInterval/SetBatchSize 注入。
type OfflineAlarmCleanupConfig struct {
	// ThresholdSeconds 设备离线满多少秒仍未恢复，则把其当前活动告警归档到历史告警。
	// <=0 时回退到 alarm.DefaultOfflineAlarmCleanupThreshold（3600s）。
	ThresholdSeconds int `mapstructure:"threshold_seconds"`
	// IntervalSeconds 后台扫描周期（秒）。<=0 时回退到 alarm.DefaultOfflineAlarmCleanupInterval（300s）。
	IntervalSeconds int `mapstructure:"interval_seconds"`
	// BatchSize 每轮扫描最多处理的离线设备数。<=0 时回退到 alarm.DefaultOfflineAlarmCleanupBatchSize（200）。
	BatchSize int `mapstructure:"batch_size"`
}

// MRConfig 配置 F05 MR 测量任务（PRD docs/project/prd/F05-mr-task-management.md）。
//
// 字段语义对齐 MR_Feature_Analysis.md：
//   - Vendor / OmcName：SPV 下发 Device.FAP.MRMgmt.Config.{i}.Vendor / OmcName 的值
//   - URLBase：MR 文件上报 URL 前缀（不含 ?fileType=... 部分）；空时用 OMC 主地址
//   - IndependentEnable：是否使用独立 MR 服务器（true → URLBase；false → OMC 主地址）
//   - SchedulerIntervalSeconds：worker 调度器扫描间隔，默认 30s
//   - HeartbeatMissThreshold：连续未命中心跳次数，超过则标 abnormal，默认 2
//
// #798：原 FileSaveDays（MR 文件保留天数，部署配置静态值，改值需重启进程）已删除。
// MR 文件保留天数并入 MinIO 原始件 ILM（sys_configs minio.retention.raw_object_days，
// 界面可配、保存即热加载），internal/mr/task.Cleaner 动态读取该配置计算 mr_files 表清理
// cutoff，不再有独立的部署期 MR 专属保留天数配置。
//
// 取消原 sys_settings 表方案（项目无此表），改为 yaml + 环境变量驱动。
type MRConfig struct {
	Vendor                   string `mapstructure:"vendor"`
	OmcName                  string `mapstructure:"omc_name"`
	URLBase                  string `mapstructure:"url_base"`
	IndependentEnable        bool   `mapstructure:"independent_enable"`
	SchedulerIntervalSeconds int    `mapstructure:"scheduler_interval_seconds"`
	HeartbeatMissThreshold   int    `mapstructure:"heartbeat_miss_threshold"`
}

// Defaults returns sensible defaults when yaml lacks the mr section.
// 调用方在 service/dispatcher 构造时用本方法兜底。
func (c MRConfig) Defaults() MRConfig {
	out := c
	if out.Vendor == "" {
		out.Vendor = "Baicells"
	}
	if out.OmcName == "" {
		out.OmcName = "Baicells-OMC"
	}
	if out.SchedulerIntervalSeconds <= 0 {
		out.SchedulerIntervalSeconds = 30
	}
	if out.HeartbeatMissThreshold <= 0 {
		out.HeartbeatMissThreshold = 2
	}
	return out
}

// NotificationConfig 配置通知中心（T-0152）：SMTP 邮件发送器 + Alertmanager
// 告警 webhook 入口。SMTP 默认 disabled，部署期配好邮件服务器后再启用。
type NotificationConfig struct {
	SMTP         SMTPConfig         `mapstructure:"smtp"`
	AlertWebhook AlertWebhookConfig `mapstructure:"alert_webhook"`
}

// SMTPConfig 配置 SMTP 邮件发送。Username 为空表示不做 SMTP AUTH；
// StartTLS 由 SMTP 服务器能力决定。
type SMTPConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Host     string        `mapstructure:"host"`
	Port     int           `mapstructure:"port"`
	Username string        `mapstructure:"username"`
	Password string        `mapstructure:"password"`
	From     string        `mapstructure:"from"`
	StartTLS bool          `mapstructure:"starttls"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// AlertWebhookConfig 配置 Alertmanager → POST /api/v1/alerts/webhook 入口。
// Token 非空时校验 Bearer；Recipients 为告警邮件收件人。
type AlertWebhookConfig struct {
	Token      string   `mapstructure:"token"`
	Recipients []string `mapstructure:"recipients"`
}

// LicenseConfig 配置旧项目 License 子系统的可调参数。
//
// LogArchive：审计日志归档 cron（P4-B）。空目录 / 0 月禁用归档；prod 推荐
// retention_months=6（PRD §5.4.5 等保 2.0 三级 8.1.4.7 合规要求）。
type LicenseConfig struct {
	Signing    LicenseSigningConfig    `mapstructure:"signing"`
	LogArchive LicenseLogArchiveConfig `mapstructure:"log_archive"`
}

// LicenseSigningConfig 配置旧项目 TrueLicense 的 JKS/DSA 验签。
type LicenseSigningConfig struct {
	LegacyKeyStorePath    string `mapstructure:"legacy_keystore_path"`
	LegacyStorePassword   string `mapstructure:"legacy_store_password"`
	LegacyKeyAlias        string `mapstructure:"legacy_key_alias"`
	LegacyVerifyIntegrity bool   `mapstructure:"legacy_verify_integrity"`
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
//
// AllowPlaintext (T-0120)：是否接受明文密码登录 / 改密。默认 false。
// 启用后前端在非 secure context（http://内网IP 之类 crypto.subtle 不可用场景）
// 可 fallback 发明文 password，后端正常处理 + audit log 标记 reason=plaintext_login。
// **仅推荐内网部署 + 完整 audit 闭环时启用**；公网 / 多租户部署应保持 false 强制
// TLS 走加密路径（参 deployments/docker/TLS-SETUP.md 自签证书快速启用）。
type LoginCryptoConfig struct {
	PrivateKeyPath string `mapstructure:"private_key_path"`
	AllowPlaintext bool   `mapstructure:"allow_plaintext"`
}

// NorthboundConfig 配置北向接口（OSS 推送）。
// 支持将 PM/Alarm/设备状态事件推送至多个上层 OSS 系统（如网管），
// 每个 PushTarget 独立配置 URL、鉴权方式、推送格式和数据类型。
type NorthboundConfig struct {
	PushTargets []PushTargetConfig `mapstructure:"push_targets"`
	// RateLimitPerEndpoint 北向导出/同步端点每分钟最大请求数；<=0 时取默认 300。
	RateLimitPerEndpoint int `mapstructure:"rate_limit_per_endpoint"`
}

// EndpointRateLimit 返回北向 per-endpoint 限流阈值（每分钟），缺省回落到 300。
func (c NorthboundConfig) EndpointRateLimit() int {
	if c.RateLimitPerEndpoint <= 0 {
		return 300
	}
	return c.RateLimitPerEndpoint
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
//
// 安全：NE Direct 是面向运维管理员的内部接口，默认关闭（Enabled=false）。
// 启用后所有端点强制 JWT/API-Key 认证 + per-endpoint/per-device 固定窗口限流 +
// 审计日志（见 internal/nedirect/middleware.go）。
type NEDirectConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	// RateLimitPerEndpoint 每个端点每分钟最大请求数；<=0 时取默认 600。
	RateLimitPerEndpoint int `mapstructure:"rate_limit_per_endpoint"`
	// RateLimitPerDevice 每个设备每分钟最大请求数；<=0 时取默认 120。
	RateLimitPerDevice int `mapstructure:"rate_limit_per_device"`
}

// EndpointRateLimit 返回端点级限流阈值（每分钟），缺省回落到 600。
func (c NEDirectConfig) EndpointRateLimit() int {
	if c.RateLimitPerEndpoint <= 0 {
		return 600
	}
	return c.RateLimitPerEndpoint
}

// DeviceRateLimit 返回设备级限流阈值（每分钟），缺省回落到 120。
func (c NEDirectConfig) DeviceRateLimit() int {
	if c.RateLimitPerDevice <= 0 {
		return 120
	}
	return c.RateLimitPerDevice
}

// ProvisionConfig 配置自动开站引擎。
// 开站引擎在 Bootstrap 触发后执行：
//  1. 匹配数据模型 → 发起 ModelUpload（FileType=11）获取 CPE 参数定义
//  2. 匹配参数模板 → 自动下发配置（AutoConfigure=true 时）
//  3. 参数同步（AutoSync）→ GPV 批量读取设备当前值存入 device_parameters 表
type ProvisionConfig struct {
	Enabled       bool                      `mapstructure:"enabled"`
	AutoConfigure bool                      `mapstructure:"auto_configure"` // Path A: 匹配模版后自动下发配置（需要参数路径映射层）
	TaskTimeout   time.Duration             `mapstructure:"task_timeout"`   // 超时自动 fail 非终态 task（默认 15 分钟）
	ModelUpload   ModelUploadConfig         `mapstructure:"model_upload"`
	AutoSync      AutoSyncConfig            `mapstructure:"auto_sync"`
	GPVResponse   GPVResponseConsumerConfig `mapstructure:"gpv_response"`
	// 周期性参数同步配置从 sys_configs (category='device') 读，不再走 YAML。
	// 参 internal/provision/periodic_sync_policy.go。
}

// GPVResponseConsumerConfig controls the two independent consumers of
// command.get_parameters.response. ProvisionQueue remains the base name of the
// established pull durable ("-pull" is appended by the event bus). RPCDurable
// is a fixed push durable; RPCStartSequence is used only for a one-time
// handoff, and must be the predecessor consumer's captured AckFloor+1.
type GPVResponseConsumerConfig struct {
	ProvisionQueue       string        `mapstructure:"provision_queue"`
	ProvisionConcurrency int           `mapstructure:"provision_concurrency"`
	ProvisionQueueDepth  int           `mapstructure:"provision_queue_depth"`
	RPCDurable           string        `mapstructure:"rpc_durable"`
	RPCStartSequence     uint64        `mapstructure:"rpc_start_sequence"`
	RPCConcurrency       int           `mapstructure:"rpc_concurrency"`
	RPCQueueDepth        int           `mapstructure:"rpc_queue_depth"`
	AckWait              time.Duration `mapstructure:"ack_wait"`
	MaxDeliver           int           `mapstructure:"max_deliver"`
	MaxAckPending        int           `mapstructure:"max_ack_pending"`
}

func (c GPVResponseConsumerConfig) Defaults() GPVResponseConsumerConfig {
	if c.ProvisionQueue == "" {
		c.ProvisionQueue = "provision-gpv"
	}
	if c.ProvisionConcurrency <= 0 {
		c.ProvisionConcurrency = 2
	}
	if c.ProvisionQueueDepth <= 0 {
		c.ProvisionQueueDepth = 256
	}
	if c.RPCDurable == "" {
		c.RPCDurable = "device-rpc-gpv"
	}
	if c.RPCConcurrency <= 0 {
		c.RPCConcurrency = 2
	}
	if c.RPCQueueDepth <= 0 {
		c.RPCQueueDepth = 1000
	}
	if c.AckWait <= 0 {
		c.AckWait = 30 * time.Second
	}
	if c.MaxDeliver <= 0 {
		c.MaxDeliver = 5
	}
	if c.MaxAckPending < 2000 {
		c.MaxAckPending = 2000
	}
	return c
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
	// ACSUploadBaseURL 是 CPE 可达的 ACS 上传服务基础 URL（不含路径），用于日志采集
	// Upload RPC 参数中的目标 URL 构造。示例：http://localhost:8080
	// 若为空，日志采集任务的 Upload RPC 将缺少目标 URL，设备将无法上传文件。
	ACSUploadBaseURL string `mapstructure:"acs_upload_base_url"`
	// RequireFirmwareSignature 控制固件下发前是否强制要求厂商签名（issue #8）。
	// 默认 false：仅对"带签名"的固件强制验签，未签名固件放行（向后兼容现网存量未签名
	// 固件，不破坏 happy-path）。置 true：未签名固件直接拒绝下发——待厂商签名供应链
	// 全量铺开后再切换。完整性校验（SHA-256 / MD5 回退）始终强制，不受本开关影响。
	RequireFirmwareSignature bool `mapstructure:"require_firmware_signature"`
}

// TopologyConfig 配置拓扑管理模块（F06）的设备同步行为。
// 拓扑模块负责维护 topo_nodes 和 topo_edges 表，用于前端拓扑图渲染。
// DeviceSync 控制从 devices 表到 topo_nodes 表的自动同步策略。
type TopologyConfig struct {
	DeviceSync TopologyDeviceSyncConfig `mapstructure:"device_sync"` // 设备到拓扑节点的同步配置
}

// TopologyDeviceSyncConfig 配置设备同步到拓扑节点的策略。
// 支持三种同步模式：事件驱动（实时）、启动时同步（历史数据）、定时兜底（容错）。
type TopologyDeviceSyncConfig struct {
	Enabled          bool          `mapstructure:"enabled"`            // 是否启用自动同步，默认 true
	InitialSync      bool          `mapstructure:"initial_sync"`       // 启动时是否执行全量同步，默认 true
	InitialSyncDelay time.Duration `mapstructure:"initial_sync_delay"` // 启动同步延迟，默认 10s（避免启动高峰）
	FallbackInterval time.Duration `mapstructure:"fallback_interval"`  // 兜底定时同步间隔，默认 1h（0 表示不启用）
	BatchSize        int           `mapstructure:"batch_size"`         // 批量同步大小，默认 100
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
	DB                  PostgresConfig            `mapstructure:"db"`
	TSDB                PostgresConfig            `mapstructure:"tsdb"`
	Redis               RedisConfig               `mapstructure:"redis"`
	PMRedis             RedisConfig               `mapstructure:"pm_redis"`
	NATS                NATSConfig                `mapstructure:"nats"`
	MinIO               MinIOConfig               `mapstructure:"minio"`
	Task                TaskConfig                `mapstructure:"task"` // T-0157 C2: 任务过期扫描器配置
	ParamSync           ParamSyncConfig           `mapstructure:"param_sync"`
	OfflineAlarmCleanup OfflineAlarmCleanupConfig `mapstructure:"offline_alarm_cleanup"` // #358: 离线设备活动告警清理阈值/周期/批量可配
	PM                  PMConfig                  `mapstructure:"pm"`                    // 设备上线时自动下发 PM 上传配置
	RawCleanup          RawCleanupConfig          `mapstructure:"raw_cleanup"`           // PM/MR 原始对象精确分批清理
	// PMConsumerConcurrency 是 PM 文件入库消费者的进程内并发订阅数（pm.file.received → 解析入库）。
	// NATS push 订阅 async 回调由 nats.go 单 goroutine 串行投递，单订阅只用 ~1 核；N 个订阅共享同一
	// durable consumer "pm-workers" 由 JetStream 负载均衡，吃满 worker 多核。<=0 时 worker 启动期
	// 回退到 GOMAXPROCS（即容器 CPU 配额），上限 32（2026-07-21 从16上调，见 cmd/worker/main.go
	// 注释）。设更大的值需同步核对 db.max_conns/tsdb.max_conns 连接池是否够用。
	PMConsumerConcurrency int `mapstructure:"pm_consumer_concurrency"`
	// PMAsyncCommit：PM 指标大批量写是否对本事务关掉 WAL 同步落盘（synchronous_commit=off）。
	// PM 数据可从 MinIO 原文件重建，关掉后提交不阻塞 fsync、显著提吞吐（崩溃最多丢已提交未刷盘的
	// 最后几 ms 行）。默认 false（durable）；写吞吐瓶颈场景置 true。
	PMAsyncCommit bool `mapstructure:"pm_async_commit"`
	// PMKPIWindowFromDB：KPI 计算是否从 DB 回读 counter（true）还是用内存刚解析的 counter 直接算
	// （false，默认，省每文件一次全量回读 SELECT）。仅当部署存在"同一窗口拆成多文件上报、需跨文件
	// 聚合 KPI"时才置 true。
	PMKPIWindowFromDB bool             `mapstructure:"pm_kpi_window_from_db"`
	DictLoader        DictLoaderConfig `mapstructure:"dict_loader"` // T-0178: worker BackupCleanup 需读 XMLBaseDir + ParamModel 子配置
	Metrics           MetricsConfig    `mapstructure:"metrics"`
	Tracer            TracerConfig     `mapstructure:"tracer"`
	Log               LogConfig        `mapstructure:"log"`
	RequestIDPrefix   string           `mapstructure:"request_id_prefix"` // 请求 ID 前缀，如 "worker"
}

const (
	RawCleanupModeShadow    = "shadow"
	RawCleanupModeFallback  = "fallback"
	RawCleanupModeExclusive = "exclusive"
)

// RawCleanupConfig controls exact-path cleanup of PM/MR raw objects.
type RawCleanupConfig struct {
	Enabled             bool          `mapstructure:"enabled"`
	Mode                string        `mapstructure:"mode"`
	BatchSize           int           `mapstructure:"batch_size"`
	MinRate             float64       `mapstructure:"min_rate"`
	MaxRate             float64       `mapstructure:"max_rate"`
	RateHeadroom        float64       `mapstructure:"rate_headroom"`
	RecalculateInterval time.Duration `mapstructure:"recalculate_interval"`
	ObjectTimeout       time.Duration `mapstructure:"object_timeout"`
	PrometheusURL       string        `mapstructure:"prometheus_url"`
}

// Defaults normalizes unset or invalid cleanup knobs to conservative values.
func (c RawCleanupConfig) Defaults() RawCleanupConfig {
	if c.Mode != RawCleanupModeShadow && c.Mode != RawCleanupModeFallback && c.Mode != RawCleanupModeExclusive {
		c.Mode = RawCleanupModeShadow
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.MinRate <= 0 {
		c.MinRate = 5
	}
	if c.MaxRate <= 0 {
		c.MaxRate = 100
	}
	if c.MaxRate < c.MinRate {
		c.MaxRate = c.MinRate
	}
	if c.RateHeadroom <= 0 {
		c.RateHeadroom = 1.5
	}
	if c.RecalculateInterval <= 0 {
		c.RecalculateInterval = 5 * time.Minute
	}
	if c.ObjectTimeout <= 0 {
		c.ObjectTimeout = 10 * time.Second
	}
	return c
}

// PMConfig 配置 PM 文件上传自动下发流程。
//
// 设备 offline→online 时（即 device.online 事件），worker 自动构造
// 3 个 SetParameterValues task 入队 ACS Redis 任务队列：
//
//	Device.FAP.PerfMgmt.Config.1.Enable                  = EnableValue (默认 "1")
//	Device.FAP.PerfMgmt.Config.1.URL                     = 渲染后的 UploadURLTemplate
//	Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval  = PeriodicUploadInterval (默认 900)
//
// UploadURLTemplate 支持 ${VAR} / ${VAR:-default} 环境变量插值（OnlineSubscriber 内 expandEnv 处理），
// 例如 "http://${OMC_PUBLIC_HOST:-localhost}:7557/smallcell/FileUploadService?fileType=PM&filename="，
// 由 docker-compose 注入 OMC_PUBLIC_HOST=172.19.1.132 等。
type PMConfig struct {
	// UploadURLTemplate 是下发给 CPE 的 URL 模板，含 ${ENV} 插值；
	// 末尾必须是 "filename=" 让 CPE 自己拼具体文件名。
	UploadURLTemplate string `mapstructure:"upload_url_template"`
	// EnableValue 是 Enable 参数的字符串值。文档（KPI上报参数整理.md §1）规定 "1"；
	// 但真机历史可能落 "true"。两种 TR-069 boolean 都接受。
	EnableValue string `mapstructure:"enable_value"`
	// PeriodicUploadInterval 文件上传周期（秒）。文档 §3 默认 900（15 分钟）。
	PeriodicUploadInterval int `mapstructure:"periodic_upload_interval"`
	// AutoSetupOnOnline 是否启用 device.online → 自动下发流程。
	// 关闭时 OnlineSubscriber 不订阅事件（兜底开关，回归 / 排查时可关）。
	AutoSetupOnOnline bool `mapstructure:"auto_setup_on_online"`

	// Timezone 已废弃（#458）：PM 聚合业务时区改读 sys_configs 统一源
	// （category='basic'/key='timezoneCode'，#456 systimezone.Provider），不再读本 YAML 字段。
	// 保留字段仅为 YAML 向后兼容（旧配置含 timezone 不报错），worker 已不再消费它。
	// 决定日/周/月桶本地零点对齐的逻辑不变，只是时区来源换成可动态改的 sys_configs。
	//
	// Deprecated: 用系统时区 sys_configs（basic/timezoneCode）替代；本字段不再生效。
	Timezone string `mapstructure:"timezone"`

	// Storage 聚合任务结果落库范围配置（T-0182，全局开关，非任务级）。
	Storage PMStorageConfig `mapstructure:"storage"`
}

// PMStorageConfig PM 聚合结果落库范围配置（T-0182）。
type PMStorageConfig struct {
	// StoreAllMetrics 控制聚合任务结果落库范围：
	//   - true（默认）：所有指标都落库，便于事后改任务指标集时无需重算
	//   - false（仅存所选）：只落任务定义的 N 个指标，省存储
	// 只影响"落哪些指标"，不影响"查看/导出限 N 个"收口规则（设计 §2.4）。
	// 注意：viper 缺省 bool 零值为 false；worker yaml 显式写 true 保证默认全存。
	StoreAllMetrics bool `mapstructure:"store_all_metrics"`
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
	// MaxRequestBodySize 限制入站 TR-069 SOAP 请求体字节数，防止恶意/故障 CPE
	// 发送超大 POST 导致 ACS 进程 OOM（南向接口无认证即可触达）。0 表示用默认 50MB。
	MaxRequestBodySize int64 `mapstructure:"max_request_body_size"`
	// MaxHeaderBytes 限制 HTTP 请求头总字节数。0 表示用默认 1MB（http.DefaultMaxHeaderBytes）。
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`
	// TrustedProxyCIDRs 仅用于恢复受信 HTTP 网关转发的 CPE 来源地址。
	// 留空时始终使用 TCP peer，防止直连 CPE 伪造转发头。
	TrustedProxyCIDRs []string `mapstructure:"trusted_proxy_cidrs"`
}

// AppServerConfig 配置 App 服务的 HTTP/gRPC 监听参数。
// Port 提供 REST API（供前端和第三方集成）；
// GRPCPort 提供 gRPC 接口（如与其他微服务通信）；
// TLSPort 提供 HTTPS REST API（生产环境推荐启用）。
type AppServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	TLSPort      int           `mapstructure:"tls_port"`
	GRPCPort     int           `mapstructure:"grpc_port"`
	TLS          TLSConfig     `mapstructure:"tls"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
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

func (c RedisConfig) configured() bool {
	return len(c.Addrs) > 0 || c.Password != "" || c.DB != 0 || c.PoolSize != 0
}

// EffectivePMRedis keeps old configurations deployable during the rolling
// migration. Once pm_redis contains any setting it is treated as an explicit,
// independently validated Redis endpoint rather than silently falling back.
func (c AppConfig) EffectivePMRedis() RedisConfig {
	if c.PMRedis.configured() {
		return c.PMRedis
	}
	return c.Redis
}

func (c WorkerConfig) EffectivePMRedis() RedisConfig {
	if c.PMRedis.configured() {
		return c.PMRedis
	}
	return c.Redis
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
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	// PublicEndpoint 浏览器可达的 MinIO 公网地址（host:port，无 scheme）。
	// 可空：留空时预签名 URL 用 Endpoint，适合后端 + MinIO 同进程或同主机场景。
	// 非空时：预签名 URL 用 PublicEndpoint 签发，浏览器 / 外部 SDK 能直接访问。
	// 典型用法：dev 环境 Endpoint="minio:9000"（docker 内网），PublicEndpoint="localhost:9000"
	// （宿主机浏览器）；生产环境 Endpoint=k8s service ClusterIP，PublicEndpoint=对外域名。
	PublicEndpoint string       `mapstructure:"public_endpoint"`
	Buckets        BucketConfig `mapstructure:"buckets"`
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
	// TraceBulk T-0137 M2：TR069 报文跟踪大报文外置 bucket（payload > 32KB 时 GZIP 存此处）。
	// 默认 "trace-bulk"；空时禁用外置（所有报文 inline）。
	TraceBulk string `mapstructure:"trace_bulk"`
	// FileBundles 文件管理 4 Tab（firmware/config/license/MR）的批量下载 zip 落地 bucket。
	// 默认 "file-bundles"；空时降级 = 关闭批量下载（前端按钮 disabled）。
	// 路径模板：{module}/{YYYY}/{MM}/{DD}/{task_id}.zip
	FileBundles string `mapstructure:"file_bundles"`
}

const (
	// ConfigBackupBucket is the only physical S3/MinIO bucket name used for
	// device configuration backups.
	ConfigBackupBucket = "config-backup"
	// LegacyConfigBackupBucket is accepted only at API/config boundaries.
	LegacyConfigBackupBucket = "config_backup"
)

// NormalizeConfigBackupBucket converts the legacy logical compatibility name
// before it reaches an S3/MinIO client. Other bucket names are left untouched.
func NormalizeConfigBackupBucket(bucket string) string {
	if bucket == LegacyConfigBackupBucket {
		return ConfigBackupBucket
	}
	return bucket
}

// NormalizeConfigBackupReference normalizes bucket-qualified internal object
// paths and OMC FileDownloadService HTTP(S) routes. Arbitrary external URLs
// are returned byte-for-byte, even when their object path happens to contain a
// directory named config_backup.
func NormalizeConfigBackupReference(reference string) string {
	if reference == LegacyConfigBackupBucket {
		return ConfigBackupBucket
	}
	if strings.HasPrefix(reference, LegacyConfigBackupBucket+"/") {
		return ConfigBackupBucket + strings.TrimPrefix(reference, LegacyConfigBackupBucket)
	}
	legacyDownloadSegment := "/smallcell/FileDownloadService/" + LegacyConfigBackupBucket + "/"
	canonicalDownloadSegment := "/smallcell/FileDownloadService/" + ConfigBackupBucket + "/"
	lowerReference := strings.ToLower(reference)
	if strings.Contains(reference, "://") {
		if (strings.HasPrefix(lowerReference, "http://") || strings.HasPrefix(lowerReference, "https://")) &&
			strings.Contains(reference, legacyDownloadSegment) {
			return strings.Replace(reference, legacyDownloadSegment, canonicalDownloadSegment, 1)
		}
		return reference
	}
	if strings.Contains(reference, legacyDownloadSegment) {
		return strings.Replace(reference, legacyDownloadSegment, canonicalDownloadSegment, 1)
	}
	return reference
}

func normalizeLoadedConfigBackup(target interface{}) {
	switch cfg := target.(type) {
	case *AppConfig:
		cfg.MinIO.Buckets.ConfigBackup = NormalizeConfigBackupBucket(cfg.MinIO.Buckets.ConfigBackup)
	case *ACSConfig:
		cfg.MinIO.Buckets.ConfigBackup = NormalizeConfigBackupBucket(cfg.MinIO.Buckets.ConfigBackup)
	case *WorkerConfig:
		cfg.MinIO.Buckets.ConfigBackup = NormalizeConfigBackupBucket(cfg.MinIO.Buckets.ConfigBackup)
	}
}

// MetricsConfig 配置 Prometheus 指标暴露端口。
// 各服务在此端口提供 /metrics 端点，供 Prometheus 采集。
// 同一端口也提供 /healthz 健康检查接口（由 HealthChecker 驱动）。
type MetricsConfig struct {
	Port  int         `mapstructure:"port"`
	Pprof PprofConfig `mapstructure:"pprof"`
}

// PprofConfig 控制是否在 metrics 端口挂载 net/http/pprof（默认全关）。
//
// 与 /metrics 同端口（内网信任边界），仅用于线上性能排查。配置文件即可开关；
// 因 viper SetEnvPrefix("OMCGO")+AutomaticEnv，等价 env 为
// OMCGO_METRICS_PPROF_ENABLED / OMCGO_METRICS_PPROF_CONTENTION。
type PprofConfig struct {
	Enabled    bool `mapstructure:"enabled"`    // 挂 /debug/pprof/*（CPU/heap/goroutine/trace）
	Contention bool `mapstructure:"contention"` // 额外开 block/mutex 采样（有开销，定位锁/连接池争用）
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

// RotationConfig 配置日志文件轮转策略。
//
// 两种归档管理模式：
//   - Compactor 模式（KeepUncompressed > 0）：lumberjack 仅负责切割；后台 compactor
//     goroutine 每 1 分钟扫描归档目录，保留最新 KeepUncompressed 个 .log 不压缩供
//     `tail -f`/`less` 直读，其余 .log 压缩为 .log.gz，超 MaxAgeDays 整体删除。
//     归档文件名精确到分钟（compactor rename 截断 lumberjack 的秒.毫秒）。
//   - Legacy 模式（KeepUncompressed = 0）：完全沿用 lumberjack 原生 MaxBackups +
//     MaxAge + Compress 三件套。用于 protocol_log 等暂未启用 compactor 的场景。
//
// MaxSizeMB 与 sys_configs.log.rotation.rotate_interval_minutes 是 OR 关系：任一满足都触发切割。
type RotationConfig struct {
	Enabled          bool `mapstructure:"enabled"`           // enable log rotation
	MaxSizeMB        int  `mapstructure:"max_size_mb"`       // max size in MB before rotation (default: 50)
	MaxAgeDays       int  `mapstructure:"max_age_days"`      // max days to retain old log files (default: 7)
	MaxBackups       int  `mapstructure:"max_backups"`       // legacy 模式总归档数上限；compactor 模式忽略
	Compress         bool `mapstructure:"compress"`          // legacy 模式 lumberjack 自动 gzip；compactor 模式忽略（compactor 自行管理压缩）
	LocalTime        bool `mapstructure:"local_time"`        // use local time for rotation
	KeepUncompressed int  `mapstructure:"keep_uncompressed"` // > 0 启用 compactor 模式，保留最新 N 个 .log 不压缩
}

// ProtocolLogConfig 配置 ACS 协议交互原始日志（独立于结构化日志）。
// 启用后将每次 CPE↔ACS 的完整 HTTP 请求/响应 SOAP XML 记录到专用文件，
// 方便协议调试和互操作测试（interop）。生产环境建议按需开启，文件可能很大。
// MaxBodySize 限制单条 XML 记录的最大字节数，超长时截断。
// When enabled, a dedicated log file records the complete raw SOAP/XML
// for every ACS-CPE HTTP request/response exchange.
type ProtocolLogConfig struct {
	Enabled  bool   `mapstructure:"enabled"`   // 总开关
	FilePath string `mapstructure:"file_path"` // 日志文件路径，如 /run/logs/acs/protocol.log
	// Level 控制每条 RPC 全量 XML 记录的日志级别（issue #218 治本）。
	//   - "" / "warn"（默认）：抓包记录抑制 —— 每报文不做整段 XML 的 zap JSON 编码，
	//     稳态零开销（高 CPU 回归的主因热点即此整段编码）。报文跟踪(trace)不受影响。
	//   - "info" / "debug"：恢复全量抓包 —— 每个 CPE↔ACS 请求/响应完整 SOAP XML 落盘，
	//     排障/互操作测试时显式开启。
	// 合法值：debug / info / warn / error（大小写不敏感），非法值回退 warn。
	Level       string         `mapstructure:"level"`
	MaxBodySize int            `mapstructure:"max_body_size"` // XML 截断阈值 bytes，0=不截断
	Rotation    RotationConfig `mapstructure:"rotation"`      // 轮转配置（复用 RotationConfig）
}

// ExpandEnv 展开配置文本中的 ${VAR} 与 ${VAR:-default} 占位符，值取自进程环境变量。
//
// 统一在 Load 时对整份 YAML 生效：密码 / DSN / 密钥等敏感值与 OMC_PUBLIC_HOST 等
// 部署期变量，均由 deploy/.env 经 docker-compose 注入容器环境，配置文件只写 ${VAR}
// 引用——实现「.env 单一真值源 + 部署期自动导入」。语义对齐 shell / docker-compose：
//   - ${VAR}            取 env，未设则展开为空串
//   - ${VAR:-default}   取 env，未设或为空则用 default
//
// 注意：未设且无默认值的占位符展开为空串，交由 GuardProductionSecrets（命中已知弱
// 默认 / 空密码场景）与各字段 Validate 在生产环境拦截，本函数本身不做合法性判断。
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":-"); idx >= 0 {
			if v := os.Getenv(key[:idx]); v != "" {
				return v
			}
			return key[idx+2:]
		}
		return os.Getenv(key)
	})
}

// readExpandedConfig 读取 YAML 文件、对整份内容做 ${VAR} 环境变量展开，再交给 viper
// 解析。替代 viper.SetConfigFile + ReadInConfig：viper 直接读文件不会做插值，必须
// 先 ExpandEnv 再以 reader 喂入。configType 由扩展名推断（OMC 全为 yaml）。
func readExpandedConfig(v *viper.Viper, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	if ext := strings.TrimPrefix(filepath.Ext(path), "."); ext != "" {
		v.SetConfigType(ext)
	} else {
		v.SetConfigType("yaml")
	}
	if err := v.ReadConfig(strings.NewReader(ExpandEnv(string(raw)))); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

// Load 从 YAML 配置文件加载配置。配置文件中的 ${VAR} / ${VAR:-default} 由 ExpandEnv
// 在解析前从环境变量展开（.env 单一真值源）；此外仍支持 viper 的 OMCGO_ 前缀整值覆盖
// （"."替换为"_"），如 OMCGO_SERVER_PORT=8080 覆盖 server.port，作为运维临时开关层。
// 各微服务入口直接调用，如：appconfig.Load("etc/config.dev.yaml", &cfg)
func Load(path string, target interface{}) error {
	v := viper.New()
	v.SetEnvPrefix("OMCGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := readExpandedConfig(v, path); err != nil {
		return err
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	normalizeLoadedConfigBackup(target)

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
	v.SetEnvPrefix("OMCGO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := readExpandedConfig(v, path); err != nil {
		return err
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
	normalizeLoadedConfigBackup(target)

	// Validate if target implements Validatable
	if v, ok := target.(Validatable); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}
