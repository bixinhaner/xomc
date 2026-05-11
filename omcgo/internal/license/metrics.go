// Package license — Prometheus metrics for enforcement and capacity.
//
// All metrics are registered via WithMetrics; nil metrics degrades to no-op
// recording so tests and lightweight environments don't need a registry.
package license

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// T-0100-P5-a I3 — 归档字节数按月度统计；监控大对象月度爆增（某月日志异常增大
// 可能预示告警风暴 / 攻击）。用 promauto 注册到 DefaultRegisterer 一次，进程级
// 单例。
var archiveBytesByMonth = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "omc_license_archive_bytes_per_month",
	Help: "License logs archived bytes per YYYY-MM month bucket.",
}, []string{"month"})

// EnforcementMetrics exposes Prometheus metrics for license enforcement
// decisions and capacity state.
type EnforcementMetrics struct {
	activeCount        prometheus.Gauge
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
		activeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_active_count",
			Help: "Current number of active licenses (0 = unprotected mode).",
		}),
		usedDevices: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_used_devices",
			Help: "Current registered device count used as the license-quota numerator.",
		}),
		maxDevices: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_max_devices",
			Help: "Largest MaxDevices among active licenses (the enforcement ceiling).",
		}),
		usageRatio: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "license_capacity_usage_ratio",
			Help: "used_devices / max_devices, in [0, 1].",
		}),
		expiryDaysByID: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "license_expiry_days_remaining",
			Help: "Days until license expiry (-1 for perpetual or unset).",
		}, []string{"license_id"}),
		enforcementByLabel: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "license_enforcement_total",
			Help: "Count of license enforcement decisions, labelled by operation and result.",
		}, []string{"operation", "result"}),
	}

	if reg != nil {
		reg.MustRegister(
			m.activeCount,
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

// SetActiveCount sets the active license gauge.
func (m *EnforcementMetrics) SetActiveCount(n int) {
	if m == nil {
		return
	}
	m.activeCount.Set(float64(n))
}

// SetCapacity sets the capacity gauges in one shot. ratio is computed by the
// caller (Quota method) and may be 0 when max == 0.
func (m *EnforcementMetrics) SetCapacity(used, max int, ratio float64) {
	if m == nil {
		return
	}
	m.usedDevices.Set(float64(used))
	m.maxDevices.Set(float64(max))
	m.usageRatio.Set(ratio)
}

// SetExpiryDaysRemaining records days-remaining for a single license.
// Pass -1 to indicate perpetual/no-expiry.
func (m *EnforcementMetrics) SetExpiryDaysRemaining(licenseID string, days int) {
	if m == nil {
		return
	}
	m.expiryDaysByID.WithLabelValues(licenseID).Set(float64(days))
}
