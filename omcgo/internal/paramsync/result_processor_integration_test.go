package paramsync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

func TestUpsertOfficialValuesMLNDCRFReportReplacesStoredValuesPerCarrier(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	deviceID := uuid.New()
	insertReleaseCandidateForTest(t, pool, deviceID, true)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, deviceID)
	})

	const (
		cell1Standard = "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
		cell2Standard = "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus"
	)
	oldUpdatedAt := time.Now().UTC().Add(-time.Hour)
	_, err := pool.Exec(ctx, `
		INSERT INTO device_parameters (
			device_id, parameter_path, parameter_value, parameter_type,
			writable, last_updated_at, fap_instance, param_group
		) VALUES
			($1, $2, '0', 'unsignedInt', true, $4, 1, 'fap_control'),
			($1, $3, '0', 'unsignedInt', true, $4, 2, 'fap_control')
	`, deviceID, cell1Standard, cell2Standard, oldUpdatedAt)
	require.NoError(t, err)

	set := &parammodel.MappingSet{Mappings: []parammodel.ParamMapping{{
		StandardPath: mlnDCRFStatusStandardPath,
		PrivatePath:  mlnDCRFStatusPrivatePath,
		Access:       "readWrite",
		DataType:     "boolean",
		IsStorable:   true,
		IsSupported:  true,
	}}}
	plan, err := NewPlanner(staticMappings{set: set}, 50).Plan(ctx, PlanCommand{
		Device:         &model.Device{ProductClass: "FAP/MLN/DC"},
		Scope:          SyncScopeReadback,
		RequestedPaths: []string{mlnDCRFStatusStandardPath},
	})
	require.NoError(t, err)
	projected := projectTaskValues([]tr069.ParameterValueStruct{
		{Name: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState", Value: "1", Type: "xsd:unsignedInt"},
		{Name: "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState", Value: "0", Type: "xsd:unsignedInt"},
	}, plan.Coverage)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	reportedAt := time.Now().UTC()
	require.NoError(t, upsertOfficialValues(ctx, tx, deviceID, projected, reportedAt))
	require.NoError(t, tx.Commit(ctx))

	type storedRFValue struct {
		value       string
		fapInstance int
		lastUpdated time.Time
	}
	stored := make(map[string]storedRFValue, 2)
	rows, err := pool.Query(ctx, `
		SELECT parameter_path, parameter_value, fap_instance, last_updated_at
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path IN ($2, $3)
	`, deviceID, cell1Standard, cell2Standard)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var path string
		var value storedRFValue
		require.NoError(t, rows.Scan(&path, &value.value, &value.fapInstance, &value.lastUpdated))
		stored[path] = value
	}
	require.NoError(t, rows.Err())

	require.Len(t, stored, 2)
	assert.Equal(t, "1", stored[cell1Standard].value)
	assert.Equal(t, 1, stored[cell1Standard].fapInstance)
	assert.True(t, stored[cell1Standard].lastUpdated.After(oldUpdatedAt))
	assert.Equal(t, "0", stored[cell2Standard].value)
	assert.Equal(t, 2, stored[cell2Standard].fapInstance)
	var missingCarrierCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM device_parameters
		WHERE device_id=$1
		  AND parameter_path='Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus'
	`, deviceID).Scan(&missingCarrierCount))
	assert.Zero(t, missingCarrierCount, "a missing RF carrier must not be fabricated as off")
	var privatePathCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM device_parameters
		WHERE device_id=$1 AND parameter_path LIKE '%AdminCellState'
	`, deviceID).Scan(&privatePathCount))
	assert.Zero(t, privatePathCount, "device_parameters must store the XML-mapped standard RF paths")
}

func TestRecoveredMLNDCRF9005ReplacesStaleStandardValueWithUnknown(t *testing.T) {
	tests := []struct {
		productClass string
		wantValue    string
		wantType     string
	}{
		{productClass: "FAP/MLN/DC", wantValue: "unknown", wantType: "string"},
		{productClass: "FAP/MLN/SC", wantValue: "0", wantType: "unsignedInt"},
	}
	for _, tt := range tests {
		for _, scope := range []SyncScope{SyncScopePartial, SyncScopeFull} {
			t.Run(tt.productClass+"_"+string(scope), func(t *testing.T) {
				testRecoveredMLNDCRF9005Value(
					t, scope, tt.productClass, tt.wantValue, tt.wantType,
				)
			})
		}
	}
}

func TestIsMLNDCDeviceMissingDeviceFailsClosed(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	isMLNDC, err := isMLNDCDevice(ctx, tx, uuid.New())

	require.NoError(t, err)
	assert.False(t, isMLNDC)
}

func testRecoveredMLNDCRF9005Value(
	t *testing.T,
	scope SyncScope,
	productClass string,
	wantValue string,
	wantType string,
) {
	t.Helper()
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	insertReleaseCandidateForTest(t, pool, req.DeviceID, true)
	_, err := pool.Exec(ctx, `UPDATE devices SET product_class=$2 WHERE id=$1`, req.DeviceID, productClass)
	require.NoError(t, err)

	const (
		privatePath  = "Device.Services.FAPService.3.CellConfig.LTE.RAN.RF.AdminCellState"
		standardPath = "Device.Services.FAPService.3.FAPControl.LTE.RFTxStatus"
	)
	_, err = pool.Exec(ctx, `
		INSERT INTO device_parameters (
			device_id, parameter_path, parameter_value, parameter_type,
			writable, last_updated_at, fap_instance, param_group
		) VALUES ($1, $2, '0', 'unsignedInt', true, now() - interval '1 hour', 3, 'fap_control')
	`, req.DeviceID, standardPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, req.DeviceID)
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
	`, runID, req.ID, req.DeviceID, req.DeviceSN, scope, coverage)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE parameter_sync_requests
		SET run_id=$2, active_run_id=$2
		WHERE id=$1
	`, req.ID, runID)
	require.NoError(t, err)

	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN:   req.DeviceSN,
		Method:     "GetParameterValues",
		Params:     []byte(`{"names":["` + privatePath + `"]}`),
		CommandKey: "param-sync-" + runID.String() + "-2",
		Source:     task.TaskSourceParamSync,
		SourceID:   runID.String(),
		CreatorID:  req.ID.String(),
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
	var value, parameterType string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parameter_value, parameter_type
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, req.DeviceID, standardPath).Scan(&value, &parameterType))
	assert.Equal(t, wantValue, value)
	assert.Equal(t, wantType, parameterType)
	var privatePathCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, req.DeviceID, privatePath).Scan(&privatePathCount))
	assert.Zero(t, privatePathCount)
}
