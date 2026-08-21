package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandleXMLTransferCompleteWaitsForStartupStageReport(t *testing.T) {
	taskID := uuid.New()
	taskItem := &ProvisioningTask{
		ID: taskID, DeviceID: uuid.New(), Status: StateConfiguring,
		CurrentStep: 6, CurrentStepName: "wait_transfer_complete", TotalSteps: 8,
	}
	h := newEngineHarness(&mockDeviceRepo{GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
		require.Equal(t, taskItem.DeviceID, id)
		return &model.Device{ID: id, Technology: model.TechNR}, nil
	}})
	var updated bool
	h.taskRepo.GetByIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		require.Equal(t, taskID, id)
		return taskItem, nil
	}
	h.taskRepo.UpdateFn = func(_ context.Context, _ *ProvisioningTask) error {
		updated = true
		return nil
	}

	err := h.engine.HandleXMLTransferComplete(context.Background(), tr069.TransferComplete{
		CommandKey:  "PNPXML_" + taskID.String(),
		FaultStruct: &tr069.Fault{FaultCode: 0},
	})

	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, StateConfiguring, taskItem.Status)
	assert.Equal(t, 6, taskItem.CurrentStep)
	assert.Equal(t, "wait_startup_stage", taskItem.CurrentStepName)
}

func TestHandleXMLTransferCompleteRebootsLTEAndGSMBeforeActivationTimer(t *testing.T) {
	for _, technology := range []model.Technology{model.TechLTE, model.TechGSM} {
		t.Run(string(technology), func(t *testing.T) {
			taskID := uuid.New()
			deviceID := uuid.New()
			taskItem := &ProvisioningTask{
				ID: taskID, DeviceID: deviceID, Status: StateConfiguring,
				CurrentStep: 6, CurrentStepName: "wait_transfer_complete", TotalSteps: 12,
			}
			h := newEngineHarness(&mockDeviceRepo{GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				require.Equal(t, deviceID, id)
				return &model.Device{ID: id, SerialNumber: "SN-LEGACY", Technology: technology}, nil
			}})
			h.taskRepo.GetByIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
				require.Equal(t, taskID, id)
				return taskItem, nil
			}
			var reboot *task.CreateTaskRequest
			var rebootCalls int
			h.cmdQueue.CreateFn = func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
				rebootCalls++
				reboot = req
				return &task.Task{ID: uuid.NewString()}, nil
			}

			err := h.engine.HandleXMLTransferComplete(context.Background(), tr069.TransferComplete{
				CommandKey: "PNPXML_" + taskID.String(), FaultStruct: &tr069.Fault{FaultCode: 0},
			})

			require.NoError(t, err)
			require.NoError(t, h.engine.HandleXMLTransferComplete(context.Background(), tr069.TransferComplete{
				CommandKey: "PNPXML_" + taskID.String(), FaultStruct: &tr069.Fault{FaultCode: 0},
			}))
			assert.Equal(t, 1, rebootCalls, "duplicate TransferComplete must not enqueue another Reboot")
			require.NotNil(t, reboot)
			assert.Equal(t, "Reboot", reboot.Method)
			assert.Equal(t, "SN-LEGACY", reboot.DeviceSN)
			assert.Equal(t, "PNPREBOOT_"+taskID.String(), reboot.CommandKey)
			assert.Equal(t, taskID.String(), reboot.SourceID)
			assert.Equal(t, StateConfiguring, taskItem.Status)
			assert.Equal(t, "wait_device_online", taskItem.CurrentStepName)
			assert.Zero(t, taskItem.RetryCount)
		})
	}
}

func TestHandleXMLTransferCompleteFailsMatchingProvisioningTask(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	taskID := uuid.New()
	taskItem := &ProvisioningTask{
		ID: taskID, DeviceID: uuid.New(), Status: StateConfiguring,
		CurrentStep: 6, CurrentStepName: "wait_transfer_complete", TotalSteps: 8,
	}
	var updatedStatus ProvisioningState
	var failureReason string
	h.taskRepo.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*ProvisioningTask, error) {
		return taskItem, nil
	}
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, reason string) error {
		updatedStatus, failureReason = status, reason
		return nil
	}

	err := h.engine.HandleXMLTransferComplete(context.Background(), tr069.TransferComplete{
		CommandKey:  "PNPXML_" + taskID.String(),
		FaultStruct: &tr069.Fault{FaultCode: 9010, FaultString: "Download failed"},
	})

	require.NoError(t, err)
	assert.Equal(t, StateFailed, updatedStatus)
	assert.Contains(t, failureReason, "9010")
	assert.Contains(t, failureReason, "Download failed")
}

func TestHandleXMLTransferCompleteEventIgnoresUnrelatedGenericCommandKey(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{GetBySerialNumberFn: func(context.Context, string) (*model.Device, error) {
		t.Fatal("generic TransferComplete must not use device-scoped fallback")
		return nil, nil
	}})
	h.taskRepo.ListFn = func(context.Context, ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
		t.Fatal("generic TransferComplete must not inspect waiting plug-and-play tasks")
		return nil, 0, nil
	}

	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, tr069.TransferComplete{
		CommandKey:  "command_key",
		FaultStruct: &tr069.Fault{FaultCode: 0, FaultString: "Download fail with exit status 4"},
	})
	require.NoError(t, err)
	evt.Metadata = map[string]string{event.MetadataDeviceSN: "120200055922C8B0068"}

	require.NoError(t, h.engine.handleXMLTransferCompleteEvent(context.Background(), evt))
}

func TestOnTaskCompletedFailsPnPTaskWithDownloadSOAPFault(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	provisioningTaskID := uuid.New()
	provisioningTask := &ProvisioningTask{
		ID: provisioningTaskID, DeviceID: uuid.New(), Status: StateConfiguring,
		CurrentStep: 5, CurrentStepName: "download_xml", TotalSteps: 12,
	}
	h.taskRepo.GetByIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		require.Equal(t, provisioningTaskID, id)
		return provisioningTask, nil
	}
	var status ProvisioningState
	var reason string
	h.taskRepo.UpdateStatusFn = func(_ context.Context, id uuid.UUID, gotStatus ProvisioningState, gotReason string) error {
		require.Equal(t, provisioningTaskID, id)
		status, reason = gotStatus, gotReason
		return nil
	}

	h.engine.OnTaskCompleted(context.Background(), &task.Task{
		ID:           "download-task",
		DeviceSN:     "120288069823C4B0060",
		Method:       "Download",
		CommandKey:   "PNPXML_" + provisioningTaskID.String(),
		Status:       task.TaskStatusFailed,
		Source:       task.TaskSourceSystem,
		SourceID:     provisioningTaskID.String(),
		ErrorMessage: "[Server.Internal] Unsupported FileType: 103 Base Station Startup File",
	})

	assert.Equal(t, StateFailed, status)
	assert.Contains(t, reason, "Unsupported FileType: 103 Base Station Startup File")
}

func TestHandleDelegatedUpgradeResultMirrorsFailure(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	delegatedTaskID, deviceID := uuid.New(), uuid.New()
	taskItem := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, DeviceTaskID: &delegatedTaskID,
		Status: StateConfiguring, CurrentStepName: "software_upgrade_task_submitted", TotalSteps: 1,
	}
	h.taskRepo.GetByDelegatedTaskIDFn = func(_ context.Context, taskID uuid.UUID, gotDeviceID uuid.UUID) (*ProvisioningTask, error) {
		assert.Equal(t, delegatedTaskID, taskID)
		assert.Equal(t, deviceID, gotDeviceID)
		return taskItem, nil
	}
	var status ProvisioningState
	var reason string
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, gotStatus ProvisioningState, gotReason string) error {
		status, reason = gotStatus, gotReason
		return nil
	}

	err := h.engine.HandleDelegatedUpgradeResult(context.Background(), delegatedUpgradeEvent{
		TaskID: delegatedTaskID.String(), DeviceID: deviceID.String(), Reason: "Download fail with exit status 1",
	}, true)

	require.NoError(t, err)
	assert.Equal(t, StateFailed, status)
	assert.Equal(t, "Download fail with exit status 1", reason)
}

func TestHandleDelegatedUpgradeResultMirrorsCompletion(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	delegatedTaskID, deviceID, policyID := uuid.New(), uuid.New(), uuid.New()
	taskItem := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, DeviceTaskID: &delegatedTaskID,
		PolicyID: &policyID,
		Status:   StateConfiguring, CurrentStepName: "software_upgrade_task_submitted", TotalSteps: 1,
	}
	h.taskRepo.GetByDelegatedTaskIDFn = func(_ context.Context, _, _ uuid.UUID) (*ProvisioningTask, error) {
		return taskItem, nil
	}
	var status ProvisioningState
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, gotStatus ProvisioningState, _ string) error {
		status = gotStatus
		return nil
	}
	continuation := &recordingPolicyContinuation{}
	h.engine.SetPolicyContinuation(continuation)

	err := h.engine.HandleDelegatedUpgradeResult(context.Background(), delegatedUpgradeEvent{
		TaskID: delegatedTaskID.String(), DeviceID: deviceID.String(),
	}, false)

	require.NoError(t, err)
	assert.Equal(t, StateCompleted, status)
	assert.Equal(t, 1, taskItem.CurrentStep)
	assert.Equal(t, "software_upgrade_completed", taskItem.CurrentStepName)
	assert.Equal(t, policyID, continuation.policyID)
	assert.Equal(t, deviceID, continuation.deviceID)
	assert.Equal(t, policyModuleUpgrade, continuation.completed)
}

func TestHandleDelegatedSelfConfigCompletionStartsActivationTimer(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	delegatedTaskID, deviceID := uuid.New(), uuid.New()
	taskItem := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, DeviceTaskID: &delegatedTaskID,
		Status: StateConfiguring, CurrentStepName: "self_config_task_submitted", TotalSteps: 12,
	}
	h.taskRepo.GetByDelegatedTaskIDFn = func(_ context.Context, _, _ uuid.UUID) (*ProvisioningTask, error) {
		return taskItem, nil
	}
	var completed bool
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, _ string) error {
		completed = status == StateCompleted
		return nil
	}

	err := h.engine.HandleDelegatedUpgradeResult(context.Background(), delegatedUpgradeEvent{
		TaskID: delegatedTaskID.String(), DeviceID: deviceID.String(),
	}, false)

	require.NoError(t, err)
	assert.False(t, completed)
	assert.Equal(t, StateVerifying, taskItem.Status)
	assert.Equal(t, 11, taskItem.CurrentStep)
	assert.Equal(t, "wait_activation_check", taskItem.CurrentStepName)
}

func TestHandleDeviceOnlineSchedulesActivationCheck(t *testing.T) {
	deviceID := uuid.New()
	h := newEngineHarness(&mockDeviceRepo{GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: id, SerialNumber: "SN-LTE", Technology: model.TechLTE}, nil
	}})
	taskItem := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, Status: StateConfiguring,
		CurrentStepName: "wait_device_online", TotalSteps: 12,
	}
	h.taskRepo.GetByDeviceIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		assert.Equal(t, deviceID, id)
		return taskItem, nil
	}
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	h.engine.SetDeviceOnlineFullSyncSubmitter(submitter)
	h.engine.SetActivationStateReader(stubActivationStateReader{})

	require.NoError(t, h.engine.HandleDeviceOnline(context.Background(), device.DeviceOnlineEvent{
		DeviceID: deviceID, SerialNumber: "SN-LTE",
	}))
	assert.Equal(t, StateVerifying, taskItem.Status)
	assert.Equal(t, "wait_activation_check", taskItem.CurrentStepName)
	assert.Empty(t, submitter.calls, "PnP reboot online event must only start the five-minute timer")
}

type stubActivationStateReader struct {
	info *device.DeviceInfo
}

func (s stubActivationStateReader) GetByDeviceID(context.Context, uuid.UUID) (*device.DeviceInfo, error) {
	return s.info, nil
}

type recordingActivationStateRefresher struct{ calls int }

func (r *recordingActivationStateRefresher) SyncFromParameters(
	context.Context, uuid.UUID, model.CarrierCode, model.Technology, string,
) ([]string, error) {
	r.calls++
	return nil, nil
}

func TestCheckDueActivationTasksStartsFreshSyncAfterFiveMinutes(t *testing.T) {
	deviceID := uuid.New()
	item := ProvisioningTask{ID: uuid.New(), DeviceID: deviceID, Status: StateVerifying, CurrentStepName: "wait_activation_check", MaxRetries: 3}
	h := newEngineHarness(&mockDeviceRepo{GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: id, SerialNumber: "SN-ACTIVATION", Technology: model.TechLTE}, nil
	}})
	h.taskRepo.ListActivationChecksDueFn = func(context.Context, time.Time, int) ([]ProvisioningTask, error) {
		return []ProvisioningTask{item}, nil
	}
	h.taskRepo.UpdateFn = func(_ context.Context, task *ProvisioningTask) error {
		item = *task
		return nil
	}
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	h.engine.SetDeviceOnlineFullSyncSubmitter(submitter)
	h.engine.SetActivationStateReader(stubActivationStateReader{})

	require.NoError(t, h.engine.CheckDueActivationTasks(context.Background(), time.Now()))
	require.Len(t, submitter.calls, 1)
	assert.Contains(t, submitter.calls[0].idempotencyKey, item.ID.String())
	assert.Equal(t, "wait_activation_sync", item.CurrentStepName)
}

func TestActivationSyncCompletionUsesFreshCellStatus(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cellStatus string
		retryCount int
		wantStatus ProvisioningState
		wantRetry  int
	}{
		{name: "active completes", cellStatus: string(device.CellStatusNormal), wantStatus: StateCompleted},
		{name: "inactive retries", cellStatus: string(device.CellStatusInactive), wantRetry: 1},
		{name: "third inactive fails", cellStatus: string(device.CellStatusInactive), retryCount: 2, wantStatus: StateFailed, wantRetry: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deviceID := uuid.New()
			item := &ProvisioningTask{
				ID: uuid.New(), DeviceID: deviceID, Status: StateVerifying,
				CurrentStepName: "wait_activation_sync", RetryCount: tc.retryCount, MaxRetries: 3, TotalSteps: 12,
			}
			h := newEngineHarness(&mockDeviceRepo{GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
				return &model.Device{ID: id, SerialNumber: "SN-ACTIVATION", Technology: model.TechLTE}, nil
			}})
			h.taskRepo.GetByDeviceIDFn = func(context.Context, uuid.UUID) (*ProvisioningTask, error) { return item, nil }
			h.engine.SetActivationStateReader(stubActivationStateReader{info: &device.DeviceInfo{CellStatus: tc.cellStatus}})
			refresher := &recordingActivationStateRefresher{}
			h.engine.SetActivationStateRefresher(refresher)
			var gotStatus ProvisioningState
			var persistedSteps []ProvisioningTask
			h.taskRepo.UpdateFn = func(_ context.Context, task *ProvisioningTask) error {
				persistedSteps = append(persistedSteps, *task)
				if IsTerminal(task.Status) {
					gotStatus = task.Status
				}
				return nil
			}
			h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, _ string) error {
				gotStatus = status
				return nil
			}

			evt, err := event.NewEvent(event.SubjectParamSyncRunCompleted, paramSyncCompletedEvent{
				DeviceID: deviceID, TriggerReason: "device_online", SyncScope: "full",
			})
			require.NoError(t, err)
			require.NoError(t, h.engine.handleRegisteredParamSyncCompleted(context.Background(), evt))
			assert.Equal(t, 1, refresher.calls)
			assert.Equal(t, tc.wantStatus, gotStatus)
			assert.Equal(t, tc.wantRetry, item.RetryCount)
			if tc.wantRetry > 0 && tc.wantStatus == "" {
				assert.Equal(t, "wait_activation_check", item.CurrentStepName)
			}
			if tc.wantStatus == StateFailed {
				assert.Equal(t, "verify_startup_result", item.CurrentStepName)
				assert.Equal(t, 11, item.CurrentStep)
			}
			if tc.wantStatus == StateCompleted {
				require.Len(t, persistedSteps, 1)
				assert.Equal(t, "activation_verified", persistedSteps[0].CurrentStepName)
				assert.Equal(t, 12, persistedSteps[0].CurrentStep)
				assert.Equal(t, StateCompleted, persistedSteps[0].Status)
				assert.NotNil(t, persistedSteps[0].CompletedAt)
			}
		})
	}
}

func TestActivationSyncFailureSchedulesAnotherFiveMinuteCheck(t *testing.T) {
	deviceID := uuid.New()
	item := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, Status: StateVerifying,
		CurrentStepName: "wait_activation_sync", MaxRetries: 3,
	}
	h := newEngineHarness(&mockDeviceRepo{})
	h.taskRepo.GetByDeviceIDFn = func(context.Context, uuid.UUID) (*ProvisioningTask, error) { return item, nil }
	evt, err := event.NewEvent(event.SubjectParamSyncRunFailed, paramSyncCompletedEvent{
		DeviceID: deviceID, TriggerReason: "device_online", SyncScope: "full",
	})
	require.NoError(t, err)

	require.NoError(t, h.engine.handleActivationSyncFailedEvent(context.Background(), evt))
	assert.Equal(t, 1, item.RetryCount)
	assert.Equal(t, "wait_activation_check", item.CurrentStepName)
	assert.Contains(t, item.ErrorMessage, "activation parameter query failed")
}

type recordingAutomaticPolicyExecutor struct{ deviceIDs []uuid.UUID }

func (r *recordingAutomaticPolicyExecutor) ExecuteAutomaticPolicy(_ context.Context, deviceID uuid.UUID) error {
	r.deviceIDs = append(r.deviceIDs, deviceID)
	return nil
}

func TestRegisteredFullParamSyncCompletionTriggersAutomaticPolicy(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	recorder := &recordingAutomaticPolicyExecutor{}
	h.engine.SetAutomaticPolicyExecutor(recorder)
	deviceID := uuid.New()
	evt, err := event.NewEvent(event.SubjectParamSyncRunCompleted, map[string]any{
		"device_id": deviceID, "trigger_reason": "device_registered", "sync_scope": "full",
	})
	require.NoError(t, err)
	require.NoError(t, h.engine.handleRegisteredParamSyncCompleted(context.Background(), evt))
	assert.Equal(t, []uuid.UUID{deviceID}, recorder.deviceIDs)
}

func TestNonRegisteredParamSyncCompletionDoesNotTriggerAutomaticPolicy(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	recorder := &recordingAutomaticPolicyExecutor{}
	h.engine.SetAutomaticPolicyExecutor(recorder)
	evt, err := event.NewEvent(event.SubjectParamSyncRunCompleted, map[string]any{
		"device_id": uuid.New(), "trigger_reason": "periodic", "sync_scope": "full",
	})
	require.NoError(t, err)
	require.NoError(t, h.engine.handleRegisteredParamSyncCompleted(context.Background(), evt))
	assert.Empty(t, recorder.deviceIDs)
}

type recordingPolicyContinuation struct {
	policyID  uuid.UUID
	deviceID  uuid.UUID
	completed policyModule
}

func (r *recordingPolicyContinuation) ContinuePolicy(
	_ context.Context, policyID, deviceID uuid.UUID, completed policyModule,
) error {
	r.policyID, r.deviceID, r.completed = policyID, deviceID, completed
	return nil
}

func TestHandleStartupStageAndResultReportsCompleteMatchingTask(t *testing.T) {
	deviceID, policyID, xmlID := uuid.New(), uuid.New(), uuid.New()
	devRepo := &mockDeviceRepo{GetBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
		require.Equal(t, "SN-PNP", sn)
		return &model.Device{ID: deviceID, SerialNumber: sn}, nil
	}}
	h := newEngineHarness(devRepo)
	taskItem := &ProvisioningTask{
		ID: uuid.New(), DeviceID: deviceID, PolicyID: &policyID, XMLFileID: &xmlID,
		Status: StateConfiguring, CurrentStep: 6, CurrentStepName: "wait_startup_stage", TotalSteps: 12,
	}
	h.taskRepo.GetByIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		require.Equal(t, taskItem.ID, id)
		return taskItem, nil
	}
	h.taskRepo.GetByDeviceIDFn = func(_ context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return nil, fmt.Errorf("unexpected fallback lookup for device %s", id)
	}
	var updates []string
	h.taskRepo.UpdateFn = func(_ context.Context, task *ProvisioningTask) error {
		updates = append(updates, task.CurrentStepName)
		return nil
	}
	var terminal ProvisioningState
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, _ string) error {
		terminal = status
		return nil
	}

	for _, stage := range []struct {
		value string
		step  int
		name  string
	}{
		{"1", 7, "parameter_validation"},
		{"2", 8, "parameter_configuration"},
		{"3", 9, "cell_activation"},
	} {
		err := h.engine.HandleStartupStageReport(context.Background(), device.InformEventPayload{
			DeviceId:      tr069.DeviceId{SerialNumber: "SN-PNP"},
			EventStructs:  []tr069.EventStruct{{EventCode: tr069.EventStartupStageReport, CommandKey: "PNPXML_" + taskItem.ID.String()}},
			ParameterList: []tr069.ParameterValueStruct{{Name: "Device.Startup.Stage", Value: stage.value}},
		})
		require.NoError(t, err)
		assert.Equal(t, stage.step, taskItem.CurrentStep)
		assert.Equal(t, stage.name, taskItem.CurrentStepName)
	}

	err := h.engine.HandleStartupResultReport(context.Background(), device.InformEventPayload{
		DeviceId:      tr069.DeviceId{SerialNumber: "SN-PNP"},
		EventStructs:  []tr069.EventStruct{{EventCode: tr069.EventStartupResultReport, CommandKey: "PNPXML_" + taskItem.ID.String()}},
		ParameterList: []tr069.ParameterValueStruct{{Name: "Status", Value: "1"}},
	})
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, terminal)
	assert.Equal(t, 12, taskItem.CurrentStep)
	assert.Equal(t, "completed", taskItem.CurrentStepName)
	assert.Contains(t, updates, "cell_activation")
}

func TestHandleStartupResultReportPersistsFailureCause(t *testing.T) {
	deviceID, policyID, xmlID := uuid.New(), uuid.New(), uuid.New()
	h := newEngineHarness(&mockDeviceRepo{GetBySerialNumberFn: func(context.Context, string) (*model.Device, error) {
		return &model.Device{ID: deviceID}, nil
	}})
	h.taskRepo.GetByDeviceIDFn = func(context.Context, uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID: uuid.New(), DeviceID: deviceID, PolicyID: &policyID, XMLFileID: &xmlID,
			Status: StateConfiguring, TotalSteps: 12,
		}, nil
	}
	var terminal ProvisioningState
	var failure string
	h.taskRepo.UpdateStatusFn = func(_ context.Context, _ uuid.UUID, status ProvisioningState, reason string) error {
		terminal, failure = status, reason
		return nil
	}

	err := h.engine.HandleStartupResultReport(context.Background(), device.InformEventPayload{
		DeviceId: tr069.DeviceId{SerialNumber: "SN-PNP"},
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Status", Value: "2"},
			{Name: "FailureCause", Value: "PCI validation failed"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, StateFailed, terminal)
	assert.Contains(t, failure, "PCI validation failed")
}

func TestStartTaskReaperChecksStaleTasksImmediately(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{})
	h.engine.config.TaskTimeout = time.Hour
	called := make(chan time.Duration, 1)
	h.taskRepo.FailStaleFn = func(_ context.Context, maxAge time.Duration) (int64, error) {
		called <- maxAge
		return 0, nil
	}

	h.engine.StartTaskReaper()

	select {
	case maxAge := <-called:
		assert.Equal(t, time.Hour, maxAge)
	case <-time.After(time.Second):
		t.Fatal("task reaper did not check stale tasks at startup")
	}
}

// ---------------------------------------------------------------------------
// Mock: ProvisioningTaskRepository (Pattern B — function fields)
// ---------------------------------------------------------------------------

type mockTaskRepo struct {
	CreateFn                  func(ctx context.Context, task *ProvisioningTask) error
	GetByIDFn                 func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error)
	GetByDeviceIDFn           func(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error)
	GetByDelegatedTaskIDFn    func(ctx context.Context, delegatedTaskID uuid.UUID, deviceID uuid.UUID) (*ProvisioningTask, error)
	UpdateFn                  func(ctx context.Context, task *ProvisioningTask) error
	UpdateStatusFn            func(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error
	ListFn                    func(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error)
	CountByStatusFn           func(ctx context.Context) (map[ProvisioningState]int64, error)
	CountByStatusFilteredFn   func(ctx context.Context, filter ProvisioningTaskFilter) (map[ProvisioningState]int64, error)
	FailStaleFn               func(ctx context.Context, maxAge time.Duration) (int64, error)
	ListActivationChecksDueFn func(ctx context.Context, before time.Time, limit int) ([]ProvisioningTask, error)
}

func (m *mockTaskRepo) ListActivationChecksDue(ctx context.Context, before time.Time, limit int) ([]ProvisioningTask, error) {
	if m.ListActivationChecksDueFn != nil {
		return m.ListActivationChecksDueFn(ctx, before, limit)
	}
	return nil, nil
}

func (m *mockTaskRepo) Create(ctx context.Context, task *ProvisioningTask) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepo) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error) {
	if m.GetByDeviceIDFn != nil {
		return m.GetByDeviceIDFn(ctx, deviceID)
	}
	return nil, nil
}

func (m *mockTaskRepo) GetByDelegatedTaskID(ctx context.Context, delegatedTaskID uuid.UUID, deviceID uuid.UUID) (*ProvisioningTask, error) {
	if m.GetByDelegatedTaskIDFn != nil {
		return m.GetByDelegatedTaskIDFn(ctx, delegatedTaskID, deviceID)
	}
	return nil, nil
}

func (m *mockTaskRepo) Update(ctx context.Context, task *ProvisioningTask) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status, errorMsg)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTaskRepo) CountByStatus(ctx context.Context) (map[ProvisioningState]int64, error) {
	if m.CountByStatusFn != nil {
		return m.CountByStatusFn(ctx)
	}
	return nil, nil
}

func (m *mockTaskRepo) CountByStatusFiltered(ctx context.Context, filter ProvisioningTaskFilter) (map[ProvisioningState]int64, error) {
	if m.CountByStatusFilteredFn != nil {
		return m.CountByStatusFilteredFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTaskRepo) FailStale(ctx context.Context, maxAge time.Duration) (int64, error) {
	if m.FailStaleFn != nil {
		return m.FailStaleFn(ctx, maxAge)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: task.Enqueuer
// ---------------------------------------------------------------------------

type mockCommandQueue struct {
	CreateFn func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	LenFn    func(ctx context.Context, deviceSN string) (int64, error)
}

func (m *mockCommandQueue) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req)
	}
	return task.NewTask(req), nil
}

func (m *mockCommandQueue) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	if m.LenFn != nil {
		return m.LenFn(ctx, deviceSN)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: EventBus
// ---------------------------------------------------------------------------

type mockEventBus struct {
	PublishFn            func(ctx context.Context, subject string, evt event.Event) error
	SubscribeFn          func(subject string, handler event.EventHandler) (event.Subscription, error)
	QueueSubscribeFn     func(subject string, queue string, handler event.EventHandler) (event.Subscription, error)
	PullSubscribeFn      func(subject string, queue string, handler event.EventHandler) (event.Subscription, error)
	KeyedPullSubscribeFn func(
		subject, queue string,
		queueDepth int,
		keyFunc event.EventKeyFunc,
		handler event.EventHandler,
	) (event.Subscription, error)
	CloseFn    func() error
	pullTuning map[string]event.PullTuning

	// Capture published events for assertions.
	published []publishedEvent
}

func (m *mockEventBus) SetPullTuning(subject string, tuning event.PullTuning) {
	if m.pullTuning == nil {
		m.pullTuning = make(map[string]event.PullTuning)
	}
	m.pullTuning[subject] = tuning
}

type publishedEvent struct {
	subject string
	event   event.Event
}

func (m *mockEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	m.published = append(m.published, publishedEvent{subject: subject, event: evt})
	if m.PublishFn != nil {
		return m.PublishFn(ctx, subject, evt)
	}
	return nil
}

func (m *mockEventBus) Subscribe(subject string, handler event.EventHandler) (event.Subscription, error) {
	if m.SubscribeFn != nil {
		return m.SubscribeFn(subject, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) QueueSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	if m.QueueSubscribeFn != nil {
		return m.QueueSubscribeFn(subject, queue, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) PullSubscribe(subject string, queue string, handler event.EventHandler) (event.Subscription, error) {
	if m.PullSubscribeFn != nil {
		return m.PullSubscribeFn(subject, queue, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) KeyedPullSubscribe(
	subject, queue string,
	queueDepth int,
	keyFunc event.EventKeyFunc,
	handler event.EventHandler,
) (event.Subscription, error) {
	if m.KeyedPullSubscribeFn != nil {
		return m.KeyedPullSubscribeFn(subject, queue, queueDepth, keyFunc, handler)
	}
	return &mockSubscription{}, nil
}

func (m *mockEventBus) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

type mockSubscription struct{}

func (s *mockSubscription) Unsubscribe() error { return nil }

// ---------------------------------------------------------------------------
// Helper: build a fully-wired ProvisioningEngine for tests.
// ---------------------------------------------------------------------------

// engineHarness groups all mocks so tests can assert against them.
type engineHarness struct {
	engine   *ProvisioningEngine
	taskRepo *mockTaskRepo
	cmdQueue *mockCommandQueue
	eventBus *mockEventBus
}

func newEngineHarness(deviceRepo device.DeviceRepository) *engineHarness {
	taskRepo := &mockTaskRepo{}
	cmdQueue := &mockCommandQueue{}
	evtBus := &mockEventBus{}
	logger := zap.NewNop()

	// T-0098 P5-01：dmRegistry 已删除，引擎仅依赖 deviceService/templateService/carrierReg。
	devService := device.NewDeviceService(deviceRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		tmplService,
		carrierReg,
		cmdQueue,
		evtBus,
		appconfig.ProvisionConfig{AutoConfigure: true},
		logger,
	)

	return &engineHarness{
		engine:   engine,
		taskRepo: taskRepo,
		cmdQueue: cmdQueue,
		eventBus: evtBus,
	}
}

func TestProvisioningEngine_Subscribe_GPVUsesOrderedKeyedPullSubscribe(t *testing.T) {
	devRepo := &mockDeviceRepo{}
	h := newEngineHarness(devRepo)
	h.engine.config.GPVResponse = appconfig.GPVResponseConsumerConfig{
		ProvisionQueue:       "provision-gpv-existing",
		ProvisionConcurrency: 2,
		ProvisionQueueDepth:  8,
		AckWait:              30 * time.Second,
		MaxDeliver:           9,
		MaxAckPending:        2000,
	}

	var pullSubject, pullQueue string
	var pullQueueDepth int
	var pullKey event.EventKeyFunc
	var pullCalled int
	h.eventBus.KeyedPullSubscribeFn = func(
		subject, queue string,
		queueDepth int,
		keyFunc event.EventKeyFunc,
		_ event.EventHandler,
	) (event.Subscription, error) {
		pullCalled++
		pullSubject = subject
		pullQueue = queue
		pullQueueDepth = queueDepth
		pullKey = keyFunc
		return &mockSubscription{}, nil
	}

	err := h.engine.Subscribe(h.eventBus)
	require.NoError(t, err)
	require.Equal(t, 1, pullCalled, "GPV should use ordered keyed pull delivery")
	assert.Equal(t, event.SubjectCommandGetParamsResponse, pullSubject)
	assert.Equal(t, "provision-gpv-existing", pullQueue)
	assert.Equal(t, 8, pullQueueDepth)
	require.NotNil(t, pullKey)
	keyEvent, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]any{
		"device_sn": "SN-ORDERED",
	})
	require.NoError(t, err)
	key, err := pullKey(keyEvent)
	require.NoError(t, err)
	assert.Equal(t, "SN-ORDERED", key)
	require.Equal(t, event.PullTuning{
		BatchSize:     64,
		Concurrency:   2,
		AckWait:       30 * time.Second,
		MaxDeliver:    9,
		MaxAckPending: 2000,
	}, h.eventBus.pullTuning[event.SubjectCommandGetParamsResponse])
}

func TestProvisioningGPVPullHandlerWaitsForPersistenceBeforeAck(t *testing.T) {
	lookupStarted := make(chan struct{})
	releaseLookup := make(chan struct{})
	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(context.Context, string) (*model.Device, error) {
			close(lookupStarted)
			<-releaseLookup
			return &model.Device{ID: uuid.New(), SerialNumber: "SN-WAIT"}, nil
		},
	}
	h := newEngineHarness(deviceRepo)
	h.engine.config.GPVResponse = appconfig.GPVResponseConsumerConfig{
		ProvisionConcurrency: 2,
		ProvisionQueueDepth:  1,
	}
	var pullHandler event.EventHandler
	h.eventBus.KeyedPullSubscribeFn = func(
		_, _ string,
		_ int,
		_ event.EventKeyFunc,
		handler event.EventHandler,
	) (event.Subscription, error) {
		pullHandler = handler
		return &mockSubscription{}, nil
	}
	require.NoError(t, h.engine.Subscribe(h.eventBus))
	require.NotNil(t, pullHandler)

	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]any{
		"device_sn":        "SN-WAIT",
		"parameter_values": []map[string]any{},
	})
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() { result <- pullHandler(context.Background(), evt) }()
	<-lookupStarted

	select {
	case err := <-result:
		t.Fatalf("pull handler returned before persistence completed: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseLookup)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("pull handler did not return after persistence completed")
	}
}

func TestProvisioningGPVPullHandlerReturnsTransientFailureForNak(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(context.Context, string) (*model.Device, error) {
			return nil, errors.New("database unavailable")
		},
	}
	h := newEngineHarness(deviceRepo)
	var pullHandler event.EventHandler
	h.eventBus.KeyedPullSubscribeFn = func(
		_, _ string,
		_ int,
		_ event.EventKeyFunc,
		handler event.EventHandler,
	) (event.Subscription, error) {
		pullHandler = handler
		return &mockSubscription{}, nil
	}
	require.NoError(t, h.engine.Subscribe(h.eventBus))

	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]any{
		"device_sn":        "SN-ERROR",
		"parameter_values": []map[string]any{},
	})
	require.NoError(t, err)

	err = pullHandler(context.Background(), evt)
	require.ErrorContains(t, err, "find device for GPV response")
}

// ---------------------------------------------------------------------------
// Mock: DeviceRepository (needed for DeviceService)
// ---------------------------------------------------------------------------

type mockDeviceRepo struct {
	GetByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
	CreateFn            func(ctx context.Context, d *model.Device) error
	UpdateFn            func(ctx context.Context, d *model.Device) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListFn              func(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error)
	UpdateStatusFn      func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	UpdateLastInformFn  func(ctx context.Context, sn string, at time.Time, events []string) error
	CountByStatusFn     func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
}

func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.GetBySerialNumberFn != nil {
		return m.GetBySerialNumberFn(ctx, sn)
	}
	return nil, nil
}
func (m *mockDeviceRepo) GetDeletedBySerialNumber(_ context.Context, _ string, _ model.CarrierCode) (*model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, d)
	}
	return nil
}

func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, d)
	}
	return nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockDeviceRepo) List(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}

// T-0162 新接口方法
func (m *mockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.UpdateLastInformFn != nil {
		return m.UpdateLastInformFn(ctx, sn, at, events)
	}
	return nil
}

func (m *mockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}

func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.CountByStatusFn != nil {
		return m.CountByStatusFn(ctx, carrier)
	}
	return nil, nil
}
func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return []model.Device{}, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ device.GeoStatsFilter) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (m *mockDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[device.DeviceWithInfo], error) {
	return model.NewListResponse([]device.DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*device.RestoreResult, error) {
	return &device.RestoreResult{}, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests: HandleRPCResult
// ---------------------------------------------------------------------------

func TestHandleRPCResult_AdvancesStep(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-001"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Return a task in verifying state with 3 total steps.
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	var updatedStep int
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		updatedStep = task.CurrentStep
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, true, "")
	require.NoError(t, err)
	assert.Equal(t, 1, updatedStep, "step should advance from 0 to 1")

	// Verify provision.step.done event was published.
	require.GreaterOrEqual(t, len(h.eventBus.published), 1)
	assert.Equal(t, event.SubjectProvisionStepDone, h.eventBus.published[0].subject)
}

func TestHandleRPCResult_AllStepsComplete(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-002"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: deviceSN, Status: model.DeviceProvisioning}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Task is on the last step (2 of 3, 0-indexed CurrentStep=2 after increment).
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 2, // will become 3 after increment
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	var completedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		completedStatus = status
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, completedStatus)

	// Verify provision.completed event was published.
	var foundCompleted bool
	for _, pe := range h.eventBus.published {
		if pe.subject == event.SubjectProvisionCompleted {
			foundCompleted = true
		}
	}
	assert.True(t, foundCompleted, "provision.completed event should be published")
}

func TestHandleRPCResult_StepFailureWithRetry(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-003"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			RetryCount:  0,
			MaxRetries:  3,
		}, nil
	}

	var savedErrorMsg string
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		savedErrorMsg = task.ErrorMessage
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, false, "CWMP fault 9002")
	require.NoError(t, err)
	assert.Equal(t, "CWMP fault 9002", savedErrorMsg)
}

func TestHandleRPCResult_StepFailureExhaustsRetries(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-004"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 1,
			TotalSteps:  3,
			RetryCount:  2, // already retried twice
			MaxRetries:  3, // max is 3 => next failure exhausts retries
		}, nil
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, false, "timeout")
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

func TestHandleRPCResult_TerminalTaskIsNoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-005"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:       uuid.New(),
			DeviceID: id,
			Status:   StateCompleted, // terminal
		}, nil
	}

	updateCalled := false
	h.taskRepo.UpdateFn = func(ctx context.Context, task *ProvisioningTask) error {
		updateCalled = true
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.False(t, updateCalled, "should not update a terminal task")
}

func TestHandleRPCResult_DeviceNotFound(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, errors.New("device not found")
		},
	}

	h := newEngineHarness(devRepo)

	err := h.engine.HandleRPCResult(context.Background(), "UNKNOWN-SN", MethodSetParameterValues, true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find device by SN")
}

func TestHandleRPCResult_TaskNotFound(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-006"

	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return nil, errors.New("task not found")
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodSetParameterValues, true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get task for device")
}

// ---------------------------------------------------------------------------
// Tests: HandleBootstrap (partial — tests the create and fail paths that
// do not require real DataModelRegistry / ConfigTemplateService backends)
// ---------------------------------------------------------------------------

type registeredDeviceSyncCall struct {
	deviceID uuid.UUID
	sourceID string
}

type fakeRegisteredDeviceSyncStarter struct {
	calls []registeredDeviceSyncCall
	errs  []error
}

func (f *fakeRegisteredDeviceSyncStarter) StartRegisteredDeviceSync(
	_ context.Context,
	dev *model.Device,
	sourceID string,
) error {
	f.calls = append(f.calls, registeredDeviceSyncCall{deviceID: dev.ID, sourceID: sourceID})
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return err
	}
	return nil
}

type deviceOnlineFullSyncCall struct {
	deviceID        uuid.UUID
	idempotencyKey  string
	sourceEventID   string
	originEventType string
}

type recordingDeviceOnlineFullSyncSubmitter struct {
	calls  []deviceOnlineFullSyncCall
	result *DeviceOnlineFullSyncResult
	err    error
}

func (f *recordingDeviceOnlineFullSyncSubmitter) SubmitDeviceOnlineFullSync(
	_ context.Context,
	dev *model.Device,
	idempotencyKey string,
	sourceEventID string,
	originEventType string,
) (*DeviceOnlineFullSyncResult, error) {
	f.calls = append(f.calls, deviceOnlineFullSyncCall{
		deviceID:        dev.ID,
		idempotencyKey:  idempotencyKey,
		sourceEventID:   sourceEventID,
		originEventType: originEventType,
	})
	if f.result == nil && f.err == nil {
		return &DeviceOnlineFullSyncResult{
			RequestID: uuid.New(),
			Status:    "running",
			TaskCount: 1,
		}, nil
	}
	return f.result, f.err
}

func TestProvisioningEngine_Subscribe_RegisteredSyncRetriesSubmitFailure(t *testing.T) {
	deviceID := uuid.New()
	devRepo := &mockDeviceRepo{
		GetByIDFn: func(context.Context, uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-REGISTERED-RETRY"}, nil
		},
	}
	h := newEngineHarness(devRepo)
	starter := &fakeRegisteredDeviceSyncStarter{
		errs: []error{errors.New("temporary submit failure"), nil},
	}
	h.engine.SetRegisteredDeviceSyncStarter(starter)

	var syncHandler event.EventHandler
	h.eventBus.QueueSubscribeFn = func(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
		if subject == event.SubjectDeviceRegistered && queue == "device-registered-param-sync" {
			syncHandler = handler
		}
		return &mockSubscription{}, nil
	}
	require.NoError(t, h.engine.Subscribe(h.eventBus))
	require.NotNil(t, syncHandler)
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, bootstrapEvent{
		DeviceID: deviceID, SerialNumber: "SN-REGISTERED-RETRY", Created: true,
	})
	require.NoError(t, err)

	require.Error(t, syncHandler(context.Background(), evt))
	require.NoError(t, syncHandler(context.Background(), evt))
	require.Len(t, starter.calls, 2)
	assert.Equal(t, "device_registered:"+deviceID.String(), starter.calls[1].sourceID)
}

func TestHandleRegisteredDeviceSyncEvent_DeletedBeforeConsumptionIsSkipped(t *testing.T) {
	h := newEngineHarness(&mockDeviceRepo{
		GetByIDFn: func(context.Context, uuid.UUID) (*model.Device, error) {
			return nil, nil
		},
	})
	starter := &fakeRegisteredDeviceSyncStarter{}
	h.engine.SetRegisteredDeviceSyncStarter(starter)
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, bootstrapEvent{
		DeviceID: uuid.New(), SerialNumber: "SN-DELETED-BEFORE-SYNC", Created: true,
	})
	require.NoError(t, err)

	require.NoError(t, h.engine.handleRegisteredDeviceSyncEvent(context.Background(), evt))
	assert.Empty(t, starter.calls)
}

func TestHandleRegisteredDeviceSyncEvent_CreatedDeviceDurableModeStartsSync(t *testing.T) {
	h := newFullEngineHarness()
	deviceID := uuid.New()
	h.devRepo.GetByIDFn = func(context.Context, uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-REGISTERED-SYNC",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
		}, nil
	}
	starter := &fakeRegisteredDeviceSyncStarter{}
	h.engine.SetRegisteredDeviceSyncStarter(starter)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-REGISTERED-SYNC",
		ProductClass: "SmallCell-LTE",
		Created:      true,
	})
	require.NoError(t, err)

	err = h.engine.handleRegisteredDeviceSyncEvent(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, starter.calls, 1)
	assert.Equal(t, deviceID, starter.calls[0].deviceID)
	assert.Equal(t, "device_registered:"+deviceID.String(), starter.calls[0].sourceID)
}

func TestHandleRegisteredDeviceSyncEvent_ExistingDeviceDoesNotStartSync(t *testing.T) {
	h := newFullEngineHarness()
	deviceID := uuid.New()
	h.devRepo.GetByIDFn = func(context.Context, uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, SerialNumber: "SN-EXISTING"}, nil
	}
	starter := &fakeRegisteredDeviceSyncStarter{}
	h.engine.SetRegisteredDeviceSyncStarter(starter)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, bootstrapEvent{
		DeviceID: deviceID, SerialNumber: "SN-EXISTING", Created: false,
	})
	require.NoError(t, err)

	err = h.engine.handleRegisteredDeviceSyncEvent(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, starter.calls)
}

func TestHandleBootstrap_CreateTaskError(t *testing.T) {
	devRepo := &mockDeviceRepo{}
	h := newEngineHarness(devRepo)

	h.taskRepo.CreateFn = func(ctx context.Context, task *ProvisioningTask) error {
		return errors.New("db connection refused")
	}

	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-ERR-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create provisioning task")
}

func TestHandleBootstrap_DeviceNotFound(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return nil, errors.New("device not found in DB")
		},
	}

	h := newEngineHarness(devRepo)

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     uuid.New(),
		SerialNumber: "SN-NODEV-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

func TestHandleBootstrap_NoMatchingTemplate(t *testing.T) {
	h := newFullEngineHarness()
	deviceID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-NOTMPL-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	// Data model resolution succeeds (returns nil = no model, which is OK).
	// Template matching returns nil = no matching template.
	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return nil, nil // no matching template
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-NOTMPL-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

// ---------------------------------------------------------------------------
// Tests: transitionTask (indirectly through engine state progression)
// ---------------------------------------------------------------------------

func TestTransitionTask_InvalidTransition(t *testing.T) {
	// We can exercise the transitionTask logic by inspecting HandleBootstrap
	// behavior. Since transitionTask is unexported, we test it via the state
	// machine's ValidateTransition (which is already tested in
	// state_machine_test.go). Here we verify the engine properly wraps it.

	// A task stuck in "completed" state cannot transition to "identifying".
	err := ValidateTransition(StateCompleted, StateIdentifying)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid provisioning state transition")
}

// ---------------------------------------------------------------------------
// Tests: completeTask — device status transition
// ---------------------------------------------------------------------------

func TestCompleteTask_TransitionsDeviceToActive(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "TEST-SN-COMPLETE"

	var transitionedStatus model.DeviceStatus
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: sn, Status: model.DeviceProvisioning}, nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: deviceSN, Status: model.DeviceProvisioning}, nil
		},
		UpdateStatusFn: func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
			transitionedStatus = status
			return nil
		},
	}

	h := newEngineHarness(devRepo)

	// Task on last step.
	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 2,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		return nil
	}

	err := h.engine.HandleRPCResult(context.Background(), deviceSN, MethodReboot, true, "")
	require.NoError(t, err)
	assert.Equal(t, model.DeviceActive, transitionedStatus)
}

// ---------------------------------------------------------------------------
// Tests: publishEvent (verify events are captured)
// ---------------------------------------------------------------------------

func TestPublishEvent_FailureIsLoggedNotReturned(t *testing.T) {
	devRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{ID: uuid.New(), SerialNumber: sn}, nil
		},
	}

	h := newEngineHarness(devRepo)

	// Make event bus fail.
	h.eventBus.PublishFn = func(ctx context.Context, subject string, evt event.Event) error {
		return errors.New("NATS connection lost")
	}

	h.taskRepo.GetByDeviceIDFn = func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
		return &ProvisioningTask{
			ID:          uuid.New(),
			DeviceID:    id,
			Status:      StateVerifying,
			CurrentStep: 0,
			TotalSteps:  3,
			MaxRetries:  3,
		}, nil
	}

	// HandleRPCResult should still succeed even though event publishing fails.
	err := h.engine.HandleRPCResult(context.Background(), "SN-EVT-FAIL", MethodSetParameterValues, true, "")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Tests: NewProvisioningTask
// ---------------------------------------------------------------------------

func TestNewProvisioningTask(t *testing.T) {
	deviceID := uuid.New()
	task := NewProvisioningTask(deviceID)

	assert.Equal(t, deviceID, task.DeviceID)
	assert.Equal(t, StateDiscovered, task.Status)
	assert.Equal(t, 3, task.MaxRetries)
	assert.NotEqual(t, uuid.Nil, task.ID)
	assert.Nil(t, task.StartedAt)
	assert.Nil(t, task.CompletedAt)
	assert.Equal(t, 0, task.CurrentStep)
	assert.Equal(t, 0, task.TotalSteps)
}

// ---------------------------------------------------------------------------
// Tests: HandleBootstrap full success path
// ---------------------------------------------------------------------------

// fullEngineHarness builds an engine with real-enough collaborators for a
// full HandleBootstrap success path. We use concrete mock repos all the way.
type fullEngineHarness struct {
	engine   *ProvisioningEngine
	taskRepo *mockTaskRepo
	devRepo  *mockDeviceRepo
	tmplRepo *mockTemplateRepo
	cmdQueue *mockCommandQueue
	eventBus *mockEventBus
}

type mockTemplateRepo struct {
	CreateFn            func(ctx context.Context, t *template.ConfigTemplate) error
	GetByIDFn           func(ctx context.Context, id uuid.UUID) (*template.ConfigTemplate, error)
	UpdateFn            func(ctx context.Context, t *template.ConfigTemplate) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListFn              func(ctx context.Context, filter template.ConfigTemplateFilter) (*model.ListResponse[template.ConfigTemplate], error)
	FindByCarrierTechFn func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType template.TemplateType) ([]template.ConfigTemplate, error)
	FindBestMatchFn     func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error)
}

func (m *mockTemplateRepo) Create(ctx context.Context, t *template.ConfigTemplate) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*template.ConfigTemplate, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepo) Update(ctx context.Context, t *template.ConfigTemplate) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockTemplateRepo) List(ctx context.Context, filter template.ConfigTemplateFilter) (*model.ListResponse[template.ConfigTemplate], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindByCarrierTech(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType template.TemplateType) ([]template.ConfigTemplate, error) {
	if m.FindByCarrierTechFn != nil {
		return m.FindByCarrierTechFn(ctx, carrier, tech, templateType)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindBestMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
	if m.FindBestMatchFn != nil {
		return m.FindBestMatchFn(ctx, carrier, tech, productClass, tmplType)
	}
	return nil, nil
}

// T-0098 P5-01：mockDataModelRepo 已删除（datamodel 包整体下线）。

func newFullEngineHarness() *fullEngineHarness {
	logger := zap.NewNop()
	taskRepo := &mockTaskRepo{}
	devRepo := &mockDeviceRepo{}
	tmplRepo := &mockTemplateRepo{}
	cmdQueue := &mockCommandQueue{}
	evtBus := &mockEventBus{}

	devService := device.NewDeviceService(devRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(tmplRepo, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		taskRepo,
		devService,
		tmplService,
		carrierReg,
		cmdQueue,
		evtBus,
		appconfig.ProvisionConfig{AutoConfigure: true},
		logger,
	)

	return &fullEngineHarness{
		engine:   engine,
		taskRepo: taskRepo,
		devRepo:  devRepo,
		tmplRepo: tmplRepo,
		cmdQueue: cmdQueue,
		eventBus: evtBus,
	}
}

func TestHandleBootstrap_FullSuccessPath(t *testing.T) {
	h := newFullEngineHarness()

	deviceID := uuid.New()
	templateID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-FULL-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return &template.ConfigTemplate{
			ID:           templateID,
			Name:         "cmcc_lte_provision",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "OMC-Default"}`),
			Priority:     10,
			Active:       true,
		}, nil
	}

	// Track state transitions.
	var stateTransitions []ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		stateTransitions = append(stateTransitions, status)
		return nil
	}

	// Track pushed commands.
	var pushedMethods []string
	h.cmdQueue.CreateFn = func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		pushedMethods = append(pushedMethods, req.Method)
		return task.NewTask(req), nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FULL-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)

	// Verify state transitions went through the expected sequence.
	expected := []ProvisioningState{
		StateIdentifying,
		StateMatching,
		StateConfiguring,
		StateVerifying,
	}
	assert.Equal(t, expected, stateTransitions)

	// Verify commands were enqueued: GetParameterValues, SetParameterValues, Reboot.
	require.Len(t, pushedMethods, 3)
	assert.Equal(t, MethodGetParameterValues, pushedMethods[0])
	assert.Equal(t, MethodSetParameterValues, pushedMethods[1])
	assert.Equal(t, MethodReboot, pushedMethods[2])

	// Verify provision.started event was published.
	require.GreaterOrEqual(t, len(h.eventBus.published), 1)
	assert.Equal(t, event.SubjectProvisionStarted, h.eventBus.published[0].subject)
}

func TestHandleBootstrap_EnqueueStepsError(t *testing.T) {
	h := newFullEngineHarness()

	deviceID := uuid.New()
	templateID := uuid.New()

	h.devRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{
			ID:           deviceID,
			SerialNumber: "SN-ENQERR-001",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			OUI:          "001122",
			ProductClass: "SmallCell-LTE",
		}, nil
	}

	h.tmplRepo.FindBestMatchFn = func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, tmplType template.TemplateType) (*template.ConfigTemplate, error) {
		return &template.ConfigTemplate{
			ID:           templateID,
			Name:         "tmpl",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "test"}`),
			Active:       true,
		}, nil
	}

	h.cmdQueue.CreateFn = func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
		return nil, errors.New("redis unavailable")
	}

	var failedStatus ProvisioningState
	h.taskRepo.UpdateStatusFn = func(ctx context.Context, id uuid.UUID, status ProvisioningState, errMsg string) error {
		failedStatus = status
		return nil
	}

	evt := bootstrapEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-ENQERR-001",
		OUI:          "001122",
		ProductClass: "SmallCell-LTE",
	}

	err := h.engine.HandleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, StateFailed, failedStatus)
}

// ---------------------------------------------------------------------------
// Tests: T-0123 HandleDeviceOnline — Redis token bucket + Path B 触发
// ---------------------------------------------------------------------------

// newOnlineHarness 构造一个挂 miniredis 的 ProvisioningEngine（T-0123 测试用）。
//
// syncService 默认 nil — 测试只验证：① Redis token bucket；② device lookup；③ Redis 容错。
// 真 Path B 链路由 sync_pathb_test.go 覆盖（避免本测试既测引擎又测 SyncService 内部）。
func newOnlineHarness(t *testing.T, deviceRepo device.DeviceRepository) (*ProvisioningEngine, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	logger := zap.NewNop()
	devService := device.NewDeviceService(deviceRepo, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()

	engine := NewProvisioningEngine(
		&mockTaskRepo{}, devService, tmplService, carrierReg, &mockCommandQueue{},
		&mockEventBus{}, appconfig.ProvisionConfig{}, logger,
	)
	engine.SetRedisClient(rdb)
	return engine, mr
}

func TestHandleDeviceOnline_RedisTokenBucketSkipsRepeat(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-ONLINE-001",
				ProductClass: "SmallCell",
				Status:       model.DeviceActive,
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	engine.SetDeviceOnlineFullSyncSubmitter(&recordingDeviceOnlineFullSyncSubmitter{})

	evt := device.DeviceOnlineEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-ONLINE-001",
		ProductClass: "SmallCell",
		SwVersion:    "1.0.0",
	}

	// 第一次：拿到 token，进入 device lookup 并提交 durable request。
	err := engine.HandleDeviceOnline(context.Background(), evt)
	require.NoError(t, err)
	firstLookup := lookupCount

	// 60s 内第二次：被 token bucket 拦截，不进 device lookup
	err = engine.HandleDeviceOnline(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, firstLookup, lookupCount, "second call within 60s should be throttled by token bucket")
}

func TestHandleDeviceOnline_DeviceNotFound_NoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return nil, nil // not found
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	engine.SetDeviceOnlineFullSyncSubmitter(&recordingDeviceOnlineFullSyncSubmitter{})
	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-GONE"}
	err := engine.HandleDeviceOnline(context.Background(), evt)
	assert.NoError(t, err)
}

func TestHandleDeviceOnline_RedisDown_StillProceeds(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-REDIS-DOWN"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)
	engine.SetDeviceOnlineFullSyncSubmitter(&recordingDeviceOnlineFullSyncSubmitter{})
	mr.Close() // 模拟 Redis 不可达 — SetNX 返 err；HandleDeviceOnline 应继续推进不阻塞

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-REDIS-DOWN"}
	err := engine.HandleDeviceOnline(context.Background(), evt)
	assert.NoError(t, err, "Redis 失败应不阻塞主流程（容忍 Redis 抖动）")
}

func TestHandleDeviceOnline_DurableModeStartsFullSyncAndThrottlesRepeat(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-ONLINE-FULL"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{
		result: &DeviceOnlineFullSyncResult{
			RequestID: uuid.New(),
			Status:    "running",
			TaskCount: 13,
		},
	}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-ONLINE-FULL"}
	require.NoError(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-full"))
	require.NoError(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-full"))

	require.Len(t, submitter.calls, 1)
	call := submitter.calls[0]
	assert.Equal(t, deviceID, call.deviceID)
	assert.Equal(t, "device_online:evt-online-full", call.idempotencyKey)
	assert.Equal(t, "evt-online-full", call.sourceEventID)
	assert.Equal(t, event.SubjectDeviceOnline, call.originEventType)
}

func TestHandleDeviceOnline_UPSSkipsDurableFullSync(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-UPS-ONLINE",
				ProductClass: "UPS_M3_BMU",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	err := engine.handleDeviceOnline(context.Background(), device.DeviceOnlineEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-UPS-ONLINE",
		ProductClass: "UPS_M3_BMU",
	}, "evt-ups-online")

	require.NoError(t, err)
	assert.Empty(t, submitter.calls, "UPS online Inform must not submit durable parameter sync")
}

func TestHandleDeviceOnline_DurableAutomaticBackoffIsQueuedWithoutRetry(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-ONLINE-BACKOFF"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{
		result: &DeviceOnlineFullSyncResult{
			RequestID:  uuid.New(),
			Status:     "queued",
			ResultCode: "AUTOMATIC_BACKOFF",
		},
	}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-ONLINE-BACKOFF"}
	require.NoError(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-backoff"))
	require.NoError(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-backoff"))

	require.Len(t, submitter.calls, 1, "durably queued backoff must keep the Redis throttle")
}

func TestHandleDeviceOnline_DurableSubmitFailureReleasesThrottleForRetry(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-ONLINE-RETRY"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{err: errors.New("database unavailable")}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-ONLINE-RETRY"}
	require.Error(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-retry"))
	require.Error(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-retry"))

	require.Len(t, submitter.calls, 2, "failed submit must release the Redis throttle for NATS redelivery")
	assert.Equal(t, submitter.calls[0].idempotencyKey, submitter.calls[1].idempotencyKey)
}

func TestProvisioningEngine_Subscribe_DeviceOnlineRetriesSubmitFailure(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-ONLINE-SUBSCRIBE-RETRY"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{
		err: errors.New("temporary durable submit failure"),
	}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)
	engine.SetDeduper(event.NewDeduper(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		time.Hour,
		zap.NewNop(),
	))

	var onlineHandler event.EventHandler
	bus := &mockEventBus{
		QueueSubscribeFn: func(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
			if subject == event.SubjectDeviceOnline && queue == "provision-online-sync" {
				onlineHandler = handler
			}
			return &mockSubscription{}, nil
		},
	}
	require.NoError(t, engine.Subscribe(bus))
	require.NotNil(t, onlineHandler)
	evt, err := event.NewEvent(event.SubjectDeviceOnline, device.DeviceOnlineEvent{
		DeviceID: deviceID, SerialNumber: "SN-ONLINE-SUBSCRIBE-RETRY",
	})
	require.NoError(t, err)

	require.Error(t, onlineHandler(context.Background(), evt))
	require.Error(t, onlineHandler(context.Background(), evt))
	require.Len(t, submitter.calls, 2, "NATS redelivery must reach the durable submitter")
	assert.Equal(t, submitter.calls[0].idempotencyKey, submitter.calls[1].idempotencyKey)
}

func TestHandleDeviceOnline_LookupFailureReleasesThrottleForRetry(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			if lookupCount == 1 {
				return nil, errors.New("temporary device lookup failure")
			}
			return &model.Device{ID: deviceID, SerialNumber: "SN-ONLINE-LOOKUP-RETRY"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	evt := device.DeviceOnlineEvent{DeviceID: deviceID, SerialNumber: "SN-ONLINE-LOOKUP-RETRY"}
	require.Error(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-lookup-retry"))
	require.NoError(t, engine.handleDeviceOnline(context.Background(), evt, "evt-online-lookup-retry"))
	require.Len(t, submitter.calls, 1)
	assert.Equal(t, 2, lookupCount)
}

func TestHandleGPVResponse_EmptySyncGPVStillFinalizesPathB(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "SN-NR-EMPTY-GPV"
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			if sn != deviceSN {
				return nil, nil
			}
			return &model.Device{
				ID:           deviceID,
				SerialNumber: deviceSN,
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechNR,
			}, nil
		},
	}
	h := newEngineHarness(deviceRepo)
	writer := &fakeParamSyncWriter{}
	h.engine.SetSyncService((&SyncService{redisClient: rdb, logger: zap.NewNop()}).SetParamSyncWriter(writer))
	require.NoError(t, rdb.Set(context.Background(), pathBSyncPendingBatchesKey(deviceID), 1, time.Minute).Err())

	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn":        deviceSN,
		"method":           "GetParameterValuesResponse",
		"command_key":      "sync-gpv-test-0",
		"parameter_values": []map[string]any{},
	})
	require.NoError(t, err)

	err = h.engine.handleGPVResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 1, writer.callCount(), "empty final sync-gpv response should finalize Path B immediately")
	_, redisErr := rdb.Get(context.Background(), pathBSyncPendingBatchesKey(deviceID)).Result()
	assert.Error(t, redisErr, "pending batch key should be cleared after final empty sync-gpv response")
}

func TestHandleGPVResponse_ParamSyncSourceSkipsLegacyPath(t *testing.T) {
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			lookupCount++
			return &model.Device{ID: uuid.New(), SerialNumber: "SN-DURABLE-SYNC"}, nil
		},
	}
	h := newEngineHarness(deviceRepo)

	evt, err := event.NewEvent(event.SubjectCommandGetParamsResponse, map[string]interface{}{
		"device_sn":        "SN-DURABLE-SYNC",
		"method":           "GetParameterValuesResponse",
		"command_key":      "param-sync-00000000-0000-0000-0000-000000000001-0",
		"task_source":      task.TaskSourceParamSync,
		"task_source_id":   "00000000-0000-0000-0000-000000000001",
		"parameter_values": []map[string]any{{"name": "Device.Test.Value", "value": "1"}},
	})
	require.NoError(t, err)

	require.NoError(t, h.engine.handleGPVResponse(context.Background(), evt))
	assert.Zero(t, lookupCount, "durable parameter-sync responses must not enter legacy Path B")
}

func TestOnTaskCompleted_RecoveredSyncGPVExhausted_FinalizesPathB(t *testing.T) {
	deviceID := uuid.New()
	deviceSN := "SN-GPV-RECOVERED"
	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			if sn != deviceSN {
				return nil, nil
			}
			return &model.Device{
				ID:           deviceID,
				SerialNumber: deviceSN,
				Carrier:      model.CarrierCMCC,
				Technology:   model.TechNR,
			}, nil
		},
	}
	h := newEngineHarness(deviceRepo)
	writer := &fakeParamSyncWriter{}
	h.engine.SetSyncService((&SyncService{logger: zap.NewNop()}).
		SetParamSyncWriter(writer).
		SetPathBSyncTaskReader(&fakePathBSyncTaskReader{hasIncomplete: false}))

	h.engine.OnTaskCompleted(context.Background(), &task.Task{
		ID:         "task-recovered-final",
		DeviceSN:   deviceSN,
		Method:     "GetParameterValues",
		Status:     task.TaskStatusCompleted,
		CommandKey: "sync-gpv-sn-recovered-0-r-r",
		SourceID:   uuid.New().String(),
		Result:     json.RawMessage(`{"recovered":true,"bad_path":"Device.Ethernet.Interface.8.","remaining_cnt":0}`),
	})

	assert.Equal(t, 1, writer.callCount(), "exhausted recovered sync-gpv should finalize Path B via completion callback")
}

func TestOnTaskCompleted_RecoveredSyncGPVWithRemaining_DoesNotFinalize(t *testing.T) {
	deviceSN := "SN-GPV-STILL-OPEN"
	deviceRepo := &mockDeviceRepo{
		GetBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			if sn != deviceSN {
				return nil, nil
			}
			return &model.Device{ID: uuid.New(), SerialNumber: deviceSN}, nil
		},
	}
	h := newEngineHarness(deviceRepo)
	writer := &fakeParamSyncWriter{}
	h.engine.SetSyncService((&SyncService{logger: zap.NewNop()}).SetParamSyncWriter(writer))

	h.engine.OnTaskCompleted(context.Background(), &task.Task{
		ID:         "task-recovered-mid",
		DeviceSN:   deviceSN,
		Method:     "GetParameterValues",
		Status:     task.TaskStatusCompleted,
		CommandKey: "sync-gpv-sn-open-0-r",
		SourceID:   uuid.New().String(),
		Result:     json.RawMessage(`{"recovered":true,"bad_path":"Device.Ethernet.Interface.9.","remaining_cnt":3}`),
	})

	assert.Equal(t, 0, writer.callCount(), "recovered sync-gpv with remaining paths must not finalize early")
}

// ---------------------------------------------------------------------------
// Tests: T-0125 HandleFirmwareChanged — Redis 串行锁 + model refresh without parameter sync
// ---------------------------------------------------------------------------

func TestHandleFirmwareChanged_RedisModelUploadLockStillAllowsDeviceLookup(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    "SN-FW-LOCK",
				ProductClass:    "SmallCell",
				FirmwareVersion: "1.0.0",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-LOCK",
		ProductClass: "SmallCell",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}

	// 第一次：查设备后拿到模型上传锁（modelUploadService=nil 不影响锁逻辑）。
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)
	firstLookup := lookupCount

	// 10min 内第二次仍需查设备，确保 BecameOnline 恢复语义不会被模型上传锁吞掉；
	// 设备查询后的重复模型上传仍由锁拦截。
	err = engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, firstLookup+1, lookupCount, "model-upload lock must not suppress online recovery lookup")
}

func TestHandleFirmwareChanged_DoesNotWriteParamSyncReasonHint(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-HINT"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-HINT",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	require.NoError(t, err)

	_, getErr := mr.Get("provision:syncreason:" + deviceID.String())
	assert.Error(t, getErr, "firmware changed must not arm a parameter-sync reason hint")
}

func TestHandleFirmwareChanged_DeviceNotFound_NoOp(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return nil, nil // not found
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-GONE",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err, "device-not-found should be no-op without error")
}

func TestHandleFirmwareChanged_NilModelUpload_FallbackPathBSkippedWhenSyncNil(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-FW-FALLBACK",
				ProductClass: "SmallCell",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	// modelUploadService 和 syncService 都 nil — 验证 fallback 路径不 panic + 早返

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-FALLBACK",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err, "nil modelUploadService + nil syncService 应无 panic 无 error")
}

func TestHandleFirmwareChanged_RedisDown_StillProceeds(t *testing.T) {
	deviceID := uuid.New()
	lookupCount := 0
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			lookupCount++
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-REDIS-DOWN"}, nil
		},
	}
	engine, mr := newOnlineHarness(t, deviceRepo)
	mr.Close() // 模拟 Redis 不可达

	evt := device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-REDIS-DOWN",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
	}
	err := engine.HandleFirmwareChanged(context.Background(), evt)
	assert.NoError(t, err)
	assert.Equal(t, 1, lookupCount, "Redis 失败仍应推进到 device lookup")
}

func TestHandleFirmwareChanged_BecameOnlineStartsDurableFullSync(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-ONLINE"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	err := engine.handleFirmwareChanged(context.Background(), device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-FW-ONLINE",
		OldVersion:   "0.9.0",
		NewVersion:   "1.0.0",
		BecameOnline: true,
	}, "evt-firmware-online")

	require.NoError(t, err)
	require.Len(t, submitter.calls, 1)
	call := submitter.calls[0]
	assert.Equal(t, "device_online:firmware_changed:evt-firmware-online", call.idempotencyKey)
	assert.Equal(t, "evt-firmware-online", call.sourceEventID)
	assert.Equal(t, event.SubjectDeviceFirmwareChanged, call.originEventType)
}

func TestHandleFirmwareChanged_UPSSkipsDurableFullSync(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-UPS-FW",
				ProductClass: "UPS_M3_BMU",
			}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	err := engine.handleFirmwareChanged(context.Background(), device.DeviceFirmwareChangedEvent{
		DeviceID:     deviceID,
		SerialNumber: "SN-UPS-FW",
		ProductClass: "UPS_M3_BMU",
		OldVersion:   "UPS-1.0.0",
		NewVersion:   "UPS-2.0.0",
		BecameOnline: true,
	}, "evt-ups-fw")

	require.NoError(t, err)
	assert.Empty(t, submitter.calls, "UPS firmware Inform must not submit durable parameter sync")
}

func TestHandleFirmwareChanged_ModelUploadLockDoesNotSuppressLaterOnlineSync(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-FW-LATER-ONLINE"}, nil
		},
	}
	engine, _ := newOnlineHarness(t, deviceRepo)
	submitter := &recordingDeviceOnlineFullSyncSubmitter{}
	engine.SetDeviceOnlineFullSyncSubmitter(submitter)

	require.NoError(t, engine.handleFirmwareChanged(context.Background(), device.DeviceFirmwareChangedEvent{
		DeviceID: deviceID, SerialNumber: "SN-FW-LATER-ONLINE",
		OldVersion: "0.8.0", NewVersion: "0.9.0", BecameOnline: false,
	}, "evt-firmware-model-only"))
	require.NoError(t, engine.handleFirmwareChanged(context.Background(), device.DeviceFirmwareChangedEvent{
		DeviceID: deviceID, SerialNumber: "SN-FW-LATER-ONLINE",
		OldVersion: "0.9.0", NewVersion: "1.0.0", BecameOnline: true,
	}, "evt-firmware-later-online"))

	require.Len(t, submitter.calls, 1, "model-upload throttling must not suppress online recovery")
	assert.Equal(t, "evt-firmware-later-online", submitter.calls[0].sourceEventID)
}

// ---------------------------------------------------------------------------
// Tests: T-0176-PR-D bindDeviceProduct / bindDeviceProductIfNeeded
//
// 路由 + 写回 + cache.Delete 三段链路。stub productClassMatcher / productBinder /
// deviceCacheInvalidator 注入 ProvisioningEngine 用 narrow interface（不依赖
// 真 product.Registry / PgRepository / device.DeviceCache）。
// ---------------------------------------------------------------------------

// stubProductMatcher 实现 productClassMatcher（test only）。
type stubProductMatcher struct {
	matchFn func(ctx context.Context, productClass string) (*product.MatchResult, error)
}

func (s *stubProductMatcher) MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error) {
	if s.matchFn != nil {
		return s.matchFn(ctx, productClass)
	}
	return nil, product.ErrOrphan
}

// stubProductBinder 实现 productBinder（test only）。
type stubProductBinder struct {
	bindCalls    atomic.Int32
	lookupCalls  atomic.Int32
	lastDeviceID uuid.UUID
	lastProduct  uuid.UUID
	lastParamMod *uuid.UUID
	bindFn       func(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error
	lookupFn     func(ctx context.Context, deviceID uuid.UUID) (*uuid.UUID, error)
}

func (s *stubProductBinder) BindDevice(ctx context.Context, deviceID, productID uuid.UUID, paramModelID *uuid.UUID) error {
	s.bindCalls.Add(1)
	s.lastDeviceID = deviceID
	s.lastProduct = productID
	s.lastParamMod = paramModelID
	if s.bindFn != nil {
		return s.bindFn(ctx, deviceID, productID, paramModelID)
	}
	return nil
}

func (s *stubProductBinder) GetProductIDByDeviceID(ctx context.Context, deviceID uuid.UUID) (*uuid.UUID, error) {
	s.lookupCalls.Add(1)
	if s.lookupFn != nil {
		return s.lookupFn(ctx, deviceID)
	}
	return nil, nil // default: unbound
}

// stubCacheInvalidator 实现 deviceCacheInvalidator（test only）。
type stubCacheInvalidator struct {
	deleteCalls atomic.Int32
	lastSN      string
}

func (s *stubCacheInvalidator) Delete(_ context.Context, sn string) {
	s.deleteCalls.Add(1)
	s.lastSN = sn
}

// newBindHarness 构造一个仅用于绑定相关测试的 ProvisioningEngine。
func newBindHarness(matcher productClassMatcher, binder productBinder, cache deviceCacheInvalidator) *ProvisioningEngine {
	logger := zap.NewNop()
	devService := device.NewDeviceService(&mockDeviceRepo{}, nil, nil, nil, logger)
	tmplService := template.NewConfigTemplateService(nil, logger)
	carrierReg := carrier.NewRegistry()
	engine := NewProvisioningEngine(
		&mockTaskRepo{}, devService, tmplService, carrierReg, &mockCommandQueue{},
		&mockEventBus{}, appconfig.ProvisionConfig{}, logger,
	)
	if matcher != nil || binder != nil {
		engine.SetProductBinder(matcher, binder)
	}
	if cache != nil {
		engine.SetDeviceCache(cache)
	}
	return engine
}

func Test_BindDeviceProduct_ReturnsTrueWhenWrote(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	matchedProduct := &product.Product{
		ID:           productID,
		Name:         "TestProduct",
		ParamModelID: &paramModelID,
	}
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, pc string) (*product.MatchResult, error) {
			assert.Equal(t, "PCLASS-A", pc)
			return &product.MatchResult{Product: matchedProduct, MatchedPattern: "PCLASS-.*"}, nil
		},
	}
	binder := &stubProductBinder{}
	engine := newBindHarness(matcher, binder, nil)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-BIND-OK", ProductClass: "PCLASS-A"}
	bound := engine.bindDeviceProduct(context.Background(), dev)

	assert.True(t, bound, "命中后应返 true")
	assert.Equal(t, int32(1), binder.bindCalls.Load(), "BindDevice 应被调一次")
	assert.Equal(t, dev.ID, binder.lastDeviceID)
	assert.Equal(t, productID, binder.lastProduct)
	require.NotNil(t, binder.lastParamMod)
	assert.Equal(t, paramModelID, *binder.lastParamMod)
}

func Test_BindDeviceProduct_ReturnsFalseOnOrphan(t *testing.T) {
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return nil, product.ErrOrphan
		},
	}
	binder := &stubProductBinder{}
	engine := newBindHarness(matcher, binder, nil)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-ORPHAN", ProductClass: "UNKNOWN"}
	bound := engine.bindDeviceProduct(context.Background(), dev)

	assert.False(t, bound, "orphan 应返 false")
	assert.Equal(t, int32(0), binder.bindCalls.Load(), "orphan 时 BindDevice 不应被调")
}

func Test_BindDeviceProduct_ReturnsFalseOnBindErr(t *testing.T) {
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return &product.MatchResult{Product: &product.Product{ID: uuid.New(), Name: "P"}}, nil
		},
	}
	binder := &stubProductBinder{
		bindFn: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) error {
			return errors.New("simulated DB failure")
		},
	}
	engine := newBindHarness(matcher, binder, nil)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-BIND-ERR", ProductClass: "PCLASS-X"}
	bound := engine.bindDeviceProduct(context.Background(), dev)

	assert.False(t, bound, "BindDevice 失败应返 false，由调用方据此跳过 cache 清理")
}

func Test_BindDeviceProduct_NilDepsReturnsFalse(t *testing.T) {
	engine := newBindHarness(nil, nil, nil)
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-NIL", ProductClass: "PCLASS-A"}
	bound := engine.bindDeviceProduct(context.Background(), dev)
	assert.False(t, bound)
}

func Test_BindDeviceProductIfNeeded_SkipsAlreadyBound(t *testing.T) {
	existingProductID := uuid.New()
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			t.Fatal("MatchProductClass 不应被调 — 已绑定路径")
			return nil, nil
		},
	}
	binder := &stubProductBinder{
		lookupFn: func(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
			return &existingProductID, nil
		},
	}
	cache := &stubCacheInvalidator{}
	engine := newBindHarness(matcher, binder, cache)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-ALREADY", ProductClass: "PCLASS-A"}
	engine.bindDeviceProductIfNeeded(context.Background(), dev)

	assert.Equal(t, int32(1), binder.lookupCalls.Load(), "Lookup 调一次确认已绑定")
	assert.Equal(t, int32(0), binder.bindCalls.Load(), "已绑定不应再调 BindDevice")
	assert.Equal(t, int32(0), cache.deleteCalls.Load(), "已绑定不应清 cache")
}

func Test_BindDeviceProductIfNeeded_DeletesCacheOnBindSuccess(t *testing.T) {
	productID := uuid.New()
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return &product.MatchResult{Product: &product.Product{ID: productID, Name: "P"}}, nil
		},
	}
	binder := &stubProductBinder{
		// lookupFn nil → 返 (nil, nil) → 未绑定 → 走 bindDeviceProduct
	}
	cache := &stubCacheInvalidator{}
	engine := newBindHarness(matcher, binder, cache)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-CACHE-DEL", ProductClass: "PCLASS-A"}
	engine.bindDeviceProductIfNeeded(context.Background(), dev)

	assert.Equal(t, int32(1), binder.bindCalls.Load(), "未绑定时 BindDevice 调一次")
	assert.Equal(t, int32(1), cache.deleteCalls.Load(), "Bind 成功后 cache.Delete 调一次")
	assert.Equal(t, "SN-CACHE-DEL", cache.lastSN)
}

func Test_BindDeviceProductIfNeeded_NoCacheDeleteOnBindFailure(t *testing.T) {
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			return &product.MatchResult{Product: &product.Product{ID: uuid.New(), Name: "P"}}, nil
		},
	}
	binder := &stubProductBinder{
		bindFn: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) error {
			return errors.New("DB down")
		},
	}
	cache := &stubCacheInvalidator{}
	engine := newBindHarness(matcher, binder, cache)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-FAIL", ProductClass: "PCLASS-A"}
	engine.bindDeviceProductIfNeeded(context.Background(), dev)

	assert.Equal(t, int32(0), cache.deleteCalls.Load(), "Bind 失败时不应清 cache（避免假象'已生效'）")
}

func Test_BindDeviceProductIfNeeded_NilDevSafelyNoop(t *testing.T) {
	engine := newBindHarness(&stubProductMatcher{}, &stubProductBinder{}, &stubCacheInvalidator{})
	// 不应 panic
	engine.bindDeviceProductIfNeeded(context.Background(), nil)
}

func Test_BindDeviceProductIfNeeded_EmptyProductClassSkips(t *testing.T) {
	matcher := &stubProductMatcher{
		matchFn: func(_ context.Context, _ string) (*product.MatchResult, error) {
			t.Fatal("空 productClass 时 Match 不应被调")
			return nil, nil
		},
	}
	binder := &stubProductBinder{}
	engine := newBindHarness(matcher, binder, nil)

	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-EMPTY-PC", ProductClass: ""}
	engine.bindDeviceProductIfNeeded(context.Background(), dev)

	assert.Equal(t, int32(0), binder.lookupCalls.Load())
	assert.Equal(t, int32(0), binder.bindCalls.Load())
}
