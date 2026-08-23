package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/pm/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPMQueueTuningCoversDeviceRegistrationWindow(t *testing.T) {
	tuning := pmQueueTuning(32)

	assert.Equal(t, 2*time.Minute, tuning.AckWait)
	assert.Equal(
		t,
		event.MaxDeliveriesForRetryHorizon(collector.DeviceRegistrationGrace),
		tuning.MaxDeliver,
	)
	assert.Equal(t, 128, tuning.MaxAckPending)
}

func TestPMTSDBConnectionBudgetCoversIngestAndHourlyFinalization(t *testing.T) {
	assert.Equal(t, int32(120), pmTSDBConnectionBudget(32, 32, 16))
	assert.Equal(t, int32(40), pmTSDBConnectionBudget(4, 4, 8))
	assert.Equal(t, int32(40), pmTSDBConnectionBudget(32, 0, 0),
		"disabled aggregation excludes finalizer and three aggregation consumers")
	assert.Equal(t, int32(1<<31-1), pmTSDBConnectionBudget(32, 32, int(^uint(0)>>1)),
		"untrusted environment concurrency must saturate instead of wrapping negative")
}

func TestEffectivePMConsumerConcurrencyEnvOverride(t *testing.T) {
	t.Setenv("PM_CONSUMER_CONCURRENCY", "1")

	assert.Equal(t, 1, effectivePMConsumerConcurrency(4))
}

func TestEffectivePMConsumerConcurrencyIgnoresInvalidEnvOverride(t *testing.T) {
	t.Setenv("PM_CONSUMER_CONCURRENCY", "invalid")

	assert.Equal(t, 4, effectivePMConsumerConcurrency(4))
}

func TestEffectivePMConsumerConcurrencyCapsEnvOverride(t *testing.T) {
	t.Setenv("PM_CONSUMER_CONCURRENCY", "99")

	assert.Equal(t, 32, effectivePMConsumerConcurrency(4))
}

func TestWorkerTSDBPoolsCoverOverlappingPMWorkloads(t *testing.T) {
	tests := []struct {
		name                string
		path                string
		finalizeConcurrency int
		consumerConcurrency int
	}{
		{name: "dev", path: "etc/config.dev.yaml", finalizeConcurrency: 4, consumerConcurrency: 8},
		{name: "local", path: "etc/config.local.yaml", finalizeConcurrency: 4, consumerConcurrency: 8},
		{name: "test", path: "etc/config.test.yaml", finalizeConcurrency: 4, consumerConcurrency: 8},
		// 生产池覆盖 32 路 finalize 与三类各 16 路聚合消费者同时工作的峰值。
		{name: "prod-tuned", path: "etc/config.prod.yaml", finalizeConcurrency: 32, consumerConcurrency: 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg appconfig.WorkerConfig
			require.NoError(t, appconfig.Load(tt.path, &cfg))

			required := pmTSDBConnectionBudget(
				cfg.PMConsumerConcurrency,
				tt.finalizeConcurrency,
				tt.consumerConcurrency,
			)
			assert.GreaterOrEqual(t, cfg.TSDB.MaxConns, required)
		})
	}
}

func TestKubernetesWorkerConfigCoversDefaultAggregationBudget(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "deployments", "k8s", "worker", "configmap.yaml",
	))
	require.NoError(t, err)
	var manifest struct {
		Data map[string]string `yaml:"data"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &manifest))

	var cfg struct {
		TSDB struct {
			MaxConns int32 `yaml:"max_conns"`
		} `yaml:"tsdb"`
		PMConsumerConcurrency int `yaml:"pm_consumer_concurrency"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(manifest.Data["config.yaml"]), &cfg))
	required := pmTSDBConnectionBudget(
		cfg.PMConsumerConcurrency,
		4,
		8,
	)
	assert.GreaterOrEqual(t, cfg.TSDB.MaxConns, required)
}
