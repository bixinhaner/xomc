package notification

import "github.com/omcgo/omcgo/internal/core/model"

const SuppressionRecipientRateLimit = "recipient_rate_limit"

type RateLimitPolicy struct {
	MaxCount         int  `json:"max_count"`
	WindowSeconds    int  `json:"window_seconds"`
	OverflowToDigest bool `json:"overflow_to_digest"`
}

type RateLimitDecision struct {
	Allowed   bool
	Aggregate bool
	Reason    string
}

// EvaluateRateLimit applies the per-recipient ceiling. Critical alarms may
// bypass digest preference, but never bypass an explicit recipient safety cap.
func EvaluateRateLimit(policy RateLimitPolicy, currentCount int, _ model.AlarmSeverity) RateLimitDecision {
	if policy.MaxCount <= 0 || currentCount < policy.MaxCount {
		return RateLimitDecision{Allowed: true}
	}
	return RateLimitDecision{
		Allowed:   false,
		Aggregate: policy.OverflowToDigest,
		Reason:    SuppressionRecipientRateLimit,
	}
}
