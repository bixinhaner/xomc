package deviceaccess

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestAccessNotificationScopeIncludesDecisionAndActionSources(t *testing.T) {
	query, args, err := storage.Psql.Select("history.id").From("notification_history history").
		Where(accessNotificationScope(ManagementFilter{
			Carrier: " cmcc ", SerialNumber: " SN-001 ",
		})).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "history.source_type = $1")
	require.Contains(t, query, "FROM device_access_decisions source_decision")
	require.Contains(t, query, "source_decision.id = history.source_id")
	require.Contains(t, query, "history.source_type = $4")
	require.Contains(t, query, "FROM device_access_actions source_action")
	require.Contains(t, query, "source_decision.id = source_action.decision_id")
	require.Contains(t, query, "source_action.id = history.source_id")
	require.NotContains(t, query, "LEFT JOIN")
	require.Equal(t, []any{
		"device_access_decision", "cmcc", "SN-001",
		"device_access_action", "cmcc", "SN-001",
	}, args)
}

func TestAccessNotificationScopeAppliesCanonicalVisibilityToBothSources(t *testing.T) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	query, args, err := storage.Psql.Select("history.id").From("notification_history history").
		Where(accessNotificationScope(ManagementFilter{
			Carrier: "cmcc", SerialNumber: "SN-001", VisibleGroups: []uuid.UUID{groupID},
		})).ToSql()

	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(query, "device_group_members"))
	require.Equal(t, 2, strings.Count(query, "device_registrations scope_registration"))
	require.Equal(t, 2, strings.Count(query, "source_decision.device_id IS NULL AND EXISTS"))
	require.Equal(t, 4, countValue(args, groupID))
}

func TestAccessNotificationScopeRejectsRestrictedActorWithoutVisibleGroups(t *testing.T) {
	query, _, err := storage.Psql.Select("history.id").From("notification_history history").
		Where(accessNotificationScope(ManagementFilter{
			Carrier: "cmcc", SerialNumber: "SN-001", VisibleGroups: []uuid.UUID{},
		})).ToSql()

	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(query, "FALSE"))
}

func countValue(values []any, target any) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
