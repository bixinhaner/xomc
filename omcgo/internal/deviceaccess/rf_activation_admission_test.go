package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type ownedIsolationStub struct{ action *Action }

func (s ownedIsolationStub) FindOwnedIsolation(context.Context, uuid.UUID) (*Action, error) {
	return s.action, nil
}

func TestRFActivationAdmissionRequiresAcceptedStateWithoutOwnedIsolation(t *testing.T) {
	deviceID := uuid.New()
	repository := &actionRepositoryFake{evaluation: EvaluationContext{State: &AccessStateProjection{
		DeviceID: &deviceID, State: AccessStateRejected, NormalTasksFrozen: true,
	}}}
	reader := NewRFActivationAdmissionReader(
		repository, ownedIsolationStub{}, &runtimeSettingsStub{enabled: true},
	)
	allowed, reason, err := reader.AllowRFActivation(context.Background(), "cmcc", "SN-1", deviceID)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "device_access_not_accepted", reason)

	repository.evaluation.State.State = AccessStateAccepted
	repository.evaluation.State.NormalTasksFrozen = false
	reader.actions = ownedIsolationStub{action: &Action{ID: uuid.New(), OwnedRFChange: true}}
	allowed, reason, err = reader.AllowRFActivation(context.Background(), "cmcc", "SN-1", deviceID)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, "device_access_owned_isolation", reason)

	reader.actions = ownedIsolationStub{}
	allowed, reason, err = reader.AllowRFActivation(context.Background(), "cmcc", "SN-1", deviceID)
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "device_access_accepted", reason)
}

func TestRFActivationAdmissionPreservesDisabledCompatibility(t *testing.T) {
	reader := NewRFActivationAdmissionReader(
		&actionRepositoryFake{}, ownedIsolationStub{}, &runtimeSettingsStub{enabled: false},
	)
	allowed, reason, err := reader.AllowRFActivation(context.Background(), "cmcc", "SN-1", uuid.New())
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "device_access_disabled_bypass", reason)
}
