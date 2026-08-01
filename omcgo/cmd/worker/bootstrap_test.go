package main

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestWorkerDoesNotExportDuplicateRedisQueueObserverSeries(t *testing.T) {
	reg := prometheus.NewRegistry()

	newWorkerTaskMetrics(reg)

	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.False(t, strings.HasPrefix(family.GetName(), "omc_redis_task_queue_"),
			"worker must leave Redis queue observation to the app process")
	}
}
