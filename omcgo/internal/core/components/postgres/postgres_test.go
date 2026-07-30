package postgres

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildQueryTracerEnablesSlowQueryMonitoringWithoutSQLDebugLogging(t *testing.T) {
	cfg := appconfig.PostgresConfig{
		LogSQL:              false,
		LogSQLSlowThreshold: 750,
	}
	reg := prometheus.NewRegistry()

	tracer := buildQueryTracer(cfg, zap.NewNop(), reg)

	slowTracer, ok := tracer.(*SlowQueryTracer)
	require.True(t, ok)
	assert.Equal(t, 750*time.Millisecond, slowTracer.Threshold())
}

func TestBuildQueryTracerChainsSQLLoggingAndSlowQueryMonitoring(t *testing.T) {
	cfg := appconfig.PostgresConfig{
		LogSQL:              true,
		LogSQLParams:        true,
		LogSQLSlowThreshold: 1000,
	}

	tracer := buildQueryTracer(cfg, zap.NewNop(), prometheus.NewRegistry())

	chain, ok := tracer.(*chainedTracer)
	require.True(t, ok)
	_, sqlLoggingEnabled := chain.a.(*OTELSQLTracer)
	_, slowMonitoringEnabled := chain.b.(*SlowQueryTracer)
	assert.True(t, sqlLoggingEnabled)
	assert.True(t, slowMonitoringEnabled)
}

func TestBuildQueryTracerDisabled(t *testing.T) {
	cfg := appconfig.PostgresConfig{}

	assert.Nil(t, buildQueryTracer(cfg, zap.NewNop(), prometheus.NewRegistry()))
	assert.Nil(t, buildQueryTracer(appconfig.PostgresConfig{LogSQL: true}, nil, prometheus.NewRegistry()))
}
