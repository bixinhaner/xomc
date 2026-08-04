package acs

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metricHelp(t *testing.T, reg *prometheus.Registry, name string) string {
	t.Helper()

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family.GetHelp()
		}
	}
	require.FailNow(t, "metric not registered", name)
	return ""
}

func TestNewACSMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewACSMetrics(reg)

	assert.NotNil(t, m)
	assert.NotNil(t, m.ActiveSessions)
	assert.NotNil(t, m.GlobalActiveSessions)
	assert.NotNil(t, m.LocalTrackedSessions)
	assert.Equal(t,
		"Deprecated compatibility alias for acs_global_active_sessions; current number of globally admitted TR069 sessions",
		metricHelp(t, reg, "acs_active_sessions"),
	)
	assert.Equal(t,
		"Current number of globally admitted TR069 sessions from the shared admission controller",
		metricHelp(t, reg, "acs_global_active_sessions"),
	)
	assert.Equal(t,
		"Current number of session IDs retained by this ACS process for up to five minutes; not real-time concurrency",
		metricHelp(t, reg, "acs_local_tracked_sessions"),
	)
	assert.NotNil(t, m.InformTotal)
	assert.NotNil(t, m.RPCDuration)
	assert.NotNil(t, m.RPCErrorsTotal)
	assert.NotNil(t, m.SessionDuration)
	assert.NotNil(t, m.AdmissionRejected)
	assert.NotNil(t, m.UECountQueueDepth)
	assert.NotNil(t, m.UECountQueueCapacity)
	assert.NotNil(t, m.UECountEnqueueTotal)
	assert.NotNil(t, m.UECountProcessTotal)
	assert.NotNil(t, m.UECountProcessDuration)
}

func TestNewACSMetrics_Operations(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewACSMetrics(reg)

	assert.NotPanics(t, func() {
		m.ActiveSessions.Inc()
		m.ActiveSessions.Dec()
		m.ActiveSessions.Set(42)
		m.GlobalActiveSessions.Set(42)
		m.LocalTrackedSessions.Set(42)
	})

	assert.NotPanics(t, func() {
		m.InformTotal.WithLabelValues("0 BOOTSTRAP").Inc()
		m.InformTotal.WithLabelValues("2 PERIODIC").Inc()
	})

	assert.NotPanics(t, func() {
		m.RPCDuration.WithLabelValues("GetParameterValues").Observe(0.123)
		m.RPCErrorsTotal.WithLabelValues("Download").Inc()
		m.SessionDuration.Observe(2.5)
		m.AdmissionRejected.Inc()
		m.UECountQueueDepth.Set(12)
		m.UECountQueueCapacity.Set(4096)
		m.UECountEnqueueTotal.WithLabelValues("accepted").Inc()
		m.UECountProcessTotal.WithLabelValues("success").Inc()
		m.UECountProcessDuration.Observe(0.02)
	})
}

func TestNewACSMetrics_DuplicateRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	_ = NewACSMetrics(reg)

	assert.Panics(t, func() {
		NewACSMetrics(reg)
	})
}
