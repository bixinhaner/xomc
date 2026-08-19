package parammodel

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryMetricsReusesCollectorsOnSameRegisterer(t *testing.T) {
	reg := prometheus.NewRegistry()

	first := NewRegistryMetrics(reg)
	second := NewRegistryMetrics(reg)

	tests := []struct {
		name   string
		first  prometheus.Collector
		second prometheus.Collector
	}{
		{name: "lookup total", first: first.lookupTotal, second: second.lookupTotal},
		{name: "lookup duration", first: first.lookupDuration, second: second.lookupDuration},
		{name: "cache hit total", first: first.cacheHitTotal, second: second.cacheHitTotal},
		{name: "refresh total", first: first.refreshTotal, second: second.refreshTotal},
		{name: "translate total", first: first.translateTotal, second: second.translateTotal},
		{name: "invalid placeholder total", first: first.invalidPlaceholderTotal, second: second.invalidPlaceholderTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Same(t, tt.first, tt.second)
		})
	}
}

func TestNewRegistryMetricsPanicsWhenExistingCollectorShapeDiffers(t *testing.T) {
	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "param_registry_lookup_total",
		Help: "Conflicting collector used to verify registration failures are not hidden.",
	})))

	require.Panics(t, func() {
		NewRegistryMetrics(reg)
	})
}
