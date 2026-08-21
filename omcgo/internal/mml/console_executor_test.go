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
		// MMLParamRef.ID == csf.id（sub_field.id）— 与 paramRefSelectExpr 对齐。
		Params: []MMLParamRef{
			{ID: sf1ID, ParamCode: "PATH1", Tr069Path: "Device.X.Y", ValueType: "string"},
			{ID: sf2ID, ParamCode: "PATH2", Tr069Path: "Device.X.Z", ValueType: "int"},
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
			{ID: sf1ID, ParamCode: "PATH1", Tr069Path: "Device.X", ValueType: "string"},
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
			{ID: sf1ID, ParamCode: "ignored", Tr069Path: "Device.X", ValueType: "string"},
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

func TestBuildEntry_MOD_OnlyIncludesSubFieldsWithValues(t *testing.T) {
	cmdID := uuid.New()
	firstSelectedSFID := uuid.New()
	secondSelectedSFID := uuid.New()
	unselectedSFID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_DEVICE_INFO",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: firstSelectedSFID, ParamCode: "ignored-first", Tr069Path: "Device.Info.First"},
			{ID: secondSelectedSFID, ParamCode: "ignored-second", Tr069Path: "Device.Info.Second"},
			{ID: unselectedSFID, ParamCode: "ignored-unselected", Tr069Path: "Device.Info.Unselected"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: firstSelectedSFID, CommandID: cmdID, MMLCode: "FIRST"},
		{ID: secondSelectedSFID, CommandID: cmdID, MMLCode: "SECOND"},
		{ID: unselectedSFID, CommandID: cmdID, MMLCode: "UNSELECTED"},
	}
	stmt := Statement{
		CommandID:     &cmdID,
		OperationType: "MOD",
		Values: map[string]string{
			"FIRST":  "first-value",
			"SECOND": "second-value",
		},
	}

	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)

	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 2)
	assert.Equal(t, "FIRST", refs[0].ParamCode)
	assert.Equal(t, "Device.Info.First", refs[0].Tr069Path)
	assert.Equal(t, "SECOND", refs[1].ParamCode)
	assert.Equal(t, "Device.Info.Second", refs[1].Tr069Path)
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"FIRST":  "first-value",
		"SECOND": "second-value",
	}, params)
}

func TestBuildEntry_MOD_NoMatchingValuePreservesLegacyRefs(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_DEVICE_INFO",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "ignored", Tr069Path: "Device.Info.Known"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, MMLCode: "KNOWN"},
	}
	stmt := Statement{
		CommandID:     &cmdID,
		OperationType: "MOD",
		Values:        map[string]string{"UNKNOWN": "legacy-value"},
	}

	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)

	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "KNOWN", refs[0].ParamCode)
	assert.Equal(t, "Device.Info.Known", refs[0].Tr069Path)
	assert.Equal(t, "legacy-value", entry["parameters"].(map[string]interface{})["UNKNOWN"])
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

// ============================================================
// R-4.3 ADD 复合流程：buildStatementCommandEntries
// ============================================================

// TestBuildEntries_ADD_WithoutValues_SingleEntry — ADD 无 values 退化为单 entry
// （与历史 1 MML = 1 RPC 行为一致）。
func TestBuildEntries_ADD_WithoutValues_SingleEntry(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_USER",
		OperationType: "ADD",
		TargetObject:  "Device.Users.User.",
	}
	stmt := Statement{OperationType: "ADD"}
	entries, err := buildStatementCommandEntries(stmt, cmd, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "AddObject", entries[0]["rpc_method"])
}

// TestBuildEntries_ADD_WithValues_CompoundTwoEntries — ADD with values 触发 2 entries：
// AddObject + SetParameterValues with .{NEW}. 占位符。
func TestBuildEntries_ADD_WithValues_CompoundTwoEntries(t *testing.T) {
	sfID := uuid.New()
	paramID := uuid.New()
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_PLMNLIST",
		OperationType: "ADD",
		TargetObject:  "Device.Services.FAPService.{i}.PLMNList.",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "PLMNID", Tr069Path: "Device.Services.FAPService.{i}.PLMNList.{i}.PLMNID"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, ParamID: paramID, MMLCode: "PLMNID"},
	}
	stmt := Statement{
		OperationType: "ADD",
		InstanceSelectors: map[string]string{
			"iα": "1",
		},
		Values: map[string]string{"PLMNID": "46000"},
	}
	entries, err := buildStatementCommandEntries(stmt, cmd, subFields)
	require.NoError(t, err)
	require.Len(t, entries, 3, "ADD with values → AddObject + SPV + conditional rollback")

	// 第 1 entry: AddObject，target_object 已 substitute iα=1
	assert.Equal(t, "AddObject", entries[0]["rpc_method"])
	params0 := entries[0]["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.", params0["object_name"])

	// 第 2 entry: SetParameterValues with .{NEW}. 占位符
	assert.Equal(t, "SetParameterValues", entries[1]["rpc_method"])
	assert.Equal(t, "MOD", entries[1]["operation_type"], "compound SPV 标 MOD 语义")
	assert.Equal(t, "spv_after_add", entries[1]["compound_phase"], "Sequencer / audit 用此识别")
	refs1 := entries[1]["param_refs"].([]MMLParamRef)
	require.Len(t, refs1, 1)
	assert.Equal(t,
		"Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID",
		refs1[0].Tr069Path,
		"外层 {i} 被 selector 替换；最末尾 {i}（新实例）替换为 {NEW}",
	)
	params1 := entries[1]["parameters"].(map[string]interface{})
	assert.Equal(t, "46000", params1["PLMNID"])

	assert.Equal(t, "DeleteObject", entries[2]["rpc_method"])
	assert.Equal(t, "rollback_after_add", entries[2]["compound_phase"])
	assert.Equal(t, true, entries[2]["compensation_only"])
	params2 := entries[2]["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.{NEW}.", params2["object_name"])
}

// TestBuildEntries_ADD_WithValues_NoSubFieldMatch_OnlyAddObject — stmt.Values keys
// 都不在 subFields 中 → 第 2 行 SPV 不构造，回退单 entry。
func TestBuildEntries_ADD_WithValues_NoSubFieldMatch_OnlyAddObject(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_X",
		OperationType: "ADD",
		TargetObject:  "Device.X.",
	}
	stmt := Statement{
		OperationType: "ADD",
		Values:        map[string]string{"NonExistent": "v"},
	}
	entries, err := buildStatementCommandEntries(stmt, cmd, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1, "无 sub_field 匹配 → 仅 AddObject")
}

// ============================================================
// #196 MOD 回读复合：buildStatementCommandEntries
// ============================================================

// TestBuildEntries_MOD_WithValues_CompoundReadbackLST — MOD with values 触发 2 entries：
// SetParameterValues（下发）+ GetParameterValues（回读核实下发的 PATH，根治 SPV 不回值导致的结果显示空）。
func TestBuildEntries_MOD_WithValues_CompoundReadbackLST(t *testing.T) {
	sfID := uuid.New()
	otherID := uuid.New()
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "MOD_KPI",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "KPI_URL", Tr069Path: "Device.X_KPI.ReportURL"},
			{ID: otherID, ParamCode: "OTHER", Tr069Path: "Device.X_KPI.Other"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, MMLCode: "KPI_URL"},
		{ID: otherID, MMLCode: "OTHER"},
	}
	// 仅下发 KPI_URL（OTHER 未填）
	stmt := Statement{
		OperationType: "MOD",
		Values:        map[string]string{"KPI_URL": "http://1.2.3.4/kpi"},
	}
	entries, err := buildStatementCommandEntries(stmt, cmd, subFields)
	require.NoError(t, err)
	require.Len(t, entries, 2, "MOD with values → SPV + 回读 LST")

	// 第 1 entry: SetParameterValues 下发
	assert.Equal(t, "SetParameterValues", entries[0]["rpc_method"])
	assert.Equal(t, "MOD", entries[0]["operation_type"])
	assert.Equal(t, "http://1.2.3.4/kpi", entries[0]["parameters"].(map[string]interface{})["KPI_URL"])

	// 第 2 entry: GetParameterValues 回读，且**只回读本次实际下发的 PATH**（KPI_URL，不含未填的 OTHER）
	assert.Equal(t, "GetParameterValues", entries[1]["rpc_method"])
	assert.Equal(t, "LST", entries[1]["operation_type"])
	assert.Equal(t, "lst_after_mod", entries[1]["compound_phase"])
	refs := entries[1]["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1, "回读仅含已下发的 KPI_URL")
	assert.Equal(t, "KPI_URL", refs[0].ParamCode)
	assert.Equal(t, "Device.X_KPI.ReportURL", refs[0].Tr069Path)
}

// TestBuildEntries_MOD_NoSubFieldMatch_OnlySPV — MOD values keys 不命中 subFields →
// 无可回读 → 仅 SPV（防御性回退，不追加空 LST）。
func TestBuildEntries_MOD_NoSubFieldMatch_OnlySPV(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "MOD_X",
		OperationType: "MOD",
	}
	stmt := Statement{
		OperationType: "MOD",
		Values:        map[string]string{"NonExistent": "v"},
	}
	entries, err := buildStatementCommandEntries(stmt, cmd, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1, "无 sub_field 匹配 → 仅 SPV，不追加回读 LST")
	assert.Equal(t, "SetParameterValues", entries[0]["rpc_method"])
}

// TestBuildRawReadbackLSTCommand — #196 raw/自定义命令通道 MOD 回读命令构造（GetParameterValues）。
func TestBuildRawReadbackLSTCommand(t *testing.T) {
	paths := []string{"Device.FAP.PerfMgmt.Config.1.URL", "Device.X.Y"}
	cmd := buildRawReadbackLSTCommand(paths)
	assert.Equal(t, "GetParameterValues", cmd["rpc_method"])
	assert.Equal(t, "LST", cmd["operation_type"])
	assert.Equal(t, "lst_after_mod", cmd["compound_phase"])
	assert.Equal(t, paths, cmd["param_paths"])
	refs := cmd["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 2)
	assert.Equal(t, "Device.FAP.PerfMgmt.Config.1.URL", refs[0].Tr069Path)
}

// TestCommandsNeedSequential — 含 lst_after_mod 复合 → 需顺序执行（保证回读 LST 在 SPV 之后）。
func TestCommandsNeedSequential(t *testing.T) {
	assert.False(t, commandsNeedSequential([]map[string]interface{}{
		{"rpc_method": "SetParameterValues"},
	}), "纯 SPV 不需顺序")
	assert.True(t, commandsNeedSequential([]map[string]interface{}{
		{"rpc_method": "SetParameterValues"},
		{"rpc_method": "GetParameterValues", "compound_phase": "lst_after_mod"},
	}), "SPV + 回读 LST → 需顺序")
}

// TestSubstituteInstanceSelectorsForADDCompound — 三种 path/selectors 组合
func TestSubstituteInstanceSelectorsForADDCompound(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		selectors map[string]string
		want      string
		wantErr   bool
	}{
		{
			name:      "0 selector + 1 .{i}. → {NEW}",
			path:      "Device.Users.User.{i}.Name",
			selectors: map[string]string{},
			want:      "Device.Users.User.{NEW}.Name",
		},
		{
			name:      "1 selector + 2 .{i}. → 外层替换为值，内层替换为 {NEW}",
			path:      "Device.Services.FAPService.{i}.PLMNList.{i}.PLMNID",
			selectors: map[string]string{"iα": "1"},
			want:      "Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID",
		},
		{
			name:      "terminal object placeholder .{i} is counted as new instance",
			path:      "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}",
			selectors: map[string]string{"iα": "1"},
			want:      "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{NEW}",
		},
		{
			name: "2 selectors + 3 .{i}. → 前 2 个替换值，最后 1 个 {NEW}",
			path: "Device.X.{i}.Y.{i}.Z.{i}.Foo",
			selectors: map[string]string{
				"iα": "1",
				"iβ": "2",
			},
			want: "Device.X.1.Y.2.Z.{NEW}.Foo",
		},
		{
			name:      "selectors+1 不等于 .{i}. 数 → error",
			path:      "Device.X.Y", // 0 个 .{i}.
			selectors: map[string]string{"iα": "1"},
			wantErr:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := substituteInstanceSelectorsForADDCompound(tc.path, tc.selectors)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
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
			{ID: sfID, ParamCode: "P", Tr069Path: "Device.X", ValueType: "string"},
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
			{ID: sfID, ParamCode: "P", Tr069Path: "Device.X", ValueType: "string"},
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
			{ID: sfID, ParamCode: "P", Tr069Path: "Device.X.Y", ValueType: "string"},
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
			{ID: sfID, ParamCode: "P", Tr069Path: "Device.X.Y", ValueType: "string"},
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
