package alarm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

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
