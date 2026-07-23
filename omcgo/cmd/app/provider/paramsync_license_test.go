package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/paramsync"
)

type fakeLicenseParamSyncSubmitter struct {
	command paramsync.SubmitCommand
	result  *paramsync.SubmitResult
	err     error
}

func (f *fakeLicenseParamSyncSubmitter) Submit(_ context.Context, command paramsync.SubmitCommand) (*paramsync.SubmitResult, error) {
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

func TestParamSyncStarter_RegisteredDeviceBypassesDeviceCanary(t *testing.T) {
	dev := &model.Device{ID: uuid.Nil, SerialNumber: "registered-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning,
	}}
	flags := paramsync.FeatureFlags{
		RunEnabled:            true,
		ResultConsumerEnabled: true,
		StagingEnabled:        true,
		CanaryPercent:         1,
	}
	require.NoError(t, flags.Validate())
	require.False(t, flags.EnabledForDevice(dev.ID.String()))
	starter := &paramSyncStarter{
		service: submitter,
		flags:   flags,
	}

	err := starter.StartRegisteredDeviceSync(
		context.Background(),
		dev,
		"device_registered:"+dev.ID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, paramsync.TriggerDeviceRegistered, submitter.command.TriggerReason)
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

func TestParamSyncStarter_StartReleaseSync_DisabledSkips(t *testing.T) {
	starter := &paramSyncStarter{flags: paramsync.FeatureFlags{RunEnabled: false}}

	submitted, err := starter.StartReleaseSync(
		context.Background(),
		&model.Device{ID: uuid.New(), SerialNumber: "release-device"},
		uuid.New(),
		uuid.New(),
	)

	require.NoError(t, err)
	assert.False(t, submitted)
}

func TestParamSyncStarter_ReleaseBypassesDeviceCanary(t *testing.T) {
	dev := &model.Device{ID: uuid.Nil, SerialNumber: "release-device"}
	submitter := &fakeLicenseParamSyncSubmitter{result: &paramsync.SubmitResult{
		Status: paramsync.RequestStatusRunning,
	}}
	flags := paramsync.FeatureFlags{
		RunEnabled:            true,
		ResultConsumerEnabled: true,
		StagingEnabled:        true,
		CanaryPercent:         1,
	}
	require.NoError(t, flags.Validate())
	require.False(t, flags.EnabledForDevice(dev.ID.String()))
	starter := &paramSyncStarter{
		service: submitter,
		flags:   flags,
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

func TestParamSyncStarter_StartDurableSync_PeriodicUsesLegacyFallbackWhenDisabled(t *testing.T) {
	starter := &paramSyncStarter{flags: paramsync.FeatureFlags{RunEnabled: false}}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "periodic-device"}

	handled, taskCount, err := starter.StartDurableSync(context.Background(), dev, "", string(paramsync.TriggerPeriodic), nil)

	require.NoError(t, err)
	assert.False(t, handled, "periodic sync should keep the same temporary sync-gpv fallback as other triggers")
	assert.Equal(t, 0, taskCount)
}

type fakeLegacyManualSyncStarter struct {
	used  bool
	count int
	err   error
}

func (f fakeLegacyManualSyncStarter) StartManualSync(context.Context, *model.Device, string, []string) (bool, int, error) {
	return f.used, f.count, f.err
}

func TestParamSyncStarter_StartManualSyncDetailed_UsesLegacyFallbackWhenDisabled(t *testing.T) {
	starter := &paramSyncStarter{
		flags:  paramsync.FeatureFlags{RunEnabled: false},
		legacy: fakeLegacyManualSyncStarter{used: true, count: 3},
	}
	dev := &model.Device{ID: uuid.New(), SerialNumber: "manual-device"}

	result, err := starter.StartManualSyncDetailed(context.Background(), dev, uuid.NewString(), nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, &device.ManualParamSyncStart{Used: true, TaskCount: 3, Status: "queued"}, result)
}
