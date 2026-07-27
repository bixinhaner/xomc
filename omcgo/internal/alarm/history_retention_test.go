package alarm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type stubHistoryRetentionRow struct {
	value any
	err   error
}

func (r stubHistoryRetentionRow) Scan(dest ...any) error {
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

type stubHistoryRetentionDB struct {
	execSQL         []string
	execArgs        [][]any
	policyDropAfter string
}

type atomicHistoryRetentionRow struct {
	value any
}

func (r atomicHistoryRetentionRow) Scan(dest ...any) error {
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

// atomicHistoryRetentionDB 模拟 retention job 的事务语义，Rollback 恢复旧 drop_after。
type atomicHistoryRetentionDB struct {
	policyDropAfter  string
	failAdd          bool
	transactionCalls []string
}

func (db *atomicHistoryRetentionDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "pg_extension") {
		return atomicHistoryRetentionRow{value: 1}
	}
	return atomicHistoryRetentionRow{value: db.policyDropAfter}
}

func (db *atomicHistoryRetentionDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.exec(sql, args...)
}

func (db *atomicHistoryRetentionDB) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	db.transactionCalls = append(db.transactionCalls, "begin")
	return &atomicHistoryRetentionTx{db: db, previousDropAfter: db.policyDropAfter}, nil
}

func (db *atomicHistoryRetentionDB) exec(sql string, args ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(sql, "remove_retention_policy"):
		db.policyDropAfter = ""
	case strings.Contains(sql, "add_retention_policy"):
		if db.failAdd {
			return pgconn.CommandTag{}, errors.New("add retention policy failed")
		}
		db.policyDropAfter = args[0].(string)
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

type atomicHistoryRetentionTx struct {
	db                *atomicHistoryRetentionDB
	previousDropAfter string
}

func (tx *atomicHistoryRetentionTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicHistoryRetentionTx) Commit(context.Context) error {
	tx.db.transactionCalls = append(tx.db.transactionCalls, "commit")
	return nil
}
func (tx *atomicHistoryRetentionTx) Rollback(context.Context) error {
	tx.db.policyDropAfter = tx.previousDropAfter
	tx.db.transactionCalls = append(tx.db.transactionCalls, "rollback")
	return nil
}
func (tx *atomicHistoryRetentionTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *atomicHistoryRetentionTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}
func (tx *atomicHistoryRetentionTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }
func (tx *atomicHistoryRetentionTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicHistoryRetentionTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return tx.db.exec(sql, args...)
}
func (tx *atomicHistoryRetentionTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *atomicHistoryRetentionTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return atomicHistoryRetentionRow{value: tx.db.policyDropAfter}
}
func (tx *atomicHistoryRetentionTx) Conn() *pgx.Conn { return nil }

func (s *stubHistoryRetentionDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "pg_extension") {
		return stubHistoryRetentionRow{value: 1}
	}
	if s.policyDropAfter == "" {
		return stubHistoryRetentionRow{err: pgx.ErrNoRows}
	}
	return stubHistoryRetentionRow{value: s.policyDropAfter}
}

func (s *stubHistoryRetentionDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execSQL = append(s.execSQL, sql)
	s.execArgs = append(s.execArgs, args)
	if strings.Contains(sql, "remove_retention_policy") {
		s.policyDropAfter = ""
	}
	if strings.Contains(sql, "add_retention_policy") {
		s.policyDropAfter = args[0].(string)
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (s *stubHistoryRetentionDB) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return &stubHistoryRetentionTx{db: s}, nil
}

type stubHistoryRetentionTx struct{ db *stubHistoryRetentionDB }

func (tx *stubHistoryRetentionTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubHistoryRetentionTx) Commit(context.Context) error   { return nil }
func (tx *stubHistoryRetentionTx) Rollback(context.Context) error { return nil }
func (tx *stubHistoryRetentionTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *stubHistoryRetentionTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *stubHistoryRetentionTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *stubHistoryRetentionTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubHistoryRetentionTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return tx.db.Exec(ctx, sql, args...)
}
func (tx *stubHistoryRetentionTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *stubHistoryRetentionTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return tx.db.QueryRow(ctx, sql, args...)
}
func (tx *stubHistoryRetentionTx) Conn() *pgx.Conn { return nil }

type stubHistoryRetentionReader struct {
	row   *HistoryRetentionConfigRow
	err   error
	calls int
}

func (s *stubHistoryRetentionReader) GetByKey(_ context.Context, category, key string) (*HistoryRetentionConfigRow, error) {
	if category != HistoryRetentionCategory || key != HistoryRetentionKey {
		return nil, nil
	}
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.row, nil
}

type stubHistoryRetentionApplier struct {
	calledWith []int
	err        error
}

func (s *stubHistoryRetentionApplier) Apply(_ context.Context, days int) error {
	s.calledWith = append(s.calledWith, days)
	return s.err
}

func TestHistoryRetentionServiceReloadAndApplyUsesConfiguredDays(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "90", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.Equal(t, []int{90}, applier.calledWith)
	require.Equal(t, 90, svc.CurrentDays())
}

func TestHistoryRetentionServiceReloadAndApplyAcceptsLegacyStringValueType(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "45", ValueType: "string"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.Equal(t, []int{45}, applier.calledWith)
	require.Equal(t, 45, svc.CurrentDays())
}

func TestHistoryRetentionServiceReloadAndApplyFallsBackToDefaultWhenMissing(t *testing.T) {
	reader := &stubHistoryRetentionReader{}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.Equal(t, []int{DefaultHistoryRetentionDays}, applier.calledWith)
	require.Equal(t, DefaultHistoryRetentionDays, svc.CurrentDays())
}

func TestHistoryRetentionServiceReloadAndApplyFallsBackToDefaultWhenInvalid(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "0", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.Equal(t, []int{DefaultHistoryRetentionDays}, applier.calledWith)
	require.Equal(t, DefaultHistoryRetentionDays, svc.CurrentDays())
}

func TestHistoryRetentionServiceSkipsReapplyWhenUnchanged(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "120", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.Equal(t, []int{120}, applier.calledWith)
}

func TestHistoryRetentionServiceOnSysConfigSavedOnlyReactsToStorageCategory(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "60", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	svc.OnSysConfigSaved(context.Background(), "security")
	require.Empty(t, applier.calledWith)

	svc.OnSysConfigSaved(context.Background(), HistoryRetentionCategory)
	require.Equal(t, []int{60}, applier.calledWith)
}

func TestHistoryRetentionServiceApplyForSysConfigCategoryReturnsApplyError(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "60", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{err: errors.New("apply failed")}
	svc := NewHistoryRetentionService(reader, applier, nil)

	err := svc.ApplyForSysConfigCategory(context.Background(), HistoryRetentionCategory)

	require.ErrorContains(t, err, "apply failed")
}

func TestHistoryRetentionServiceApplyForSysConfigCategoryUsesDefaultWhenMissing(t *testing.T) {
	reader := &stubHistoryRetentionReader{err: commonerrors.ErrNotFound}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ApplyForSysConfigCategory(context.Background(), HistoryRetentionCategory))
	require.Equal(t, []int{DefaultHistoryRetentionDays}, applier.calledWith)
}

func TestHistoryRetentionServiceApplyForSysConfigCategoryReconcilesUnchangedCachedValue(t *testing.T) {
	reader := &stubHistoryRetentionReader{row: &HistoryRetentionConfigRow{Value: "60", ValueType: "int"}}
	applier := &stubHistoryRetentionApplier{}
	svc := NewHistoryRetentionService(reader, applier, nil)

	require.NoError(t, svc.ReloadAndApply(context.Background()))
	require.NoError(t, svc.ApplyForSysConfigCategory(context.Background(), HistoryRetentionCategory))
	require.Equal(t, []int{60, 60}, applier.calledWith,
		"tracked config apply must reconcile TimescaleDB even when the in-memory value is unchanged")
}

func TestTimescaleHistoryRetentionApplierApplyUsesFixedDailySchedule(t *testing.T) {
	db := &stubHistoryRetentionDB{}
	applier := &TimescaleHistoryRetentionApplier{pool: db}

	require.NoError(t, applier.Apply(context.Background(), 20))
	require.Len(t, db.execSQL, 2)
	require.Contains(t, db.execSQL[1], "schedule_interval => INTERVAL '1 day'")
	require.Contains(t, db.execSQL[1], "initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08'")
	require.Contains(t, db.execSQL[1], "timezone => 'Asia/Shanghai'")
	require.NotContains(t, db.execSQL[1], "run_job")
	require.NotContains(t, db.execSQL[1], "drop_chunks")
	require.NotContains(t, db.execSQL[1], "DELETE FROM alarms_history")
	require.Equal(t, []any{"20 days"}, db.execArgs[1])
}

func TestTimescaleHistoryRetentionApplierRollsBackOldPolicyWhenAddFailsAtomic(t *testing.T) {
	db := &atomicHistoryRetentionDB{policyDropAfter: "365 days", failAdd: true}
	applier := &TimescaleHistoryRetentionApplier{pool: db}

	err := applier.Apply(context.Background(), 20)

	require.ErrorContains(t, err, "add retention policy failed")
	require.Equal(t, "365 days", db.policyDropAfter, "rollback must preserve the old policy")
	require.Equal(t, []string{"begin", "rollback"}, db.transactionCalls)
}

func TestTimescaleHistoryRetentionApplierConstructedWithNilPoolReturnsError(t *testing.T) {
	applier := NewTimescaleHistoryRetentionApplier(nil)
	require.EqualError(t, applier.Apply(context.Background(), 20), "timescale pool is nil")
}
