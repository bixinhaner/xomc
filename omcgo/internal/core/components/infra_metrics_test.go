package components

import (
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/stretchr/testify/require"
)

// TestNewInfra_ExposesGoRuntimeMetrics 验证 NewInfra 构造的 MetricsReg 注册了
// Go runtime 采集器 —— 这是「服务资源监控」的运行时基线（协程/堆/GC）。
// 修复前 registry 是空的 prometheus.NewRegistry()，go_goroutines 等不会暴露，
// 导致 :3030 的协程/内存面板与 connection-pool 告警全部失灵。
func TestNewInfra_ExposesGoRuntimeMetrics(t *testing.T) {
	inf, err := NewInfra(appconfig.LogConfig{Level: "error", Format: "console", OutputPaths: []string{"stdout"}}, 0)
	require.NoError(t, err)

	families, err := inf.MetricsReg.Gather()
	require.NoError(t, err)

	names := make(map[string]bool, len(families))
	hasMemstats := false
	for _, f := range families {
		names[f.GetName()] = true
		if strings.HasPrefix(f.GetName(), "go_memstats_") {
			hasMemstats = true
		}
	}

	// go_goroutines 是跨平台稳定的运行时指标，是协程监控的直接数据源。
	require.True(t, names["go_goroutines"], "Infra.MetricsReg 必须暴露 go_goroutines（Go 采集器未接线）")
	// 堆内存指标族（go_memstats_*）证明 GoCollector 已完整注册。
	require.True(t, hasMemstats, "Infra.MetricsReg 必须暴露 go_memstats_*（Go 采集器未接线）")
}
