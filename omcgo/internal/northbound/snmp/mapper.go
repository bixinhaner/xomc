package snmp

import (
	"errors"
	"strconv"
	"strings"
)

// defaultMapper maps the current AlarmEvent shape to omcAlarmMIB.mib. The
// VarBind order must remain identical to omcAlarmNotification OBJECTS.
type defaultMapper struct{}

// DefaultAlarmMapper returns the package-internal placeholder mapper.
// Exported only so tests (and Engine bootstrap) can construct it explicitly.
func DefaultAlarmMapper() AlarmMapper {
	return defaultMapper{}
}

func (defaultMapper) MapAlarmToTrapPDU(alarm *AlarmEvent) ([]Variable, error) {
	if alarm == nil {
		return nil, errors.New("snmp: nil alarm")
	}
	if alarm.AlarmID == "" {
		return nil, errors.New("snmp: alarm missing AlarmID")
	}
	if alarm.DeviceSerial == "" {
		return nil, errors.New("snmp: alarm missing DeviceSerial")
	}

	eventMillis := alarm.OccurTime.UnixMilli()
	if eventMillis < 0 {
		// guard against zero-value time being interpreted as 1970 by OSS
		eventMillis = 0
	}

	notificationID := 1
	if raw := alarmExtra(alarm, "notificationID", ""); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			notificationID = parsed
		}
	}

	return []Variable{
		{OID: OIDNotificationID, Type: VarTypeInteger, Value: notificationID},
		{OID: OIDAlarmUniqueID, Type: VarTypeOctetString, Value: alarmExtra(alarm, "alarmUniqueId", alarm.AlarmID)},
		{OID: OIDNotificationType, Type: VarTypeOctetString, Value: notificationType(alarm)},
		{OID: OIDEventTime, Type: VarTypeCounter64, Value: uint64(eventMillis)},
		{OID: OIDEquipmentSDN, Type: VarTypeOctetString, Value: alarmExtra(alarm, "equipmentSDN", alarm.DeviceSerial)},
		{OID: OIDEquipmentName, Type: VarTypeOctetString, Value: alarmExtra(alarm, "equipmentName", alarm.DeviceSerial)},
		{OID: OIDEquipmentClass, Type: VarTypeOctetString, Value: alarmExtra(alarm, "equipmentClass", alarm.Carrier)},
		{OID: OIDObjectSDN, Type: VarTypeOctetString, Value: alarmExtra(alarm, "objectSDN", alarm.DeviceSerial)},
		{OID: OIDObjectInstanceName, Type: VarTypeOctetString, Value: alarmExtra(alarm, "objectInstanceName", "")},
		{OID: OIDObjectClass, Type: VarTypeOctetString, Value: alarmExtra(alarm, "objectClass", "")},
		{OID: OIDAdditionalText, Type: VarTypeOctetString, Value: alarmExtra(alarm, "additionalText", "")},
		{OID: OIDDeviceVendorOUI, Type: VarTypeOctetString, Value: alarmExtra(alarm, "deviceVendorOUI", "")},
		{OID: OIDSpecificProblemID, Type: VarTypeOctetString, Value: alarmExtra(alarm, "specificProblemID", alarm.AlarmID)},
		{OID: OIDSpecificProblem, Type: VarTypeOctetString, Value: alarmExtra(alarm, "specificProblem", alarm.AlarmType)},
		{OID: OIDAlarmType, Type: VarTypeOctetString, Value: alarmExtra(alarm, "alarmType", alarm.AlarmType)},
		{OID: OIDPerceivedSeverity, Type: VarTypeOctetString, Value: alarmExtra(alarm, "perceivedSeverity", mibSeverity(alarm.Severity))},
		{OID: OIDProbableCause, Type: VarTypeOctetString, Value: alarmExtra(alarm, "probableCause", "")},
		{OID: OIDAdditionalInformation, Type: VarTypeOctetString, Value: alarmExtra(alarm, "additionalInformation", alarm.Carrier)},
	}, nil
}

// severityToInt maps a textual severity to the CMCC integer encoding.
// Unknown severities map to 5 (treated as indeterminate / cleared) rather
// than rejecting the trap — telemetry is still useful even with unknown
// severity, and OSS-side will surface anomalies.
func severityToInt(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 1
	case "major":
		return 2
	case "minor":
		return 3
	case "warning":
		return 4
	case "cleared", "indeterminate", "":
		return 5
	default:
		return 5
	}
}

func notificationType(alarm *AlarmEvent) string {
	if value := alarmExtra(alarm, "notificationType", ""); value != "" {
		return value
	}
	switch strings.ToLower(strings.TrimSpace(alarm.Severity)) {
	case "cleared", "clear":
		return "0"
	default:
		return "1"
	}
}

func mibSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "critical"
	case "major":
		return "major"
	case "minor":
		return "minor"
	case "warning":
		return "warning"
	case "cleared", "clear":
		return "cleared"
	default:
		return "unknown"
	}
}

func alarmExtra(alarm *AlarmEvent, key, fallback string) string {
	if alarm == nil || alarm.Extra == nil {
		return fallback
	}
	if value := strings.TrimSpace(alarm.Extra[key]); value != "" {
		return value
	}
	return fallback
}
