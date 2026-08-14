package reportsubscription

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNextRunAtUsesConfiguredTimezone(t *testing.T) {
	t.Parallel()
	location := time.FixedZone("UTC+8", 8*60*60)
	after := time.Date(2026, 8, 11, 7, 30, 0, 0, location)
	next, err := NextRunAt(after, []string{"20:00", "08:00", "08:00"}, location)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 11, 8, 0, 0, 0, location), next)
}

func TestNaturalWindow(t *testing.T) {
	t.Parallel()
	location := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 11, 13, 53, 0, 0, location)
	tests := []struct {
		period Period
		start  time.Time
		end    time.Time
	}{
		{Period15Min, time.Date(2026, 8, 11, 13, 30, 0, 0, location), time.Date(2026, 8, 11, 13, 45, 0, 0, location)},
		{PeriodHourly, time.Date(2026, 8, 11, 12, 0, 0, 0, location), time.Date(2026, 8, 11, 13, 0, 0, 0, location)},
		{PeriodDaily, time.Date(2026, 8, 10, 0, 0, 0, 0, location), time.Date(2026, 8, 11, 0, 0, 0, 0, location)},
	}
	for _, tt := range tests {
		start, end, err := NaturalWindow(tt.period, now, location)
		require.NoError(t, err)
		require.Equal(t, tt.start, start)
		require.Equal(t, tt.end, end)
	}
}

func TestScheduledWindowCollapsesDowntimeToLatestClosedWindow(t *testing.T) {
	t.Parallel()
	location := time.FixedZone("UTC+8", 8*60*60)
	dueAt := time.Date(2026, 8, 1, 8, 0, 0, 0, location)
	now := time.Date(2026, 8, 9, 12, 30, 0, 0, location)
	start, end, nextRun, err := scheduledWindow(dueSubscription{Subscription: Subscription{
		Period: PeriodDaily, SendTimes: []string{"08:00"}, NextRunAt: &dueAt,
	}}, now, location)

	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 8, 0, 0, 0, 0, location), start)
	require.Equal(t, time.Date(2026, 8, 9, 0, 0, 0, 0, location), end)
	require.Equal(t, time.Date(2026, 8, 10, 8, 0, 0, 0, location), nextRun)
	require.True(t, nextRun.After(now))
}
