package stream

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowShardCountScalesWithEntityCount(t *testing.T) {
	require.Equal(t, 1, windowShardCount(4, GranularityHourly))
	require.Equal(t, 16, windowShardCount(40_000, GranularityHourly))
	require.Equal(t, 16, windowShardCount(240_000, GranularityDaily))
	require.Equal(t, 16, windowShardCount(70_000, GranularityWeekly))
	require.Equal(t, 16, windowShardCount(310_000, GranularityMonthly))
}

func TestWindowShardIsStableAndBounded(t *testing.T) {
	first := windowShard("device-10001", 16)
	require.Equal(t, first, windowShard("device-10001", 16))
	require.GreaterOrEqual(t, first, 0)
	require.Less(t, first, 16)
}

func TestAccumulatorDefinitionIDIsCompactAndStable(t *testing.T) {
	value := ContributionValue{
		Dimension: DimensionDevice, DimensionKey: "device-10001",
		ObjectLDN:  "Device.Services.FAPService.1.CellConfig.LTE.RAN",
		MetricPath: "pmRadioRecInterferencePwr", Operation: AggregationAvg,
	}
	first, err := accumulatorDefinitionID(value)
	require.NoError(t, err)
	second, err := accumulatorDefinitionID(value)
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.Len(t, first, 24)
	require.NotContains(t, first, "device-10001")
}
