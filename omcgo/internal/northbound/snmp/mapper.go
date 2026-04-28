package snmp

import (
	"errors"
	"strings"
)

// defaultMapper is the package-internal AlarmMapper used by Engine until
// T-0017 wires real per-carrier mappers via internal/carrier.
//
// It emulates the CMCC field ordering described in
// docs/project/prd/F08-oss-protocol.md §4:
//
//	device_serial → alarm_identifier → severity → occur_time → alarm_type → carrier
//
// Severity is encoded as integer (CMCC convention):
//
//	1=critical, 2=major, 3=minor, 4=warning, 5=cleared/indeterminate
//
// IMPORTANT: this mapper is a placeholder. T-0017 will replace it with three
// adapter implementations (cmccMapper / ctccMapper / cuccMapper) wired
// through Carrier.MapAlarmToTrapPDU. Do not extend this mapper — extend the
// Carrier adapters instead.
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

	occurUnix := alarm.OccurTime.Unix()
	if occurUnix < 0 {
		// guard against zero-value time being interpreted as 1970 by OSS
		occurUnix = 0
	}

	vars := []Variable{
		{OID: OIDDeviceSerial, Type: VarTypeOctetString, Value: alarm.DeviceSerial},
		{OID: OIDAlarmIdentifier, Type: VarTypeOctetString, Value: alarm.AlarmID},
		{OID: OIDAlarmSeverity, Type: VarTypeInteger, Value: severityToInt(alarm.Severity)},
		{OID: OIDOccurTime, Type: VarTypeCounter64, Value: uint64(occurUnix)},
	}

	if alarm.AlarmType != "" {
		vars = append(vars, Variable{
			OID:   OIDAlarmType,
			Type:  VarTypeOctetString,
			Value: alarm.AlarmType,
		})
	}
	if alarm.Carrier != "" {
		vars = append(vars, Variable{
			OID:   OIDCarrierTag,
			Type:  VarTypeOctetString,
			Value: alarm.Carrier,
		})
	}

	return vars, nil
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
