// Package software — Prometheus metrics for canary upgrade strategy.
package software

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// CanaryMetrics exposes Prometheus metrics for canary stage transitions
// and rate observations. nil-safe (every method tolerates a nil receiver).
type CanaryMetrics struct {
	stageAdvanceTotal *prometheus.CounterVec
	stageFailureRate  *prometheus.GaugeVec
	activeTasks       prometheus.Gauge
	devicesInStage    *prometheus.GaugeVec
}

// NewCanaryMetrics registers the four canary metrics on the given registry.
// Pass nil to skip registration (the returned metrics still record locally
// for tests using their own Registry).
func NewCanaryMetrics(reg prometheus.Registerer) *CanaryMetrics {
	m := &CanaryMetrics{
		stageAdvanceTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "software_canary_stage_advance_total",
			Help: "Count of canary stage transitions, labelled by outcome (advanced/paused/aborted/threshold_exceeded/completed).",
		}, []string{"result"}),
		stageFailureRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "software_canary_stage_failure_rate",
			Help: "Current failure rate of the active stage (0.0 - 1.0).",
		}, []string{"stage"}),
		activeTasks: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "software_canary_active_tasks",
			Help: "Number of canary tasks currently in 'running' or 'paused' state.",
		}),
		devicesInStage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "software_canary_devices_in_stage",
			Help: "Number of devices that should be active at the end of the named stage.",
		}, []string{"stage"}),
	}
	if reg != nil {
		reg.MustRegister(
			m.stageAdvanceTotal,
			m.stageFailureRate,
			m.activeTasks,
			m.devicesInStage,
		)
	}
	return m
}

// RecordAdvance bumps the advance counter.
func (m *CanaryMetrics) RecordAdvance(result string) {
	if m == nil {
		return
	}
	m.stageAdvanceTotal.WithLabelValues(result).Inc()
}

// SetStageFailureRate publishes the current observed failure rate for a stage.
// stage is 1-indexed.
func (m *CanaryMetrics) SetStageFailureRate(stage int, rate float64) {
	if m == nil {
		return
	}
	m.stageFailureRate.WithLabelValues(strconv.Itoa(stage)).Set(rate)
}

// SetActiveTasks publishes the count of canary tasks in non-terminal states.
func (m *CanaryMetrics) SetActiveTasks(n int) {
	if m == nil {
		return
	}
	m.activeTasks.Set(float64(n))
}

// SetDevicesInStage publishes the device count for a numbered stage.
func (m *CanaryMetrics) SetDevicesInStage(stage, count int) {
	if m == nil {
		return
	}
	m.devicesInStage.WithLabelValues(strconv.Itoa(stage)).Set(float64(count))
}
