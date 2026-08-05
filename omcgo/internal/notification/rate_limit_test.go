package notification

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
)

func TestEvaluateRateLimit_CriticalStillRespectsRecipientLimit(t *testing.T) {
	result := EvaluateRateLimit(RateLimitPolicy{MaxCount: 5}, 5, model.AlarmCritical)
	require.False(t, result.Allowed)
	require.Equal(t, SuppressionRecipientRateLimit, result.Reason)
}

func TestEvaluateRateLimit_OverflowCanBecomeDigestFact(t *testing.T) {
	result := EvaluateRateLimit(RateLimitPolicy{MaxCount: 2, OverflowToDigest: true}, 2, model.AlarmMajor)
	require.False(t, result.Allowed)
	require.True(t, result.Aggregate)
}
