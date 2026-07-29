package paramsync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

func TestMappedMLNRFStateCannotBeOverwrittenByLaterRawShadow(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	deviceID := uuid.New()
	insertReleaseCandidateForTest(t, pool, deviceID, true)
	_, err := pool.Exec(ctx,
		`UPDATE devices SET product_class=$2 WHERE id=$1`,
		deviceID,
		mlnDCProductClass,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, deviceID)
	})

	const standardPath = "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	coverage := []CoverageScope{{
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			Access:       "readWrite",
			DataType:     "boolean",
			IsStorable:   true,
		}},
	}}
	mapped := projectTaskValues([]tr069.ParameterValueStruct{{
		Name:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
		Value: "1",
		Type:  "xsd:unsignedInt",
	}}, coverage)
	rawShadow := projectTaskValues([]tr069.ParameterValueStruct{{
		Name:  standardPath,
		Value: "0",
		Type:  "xsd:string",
	}}, coverage)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	rawShadow, err = filterMLNDCRawRFShadowValues(ctx, tx, deviceID, rawShadow, coverage)
	require.NoError(t, err)
	require.Empty(t, rawShadow)
	now := time.Now().UTC()
	require.NoError(t, upsertOfficialValues(ctx, tx, deviceID, mapped, now))
	require.NoError(t, upsertOfficialValues(ctx, tx, deviceID, rawShadow, now))
	require.NoError(t, tx.Commit(ctx))

	var value string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parameter_value
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, deviceID, standardPath).Scan(&value))
	assert.Equal(t, "1", value)
}

func TestRecoveredMLNDCRF9005ReplacesStaleOffWithUnknown(t *testing.T) {
	tests := []struct {
		name         string
		productClass string
		scope        SyncScope
		wantValue    string
	}{
		{name: "MLN_DC_partial", productClass: "FAP/MLN/DC", scope: SyncScopePartial, wantValue: "unknown"},
		{name: "MLN_DC_full", productClass: "FAP/MLN/DC", scope: SyncScopeFull, wantValue: "unknown"},
		{name: "MLN_SC_partial", productClass: "FAP/MLN/SC", scope: SyncScopePartial, wantValue: "0"},
		{name: "MLN_SC_full", productClass: "FAP/MLN/SC", scope: SyncScopeFull, wantValue: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRecoveredMLNDCRF9005(t, tt.productClass, tt.scope, tt.wantValue)
		})
	}
}

func testRecoveredMLNDCRF9005(
	t *testing.T,
	productClass string,
	scope SyncScope,
	wantValue string,
) {
	t.Helper()
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	insertReleaseCandidateForTest(t, pool, request.DeviceID, true)
	_, err := pool.Exec(ctx,
		`UPDATE devices SET product_class=$2 WHERE id=$1`,
		request.DeviceID,
		productClass,
	)
	require.NoError(t, err)

	const (
		privatePath  = "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState"
		standardPath = "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus"
	)
	_, err = pool.Exec(ctx, `
		INSERT INTO device_parameters (
			device_id, parameter_path, parameter_value, parameter_type,
			writable, last_updated_at, fap_instance, param_group
		) VALUES ($1, $2, '0', 'unsignedInt', true, now(), 2, 'fap_control')
	`, request.DeviceID, standardPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, request.DeviceID)
	})

	coverage, err := json.Marshal([]CoverageScope{{
		Path:     mlnDCRFStatusStandardPath,
		Complete: true,
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			Access:       "readWrite",
			DataType:     "boolean",
			IsStorable:   true,
		}},
	}})
	require.NoError(t, err)
	runID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO parameter_sync_runs (
			id, request_id, device_id, device_sn, trigger_reason, sync_scope,
			status, coverage, expected_task_count, terminal_task_count,
			processed_task_count, failed_task_count
		) VALUES ($1, $2, $3, $4, 'manual', $5, 'waiting_device',
		          $6, 1, 1, 0, 0)
	`, runID, request.ID, request.DeviceID, request.DeviceSN, scope, coverage)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE parameter_sync_requests
		SET run_id=$2, active_run_id=$2
		WHERE id=$1
	`, request.ID, runID)
	require.NoError(t, err)

	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN:   request.DeviceSN,
		Method:     "GetParameterValues",
		Params:     []byte(`{"names":["` + privatePath + `"]}`),
		CommandKey: "param-sync-" + runID.String() + "-1",
		Source:     task.TaskSourceParamSync,
		SourceID:   runID.String(),
		CreatorID:  request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, completed))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_tasks WHERE id=$1`, completed.ID)
	})
	recoveredResult, err := json.Marshal(map[string]any{
		"recovered":  true,
		"bad_path":   privatePath,
		"fault_code": 9005,
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE device_tasks
		SET status='completed', completed_at=now(), result=$2
		WHERE id=$1
	`, completed.ID, recoveredResult)
	require.NoError(t, err)

	recovered, err := NewReconciler(pool, nil, nil).
		WithResultProcessor(NewPGResultProcessor(pool)).
		RecoverMissingResults(ctx, 20, 200, 200)

	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	var value string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parameter_value
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, request.DeviceID, standardPath).Scan(&value))
	assert.Equal(t, wantValue, value)
}
