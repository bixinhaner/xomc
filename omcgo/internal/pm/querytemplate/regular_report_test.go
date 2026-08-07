package querytemplate

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeReportAddresses(t *testing.T) {
	addresses, err := normalizeReportAddresses([]string{"NOC@example.com;ops@example.com", "noc@example.com"})
	require.NoError(t, err)
	require.Equal(t, []string{"noc@example.com", "ops@example.com"}, addresses)

	_, err = normalizeReportAddresses([]string{"NOC <noc@example.com>"})
	require.Error(t, err)
	_, err = normalizeReportAddresses([]string{"invalid"})
	require.Error(t, err)
}

func TestValidRegularReportPeriod(t *testing.T) {
	require.True(t, validRegularReportPeriod(RegularReport15Min))
	require.True(t, validRegularReportPeriod(RegularReportHour))
	require.True(t, validRegularReportPeriod(RegularReportDay))
	require.False(t, validRegularReportPeriod("week"))
}

func TestRegularReportService_RejectsEnableUntilRunnerIsAvailable(t *testing.T) {
	service := NewRegularReportService(nil, nil)
	_, err := service.Update(context.Background(), uuid.New(), 1, RegularReportInput{Enabled: true})
	require.ErrorIs(t, err, ErrRegularReportRunnerNotReady)
}

func TestBuildRegularReportExportParams_BindsWindowAndGranularity(t *testing.T) {
	start := time.Date(2026, 8, 6, 10, 15, 0, 0, time.FixedZone("CST", 8*60*60))
	end := start.Add(15 * time.Minute)
	params, err := BuildRegularReportExportParams([]byte(`{"metric_paths":["kpi.a"]}`), RegularReport15Min, start, end)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(params, &decoded))
	require.Equal(t, "kpi.a", decoded["metric_paths"].([]any)[0])
	require.Equal(t, "15min", decoded["granularity"])
	require.Equal(t, start.UTC().Format(time.RFC3339), decoded["start_time"])
	require.Equal(t, end.UTC().Format(time.RFC3339), decoded["end_time"])
}
