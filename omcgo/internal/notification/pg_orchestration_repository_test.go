package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var _ OrchestrationRepository = (*PgOrchestrationRepository)(nil)

func TestBuildAggregationInsert_UsesStableBucketKeyAndReturnsOnlyNewBucket(t *testing.T) {
	query, args, err := buildAggregationInsert(AggregationFact{
		RuleVersionID: uuid.New(), Channel: "email", RecipientFingerprint: []byte("recipient"),
		ScopeFingerprint: []byte("scope"), Severity: 3,
		WindowStart: time.Now().UTC(), WindowEnd: time.Now().UTC().Add(15 * time.Minute), EventCount: 1,
	})
	require.NoError(t, err)
	require.Len(t, args, 10)
	normalized := strings.ToUpper(query)
	require.Contains(t, normalized, "ON CONFLICT (RULE_VERSION_ID, CHANNEL, RECIPIENT_FINGERPRINT, SCOPE_FINGERPRINT, SEVERITY, WINDOW_STARTED_AT)")
	require.Contains(t, normalized, "DO NOTHING RETURNING ID")
}
