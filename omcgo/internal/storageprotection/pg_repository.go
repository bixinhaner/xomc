package storageprotection

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var policyColumns = []string{
	"id", "target_type", "target_id", "write_scope", "enabled",
	"warn_used_percent", "block_used_percent", "recover_used_percent",
	"check_interval_seconds", "unknown_behavior", "current_state",
	"state_observations", "last_observed_ratio", "last_observed_at",
	"last_state_changed_at", "updated_by", "version", "created_at", "updated_at",
}

var eventColumns = []string{
	"policy_id", "target_type", "target_id", "write_scope", "previous_state",
	"new_state", "reason", "observed_ratio", "policy_version", "operator_id", "created_at",
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

func (r *PgRepository) EnsureDefaultPolicy(ctx context.Context) error {
	policy := DefaultPolicy()
	if err := validatePolicy(&policy); err != nil {
		return err
	}
	query, args, err := storage.Psql.Insert("storage_protection_policies").
		Columns("id", "target_type", "target_id", "write_scope", "enabled", "warn_used_percent", "recover_used_percent", "block_used_percent", "check_interval_seconds", "unknown_behavior", "current_state", "updated_by").
		Values(policy.ID, string(policy.TargetType), policy.TargetID, string(policy.WriteScope), policy.Enabled, policy.WarnUsedPercent, policy.RecoverUsedPercent, policy.BlockUsedPercent, policy.CheckIntervalSeconds, string(policy.UnknownBehavior), string(policy.CurrentState), policy.UpdatedBy).
		Suffix("ON CONFLICT (target_type, target_id, write_scope) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build default storage protection policy seed: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ensure default storage protection policy: %w", err)
	}
	return nil
}

func (r *PgRepository) GetEnabledPolicy(ctx context.Context, targetType TargetType, targetID string, scope WriteScope) (*Policy, error) {
	for _, candidateScope := range policyLookupScopes(scope) {
		query, args, err := storage.Psql.Select(policyColumns...).From("storage_protection_policies").
			Where(sq.Eq{"target_type": string(targetType), "target_id": targetID, "write_scope": string(candidateScope), "enabled": true}).
			Limit(1).ToSql()
		if err != nil {
			return nil, fmt.Errorf("build storage protection policy query: %w", err)
		}
		var policy Policy
		if err := scanPolicy(r.pool.QueryRow(ctx, query, args...), &policy); err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("get storage protection policy: %w", err)
		}
		return &policy, nil
	}
	return nil, nil
}

func policyLookupScopes(scope WriteScope) []WriteScope {
	if scope == WriteScopeAll {
		return []WriteScope{WriteScopeAll}
	}
	return []WriteScope{scope, WriteScopeAll}
}

func (r *PgRepository) List(ctx context.Context) ([]Policy, error) {
	query, args, err := storage.Psql.Select(policyColumns...).From("storage_protection_policies").OrderBy("target_type", "target_id", "write_scope").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build storage protection policy list: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list storage protection policies: %w", err)
	}
	defer rows.Close()
	policies := make([]Policy, 0)
	for rows.Next() {
		var policy Policy
		if err := scanPolicy(rows, &policy); err != nil {
			return nil, fmt.Errorf("scan storage protection policy: %w", err)
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage protection policies: %w", err)
	}
	return policies, nil
}

func (r *PgRepository) Save(ctx context.Context, policy *Policy) (*Policy, error) {
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	query, args, err := storage.Psql.Insert("storage_protection_policies").
		Columns("target_type", "target_id", "write_scope", "enabled", "warn_used_percent", "block_used_percent", "recover_used_percent", "check_interval_seconds", "unknown_behavior", "updated_by").
		Values(string(policy.TargetType), policy.TargetID, string(policy.WriteScope), policy.Enabled, policy.WarnUsedPercent, policy.BlockUsedPercent, policy.RecoverUsedPercent, policy.CheckIntervalSeconds, string(policy.UnknownBehavior), policy.UpdatedBy).
		Suffix("ON CONFLICT (target_type, target_id, write_scope) DO UPDATE SET enabled=EXCLUDED.enabled, warn_used_percent=EXCLUDED.warn_used_percent, block_used_percent=EXCLUDED.block_used_percent, recover_used_percent=EXCLUDED.recover_used_percent, check_interval_seconds=EXCLUDED.check_interval_seconds, unknown_behavior=EXCLUDED.unknown_behavior, updated_by=EXCLUDED.updated_by, version=storage_protection_policies.version+1, updated_at=now() RETURNING " + joinColumns(policyColumns)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build save storage protection policy: %w", err)
	}
	var saved Policy
	if err := scanPolicy(r.pool.QueryRow(ctx, query, args...), &saved); err != nil {
		return nil, fmt.Errorf("save storage protection policy: %w", err)
	}
	return &saved, nil
}

func (r *PgRepository) UpdateState(ctx context.Context, policy *Policy) error {
	if policy == nil || policy.ID == "" {
		return fmt.Errorf("update storage protection state: policy id is required")
	}
	query, args, err := storage.Psql.Update("storage_protection_policies").
		Set("current_state", string(policy.CurrentState)).
		Set("state_observations", policy.StateObservations).
		Set("last_observed_ratio", policy.LastObservedRatio).
		Set("last_observed_at", policy.LastObservedAt).
		Set("last_state_changed_at", policy.LastStateChangedAt).
		Set("version", policy.Version).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": policy.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update storage protection state: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update storage protection state: %w", err)
	}
	return nil
}

func (r *PgRepository) RecordEvent(ctx context.Context, event Event) error {
	policyID, err := uuid.Parse(event.PolicyID)
	if err != nil {
		return fmt.Errorf("parse storage protection policy id: %w", err)
	}
	query, args, err := storage.Psql.Insert("storage_protection_events").
		Columns("policy_id", "target_type", "target_id", "write_scope", "previous_state", "new_state", "reason", "observed_ratio", "policy_version", "operator_id").
		Values(policyID, string(event.TargetType), event.TargetID, string(event.WriteScope), nullableState(event.PreviousState), string(event.NewState), event.Reason, event.ObservedRatio, event.PolicyVersion, event.OperatorID).ToSql()
	if err != nil {
		return fmt.Errorf("build storage protection event: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record storage protection event: %w", err)
	}
	return nil
}

func (r *PgRepository) ListEvents(ctx context.Context, targetType TargetType, targetID string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	queryBuilder := storage.Psql.Select(eventColumns...).From("storage_protection_events").OrderBy("created_at DESC").Limit(uint64(limit))
	if targetType != "" {
		queryBuilder = queryBuilder.Where(sq.Eq{"target_type": string(targetType)})
	}
	if targetID != "" {
		queryBuilder = queryBuilder.Where(sq.Eq{"target_id": targetID})
	}
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build storage protection event list: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list storage protection events: %w", err)
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := scanEvent(rows, &event); err != nil {
			return nil, fmt.Errorf("scan storage protection event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage protection events: %w", err)
	}
	return events, nil
}

func (r *PgRepository) CleanupEvents(ctx context.Context, before time.Time, keepLatest int) (int64, error) {
	var deleted int64
	if !before.IsZero() {
		result, err := r.pool.Exec(ctx, "DELETE FROM storage_protection_events WHERE created_at < $1", before)
		if err != nil {
			return 0, fmt.Errorf("delete expired storage protection events: %w", err)
		}
		deleted += result.RowsAffected()
	}
	if keepLatest > 0 {
		result, err := r.pool.Exec(ctx, `
DELETE FROM storage_protection_events
WHERE id IN (
    SELECT id
    FROM storage_protection_events
    ORDER BY created_at DESC, id DESC
    OFFSET $1
)`, keepLatest)
		if err != nil {
			return deleted, fmt.Errorf("delete excess storage protection events: %w", err)
		}
		deleted += result.RowsAffected()
	}
	return deleted, nil
}

func nullableState(state State) any {
	if state == "" {
		return nil
	}
	return string(state)
}

func joinColumns(columns []string) string {
	result := ""
	for i, column := range columns {
		if i > 0 {
			result += ", "
		}
		result += column
	}
	return result
}

type rowScanner interface{ Scan(...any) error }

func scanPolicy(row rowScanner, policy *Policy) error {
	return row.Scan(
		&policy.ID, &policy.TargetType, &policy.TargetID, &policy.WriteScope, &policy.Enabled,
		&policy.WarnUsedPercent, &policy.BlockUsedPercent, &policy.RecoverUsedPercent,
		&policy.CheckIntervalSeconds, &policy.UnknownBehavior, &policy.CurrentState,
		&policy.StateObservations, &policy.LastObservedRatio, &policy.LastObservedAt,
		&policy.LastStateChangedAt, &policy.UpdatedBy, &policy.Version, &policy.CreatedAt, &policy.UpdatedAt,
	)
}

func scanEvent(row rowScanner, event *Event) error {
	var previousState *string
	if err := row.Scan(
		&event.PolicyID, &event.TargetType, &event.TargetID, &event.WriteScope,
		&previousState, &event.NewState, &event.Reason, &event.ObservedRatio,
		&event.PolicyVersion, &event.OperatorID, &event.CreatedAt,
	); err != nil {
		return err
	}
	if previousState != nil {
		event.PreviousState = State(*previousState)
	}
	return nil
}
