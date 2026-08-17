package alarm

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRenderAlarmEmailContainsActiveAndClearedDetails(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	window := AlarmEmailWindow{
		Start: time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 17, 5, 10, 0, 0, time.UTC),
	}
	clearedAt := time.Date(2026, 8, 17, 5, 7, 0, 0, time.UTC)
	subject, body, err := RenderAlarmEmail("Test OMC", location, window, []AlarmEmailItem{
		{
			AlarmIdentifier: "11184",
			Severity:        1,
			ProbableCause:   "Cell unavailable",
			DeviceLabel:     "SN-001",
			RaisedAt:        time.Date(2026, 8, 17, 5, 2, 0, 0, time.UTC),
			Description:     "Radio unavailable",
			Advice:          "Check RF status",
		},
		{
			AlarmIdentifier: "11217",
			Severity:        2,
			ProbableCause:   "Packet loss",
			DeviceLabel:     "SN-002",
			RaisedAt:        time.Date(2026, 8, 17, 5, 3, 0, 0, time.UTC),
			ClearedAt:       &clearedAt,
			Description:     "Transient packet loss",
			Advice:          "Check transport link",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "[Test OMC] 告警通知 / Alarm Notification", subject)
	require.Contains(t, body, "活动告警 / Active: 1")
	require.Contains(t, body, "清除告警 / Cleared: 1")
	require.Contains(t, body, "2026-08-17 13:07:00")
	require.Contains(t, body, "Check transport link")
	require.Contains(t, body, "紧急 / Critical")
	require.Contains(t, body, "重要 / Major")
}

func TestAlarmEmailSeverityLabelNormalizesLegacyCodes(t *testing.T) {
	require.Equal(t, "紧急 / Critical", alarmEmailSeverityLabel(31001))
	require.Equal(t, "警告 / Warning", alarmEmailSeverityLabel(31004))
}

func TestRenderAlarmEmailEscapesBusinessFields(t *testing.T) {
	window := AlarmEmailWindow{Start: time.Unix(0, 0), End: time.Unix(60, 0)}
	_, body, err := RenderAlarmEmail("<OMC>", time.UTC, window, []AlarmEmailItem{{
		AlarmIdentifier: "<script>alert(1)</script>",
		RaisedAt:        time.Unix(1, 0),
	}})
	require.NoError(t, err)
	require.NotContains(t, body, "<script>")
	require.True(t, strings.Contains(body, "&lt;script&gt;") || strings.Contains(body, "&lt;script&gt;alert"))
}
