package deviceaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ImportBatchStore interface {
	CreateImportBatch(context.Context, ImportBatch, []ImportRow) (ImportBatch, error)
}

type ImportListSnapshotStore interface {
	ExistingListEntries(context.Context, PolicyActor, ListEntryType, []string, bool) ([]importListSnapshotEntry, error)
}

type ImportRuleDimensionStore interface {
	LoadRuleDimension(context.Context, PolicyActor, string, string, ImportDimension) ([]RuleDimensionValue, error)
}

type ImportBatchMutationStore interface {
	CommitImport(context.Context, PolicyActor, string) (ImportBatch, error)
	RollbackImport(context.Context, PolicyActor, string) (ImportBatch, error)
}

type ImportBatchReadStore interface {
	ListImportBatches(context.Context, ImportBatchListFilter) ([]ImportBatch, int64, error)
	GetImportBatch(context.Context, string, string) (ImportBatchDetail, error)
	ListImportErrors(context.Context, string, string) ([]ImportRow, error)
}

type ImportService struct {
	store      ImportBatchStore
	identities IdentityVisibilityChecker
}

func NewImportService(store ImportBatchStore) *ImportService {
	return &ImportService{store: store}
}

func (s *ImportService) SetIdentityVisibilityChecker(checker IdentityVisibilityChecker) {
	s.identities = checker
}

func (s *ImportService) PreviewAccessList(ctx context.Context, actor PolicyActor, request ImportPreviewRequest) (ImportPreviewResult, error) {
	if err := validatePolicyActor(actor); err != nil {
		return ImportPreviewResult{}, err
	}
	if s == nil || s.store == nil {
		return ImportPreviewResult{}, fmt.Errorf("preview access list import: %w", ErrAccessGateDependencyMissing)
	}
	if request.Type == "" {
		request.Type = ImportTypeAccessList
	}
	if request.Type != ImportTypeAccessList {
		return ImportPreviewResult{}, fmt.Errorf("%w: import type %q is not supported", ErrImportTemplateInvalid, request.Type)
	}
	if request.Mode == "" {
		request.Mode = ImportModeAppend
	}
	if request.Mode != ImportModeAppend && request.Mode != ImportModeReplace {
		return ImportPreviewResult{}, fmt.Errorf("%w: unsupported import mode %q", ErrImportTemplateInvalid, request.Mode)
	}
	if request.FailurePolicy == "" {
		request.FailurePolicy = ImportFailureStrict
	}
	if request.FailurePolicy != ImportFailureStrict && request.FailurePolicy != ImportFailureValidOnly {
		return ImportPreviewResult{}, fmt.Errorf("%w: unsupported failure policy %q", ErrImportTemplateInvalid, request.FailurePolicy)
	}
	if strings.TrimSpace(request.SourceFilename) == "" {
		return ImportPreviewResult{}, fmt.Errorf("%w: source filename is required", ErrImportTemplateInvalid)
	}

	preview, err := ParseAccessListFile(request.Content, request.SourceFilename, AccessListImportOptions{
		EntryType: request.EntryType, FailurePolicy: request.FailurePolicy,
	})
	if err != nil {
		return ImportPreviewResult{}, fmt.Errorf("parse access list import: %w", err)
	}
	for _, entry := range preview.Entries {
		if err := authorizeIdentityScope(ctx, s.identities, actor, entry.IdentityValue); err != nil {
			return ImportPreviewResult{}, err
		}
	}

	changedCount := len(preview.Entries)
	disableCount := 0
	if snapshots, ok := s.store.(ImportListSnapshotStore); ok {
		scope, snapshotErr := snapshots.ExistingListEntries(ctx, actor, request.EntryType, entrySerialNumbers(preview.Entries), request.Mode == ImportModeReplace)
		if snapshotErr != nil {
			return ImportPreviewResult{}, fmt.Errorf("load access list import impact: %w", snapshotErr)
		}
		request.scopeSHA256 = importListScopeSHA256(scope)
		existing := importListSnapshotMap(scope)
		changedCount = 0
		inFile := make(map[string]struct{}, len(preview.Entries))
		for _, entry := range preview.Entries {
			inFile[entry.IdentityValue] = struct{}{}
			current, exists := existing[entry.IdentityValue]
			if !exists || !sameImportedListEntry(current, entry) {
				changedCount++
				continue
			}
			markImportRowNoChange(preview.Rows, entry.IdentityValue)
		}
		if request.Mode == ImportModeReplace {
			for serialNumber, entry := range existing {
				if entry.Status != ListEntryStatusActive {
					continue
				}
				if _, exists := inFile[serialNumber]; !exists {
					disableCount++
				}
			}
		}
	}

	now := time.Now().UTC()
	batch := ImportBatch{
		ID: uuid.NewString(), Carrier: actor.Carrier, Type: request.Type, EntryType: request.EntryType,
		Mode: request.Mode, FailurePolicy: request.FailurePolicy, Status: ImportBatchValidated,
		SourceFilename: request.SourceFilename, ContentSHA256: contentSHA256(request.Content),
		ScopeSHA256: request.scopeSHA256,
		TotalCount:  preview.TotalCount, ValidCount: preview.ValidCount, InvalidCount: preview.InvalidCount,
		ChangedCount: changedCount, IdempotencyKey: strings.TrimSpace(request.IdempotencyKey),
		CreatedBy: actor.SubjectID, CreatedAt: now, UpdatedAt: now,
	}
	for index := range preview.Rows {
		preview.Rows[index].BatchID = batch.ID
		preview.Rows[index].CreatedAt = now
	}
	persisted, err := s.store.CreateImportBatch(ctx, batch, preview.Rows)
	if err != nil {
		return ImportPreviewResult{}, fmt.Errorf("persist access list import preview: %w", err)
	}
	return ImportPreviewResult{Batch: persisted, Rows: preview.Rows, Entries: preview.Entries, DisableCount: disableCount}, nil
}

func (s *ImportService) PreviewRuleDimension(ctx context.Context, actor PolicyActor, request ImportPreviewRequest) (RuleDimensionPreviewResult, error) {
	if err := validatePolicyActor(actor); err != nil {
		return RuleDimensionPreviewResult{}, err
	}
	store, ok := s.store.(ImportRuleDimensionStore)
	if !ok {
		return RuleDimensionPreviewResult{}, fmt.Errorf("preview rule dimension import: %w", ErrAccessGateDependencyMissing)
	}
	if request.Type == "" {
		request.Type = ImportTypeRuleDimension
	}
	if request.Type != ImportTypeRuleDimension {
		return RuleDimensionPreviewResult{}, fmt.Errorf("%w: import type %q is not supported", ErrImportTemplateInvalid, request.Type)
	}
	if request.Mode == "" {
		request.Mode = ImportModeAppend
	}
	if request.Mode != ImportModeAppend && request.Mode != ImportModeReplace {
		return RuleDimensionPreviewResult{}, fmt.Errorf("%w: unsupported import mode %q", ErrImportTemplateInvalid, request.Mode)
	}
	if request.FailurePolicy == "" {
		request.FailurePolicy = ImportFailureStrict
	}
	if request.FailurePolicy != ImportFailureStrict && request.FailurePolicy != ImportFailureValidOnly {
		return RuleDimensionPreviewResult{}, fmt.Errorf("%w: unsupported failure policy %q", ErrImportTemplateInvalid, request.FailurePolicy)
	}
	if strings.TrimSpace(request.TargetPolicyVersionID) == "" || strings.TrimSpace(request.TargetRuleID) == "" {
		return RuleDimensionPreviewResult{}, fmt.Errorf("%w: target draft and rule are required", ErrImportTemplateInvalid)
	}
	if strings.TrimSpace(request.SourceFilename) == "" {
		return RuleDimensionPreviewResult{}, fmt.Errorf("%w: source filename is required", ErrImportTemplateInvalid)
	}
	preview, err := ParseRuleDimensionFile(request.Content, request.SourceFilename, request.Dimension)
	if err != nil {
		return RuleDimensionPreviewResult{}, fmt.Errorf("parse rule dimension import: %w", err)
	}
	if request.Dimension == ImportDimensionSN {
		for _, value := range preview.Values {
			if err := authorizeIdentityScope(ctx, s.identities, actor, value.SerialNumber); err != nil {
				return RuleDimensionPreviewResult{}, err
			}
		}
	}
	existing, err := store.LoadRuleDimension(ctx, actor, request.TargetPolicyVersionID, request.TargetRuleID, request.Dimension)
	if err != nil {
		return RuleDimensionPreviewResult{}, fmt.Errorf("load rule dimension import impact: %w", err)
	}
	existingKeys := ruleDimensionKeySet(existing)
	fileKeys := ruleDimensionKeySet(preview.Values)
	changedCount := 0
	for _, value := range preview.Values {
		key := ruleDimensionValueKey(value)
		if _, exists := existingKeys[key]; exists {
			markDimensionRowNoChange(preview.Rows, key)
			continue
		}
		changedCount++
	}
	disableCount := 0
	if request.Mode == ImportModeReplace {
		for key := range existingKeys {
			if _, retained := fileKeys[key]; !retained {
				disableCount++
			}
		}
	}
	now := time.Now().UTC()
	batch := ImportBatch{
		ID: uuid.NewString(), Carrier: actor.Carrier, Type: ImportTypeRuleDimension,
		TargetPolicyVersionID: request.TargetPolicyVersionID, TargetRuleID: request.TargetRuleID, Dimension: request.Dimension,
		Mode: request.Mode, FailurePolicy: request.FailurePolicy, Status: ImportBatchValidated,
		SourceFilename: request.SourceFilename, ContentSHA256: contentSHA256(request.Content),
		ScopeSHA256: ruleDimensionScopeSHA256(existing),
		TotalCount:  preview.TotalCount, ValidCount: preview.ValidCount, InvalidCount: preview.InvalidCount,
		ChangedCount: changedCount, IdempotencyKey: strings.TrimSpace(request.IdempotencyKey),
		CreatedBy: actor.SubjectID, CreatedAt: now, UpdatedAt: now,
	}
	for index := range preview.Rows {
		preview.Rows[index].BatchID = batch.ID
		preview.Rows[index].CreatedAt = now
	}
	persisted, err := s.store.CreateImportBatch(ctx, batch, preview.Rows)
	if err != nil {
		return RuleDimensionPreviewResult{}, fmt.Errorf("persist rule dimension import preview: %w", err)
	}
	return RuleDimensionPreviewResult{Batch: persisted, Rows: preview.Rows, Values: preview.Values, DisableCount: disableCount}, nil
}

func (s *ImportService) ExportRuleDimension(ctx context.Context, actor PolicyActor, policyVersionID, ruleID string, dimension ImportDimension) ([]byte, error) {
	if err := validatePolicyActor(actor); err != nil {
		return nil, err
	}
	store, ok := s.store.(ImportRuleDimensionStore)
	if !ok {
		return nil, fmt.Errorf("export rule dimension: %w", ErrAccessGateDependencyMissing)
	}
	values, err := store.LoadRuleDimension(ctx, actor, policyVersionID, ruleID, dimension)
	if err != nil {
		return nil, fmt.Errorf("load rule dimension export: %w", err)
	}
	content, err := GenerateRuleDimensionCSV(dimension, values)
	if err != nil {
		return nil, fmt.Errorf("generate rule dimension export: %w", err)
	}
	return content, nil
}

func (s *ImportService) ClearRuleDimension(ctx context.Context, actor PolicyActor, policyVersionID, ruleID string, dimension ImportDimension) (ImportBatch, error) {
	template, err := GenerateRuleDimensionTemplate(dimension)
	if err != nil {
		return ImportBatch{}, err
	}
	preview, err := s.PreviewRuleDimension(ctx, actor, ImportPreviewRequest{
		Type: ImportTypeRuleDimension, Mode: ImportModeReplace, FailurePolicy: ImportFailureStrict,
		TargetPolicyVersionID: policyVersionID, TargetRuleID: ruleID, Dimension: dimension,
		SourceFilename: "clear-" + string(dimension) + ".csv", Content: template, IdempotencyKey: "clear-" + uuid.NewString(),
	})
	if err != nil {
		return ImportBatch{}, fmt.Errorf("preview rule dimension clear: %w", err)
	}
	batch, err := s.CommitImport(ctx, actor, preview.Batch.ID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("commit rule dimension clear: %w", err)
	}
	return batch, nil
}

func (s *ImportService) CommitImport(ctx context.Context, actor PolicyActor, batchID string) (ImportBatch, error) {
	return s.CommitImportConfirmed(ctx, actor, batchID, ImportCommitOptions{})
}

func (s *ImportService) CommitImportConfirmed(ctx context.Context, actor PolicyActor, batchID string, options ImportCommitOptions) (ImportBatch, error) {
	if err := validatePolicyActor(actor); err != nil {
		return ImportBatch{}, err
	}
	store, ok := s.importMutationStore()
	if !ok {
		return ImportBatch{}, fmt.Errorf("commit access list import: %w", ErrAccessGateDependencyMissing)
	}
	if err := s.authorizeImportMutation(ctx, actor, batchID); err != nil {
		return ImportBatch{}, err
	}
	reads, ok := s.importReadStore()
	if !ok {
		return ImportBatch{}, fmt.Errorf("confirm access list import: %w", ErrAccessGateDependencyMissing)
	}
	detail, err := reads.GetImportBatch(ctx, actor.Carrier, batchID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("load import batch confirmation state: %w", err)
	}
	if detail.Batch.Mode == ImportModeReplace &&
		detail.Batch.FailurePolicy == ImportFailureValidOnly &&
		detail.Batch.InvalidCount > 0 && !options.ConfirmReplaceWithInvalid {
		return ImportBatch{}, ErrImportReplaceConfirmationRequired
	}
	return store.CommitImport(ctx, actor, batchID)
}

func (s *ImportService) RollbackImport(ctx context.Context, actor PolicyActor, batchID string) (ImportBatch, error) {
	if err := validatePolicyActor(actor); err != nil {
		return ImportBatch{}, err
	}
	store, ok := s.importMutationStore()
	if !ok {
		return ImportBatch{}, fmt.Errorf("rollback access list import: %w", ErrAccessGateDependencyMissing)
	}
	if err := s.authorizeImportMutation(ctx, actor, batchID); err != nil {
		return ImportBatch{}, err
	}
	return store.RollbackImport(ctx, actor, batchID)
}

func (s *ImportService) authorizeImportMutation(ctx context.Context, actor PolicyActor, batchID string) error {
	if actor.VisibleGroups == nil {
		return nil
	}
	reads, ok := s.importReadStore()
	if !ok {
		return fmt.Errorf("authorize import batch mutation: %w", ErrAccessGateDependencyMissing)
	}
	detail, err := reads.GetImportBatch(ctx, actor.Carrier, batchID)
	if err != nil {
		return fmt.Errorf("load import batch for authorization: %w", err)
	}
	serialNumbers := make([]string, 0, len(detail.Rows))
	for _, row := range detail.Rows {
		if serialNumber, _ := row.NormalizedValue["serial_number"].(string); strings.TrimSpace(serialNumber) != "" {
			serialNumbers = appendUniqueString(serialNumbers, strings.TrimSpace(serialNumber))
		}
	}
	if detail.Batch.Type == ImportTypeAccessList && len(detail.Batch.Snapshot) > 0 {
		var snapshot []importListSnapshotEntry
		if err := json.Unmarshal(detail.Batch.Snapshot, &snapshot); err != nil {
			return fmt.Errorf("decode import batch authorization snapshot: %w", err)
		}
		for _, entry := range snapshot {
			serialNumbers = appendUniqueString(serialNumbers, entry.IdentityValue)
		}
	}
	for _, serialNumber := range serialNumbers {
		if err := authorizeIdentityScope(ctx, s.identities, actor, serialNumber); err != nil {
			return err
		}
	}
	return nil
}

func (s *ImportService) ListImportBatches(ctx context.Context, actor PolicyActor, filter ImportBatchListFilter) ([]ImportBatch, int64, error) {
	if err := validatePolicyActor(actor); err != nil {
		return nil, 0, err
	}
	store, ok := s.importReadStore()
	if !ok {
		return nil, 0, fmt.Errorf("list access list imports: %w", ErrAccessGateDependencyMissing)
	}
	filter.Carrier = actor.Carrier
	return store.ListImportBatches(ctx, filter)
}

func (s *ImportService) GetImportBatch(ctx context.Context, actor PolicyActor, batchID string) (ImportBatchDetail, error) {
	if err := validatePolicyActor(actor); err != nil {
		return ImportBatchDetail{}, err
	}
	store, ok := s.importReadStore()
	if !ok {
		return ImportBatchDetail{}, fmt.Errorf("get access list import: %w", ErrAccessGateDependencyMissing)
	}
	return store.GetImportBatch(ctx, actor.Carrier, batchID)
}

func (s *ImportService) ListImportErrors(ctx context.Context, actor PolicyActor, batchID string) ([]ImportRow, error) {
	if err := validatePolicyActor(actor); err != nil {
		return nil, err
	}
	store, ok := s.importReadStore()
	if !ok {
		return nil, fmt.Errorf("list access list import errors: %w", ErrAccessGateDependencyMissing)
	}
	return store.ListImportErrors(ctx, actor.Carrier, batchID)
}

func (s *ImportService) importMutationStore() (ImportBatchMutationStore, bool) {
	if s == nil || s.store == nil {
		return nil, false
	}
	store, ok := s.store.(ImportBatchMutationStore)
	return store, ok
}

func (s *ImportService) importReadStore() (ImportBatchReadStore, bool) {
	if s == nil || s.store == nil {
		return nil, false
	}
	store, ok := s.store.(ImportBatchReadStore)
	return store, ok
}

func entrySerialNumbers(entries []CompiledListEntry) []string {
	serialNumbers := make([]string, 0, len(entries))
	for _, entry := range entries {
		serialNumbers = append(serialNumbers, entry.IdentityValue)
	}
	return serialNumbers
}

func contentSHA256(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func markImportRowNoChange(rows []ImportRow, serialNumber string) {
	for index := range rows {
		if rows[index].ValidationStatus == ImportRowValid && rows[index].NormalizedValue["serial_number"] == serialNumber {
			rows[index].ValidationStatus = ImportRowNoChange
			return
		}
	}
}

func sameImportedListEntry(left, right CompiledListEntry) bool {
	return left.Status == ListEntryStatusActive && strings.TrimSpace(left.Reason) == strings.TrimSpace(right.Reason) &&
		(right.ValidFrom == nil || sameImportTime(left.ValidFrom, right.ValidFrom)) && sameImportTime(left.ValidUntil, right.ValidUntil)
}

func sameImportTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func importListSnapshotMap(entries []importListSnapshotEntry) map[string]CompiledListEntry {
	result := make(map[string]CompiledListEntry, len(entries))
	for _, entry := range entries {
		validFrom := entry.ValidFrom
		result[entry.IdentityValue] = CompiledListEntry{
			ID: entry.ID.String(), Type: entry.EntryType, IdentityType: entry.IdentityType,
			IdentityValue: entry.IdentityValue, Reason: entry.Reason, ValidFrom: &validFrom,
			ValidUntil: entry.ValidUntil, Status: entry.Status,
		}
	}
	return result
}

func importListScopeSHA256(entries []importListSnapshotEntry) string {
	ordered := append([]importListSnapshotEntry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].IdentityValue == ordered[j].IdentityValue {
			return ordered[i].ID.String() < ordered[j].ID.String()
		}
		return ordered[i].IdentityValue < ordered[j].IdentityValue
	})
	payload, _ := json.Marshal(ordered)
	return contentSHA256(payload)
}

func ruleDimensionKeySet(values []RuleDimensionValue) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[ruleDimensionValueKey(value)] = struct{}{}
	}
	return result
}

func ruleDimensionScopeSHA256(values []RuleDimensionValue) string {
	keys := make([]string, 0, len(values))
	for _, value := range values {
		keys = append(keys, ruleDimensionValueKey(value))
	}
	sort.Strings(keys)
	payload, _ := json.Marshal(keys)
	return contentSHA256(payload)
}

func ruleDimensionValueKey(value RuleDimensionValue) string {
	switch value.Dimension {
	case ImportDimensionSN:
		return value.SerialNumber
	case ImportDimensionTAC, ImportDimensionECGI:
		return value.Value
	case ImportDimensionIP:
		if value.IPRange != nil {
			return value.IPRange.Start + "\x00" + value.IPRange.End
		}
	case ImportDimensionGPS:
		if value.GeoBounds != nil {
			return fmt.Sprintf("%g~%g;%g~%g", value.GeoBounds.MinLongitude, value.GeoBounds.MaxLongitude, value.GeoBounds.MinLatitude, value.GeoBounds.MaxLatitude)
		}
	}
	return ""
}

func markDimensionRowNoChange(rows []ImportRow, key string) {
	for index := range rows {
		if rows[index].ValidationStatus != ImportRowValid {
			continue
		}
		dimension, _ := rows[index].NormalizedValue["dimension"].(string)
		value, err := ruleDimensionValueFromRow(rows[index], ImportDimension(dimension))
		if err == nil && ruleDimensionValueKey(value) == key {
			rows[index].ValidationStatus = ImportRowNoChange
			return
		}
	}
}
