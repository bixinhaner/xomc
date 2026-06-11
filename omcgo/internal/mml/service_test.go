package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
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
	listFn          func(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error)
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
	getByCodeFn     func(ctx context.Context, code string) (*MMLCommand, error)
	listByGroupIDFn func(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error)
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

func (m *mockCommandRepo) ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error) {
	if m.listByGroupIDFn != nil {
		return m.listByGroupIDFn(ctx, groupID)
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

func (m *mockTaskRepo) UpdateExportAggregate(ctx context.Context, id uuid.UUID, objectKey string, t time.Time) error {
	return nil
}

func (m *mockTaskRepo) UpdateExportDevice(ctx context.Context, id uuid.UUID, deviceSN, objectKey string, t time.Time) error {
	return nil
}

type mockCustomCommandRepo struct {
	createFn       func(ctx context.Context, tmpl *MMLCustomCommand) error
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error)
	updateFn       func(ctx context.Context, tmpl *MMLCustomCommand) error
	deleteFn       func(ctx context.Context, id uuid.UUID) error
	listFn         func(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error)
	nameExistsFn   func(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
	publicNameFn   func(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error)
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

func (m *mockCustomCommandRepo) NameExistsForPrivate(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	if m.nameExistsFn != nil {
		return m.nameExistsFn(ctx, ownerID, name, excludeID)
	}
	return false, nil
}

func (m *mockCustomCommandRepo) NameExistsForPublic(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	if m.publicNameFn != nil {
		return m.publicNameFn(ctx, name, excludeID)
	}
	return false, nil
}

// mockRoleQuerier 模拟 admin.PgRoleRepository.GetUserVisibleGroupIDs；T-0090-c
// 用于 service 层 RBAC 派生 test。
type mockRoleQuerier struct {
	groupsByUser map[uuid.UUID][]uuid.UUID
	err          error
}

func (m *mockRoleQuerier) GetUserVisibleGroupIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.groupsByUser[userID], nil
}

// --- Helper ---

func newTestService(cmdRepo *mockCommandRepo, scriptRepo *mockScriptRepo, taskRepo *mockTaskRepo) *Service {
	return NewService(cmdRepo, scriptRepo, taskRepo, &mockCustomCommandRepo{}, nil, zap.NewNop())
}

// --- Tests: ExecuteGroup not-found（issue #125-mml 问题 2）---

// TestService_ExecuteGroup_NilGroupID_NotFound：全零 UUID 应翻 commonerrors.ErrNotFound
// （→ 404），且文案不再是误导性的「group_id required」。
func TestService_ExecuteGroup_NilGroupID_NotFound(t *testing.T) {
	svc := newTestService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{})

	_, err := svc.ExecuteGroup(context.Background(), ExecuteGroupRequest{
		GroupID:   uuid.Nil,
		DeviceSNs: []string{"SMK-NO-SUCH-DEV"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"nil group 应 wrap ErrNotFound（→404），got %v", err)
	assert.NotContains(t, err.Error(), "group_id required",
		"不得保留误导性的 group_id required 文案")
}

// TestService_ExecuteGroup_UnknownGroup_NotFound：随机 UUID 但 group 下无命令时,
// 同归 not-found（Service 无独立 group 仓储）→ wrap ErrNotFound（404），非裸 500。
func TestService_ExecuteGroup_UnknownGroup_NotFound(t *testing.T) {
	cmdRepo := &mockCommandRepo{
		listByGroupIDFn: func(_ context.Context, _ uuid.UUID) ([]MMLCommand, error) {
			return []MMLCommand{}, nil
		},
	}
	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})

	_, err := svc.ExecuteGroup(context.Background(), ExecuteGroupRequest{
		GroupID:   uuid.New(),
		DeviceSNs: []string{"SMK-NO-SUCH-DEV"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"unknown group 应 wrap ErrNotFound（→404），got %v", err)
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

// 000090 schema 重建后 mml_commands.target_paths 是 []string；label 由 service 兜底
// 等于 path；writable 完全由 operation_type 派生。下面两个测试覆盖 LST（只读）和
// MOD（可写）两个典型形态。
func TestService_GetCommandParamPaths_LST_ReadOnly(t *testing.T) {
	cmdID := uuid.New()
	expected := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_CELL_CONFIG",
		OperationType: "LST",
		TargetPaths: []string{
			"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF",
		},
	}

	cmdRepo := &mockCommandRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommandParamPaths(context.Background(), cmdID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "LST_CELL_CONFIG", result.CommandCode)
	assert.Equal(t, "LST", result.OperationType)
	assert.Equal(t, []string{"LST"}, result.SupportedOperations, "single op derived from operation_type")
	require.Len(t, result.ParamPaths, 1)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF", result.ParamPaths[0].Path)
	assert.Equal(t, result.ParamPaths[0].Path, result.ParamPaths[0].Label)
	assert.False(t, result.ParamPaths[0].Writable, "LST is read-only")
}

func TestService_GetCommandParamPaths_MOD_Writable_MultiPaths(t *testing.T) {
	cmdID := uuid.New()
	expected := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_CELL_CONFIG",
		OperationType: "MOD",
		TargetPaths: []string{
			"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF",
			"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.CellState",
		},
	}

	cmdRepo := &mockCommandRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return expected, nil
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	result, err := svc.GetCommandParamPaths(context.Background(), cmdID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.ParamPaths, 2)
	for _, p := range result.ParamPaths {
		assert.True(t, p.Writable, "MOD operation is writable")
		assert.Equal(t, p.Path, p.Label, "label falls back to path absent i18n")
	}
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

// Sprint B-6: 孤儿 command_code（FE Console 的 mml_custom_command 引用已下线码）
// 必须降级为透传 entry 而非 500，确保 FE 用户体验不被 standard-model 重建影响。
func TestService_ExecuteCommand_OrphanCommandCode_FallsBackInsteadOfErroring(t *testing.T) {
	cmdRepo := &mockCommandRepo{
		getByCodeFn: func(ctx context.Context, code string) (*MMLCommand, error) {
			assert.Equal(t, "ORPHAN_OLD_CODE", code)
			return nil, commonerrors.ErrNotFound
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
		CommandCode:   "ORPHAN_OLD_CODE",
		DeviceSNs:     []string{"SN-orphan"},
		Parameters:    map[string]interface{}{"x": "y"},
		OperationType: "LST",
		Creator:       "admin",
	}

	result, err := svc.ExecuteCommand(context.Background(), req)
	require.NoError(t, err, "orphan command_code should NOT 500")
	require.NotNil(t, result)
	require.NotNil(t, capturedTask)

	require.Len(t, capturedTask.Commands, 1)
	entry := capturedTask.Commands[0]
	assert.Equal(t, "ORPHAN_OLD_CODE", entry["command_code"])
	assert.Equal(t, true, entry["orphan"], "orphan marker required for FE/audit")
	assert.Equal(t, "LST", entry["operation_type"])
	_, hasRPC := entry["rpc_method"]
	assert.False(t, hasRPC, "rpc_method not set; Fanouter will skip per existing logic")
}

// 其它（非 NotFound）错误依旧抛出 — 不能把 DB 连接抖动也吞掉。
func TestService_ExecuteCommand_NonNotFoundLookupError_StillFails(t *testing.T) {
	cmdRepo := &mockCommandRepo{
		getByCodeFn: func(ctx context.Context, code string) (*MMLCommand, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	svc := newTestService(cmdRepo, &mockScriptRepo{}, &mockTaskRepo{})
	_, err := svc.ExecuteCommand(context.Background(), ExecuteRequest{
		CommandCode: "ANY",
		DeviceSNs:   []string{"SN"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve command code")
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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))

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

// ---------------------------------------------------------------------------
// T-0090-c: ListCustomCommands RBAC visibility tests
// ---------------------------------------------------------------------------
//
// 设计目标（subtask 文件 §0 sub-task c + R-NEW-2 mitigation）：
//   - public 命令始终可见（不受 RBAC 影响）
//   - private 命令：(creator==username self-fallback) OR (creator's group ∈ visible)
//   - 跨用户隔离：admin_a / admin_b / admin_c (groups [A,B]) / admin_d (no group)
//   - 派生失败降级（roleQuerier 报错）只信 creator-self，不放行 group-share
//   - roleQuerier 未注入（旧服务）只信 creator-self（向后兼容）
//
// 测试粒度：直接断言 repo.List 收到的 filter；不验 SQL 行为（PG 集成测试覆盖）。
// 每个 case 都关注一个 admin（user_id + username + groups），跑 List 后看 filter
// 是否正确派生 VisibleGroupIDs / 透传 Creator。

func newCustomCmdServiceForRBAC(repo *mockCustomCommandRepo, rq RoleQuerier) *Service {
	svc := NewService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{}, repo, nil, zap.NewNop())
	if rq != nil {
		svc.SetRoleQuerier(rq)
	}
	return svc
}

func TestService_ListCustomCommands_RBAC_AdminAWithGroupA(t *testing.T) {
	adminA := uuid.New()
	groupA := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	svc := newCustomCmdServiceForRBAC(repo, &mockRoleQuerier{
		groupsByUser: map[uuid.UUID][]uuid.UUID{adminA: {groupA}},
	})

	usernameA := "admin_a"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameA,
		UserID:      &adminA,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{groupA}, captured.VisibleGroupIDs,
		"admin_a 在 group_A 时，VisibleGroupIDs 应包含 group_A")
	assert.Equal(t, "admin_a", *captured.Creator,
		"Creator self-fallback username 必须透传到 repo")
}

func TestService_ListCustomCommands_RBAC_AdminBWithGroupB(t *testing.T) {
	adminA := uuid.New()
	adminB := uuid.New()
	groupA := uuid.New()
	groupB := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	svc := newCustomCmdServiceForRBAC(repo, &mockRoleQuerier{
		groupsByUser: map[uuid.UUID][]uuid.UUID{
			adminA: {groupA},
			adminB: {groupB},
		},
	})

	usernameB := "admin_b"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameB,
		UserID:      &adminB,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{groupB}, captured.VisibleGroupIDs,
		"admin_b 在 group_B 时，VisibleGroupIDs 仅 group_B，跨用户隔离")
	assert.NotContains(t, captured.VisibleGroupIDs, groupA,
		"admin_b 不应看到 group_A — R-NEW-2 防泄露")
}

func TestService_ListCustomCommands_RBAC_AdminCWithMultipleGroups(t *testing.T) {
	adminC := uuid.New()
	groupA := uuid.New()
	groupB := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	svc := newCustomCmdServiceForRBAC(repo, &mockRoleQuerier{
		groupsByUser: map[uuid.UUID][]uuid.UUID{adminC: {groupA, groupB}},
	})

	usernameC := "admin_c"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameC,
		UserID:      &adminC,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{groupA, groupB}, captured.VisibleGroupIDs,
		"admin_c 同时属 group_A+B 时，可见两组所有 private 命令（group-share 路径）")
}

func TestService_ListCustomCommands_RBAC_AdminDWithoutGroups(t *testing.T) {
	adminD := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	svc := newCustomCmdServiceForRBAC(repo, &mockRoleQuerier{
		groupsByUser: map[uuid.UUID][]uuid.UUID{adminD: nil}, // 无 group
	})

	usernameD := "admin_d"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameD,
		UserID:      &adminD,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Empty(t, captured.VisibleGroupIDs,
		"admin_d 无 group 时 VisibleGroupIDs 应为空；private 仅 creator-self fallback 可见")
	assert.Equal(t, "admin_d", *captured.Creator,
		"无 group 时 Creator self-fallback 仍透传 — 用户自己创建的 private 至少自己能看到")
}

func TestService_ListCustomCommands_RBAC_QuerierError_DegradesToCreatorSelf(t *testing.T) {
	adminA := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	rq := &mockRoleQuerier{err: assertError("DB unreachable")}
	svc := newCustomCmdServiceForRBAC(repo, rq)

	usernameA := "admin_a"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameA,
		UserID:      &adminA,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err, "派生失败不应阻断查询；service 须降级")
	assert.Empty(t, captured.VisibleGroupIDs,
		"GetUserVisibleGroupIDs 报错时 VisibleGroupIDs 须保持空（不残留旧值/不放行 group-share）")
	assert.Equal(t, "admin_a", *captured.Creator,
		"降级后 Creator self-fallback 仍透传，私有命令至少自己能看到")
}

func TestService_ListCustomCommands_RBAC_NoQuerierInjected_FallbackToCreatorOnly(t *testing.T) {
	adminA := uuid.New()

	var captured CustomCommandFilter
	repo := &mockCustomCommandRepo{
		listFn: func(_ context.Context, f CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
			captured = f
			return model.NewListResponse([]MMLCustomCommand{}, 0, 1, 20), nil
		},
	}
	// RoleQuerier 未注入（向后兼容场景：旧测试 / 旧部署）
	svc := newCustomCmdServiceForRBAC(repo, nil)

	usernameA := "admin_a"
	_, err := svc.ListCustomCommands(context.Background(), CustomCommandFilter{
		Creator:     &usernameA,
		UserID:      &adminA,
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Empty(t, captured.VisibleGroupIDs,
		"RoleQuerier 未注入时不应触发派生 — VisibleGroupIDs 须为空")
	assert.Equal(t, "admin_a", *captured.Creator,
		"无 RoleQuerier 场景下纯 creator-self 过滤（向后兼容）")
}

// assertError 是测试用 sentinel error，避免引入 errors 包仅为构造常量错误。
type assertError string

func (e assertError) Error() string { return string(e) }

// ========================================================================
// 私有模板 CRUD + 用户级唯一性 — mml-user-private-template-crud-20260520.md §6.1
// ========================================================================

func newCRUDServiceWithRepo(repo *mockCustomCommandRepo) *Service {
	return NewService(&mockCommandRepo{}, &mockScriptRepo{}, &mockTaskRepo{}, repo, nil, zap.NewNop())
}

// case 1: CreateCustomCommand_PrivateNameDuplicate → 409
func TestService_CreateCustomCommand_PrivateNameDuplicate(t *testing.T) {
	ownerID := uuid.New()
	repo := &mockCustomCommandRepo{
		nameExistsFn: func(_ context.Context, oid uuid.UUID, name string, _ *uuid.UUID) (bool, error) {
			assert.Equal(t, ownerID, oid)
			assert.Equal(t, "重启", name)
			return true, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	_, err := svc.CreateCustomCommand(context.Background(), &MMLCustomCommand{
		CommandName:  "重启",
		CommandScope: "private",
		OwnerUserID:  &ownerID,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists), "should wrap ErrAlreadyExists → 409")
}

// case 2: 同名跨 owner 在 service pre-check 阶段允许（DB 索引也允许）
func TestService_CreateCustomCommand_PrivateNameAcrossOwners_Allowed(t *testing.T) {
	repo := &mockCustomCommandRepo{
		nameExistsFn: func(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
			return false, nil // 跨 owner 查询时返回 false
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	ownerA := uuid.New()
	_, err := svc.CreateCustomCommand(context.Background(), &MMLCustomCommand{
		CommandName: "重启", CommandScope: "private", OwnerUserID: &ownerA,
	})
	require.NoError(t, err)
}

// case 3: public scope 走全局唯一检查（NameExistsForPublic），无撞名时允许；
// 不应触发私有命名空间检查 NameExistsForPrivate。
func TestService_CreateCustomCommand_PublicUniqueName_Allowed(t *testing.T) {
	privateCalls := 0
	publicChecked := false
	repo := &mockCustomCommandRepo{
		nameExistsFn: func(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
			privateCalls++
			return true, nil
		},
		publicNameFn: func(_ context.Context, name string, excludeID *uuid.UUID) (bool, error) {
			publicChecked = true
			assert.Equal(t, "公共重启", name)
			assert.Nil(t, excludeID, "Create 不排除自身")
			return false, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	ownerA := uuid.New()
	_, err := svc.CreateCustomCommand(context.Background(), &MMLCustomCommand{
		CommandName: "公共重启", CommandScope: "public", OwnerUserID: &ownerA,
	})
	require.NoError(t, err)
	assert.True(t, publicChecked, "public scope 应走 NameExistsForPublic")
	assert.Zero(t, privateCalls, "public scope 不应触发 NameExistsForPrivate")
}

// case 3b: public scope 全局撞名（任意用户已有同名公共命令）→ 409
func TestService_CreateCustomCommand_PublicNameDuplicate_409(t *testing.T) {
	repo := &mockCustomCommandRepo{
		publicNameFn: func(_ context.Context, name string, _ *uuid.UUID) (bool, error) {
			assert.Equal(t, "公共重启", name)
			return true, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	ownerA := uuid.New()
	_, err := svc.CreateCustomCommand(context.Background(), &MMLCustomCommand{
		CommandName: "公共重启", CommandScope: "public", OwnerUserID: &ownerA,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists), "public 撞名应 → 409")
}

// case 3c: public 改名撞已有公共名 → 409，且排除自身
func TestService_UpdateCustomCommand_PublicRenameDuplicate_409(t *testing.T) {
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "public", Creator: "alice",
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		publicNameFn: func(_ context.Context, name string, excludeID *uuid.UUID) (bool, error) {
			assert.Equal(t, "Y", name)
			require.NotNil(t, excludeID)
			assert.Equal(t, existing.ID, *excludeID, "Update 改名时必须排除自身")
			return true, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	// super_admin 绕过 owner 鉴权，专测公共唯一性分支
	_, err := svc.UpdateCustomCommand(context.Background(), existing.ID,
		&MMLCustomCommand{CommandName: "Y"}, uuid.New(), "super", true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists), "public 改名撞名应 → 409")
}

// case 4: 非 owner 修改私有模板 → 403
func TestService_UpdateCustomCommand_NonOwner_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) {
			return existing, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	_, err := svc.UpdateCustomCommand(context.Background(), existing.ID,
		&MMLCustomCommand{CommandName: "X2"}, otherID, "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden), "non-owner update should be 403")
}

// case 5: owner 自己 update 同名（无变更）→ OK 且不查 NameExistsForPrivate
func TestService_UpdateCustomCommand_OwnerSameName_OK(t *testing.T) {
	ownerID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "重启", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	nameQueried := false
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		nameExistsFn: func(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
			nameQueried = true
			return true, nil
		},
		updateFn: func(_ context.Context, _ *MMLCustomCommand) error { return nil },
	}
	svc := newCRUDServiceWithRepo(repo)

	_, err := svc.UpdateCustomCommand(context.Background(), existing.ID,
		&MMLCustomCommand{CommandName: "重启"}, ownerID, "alice", false)
	require.NoError(t, err)
	assert.False(t, nameQueried, "name unchanged → 不查 NameExistsForPrivate")
}

// case 6: super_admin 跨用户 update → OK
func TestService_UpdateCustomCommand_SuperAdmin_BypassOwner(t *testing.T) {
	ownerID := uuid.New()
	superID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		updateFn:  func(_ context.Context, _ *MMLCustomCommand) error { return nil },
	}
	svc := newCRUDServiceWithRepo(repo)

	_, err := svc.UpdateCustomCommand(context.Background(), existing.ID,
		&MMLCustomCommand{CommandName: "X"}, superID, "super", true)
	require.NoError(t, err)
}

// case 7: owner 改名撞同 owner 已有私有名 → 409
func TestService_UpdateCustomCommand_RenameToOtherExisting_409(t *testing.T) {
	ownerID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		nameExistsFn: func(_ context.Context, oid uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
			assert.Equal(t, ownerID, oid)
			assert.Equal(t, "Y", name)
			require.NotNil(t, excludeID)
			assert.Equal(t, existing.ID, *excludeID, "Update 改名时必须排除自身")
			return true, nil
		},
	}
	svc := newCRUDServiceWithRepo(repo)

	_, err := svc.UpdateCustomCommand(context.Background(), existing.ID,
		&MMLCustomCommand{CommandName: "Y"}, ownerID, "alice", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists), "should be 409")
}

// case 8: Delete 私有 + 非 owner 非 super → 403（修复历史 G2）
func TestService_DeleteCustomCommand_PrivateNonOwner_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	svc := newCRUDServiceWithRepo(repo)

	err := svc.DeleteCustomCommand(context.Background(), existing.ID, otherID, "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden), "non-owner private delete should be 403")
}

// case 9: Delete 私有 + owner → OK
func TestService_DeleteCustomCommand_PrivateOwner_OK(t *testing.T) {
	ownerID := uuid.New()
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		deleteFn:  func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := newCRUDServiceWithRepo(repo)

	err := svc.DeleteCustomCommand(context.Background(), existing.ID, ownerID, "alice", false)
	require.NoError(t, err)
}

// case 10: Delete + legacy NULL owner_user_id → 走 creator (username) 兜底鉴权
func TestService_DeleteCustomCommand_LegacyNullOwner_CreatorFallback(t *testing.T) {
	existing := &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: nil, // 历史脏数据
	}
	repo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
		deleteFn:  func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := newCRUDServiceWithRepo(repo)

	// 任何 user_id 都行，关键是 username == creator
	err := svc.DeleteCustomCommand(context.Background(), existing.ID, uuid.New(), "alice", false)
	require.NoError(t, err, "owner_user_id 为 NULL 时应按 creator (username) 兜底放行")
}
