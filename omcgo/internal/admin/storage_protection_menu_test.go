package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyStorageProtectionMenuRetireSQLContract(t *testing.T) {
	query, args, err := buildLegacyStorageProtectionMenuRetireSQL()
	require.NoError(t, err)

	require.Contains(t, query, "UPDATE menus")
	require.Contains(t, query, "route_path = $1")
	require.Contains(t, query, "component_path = $2")
	require.Contains(t, query, "show_status = $3")
	require.Contains(t, query, "status = $4")
	require.Contains(t, query, "updated_at = now()")
	require.Contains(t, query, "route_path = $6")
	require.Contains(t, query, "component_path = $7")
	require.Equal(t, []interface{}{
		retiredStorageProtectionRoutePath,
		retiredStorageProtectionComponentPath,
		string(MenuHide),
		string(MenuStatusDisabled),
		legacyStorageProtectionMenuID,
		legacyStorageProtectionRoutePath,
		legacyStorageProtectionComponentPath,
	}, stringifyUUIDArgs(args))
}

func stringifyUUIDArgs(args []interface{}) []interface{} {
	out := make([]interface{}, len(args))
	for i, arg := range args {
		out[i] = arg
		if s, ok := arg.(interface{ String() string }); ok {
			out[i] = s.String()
		}
	}
	return out
}
