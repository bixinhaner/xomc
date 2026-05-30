package adhoc

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0186：运行历史查询 SQL 倒序 + 分页（不依赖 DB，断言生成 SQL）。
func Test_BuildListRunsSQL_OrderDescAndPagination(t *testing.T) {
	taskID := uuid.New()

	t.Run("默认倒序 started_at 无分页", func(t *testing.T) {
		q, args, err := buildListRunsSQL(taskID, 0, 0)
		require.NoError(t, err)
		assert.True(t, strings.Contains(q, "ORDER BY started_at DESC"), "应按 started_at 倒序: %s", q)
		assert.False(t, strings.Contains(q, "LIMIT"), "limit=0 不应带 LIMIT: %s", q)
		assert.False(t, strings.Contains(q, "OFFSET"), "offset=0 不应带 OFFSET: %s", q)
		require.Len(t, args, 1)
		assert.Equal(t, taskID.String(), args[0])
	})

	t.Run("带 limit 与 offset 分页", func(t *testing.T) {
		q, _, err := buildListRunsSQL(taskID, 20, 40)
		require.NoError(t, err)
		assert.True(t, strings.Contains(q, "ORDER BY started_at DESC"), q)
		assert.True(t, strings.Contains(q, "LIMIT 20"), "应带 LIMIT 20: %s", q)
		assert.True(t, strings.Contains(q, "OFFSET 40"), "应带 OFFSET 40: %s", q)
	})

	t.Run("仅 limit", func(t *testing.T) {
		q, _, err := buildListRunsSQL(taskID, 5, 0)
		require.NoError(t, err)
		assert.True(t, strings.Contains(q, "LIMIT 5"), q)
		assert.False(t, strings.Contains(q, "OFFSET"), q)
	})
}
