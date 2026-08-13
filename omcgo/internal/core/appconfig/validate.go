package appconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// Validatable is implemented by config types that support self-validation.
type Validatable interface {
	Validate() error
}

// Validate checks AppConfig for common configuration errors.
// Returns a joined error listing all validation failures.
func (c *AppConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.TSDB.validate("tsdb"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validatePMRedis(c.Redis, c.PMRedis); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.JWT.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Server.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.DictLoader.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.ParamRegistry.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.ParamSync.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Notification.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Dashboard.Defaults().Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.MinIO.Buckets.validateConfigBackup(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (c DashboardConfig) Validate() error {
	var errs []string
	if c.QueryTimeout <= 0 {
		errs = append(errs, "dashboard.query_timeout must be > 0")
	}
	if c.StatementTimeout <= 0 || c.StatementTimeout >= c.QueryTimeout {
		errs = append(errs, "dashboard.statement_timeout must be > 0 and less than query_timeout")
	}
	if c.MaxConcurrent < 1 || c.MaxConcurrent > 64 {
		errs = append(errs, "dashboard.max_concurrent must be between 1 and 64")
	}
	if c.QueueTimeout <= 0 {
		errs = append(errs, "dashboard.queue_timeout must be > 0")
	}
	if c.FreshCacheTTL <= 0 || c.FreshCacheTTL >= 5*time.Minute {
		errs = append(errs, "dashboard.fresh_cache_ttl must be > 0 and less than 5m")
	}
	if c.StaleTTL < 2*c.FreshCacheTTL {
		errs = append(errs, "dashboard.stale_ttl must be at least twice fresh_cache_ttl")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// Validate checks ACSConfig for common configuration errors.
func (c *ACSConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	// KPI/时序库物理分离：ACS 写 trace_messages（已迁时序库），tsdb 池必填（仿 db）。
	if err := c.TSDB.validate("tsdb"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Server.validateACS(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.ParamSync.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if c.Session.Timeout > 0 && c.Session.MaxConcurrent <= 0 {
		errs = append(errs, "session.max_concurrent must be > 0 when session is configured")
	}
	if c.Auth.Mode != "" && c.Auth.Mode != "digest" && c.Auth.Mode != "basic" && c.Auth.Mode != "none" {
		errs = append(errs, fmt.Sprintf("auth.mode must be digest, basic, or none, got %q", c.Auth.Mode))
	}
	if err := c.MinIO.Buckets.validateConfigBackup(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// Validate checks WorkerConfig for common configuration errors.
func (c *WorkerConfig) Validate() error {
	var errs []string

	if err := c.DB.validate("db"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.TSDB.validate("tsdb"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validatePMRedis(c.Redis, c.PMRedis); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Log.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.Metrics.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.ParamSync.validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.MinIO.Buckets.validateConfigBackup(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (c BucketConfig) validateConfigBackup() error {
	if c.ConfigBackup == "" {
		return nil
	}
	if err := s3utils.CheckValidBucketNameStrict(c.ConfigBackup); err != nil {
		return fmt.Errorf("minio.buckets.config_backup must be a valid S3 bucket name: %w", err)
	}
	return nil
}

func (c PostgresConfig) validate(prefix string) error {
	if c.DSN == "" {
		return fmt.Errorf("%s.dsn must not be empty", prefix)
	}
	if c.MaxConns > 0 && c.MinConns > c.MaxConns {
		return fmt.Errorf("%s.min_conns (%d) must not exceed max_conns (%d)", prefix, c.MinConns, c.MaxConns)
	}
	return nil
}

func (c RedisConfig) validate() error {
	if len(c.Addrs) == 0 {
		return fmt.Errorf("redis.addrs must not be empty")
	}
	return nil
}

func validatePMRedis(core, pm RedisConfig) error {
	if !pm.configured() {
		if IsProductionEnv() {
			return fmt.Errorf("pm_redis.addrs must not be empty in production")
		}
		return nil
	}
	if len(pm.Addrs) == 0 {
		return fmt.Errorf("pm_redis.addrs must not be empty")
	}
	if IsProductionEnv() && redisAddressSetsOverlap(core.Addrs, pm.Addrs) {
		return fmt.Errorf("pm_redis must be physically isolated from redis; a different DB index is not isolation")
	}
	return nil
}

func redisAddressSetsOverlap(left, right []string) bool {
	addresses := make(map[string]struct{}, len(left))
	for _, addr := range left {
		addresses[strings.ToLower(strings.TrimSpace(addr))] = struct{}{}
	}
	for _, addr := range right {
		key := strings.ToLower(strings.TrimSpace(addr))
		if _, exists := addresses[key]; exists {
			return true
		}
	}
	return false
}

func (c JWTConfig) validate() error {
	if c.Secret == "" {
		return fmt.Errorf("jwt.secret must not be empty")
	}
	if len(c.Secret) < 16 {
		return fmt.Errorf("jwt.secret must be at least 16 characters, got %d", len(c.Secret))
	}
	return nil
}

func (c AppServerConfig) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

func (c ACSServerConfig) validateACS() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Port)
	}
	if c.TLS.Enabled {
		if c.TLSPort < 1 || c.TLSPort > 65535 {
			return fmt.Errorf("server.tls_port must be between 1 and 65535 when TLS enabled, got %d", c.TLSPort)
		}
	}
	return nil
}

func (c LogConfig) validate() error {
	switch c.Level {
	case "", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level must be debug, info, warn, or error, got %q", c.Level)
	}
	switch c.Format {
	case "", "json", "console":
	default:
		return fmt.Errorf("log.format must be json or console, got %q", c.Format)
	}
	return nil
}

func (c MetricsConfig) validate() error {
	if c.Port != 0 && (c.Port < 1 || c.Port > 65535) {
		return fmt.Errorf("metrics.port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

func (c DictLoaderConfig) validate() error {
	// All zero values are acceptable — the dictloader package applies defaults
	// (LoadConcurrency=4, CacheVersionPollInterval=30s) at construction.
	if c.LoadConcurrency < 0 {
		return fmt.Errorf("dict_loader.load_concurrency must not be negative, got %d", c.LoadConcurrency)
	}
	if c.LoadConcurrency > 64 {
		return fmt.Errorf("dict_loader.load_concurrency must not exceed 64, got %d", c.LoadConcurrency)
	}
	if c.CacheVersionPollInterval < 0 {
		return fmt.Errorf("dict_loader.cache_version_poll_interval must not be negative, got %s", c.CacheVersionPollInterval)
	}
	return nil
}

func (c NotificationConfig) validate() error {
	// SMTP 启用时必须配齐发信所需字段，否则启动期就暴露问题（fail fast），
	// 而非等到第一封告警邮件发不出才发现。disabled 时不校验。
	if c.SMTP.Enabled {
		if c.SMTP.Host == "" {
			return fmt.Errorf("notification.smtp.host must not be empty when smtp.enabled=true")
		}
		if c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
			return fmt.Errorf("notification.smtp.port must be between 1 and 65535 when smtp.enabled=true, got %d", c.SMTP.Port)
		}
		if c.SMTP.From == "" {
			return fmt.Errorf("notification.smtp.from must not be empty when smtp.enabled=true")
		}
	}
	return nil
}

func (c ParamRegistryConfig) validate() error {
	// 零值合法——RedisCache 在构造时把 ≤0 退化为默认 24h。
	if c.DefaultTTL < 0 {
		return fmt.Errorf("param_registry.default_ttl must not be negative, got %s", c.DefaultTTL)
	}
	if c.DiscoveredTTL < 0 {
		return fmt.Errorf("param_registry.discovered_ttl must not be negative, got %s", c.DiscoveredTTL)
	}
	return nil
}

func (c ParamSyncConfig) validate() error {
	if c.ManualOfflineMode != "" && c.ManualOfflineMode != "queue" && c.ManualOfflineMode != "reject" {
		return fmt.Errorf("param_sync.manual_offline_mode must be queue or reject, got %q", c.ManualOfflineMode)
	}
	if c.ResultConsumerShardCount < 0 {
		return fmt.Errorf("param_sync.result_consumer_shard_count must not be negative, got %d", c.ResultConsumerShardCount)
	}
	if c.ResultConsumerQueueDepth < 0 {
		return fmt.Errorf("param_sync.result_consumer_queue_depth must not be negative, got %d", c.ResultConsumerQueueDepth)
	}
	if c.ResultConsumerPullBatchSize < 0 {
		return fmt.Errorf("param_sync.result_consumer_pull_batch_size must not be negative, got %d", c.ResultConsumerPullBatchSize)
	}
	if c.ResultConsumerPullConcurrency < 0 {
		return fmt.Errorf("param_sync.result_consumer_pull_concurrency must not be negative, got %d", c.ResultConsumerPullConcurrency)
	}
	if c.ResultConsumerAckWait < 0 {
		return fmt.Errorf("param_sync.result_consumer_ack_wait must not be negative, got %s", c.ResultConsumerAckWait)
	}
	if c.ResultConsumerMaxAckPending < 0 {
		return fmt.Errorf("param_sync.result_consumer_max_ack_pending must not be negative, got %d", c.ResultConsumerMaxAckPending)
	}
	if c.RecoveryRunLimit < 0 {
		return fmt.Errorf("param_sync.recovery_run_limit must not be negative, got %d", c.RecoveryRunLimit)
	}
	if c.RecoveryTaskLimitPerRun < 0 {
		return fmt.Errorf("param_sync.recovery_task_limit_per_run must not be negative, got %d", c.RecoveryTaskLimitPerRun)
	}
	if c.RecoveryTaskBudget < 0 {
		return fmt.Errorf("param_sync.recovery_task_budget must not be negative, got %d", c.RecoveryTaskBudget)
	}
	return nil
}
