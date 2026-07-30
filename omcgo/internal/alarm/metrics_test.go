package alarm

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAlarmMetrics_ReconciliationOutcomesAreCounted(t *testing.T) {
	metrics := NewAlarmMetrics(prometheus.NewRegistry())
	outcomes := []string{
		"clear_exact_miss",
		"clear_unique_fallback",
		"clear_ambiguous",
		"sync_duplicate_cleared",
	}

	for _, outcome := range outcomes {
		metrics.ReconciliationTotal.WithLabelValues(outcome).Inc()
		assert.Equal(t, float64(1), testutil.ToFloat64(metrics.ReconciliationTotal.WithLabelValues(outcome)))
	}
}
