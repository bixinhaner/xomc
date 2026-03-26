package device

import "github.com/prometheus/client_golang/prometheus"

// DeviceMetrics holds Prometheus metrics for the device management module.
type DeviceMetrics struct {
	DevicesTotal       *prometheus.GaugeVec
	RegistrationsTotal *prometheus.CounterVec

	// 批量处理器指标
	BatchFlushTotal    prometheus.Counter
	BatchFlushSize     prometheus.Histogram
	BatchFlushDuration prometheus.Histogram
	BatchFlushFailed   prometheus.Counter
	BatchDropped       prometheus.Counter
}

// NewDeviceMetrics creates and registers device metrics.
func NewDeviceMetrics(reg prometheus.Registerer) *DeviceMetrics {
	m := &DeviceMetrics{
		DevicesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "omc_devices_total",
			Help: "Current number of devices by status and carrier",
		}, []string{"status", "carrier"}),
		RegistrationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_device_registrations_total",
			Help: "Total number of device registrations",
		}, []string{"carrier"}),

		BatchFlushTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_device_batch_flush_total",
			Help: "Total number of batch flush operations",
		}),
		BatchFlushSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_device_batch_flush_size",
			Help:    "Number of devices per batch flush",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 200, 500},
		}),
		BatchFlushDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_device_batch_flush_duration_seconds",
			Help:    "Duration of batch flush operations in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		BatchFlushFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_device_batch_flush_failed_total",
			Help: "Total number of batch flush failures after retries",
		}),
		BatchDropped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_device_batch_dropped_total",
			Help: "Total number of inform events dropped due to full worker channel",
		}),
	}

	reg.MustRegister(
		m.DevicesTotal, m.RegistrationsTotal,
		m.BatchFlushTotal, m.BatchFlushSize, m.BatchFlushDuration,
		m.BatchFlushFailed, m.BatchDropped,
	)
	return m
}
