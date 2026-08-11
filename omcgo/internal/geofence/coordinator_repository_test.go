package geofence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCoordinatorReadQueriesPreserveDeviceThenStableBindingLockOrder(t *testing.T) {
	deviceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	higherBindingID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	lowerBindingID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	effectiveQuery, effectiveArgs, err := buildLockEffectiveStateQuery(deviceID)
	require.NoError(t, err)
	assert.Contains(t, effectiveQuery, "FROM device_geofence_effective_states")
	assert.Contains(t, effectiveQuery, "FOR UPDATE")
	assert.Contains(t, effectiveArgs, deviceID.String())

	snapshotQuery, snapshotArgs, err := buildActiveBindingSnapshotQuery(deviceID, "cmcc")
	require.NoError(t, err)
	assert.Contains(t, snapshotQuery, "FROM device_geofence_bindings b")
	assert.Contains(t, snapshotQuery, "JOIN geofence_definitions d")
	assert.Contains(t, snapshotQuery, "JOIN geofence_versions v")
	assert.Contains(t, snapshotQuery, "JOIN geofence_carrier_settings c")
	assert.Contains(t, snapshotQuery, "JOIN sys_configs s")
	assert.Contains(t, snapshotQuery, "d.current_version_id = v.id")
	assert.Contains(t, snapshotQuery, "b.status =")
	assert.Contains(t, snapshotQuery, "d.status =")
	assert.Contains(t, snapshotQuery, "c.mode IN")
	assert.Contains(t, snapshotQuery, "s.category =")
	assert.Contains(t, snapshotQuery, "s.key =")
	assert.Contains(t, snapshotQuery, "s.value IN")
	assert.Contains(t, snapshotQuery, "CASE WHEN c.mode = 'enforce'")
	assert.Contains(t, snapshotQuery, "ORDER BY b.id")
	assert.Contains(t, snapshotQuery, "FOR SHARE OF b, d, v")
	assert.Contains(t, snapshotArgs, deviceID.String())
	assert.Contains(t, snapshotArgs, "cmcc")
	assert.Contains(t, snapshotArgs, BindingStatusActive)
	assert.Contains(t, snapshotArgs, DefinitionStatusEnabled)
	assert.Contains(t, snapshotArgs, "geofence")
	assert.Contains(t, snapshotArgs, "mode")

	stateQuery, stateArgs, err := buildLockBindingStatesQuery(
		[]uuid.UUID{higherBindingID, lowerBindingID},
	)
	require.NoError(t, err)
	assert.Contains(t, stateQuery, "FROM device_geofence_states")
	assert.Contains(t, stateQuery, "ORDER BY binding_id")
	assert.Contains(t, stateQuery, "FOR UPDATE")
	assert.Contains(t, stateArgs, higherBindingID)
	assert.Contains(t, stateArgs, lowerBindingID)
}

func TestEffectiveActionLevelOnlyDeactivatesInEnforceMode(t *testing.T) {
	assert.Equal(t, ActionLevelNotifyOnly,
		effectiveActionLevel(RuntimeModeObserve, ActionLevelDeactivate))
	assert.Equal(t, ActionLevelDeactivate,
		effectiveActionLevel(RuntimeModeEnforce, ActionLevelDeactivate))
	assert.Equal(t, ActionLevelManualReview,
		effectiveActionLevel(RuntimeModeObserve, ActionLevelManualReview))
}

func TestBuildLatestObservationQueryUsesDeviceVisibilityGuard(t *testing.T) {
	deviceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	query, args, err := buildLatestObservationQuery(deviceID)
	require.NoError(t, err)
	assert.Contains(t, query, "FROM device_location_observations o")
	assert.Contains(t, query, "JOIN devices d ON d.id = o.device_id")
	assert.Contains(t, query, "o.device_id =")
	assert.Contains(t, query, "d.deleted_at IS NULL")
	assert.Contains(t, args, deviceID.String())
}

func TestBuildCoordinatorWriteQueriesPersistImmutableEvidenceAndVersions(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	evaluation := coordinatorEvaluation{
		ID:                     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		BindingID:              uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		DeviceID:               uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		GeofenceID:             uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		GeofenceVersionID:      uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		ObservationVersion:     7,
		Latitude:               30.25,
		Longitude:              120.16,
		ObservedAt:             evaluatedAt.Add(-time.Minute),
		ReceivedAt:             evaluatedAt.Add(-time.Minute),
		RuleType:               RuleTypePolygonAllowZone,
		RawPosition:            RawPositionOutside,
		PreviousConfirmedState: ConfirmedStateInside,
		ConfirmedState:         ConfirmedStateOutside,
		CandidateCount:         0,
		StateEdge:              true,
		Status:                 evaluationStatusCompleted,
		ReasonCode:             "confirmed_outside",
		EvaluatedAt:            evaluatedAt,
	}

	insertSQL, insertArgs, err := buildInsertEvaluationQuery(evaluation)
	require.NoError(t, err)
	assert.Contains(t, insertSQL, "INSERT INTO geofence_evaluations")
	assert.Contains(t, insertSQL, "previous_confirmed_state")
	assert.Contains(t, insertSQL, "observation_version")
	assert.Contains(t, insertSQL, "RETURNING id")
	assert.Contains(t, insertArgs, evaluation.ID)
	assert.Contains(t, insertArgs, evaluation.ObservationVersion)

	stateSQL, stateArgs, err := buildUpdateBindingStateQuery(evaluation)
	require.NoError(t, err)
	assert.Contains(t, stateSQL, "UPDATE device_geofence_states")
	assert.Contains(t, stateSQL, "state_version + 1")
	assert.Contains(t, stateSQL, "last_evaluation_id")
	assert.Contains(t, stateSQL, "last_observation_version")
	assert.Contains(t, stateArgs, evaluation.ID)

	effective := effectiveStateWrite{
		DeviceID:               evaluation.DeviceID,
		State:                  EffectiveStateOutside,
		ActionLevel:            ActionLevelDeactivate,
		TriggerBindingID:       &evaluation.BindingID,
		ObservationVersion:     evaluation.ObservationVersion,
		StateChanged:           true,
		EvaluationHealth:       evaluationHealthHealthy,
		SuccessfulEvaluationAt: evaluatedAt,
		UpdatedAt:              evaluatedAt,
	}
	effectiveSQL, effectiveArgs, err := buildUpdateEffectiveStateQuery(effective)
	require.NoError(t, err)
	assert.Contains(t, effectiveSQL, "UPDATE device_geofence_effective_states")
	assert.Contains(t, effectiveSQL, "state_version + 1")
	assert.Contains(t, effectiveSQL, "last_observation_version")
	assert.Contains(t, effectiveSQL, "evaluation_health")
	assert.Contains(t, effectiveArgs, EffectiveStateOutside)
	assert.Contains(t, effectiveArgs, evaluation.ObservationVersion)
}

func TestBuildCoordinatorEventsUseStableBusinessDedupeKeys(t *testing.T) {
	evaluation := coordinatorEvaluation{
		ID:                 uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		BindingID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		DeviceID:           uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		GeofenceID:         uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		GeofenceVersionID:  uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		ObservationVersion: 9,
		RuleType:           RuleTypePolygonAllowZone,
		StateEdge:          true,
		ConfirmedState:     ConfirmedStateOutside,
		Status:             evaluationStatusCompleted,
	}

	records, err := buildEvaluationOutboxRecords(evaluation, "SN-1", "cmcc", 4)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(
		t,
		"geofence.evaluation.completed:22222222-2222-2222-2222-222222222222:"+
			"55555555-5555-5555-5555-555555555555:9",
		records[0].DedupeKey,
	)
	assert.Equal(t, "geofence.rule.exited", records[1].Subject)
	assert.Contains(t, records[1].DedupeKey, evaluation.BindingID.String())
	for _, record := range records {
		var payload map[string]any
		require.NoError(t, json.Unmarshal(record.Payload, &payload))
		assert.Equal(t, evaluation.DeviceID.String(), payload["device_id"])
		assert.Equal(t, evaluation.ID.String(), payload["evaluation_id"])
	}
}

func TestBuildDeviceEdgeOnlyEmitsRealEffectiveTransitions(t *testing.T) {
	deviceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bindingID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		previous    lockedEffectiveState
		next        EffectiveSnapshot
		wantSubject string
	}{
		{
			name:        "exit",
			previous:    lockedEffectiveState{State: EffectiveStateInside, ActionLevel: ActionLevelNone, StateVersion: 3},
			next:        EffectiveSnapshot{State: EffectiveStateOutside, RequiredActionLevel: ActionLevelNotifyOnly, TriggerBindingID: &bindingID},
			wantSubject: "geofence.device.exited",
		},
		{
			name:        "policy escalation",
			previous:    lockedEffectiveState{State: EffectiveStateOutside, ActionLevel: ActionLevelNotifyOnly, StateVersion: 3},
			next:        EffectiveSnapshot{State: EffectiveStateOutside, RequiredActionLevel: ActionLevelDeactivate, TriggerBindingID: &bindingID},
			wantSubject: "geofence.device.policy_escalated",
		},
		{
			name:        "entered",
			previous:    lockedEffectiveState{State: EffectiveStateOutside, ActionLevel: ActionLevelDeactivate, StateVersion: 3},
			next:        EffectiveSnapshot{State: EffectiveStateInside, RequiredActionLevel: ActionLevelNone},
			wantSubject: "geofence.device.entered",
		},
		{
			name:        "unknown",
			previous:    lockedEffectiveState{State: EffectiveStateInside, ActionLevel: ActionLevelNone, StateVersion: 3},
			next:        EffectiveSnapshot{State: EffectiveStateUnknown, RequiredActionLevel: ActionLevelNone, TriggerBindingID: &bindingID},
			wantSubject: "geofence.device.unknown",
		},
		{
			name:     "unchanged",
			previous: lockedEffectiveState{State: EffectiveStateInside, ActionLevel: ActionLevelNone, StateVersion: 3},
			next:     EffectiveSnapshot{State: EffectiveStateInside, RequiredActionLevel: ActionLevelNone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, ok, err := buildDeviceEdgeOutboxRecord(
				deviceID,
				"SN-1",
				"cmcc",
				9,
				tt.previous,
				tt.next,
				now,
			)
			require.NoError(t, err)
			if tt.wantSubject == "" {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.wantSubject, record.Subject)
			assert.Contains(t, record.DedupeKey, ":4")
		})
	}
}

type coordinatorRowStub struct {
	scan func(...any) error
}

func (r coordinatorRowStub) Scan(dest ...any) error {
	return r.scan(dest...)
}

type coordinatorTxStub struct {
	pgx.Tx
	row           pgx.Row
	queryErr      error
	execCalls     int
	commitCalls   int
	rollbackCalls int
}

func (tx *coordinatorTxStub) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	return tx.row
}

func (tx *coordinatorTxStub) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, tx.queryErr
}

func (tx *coordinatorTxStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	tx.execCalls++
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *coordinatorTxStub) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *coordinatorTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	return nil
}

type coordinatorDBStub struct {
	tx         *coordinatorTxStub
	beginCalls int
}

func (db *coordinatorDBStub) Begin(context.Context) (pgx.Tx, error) {
	db.beginCalls++
	return db.tx, nil
}

func (db *coordinatorDBStub) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	panic("unexpected database Query")
}

func (db *coordinatorDBStub) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	panic("unexpected database QueryRow")
}

func (db *coordinatorDBStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	panic("unexpected database Exec")
}

func TestPgCoordinatorRepositoryDuplicateObservationCommitsNoOp(t *testing.T) {
	tx := &coordinatorTxStub{
		row: coordinatorRowStub{scan: func(dest ...any) error {
			*dest[0].(*EffectiveState) = EffectiveStateInside
			*dest[1].(*ActionLevel) = ActionLevelNone
			*dest[2].(*int64) = 3
			*dest[3].(*int64) = 7
			*dest[4].(**uuid.UUID) = nil
			return nil
		}},
	}
	repository := NewPgCoordinatorRepository(&coordinatorDBStub{tx: tx})

	result, err := repository.EvaluateLocation(
		context.Background(),
		event.DeviceLocationObservedPayload{
			DeviceID:           uuid.New(),
			SerialNumber:       "SN-1",
			Carrier:            "cmcc",
			ObservationVersion: 7,
			ObservedAt:         time.Now().UTC(),
			ReceivedAt:         time.Now().UTC(),
		},
		time.Now().UTC(),
	)

	require.NoError(t, err)
	assert.True(t, result.NoOp)
	assert.Equal(t, "duplicate_observation", result.Reason)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Equal(t, 0, tx.execCalls)
}

func TestPgCoordinatorRepositoryRollsBackWhenBindingSnapshotFails(t *testing.T) {
	snapshotErr := errors.New("database unavailable")
	tx := &coordinatorTxStub{
		row: coordinatorRowStub{scan: func(dest ...any) error {
			*dest[0].(*EffectiveState) = EffectiveStateInside
			*dest[1].(*ActionLevel) = ActionLevelNone
			*dest[2].(*int64) = 3
			*dest[3].(*int64) = 6
			*dest[4].(**uuid.UUID) = nil
			return nil
		}},
		queryErr: snapshotErr,
	}
	repository := NewPgCoordinatorRepository(&coordinatorDBStub{tx: tx})

	_, err := repository.EvaluateLocation(
		context.Background(),
		event.DeviceLocationObservedPayload{
			DeviceID:           uuid.New(),
			SerialNumber:       "SN-1",
			Carrier:            "cmcc",
			ObservationVersion: 7,
			ObservedAt:         time.Now().UTC(),
			ReceivedAt:         time.Now().UTC(),
		},
		time.Now().UTC(),
	)

	require.ErrorIs(t, err, snapshotErr)
	assert.Zero(t, tx.commitCalls)
	assert.Equal(t, 1, tx.rollbackCalls)
}
