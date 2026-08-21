package deviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/task"
)

type PgActionStore struct {
	db storage.DB
}

func NewPgActionStore(pool *pgxpool.Pool) *PgActionStore {
	return newPgActionStoreWithDB(storage.NewPoolDB(pool))
}

func newPgActionStoreWithDB(db storage.DB) *PgActionStore {
	return &PgActionStore{db: db}
}

const actionColumns = `a.id, a.device_id, a.candidate_id, a.decision_id, a.action_type, a.direction,
	a.status, a.device_task_id,
	a.recovery_of_action_id, a.owned_rf_change, COALESCE(a.rf_change_paths, '[]'::jsonb), a.idempotency_key, a.requested_by,
	a.attempts, a.max_attempts, a.next_attempt_at, COALESCE(a.last_failure_code, ''), a.dead_at,
	a.manual_repair_required, COALESCE(a.error_message, ''), a.dispatched_at, a.completed_at,
	a.created_at, a.updated_at, COALESCE(d.serial_number, c.serial_number),
	COALESCE(NULLIF(d.model_name, ''), NULLIF(c.product_class, ''), ''),
	COALESCE(d.carrier::text, c.carrier), COALESCE(d.technology::text, c.technology, ''),
	COALESCE(c.rf_control_paths, '[]'::jsonb), COALESCE(c.oui, ''),
	COALESCE(a.bound_session_id, ''), COALESCE(a.bound_request_id, '')`

func (s *PgActionStore) Plan(ctx context.Context, plan ActionPlan) (Action, bool, error) {
	if s == nil || s.db == nil {
		return Action{}, false, ErrAccessGateDependencyMissing
	}
	if !exactlyOneActionTarget(plan.DeviceID, plan.CandidateID) || plan.DecisionID == uuid.Nil || strings.TrimSpace(plan.IdempotencyKey) == "" {
		return Action{}, false, fmt.Errorf("plan RF action: invalid identity")
	}
	if !validActionContract(plan.ActionType, plan.Direction) {
		return Action{}, false, fmt.Errorf("plan RF action: invalid type/direction")
	}
	if plan.CandidateID != nil && (plan.ActionType != ActionTypeRFOff || plan.Direction != ActionDirectionContain) {
		return Action{}, false, fmt.Errorf("plan candidate RF action: only containment is permitted")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Action{}, false, fmt.Errorf("begin RF action plan: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lockKey := "device-access-action:" + actionPlanTargetKey(plan)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", lockKey); err != nil {
		return Action{}, false, fmt.Errorf("lock RF action plan: %w", err)
	}
	existingID, err := findConflictingActionPlan(ctx, tx, plan)
	if err != nil {
		return Action{}, false, err
	}
	if existingID != uuid.Nil {
		if err := tx.Commit(ctx); err != nil {
			return Action{}, false, fmt.Errorf("commit existing RF action plan: %w", err)
		}
		action, err := s.Get(ctx, existingID)
		return action, false, err
	}
	id := uuid.New()
	query, args, err := storage.Psql.Insert("device_access_actions").
		Columns(
			"id", "device_id", "candidate_id", "decision_id", "action_type", "direction", "status",
			"recovery_of_action_id", "idempotency_key", "requested_by",
		).
		Values(
			id, plan.DeviceID, plan.CandidateID, plan.DecisionID, plan.ActionType, plan.Direction,
			ActionStatusPendingDispatch, plan.RecoveryOfActionID, plan.IdempotencyKey, plan.RequestedBy,
		).
		Suffix("ON CONFLICT (idempotency_key) DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return Action{}, false, fmt.Errorf("build RF action plan insert: %w", err)
	}
	var insertedID uuid.UUID
	err = tx.QueryRow(ctx, query, args...).Scan(&insertedID)
	created := err == nil
	if err != nil && err != pgx.ErrNoRows {
		return Action{}, false, fmt.Errorf("insert RF action plan: %w", err)
	}
	if !created {
		query, args, err = storage.Psql.Select("id").From("device_access_actions").
			Where(sq.Eq{"idempotency_key": plan.IdempotencyKey}).ToSql()
		if err != nil {
			return Action{}, false, fmt.Errorf("build idempotent RF action lookup: %w", err)
		}
		if err := tx.QueryRow(ctx, query, args...).Scan(&insertedID); err != nil {
			return Action{}, false, fmt.Errorf("load idempotent RF action: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Action{}, false, fmt.Errorf("commit RF action plan: %w", err)
	}
	action, err := s.Get(ctx, insertedID)
	if err != nil {
		return Action{}, false, err
	}
	return action, created, nil
}

func actionPlanTargetKey(plan ActionPlan) string {
	if plan.DeviceID != nil {
		return "device:" + plan.DeviceID.String()
	}
	return "candidate:" + plan.CandidateID.String()
}

func actionPlanTargetPredicate(plan ActionPlan, prefix string) sq.Sqlizer {
	if plan.DeviceID != nil {
		return sq.Eq{prefix + "device_id": *plan.DeviceID}
	}
	return sq.Eq{prefix + "candidate_id": *plan.CandidateID}
}

type actionPlanQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findConflictingActionPlan(ctx context.Context, db actionPlanQuerier, plan ActionPlan) (uuid.UUID, error) {
	activeOrRetryable := sq.Or{
		sq.Eq{"status": []ActionStatus{
			ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying, ActionStatusRetryWait,
		}},
		sq.And{
			sq.Eq{"status": ActionStatusSucceeded},
			sq.Eq{"owned_rf_change": true},
			sq.Expr("jsonb_array_length(COALESCE(rf_change_paths, '[]'::jsonb)) > 0"),
		},
	}
	builder := storage.Psql.Select("id").From("device_access_actions").
		Where(actionPlanTargetPredicate(plan, "")).
		Where(sq.Eq{"action_type": plan.ActionType}).
		Where(activeOrRetryable).OrderBy("created_at DESC").Limit(1)
	if plan.ActionType == ActionTypeRFOff {
		builder = builder.Where(`NOT EXISTS (
			SELECT 1 FROM device_access_actions recovery
			WHERE recovery.recovery_of_action_id = device_access_actions.id
			AND recovery.action_type = 'rf_on' AND recovery.status = 'succeeded'
		)`)
	} else if plan.RecoveryOfActionID != nil {
		builder = builder.Where(sq.Eq{"recovery_of_action_id": *plan.RecoveryOfActionID})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build conflicting RF action plan query: %w", err)
	}
	var id uuid.UUID
	if err := db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("query conflicting RF action plan: %w", err)
	}
	return id, nil
}

func (s *PgActionStore) Get(ctx context.Context, id uuid.UUID) (Action, error) {
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Where(sq.Eq{"a.id": id}).ToSql()
	if err != nil {
		return Action{}, fmt.Errorf("build RF action query: %w", err)
	}
	item, err := scanAction(s.db.QueryRow(ctx, query, args...))
	if err != nil {
		return Action{}, fmt.Errorf("get RF action: %w", err)
	}
	return item, nil
}

func (s *PgActionStore) List(ctx context.Context, filter ActionListFilter) ([]Action, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := actionFilters(filter)
	from := "device_access_actions a"
	if value := strings.TrimSpace(filter.Carrier); value != "" {
		where = append(where, sq.Eq{"decision.carrier": value})
	}
	if value := strings.TrimSpace(filter.SerialNumber); value != "" {
		where = append(where, sq.ILike{"decision.serial_number": "%" + value + "%"})
	}
	countBuilder := storage.Psql.Select("COUNT(*)").
		From(from).LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Join("device_access_decisions decision ON decision.id = a.decision_id").
		Where(where)
	countBuilder = applyAccessIdentityVisibility(countBuilder, "a.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups)
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build RF action count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count RF actions: %w", err)
	}
	listBuilder := storage.Psql.Select(actionColumns).
		From(from).LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Join("device_access_decisions decision ON decision.id = a.decision_id").
		Where(where).
		OrderBy("a.created_at DESC", "a.id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = applyAccessIdentityVisibility(listBuilder, "a.device_id", "decision.carrier", "decision.serial_number", filter.VisibleGroups)
	query, args, err := listBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build RF action list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query RF actions: %w", err)
	}
	defer rows.Close()
	items := make([]Action, 0, pageSize)
	for rows.Next() {
		item, err := scanAction(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan RF action list: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate RF action list: %w", err)
	}
	return items, total, nil
}

func actionFilters(filter ActionListFilter) sq.And {
	where := sq.And{}
	if filter.Status != "" {
		where = append(where, sq.Eq{"a.status": filter.Status})
	}
	if filter.Decision != "" {
		where = append(where, sq.Eq{"decision.decision": filter.Decision})
	}
	if filter.ReasonCode != "" {
		where = append(where, sq.Eq{"decision.reason_code": filter.ReasonCode})
	}
	if filter.PolicyVersionID != nil {
		where = append(where, sq.Eq{"decision.policy_version_id": *filter.PolicyVersionID})
	}
	if filter.StartedAt != nil {
		where = append(where, sq.GtOrEq{"a.created_at": *filter.StartedAt})
	}
	if filter.EndedAt != nil {
		where = append(where, sq.LtOrEq{"a.created_at": *filter.EndedAt})
	}
	return where
}

func (s *PgActionStore) FindOwnedIsolation(ctx context.Context, deviceID uuid.UUID) (*Action, error) {
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Where(sq.Eq{
			"a.device_id": deviceID, "a.action_type": ActionTypeRFOff,
			"a.status": ActionStatusSucceeded, "a.owned_rf_change": true,
		}).
		Where("jsonb_array_length(COALESCE(a.rf_change_paths, '[]'::jsonb)) > 0").
		Where(`NOT EXISTS (
			SELECT 1 FROM device_access_actions recovery
			WHERE recovery.recovery_of_action_id = a.id
			AND recovery.action_type = 'rf_on'
			AND recovery.status IN ('pending_dispatch','dispatching','verifying','retry_wait','succeeded')
		)`).
		OrderBy("a.completed_at DESC NULLS LAST", "a.created_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build owned RF isolation query: %w", err)
	}
	item, err := scanAction(s.db.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find owned RF isolation: %w", err)
	}
	return &item, nil
}

func (s *PgActionStore) FindOpenContainment(ctx context.Context, deviceID uuid.UUID) (*Action, error) {
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Where(sq.Eq{
			"a.device_id":   deviceID,
			"a.action_type": ActionTypeRFOff,
			"a.status": []ActionStatus{
				ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying,
				ActionStatusRetryWait, ActionStatusSucceeded,
			},
		}).
		Where(sq.Or{
			sq.NotEq{"a.status": ActionStatusSucceeded},
			sq.And{
				sq.Eq{"a.owned_rf_change": true},
				sq.Expr("jsonb_array_length(COALESCE(a.rf_change_paths, '[]'::jsonb)) > 0"),
			},
		}).
		Where(`NOT EXISTS (
			SELECT 1 FROM device_access_actions recovery
			WHERE recovery.recovery_of_action_id = a.id
			AND recovery.action_type = 'rf_on'
			AND recovery.status = 'succeeded'
		)`).
		OrderBy("a.created_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build open RF containment query: %w", err)
	}
	item, err := scanAction(s.db.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find open RF containment: %w", err)
	}
	return &item, nil
}

func (s *PgActionStore) FindOpenCandidateContainment(ctx context.Context, candidateID uuid.UUID) (*Action, error) {
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Where(sq.Eq{
			"a.candidate_id": candidateID,
			"a.action_type":  ActionTypeRFOff,
			"a.status": []ActionStatus{
				ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying,
				ActionStatusRetryWait, ActionStatusSucceeded,
			},
		}).
		Where(sq.Or{
			sq.NotEq{"a.status": ActionStatusSucceeded},
			sq.And{
				sq.Eq{"a.owned_rf_change": true},
				sq.Expr("jsonb_array_length(COALESCE(a.rf_change_paths, '[]'::jsonb)) > 0"),
			},
		}).
		OrderBy("a.created_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build open candidate RF containment query: %w", err)
	}
	item, err := scanAction(s.db.QueryRow(ctx, query, args...))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find open candidate RF containment: %w", err)
	}
	return &item, nil
}

// PromoteCandidateTarget transfers RF ownership only after the accepted
// endpoint exists in the formal inventory. The candidate row, current access
// projection and all of its RF actions move together, so an accepted device
// can recover only paths that access control previously verified as changed.
func (s *PgActionStore) PromoteCandidateTarget(
	ctx context.Context,
	candidateID uuid.UUID,
	carrier, serialNumber string,
) (uuid.UUID, error) {
	if candidateID == uuid.Nil || strings.TrimSpace(carrier) == "" || strings.TrimSpace(serialNumber) == "" {
		return uuid.Nil, fmt.Errorf("promote candidate RF target: invalid identity")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin candidate RF target promotion: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	query, args, err := storage.Psql.Select("id").From("device_access_candidates").
		Where(sq.Eq{"id": candidateID, "carrier": strings.TrimSpace(carrier), "serial_number": strings.TrimSpace(serialNumber)}).
		Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build candidate RF target lock: %w", err)
	}
	var lockedCandidateID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&lockedCandidateID); err != nil {
		return uuid.Nil, fmt.Errorf("lock candidate RF target: %w", err)
	}

	query, args, err = storage.Psql.Select("id").From("devices").
		Where(sq.Eq{"carrier": strings.TrimSpace(carrier), "serial_number": strings.TrimSpace(serialNumber)}).
		Where(sq.Expr("deleted_at IS NULL")).Limit(1).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build promoted device lookup: %w", err)
	}
	var deviceID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&deviceID); err != nil {
		return uuid.Nil, fmt.Errorf("load promoted formal device: %w", err)
	}

	updates := []struct {
		table string
		where sq.Sqlizer
	}{
		{table: "device_access_candidates", where: sq.Eq{"id": candidateID}},
		{table: "device_access_states", where: sq.Eq{"candidate_id": candidateID}},
		{table: "device_access_actions", where: sq.Eq{"candidate_id": candidateID}},
	}
	for _, update := range updates {
		builder := storage.Psql.Update(update.table).Set("device_id", deviceID).Where(update.where)
		if update.table != "device_access_candidates" {
			builder = builder.Set("candidate_id", nil)
		}
		if update.table == "device_access_candidates" {
			builder = builder.Set("updated_at", time.Now().UTC())
		}
		updateSQL, updateArgs, buildErr := builder.ToSql()
		if buildErr != nil {
			return uuid.Nil, fmt.Errorf("build %s candidate RF promotion: %w", update.table, buildErr)
		}
		if _, execErr := tx.Exec(ctx, updateSQL, updateArgs...); execErr != nil {
			return uuid.Nil, fmt.Errorf("promote %s candidate RF ownership: %w", update.table, execErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit candidate RF target promotion: %w", err)
	}
	return deviceID, nil
}

func (s *PgActionStore) BeginDispatch(ctx context.Context, actionID uuid.UUID, at time.Time) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusPendingDispatch}, map[string]any{
		"status":   ActionStatusDispatching,
		"attempts": sq.Expr("attempts + 1"), "device_task_id": nil,
		"next_attempt_at": nil, "dispatched_at": at, "updated_at": at,
	}, "begin action dispatch", sq.Expr("attempts < max_attempts"))
}

func (s *PgActionStore) AttachDispatchTask(ctx context.Context, actionID, deviceTaskID uuid.UUID) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching}, map[string]any{
		"device_task_id": deviceTaskID, "updated_at": time.Now().UTC(),
	}, "attach action dispatch task", sq.Expr("device_task_id IS NULL"))
}

func (s *PgActionStore) SetRFChangePaths(ctx context.Context, actionID uuid.UUID, paths []string, at time.Time) error {
	raw, err := json.Marshal(paths)
	if err != nil {
		return fmt.Errorf("encode RF change paths: %w", err)
	}
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching, ActionStatusVerifying}, map[string]any{
		"rf_change_paths": json.RawMessage(raw), "updated_at": at,
	}, "persist RF change paths", sq.Expr("rf_change_paths = '[]'::jsonb"))
}

func (s *PgActionStore) AdvanceDispatchTask(
	ctx context.Context,
	actionID, currentTaskID, nextTaskID uuid.UUID,
	status ActionStatus,
	at time.Time,
) error {
	if status != ActionStatusDispatching && status != ActionStatusVerifying {
		return fmt.Errorf("advance RF action task: invalid status %q", status)
	}
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching, ActionStatusVerifying}, map[string]any{
		"status": status, "device_task_id": nextTaskID, "updated_at": at,
	}, "advance RF action task", sq.Or{
		sq.Eq{"device_task_id": currentTaskID},
		sq.Expr("device_task_id IS NULL"),
	})
}

func (s *PgActionStore) MarkTerminal(
	ctx context.Context,
	actionID uuid.UUID,
	status ActionStatus,
	ownedRFChange bool,
	failureCode string,
	message string,
	at time.Time,
) error {
	if status != ActionStatusSucceeded && status != ActionStatusFailed && status != ActionStatusCancelled {
		return fmt.Errorf("mark RF action terminal: invalid status %q", status)
	}
	values := map[string]any{
		"status": status, "owned_rf_change": ownedRFChange,
		"error_message": nullableString(strings.TrimSpace(message)),
		"completed_at":  at, "next_attempt_at": nil,
		"manual_repair_required": false, "updated_at": at,
	}
	if status == ActionStatusFailed {
		values["last_failure_code"] = nullableString(strings.TrimSpace(failureCode))
	}
	eventType := ""
	eventName := ""
	if status == ActionStatusFailed {
		eventType = event.SubjectDeviceAccessActionFailed
		eventName = "DEVICE_ACCESS_ACTION_FAILED"
	}
	return s.updateWithLifecycleOutbox(ctx, actionID,
		[]ActionStatus{ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying, ActionStatusRetryWait},
		values, "mark action terminal", eventType, eventName, at, status == ActionStatusSucceeded)
}

func (s *PgActionStore) ScheduleRetry(
	ctx context.Context,
	actionID uuid.UUID,
	failureCode, message string,
	nextAttemptAt time.Time,
	dead bool,
) error {
	now := time.Now().UTC()
	values := map[string]any{
		"status": ActionStatusRetryWait, "device_task_id": nil, "owned_rf_change": false,
		"bound_session_id": nil, "bound_request_id": nil,
		"last_failure_code": nullableString(strings.TrimSpace(failureCode)),
		"error_message":     nullableString(strings.TrimSpace(message)),
		"next_attempt_at":   nextAttemptAt, "completed_at": nil, "updated_at": now,
	}
	if dead {
		values["status"] = ActionStatusDead
		values["next_attempt_at"] = nil
		values["dead_at"] = now
		values["completed_at"] = now
		values["manual_repair_required"] = true
		return s.updateWithLifecycleOutbox(ctx, actionID,
			[]ActionStatus{ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying},
			values, "mark RF action dead", event.SubjectDeviceAccessActionFailed,
			"DEVICE_ACCESS_ACTION_FAILED", now, false)
	}
	return s.updateWithLifecycleOutbox(ctx, actionID,
		[]ActionStatus{ActionStatusPendingDispatch, ActionStatusDispatching, ActionStatusVerifying},
		values, "schedule RF action retry", event.SubjectDeviceAccessActionFailed,
		"DEVICE_ACCESS_ACTION_FAILED", now, false)
}

type actionLifecycleContext struct {
	DecisionID      uuid.UUID
	DeviceID        *uuid.UUID
	CandidateID     *uuid.UUID
	ActionType      ActionType
	Status          ActionStatus
	Attempts        int
	RequestID       string
	LastFailureCode string
	ErrorMessage    string
	Carrier         string
	SerialNumber    string
	PolicyVersionID *uuid.UUID
}

func (s *PgActionStore) updateWithLifecycleOutbox(
	ctx context.Context,
	actionID uuid.UUID,
	from []ActionStatus,
	values map[string]any,
	operation, eventType, eventName string,
	occurredAt time.Time,
	emitRecoveryWhenPreviouslyFailed bool,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", operation, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	builder := storage.Psql.Update("device_access_actions").Where(sq.Eq{"id": actionID, "status": from})
	for column, value := range values {
		builder = builder.Set(column, value)
	}
	if values["status"] == ActionStatusSucceeded {
		builder = builder.Where(`(
			(action_type = 'rf_off' AND jsonb_array_length(COALESCE(rf_change_paths, '[]'::jsonb)) = 0
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence
					WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'baseline_gpv' AND evidence.status = 'succeeded'))
			OR
			(action_type = 'rf_off' AND jsonb_array_length(COALESCE(rf_change_paths, '[]'::jsonb)) > 0
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'baseline_gpv' AND evidence.status = 'succeeded')
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'spv' AND evidence.status = 'succeeded')
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'readback_gpv' AND evidence.status = 'succeeded'))
			OR
			(action_type = 'rf_on'
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'spv' AND evidence.status = 'succeeded')
				AND EXISTS (SELECT 1 FROM device_access_action_attempts evidence WHERE evidence.action_id = device_access_actions.id AND evidence.phase = 'readback_gpv' AND evidence.status = 'succeeded'))
		)`)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build %s: %w", operation, err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", operation, ErrActionStateChanged)
	}

	details, err := loadActionLifecycleContext(ctx, tx, actionID)
	if err != nil {
		return fmt.Errorf("load %s event context: %w", operation, err)
	}
	if emitRecoveryWhenPreviouslyFailed && (details.Attempts > 1 || details.LastFailureCode != "") {
		eventType = event.SubjectDeviceAccessActionRecovered
		eventName = "DEVICE_ACCESS_ACTION_RECOVERED"
	}
	if eventType != "" {
		if err := insertActionLifecycleOutbox(ctx, tx, actionID, details, eventType, eventName, occurredAt); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", operation, err)
	}
	return nil
}

func loadActionLifecycleContext(ctx context.Context, tx pgx.Tx, actionID uuid.UUID) (actionLifecycleContext, error) {
	query, args, err := storage.Psql.Select(
		"a.decision_id", "a.device_id", "a.candidate_id", "a.action_type", "a.status", "a.attempts",
		"dec.trigger_event_id", "COALESCE(a.last_failure_code, '')", "COALESCE(a.error_message, '')",
		"dec.carrier", "dec.serial_number", "dec.policy_version_id",
	).From("device_access_actions a").
		Join("device_access_decisions dec ON dec.id = a.decision_id").
		Where(sq.Eq{"a.id": actionID}).ToSql()
	if err != nil {
		return actionLifecycleContext{}, fmt.Errorf("build action lifecycle lookup: %w", err)
	}
	var details actionLifecycleContext
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&details.DecisionID, &details.DeviceID, &details.CandidateID, &details.ActionType, &details.Status, &details.Attempts,
		&details.RequestID, &details.LastFailureCode, &details.ErrorMessage,
		&details.Carrier, &details.SerialNumber, &details.PolicyVersionID,
	); err != nil {
		return actionLifecycleContext{}, fmt.Errorf("query action lifecycle context: %w", err)
	}
	return details, nil
}

func insertActionLifecycleOutbox(
	ctx context.Context,
	tx pgx.Tx,
	actionID uuid.UUID,
	details actionLifecycleContext,
	eventType, eventName string,
	occurredAt time.Time,
) error {
	eventID := uuid.New()
	payload, err := json.Marshal(ActionLifecycleEvent{
		EventID: eventID.String(), EventName: eventName, RequestID: details.RequestID,
		DecisionID: details.DecisionID, ActionID: actionID, DeviceID: details.DeviceID, CandidateID: details.CandidateID,
		Carrier: details.Carrier, SerialNumber: details.SerialNumber, PolicyVersion: details.PolicyVersionID,
		ActionType: details.ActionType, ActionStatus: details.Status, Attempt: details.Attempts,
		ReasonCode: details.LastFailureCode, Message: details.ErrorMessage, OccurredAt: occurredAt,
	})
	if err != nil {
		return fmt.Errorf("encode RF action lifecycle event: %w", err)
	}
	query, args, err := storage.Psql.Insert("device_access_outbox").Columns(
		"aggregate_type", "aggregate_id", "event_type", "event_key", "payload",
		"next_attempt_at", "created_at", "updated_at",
	).Values(
		"device_access_action", actionID, eventType, eventID.String(), payload,
		occurredAt, occurredAt, occurredAt,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build RF action lifecycle outbox: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert RF action lifecycle outbox: %w", err)
	}
	return nil
}

func (s *PgActionStore) ResetForManualRetry(ctx context.Context, actionID uuid.UUID, reason string, at time.Time) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reset RF action for manual retry: repair reason is required")
	}
	return s.update(ctx, actionID, []ActionStatus{ActionStatusFailed, ActionStatusDead}, map[string]any{
		"status": ActionStatusPendingDispatch, "device_task_id": nil,
		"bound_session_id": nil, "bound_request_id": nil,
		"owned_rf_change": false, "error_message": nil,
		"max_attempts":    sq.Expr("attempts + ?", defaultActionMaxAttempts),
		"next_attempt_at": at,
		"dead_at":         nil, "manual_repair_required": false,
		"dispatched_at": nil, "completed_at": nil, "updated_at": at,
	}, "reset action for manual retry")
}

func (s *PgActionStore) RecordAttemptQueued(ctx context.Context, attempt ActionAttempt) error {
	if attempt.ActionID == uuid.Nil || attempt.AttemptNo < 1 || attempt.DeviceTaskID == nil {
		return fmt.Errorf("record queued RF action attempt: invalid identity")
	}
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	if len(attempt.RequestSummary) == 0 {
		attempt.RequestSummary = json.RawMessage(`{}`)
	}
	query, args, err := storage.Psql.Insert("device_access_action_attempts").
		Columns(
			"id", "action_id", "attempt_no", "phase", "device_task_id", "command_key",
			"status", "request_summary", "started_at",
		).
		Values(
			attempt.ID, attempt.ActionID, attempt.AttemptNo, attempt.Phase, attempt.DeviceTaskID,
			nullableString(attempt.CommandKey), ActionAttemptQueued, attempt.RequestSummary, attempt.StartedAt,
		).
		Suffix(`ON CONFLICT (action_id, attempt_no, phase) DO UPDATE SET
			device_task_id = EXCLUDED.device_task_id,
			command_key = EXCLUDED.command_key,
			status = CASE
				WHEN device_access_action_attempts.status IN ('succeeded','failed','timeout','cancelled')
				THEN device_access_action_attempts.status ELSE EXCLUDED.status END,
			request_summary = EXCLUDED.request_summary`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build queued RF action attempt: %w", err)
	}
	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record queued RF action attempt: %w", err)
	}
	return nil
}

func (s *PgActionStore) RecordAttemptCompleted(ctx context.Context, attempt ActionAttempt) error {
	if attempt.ActionID == uuid.Nil || attempt.AttemptNo < 1 || attempt.Phase == "" {
		return fmt.Errorf("complete RF action attempt: invalid identity")
	}
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	if len(attempt.RequestSummary) == 0 {
		attempt.RequestSummary = json.RawMessage(`{}`)
	}
	if len(attempt.ResponseSummary) == 0 {
		attempt.ResponseSummary = json.RawMessage(`{}`)
	}
	query, args, err := storage.Psql.Insert("device_access_action_attempts").
		Columns(
			"id", "action_id", "attempt_no", "phase", "device_task_id", "command_key", "status",
			"fault_code", "failure_code", "error_message", "request_summary", "response_summary",
			"trace_id", "started_at", "completed_at",
		).
		Values(
			attempt.ID, attempt.ActionID, attempt.AttemptNo, attempt.Phase, attempt.DeviceTaskID,
			nullableString(attempt.CommandKey), attempt.Status, nullableString(attempt.FaultCode),
			nullableString(attempt.FailureCode), nullableString(attempt.ErrorMessage), attempt.RequestSummary,
			attempt.ResponseSummary, nullableString(attempt.TraceID), attempt.StartedAt, attempt.CompletedAt,
		).
		Suffix(`ON CONFLICT (action_id, attempt_no, phase) DO UPDATE SET
			device_task_id = COALESCE(EXCLUDED.device_task_id, device_access_action_attempts.device_task_id),
			command_key = COALESCE(EXCLUDED.command_key, device_access_action_attempts.command_key),
			status = EXCLUDED.status,
			fault_code = EXCLUDED.fault_code,
			failure_code = EXCLUDED.failure_code,
			error_message = EXCLUDED.error_message,
			response_summary = EXCLUDED.response_summary,
			trace_id = COALESCE(EXCLUDED.trace_id, device_access_action_attempts.trace_id),
			completed_at = EXCLUDED.completed_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build completed RF action attempt: %w", err)
	}
	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record completed RF action attempt: %w", err)
	}
	return nil
}

func (s *PgActionStore) ListAttempts(ctx context.Context, actionID uuid.UUID) ([]ActionAttempt, error) {
	query, args, err := storage.Psql.Select(
		"id", "action_id", "attempt_no", "phase", "device_task_id", "COALESCE(command_key, '')", "status",
		"COALESCE(fault_code, '')", "COALESCE(failure_code, '')", "COALESCE(error_message, '')",
		"request_summary", "response_summary", "COALESCE(trace_id, '')", "started_at", "completed_at",
	).From("device_access_action_attempts").Where(sq.Eq{"action_id": actionID}).
		OrderBy("attempt_no ASC", "started_at ASC", "id ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build RF action attempt list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query RF action attempts: %w", err)
	}
	defer rows.Close()
	items := make([]ActionAttempt, 0)
	for rows.Next() {
		var item ActionAttempt
		if err := rows.Scan(
			&item.ID, &item.ActionID, &item.AttemptNo, &item.Phase, &item.DeviceTaskID, &item.CommandKey,
			&item.Status, &item.FaultCode, &item.FailureCode, &item.ErrorMessage,
			&item.RequestSummary, &item.ResponseSummary, &item.TraceID, &item.StartedAt, &item.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan RF action attempt: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate RF action attempts: %w", err)
	}
	return items, nil
}

func (s *PgActionStore) ClaimDue(ctx context.Context, now time.Time, limit int) ([]Action, error) {
	if limit < 1 {
		limit = 20
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin due RF action claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select("a.id").From("device_access_actions a").
		Where(sq.Or{
			sq.Eq{"a.status": ActionStatusPendingDispatch},
			sq.Eq{"a.status": ActionStatusRetryWait},
		}).
		Where(sq.Expr("a.attempts < a.max_attempts")).
		Where(sq.Or{sq.Expr("a.next_attempt_at IS NULL"), sq.LtOrEq{"a.next_attempt_at": now}}).
		OrderBy("COALESCE(a.next_attempt_at, a.created_at) ASC", "a.id ASC").Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build due RF action claim: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query due RF actions: %w", err)
	}
	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan due RF action: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate due RF actions: %w", err)
	}
	rows.Close()
	for _, id := range ids {
		updateSQL, updateArgs, buildErr := storage.Psql.Update("device_access_actions").
			Set("status", ActionStatusDispatching).
			Set("attempts", sq.Expr("attempts + 1")).
			Set("device_task_id", nil).
			Set("next_attempt_at", nil).
			Set("dispatched_at", now).
			Set("updated_at", now).
			Where(sq.Eq{"id": id}).ToSql()
		if buildErr != nil {
			return nil, fmt.Errorf("build claimed RF action update: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, updateSQL, updateArgs...); execErr != nil {
			return nil, fmt.Errorf("claim due RF action: %w", execErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit due RF action claim: %w", err)
	}
	actions := make([]Action, 0, len(ids))
	for _, id := range ids {
		action, loadErr := s.Get(ctx, id)
		if loadErr != nil {
			return nil, fmt.Errorf("load claimed RF action: %w", loadErr)
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *PgActionStore) ListRecoverable(ctx context.Context, staleBefore time.Time, limit int) ([]Action, error) {
	if limit < 1 {
		limit = 20
	}
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		LeftJoin("devices d ON d.id = a.device_id").
		LeftJoin("device_access_candidates c ON c.id = a.candidate_id").
		Where(sq.Eq{"a.status": []ActionStatus{ActionStatusDispatching, ActionStatusVerifying}}).
		Where(sq.LtOrEq{"a.updated_at": staleBefore}).OrderBy("a.updated_at ASC", "a.id ASC").
		Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build recoverable RF action list: %w", err)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query recoverable RF actions: %w", err)
	}
	defer rows.Close()
	items := make([]Action, 0, limit)
	for rows.Next() {
		item, scanErr := scanAction(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan recoverable RF action: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recoverable RF actions: %w", err)
	}
	return items, nil
}

func (s *PgActionStore) LoadActionTask(ctx context.Context, action Action) (*task.Task, error) {
	builder := storage.Psql.Select(
		"id::text", "device_sn", "method", "COALESCE(params, '{}'::jsonb)", "COALESCE(command_key, '')",
		"status", "COALESCE(result, '{}'::jsonb)", "COALESCE(error_code, 0)", "COALESCE(error_message, '')",
		"source", "COALESCE(source_id::text, '')", "admission_class",
	).From("device_tasks")
	if action.DeviceTaskID != nil {
		builder = builder.Where(sq.Eq{"id": *action.DeviceTaskID})
	} else {
		builder = builder.Where(sq.Eq{"source_id": action.ID}).
			Where(sq.Eq{"command_key": []string{
				rfBaselineCommandKey(action), rfWriteCommandKey(action), rfReadbackCommandKey(action),
			}}).OrderBy("created_at DESC").Limit(1)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build RF action task lookup: %w", err)
	}
	var item task.Task
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.DeviceSN, &item.Method, &item.Params, &item.CommandKey, &item.Status,
		&item.Result, &item.ErrorCode, &item.ErrorMessage, &item.Source, &item.SourceID, &item.AdmissionClass,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load RF action task: %w", err)
	}
	return &item, nil
}

func (s *PgActionStore) TouchAction(ctx context.Context, actionID uuid.UUID, at time.Time) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching, ActionStatusVerifying}, map[string]any{
		"updated_at": at,
	}, "touch recoverable RF action")
}

func (s *PgActionStore) WakeRetryForDevice(ctx context.Context, deviceID uuid.UUID, at time.Time) error {
	query, args, err := storage.Psql.Update("device_access_actions").
		Set("next_attempt_at", at).Set("updated_at", at).
		Where(sq.Eq{"device_id": deviceID, "status": ActionStatusRetryWait}).ToSql()
	if err != nil {
		return fmt.Errorf("build RF action Inform wake-up: %w", err)
	}
	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("wake RF actions after Inform: %w", err)
	}
	return nil
}

func (s *PgActionStore) BindCandidateExecution(
	ctx context.Context,
	actionID uuid.UUID,
	sessionID, requestID string,
	at time.Time,
) error {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("bind candidate RF execution: session and request are required")
	}
	query, args, err := storage.Psql.Update("device_access_actions").
		Set("bound_session_id", strings.TrimSpace(sessionID)).
		Set("bound_request_id", strings.TrimSpace(requestID)).
		Set("updated_at", at).
		Where(sq.Eq{
			"id": actionID, "status": []ActionStatus{ActionStatusDispatching, ActionStatusVerifying},
		}).Where(sq.Expr("candidate_id IS NOT NULL")).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate RF execution binding: %w", err)
	}
	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bind candidate RF execution: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("bind candidate RF execution: %w", ErrActionStateChanged)
	}
	return nil
}

func (s *PgActionStore) update(
	ctx context.Context,
	id uuid.UUID,
	from []ActionStatus,
	values map[string]any,
	operation string,
	predicates ...sq.Sqlizer,
) error {
	builder := storage.Psql.Update("device_access_actions").Where(sq.Eq{"id": id, "status": from})
	for _, predicate := range predicates {
		builder = builder.Where(predicate)
	}
	for column, value := range values {
		builder = builder.Set(column, value)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build %s: %w", operation, err)
	}
	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s: %w", operation, ErrActionStateChanged)
	}
	return nil
}

type actionScanner interface {
	Scan(dest ...any) error
}

func scanAction(row actionScanner) (Action, error) {
	var item Action
	var rfChangePaths []byte
	var candidateRFPaths []byte
	err := row.Scan(
		&item.ID, &item.DeviceID, &item.CandidateID, &item.DecisionID, &item.ActionType, &item.Direction,
		&item.Status, &item.DeviceTaskID,
		&item.RecoveryOfActionID, &item.OwnedRFChange, &rfChangePaths, &item.IdempotencyKey, &item.RequestedBy,
		&item.Attempts, &item.MaxAttempts, &item.NextAttemptAt, &item.LastFailureCode, &item.DeadAt,
		&item.ManualRepairNeeded, &item.ErrorMessage, &item.DispatchedAt, &item.CompletedAt,
		&item.CreatedAt, &item.UpdatedAt, &item.SerialNumber, &item.ProductName, &item.Carrier, &item.Technology,
		&candidateRFPaths, &item.CandidateOUI, &item.BoundSessionID, &item.BoundRequestID,
	)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(rfChangePaths, &item.RFChangePaths); err != nil {
		return Action{}, fmt.Errorf("decode RF change paths: %w", err)
	}
	if err := json.Unmarshal(candidateRFPaths, &item.CandidateRFPaths); err != nil {
		return Action{}, fmt.Errorf("decode candidate RF control paths: %w", err)
	}
	return item, nil
}

func validActionContract(actionType ActionType, direction ActionDirection) bool {
	return actionType == ActionTypeRFOff && direction == ActionDirectionContain ||
		actionType == ActionTypeRFOn && direction == ActionDirectionRelease
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

var _ ActionStore = (*PgActionStore)(nil)
