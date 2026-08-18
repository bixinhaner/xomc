package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PgPolicyProvider struct {
	db storage.DB
}

func NewPgPolicyProvider(pool *pgxpool.Pool) *PgPolicyProvider {
	return &PgPolicyProvider{db: storage.NewPoolDB(pool)}
}

func newPgPolicyProviderWithDB(db storage.DB) *PgPolicyProvider {
	return &PgPolicyProvider{db: db}
}

func (p *PgPolicyProvider) Load(ctx context.Context, carrier string) (CompiledPolicy, error) {
	carrier = strings.TrimSpace(carrier)
	if carrier == "" {
		return CompiledPolicy{}, ErrCarrierRequired
	}
	if p == nil || p.db == nil {
		return CompiledPolicy{}, fmt.Errorf("load device access policy: %w", ErrAccessGateDependencyMissing)
	}

	versionID, defaultAction, err := p.loadActiveVersion(ctx, carrier)
	policy := CompiledPolicy{DefaultAction: PolicyDefaultActionReject}
	switch {
	case err == nil:
		policy.VersionID = versionID.String()
		policy.DefaultAction = defaultAction
		if err := p.loadRules(ctx, versionID, &policy); err != nil {
			return CompiledPolicy{}, err
		}
		if err := p.loadConditions(ctx, versionID, &policy); err != nil {
			return CompiledPolicy{}, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		// Keep active lists effective before the first policy is published. Devices
		// outside the allow-list follow the legacy no-template behavior and reject.
	default:
		return CompiledPolicy{}, err
	}
	if err := p.loadListEntries(ctx, carrier, &policy); err != nil {
		return CompiledPolicy{}, err
	}
	return policy, nil
}

func (p *PgPolicyProvider) loadActiveVersion(ctx context.Context, carrier string) (uuid.UUID, PolicyDefaultAction, error) {
	query, args, err := storage.Psql.
		Select("pv.id", "pv.default_action").
		From("device_access_policy_sets ps").
		Join("device_access_policy_versions pv ON pv.id = ps.active_version_id").
		Where(sq.Eq{"ps.carrier": carrier, "ps.enabled": true, "pv.status": "published"}).
		ToSql()
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("build active device access policy query: %w", err)
	}
	var versionID uuid.UUID
	var defaultAction PolicyDefaultAction
	if err := p.db.QueryRow(ctx, query, args...).Scan(&versionID, &defaultAction); err != nil {
		return uuid.Nil, "", fmt.Errorf("load active device access policy: %w", err)
	}
	return versionID, defaultAction, nil
}

func (p *PgPolicyProvider) loadRules(ctx context.Context, versionID uuid.UUID, policy *CompiledPolicy) error {
	query, args, err := storage.Psql.
		Select("id", "name", "enabled", "serial_scope_type", "serial_scope").
		From("device_access_rules").
		Where(sq.Eq{"policy_version_id": versionID}).
		OrderBy("created_order ASC", "id ASC").
		ToSql()
	if err != nil {
		return fmt.Errorf("build device access rules query: %w", err)
	}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query device access rules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rule CompiledRule
		var ruleID uuid.UUID
		var scopeType string
		var scopeJSON []byte
		if err := rows.Scan(&ruleID, &rule.Name, &rule.Enabled, &scopeType, &scopeJSON); err != nil {
			return fmt.Errorf("scan device access rule: %w", err)
		}
		rule.ID = ruleID.String()
		rule.SerialScope.Type = SerialScopeType(scopeType)
		var scope SerialScope
		if err := json.Unmarshal(scopeJSON, &scope); err != nil {
			return fmt.Errorf("decode device access rule scope %s: %w", rule.ID, err)
		}
		scope.Type = rule.SerialScope.Type
		rule.SerialScope = scope
		policy.Rules = append(policy.Rules, rule)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate device access rules: %w", err)
	}
	return nil
}

func (p *PgPolicyProvider) loadConditions(ctx context.Context, versionID uuid.UUID, policy *CompiledPolicy) error {
	query, args, err := storage.Psql.
		Select("c.id", "c.rule_id", "c.condition_type", "c.operator", "c.expected_value", "c.required", "c.evidence_ttl_seconds").
		From("device_access_conditions c").
		Join("device_access_rules r ON r.id = c.rule_id").
		Where(sq.Eq{"r.policy_version_id": versionID}).
		OrderBy("r.created_order ASC", "c.id ASC").
		ToSql()
	if err != nil {
		return fmt.Errorf("build device access conditions query: %w", err)
	}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query device access conditions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var condition CompiledCondition
		var conditionID, ruleID uuid.UUID
		var conditionType, operator string
		var expectedJSON []byte
		var ttlSeconds int32
		if err := rows.Scan(&conditionID, &ruleID, &conditionType, &operator, &expectedJSON, &condition.Required, &ttlSeconds); err != nil {
			return fmt.Errorf("scan device access condition: %w", err)
		}
		condition.ID = conditionID.String()
		condition.Type = ConditionType(conditionType)
		condition.Operator = ConditionOperator(operator)
		condition.EvidenceTTL = time.Duration(ttlSeconds) * time.Second
		if err := decodeConditionExpected(&condition, expectedJSON); err != nil {
			return fmt.Errorf("decode device access condition %s: %w", condition.ID, err)
		}
		for index := range policy.Rules {
			if policy.Rules[index].ID == ruleID.String() {
				policy.Rules[index].Conditions = append(policy.Rules[index].Conditions, condition)
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate device access conditions: %w", err)
	}
	return nil
}

func (p *PgPolicyProvider) loadListEntries(ctx context.Context, carrier string, policy *CompiledPolicy) error {
	query, args, err := storage.Psql.
		Select("id", "entry_type", "identity_type", "identity_value", "valid_from", "valid_until", "status").
		From("device_access_list_entries").
		Where(sq.Eq{"carrier": carrier}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build device access list query: %w", err)
	}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query device access list: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry CompiledListEntry
		var entryID uuid.UUID
		if err := rows.Scan(&entryID, &entry.Type, &entry.IdentityType, &entry.IdentityValue, &entry.ValidFrom, &entry.ValidUntil, &entry.Status); err != nil {
			return fmt.Errorf("scan device access list entry: %w", err)
		}
		entry.ID = entryID.String()
		policy.ListEntries = append(policy.ListEntries, entry)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate device access list: %w", err)
	}
	return nil
}

func decodeConditionExpected(condition *CompiledCondition, raw []byte) error {
	switch condition.Operator {
	case ConditionOperatorIn:
		return json.Unmarshal(raw, &condition.ExpectedAny)
	case ConditionOperatorWithinRadius:
		condition.GeoFence = &GeoFence{}
		return json.Unmarshal(raw, condition.GeoFence)
	default:
		if err := json.Unmarshal(raw, &condition.Expected); err == nil {
			return nil
		}
		return fmt.Errorf("expected value must be a string")
	}
}
