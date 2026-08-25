package attention

import (
	"context"
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
)

const alarmSourcePageSize = 1000

type AlarmReader interface {
	ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error)
}

type AlarmSource struct {
	reader   AlarmReader
	severity model.AlarmSeverity
}

func NewAlarmAbnormalitySource(reader AlarmReader, severity model.AlarmSeverity) *AlarmSource {
	return &AlarmSource{reader: reader, severity: severity}
}

func (s *AlarmSource) Name() string {
	return "active_alarm:" + severityLabel(s.severity)
}

func (s *AlarmSource) Rank() int {
	if s.severity == model.AlarmCritical {
		return 400
	}
	return 300
}

func (s *AlarmSource) Permission() Permission {
	return Permission{Resource: "/api/v1/alarms/:id", Action: "GET"}
}

func (s *AlarmSource) ListPrefix(ctx context.Context, scope Scope, limit int) (SourceResult, error) {
	return s.ListWindow(ctx, scope, 0, limit)
}

func (s *AlarmSource) ListWindow(ctx context.Context, scope Scope, offset, limit int) (SourceResult, error) {
	if s == nil || s.reader == nil {
		return SourceResult{}, fmt.Errorf("alarm attention reader is not configured")
	}
	if offset < 0 || limit < 1 {
		return SourceResult{Items: []Item{}}, nil
	}

	items := make([]Item, 0, limit)
	var total int64
	pageSize := min(alarmSourcePageSize, limit)
	page := offset/pageSize + 1
	withinPage := offset % pageSize
	firstQuery := true
	for len(items) < limit {
		filter := alarm.AlarmFilter{
			Severity:      &s.severity,
			VisibleGroups: scope.VisibleGroups,
			ListRequest:   model.ListRequest{Page: page, PageSize: pageSize},
		}
		response, err := s.reader.ListActive(ctx, filter)
		if err != nil {
			return SourceResult{}, fmt.Errorf("list %s alarms: %w", s.Name(), err)
		}
		if firstQuery {
			total = response.Total
			firstQuery = false
		}
		start := min(withinPage, len(response.Items))
		for index := start; index < len(response.Items) && len(items) < limit; index++ {
			items = append(items, s.mapItem(response.Items[index]))
		}
		if len(response.Items) < pageSize || int64(offset+len(items)) >= total {
			break
		}
		page++
		withinPage = 0
	}
	return SourceResult{Total: total, Items: items}, nil
}

func (s *AlarmSource) mapItem(alarmItem model.Alarm) Item {
	title := strings.TrimSpace(alarmItem.Description)
	if title == "" {
		title = alarmItem.AlarmIdentifier
	}
	summary := ""
	if alarmItem.ProbableCause != nil {
		summary = strings.TrimSpace(*alarmItem.ProbableCause)
	}
	if summary == "" && alarmItem.ExplicitCause != nil {
		summary = strings.TrimSpace(*alarmItem.ExplicitCause)
	}
	targetName := alarmItem.DeviceSN
	if alarmItem.DeviceName != nil && strings.TrimSpace(*alarmItem.DeviceName) != "" {
		targetName = strings.TrimSpace(*alarmItem.DeviceName)
	}
	updatedAt := alarmItem.LastUpdatedAt
	if updatedAt.IsZero() {
		updatedAt = alarmItem.UpdatedAt
	}

	item := Item{
		ID: "abnormal:alarm:" + alarmItem.ID.String(), Kind: KindActiveAlarm, Source: "alarm",
		SourceID: alarmItem.ID.String(), Title: title, Summary: summary,
		Severity: severityLabel(alarmItem.Severity), Priority: severityLabel(alarmItem.Severity),
		Target:      &Target{Type: "device", ID: alarmItem.DeviceID.String(), Name: targetName, SerialNumber: alarmItem.DeviceSN},
		DetailRoute: "/alarm/current?alarmId=" + alarmItem.ID.String(), AllowedActions: []Action{ActionViewAlarm},
		UpdatedAt: &updatedAt,
	}
	occurredAt := alarmItem.RaisedAt
	item.OccurredAt = &occurredAt
	return item
}

func severityLabel(severity model.AlarmSeverity) string {
	switch severity {
	case model.AlarmCritical, model.AlarmSeverity(31001):
		return "critical"
	case model.AlarmMajor, model.AlarmSeverity(31002):
		return "major"
	case model.AlarmMinor, model.AlarmSeverity(31003):
		return "minor"
	case model.AlarmWarning, model.AlarmSeverity(31004):
		return "warning"
	default:
		return "unknown"
	}
}
