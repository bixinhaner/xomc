package alarm

import "github.com/prometheus/client_golang/prometheus"

// AlarmMetrics holds Prometheus metrics for the alarm management module.
type AlarmMetrics struct {
	ActiveTotal   *prometheus.GaugeVec
	ReceivedTotal *prometheus.CounterVec
}

// NewAlarmMetrics creates and registers alarm metrics.
func NewAlarmMetrics(reg prometheus.Registerer) *AlarmMetrics {
	m := &AlarmMetrics{
		ActiveTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_alarms_active_total",
			Help: "Current number of active alarms by severity and carrier",
		}, []string{"severity", "carrier"}),
		ReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_alarms_received_total",
			Help: "Total number of alarms received by severity",
		}, []string{"severity"}),
	}

	reg.MustRegister(m.ActiveTotal, m.ReceivedTotal)
	return m
}
