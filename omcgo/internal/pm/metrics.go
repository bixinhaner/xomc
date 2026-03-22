package pm

import "github.com/prometheus/client_golang/prometheus"

// PMMetrics holds Prometheus metrics for the PM collection module.
type PMMetrics struct {
	FilesProcessedTotal    *prometheus.CounterVec
	ProcessingDurationSecs prometheus.Histogram
}

// NewPMMetrics creates and registers PM metrics.
func NewPMMetrics(reg prometheus.Registerer) *PMMetrics {
	m := &PMMetrics{
		FilesProcessedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_pm_files_processed_total",
			Help: "Total number of PM files processed by status",
		}, []string{"status"}),
		ProcessingDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_pm_processing_duration_seconds",
			Help:    "Duration of PM file processing in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}),
	}

	reg.MustRegister(m.FilesProcessedTotal, m.ProcessingDurationSecs)
	return m
}
