package geofence

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type BatchMetrics struct {
	itemsTotal   *prometheus.CounterVec
	bindDuration prometheus.Histogram
	bindSize     prometheus.Histogram
}

func NewBatchMetrics(registerer prometheus.Registerer) *BatchMetrics {
	metrics := &BatchMetrics{
		itemsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_geofence_batch_items_total",
			Help: "Manual geofence binding item outcomes by bounded status and reason.",
		}, []string{"status", "reason_code"}),
		bindDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_geofence_batch_bind_duration_seconds",
			Help:    "Duration of one manual geofence binding runner invocation.",
			Buckets: prometheus.DefBuckets,
		}),
		bindSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "omc_geofence_batch_bind_size",
			Help:    "Number of devices selected in one manual geofence binding run.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
	}
	if registerer != nil {
		registerer.MustRegister(
			metrics.itemsTotal,
			metrics.bindDuration,
			metrics.bindSize,
		)
	}
	return metrics
}

func (m *BatchMetrics) observeItem(status BatchItemStatus, reasonCode string) {
	if m == nil {
		return
	}
	m.itemsTotal.WithLabelValues(
		boundedBatchMetricStatus(status),
		boundedBatchMetricReason(reasonCode),
	).Inc()
}

func (m *BatchMetrics) observeBind(duration time.Duration, size int) {
	if m == nil {
		return
	}
	m.bindDuration.Observe(duration.Seconds())
	m.bindSize.Observe(float64(size))
}

func boundedBatchMetricStatus(status BatchItemStatus) string {
	switch status {
	case BatchItemPending, BatchItemSucceeded, BatchItemSkipped, BatchItemFailed:
		return string(status)
	default:
		return "unknown"
	}
}

func boundedBatchMetricReason(reasonCode string) string {
	switch reasonCode {
	case "":
		return "none"
	case ReasonDeviceUnavailable,
		ReasonGeofenceNotEnabled,
		ReasonCarrierMismatch,
		ReasonBaselineOwnerMismatch,
		ReasonAlreadyBound,
		ReasonBindingSuspended,
		ReasonActiveRuleConflict,
		ReasonBindingChanged,
		ReasonReassigned,
		ReasonGeofenceVersionChanged,
		ReasonProcessingError:
		return reasonCode
	default:
		return "other"
	}
}
