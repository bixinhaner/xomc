package device

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	DeviceControlSourceGeofence = "geofence"
	DeviceControlSourceAccess   = "device_access"

	DeviceControlPhaseDeactivating   = "deactivating"
	DeviceControlPhaseVerifying      = "verifying"
	DeviceControlPhaseDeactivated    = "deactivated"
	DeviceControlPhasePartialFailed  = "partial_failed"
	DeviceControlPhaseFailed         = "failed"
	DeviceControlPhaseRecovering     = "recovering"
	DeviceControlPhaseRecoveryFailed = "recovery_failed"
)

// DeviceControlSummary is the device-facing projection of an authoritative
// geofence control action. It intentionally carries no inferred alarm cause.
type DeviceControlSummary struct {
	SourceType  string     `json:"source_type"`
	SourceID    *uuid.UUID `json:"source_id,omitempty"`
	SourceName  string     `json:"source_name"`
	ReasonCode  string     `json:"reason_code"`
	Phase       string     `json:"phase"`
	ActionID    uuid.UUID  `json:"action_id"`
	TriggeredAt time.Time  `json:"triggered_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
}

type DeviceControlSummaryReader interface {
	ListCurrentByDeviceIDs(
		context.Context,
		[]uuid.UUID,
	) (map[uuid.UUID]DeviceControlSummary, error)
	ListHistoryByDeviceID(
		context.Context,
		uuid.UUID,
		int,
		int,
	) (*DeviceControlActionHistoryList, error)
}

type DeviceControlParameterState struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

type accessParameterValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DeviceControlEvaluationEvidence struct {
	ID                 uuid.UUID `json:"id"`
	ObservationVersion int64     `json:"observation_version"`
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	ObservedAt         time.Time `json:"observed_at"`
	RuleType           string    `json:"rule_type"`
	SignedDistanceM    *float64  `json:"signed_distance_meters,omitempty"`
	ConfirmedState     string    `json:"confirmed_state"`
	ReasonCode         string    `json:"reason_code"`
}

type DeviceControlActionHistory struct {
	ID                    uuid.UUID                        `json:"id"`
	ParentActionID        *uuid.UUID                       `json:"parent_action_id,omitempty"`
	SourceType            string                           `json:"source_type"`
	SourceID              *uuid.UUID                       `json:"source_id,omitempty"`
	SourceName            string                           `json:"source_name"`
	ReasonCode            string                           `json:"reason_code"`
	ObservationVersion    *int64                           `json:"observation_version,omitempty"`
	EffectiveStateVersion int64                            `json:"effective_state_version"`
	ActionType            string                           `json:"action_type"`
	Status                string                           `json:"status"`
	BeforeState           []DeviceControlParameterState    `json:"before_state"`
	RequestedState        []DeviceControlParameterState    `json:"requested_state"`
	VerifiedState         []DeviceControlParameterState    `json:"verified_state"`
	Evaluation            *DeviceControlEvaluationEvidence `json:"evaluation,omitempty"`
	LastError             string                           `json:"last_error,omitempty"`
	CreatedAt             time.Time                        `json:"created_at"`
	UpdatedAt             time.Time                        `json:"updated_at"`
	CompletedAt           *time.Time                       `json:"completed_at,omitempty"`
}

type DeviceControlActionHistoryList struct {
	Items    []DeviceControlActionHistory `json:"items"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type PgDeviceControlSummaryReader struct {
	pool *pgxpool.Pool
}

func NewPgDeviceControlSummaryReader(pool *pgxpool.Pool) *PgDeviceControlSummaryReader {
	return &PgDeviceControlSummaryReader{pool: pool}
}

func buildCurrentDeviceControlSummariesQuery(
	deviceIDs []uuid.UUID,
) (string, []any, error) {
	// Keep nested builders on question-mark placeholders. The outer PostgreSQL
	// builder performs the single final Dollar conversion; converting a nested
	// query early makes its $1 collide with the outer query's parameters.
	recovery := sq.Select(
		"child.id",
		"child.status",
		"child.last_error",
		"child.created_at",
		"child.completed_at",
	).
		From("geofence_control_actions child").
		Where("child.parent_action_id = deactivation.id").
		Where(sq.Eq{"child.action_type": "activate"}).
		OrderBy("child.created_at DESC").
		Limit(1)

	return storage.Psql.
		Select(
			"deactivation.device_id",
			"deactivation.id",
			"COALESCE(deactivation.geofence_id, binding.geofence_id) AS source_id",
			"COALESCE(definition.name, '') AS source_name",
			"deactivation.trigger_reason_code",
			"deactivation.status",
			"deactivation.last_error",
			"deactivation.created_at",
			"deactivation.completed_at",
			"recovery.id",
			"recovery.status",
			"recovery.last_error",
			"recovery.created_at",
			"recovery.completed_at",
		).
		Options("DISTINCT ON (deactivation.device_id)").
		From("geofence_control_actions deactivation").
		LeftJoin("device_geofence_bindings binding ON binding.id = deactivation.binding_id").
		LeftJoin("geofence_definitions definition ON definition.id = COALESCE(deactivation.geofence_id, binding.geofence_id)").
		JoinClause(sq.Expr("LEFT JOIN LATERAL (?) recovery ON TRUE", recovery)).
		Where(sq.Eq{
			"deactivation.device_id":   deviceIDs,
			"deactivation.action_type": "deactivate",
		}).
		OrderBy("deactivation.device_id", "deactivation.created_at DESC").
		ToSql()
}

func resolveDeviceControlPhase(
	deactivationStatus string,
	recoveryStatus *string,
) (string, bool) {
	if recoveryStatus != nil {
		switch *recoveryStatus {
		case "verified":
			return "", false
		case "failed", "partial_failed":
			return DeviceControlPhaseRecoveryFailed, true
		default:
			return DeviceControlPhaseRecovering, true
		}
	}

	switch deactivationStatus {
	case "pending", "executing":
		return DeviceControlPhaseDeactivating, true
	case "verifying":
		return DeviceControlPhaseVerifying, true
	case "verified":
		return DeviceControlPhaseDeactivated, true
	case "partial_failed":
		return DeviceControlPhasePartialFailed, true
	default:
		return DeviceControlPhaseFailed, true
	}
}

func validDeviceControlPhase(phase string) bool {
	switch phase {
	case DeviceControlPhaseDeactivating, DeviceControlPhaseVerifying,
		DeviceControlPhaseDeactivated, DeviceControlPhasePartialFailed,
		DeviceControlPhaseFailed, DeviceControlPhaseRecovering,
		DeviceControlPhaseRecoveryFailed:
		return true
	default:
		return false
	}
}

func currentDeviceControlFilterCondition(filter DeviceFilter) sq.Sqlizer {
	if filter.ControlSource == nil && len(filter.ControlPhases) == 0 {
		return nil
	}
	if filter.ControlSource != nil && *filter.ControlSource != DeviceControlSourceGeofence {
		return sq.Expr("FALSE")
	}

	// This condition is embedded into list/count builders that may already have
	// parameters. Leave all nested placeholders as '?' so the parent builder can
	// number the complete statement exactly once.
	recovery := sq.Select("child.id", "child.status").
		From("geofence_control_actions child").
		Where("child.parent_action_id = deactivation.id").
		Where(sq.Eq{"child.action_type": "activate"}).
		OrderBy("child.created_at DESC").
		Limit(1)
	current := sq.Select("1").
		From("geofence_control_actions deactivation").
		JoinClause(sq.Expr("LEFT JOIN LATERAL (?) recovery ON TRUE", recovery)).
		Where(`deactivation.id = (
			SELECT latest.id
			FROM geofence_control_actions latest
			WHERE latest.device_id = d.id AND latest.action_type = 'deactivate'
			ORDER BY latest.created_at DESC
			LIMIT 1
		)`)

	phaseConditions := make(sq.Or, 0, len(filter.ControlPhases))
	for _, phase := range filter.ControlPhases {
		switch phase {
		case DeviceControlPhaseDeactivating:
			phaseConditions = append(phaseConditions, sq.And{
				sq.Eq{"recovery.id": nil},
				sq.Eq{"deactivation.status": []string{"pending", "executing"}},
			})
		case DeviceControlPhaseVerifying:
			phaseConditions = append(phaseConditions, sq.And{
				sq.Eq{"recovery.id": nil}, sq.Eq{"deactivation.status": "verifying"},
			})
		case DeviceControlPhaseDeactivated:
			phaseConditions = append(phaseConditions, sq.And{
				sq.Eq{"recovery.id": nil}, sq.Eq{"deactivation.status": "verified"},
			})
		case DeviceControlPhasePartialFailed:
			phaseConditions = append(phaseConditions, sq.And{
				sq.Eq{"recovery.id": nil}, sq.Eq{"deactivation.status": "partial_failed"},
			})
		case DeviceControlPhaseFailed:
			phaseConditions = append(phaseConditions, sq.And{
				sq.Eq{"recovery.id": nil}, sq.Eq{"deactivation.status": "failed"},
			})
		case DeviceControlPhaseRecovering:
			phaseConditions = append(phaseConditions, sq.Eq{
				"recovery.status": []string{"pending", "executing", "verifying"},
			})
		case DeviceControlPhaseRecoveryFailed:
			phaseConditions = append(phaseConditions, sq.Eq{
				"recovery.status": []string{"failed", "partial_failed"},
			})
		}
	}
	if len(phaseConditions) > 0 {
		current = current.Where(phaseConditions)
	} else {
		current = current.Where(sq.Or{
			sq.Eq{"recovery.id": nil},
			sq.NotEq{"recovery.status": "verified"},
		})
	}
	return sq.Expr("EXISTS (?)", current)
}

func (r *PgDeviceControlSummaryReader) ListCurrentByDeviceIDs(
	ctx context.Context,
	deviceIDs []uuid.UUID,
) (map[uuid.UUID]DeviceControlSummary, error) {
	result := make(map[uuid.UUID]DeviceControlSummary)
	if len(deviceIDs) == 0 {
		return result, nil
	}
	query, args, err := buildCurrentDeviceControlSummariesQuery(deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("build current device control summaries: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query current device control summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			deviceID           uuid.UUID
			actionID           uuid.UUID
			sourceID           *uuid.UUID
			sourceName         string
			reasonCode         string
			deactivationStatus string
			deactivationError  string
			deactivationAt     time.Time
			deactivationDoneAt *time.Time
			recoveryID         *uuid.UUID
			recoveryStatus     *string
			recoveryError      *string
			recoveryAt         *time.Time
			recoveryDoneAt     *time.Time
		)
		if err := rows.Scan(
			&deviceID,
			&actionID,
			&sourceID,
			&sourceName,
			&reasonCode,
			&deactivationStatus,
			&deactivationError,
			&deactivationAt,
			&deactivationDoneAt,
			&recoveryID,
			&recoveryStatus,
			&recoveryError,
			&recoveryAt,
			&recoveryDoneAt,
		); err != nil {
			return nil, fmt.Errorf("scan current device control summary: %w", err)
		}

		phase, visible := resolveDeviceControlPhase(deactivationStatus, recoveryStatus)
		if !visible {
			continue
		}
		summary := DeviceControlSummary{
			SourceType:  DeviceControlSourceGeofence,
			SourceID:    sourceID,
			SourceName:  sourceName,
			ReasonCode:  reasonCode,
			Phase:       phase,
			ActionID:    actionID,
			TriggeredAt: deactivationAt,
			CompletedAt: deactivationDoneAt,
			LastError:   deactivationError,
		}
		if recoveryID != nil {
			summary.ActionID = *recoveryID
			summary.CompletedAt = recoveryDoneAt
			if recoveryError != nil {
				summary.LastError = *recoveryError
			}
		}
		result[deviceID] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current device control summaries: %w", err)
	}
	return result, nil
}

func (r *PgDeviceControlSummaryReader) ListHistoryByDeviceID(
	ctx context.Context,
	deviceID uuid.UUID,
	page int,
	pageSize int,
) (*DeviceControlActionHistoryList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	countQuery, countArgs, err := storage.Psql.
		Select("COUNT(*)").
		From("geofence_control_actions action").
		Where(sq.Eq{"action.device_id": deviceID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device control history count: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count device control history: %w", err)
	}
	accessCountQuery, accessCountArgs, err := storage.Psql.Select("COUNT(*)").
		From("device_access_actions action").Where(sq.Eq{"action.device_id": deviceID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device access control history count: %w", err)
	}
	var accessTotal int64
	if err := r.pool.QueryRow(ctx, accessCountQuery, accessCountArgs...).Scan(&accessTotal); err != nil {
		return nil, fmt.Errorf("count device access control history: %w", err)
	}
	total += accessTotal
	offset := (page - 1) * pageSize
	fetchLimit := offset + pageSize

	query, args, err := storage.Psql.
		Select(
			"action.id", "action.parent_action_id",
			"COALESCE(action.geofence_id, binding.geofence_id)",
			"COALESCE(definition.name, '')", "action.trigger_reason_code",
			"action.trigger_observation_version", "action.effective_state_version",
			"action.action_type", "action.status", "action.before_state",
			"action.requested_state", "action.verified_state", "action.last_error",
			"action.created_at", "action.updated_at", "action.completed_at",
			"evaluation.id", "evaluation.observation_version", "evaluation.latitude",
			"evaluation.longitude", "evaluation.observed_at", "evaluation.rule_type",
			"evaluation.signed_distance_meters", "evaluation.confirmed_state",
			"evaluation.reason_code",
		).
		From("geofence_control_actions action").
		LeftJoin("device_geofence_bindings binding ON binding.id = action.binding_id").
		LeftJoin("geofence_definitions definition ON definition.id = COALESCE(action.geofence_id, binding.geofence_id)").
		LeftJoin("geofence_evaluations evaluation ON evaluation.id = action.trigger_evaluation_id").
		Where(sq.Eq{"action.device_id": deviceID}).
		OrderBy("action.created_at DESC", "action.id DESC").
		Limit(uint64(fetchLimit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device control history: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device control history: %w", err)
	}
	defer rows.Close()

	items := make([]DeviceControlActionHistory, 0, pageSize)
	for rows.Next() {
		var (
			item                                    DeviceControlActionHistory
			beforeRaw, requestedRaw, verifiedRaw    []byte
			evaluationID                            *uuid.UUID
			evaluationObservation                   *int64
			evaluationLatitude, evaluationLongitude *float64
			evaluationObservedAt                    *time.Time
			evaluationRuleType, evaluationState     *string
			evaluationDistance                      *float64
			evaluationReason                        *string
		)
		if err := rows.Scan(
			&item.ID, &item.ParentActionID, &item.SourceID, &item.SourceName,
			&item.ReasonCode, &item.ObservationVersion, &item.EffectiveStateVersion,
			&item.ActionType, &item.Status, &beforeRaw, &requestedRaw, &verifiedRaw,
			&item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.CompletedAt,
			&evaluationID, &evaluationObservation, &evaluationLatitude,
			&evaluationLongitude, &evaluationObservedAt, &evaluationRuleType,
			&evaluationDistance, &evaluationState, &evaluationReason,
		); err != nil {
			return nil, fmt.Errorf("scan device control history: %w", err)
		}
		item.SourceType = DeviceControlSourceGeofence
		if err := unmarshalDeviceControlState(beforeRaw, &item.BeforeState); err != nil {
			return nil, fmt.Errorf("decode device control before state: %w", err)
		}
		if err := unmarshalDeviceControlState(requestedRaw, &item.RequestedState); err != nil {
			return nil, fmt.Errorf("decode device control requested state: %w", err)
		}
		if err := unmarshalDeviceControlState(verifiedRaw, &item.VerifiedState); err != nil {
			return nil, fmt.Errorf("decode device control verified state: %w", err)
		}
		if evaluationID != nil && evaluationObservation != nil &&
			evaluationLatitude != nil && evaluationLongitude != nil &&
			evaluationObservedAt != nil && evaluationRuleType != nil &&
			evaluationState != nil && evaluationReason != nil {
			item.Evaluation = &DeviceControlEvaluationEvidence{
				ID: *evaluationID, ObservationVersion: *evaluationObservation,
				Latitude: *evaluationLatitude, Longitude: *evaluationLongitude,
				ObservedAt: *evaluationObservedAt, RuleType: *evaluationRuleType,
				SignedDistanceM: evaluationDistance, ConfirmedState: *evaluationState,
				ReasonCode: *evaluationReason,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device control history: %w", err)
	}
	accessItems, err := r.listDeviceAccessControlHistory(ctx, deviceID, fetchLimit)
	if err != nil {
		return nil, err
	}
	items = append(items, accessItems...)
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].CreatedAt.Equal(items[right].CreatedAt) {
			return items[left].ID.String() > items[right].ID.String()
		}
		return items[left].CreatedAt.After(items[right].CreatedAt)
	})
	if offset >= len(items) {
		items = []DeviceControlActionHistory{}
	} else {
		end := offset + pageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[offset:end]
	}
	return &DeviceControlActionHistoryList{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (r *PgDeviceControlSummaryReader) listDeviceAccessControlHistory(
	ctx context.Context,
	deviceID uuid.UUID,
	limit int,
) ([]DeviceControlActionHistory, error) {
	baseline := latestAccessAttemptSummary("baseline_gpv", "response_summary")
	write := latestAccessAttemptSummary("spv", "request_summary")
	readback := latestAccessAttemptSummary("readback_gpv", "response_summary")
	query, args, err := storage.Psql.Select(
		"action.id", "action.recovery_of_action_id", "decision.id", "decision.reason_code",
		"decision.decision_version", "action.action_type", "action.status",
		"baseline.summary", "write_attempt.summary", "readback.summary",
		"COALESCE(action.error_message, '')", "action.created_at", "action.updated_at", "action.completed_at",
	).From("device_access_actions action").
		Join("device_access_decisions decision ON decision.id = action.decision_id").
		JoinClause(sq.Expr("LEFT JOIN LATERAL (?) baseline ON TRUE", baseline)).
		JoinClause(sq.Expr("LEFT JOIN LATERAL (?) write_attempt ON TRUE", write)).
		JoinClause(sq.Expr("LEFT JOIN LATERAL (?) readback ON TRUE", readback)).
		Where(sq.Eq{"action.device_id": deviceID}).
		OrderBy("action.created_at DESC", "action.id DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device access control history: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query device access control history: %w", err)
	}
	defer rows.Close()
	items := make([]DeviceControlActionHistory, 0, limit)
	for rows.Next() {
		var item DeviceControlActionHistory
		var actionType, actionStatus string
		var baselineRaw, writeRaw, readbackRaw []byte
		if err := rows.Scan(
			&item.ID, &item.ParentActionID, &item.SourceID, &item.ReasonCode,
			&item.EffectiveStateVersion, &actionType, &actionStatus,
			&baselineRaw, &writeRaw, &readbackRaw, &item.LastError,
			&item.CreatedAt, &item.UpdatedAt, &item.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device access control history: %w", err)
		}
		item.SourceType = DeviceControlSourceAccess
		item.ActionType = mapAccessActionType(actionType)
		item.BeforeState = accessResponseParameterStates(baselineRaw)
		item.RequestedState = accessRequestParameterStates(writeRaw)
		item.VerifiedState = accessResponseParameterStates(readbackRaw)
		if len(item.VerifiedState) == 0 && len(item.RequestedState) == 0 {
			// A containment action that found RF already disabled completes after
			// the baseline GPV; that same real observation is its verification.
			item.VerifiedState = append([]DeviceControlParameterState(nil), item.BeforeState...)
		}
		item.Status = mapAccessActionStatus(actionStatus, len(item.VerifiedState) > 0)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device access control history: %w", err)
	}
	return items, nil
}

func latestAccessAttemptSummary(phase, column string) sq.SelectBuilder {
	selectedColumn := "attempt.response_summary AS summary"
	if column == "request_summary" {
		selectedColumn = "attempt.request_summary AS summary"
	}
	return sq.Select(selectedColumn).From("device_access_action_attempts attempt").
		Where("attempt.action_id = action.id").Where(sq.Eq{"attempt.phase": phase, "attempt.status": "succeeded"}).
		OrderBy("attempt.attempt_no DESC", "attempt.completed_at DESC NULLS LAST", "attempt.id DESC").Limit(1)
}

func mapAccessActionType(actionType string) string {
	if actionType == "rf_on" {
		return "activate"
	}
	return "deactivate"
}

func mapAccessActionStatus(status string, hasVerifiedGPV bool) string {
	switch status {
	case "pending_dispatch", "retry_wait":
		return "pending"
	case "dispatching":
		return "executing"
	case "verifying":
		return "verifying"
	case "succeeded":
		if hasVerifiedGPV {
			return "verified"
		}
		return "evidence_missing"
	default:
		return "failed"
	}
}

func accessResponseParameterStates(raw []byte) []DeviceControlParameterState {
	var wrapper struct {
		Result struct {
			Values []accessParameterValue `json:"standard_parameter_values"`
		} `json:"result"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &wrapper) != nil {
		return []DeviceControlParameterState{}
	}
	return accessParameterStates(wrapper.Result.Values)
}

func accessRequestParameterStates(raw []byte) []DeviceControlParameterState {
	var wrapper struct {
		Parameters struct {
			Values []accessParameterValue `json:"values"`
		} `json:"parameters"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &wrapper) != nil {
		return []DeviceControlParameterState{}
	}
	return accessParameterStates(wrapper.Parameters.Values)
}

func accessParameterStates(values []accessParameterValue) []DeviceControlParameterState {
	states := make([]DeviceControlParameterState, 0, len(values))
	for _, value := range values {
		if value.Name != "" {
			states = append(states, DeviceControlParameterState{Path: value.Name, Value: value.Value})
		}
	}
	return states
}

func unmarshalDeviceControlState(data []byte, target *[]DeviceControlParameterState) error {
	if len(data) == 0 {
		*target = []DeviceControlParameterState{}
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	if *target == nil {
		*target = []DeviceControlParameterState{}
	}
	return nil
}
