package admin

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestAdminMetricsObservesConfigApplyResult(t *testing.T) {
	metrics := NewAdminMetrics(prometheus.NewRegistry())

	metrics.ObserveConfigApply(ConfigApplyObservation{
		Category: "storage",
		Target:   "minio_presign_endpoint",
		Result:   ConfigApplyObservationFailed,
		Err:      errors.New("minio unavailable"),
	})

	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.ConfigApplyAttempts.WithLabelValues("storage", "minio_presign_endpoint", ConfigApplyObservationFailed),
	))
}
