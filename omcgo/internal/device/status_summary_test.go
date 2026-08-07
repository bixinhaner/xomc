package device

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildTechnologyStatusSummaryQuery_UsesNonCPEStatusDimensions(t *testing.T) {
	query, args, err := BuildTechnologyStatusSummaryQuery()
	require.NoError(t, err)
	require.Len(t, args, 3)
	require.Contains(t, query, "d.technology IN ($1,$2,$3)")
	require.Contains(t, query, "d.is_online = TRUE")
	require.Contains(t, query, "COALESCE(di.op_state, '0') = '1'")
	require.Contains(t, query, "COUNT(DISTINCT d.id)")
	require.Contains(t, query, "excluded_cpe_count")
	require.Contains(t, query, "COALESCE((")
	require.Contains(t, query, "d.product_class ILIKE '%cpe%'")
	require.Contains(t, query, "d.deleted_at IS NULL")
}
