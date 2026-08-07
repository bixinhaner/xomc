package pageconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
)

func (s *Service) sendSNMPAlarmToTarget(ctx context.Context, target SNMPAlarmTarget, alarm *nbsnmp.AlarmEvent, eventType string) PageConfigEvent {
	if alarm == nil {
		alarm = sampleSNMPAlarm()
	}
	effectiveAlarm := snmpAlarmForTarget(target, alarm)
	vars, err := nbsnmp.DefaultAlarmMapper().MapAlarmToTrapPDU(effectiveAlarm)
	if err != nil {
		result := SNMPSendResult{
			Success:          false,
			AlarmID:          effectiveAlarm.AlarmID,
			TargetKey:        target.Key,
			Host:             target.TargetHost,
			Port:             target.TargetPort,
			Version:          target.Version,
			NotificationType: target.NotificationType,
			Error:            err.Error(),
			Message:          err.Error(),
		}
		return eventFromSNMPSend(target, result, "")
	}

	if s.snmpSender == nil {
		result := SNMPSendResult{
			Success:          false,
			AlarmID:          effectiveAlarm.AlarmID,
			TargetKey:        target.Key,
			Host:             target.TargetHost,
			Port:             target.TargetPort,
			Version:          target.Version,
			NotificationType: target.NotificationType,
			VarbindCount:     len(vars),
			Error:            "SNMP sender is not configured",
			Message:          "SNMP sender is not configured",
		}
		event := eventFromSNMPSend(target, result, snmpPayloadFromVars(vars))
		event.EventType = eventType
		return event
	}

	start := time.Now()
	runtimeTarget := snmpRuntimeTarget(target)
	err = s.snmpSender.Send(ctx, runtimeTarget, vars)
	latency := time.Since(start)
	result := SNMPSendResult{
		Success:          err == nil,
		AlarmID:          effectiveAlarm.AlarmID,
		TargetKey:        target.Key,
		Host:             target.TargetHost,
		Port:             target.TargetPort,
		Version:          target.Version,
		NotificationType: target.NotificationType,
		VarbindCount:     len(vars),
		LatencyMs:        latency.Milliseconds(),
	}
	if err != nil {
		result.Error = scrubSecret(err.Error(), target.Community, target.AuthCredential, target.PrivCredential)
		result.Message = result.Error
	} else {
		result.Message = fmt.Sprintf("SNMP %s sent to %s:%d", target.NotificationType, target.TargetHost, target.TargetPort)
	}
	event := eventFromSNMPSend(target, result, snmpPayloadFromVars(vars))
	event.EventType = eventType
	return event
}

func snmpRuntimeTarget(target SNMPAlarmTarget) *nbsnmp.TrapTarget {
	return &nbsnmp.TrapTarget{
		ID:           target.Key,
		OSSName:      firstNonEmpty(target.Name, target.Key),
		Host:         strings.TrimSpace(target.TargetHost),
		Port:         uint16(target.TargetPort),
		Version:      runtimeSNMPVersion(target.Version),
		Community:    target.Community,
		Username:     target.SecurityName,
		AuthProtocol: runtimeSNMPAuthProtocol(target.AuthProtocol),
		AuthPassword: target.AuthCredential,
		PrivProtocol: runtimeSNMPPrivProtocol(target.PrivProtocol),
		PrivPassword: target.PrivCredential,
		Timeout:      time.Duration(target.TimeoutSeconds) * time.Second,
		Retries:      target.Retries,
		Inform:       strings.EqualFold(target.NotificationType, "Inform"),
		Enabled:      target.Enabled,
	}
}

func runtimeSNMPVersion(version string) nbsnmp.SNMPVersion {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "v3":
		return nbsnmp.VersionV3
	default:
		return nbsnmp.VersionV2c
	}
}

func runtimeSNMPAuthProtocol(protocol string) nbsnmp.AuthProtocol {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "MD5":
		return nbsnmp.AuthMD5
	case "SHA", "SHA1":
		return nbsnmp.AuthSHA
	case "SHA224":
		return nbsnmp.AuthSHA224
	case "SHA256":
		return nbsnmp.AuthSHA256
	case "SHA384":
		return nbsnmp.AuthSHA384
	case "SHA512":
		return nbsnmp.AuthSHA512
	default:
		return nbsnmp.AuthNoAuth
	}
}

func runtimeSNMPPrivProtocol(protocol string) nbsnmp.PrivProtocol {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "DES":
		return nbsnmp.PrivDES
	case "AES", "AES128":
		return nbsnmp.PrivAES
	case "AES192":
		return nbsnmp.PrivAES192
	case "AES256":
		return nbsnmp.PrivAES256
	default:
		return nbsnmp.PrivNoPriv
	}
}

func sampleSNMPAlarm() *nbsnmp.AlarmEvent {
	return &nbsnmp.AlarmEvent{
		AlarmID:      "920188",
		DeviceSerial: "867294050000001",
		Severity:     "major",
		AlarmType:    "CELL_UNAVAILABLE",
		Carrier:      "cmcc",
		OccurTime:    time.Date(2026, 8, 4, 17, 0, 0, 0, time.Local),
		Extra: map[string]string{
			"notificationID":        "1",
			"alarmUniqueId":         "40123",
			"equipmentName":         "867294050000001",
			"equipmentClass":        "LTE",
			"objectSDN":             "867294050000001",
			"objectInstanceName":    "Cell-1",
			"objectClass":           "Cell",
			"additionalText":        "Northbound SNMP test alarm",
			"deviceVendorOUI":       "Baicells",
			"specificProblemID":     "CELL1",
			"specificProblem":       "CELL_UNAVAILABLE",
			"probableCause":         "Cell unavailable",
			"additionalInformation": "page-config-test",
		},
	}
}

func snmpAlarmForTarget(target SNMPAlarmTarget, alarm *nbsnmp.AlarmEvent) *nbsnmp.AlarmEvent {
	if alarm == nil {
		return sampleSNMPAlarm()
	}
	effective := *alarm
	if alarm.Extra != nil {
		effective.Extra = make(map[string]string, len(alarm.Extra)+1)
		for key, value := range alarm.Extra {
			effective.Extra[key] = value
		}
	}
	if effective.Extra == nil {
		effective.Extra = map[string]string{}
	}
	if effective.Extra["notificationType"] == "0" && strings.EqualFold(target.ClearSeverityPolicy, "清除置 0") {
		effective.Extra["perceivedSeverity"] = "0"
	}
	return &effective
}

func snmpPayloadFromVars(vars []nbsnmp.Variable) string {
	if len(vars) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range vars {
		b.WriteString(item.OID)
		b.WriteString(" = ")
		b.WriteString(fmt.Sprint(item.Value))
		b.WriteString("\n")
	}
	return b.String()
}
