package alarm

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

type AlarmEmailItem struct {
	AlarmIdentifier string
	Severity        int16
	ProbableCause   string
	DeviceLabel     string
	RaisedAt        time.Time
	ClearedAt       *time.Time
	Description     string
	Advice          string
}

type alarmEmailTemplateData struct {
	OMCName      string
	Timezone     string
	WindowStart  string
	WindowEnd    string
	ActiveCount  int
	ClearedCount int
	Severity     []alarmEmailSeverityCount
	Items        []alarmEmailTemplateItem
}

type alarmEmailSeverityCount struct {
	Severity string
	Count    int
}

type alarmEmailTemplateItem struct {
	AlarmIdentifier string
	Severity        string
	ProbableCause   string
	DeviceLabel     string
	RaisedAt        string
	ClearedAt       string
	Description     string
	Advice          string
}

var alarmEmailHTMLTemplate = template.Must(template.New("alarm-email").Parse(`<!doctype html>
<html><body>
<h2>{{.OMCName}} - 告警通知 / Alarm Notification</h2>
<p>统计时间 / Window: {{.WindowStart}} — {{.WindowEnd}} ({{.Timezone}})</p>
<p>活动告警 / Active: {{.ActiveCount}}; 清除告警 / Cleared: {{.ClearedCount}}</p>
<p>级别汇总 / Severity:{{range .Severity}} {{.Severity}}={{.Count}}{{end}}</p>
<table border="1" cellspacing="0" cellpadding="6">
<thead><tr><th>Alarm Identifier</th><th>Severity</th><th>Probable Cause</th><th>Device</th><th>Raised At</th><th>Cleared At</th><th>Description</th><th>Advice</th></tr></thead>
<tbody>{{range .Items}}<tr><td>{{.AlarmIdentifier}}</td><td>{{.Severity}}</td><td>{{.ProbableCause}}</td><td>{{.DeviceLabel}}</td><td>{{.RaisedAt}}</td><td>{{.ClearedAt}}</td><td>{{.Description}}</td><td>{{.Advice}}</td></tr>{{end}}</tbody>
</table>
</body></html>`))

func RenderAlarmEmail(omcName string, location *time.Location, window AlarmEmailWindow, items []AlarmEmailItem) (subject, body string, err error) {
	if !window.Start.Before(window.End) {
		return "", "", ErrAlarmEmailInvalidWindow
	}
	if location == nil {
		location = time.UTC
	}
	omcName = strings.TrimSpace(omcName)
	if omcName == "" {
		omcName = "OMC"
	}

	data := alarmEmailTemplateData{
		OMCName:     omcName,
		Timezone:    location.String(),
		WindowStart: formatAlarmEmailTime(window.Start, location),
		WindowEnd:   formatAlarmEmailTime(window.End, location),
		Items:       make([]alarmEmailTemplateItem, 0, len(items)),
	}
	severityCounts := make(map[int16]int)
	for _, item := range items {
		severityCounts[item.Severity]++
		clearedAt := "—"
		if item.ClearedAt != nil {
			data.ClearedCount++
			clearedAt = formatAlarmEmailTime(*item.ClearedAt, location)
		} else {
			data.ActiveCount++
		}
		data.Items = append(data.Items, alarmEmailTemplateItem{
			AlarmIdentifier: item.AlarmIdentifier,
			Severity:        alarmEmailSeverityLabel(item.Severity),
			ProbableCause:   item.ProbableCause,
			DeviceLabel:     item.DeviceLabel,
			RaisedAt:        formatAlarmEmailTime(item.RaisedAt, location),
			ClearedAt:       clearedAt,
			Description:     item.Description,
			Advice:          item.Advice,
		})
	}
	severityKeys := make([]int, 0, len(severityCounts))
	for severity := range severityCounts {
		severityKeys = append(severityKeys, int(severity))
	}
	sort.Ints(severityKeys)
	for _, severity := range severityKeys {
		data.Severity = append(data.Severity, alarmEmailSeverityCount{
			Severity: alarmEmailSeverityLabel(int16(severity)),
			Count:    severityCounts[int16(severity)],
		})
	}

	var rendered bytes.Buffer
	if err := alarmEmailHTMLTemplate.Execute(&rendered, data); err != nil {
		return "", "", fmt.Errorf("render alarm email: %w", err)
	}
	return fmt.Sprintf("[%s] 告警通知 / Alarm Notification", omcName), rendered.String(), nil
}

func alarmEmailSeverityLabel(severity int16) string {
	switch canonicalAlarmSeverity(model.AlarmSeverity(severity)) {
	case model.AlarmCritical:
		return "紧急 / Critical"
	case model.AlarmMajor:
		return "重要 / Major"
	case model.AlarmMinor:
		return "次要 / Minor"
	case model.AlarmWarning:
		return "警告 / Warning"
	default:
		return fmt.Sprintf("%d", severity)
	}
}

func formatAlarmEmailTime(value time.Time, location *time.Location) string {
	return value.In(location).Format("2006-01-02 15:04:05")
}
