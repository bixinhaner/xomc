package mml

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Mock Repositories ---

type mockCommandRepo struct {
	listFn      func(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error)
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
	getByCodeFn func(ctx context.Context, code string) (*MMLCommand, error)
}

func (m *mockCommandRepo) List(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockCommandRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockCommandRepo) GetByCode(ctx context.Context, code string) (*MMLCommand, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(ctx, code)
	}
	return nil, nil
}

type mockScriptRepo struct {
	createFn  func(ctx context.Context, script *MMLScript) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*MMLScript, error)
	updateFn  func(ctx context.Context, script *MMLScript) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
}

func (m *mockScriptRepo) Create(ctx context.Context, script *MMLScript) error {
	if m.createFn != nil {
		return m.createFn(ctx, script)
	}
	return nil
}

func (m *mockScriptRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockScriptRepo) Update(ctx context.Context, script *MMLScript) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, script)
	}
	return nil
}

func (m *mockScriptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockScriptRepo) List(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

type mockTaskRepo struct {
	createFn  func(ctx context.Context, task *MMLTask) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*MMLTask, error)
	updateFn  func(ctx context.Context, task *MMLTask) error
	listFn    func(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
}

func (m *mockTaskRepo) Create(ctx context.Context, task *MMLTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepo) Update(ctx context.Context, task *MMLTask) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// --- Helper ---

func newTestService(cmdRepo *mockCommandRepo, scriptRepo *mockScriptRepo, taskRepo *mockTaskRepo) *Service {
	return NewService(cmdRepo, scriptRepo, taskRepo, zap.NewNop())
}

// --- Tests: ListCommands ---

func TestService_ListCommands(t *testing.T) {
	expected := &model.ListResponse[MMLCommand]{
		Items:      []MMLCommand{{ID: uuid.New(), CommandCode: "GET_PARAM", CommandName: "Get Parameters"}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	filter := CommandFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	}

	cmdRepo := &mockCommandRepo{
		listFn: func(ctx context.Context, f CommandFilter) (*model.ListResponse[MMLCommand], error) {
			assert.Equal(t, filter, f)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.ListCommands(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: GetCommand ---

func TestService_GetCommand(t *testing.T) {
	cmdID := uuid.New()
	expected := &MMLCommand{
		ID:          cmdID,
		CommandCode: "GET_PARAM",
		CommandName: "Get Parameters",
		RPCMethod:   "GetParameterValues",
	}

	cmdRepo := &mockCommandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommand(context.Background(), cmdID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: GetCommandByCode ---

func TestService_GetCommandByCode(t *testing.T) {
	expected := &MMLCommand{
		ID:          uuid.New(),
		CommandCode: "SET_PARAM",
		CommandName: "Set Parameters",
		RPCMethod:   "SetParameterValues",
	}

	cmdRepo := &mockCommandRepo{
		getByCodeFn: func(ctx context.Context, code string) (*MMLCommand, error) {
			assert.Equal(t, "SET_PARAM", code)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommandByCode(context.Background(), "SET_PARAM")

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: CreateScript ---

func TestService_CreateScript(t *testing.T) {
	var captured *MMLScript
	scriptRepo := &mockScriptRepo{
		createFn: func(ctx context.Context, script *MMLScript) error {
			captured = script
			script.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, scriptRepo, &mockTaskRepo{})

	script := &MMLScript{
		ScriptName:  "Test Script",
		Description: "A test script",
		Content:     "GET_PARAM Device.Info",
		DeviceType:  "femto",
		Creator:     "admin",
		Tags:        nil, // should default to []
	}

	result, err := svc.CreateScript(context.Background(), script)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, captured.Tags)
	assert.Empty(t, captured.Tags, "nil tags should default to empty slice")
}

func TestService_CreateScript_WithTags(t *testing.T) {
	scriptRepo := &mockScriptRepo{
		createFn: func(ctx context.Context, script *MMLScript) error {
			script.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, scriptRepo, &mockTaskRepo{})

	tags := []string{"lte", "config"}
	script := &MMLScript{
		ScriptName:  "Tagged Script",
		Description: "A script with tags",
		Content:     "SET_PARAM Device.Config",
		DeviceType:  "pico",
		Creator:     "admin",
		Tags:        tags,
	}

	result, err := svc.CreateScript(context.Background(), script)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, tags, result.Tags, "non-nil tags should be preserved")
}

// --- Tests: UpdateScript ---

func TestService_UpdateScript(t *testing.T) {
	scriptID := uuid.New()
	existing := &MMLScript{
		ID:          scriptID,
		ScriptName:  "Old Name",
		Description: "Old description",
		Content:     "OLD_COMMAND",
		DeviceType:  "femto",
		Creator:     "admin",
		Tags:        []string{"original"},
	}

	var updatedScript *MMLScript
	scriptRepo := &mockScriptRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
			assert.Equal(t, scriptID, id)
			return existing, nil
		},
		updateFn: func(ctx context.Context, s *MMLScript) error {
			updatedScript = s
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, scriptRepo, &mockTaskRepo{})

	update := &MMLScript{
		ScriptName:  "New Name",
		Description: "New description",
		Content:     "NEW_COMMAND",
		DeviceType:  "pico",
		Tags:        []string{"updated", "v2"},
	}

	result, err := svc.UpdateScript(context.Background(), scriptID, update)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", updatedScript.ScriptName)
	assert.Equal(t, "New description", updatedScript.Description)
	assert.Equal(t, "NEW_COMMAND", updatedScript.Content)
	assert.Equal(t, "pico", updatedScript.DeviceType)
	assert.Equal(t, []string{"updated", "v2"}, updatedScript.Tags)
}

func TestService_UpdateScript_NilTagsPreservesExisting(t *testing.T) {
	scriptID := uuid.New()
	existing := &MMLScript{
		ID:          scriptID,
		ScriptName:  "Old Name",
		Description: "Old description",
		Content:     "OLD_COMMAND",
		DeviceType:  "femto",
		Creator:     "admin",
		Tags:        []string{"keep-me"},
	}

	var updatedScript *MMLScript
	scriptRepo := &mockScriptRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, s *MMLScript) error {
			updatedScript = s
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, scriptRepo, &mockTaskRepo{})

	update := &MMLScript{
		ScriptName:  "Updated Name",
		Description: "Updated description",
		Content:     "UPDATED_CMD",
		DeviceType:  "micro",
		Tags:        nil, // should preserve existing
	}

	result, err := svc.UpdateScript(context.Background(), scriptID, update)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Name", updatedScript.ScriptName)
	assert.Equal(t, []string{"keep-me"}, updatedScript.Tags, "nil tags should preserve existing")
}

// --- Tests: DeleteScript ---

func TestService_DeleteScript(t *testing.T) {
	scriptID := uuid.New()
	var deletedID uuid.UUID

	scriptRepo := &mockScriptRepo{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, scriptRepo, &mockTaskRepo{})
	err := svc.DeleteScript(context.Background(), scriptID)

	require.NoError(t, err)
	assert.Equal(t, scriptID, deletedID)
}

// --- Tests: ExecuteCommand ---

func TestService_ExecuteCommand_WithCode(t *testing.T) {
	resolvedCmd := &MMLCommand{
		ID:          uuid.New(),
		CommandCode: "GET_PARAM",
		RPCMethod:   "GetParameterValues",
	}

	cmdRepo := &mockCommandRepo{
		getByCodeFn: func(ctx context.Context, code string) (*MMLCommand, error) {
			assert.Equal(t, "GET_PARAM", code)
			return resolvedCmd, nil
		},
	}

	var capturedTask *MMLTask
	taskRepo := &mockTaskRepo{
		createFn: func(ctx context.Context, task *MMLTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, taskRepo)

	req := ExecuteRequest{
		CommandCode: "GET_PARAM",
		DeviceSNs:   []string{"SN-001", "SN-002"},
		Parameters:  map[string]interface{}{"path": "Device.Info"},
		TaskName:    "Query device info",
		Creator:     "admin",
	}

	result, err := svc.ExecuteCommand(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, capturedTask)
	assert.Equal(t, TaskPending, capturedTask.Status)
	assert.Equal(t, "Query device info", capturedTask.TaskName)
	assert.Equal(t, []string{"SN-001", "SN-002"}, capturedTask.DeviceSNs)
	assert.Equal(t, "admin", capturedTask.Creator)
	assert.Empty(t, capturedTask.Results)

	// Verify the resolved command was appended to the commands list
	require.Len(t, capturedTask.Commands, 1)
	entry := capturedTask.Commands[0]
	assert.Equal(t, "GET_PARAM", entry["command_code"])
	assert.Equal(t, "GetParameterValues", entry["rpc_method"])
	assert.Equal(t, map[string]interface{}{"path": "Device.Info"}, entry["parameters"])
}

func TestService_ExecuteCommand_WithoutCode(t *testing.T) {
	rawCommands := []map[string]interface{}{
		{"command_code": "RAW_CMD_1", "rpc_method": "Reboot"},
		{"command_code": "RAW_CMD_2", "rpc_method": "FactoryReset"},
	}

	var capturedTask *MMLTask
	taskRepo := &mockTaskRepo{
		createFn: func(ctx context.Context, task *MMLTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)

	req := ExecuteRequest{
		// CommandCode intentionally empty
		DeviceSNs: []string{"SN-003"},
		TaskName:  "Raw command execution",
		Creator:   "operator",
		Commands:  rawCommands,
	}

	result, err := svc.ExecuteCommand(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, capturedTask)
	assert.Equal(t, TaskPending, capturedTask.Status)
	assert.Equal(t, "Raw command execution", capturedTask.TaskName)
	assert.Len(t, capturedTask.Commands, 2, "should use raw commands as-is")
	assert.Equal(t, "RAW_CMD_1", capturedTask.Commands[0]["command_code"])
	assert.Equal(t, "RAW_CMD_2", capturedTask.Commands[1]["command_code"])
}

func TestService_ExecuteCommand_NilCommandsDefaultsToEmpty(t *testing.T) {
	var capturedTask *MMLTask
	taskRepo := &mockTaskRepo{
		createFn: func(ctx context.Context, task *MMLTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)

	req := ExecuteRequest{
		// CommandCode intentionally empty
		// Commands intentionally nil
		DeviceSNs: []string{"SN-004"},
		TaskName:  "Empty task",
		Creator:   "admin",
	}

	result, err := svc.ExecuteCommand(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, capturedTask)
	assert.NotNil(t, capturedTask.Commands)
	assert.Empty(t, capturedTask.Commands, "nil commands should default to empty slice")
}
