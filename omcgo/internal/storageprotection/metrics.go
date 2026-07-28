package storageprotection

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	CapacityBytes      *prometheus.GaugeVec
	UsedBytes          *prometheus.GaugeVec
	UsedRatio          *prometheus.GaugeVec
	AdmissionState     *prometheus.GaugeVec
	WriteRejectedTotal *prometheus.CounterVec
	WriteDegradedTotal *prometheus.CounterVec
	CheckFailuresTotal *prometheus.CounterVec
	PolicyInfo         *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_storage_capacity_bytes",
			Help: "Observed storage capacity by target and write scope.",
		}, []string{"target_type", "target_id", "write_scope"}),
		UsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_storage_used_bytes",
			Help: "Observed storage used bytes by target and write scope.",
		}, []string{"target_type", "target_id", "write_scope"}),
		UsedRatio: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_storage_used_ratio",
			Help: "Observed storage used ratio by target and write scope.",
		}, []string{"target_type", "target_id", "write_scope"}),
		AdmissionState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_storage_admission_state",
			Help: "Storage admission state: normal=0, warning=0, blocked=1, unknown=2.",
		}, []string{"target_type", "target_id", "write_scope"}),
		WriteRejectedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_storage_write_rejected_total",
			Help: "OMC business writes rejected by storage protection.",
		}, []string{"target_type", "target_id", "write_scope", "reason"}),
		WriteDegradedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_storage_write_degraded_total",
			Help: "OMC writes degraded because storage protection is warning or unknown.",
		}, []string{"target_type", "target_id", "write_scope", "reason"}),
		CheckFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_storage_check_failures_total",
			Help: "Storage capacity/admission check failures.",
		}, []string{"target_type", "target_id", "write_scope"}),
		PolicyInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_storage_policy_info",
			Help: "Configured storage protection policy thresholds.",
		}, []string{"target_type", "target_id", "write_scope", "enabled", "unknown_behavior"}),
	}
	if reg != nil {
		reg.MustRegister(m.CapacityBytes, m.UsedBytes, m.UsedRatio, m.AdmissionState,
			m.WriteRejectedTotal, m.WriteDegradedTotal, m.CheckFailuresTotal, m.PolicyInfo)
	}
	return m
}

func stateValue(state State) float64 {
	if state == StateBlocked {
		return 1
	}
	if state == StateUnknown {
		return 2
	}
	return 0
}
