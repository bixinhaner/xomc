package appconfig

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ACSConfig is the configuration for the ACS engine.
type ACSConfig struct {
	Server    ACSServerConfig    `mapstructure:"server"`
	Session   SessionConfig      `mapstructure:"session"`
	RateLimit RateLimitConfig    `mapstructure:"rate_limit"`
	Auth      AuthConfig         `mapstructure:"auth"`
	Redis     RedisConfig        `mapstructure:"redis"`
	NATS      NATSConfig         `mapstructure:"nats"`
	DB        PostgresConfig     `mapstructure:"db"`
	Metrics   MetricsConfig      `mapstructure:"metrics"`
	Tracer    TracerConfig       `mapstructure:"tracer"`
	Log       LogConfig          `mapstructure:"log"`
}

// CORSConfig holds CORS middleware settings.
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// AppConfig is the configuration for the main application.
type AppConfig struct {
	Server     AppServerConfig  `mapstructure:"server"`
	DB         PostgresConfig   `mapstructure:"db"`
	TSDB       PostgresConfig   `mapstructure:"tsdb"`
	Redis      RedisConfig      `mapstructure:"redis"`
	NATS       NATSConfig       `mapstructure:"nats"`
	MinIO      MinIOConfig      `mapstructure:"minio"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Northbound NorthboundConfig `mapstructure:"northbound"`
	NEDirect   NEDirectConfig   `mapstructure:"ne_direct"`
	Metrics    MetricsConfig    `mapstructure:"metrics"`
	Tracer     TracerConfig     `mapstructure:"tracer"`
	Log        LogConfig        `mapstructure:"log"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// NorthboundConfig holds northbound/OSS interface settings.
type NorthboundConfig struct {
	PushTargets []PushTargetConfig `mapstructure:"push_targets"`
}

// PushTargetConfig defines a single northbound push target.
type PushTargetConfig struct {
	ID         string   `mapstructure:"id"`
	URL        string   `mapstructure:"url"`
	AuthType   string   `mapstructure:"auth_type"`
	AuthToken  string   `mapstructure:"auth_token"`
	DataTypes  []string `mapstructure:"data_types"`
	Format     string   `mapstructure:"format"`
	BatchSize  int      `mapstructure:"batch_size"`
	RetryCount int      `mapstructure:"retry_count"`
	Enabled    bool     `mapstructure:"enabled"`
}

// NEDirectConfig holds NE Direct connection settings (CMCC only).
type NEDirectConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// WorkerConfig is the configuration for the background worker process.
type WorkerConfig struct {
	DB      PostgresConfig `mapstructure:"db"`
	TSDB    PostgresConfig `mapstructure:"tsdb"`
	Redis   RedisConfig    `mapstructure:"redis"`
	NATS    NATSConfig     `mapstructure:"nats"`
	MinIO   MinIOConfig    `mapstructure:"minio"`
	Metrics MetricsConfig  `mapstructure:"metrics"`
	Tracer  TracerConfig   `mapstructure:"tracer"`
	Log     LogConfig      `mapstructure:"log"`
}

// ACSServerConfig holds the ACS HTTP server settings.
type ACSServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	TLSPort      int           `mapstructure:"tls_port"`
	TLS          TLSConfig     `mapstructure:"tls"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// AppServerConfig holds the main application server settings.
type AppServerConfig struct {
	Host     string    `mapstructure:"host"`
	Port     int       `mapstructure:"port"`
	TLSPort  int       `mapstructure:"tls_port"`
	GRPCPort int       `mapstructure:"grpc_port"`
	TLS      TLSConfig `mapstructure:"tls"`
}

// TLSConfig holds TLS certificate settings.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// SessionConfig holds ACS session settings.
type SessionConfig struct {
	Timeout       time.Duration `mapstructure:"timeout"`
	MaxConcurrent int64         `mapstructure:"max_concurrent"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	PerDevice      int           `mapstructure:"per_device"`       // 每设备每分钟最大 Inform 数
	Burst          int           `mapstructure:"burst"`            // token bucket 突发容量
	MaxDevices     int           `mapstructure:"max_devices"`      // 限流器追踪的最大设备数
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"` // 清理扫描间隔
	CleanupTimeout  time.Duration `mapstructure:"cleanup_timeout"`  // 设备不活跃淘汰超时
}

// AuthConfig holds CPE authentication settings.
type AuthConfig struct {
	Mode     string `mapstructure:"mode"` // digest, basic, none
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	DSN                 string        `mapstructure:"dsn"`
	MaxConns            int32         `mapstructure:"max_conns"`
	MinConns            int32         `mapstructure:"min_conns"`
	MaxConnLifetime     time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime     time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addrs    []string `mapstructure:"addrs"`
	Password string   `mapstructure:"password"`
	DB       int      `mapstructure:"db"`
	PoolSize int      `mapstructure:"pool_size"`
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	MaxReconnect  int           `mapstructure:"max_reconnect"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}

// MinIOConfig holds MinIO/S3 connection settings.
type MinIOConfig struct {
	Endpoint  string       `mapstructure:"endpoint"`
	AccessKey string       `mapstructure:"access_key"`
	SecretKey string       `mapstructure:"secret_key"`
	UseSSL    bool         `mapstructure:"use_ssl"`
	Buckets   BucketConfig `mapstructure:"buckets"`
}

// BucketConfig defines the MinIO bucket names.
type BucketConfig struct {
	PMFiles      string `mapstructure:"pm_files"`
	MRFiles      string `mapstructure:"mr_files"`
	Firmware     string `mapstructure:"firmware"`
	ConfigBackup string `mapstructure:"config_backup"`
	Logs         string `mapstructure:"logs"`
	Reports      string `mapstructure:"reports"`
}

// MetricsConfig holds Prometheus metrics server settings.
type MetricsConfig struct {
	Port int `mapstructure:"port"`
}

// TracerConfig holds OpenTelemetry tracing settings.
type TracerConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

// LogConfig holds structured logging settings.
type LogConfig struct {
	Level       string         `mapstructure:"level"`
	Format      string         `mapstructure:"format"`       // json, console
	OutputPaths []string       `mapstructure:"output_paths"` // output destinations, e.g. ["stdout", "/var/log/omcgo/app.log"]
	Rotation    RotationConfig `mapstructure:"rotation"`     // log rotation settings
}

// RotationConfig holds log rotation settings.
type RotationConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // enable log rotation
	MaxSizeMB  int    `mapstructure:"max_size_mb"` // max size in MB before rotation (default: 20)
	MaxAgeDays int    `mapstructure:"max_age_days"` // max days to retain old log files (default: 7)
	MaxBackups int    `mapstructure:"max_backups"`  // max number of old log files to retain (default: 100)
	Compress   bool   `mapstructure:"compress"`     // compress rotated files
	LocalTime  bool   `mapstructure:"local_time"`   // use local time for rotation
}

// Load reads a configuration file and unmarshals it into the target struct.
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

	return nil
}

// LoadWithEnvOverride loads config from file and applies environment variable overrides.
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

	return nil
}
