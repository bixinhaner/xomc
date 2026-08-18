package deviceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/require"
)

type taskGuardCarrierStub struct {
	carrier string
}

type securityActionAuthorizerStub struct {
	called bool
}

func (s *securityActionAuthorizerStub) AuthorizeSecurityAction(
	context.Context, task.TaskAdmissionRequest,
) (bool, string, error) {
	s.called = true
	return true, "automatic_security_action_readback", nil
}

func (s taskGuardCarrierStub) ResolveCarrier(context.Context, string) (string, error) {
	return s.carrier, nil
}

func TestAccessTaskGuardAllowsOnlyAcceptedNormalTasks(t *testing.T) {
	for _, test := range []struct {
		name      string
		state     AccessState
		frozen    bool
		class     task.AdmissionClass
		wantAllow bool
	}{
		{name: "accepted normal", state: AccessStateAccepted, class: task.AdmissionClassNormal, wantAllow: true},
		{name: "review normal", state: AccessStateReviewRequired, class: task.AdmissionClassNormal, wantAllow: false},
		{name: "accepted frozen normal", state: AccessStateAccepted, frozen: true, class: task.AdmissionClassNormal, wantAllow: false},
		{name: "review access probe", state: AccessStateReviewRequired, class: task.AdmissionClassAccessProbe, wantAllow: true},
		{name: "unclassified security action stays blocked", state: AccessStateAccepted, class: task.AdmissionClassSecurityAction, wantAllow: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &accessGateRepositoryStub{context: EvaluationContext{
				State: &AccessStateProjection{State: test.state, NormalTasksFrozen: test.frozen},
			}}
			guard := NewAccessTaskGuard(repository, taskGuardCarrierStub{carrier: "cmcc"})

			request := task.TaskAdmissionRequest{
				DeviceSN:       "SN-1",
				AdmissionClass: test.class,
				Source:         task.TaskSourceAPI,
				Method:         "Reboot",
			}
			if test.class == task.AdmissionClassAccessProbe {
				params, err := json.Marshal(gpsProbeTaskParams{
					Names:           []string{"Device.FAP.GPS.LockedLatitude", "Device.FAP.GPS.LockedLongitude"},
					Carrier:         "cmcc",
					SerialNumber:    "SN-1",
					EvidenceVersion: 1,
					Paths: []GPSParameterPath{
						{StandardPath: gpsLatitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLatitude"},
						{StandardPath: gpsLongitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLongitude"},
					},
					NeedGPS: true,
				})
				require.NoError(t, err)
				request.Source = task.TaskSourceDeviceAccess
				request.Method = "GetParameterValues"
				request.Params = params
			}

			allowed, _, err := guard.Allow(context.Background(), request)

			require.NoError(t, err)
			require.Equal(t, test.wantAllow, allowed)
		})
	}
}

func TestAccessTaskGuardBusinessSwitchOffAllowsNormalTasksAndBlocksWorkflowTasks(t *testing.T) {
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{}, taskGuardCarrierStub{carrier: "cmcc"})
	guard.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-OFF", AdmissionClass: task.AdmissionClassNormal, Method: "Reboot",
	})
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "access_control_disabled", reason)

	for _, class := range []task.AdmissionClass{task.AdmissionClassAccessProbe, task.AdmissionClassSecurityAction} {
		allowed, reason, err = guard.Allow(context.Background(), task.TaskAdmissionRequest{
			DeviceSN: "SN-OFF", AdmissionClass: class,
		})
		require.NoError(t, err)
		require.False(t, allowed)
		require.Equal(t, "access_control_disabled", reason)
	}
}

func TestAccessTaskGuardBusinessSwitchOffAllowsOnlyAuthorizedReadbackToFinish(t *testing.T) {
	authorizer := &securityActionAuthorizerStub{}
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{}, taskGuardCarrierStub{carrier: "cmcc"})
	guard.SetRuntimeSettingsReader(&runtimeSettingsStub{enabled: false})
	guard.SetSecurityActionAuthorizer(authorizer)

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-OFF", AdmissionClass: task.AdmissionClassSecurityAction,
		Method: "GetParameterValues",
	})

	require.NoError(t, err)
	require.True(t, allowed)
	require.True(t, authorizer.called)
	require.Equal(t, "automatic_security_action_readback", reason)
}

func TestAccessTaskGuardUsesSnapshotForNormalTasks(t *testing.T) {
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateAccepted},
	}}, taskGuardCarrierStub{carrier: "cmcc"})
	guard.SetAccessSnapshotReader(func(context.Context, string) (string, bool, bool, error) {
		return string(AccessStateAccepted), false, true, nil
	})

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-SNAPSHOT", AdmissionClass: task.AdmissionClassNormal, Method: "Reboot",
	})
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "accepted_durable_verified", reason)
}

func TestAccessTaskGuardFallsBackToDurableStateWhenSnapshotMissingOrUnavailable(t *testing.T) {
	for _, test := range []struct {
		name   string
		reader AccessSnapshotStateReader
	}{
		{
			name: "missing",
			reader: func(context.Context, string) (string, bool, bool, error) {
				return "", false, false, nil
			},
		},
		{
			name: "unavailable",
			reader: func(context.Context, string) (string, bool, bool, error) {
				return "", false, false, errors.New("redis unavailable")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			guard := NewAccessTaskGuard(&accessGateRepositoryStub{context: EvaluationContext{
				State: &AccessStateProjection{State: AccessStateAccepted},
			}}, taskGuardCarrierStub{carrier: "cmcc"})
			guard.SetAccessSnapshotReader(test.reader)
			allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
				DeviceSN: "SN-SNAPSHOT", AdmissionClass: task.AdmissionClassNormal, Method: "Reboot",
			})
			require.NoError(t, err)
			require.True(t, allowed)
			require.Equal(t, "accepted_durable_verified", reason)
		})
	}
}

func TestAccessTaskGuardDoesNotTrustStaleAcceptedSnapshot(t *testing.T) {
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateRejected, NormalTasksFrozen: true},
	}}, taskGuardCarrierStub{carrier: "cmcc"})
	guard.SetAccessSnapshotReader(func(context.Context, string) (string, bool, bool, error) {
		return string(AccessStateAccepted), false, true, nil
	})

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-STALE", AdmissionClass: task.AdmissionClassNormal, Method: "Reboot",
	})

	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "normal_tasks_frozen_by_access_state", reason)
}

func TestAccessTaskGuardFallsBackFailClosedWhenDurableStateIsNotAccepted(t *testing.T) {
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateReviewRequired, NormalTasksFrozen: true},
	}}, taskGuardCarrierStub{carrier: "cmcc"})
	guard.SetAccessSnapshotReader(func(context.Context, string) (string, bool, bool, error) {
		return "", false, false, nil
	})

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-SNAPSHOT", AdmissionClass: task.AdmissionClassNormal, Method: "Reboot",
	})

	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "normal_tasks_frozen_by_access_state", reason)
}

func TestAccessTaskGuardRejectsForgedAccessProbe(t *testing.T) {
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateReviewRequired, NormalTasksFrozen: true},
	}}
	guard := NewAccessTaskGuard(repository, taskGuardCarrierStub{carrier: "cmcc"})

	for _, request := range []task.TaskAdmissionRequest{
		{
			DeviceSN:       "SN-1",
			AdmissionClass: task.AdmissionClassAccessProbe,
			Source:         task.TaskSourceAPI,
			Method:         "Reboot",
		},
		{
			DeviceSN:       "SN-1",
			AdmissionClass: task.AdmissionClassAccessProbe,
			Source:         task.TaskSourceDeviceAccess,
			Method:         "SetParameterValues",
			Params:         json.RawMessage(`{"names":["Device.ManagementServer.Password"]}`),
		},
	} {
		allowed, reason, err := guard.Allow(context.Background(), request)

		require.Error(t, err)
		require.False(t, allowed)
		require.Equal(t, "invalid_access_probe_contract", reason)
	}
}

func TestAccessTaskGuardRejectsAccessProbeForDifferentCarrier(t *testing.T) {
	guard := NewAccessTaskGuard(&accessGateRepositoryStub{}, taskGuardCarrierStub{carrier: "cmcc"})
	params, err := json.Marshal(gpsProbeTaskParams{
		Names:           []string{"Device.FAP.GPS.LockedLatitude", "Device.FAP.GPS.LockedLongitude"},
		Carrier:         "ctcc",
		SerialNumber:    "SN-1",
		EvidenceVersion: 1,
		Paths: []GPSParameterPath{
			{StandardPath: gpsLatitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLatitude"},
			{StandardPath: gpsLongitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLongitude"},
		},
		NeedGPS: true,
	})
	require.NoError(t, err)

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-1", AdmissionClass: task.AdmissionClassAccessProbe,
		Source: task.TaskSourceDeviceAccess, Method: "GetParameterValues", Params: params,
	})

	require.ErrorContains(t, err, "carrier does not match")
	require.False(t, allowed)
	require.Equal(t, "invalid_access_probe_contract", reason)
}

func TestAccessTaskGuardAllowsBoundedServingCellDiscovery(t *testing.T) {
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateReviewRequired, NormalTasksFrozen: true},
	}}
	guard := NewAccessTaskGuard(repository, taskGuardCarrierStub{carrier: "cmcc"})
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-1", EvidenceVersion: 1,
		NeedTAC: true, ProductClass: "FAP/BU1810",
		DiscoveryPath: servingCellStandardRoot, DiscoveryStandardPath: servingCellStandardRoot,
		NextLevel: true,
	})
	require.NoError(t, err)

	allowed, _, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-1", AdmissionClass: task.AdmissionClassAccessProbe,
		Source: task.TaskSourceDeviceAccess, Method: "GetParameterNames", Params: params,
	})

	require.NoError(t, err)
	require.True(t, allowed)
}

func TestAccessTaskGuardAllowsServingPLMNListECGIProbe(t *testing.T) {
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateReviewRequired, NormalTasksFrozen: true},
	}}
	guard := NewAccessTaskGuard(repository, taskGuardCarrierStub{carrier: "cmcc"})
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-1", EvidenceVersion: 1, NeedECGI: true,
		Names: []string{"serving-plmns", "eci"},
		RadioPaths: []RadioParameterPath{
			{
				StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
				PrivatePath:  "serving-plmns", FAPInstance: 1, Kind: radioPathServingPLMNs,
			},
			{
				StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
				PrivatePath:  "eci", FAPInstance: 1, Kind: radioPathECI,
			},
		},
	})
	require.NoError(t, err)

	allowed, _, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-1", AdmissionClass: task.AdmissionClassAccessProbe,
		Source: task.TaskSourceDeviceAccess, Method: "GetParameterValues", Params: params,
	})

	require.NoError(t, err)
	require.True(t, allowed)
}

func TestAccessTaskGuardRejectsServingPLMNAndECIFromDifferentCells(t *testing.T) {
	repository := &accessGateRepositoryStub{context: EvaluationContext{
		State: &AccessStateProjection{State: AccessStateReviewRequired, NormalTasksFrozen: true},
	}}
	guard := NewAccessTaskGuard(repository, taskGuardCarrierStub{carrier: "cmcc"})
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-1", EvidenceVersion: 1, NeedECGI: true,
		Names: []string{"serving-plmns", "eci"},
		RadioPaths: []RadioParameterPath{
			{
				StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
				PrivatePath:  "serving-plmns", FAPInstance: 1, Kind: radioPathServingPLMNs,
			},
			{
				StandardPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity",
				PrivatePath:  "eci", FAPInstance: 2, Kind: radioPathECI,
			},
		},
	})
	require.NoError(t, err)

	allowed, reason, err := guard.Allow(context.Background(), task.TaskAdmissionRequest{
		DeviceSN: "SN-1", AdmissionClass: task.AdmissionClassAccessProbe,
		Source: task.TaskSourceDeviceAccess, Method: "GetParameterValues", Params: params,
	})

	require.Error(t, err)
	require.False(t, allowed)
	require.Equal(t, "invalid_access_probe_contract", reason)
}
