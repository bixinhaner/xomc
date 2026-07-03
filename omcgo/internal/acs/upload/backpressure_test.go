package upload

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDiskUsagePct(t *testing.T) {
	tests := []struct {
		name string
		text string
		want float64
		err  bool
	}{
		{
			name: "usable capacity total+free",
			text: `# HELP minio_cluster_capacity_usable_total_bytes x
minio_cluster_capacity_usable_total_bytes{server="m1"} 1000
minio_cluster_capacity_usable_free_bytes{server="m1"} 200`,
			want: 80, // (1000-200)/1000
		},
		{
			name: "node disk free summed across drives",
			text: `minio_node_disk_total_bytes{drive="/d1"} 500
minio_node_disk_total_bytes{drive="/d2"} 500
minio_node_disk_free_bytes{drive="/d1"} 100
minio_node_disk_free_bytes{drive="/d2"} 150`,
			want: 75, // total 1000, free 250 → used 750
		},
		{
			name: "node disk used when no free",
			text: `minio_node_disk_total_bytes 1000
minio_node_disk_used_bytes 900`,
			want: 90,
		},
		{
			name: "no usable metrics",
			text: `some_other_metric 1`,
			err:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDiskUsagePct(tt.text)
			if tt.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestDecideBackpressure_Hysteresis(t *testing.T) {
	cfg := BackpressureConfig{
		Enabled: true, DiskHighPct: 85, DiskLowPct: 75,
	}

	// 未背压：低于高水位不触发。
	assert.False(t, decideBackpressure(false, 80, cfg))
	// 未背压：磁盘越高水位 → 进入。
	assert.True(t, decideBackpressure(false, 90, cfg))
	// 已背压：介于高低水位之间 → 保持（迟滞，不抖动）。
	assert.True(t, decideBackpressure(true, 80, cfg))
	// 已背压：磁盘回落到低水位以下 → 解除。
	assert.False(t, decideBackpressure(true, 70, cfg))

	// disabled 恒不背压。
	off := cfg
	off.Enabled = false
	assert.False(t, decideBackpressure(true, 99, off))
}

func TestDecideBackpressure_UnknownSignalsFailOpen(t *testing.T) {
	cfg := BackpressureConfig{Enabled: true, DiskHighPct: 85, DiskLowPct: 75}
	// 磁盘不可用(-1) → 不进入背压（fail-open）。
	assert.False(t, decideBackpressure(false, -1, cfg))
	// 已背压 + 磁盘不可用 → 解除（fail-open，不长期误堵）。
	assert.False(t, decideBackpressure(true, -1, cfg))
}

func TestLoadBackpressureConfig_DefaultsAndClamp(t *testing.T) {
	// nil lookup → 全默认。
	def := loadBackpressureConfig(context.Background(), nil)
	assert.True(t, def.Enabled)
	assert.Equal(t, 85.0, def.DiskHighPct)
	assert.Equal(t, 75.0, def.DiskLowPct)
	assert.Equal(t, 30*time.Second, def.Interval)

	// low > high → 夹到 high（防迟滞失效）；interval 过小 → 夹到下限。
	values := map[string]string{
		bpKeyDiskHighPct: "80",
		bpKeyDiskLowPct:  "90", // > high
		bpKeyInterval:    "1",  // < min 5s
		bpKeyEnabled:     "false",
	}
	lookup := func(_ context.Context, category, key string) (string, bool) {
		if category != BackpressureCategory {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}
	cfg := loadBackpressureConfig(context.Background(), lookup)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 80.0, cfg.DiskHighPct)
	assert.Equal(t, 80.0, cfg.DiskLowPct) // clamped to high
	assert.Equal(t, bpMinInterval, cfg.Interval)
}

func TestWatchdog_AllowedReflectsState(t *testing.T) {
	w := NewWatchdog(nil, nil, NewBackpressureMetrics(nil), nil)
	assert.True(t, w.Allowed(), "默认非背压")
	w.active.Store(true)
	assert.False(t, w.Allowed())
}
