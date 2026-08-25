package deviceaccess

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestStateFiltersCoverAuditDimensionsWithParameters(t *testing.T) {
	policyVersionID := uuid.New()
	matchedRuleID := uuid.New()
	startedAt := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)
	filter := ManagementFilter{
		Carrier: "cmcc", SerialNumber: "SN-AUDIT", State: AccessStateRejected,
		Decision: EffectiveActionReject, ReasonCode: ReasonRuleMatched,
		ActionStatus: ActionStatusDead,
		Dimension:    ConditionTypeGPS, PolicyVersionID: &policyVersionID, MatchedRuleID: &matchedRuleID,
		StartedAt: &startedAt, EndedAt: &endedAt,
	}

	query, args, err := storage.Psql.Select("COUNT(*)").From("device_access_states s").
		Where(stateFilters(filter)).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "filtered_decision.matched_rule_id")
	require.Contains(t, query, "dimension_check.check_type")
	require.Contains(t, query, "filtered_action.status")
	require.NotContains(t, query, "SN-AUDIT")
	require.Contains(t, args, "%SN-AUDIT%")
	require.Contains(t, args, policyVersionID.String())
	require.Contains(t, args, matchedRuleID)
}

func TestActionFiltersCoverResultPolicyAndTime(t *testing.T) {
	policyVersionID := uuid.New()
	startedAt := time.Date(2026, 8, 19, 1, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(time.Hour)
	filter := ActionListFilter{
		Status: ActionStatusDead, Decision: EffectiveActionReject, ReasonCode: ReasonRuleMatched,
		PolicyVersionID: &policyVersionID,
		StartedAt:       &startedAt, EndedAt: &endedAt,
	}

	query, args, err := storage.Psql.Select("COUNT(*)").From("device_access_actions a").
		Join("device_access_decisions decision ON decision.id = a.decision_id").
		Where(actionFilters(filter)).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "decision.decision")
	require.Contains(t, query, "decision.reason_code")
	require.Contains(t, query, "decision.policy_version_id")
	require.Contains(t, args, policyVersionID.String())
}

func TestCandidateAttentionFiltersUseExactIDPendingExpiryAndStableOrder(t *testing.T) {
	candidateID := uuid.New()
	now := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	filter := ManagementFilter{
		CandidateID: &candidateID, Status: "pending", ExpiresAfter: &now,
		SortBy: "first_seen_at", SortDir: "asc",
	}

	query, args, err := storage.Psql.Select("c.id").From("device_access_candidates c").
		Where(candidateFilters(filter)).OrderBy(candidateOrderBy(filter)...).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "c.id = $1")
	require.Contains(t, query, "c.review_status = $2")
	require.Contains(t, query, "c.device_id IS NULL")
	require.Contains(t, query, "c.expires_at > $3")
	require.Contains(t, query, "ORDER BY c.first_seen_at ASC, c.id ASC")
	require.Equal(t, []any{candidateID.String(), "pending", now}, args)
}

func TestCandidateAttentionOrderingUsesAllowlist(t *testing.T) {
	require.Equal(t, []string{"c.last_seen_at DESC", "c.id ASC"}, candidateOrderBy(ManagementFilter{
		SortBy: "last_seen_at; DROP TABLE device_access_candidates", SortDir: "asc",
	}))
}
