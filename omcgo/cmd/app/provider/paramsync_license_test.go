package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/paramsync"
)

type fakeLicenseParamSyncSubmitter struct {
	command paramsync.SubmitCommand
	result  *paramsync.SubmitResult
	err     error
	calls   int
}

func (f *fakeLicenseParamSyncSubmitter) Submit(_ context.Context, command paramsync.SubmitCommand) (*paramsync.SubmitResult, error) {
	f.calls++
	f.command = command
	return f.result, f.err
}

func (f *fakeLicenseParamSyncSubmitter) GetRequest(context.Context, uuid.UUID) (*paramsync.SyncRequest, error) {
	return nil, nil
}

func TestSubmitLicenseParamSync_UsesDurableLicenseRequest(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "license-device"}
	sourceID := uuid.NewString()
	paths := []string{"Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE."}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning, TaskCount: 2,
	}}

	taskCount, err := submitLicenseParamSync(context.Background(), submitter, dev, sourceID, paths)
	require.NoError(t, err)
	assert.Equal(t, 2, taskCount)
	assert.Equal(t, "license", submitter.command.CallerType)
	assert.Equal(t, paramsync.TriggerLicense, submitter.command.TriggerReason)
	assert.Equal(t, paramsync.SyncScopePartial, submitter.command.Scope)
	assert.Equal(t, sourceID, submitter.command.IdempotencyKey)
	assert.Equal(t, paths, submitter.command.RequestedPaths)
}

func TestSubmitLicenseParamSync_RejectsUnavailablePlan(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "license-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRejected, ResultCode: paramsync.ResultCodePathBUnavailable,
	}}

	_, err := submitLicenseParamSync(context.Background(), submitter, dev, uuid.NewString(), []string{"Device.LICENSE."})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "durable license parameter sync unavailable")
}

func TestSubmitRegisteredDeviceSync_UsesDurableFullRequest(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "registered-device"}
	sourceID := "device_registered:" + dev.ID.String()
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning, TaskCount: 4,
	}}

	err := submitRegisteredDeviceSync(
		context.Background(),
		submitter,
		dev,
		sourceID,
	)

	require.NoError(t, err)
	assert.Equal(t, "provision", submitter.command.CallerType)
	assert.Equal(t, paramsync.TriggerDeviceRegistered, submitter.command.TriggerReason)
	assert.Equal(t, paramsync.SyncScopeFull, submitter.command.Scope)
	assert.Equal(t, sourceID, submitter.command.IdempotencyKey)
	assert.Equal(t, sourceID, submitter.command.SourceEventID)
	assert.Equal(t, "device_registered", submitter.command.OriginEventType)
}

func TestSubmitRegisteredDeviceSync_PathUnavailableDoesNotFallBackOrFailProvisioning(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "registered-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status:     paramsync.RequestStatusRejected,
		ResultCode: paramsync.ResultCodePathBUnavailable,
	}}

	err := submitRegisteredDeviceSync(
		context.Background(),
		submitter,
		dev,
		"device_registered:"+dev.ID.String(),
	)

	require.NoError(t, err)
}

func TestSubmitRegisteredDeviceSync_RetryUsesNewIdempotencyKeyAndStableSourceEvent(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "registered-device"}
	sourceID := "device_registered:" + dev.ID.String()
	attemptKey := sourceID + ":" + uuid.NewString()
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning,
	}}

	err := submitRegisteredDeviceSync(context.Background(), submitter, dev, sourceID, attemptKey)

	require.NoError(t, err)
	assert.Equal(t, attemptKey, submitter.command.IdempotencyKey)
	assert.Equal(t, sourceID, submitter.command.SourceEventID)
}

func TestSubmitRegisteredDeviceSync_SkipsUPS(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS_M3_BMU"}
	submitter := &fakeLicenseParamSyncSubmitter{}

	err := submitRegisteredDeviceSync(context.Background(), submitter, dev, "device_registered:"+dev.ID.String())

	require.NoError(t, err)
	assert.Zero(t, submitter.calls)
}

func TestParamSyncStarter_RegisteredDeviceUsesDurableSync(t *testing.T) {
	dev := &model.Device{ID: uuid.Nil, SerialNumber: "registered-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning,
	}}
	starter := &paramSyncStarter{
		service: submitter,
	}

	err := starter.StartRegisteredDeviceSync(
		context.Background(),
		dev,
		"device_registered:"+dev.ID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, paramsync.TriggerDeviceRegistered, submitter.command.TriggerReason)
}

func TestParamSyncStarter_RegisteredDeviceSkipsUPS(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS/FSU2024"}
	submitter := &fakeLicenseParamSyncSubmitter{}
	starter := &paramSyncStarter{service: submitter}

	err := starter.StartRegisteredDeviceSync(context.Background(), dev, "device_registered:"+dev.ID.String())

	require.NoError(t, err)
	assert.Zero(t, submitter.calls)
}

func TestParamSyncStarter_DeviceOnlineUsesDirectDurableFullRequest(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "online-device"}
	requestID := uuid.New()
	runID := uuid.New()
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		RequestID: requestID,
		RunID:     &runID,
		Status:    paramsync.RequestStatusRunning,
		TaskCount: 13,
	}}
	starter := &paramSyncStarter{
		service: submitter,
	}

	result, err := starter.SubmitDeviceOnlineFullSync(
		context.Background(),
		dev,
		"device_online:event-1",
		"event-1",
		"device.online",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, requestID, result.RequestID)
	assert.Equal(t, &runID, result.RunID)
	assert.Equal(t, "running", result.Status)
	assert.Equal(t, 13, result.TaskCount)
	assert.Equal(t, "provision", submitter.command.CallerType)
	assert.Equal(t, paramsync.TriggerDeviceOnline, submitter.command.TriggerReason)
	assert.Equal(t, paramsync.SyncScopeFull, submitter.command.Scope)
	assert.Empty(t, submitter.command.RequestedPaths)
	assert.Equal(t, "device_online:event-1", submitter.command.IdempotencyKey)
	assert.Equal(t, "event-1", submitter.command.SourceEventID)
	assert.Equal(t, "device.online", submitter.command.OriginEventType)
}

func TestParamSyncStarter_DeviceOnlineSkipsUPS(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS_M3_BMU"}
	submitter := &fakeLicenseParamSyncSubmitter{}
	starter := &paramSyncStarter{service: submitter}

	result, err := starter.SubmitDeviceOnlineFullSync(
		context.Background(),
		dev,
		"device_online:event-ups",
		"event-ups",
		"device.online",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "skipped", result.Status)
	assert.Equal(t, "UPS_UNSUPPORTED", result.ResultCode)
	assert.Zero(t, submitter.calls)
}

func TestParamSyncStarter_DeviceOnlinePreservesQueuedAutomaticBackoffResult(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "online-device"}
	requestID := uuid.New()
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		RequestID:  requestID,
		Status:     paramsync.RequestStatusQueued,
		ResultCode: paramsync.ResultCodeAutomaticBackoff,
	}}
	starter := &paramSyncStarter{
		service: submitter,
	}

	result, err := starter.SubmitDeviceOnlineFullSync(
		context.Background(),
		dev,
		"device_online:event-backoff",
		"event-backoff",
		"device.online",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, requestID, result.RequestID)
	assert.Equal(t, "queued", result.Status)
	assert.Equal(t, "AUTOMATIC_BACKOFF", result.ResultCode)
	assert.Zero(t, result.TaskCount)
}

func TestParamSyncStarter_OMCRedeployUsesExplicitOrigin(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "redeploy-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		RequestID: uuid.New(), Status: paramsync.RequestStatusRunning,
	}}
	starter := &paramSyncStarter{service: submitter}

	_, err := starter.SubmitStartupDeviceOnlineFullSync(
		context.Background(), dev, "omc-redeploy:key", "omc-redeploy:event",
	)

	require.NoError(t, err)
	assert.Equal(t, "omc.redeploy", submitter.command.OriginEventType)
	assert.Equal(t, "omc-redeploy:key", submitter.command.IdempotencyKey)
	assert.Equal(t, "omc-redeploy:event", submitter.command.SourceEventID)
}

func TestParamSyncStarter_OMCRedeploySkipsUPS(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS/FSU2024"}
	submitter := &fakeLicenseParamSyncSubmitter{}
	starter := &paramSyncStarter{service: submitter}

	result, err := starter.SubmitStartupDeviceOnlineFullSync(
		context.Background(), dev, "omc-redeploy:key", "omc-redeploy:event",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "skipped", result.Status)
	assert.Equal(t, "UPS_UNSUPPORTED", result.ResultCode)
	assert.Zero(t, submitter.calls)
}

func TestSubmitReleaseSync_UsesCampaignAndAttemptMetadata(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "release-device"}
	campaignID := uuid.New()
	attemptID := uuid.New()
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning, TaskCount: 5,
	}}

	submitted, err := submitReleaseSync(
		context.Background(),
		submitter,
		dev,
		campaignID,
		attemptID,
	)

	require.NoError(t, err)
	assert.True(t, submitted)
	assert.Equal(t, paramsync.TriggerOMCUpgrade, submitter.command.TriggerReason)
	assert.Equal(t, paramsync.SyncScopeFull, submitter.command.Scope)
	require.NotNil(t, submitter.command.CampaignID)
	assert.Equal(t, campaignID, *submitter.command.CampaignID)
	assert.Equal(t, "omc_upgrade", submitter.command.OriginEventType)
	assert.Contains(t, submitter.command.SourceEventID, campaignID.String())
	assert.Contains(t, submitter.command.IdempotencyKey, attemptID.String())
}

func TestSubmitReleaseSync_SkipsUPS(t *testing.T) {
	dev := &model.Device{ID: uuid.New(), SerialNumber: "ups-device", ProductClass: "UPS_M3_BMU"}
	submitter := &fakeLicenseParamSyncSubmitter{}

	submitted, err := submitReleaseSync(context.Background(), submitter, dev, uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.False(t, submitted)
	assert.Zero(t, submitter.calls)
}

func TestParamSyncStarter_ReleaseUsesDurableSync(t *testing.T) {
	dev := &model.Device{ID: uuid.Nil, SerialNumber: "release-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning,
	}}
	starter := &paramSyncStarter{
		service: submitter,
	}

	submitted, err := starter.StartReleaseSync(
		context.Background(),
		dev,
		uuid.New(),
		uuid.New(),
	)

	require.NoError(t, err)
	assert.True(t, submitted)
	assert.Equal(t, paramsync.TriggerOMCUpgrade, submitter.command.TriggerReason)
}
