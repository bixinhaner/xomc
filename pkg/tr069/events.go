package tr069

// TR069 Inform event codes per CWMP specification.
const (
	EventBootstrap           = "0 BOOTSTRAP"
	EventBoot                = "1 BOOT"
	EventPeriodic            = "2 PERIODIC"
	EventScheduled           = "3 SCHEDULED"
	EventValueChange         = "4 VALUE CHANGE"
	EventKicked              = "5 KICKED"
	EventConnectionRequest   = "6 CONNECTION REQUEST"
	EventTransferComplete    = "7 TRANSFER COMPLETE"
	EventDiagnosticsComplete = "8 DIAGNOSTICS COMPLETE"
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

// EventCodes extracts just the event codes from an event list.
func EventCodes(events []EventStruct) []string {
	codes := make([]string, len(events))
	for i, e := range events {
		codes[i] = e.EventCode
	}
	return codes
}
