package regularreport

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const JobType = "pm_kpi_regular_report"

type Period string

const (
	Period15Min  Period = "15min"
	PeriodHourly Period = "hourly"
	PeriodDaily  Period = "daily"
)

type Config struct {
	Enabled      bool     `json:"enabled"`
	SendTime     string   `json:"send_time"`
	Periods      []Period `json:"periods"`
	EmailEnabled bool     `json:"email_enabled"`
	Recipients   []string `json:"recipients"`
}

type Template struct {
	ID        uuid.UUID
	Name      string
	CreatorID uuid.UUID
	Payload   json.RawMessage
	Config    Config
}

type RunStatus string

const (
	RunPending       RunStatus = "pending"
	RunProcessing    RunStatus = "processing"
	RunSent          RunStatus = "sent"
	RunPartialFailed RunStatus = "partial_failed"
	RunFailed        RunStatus = "failed"
)

type Run struct {
	ID           uuid.UUID
	TemplateID   *uuid.UUID
	TemplateName string
	CreatorID    uuid.UUID
	Period       Period
	ScheduledAt  time.Time
	WindowStart  time.Time
	WindowEnd    time.Time
	ExportTaskID uuid.UUID
	Status       RunStatus
	Subject      string
	LastError    string
}

type Delivery struct {
	ID        uuid.UUID
	RunID     uuid.UUID
	Recipient string
	Status    string
	Attempt   int
}

type JobPayload struct {
	RunID string `json:"run_id"`
}

func ParseConfig(payload []byte) (Config, error) {
	var root struct {
		RegularReport *Config `json:"regular_report"`
	}
	if len(payload) == 0 {
		return Config{}, nil
	}
	if err := json.Unmarshal(payload, &root); err != nil {
		return Config{}, fmt.Errorf("parse regular report config: %w", err)
	}
	if root.RegularReport == nil {
		return Config{}, nil
	}
	config := *root.RegularReport
	// 关闭配置必须始终可保存，便于管理员先停用再修正旧的无效地址。
	// worker 只扫描 enabled=true 的模板，因此关闭态原值不会进入发送链路。
	if !config.Enabled {
		return config, nil
	}
	recipients, err := normalizeRecipients(config.Recipients)
	if err != nil {
		return Config{}, err
	}
	config.Recipients = recipients
	config.Periods, err = normalizePeriods(config.Periods)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func ValidatePayload(payload []byte) error {
	config, err := ParseConfig(payload)
	if err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	if _, _, err := parseSendTime(config.SendTime); err != nil {
		return err
	}
	if len(config.Periods) == 0 {
		return fmt.Errorf("regular_report.periods must include at least one of 15min, hourly, daily")
	}
	if !config.EmailEnabled {
		return fmt.Errorf("regular_report.email_enabled must be true when the report is enabled")
	}
	if len(config.Recipients) == 0 {
		return fmt.Errorf("regular_report.recipients must not be empty when email is enabled")
	}
	var template struct {
		MetricPaths []string `json:"metric_paths"`
	}
	if err := json.Unmarshal(payload, &template); err != nil {
		return fmt.Errorf("parse regular report metric paths: %w", err)
	}
	hasMetric := false
	for _, metricPath := range template.MetricPaths {
		if strings.TrimSpace(metricPath) != "" {
			hasMetric = true
			break
		}
	}
	if !hasMetric {
		return fmt.Errorf("regular_report requires at least one metric_path")
	}
	return nil
}

func ScheduledAt(now time.Time, sendTime string, location *time.Location) (time.Time, error) {
	if location == nil {
		location = time.UTC
	}
	hour, minute, err := parseSendTime(sendTime)
	if err != nil {
		return time.Time{}, err
	}
	local := now.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, location), nil
}

func CompleteWindow(scheduledAt time.Time, period Period, location *time.Location) (time.Time, time.Time, error) {
	if location == nil {
		location = time.UTC
	}
	local := scheduledAt.In(location)
	switch period {
	case Period15Min:
		end := local.Truncate(15 * time.Minute)
		return end.Add(-15 * time.Minute), end, nil
	case PeriodHourly:
		end := local.Truncate(time.Hour)
		return end.Add(-time.Hour), end, nil
	case PeriodDaily:
		end := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
		return end.AddDate(0, 0, -1), end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported regular report period %q", period)
	}
}

func BuildExportParams(payload []byte, period Period, start, end time.Time) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("parse query template payload: %w", err)
	}
	params := make(map[string]any)
	for _, key := range []string{"device_sns", "metric_paths"} {
		if raw := root[key]; len(raw) > 0 {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("parse query template %s: %w", key, err)
			}
			params[key] = value
		}
	}
	if raw := root["device_type"]; len(raw) > 0 {
		var deviceType string
		if err := json.Unmarshal(raw, &deviceType); err != nil {
			return nil, fmt.Errorf("parse query template device_type: %w", err)
		}
		technologyByDeviceType := map[string]string{"ENB": "lte", "GNB": "nr", "GSM": "gsm"}
		if technology := technologyByDeviceType[strings.ToUpper(deviceType)]; technology != "" {
			params["technologies"] = []string{technology}
		}
	}
	params["granularity"] = string(period)
	params["dimension"] = "device"
	params["start_time"] = start.Format(time.RFC3339)
	params["end_time"] = end.Format(time.RFC3339)
	return json.Marshal(params)
}

func parseSendTime(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if len(value) != 5 || value[2] != ':' {
		return 0, 0, fmt.Errorf("regular_report.send_time must use HH:mm")
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, fmt.Errorf("regular_report.send_time must use HH:mm")
	}
	return parsed.Hour(), parsed.Minute(), nil
}

func normalizePeriods(periods []Period) ([]Period, error) {
	seen := make(map[Period]struct{}, len(periods))
	for _, period := range periods {
		switch period {
		case Period15Min, PeriodHourly, PeriodDaily:
			seen[period] = struct{}{}
		default:
			return nil, fmt.Errorf("regular_report.periods contains unsupported value %q", period)
		}
	}
	result := make([]Period, 0, len(seen))
	for _, period := range []Period{Period15Min, PeriodHourly, PeriodDaily} {
		if _, ok := seen[period]; ok {
			result = append(result, period)
		}
	}
	return result, nil
}

func normalizeRecipients(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := mail.ParseAddress(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("regular_report.recipients contains invalid address: %w", err)
		}
		key := strings.ToLower(parsed.Address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, parsed.Address)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result, nil
}
