package task

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskMetrics_Registered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	require.NotNil(t, m)
	assert.NotNil(t, m.PendingTotal)
	assert.NotNil(t, m.CompletedTotal)
	assert.NotNil(t, m.DurationSeconds)
	assert.NotNil(t, m.NoHandlerTotal)

	// 确认指标真的注册了
	families, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	// CounterVec/HistogramVec 在没有 observation 的情况下不会被 Gather 返回
	// 但 Gauge 会，所以至少 PendingTotal 出现
	assert.True(t, names["omc_tasks_pending_total"], "PendingTotal gauge should be registered")
}

func TestNewTaskMetrics_CounterIncrements(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)

	m.PendingTotal.Inc()
	m.CompletedTotal.WithLabelValues("mml", "completed").Inc()
	m.DurationSeconds.WithLabelValues("mml").Observe(1.5)
	m.NoHandlerTotal.WithLabelValues("unknown-source").Inc()

	families, err := reg.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, families)
}
