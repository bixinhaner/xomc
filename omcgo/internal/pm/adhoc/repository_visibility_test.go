package adhoc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildListSQL_DefaultVisibilityScope(t *testing.T) {
	sql, args, err := buildListSQL(ListFilter{
		CurrentUser: "bob",
		Limit:       50,
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "task_subtype")
	assert.Contains(t, sql, "is_builtin")
	assert.Contains(t, sql, "visibility")
	assert.Contains(t, sql, "creator")
	assert.Contains(t, sql, "OR", "默认列表应是内置 / public / 当前用户任务的并集")
	assert.Contains(t, sql, "LIMIT")
	assert.Contains(t, args, "bob")
	assert.Contains(t, args, string(VisibilityPublic))
}

func Test_buildListSQL_AdminAllSkipsVisibilityScope(t *testing.T) {
	sql, args, err := buildListSQL(ListFilter{
		CurrentUser: "admin-user",
		IncludeAll:  true,
		Limit:       50,
	})
	require.NoError(t, err)

	whereClause := sql
	if idx := strings.Index(sql, "ORDER BY"); idx >= 0 {
		whereClause = sql[:idx]
	}
	assert.NotContains(t, whereClause, "visibility =", "admin all=true 不应加 visibility 范围过滤")
	assert.NotContains(t, args, "admin-user", "admin all=true 不需要当前用户参数")
}

func Test_buildListSQL_CustomOnlyStillIncludesPublicOrCurrentUser(t *testing.T) {
	customOnly := false
	sql, args, err := buildListSQL(ListFilter{
		CurrentUser: "bob",
		IsBuiltin:   &customOnly,
		Limit:       50,
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "is_builtin")
	assert.Contains(t, sql, "visibility")
	assert.Contains(t, sql, "creator")
	assert.Contains(t, args, false)
	assert.Contains(t, args, "bob")
	assert.Contains(t, args, string(VisibilityPublic))
}
