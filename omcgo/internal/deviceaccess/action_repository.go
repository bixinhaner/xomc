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
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/storage"
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

const actionColumns = `a.id, a.device_id, a.decision_id, a.action_type, a.direction,
	a.status, a.device_task_id,
	a.recovery_of_action_id, a.owned_rf_change, COALESCE(a.rf_change_paths, '[]'::jsonb), a.idempotency_key, a.requested_by,
	a.attempts, COALESCE(a.error_message, ''), a.dispatched_at, a.completed_at,
	a.created_at, a.updated_at, d.serial_number,
	COALESCE(NULLIF(d.model_name, ''), ''), d.carrier, d.technology`

func (s *PgActionStore) Plan(ctx context.Context, plan ActionPlan) (Action, bool, error) {
	if s == nil || s.db == nil {
		return Action{}, false, ErrAccessGateDependencyMissing
	}
	if plan.DeviceID == uuid.Nil || plan.DecisionID == uuid.Nil || strings.TrimSpace(plan.IdempotencyKey) == "" {
		return Action{}, false, fmt.Errorf("plan RF action: invalid identity")
	}
	if !validActionContract(plan.ActionType, plan.Direction) {
		return Action{}, false, fmt.Errorf("plan RF action: invalid type/direction")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Action{}, false, fmt.Errorf("begin RF action plan: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lockKey := "device-access-action:" + plan.DeviceID.String()
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
			"id", "device_id", "decision_id", "action_type", "direction", "status",
			"recovery_of_action_id", "idempotency_key", "requested_by",
		).
		Values(
			id, plan.DeviceID, plan.DecisionID, plan.ActionType, plan.Direction,
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

type actionPlanQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findConflictingActionPlan(ctx context.Context, db actionPlanQuerier, plan ActionPlan) (uuid.UUID, error) {
	activeOrRetryable := sq.Or{
		sq.Eq{"status": []ActionStatus{ActionStatusPendingDispatch, ActionStatusDispatching}},
		sq.And{
			sq.Eq{"status": ActionStatusSucceeded},
			sq.Eq{"owned_rf_change": true},
			sq.Expr("jsonb_array_length(COALESCE(rf_change_paths, '[]'::jsonb)) > 0"),
		},
		sq.And{sq.Eq{"status": ActionStatusFailed}, sq.Lt{"attempts": defaultActionMaxAttempts}},
	}
	builder := storage.Psql.Select("id").From("device_access_actions").
		Where(sq.Eq{"device_id": plan.DeviceID, "action_type": plan.ActionType}).
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
		Join("devices d ON d.id = a.device_id").
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
	where := sq.And{}
	if value := strings.TrimSpace(filter.Carrier); value != "" {
		where = append(where, sq.Eq{"d.carrier": value})
	}
	if value := strings.TrimSpace(filter.SerialNumber); value != "" {
		where = append(where, sq.ILike{"d.serial_number": "%" + value + "%"})
	}
	if filter.Status != "" {
		where = append(where, sq.Eq{"a.status": filter.Status})
	}
	countBuilder := storage.Psql.Select("COUNT(*)").
		From("device_access_actions a").Join("devices d ON d.id = a.device_id").
		Where(where)
	countBuilder = authz.ApplyDeviceVisibilityFilter(countBuilder, "d.id", filter.VisibleGroups)
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build RF action count: %w", err)
	}
	var total int64
	if err := s.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count RF actions: %w", err)
	}
	listBuilder := storage.Psql.Select(actionColumns).
		From("device_access_actions a").Join("devices d ON d.id = a.device_id").
		Where(where).
		OrderBy("a.created_at DESC", "a.id DESC").
		Limit(uint64(pageSize)).Offset(uint64((page - 1) * pageSize))
	listBuilder = authz.ApplyDeviceVisibilityFilter(listBuilder, "d.id", filter.VisibleGroups)
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

func (s *PgActionStore) FindOwnedIsolation(ctx context.Context, deviceID uuid.UUID) (*Action, error) {
	query, args, err := storage.Psql.Select(actionColumns).
		From("device_access_actions a").
		Join("devices d ON d.id = a.device_id").
		Where(sq.Eq{
			"a.device_id": deviceID, "a.action_type": ActionTypeRFOff,
			"a.status": ActionStatusSucceeded, "a.owned_rf_change": true,
		}).
		Where("jsonb_array_length(COALESCE(a.rf_change_paths, '[]'::jsonb)) > 0").
		Where(`NOT EXISTS (
			SELECT 1 FROM device_access_actions recovery
			WHERE recovery.recovery_of_action_id = a.id
			AND recovery.action_type = 'rf_on'
			AND recovery.status IN ('pending_dispatch','dispatching','succeeded')
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
		Join("devices d ON d.id = a.device_id").
		Where(sq.Eq{
			"a.device_id":   deviceID,
			"a.action_type": ActionTypeRFOff,
			"a.status": []ActionStatus{
				ActionStatusPendingDispatch, ActionStatusDispatching,
				ActionStatusSucceeded, ActionStatusFailed,
			},
		}).
		Where(sq.Or{
			sq.NotEq{"a.status": ActionStatusFailed},
			sq.Lt{"a.attempts": defaultActionMaxAttempts},
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

func (s *PgActionStore) BeginDispatch(ctx context.Context, actionID uuid.UUID, at time.Time) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusPendingDispatch}, map[string]any{
		"status":   ActionStatusDispatching,
		"attempts": sq.Expr("attempts + 1"), "dispatched_at": at, "updated_at": at,
	}, "begin action dispatch")
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
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching}, map[string]any{
		"rf_change_paths": json.RawMessage(raw), "updated_at": at,
	}, "persist RF change paths", sq.Expr("rf_change_paths = '[]'::jsonb"))
}

func (s *PgActionStore) MarkVerifying(
	ctx context.Context,
	actionID, writeTaskID, readbackTaskID uuid.UUID,
	at time.Time,
) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusDispatching}, map[string]any{
		"device_task_id": readbackTaskID, "updated_at": at,
	}, "mark action RF readback", sq.Or{
		sq.Eq{"device_task_id": writeTaskID},
		sq.Expr("device_task_id IS NULL"),
	})
}

func (s *PgActionStore) MarkTerminal(
	ctx context.Context,
	actionID uuid.UUID,
	status ActionStatus,
	ownedRFChange bool,
	message string,
	at time.Time,
) error {
	if status != ActionStatusSucceeded && status != ActionStatusFailed && status != ActionStatusCancelled {
		return fmt.Errorf("mark RF action terminal: invalid status %q", status)
	}
	return s.update(ctx, actionID,
		[]ActionStatus{ActionStatusPendingDispatch, ActionStatusDispatching},
		map[string]any{
			"status": status, "owned_rf_change": ownedRFChange,
			"error_message": nullableString(strings.TrimSpace(message)),
			"completed_at":  at, "updated_at": at,
		}, "mark action terminal")
}

func (s *PgActionStore) ResetForRetry(ctx context.Context, actionID uuid.UUID) error {
	return s.update(ctx, actionID, []ActionStatus{ActionStatusFailed}, map[string]any{
		"status": ActionStatusPendingDispatch, "device_task_id": nil,
		"owned_rf_change": false, "error_message": nil,
		"dispatched_at": nil, "completed_at": nil, "updated_at": time.Now().UTC(),
	}, "reset action for retry")
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
	err := row.Scan(
		&item.ID, &item.DeviceID, &item.DecisionID, &item.ActionType, &item.Direction,
		&item.Status, &item.DeviceTaskID,
		&item.RecoveryOfActionID, &item.OwnedRFChange, &rfChangePaths, &item.IdempotencyKey, &item.RequestedBy,
		&item.Attempts, &item.ErrorMessage, &item.DispatchedAt, &item.CompletedAt,
		&item.CreatedAt, &item.UpdatedAt, &item.SerialNumber, &item.ProductName, &item.Carrier, &item.Technology,
	)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(rfChangePaths, &item.RFChangePaths); err != nil {
		return Action{}, fmt.Errorf("decode RF change paths: %w", err)
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
