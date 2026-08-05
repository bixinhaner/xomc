package tr069

import "strings"

// TR069 Inform event codes per CWMP specification.
const (
	EventBootstrap                  = "0 BOOTSTRAP"
	EventBoot                       = "1 BOOT"
	EventPeriodic                   = "2 PERIODIC"
	EventScheduled                  = "3 SCHEDULED"
	EventValueChange                = "4 VALUE CHANGE"
	EventKicked                     = "5 KICKED"
	EventConnectionRequest          = "6 CONNECTION REQUEST"
	EventTransferComplete           = "7 TRANSFER COMPLETE"
	EventDiagnosticsComplete        = "8 DIAGNOSTICS COMPLETE"
	EventRequestDownload            = "9 REQUEST DOWNLOAD"
	EventAutonomousTransferComplete = "10 AUTONOMOUS TRANSFER COMPLETE"
)

// Vendor-specific event codes (M = Manufacturer).
const (
	EventMReboot   = "M Reboot"
	EventMDownload = "M Download"
	EventMUpload   = "M Upload"
)

// 5G upgrade finish event code (vendor-specific).
const (
	EventUpgradeFinish = "102 UPGRADE FINISH"
)

// CMCC extended event codes.
const (
	EventAddObject           = "103 ADD OBJECT"
	EventDeleteObject        = "104 DELETE OBJECT"
	EventStartupStageReport  = "105 STARTUP STAGE REPORT"
	EventStartupResultReport = "106 STARTUP RESULT REPORT"
)

// IsBootstrap returns true if the event list contains a BOOTSTRAP event.
func IsBootstrap(events []EventStruct) bool {
	return HasEvent(events, EventBootstrap)
}

// IsPeriodic returns true if the event list contains a PERIODIC event.
func IsPeriodic(events []EventStruct) bool {
	return HasEvent(events, EventPeriodic)
}

// IsBoot returns true if the event list contains a BOOT event.
func IsBoot(events []EventStruct) bool {
	return HasEvent(events, EventBoot)
}

// IsValueChange returns true if the event list contains a VALUE CHANGE event.
func IsValueChange(events []EventStruct) bool {
	return HasEvent(events, EventValueChange)
}

// HasEvent checks if the event list contains a specific event code.
func HasEvent(events []EventStruct, code string) bool {
	for _, e := range events {
		if e.EventCode == code {
			return true
		}
	}
	return false
}

// IsAlarm returns true if the event list contains a CMCC alarm event.
func IsAlarm(events []EventStruct) bool {
	for _, e := range events {
		if strings.Contains(e.EventCode, "ALARM") {
			return true
		}
	}
	return false
}

// IsConnectionRequest returns true if the event list contains a CONNECTION REQUEST event.
func IsConnectionRequest(events []EventStruct) bool {
	return HasEvent(events, EventConnectionRequest)
}

// IsRebootComplete returns true if the event list contains an M Reboot event,
// indicating the device has rebooted in response to a Reboot RPC.
func IsRebootComplete(events []EventStruct) bool {
	return HasEvent(events, EventMReboot)
}

// IsDownloadComplete returns true if the event list contains an M Download event.
func IsDownloadComplete(events []EventStruct) bool {
	return HasEvent(events, EventMDownload)
}

// IsRequestDownload returns true if the event list contains a REQUEST DOWNLOAD event.
func IsRequestDownload(events []EventStruct) bool {
	return HasEvent(events, EventRequestDownload)
}

// IsAutonomousTransferComplete returns true if the event list contains an AUTONOMOUS TRANSFER COMPLETE event.
func IsAutonomousTransferComplete(events []EventStruct) bool {
	return HasEvent(events, EventAutonomousTransferComplete)
}

// IsUpgradeFinish returns true if the event list contains a 102 UPGRADE FINISH event.
func IsUpgradeFinish(events []EventStruct) bool {
	return HasEvent(events, EventUpgradeFinish)
}

// IsStartupStageReport reports whether the device is notifying OMC that an
// automatic-start stage (validation/configuration/cell activation) completed.
func IsStartupStageReport(events []EventStruct) bool {
	return HasEvent(events, EventStartupStageReport)
}

// IsStartupResultReport reports whether the device is publishing the terminal
// result of one automatic-start execution.
func IsStartupResultReport(events []EventStruct) bool {
	return HasEvent(events, EventStartupResultReport)
}

// EventCodes extracts just the event codes from an event list.
func EventCodes(events []EventStruct) []string {
	codes := make([]string, len(events))
	for i, e := range events {
		codes[i] = e.EventCode
	}
	return codes
}
