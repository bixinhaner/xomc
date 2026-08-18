package deviceaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementProductNameProjectionsMatchDeviceListSemantics(t *testing.T) {
	assert.Equal(t, "COALESCE(NULLIF(d.model_name, ''), '')", accessStateProductNameSQL)
	assert.Equal(t, "COALESCE(NULLIF(d.model_name, ''), '')", accessListProductNameSQL)
	assert.Contains(t, actionColumns, "COALESCE(NULLIF(d.model_name, ''), '')")
	assert.NotContains(t, actionColumns, "device_name")
}

type candidateOwnershipRegistrarStub struct {
	called  bool
	request CandidateOwnershipRegistration
	err     error
}

func (s *candidateOwnershipRegistrarStub) EnsureCandidateOwnership(
	_ context.Context,
	_ pgx.Tx,
	request CandidateOwnershipRegistration,
) error {
	s.called = true
	s.request = request
	return s.err
}

func candidateReviewStore(t *testing.T, deviceID *uuid.UUID, registrar CandidateOwnershipRegistrar) (*PgManagementStore, *repositoryTestTx) {
	t.Helper()
	tx := &repositoryTestTx{queryRow: func(_ string, _ ...any) pgx.Row {
		return repositoryTestRow{scan: func(dest ...any) error {
			require.Len(t, dest, 2)
			*dest[0].(*string) = "SN-CANDIDATE"
			*dest[1].(**uuid.UUID) = deviceID
			return nil
		}}
	}}
	return &PgManagementStore{
		db:                 &repositoryTestDB{tx: tx},
		ownershipRegistrar: registrar,
	}, tx
}

func TestReviewCandidateAllowRegistersOwnershipBeforeApproval(t *testing.T) {
	registrar := &candidateOwnershipRegistrarStub{}
	store, tx := candidateReviewStore(t, nil, registrar)
	reviewerID := uuid.New()

	err := store.ReviewCandidate(
		context.Background(), "cmcc", uuid.New(), reviewerID, nil, ListEntryTypeAllow, "approved",
	)

	require.NoError(t, err)
	require.True(t, registrar.called)
	assert.Equal(t, CandidateOwnershipRegistration{
		Carrier: "cmcc", SerialNumber: "SN-CANDIDATE", CreatedBy: reviewerID.String(),
	}, registrar.request)
	assert.Len(t, tx.execSQL, 3)
	assert.True(t, tx.committed)
}

func TestReviewCandidateAllowRollsBackWhenOwnershipRegistrationFails(t *testing.T) {
	registrar := &candidateOwnershipRegistrarStub{err: errors.New("registration unavailable")}
	store, tx := candidateReviewStore(t, nil, registrar)

	err := store.ReviewCandidate(
		context.Background(), "cmcc", uuid.New(), uuid.New(), nil, ListEntryTypeAllow, "approved",
	)

	require.ErrorContains(t, err, "register approved candidate ownership")
	assert.Empty(t, tx.execSQL)
	assert.False(t, tx.committed)
	assert.True(t, tx.rolledBack)
}

func TestReviewCandidateDoesNotCreateRegistrationForFormalDeviceOrDenial(t *testing.T) {
	t.Run("formal device", func(t *testing.T) {
		registrar := &candidateOwnershipRegistrarStub{}
		deviceID := uuid.New()
		store, tx := candidateReviewStore(t, &deviceID, registrar)

		require.NoError(t, store.ReviewCandidate(
			context.Background(), "cmcc", uuid.New(), uuid.New(), nil, ListEntryTypeAllow, "approved",
		))
		assert.False(t, registrar.called)
		assert.True(t, tx.committed)
	})

	t.Run("denial", func(t *testing.T) {
		registrar := &candidateOwnershipRegistrarStub{}
		store, tx := candidateReviewStore(t, nil, registrar)

		require.NoError(t, store.ReviewCandidate(
			context.Background(), "cmcc", uuid.New(), uuid.New(), nil, ListEntryTypeDeny, "rejected",
		))
		assert.False(t, registrar.called)
		assert.True(t, tx.committed)
	})
}
