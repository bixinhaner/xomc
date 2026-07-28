package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWindowForUsesBusinessTimezoneBoundaries(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	slot := time.Date(2026, 7, 25, 16, 15, 0, 0, time.UTC)

	daily, err := WindowFor(slot, GranularityDaily, shanghai)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 7, 25, 16, 0, 0, 0, time.UTC), daily.Start)
	require.Equal(t, time.Date(2026, 7, 26, 16, 0, 0, 0, time.UTC), daily.End)

	monthly, err := WindowFor(slot, GranularityMonthly, shanghai)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC), monthly.Start)
	require.Equal(t, time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC), monthly.End)
}
