package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// instance_selectors_test.go — R-4 多层 {i} 实例选择器
//
// 覆盖：
//   - substituteInstanceSelectors 纯逻辑（单层 / 多层 / 数量不匹配 / 空入参）
//   - applyInstanceSelectorsToRefs LST/MOD param_refs in-place 替换
//   - buildStatementCommandEntry 4 个 op 分支端到端：LST/MOD/ADD/RMV
//   - StructuredExecuteRequest 端到端：adapter 透传 selectors，executor 替换路径
// ============================================================

func TestSubstituteInstanceSelectors(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		selectors map[string]string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "空 selectors + 无占位符路径 → 原样返回",
			path:      "Device.DeviceInfo.SoftwareVersion",
			selectors: nil,
			wantPath:  "Device.DeviceInfo.SoftwareVersion",
		},
		{
			name:      "单层 {i}",
			path:      "Device.IP.Interface.{i}.IPAddress",
			selectors: map[string]string{"iα": "5"},
			wantPath:  "Device.IP.Interface.5.IPAddress",
		},
		{
			name:      "双层 {i}（按 selector key 字典序左到右）",
			path:      "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress",
			selectors: map[string]string{"iα": "1", "iβ": "2"},
			wantPath:  "Device.IP.Interface.1.IPv4Address.2.IPAddress",
		},
		{
			name:      "三层 {i}（v2.3 catalog 实际最深路径）",
			path:      "Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.HardwareVersion",
			selectors: map[string]string{"iα": "1", "iβ": "2", "iγ": "3"},
			wantPath:  "Device.DeviceInfo.MU.1.Slot.2.EU.3.HardwareVersion",
		},
		{
			name:      "数量不匹配 — selectors 多",
			path:      "Device.IP.Interface.{i}.IPAddress",
			selectors: map[string]string{"iα": "1", "iβ": "2"},
			wantErr:   true,
		},
		{
			name:      "数量不匹配 — selectors 少",
			path:      "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress",
			selectors: map[string]string{"iα": "1"},
			wantErr:   true,
		},
		{
			name:      "路径无占位符但传了 selectors → 错误（防御性）",
			path:      "Device.DeviceInfo.SoftwareVersion",
			selectors: map[string]string{"iα": "1"},
			wantErr:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := substituteInstanceSelectors(c.path, c.selectors)
			if c.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidRequest),
					"should wrap ErrInvalidRequest, got: %v", err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.wantPath, got)
		})
	}
}

func TestSubstituteQueryInstanceSelectors(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		selectors map[string]string
		want      string
	}{
		{
			name: "legacy empty selector map keeps template for payload fallback",
			path: "Device.A.{i}.Value",
			want: "Device.A.{i}.Value",
		},
		{
			name:      "single blank selector truncates to parent object",
			path:      "Device.A.{i}.Value",
			selectors: map[string]string{"i01": ""},
			want:      "Device.A.",
		},
		{
			name:      "last blank selector truncates after resolved outer instance",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i01": "1", "i02": ""},
			want:      "Device.A.1.B.",
		},
		{
			name:      "terminal object placeholder blank selector truncates to collection path",
			path:      "Device.A.{i}.B.{i}",
			selectors: map[string]string{"i01": "1", "i02": ""},
			want:      "Device.A.1.B.",
		},
		{
			name:      "terminal object placeholder concrete selector resolves instance",
			path:      "Device.A.{i}.B.{i}",
			selectors: map[string]string{"i01": "1", "i02": "2"},
			want:      "Device.A.1.B.2",
		},
		{
			name:      "missing trailing selector truncates at that layer",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i01": "3"},
			want:      "Device.A.3.B.",
		},
		{
			name:      "extra selector is ignored by shallow path",
			path:      "Device.A.{i}.Value",
			selectors: map[string]string{"i01": "4", "i02": "9"},
			want:      "Device.A.4.Value",
		},
		{
			name:      "numbered sparse map truncates at missing middle layer",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i01": "1", "i03": "3"},
			want:      "Device.A.1.B.",
		},
		{
			name:      "numbered first layer blank truncates before later value",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i01": "", "i02": "2"},
			want:      "Device.A.",
		},
		{
			name:      "numbered three layers use their fixed layer keys",
			path:      "Device.A.{i}.B.{i}.C.{i}.Value",
			selectors: map[string]string{"i01": "1", "i02": "2", "i03": "3"},
			want:      "Device.A.1.B.2.C.3.Value",
		},
		{
			name:      "legacy Greek keys retain sorted positional mapping",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"iβ": "2", "iα": "1"},
			want:      "Device.A.1.B.2.Value",
		},
		{
			name:      "mixed selectors keep numbered key bound to its exact layer",
			path:      "Device.A.{i}.B.{i}.C.{i}.Value",
			selectors: map[string]string{"i03": "3", "iα": "1", "iβ": "2"},
			want:      "Device.A.1.B.2.C.3.Value",
		},
		{
			name:      "mixed i02 leaves layer one for legacy selector",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i02": "2", "iα": "1"},
			want:      "Device.A.1.B.2.Value",
		},
		{
			name:      "out of range numbered key is ignored before legacy fill",
			path:      "Device.A.{i}.B.{i}.Value",
			selectors: map[string]string{"i03": "9", "iα": "1", "iβ": "2"},
			want:      "Device.A.1.B.2.Value",
		},
		{
			name:      "mixed blank numbered selector truncates its layer and ignores later values",
			path:      "Device.A.{i}.B.{i}.C.{i}.Value",
			selectors: map[string]string{"i02": "", "iα": "1", "iβ": "3"},
			want:      "Device.A.1.B.",
		},
		{
			name:      "mixed sparse selectors truncate at first unbound layer",
			path:      "Device.A.{i}.B.{i}.C.{i}.D.{i}.Value",
			selectors: map[string]string{"i02": "2", "i04": "4", "iα": "1"},
			want:      "Device.A.1.B.2.C.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, substituteQueryInstanceSelectors(tc.path, tc.selectors))
		})
	}
}

func TestApplyInstanceSelectorsToRefs(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "P1", Tr069Path: "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress"},
		{ParamCode: "P2", Tr069Path: "Device.IP.Interface.{i}.IPv4Address.{i}.Enable"},
	}
	err := applyInstanceSelectorsToRefs(refs, map[string]string{"iα": "1", "iβ": "2"})
	require.NoError(t, err)
	assert.Equal(t, "Device.IP.Interface.1.IPv4Address.2.IPAddress", refs[0].Tr069Path)
	assert.Equal(t, "Device.IP.Interface.1.IPv4Address.2.Enable", refs[1].Tr069Path)
}

func TestApplyQueryInstanceSelectorsToRefs_MixedDepth(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "P1", Tr069Path: "Device.A.{i}.Value"},
		{ParamCode: "P2", Tr069Path: "Device.A.{i}.B.{i}.Value"},
	}
	applyQueryInstanceSelectorsToRefs(refs, map[string]string{"i01": "1", "i02": ""})
	assert.Equal(t, "Device.A.1.Value", refs[0].Tr069Path)
	assert.Equal(t, "Device.A.1.B.", refs[1].Tr069Path)
}

func TestApplyInstanceSelectorsToRefs_EmptySelectors_NoOp(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "P1", Tr069Path: "Device.X.Y"},
	}
	err := applyInstanceSelectorsToRefs(refs, nil)
	require.NoError(t, err)
	assert.Equal(t, "Device.X.Y", refs[0].Tr069Path)
}

func TestApplyInstanceSelectorsToRefs_MismatchPropagates(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "P1", Tr069Path: "Device.IP.Interface.{i}.IPAddress"},
	}
	// path 单层 {i}，selectors 两个 → mismatch
	err := applyInstanceSelectorsToRefs(refs, map[string]string{"iα": "1", "iβ": "2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "P1") // 包含问题 ref 标识
}

// buildEntry_LST_WithSelectors 验证 LST 分支 param_refs 路径被替换。
func TestBuildEntry_LST_InstanceSelectors(t *testing.T) {
	cmdID := uuid.New()
	pID := uuid.New()
	sfID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_IFACE",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "ADDR", Tr069Path: "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "ADDR"},
	}
	stmt := Statement{
		CommandID:           &cmdID,
		OperationType:       "LST",
		SelectedSubFieldIDs: []uuid.UUID{sfID},
		InstanceSelectors:   map[string]string{"iα": "1", "iβ": "2"},
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "Device.IP.Interface.1.IPv4Address.2.IPAddress", refs[0].Tr069Path)
}

func TestBuildEntry_LST_BlankFinalInstanceUsesPartialPath(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_VALUE",
		OperationType: "LST",
		Params: []MMLParamRef{{
			ID: sfID, ParamCode: "VALUE",
			Tr069Path: "Device.A.{i}.B.{i}.Value",
		}},
	}
	subFields := []MMLCommandSubField{{
		ID: sfID, CommandID: cmdID, MMLCode: "VALUE",
	}}
	stmt := Statement{
		CommandID:           &cmdID,
		OperationType:       "LST",
		SelectedSubFieldIDs: []uuid.UUID{sfID},
		InstanceSelectors:   map[string]string{"i01": "1", "i02": ""},
	}

	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "Device.A.1.B.", refs[0].Tr069Path)
	assert.Equal(t, []string{"Device.A.{i}.B.{i}.Value"}, entry["selected_standard_paths"])

	payload, err := BuildTR069Params(
		entry["rpc_method"].(string),
		refs,
		nil,
		entry["operation_type"].(string),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"names":["Device.A.1.B."]}`, string(payload))
}

// buildEntry_MOD_WithSelectors 验证 MOD 分支 param_refs 路径被替换。
func TestBuildEntry_MOD_InstanceSelectors(t *testing.T) {
	cmdID := uuid.New()
	pID := uuid.New()
	sfID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_IFACE",
		OperationType: "MOD",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "ADDR", Tr069Path: "Device.IP.Interface.{i}.IPAddress"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "ADDR"},
	}
	stmt := Statement{
		CommandID:         &cmdID,
		OperationType:     "MOD",
		Values:            map[string]string{"ADDR": "10.0.0.1"},
		InstanceSelectors: map[string]string{"iα": "3"},
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, subFields)
	require.NoError(t, err)
	refs := entry["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	assert.Equal(t, "Device.IP.Interface.3.IPAddress", refs[0].Tr069Path)
}

// buildEntry_ADD_WithSelectors 验证 ADD 分支 targetObject 被替换。
func TestBuildEntry_ADD_InstanceSelectors(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADD_ADDR",
		OperationType: "ADD",
		TargetObject:  "Device.IP.Interface.{i}.IPv4Address.",
	}
	stmt := Statement{
		OperationType:     "ADD",
		Values:            map[string]string{"X": "v"},
		InstanceSelectors: map[string]string{"iα": "2"},
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.NoError(t, err)
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.IP.Interface.2.IPv4Address.", params["object_name"])
}

// buildEntry_RMV_WithSelectors 验证 RMV 分支 targetObject 被替换；末位实例号由
// RmvInstanceIndex 提供（用户决策 2026-05-20：RMV 保持单实例）。
func TestBuildEntry_RMV_InstanceSelectors(t *testing.T) {
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "RMV_ADDR",
		OperationType: "RMV",
		TargetObject:  "Device.IP.Interface.{i}.IPv4Address.",
	}
	idx := 5
	stmt := Statement{
		OperationType:     "RMV",
		InstanceSelectors: map[string]string{"iα": "2"}, // 上层 IP.Interface.2
		RmvInstanceIndex:  &idx,                         // 删除 IPv4Address.5
	}
	entry, err := buildStatementCommandEntry(stmt, cmd, nil)
	require.NoError(t, err)
	params := entry["parameters"].(map[string]interface{})
	assert.Equal(t, "Device.IP.Interface.2.IPv4Address.5.", params["object_name"])
}

// 端到端：StructuredExecuteRequest 含 selectors → ExecuteStructured → 落到 task.Commands。
func TestExecuteStructured_InstanceSelectorsEndToEnd(t *testing.T) {
	cmdID := uuid.New()
	pID := uuid.New()
	sfID := uuid.New()
	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST_MU_SLOT",
		LogicalCode:   "MU_SLOT",
		OperationType: "LST",
		Params: []MMLParamRef{
			{ID: sfID, ParamCode: "VER", Tr069Path: "Device.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sfID, CommandID: cmdID, ParamID: pID, MMLCode: "VER"},
	}
	cmdRepo := newFakeCommandRepo()
	cmdRepo.byID[cmdID] = cmd
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = subFields

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	creator := &fakeTaskCreator{}
	task, err := svc.ExecuteStructured(context.Background(), StructuredExecuteRequest{
		Statements: []StructuredStatement{
			{
				CommandID:     cmdID,
				OperationType: "LST",
				Paths:         []string{"Device.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion"},
				InstanceSelectors: map[string]string{
					"iα": "1",
					"iβ": "2",
				},
			},
		},
		DeviceSNs: []string{"DEV1"},
	}, creator)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Len(t, creator.captured.Commands, 1)
	refs := creator.captured.Commands[0]["param_refs"].([]MMLParamRef)
	require.Len(t, refs, 1)
	// 路径替换后流入 fanout
	assert.Equal(t, "Device.DeviceInfo.MU.1.Slot.2.3GPPSpecVersion", refs[0].Tr069Path)
}
