package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgRepository struct {
	db storage.DB
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return newPgRepositoryWithDB(storage.NewPoolDB(pool))
}

func newPgRepositoryWithDB(db storage.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) UpsertCandidateObservation(
	ctx context.Context,
	observation Observation,
) (Candidate, error) {
	if err := validateIdentity(observation.Carrier, observation.SerialNumber); err != nil {
		return Candidate{}, err
	}

	query, args, err := storage.Psql.Insert("device_access_candidates").
		Columns(
			"carrier", "serial_number", "oui", "product_class", "software_version",
			"observed_remote_ip", "first_seen_at", "last_seen_at", "expires_at",
		).
		Values(
			observation.Carrier, observation.SerialNumber, observation.OUI,
			nullableString(observation.ProductClass), nullableString(observation.SoftwareVersion),
			nullableAddr(observation.RemoteIP), observation.ObservedAt, observation.ObservedAt,
			observation.ExpiresAt,
		).
		Suffix(`ON CONFLICT (carrier,serial_number) DO UPDATE SET
			oui = EXCLUDED.oui,
			product_class = EXCLUDED.product_class,
			software_version = EXCLUDED.software_version,
			observed_remote_ip = EXCLUDED.observed_remote_ip,
			last_seen_at = EXCLUDED.last_seen_at,
			inform_count = device_access_candidates.inform_count + 1,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.last_seen_at
			RETURNING id, carrier, serial_number, oui, COALESCE(product_class, ''),
				COALESCE(software_version, ''), observed_remote_ip, first_seen_at, last_seen_at,
				inform_count, review_status, reviewed_by, reviewed_at, expires_at`).
		ToSql()
	if err != nil {
		return Candidate{}, fmt.Errorf("build candidate observation upsert: %w", err)
	}

	var candidate Candidate
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&candidate.ID,
		&candidate.Carrier,
		&candidate.SerialNumber,
		&candidate.OUI,
		&candidate.ProductClass,
		&candidate.SoftwareVersion,
		&candidate.RemoteIP,
		&candidate.FirstSeenAt,
		&candidate.LastSeenAt,
		&candidate.InformCount,
		&candidate.ReviewStatus,
		&candidate.ReviewedBy,
		&candidate.ReviewedAt,
		&candidate.ExpiresAt,
	)
	if err != nil {
		return Candidate{}, fmt.Errorf("upsert candidate observation: %w", err)
	}
	return candidate, nil
}

func (r *PgRepository) LoadEvaluationContext(
	ctx context.Context,
	carrier string,
	serialNumber string,
) (EvaluationContext, error) {
	if err := validateIdentity(carrier, serialNumber); err != nil {
		return EvaluationContext{}, err
	}

	stateQuery, stateArgs, err := storage.Psql.
		Select(
			"id", "carrier", "serial_number", "device_id", "candidate_id", "state",
			"effective_decision", "reason_code", "policy_version_id", "evidence_version",
			"decision_version", "normal_tasks_frozen",
		).
		From("device_access_states").
		Where(sq.Eq{"carrier": carrier}).
		Where(sq.Eq{"serial_number": serialNumber}).
		ToSql()
	if err != nil {
		return EvaluationContext{}, fmt.Errorf("build evaluation state query: %w", err)
	}

	var result EvaluationContext
	var state AccessStateProjection
	err = r.db.QueryRow(ctx, stateQuery, stateArgs...).Scan(
		&state.ID,
		&state.Carrier,
		&state.SerialNumber,
		&state.DeviceID,
		&state.CandidateID,
		&state.State,
		&state.EffectiveDecision,
		&state.ReasonCode,
		&state.PolicyVersionID,
		&state.EvidenceVersion,
		&state.DecisionVersion,
		&state.NormalTasksFrozen,
	)
	switch {
	case err == nil:
		result.State = &state
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return EvaluationContext{}, fmt.Errorf("load device access state: %w", err)
	}

	evidenceQuery, evidenceArgs, err := storage.Psql.
		Select(
			"evidence_type", "evidence_status", "normalized_value", "value_hash", "source", "task_id",
			"observed_at", "expires_at", "evidence_version",
		).
		From("device_access_evidence").
		Where(sq.Eq{"carrier": carrier}).
		Where(sq.Eq{"serial_number": serialNumber}).
		OrderBy("evidence_version DESC", "evidence_type ASC").
		ToSql()
	if err != nil {
		return EvaluationContext{}, fmt.Errorf("build evaluation evidence query: %w", err)
	}
	rows, err := r.db.Query(ctx, evidenceQuery, evidenceArgs...)
	if err != nil {
		return EvaluationContext{}, fmt.Errorf("query evaluation evidence: %w", err)
	}
	defer rows.Close()

	result.Evidence.Carrier = carrier
	result.Evidence.SerialNumber = serialNumber
	for rows.Next() {
		var record EvidenceRecord
		var version int64
		if err := rows.Scan(
			&record.Type,
			&record.Status,
			&record.NormalizedValue,
			&record.ValueHash,
			&record.Source,
			&record.TaskID,
			&record.ObservedAt,
			&record.ExpiresAt,
			&version,
		); err != nil {
			return EvaluationContext{}, fmt.Errorf("scan evaluation evidence: %w", err)
		}
		if result.Evidence.Version == 0 {
			result.Evidence.Version = version
		}
		if version == result.Evidence.Version {
			result.Evidence.Records = append(result.Evidence.Records, record)
		}
	}
	if err := rows.Err(); err != nil {
		return EvaluationContext{}, fmt.Errorf("iterate evaluation evidence: %w", err)
	}
	return result, nil
}

func (r *PgRepository) AppendEvidence(ctx context.Context, evidence EvidenceBatch) (int64, error) {
	if err := validateEvidenceBatch(evidence); err != nil {
		return 0, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin append evidence transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Candidate observation is created before any evidence write. Locking that
	// stable identity row serializes Inform and probe callbacks for one device
	// without blocking unrelated devices.
	if err := lockEvidenceIdentity(ctx, tx, evidence.Carrier, evidence.SerialNumber); err != nil {
		return 0, err
	}
	latestVersion, err := latestEvidenceVersion(ctx, tx, evidence.Carrier, evidence.SerialNumber)
	if err != nil {
		return 0, err
	}
	baseVersion := evidence.Version - 1
	if baseVersion > latestVersion {
		return 0, fmt.Errorf("evidence base version %d is newer than stored version %d", baseVersion, latestVersion)
	}
	baseRecords, err := loadEvidenceVersion(ctx, tx, evidence.Carrier, evidence.SerialNumber, baseVersion)
	if err != nil {
		return 0, err
	}
	latestRecords := baseRecords
	if latestVersion != baseVersion {
		latestRecords, err = loadEvidenceVersion(ctx, tx, evidence.Carrier, evidence.SerialNumber, latestVersion)
		if err != nil {
			return 0, err
		}
	}
	replacements := changedAgainstBase(baseRecords, evidence.Records)
	if len(replacements) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit unchanged evidence transaction: %w", err)
		}
		return latestVersion, nil
	}
	evidence.Version = latestVersion + 1
	evidence.Records = mergeEvidenceRecords(latestRecords, replacements)
	if err := insertEvidenceRecords(ctx, tx, evidence); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit append evidence transaction: %w", err)
	}
	return evidence.Version, nil
}

func lockEvidenceIdentity(ctx context.Context, tx pgx.Tx, carrier, serialNumber string) error {
	query, args, err := storage.Psql.
		Select("id").
		From("device_access_candidates").
		Where(sq.Eq{"carrier": carrier, "serial_number": serialNumber}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build evidence identity lock query: %w", err)
	}
	var candidateID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&candidateID); err != nil {
		return fmt.Errorf("lock evidence identity: %w", err)
	}
	return nil
}

func latestEvidenceVersion(ctx context.Context, tx pgx.Tx, carrier, serialNumber string) (int64, error) {
	query, args, err := storage.Psql.
		Select("COALESCE(MAX(evidence_version), 0)").
		From("device_access_evidence").
		Where(sq.Eq{"carrier": carrier, "serial_number": serialNumber}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build latest evidence version query: %w", err)
	}
	var version int64
	if err := tx.QueryRow(ctx, query, args...).Scan(&version); err != nil {
		return 0, fmt.Errorf("load latest evidence version: %w", err)
	}
	return version, nil
}

func loadEvidenceVersion(
	ctx context.Context,
	tx pgx.Tx,
	carrier string,
	serialNumber string,
	version int64,
) ([]EvidenceRecord, error) {
	if version <= 0 {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select(
			"evidence_type", "evidence_status", "normalized_value", "value_hash", "source",
			"task_id", "observed_at", "expires_at",
		).
		From("device_access_evidence").
		Where(sq.Eq{"carrier": carrier, "serial_number": serialNumber, "evidence_version": version}).
		OrderBy("evidence_type ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build evidence version query: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query evidence version %d: %w", version, err)
	}
	defer rows.Close()
	var records []EvidenceRecord
	for rows.Next() {
		var record EvidenceRecord
		if err := rows.Scan(
			&record.Type, &record.Status, &record.NormalizedValue, &record.ValueHash,
			&record.Source, &record.TaskID, &record.ObservedAt, &record.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan evidence version %d: %w", version, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evidence version %d: %w", version, err)
	}
	return records, nil
}

func changedAgainstBase(base, proposed []EvidenceRecord) []EvidenceRecord {
	baseByType := make(map[ConditionType]EvidenceRecord, len(base))
	for _, record := range base {
		baseByType[record.Type] = record
	}
	changed := make([]EvidenceRecord, 0, len(proposed))
	for _, record := range proposed {
		previous, exists := baseByType[record.Type]
		if !exists || evidenceRecordDiffers(previous, record) {
			changed = append(changed, record)
		}
	}
	return changed
}

func evidenceRecordDiffers(left, right EvidenceRecord) bool {
	leftStatus, rightStatus := left.Status, right.Status
	if leftStatus == "" {
		leftStatus = EvidenceStatusAvailable
	}
	if rightStatus == "" {
		rightStatus = EvidenceStatusAvailable
	}
	return leftStatus != rightStatus ||
		left.ValueHash != right.ValueHash ||
		left.Source != right.Source ||
		left.ObservedAt != right.ObservedAt ||
		!equalUUIDPointers(left.TaskID, right.TaskID) ||
		!equalTimePointers(left.ExpiresAt, right.ExpiresAt)
}

func equalUUIDPointers(left, right *uuid.UUID) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func equalTimePointers(left, right *time.Time) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Equal(*right)
}

func (r *PgRepository) SaveDecision(ctx context.Context, change DecisionChange) (SavedDecision, error) {
	if err := validateDecisionChange(change); err != nil {
		return SavedDecision{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SavedDecision{}, fmt.Errorf("begin save decision transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// The state row does not exist for a first decision, so it cannot serialize
	// concurrent Inform/probe callbacks. Every decision has already observed a
	// candidate; lock that stable identity row before the idempotency and version
	// checks so only one decision version can be created at a time.
	if err := lockEvidenceIdentity(ctx, tx, change.Carrier, change.SerialNumber); err != nil {
		return SavedDecision{}, err
	}

	saved, found, err := findSavedDecision(ctx, tx, change)
	if err != nil {
		return SavedDecision{}, err
	}
	if found {
		if err := tx.Commit(ctx); err != nil {
			return SavedDecision{}, fmt.Errorf("commit idempotent save decision transaction: %w", err)
		}
		return saved, nil
	}

	current, err := lockAccessState(ctx, tx, change.Carrier, change.SerialNumber)
	if err != nil {
		return SavedDecision{}, err
	}
	if current.DecisionVersion != change.ExpectedDecisionVersion {
		return SavedDecision{}, fmt.Errorf(
			"save decision expected version %d but found %d: %w",
			change.ExpectedDecisionVersion,
			current.DecisionVersion,
			ErrDecisionVersionConflict,
		)
	}

	if change.Evidence != nil {
		if err := insertEvidenceRecords(ctx, tx, *change.Evidence); err != nil {
			return SavedDecision{}, err
		}
	}

	nextVersion := current.DecisionVersion + 1
	if err := upsertAccessState(ctx, tx, current.ID, change, nextVersion); err != nil {
		return SavedDecision{}, err
	}

	decisionID := uuid.New()
	if err := insertDecision(ctx, tx, decisionID, current.State, nextVersion, change); err != nil {
		return SavedDecision{}, err
	}
	for _, check := range change.Decision.Checks {
		if err := insertDecisionCheck(ctx, tx, decisionID, check); err != nil {
			return SavedDecision{}, err
		}
	}
	if err := insertOutboxEvent(ctx, tx, decisionID, nextVersion, change, change.Outbox); err != nil {
		return SavedDecision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SavedDecision{}, fmt.Errorf("commit save decision transaction: %w", err)
	}
	return SavedDecision{
		ID:              decisionID,
		State:           change.Decision.State,
		DecisionVersion: nextVersion,
		OccurredAt:      change.OccurredAt,
	}, nil
}

func (r *PgRepository) FindDevice(
	ctx context.Context,
	carrier string,
	serialNumber string,
) (*AssetDeviceRecord, error) {
	if err := validateIdentity(carrier, serialNumber); err != nil {
		return nil, err
	}
	query, args, err := buildAssetDeviceQuery(carrier, serialNumber)
	if err != nil {
		return nil, err
	}
	var record AssetDeviceRecord
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&record.ID,
		&record.Carrier,
		&record.SerialNumber,
		&record.OUI,
		&record.ProductClass,
		&record.SoftwareVersion,
		&record.GroupID,
		&record.SiteID,
		&record.LifecycleState,
		&record.IsOnline,
		&record.LastInformAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find asset device: %w", err)
	}
	return &record, nil
}

func (r *PgRepository) FindRegistration(
	ctx context.Context,
	carrier string,
	serialNumber string,
) (*AssetRegistrationRecord, error) {
	if err := validateIdentity(carrier, serialNumber); err != nil {
		return nil, err
	}
	query, args, err := buildAssetRegistrationQuery(carrier, serialNumber)
	if err != nil {
		return nil, err
	}
	var record AssetRegistrationRecord
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&record.ID,
		&record.Carrier,
		&record.SerialNumber,
		&record.DeviceID,
		&record.GroupID,
		&record.SiteName,
		&record.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find asset registration: %w", err)
	}
	return &record, nil
}

func (r *PgRepository) ExistsUnderOtherCarrier(
	ctx context.Context,
	carrier string,
	serialNumber string,
) (bool, error) {
	if err := validateIdentity(carrier, serialNumber); err != nil {
		return false, err
	}
	query, args, err := buildOtherCarrierQuery(carrier, serialNumber)
	if err != nil {
		return false, err
	}
	var exists bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check asset under other carrier: %w", err)
	}
	return exists, nil
}

func (r *PgRepository) ResolveCarrier(ctx context.Context, serialNumber string) (string, error) {
	serialNumber = strings.TrimSpace(serialNumber)
	if serialNumber == "" {
		return "", ErrSerialNumberRequired
	}

	// Task admission is authorized by access-control authority data, not by a
	// default carrier or a possibly stale business asset. A first access probe
	// has a candidate but no state yet; accepted devices have both.
	carriers := make(map[string]struct{}, 2)
	for _, table := range []string{"device_access_states", "device_access_candidates"} {
		query, args, err := storage.Psql.
			Select("COUNT(DISTINCT carrier)", "MIN(carrier)").
			From(table).
			Where(sq.Eq{"serial_number": serialNumber}).
			ToSql()
		if err != nil {
			return "", fmt.Errorf("build carrier resolution query for %s: %w", table, err)
		}
		var count int64
		var carrier *string
		if err := r.db.QueryRow(ctx, query, args...).Scan(&count, &carrier); err != nil {
			return "", fmt.Errorf("resolve carrier from %s: %w", table, err)
		}
		if count > 1 {
			return "", ErrCarrierAmbiguous
		}
		if count == 1 && carrier != nil && strings.TrimSpace(*carrier) != "" {
			carriers[strings.TrimSpace(*carrier)] = struct{}{}
		}
	}
	switch len(carriers) {
	case 0:
		return "", ErrCarrierNotFound
	case 1:
		for carrier := range carriers {
			return carrier, nil
		}
	default:
		return "", ErrCarrierAmbiguous
	}
	return "", ErrCarrierNotFound
}

func buildAssetDeviceQuery(carrier, serialNumber string) (string, []any, error) {
	return storage.Psql.
		Select(
			"d.id", "d.carrier", "d.serial_number", "d.oui", "COALESCE(d.product_class, '')",
			"COALESCE(d.firmware_version, '')",
			"(SELECT dgm.group_id FROM device_group_members dgm WHERE dgm.device_id = d.id ORDER BY dgm.added_at DESC LIMIT 1)",
			"COALESCE(d.site_id, '')", "d.lifecycle_state", "d.is_online", "d.last_inform_at",
		).
		From("devices d").
		Where(sq.Eq{"d.carrier": carrier}).
		Where(sq.Eq{"d.serial_number": serialNumber}).
		Where("d.deleted_at IS NULL").
		Limit(1).
		ToSql()
}

func buildAssetRegistrationQuery(carrier, serialNumber string) (string, []any, error) {
	return storage.Psql.
		Select(
			"id", "carrier", "serial_number", "device_id", "group_id",
			"COALESCE(site_name, '')", "status",
		).
		From("device_registrations").
		Where(sq.Eq{"carrier": carrier}).
		Where(sq.Eq{"serial_number": serialNumber}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
}

func buildOtherCarrierQuery(carrier, serialNumber string) (string, []any, error) {
	return storage.Psql.
		Select(`EXISTS (
				SELECT 1
				FROM devices d, requested_identity requested
				WHERE d.carrier <> requested.carrier
				  AND d.serial_number = requested.serial_number
				  AND d.deleted_at IS NULL
			) OR EXISTS (
				SELECT 1
				FROM device_registrations registration, requested_identity requested
				WHERE registration.carrier <> requested.carrier
				  AND registration.serial_number = requested.serial_number
			)`).
		Prefix(
			"WITH requested_identity AS (SELECT ?::varchar AS carrier, ?::varchar AS serial_number)",
			carrier,
			serialNumber,
		).
		ToSql()
}

func validateIdentity(carrier, serialNumber string) error {
	if strings.TrimSpace(carrier) == "" {
		return ErrCarrierRequired
	}
	if strings.TrimSpace(serialNumber) == "" {
		return ErrSerialNumberRequired
	}
	return nil
}

func validateEvidenceBatch(evidence EvidenceBatch) error {
	if err := validateIdentity(evidence.Carrier, evidence.SerialNumber); err != nil {
		return err
	}
	if evidence.Version <= 0 {
		return fmt.Errorf("evidence version must be positive")
	}
	return nil
}

func validateDecisionChange(change DecisionChange) error {
	if err := validateIdentity(change.Carrier, change.SerialNumber); err != nil {
		return err
	}
	if change.TriggerEventID == "" {
		return fmt.Errorf("trigger event id is required")
	}
	if change.Outbox.EventKey == "" {
		return fmt.Errorf("outbox event key is required")
	}
	if change.Evidence != nil {
		if err := validateEvidenceBatch(*change.Evidence); err != nil {
			return fmt.Errorf("validate decision evidence: %w", err)
		}
		if change.Evidence.Carrier != change.Carrier ||
			change.Evidence.SerialNumber != change.SerialNumber ||
			change.Evidence.Version != change.EvidenceVersion {
			return fmt.Errorf("decision evidence identity or version mismatch")
		}
	}
	return nil
}

func findSavedDecision(
	ctx context.Context,
	tx pgx.Tx,
	change DecisionChange,
) (SavedDecision, bool, error) {
	query, args, err := storage.Psql.
		Select("id", "new_state", "decision_version", "occurred_at").
		From("device_access_decisions").
		Where(sq.Eq{
			"carrier":          change.Carrier,
			"serial_number":    change.SerialNumber,
			"trigger_event_id": change.TriggerEventID,
		}).
		ToSql()
	if err != nil {
		return SavedDecision{}, false, fmt.Errorf("build idempotent decision query: %w", err)
	}
	var saved SavedDecision
	err = tx.QueryRow(ctx, query, args...).Scan(
		&saved.ID,
		&saved.State,
		&saved.DecisionVersion,
		&saved.OccurredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedDecision{}, false, nil
	}
	if err != nil {
		return SavedDecision{}, false, fmt.Errorf("load idempotent decision: %w", err)
	}
	return saved, true, nil
}

func lockAccessState(
	ctx context.Context,
	tx pgx.Tx,
	carrier string,
	serialNumber string,
) (AccessStateProjection, error) {
	query, args, err := storage.Psql.
		Select(
			"id", "device_id", "candidate_id", "state", "decision_version",
			"evidence_version", "policy_version_id", "normal_tasks_frozen",
		).
		From("device_access_states").
		Where(sq.Eq{"carrier": carrier}).
		Where(sq.Eq{"serial_number": serialNumber}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return AccessStateProjection{}, fmt.Errorf("build access state lock query: %w", err)
	}
	var state AccessStateProjection
	err = tx.QueryRow(ctx, query, args...).Scan(
		&state.ID,
		&state.DeviceID,
		&state.CandidateID,
		&state.State,
		&state.DecisionVersion,
		&state.EvidenceVersion,
		&state.PolicyVersionID,
		&state.NormalTasksFrozen,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccessStateProjection{
			ID:           uuid.New(),
			Carrier:      carrier,
			SerialNumber: serialNumber,
		}, nil
	}
	if err != nil {
		return AccessStateProjection{}, fmt.Errorf("lock device access state: %w", err)
	}
	return state, nil
}

func insertEvidenceRecords(ctx context.Context, tx pgx.Tx, evidence EvidenceBatch) error {
	for _, record := range evidence.Records {
		status := record.Status
		if status == "" {
			status = EvidenceStatusAvailable
		}
		query, args, err := storage.Psql.Insert("device_access_evidence").
			Columns(
				"carrier", "serial_number", "evidence_version", "evidence_type",
				"evidence_status", "normalized_value", "value_hash", "source", "task_id", "observed_at", "expires_at",
			).
			Values(
				evidence.Carrier, evidence.SerialNumber, evidence.Version, record.Type,
				status, record.NormalizedValue, record.ValueHash, record.Source, record.TaskID,
				record.ObservedAt, record.ExpiresAt,
			).
			ToSql()
		if err != nil {
			return fmt.Errorf("build evidence insert: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert device access evidence: %w", err)
		}
	}
	return nil
}

func upsertAccessState(
	ctx context.Context,
	tx pgx.Tx,
	stateID uuid.UUID,
	change DecisionChange,
	nextVersion int64,
) error {
	query, args, err := storage.Psql.Insert("device_access_states").
		Columns(
			"id", "carrier", "serial_number", "device_id", "candidate_id", "state",
			"effective_decision", "reason_code", "policy_version_id", "evidence_version",
			"decision_version", "normal_tasks_frozen", "last_decided_at", "updated_at",
		).
		Values(
			stateID, change.Carrier, change.SerialNumber, change.DeviceID, change.CandidateID,
			change.Decision.State, change.Decision.EffectiveAction, change.Decision.ReasonCode,
			change.PolicyVersionID, change.EvidenceVersion, nextVersion, change.Decision.FreezeNormal,
			change.OccurredAt, change.OccurredAt,
		).
		Suffix(`ON CONFLICT (carrier,serial_number) DO UPDATE SET
			device_id = EXCLUDED.device_id,
			candidate_id = EXCLUDED.candidate_id,
			state = EXCLUDED.state,
			effective_decision = EXCLUDED.effective_decision,
			reason_code = EXCLUDED.reason_code,
			policy_version_id = EXCLUDED.policy_version_id,
			evidence_version = EXCLUDED.evidence_version,
			decision_version = EXCLUDED.decision_version,
			normal_tasks_frozen = EXCLUDED.normal_tasks_frozen,
			last_decided_at = EXCLUDED.last_decided_at,
			updated_at = EXCLUDED.updated_at
			WHERE device_access_states.decision_version = ?`, change.ExpectedDecisionVersion).
		ToSql()
	if err != nil {
		return fmt.Errorf("build access state upsert: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert device access state: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("upsert device access state: %w", ErrDecisionVersionConflict)
	}
	return nil
}

func insertDecision(
	ctx context.Context,
	tx pgx.Tx,
	decisionID uuid.UUID,
	previousState AccessState,
	decisionVersion int64,
	change DecisionChange,
) error {
	matchedRuleID, err := nullableUUID(change.Decision.MatchedRuleID)
	if err != nil {
		return fmt.Errorf("parse matched rule id: %w", err)
	}
	matchedEntryID, err := nullableUUID(change.Decision.MatchedEntryID)
	if err != nil {
		return fmt.Errorf("parse matched list entry id: %w", err)
	}
	query, args, err := storage.Psql.Insert("device_access_decisions").
		Columns(
			"id", "carrier", "serial_number", "device_id", "candidate_id", "trigger_type",
			"trigger_event_id", "previous_state", "new_state", "decision", "reason_code",
			"policy_version_id", "evidence_version", "decision_version", "matched_list_entry_id", "matched_rule_id",
			"occurred_at",
		).
		Values(
			decisionID, change.Carrier, change.SerialNumber, change.DeviceID, change.CandidateID,
			change.TriggerType, change.TriggerEventID, nullableString(string(previousState)), change.Decision.State,
			change.Decision.EffectiveAction, change.Decision.ReasonCode, change.PolicyVersionID,
			change.EvidenceVersion, decisionVersion, matchedEntryID, matchedRuleID, change.OccurredAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build decision insert at version %d: %w", decisionVersion, err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert device access decision: %w", err)
	}
	return nil
}

func insertDecisionCheck(
	ctx context.Context,
	tx pgx.Tx,
	decisionID uuid.UUID,
	check DecisionCheck,
) error {
	query, args, err := storage.Psql.Insert("device_access_decision_checks").
		Columns(
			"id", "decision_id", "check_type", "result", "expected_summary",
			"observed_summary", "evidence_source", "observed_at", "reason_code",
		).
		Values(
			uuid.New(), decisionID, check.CheckType, check.Result,
			nullableString(check.ExpectedSummary), nullableString(check.ObservedSummary),
			nullableString(check.EvidenceSource), nullableTime(check.ObservedAt),
			nullableString(string(check.ReasonCode)),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build decision check insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert device access decision check: %w", err)
	}
	return nil
}

func insertOutboxEvent(
	ctx context.Context,
	tx pgx.Tx,
	decisionID uuid.UUID,
	decisionVersion int64,
	change DecisionChange,
	event OutboxEvent,
) error {
	payload := map[string]any{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode decision outbox payload: %w", err)
	}
	payload["decision_id"] = decisionID
	payload["decision_version"] = decisionVersion
	payload["device_id"] = change.DeviceID
	enrichedPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode decision outbox payload: %w", err)
	}
	query, args, err := storage.Psql.Insert("device_access_outbox").
		Columns(
			"aggregate_type", "aggregate_id", "event_type", "event_key", "payload",
			"next_attempt_at", "created_at", "updated_at",
		).
		Values(
			"device_access_decision", decisionID, event.EventType, event.EventKey,
			enrichedPayload, change.OccurredAt, change.OccurredAt, change.OccurredAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build outbox event insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert device access outbox event: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableAddr(value netip.Addr) any {
	if !value.IsValid() {
		return nil
	}
	return value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func nullableUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse UUID %q: %w", value, err)
	}
	return &parsed, nil
}

var _ Repository = (*PgRepository)(nil)
var _ AssetLookup = (*PgRepository)(nil)

// Keep json.RawMessage's database representation explicit at this boundary.
var _ json.Marshaler = json.RawMessage{}
