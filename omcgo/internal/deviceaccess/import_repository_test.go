package deviceaccess

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPgImportStoreCreatesBatchAndRowsAtomically(t *testing.T) {
	tx := &repositoryTestTx{}
	store := NewPgImportStore(&repositoryTestDB{tx: tx})
	now := time.Now().UTC()
	batch := ImportBatch{ID: uuid.NewString(), Carrier: "cmcc", Type: ImportTypeAccessList, EntryType: ListEntryTypeDeny,
		Mode: ImportModeAppend, FailurePolicy: ImportFailureStrict, Status: ImportBatchValidated,
		SourceFilename: "lists.csv", ContentSHA256: contentSHA256([]byte("content")), TotalCount: 1, ValidCount: 1,
		CreatedBy: uuid.NewString(), CreatedAt: now, UpdatedAt: now}

	_, err := store.CreateImportBatch(context.Background(), batch, []ImportRow{{RowNumber: 2, ValidationStatus: ImportRowValid, CreatedAt: now}})

	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 2)
	require.Contains(t, tx.execSQL[0], "device_access_import_batches")
	require.Contains(t, tx.execSQL[1], "device_access_import_rows")
}

func TestPgImportStoreRollsBackWhenAnyRowCannotPersist(t *testing.T) {
	tx := &repositoryTestTx{failAt: 2}
	store := NewPgImportStore(&repositoryTestDB{tx: tx})
	now := time.Now().UTC()
	batch := ImportBatch{ID: uuid.NewString(), Carrier: "cmcc", Type: ImportTypeAccessList, EntryType: ListEntryTypeDeny,
		Mode: ImportModeAppend, FailurePolicy: ImportFailureStrict, Status: ImportBatchValidated,
		SourceFilename: "lists.csv", ContentSHA256: contentSHA256([]byte("content")), TotalCount: 1, ValidCount: 1,
		CreatedBy: uuid.NewString(), CreatedAt: now, UpdatedAt: now}

	_, err := store.CreateImportBatch(context.Background(), batch, []ImportRow{{RowNumber: 2, ValidationStatus: ImportRowValid, CreatedAt: now}})

	require.Error(t, err)
	require.False(t, tx.committed)
	require.True(t, tx.rolledBack)
}

func TestImportedListUpsertIsParameterizedAndReturnsTargetID(t *testing.T) {
	want := uuid.New()
	tx := &repositoryTestTx{queryRow: func(query string, args ...any) pgx.Row {
		require.Contains(t, query, "ON CONFLICT")
		require.Contains(t, query, "RETURNING id")
		require.NotContains(t, query, "SN-1")
		require.Contains(t, args, "SN-1")
		return repositoryTestRow{scan: func(dest ...any) error { *dest[0].(*uuid.UUID) = want; return nil }}
	}}

	got, err := upsertImportedListEntry(context.Background(), tx, "cmcc", uuid.New(), CompiledListEntry{
		Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-1", Status: ListEntryStatusActive,
	}, time.Now().UTC())

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestCompiledRuleDimensionExpandsRangeAlternativesWithoutChangingRuleANDSemantics(t *testing.T) {
	rule := CompiledRule{Conditions: []CompiledCondition{
		{Type: ConditionTypeObservedIP, Operator: ConditionOperatorIPRange, IPRanges: []IPRange{
			{Start: "10.0.0.1", End: "10.0.0.10"}, {Start: "192.0.2.1", End: "192.0.2.10"},
		}},
		{Type: ConditionTypeGPS, Operator: ConditionOperatorWithinBounds, GeoBoundsAny: []GeoBounds{
			{MinLatitude: 30, MaxLatitude: 31, MinLongitude: 120, MaxLongitude: 121},
			{MinLatitude: 39, MaxLatitude: 41, MinLongitude: 115, MaxLongitude: 117},
		}},
	}}

	require.Len(t, compiledRuleDimension(rule, ImportDimensionIP), 2)
	require.Len(t, compiledRuleDimension(rule, ImportDimensionGPS), 2)
}

func TestCreateReverseListBatchAuditsEveryAffectedIdentityAndHashesRestoreTarget(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	restoredID := uuid.New()
	removedID := uuid.New()
	upsertedID := uuid.New()
	tx := &repositoryTestTx{queryRow: func(query string, _ ...any) pgx.Row {
		require.Contains(t, query, "RETURNING id")
		return repositoryTestRow{scan: func(dest ...any) error {
			*dest[0].(*uuid.UUID) = upsertedID
			return nil
		}}
	}}
	original := ImportBatch{
		ID: uuid.NewString(), Carrier: "cmcc", Type: ImportTypeAccessList,
		EntryType: ListEntryTypeDeny,
	}
	before := []importListSnapshotEntry{{
		ID: restoredID, EntryType: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber,
		IdentityValue: "SN-RESTORE", Status: ListEntryStatusActive, ValidFrom: now,
	}}
	current := []importListSnapshotEntry{
		{ID: restoredID, EntryType: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-RESTORE", Status: ListEntryStatusDisabled, ValidFrom: now},
		{ID: removedID, EntryType: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-REMOVE", Status: ListEntryStatusActive, ValidFrom: now},
	}

	reverse, err := createReverseListBatch(
		context.Background(), tx, original, before, current,
		[]string{"SN-REMOVE", "SN-RESTORE"}, nil,
	)

	require.NoError(t, err)
	require.Equal(t, original.ID, reverse.ReversalOfBatchID)
	require.Equal(t, 2, reverse.TotalCount)
	require.Equal(t, 1, reverse.ValidCount)
	require.Equal(t, 2, reverse.ChangedCount)
	restoreTarget := []CompiledListEntry{{
		Type: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-RESTORE",
		Status: ListEntryStatusActive, ValidFrom: &now,
	}}
	targetJSON, marshalErr := json.Marshal(restoreTarget)
	require.NoError(t, marshalErr)
	require.Equal(t, contentSHA256(targetJSON), reverse.ContentSHA256)
	require.NotEqual(t, contentSHA256(reverse.Snapshot), reverse.ContentSHA256)
	require.NotEmpty(t, tx.execSQL)
}

func TestImportedGPSDimensionPreservesAllowMissingFromMultipleBounds(t *testing.T) {
	t.Parallel()
	require.True(t, importedGPSAllowMissing(CompiledCondition{GeoBoundsAny: []GeoBounds{
		{MinLatitude: 1, MaxLatitude: 2},
		{MinLatitude: 3, MaxLatitude: 4, AllowMissing: true},
	}}))
	require.True(t, importedGPSAllowMissing(CompiledCondition{GeoBounds: &GeoBounds{AllowMissing: true}}))
	require.False(t, importedGPSAllowMissing(CompiledCondition{GeoBoundsAny: []GeoBounds{{}}}))
}
