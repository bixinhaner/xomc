package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/outbox"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type evaluationStatus string
type evaluationHealth string

const (
	evaluationStatusCompleted evaluationStatus = "completed"
	evaluationStatusFailed    evaluationStatus = "failed"

	evaluationHealthHealthy evaluationHealth = "healthy"
	evaluationHealthStale   evaluationHealth = "stale"
	evaluationHealthFailed  evaluationHealth = "failed"
)

type lockedEffectiveState struct {
	State                  EffectiveState
	ActionLevel            ActionLevel
	StateVersion           int64
	LastObservationVersion int64
	TriggerBindingID       *uuid.UUID
}

type CoordinatorResult struct {
	Processed             bool
	NoOp                  bool
	Failed                bool
	Reason                string
	EvaluationCount       int
	EffectiveState        EffectiveState
	EffectiveStateVersion int64
}

type CoordinatorRepository interface {
	EvaluateLocation(
		context.Context,
		event.DeviceLocationObservedPayload,
		time.Time,
	) (CoordinatorResult, error)
	ReevaluateLatest(context.Context, uuid.UUID, time.Time) (CoordinatorResult, error)
}

type PgCoordinatorRepository struct {
	db     storage.DB
	outbox *outbox.Repository
	newID  func() uuid.UUID
}

func NewPgCoordinatorRepository(db storage.DB) *PgCoordinatorRepository {
	return &PgCoordinatorRepository{
		db:     db,
		outbox: outbox.NewRepository(),
		newID:  uuid.New,
	}
}

type coordinatorBindingSnapshot struct {
	BindingID         uuid.UUID
	GeofenceID        uuid.UUID
	RuleType          RuleType
	GeofenceVersionID uuid.UUID
	Geometry          json.RawMessage
	Policy            json.RawMessage
	EffectiveMode     RuntimeMode
}

type lockedBindingState struct {
	BindingID              uuid.UUID
	ConfirmedState         ConfirmedState
	CandidateState         CandidateState
	CandidateCount         int
	CandidateSince         *time.Time
	StateVersion           int64
	LastObservationVersion int64
}

type coordinatorEvaluation struct {
	ID                         uuid.UUID
	BindingID                  uuid.UUID
	DeviceID                   uuid.UUID
	GeofenceID                 uuid.UUID
	GeofenceVersionID          uuid.UUID
	ObservationVersion         int64
	Latitude                   float64
	Longitude                  float64
	GPSHeight                  *float64
	ObservedAt                 time.Time
	ReceivedAt                 time.Time
	DeviceReportedAt           *time.Time
	GPSLockStatus              *string
	SatelliteCount             *int
	AccuracyMeters             *float64
	SourcePath                 string
	PreviousObservationVersion *int64
	MovementDistanceMeters     *float64
	ElapsedSeconds             *float64
	ImpliedSpeedMPS            *float64
	RuleType                   RuleType
	RawPosition                RawPosition
	SignedDistanceMeters       *float64
	PreviousConfirmedState     ConfirmedState
	ConfirmedState             ConfirmedState
	CandidateState             CandidateState
	CandidateCount             int
	CandidateSince             *time.Time
	StateEdge                  bool
	Status                     evaluationStatus
	ReasonCode                 string
	FailureStage               string
	ErrorCode                  string
	ErrorSummary               string
	EvaluatedAt                time.Time
}

type effectiveStateWrite struct {
	DeviceID               uuid.UUID
	State                  EffectiveState
	ActionLevel            ActionLevel
	TriggerBindingID       *uuid.UUID
	ObservationVersion     int64
	StateChanged           bool
	EvaluationHealth       evaluationHealth
	ErrorCode              string
	SuccessfulEvaluationAt time.Time
	UpdatedAt              time.Time
}

func buildLockEffectiveStateQuery(deviceID uuid.UUID) (string, []any, error) {
	query, args, err := storage.Psql.
		Select(
			"effective_state",
			"required_action_level",
			"state_version",
			"COALESCE(last_observation_version, 0)",
			"trigger_binding_id",
		).
		From("device_geofence_effective_states").
		Where(sq.Eq{"device_id": deviceID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock effective geofence state: %w", err)
	}
	return query, args, nil
}

func buildLatestObservationQuery(deviceID uuid.UUID) (string, []any, error) {
	query, args, err := storage.Psql.
		Select(
			"o.device_id",
			"d.serial_number",
			"d.carrier",
			"o.version",
			"o.latitude",
			"o.longitude",
			"o.observed_at",
			"o.received_at",
			"o.source_path",
		).
		From("device_location_observations o").
		Join("devices d ON d.id = o.device_id").
		Where(sq.Eq{
			"o.device_id":  deviceID,
			"d.deleted_at": nil,
		}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build latest geofence observation query: %w", err)
	}
	return query, args, nil
}

func (r *PgCoordinatorRepository) EvaluateLocation(
	ctx context.Context,
	payload event.DeviceLocationObservedPayload,
	evaluatedAt time.Time,
) (CoordinatorResult, error) {
	return r.evaluateLocation(ctx, payload, evaluatedAt, false)
}

func (r *PgCoordinatorRepository) ReevaluateLatest(
	ctx context.Context,
	deviceID uuid.UUID,
	evaluatedAt time.Time,
) (CoordinatorResult, error) {
	var payload event.DeviceLocationObservedPayload
	query, args, err := buildLatestObservationQuery(deviceID)
	if err != nil {
		return CoordinatorResult{}, err
	}
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&payload.DeviceID,
		&payload.SerialNumber,
		&payload.Carrier,
		&payload.ObservationVersion,
		&payload.Latitude,
		&payload.Longitude,
		&payload.ObservedAt,
		&payload.ReceivedAt,
		&payload.SourcePath,
	)
	if err == pgx.ErrNoRows {
		return CoordinatorResult{NoOp: true, Reason: "no_latest_observation"}, nil
	}
	if err != nil {
		return CoordinatorResult{}, fmt.Errorf("read latest geofence observation: %w", err)
	}
	return r.evaluateLocation(ctx, payload, evaluatedAt, true)
}

func (r *PgCoordinatorRepository) evaluateLocation(
	ctx context.Context,
	payload event.DeviceLocationObservedPayload,
	evaluatedAt time.Time,
	force bool,
) (CoordinatorResult, error) {
	if r == nil || r.db == nil {
		return CoordinatorResult{}, fmt.Errorf(
			"evaluate geofence location: database is required",
		)
	}
	if payload.DeviceID == uuid.Nil {
		return CoordinatorResult{}, fmt.Errorf(
			"evaluate geofence location: device ID is required",
		)
	}
	if payload.ObservationVersion <= 0 {
		return CoordinatorResult{}, fmt.Errorf(
			"evaluate geofence location: observation version must be positive",
		)
	}
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CoordinatorResult{}, fmt.Errorf(
			"begin geofence evaluation transaction: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	previous, found, err := lockEffectiveState(ctx, tx, payload.DeviceID)
	if err != nil {
		return CoordinatorResult{}, err
	}
	if !found {
		if err := tx.Commit(ctx); err != nil {
			return CoordinatorResult{}, fmt.Errorf(
				"commit unmanaged geofence evaluation no-op: %w",
				err,
			)
		}
		return CoordinatorResult{
			NoOp:   true,
			Reason: "no_effective_state",
		}, nil
	}
	if !force && payload.ObservationVersion <= previous.LastObservationVersion {
		reason := "older_observation"
		if payload.ObservationVersion == previous.LastObservationVersion {
			reason = "duplicate_observation"
		}
		if err := tx.Commit(ctx); err != nil {
			return CoordinatorResult{}, fmt.Errorf(
				"commit replay-protected geofence evaluation: %w",
				err,
			)
		}
		return CoordinatorResult{
			NoOp:                  true,
			Reason:                reason,
			EffectiveState:        previous.State,
			EffectiveStateVersion: previous.StateVersion,
		}, nil
	}

	snapshots, err := loadActiveBindingSnapshots(
		ctx,
		tx,
		payload.DeviceID,
		payload.Carrier,
	)
	if err != nil {
		return CoordinatorResult{}, err
	}
	if len(snapshots) == 0 {
		next := EffectiveSnapshot{
			State:               EffectiveStateUnmanaged,
			RequiredActionLevel: ActionLevelNone,
		}
		stateChanged := effectiveStateChanged(previous, next)
		if err := updateEffectiveState(
			ctx,
			tx,
			effectiveStateWrite{
				DeviceID:               payload.DeviceID,
				State:                  next.State,
				ActionLevel:            next.RequiredActionLevel,
				ObservationVersion:     payload.ObservationVersion,
				StateChanged:           stateChanged,
				EvaluationHealth:       evaluationHealthHealthy,
				SuccessfulEvaluationAt: evaluatedAt,
				UpdatedAt:              evaluatedAt,
			},
		); err != nil {
			return CoordinatorResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return CoordinatorResult{}, fmt.Errorf(
				"commit unmanaged geofence evaluation: %w",
				err,
			)
		}
		version := previous.StateVersion
		if stateChanged {
			version++
		}
		return CoordinatorResult{
			Processed:             true,
			Reason:                "no_active_bindings",
			EffectiveState:        next.State,
			EffectiveStateVersion: version,
		}, nil
	}

	bindingIDs := make([]uuid.UUID, 0, len(snapshots))
	for _, snapshot := range snapshots {
		bindingIDs = append(bindingIDs, snapshot.BindingID)
	}
	states, err := lockBindingStates(ctx, tx, bindingIDs)
	if err != nil {
		return CoordinatorResult{}, err
	}
	if len(states) != len(snapshots) {
		return CoordinatorResult{}, fmt.Errorf(
			"lock geofence binding states: got %d states for %d active bindings",
			len(states),
			len(snapshots),
		)
	}

	evaluations := make([]coordinatorEvaluation, 0, len(snapshots))
	aggregates := make([]BindingEvaluation, 0, len(snapshots))
	stateVersions := make(map[uuid.UUID]int64, len(snapshots))
	for _, snapshot := range snapshots {
		state, ok := states[snapshot.BindingID]
		if !ok {
			return CoordinatorResult{}, fmt.Errorf(
				"lock geofence binding state %s: missing row",
				snapshot.BindingID,
			)
		}
		policy, actionLevel, err := decodeEvaluationPolicy(snapshot.Policy)
		if err != nil {
			return r.persistDeterministicFailure(
				ctx,
				tx,
				payload,
				evaluatedAt,
				previous,
				snapshot,
				state,
				"policy_decode",
				"invalid_policy",
				err,
			)
		}
		actionLevel = effectiveActionLevel(snapshot.EffectiveMode, actionLevel)
		result, err := Evaluate(EvaluationInput{
			RuleType: snapshot.RuleType,
			Geometry: snapshot.Geometry,
			Position: PositionSnapshot{
				Longitude:              payload.Longitude,
				Latitude:               payload.Latitude,
				ObservationVersion:     payload.ObservationVersion,
				ObservedAt:             payload.ObservedAt,
				ReceivedAt:             payload.ReceivedAt,
				DeviceReportedAt:       payload.DeviceReportedAt,
				MovementDistanceMeters: payload.MovementDistanceMeters,
				ImpliedSpeedMPS:        payload.ImpliedSpeedMPS,
			},
			Previous: EvaluationState{
				ConfirmedState:         state.ConfirmedState,
				CandidateState:         state.CandidateState,
				CandidateCount:         state.CandidateCount,
				CandidateSince:         state.CandidateSince,
				LastObservationVersion: state.LastObservationVersion,
			},
			Policy:      policy,
			EvaluatedAt: evaluatedAt,
		})
		if err != nil {
			return r.persistDeterministicFailure(
				ctx,
				tx,
				payload,
				evaluatedAt,
				previous,
				snapshot,
				state,
				"geometry_evaluation",
				"invalid_geometry",
				err,
			)
		}
		evaluation := completedCoordinatorEvaluation(
			r.newID(),
			payload,
			evaluatedAt,
			snapshot,
			state,
			result,
		)
		evaluations = append(evaluations, evaluation)
		aggregates = append(aggregates, BindingEvaluation{
			BindingID:      snapshot.BindingID,
			ConfirmedState: result.ConfirmedState,
			ActionLevel:    actionLevel,
		})
		stateVersions[snapshot.BindingID] = state.StateVersion + 1
	}

	for _, evaluation := range evaluations {
		if err := insertCoordinatorEvaluation(ctx, tx, evaluation); err != nil {
			return CoordinatorResult{}, err
		}
		if err := updateBindingState(ctx, tx, evaluation); err != nil {
			return CoordinatorResult{}, err
		}
		records, err := buildEvaluationOutboxRecords(
			evaluation,
			payload.SerialNumber,
			payload.Carrier,
			stateVersions[evaluation.BindingID],
		)
		if err != nil {
			return CoordinatorResult{}, err
		}
		for _, record := range records {
			if err := r.outbox.InsertTx(ctx, tx, record); err != nil {
				return CoordinatorResult{}, fmt.Errorf(
					"enqueue geofence evaluation event: %w",
					err,
				)
			}
		}
	}

	next := aggregateEffectiveState(aggregates)
	stateChanged := effectiveStateChanged(previous, next)
	var triggerEvaluationID *uuid.UUID
	if next.TriggerBindingID != nil {
		for index := range evaluations {
			if evaluations[index].BindingID == *next.TriggerBindingID {
				evaluationID := evaluations[index].ID
				triggerEvaluationID = &evaluationID
				break
			}
		}
	}
	deviceRecord, hasDeviceEdge, err := buildDeviceEdgeOutboxRecord(
		payload.DeviceID,
		payload.SerialNumber,
		payload.Carrier,
		payload.ObservationVersion,
		triggerEvaluationID,
		previous,
		next,
		evaluatedAt,
	)
	if err != nil {
		return CoordinatorResult{}, err
	}
	if err := updateEffectiveState(ctx, tx, effectiveStateWrite{
		DeviceID:               payload.DeviceID,
		State:                  next.State,
		ActionLevel:            next.RequiredActionLevel,
		TriggerBindingID:       next.TriggerBindingID,
		ObservationVersion:     payload.ObservationVersion,
		StateChanged:           stateChanged,
		EvaluationHealth:       evaluationHealthHealthy,
		SuccessfulEvaluationAt: evaluatedAt,
		UpdatedAt:              evaluatedAt,
	}); err != nil {
		return CoordinatorResult{}, err
	}
	if hasDeviceEdge {
		if err := r.outbox.InsertTx(ctx, tx, deviceRecord); err != nil {
			return CoordinatorResult{}, fmt.Errorf(
				"enqueue geofence device state event: %w",
				err,
			)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return CoordinatorResult{}, fmt.Errorf(
			"commit geofence evaluation transaction: %w",
			err,
		)
	}
	version := previous.StateVersion
	if stateChanged {
		version++
	}
	return CoordinatorResult{
		Processed:             true,
		Reason:                "evaluated",
		EvaluationCount:       len(evaluations),
		EffectiveState:        next.State,
		EffectiveStateVersion: version,
	}, nil
}

func lockEffectiveState(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
) (lockedEffectiveState, bool, error) {
	query, args, err := buildLockEffectiveStateQuery(deviceID)
	if err != nil {
		return lockedEffectiveState{}, false, err
	}
	var state lockedEffectiveState
	err = tx.QueryRow(ctx, query, args...).Scan(
		&state.State,
		&state.ActionLevel,
		&state.StateVersion,
		&state.LastObservationVersion,
		&state.TriggerBindingID,
	)
	if err == pgx.ErrNoRows {
		return lockedEffectiveState{}, false, nil
	}
	if err != nil {
		return lockedEffectiveState{}, false,
			fmt.Errorf("lock effective geofence state: %w", err)
	}
	return state, true, nil
}

func loadActiveBindingSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	deviceID uuid.UUID,
	carrier string,
) ([]coordinatorBindingSnapshot, error) {
	query, args, err := buildActiveBindingSnapshotQuery(deviceID, carrier)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("snapshot active geofence bindings: %w", err)
	}
	defer rows.Close()
	snapshots := make([]coordinatorBindingSnapshot, 0, 2)
	for rows.Next() {
		var snapshot coordinatorBindingSnapshot
		if err := rows.Scan(
			&snapshot.BindingID,
			&snapshot.GeofenceID,
			&snapshot.RuleType,
			&snapshot.GeofenceVersionID,
			&snapshot.Geometry,
			&snapshot.Policy,
			&snapshot.EffectiveMode,
		); err != nil {
			return nil, fmt.Errorf("scan active geofence binding: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active geofence bindings: %w", err)
	}
	return snapshots, nil
}

func lockBindingStates(
	ctx context.Context,
	tx pgx.Tx,
	bindingIDs []uuid.UUID,
) (map[uuid.UUID]lockedBindingState, error) {
	query, args, err := buildLockBindingStatesQuery(bindingIDs)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("lock geofence binding states: %w", err)
	}
	defer rows.Close()
	states := make(map[uuid.UUID]lockedBindingState, len(bindingIDs))
	for rows.Next() {
		var state lockedBindingState
		if err := rows.Scan(
			&state.BindingID,
			&state.ConfirmedState,
			&state.CandidateState,
			&state.CandidateCount,
			&state.CandidateSince,
			&state.StateVersion,
			&state.LastObservationVersion,
		); err != nil {
			return nil, fmt.Errorf("scan geofence binding state: %w", err)
		}
		states[state.BindingID] = state
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence binding states: %w", err)
	}
	return states, nil
}

func completedCoordinatorEvaluation(
	id uuid.UUID,
	payload event.DeviceLocationObservedPayload,
	evaluatedAt time.Time,
	snapshot coordinatorBindingSnapshot,
	previous lockedBindingState,
	result EvaluationResult,
) coordinatorEvaluation {
	return coordinatorEvaluation{
		ID:                         id,
		BindingID:                  snapshot.BindingID,
		DeviceID:                   payload.DeviceID,
		GeofenceID:                 snapshot.GeofenceID,
		GeofenceVersionID:          snapshot.GeofenceVersionID,
		ObservationVersion:         payload.ObservationVersion,
		Latitude:                   payload.Latitude,
		Longitude:                  payload.Longitude,
		GPSHeight:                  payload.GPSHeight,
		ObservedAt:                 payload.ObservedAt,
		ReceivedAt:                 payload.ReceivedAt,
		DeviceReportedAt:           payload.DeviceReportedAt,
		GPSLockStatus:              payload.GPSLockStatus,
		SatelliteCount:             payload.SatelliteCount,
		AccuracyMeters:             payload.AccuracyMeters,
		SourcePath:                 payload.SourcePath,
		PreviousObservationVersion: payload.PreviousObservationVersion,
		MovementDistanceMeters:     payload.MovementDistanceMeters,
		ElapsedSeconds:             payload.ElapsedSeconds,
		ImpliedSpeedMPS:            payload.ImpliedSpeedMPS,
		RuleType:                   snapshot.RuleType,
		RawPosition:                result.RawPosition,
		SignedDistanceMeters:       result.SignedDistanceMeters,
		PreviousConfirmedState:     previous.ConfirmedState,
		ConfirmedState:             result.ConfirmedState,
		CandidateState:             result.CandidateState,
		CandidateCount:             result.CandidateCount,
		CandidateSince:             result.CandidateSince,
		StateEdge:                  result.StateEdge,
		Status:                     evaluationStatusCompleted,
		ReasonCode:                 result.Reason,
		EvaluatedAt:                evaluatedAt,
	}
}

func (r *PgCoordinatorRepository) persistDeterministicFailure(
	ctx context.Context,
	tx pgx.Tx,
	payload event.DeviceLocationObservedPayload,
	evaluatedAt time.Time,
	previous lockedEffectiveState,
	snapshot coordinatorBindingSnapshot,
	state lockedBindingState,
	failureStage string,
	errorCode string,
	evaluationErr error,
) (CoordinatorResult, error) {
	evaluation := coordinatorEvaluation{
		ID:                         r.newID(),
		BindingID:                  snapshot.BindingID,
		DeviceID:                   payload.DeviceID,
		GeofenceID:                 snapshot.GeofenceID,
		GeofenceVersionID:          snapshot.GeofenceVersionID,
		ObservationVersion:         payload.ObservationVersion,
		Latitude:                   payload.Latitude,
		Longitude:                  payload.Longitude,
		GPSHeight:                  payload.GPSHeight,
		ObservedAt:                 payload.ObservedAt,
		ReceivedAt:                 payload.ReceivedAt,
		DeviceReportedAt:           payload.DeviceReportedAt,
		GPSLockStatus:              payload.GPSLockStatus,
		SatelliteCount:             payload.SatelliteCount,
		AccuracyMeters:             payload.AccuracyMeters,
		SourcePath:                 payload.SourcePath,
		PreviousObservationVersion: payload.PreviousObservationVersion,
		MovementDistanceMeters:     payload.MovementDistanceMeters,
		ElapsedSeconds:             payload.ElapsedSeconds,
		ImpliedSpeedMPS:            payload.ImpliedSpeedMPS,
		RuleType:                   snapshot.RuleType,
		PreviousConfirmedState:     normalizedConfirmedState(state.ConfirmedState),
		ConfirmedState:             normalizedConfirmedState(state.ConfirmedState),
		CandidateState:             state.CandidateState,
		CandidateCount:             state.CandidateCount,
		CandidateSince:             state.CandidateSince,
		Status:                     evaluationStatusFailed,
		ReasonCode:                 "evaluation_failed",
		FailureStage:               failureStage,
		ErrorCode:                  errorCode,
		ErrorSummary:               truncateCoordinatorError(evaluationErr.Error()),
		EvaluatedAt:                evaluatedAt,
	}
	if err := insertCoordinatorEvaluation(ctx, tx, evaluation); err != nil {
		return CoordinatorResult{}, err
	}
	records, err := buildEvaluationOutboxRecords(
		evaluation,
		payload.SerialNumber,
		payload.Carrier,
		state.StateVersion,
	)
	if err != nil {
		return CoordinatorResult{}, err
	}
	for _, record := range records {
		if err := r.outbox.InsertTx(ctx, tx, record); err != nil {
			return CoordinatorResult{}, fmt.Errorf(
				"enqueue failed geofence evaluation event: %w",
				err,
			)
		}
	}
	if err := updateEffectiveState(ctx, tx, effectiveStateWrite{
		DeviceID:           payload.DeviceID,
		State:              previous.State,
		ActionLevel:        previous.ActionLevel,
		TriggerBindingID:   previous.TriggerBindingID,
		ObservationVersion: payload.ObservationVersion,
		StateChanged:       false,
		EvaluationHealth:   evaluationHealthFailed,
		ErrorCode:          errorCode,
		UpdatedAt:          evaluatedAt,
	}); err != nil {
		return CoordinatorResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CoordinatorResult{}, fmt.Errorf(
			"commit failed geofence evaluation: %w",
			err,
		)
	}
	return CoordinatorResult{
		Processed:             true,
		Failed:                true,
		Reason:                errorCode,
		EvaluationCount:       1,
		EffectiveState:        previous.State,
		EffectiveStateVersion: previous.StateVersion,
	}, nil
}

func insertCoordinatorEvaluation(
	ctx context.Context,
	tx pgx.Tx,
	evaluation coordinatorEvaluation,
) error {
	query, args, err := buildInsertEvaluationQuery(evaluation)
	if err != nil {
		return err
	}
	var insertedID uuid.UUID
	if err := tx.QueryRow(ctx, query, args...).Scan(&insertedID); err != nil {
		return fmt.Errorf("insert geofence evaluation: %w", err)
	}
	if insertedID != evaluation.ID {
		return fmt.Errorf(
			"insert geofence evaluation: returned ID %s, expected %s",
			insertedID,
			evaluation.ID,
		)
	}
	return nil
}

func updateBindingState(
	ctx context.Context,
	tx pgx.Tx,
	evaluation coordinatorEvaluation,
) error {
	query, args, err := buildUpdateBindingStateQuery(evaluation)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update geofence binding state: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"update geofence binding state: affected %d rows",
			tag.RowsAffected(),
		)
	}
	return nil
}

func updateEffectiveState(
	ctx context.Context,
	tx pgx.Tx,
	write effectiveStateWrite,
) error {
	query, args, err := buildUpdateEffectiveStateQuery(write)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update effective geofence state: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"update effective geofence state: affected %d rows",
			tag.RowsAffected(),
		)
	}
	return nil
}

func effectiveStateChanged(
	previous lockedEffectiveState,
	next EffectiveSnapshot,
) bool {
	return previous.State != next.State ||
		previous.ActionLevel != next.RequiredActionLevel
}

func effectiveActionLevel(mode RuntimeMode, configured ActionLevel) ActionLevel {
	if mode != RuntimeModeEnforce && configured == ActionLevelDeactivate {
		return ActionLevelNotifyOnly
	}
	return configured
}

func truncateCoordinatorError(message string) string {
	const maxRunes = 512
	runes := []rune(message)
	if len(runes) <= maxRunes {
		return message
	}
	return string(runes[:maxRunes])
}

func buildActiveBindingSnapshotQuery(
	deviceID uuid.UUID,
	carrier string,
) (string, []any, error) {
	query, args, err := storage.Psql.
		Select(
			"b.id",
			"b.geofence_id",
			"b.rule_type",
			"v.id",
			"v.geometry_json",
			"v.policy_json",
			"CASE WHEN c.mode = 'enforce' AND s.value = 'enforce' "+
				"THEN 'enforce' ELSE 'observe' END",
		).
		From("device_geofence_bindings b").
		Join("geofence_definitions d ON d.id = b.geofence_id").
		Join("geofence_versions v ON d.current_version_id = v.id").
		Join("geofence_carrier_settings c ON c.carrier = d.carrier").
		Join(
			"sys_configs s ON s.category = ? AND s.key = ?",
			geofenceConfigCategory,
			geofenceModeConfigKey,
		).
		Where(sq.Eq{
			"b.device_id": deviceID,
			"b.status":    BindingStatusActive,
			"d.status":    DefinitionStatusEnabled,
			"d.carrier":   carrier,
			"v.status":    VersionStatusPublished,
			"c.mode": []RuntimeMode{
				RuntimeModeObserve,
				RuntimeModeEnforce,
			},
			"s.value": []RuntimeMode{
				RuntimeModeObserve,
				RuntimeModeEnforce,
			},
		}).
		OrderBy("b.id").
		Suffix("FOR SHARE OF b, d, v, c, s").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build active geofence binding snapshot: %w", err)
	}
	return query, args, nil
}

func buildLockBindingStatesQuery(bindingIDs []uuid.UUID) (string, []any, error) {
	query, args, err := storage.Psql.
		Select(
			"binding_id",
			"confirmed_state",
			"COALESCE(candidate_state, '')",
			"candidate_count",
			"candidate_since",
			"state_version",
			"COALESCE(last_observation_version, 0)",
		).
		From("device_geofence_states").
		Where(sq.Eq{"binding_id": bindingIDs}).
		OrderBy("binding_id").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build lock geofence binding states: %w", err)
	}
	return query, args, nil
}

func buildInsertEvaluationQuery(evaluation coordinatorEvaluation) (string, []any, error) {
	query, args, err := storage.Psql.
		Insert("geofence_evaluations").
		Columns(
			"id",
			"binding_id",
			"device_id",
			"geofence_id",
			"geofence_version_id",
			"observation_version",
			"latitude",
			"longitude",
			"gps_height",
			"observed_at",
			"received_at",
			"device_reported_at",
			"gps_lock_status",
			"satellite_count",
			"accuracy_meters",
			"source_path",
			"previous_observation_version",
			"movement_distance_meters",
			"elapsed_seconds",
			"implied_speed_mps",
			"rule_type",
			"raw_position",
			"signed_distance_meters",
			"previous_confirmed_state",
			"confirmed_state",
			"candidate_state",
			"candidate_count",
			"candidate_since",
			"state_edge",
			"status",
			"reason_code",
			"failure_stage",
			"error_code",
			"error_summary",
			"evaluated_at",
		).
		Values(
			evaluation.ID,
			evaluation.BindingID,
			evaluation.DeviceID,
			evaluation.GeofenceID,
			evaluation.GeofenceVersionID,
			evaluation.ObservationVersion,
			evaluation.Latitude,
			evaluation.Longitude,
			evaluation.GPSHeight,
			evaluation.ObservedAt,
			evaluation.ReceivedAt,
			evaluation.DeviceReportedAt,
			evaluation.GPSLockStatus,
			evaluation.SatelliteCount,
			evaluation.AccuracyMeters,
			evaluation.SourcePath,
			evaluation.PreviousObservationVersion,
			evaluation.MovementDistanceMeters,
			evaluation.ElapsedSeconds,
			evaluation.ImpliedSpeedMPS,
			evaluation.RuleType,
			nullableRawPosition(evaluation.RawPosition),
			evaluation.SignedDistanceMeters,
			evaluation.PreviousConfirmedState,
			evaluation.ConfirmedState,
			nullableCandidateState(evaluation.CandidateState),
			evaluation.CandidateCount,
			evaluation.CandidateSince,
			evaluation.StateEdge,
			evaluation.Status,
			evaluation.ReasonCode,
			nullableString(evaluation.FailureStage),
			nullableString(evaluation.ErrorCode),
			nullableString(evaluation.ErrorSummary),
			evaluation.EvaluatedAt,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build insert geofence evaluation: %w", err)
	}
	return query, args, nil
}

func buildUpdateBindingStateQuery(evaluation coordinatorEvaluation) (string, []any, error) {
	query, args, err := storage.Psql.
		Update("device_geofence_states").
		Set("confirmed_state", evaluation.ConfirmedState).
		Set("candidate_state", nullableCandidateState(evaluation.CandidateState)).
		Set("candidate_count", evaluation.CandidateCount).
		Set("candidate_since", evaluation.CandidateSince).
		Set("state_version", sq.Expr("state_version + 1")).
		Set("last_geofence_version_id", evaluation.GeofenceVersionID).
		Set("last_observation_version", evaluation.ObservationVersion).
		Set("last_observed_at", evaluation.ObservedAt).
		Set("last_distance_to_boundary", evaluation.SignedDistanceMeters).
		Set("last_evaluation_id", evaluation.ID).
		Set("updated_at", evaluation.EvaluatedAt).
		Where(sq.Eq{"binding_id": evaluation.BindingID}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build update geofence binding state: %w", err)
	}
	return query, args, nil
}

func buildUpdateEffectiveStateQuery(write effectiveStateWrite) (string, []any, error) {
	builder := storage.Psql.
		Update("device_geofence_effective_states").
		Set("effective_state", write.State).
		Set("required_action_level", write.ActionLevel).
		Set("trigger_binding_id", write.TriggerBindingID).
		Set("last_observation_version", write.ObservationVersion).
		Set("evaluation_health", write.EvaluationHealth).
		Set("last_evaluation_error_code", nullableString(write.ErrorCode)).
		Set("updated_at", write.UpdatedAt)
	if write.StateChanged {
		builder = builder.Set("state_version", sq.Expr("state_version + 1"))
	} else {
		builder = builder.Set("state_version", sq.Expr("state_version"))
	}
	if !write.SuccessfulEvaluationAt.IsZero() {
		builder = builder.Set(
			"last_successful_evaluation_at",
			write.SuccessfulEvaluationAt,
		)
	}
	query, args, err := builder.
		Where(sq.Eq{"device_id": write.DeviceID}).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build update effective geofence state: %w", err)
	}
	return query, args, nil
}

func buildEvaluationOutboxRecords(
	evaluation coordinatorEvaluation,
	serialNumber string,
	carrier string,
	stateVersion int64,
) ([]outbox.Record, error) {
	subject := event.SubjectGeofenceEvaluationCompleted
	health := evaluationHealthHealthy
	if evaluation.Status == evaluationStatusFailed {
		subject = event.SubjectGeofenceEvaluationFailed
		health = evaluationHealthFailed
	}
	payload := event.GeofenceEvaluationPayload{
		DeviceID:             evaluation.DeviceID,
		SerialNumber:         serialNumber,
		Carrier:              carrier,
		BindingID:            evaluation.BindingID,
		RuleType:             string(evaluation.RuleType),
		GeofenceID:           evaluation.GeofenceID,
		GeofenceVersionID:    evaluation.GeofenceVersionID,
		ObservationVersion:   evaluation.ObservationVersion,
		StateVersion:         stateVersion,
		EvaluationID:         evaluation.ID,
		ConfirmedState:       string(evaluation.ConfirmedState),
		EvaluationHealth:     string(health),
		ReasonCode:           evaluation.ReasonCode,
		ErrorCode:            evaluation.ErrorCode,
		OccurredAt:           evaluation.EvaluatedAt,
		SignedDistanceMeters: evaluation.SignedDistanceMeters,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal geofence evaluation event: %w", err)
	}
	identity := fmt.Sprintf(
		"%s:%s:%d",
		evaluation.BindingID,
		evaluation.GeofenceVersionID,
		evaluation.ObservationVersion,
	)
	records := []outbox.Record{{
		AggregateType: "geofence_binding",
		AggregateID:   evaluation.BindingID.String(),
		Subject:       subject,
		Payload:       payloadJSON,
		DedupeKey:     subject + ":" + identity,
	}}
	if evaluation.Status != evaluationStatusCompleted || !evaluation.StateEdge {
		return records, nil
	}
	switch evaluation.ConfirmedState {
	case ConfirmedStateOutside:
		subject = event.SubjectGeofenceRuleExited
	case ConfirmedStateInside:
		subject = event.SubjectGeofenceRuleEntered
	default:
		return records, nil
	}
	records = append(records, outbox.Record{
		AggregateType: "geofence_binding",
		AggregateID:   evaluation.BindingID.String(),
		Subject:       subject,
		Payload:       payloadJSON,
		DedupeKey:     subject + ":" + identity,
	})
	return records, nil
}

func buildDeviceEdgeOutboxRecord(
	deviceID uuid.UUID,
	serialNumber string,
	carrier string,
	observationVersion int64,
	triggerEvaluationID *uuid.UUID,
	previous lockedEffectiveState,
	next EffectiveSnapshot,
	occurredAt time.Time,
) (outbox.Record, bool, error) {
	var subject, reason string
	switch {
	case next.State == EffectiveStateOutside &&
		previous.State != EffectiveStateOutside:
		subject = event.SubjectGeofenceDeviceExited
		reason = "effective_state_outside"
	case next.State == EffectiveStateOutside &&
		previous.State == EffectiveStateOutside &&
		actionLevelRank(next.RequiredActionLevel) > actionLevelRank(previous.ActionLevel):
		subject = event.SubjectGeofenceDeviceEscalated
		reason = "required_action_level_escalated"
	case previous.State == EffectiveStateOutside &&
		next.State == EffectiveStateInside:
		subject = event.SubjectGeofenceDeviceEntered
		reason = "effective_state_inside"
	case next.State == EffectiveStateUnknown &&
		previous.State != EffectiveStateUnknown:
		subject = event.SubjectGeofenceDeviceUnknown
		reason = "effective_state_unknown"
	default:
		return outbox.Record{}, false, nil
	}
	nextVersion := previous.StateVersion + 1
	payloadJSON, err := json.Marshal(event.GeofenceDeviceStatePayload{
		DeviceID:              deviceID,
		SerialNumber:          serialNumber,
		Carrier:               carrier,
		TriggerBindingID:      next.TriggerBindingID,
		TriggerEvaluationID:   triggerEvaluationID,
		ObservationVersion:    observationVersion,
		EffectiveState:        string(next.State),
		RequiredActionLevel:   string(next.RequiredActionLevel),
		EffectiveStateVersion: nextVersion,
		EvaluationHealth:      string(evaluationHealthHealthy),
		ReasonCode:            reason,
		OccurredAt:            occurredAt,
	})
	if err != nil {
		return outbox.Record{}, false,
			fmt.Errorf("marshal geofence device state event: %w", err)
	}
	return outbox.Record{
		AggregateType: "device",
		AggregateID:   deviceID.String(),
		Subject:       subject,
		Payload:       payloadJSON,
		DedupeKey: fmt.Sprintf(
			"%s:%s:%d",
			subject,
			deviceID,
			nextVersion,
		),
	}, true, nil
}

func nullableRawPosition(value RawPosition) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableCandidateState(value CandidateState) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
