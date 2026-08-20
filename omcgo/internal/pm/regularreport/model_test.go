package regularreport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidatePayload(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidatePayload([]byte(`{"metric_paths":["K001"],"regular_report":{"enabled":true,"send_time":"08:30","periods":["daily","hourly"],"email_enabled":true,"recipients":["ops@example.com"]}}`)))
	require.ErrorContains(t, ValidatePayload([]byte(`{"regular_report":{"enabled":true,"send_time":"8:30","periods":["daily"],"email_enabled":true,"recipients":["ops@example.com"]}}`)), "HH:mm")
	require.ErrorContains(t, ValidatePayload([]byte(`{"regular_report":{"enabled":true,"send_time":"08:30","periods":[],"email_enabled":true,"recipients":["ops@example.com"]}}`)), "periods")
	require.ErrorContains(t, ValidatePayload([]byte(`{"regular_report":{"enabled":true,"send_time":"08:30","periods":["weekly"],"email_enabled":true,"recipients":["ops@example.com"]}}`)), "unsupported")
	require.ErrorContains(t, ValidatePayload([]byte(`{"regular_report":{"enabled":true,"send_time":"08:30","periods":["daily"],"email_enabled":true,"recipients":[]}}`)), "recipients")
	require.ErrorContains(t, ValidatePayload([]byte(`{"metric_paths":[],"regular_report":{"enabled":true,"send_time":"08:30","periods":["daily"],"email_enabled":true,"recipients":["ops@example.com"]}}`)), "metric_path")
	require.ErrorContains(t, ValidatePayload([]byte(`{"metric_paths":["  "],"regular_report":{"enabled":true,"send_time":"08:30","periods":["daily"],"email_enabled":true,"recipients":["ops@example.com"]}}`)), "metric_path")
	require.NoError(t, ValidatePayload([]byte(`{"regular_report":{"enabled":false,"recipients":["legacy-invalid-address"]}}`)))
}

func TestCompleteWindowUsesSystemTimezone(t *testing.T) {
	t.Parallel()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	scheduled := time.Date(2026, 8, 17, 8, 37, 0, 0, location)

	start, end, err := CompleteWindow(scheduled, Period15Min, location)
	require.NoError(t, err)
	require.Equal(t, "2026-08-17 08:15", start.Format("2006-01-02 15:04"))
	require.Equal(t, "2026-08-17 08:30", end.Format("2006-01-02 15:04"))

	start, end, err = CompleteWindow(scheduled, PeriodDaily, location)
	require.NoError(t, err)
	require.Equal(t, "2026-08-16 00:00", start.Format("2006-01-02 15:04"))
	require.Equal(t, "2026-08-17 00:00", end.Format("2006-01-02 15:04"))
}

func TestBuildExportParamsOverridesTemplateWindow(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	got, err := BuildExportParams([]byte(`{"device_sns":["SN-1"],"metric_paths":["K-1"],"device_type":"ENB","granularity":"daily","time_range_preset":"last_30d"}`), PeriodHourly, start, end)
	require.NoError(t, err)
	require.JSONEq(t, `{"device_sns":["SN-1"],"metric_paths":["K-1"],"technologies":["lte"],"granularity":"hourly","dimension":"device","start_time":"2026-08-17T07:00:00Z","end_time":"2026-08-17T08:00:00Z"}`, string(got))
}
