package appconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("APPCFG_VAR_X1", "value-x1")
	cases := []struct {
		in   string
		want string
	}{
		{"plain text", "plain text"},
		{"${APPCFG_VAR_X1}", "value-x1"},
		{"${APPCFG_UNSET_QQQ:-fallback}", "fallback"},
		{"${APPCFG_VAR_X1:-fallback}", "value-x1"}, // env wins over default
		{"prefix-${APPCFG_VAR_X1}-suffix", "prefix-value-x1-suffix"},
		{"${APPCFG_UNSET_QQQ}", ""}, // 未设且无默认 → 空串
		{"postgres://omcgo:${APPCFG_VAR_X1}@postgres:5432/omcgo?sslmode=require",
			"postgres://omcgo:value-x1@postgres:5432/omcgo?sslmode=require"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ExpandEnv(c.in), "input: %s", c.in)
	}
}

func TestNormalizeLoadedConfigBackup_AppliesToAllServiceConfigs(t *testing.T) {
	app := &AppConfig{MinIO: MinIOConfig{Buckets: BucketConfig{ConfigBackup: "config_backup"}}}
	acs := &ACSConfig{MinIO: MinIOConfig{Buckets: BucketConfig{ConfigBackup: "config_backup"}}}
	worker := &WorkerConfig{MinIO: MinIOConfig{Buckets: BucketConfig{ConfigBackup: "config_backup"}}}

	for _, target := range []any{app, acs, worker} {
		normalizeLoadedConfigBackup(target)
	}

	assert.Equal(t, "config-backup", app.MinIO.Buckets.ConfigBackup)
	assert.Equal(t, "config-backup", acs.MinIO.Buckets.ConfigBackup)
	assert.Equal(t, "config-backup", worker.MinIO.Buckets.ConfigBackup)
}

// TestLoad_ExpandsEnvPlaceholders 验证 Load 在解析前对整份 YAML 做 ${VAR} 展开，
// 即 .env 经环境变量注入、配置文件用 ${VAR} 引用的统一机制端到端可用。
func TestLoad_ExpandsEnvPlaceholders(t *testing.T) {
	t.Setenv("APPCFG_DB_PASSWORD", "s3cret")
	// APPCFG_DB_USER 故意不设，验证 :-default 回退
	require.NoError(t, os.Unsetenv("APPCFG_DB_USER"))

	dir := t.TempDir()
	f := filepath.Join(dir, "x.yaml")
	yaml := `db:
  dsn: "postgres://${APPCFG_DB_USER:-omcgo}:${APPCFG_DB_PASSWORD}@postgres:5432/omcgo?sslmode=require"
`
	require.NoError(t, os.WriteFile(f, []byte(yaml), 0o600))

	var cfg struct {
		DB struct {
			DSN string `mapstructure:"dsn"`
		} `mapstructure:"db"`
	}
	require.NoError(t, Load(f, &cfg))
	assert.Equal(t,
		"postgres://omcgo:s3cret@postgres:5432/omcgo?sslmode=require",
		cfg.DB.DSN)
}
