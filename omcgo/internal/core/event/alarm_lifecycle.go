package event

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	// AlarmLifecycleSchemaVersion is incremented only for incompatible payload changes.
	AlarmLifecycleSchemaVersion          = 1
	alarmLifecycleExtensionValueMaxBytes = 4 << 10
	alarmLifecycleExtensionsMaxBytes     = 16 << 10
)

var alarmLifecycleExtensionKeys = [...]string{
	"managed_object_instance",
	"additional_information",
	"additional_text",
	"notification_type",
}

// AlarmLifecycleType identifies a persisted managed-element alarm transition.
// It is deliberately separate from the device-reported alarm event type.
type AlarmLifecycleType string

const (
	AlarmLifecycleRaised         AlarmLifecycleType = "raised"
	AlarmLifecycleUpdated        AlarmLifecycleType = "updated"
	AlarmLifecycleAcknowledged   AlarmLifecycleType = "acknowledged"
	AlarmLifecycleUnacknowledged AlarmLifecycleType = "unacknowledged"
	AlarmLifecycleCleared        AlarmLifecycleType = "cleared"
)

// AlarmChangeField is the stable vocabulary for fields changed by an updated
// lifecycle event. Consumers must ignore unknown values for forward compatibility.
type AlarmChangeField string

const (
	AlarmChangeSeverity    AlarmChangeField = "severity"
	AlarmChangeStatus      AlarmChangeField = "status"
	AlarmChangeDescription AlarmChangeField = "description"
	AlarmChangeCount       AlarmChangeField = "alarm_count"
	AlarmChangeExtensions  AlarmChangeField = "extensions"
)

// AlarmLifecycleSnapshot is an explicit external contract. Do not replace it
// with model.Alarm: persistence-model changes must not silently alter events.
type AlarmLifecycleSnapshot struct {
	AlarmID             uuid.UUID           `json:"alarm_id"`
	DeviceID            uuid.UUID           `json:"device_id"`
	DeviceSN            string              `json:"device_sn"`
	Carrier             model.CarrierCode   `json:"carrier"`
	Technology          *string             `json:"technology,omitempty"`
	NEType              *string             `json:"ne_type,omitempty"`
	Severity            model.AlarmSeverity `json:"severity"`
	AlarmType           string              `json:"alarm_type"`
	AlarmIdentifier     string              `json:"alarm_identifier"`
	AlarmName           *string             `json:"alarm_name,omitempty"`
	Description         string              `json:"description"`
	SpecificProblem     *string             `json:"specific_problem,omitempty"`
	HandlingSuggestion  *string             `json:"handling_suggestion,omitempty"`
	AlarmSource         *string             `json:"alarm_source,omitempty"`
	AlarmEventType      *string             `json:"alarm_event_type,omitempty"`
	ProbableCause       *string             `json:"probable_cause,omitempty"`
	Status              model.AlarmStatus   `json:"status"`
	RaisedAt            time.Time           `json:"raised_at"`
	AcknowledgedAt      *time.Time          `json:"acknowledged_at,omitempty"`
	ClearedAt           *time.Time          `json:"cleared_at,omitempty"`
	FirstRaisedAt       time.Time           `json:"first_raised_at,omitempty"`
	LastUpdatedAt       time.Time           `json:"last_updated_at,omitempty"`
	AlarmCount          int                 `json:"alarm_count"`
	DeviceName          *string             `json:"device_name,omitempty"`
	NetworkLocation     *string             `json:"network_location,omitempty"`
	ExplicitCause       *string             `json:"explicit_cause,omitempty"`
	AcknowledgedBy      *string             `json:"acknowledged_by,omitempty"`
	AcknowledgementNote *string             `json:"acknowledgement_note,omitempty"`
	ClearedBy           *string             `json:"cleared_by,omitempty"`
	ClearNote           *string             `json:"clear_note,omitempty"`
	IsUnknown           bool                `json:"is_unknown"`
	Extensions          map[string]string   `json:"extensions,omitempty"`
}

// AlarmLifecyclePayload is the versioned canonical managed-element alarm event.
type AlarmLifecyclePayload struct {
	SchemaVersion    int                    `json:"schema_version"`
	EventID          uuid.UUID              `json:"event_id"`
	LifecycleType    AlarmLifecycleType     `json:"lifecycle_type"`
	OccurredAt       time.Time              `json:"occurred_at"`
	OccurrenceID     uuid.UUID              `json:"occurrence_id"`
	AlarmVersion     int64                  `json:"alarm_version"`
	ChangeMask       []AlarmChangeField     `json:"change_mask,omitempty"`
	PreviousSeverity *model.AlarmSeverity   `json:"previous_severity,omitempty"`
	Snapshot         AlarmLifecycleSnapshot `json:"snapshot"`
}

// NewAlarmLifecyclePayload builds an immutable snapshot. occurredAt and version
// are supplied by the persistence transaction; this package does not infer them.
func NewAlarmLifecyclePayload(
	lifecycleType AlarmLifecycleType,
	alarm model.Alarm,
	previous *model.Alarm,
	version int64,
	occurredAt time.Time,
	changeMask []AlarmChangeField,
) (AlarmLifecyclePayload, error) {
	if _, err := lifecycleType.Subject(); err != nil {
		return AlarmLifecyclePayload{}, err
	}
	if alarm.ID == uuid.Nil {
		return AlarmLifecyclePayload{}, fmt.Errorf("alarm lifecycle occurrence ID is required")
	}
	if alarm.DeviceID == uuid.Nil {
		return AlarmLifecyclePayload{}, fmt.Errorf("managed-element alarm device ID is required")
	}
	if version < 1 {
		return AlarmLifecyclePayload{}, fmt.Errorf("alarm lifecycle version must be positive: %d", version)
	}
	if occurredAt.IsZero() {
		return AlarmLifecyclePayload{}, fmt.Errorf("alarm lifecycle occurred time is required")
	}
	if containsAlarmChange(changeMask, AlarmChangeSeverity) && previous == nil {
		return AlarmLifecyclePayload{}, fmt.Errorf("previous alarm snapshot is required for severity change")
	}

	payload := AlarmLifecyclePayload{
		SchemaVersion: AlarmLifecycleSchemaVersion,
		EventID:       uuid.New(),
		LifecycleType: lifecycleType,
		OccurredAt:    occurredAt,
		OccurrenceID:  alarm.ID,
		AlarmVersion:  version,
		ChangeMask:    append([]AlarmChangeField(nil), changeMask...),
		Snapshot: AlarmLifecycleSnapshot{
			AlarmID:            alarm.ID,
			DeviceID:           alarm.DeviceID,
			DeviceSN:           alarm.DeviceSN,
			Carrier:            alarm.Carrier,
			Technology:         copyString(alarm.Technology),
			NEType:             additionalString(alarm.AdditionalInfo, "ne_type"),
			Severity:           alarm.Severity,
			AlarmType:          alarm.AlarmType,
			AlarmIdentifier:    alarm.AlarmIdentifier,
			AlarmName:          optionalString(alarm.Description),
			Description:        alarm.Description,
			SpecificProblem:    additionalString(alarm.AdditionalInfo, "specific_problem"),
			HandlingSuggestion: additionalString(alarm.AdditionalInfo, "handling_suggestion"),
			AlarmSource:        copyString(alarm.AlarmSource),
			AlarmEventType:     copyString(alarm.EventType),
			ProbableCause:      copyString(alarm.ProbableCause),
			Status:             alarm.Status,
			RaisedAt:           alarm.RaisedAt,
			AcknowledgedAt:     copyTime(alarm.AcknowledgedAt),
			ClearedAt:          copyTime(alarm.ClearedAt),
			FirstRaisedAt:      alarm.FirstRaisedAt,
			LastUpdatedAt:      alarm.LastUpdatedAt,
			// AckCount is the current persistence model's legacy name for the
			// number of reports merged into this occurrence.
			AlarmCount:          alarm.AckCount,
			DeviceName:          copyString(alarm.DeviceName),
			NetworkLocation:     copyString(alarm.NetworkLocation),
			ExplicitCause:       copyString(alarm.ExplicitCause),
			AcknowledgedBy:      copyString(alarm.AcknowledgedBy),
			AcknowledgementNote: copyString(alarm.AckNote),
			ClearedBy:           copyString(alarm.ClearedBy),
			ClearNote:           copyString(alarm.ClearNote),
			IsUnknown:           alarm.IsUnknown,
			Extensions:          copyAlarmLifecycleExtensions(alarm.AdditionalInfo),
		},
	}
	if previous != nil && containsAlarmChange(changeMask, AlarmChangeSeverity) {
		severity := previous.Severity
		payload.PreviousSeverity = &severity
	}
	return payload, nil
}

// Subject returns the canonical JetStream subject for the lifecycle transition.
func (t AlarmLifecycleType) Subject() (string, error) {
	switch t {
	case AlarmLifecycleRaised:
		return SubjectDomainAlarmLifecycleRaised, nil
	case AlarmLifecycleUpdated:
		return SubjectDomainAlarmLifecycleUpdated, nil
	case AlarmLifecycleAcknowledged:
		return SubjectDomainAlarmLifecycleAcknowledged, nil
	case AlarmLifecycleUnacknowledged:
		return SubjectDomainAlarmLifecycleUnacknowledged, nil
	case AlarmLifecycleCleared:
		return SubjectDomainAlarmLifecycleCleared, nil
	default:
		return "", fmt.Errorf("unsupported alarm lifecycle type %q", t)
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}

func additionalString(values map[string]string, key string) *string {
	if len(values) == 0 {
		return nil
	}
	return optionalString(values[key])
}

func containsAlarmChange(fields []AlarmChangeField, want AlarmChangeField) bool {
	for _, field := range fields {
		if field == want {
			return true
		}
	}
	return false
}

func copyAlarmLifecycleExtensions(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(alarmLifecycleExtensionKeys))
	remaining := alarmLifecycleExtensionsMaxBytes
	for _, key := range alarmLifecycleExtensionKeys {
		value, ok := source[key]
		if !ok || remaining <= 0 {
			continue
		}
		limit := min(alarmLifecycleExtensionValueMaxBytes, remaining)
		value = truncateUTF8(value, limit)
		result[key] = value
		remaining -= len(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
