package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultDays_Complete(t *testing.T) {
	for _, k := range AllKeys() {
		days, ok := DefaultDays[k]
		require.True(t, ok, "default missing for %s", k)
		require.GreaterOrEqual(t, days, MinRetentionDays, "default too short for %s", k)
		require.LessOrEqual(t, days, MaxRetentionDays, "default too long for %s", k)
	}
}

func TestDefaultDays_PyramidShape(t *testing.T) {
	// 金字塔保留：粒度越粗保留越长（或同等长）。
	require.LessOrEqual(t, DefaultDays[KeyRaw15MinDays], DefaultDays[KeyHourlyDays])
	require.LessOrEqual(t, DefaultDays[KeyHourlyDays], DefaultDays[KeyDailyDays])
	require.LessOrEqual(t, DefaultDays[KeyDailyDays], DefaultDays[KeyWeeklyDays])
	require.LessOrEqual(t, DefaultDays[KeyWeeklyDays], DefaultDays[KeyMonthlyDays])
}

func TestKeyForGranularity_Bidirectional(t *testing.T) {
	for _, g := range []Granularity{
		Granularity15Min, GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
	} {
		k, ok := KeyForGranularity(g)
		require.True(t, ok, "no key for %s", g)
		g2, ok2 := GranularityForKey(k)
		require.True(t, ok2)
		require.Equal(t, g, g2)
	}

	_, ok := KeyForGranularity(Granularity("unknown"))
	require.False(t, ok)

	_, ok = GranularityForKey(PolicyKey("unknown"))
	require.False(t, ok)
}

func TestDefaultDuration(t *testing.T) {
	got := DefaultDuration(KeyRaw15MinDays)
	want := time.Duration(30) * 24 * time.Hour
	require.Equal(t, want, got)

	require.Equal(t, time.Duration(0), DefaultDuration(PolicyKey("unknown")))
}

func TestValidateDays(t *testing.T) {
	require.NoError(t, ValidateDays(1))
	require.NoError(t, ValidateDays(30))
	require.NoError(t, ValidateDays(3650))

	require.ErrorIs(t, ValidateDays(0), ErrRetentionTooShort)
	require.ErrorIs(t, ValidateDays(-1), ErrRetentionTooShort)
	require.ErrorIs(t, ValidateDays(3651), ErrRetentionTooLong)
}
