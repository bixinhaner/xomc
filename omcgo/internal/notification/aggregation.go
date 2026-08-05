package notification

import (
	"crypto/sha256"
	"time"

	"github.com/google/uuid"
)

const defaultAggregationWindow = 15 * time.Minute
const minimumAggregationWindow = time.Minute

// AggregationFact describes one event contribution to a digest bucket. It is
// deliberately provider-neutral: digest rendering and delivery happen later.
type AggregationFact struct {
	RuleVersionID        uuid.UUID
	Channel              string
	RecipientFingerprint []byte
	ScopeFingerprint     []byte
	Severity             int16
	WindowStart          time.Time
	WindowEnd            time.Time
	EventCount           int
}

func aggregationWindow(now time.Time, seconds int) (time.Time, time.Time) {
	window := time.Duration(seconds) * time.Second
	if window < minimumAggregationWindow {
		window = defaultAggregationWindow
	}
	start := now.UTC().Truncate(window)
	return start, start.Add(window)
}

// digestScheduleSequence makes the persistence dedup key unique per digest
// window without adding another schema concept. aggregationWindow guarantees
// at least one minute, so consecutive windows cannot share this sequence.
func digestScheduleSequence(windowEnd time.Time) int {
	return int(windowEnd.UTC().Unix() / int64(time.Minute/time.Second))
}

func aggregationScopeFingerprint(deviceID uuid.UUID) []byte {
	sum := sha256.Sum256(deviceID[:])
	return sum[:]
}
