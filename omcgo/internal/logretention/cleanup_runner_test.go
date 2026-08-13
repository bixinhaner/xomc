package logretention

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePool 记录每条 DELETE，并按表名预设「每批删多少行」的序列，模拟分批收敛。
type fakePool struct {
	calls     []string
	failTable string // 该表名在 SQL 中出现时返回错误
	// rowsByTable: 表名 -> 每次调用依次返回的 RowsAffected；耗尽后返回 0。
	rowsByTable map[string][]int64
	cursor      map[string]int
}

func newFakePool() *fakePool {
	return &fakePool{rowsByTable: map[string][]int64{}, cursor: map[string]int{}}
}

func (f *fakePool) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.calls = append(f.calls, sql)
	var table string
	for _, t := range Tables {
		if strings.Contains(sql, "FROM "+t.Table+" ") {
			table = t.Table
			break
		}
	}
	if f.failTable != "" && table == f.failTable {
		return pgconn.CommandTag{}, fmt.Errorf("boom")
	}
	seq := f.rowsByTable[table]
	idx := f.cursor[table]
	var n int64
	if idx < len(seq) {
		n = seq[idx]
	}
	f.cursor[table] = idx + 1
	return pgconn.NewCommandTag(fmt.Sprintf("DELETE %d", n)), nil
}

func staticLookup(values map[string]string) ConfigLookup {
	return func(_ context.Context, category, key string) (string, bool) {
		if category != Category {
			return "", false
		}
		v, ok := values[key]
		return v, ok
	}
}

func TestRetentionPolicy_Defaults(t *testing.T) {
	// nil lookup → 统一数据库日志默认值；enabled 默认 true。
	p := NewRetentionPolicy(nil, nil)
	assert.True(t, p.Enabled(context.Background()))
	assert.Equal(t, DefaultDatabaseLogDays, p.DatabaseDays(context.Background()))
}

func TestRetentionPolicy_ReadAndValidate(t *testing.T) {
	p := NewRetentionPolicy(staticLookup(map[string]string{
		KeyDatabaseDays: "365",
		KeyEnabled:      "false",
	}), nil)
	ctx := context.Background()
	assert.False(t, p.Enabled(ctx))
	assert.Equal(t, 365, p.DatabaseDays(ctx))

	invalid := NewRetentionPolicy(staticLookup(map[string]string{KeyDatabaseDays: "notanint"}), nil)
	assert.Equal(t, DefaultDatabaseLogDays, invalid.DatabaseDays(ctx))
}

func TestCleanupRunner_DisabledSkips(t *testing.T) {
	pool := newFakePool()
	p := NewRetentionPolicy(staticLookup(map[string]string{KeyEnabled: "false"}), nil)
	r := NewCleanupRunner(pool, p, nil)

	res, err := r.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, pool.calls, "disabled 时不应执行任何 DELETE")

	var m map[string]any
	require.NoError(t, json.Unmarshal(res, &m))
	assert.Equal(t, true, m["skipped"])
}

func TestCleanupRunner_BatchesUntilDrained(t *testing.T) {
	pool := newFakePool()
	// audit_logs：第一批满批(5000)→继续，第二批不足(1200)→停。共删 6200。
	pool.rowsByTable["audit_logs"] = []int64{deleteBatchSize, 1200}
	// 其它表：第一批就不足一批 → 各一次调用。
	for _, tbl := range Tables {
		if tbl.Table != "audit_logs" {
			pool.rowsByTable[tbl.Table] = []int64{3}
		}
	}
	p := NewRetentionPolicy(nil, nil) // 默认 enabled + 默认天数
	r := NewCleanupRunner(pool, p, nil)

	res, err := r.Run(context.Background(), nil)
	require.NoError(t, err)

	// audit_logs 应被调用 2 次（满批后再来一次），其余表各 1 次。
	auditCalls := 0
	for _, c := range pool.calls {
		if strings.Contains(c, "FROM audit_logs ") {
			auditCalls++
		}
	}
	assert.Equal(t, 2, auditCalls)
	assert.Len(t, pool.calls, 2+(len(Tables)-1)) // audit 2 次 + 其余各 1 次

	var m struct {
		Deleted map[string]int `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(res, &m))
	assert.Equal(t, deleteBatchSize+1200, m.Deleted["audit"])
	assert.Equal(t, 3, m.Deleted["event"])
}

func TestCleanupRunner_PerTableErrorIsolated(t *testing.T) {
	pool := newFakePool()
	pool.failTable = "ne_message_logs" // 该表报错
	for _, tbl := range Tables {
		pool.rowsByTable[tbl.Table] = []int64{2}
	}
	p := NewRetentionPolicy(nil, nil)
	r := NewCleanupRunner(pool, p, nil)

	res, err := r.Run(context.Background(), nil)
	require.NoError(t, err, "单表报错不应让整个 Run 失败")

	var m struct {
		Deleted map[string]int `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(res, &m))
	assert.Equal(t, 0, m.Deleted["ne_message"], "报错表删 0")
	assert.Equal(t, 2, m.Deleted["audit"], "其它表照常清理")
}

// 确保 SQL 用了每张表正确的时间列（白名单拼接正确性回归）。
func TestCleanupRunner_UsesCorrectTimeColumn(t *testing.T) {
	pool := newFakePool()
	for _, tbl := range Tables {
		pool.rowsByTable[tbl.Table] = []int64{0}
	}
	r := NewCleanupRunner(pool, NewRetentionPolicy(nil, nil), nil)
	_, err := r.Run(context.Background(), nil)
	require.NoError(t, err)

	byTable := map[string]string{}
	for _, c := range pool.calls {
		for _, tbl := range Tables {
			if strings.Contains(c, "FROM "+tbl.Table+" WHERE "+tbl.TimeCol+" <") {
				byTable[tbl.Table] = tbl.TimeCol
			}
		}
	}
	for _, tbl := range Tables {
		assert.Equal(t, tbl.TimeCol, byTable[tbl.Table], "table %s should filter on %s", tbl.Table, tbl.TimeCol)
	}
}

func TestCleanupRunner_UsesTableSpecificRetentionDays(t *testing.T) {
	pool := newFakePool()
	for _, tbl := range Tables {
		pool.rowsByTable[tbl.Table] = []int64{0}
	}
	p := NewRetentionPolicy(staticLookup(map[string]string{KeyDatabaseDays: "365"}), nil)
	r := NewCleanupRunner(pool, p, nil)

	res, err := r.Run(context.Background(), nil)
	require.NoError(t, err)

	var m struct {
		RetentionDays map[string]int `json:"retention_days"`
	}
	require.NoError(t, json.Unmarshal(res, &m))
	assert.Equal(t, 365, m.RetentionDays["audit"])
	assert.Equal(t, DefaultNorthboundAPIInvocationLogDays, m.RetentionDays["northbound_api"])
}
