package notification

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestRenderZedStatusSummaryEmail_UsesLocalTimeAndAllTechnologyCounts(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	subject, body, err := RenderZedStatusSummaryEmail(ZedStatusSummarySnapshot{
		GeneratedAt: time.Date(2026, 8, 10, 3, 0, 0, 0, time.UTC),
		TimeZone:    "Asia/Shanghai",
		ByTechnology: map[model.Technology]StatusSummaryCounts{
			model.TechGSM: {Total: 12},
			model.TechLTE: {Total: 81, Online: 14, Activated: 10},
			model.TechNR:  {Total: 7, Online: 5, Activated: 4},
		},
		ExcludedCPE: 9,
	}, location)

	require.NoError(t, err)
	require.Equal(t, "Zed Mobile 设备状态汇总", subject)
	require.Contains(t, body, "统计时间: 2026-08-10 11:00:00 CST")
	require.Contains(t, body, "2G: 总数 12，在线 0，激活 0")
	require.Contains(t, body, "4G: 总数 81，在线 14，激活 10")
	require.Contains(t, body, "5G: 总数 7，在线 5，激活 4")
	require.Contains(t, body, "排除 CPE: 9")
}

func TestRenderZedStatusSummaryEmail_RejectsMissingGeneratedTime(t *testing.T) {
	_, _, err := RenderZedStatusSummaryEmail(ZedStatusSummarySnapshot{}, time.UTC)
	require.ErrorContains(t, err, "generated time is required")
}
