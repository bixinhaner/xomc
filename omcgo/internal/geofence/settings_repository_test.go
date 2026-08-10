package geofence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSettingsReadQueriesUseExactKeysAndStableOrder(t *testing.T) {
	systemSQL, systemArgs, err := buildGetSystemModeQuery()
	require.NoError(t, err)
	assert.Contains(t, systemSQL, "FROM sys_configs")
	assert.Contains(t, systemSQL, "category =")
	assert.Contains(t, systemSQL, "key =")
	assert.Contains(t, systemArgs, "geofence")
	assert.Contains(t, systemArgs, "mode")

	carriersSQL, _, err := buildListCarrierSettingsQuery()
	require.NoError(t, err)
	assert.Contains(t, carriersSQL, "FROM geofence_carrier_settings")
	assert.Contains(t, carriersSQL, "ORDER BY carrier")
}

func TestBuildSettingsUpdateQueriesLockThenWriteBothLevels(t *testing.T) {
	now := time.Date(2026, 7, 30, 13, 0, 0, 0, time.UTC)
	actorID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	systemLockSQL, _, err := buildLockSystemModeQuery()
	require.NoError(t, err)
	assert.Contains(t, systemLockSQL, "FROM sys_configs")
	assert.Contains(t, systemLockSQL, "FOR UPDATE")

	carrierLockSQL, carrierLockArgs, err := buildLockCarrierSettingsQuery(
		[]string{"ctcc", "cmcc"},
	)
	require.NoError(t, err)
	assert.Contains(t, carrierLockSQL, "FROM geofence_carrier_settings")
	assert.Contains(t, carrierLockSQL, "ORDER BY carrier")
	assert.Contains(t, carrierLockSQL, "FOR UPDATE")
	assert.Contains(t, carrierLockArgs, "ctcc")
	assert.Contains(t, carrierLockArgs, "cmcc")

	systemUpdateSQL, systemUpdateArgs, err := buildUpsertSystemModeQuery(
		RuntimeModeObserve,
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, systemUpdateSQL, "INSERT INTO sys_configs")
	assert.Contains(t, systemUpdateSQL, "ON CONFLICT (category,key) DO UPDATE")
	assert.Contains(t, systemUpdateArgs, RuntimeModeObserve)

	carrierUpdateSQL, carrierUpdateArgs, err := buildUpsertCarrierSettingQuery(
		CarrierSetting{
			Carrier: "cmcc", Mode: RuntimeModeObserve,
			DefaultBaselineRadiusMeters: 120,
		},
		actorID,
		now,
	)
	require.NoError(t, err)
	assert.Contains(t, carrierUpdateSQL, "INSERT INTO geofence_carrier_settings")
	assert.Contains(t, carrierUpdateSQL, "ON CONFLICT (carrier) DO UPDATE")
	assert.Contains(t, carrierUpdateArgs, actorID)
	assert.Contains(t, carrierUpdateArgs, float64(120))
}

func TestBuildSettingsPreviewQueriesCountOnlyEnabledAndActiveRows(t *testing.T) {
	queries, err := buildSettingsPreviewQueries(
		Settings{
			SystemMode: RuntimeModeOff,
			Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: RuntimeModeOff,
				DefaultBaselineRadiusMeters: 100,
			}},
		},
		Settings{SystemMode: RuntimeModeObserve, Carriers: []CarrierSetting{
			{Carrier: "cmcc", Mode: RuntimeModeObserve, DefaultBaselineRadiusMeters: 100},
		}},
	)
	require.NoError(t, err)
	require.Len(t, queries, 3)
	assert.Contains(t, queries[0].sql, "geofence_definitions")
	assert.Contains(t, queries[0].sql, "status =")
	assert.Contains(t, queries[1].sql, "device_geofence_bindings")
	assert.Contains(t, queries[1].sql, "status =")
	assert.Contains(t, queries[2].sql, "COUNT(DISTINCT b.device_id)")
	assert.Contains(t, queries[2].sql, "geofence_definitions")
}

type settingsTxStub struct {
	pgx.Tx
	failExecAt    int
	execErr       error
	execCalls     int
	commitCalls   int
	rollbackCalls int
}

func (tx *settingsTxStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	tx.execCalls++
	if tx.execCalls == tx.failExecAt {
		return pgconn.CommandTag{}, tx.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *settingsTxStub) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *settingsTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	return nil
}

type settingsDBStub struct {
	tx *settingsTxStub
}

func (db *settingsDBStub) Begin(context.Context) (pgx.Tx, error) {
	return db.tx, nil
}

func (*settingsDBStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (*settingsDBStub) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

func (*settingsDBStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

func TestPgSettingsRepositoryRollsBackSystemModeWhenCarrierWriteFails(t *testing.T) {
	carrierErr := errors.New("carrier update failed")
	tx := &settingsTxStub{failExecAt: 4, execErr: carrierErr}
	repository := NewPgSettingsRepository(&settingsDBStub{tx: tx})

	_, err := repository.UpdateSettings(
		context.Background(),
		Settings{
			SystemMode: RuntimeModeObserve,
			Carriers: []CarrierSetting{{
				Carrier: "cmcc", Mode: RuntimeModeObserve,
				DefaultBaselineRadiusMeters: 100,
			}},
		},
		uuid.New(),
		time.Now().UTC(),
	)

	require.ErrorIs(t, err, carrierErr)
	assert.Zero(t, tx.commitCalls)
	assert.Equal(t, 1, tx.rollbackCalls)
}
