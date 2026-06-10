package event

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBusMetrics_Registered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)
	require.NotNil(t, m)
	assert.NotNil(t, m.DeliveryTotal)

	// CounterVec 无观测时 Gather 不返回，先打一个点再断言注册成功。
	m.inc("pm.file.received", deliveryOutcomeAck)
	families, err := reg.Gather()
	require.NoError(t, err)
	names := map[string]bool{}
	for _, f := range families {
		names[f.GetName()] = true
	}
	assert.True(t, names["omc_eventbus_delivery_total"], "delivery counter should be registered")
}

func TestEventBusMetrics_Inc_ByOutcome(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewEventBusMetrics(reg)

	m.inc("alarm.raised", deliveryOutcomeTerminated)
	m.inc("alarm.raised", deliveryOutcomeTerminated)
	m.inc("alarm.raised", deliveryOutcomeDropped)

	assert.Equal(t, float64(2),
		testutil.ToFloat64(m.DeliveryTotal.WithLabelValues("alarm.raised", deliveryOutcomeTerminated)))
	assert.Equal(t, float64(1),
		testutil.ToFloat64(m.DeliveryTotal.WithLabelValues("alarm.raised", deliveryOutcomeDropped)))
}

// nil 安全：metrics 未注入（单进程 / 单测）时 inc 不 panic。
func TestEventBusMetrics_NilSafe(t *testing.T) {
	var m *EventBusMetrics
	assert.NotPanics(t, func() {
		m.inc("any.subject", deliveryOutcomeAck)
	})
}
