package deviceaccess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type importStoreStub struct {
	batch     ImportBatch
	rows      []ImportRow
	existing  []importListSnapshotEntry
	dimension []RuleDimensionValue
	createErr error
}

func (s *importStoreStub) LoadRuleDimension(context.Context, PolicyActor, string, string, ImportDimension) ([]RuleDimensionValue, error) {
	return append([]RuleDimensionValue(nil), s.dimension...), nil
}

func (s *importStoreStub) CreateImportBatch(_ context.Context, batch ImportBatch, rows []ImportRow) (ImportBatch, error) {
	if s.createErr != nil {
		return ImportBatch{}, s.createErr
	}
	s.batch = batch
	s.rows = append([]ImportRow(nil), rows...)
	return batch, nil
}

func (s *importStoreStub) ExistingListEntries(context.Context, PolicyActor, ListEntryType, []string, bool) ([]importListSnapshotEntry, error) {
	return s.existing, nil
}

func (s *importStoreStub) CommitImport(_ context.Context, _ PolicyActor, batchID string) (ImportBatch, error) {
	return ImportBatch{ID: batchID, Status: ImportBatchCommitted}, nil
}

func (s *importStoreStub) RollbackImport(_ context.Context, _ PolicyActor, batchID string) (ImportBatch, error) {
	return ImportBatch{ID: batchID, Status: ImportBatchRolledBack}, nil
}

func (s *importStoreStub) ListImportBatches(_ context.Context, filter ImportBatchListFilter) ([]ImportBatch, int64, error) {
	return []ImportBatch{{ID: "batch-1", Carrier: filter.Carrier}}, 1, nil
}

func (s *importStoreStub) GetImportBatch(_ context.Context, carrier, batchID string) (ImportBatchDetail, error) {
	batch := s.batch
	batch.ID = batchID
	batch.Carrier = carrier
	return ImportBatchDetail{Batch: batch, Rows: s.rows}, nil
}

func (s *importStoreStub) ListImportErrors(context.Context, string, string) ([]ImportRow, error) {
	return s.rows, nil
}

func TestImportServicePreviewPersistsScopedBatchAndImpact(t *testing.T) {
	now := time.Now().UTC()
	store := &importStoreStub{existing: []importListSnapshotEntry{
		{ID: uuid.New(), EntryType: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-EXISTING", ValidFrom: now, Status: ListEntryStatusActive},
		{ID: uuid.New(), EntryType: ListEntryTypeDeny, IdentityType: IdentityTypeSerialNumber, IdentityValue: "SN-REMOVED", ValidFrom: now, Status: ListEntryStatusActive},
	}}
	service := NewImportService(store)
	content := []byte("Serial Number,List Type,Reason,Valid From,Valid Until\nSN-NEW,deny,,,\nSN-EXISTING,deny,,,\nSN-NEW,deny,,,\n")

	result, err := service.PreviewAccessList(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}, ImportPreviewRequest{
		EntryType: ListEntryTypeDeny, SourceFilename: "lists.csv", Content: content,
		Mode: ImportModeReplace, FailurePolicy: ImportFailureStrict, IdempotencyKey: "preview-1",
	})

	require.NoError(t, err)
	require.NotEmpty(t, result.Batch.ID)
	require.Equal(t, ImportBatchValidated, result.Batch.Status)
	require.Equal(t, 3, result.Batch.TotalCount)
	require.Equal(t, 2, result.Batch.ValidCount)
	require.Equal(t, 1, result.Batch.InvalidCount)
	require.Equal(t, 1, result.Batch.ChangedCount)
	require.Equal(t, 1, result.DisableCount)
	require.Equal(t, ImportRowNoChange, result.Rows[1].ValidationStatus)
	require.Equal(t, contentSHA256(content), result.Batch.ContentSHA256)
	require.Equal(t, importListScopeSHA256(store.existing), result.Batch.ScopeSHA256)
}

func TestRuleDimensionPreviewPinsCurrentScopeHash(t *testing.T) {
	store := &importStoreStub{dimension: []RuleDimensionValue{{Dimension: ImportDimensionTAC, Value: "100"}}}
	result, err := NewImportService(store).PreviewRuleDimension(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: uuid.NewString(),
	}, ImportPreviewRequest{
		Type: ImportTypeRuleDimension, TargetPolicyVersionID: uuid.NewString(), TargetRuleID: uuid.NewString(),
		Dimension: ImportDimensionTAC, Mode: ImportModeReplace, FailurePolicy: ImportFailureStrict,
		SourceFilename: "tac.csv", Content: []byte("TAC\n200\n"),
	})

	require.NoError(t, err)
	require.Equal(t, ruleDimensionScopeSHA256(store.dimension), result.Batch.ScopeSHA256)
}

func TestImportServiceReauthorizesEveryIdentityBeforeCommit(t *testing.T) {
	store := &importStoreStub{
		batch: ImportBatch{Type: ImportTypeAccessList},
		rows:  []ImportRow{{NormalizedValue: map[string]any{"serial_number": "SN-OTHER-GROUP"}}},
	}
	checker := &identityVisibilityStub{}
	service := NewImportService(store)
	service.SetIdentityVisibilityChecker(checker)
	groupID := uuid.New()

	_, err := service.CommitImport(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: uuid.NewString(), VisibleGroups: []uuid.UUID{groupID},
	}, uuid.NewString())

	require.Error(t, err)
	require.Equal(t, "SN-OTHER-GROUP", checker.serialNumber)
	require.Equal(t, []uuid.UUID{groupID}, checker.groups)
}

func TestImportServiceRequiresExplicitConfirmationForRiskyReplace(t *testing.T) {
	store := &importStoreStub{batch: ImportBatch{
		Type: ImportTypeAccessList, Mode: ImportModeReplace,
		FailurePolicy: ImportFailureValidOnly, InvalidCount: 2,
	}}
	service := NewImportService(store)
	actor := PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}

	_, err := service.CommitImport(context.Background(), actor, "batch-1")
	require.ErrorIs(t, err, ErrImportReplaceConfirmationRequired)

	result, err := service.CommitImportConfirmed(context.Background(), actor, "batch-1", ImportCommitOptions{
		ConfirmReplaceWithInvalid: true,
	})
	require.NoError(t, err)
	require.Equal(t, ImportBatchCommitted, result.Status)
}

func TestImportServiceRejectsUnsupportedTypeBeforePersistence(t *testing.T) {
	store := &importStoreStub{}
	_, err := NewImportService(store).PreviewAccessList(context.Background(), PolicyActor{
		Carrier: "cmcc", SubjectID: "operator-1", VisibleGroups: []uuid.UUID{uuid.New()},
	}, ImportPreviewRequest{Type: ImportTypeRuleDimension, EntryType: ListEntryTypeDeny, SourceFilename: "rules.csv"})

	require.ErrorIs(t, err, ErrImportTemplateInvalid)
	require.Empty(t, store.batch.ID)
}

func TestImportServiceReturnsPersistenceFailure(t *testing.T) {
	store := &importStoreStub{createErr: errors.New("database unavailable")}
	_, err := NewImportService(store).PreviewAccessList(context.Background(), PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}, ImportPreviewRequest{
		EntryType: ListEntryTypeDeny, SourceFilename: "lists.csv",
		Content: []byte("Serial Number,List Type,Reason,Valid From,Valid Until\nSN-1,deny,,,\n"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "persist access list import preview")
}
