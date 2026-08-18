package ops

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestActiveMaintenanceWindowQueryIncludesApprovedExecutableWindows(t *testing.T) {
	now := time.Date(2026, 8, 18, 5, 30, 0, 0, time.UTC)
	query, args, err := activeMaintenanceWindowQuery(now).ToSql()

	require.NoError(t, err)
	require.Contains(t, query, "status IN ($1,$2)")
	require.Equal(t, []any{string(MWApproved), string(MWActive), now, now}, args)
}
