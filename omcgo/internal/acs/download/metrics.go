// Package download — Prometheus metrics for on-the-fly decompression (T-0072).
package download

import "github.com/prometheus/client_golang/prometheus"

// DecompressMetrics holds the download-side decompression collectors.
// All Record* methods are nil-safe so production wiring (with registry) and
// tests (without) share the same call sites.
type DecompressMetrics struct {
	successTotal *prometheus.CounterVec
	errorsTotal  *prometheus.CounterVec
}

// NewDecompressMetrics registers the collectors on the given registry.
// Pass nil for tests; the returned struct is still usable.
func NewDecompressMetrics(reg prometheus.Registerer) *DecompressMetrics {
	m := &DecompressMetrics{
		successTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_download_decompress_total",
			Help: "Successful on-the-fly decompression events on the backup download path, by format.",
		}, []string{"format"}),
		errorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_backup_download_decompress_errors_total",
			Help: "Decompression failures by format and reason (open|copy).",
		}, []string{"format", "reason"}),
	}
	if reg != nil {
		reg.MustRegister(m.successTotal, m.errorsTotal)
	}
	return m
}

// RecordSuccess increments the success counter for the given format.
func (m *DecompressMetrics) RecordSuccess(format string) {
	if m == nil {
		return
	}
	m.successTotal.WithLabelValues(format).Inc()
}

// RecordError increments the error counter. reason ∈ {"open", "copy"}.
func (m *DecompressMetrics) RecordError(format, reason string) {
	if m == nil {
		return
	}
	m.errorsTotal.WithLabelValues(format, reason).Inc()
}
