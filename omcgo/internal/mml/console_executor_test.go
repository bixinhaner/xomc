package mml

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// console_executor_test.go — T-0123-P1 S3-D2 下
//
// 覆盖：
//   - buildStatementCommandEntry 四种 op（LST/MOD/ADD/RMV）的 entry shape
//   - ExecuteStatements 多 statement → sequential=true，单 statement → sequential=false
//   - 失败语义：空 statements、缺 target_object、缺 RmvInstanceIndex
//   - BuildTR069Params 对 LST/MOD/ADD/RMV 4 类 entry 的输出 wire 格式
// ============================================================

// fakeTaskCreator 用 closure 记录调用，便于断言 sequential / commands。
type fakeTaskCreator struct {
	captured   *MMLTask
	sequential bool
	err        error
}

func (f *fakeTaskCreator) CreateAndFanoutTask(_ context.Context, task *MMLTask, sequential bool) error {
	f.captured = task
	f.sequential = sequential
	if f.err != nil {
		return f.err
	}
	task.ID = uuid.New()
	task.Status = TaskRunning
	return nil
}

// ============================================================
// buildStatementCommandEntry — 四种 op
// ============================================================

func TestBuildEntry_LST_SelectedSubset(t *testing.T) {
	cmdID := uuid.New()
	sf1ID := uuid.New()
	sf2ID := uuid.New()
	p1ID := uuid.New()
	p2ID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_DEVICE_INFO",
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: p1ID, ParamCode: "PATH1", Tr069Path: "Device.X.Y", ValueType: "string"},
			{ID: p2ID, ParamCode: "PATH2", Tr069Path: "Device.X.Z", ValueType: "int"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sf1ID, CommandID: cmdID, ParamID: p1ID, MMLCode: "MMLC1"},
		{ID: sf2ID, CommandID: cmdID, ParamID: p2ID, MMLCode: "MMLC2"},
	}
	stmt := Statement{
		CommandID:           &cmdID,
		LogicalCode:         "DEVICE_INFO",
		OperationType:       "LST",
		SelectedSubFieldIDs: []uuid.UUID{sf1ID}, // 仅选 sf1
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	assert.Equal(t, "GetParameterValues", entry["rpc_method"])
	assert.Equal(t, "LST", entry["operation_type"])
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "Device.X.Y", refs[0].Tr069Path)
}

func TestBuildEntry_LST_EmptySelectionDefaultsToAll(t *testing.T) {
	cmdID := uuid.New()
	sf1ID := uuid.New()
	p1ID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_DEVICE_INFO",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: p1ID, ParamCode: "PATH1", Tr069Path: "Device.X", ValueType: "string"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sf1ID, CommandID: cmdID, ParamID: p1ID, MMLCode: "MMLC1"},
	}
	stmt := Statement{
		CommandID:     &cmdID,
		OperationType: "LST",
		// SelectedSubFieldIDs 为空 → 默认全选
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "Device.X", refs[0].Tr069Path)
}

func TestBuildEntry_LST_NoSubFields_Errors(t *testing.T) {
	cmd := &MMLCommand{ID: uuid.New(), OperationType: "LST"}
	stmt := Statement{OperationType: "LST"}
	_, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no usable sub_fields")
}

func TestBuildEntry_MOD(t *testing.T) {
	cmdID := uuid.New()
	sf1ID := uuid.New()
	p1ID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_DEVICE_INFO",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: p1ID, ParamCode: "ignored", Tr069Path: "Device.X", ValueType: "string"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sf1ID, CommandID: cmdID, ParamID: p1ID, MMLCode: "MyField"},
	}
	stmt := Statement{
		CommandID:     &cmdID,
		OperationType: "MOD",
		Values:        map[string]string{"MyField": "newValue"},
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	assert.Equal(t, "SetParameterValues", entry["rpc_method"])
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "MyField", refs[0].ParamCode) // ParamCode 改写为 MMLCode
	assert.Equal(t, "Device.X", refs[0].Tr069Path)
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "newValue", params["MyField"])
}

func TestBuildEntry_MOD_EmptyValues_Errors(t *testing.T) {
	cmd := &MMLCommand{ID: uuid.New(), OperationType: "MOD"}
	stmt := Statement{OperationType: "MOD"}
	_, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty values")
}

func TestBuildEntry_ADD(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_USER",
		OperationType: "ADD",
		TargetObject:  "Device.Users.User.",
	}
	stmt := Statement{
		OperationType: "ADD",
		Values:        map[string]string{"Name": "alice"},
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "AddObject", entry["rpc_method"])
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.Users.User.", params["object_name"])
	assert.Equal(t, "alice", params["Name"]) // SPV 字段透传保留
}

func TestBuildEntry_ADD_MissingTargetObject_Errors(t *testing.T) {
	cmd := &MMLCommand{ID: uuid.New(), CommandCode: "ADD_X", OperationType: "ADD"}
	stmt := Statement{OperationType: "ADD"}
	_, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing target_object")
}

func TestBuildEntry_ADD_AppendsTrailingDot(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_X",
		OperationType: "ADD",
		TargetObject:  "Device.Users.User", // 无尾点
	}
	stmt := Statement{OperationType: "ADD"}
	entry, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.NoError(t, err)
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.Users.User.", params["object_name"])
}

func TestBuildEntry_RMV(t *testing.T) {
	idx := 5
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "RMV_USER",
		OperationType: "RMV",
		TargetObject:  "Device.Users.User.",
	}
	stmt := Statement{
		OperationType:    "RMV",
		RmvInstanceIndex: &idx,
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "DeleteObject", entry["rpc_method"])
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.Users.User.5.", params["object_name"])
}

func TestBuildEntry_RMV_MissingIndex_Errors(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "RMV_X",
		OperationType: "RMV",
		TargetObject:  "Device.X.",
	}
	stmt := Statement{OperationType: "RMV"}
	_, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing instance index")
}

// 用户决策 2026-05-20：RMV 保持单实例（1 MML 命令 = 1 RPC，禁止 R-7 多选展开）。
// 原 TestResolveRMVIndices / TestBuildEntry_RMV_MultiInstance /
// TestExecuteStatements_MultiRMV_TriggersSequential / TestExecuteStatements_SingleRMV_NotSequential
// 已移除；单实例 RMV 行为由 TestBuildEntry_RMV (上方) 覆盖。

func TestBuildEntry_UnsupportedOp(t *testing.T) {
	cmd := &MMLCommand{ID: uuid.New(), OperationType: "DSP"}
	stmt := Statement{OperationType: "DSP"}
	_, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// ============================================================
// ExecuteStatements — 编排
// ============================================================

func TestExecuteStatements_SingleStatement_NotSequential(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	pID := uuid.New()

	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_X",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: pID, ParamCode: "P", Tr069Path: "Device.X", ValueType: "string"},
		},
	})
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "MMLC"},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	creator := &fakeTaskCreator{}
	task, err := svc.ExecuteStatements(context.Background(), ExecuteStatementsRequest{
		Statements: []Statement{
			{CommandID: &cmdID, OperationType: "LST", SelectedSubFieldIDs: []uuid.UUID{sfID}},
		},
		DeviceSNs: []string{"DEV1", "DEV2"},
		Creator:   "tester",
	}, creator)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.False(t, creator.sequential, "single statement should not be sequential")
	require.NotNil(t, creator.captured)
	require.Len(t, creator.captured.Commands, 1)
	assert.Equal(t, "GetParameterValues", creator.captured.Commands[0]["rpc_method"])
}

func TestExecuteStatements_MultipleStatements_Sequential(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	pID := uuid.New()

	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_X",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: pID, ParamCode: "P", Tr069Path: "Device.X", ValueType: "string"},
		},
	})
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "MMLC"},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	creator := &fakeTaskCreator{}
	_, err := svc.ExecuteStatements(context.Background(), ExecuteStatementsRequest{
		Statements: []Statement{
			{CommandID: &cmdID, OperationType: "LST", SelectedSubFieldIDs: []uuid.UUID{sfID}},
			{CommandID: &cmdID, OperationType: "LST", SelectedSubFieldIDs: []uuid.UUID{sfID}},
		},
		DeviceSNs: []string{"DEV1"},
	}, creator)
	require.NoError(t, err)
	assert.True(t, creator.sequential, "multiple statements should be sequential")
	require.Len(t, creator.captured.Commands, 2)
}

func TestExecuteStatements_EmptyStatements_Errors(t *testing.T) {
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	_, err := svc.ExecuteStatements(context.Background(), ExecuteStatementsRequest{
		DeviceSNs: []string{"DEV1"},
	}, &fakeTaskCreator{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRequest), "should wrap ErrInvalidRequest")
	assert.Contains(t, err.Error(), "statements is empty")
}

func TestExecuteStatements_EmptyDeviceSNs_Errors(t *testing.T) {
	cmdID := uuid.New()
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	_, err := svc.ExecuteStatements(context.Background(), ExecuteStatementsRequest{
		Statements: []Statement{{CommandID: &cmdID, OperationType: "LST"}},
	}, &fakeTaskCreator{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRequest), "should wrap ErrInvalidRequest")
	assert.Contains(t, err.Error(), "device_sns is empty")
}

// TestBuildStatementCommands_WrapsErrInvalidRequest 确保编译错误（如 sub_field 缺失）
// 通过 ErrInvalidRequest 透传，handler 可据此返 400。
func TestBuildStatementCommands_WrapsErrInvalidRequest(t *testing.T) {
	cmdID := uuid.New()
	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_X",
		OperationType: "LST",
	})
	// 空 sub_fields → buildStatementCommandEntry LST 路径返 "no usable sub_fields"
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = []MMLCommandSubField{}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	_, err := svc.BuildStatementCommands(context.Background(), []Statement{
		{CommandID: &cmdID, OperationType: "LST"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRequest),
		"compile failure should wrap ErrInvalidRequest, got: %v", err)
}

func TestExecuteStatements_NilTaskCreator_Errors(t *testing.T) {
	cmdID := uuid.New()
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	_, err := svc.ExecuteStatements(context.Background(), ExecuteStatementsRequest{
		Statements: []Statement{{CommandID: &cmdID, OperationType: "LST"}},
		DeviceSNs:  []string{"DEV1"},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "taskCreator not wired")
}

// ============================================================
// 整体路径：BuildStatementCommands entry → BuildTR069Params wire 格式
// ============================================================

func TestEntry_GetParameterValues_WireFormat(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	pID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_X",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: pID, ParamCode: "P", Tr069Path: "Device.X.Y", ValueType: "string"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "MMLC"},
	}
	entry, err := buildStatementCommandEntry(
		Statement{CommandID: &cmdID, OperationType: "LST", SelectedSubFieldIDs: []uuid.UUID{sfID}},
		cmd, subFields,
	)
	require.NoError(t, err)

	// fanout 会走 BuildTR069Params(rpc_method, param_refs, parameters, operation_type)
	payload, err := BuildTR069Params(
		entry["rpc_method"].(string),
		entry["param_refs"].([]MMLParamRef),
		nil,
		entry["operation_type"].(string),
	)
	require.NoError(t, err)
	var m map[string][]string
	require.NoError(t, json.Unmarshal(payload, &m))
	assert.Equal(t, []string{"Device.X.Y"}, m["names"])
}

func TestEntry_SetParameterValues_WireFormat(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	pID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_X",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: pID, ParamCode: "P", Tr069Path: "Device.X.Y", ValueType: "string"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "MMLC"},
	}
	entry, err := buildStatementCommandEntry(
		Statement{CommandID: &cmdID, OperationType: "MOD", Values: map[string]string{"MMLC": "v1"}},
		cmd, subFields,
	)
	require.NoError(t, err)

	formValues := entry["parameters"].(map[string]interface{})
	payload, err := BuildTR069Params(
		entry["rpc_method"].(string),
		entry["param_refs"].([]MMLParamRef),
		formValues,
		entry["operation_type"].(string),
	)
	require.NoError(t, err)
	var m map[string][]map[string]string
	require.NoError(t, json.Unmarshal(payload, &m))
	require.Len(t, m["values"], 1)
	assert.Equal(t, "Device.X.Y", m["values"][0]["name"])
	assert.Equal(t, "v1", m["values"][0]["value"])
}

func TestEntry_AddObject_WireFormat(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_USER",
		OperationType: "ADD",
		TargetObject:  "Device.Users.User.",
	}
	entry, err := buildStatementCommandEntry(
		Statement{OperationType: "ADD"},
		cmd, nil,
	)
	require.NoError(t, err)

	payload, err := BuildTR069Params(
		entry["rpc_method"].(string),
		nil,
		entry["parameters"].(map[string]interface{}),
		entry["operation_type"].(string),
	)
	require.NoError(t, err)
	var m map[string]string
	require.NoError(t, json.Unmarshal(payload, &m))
	assert.Equal(t, "Device.Users.User.", m["object_name"])
}

func TestEntry_DeleteObject_WireFormat(t *testing.T) {
	idx := 3
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "RMV_USER",
		OperationType: "RMV",
		TargetObject:  "Device.Users.User.",
	}
	entry, err := buildStatementCommandEntry(
		Statement{OperationType: "RMV", RmvInstanceIndex: &idx},
		cmd, nil,
	)
	require.NoError(t, err)

	payload, err := BuildTR069Params(
		entry["rpc_method"].(string),
		nil,
		entry["parameters"].(map[string]interface{}),
		entry["operation_type"].(string),
	)
	require.NoError(t, err)
	var m map[string]string
	require.NoError(t, json.Unmarshal(payload, &m))
	assert.Equal(t, "Device.Users.User.3.", m["object_name"])
}
