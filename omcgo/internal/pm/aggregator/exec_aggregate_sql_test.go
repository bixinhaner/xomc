package aggregator

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAggTx 是 execAggregateSQL 事务路径的最小 pgx.Tx 假实现：记录每次 Exec 的
// SQL/args 顺序，便于断言 "SET LOCAL work_mem 先于聚合 SQL 执行、且在同一事务内"。
type fakeAggTx struct {
	execSQLs  []string
	execArgs  [][]any
	execErrAt int // 第几次 Exec（0-based）返回 execErr；<0 表示都不出错
	execErr   error
	commits   int
	rollbacks int
	commitErr error
}

func (f *fakeAggTx) Begin(context.Context) (pgx.Tx, error) { return f, nil }

func (f *fakeAggTx) Commit(context.Context) error {
	f.commits++
	return f.commitErr
}

func (f *fakeAggTx) Rollback(context.Context) error {
	f.rollbacks++
	return nil
}

func (f *fakeAggTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeAggTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeAggTx) LargeObjects() pgx.LargeObjects                        { return pgx.LargeObjects{} }
func (f *fakeAggTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (f *fakeAggTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	idx := len(f.execSQLs)
	f.execSQLs = append(f.execSQLs, sql)
	f.execArgs = append(f.execArgs, args)
	if f.execErr != nil && idx == f.execErrAt {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (f *fakeAggTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeAggTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (f *fakeAggTx) Conn() *pgx.Conn                                         { return nil }

// fakeAggBeginner 同时满足 PgQuerier（可赋给 a.db）与 txBeginner（execAggregateSQL
// 的类型断言目标）。Begin 成功时，Exec/Query/QueryRow 不应被直接调用（都应走 tx 分支）。
type fakeAggBeginner struct {
	tx       *fakeAggTx
	beginErr error
	begins   int
}

func (f *fakeAggBeginner) Begin(context.Context) (pgx.Tx, error) {
	f.begins++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.tx, nil
}
func (f *fakeAggBeginner) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("fakeAggBeginner.Exec should not be called when Begin succeeds")
}
func (f *fakeAggBeginner) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeAggBeginner.Query unimplemented")
}
func (f *fakeAggBeginner) QueryRow(context.Context, string, ...any) pgx.Row {
	return errRow{err: errors.New("fakeAggBeginner.QueryRow unimplemented")}
}

func Test_execAggregateSQL_WithTxBeginner_SetsLocalWorkMemBeforeSQL_ThenCommits(t *testing.T) {
	tx := &fakeAggTx{execErrAt: -1}
	beginner := &fakeAggBeginner{tx: tx}
	a := New(beginner, nil, nil)

	tag, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.NoError(t, err)
	assert.Equal(t, int64(1), tag.RowsAffected())

	require.Len(t, tx.execSQLs, 2, "应先 SET LOCAL work_mem 再跑聚合 SQL，两次 Exec")
	assert.Contains(t, tx.execSQLs[0], "SET LOCAL work_mem")
	assert.Contains(t, tx.execSQLs[0], aggregateWorkMem)
	assert.Equal(t, "INSERT INTO pm_metrics_hourly ...", tx.execSQLs[1])
	assert.Equal(t, []any{"hourly"}, tx.execArgs[1])
	assert.Equal(t, 1, tx.commits, "应提交一次")
	assert.Equal(t, 1, beginner.begins, "应开一次事务")
}

func Test_execAggregateSQL_FallsBackToPlainExec_WhenDBDoesNotSupportBegin(t *testing.T) {
	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	a := New(db, nil, nil)

	tag, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.NoError(t, err)
	assert.Equal(t, int64(5), tag.RowsAffected())
	assert.Equal(t, "INSERT INTO pm_metrics_hourly ...", db.execSQL, "不支持 Begin 时应退回普通 Exec，不加 SET LOCAL")
}

func Test_execAggregateSQL_BeginFails_ReturnsWrappedError_NoCommit(t *testing.T) {
	beginner := &fakeAggBeginner{beginErr: errors.New("connection refused")}
	a := New(beginner, nil, nil)

	_, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin aggregate tx")
}

func Test_execAggregateSQL_SetLocalWorkMemFails_ReturnsError_NoCommit(t *testing.T) {
	tx := &fakeAggTx{execErrAt: 0, execErr: errors.New("syntax error")}
	beginner := &fakeAggBeginner{tx: tx}
	a := New(beginner, nil, nil)

	_, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set local work_mem")
	assert.Equal(t, 0, tx.commits, "SET LOCAL 失败不应提交")
	assert.Len(t, tx.execSQLs, 1, "SET LOCAL 失败后不应再跑聚合 SQL")
}

func Test_execAggregateSQL_MainSQLFails_ReturnsRawError_NoCommit(t *testing.T) {
	wantErr := errors.New("deadlock detected")
	tx := &fakeAggTx{execErrAt: 1, execErr: wantErr}
	beginner := &fakeAggBeginner{tx: tx}
	a := New(beginner, nil, nil)

	_, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 0, tx.commits, "聚合 SQL 失败不应提交")
}

func Test_execAggregateSQL_CommitFails_ReturnsWrappedError(t *testing.T) {
	tx := &fakeAggTx{execErrAt: -1, commitErr: errors.New("could not serialize access")}
	beginner := &fakeAggBeginner{tx: tx}
	a := New(beginner, nil, nil)

	_, err := a.execAggregateSQL(context.Background(), "INSERT INTO pm_metrics_hourly ...", "hourly")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit aggregate tx")
}
