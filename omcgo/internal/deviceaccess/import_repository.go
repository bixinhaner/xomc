package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgImportStore struct {
	db storage.DB
}

func NewPgImportStore(db storage.DB) *PgImportStore {
	return &PgImportStore{db: db}
}

func (s *PgImportStore) LoadRuleDimension(ctx context.Context, actor PolicyActor, policyVersionID, ruleID string, dimension ImportDimension) ([]RuleDimensionValue, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("load rule dimension: %w", ErrAccessGateDependencyMissing)
	}
	version, err := (&PgPolicyStore{db: s.db}).GetVersion(ctx, policyVersionID)
	if err != nil {
		return nil, fmt.Errorf("load rule dimension policy: %w", err)
	}
	if version.Carrier != actor.Carrier {
		return nil, ErrPolicyCarrierScope
	}
	if version.Status != PolicyVersionDraft {
		return nil, ErrPolicyVersionImmutable
	}
	for _, rule := range version.Policy.Rules {
		if rule.ID == ruleID {
			return compiledRuleDimension(rule, dimension), nil
		}
	}
	return nil, pgx.ErrNoRows
}

func compiledRuleDimension(rule CompiledRule, dimension ImportDimension) []RuleDimensionValue {
	result := make([]RuleDimensionValue, 0)
	if dimension == ImportDimensionSN {
		if rule.SerialScope.Type == SerialScopeList {
			for _, serialNumber := range rule.SerialScope.Values {
				result = append(result, RuleDimensionValue{Dimension: dimension, SerialNumber: serialNumber})
			}
		}
		return result
	}
	conditionType := conditionTypeForDimension(dimension)
	for _, condition := range rule.Conditions {
		if condition.Type != conditionType {
			continue
		}
		switch dimension {
		case ImportDimensionTAC, ImportDimensionECGI:
			switch condition.Operator {
			case ConditionOperatorEqual:
				result = append(result, RuleDimensionValue{Dimension: dimension, Value: condition.Expected})
			case ConditionOperatorIn:
				for _, value := range condition.ExpectedAny {
					result = append(result, RuleDimensionValue{Dimension: dimension, Value: value})
				}
			}
		case ImportDimensionIP:
			for _, value := range condition.IPRanges {
				rangeCopy := value
				result = append(result, RuleDimensionValue{Dimension: dimension, IPRange: &rangeCopy})
			}
			if condition.Operator == ConditionOperatorIPRange && condition.IPRange != nil {
				rangeCopy := *condition.IPRange
				result = append(result, RuleDimensionValue{Dimension: dimension, IPRange: &rangeCopy})
			}
		case ImportDimensionGPS:
			for _, value := range condition.GeoBoundsAny {
				boundsCopy := value
				result = append(result, RuleDimensionValue{Dimension: dimension, GeoBounds: &boundsCopy})
			}
			if condition.Operator == ConditionOperatorWithinBounds && condition.GeoBounds != nil {
				boundsCopy := *condition.GeoBounds
				result = append(result, RuleDimensionValue{Dimension: dimension, GeoBounds: &boundsCopy})
			}
		}
	}
	return result
}

func conditionTypeForDimension(dimension ImportDimension) ConditionType {
	switch dimension {
	case ImportDimensionTAC:
		return ConditionTypeTAC
	case ImportDimensionECGI:
		return ConditionTypeECGI
	case ImportDimensionIP:
		return ConditionTypeObservedIP
	case ImportDimensionGPS:
		return ConditionTypeGPS
	default:
		return ""
	}
}

func (s *PgImportStore) ExistingListEntries(ctx context.Context, actor PolicyActor, entryType ListEntryType, serialNumbers []string, includeAll bool) ([]importListSnapshotEntry, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("load import impact: %w", ErrAccessGateDependencyMissing)
	}
	if !includeAll && len(serialNumbers) == 0 {
		return []importListSnapshotEntry{}, nil
	}
	return loadImportListSnapshot(ctx, s.db, actor, entryType, serialNumbers, includeAll)
}

func (s *PgImportStore) CreateImportBatch(ctx context.Context, batch ImportBatch, rows []ImportRow) (ImportBatch, error) {
	if s == nil || s.db == nil {
		return ImportBatch{}, fmt.Errorf("create import batch: %w", ErrAccessGateDependencyMissing)
	}
	batchID, err := uuid.Parse(batch.ID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import batch id: %w", err)
	}
	createdBy, err := nullableUUID(batch.CreatedBy)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import batch creator: %w", err)
	}
	targetPolicyVersionID, err := nullableUUID(batch.TargetPolicyVersionID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import target policy version: %w", err)
	}
	targetRuleID, err := nullableUUID(batch.TargetRuleID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import target rule: %w", err)
	}
	reversalOfBatchID, err := nullableUUID(batch.ReversalOfBatchID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse reversed import batch: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("begin create import batch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	query, args, err := storage.Psql.Insert("device_access_import_batches").
		Columns("id", "carrier", "import_type", "entry_type", "target_policy_version_id", "target_rule_id", "dimension", "mode", "failure_policy", "status", "source_filename", "content_sha256", "scope_sha256", "total_count", "valid_count", "invalid_count", "changed_count", "snapshot", "idempotency_key", "created_by", "reversal_of_batch_id", "created_at", "updated_at").
		Values(batchID, batch.Carrier, batch.Type, nullableString(string(batch.EntryType)), targetPolicyVersionID, targetRuleID, nullableString(string(batch.Dimension)), batch.Mode, batch.FailurePolicy, batch.Status,
			batch.SourceFilename, batch.ContentSHA256, nullableString(batch.ScopeSHA256), batch.TotalCount, batch.ValidCount, batch.InvalidCount,
			batch.ChangedCount, sq.Expr("'{}'::jsonb"), nullableString(batch.IdempotencyKey), createdBy, reversalOfBatchID, batch.CreatedAt, batch.UpdatedAt).
		Suffix("ON CONFLICT (carrier, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING").ToSql()
	if err != nil {
		return ImportBatch{}, fmt.Errorf("build import batch insert: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("insert import batch: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ImportBatch{}, ErrImportBatchConflict
	}
	for _, row := range rows {
		rawValue, err := json.Marshal(row.RawValue)
		if err != nil {
			return ImportBatch{}, fmt.Errorf("encode import row %d: %w", row.RowNumber, err)
		}
		normalizedValue, err := json.Marshal(row.NormalizedValue)
		if err != nil {
			return ImportBatch{}, fmt.Errorf("encode normalized import row %d: %w", row.RowNumber, err)
		}
		query, args, err := storage.Psql.Insert("device_access_import_rows").
			Columns("id", "batch_id", "row_number", "raw_value", "normalized_value", "validation_status", "error_code", "error_message", "created_at").
			Values(uuid.New(), batchID, row.RowNumber, rawValue, normalizedValue, row.ValidationStatus,
				nullableString(row.ErrorCode), nullableString(row.ErrorMessage), row.CreatedAt).ToSql()
		if err != nil {
			return ImportBatch{}, fmt.Errorf("build import row insert: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return ImportBatch{}, fmt.Errorf("insert import row %d: %w", row.RowNumber, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, fmt.Errorf("commit import preview: %w", err)
	}
	return batch, nil
}

func (s *PgImportStore) ListImportBatches(ctx context.Context, filter ImportBatchListFilter) ([]ImportBatch, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("list import batches: %w", ErrAccessGateDependencyMissing)
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := sq.Eq{"carrier": filter.Carrier}
	if filter.Type != "" {
		where["import_type"] = filter.Type
	}
	if filter.TargetPolicyVersionID != "" {
		where["target_policy_version_id"] = filter.TargetPolicyVersionID
	}
	if filter.TargetRuleID != "" {
		where["target_rule_id"] = filter.TargetRuleID
	}
	if filter.Dimension != "" {
		where["dimension"] = filter.Dimension
	}
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").From("device_access_import_batches").Where(where).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build import batch count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import batches: %w", err)
	}
	query, args, err := importBatchSelect().Where(where).OrderBy("created_at DESC", "id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build import batch list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query import batches: %w", err)
	}
	defer rows.Close()
	items := make([]ImportBatch, 0, pageSize)
	for rows.Next() {
		batch, err := scanImportBatch(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan import batch: %w", err)
		}
		items = append(items, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate import batches: %w", err)
	}
	return items, total, nil
}

func (s *PgImportStore) GetImportBatch(ctx context.Context, carrier, batchID string) (ImportBatchDetail, error) {
	id, err := uuid.Parse(strings.TrimSpace(batchID))
	if err != nil {
		return ImportBatchDetail{}, fmt.Errorf("parse import batch id: %w", err)
	}
	batch, err := loadImportBatch(ctx, s.db, carrier, id, false)
	if err != nil {
		return ImportBatchDetail{}, err
	}
	rows, err := loadImportRows(ctx, s.db, id)
	if err != nil {
		return ImportBatchDetail{}, err
	}
	return ImportBatchDetail{Batch: batch, Rows: rows}, nil
}

func (s *PgImportStore) ListImportErrors(ctx context.Context, carrier, batchID string) ([]ImportRow, error) {
	detail, err := s.GetImportBatch(ctx, carrier, batchID)
	if err != nil {
		return nil, err
	}
	result := make([]ImportRow, 0)
	for _, row := range detail.Rows {
		if row.ValidationStatus == ImportRowInvalid || row.ValidationStatus == ImportRowDuplicate {
			result = append(result, row)
		}
	}
	return result, nil
}

func (s *PgImportStore) CommitImport(ctx context.Context, actor PolicyActor, batchID string) (ImportBatch, error) {
	return s.mutateImport(ctx, actor, batchID, false)
}

func (s *PgImportStore) RollbackImport(ctx context.Context, actor PolicyActor, batchID string) (ImportBatch, error) {
	return s.mutateImport(ctx, actor, batchID, true)
}

func (s *PgImportStore) mutateImport(ctx context.Context, actor PolicyActor, batchID string, rollback bool) (ImportBatch, error) {
	if s == nil || s.db == nil {
		return ImportBatch{}, fmt.Errorf("mutate import batch: %w", ErrAccessGateDependencyMissing)
	}
	id, err := uuid.Parse(strings.TrimSpace(batchID))
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import batch id: %w", err)
	}
	operatorID, err := nullableUUID(actor.SubjectID)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("parse import operator: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("begin import batch mutation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	batch, err := loadImportBatch(ctx, tx, actor.Carrier, id, true)
	if err != nil {
		return ImportBatch{}, err
	}
	if rollback {
		return rollbackLockedImport(ctx, tx, batch, operatorID, actor)
	}
	return commitLockedImport(ctx, tx, batch, operatorID, actor)
}

func commitLockedImport(ctx context.Context, tx pgx.Tx, batch ImportBatch, operatorID *uuid.UUID, actor PolicyActor) (ImportBatch, error) {
	if batch.Status == ImportBatchCommitted {
		return batch, nil
	}
	if batch.Status != ImportBatchValidated {
		return ImportBatch{}, fmt.Errorf("commit import batch: %w", ErrImportBatchConflict)
	}
	if batch.FailurePolicy == ImportFailureStrict && batch.InvalidCount > 0 {
		return ImportBatch{}, fmt.Errorf("commit import batch with invalid rows: %w", ErrImportBatchConflict)
	}
	if batch.Type == ImportTypeRuleDimension {
		return commitLockedRuleDimension(ctx, tx, batch, operatorID)
	}
	if batch.Type != ImportTypeAccessList {
		return ImportBatch{}, fmt.Errorf("commit import batch: %w", ErrImportBatchConflict)
	}
	rows, err := loadImportRows(ctx, tx, uuid.MustParse(batch.ID))
	if err != nil {
		return ImportBatch{}, err
	}
	entries, err := compiledImportRowEntries(rows, batch.EntryType)
	if err != nil {
		return ImportBatch{}, err
	}
	serialNumbers := serialNumbersFromRows(entries)
	snapshot, err := loadImportListSnapshot(ctx, tx, actor, batch.EntryType, serialNumbers, batch.Mode == ImportModeReplace)
	if err != nil {
		return ImportBatch{}, err
	}
	if batch.ScopeSHA256 == "" || batch.ScopeSHA256 != importListScopeSHA256(snapshot) {
		return ImportBatch{}, fmt.Errorf("access list changed after import preview: %w", ErrImportBatchConflict)
	}
	now := time.Now().UTC()
	if batch.Mode == ImportModeReplace {
		activeIDs := make([]uuid.UUID, 0, len(snapshot))
		for _, old := range snapshot {
			if old.Status == ListEntryStatusActive {
				activeIDs = append(activeIDs, old.ID)
			}
		}
		if len(activeIDs) > 0 {
			query, args, err := storage.Psql.Update("device_access_list_entries").
				Set("status", ListEntryStatusDisabled).Set("reason_code", "import_replaced").
				Set("reason", "replaced by import batch").Set("source_batch_id", uuid.MustParse(batch.ID)).Set("updated_at", now).
				Where(sq.Eq{"id": activeIDs, "carrier": batch.Carrier, "entry_type": batch.EntryType, "identity_type": IdentityTypeSerialNumber, "status": ListEntryStatusActive}).ToSql()
			if err != nil {
				return ImportBatch{}, fmt.Errorf("build replace access list: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return ImportBatch{}, fmt.Errorf("replace access list: %w", err)
			}
		}
		for _, old := range snapshot {
			serialNumbers = appendUniqueString(serialNumbers, old.IdentityValue)
		}
	}
	batchUUID := uuid.MustParse(batch.ID)
	for _, rowEntry := range entries {
		entryID, err := upsertImportedListEntry(ctx, tx, batch.Carrier, batchUUID, rowEntry.Entry, now)
		if err != nil {
			return ImportBatch{}, err
		}
		query, args, err := storage.Psql.Update("device_access_import_rows").Set("target_id", entryID).
			Where(sq.Eq{"batch_id": batchUUID, "row_number": rowEntry.RowNumber}).ToSql()
		if err != nil {
			return ImportBatch{}, fmt.Errorf("build import row target update: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return ImportBatch{}, fmt.Errorf("update import row target: %w", err)
		}
	}
	for _, serialNumber := range serialNumbers {
		if err := insertListReevaluationOutbox(ctx, tx, batch.Carrier, serialNumber, now); err != nil {
			return ImportBatch{}, err
		}
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("encode import rollback snapshot: %w", err)
	}
	if err := updateImportBatchStatus(ctx, tx, batchUUID, ImportBatchCommitted, operatorID, nil, now, snapshotJSON); err != nil {
		return ImportBatch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, fmt.Errorf("commit imported access list: %w", err)
	}
	batch.Status = ImportBatchCommitted
	batch.Snapshot = snapshotJSON
	batch.UpdatedAt = now
	batch.CommittedAt = &now
	if operatorID != nil {
		batch.CommittedBy = operatorID.String()
	}
	return batch, nil
}

func rollbackLockedImport(ctx context.Context, tx pgx.Tx, batch ImportBatch, operatorID *uuid.UUID, actor PolicyActor) (ImportBatch, error) {
	if batch.Status == ImportBatchRolledBack {
		return batch, nil
	}
	if batch.Status != ImportBatchCommitted {
		return ImportBatch{}, fmt.Errorf("rollback import batch: %w", ErrImportBatchConflict)
	}
	if batch.Type == ImportTypeRuleDimension {
		return rollbackLockedRuleDimension(ctx, tx, batch, operatorID)
	}
	if batch.Type != ImportTypeAccessList {
		return ImportBatch{}, fmt.Errorf("rollback import batch: %w", ErrImportBatchConflict)
	}
	reverseBatch, err := rollbackImportBatch(ctx, tx, batch, actor, operatorID)
	if err != nil {
		return ImportBatch{}, err
	}
	now := time.Now().UTC()
	if err := updateImportBatchStatus(ctx, tx, uuid.MustParse(batch.ID), ImportBatchRolledBack, nil, operatorID, now, nil); err != nil {
		return ImportBatch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, fmt.Errorf("commit import rollback: %w", err)
	}
	if reverseBatch != nil {
		return *reverseBatch, nil
	}
	batch.Status = ImportBatchRolledBack
	batch.UpdatedAt = now
	batch.RolledBackAt = &now
	if operatorID != nil {
		batch.RolledBackBy = operatorID.String()
	}
	return batch, nil
}

type ruleDimensionSnapshot struct {
	Before  []RuleDimensionValue `json:"before"`
	Applied []RuleDimensionValue `json:"applied"`
}

func commitLockedRuleDimension(ctx context.Context, tx pgx.Tx, batch ImportBatch, operatorID *uuid.UUID) (ImportBatch, error) {
	version, rule, err := lockRuleDimensionTarget(ctx, tx, batch)
	if err != nil {
		return ImportBatch{}, err
	}
	rows, err := loadImportRows(ctx, tx, uuid.MustParse(batch.ID))
	if err != nil {
		return ImportBatch{}, err
	}
	incoming, err := ruleDimensionValuesFromRows(rows, batch.Dimension)
	if err != nil {
		return ImportBatch{}, err
	}
	before := compiledRuleDimension(rule, batch.Dimension)
	if batch.ScopeSHA256 == "" || batch.ScopeSHA256 != ruleDimensionScopeSHA256(before) {
		return ImportBatch{}, fmt.Errorf("rule dimension changed after import preview: %w", ErrImportBatchConflict)
	}
	applied := incoming
	if batch.Mode == ImportModeAppend {
		applied = append(append([]RuleDimensionValue{}, before...), incoming...)
	}
	applied = uniqueRuleDimensionValues(applied)
	targets, err := replaceRuleDimension(ctx, tx, rule, batch.Dimension, applied)
	if err != nil {
		return ImportBatch{}, err
	}
	if err := updateRuleDimensionRowTargets(ctx, tx, uuid.MustParse(batch.ID), rows, batch.Dimension, targets, uuid.MustParse(batch.TargetRuleID)); err != nil {
		return ImportBatch{}, err
	}
	if err := refreshDraftContentHash(ctx, tx, version); err != nil {
		return ImportBatch{}, err
	}
	snapshotJSON, err := json.Marshal(ruleDimensionSnapshot{Before: before, Applied: applied})
	if err != nil {
		return ImportBatch{}, fmt.Errorf("encode rule dimension rollback snapshot: %w", err)
	}
	now := time.Now().UTC()
	if err := updateImportBatchStatus(ctx, tx, uuid.MustParse(batch.ID), ImportBatchCommitted, operatorID, nil, now, snapshotJSON); err != nil {
		return ImportBatch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, fmt.Errorf("commit imported rule dimension: %w", err)
	}
	batch.Status, batch.Snapshot, batch.UpdatedAt, batch.CommittedAt = ImportBatchCommitted, snapshotJSON, now, &now
	if operatorID != nil {
		batch.CommittedBy = operatorID.String()
	}
	return batch, nil
}

func rollbackLockedRuleDimension(ctx context.Context, tx pgx.Tx, batch ImportBatch, operatorID *uuid.UUID) (ImportBatch, error) {
	version, rule, err := lockRuleDimensionTarget(ctx, tx, batch)
	if err != nil {
		return ImportBatch{}, err
	}
	var snapshot ruleDimensionSnapshot
	if err := json.Unmarshal(batch.Snapshot, &snapshot); err != nil {
		return ImportBatch{}, fmt.Errorf("decode rule dimension rollback snapshot: %w", err)
	}
	current := compiledRuleDimension(rule, batch.Dimension)
	if !sameRuleDimensionSet(current, snapshot.Applied) {
		reverse, err := createReverseRuleDimensionBatch(ctx, tx, batch, version, rule, current, snapshot.Before, operatorID)
		if err != nil {
			return ImportBatch{}, err
		}
		now := time.Now().UTC()
		if err := updateImportBatchStatus(ctx, tx, uuid.MustParse(batch.ID), ImportBatchRolledBack, nil, operatorID, now, nil); err != nil {
			return ImportBatch{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ImportBatch{}, fmt.Errorf("commit reverse rule dimension batch: %w", err)
		}
		return reverse, nil
	}
	if _, err := replaceRuleDimension(ctx, tx, rule, batch.Dimension, snapshot.Before); err != nil {
		return ImportBatch{}, err
	}
	if err := refreshDraftContentHash(ctx, tx, version); err != nil {
		return ImportBatch{}, err
	}
	now := time.Now().UTC()
	if err := updateImportBatchStatus(ctx, tx, uuid.MustParse(batch.ID), ImportBatchRolledBack, nil, operatorID, now, nil); err != nil {
		return ImportBatch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportBatch{}, fmt.Errorf("commit rule dimension rollback: %w", err)
	}
	batch.Status, batch.UpdatedAt, batch.RolledBackAt = ImportBatchRolledBack, now, &now
	if operatorID != nil {
		batch.RolledBackBy = operatorID.String()
	}
	return batch, nil
}

func createReverseRuleDimensionBatch(
	ctx context.Context,
	tx pgx.Tx,
	original ImportBatch,
	version PolicyVersion,
	rule CompiledRule,
	current, desired []RuleDimensionValue,
	operatorID *uuid.UUID,
) (ImportBatch, error) {
	now := time.Now().UTC()
	reverseID := uuid.New()
	snapshotJSON, err := json.Marshal(ruleDimensionSnapshot{Before: current, Applied: desired})
	if err != nil {
		return ImportBatch{}, fmt.Errorf("encode reverse rule dimension snapshot: %w", err)
	}
	desiredJSON, err := json.Marshal(desired)
	if err != nil {
		return ImportBatch{}, fmt.Errorf("encode reverse rule dimension values: %w", err)
	}
	rows := make([]ImportRow, 0, len(desired))
	for index, value := range desired {
		normalized := normalizedDimensionRow(value)
		rows = append(rows, ImportRow{
			RowNumber: index + 1, RawValue: normalized, NormalizedValue: normalized,
			ValidationStatus: ImportRowValid, CreatedAt: now,
		})
	}
	affectedCount := len(current)
	if len(desired) > affectedCount {
		affectedCount = len(desired)
	}
	reverse := ImportBatch{
		ID: reverseID.String(), Carrier: original.Carrier, Type: ImportTypeRuleDimension,
		TargetPolicyVersionID: original.TargetPolicyVersionID, TargetRuleID: original.TargetRuleID, Dimension: original.Dimension,
		Mode: ImportModeReplace, FailurePolicy: ImportFailureStrict, Status: ImportBatchCommitted,
		SourceFilename: "reverse-" + original.ID + ".internal", ContentSHA256: contentSHA256(desiredJSON),
		ScopeSHA256: ruleDimensionScopeSHA256(current), TotalCount: affectedCount, ValidCount: len(rows), ChangedCount: affectedCount,
		IdempotencyKey: "reverse:" + original.ID, ReversalOfBatchID: original.ID,
		CreatedAt: now, UpdatedAt: now, CommittedAt: &now, Snapshot: snapshotJSON,
	}
	if operatorID != nil {
		reverse.CreatedBy = operatorID.String()
		reverse.CommittedBy = operatorID.String()
	}
	if err := insertCommittedReverseBatch(ctx, tx, reverse, rows, operatorID); err != nil {
		return ImportBatch{}, err
	}
	targets, err := replaceRuleDimension(ctx, tx, rule, original.Dimension, desired)
	if err != nil {
		return ImportBatch{}, err
	}
	if err := updateRuleDimensionRowTargets(ctx, tx, reverseID, rows, original.Dimension, targets, uuid.MustParse(original.TargetRuleID)); err != nil {
		return ImportBatch{}, err
	}
	if err := refreshDraftContentHash(ctx, tx, version); err != nil {
		return ImportBatch{}, err
	}
	return reverse, nil
}

func lockRuleDimensionTarget(ctx context.Context, tx pgx.Tx, batch ImportBatch) (PolicyVersion, CompiledRule, error) {
	versionID, err := uuid.Parse(batch.TargetPolicyVersionID)
	if err != nil {
		return PolicyVersion{}, CompiledRule{}, fmt.Errorf("parse target policy version: %w", err)
	}
	ruleID, err := uuid.Parse(batch.TargetRuleID)
	if err != nil {
		return PolicyVersion{}, CompiledRule{}, fmt.Errorf("parse target rule: %w", err)
	}
	query, args, err := storage.Psql.Select("pv.status", "ps.carrier").
		From("device_access_rules r").
		Join("device_access_policy_versions pv ON pv.id = r.policy_version_id").
		Join("device_access_policy_sets ps ON ps.id = pv.policy_set_id").
		Where(sq.Eq{"r.id": ruleID, "r.policy_version_id": versionID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return PolicyVersion{}, CompiledRule{}, fmt.Errorf("build rule dimension target lock: %w", err)
	}
	var status PolicyVersionStatus
	var carrier string
	if err := tx.QueryRow(ctx, query, args...).Scan(&status, &carrier); err != nil {
		return PolicyVersion{}, CompiledRule{}, fmt.Errorf("lock rule dimension target: %w", err)
	}
	if carrier != batch.Carrier {
		return PolicyVersion{}, CompiledRule{}, ErrPolicyCarrierScope
	}
	if status != PolicyVersionDraft {
		return PolicyVersion{}, CompiledRule{}, ErrPolicyVersionImmutable
	}
	version, err := (&PgPolicyStore{db: tx}).GetVersion(ctx, versionID.String())
	if err != nil {
		return PolicyVersion{}, CompiledRule{}, fmt.Errorf("load locked policy version: %w", err)
	}
	for _, rule := range version.Policy.Rules {
		if rule.ID == ruleID.String() {
			return version, rule, nil
		}
	}
	return PolicyVersion{}, CompiledRule{}, pgx.ErrNoRows
}

func ruleDimensionValuesFromRows(rows []ImportRow, dimension ImportDimension) ([]RuleDimensionValue, error) {
	values := make([]RuleDimensionValue, 0, len(rows))
	for _, row := range rows {
		if row.ValidationStatus != ImportRowValid && row.ValidationStatus != ImportRowNoChange {
			continue
		}
		value, err := ruleDimensionValueFromRow(row, dimension)
		if err != nil {
			return nil, fmt.Errorf("decode import row %d: %w", row.RowNumber, err)
		}
		values = append(values, value)
	}
	return values, nil
}

func uniqueRuleDimensionValues(values []RuleDimensionValue) []RuleDimensionValue {
	seen := make(map[string]struct{}, len(values))
	result := make([]RuleDimensionValue, 0, len(values))
	for _, value := range values {
		key := ruleDimensionValueKey(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sameRuleDimensionSet(left, right []RuleDimensionValue) bool {
	leftSet, rightSet := ruleDimensionKeySet(left), ruleDimensionKeySet(right)
	if len(leftSet) != len(rightSet) {
		return false
	}
	for key := range leftSet {
		if _, exists := rightSet[key]; !exists {
			return false
		}
	}
	return true
}

func replaceRuleDimension(ctx context.Context, tx pgx.Tx, rule CompiledRule, dimension ImportDimension, values []RuleDimensionValue) (map[string]uuid.UUID, error) {
	ruleID := uuid.MustParse(rule.ID)
	targets := make(map[string]uuid.UUID, len(values))
	if dimension == ImportDimensionSN {
		scope := SerialScope{Type: SerialScopeAll}
		if len(values) > 0 {
			scope.Type = SerialScopeList
			for _, value := range values {
				scope.Values = append(scope.Values, value.SerialNumber)
				targets[ruleDimensionValueKey(value)] = ruleID
			}
		}
		scopeJSON, err := json.Marshal(scope)
		if err != nil {
			return nil, fmt.Errorf("encode imported serial scope: %w", err)
		}
		query, args, err := storage.Psql.Update("device_access_rules").Set("serial_scope_type", scope.Type).Set("serial_scope", scopeJSON).Where(sq.Eq{"id": ruleID}).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build imported serial scope update: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("update imported serial scope: %w", err)
		}
		return targets, nil
	}

	conditionType := conditionTypeForDimension(dimension)
	required, ttlSeconds, allowMissing := true, int64(900), false
	for _, condition := range rule.Conditions {
		if condition.Type != conditionType || !legacyDimensionCondition(condition, dimension) {
			continue
		}
		required, ttlSeconds = condition.Required, int64(condition.EvidenceTTL/time.Second)
		allowMissing = importedGPSAllowMissing(condition)
		break
	}
	deleteBuilder := storage.Psql.Delete("device_access_conditions").Where(sq.Eq{"rule_id": ruleID, "condition_type": conditionType})
	if dimension == ImportDimensionIP {
		deleteBuilder = deleteBuilder.Where(sq.Eq{"operator": ConditionOperatorIPRange})
	}
	if dimension == ImportDimensionGPS {
		deleteBuilder = deleteBuilder.Where(sq.Eq{"operator": ConditionOperatorWithinBounds})
	}
	query, args, err := deleteBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build imported rule dimension delete: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("delete previous rule dimension: %w", err)
	}

	if (dimension == ImportDimensionTAC || dimension == ImportDimensionECGI) && len(values) > 0 {
		conditionID := uuid.New()
		expected := make([]string, 0, len(values))
		for _, value := range values {
			expected = append(expected, value.Value)
			targets[ruleDimensionValueKey(value)] = conditionID
		}
		expectedJSON, err := json.Marshal(expected)
		if err != nil {
			return nil, fmt.Errorf("encode imported rule dimension values: %w", err)
		}
		if err := insertImportedCondition(ctx, tx, conditionID, ruleID, conditionType, ConditionOperatorIn, expectedJSON, required, ttlSeconds); err != nil {
			return nil, err
		}
		return targets, nil
	}
	if (dimension == ImportDimensionIP || dimension == ImportDimensionGPS) && len(values) > 0 {
		conditionID := uuid.New()
		var operator ConditionOperator
		var expected []byte
		switch dimension {
		case ImportDimensionIP:
			operator = ConditionOperatorIPRange
			ranges := make([]IPRange, 0, len(values))
			for _, value := range values {
				ranges = append(ranges, *value.IPRange)
				targets[ruleDimensionValueKey(value)] = conditionID
			}
			expected, err = json.Marshal(ranges)
		case ImportDimensionGPS:
			operator = ConditionOperatorWithinBounds
			bounds := make([]GeoBounds, 0, len(values))
			for _, value := range values {
				item := *value.GeoBounds
				item.AllowMissing = allowMissing
				bounds = append(bounds, item)
				targets[ruleDimensionValueKey(value)] = conditionID
			}
			expected, err = json.Marshal(bounds)
		}
		if err != nil {
			return nil, fmt.Errorf("encode imported range alternatives: %w", err)
		}
		if err := insertImportedCondition(ctx, tx, conditionID, ruleID, conditionType, operator, expected, required, ttlSeconds); err != nil {
			return nil, err
		}
		return targets, nil
	}
	for _, value := range values {
		conditionID := uuid.New()
		var operator ConditionOperator
		var expected []byte
		switch dimension {
		case ImportDimensionIP:
			operator = ConditionOperatorIPRange
			expected, err = json.Marshal(value.IPRange)
		case ImportDimensionGPS:
			operator = ConditionOperatorWithinBounds
			bounds := *value.GeoBounds
			bounds.AllowMissing = allowMissing
			expected, err = json.Marshal(bounds)
		}
		if err != nil {
			return nil, fmt.Errorf("encode imported range condition: %w", err)
		}
		if err := insertImportedCondition(ctx, tx, conditionID, ruleID, conditionType, operator, expected, required, ttlSeconds); err != nil {
			return nil, err
		}
		targets[ruleDimensionValueKey(value)] = conditionID
	}
	return targets, nil
}

func importedGPSAllowMissing(condition CompiledCondition) bool {
	if condition.GeoBounds != nil && condition.GeoBounds.AllowMissing {
		return true
	}
	for _, bounds := range condition.GeoBoundsAny {
		if bounds.AllowMissing {
			return true
		}
	}
	return false
}

func legacyDimensionCondition(condition CompiledCondition, dimension ImportDimension) bool {
	switch dimension {
	case ImportDimensionTAC, ImportDimensionECGI:
		return condition.Operator == ConditionOperatorEqual || condition.Operator == ConditionOperatorIn
	case ImportDimensionIP:
		return condition.Operator == ConditionOperatorIPRange
	case ImportDimensionGPS:
		return condition.Operator == ConditionOperatorWithinBounds
	default:
		return false
	}
}

func insertImportedCondition(ctx context.Context, tx pgx.Tx, conditionID, ruleID uuid.UUID, conditionType ConditionType, operator ConditionOperator, expected []byte, required bool, ttlSeconds int64) error {
	query, args, err := storage.Psql.Insert("device_access_conditions").
		Columns("id", "rule_id", "condition_type", "operator", "expected_value", "required", "evidence_ttl_seconds").
		Values(conditionID, ruleID, conditionType, operator, expected, required, ttlSeconds).ToSql()
	if err != nil {
		return fmt.Errorf("build imported rule dimension insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert imported rule dimension: %w", err)
	}
	return nil
}

func updateRuleDimensionRowTargets(ctx context.Context, tx pgx.Tx, batchID uuid.UUID, rows []ImportRow, dimension ImportDimension, targets map[string]uuid.UUID, fallback uuid.UUID) error {
	for _, row := range rows {
		if row.ValidationStatus != ImportRowValid && row.ValidationStatus != ImportRowNoChange {
			continue
		}
		value, err := ruleDimensionValueFromRow(row, dimension)
		if err != nil {
			return err
		}
		target := targets[ruleDimensionValueKey(value)]
		if target == uuid.Nil {
			target = fallback
		}
		query, args, err := storage.Psql.Update("device_access_import_rows").Set("target_id", target).
			Where(sq.Eq{"batch_id": batchID, "row_number": row.RowNumber}).ToSql()
		if err != nil {
			return fmt.Errorf("build rule dimension row target update: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("update rule dimension row target: %w", err)
		}
	}
	return nil
}

func refreshDraftContentHash(ctx context.Context, tx pgx.Tx, version PolicyVersion) error {
	reloaded, err := (&PgPolicyStore{db: tx}).GetVersion(ctx, version.ID)
	if err != nil {
		return fmt.Errorf("reload imported policy draft: %w", err)
	}
	reloaded.Policy.VersionID = ""
	payload, err := json.Marshal(reloaded.Policy)
	if err != nil {
		return fmt.Errorf("encode imported policy draft: %w", err)
	}
	query, args, err := storage.Psql.Update("device_access_policy_versions").Set("content_hash", contentHash(payload)).Where(sq.Eq{"id": uuid.MustParse(version.ID), "status": PolicyVersionDraft}).ToSql()
	if err != nil {
		return fmt.Errorf("build imported policy hash update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update imported policy hash: %w", err)
	}
	return nil
}

type importScanner interface{ Scan(...any) error }
type importQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type importQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func importBatchSelect() sq.SelectBuilder {
	return storage.Psql.Select(
		"id", "carrier", "import_type", "entry_type", "target_policy_version_id", "target_rule_id", "dimension", "mode", "failure_policy", "status", "source_filename",
		"content_sha256", "scope_sha256", "total_count", "valid_count", "invalid_count", "changed_count", "idempotency_key",
		"created_by", "committed_by", "rolled_back_by", "reversal_of_batch_id", "created_at", "updated_at", "committed_at", "rolled_back_at", "snapshot",
	).From("device_access_import_batches")
}

func loadImportBatch(ctx context.Context, queryRower importQueryRower, carrier string, id uuid.UUID, lock bool) (ImportBatch, error) {
	builder := importBatchSelect().Where(sq.Eq{"id": id, "carrier": carrier})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return ImportBatch{}, fmt.Errorf("build import batch query: %w", err)
	}
	batch, err := scanImportBatch(queryRower.QueryRow(ctx, query, args...))
	if err != nil {
		return ImportBatch{}, fmt.Errorf("load import batch: %w", err)
	}
	return batch, nil
}

func scanImportBatch(scanner importScanner) (ImportBatch, error) {
	var batch ImportBatch
	var entryType, dimension, scopeSHA256, idempotencyKey *string
	var targetPolicyVersionID, targetRuleID, createdBy, committedBy, rolledBackBy, reversalOfBatchID *uuid.UUID
	var snapshot []byte
	err := scanner.Scan(&batch.ID, &batch.Carrier, &batch.Type, &entryType, &targetPolicyVersionID, &targetRuleID, &dimension, &batch.Mode, &batch.FailurePolicy, &batch.Status,
		&batch.SourceFilename, &batch.ContentSHA256, &scopeSHA256, &batch.TotalCount, &batch.ValidCount, &batch.InvalidCount,
		&batch.ChangedCount, &idempotencyKey, &createdBy, &committedBy, &rolledBackBy, &reversalOfBatchID, &batch.CreatedAt,
		&batch.UpdatedAt, &batch.CommittedAt, &batch.RolledBackAt, &snapshot)
	if err != nil {
		return ImportBatch{}, err
	}
	if entryType != nil {
		batch.EntryType = ListEntryType(*entryType)
	}
	if targetPolicyVersionID != nil {
		batch.TargetPolicyVersionID = targetPolicyVersionID.String()
	}
	if targetRuleID != nil {
		batch.TargetRuleID = targetRuleID.String()
	}
	if dimension != nil {
		batch.Dimension = ImportDimension(*dimension)
	}
	if idempotencyKey != nil {
		batch.IdempotencyKey = *idempotencyKey
	}
	if scopeSHA256 != nil {
		batch.ScopeSHA256 = *scopeSHA256
	}
	if createdBy != nil {
		batch.CreatedBy = createdBy.String()
	}
	if committedBy != nil {
		batch.CommittedBy = committedBy.String()
	}
	if rolledBackBy != nil {
		batch.RolledBackBy = rolledBackBy.String()
	}
	if reversalOfBatchID != nil {
		batch.ReversalOfBatchID = reversalOfBatchID.String()
	}
	batch.Snapshot = snapshot
	return batch, nil
}

func loadImportRows(ctx context.Context, queryer importQuerier, batchID uuid.UUID) ([]ImportRow, error) {
	query, args, err := storage.Psql.Select("id", "row_number", "raw_value", "normalized_value", "validation_status", "error_code", "error_message", "target_id", "created_at").
		From("device_access_import_rows").Where(sq.Eq{"batch_id": batchID}).OrderBy("row_number ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build import rows query: %w", err)
	}
	rows, err := queryer.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query import rows: %w", err)
	}
	defer rows.Close()
	result := make([]ImportRow, 0)
	for rows.Next() {
		var row ImportRow
		var raw, normalized []byte
		var errorCode, errorMessage *string
		var targetID *uuid.UUID
		if err := rows.Scan(&row.ID, &row.RowNumber, &raw, &normalized, &row.ValidationStatus, &errorCode, &errorMessage, &targetID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan import row: %w", err)
		}
		row.BatchID = batchID.String()
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &row.RawValue); err != nil {
				return nil, fmt.Errorf("decode import row raw value: %w", err)
			}
		}
		if len(normalized) > 0 {
			if err := json.Unmarshal(normalized, &row.NormalizedValue); err != nil {
				return nil, fmt.Errorf("decode import row normalized value: %w", err)
			}
		}
		if errorCode != nil {
			row.ErrorCode = *errorCode
		}
		if errorMessage != nil {
			row.ErrorMessage = *errorMessage
		}
		if targetID != nil {
			row.TargetID = targetID.String()
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import rows: %w", err)
	}
	return result, nil
}

type compiledImportRowEntry struct {
	RowNumber int
	Entry     CompiledListEntry
}

func compiledImportRowEntries(rows []ImportRow, entryType ListEntryType) ([]compiledImportRowEntry, error) {
	entries := make([]compiledImportRowEntry, 0, len(rows))
	for _, row := range rows {
		if row.ValidationStatus != ImportRowValid && row.ValidationStatus != ImportRowNoChange {
			continue
		}
		serialNumber, _ := row.NormalizedValue["serial_number"].(string)
		if serialNumber == "" {
			return nil, fmt.Errorf("import row %d has no serial number: %w", row.RowNumber, ErrImportBatchConflict)
		}
		entry := CompiledListEntry{Type: entryType, IdentityType: IdentityTypeSerialNumber, IdentityValue: serialNumber, Status: ListEntryStatusActive}
		entry.Reason, _ = row.NormalizedValue["reason"].(string)
		if value, ok := row.NormalizedValue["valid_from"].(string); ok {
			parsed, err := parseImportTime(value)
			if err != nil {
				return nil, err
			}
			entry.ValidFrom = parsed
		}
		if value, ok := row.NormalizedValue["valid_until"].(string); ok {
			parsed, err := parseImportTime(value)
			if err != nil {
				return nil, err
			}
			entry.ValidUntil = parsed
		}
		entries = append(entries, compiledImportRowEntry{RowNumber: row.RowNumber, Entry: entry})
	}
	return entries, nil
}

func serialNumbersFromRows(entries []compiledImportRowEntry) []string {
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.Entry.IdentityValue)
	}
	return values
}

type importListSnapshotEntry struct {
	ID            uuid.UUID       `json:"id"`
	EntryType     ListEntryType   `json:"entry_type"`
	IdentityType  IdentityType    `json:"identity_type"`
	IdentityValue string          `json:"identity_value"`
	ReasonCode    string          `json:"reason_code"`
	Reason        string          `json:"reason,omitempty"`
	ValidFrom     time.Time       `json:"valid_from"`
	ValidUntil    *time.Time      `json:"valid_until,omitempty"`
	Status        ListEntryStatus `json:"status"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty"`
	ApprovedBy    *uuid.UUID      `json:"approved_by,omitempty"`
	SourceBatchID *uuid.UUID      `json:"source_batch_id,omitempty"`
}

func loadImportListSnapshot(ctx context.Context, queryer importQuerier, actor PolicyActor, entryType ListEntryType, serialNumbers []string, replace bool) ([]importListSnapshotEntry, error) {
	builder := storage.Psql.Select("le.id", "le.entry_type", "le.identity_type", "le.identity_value", "le.reason_code", "COALESCE(le.reason, '')", "le.valid_from", "le.valid_until", "le.status", "le.created_by", "le.approved_by", "le.source_batch_id").
		From("device_access_list_entries le").
		LeftJoin("devices d ON d.carrier = le.carrier AND d.serial_number = le.identity_value").
		Where(sq.Eq{"le.carrier": actor.Carrier, "le.entry_type": entryType, "le.identity_type": IdentityTypeSerialNumber})
	if !replace {
		builder = builder.Where(sq.Eq{"le.identity_value": serialNumbers})
	}
	builder = applyAccessIdentityVisibility(builder, "d.id", "le.carrier", "le.identity_value", actor.VisibleGroups)
	query, args, err := builder.OrderBy("le.identity_value ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build import snapshot query: %w", err)
	}
	rows, err := queryer.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query import snapshot: %w", err)
	}
	defer rows.Close()
	result := make([]importListSnapshotEntry, 0)
	for rows.Next() {
		var entry importListSnapshotEntry
		if err := rows.Scan(&entry.ID, &entry.EntryType, &entry.IdentityType, &entry.IdentityValue, &entry.ReasonCode, &entry.Reason, &entry.ValidFrom, &entry.ValidUntil, &entry.Status, &entry.CreatedBy, &entry.ApprovedBy, &entry.SourceBatchID); err != nil {
			return nil, fmt.Errorf("scan import snapshot: %w", err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import snapshot: %w", err)
	}
	return result, nil
}

func upsertImportedListEntry(ctx context.Context, tx pgx.Tx, carrier string, batchID uuid.UUID, entry CompiledListEntry, now time.Time) (uuid.UUID, error) {
	query, args, err := storage.Psql.Insert("device_access_list_entries").
		Columns("id", "carrier", "entry_type", "identity_type", "identity_value", "reason_code", "reason", "valid_from", "valid_until", "status", "source_batch_id", "created_at", "updated_at").
		Values(uuid.New(), carrier, entry.Type, entry.IdentityType, entry.IdentityValue, "managed_by_import", nullableString(entry.Reason), sq.Expr("COALESCE(?, now())", entry.ValidFrom), entry.ValidUntil, ListEntryStatusActive, batchID, now, now).
		Suffix(`ON CONFLICT (carrier,entry_type,identity_type,identity_value) DO UPDATE SET
			reason_code = EXCLUDED.reason_code, reason = EXCLUDED.reason, valid_from = COALESCE(?, device_access_list_entries.valid_from),
			valid_until = EXCLUDED.valid_until, status = EXCLUDED.status, source_batch_id = EXCLUDED.source_batch_id,
			updated_at = EXCLUDED.updated_at RETURNING id`, entry.ValidFrom).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build imported list entry upsert: %w", err)
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert imported list entry %s: %w", entry.IdentityValue, err)
	}
	return id, nil
}

func updateImportBatchStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status ImportBatchStatus, committedBy, rolledBackBy *uuid.UUID, now time.Time, snapshot []byte) error {
	update := storage.Psql.Update("device_access_import_batches").Set("status", status).Set("updated_at", now)
	if committedBy != nil {
		update = update.Set("committed_by", committedBy).Set("committed_at", now)
	}
	if rolledBackBy != nil {
		update = update.Set("rolled_back_by", rolledBackBy).Set("rolled_back_at", now)
	}
	if snapshot != nil {
		update = update.Set("snapshot", snapshot)
	}
	query, args, err := update.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build import batch status update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update import batch status: %w", err)
	}
	return nil
}

func rollbackImportBatch(ctx context.Context, tx pgx.Tx, batch ImportBatch, actor PolicyActor, operatorID *uuid.UUID) (*ImportBatch, error) {
	var snapshot []importListSnapshotEntry
	if len(batch.Snapshot) > 0 {
		if err := json.Unmarshal(batch.Snapshot, &snapshot); err != nil {
			return nil, fmt.Errorf("decode import rollback snapshot: %w", err)
		}
	}
	rows, err := loadImportRows(ctx, tx, uuid.MustParse(batch.ID))
	if err != nil {
		return nil, err
	}
	entries, err := compiledImportRowEntries(rows, batch.EntryType)
	if err != nil {
		return nil, err
	}
	identities := serialNumbersFromRows(entries)
	for _, old := range snapshot {
		identities = appendUniqueString(identities, old.IdentityValue)
	}
	if len(identities) == 0 {
		return nil, nil
	}
	scoped, err := loadImportListSnapshot(ctx, tx, actor, batch.EntryType, identities, false)
	if err != nil {
		return nil, err
	}
	if len(scoped) != len(identities) {
		return createReverseListBatch(ctx, tx, batch, snapshot, scoped, identities, operatorID)
	}
	batchUUID := uuid.MustParse(batch.ID)
	query, args, err := storage.Psql.Select("identity_value", "source_batch_id").From("device_access_list_entries").
		Where(sq.Eq{"carrier": batch.Carrier, "entry_type": batch.EntryType, "identity_type": IdentityTypeSerialNumber, "identity_value": identities}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build rollback conflict query: %w", err)
	}
	currentRows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rollback conflict: %w", err)
	}
	seen := make(map[string]struct{}, len(identities))
	conflicted := false
	for currentRows.Next() {
		var identity string
		var sourceBatchID *uuid.UUID
		if err := currentRows.Scan(&identity, &sourceBatchID); err != nil {
			currentRows.Close()
			return nil, fmt.Errorf("scan rollback conflict: %w", err)
		}
		seen[identity] = struct{}{}
		if sourceBatchID == nil || *sourceBatchID != batchUUID {
			conflicted = true
		}
	}
	if err := currentRows.Err(); err != nil {
		currentRows.Close()
		return nil, fmt.Errorf("iterate rollback conflict: %w", err)
	}
	currentRows.Close()
	for _, identity := range identities {
		if _, exists := seen[identity]; !exists {
			conflicted = true
		}
	}
	if conflicted {
		return createReverseListBatch(ctx, tx, batch, snapshot, scoped, identities, operatorID)
	}
	now := time.Now().UTC()
	query, args, err = storage.Psql.Update("device_access_list_entries").Set("status", ListEntryStatusDisabled).
		Set("reason_code", "import_rolled_back").Set("reason", "rolled back import batch").Set("updated_at", now).
		Where(sq.Eq{"carrier": batch.Carrier, "entry_type": batch.EntryType, "identity_type": IdentityTypeSerialNumber, "identity_value": identities, "source_batch_id": batchUUID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build rollback disable: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("disable imported entries during rollback: %w", err)
	}
	for _, old := range snapshot {
		query, args, err := storage.Psql.Update("device_access_list_entries").Set("reason_code", old.ReasonCode).Set("reason", nullableString(old.Reason)).
			Set("valid_from", old.ValidFrom).Set("valid_until", old.ValidUntil).Set("status", old.Status).Set("created_by", old.CreatedBy).
			Set("approved_by", old.ApprovedBy).Set("source_batch_id", old.SourceBatchID).Set("updated_at", now).
			Where(sq.Eq{"id": old.ID, "carrier": batch.Carrier, "entry_type": batch.EntryType}).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build rollback restore: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("restore imported entry %s: %w", old.IdentityValue, err)
		}
	}
	for _, identity := range identities {
		if err := insertListReevaluationOutbox(ctx, tx, batch.Carrier, identity, now); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func createReverseListBatch(
	ctx context.Context,
	tx pgx.Tx,
	original ImportBatch,
	beforeOriginal, current []importListSnapshotEntry,
	identities []string,
	operatorID *uuid.UUID,
) (*ImportBatch, error) {
	now := time.Now().UTC()
	reverseID := uuid.New()
	currentByIdentity := make(map[string]importListSnapshotEntry, len(current))
	for _, entry := range current {
		currentByIdentity[entry.IdentityValue] = entry
	}
	beforeByIdentity := make(map[string]importListSnapshotEntry, len(beforeOriginal))
	for _, entry := range beforeOriginal {
		beforeByIdentity[entry.IdentityValue] = entry
	}
	activeDesired := make([]CompiledListEntry, 0, len(beforeOriginal))
	for _, identity := range identities {
		old, existed := beforeByIdentity[identity]
		if !existed || old.Status != ListEntryStatusActive {
			continue
		}
		validFrom := old.ValidFrom
		activeDesired = append(activeDesired, CompiledListEntry{
			Type: old.EntryType, IdentityType: old.IdentityType, IdentityValue: old.IdentityValue,
			Status: ListEntryStatusActive, Reason: old.Reason, ValidFrom: &validFrom, ValidUntil: old.ValidUntil,
		})
	}
	sort.Slice(activeDesired, func(i, j int) bool { return activeDesired[i].IdentityValue < activeDesired[j].IdentityValue })
	currentSnapshot := make([]importListSnapshotEntry, 0, len(identities))
	for _, identity := range identities {
		if entry, exists := currentByIdentity[identity]; exists {
			currentSnapshot = append(currentSnapshot, entry)
		}
	}
	snapshotJSON, err := json.Marshal(currentSnapshot)
	if err != nil {
		return nil, fmt.Errorf("encode reverse import snapshot: %w", err)
	}
	desiredJSON, err := json.Marshal(activeDesired)
	if err != nil {
		return nil, fmt.Errorf("encode reverse import target: %w", err)
	}
	reverse := ImportBatch{
		ID: reverseID.String(), Carrier: original.Carrier, Type: ImportTypeAccessList, EntryType: original.EntryType,
		Mode: ImportModeAppend, FailurePolicy: ImportFailureStrict, Status: ImportBatchCommitted,
		SourceFilename: "reverse-" + original.ID + ".internal", ContentSHA256: contentSHA256(desiredJSON),
		ScopeSHA256: importListScopeSHA256(currentSnapshot), TotalCount: len(identities), ValidCount: len(activeDesired), ChangedCount: len(identities),
		IdempotencyKey: "reverse:" + original.ID, ReversalOfBatchID: original.ID,
		CreatedAt: now, UpdatedAt: now, CommittedAt: &now, Snapshot: snapshotJSON,
	}
	if operatorID != nil {
		reverse.CreatedBy = operatorID.String()
		reverse.CommittedBy = operatorID.String()
	}
	// Insert the compensating batch before assigning it as source_batch_id so
	// the list-entry foreign key remains valid throughout the transaction.
	if err := insertCommittedReverseBatch(ctx, tx, reverse, nil, operatorID); err != nil {
		return nil, err
	}
	disableIDs := make([]uuid.UUID, 0, len(currentSnapshot))
	for _, entry := range currentSnapshot {
		if entry.Status == ListEntryStatusActive {
			disableIDs = append(disableIDs, entry.ID)
		}
	}
	if len(disableIDs) > 0 {
		query, args, err := storage.Psql.Update("device_access_list_entries").
			Set("status", ListEntryStatusDisabled).Set("reason_code", "import_reversed").
			Set("reason", "reversed by compensating import batch").Set("source_batch_id", reverseID).Set("updated_at", now).
			Where(sq.Eq{"id": disableIDs, "carrier": original.Carrier, "entry_type": original.EntryType}).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build reverse list disable: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("disable current list entries for reverse batch: %w", err)
		}
	}
	rows := make([]ImportRow, 0, len(activeDesired))
	for index, entry := range activeDesired {
		entryID, err := upsertImportedListEntry(ctx, tx, original.Carrier, reverseID, entry, now)
		if err != nil {
			return nil, err
		}
		rows = append(rows, ImportRow{
			RowNumber: index + 1, RawValue: normalizedAccessListRow(entry), NormalizedValue: normalizedAccessListRow(entry),
			ValidationStatus: ImportRowValid, TargetID: entryID.String(), CreatedAt: now,
		})
	}
	for _, identity := range identities {
		if err := insertListReevaluationOutbox(ctx, tx, original.Carrier, identity, now); err != nil {
			return nil, err
		}
	}
	if err := insertReverseImportRows(ctx, tx, reverse, rows); err != nil {
		return nil, err
	}
	return &reverse, nil
}

func insertCommittedReverseBatch(ctx context.Context, tx pgx.Tx, batch ImportBatch, rows []ImportRow, operatorID *uuid.UUID) error {
	reversalID := uuid.MustParse(batch.ReversalOfBatchID)
	targetPolicyVersionID, err := nullableUUID(batch.TargetPolicyVersionID)
	if err != nil {
		return fmt.Errorf("parse reverse import policy target: %w", err)
	}
	targetRuleID, err := nullableUUID(batch.TargetRuleID)
	if err != nil {
		return fmt.Errorf("parse reverse import rule target: %w", err)
	}
	query, args, err := storage.Psql.Insert("device_access_import_batches").
		Columns("id", "carrier", "import_type", "entry_type", "target_policy_version_id", "target_rule_id", "dimension", "mode", "failure_policy", "status", "source_filename", "content_sha256", "scope_sha256", "total_count", "valid_count", "invalid_count", "changed_count", "snapshot", "idempotency_key", "created_by", "committed_by", "reversal_of_batch_id", "created_at", "updated_at", "committed_at").
		Values(uuid.MustParse(batch.ID), batch.Carrier, batch.Type, nullableString(string(batch.EntryType)), targetPolicyVersionID, targetRuleID, nullableString(string(batch.Dimension)), batch.Mode, batch.FailurePolicy, batch.Status,
			batch.SourceFilename, batch.ContentSHA256, nullableString(batch.ScopeSHA256), batch.TotalCount, batch.ValidCount, batch.InvalidCount, batch.ChangedCount, batch.Snapshot, nullableString(batch.IdempotencyKey), operatorID, operatorID, reversalID, batch.CreatedAt, batch.UpdatedAt, batch.CommittedAt).
		Suffix("ON CONFLICT (carrier, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING").ToSql()
	if err != nil {
		return fmt.Errorf("build reverse import batch insert: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert reverse import batch: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("reverse import batch already exists: %w", ErrImportBatchConflict)
	}
	return insertReverseImportRows(ctx, tx, batch, rows)
}

func insertReverseImportRows(ctx context.Context, tx pgx.Tx, batch ImportBatch, rows []ImportRow) error {
	for _, row := range rows {
		rawValue, err := json.Marshal(row.RawValue)
		if err != nil {
			return fmt.Errorf("encode reverse import row %d: %w", row.RowNumber, err)
		}
		normalizedValue, err := json.Marshal(row.NormalizedValue)
		if err != nil {
			return fmt.Errorf("encode normalized reverse import row %d: %w", row.RowNumber, err)
		}
		targetID, err := nullableUUID(row.TargetID)
		if err != nil {
			return fmt.Errorf("parse reverse import row target: %w", err)
		}
		query, args, err := storage.Psql.Insert("device_access_import_rows").
			Columns("id", "batch_id", "row_number", "raw_value", "normalized_value", "validation_status", "target_id", "created_at").
			Values(uuid.New(), uuid.MustParse(batch.ID), row.RowNumber, rawValue, normalizedValue, row.ValidationStatus, targetID, row.CreatedAt).ToSql()
		if err != nil {
			return fmt.Errorf("build reverse import row insert: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert reverse import row %d: %w", row.RowNumber, err)
		}
	}
	return nil
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

var _ ImportBatchStore = (*PgImportStore)(nil)
var _ ImportListSnapshotStore = (*PgImportStore)(nil)
var _ ImportBatchMutationStore = (*PgImportStore)(nil)
var _ ImportBatchReadStore = (*PgImportStore)(nil)
