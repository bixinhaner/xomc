package software

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpgradeExecutor_HandleRebootComplete_UsesACSNestedDeviceIDPayload(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name: "ACS nested device_id serial_number",
			payload: map[string]interface{}{
				"device_id": map[string]interface{}{
					"serial_number": "SN-ROLLBACK-BOOT",
				},
				"events": []string{"1 BOOT"},
			},
		},
		{
			name: "legacy flat device_sn",
			payload: map[string]interface{}{
				"device_sn": "SN-ROLLBACK-BOOT",
				"events":    []string{"1 BOOT"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			deviceID := uuid.New()
			taskID := uuid.New()
			subTaskID := uuid.New()
			deviceSN := "SN-ROLLBACK-BOOT"

			parent := &UpgradeTask{
				ID:         taskID,
				TaskType:   TaskTypeRollback,
				Status:     TaskInProgress,
				TotalCount: 1,
			}
			subTask := &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				DeviceID: deviceID,
				DeviceSN: deviceSN,
				Status:   UpgradeRebooting,
			}

			var completedStatus UpgradeState
			var finalizedStatus TaskStatus
			var finalizedResult TaskResult
			var backfilledVersion string

			taskRepo := &svcMockTaskRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
					require.Equal(t, taskID, id)
					return parent, nil
				},
				incrementCountsFn: func(_ context.Context, id uuid.UUID, successDelta, failDelta int) error {
					require.Equal(t, taskID, id)
					parent.SuccessCount += successDelta
					parent.FailCount += failDelta
					return nil
				},
				updateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
					require.Equal(t, taskID, id)
					finalizedStatus = status
					finalizedResult = result
					return nil
				},
			}
			subRepo := &svcMockSubTaskRepo{
				getActiveByDeviceFn: func(_ context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
					require.Equal(t, deviceID, id)
					return subTask, nil
				},
				updateStatusFn: func(_ context.Context, id uuid.UUID, status UpgradeState, _ string) error {
					require.Equal(t, subTaskID, id)
					completedStatus = status
					subTask.Status = status
					return nil
				},
				updateDestByIDFn: func(_ context.Context, id uuid.UUID, destVersion string) error {
					require.Equal(t, subTaskID, id)
					backfilledVersion = destVersion
					return nil
				},
			}
			deviceRepo := &svcMockDeviceRepo{
				getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
					require.Equal(t, deviceSN, sn)
					return &model.Device{
						ID:              deviceID,
						SerialNumber:    deviceSN,
						FirmwareVersion: "BaiBLQ_5.1.11.1",
					}, nil
				},
			}
			mr := miniredis.RunT(t)
			redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { _ = redisClient.Close() })

			exec := NewUpgradeExecutor(
				taskRepo, subRepo, deviceRepo, &svcMockFirmwareRepo{},
				&svcMockCmdQueue{}, nil, redisClient, &svcMockEventBus{}, zap.NewNop(),
			)

			evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, tc.payload)
			require.NoError(t, err)

			require.NoError(t, exec.HandleRebootComplete(ctx, evt))

			assert.Equal(t, UpgradeCompleted, completedStatus)
			assert.Equal(t, 1, parent.SuccessCount)
			assert.Equal(t, 0, parent.FailCount)
			assert.Equal(t, TaskEnded, finalizedStatus)
			assert.Equal(t, TaskResultSuccess, finalizedResult)
			assert.Equal(t, "BaiBLQ_5.1.11.1", backfilledVersion)
		})
	}
}

func TestUpgradeExecutor_HandleRebootComplete_UPSUpgradeVerifiesSoftwareVersion(t *testing.T) {
	ctx := context.Background()
	deviceID := uuid.New()
	taskID := uuid.New()
	subTaskID := uuid.New()
	deviceSN := "SN-UPS-BOOT-OK"
	targetVersion := "UPS-AP-V1.0.1"

	parent := &UpgradeTask{
		ID:         taskID,
		TaskType:   TaskTypeUpgrade,
		Status:     TaskInProgress,
		TotalCount: 1,
	}
	subTask := &UpgradeSubTask{
		ID:          subTaskID,
		TaskID:      taskID,
		DeviceID:    deviceID,
		DeviceSN:    deviceSN,
		Status:      UpgradeRebooting,
		DestVersion: targetVersion,
	}

	var completedStatus UpgradeState
	var finalizedStatus TaskStatus
	var finalizedResult TaskResult

	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			require.Equal(t, taskID, id)
			return parent, nil
		},
		incrementCountsFn: func(_ context.Context, id uuid.UUID, successDelta, failDelta int) error {
			require.Equal(t, taskID, id)
			parent.SuccessCount += successDelta
			parent.FailCount += failDelta
			return nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
			require.Equal(t, taskID, id)
			finalizedStatus = status
			finalizedResult = result
			return nil
		},
	}
	subRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
			require.Equal(t, deviceID, id)
			return subTask, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status UpgradeState, _ string) error {
			require.Equal(t, subTaskID, id)
			completedStatus = status
			subTask.Status = status
			return nil
		},
	}
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, deviceSN, sn)
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    deviceSN,
				ProductClass:    "UPS_M3_BMU",
				FirmwareVersion: "UPS-AP-V1.0.0",
				Technology:      model.TechLTE,
			}, nil
		},
	}
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	exec := NewUpgradeExecutor(
		taskRepo, subRepo, deviceRepo, &svcMockFirmwareRepo{},
		&svcMockCmdQueue{}, nil, redisClient, &svcMockEventBus{}, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, map[string]any{
		"device_id": map[string]any{"serial_number": deviceSN},
		"events":    []string{"1 BOOT"},
		"parameter_list": []tr069.ParameterValueStruct{{
			Name:  upsSoftwareVersionParamPath,
			Value: targetVersion,
		}},
	})
	require.NoError(t, err)

	require.NoError(t, exec.HandleRebootComplete(ctx, evt))

	assert.Equal(t, UpgradeCompleted, completedStatus)
	assert.Equal(t, 1, parent.SuccessCount)
	assert.Equal(t, 0, parent.FailCount)
	assert.Equal(t, TaskEnded, finalizedStatus)
	assert.Equal(t, TaskResultSuccess, finalizedResult)
}

func TestUpgradeExecutor_HandleRebootComplete_UPSUpgradeFailsOnVersionMismatch(t *testing.T) {
	ctx := context.Background()
	deviceID := uuid.New()
	taskID := uuid.New()
	subTaskID := uuid.New()
	deviceSN := "SN-UPS-BOOT-MISMATCH"
	targetVersion := "UPS-AP-V1.0.1"

	parent := &UpgradeTask{
		ID:         taskID,
		TaskType:   TaskTypeUpgrade,
		Status:     TaskInProgress,
		TotalCount: 1,
	}
	subTask := &UpgradeSubTask{
		ID:          subTaskID,
		TaskID:      taskID,
		DeviceID:    deviceID,
		DeviceSN:    deviceSN,
		Status:      UpgradeRebooting,
		DestVersion: targetVersion,
	}

	var failedStatus UpgradeState
	var failureMessage string
	var failureCode FailureCode
	var finalizedStatus TaskStatus
	var finalizedResult TaskResult

	taskRepo := &svcMockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
			require.Equal(t, taskID, id)
			return parent, nil
		},
		incrementCountsFn: func(_ context.Context, id uuid.UUID, successDelta, failDelta int) error {
			require.Equal(t, taskID, id)
			parent.SuccessCount += successDelta
			parent.FailCount += failDelta
			return nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
			require.Equal(t, taskID, id)
			finalizedStatus = status
			finalizedResult = result
			return nil
		},
	}
	subRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
			require.Equal(t, deviceID, id)
			return subTask, nil
		},
		updateStatusWithCodeFn: func(_ context.Context, id uuid.UUID, status UpgradeState, msg string, code FailureCode) error {
			require.Equal(t, subTaskID, id)
			failedStatus = status
			failureMessage = msg
			failureCode = code
			subTask.Status = status
			return nil
		},
	}
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			require.Equal(t, deviceSN, sn)
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    deviceSN,
				ProductClass:    "UPS_M3_BMU",
				FirmwareVersion: "UPS-AP-V1.0.0",
				Technology:      model.TechLTE,
			}, nil
		},
	}
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	exec := NewUpgradeExecutor(
		taskRepo, subRepo, deviceRepo, &svcMockFirmwareRepo{},
		&svcMockCmdQueue{}, nil, redisClient, &svcMockEventBus{}, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, map[string]any{
		"device_sn": deviceSN,
		"events":    []string{"1 BOOT"},
		"parameter_list": []tr069.ParameterValueStruct{{
			Name:  upsSoftwareVersionParamPath,
			Value: "UPS-AP-V1.0.0",
		}},
	})
	require.NoError(t, err)

	require.NoError(t, exec.HandleRebootComplete(ctx, evt))

	assert.Equal(t, UpgradeFailed, failedStatus)
	assert.Equal(t, FailureVersionMismatch, failureCode)
	assert.Contains(t, failureMessage, "reported version UPS-AP-V1.0.0")
	assert.Contains(t, failureMessage, "target version UPS-AP-V1.0.1")
	assert.Equal(t, 0, parent.SuccessCount)
	assert.Equal(t, 1, parent.FailCount)
	assert.Equal(t, TaskEnded, finalizedStatus)
	assert.Equal(t, TaskResultFailed, finalizedResult)
}
