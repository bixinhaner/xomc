package device

import "github.com/prometheus/client_golang/prometheus"

// DeviceMetrics holds Prometheus metrics for the device management module.
type DeviceMetrics struct {
	DevicesTotal       *prometheus.GaugeVec
	RegistrationsTotal *prometheus.CounterVec
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
	}

	reg.MustRegister(m.DevicesTotal, m.RegistrationsTotal)
	return m
}
