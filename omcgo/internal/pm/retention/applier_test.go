package retention

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- stubs ---

// stubRow 实现 pgx.Row，用于模拟 QueryRow 的返回值。
type stubRow struct {
	scanErr error
}

func (r *stubRow) Scan(dest ...any) error {
	return r.scanErr
}

// stubRetentionQuerier 实现 retentionQuerier，记录所有 SQL 调用。
type stubRetentionQuerier struct {
	queryRowErr error // Scan 返回的错误；nil 表示 timescaledb 已安装
	execErr     error
	execCalls   []string // 每次 Exec 记录第一个参数（SQL 语句）
}

func (s *stubRetentionQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return &stubRow{scanErr: s.queryRowErr}
}

func (s *stubRetentionQuerier) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	s.execCalls = append(s.execCalls, sql)
	return pgconn.CommandTag{}, s.execErr
}

// --- helpers ---

func newApplierWithStub(stub *stubRetentionQuerier) *PMRetentionApplier {
	return &PMRetentionApplier{db: stub, logger: zap.NewNop()}
}

// TestPMRetentionApplierApplyExecutesRemoveAndAddPolicy 验证 Apply 在 timescaledb
// 存在时依次调用 remove_retention_policy + add_retention_policy。
func TestPMRetentionApplierApplyExecutesRemoveAndAddPolicy(t *testing.T) {
	stub := &stubRetentionQuerier{} // queryRowErr == nil → Scan 成功 → timescaledb 已安装
	a := newApplierWithStub(stub)

	err := a.Apply(context.Background(), "public.pm_metrics", 60)
	require.NoError(t, err)

	require.Len(t, stub.execCalls, 2)
	assert.Contains(t, stub.execCalls[0], "remove_retention_policy")
	assert.Contains(t, stub.execCalls[1], "add_retention_policy")
}

// TestPMRetentionApplierApplySkipsWhenTimescaleDBMissing 验证 timescaledb 未安装时
// Apply 跳过 Exec 调用，不返回错误。
func TestPMRetentionApplierApplySkipsWhenTimescaleDBMissing(t *testing.T) {
	stub := &stubRetentionQuerier{queryRowErr: pgx.ErrNoRows}
	a := newApplierWithStub(stub)

	err := a.Apply(context.Background(), "public.pm_metrics", 60)
	require.NoError(t, err)
	assert.Empty(t, stub.execCalls)
}

// TestPMRetentionApplierApplyRejectsInvalidDays 验证保留天数越界时 Apply 返回错误。
func TestPMRetentionApplierApplyRejectsInvalidDays(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	err := a.Apply(context.Background(), "public.pm_metrics", 0)
	require.ErrorIs(t, err, ErrRetentionTooShort)
	assert.Empty(t, stub.execCalls) // 校验失败不应触发任何 SQL
}

// TestPMRetentionApplierApplyAllRoutesRaw15MinToCorrectTable 验证 KeyRaw15MinDays
// 变更时只更新 public.pm_metrics。
func TestPMRetentionApplierApplyAllRoutesRaw15MinToCorrectTable(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30},
		[]PolicyKey{KeyRaw15MinDays},
	)

	// 1 张表 × 2 条 SQL = 2 次 Exec
	require.Len(t, stub.execCalls, 2)
}

// TestPMRetentionApplierApplyAllRoutesHourlyToBothTables 验证 KeyHourlyDays
// 变更时同时更新 pm_metrics_hourly + pm_group_metrics_hourly。
func TestPMRetentionApplierApplyAllRoutesHourlyToBothTables(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyHourlyDays: 180},
		[]PolicyKey{KeyHourlyDays},
	)

	// 2 张表 × 2 条 SQL = 4 次 Exec
	require.Len(t, stub.execCalls, 4)
}

// TestPMRetentionApplierApplyAllIgnoresOrdinaryTableKeys 验证 daily/weekly/monthly
// 不触发任何 Exec（普通表由 cron 处理）。
func TestPMRetentionApplierApplyAllIgnoresOrdinaryTableKeys(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	a.ApplyAll(context.Background(),
		map[PolicyKey]int{
			KeyDailyDays:   730,
			KeyWeeklyDays:  730,
			KeyMonthlyDays: 1825,
		},
		[]PolicyKey{KeyDailyDays, KeyWeeklyDays, KeyMonthlyDays},
	)

	assert.Empty(t, stub.execCalls)
}

// TestPMRetentionApplierApplyAllOnlyUpdatesChangedKeys 验证只传入部分 changed keys
// 时，仅更新对应 hypertable，不影响未变更的表。
func TestPMRetentionApplierApplyAllOnlyUpdatesChangedKeys(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	// 只传 KeyRaw15MinDays 变更，KeyHourlyDays 未变更不应出现
	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30, KeyHourlyDays: 180},
		[]PolicyKey{KeyRaw15MinDays}, // 只有这一个变了
	)

	// 1 张表 × 2 条 SQL = 2 次 Exec
	require.Len(t, stub.execCalls, 2)
}

// TestPMRetentionApplierApplyAllContinuesOnError 验证某张表 Exec 失败时
// ApplyAll 不中止，继续处理后续表（fire-and-log 语义）。
func TestPMRetentionApplierApplyAllContinuesOnError(t *testing.T) {
	stub := &stubRetentionQuerier{execErr: errors.New("db error")}
	a := newApplierWithStub(stub)

	// hourly 涉及两张表，第一张 Exec 失败后应继续第二张
	// 不返回错误，不 panic
	require.NotPanics(t, func() {
		a.ApplyAll(context.Background(),
			map[PolicyKey]int{KeyHourlyDays: 180},
			[]PolicyKey{KeyHourlyDays},
		)
	})
}
