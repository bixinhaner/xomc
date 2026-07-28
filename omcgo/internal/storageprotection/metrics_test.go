package storageprotection

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestStorageProtectionMetrics(t *testing.T) {
	m := NewMetrics(prometheus.NewRegistry())
	m.AdmissionState.WithLabelValues("filesystem", "root", "all").Set(stateValue(StateBlocked))
	m.WriteRejectedTotal.WithLabelValues("filesystem", "root", "all", "blocked").Inc()
	require.Equal(t, float64(1), testutil.ToFloat64(m.AdmissionState.WithLabelValues("filesystem", "root", "all")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.WriteRejectedTotal.WithLabelValues("filesystem", "root", "all", "blocked")))
}
