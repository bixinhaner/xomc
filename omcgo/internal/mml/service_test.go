package mml

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
)

// stubDeviceTaskCreator captures fan-out invocations for assertion.
type stubDeviceTaskCreator struct {
	calls [][]*task.CreateTaskRequest
}

func (s *stubDeviceTaskCreator) BatchCreateTasks(_ context.Context, reqs []*task.CreateTaskRequest) ([]*task.Task, error) {
	s.calls = append(s.calls, reqs)
	out := make([]*task.Task, len(reqs))
	for i := range reqs {
		out[i] = &task.Task{ID: uuid.New().String()}
	}
	return out, nil
}

// stubCmdParamRepo returns canned MMLParamRef sets keyed by command ID.
// Used to provide param_refs to Fanouter without touching the real DB.
type stubCmdParamRepo struct {
	refs map[uuid.UUID][]MMLParamRef
}

func (s *stubCmdParamRepo) ListByCommandID(_ context.Context, id uuid.UUID) ([]MMLParamRef, error) {
	return s.refs[id], nil
}

func (s *stubCmdParamRepo) ListByCommandIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID][]MMLParamRef, error) {
	out := make(map[uuid.UUID][]MMLParamRef, len(ids))
	for _, id := range ids {
		if r, ok := s.refs[id]; ok {
			out[id] = r
		}
	}
	return out, nil
}

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
	createFn          func(ctx context.Context, script *MMLScript) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*MMLScript, error)
	updateFn          func(ctx context.Context, script *MMLScript) error
	updateLifecycleFn func(ctx context.Context, script *MMLScript) error
	updateLastRunFn   func(ctx context.Context, id uuid.UUID, status string, at time.Time) error
	deleteFn          func(ctx context.Context, id uuid.UUID) error
	listFn            func(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
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

func (m *mockScriptRepo) UpdateLifecycle(ctx context.Context, script *MMLScript) error {
	if m.updateLifecycleFn != nil {
		return m.updateLifecycleFn(ctx, script)
	}
	return nil
}

func (m *mockScriptRepo) UpdateLastRun(ctx context.Context, id uuid.UUID, status string, at time.Time) error {
	if m.updateLastRunFn != nil {
		return m.updateLastRunFn(ctx, id, status, at)
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
	createFn        func(ctx context.Context, task *MMLTask) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*MMLTask, error)
	updateFn        func(ctx context.Context, task *MMLTask) error
	updateStatusFn  func(ctx context.Context, id uuid.UUID, status TaskStatus) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	listFn          func(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
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

func (m *mockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTaskRepo) IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error {
	return nil
}

func (m *mockTaskRepo) ListByScriptID(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error) {
	return model.NewListResponse([]MMLTask{}, 0, req.Page, req.PageSize), nil
}

type mockCustomCommandRepo struct {
	createFn  func(ctx context.Context, tmpl *MMLCustomCommand) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error)
	updateFn  func(ctx context.Context, tmpl *MMLCustomCommand) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error)
}

func (m *mockCustomCommandRepo) Create(ctx context.Context, tmpl *MMLCustomCommand) error {
	if m.createFn != nil {
		return m.createFn(ctx, tmpl)
	}
	return nil
}

func (m *mockCustomCommandRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockCustomCommandRepo) Update(ctx context.Context, tmpl *MMLCustomCommand) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tmpl)
	}
	return nil
}

func (m *mockCustomCommandRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockCustomCommandRepo) List(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// --- Helper ---

func newTestService(cmdRepo *mockCommandRepo, scriptRepo *mockScriptRepo, taskRepo *mockTaskRepo) *Service {
	return NewService(cmdRepo, scriptRepo, taskRepo, &mockCustomCommandRepo{}, nil, zap.NewNop())
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

func TestService_GetCommandParamPaths_StringPaths(t *testing.T) {
	cmdID := uuid.New()
	paramPathsJSON := json.RawMessage(`[
		"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF"
	]`)
	expected := &MMLCommand{
		ID:                  cmdID,
		CommandCode:         "MOD CELL",
		OperationType:       "MOD",
		ParamPaths:          paramPathsJSON,
		SupportedOperations: []string{"LST", "MOD"},
	}

	cmdRepo := &mockCommandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommandParamPaths(context.Background(), cmdID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "MOD CELL", result.CommandCode)
	assert.Equal(t, "MOD", result.OperationType)
	assert.Equal(t, []string{"LST", "MOD"}, result.SupportedOperations)
	require.Len(t, result.ParamPaths, 1)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF", result.ParamPaths[0].Path)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF", result.ParamPaths[0].Label)
	assert.True(t, result.ParamPaths[0].Writable)
}

func TestService_GetCommandParamPaths_ObjectPaths(t *testing.T) {
	cmdID := uuid.New()
	paramPathsJSON := json.RawMessage(`[
		{"path":"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF","label":"LTE射频参数","writable":true},
		{"path":"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.CellState","label":"CellState"}
	]`)
	expected := &MMLCommand{
		ID:                  cmdID,
		CommandCode:         "MOD CELL",
		OperationType:       "MOD",
		ParamPaths:          paramPathsJSON,
		SupportedOperations: []string{"LST", "MOD"},
	}

	cmdRepo := &mockCommandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommandParamPaths(context.Background(), cmdID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.ParamPaths, 2)
	assert.Equal(t, "LTE射频参数", result.ParamPaths[0].Label)
	assert.True(t, result.ParamPaths[0].Writable)
	assert.Equal(t, "CellState", result.ParamPaths[1].Label)
	assert.True(t, result.ParamPaths[1].Writable)
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
		Tags:        []string{"updated", "v2"},
	}

	result, err := svc.UpdateScript(context.Background(), scriptID, update)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", updatedScript.ScriptName)
	assert.Equal(t, "New description", updatedScript.Description)
	assert.Equal(t, "NEW_COMMAND", updatedScript.Content)
	assert.Equal(t, []string{"updated", "v2"}, updatedScript.Tags)
}

func TestService_UpdateScript_NilTagsPreservesExisting(t *testing.T) {
	scriptID := uuid.New()
	existing := &MMLScript{
		ID:          scriptID,
		ScriptName:  "Old Name",
		Description: "Old description",
		Content:     "OLD_COMMAND",
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

// --- Tests: ExecuteCommand sets TotalDevices ---

func TestService_ExecuteCommand_TotalDevicesSet(t *testing.T) {
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
		DeviceSNs: []string{"SN-001", "SN-002", "SN-003"},
		TaskName:  "Multi-device task",
		Creator:   "admin",
	}

	result, err := svc.ExecuteCommand(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, capturedTask.TotalDevices, "TotalDevices should match device count")
}

// --- Tests: ExecuteCommand defaults ExecuteType to "immediate" when empty ---
//
// 回归测试 to-do-list.md #1：前端调 /api/v1/mml/execute 不带 execute_type
// 时，handler 补默认；这里验证 service 层的兜底也生效，确保 Fanouter 被触发、
// device_tasks 正确派生、mml_tasks 状态迁移到 running。

func TestService_ExecuteCommand_EmptyExecuteType_DefaultsToImmediateAndFansOut(t *testing.T) {
	resolvedCmd := &MMLCommand{
		ID:          uuid.New(),
		CommandCode: "LST DEVICE_INFO",
		RPCMethod:   "GetParameterValues",
	}
	cmdRepo := &mockCommandRepo{
		getByCodeFn: func(_ context.Context, _ string) (*MMLCommand, error) {
			return resolvedCmd, nil
		},
	}

	var capturedTask *MMLTask
	var updatedStatus TaskStatus
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, s TaskStatus) error {
			updatedStatus = s
			return nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, taskRepo)

	// Fanouter 现在依赖 param_refs 翻译为 TR-069 wire 格式（fanout.go BuildTR069Params）。
	// LST 命令没有 refs 时 Fanouter 会跳过 command，这里挂上 stub 仓库提供最小可用 refs。
	svc.SetCmdParamRepo(&stubCmdParamRepo{
		refs: map[uuid.UUID][]MMLParamRef{
			resolvedCmd.ID: {
				{ParamCode: "DEVICE_INFO_HW", Tr069Path: "Device.DeviceInfo.HardwareVersion", ValueType: "string"},
			},
		},
	})

	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		CommandCode: "LST DEVICE_INFO",
		DeviceSNs:   []string{"SN-001", "SN-002"},
		Parameters:  map[string]interface{}{},
		TaskName:    "LST DEVICE_INFO",
		Creator:     "admin",
		// ExecuteType intentionally empty (前端默认场景)
	}

	result, err := svc.ExecuteCommand(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, capturedTask)

	assert.Equal(t, ExecuteImmediate, capturedTask.ExecuteType,
		"empty ExecuteType 必须被兜底补齐为 immediate")
	assert.Len(t, stub.calls, 1, "Fanouter 必须被调用")
	assert.Len(t, stub.calls[0], 2, "每个设备产生一条 device_task (2 devices × 1 cmd)")
	assert.Equal(t, TaskRunning, updatedStatus,
		"Fanout 成功后 mml_task 状态必须迁移到 running")
}

// 回归 MML 控制台"参数路径直接执行"需求：用户没在命令树选命令，
// 直接给 N 条 TR-069 路径 + LST/DSP，期望走通 fanout 并产出
// device_task with params={"names": [paths...]}.
func TestService_ExecuteCommand_RawParamPaths_LST(t *testing.T) {
	cmdRepo := &mockCommandRepo{}
	var capturedTask *MMLTask
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
	}
	svc := newTestService(cmdRepo, &mockScriptRepo{}, taskRepo)

	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		// CommandCode 故意留空
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.DeviceInfo.HardwareVersion", "Device.DeviceInfo.SoftwareVersion"},
		OperationType: "LST",
		Creator:       "admin",
	}

	result, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Len(t, capturedTask.Commands, 1, "应当合成 1 条 command")
	cmd := capturedTask.Commands[0]
	assert.Equal(t, "GetParameterValues", cmd["rpc_method"])
	assert.Equal(t, "LST", cmd["operation_type"])
	assert.Equal(t, "RAW LST", cmd["command_code"], "无命令时 command_code 给个可识别标记")

	// 关键：fanout 后 device_task.params 应该是 TR-069 wire 格式 {"names":[...]}
	require.Len(t, stub.calls, 1, "Fanouter 必须被调用")
	require.Len(t, stub.calls[0], 1, "1 device × 1 cmd = 1 device_task")
	dt := stub.calls[0][0]
	assert.Equal(t, "GetParameterValues", dt.Method)

	var got struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(dt.Params, &got))
	assert.Equal(t, []string{
		"Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.SoftwareVersion",
	}, got.Names)
}

func TestService_ExecuteCommand_RawParamPaths_DSP_DefaultsToLST(t *testing.T) {
	// 不传 OperationType（默认 LST），仍应走通
	cmdRepo := &mockCommandRepo{}
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
	}
	svc := newTestService(cmdRepo, &mockScriptRepo{}, taskRepo)
	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		DeviceSNs:  []string{"SN-A"},
		ParamPaths: []string{"Device.DeviceInfo.SerialNumber"},
		Creator:    "admin",
	}
	_, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 1)
}

func TestService_ExecuteCommand_RawParamPaths_MOD(t *testing.T) {
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
	}
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.DeviceInfo.X", "Device.DeviceInfo.Y"},
		ParamValues:   []string{"42", "true"},
		OperationType: "MOD",
		Creator:       "admin",
	}
	_, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 1, "1 device × 1 SetParameterValues command")
	dt := stub.calls[0][0]
	assert.Equal(t, "SetParameterValues", dt.Method)

	var got struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(dt.Params, &got))
	require.Len(t, got.Values, 2)
	pairs := map[string]string{}
	for _, v := range got.Values {
		pairs[v.Name] = v.Value
	}
	assert.Equal(t, "42", pairs["Device.DeviceInfo.X"])
	assert.Equal(t, "true", pairs["Device.DeviceInfo.Y"])
}

func TestService_ExecuteCommand_RawParamPaths_MOD_EmptyValueFails(t *testing.T) {
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})
	_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.X"},
		ParamValues:   []string{""}, // 显式空值
		OperationType: "MOD",
		Creator:       "admin",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "param_values")
}

func TestService_ExecuteCommand_RawParamPaths_ADD(t *testing.T) {
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
	}
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.WiFi.SSID."},
		OperationType: "ADD",
		Creator:       "admin",
	}
	_, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 1, "ADD 单 path → 1 个 AddObject device_task")
	dt := stub.calls[0][0]
	assert.Equal(t, "AddObject", dt.Method)
	var got struct {
		ObjectName string `json:"object_name"`
	}
	require.NoError(t, json.Unmarshal(dt.Params, &got))
	assert.True(t, len(got.ObjectName) > 0 && got.ObjectName[len(got.ObjectName)-1] == '.',
		"object_name 必须以 . 结尾，got=%q", got.ObjectName)
}

func TestService_ExecuteCommand_RawParamPaths_ADD_RejectsMultiPath(t *testing.T) {
	// AddObject 协议规定单次仅一个 object_name，多 path 在协议层无意义。
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})
	_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.WiFi.SSID.", "Device.WiFi.AccessPoint."},
		OperationType: "ADD",
		Creator:       "admin",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AddObject only accepts one object path")
}

func TestService_ExecuteCommand_RawParamPaths_RMV_RejectsMultiPath(t *testing.T) {
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})
	_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.WiFi.SSID.1.", "Device.WiFi.SSID.2."},
		OperationType: "RMV",
		Creator:       "admin",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeleteObject only accepts one object path")
}

func TestService_ExecuteCommand_RawParamPaths_RMV(t *testing.T) {
	taskRepo := &mockTaskRepo{
		createFn: func(_ context.Context, task *MMLTask) error {
			task.ID = uuid.New()
			return nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
	}
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	stub := &stubDeviceTaskCreator{}
	svc.SetFanouter(NewFanouter(stub, zap.NewNop()))

	req := ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"Device.WiFi.SSID.1."},
		OperationType: "RMV",
		Creator:       "admin",
	}
	_, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err)

	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 1)
	assert.Equal(t, "DeleteObject", stub.calls[0][0].Method)
}

func TestService_ExecuteCommand_RawParamPaths_RejectsUnsupportedOp(t *testing.T) {
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})
	for _, op := range []string{"RST", "ACT", "DEA", "CLR", "UPG"} {
		t.Run(op, func(t *testing.T) {
			_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
				DeviceSNs:     []string{"SN-A"},
				ParamPaths:    []string{"Device.X"},
				OperationType: op,
				Creator:       "admin",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported operation_type")
		})
	}
}

func TestService_ExecuteCommand_RawParamPaths_RejectsAllEmpty(t *testing.T) {
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})

	_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
		DeviceSNs:     []string{"SN-A"},
		ParamPaths:    []string{"", "  ", ""},
		OperationType: "LST",
		Creator:       "admin",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all paths empty")
}

// --- Tests: Task control methods ---

func TestService_StartTask_PendingToRunning(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskPending}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			assert.Equal(t, taskID, id)
			return existing, nil
		},
		updateStatusFn: func(ctx context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, taskID, id)
			assert.Equal(t, TaskRunning, status)
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	result, err := svc.StartTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, TaskRunning, result.Status)
}

func TestService_StartTask_RunningFails(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskRunning}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			return existing, nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	_, err := svc.StartTask(context.Background(), taskID)

	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestService_PauseTask_RunningToPaused(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskRunning}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			return existing, nil
		},
		updateStatusFn: func(ctx context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, TaskPaused, status)
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	result, err := svc.PauseTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, TaskPaused, result.Status)
}

func TestService_CancelTask(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskRunning}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			return existing, nil
		},
		updateStatusFn: func(ctx context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, TaskCancelled, status)
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	result, err := svc.CancelTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, TaskCancelled, result.Status)
}

func TestService_DeleteTask_RunningFails(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskRunning}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			return existing, nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	err := svc.DeleteTask(context.Background(), taskID)

	require.ErrorIs(t, err, ErrCannotDeleteRunning)
}

func TestService_DeleteTask_NonRunning(t *testing.T) {
	taskID := uuid.New()
	existing := &MMLTask{ID: taskID, Status: TaskCompleted}

	var deletedID uuid.UUID
	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
			return existing, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}

	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, taskRepo)
	err := svc.DeleteTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, taskID, deletedID)
}
