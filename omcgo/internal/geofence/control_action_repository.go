package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type ControlActionRepository interface {
	Create(context.Context, *ControlAction) (*ControlAction, bool, error)
	GetByID(context.Context, uuid.UUID) (*ControlAction, error)
	GetByActionKey(context.Context, string) (*ControlAction, error)
	ListByGeofence(context.Context, uuid.UUID, int) ([]ControlAction, error)
	FindRecoverableDeactivation(context.Context, uuid.UUID) (*ControlAction, error)
	ListDueVerifications(context.Context, time.Time, int) ([]ControlAction, error)
	UpdateStatus(context.Context, uuid.UUID, ControlActionStatus) error
	BeginVerification(context.Context, uuid.UUID, time.Time, time.Time) error
	ScheduleVerification(
		context.Context,
		uuid.UUID,
		[]ControlParameterState,
		string,
		int,
		time.Time,
	) error
	CompleteVerification(
		context.Context,
		uuid.UUID,
		[]ControlParameterState,
		ControlActionStatus,
		string,
		time.Time,
	) error
}

type PgControlActionRepository struct {
	pool *pgxpool.Pool
}

func NewPgControlActionRepository(pool *pgxpool.Pool) *PgControlActionRepository {
	return &PgControlActionRepository{pool: pool}
}

var controlActionColumns = []string{
	"id", "action_key", "parent_action_id", "device_id", "device_sn",
	"geofence_id", "binding_id", "effective_state_version", "action_type",
	"status", "contract_version", "before_state", "requested_state", "terminal_state", "verified_state",
	"verification_attempt", "next_verification_at", "verification_deadline",
	"last_error", "created_at", "updated_at",
	"completed_at",
}

func buildCreateControlActionQuery(action *ControlAction) (string, []any, error) {
	if action == nil {
		return "", nil, fmt.Errorf("control action is required")
	}
	beforeState, err := marshalControlState(action.BeforeState)
	if err != nil {
		return "", nil, fmt.Errorf("marshal control before state: %w", err)
	}
	requestedState, err := marshalControlState(action.RequestedState)
	if err != nil {
		return "", nil, fmt.Errorf("marshal control requested state: %w", err)
	}
	verifiedState, err := marshalControlState(action.VerifiedState)
	if err != nil {
		return "", nil, fmt.Errorf("marshal control verified state: %w", err)
	}
	terminalState, err := marshalControlState(action.TerminalState)
	if err != nil {
		return "", nil, fmt.Errorf("marshal control terminal state: %w", err)
	}
	return storage.Psql.Insert("geofence_control_actions").
		Columns(controlActionColumns...).
		Values(
			action.ID, action.ActionKey, action.ParentActionID, action.DeviceID,
			action.DeviceSN, action.GeofenceID, action.BindingID,
			action.EffectiveStateVersion, action.ActionType, action.Status,
			action.ContractVersion, beforeState, requestedState, terminalState, verifiedState,
			action.VerificationAttempt, action.NextVerificationAt, action.VerificationDeadline,
			action.LastError,
			action.CreatedAt, action.UpdatedAt,
			action.CompletedAt,
		).
		Suffix("ON CONFLICT (action_key) DO NOTHING RETURNING " +
			"id, action_key, parent_action_id, device_id, device_sn, geofence_id, " +
			"binding_id, effective_state_version, action_type, status, contract_version, " +
			"before_state, requested_state, terminal_state, verified_state, " +
			"verification_attempt, next_verification_at, verification_deadline, last_error, " +
			"created_at, updated_at, completed_at").
		ToSql()
}

func buildGetControlActionByKeyQuery(actionKey string) (string, []any, error) {
	return storage.Psql.Select(controlActionColumns...).
		From("geofence_control_actions").
		Where(sq.Eq{"action_key": actionKey}).
		ToSql()
}

func buildGetControlActionByIDQuery(actionID uuid.UUID) (string, []any, error) {
	return storage.Psql.Select(controlActionColumns...).
		From("geofence_control_actions").
		Where(sq.Eq{"id": actionID}).
		ToSql()
}

func buildListControlActionsByGeofenceQuery(
	geofenceID uuid.UUID,
	limit uint64,
) (string, []any, error) {
	return storage.Psql.Select(prefixColumns("action", controlActionColumns)...).
		From("geofence_control_actions action").
		LeftJoin("device_geofence_bindings binding ON binding.id = action.binding_id").
		LeftJoin("geofence_control_actions parent ON parent.id = action.parent_action_id").
		LeftJoin("device_geofence_bindings parent_binding ON parent_binding.id = parent.binding_id").
		Where(sq.Or{
			sq.Eq{"action.geofence_id": geofenceID},
			sq.Eq{"binding.geofence_id": geofenceID},
			sq.Eq{"parent.geofence_id": geofenceID},
			sq.Eq{"parent_binding.geofence_id": geofenceID},
		}).
		OrderBy("action.created_at DESC").
		Limit(limit).
		ToSql()
}

func buildFindRecoverableDeactivationQuery(deviceID uuid.UUID) (string, []any, error) {
	return storage.Psql.Select(prefixColumns("action", controlActionColumns)...).
		From("geofence_control_actions action").
		Where(sq.Eq{
			"action.device_id":   deviceID,
			"action.action_type": ControlActionDeactivate,
			"action.status":      ControlActionVerified,
		}).
		Where(sq.GtOrEq{"action.contract_version": GeofenceControlContractVersion}).
		Where("jsonb_array_length(action.terminal_state) > 0").
		Where("NOT EXISTS (SELECT 1 FROM geofence_control_actions child " +
			"WHERE child.parent_action_id = action.id AND child.action_type = 'activate')").
		OrderBy("action.created_at DESC").
		Limit(1).
		ToSql()
}

func buildListDueControlVerificationsQuery(now time.Time, limit uint64) (string, []any, error) {
	return storage.Psql.Select(controlActionColumns...).
		From("geofence_control_actions").
		Where(sq.Eq{"status": ControlActionVerifying}).
		Where(sq.GtOrEq{"contract_version": GeofenceControlContractVersion}).
		Where(sq.LtOrEq{"next_verification_at": now}).
		OrderBy("next_verification_at ASC").
		Limit(limit).
		ToSql()
}

func prefixColumns(prefix string, columns []string) []string {
	prefixed := make([]string, len(columns))
	for index, column := range columns {
		prefixed[index] = prefix + "." + column
	}
	return prefixed
}

func buildUpdateControlActionStatusQuery(
	actionID uuid.UUID,
	status ControlActionStatus,
) (string, []any, error) {
	query := storage.Psql.Update("geofence_control_actions").
		Set("status", status).
		Set("updated_at", sq.Expr("now()"))
	if status == ControlActionVerifying {
		query = query.Set("next_verification_at", nil)
	}
	return query.
		Where(sq.Eq{"id": actionID}).
		Where(sq.Eq{"status": []ControlActionStatus{
			ControlActionPending, ControlActionExecuting, ControlActionVerifying,
		}}).
		ToSql()
}

func buildBeginControlVerificationQuery(
	actionID uuid.UUID,
	deadline time.Time,
	nextAt time.Time,
) (string, []any, error) {
	return storage.Psql.Update("geofence_control_actions").
		Set("status", ControlActionVerifying).
		Set("verification_deadline", deadline).
		Set("next_verification_at", nextAt).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": actionID}).
		Where(sq.Eq{"status": []ControlActionStatus{
			ControlActionPending, ControlActionExecuting, ControlActionVerifying,
		}}).
		Where("verification_deadline IS NULL").
		ToSql()
}

func buildCompleteControlVerificationQuery(
	actionID uuid.UUID,
	verified []ControlParameterState,
	status ControlActionStatus,
	lastError string,
	completedAt time.Time,
) (string, []any, error) {
	verifiedState, err := marshalControlState(verified)
	if err != nil {
		return "", nil, fmt.Errorf("marshal verified control state: %w", err)
	}
	return storage.Psql.Update("geofence_control_actions").
		Set("verified_state", verifiedState).
		Set("status", status).
		Set("last_error", lastError).
		Set("next_verification_at", nil).
		Set("updated_at", completedAt).
		Set("completed_at", completedAt).
		Where(sq.Eq{"id": actionID}).
		Where(sq.Eq{"status": []ControlActionStatus{
			ControlActionPending, ControlActionExecuting, ControlActionVerifying,
		}}).
		ToSql()
}

func buildScheduleControlVerificationQuery(
	actionID uuid.UUID,
	verified []ControlParameterState,
	lastError string,
	attempt int,
	nextAt time.Time,
) (string, []any, error) {
	verifiedState, err := marshalControlState(verified)
	if err != nil {
		return "", nil, fmt.Errorf("marshal scheduled verification state: %w", err)
	}
	return storage.Psql.Update("geofence_control_actions").
		Set("verified_state", verifiedState).
		Set("status", ControlActionVerifying).
		Set("last_error", lastError).
		Set("verification_attempt", attempt).
		Set("next_verification_at", nextAt).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": actionID}).
		Where(sq.Eq{"status": ControlActionVerifying}).
		Where(sq.Eq{"verification_attempt": attempt - 1}).
		ToSql()
}

type controlActionScanner interface {
	Scan(...any) error
}

func scanControlAction(row controlActionScanner) (*ControlAction, error) {
	var action ControlAction
	var beforeState, requestedState, terminalState, verifiedState json.RawMessage
	if err := row.Scan(
		&action.ID, &action.ActionKey, &action.ParentActionID, &action.DeviceID,
		&action.DeviceSN, &action.GeofenceID, &action.BindingID,
		&action.EffectiveStateVersion, &action.ActionType, &action.Status,
		&action.ContractVersion, &beforeState, &requestedState, &terminalState, &verifiedState,
		&action.VerificationAttempt, &action.NextVerificationAt, &action.VerificationDeadline,
		&action.LastError,
		&action.CreatedAt, &action.UpdatedAt,
		&action.CompletedAt,
	); err != nil {
		return nil, err
	}
	for _, item := range []struct {
		raw    json.RawMessage
		target any
	}{
		{raw: beforeState, target: &action.BeforeState},
		{raw: requestedState, target: &action.RequestedState},
		{raw: terminalState, target: &action.TerminalState},
		{raw: verifiedState, target: &action.VerifiedState},
	} {
		raw, target := string(item.raw), item.target
		if raw == "" || raw == "null" {
			continue
		}
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			return nil, fmt.Errorf("decode control action state: %w", err)
		}
	}
	return &action, nil
}

func (r *PgControlActionRepository) Create(
	ctx context.Context,
	action *ControlAction,
) (*ControlAction, bool, error) {
	query, args, err := buildCreateControlActionQuery(action)
	if err != nil {
		return nil, false, err
	}
	created, err := scanControlAction(r.pool.QueryRow(ctx, query, args...))
	if err == nil {
		return created, true, nil
	}
	if err != pgx.ErrNoRows {
		return nil, false, fmt.Errorf("create geofence control action: %w", err)
	}
	existing, err := r.GetByActionKey(ctx, action.ActionKey)
	if err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *PgControlActionRepository) GetByActionKey(
	ctx context.Context,
	actionKey string,
) (*ControlAction, error) {
	query, args, err := buildGetControlActionByKeyQuery(actionKey)
	if err != nil {
		return nil, fmt.Errorf("build get geofence control action: %w", err)
	}
	action, err := scanControlAction(r.pool.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get geofence control action: %w", err)
	}
	return action, nil
}

func (r *PgControlActionRepository) GetByID(
	ctx context.Context,
	actionID uuid.UUID,
) (*ControlAction, error) {
	query, args, err := buildGetControlActionByIDQuery(actionID)
	if err != nil {
		return nil, fmt.Errorf("build get geofence control action by ID: %w", err)
	}
	action, err := scanControlAction(r.pool.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get geofence control action by ID: %w", err)
	}
	return action, nil
}

func (r *PgControlActionRepository) ListByGeofence(
	ctx context.Context,
	geofenceID uuid.UUID,
	limit int,
) ([]ControlAction, error) {
	if geofenceID == uuid.Nil {
		return nil, fmt.Errorf("geofence is required")
	}
	if limit <= 0 {
		limit = 100
	}
	query, args, err := buildListControlActionsByGeofenceQuery(
		geofenceID,
		uint64(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("build list geofence control actions: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list geofence control actions: %w", err)
	}
	defer rows.Close()
	actions := make([]ControlAction, 0)
	for rows.Next() {
		action, scanErr := scanControlAction(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan geofence control action: %w", scanErr)
		}
		actions = append(actions, *action)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence control actions: %w", err)
	}
	return actions, nil
}

func (r *PgControlActionRepository) FindRecoverableDeactivation(
	ctx context.Context,
	deviceID uuid.UUID,
) (*ControlAction, error) {
	query, args, err := buildFindRecoverableDeactivationQuery(deviceID)
	if err != nil {
		return nil, fmt.Errorf("build recoverable geofence deactivation lookup: %w", err)
	}
	action, err := scanControlAction(r.pool.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find recoverable geofence deactivation: %w", err)
	}
	return action, nil
}

func (r *PgControlActionRepository) ListDueVerifications(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]ControlAction, error) {
	if limit <= 0 {
		limit = 100
	}
	query, args, err := buildListDueControlVerificationsQuery(now, uint64(limit))
	if err != nil {
		return nil, fmt.Errorf("build due geofence verification lookup: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list due geofence verifications: %w", err)
	}
	defer rows.Close()
	actions := make([]ControlAction, 0)
	for rows.Next() {
		action, scanErr := scanControlAction(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan due geofence verification: %w", scanErr)
		}
		actions = append(actions, *action)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due geofence verifications: %w", err)
	}
	return actions, nil
}

func (r *PgControlActionRepository) UpdateStatus(
	ctx context.Context,
	actionID uuid.UUID,
	status ControlActionStatus,
) error {
	query, args, err := buildUpdateControlActionStatusQuery(actionID, status)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update geofence control action status: %w", err)
	}
	if result.RowsAffected() != 1 {
		// A duplicate/out-of-order event may race with a terminal update. Status
		// transitions are monotonic, so a stale update is an idempotent no-op.
		return nil
	}
	return nil
}

func (r *PgControlActionRepository) BeginVerification(
	ctx context.Context,
	actionID uuid.UUID,
	deadline time.Time,
	nextAt time.Time,
) error {
	query, args, err := buildBeginControlVerificationQuery(actionID, deadline, nextAt)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("begin geofence control verification: %w", err)
	}
	if result.RowsAffected() > 1 {
		return fmt.Errorf("begin geofence control verification: action %s updated more than once", actionID)
	}
	return nil
}

func (r *PgControlActionRepository) ScheduleVerification(
	ctx context.Context,
	actionID uuid.UUID,
	verified []ControlParameterState,
	lastError string,
	attempt int,
	nextAt time.Time,
) error {
	query, args, err := buildScheduleControlVerificationQuery(
		actionID, verified, lastError, attempt, nextAt,
	)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("schedule geofence control verification: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil
	}
	return nil
}

func (r *PgControlActionRepository) CompleteVerification(
	ctx context.Context,
	actionID uuid.UUID,
	verified []ControlParameterState,
	status ControlActionStatus,
	lastError string,
	completedAt time.Time,
) error {
	query, args, err := buildCompleteControlVerificationQuery(
		actionID, verified, status, lastError, completedAt,
	)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("complete geofence control verification: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil
	}
	return nil
}
