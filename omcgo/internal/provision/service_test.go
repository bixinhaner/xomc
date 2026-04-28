package provision

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// stubEngine satisfies engineHandle for tests without spinning up the full
// ProvisioningEngine.
type stubEngine struct {
	bootstrapFn func(ctx context.Context, evt bootstrapEvent) error
	rpcResultFn func(ctx context.Context, sn, method string, ok bool, msg string) error

	bootstrapCalls []bootstrapEvent
	rpcCalls       int
}

func (s *stubEngine) HandleBootstrap(ctx context.Context, evt bootstrapEvent) error {
	s.bootstrapCalls = append(s.bootstrapCalls, evt)
	if s.bootstrapFn != nil {
		return s.bootstrapFn(ctx, evt)
	}
	return nil
}

func (s *stubEngine) HandleRPCResult(ctx context.Context, sn, method string, ok bool, msg string) error {
	s.rpcCalls++
	if s.rpcResultFn != nil {
		return s.rpcResultFn(ctx, sn, method, ok, msg)
	}
	return nil
}

// stubDeviceLookup satisfies deviceLookup for tests.
type stubDeviceLookup struct {
	devicesByID map[uuid.UUID]*deviceForProvision
	devicesBySN map[string]*deviceForProvision
	getByIDErr  error
	getBySNErr  error
}

func (s *stubDeviceLookup) GetDevice(ctx context.Context, id uuid.UUID) (*deviceForProvision, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	d, ok := s.devicesByID[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return d, nil
}

func (s *stubDeviceLookup) GetBySerialNumber(ctx context.Context, sn string) (*deviceForProvision, error) {
	if s.getBySNErr != nil {
		return nil, s.getBySNErr
	}
	d, ok := s.devicesBySN[sn]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return d, nil
}

// helpers ---------------------------------------------------------------------

func makeDevice(t *testing.T, sn string) *deviceForProvision {
	t.Helper()
	return &deviceForProvision{
		ID:           uuid.New(),
		SerialNumber: sn,
		OUI:          "001122",
		ProductClass: "TestClass",
		Carrier:      "cmcc",
		Technology:   "lte",
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestService_DiscoverDevice(t *testing.T) {
	dev := makeDevice(t, "SN-001")
	devs := &stubDeviceLookup{
		devicesBySN: map[string]*deviceForProvision{"SN-001": dev},
	}
	svc := newServiceWithMocks(nil, nil, devs)

	t.Run("found", func(t *testing.T) {
		got, err := svc.DiscoverDevice(context.Background(), "SN-001")
		require.NoError(t, err)
		assert.Equal(t, dev.ID, got.ID)
		assert.Equal(t, "cmcc", got.Carrier)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.DiscoverDevice(context.Background(), "missing")
		assert.ErrorIs(t, err, ErrDeviceNotFound)
	})

	t.Run("empty serial", func(t *testing.T) {
		_, err := svc.DiscoverDevice(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("device lookup not configured", func(t *testing.T) {
		empty := newServiceWithMocks(nil, nil, nil)
		_, err := empty.DiscoverDevice(context.Background(), "SN-001")
		assert.Error(t, err)
	})
}

func TestService_TriggerProvisioning(t *testing.T) {
	dev := makeDevice(t, "SN-002")
	devs := &stubDeviceLookup{
		devicesByID: map[uuid.UUID]*deviceForProvision{dev.ID: dev},
	}
	eng := &stubEngine{}
	svc := newServiceWithMocks(eng, nil, devs)

	err := svc.TriggerProvisioning(context.Background(), dev.ID)
	require.NoError(t, err)
	require.Len(t, eng.bootstrapCalls, 1)
	assert.Equal(t, dev.SerialNumber, eng.bootstrapCalls[0].SerialNumber)
	assert.Equal(t, dev.OUI, eng.bootstrapCalls[0].OUI)
	assert.Equal(t, dev.Carrier, eng.bootstrapCalls[0].Carrier)
	assert.Equal(t, dev.Technology, eng.bootstrapCalls[0].Technology)
}

func TestService_TriggerProvisioning_DeviceMissing(t *testing.T) {
	devs := &stubDeviceLookup{}
	eng := &stubEngine{}
	svc := newServiceWithMocks(eng, nil, devs)

	err := svc.TriggerProvisioning(context.Background(), uuid.New())
	require.Error(t, err)
	// engine must not have been called when device lookup failed
	assert.Empty(t, eng.bootstrapCalls)
}

func TestService_TriggerProvisioning_EngineNotConfigured(t *testing.T) {
	dev := makeDevice(t, "SN-003")
	devs := &stubDeviceLookup{
		devicesByID: map[uuid.UUID]*deviceForProvision{dev.ID: dev},
	}
	svc := newServiceWithMocks(nil, nil, devs)

	err := svc.TriggerProvisioning(context.Background(), dev.ID)
	require.Error(t, err)
}

func TestService_GetTaskState(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()
	repo := &mockTaskRepo{
		GetByDeviceIDFn: func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
			if id == deviceID {
				return &ProvisioningTask{ID: taskID, DeviceID: deviceID, Status: StateConfiguring}, nil
			}
			return nil, nil
		},
	}
	svc := newServiceWithMocks(nil, repo, nil)

	t.Run("returns current state", func(t *testing.T) {
		state, err := svc.GetTaskState(context.Background(), deviceID)
		require.NoError(t, err)
		assert.Equal(t, StateConfiguring, state)
	})

	t.Run("task not found", func(t *testing.T) {
		_, err := svc.GetTaskState(context.Background(), uuid.New())
		assert.ErrorIs(t, err, ErrTaskNotFound)
	})
}

func TestService_GetTask(t *testing.T) {
	taskID := uuid.New()
	repo := &mockTaskRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
			if id == taskID {
				return &ProvisioningTask{ID: taskID, Status: StateCompleted}, nil
			}
			return nil, nil
		},
	}
	svc := newServiceWithMocks(nil, repo, nil)

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetTask(context.Background(), taskID)
		require.NoError(t, err)
		assert.Equal(t, taskID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetTask(context.Background(), uuid.New())
		assert.ErrorIs(t, err, ErrTaskNotFound)
	})
}

func TestService_ListTasks(t *testing.T) {
	expected := []ProvisioningTask{
		{ID: uuid.New(), Status: StateCompleted},
		{ID: uuid.New(), Status: StateFailed},
	}
	repo := &mockTaskRepo{
		ListFn: func(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
			return expected, int64(len(expected)), nil
		},
	}
	svc := newServiceWithMocks(nil, repo, nil)

	tasks, total, err := svc.ListTasks(context.Background(), ProvisioningTaskFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, tasks, 2)
}

func TestService_RetryFailedTask(t *testing.T) {
	deviceID := uuid.New()
	failedID := uuid.New()
	completedID := uuid.New()

	repo := &mockTaskRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
			switch id {
			case failedID:
				return &ProvisioningTask{ID: failedID, DeviceID: deviceID, Status: StateFailed}, nil
			case completedID:
				return &ProvisioningTask{ID: completedID, DeviceID: deviceID, Status: StateCompleted}, nil
			}
			return nil, nil
		},
		CreateFn: func(ctx context.Context, task *ProvisioningTask) error {
			return nil
		},
	}
	svc := newServiceWithMocks(nil, repo, nil)

	t.Run("retry creates fresh task", func(t *testing.T) {
		newTask, err := svc.RetryFailedTask(context.Background(), failedID)
		require.NoError(t, err)
		require.NotNil(t, newTask)
		assert.Equal(t, deviceID, newTask.DeviceID)
		assert.Equal(t, StateDiscovered, newTask.Status)
		assert.NotEqual(t, failedID, newTask.ID, "new task must have its own ID")
	})

	t.Run("non-failed task is not retryable", func(t *testing.T) {
		_, err := svc.RetryFailedTask(context.Background(), completedID)
		assert.ErrorIs(t, err, ErrTaskNotRetryable)
	})

	t.Run("missing task", func(t *testing.T) {
		_, err := svc.RetryFailedTask(context.Background(), uuid.New())
		assert.ErrorIs(t, err, ErrTaskNotFound)
	})
}

func TestService_RetryFailedTask_CreateError(t *testing.T) {
	failedID := uuid.New()
	deviceID := uuid.New()
	createErr := errors.New("db down")
	repo := &mockTaskRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
			return &ProvisioningTask{ID: failedID, DeviceID: deviceID, Status: StateFailed}, nil
		},
		CreateFn: func(ctx context.Context, task *ProvisioningTask) error {
			return createErr
		},
	}
	svc := newServiceWithMocks(nil, repo, nil)

	_, err := svc.RetryFailedTask(context.Background(), failedID)
	require.Error(t, err)
	assert.ErrorIs(t, err, createErr)
}

func TestService_HandleRPCResult(t *testing.T) {
	eng := &stubEngine{}
	svc := newServiceWithMocks(eng, nil, nil)

	err := svc.HandleRPCResult(context.Background(), "SN-007", MethodSetParameterValues, true, "")
	require.NoError(t, err)
	assert.Equal(t, 1, eng.rpcCalls)
}

func TestService_StateMachineHelpers(t *testing.T) {
	svc := newServiceWithMocks(nil, nil, nil)

	t.Run("IsTerminalState", func(t *testing.T) {
		assert.True(t, svc.IsTerminalState(StateCompleted))
		assert.True(t, svc.IsTerminalState(StateFailed))
		assert.False(t, svc.IsTerminalState(StateConfiguring))
	})

	t.Run("ValidateStateTransition allowed", func(t *testing.T) {
		require.NoError(t, svc.ValidateStateTransition(StateDiscovered, StateIdentifying))
	})

	t.Run("ValidateStateTransition rejected", func(t *testing.T) {
		err := svc.ValidateStateTransition(StateCompleted, StateConfiguring)
		assert.Error(t, err)
	})
}

func TestNewService_NilDependenciesAreSafe(t *testing.T) {
	// Passing nil engine + nil repo + nil device service is a valid
	// construction path used by partially-wired tests / boot phases. The
	// resulting Service should error gracefully on operations that need
	// the missing collaborator, without panicking.
	svc := NewService(nil, nil, nil)
	require.NotNil(t, svc)

	_, err := svc.DiscoverDevice(context.Background(), "SN-X")
	assert.Error(t, err)

	err = svc.TriggerProvisioning(context.Background(), uuid.New())
	assert.Error(t, err)

	_, err = svc.GetTask(context.Background(), uuid.New())
	assert.Error(t, err)
}
