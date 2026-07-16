package alarm

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type stubHistoryRetentionRow struct {
	value int
}

func (r stubHistoryRetentionRow) Scan(dest ...any) error {
	*(dest[0].(*int)) = r.value
	return nil
}

type stubHistoryRetentionDB struct {
	execSQL  []string
	execArgs [][]any
}

func (s *stubHistoryRetentionDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return stubHistoryRetentionRow{value: 1}
}

func (s *stubHistoryRetentionDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execSQL = append(s.execSQL, sql)
	s.execArgs = append(s.execArgs, args)
	return pgconn.NewCommandTag("SELECT 1"), nil
}

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

func TestTimescaleHistoryRetentionApplierConstructedWithNilPoolReturnsError(t *testing.T) {
	applier := NewTimescaleHistoryRetentionApplier(nil)
	require.EqualError(t, applier.Apply(context.Background(), 20), "timescale pool is nil")
}
