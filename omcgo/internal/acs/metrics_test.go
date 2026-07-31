package acs

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewACSMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewACSMetrics(reg)

	assert.NotNil(t, m)
	assert.NotNil(t, m.ActiveSessions)
	assert.NotNil(t, m.GlobalActiveSessions)
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
