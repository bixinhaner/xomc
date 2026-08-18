package deviceaccess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type runtimeSettingsStub struct {
	enabled bool
	err     error
	last    RuntimeSettings
}

func (s *runtimeSettingsStub) GetRuntimeSettings(_ context.Context, carrier string) (RuntimeSettings, error) {
	return RuntimeSettings{Carrier: carrier, Enabled: s.enabled}, s.err
}

func (s *runtimeSettingsStub) UpdateRuntimeSettings(_ context.Context, carrier string, enabled bool, updatedBy string) (RuntimeSettings, error) {
	s.last = RuntimeSettings{Carrier: carrier, Enabled: enabled, UpdatedBy: updatedBy, UpdatedAt: time.Now().UTC()}
	return s.last, s.err
}

func TestPgRuntimeSettingsStoreDefaultsMissingOperatorToDisabled(t *testing.T) {
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(...any) error { return pgx.ErrNoRows }}}
	store := newPgRuntimeSettingsStoreWithDB(db)

	settings, err := store.GetRuntimeSettings(context.Background(), " CMCC ")

	require.NoError(t, err)
	require.Equal(t, RuntimeSettings{Carrier: "cmcc", Enabled: false}, settings)
	require.Contains(t, db.lastSQL, "device_access_runtime_settings")
	require.Equal(t, []any{"cmcc"}, db.lastArgs)
}

func TestPgRuntimeSettingsStoreUpsertsOnlyTheRequestedOperator(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	db := &repositoryTestDB{row: repositoryTestRow{scan: func(dest ...any) error {
		*dest[0].(*string) = "ctcc"
		*dest[1].(*bool) = true
		*dest[2].(*string) = "user-1"
		*dest[3].(*time.Time) = now
		return nil
	}}}
	store := newPgRuntimeSettingsStoreWithDB(db)

	settings, err := store.UpdateRuntimeSettings(context.Background(), "CTCC", true, "user-1")

	require.NoError(t, err)
	require.True(t, settings.Enabled)
	require.Equal(t, "ctcc", settings.Carrier)
	require.Contains(t, db.lastSQL, "ON CONFLICT (carrier) DO UPDATE")
	require.Equal(t, "ctcc", db.lastArgs[0])
	require.Equal(t, true, db.lastArgs[1])
	require.Equal(t, "user-1", db.lastArgs[2])
}

func TestRuntimeAccessEnabledPropagatesStoreFailure(t *testing.T) {
	want := errors.New("database unavailable")
	enabled, err := runtimeAccessEnabled(context.Background(), &runtimeSettingsStub{err: want}, "cmcc")
	require.False(t, enabled)
	require.ErrorIs(t, err, want)
}
