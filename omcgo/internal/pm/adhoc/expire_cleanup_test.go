package adhoc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubExecer 记录 Exec 的 SQL，并返回预置的 RowsAffected / error。
type stubExecer struct {
	lastSQL string
	calls   int
	rows    int64
	err     error
}

func (s *stubExecer) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	s.calls++
	s.lastSQL = sql
	if s.err != nil {
		return pgconn.CommandTag{}, s.err
	}
	return pgconn.NewCommandTag("DELETE " + itoa(s.rows)), nil
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// Test_ExpireCleanup_SQLShape：断言删除 SQL 的过滤条件齐全：
// 只删 adhoc 子类型 + oneshot + 非内置 + 按 created_at+expire_days 过期，且只动 pm_tasks。
func Test_ExpireCleanup_SQLShape(t *testing.T) {
	db := &stubExecer{rows: 0}
	c := NewExpireCleanup(db, nil)
	_, err := c.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, db.calls)

	sql := db.lastSQL
	assert.True(t, strings.HasPrefix(strings.TrimSpace(sql), "DELETE FROM pm_tasks"), "只删 pm_tasks 任务定义行")
	assert.Contains(t, sql, "task_subtype = 'adhoc_aggregation'", "限 adhoc 子类型")
	assert.Contains(t, sql, "mode = 'oneshot'", "只删 oneshot（continuous 排除）")
	assert.Contains(t, sql, "is_builtin = false", "内置任务排除")
	assert.Contains(t, sql, "created_at < NOW() -", "按 created_at + expire_days 过期")
	assert.Contains(t, sql, "expire_days", "按每行自己的 expire_days 算过期")
	// 绝不碰结果表
	assert.NotContains(t, sql, "pm_adhoc_aggregation_results", "绝不触碰结果表")
}

// Test_ExpireCleanup_ReturnsDeletedCount：返回删除行数（来自 CommandTag.RowsAffected）。
func Test_ExpireCleanup_ReturnsDeletedCount(t *testing.T) {
	db := &stubExecer{rows: 7}
	c := NewExpireCleanup(db, nil)
	deleted, err := c.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(7), deleted, "返回 CommandTag 的 RowsAffected")
}

// Test_ExpireCleanup_NothingToDelete：无过期任务时返回 0，不报错。
func Test_ExpireCleanup_NothingToDelete(t *testing.T) {
	db := &stubExecer{rows: 0}
	c := NewExpireCleanup(db, nil)
	deleted, err := c.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

// Test_ExpireCleanup_DBError：Exec 失败时透传错误（不静默吞）。
func Test_ExpireCleanup_DBError(t *testing.T) {
	db := &stubExecer{err: errors.New("boom")}
	c := NewExpireCleanup(db, nil)
	_, err := c.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete expired tasks")
}
