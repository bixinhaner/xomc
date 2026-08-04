package alarm

import "github.com/prometheus/client_golang/prometheus"

// AlarmMetrics holds Prometheus metrics for the alarm management module.
type AlarmMetrics struct {
	ActiveTotal            *prometheus.GaugeVec
	ReceivedTotal          *prometheus.CounterVec
	ReconciliationTotal    *prometheus.CounterVec
	OutboxBacklog          prometheus.Gauge
	OutboxOldestAgeSeconds prometheus.Gauge
	OutboxPublishedTotal   prometheus.Counter
	OutboxRetryTotal       prometheus.Counter
	OutboxDeadTotal        prometheus.Counter
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
		ReconciliationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_alarm_reconciliation_total",
			Help: "Total number of alarm reconciliation outcomes",
		}, []string{"result"}),
		OutboxBacklog: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_alarm_outbox_backlog",
			Help: "Current due or publishing alarm lifecycle Outbox events.",
		}),
		OutboxOldestAgeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "omc_alarm_outbox_oldest_age_seconds",
			Help: "Age in seconds of the oldest due or publishing alarm lifecycle Outbox event.",
		}),
		OutboxPublishedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_alarm_outbox_publish_total",
			Help: "Total alarm lifecycle Outbox events acknowledged by JetStream.",
		}),
		OutboxRetryTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_alarm_outbox_retry_total",
			Help: "Total failed alarm lifecycle Outbox publication attempts scheduled for retry.",
		}),
		OutboxDeadTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_alarm_outbox_dead_total",
			Help: "Total alarm lifecycle Outbox events moved to the dead state.",
		}),
	}

	if reg != nil {
		reg.MustRegister(
			m.ActiveTotal, m.ReceivedTotal, m.ReconciliationTotal,
			m.OutboxBacklog, m.OutboxOldestAgeSeconds, m.OutboxPublishedTotal,
			m.OutboxRetryTotal, m.OutboxDeadTotal,
		)
	}
	return m
}
