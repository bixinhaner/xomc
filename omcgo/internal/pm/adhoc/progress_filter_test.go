package adhoc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
)

func TestProgressResultDTOsAppliesDashboardDimensionAndCalendarFilters(t *testing.T) {
	start := time.Date(2026, 7, 27, 3, 0, 0, 0, time.UTC) // Asia/Shanghai Monday 11:00
	targetProduct := uuid.New().String()
	rows := []pmstream.ProgressResult{
		{
			ID: uuid.New(), TaskID: uuid.New(), TaskVersionID: uuid.New(),
			Granularity: pmstream.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
			Dimension: pmstream.DimensionProduct, DimensionKey: targetProduct,
			MetricPath: "K900010076",
		},
		{
			ID: uuid.New(), TaskID: uuid.New(), TaskVersionID: uuid.New(),
			Granularity: pmstream.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
			Dimension: pmstream.DimensionProduct, DimensionKey: uuid.New().String(),
			MetricPath: "K900010076",
		},
		{
			ID: uuid.New(), TaskID: uuid.New(), TaskVersionID: uuid.New(),
			Granularity: pmstream.GranularityDaily,
			WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
			Dimension: pmstream.DimensionNetwork, DimensionKey: "Network",
			MetricPath: "K900010076",
		},
	}
	filter := resultsFilter{
		ProductIDs: []string{targetProduct},
		SubsetLDNs: []string{targetProduct},
		Weekdays: []int{1}, Hours: []int{11},
		CalendarTimezone: "Asia/Shanghai",
	}

	got := progressResultDTOs(rows, filter)

	require.Len(t, got, 1)
	require.Equal(t, targetProduct, got[0].ProductID)

	filter.Hours = []int{12}
	require.Empty(t, progressResultDTOs(rows, filter))
}
