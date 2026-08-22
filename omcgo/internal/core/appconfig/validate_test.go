package appconfig

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*AppConfig)
		wantErr string // substring expected in error message
	}{
		{
			name:   "valid config passes",
			modify: func(*AppConfig) {},
		},
		{
			name:    "empty DB DSN",
			modify:  func(c *AppConfig) { c.DB.DSN = "" },
			wantErr: "db.dsn must not be empty",
		},
		{
			name:    "min_conns exceeds max_conns",
			modify:  func(c *AppConfig) { c.DB.MaxConns = 5; c.DB.MinConns = 10 },
			wantErr: "min_conns",
		},
		{
			name:    "empty Redis addrs",
			modify:  func(c *AppConfig) { c.Redis.Addrs = nil },
			wantErr: "redis.addrs must not be empty",
		},
		{
			name:    "empty JWT secret",
			modify:  func(c *AppConfig) { c.JWT.Secret = "" },
			wantErr: "jwt.secret must not be empty",
		},
		{
			name:    "JWT secret too short",
			modify:  func(c *AppConfig) { c.JWT.Secret = "short" },
			wantErr: "jwt.secret must be at least 16",
		},
		{
			name:    "invalid server port",
			modify:  func(c *AppConfig) { c.Server.Port = 0 },
			wantErr: "server.port must be between 1 and 65535",
		},
		{
			name:    "invalid log level",
			modify:  func(c *AppConfig) { c.Log.Level = "verbose" },
			wantErr: "log.level must be",
		},
		{
			name:    "invalid log format",
			modify:  func(c *AppConfig) { c.Log.Format = "yaml" },
			wantErr: "log.format must be",
		},
		{
			name:    "invalid metrics port",
			modify:  func(c *AppConfig) { c.Metrics.Port = 99999 },
			wantErr: "metrics.port must be between 1 and 65535",
		},
		{
			name:    "negative dict_loader concurrency",
			modify:  func(c *AppConfig) { c.DictLoader.LoadConcurrency = -1 },
			wantErr: "dict_loader.load_concurrency must not be negative",
		},
		{
			name:    "dict_loader concurrency exceeds cap",
			modify:  func(c *AppConfig) { c.DictLoader.LoadConcurrency = 128 },
			wantErr: "dict_loader.load_concurrency must not exceed 64",
		},
		{
			name:    "negative dict_loader poll interval",
			modify:  func(c *AppConfig) { c.DictLoader.CacheVersionPollInterval = -1 * time.Second },
			wantErr: "dict_loader.cache_version_poll_interval must not be negative",
		},
		{
			name:    "invalid config backup bucket",
			modify:  func(c *AppConfig) { c.MinIO.Buckets.ConfigBackup = "config_backup_invalid" },
			wantErr: "minio.buckets.config_backup must be a valid S3 bucket name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validAppConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErr),
					"expected error containing %q, got %q", tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPMRedisFallbackUsesCoreConfiguration(t *testing.T) {
	core := RedisConfig{Addrs: []string{"redis-core:6379"}, Password: "secret", DB: 3, PoolSize: 40}
	appCfg := AppConfig{Redis: core}
	workerCfg := WorkerConfig{Redis: core}

	require.Equal(t, core, appCfg.EffectivePMRedis())
	require.Equal(t, core, workerCfg.EffectivePMRedis())
}

func TestProductionRejectsExplicitSharedPMRedis(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "release")

	appCfg := validAppConfig()
	appCfg.Redis = RedisConfig{Addrs: []string{"redis-b:6379", "redis-a:6379"}, DB: 0}
	appCfg.PMRedis = RedisConfig{Addrs: []string{"redis-a:6379", "redis-b:6379"}, DB: 9}
	require.ErrorContains(t, appCfg.Validate(), "pm_redis must be physically isolated")

	workerCfg := validWorkerConfig()
	workerCfg.Redis = RedisConfig{Addrs: []string{"redis-core:6379"}, DB: 0}
	workerCfg.PMRedis = RedisConfig{Addrs: []string{"redis-core:6379"}, DB: 12}
	require.ErrorContains(t, workerCfg.Validate(), "pm_redis must be physically isolated")
}

func TestProductionRejectsMissingPMRedisCompatibilityFallback(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "release")
	cfg := validWorkerConfig()
	require.ErrorContains(t, cfg.Validate(), "pm_redis.addrs must not be empty in production")
}

func TestProductionRejectsAnyPMRedisAddressOverlap(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "release")
	cfg := validWorkerConfig()
	cfg.Redis.Addrs = []string{"redis-core-a:6379", "redis-shared:6379"}
	cfg.PMRedis = RedisConfig{Addrs: []string{"redis-pm-a:6379", " REDIS-SHARED:6379 "}}

	require.ErrorContains(t, cfg.Validate(), "pm_redis must be physically isolated")
}

func TestAppAndWorkerConfigExamplesUseDedicatedPMRedis(t *testing.T) {
	for _, service := range []string{"app", "worker"} {
		for _, env := range []string{"dev", "test", "local", "prod"} {
			path := "../../../cmd/" + service + "/etc/config." + env + ".yaml"
			t.Run(service+"/"+env, func(t *testing.T) {
				if service == "app" {
					t.Setenv("OMCGO_JWT_SECRET", "test-only-dedicated-redis-config-secret")
					var cfg AppConfig
					require.NoError(t, Load(path, &cfg))
					require.NotEmpty(t, cfg.PMRedis.Addrs)
					require.False(t, redisAddressSetsOverlap(cfg.Redis.Addrs, cfg.PMRedis.Addrs))
					return
				}
				var cfg WorkerConfig
				require.NoError(t, Load(path, &cfg))
				require.NotEmpty(t, cfg.PMRedis.Addrs)
				require.False(t, redisAddressSetsOverlap(cfg.Redis.Addrs, cfg.PMRedis.Addrs))
			})
		}
	}
}

func TestACSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*ACSConfig)
		wantErr string
	}{
		{
			name:   "valid config passes",
			modify: func(*ACSConfig) {},
		},
		{
			name:    "invalid auth mode",
			modify:  func(c *ACSConfig) { c.Auth.Mode = "oauth" },
			wantErr: "auth.mode must be",
		},
		{
			name:    "TLS enabled without tls_port",
			modify:  func(c *ACSConfig) { c.Server.TLS.Enabled = true; c.Server.TLSPort = 0 },
			wantErr: "tls_port must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validACSConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErr),
					"expected error containing %q, got %q", tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestACSServerConfigRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	err := (ACSServerConfig{Port: 7547, TrustedProxyCIDRs: []string{"not-a-cidr"}}).validateACS()
	require.ErrorContains(t, err, "server.trusted_proxy_cidrs")
}

func TestBackpressureConfigDefaults(t *testing.T) {
	cfg := (BackpressureConfig{}).Defaults()
	assert.Equal(t, 5000, cfg.QueuePendingHigh)
	assert.Equal(t, 1000, cfg.QueuePendingLow)
	assert.Equal(t, 10*time.Minute, cfg.QueueOldestHigh)
	assert.Equal(t, 2*time.Minute, cfg.QueueOldestLow)
	assert.Equal(t, 5*time.Minute, cfg.QueueSlopeWindow)

	clamped := (BackpressureConfig{
		QueuePendingHigh: 100,
		QueuePendingLow:  200,
		QueueOldestHigh:  time.Minute,
		QueueOldestLow:   2 * time.Minute,
	}).Defaults()
	assert.Equal(t, 100, clamped.QueuePendingLow)
	assert.Equal(t, time.Minute, clamped.QueueOldestLow)
}

func TestACSDeploymentBackpressureFallbacksMatchDefaults(t *testing.T) {
	t.Setenv("DOCKER_COMPOSE_SUBNET", "192.0.2.0/24")
	defaults := (BackpressureConfig{}).Defaults()
	for _, path := range []string{
		"../../../cmd/acs/etc/config.dev.yaml",
		"../../../cmd/acs/etc/config.prod.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			var cfg ACSConfig
			require.NoError(t, Load(path, &cfg))
			fallback := cfg.Backpressure.Defaults()
			assert.Equal(t, defaults.QueuePendingHigh, fallback.QueuePendingHigh)
			assert.Equal(t, defaults.QueuePendingLow, fallback.QueuePendingLow)
			assert.Equal(t, defaults.QueueOldestHigh, fallback.QueueOldestHigh)
			assert.Equal(t, defaults.QueueOldestLow, fallback.QueueOldestLow)
		})
	}
}

func TestDashboardConfigDefaults(t *testing.T) {
	cfg := (DashboardConfig{}).Defaults()
	assert.Equal(t, 3*time.Second, cfg.QueryTimeout)
	assert.Equal(t, 2500*time.Millisecond, cfg.StatementTimeout)
	assert.Equal(t, 4, cfg.MaxConcurrent)
	assert.Equal(t, 100*time.Millisecond, cfg.QueueTimeout)
	assert.Equal(t, 4*time.Minute+30*time.Second, cfg.FreshCacheTTL)
	assert.Equal(t, 15*time.Minute, cfg.StaleTTL)
}

func TestDashboardConfigValidateRejectsUnsafeBounds(t *testing.T) {
	tests := []DashboardConfig{
		{QueryTimeout: 2 * time.Second, StatementTimeout: 3 * time.Second, MaxConcurrent: 4, QueueTimeout: time.Second, FreshCacheTTL: time.Minute, StaleTTL: 15 * time.Minute},
		{QueryTimeout: 3 * time.Second, StatementTimeout: 2 * time.Second, MaxConcurrent: 0, QueueTimeout: time.Second, FreshCacheTTL: time.Minute, StaleTTL: 15 * time.Minute},
		{QueryTimeout: 3 * time.Second, StatementTimeout: 2 * time.Second, MaxConcurrent: 4, QueueTimeout: time.Second, FreshCacheTTL: 5 * time.Minute, StaleTTL: 15 * time.Minute},
		{QueryTimeout: 3 * time.Second, StatementTimeout: 2 * time.Second, MaxConcurrent: 4, QueueTimeout: time.Second, FreshCacheTTL: 4 * time.Minute, StaleTTL: 5 * time.Minute},
	}
	for _, cfg := range tests {
		require.Error(t, cfg.Validate())
	}
}

func TestLoad_ValidatesConfig(t *testing.T) {
	// Write a minimal valid config to a temp file, then load it.
	// This tests that Load() calls Validate() automatically.
	t.Run("missing DSN triggers validation error", func(t *testing.T) {
		var cfg AppConfig
		// Use a nonexistent file — Load should fail on file read, not validation.
		err := Load("nonexistent.yaml", &cfg)
		require.Error(t, err)
	})
}

// validAppConfig returns a minimal valid AppConfig for testing.
func validAppConfig() AppConfig {
	return AppConfig{
		Server:  AppServerConfig{Port: 8080},
		DB:      PostgresConfig{DSN: "postgres://user:pass@localhost:5432/omcgo", MaxConns: 10},
		TSDB:    PostgresConfig{DSN: "postgres://user:pass@localhost:5432/omcgo_ts", MaxConns: 5},
		Redis:   RedisConfig{Addrs: []string{"localhost:6379"}},
		JWT:     JWTConfig{Secret: "this-is-a-valid-secret-1234567890"},
		Log:     LogConfig{Level: "info", Format: "json"},
		Metrics: MetricsConfig{Port: 9090},
	}
}

func validACSConfig() ACSConfig {
	return ACSConfig{
		Server:  ACSServerConfig{Port: 7547},
		Session: SessionConfig{Timeout: 5 * time.Minute, MaxConcurrent: 5000},
		Auth:    AuthConfig{Mode: "digest"},
		DB:      PostgresConfig{DSN: "postgres://user:pass@localhost:5432/omcgo", MaxConns: 10},
		// KPI/时序库物理分离：ACS 写 trace_messages（已迁时序库），tsdb 池现为必填。
		TSDB:    PostgresConfig{DSN: "postgres://user:pass@localhost:5433/omcgo_ts", MaxConns: 5},
		Redis:   RedisConfig{Addrs: []string{"localhost:6379"}},
		Log:     LogConfig{Level: "info", Format: "json"},
		Metrics: MetricsConfig{Port: 9091},
	}
}

func validWorkerConfig() WorkerConfig {
	return WorkerConfig{
		DB:      PostgresConfig{DSN: "postgres://user:pass@localhost:5432/omcgo", MaxConns: 10},
		TSDB:    PostgresConfig{DSN: "postgres://user:pass@localhost:5433/omcgo_ts", MaxConns: 5},
		Redis:   RedisConfig{Addrs: []string{"localhost:6379"}},
		Log:     LogConfig{Level: "info", Format: "json"},
		Metrics: MetricsConfig{Port: 9092},
	}
}
