// Package software — Prometheus metrics for firmware integrity / signature
// verification before南向下发 (issue #8).
//
// Two counters cover the audit needs:
//   - software_firmware_verify_total{result}        — verification outcomes, by
//     result: "pass" / "fail" / "legacy_md5" (passed but on the MD5 fallback path)
//   - software_firmware_verify_fail_total{code}     — verification failures, by
//     FailureCode (INTEGRITY_CHECK_FAILED / SIGNATURE_CHECK_FAILED)
//
// Both are nil-safe so production wiring (DI passes a registry) and tests (no
// registry) share one method surface — mirrors RollbackMetrics / CanaryMetrics.
package software

import "github.com/prometheus/client_golang/prometheus"

// FirmwareMetrics holds the verification-specific Prometheus collectors.
type FirmwareMetrics struct {
	verifyTotal      *prometheus.CounterVec
	verifyFailByCode *prometheus.CounterVec
}

// Verification result labels for software_firmware_verify_total{result}.
const (
	verifyResultPass      = "pass"
	verifyResultFail      = "fail"
	verifyResultLegacyMD5 = "legacy_md5"
)

// NewFirmwareMetrics registers the firmware verification metrics on the given
// registry. Pass nil for tests; the returned struct is still usable.
func NewFirmwareMetrics(reg prometheus.Registerer) *FirmwareMetrics {
	m := &FirmwareMetrics{
		verifyTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "software_firmware_verify_total",
			Help: "Count of firmware verification attempts before南向下发, labelled by result (pass/fail/legacy_md5).",
		}, []string{"result"}),
		verifyFailByCode: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "software_firmware_verify_fail_total",
			Help: "Count of firmware verification failures, labelled by failure code (INTEGRITY_CHECK_FAILED/SIGNATURE_CHECK_FAILED).",
		}, []string{"code"}),
	}
	if reg != nil {
		reg.MustRegister(m.verifyTotal, m.verifyFailByCode)
	}
	return m
}

// RecordPass bumps the pass counter. legacyMD5 distinguishes the SHA-256 path
// from the MD5 fallback path so operators can track存量固件迁移进度.
func (m *FirmwareMetrics) RecordPass(legacyMD5 bool) {
	if m == nil {
		return
	}
	if legacyMD5 {
		m.verifyTotal.WithLabelValues(verifyResultLegacyMD5).Inc()
		return
	}
	m.verifyTotal.WithLabelValues(verifyResultPass).Inc()
}

// RecordFail bumps the fail counter and the per-code failure counter.
func (m *FirmwareMetrics) RecordFail(code FailureCode) {
	if m == nil {
		return
	}
	m.verifyTotal.WithLabelValues(verifyResultFail).Inc()
	m.verifyFailByCode.WithLabelValues(string(code)).Inc()
}
