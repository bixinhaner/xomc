// Package license — Prometheus metrics for enforcement and capacity.
//
// Step 5 起：老 license_active_count gauge（multi-license 模型 active 数量）
// 已删除；system_license singleton 模型下"是否有 license"由 capacity_max=0
// 反映（max=0 = 无 license，受控业务 fail-closed 拒绝），不必单列 active_count。
//
// All metrics are registered via NewEnforcementMetrics; nil metrics degrades
// to no-op recording so tests and lightweight environments don't need a registry.
package license

import (
	"github.com/prometheus/client_golang/prometheus"
)

// EnforcementMetrics exposes Prometheus metrics for license enforcement
// decisions and capacity state.
type EnforcementMetrics struct {
	usedDevices        prometheus.Gauge
	maxDevices         prometheus.Gauge
	usageRatio         prometheus.Gauge
	expiryDaysByID     *prometheus.GaugeVec
	enforcementByLabel *prometheus.CounterVec
}

// NewEnforcementMetrics registers the license enforcement metrics on the
// given registry. registry may be a custom *prometheus.Registry or
// prometheus.DefaultRegisterer; tests typically pass a fresh Registry.
func NewEnforcementMetrics(reg prometheus.Registerer) *EnforcementMetrics {
	m := &EnforcementMetrics{
		usedDevices: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_used_devices",
			Help: "Current registered device count used as the license-quota numerator.",
		}),
		maxDevices: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_max_devices",
			Help: "Sum of system_license.devices_support quotas (0 = no license; operations denied fail-closed).",
		}),
		usageRatio: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_usage_ratio",
			Help: "used_devices / max_devices, in [0, 1].",
		}),
		expiryDaysByID: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "license_expiry_days_remaining",
			Help: "Days until license expiry (-1 for perpetual or unset). Label is always 'system'.",
		}, []string{"license_id"}),
		enforcementByLabel: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "license_enforcement_total",
			Help: "Count of license enforcement decisions, labelled by operation and result.",
		}, []string{"operation", "result"}),
	}

	if reg != nil {
		reg.MustRegister(
			m.usedDevices,
			m.maxDevices,
			m.usageRatio,
			m.expiryDaysByID,
			m.enforcementByLabel,
		)
	}
	return m
}

// RecordEnforcement bumps the per-operation/result counter.
func (m *EnforcementMetrics) RecordEnforcement(operation, result string) {
	if m == nil {
		return
	}
	m.enforcementByLabel.WithLabelValues(operation, result).Inc()
}

// SetCapacity sets the capacity gauges in one shot. ratio is computed by the
// caller (Quota/CheckCapacity path) and may be 0 when max == 0.
func (m *EnforcementMetrics) SetCapacity(used, max int, ratio float64) {
	if m == nil {
		return
	}
	m.usedDevices.Set(float64(used))
	m.maxDevices.Set(float64(max))
	m.usageRatio.Set(ratio)
}

// SetExpiryDaysRemaining records days-remaining for the (singleton) system
// license. Pass -1 to indicate perpetual/no-expiry.
func (m *EnforcementMetrics) SetExpiryDaysRemaining(licenseID string, days int) {
	if m == nil {
		return
	}
	m.expiryDaysByID.WithLabelValues(licenseID).Set(float64(days))
}
