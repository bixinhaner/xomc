package deviceaccess

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgPolicyStore struct {
	db storage.DB
}

func NewPgPolicyStore(pool storage.DB) *PgPolicyStore {
	return &PgPolicyStore{db: pool}
}

func (s *PgPolicyStore) CreateDraft(ctx context.Context, version PolicyVersion) (PolicyVersion, error) {
	if s == nil || s.db == nil {
		return PolicyVersion{}, fmt.Errorf("create policy draft: %w", ErrAccessGateDependencyMissing)
	}
	if strings.TrimSpace(version.Carrier) == "" {
		return PolicyVersion{}, ErrCarrierRequired
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("begin create policy draft: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	policySetID, policySetName, err := findOrCreatePolicySet(ctx, tx, version)
	if err != nil {
		return PolicyVersion{}, err
	}
	// A carrier owns one enabled policy set. Draft names are therefore family
	// names, not per-version labels. Return the persisted family name so the API
	// never acknowledges a name that the list/get endpoints cannot display.
	version.Name = policySetName
	versionID := uuid.New()
	var nextVersion int64
	query, args, err := storage.Psql.
		Select("COALESCE(MAX(version), 0) + 1").
		From("device_access_policy_versions").
		Where(sq.Eq{"policy_set_id": policySetID}).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build next policy version query: %w", err)
	}
	if err := tx.QueryRow(ctx, query, args...).Scan(&nextVersion); err != nil {
		return PolicyVersion{}, fmt.Errorf("next policy version: %w", err)
	}
	createdBy, err := nullableUUID(version.CreatedBy)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy creator: %w", err)
	}
	bypassProfiles, err := json.Marshal(version.Policy.BypassProfiles)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("encode bypass profiles: %w", err)
	}
	query, args, err = storage.Psql.Insert("device_access_policy_versions").
		Columns("id", "policy_set_id", "version", "status", "default_action", "failure_mode", "collection_timeout_seconds", "bypass_profiles", "content_hash", "impact_summary", "created_by").
		Values(versionID, policySetID, nextVersion, PolicyVersionDraft, version.Policy.DefaultAction, version.Policy.FailureMode, int64(version.Policy.CollectionTimeout/time.Second), bypassProfiles, version.ContentHash, sq.Expr("'{}'::jsonb"), createdBy).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy version insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("insert policy version: %w", err)
	}
	if err := insertCompiledPolicy(ctx, tx, versionID, version.Policy); err != nil {
		return PolicyVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PolicyVersion{}, fmt.Errorf("commit create policy draft: %w", err)
	}
	version.ID = versionID.String()
	version.Version = nextVersion
	version.Status = PolicyVersionDraft
	version.Policy.VersionID = version.ID
	return version, nil
}

func (s *PgPolicyStore) UpdateDraft(ctx context.Context, version PolicyVersion) (PolicyVersion, error) {
	if s == nil || s.db == nil {
		return PolicyVersion{}, fmt.Errorf("update policy draft: %w", ErrAccessGateDependencyMissing)
	}
	versionID, err := uuid.Parse(strings.TrimSpace(version.ID))
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy version id: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("begin policy draft update: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	query, args, err := storage.Psql.Select("pv.status", "ps.carrier").From("device_access_policy_versions pv").
		Join("device_access_policy_sets ps ON ps.id = pv.policy_set_id").Where(sq.Eq{"pv.id": versionID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy draft lock: %w", err)
	}
	var status PolicyVersionStatus
	var carrier string
	if err := tx.QueryRow(ctx, query, args...).Scan(&status, &carrier); err != nil {
		return PolicyVersion{}, fmt.Errorf("lock policy draft: %w", err)
	}
	if carrier != version.Carrier {
		return PolicyVersion{}, ErrPolicyCarrierScope
	}
	if status != PolicyVersionDraft {
		return PolicyVersion{}, ErrPolicyVersionImmutable
	}
	if err := syncCompiledPolicy(ctx, tx, versionID, version.Policy); err != nil {
		return PolicyVersion{}, err
	}
	bypassProfiles, err := json.Marshal(version.Policy.BypassProfiles)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("encode bypass profiles: %w", err)
	}
	query, args, err = storage.Psql.Update("device_access_policy_versions").
		Set("default_action", version.Policy.DefaultAction).
		Set("failure_mode", version.Policy.FailureMode).
		Set("collection_timeout_seconds", int64(version.Policy.CollectionTimeout/time.Second)).
		Set("bypass_profiles", bypassProfiles).
		Set("content_hash", version.ContentHash).Where(sq.Eq{"id": versionID, "status": PolicyVersionDraft}).ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy draft metadata update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("update policy draft metadata: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PolicyVersion{}, fmt.Errorf("commit policy draft update: %w", err)
	}
	version.Status = PolicyVersionDraft
	version.Policy.VersionID = version.ID
	return version, nil
}

func syncCompiledPolicy(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, policy CompiledPolicy) error {
	ruleIDs := make([]uuid.UUID, 0, len(policy.Rules))
	for order, rule := range policy.Rules {
		ruleID, err := uuid.Parse(rule.ID)
		if err != nil {
			return fmt.Errorf("parse policy rule id %q: %w", rule.ID, err)
		}
		ruleIDs = append(ruleIDs, ruleID)
		scope, err := json.Marshal(rule.SerialScope)
		if err != nil {
			return fmt.Errorf("encode rule scope: %w", err)
		}
		query, args, err := storage.Psql.Update("device_access_rules").
			Set("name", rule.Name).Set("enabled", rule.Enabled).Set("priority", rule.Priority).
			Set("serial_scope_type", rule.SerialScope.Type).Set("serial_scope", scope).Set("created_order", order).
			Where(sq.Eq{"id": ruleID, "policy_version_id": versionID}).ToSql()
		if err != nil {
			return fmt.Errorf("build policy rule update: %w", err)
		}
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("update policy rule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			query, args, err = storage.Psql.Insert("device_access_rules").
				Columns("id", "policy_version_id", "name", "enabled", "priority", "serial_scope_type", "serial_scope", "created_order").
				Values(ruleID, versionID, rule.Name, rule.Enabled, rule.Priority, rule.SerialScope.Type, scope, order).ToSql()
			if err != nil {
				return fmt.Errorf("build policy rule insert: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return fmt.Errorf("insert policy rule: %w", err)
			}
		}
		conditionIDs := make([]uuid.UUID, 0, len(rule.Conditions))
		for _, condition := range rule.Conditions {
			conditionID, err := uuid.Parse(condition.ID)
			if err != nil {
				return fmt.Errorf("parse policy condition id %q: %w", condition.ID, err)
			}
			conditionIDs = append(conditionIDs, conditionID)
			expected, err := conditionExpectedJSON(condition)
			if err != nil {
				return fmt.Errorf("encode policy condition %s: %w", condition.ID, err)
			}
			query, args, err := storage.Psql.Update("device_access_conditions").
				Set("condition_type", condition.Type).Set("operator", condition.Operator).Set("expected_value", expected).
				Set("required", condition.Required).Set("evidence_ttl_seconds", int64(condition.EvidenceTTL/time.Second)).
				Where(sq.Eq{"id": conditionID, "rule_id": ruleID}).ToSql()
			if err != nil {
				return fmt.Errorf("build policy condition update: %w", err)
			}
			tag, err := tx.Exec(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("update policy condition: %w", err)
			}
			if tag.RowsAffected() == 0 {
				query, args, err = storage.Psql.Insert("device_access_conditions").
					Columns("id", "rule_id", "condition_type", "operator", "expected_value", "required", "evidence_ttl_seconds").
					Values(conditionID, ruleID, condition.Type, condition.Operator, expected, condition.Required, int64(condition.EvidenceTTL/time.Second)).ToSql()
				if err != nil {
					return fmt.Errorf("build policy condition insert: %w", err)
				}
				if _, err := tx.Exec(ctx, query, args...); err != nil {
					return fmt.Errorf("insert policy condition: %w", err)
				}
			}
		}
		deleteConditions := storage.Psql.Delete("device_access_conditions").Where(sq.Eq{"rule_id": ruleID})
		if len(conditionIDs) > 0 {
			deleteConditions = deleteConditions.Where(sq.NotEq{"id": conditionIDs})
		}
		query, args, err = deleteConditions.ToSql()
		if err != nil {
			return fmt.Errorf("build removed condition delete: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("delete removed policy conditions: %w", err)
		}
	}
	deleteRules := storage.Psql.Delete("device_access_rules").Where(sq.Eq{"policy_version_id": versionID})
	if len(ruleIDs) > 0 {
		deleteRules = deleteRules.Where(sq.NotEq{"id": ruleIDs})
	}
	query, args, err := deleteRules.ToSql()
	if err != nil {
		return fmt.Errorf("build removed rule delete: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete removed policy rules: %w", err)
	}
	return nil
}

func (s *PgPolicyStore) GetVersion(ctx context.Context, versionID string) (PolicyVersion, error) {
	id, err := uuid.Parse(strings.TrimSpace(versionID))
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy version id: %w", err)
	}
	if s == nil || s.db == nil {
		return PolicyVersion{}, fmt.Errorf("get policy version: %w", ErrAccessGateDependencyMissing)
	}
	var version PolicyVersion
	var status string
	var createdBy, publishedBy sql.NullString
	query, args, err := storage.Psql.
		Select(
			"pv.id", "ps.carrier", "ps.name", "pv.version", "pv.status", "pv.default_action", "pv.failure_mode", "pv.collection_timeout_seconds", "pv.bypass_profiles", "pv.content_hash",
			"COALESCE(pv.created_by::text, '')", "COALESCE(pv.published_by::text, '')", "pv.published_at",
		).
		From("device_access_policy_versions pv").
		Join("device_access_policy_sets ps ON ps.id = pv.policy_set_id").
		Where(sq.Eq{"pv.id": id}).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy version query: %w", err)
	}
	var collectionTimeoutSeconds int64
	var bypassProfiles []byte
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&version.ID, &version.Carrier, &version.Name, &version.Version, &status, &version.Policy.DefaultAction, &version.Policy.FailureMode, &collectionTimeoutSeconds, &bypassProfiles, &version.ContentHash,
		&createdBy, &publishedBy, &version.PublishedAt,
	); err != nil {
		return PolicyVersion{}, fmt.Errorf("get policy version: %w", err)
	}
	version.Status = PolicyVersionStatus(status)
	version.CreatedBy = createdBy.String
	version.PublishedBy = publishedBy.String
	version.Policy.VersionID = version.ID
	version.Policy.CollectionTimeout = time.Duration(collectionTimeoutSeconds) * time.Second
	if err := json.Unmarshal(bypassProfiles, &version.Policy.BypassProfiles); err != nil {
		return PolicyVersion{}, fmt.Errorf("decode policy version bypass profiles: %w", err)
	}
	provider := newPgPolicyProviderWithDB(s.db)
	if err := provider.loadRules(ctx, id, &version.Policy); err != nil {
		return PolicyVersion{}, fmt.Errorf("load policy version rules: %w", err)
	}
	if err := provider.loadConditions(ctx, id, &version.Policy); err != nil {
		return PolicyVersion{}, fmt.Errorf("load policy version conditions: %w", err)
	}
	return version, nil
}

func (s *PgPolicyStore) PreviousVersion(ctx context.Context, versionID string) (PolicyVersion, error) {
	if s == nil || s.db == nil {
		return PolicyVersion{}, fmt.Errorf("get previous policy version: %w", ErrAccessGateDependencyMissing)
	}
	id, err := uuid.Parse(strings.TrimSpace(versionID))
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy version id: %w", err)
	}
	query, args, err := storage.Psql.Select("previous.id").
		From("device_access_policy_versions current").
		Join("device_access_policy_versions previous ON previous.policy_set_id = current.policy_set_id AND previous.version < current.version").
		Where(sq.Eq{"current.id": id}).OrderBy("previous.version DESC").Limit(1).ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build previous policy version query: %w", err)
	}
	var previousID uuid.UUID
	if err := s.db.QueryRow(ctx, query, args...).Scan(&previousID); err != nil {
		return PolicyVersion{}, fmt.Errorf("query previous policy version: %w", err)
	}
	return s.GetVersion(ctx, previousID.String())
}

func (s *PgPolicyStore) DeleteDraft(ctx context.Context, versionID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("delete policy draft: %w", ErrAccessGateDependencyMissing)
	}
	id, err := uuid.Parse(strings.TrimSpace(versionID))
	if err != nil {
		return fmt.Errorf("parse policy version id: %w", err)
	}
	query, args, err := storage.Psql.Delete("device_access_policy_versions").
		Where(sq.Eq{"id": id, "status": PolicyVersionDraft}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build policy draft delete: %w", err)
	}
	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete policy draft row: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPolicyVersionImmutable
	}
	return nil
}

func (s *PgPolicyStore) Publish(ctx context.Context, versionID, subjectID string) (PolicyVersion, error) {
	if s == nil || s.db == nil {
		return PolicyVersion{}, fmt.Errorf("publish policy version: %w", ErrAccessGateDependencyMissing)
	}
	id, err := uuid.Parse(strings.TrimSpace(versionID))
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy version id: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("begin publish policy: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var policySetID uuid.UUID
	var carrier string
	var status string
	query, args, err := storage.Psql.
		Select("pv.policy_set_id", "ps.carrier", "pv.status").
		From("device_access_policy_versions pv").
		Join("device_access_policy_sets ps ON ps.id = pv.policy_set_id").
		Where(sq.Eq{"pv.id": id}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy version lock query: %w", err)
	}
	if err := tx.QueryRow(ctx, query, args...).Scan(&policySetID, &carrier, &status); err != nil {
		return PolicyVersion{}, fmt.Errorf("lock policy version: %w", err)
	}
	if status != string(PolicyVersionDraft) {
		return PolicyVersion{}, fmt.Errorf("publish policy version: %w", ErrPolicyVersionImmutable)
	}
	query, args, err = storage.Psql.Update("device_access_policy_versions").
		Set("status", PolicyVersionRetired).
		Where(sq.Eq{"policy_set_id": policySetID, "status": PolicyVersionPublished}).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build previous policy retirement: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("retire previous policy version: %w", err)
	}
	publishedAt := time.Now().UTC()
	publishedBy, err := nullableUUID(subjectID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("parse policy publisher: %w", err)
	}
	query, args, err = storage.Psql.Update("device_access_policy_versions").
		Set("status", PolicyVersionPublished).
		Set("published_by", publishedBy).
		Set("published_at", publishedAt).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy publish update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("publish policy version: %w", err)
	}
	query, args, err = storage.Psql.Update("device_access_policy_sets").
		Set("active_version_id", id).
		Set("updated_at", publishedAt).
		Where(sq.Eq{"id": policySetID}).
		ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build active policy update: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("activate policy version: %w", err)
	}
	publishEventID := uuid.New()
	publishEvent := PolicyPublishedEvent{
		Carrier: carrier, PolicyVersionID: id.String(), TriggerEventID: publishEventID.String(),
	}
	payload, err := json.Marshal(publishEvent)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("encode policy published event: %w", err)
	}
	query, args, err = storage.Psql.Insert("device_access_outbox").
		Columns("aggregate_type", "aggregate_id", "event_type", "event_key", "payload", "next_attempt_at", "created_at", "updated_at").
		Values("device_access_policy", id, event.SubjectDeviceAccessPolicyPublished,
			"device-access-policy-published:"+id.String(), payload, publishedAt, publishedAt, publishedAt).
		Suffix("ON CONFLICT (event_key) DO NOTHING").ToSql()
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("build policy published outbox: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return PolicyVersion{}, fmt.Errorf("insert policy published outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PolicyVersion{}, fmt.Errorf("commit publish policy: %w", err)
	}
	return PolicyVersion{ID: id.String(), Carrier: carrier, Status: PolicyVersionPublished, PublishedBy: subjectID, PublishedAt: &publishedAt}, nil
}

func (s *PgPolicyStore) UpsertListEntryAndQueue(ctx context.Context, carrier string, entry CompiledListEntry) error {
	return s.UpsertListEntriesAndQueue(ctx, carrier, []CompiledListEntry{entry})
}

func (s *PgPolicyStore) UpsertListEntriesAndQueue(ctx context.Context, carrier string, entries []CompiledListEntry) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("upsert access list entries: %w", ErrAccessGateDependencyMissing)
	}
	if strings.TrimSpace(carrier) == "" {
		return ErrCarrierRequired
	}
	if len(entries) == 0 {
		return fmt.Errorf("upsert access list entries: %w", ErrInvalidAccessInput)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin access list update: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	now := time.Now().UTC()
	for _, entry := range entries {
		query, args, err := buildListEntryUpsert(carrier, entry)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert access list entry %s: %w", entry.IdentityValue, err)
		}
		if err := insertListReevaluationOutbox(ctx, tx, carrier, entry.IdentityValue, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit access list update: %w", err)
	}
	return nil
}

func insertListReevaluationOutbox(ctx context.Context, tx pgx.Tx, carrier, serialNumber string, now time.Time) error {
	eventID := uuid.New()
	request := ReevaluationRequest{
		Carrier: carrier, SerialNumber: serialNumber, TriggerType: TriggerAccessListChanged,
		TriggerEventID: eventID.String(),
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode reevaluation for %s: %w", serialNumber, err)
	}
	query, args, err := storage.Psql.Insert("device_access_outbox").
		Columns("aggregate_type", "aggregate_id", "event_type", "event_key", "payload", "next_attempt_at", "created_at", "updated_at").
		Values(
			"device_access_reevaluation", eventID, event.SubjectDeviceAccessReevaluationRequested,
			fmt.Sprintf("device-access-reevaluation:%s:%s", serialNumber, eventID), payload, now, now, now,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reevaluation outbox insert for %s: %w", serialNumber, err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert reevaluation outbox for %s: %w", serialNumber, err)
	}
	return nil
}

func (s *PgPolicyStore) DisableListEntriesAndQueue(
	ctx context.Context,
	carrier string,
	entryType ListEntryType,
	serialNumbers []string,
	reason string,
) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("disable access list entries: %w", ErrAccessGateDependencyMissing)
	}
	if strings.TrimSpace(carrier) == "" {
		return ErrCarrierRequired
	}
	if len(serialNumbers) == 0 {
		return fmt.Errorf("disable access list entries: %w", ErrInvalidAccessInput)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin access list batch disable: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	now := time.Now().UTC()
	query, args, err := buildListBatchDisable(carrier, entryType, serialNumbers, reason, now)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("disable access list entries: %w", err)
	}
	disabled := make(map[string]struct{}, len(serialNumbers))
	for rows.Next() {
		var serialNumber string
		if err := rows.Scan(&serialNumber); err != nil {
			rows.Close()
			return fmt.Errorf("scan disabled access list entry: %w", err)
		}
		disabled[serialNumber] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate disabled access list entries: %w", err)
	}
	rows.Close()
	if len(disabled) != len(serialNumbers) {
		return ErrListBatchConflict
	}
	for _, serialNumber := range serialNumbers {
		if _, ok := disabled[serialNumber]; !ok {
			return ErrListBatchConflict
		}
		if err := insertListReevaluationOutbox(ctx, tx, carrier, serialNumber, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit access list batch disable: %w", err)
	}
	return nil
}

func buildListBatchDisable(
	carrier string,
	entryType ListEntryType,
	serialNumbers []string,
	reason string,
	now time.Time,
) (string, []any, error) {
	query, args, err := storage.Psql.Update("device_access_list_entries").
		Set("status", ListEntryStatusDisabled).
		Set("reason_code", "batch_disabled").
		Set("reason", reason).
		Set("updated_at", now).
		Where(sq.Eq{
			"carrier": carrier, "entry_type": entryType, "identity_type": IdentityTypeSerialNumber,
			"identity_value": serialNumbers, "status": ListEntryStatusActive,
		}).
		Suffix("RETURNING identity_value").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build access list batch disable: %w", err)
	}
	return query, args, nil
}

func buildListEntryUpsert(carrier string, entry CompiledListEntry) (string, []any, error) {
	id, err := uuid.Parse(entry.ID)
	if err != nil {
		id = uuid.New()
	}
	query, args, err := storage.Psql.Insert("device_access_list_entries").
		Columns(
			"id", "carrier", "entry_type", "identity_type", "identity_value",
			"reason_code", "reason", "valid_from", "valid_until", "status",
		).
		Values(
			id, carrier, entry.Type, entry.IdentityType, entry.IdentityValue,
			"managed_by_policy", nullableString(entry.Reason), sq.Expr("COALESCE(?, now())", entry.ValidFrom), entry.ValidUntil, entry.Status,
		).
		Suffix(`ON CONFLICT (carrier,entry_type,identity_type,identity_value) DO UPDATE SET
			reason_code = EXCLUDED.reason_code,
			reason = EXCLUDED.reason,
			valid_from = EXCLUDED.valid_from,
			valid_until = EXCLUDED.valid_until,
			status = EXCLUDED.status,
			updated_at = now()`).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build access list entry upsert: %w", err)
	}
	return query, args, nil
}

func findOrCreatePolicySet(ctx context.Context, tx pgx.Tx, version PolicyVersion) (uuid.UUID, string, error) {
	var id uuid.UUID
	var name string
	query, args, err := storage.Psql.
		Select("id", "name").
		From("device_access_policy_sets").
		Where(sq.Eq{"carrier": version.Carrier, "enabled": true}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("build policy set query: %w", err)
	}
	err = tx.QueryRow(ctx, query, args...).Scan(&id, &name)
	if err == nil {
		return id, name, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, "", fmt.Errorf("find policy set: %w", err)
	}
	id = uuid.New()
	name = strings.TrimSpace(version.Name)
	query, args, err = storage.Psql.Insert("device_access_policy_sets").
		Columns("id", "name", "carrier", "enabled").
		Values(id, name, version.Carrier, true).
		ToSql()
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("build policy set insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return uuid.Nil, "", fmt.Errorf("insert policy set: %w", err)
	}
	return id, name, nil
}

func insertCompiledPolicy(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, policy CompiledPolicy) error {
	for order, rule := range policy.Rules {
		ruleID := parseOrNewUUID(rule.ID)
		scope, err := json.Marshal(rule.SerialScope)
		if err != nil {
			return fmt.Errorf("encode rule scope: %w", err)
		}
		query, args, err := storage.Psql.Insert("device_access_rules").
			Columns("id", "policy_version_id", "name", "enabled", "priority", "serial_scope_type", "serial_scope", "created_order").
			Values(ruleID, versionID, rule.Name, rule.Enabled, rule.Priority, rule.SerialScope.Type, scope, order).
			ToSql()
		if err != nil {
			return fmt.Errorf("build policy rule insert: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert policy rule: %w", err)
		}
		for _, condition := range rule.Conditions {
			expected, err := conditionExpectedJSON(condition)
			if err != nil {
				return fmt.Errorf("encode policy condition %s: %w", condition.ID, err)
			}
			query, args, err := storage.Psql.Insert("device_access_conditions").
				Columns("id", "rule_id", "condition_type", "operator", "expected_value", "required", "evidence_ttl_seconds").
				Values(
					parseOrNewUUID(condition.ID), ruleID, condition.Type, condition.Operator,
					expected, condition.Required, int64(condition.EvidenceTTL/time.Second),
				).
				ToSql()
			if err != nil {
				return fmt.Errorf("build policy condition insert: %w", err)
			}
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return fmt.Errorf("insert policy condition: %w", err)
			}
		}
	}
	return nil
}

func conditionExpectedJSON(condition CompiledCondition) ([]byte, error) {
	switch condition.Operator {
	case ConditionOperatorIn:
		return json.Marshal(condition.ExpectedAny)
	case ConditionOperatorIPRange:
		if len(condition.IPRanges) > 0 {
			return json.Marshal(condition.IPRanges)
		}
		return json.Marshal(condition.IPRange)
	case ConditionOperatorWithinRadius:
		return json.Marshal(condition.GeoFence)
	case ConditionOperatorWithinBounds:
		if len(condition.GeoBoundsAny) > 0 {
			return json.Marshal(condition.GeoBoundsAny)
		}
		return json.Marshal(condition.GeoBounds)
	default:
		return json.Marshal(condition.Expected)
	}
}

func parseOrNewUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.New()
	}
	return id
}

var _ PolicyStore = (*PgPolicyStore)(nil)
var _ ListEntryStore = (*PgPolicyStore)(nil)
var _ ListEntryBatchStore = (*PgPolicyStore)(nil)
