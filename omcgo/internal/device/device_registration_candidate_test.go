package device

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type candidateRegistrationRow struct {
	scan func(...any) error
}

func (r candidateRegistrationRow) Scan(dest ...any) error { return r.scan(dest...) }

type candidateRegistrationTx struct {
	pgx.Tx
	rows    []pgx.Row
	execSQL []string
}

func (tx *candidateRegistrationTx) QueryRow(context.Context, string, ...any) pgx.Row {
	row := tx.rows[0]
	tx.rows = tx.rows[1:]
	return row
}

func (tx *candidateRegistrationTx) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	tx.execSQL = append(tx.execSQL, query)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func candidateRegistration(groupID uuid.UUID) *DeviceRegistration {
	return &DeviceRegistration{
		SerialNumber: "SN-CANDIDATE",
		GroupID:      &groupID,
		Carrier:      model.CarrierCMCC,
		Remark:       "approved",
		CreatedBy:    uuid.NewString(),
	}
}

func boolRow(value bool) pgx.Row {
	return candidateRegistrationRow{scan: func(dest ...any) error {
		*dest[0].(*bool) = value
		return nil
	}}
}

func TestEnsureCandidateOwnershipTxCreatesPendingRegistration(t *testing.T) {
	tx := &candidateRegistrationTx{rows: []pgx.Row{
		boolRow(false),
		boolRow(false),
		candidateRegistrationRow{scan: func(...any) error { return pgx.ErrNoRows }},
	}}

	err := (&PgRegistrationRepository{}).EnsureCandidateOwnershipTx(
		context.Background(), tx, candidateRegistration(uuid.New()),
	)

	require.NoError(t, err)
	require.Len(t, tx.execSQL, 1)
	assert.Contains(t, tx.execSQL[0], "INSERT INTO device_registrations")
	assert.NotContains(t, tx.execSQL[0], "SN-CANDIDATE")
}

func TestEnsureCandidateOwnershipTxRejectsCrossCarrierIdentity(t *testing.T) {
	tx := &candidateRegistrationTx{rows: []pgx.Row{boolRow(true)}}

	err := (&PgRegistrationRepository{}).EnsureCandidateOwnershipTx(
		context.Background(), tx, candidateRegistration(uuid.New()),
	)

	require.ErrorContains(t, err, "another carrier")
	require.ErrorIs(t, err, ErrRegistrationOwnershipConflict)
	assert.Empty(t, tx.execSQL)
}

func TestEnsureCandidateOwnershipTxRenewsExpiredRegistrationInPlace(t *testing.T) {
	existingID := uuid.New()
	tx := &candidateRegistrationTx{rows: []pgx.Row{
		boolRow(false),
		boolRow(false),
		candidateRegistrationRow{scan: func(dest ...any) error {
			*dest[0].(*uuid.UUID) = existingID
			*dest[1].(*string) = string(global.RegistrationExpired)
			return nil
		}},
	}}

	err := (&PgRegistrationRepository{}).EnsureCandidateOwnershipTx(
		context.Background(), tx, candidateRegistration(uuid.New()),
	)

	require.NoError(t, err)
	require.Len(t, tx.execSQL, 1)
	assert.True(t, strings.HasPrefix(tx.execSQL[0], "UPDATE device_registrations"))
}
