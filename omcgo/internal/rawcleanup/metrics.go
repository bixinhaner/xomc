package rawcleanup

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	Candidates *prometheus.GaugeVec
	Deleted    *prometheus.CounterVec
	Failed     *prometheus.CounterVec
	Rate       prometheus.Gauge
	Duration   prometheus.Histogram
	OldestAge  *prometheus.GaugeVec
	Paused     *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Candidates: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "omc_raw_cleanup_candidates", Help: "Raw object cleanup candidates in the last batch."}, []string{"kind"}),
		Deleted:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "omc_raw_cleanup_deleted_total", Help: "Raw objects deleted by kind."}, []string{"kind"}),
		Failed:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "omc_raw_cleanup_failed_total", Help: "Raw object deletion failures."}, []string{"kind", "reason"}),
		Rate:       prometheus.NewGauge(prometheus.GaugeOpts{Name: "omc_raw_cleanup_rate_target", Help: "Current precise cleanup target objects per second."}),
		Duration:   prometheus.NewHistogram(prometheus.HistogramOpts{Name: "omc_raw_cleanup_batch_duration_seconds", Help: "Exact deletion batch duration.", Buckets: prometheus.DefBuckets}),
		OldestAge:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "omc_raw_cleanup_oldest_expired_age_seconds", Help: "Age beyond raw-object expiry of the oldest candidate."}, []string{"kind"}),
		Paused:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "omc_raw_cleanup_paused", Help: "Whether cleanup is paused for a bounded reason."}, []string{"reason"}),
	}
	if reg != nil {
		reg.MustRegister(m.Candidates, m.Deleted, m.Failed, m.Rate, m.Duration, m.OldestAge, m.Paused)
	}
	return m
}
