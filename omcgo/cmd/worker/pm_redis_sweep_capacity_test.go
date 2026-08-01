package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPMRedisSweepCapacityKeepsAheadOfTwentyThousandHourlyWindows(t *testing.T) {
	const arrivingWindowsPerHour = 20_000

	capacity := int(time.Hour/pmRedisSweepInterval) * pmRedisSweepScanLimit

	require.GreaterOrEqual(t, capacity, arrivingWindowsPerHour*2,
		"published-state cleanup needs recovery headroom above the production arrival rate")
}
