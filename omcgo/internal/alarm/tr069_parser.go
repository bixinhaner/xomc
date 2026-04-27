package alarm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// TR069Alarm represents a single alarm parsed from Device.FaultMgmt.CurrentAlarm.{i}.* parameters.
type TR069Alarm struct {
	Index                 int
	AlarmIdentifier       string
	PerceivedSeverity     string
	AlarmRaisedTime       time.Time
	AlarmChangedTime      time.Time
	EventType             string
	ProbableCause         string
	SpecificProblem       string
	AdditionalInformation string
	AdditionalText        string
	ManagedObjectInstance string
}

// currentAlarmFieldRE extracts the index and field name from a parameter path
// like "Device.FaultMgmt.CurrentAlarm.3.PerceivedSeverity".
var currentAlarmFieldRE = regexp.MustCompile(`^Device\.FaultMgmt\.CurrentAlarm\.(\d+)\.(.+)$`)

// ParseCurrentAlarmParams parses a GPV response parameter list into structured alarms.
// Parameters are grouped by their index ({i} in the CurrentAlarm path.
func ParseCurrentAlarmParams(params []tr069.ParameterValueStruct) ([]TR069Alarm, error) {
	indexMap := make(map[int]*TR069Alarm)

	for _, p := range params {
		m := currentAlarmFieldRE.FindStringSubmatch(p.Name)
		if m == nil {
			continue // Not a CurrentAlarm parameter
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		field := m[2]

		alarm, ok := indexMap[idx]
		if !ok {
			alarm = &TR069Alarm{Index: idx}
			indexMap[idx] = alarm
		}

		switch field {
		case "AlarmIdentifier":
			alarm.AlarmIdentifier = p.Value
		case "PerceivedSeverity":
			alarm.PerceivedSeverity = p.Value
		case "AlarmRaisedTime":
			alarm.AlarmRaisedTime = parseTR069Time(p.Value)
		case "AlarmChangedTime":
			alarm.AlarmChangedTime = parseTR069Time(p.Value)
		case "EventType":
			alarm.EventType = p.Value
		case "ProbableCause":
			alarm.ProbableCause = p.Value
		case "SpecificProblem":
		alarm.SpecificProblem = p.Value
		case "AdditionalInformation":
			alarm.AdditionalInformation = p.Value
		case "AdditionalText":
			alarm.AdditionalText = p.Value
		case "ManagedObjectInstance":
		alarm.ManagedObjectInstance = p.Value
		}
	}

	// Convert map to sorted slice
	result := make([]TR069Alarm, 0, len(indexMap))
	for _, a := range indexMap {
		if a.AlarmIdentifier == "" {
			continue // Skip entries without identifier
		}
		result = append(result, *a)
	}

	// Sort by index (bubble sort)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Index < result[i].Index {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// tr069TimeLayouts lists formats accepted by parseTR069Time, in priority order.
var tr069TimeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05.000-07:00",
}

// parseTR069Time parses a TR-069 dateTime value.
// Falls back to zero time for empty or epoch values.
// Supports formats with and without timezone suffix.
func parseTR069Time(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	var t time.Time
	var err error
	for _, layout := range tr069TimeLayouts {
		t, err = time.Parse(layout, s)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}
	}
	// TR-069 devices sometimes send epoch time (1970-01-01T00:00:00) which is invalid.
	if t.Year() < 2000 {
		return time.Time{}
	}
	return t
}

// ToModel converts a TR069Alarm to a model.Alarm for persistence.
func (t *TR069Alarm) ToModel(deviceID uuid.UUID, deviceSN string, carrier model.CarrierCode) *model.Alarm {
	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        deviceSN,
		Carrier:         carrier,
		AlarmIdentifier: t.AlarmIdentifier,
		Description:     t.SpecificProblem,
		EventType:       strPtr(t.EventType),
		AlarmSource:     strPtr("TR069"),
		Severity:        mapSeverity(t.PerceivedSeverity),
		RaisedAt:        t.AlarmRaisedTime,
		Status:          model.AlarmActive,
		AdditionalInfo:   make(map[string]string),
	}

	// Use AlarmRaisedTime as LastUpdatedAt; fall back to AlarmChangedTime.
	if !t.AlarmRaisedTime.IsZero() {
		alarm.LastUpdatedAt = t.AlarmRaisedTime
	} else if !t.AlarmChangedTime.IsZero() {
		alarm.LastUpdatedAt = t.AlarmChangedTime
	}

	// Store additional info
	if t.AdditionalInformation != "" {
		alarm.AdditionalInfo["additional_information"] = t.AdditionalInformation
	}
	if t.AdditionalText != "" {
		alarm.AdditionalInfo["additional_text"] = t.AdditionalText
	}
	// ProbableCause is NOT NULL in DB; fall back to SpecificProblem then AlarmIdentifier.
	pc := t.ProbableCause
	if pc == "" {
		pc = t.SpecificProblem
	}
	if pc == "" {
		pc = t.AlarmIdentifier
	}
	alarm.ProbableCause = strPtr(pc)
	if t.ManagedObjectInstance != "" {
		alarm.AdditionalInfo["managed_object_instance"] = t.ManagedObjectInstance
	}

	return alarm
}

// mapSeverity maps TR-069 severity strings to model.AlarmSeverity.
func mapSeverity(s string) model.AlarmSeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return global.AlarmCritical
	case "MAJOR":
		return global.AlarmMajor
	case "MINOR":
		return global.AlarmMinor
	case "WARNING":
		return global.AlarmWarning
	default:
		return global.AlarmMajor // Default to Major for unknown
	}
}

// FormatAlarmParamsForGPV generates the parameter names for a GPV query.
func FormatAlarmParamsForGPV() json.RawMessage {
	return json.RawMessage(`{"names":["Device.FaultMgmt.CurrentAlarm."]}`)
}

// Validate ensures the parsed alarm has required fields.
func (t *TR069Alarm) Validate() error {
	if t.AlarmIdentifier == "" {
		return fmt.Errorf("alarm at index %d has no AlarmIdentifier", t.Index)
	}
	return nil
}

// --- ExpeditedEvent parsing ---

// NotificationType constants for Device.FaultMgmt.ExpeditedEvent.{i}.NotificationType.
const (
	NotificationNewAlarm      = "NewAlarm"
	NotificationChangedAlarm  = "ChangedAlarm"
	NotificationClearedAlarm  = "ClearedAlarm"
)

// ExpeditedEvent represents a single real-time alarm notification parsed from
// Device.FaultMgmt.ExpeditedEvent.{i}.* parameters carried in VALUE CHANGE Informs.
type ExpeditedEvent struct {
	Index                 int
	NotificationType      string // NewAlarm, ChangedAlarm, ClearedAlarm
	AlarmIdentifier       string
	PerceivedSeverity     string
	EventType             string
	ProbableCause         string
	SpecificProblem       string
	AdditionalInformation string
	AdditionalText        string
	EventTime             time.Time
	ManagedObjectInstance string
}

// expeditedEventFieldRE extracts the index and field name from a parameter path
// like "Device.FaultMgmt.ExpeditedEvent.10.NotificationType".
var expeditedEventFieldRE = regexp.MustCompile(`^Device\.FaultMgmt\.ExpeditedEvent\.(\d+)\.(.+)$`)

// HasExpeditedEventParams returns true if any parameter in the list belongs to
// the ExpeditedEvent subtree (Device.FaultMgmt.ExpeditedEvent.*).
func HasExpeditedEventParams(params []tr069.ParameterValueStruct) bool {
	for _, p := range params {
		if strings.HasPrefix(p.Name, "Device.FaultMgmt.ExpeditedEvent.") {
			return true
		}
	}
	return false
}

// FilterExpeditedEventParams returns only the parameters that belong to the
// ExpeditedEvent subtree, filtering out unrelated VALUE CHANGE parameters.
func FilterExpeditedEventParams(params []tr069.ParameterValueStruct) []tr069.ParameterValueStruct {
	filtered := make([]tr069.ParameterValueStruct, 0, len(params))
	for _, p := range params {
		if strings.HasPrefix(p.Name, "Device.FaultMgmt.ExpeditedEvent.") {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// ParseExpeditedEventParams parses ExpeditedEvent parameters into structured events.
// Multiple events can appear in a single Inform (different {i} indices).
func ParseExpeditedEventParams(params []tr069.ParameterValueStruct) ([]ExpeditedEvent, error) {
	indexMap := make(map[int]*ExpeditedEvent)

	for _, p := range params {
		m := expeditedEventFieldRE.FindStringSubmatch(p.Name)
		if m == nil {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		field := m[2]

		ev, ok := indexMap[idx]
		if !ok {
			ev = &ExpeditedEvent{Index: idx}
			indexMap[idx] = ev
		}

		switch field {
		case "NotificationType":
			ev.NotificationType = p.Value
		case "AlarmIdentifier":
			ev.AlarmIdentifier = p.Value
		case "PerceivedSeverity":
			ev.PerceivedSeverity = p.Value
		case "EventType":
			ev.EventType = p.Value
		case "ProbableCause":
			ev.ProbableCause = p.Value
		case "SpecificProblem":
			ev.SpecificProblem = p.Value
		case "AdditionalInformation":
			ev.AdditionalInformation = p.Value
		case "AdditionalText":
			ev.AdditionalText = p.Value
		case "EventTime":
			ev.EventTime = parseTR069Time(p.Value)
		case "ManagedObjectInstance":
			ev.ManagedObjectInstance = p.Value
		}
	}

	result := make([]ExpeditedEvent, 0, len(indexMap))
	for _, ev := range indexMap {
		if ev.AlarmIdentifier == "" {
			continue
		}
		result = append(result, *ev)
	}

	// Sort by index
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Index < result[i].Index {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// ToModel converts an ExpeditedEvent to a model.Alarm for persistence.
// For NewAlarm, this produces a full alarm ready for engine.Process().
// For ChangedAlarm/ClearedAlarm, only the mutable fields are meaningful.
func (e *ExpeditedEvent) ToModel(deviceID uuid.UUID, deviceSN string, carrier model.CarrierCode) *model.Alarm {
	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        deviceSN,
		Carrier:         carrier,
		AlarmIdentifier: e.AlarmIdentifier,
		Description:     e.SpecificProblem,
		EventType:       strPtr(e.EventType),
		AlarmSource:     strPtr("TR069"),
		Severity:        mapSeverity(e.PerceivedSeverity),
		Status:          model.AlarmActive,
		AdditionalInfo:  make(map[string]string),
	}

	if !e.EventTime.IsZero() {
		alarm.RaisedAt = e.EventTime
		alarm.LastUpdatedAt = e.EventTime
	}

	pc := e.ProbableCause
	if pc == "" {
		pc = e.SpecificProblem
	}
	if pc == "" {
		pc = e.AlarmIdentifier
	}
	alarm.ProbableCause = strPtr(pc)

	if e.AdditionalInformation != "" {
		alarm.AdditionalInfo["additional_information"] = e.AdditionalInformation
	}
	if e.AdditionalText != "" {
		alarm.AdditionalInfo["additional_text"] = e.AdditionalText
	}
	if e.ManagedObjectInstance != "" {
		alarm.AdditionalInfo["managed_object_instance"] = e.ManagedObjectInstance
	}
	alarm.AdditionalInfo["notification_type"] = e.NotificationType

	return alarm
}

// Validate ensures the ExpeditedEvent has required fields.
func (e *ExpeditedEvent) Validate() error {
	if e.AlarmIdentifier == "" {
		return fmt.Errorf("expedited event at index %d has no AlarmIdentifier", e.Index)
	}
	switch e.NotificationType {
	case NotificationNewAlarm, NotificationChangedAlarm, NotificationClearedAlarm:
		// valid
	default:
		return fmt.Errorf("expedited event at index %d has unknown NotificationType: %q", e.Index, e.NotificationType)
	}
	return nil
}
