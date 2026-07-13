package indicator

import "regexp"

var durationTokenPattern = regexp.MustCompile(`\bDuration\b`)

const (
	StatisDurationCounterENB = "C000060273"
	StatisDurationCounterGNB = "C010120025"
	StatisDurationCounterGSM = "CGSM0080001"
)

// CompileRuntimeArithmetic compiles KPI formula reserved words into runtime
// counter IDs. Persisted formulas may keep readable tokens like Duration, but
// KPI runtime paths consume numbered counter IDs only.
func CompileRuntimeArithmetic(dt DeviceType, arithmetic string) string {
	durationID, ok := StatisDurationCounterID(dt)
	if !ok {
		return arithmetic
	}
	return durationTokenPattern.ReplaceAllString(arithmetic, durationID)
}

func StatisDurationCounterID(dt DeviceType) (string, bool) {
	switch dt {
	case DeviceTypeENB:
		return StatisDurationCounterENB, true
	case DeviceTypeGNB:
		return StatisDurationCounterGNB, true
	case DeviceTypeGSM:
		return StatisDurationCounterGSM, true
	default:
		return "", false
	}
}
