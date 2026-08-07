package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

type StatusSummaryCounts struct {
	Total     int64
	Online    int64
	Activated int64
}

type ZedStatusSummarySnapshot struct {
	GeneratedAt  time.Time
	TimeZone     string
	ByTechnology map[model.Technology]StatusSummaryCounts
	ExcludedCPE  int64
}

func RenderZedStatusSummaryEmail(snapshot ZedStatusSummarySnapshot, location *time.Location) (string, string, error) {
	if location == nil {
		location = time.UTC
	}
	if snapshot.GeneratedAt.IsZero() {
		return "", "", fmt.Errorf("status summary generated time is required")
	}
	if snapshot.TimeZone == "" {
		snapshot.TimeZone = location.String()
	}
	localGeneratedAt := snapshot.GeneratedAt.In(location).Format("2006-01-02 15:04:05 MST")
	lines := []string{
		"Zed Mobile 设备状态汇总",
		"统计时间: " + localGeneratedAt,
		"时区: " + snapshot.TimeZone,
		"",
		statusSummaryLine("2G", snapshot.ByTechnology[model.TechGSM]),
		statusSummaryLine("4G", snapshot.ByTechnology[model.TechLTE]),
		statusSummaryLine("5G", snapshot.ByTechnology[model.TechNR]),
		"",
		fmt.Sprintf("排除 CPE: %d", snapshot.ExcludedCPE),
	}
	return "Zed Mobile 设备状态汇总", strings.Join(lines, "\n"), nil
}

func statusSummaryLine(label string, counts StatusSummaryCounts) string {
	return fmt.Sprintf("%s: 总数 %d，在线 %d，激活 %d", label, counts.Total, counts.Online, counts.Activated)
}
