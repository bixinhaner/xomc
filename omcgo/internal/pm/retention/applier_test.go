package retention

import (
	"context"
	"errors"
	"strconv"
	"strings"
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
	value   any
}

func (r *stubRow) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	switch value := r.value.(type) {
	case int:
		*(dest[0].(*int)) = value
	case string:
		*(dest[0].(*string)) = value
	}
	return nil
}

// stubRetentionQuerier 实现 retentionQuerier，记录所有 SQL 调用。
type stubRetentionQuerier struct {
	queryRowErr     error // Scan 返回的错误；nil 表示 timescaledb 已安装
	execErr         error
	execCalls       []string // 每次 Exec 记录第一个参数（SQL 语句）
	policyDropAfter string
	beginCalls      int
	commitCalls     int
	rollbackCalls   int
}

// atomicRetentionRow 允许原子更新测试为 extension 和 retention 查询分别提供结果。
type atomicRetentionRow struct {
	value any
	err   error
}

func (r atomicRetentionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	switch value := r.value.(type) {
	case int:
		*(dest[0].(*int)) = value
	case string:
		*(dest[0].(*string)) = value
	default:
		return errors.New("unexpected scan value")
	}
	return nil
}

// atomicRetentionDB 模拟 policy 的事务语义：Rollback 恢复 remove 前的 drop_after。
// 它也记录 begin/rollback/commit 顺序，防止实现退回到跨语句更新。
type atomicRetentionDB struct {
	policyDropAfter  string
	failAdd          bool
	transactionCalls []string
}

func (db *atomicRetentionDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "pg_extension") {
		return atomicRetentionRow{value: 1}
	}
	return atomicRetentionRow{value: db.policyDropAfter}
}

func (db *atomicRetentionDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.exec(sql, args...)
}

func (db *atomicRetentionDB) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	db.transactionCalls = append(db.transactionCalls, "begin")
	return &atomicRetentionTx{db: db, previousDropAfter: db.policyDropAfter}, nil
}

func (db *atomicRetentionDB) exec(sql string, args ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(sql, "remove_retention_policy"):
		db.policyDropAfter = ""
	case strings.Contains(sql, "add_retention_policy"):
		if db.failAdd {
			return pgconn.CommandTag{}, errors.New("add retention policy failed")
		}
		db.policyDropAfter = normalizedDropAfter(args[1].(int))
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

type atomicRetentionTx struct {
	db                *atomicRetentionDB
	previousDropAfter string
}

func (tx *atomicRetentionTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicRetentionTx) Commit(context.Context) error {
	tx.db.transactionCalls = append(tx.db.transactionCalls, "commit")
	return nil
}
func (tx *atomicRetentionTx) Rollback(context.Context) error {
	tx.db.policyDropAfter = tx.previousDropAfter
	tx.db.transactionCalls = append(tx.db.transactionCalls, "rollback")
	return nil
}
func (tx *atomicRetentionTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *atomicRetentionTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *atomicRetentionTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *atomicRetentionTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicRetentionTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return tx.db.exec(sql, args...)
}
func (tx *atomicRetentionTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicRetentionTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "INTERVAL '1 day'") {
		return atomicRetentionRow{value: normalizedDropAfter(args[0].(int))}
	}
	return atomicRetentionRow{value: tx.db.policyDropAfter}
}
func (tx *atomicRetentionTx) Conn() *pgx.Conn { return nil }

func (s *stubRetentionQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "pg_extension") {
		return &stubRow{scanErr: s.queryRowErr, value: 1}
	}
	if strings.Contains(sql, "INTERVAL '1 day'") {
		return &stubRow{value: normalizedDropAfter(args[0].(int))}
	}
	if s.policyDropAfter == "" {
		return &stubRow{scanErr: pgx.ErrNoRows}
	}
	return &stubRow{value: s.policyDropAfter}
}

func (s *stubRetentionQuerier) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execCalls = append(s.execCalls, sql)
	if s.execErr != nil {
		return pgconn.CommandTag{}, s.execErr
	}
	if strings.Contains(sql, "remove_retention_policy") {
		s.policyDropAfter = ""
	}
	if strings.Contains(sql, "add_retention_policy") {
		s.policyDropAfter = normalizedDropAfter(args[1].(int))
	}
	return pgconn.CommandTag{}, nil
}

func (s *stubRetentionQuerier) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	s.beginCalls++
	return &stubRetentionTx{stub: s}, nil
}

type stubRetentionTx struct{ stub *stubRetentionQuerier }

func (tx *stubRetentionTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubRetentionTx) Commit(context.Context) error {
	tx.stub.commitCalls++
	return nil
}
func (tx *stubRetentionTx) Rollback(context.Context) error {
	tx.stub.rollbackCalls++
	return nil
}
func (tx *stubRetentionTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *stubRetentionTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *stubRetentionTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *stubRetentionTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubRetentionTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return tx.stub.Exec(ctx, sql, args...)
}
func (tx *stubRetentionTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubRetentionTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return tx.stub.QueryRow(ctx, sql, args...)
}
func (tx *stubRetentionTx) Conn() *pgx.Conn { return nil }

// --- helpers ---

func newApplierWithStub(stub *stubRetentionQuerier) *PMRetentionApplier {
	return &PMRetentionApplier{db: stub, logger: zap.NewNop()}
}

func normalizedDropAfter(days int) string {
	if days == 1 {
		return "1 day"
	}
	return strconv.Itoa(days) + " days"
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

func TestPMRetentionApplierApplyRollsBackOldPolicyWhenAddFailsAtomic(t *testing.T) {
	db := &atomicRetentionDB{policyDropAfter: "30 days", failAdd: true}
	a := &PMRetentionApplier{db: db, logger: zap.NewNop()}

	err := a.Apply(context.Background(), "public.pm_metrics", 60)

	require.ErrorContains(t, err, "add retention policy failed")
	require.Equal(t, "30 days", db.policyDropAfter, "rollback must preserve the old policy")
	require.Equal(t, []string{"begin", "rollback"}, db.transactionCalls)
}

func TestPMRetentionApplierApplyNormalizesPostCheckDropAfter(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected string
	}{
		{name: "one day singular", days: 1, expected: "1 day"},
		{name: "two days plural", days: 2, expected: "2 days"},
		{name: "thirty days plural", days: 30, expected: "30 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &atomicRetentionDB{policyDropAfter: "30 days"}
			a := &PMRetentionApplier{db: db, logger: zap.NewNop()}

			err := a.Apply(context.Background(), "public.pm_metrics", tt.days)

			require.NoError(t, err)
			require.Equal(t, tt.expected, db.policyDropAfter)
			require.Equal(t, []string{"begin", "commit"}, db.transactionCalls)
		})
	}
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

func TestPMRetentionApplierApplyAllRoutesRawSparseTables(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30},
		[]PolicyKey{KeyRaw15MinDays},
	)

	require.Len(t, stub.execCalls, 4)
}

func TestPMRetentionApplierApplyAllIgnoresHourlyMixedResultTable(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyHourlyDays: 180},
		[]PolicyKey{KeyHourlyDays},
	)

	assert.Empty(t, stub.execCalls)
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

	// 只传 KeyRaw15MinDays 变更；两张原始稀疏表原子更新。
	a.ApplyAll(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30, KeyHourlyDays: 180},
		[]PolicyKey{KeyRaw15MinDays}, // 只有这一个变了
	)

	require.Len(t, stub.execCalls, 4)
}

// TestPMRetentionApplierApplyAllContinuesOnError 验证某张表 Exec 失败时
// ApplyAll 不中止，继续处理后续表（fire-and-log 语义）。
func TestPMRetentionApplierApplyAllReturnsFirstError(t *testing.T) {
	stub := &stubRetentionQuerier{execErr: errors.New("db error")}
	a := newApplierWithStub(stub)

	err := a.ApplyAllWithError(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30},
		[]PolicyKey{KeyRaw15MinDays},
	)
	require.ErrorContains(t, err, "db error")
}

func TestPMRetentionApplierApplyAllUpdatesAffectedHypertablesInOneTransaction(t *testing.T) {
	stub := &stubRetentionQuerier{}
	a := newApplierWithStub(stub)

	err := a.ApplyAllWithError(context.Background(),
		map[PolicyKey]int{KeyRaw15MinDays: 30, KeyHourlyDays: 180},
		[]PolicyKey{KeyRaw15MinDays, KeyHourlyDays},
	)

	require.NoError(t, err)
	require.Equal(t, 1, stub.beginCalls)
	require.Equal(t, 1, stub.commitCalls)
	require.Equal(t, 0, stub.rollbackCalls)
	require.Len(t, stub.execCalls, 4) // two raw hypertables, remove + add each
}
