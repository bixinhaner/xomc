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
// console_structured_test.go — R-9.2 结构化适配层
//
// 覆盖：
//   - StructuredToStatement happy path（LST / MOD / RMV）
//   - unknown_paths 汇总 + 排序
//   - instance_selectors 直接报错（R-4 未实施）
//   - command_id 缺失 / 命令不存在
//   - ExecuteStructured 端到端：sub_fields path 解析 → 委托 ExecuteStatements
// ============================================================

// structuredFixture 提供共享的命令 + sub_fields 测试装置：
//
//	cmd 含 3 个 sub_field，每个绑定不同 standardPath
//	Path1 = "Device.IP.Address",   MMLCode=ADDR
//	Path2 = "Device.IP.Netmask",   MMLCode=MASK
//	Path3 = "Device.IP.Enable",    MMLCode=ENBL
type structuredFixture struct {
	cmd       *MMLCommand
	subFields []MMLCommandSubField
	pIDs      [3]uuid.UUID
	sfIDs     [3]uuid.UUID
}

func newStructuredFixture() structuredFixture {
	cmdID := uuid.New()
	var p, sf [3]uuid.UUID
	for i := range p {
		p[i] = uuid.New()
		sf[i] = uuid.New()
	}
	paths := []string{
		"Device.IP.Address",
		"Device.IP.Netmask",
		"Device.IP.Enable",
	}
	codes := []string{"ADDR", "MASK", "ENBL"}

	cmd := &MMLCommand{
		ID:            cmdID,
		CommandCode:   "MOD_IP",
		LogicalCode:   "IP",
		OperationType: "MOD",
		TargetObject:  "",
		// 生产 ID 语义（migration 000113）：
		//   MMLParamRef.ID == mml_command_sub_fields.id（来自 paramRefSelectExpr.csf.id）
		// 所以 cmd.Params[i].ID == sf[i].ID（同一个 csf.id），而 p[i]（standard_params.id）
		// 仅出现在 sub_field.ParamID 字段（soft alias of standard_path_id），不与
		// MMLParamRef.ID 共用空间。旧 fixture 用同一 p[i] 同时填两边，遮蔽了
		// buildPathToSubFieldIndex / buildLSTParamRefs 的 ID-mismatch 真 bug。
		Params: []MMLParamRef{
			{ID: sf[0], ParamCode: codes[0], Tr069Path: paths[0], ValueType: "string"},
			{ID: sf[1], ParamCode: codes[1], Tr069Path: paths[1], ValueType: "string"},
			{ID: sf[2], ParamCode: codes[2], Tr069Path: paths[2], ValueType: "boolean"},
		},
	}
	subFields := []MMLCommandSubField{
		{ID: sf[0], CommandID: cmdID, ParamID: p[0], MMLCode: codes[0]},
		{ID: sf[1], CommandID: cmdID, ParamID: p[1], MMLCode: codes[1]},
		{ID: sf[2], CommandID: cmdID, ParamID: p[2], MMLCode: codes[2]},
	}
	return structuredFixture{cmd: cmd, subFields: subFields, pIDs: p, sfIDs: sf}
}

// buildSvcWithFixture 把 fixture 安装到 fakeCommandRepo / fakeSubFieldRepo。
func (fx structuredFixture) install() *ConsoleService {
	cmdRepo := newFakeCommandRepo()
	cmdRepo.byID[fx.cmd.ID] = fx.cmd
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[fx.cmd.ID] = fx.subFields
	return NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
}

func TestStructuredToStatement_LST_PathsToSelectedSubFieldIDs(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "LST",
		Paths: []string{
			"Device.IP.Address",
			"Device.IP.Enable",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "LST", stmt.OperationType)
	assert.Equal(t, "IP", stmt.LogicalCode)
	require.NotNil(t, stmt.CommandID)
	assert.Equal(t, fx.cmd.ID, *stmt.CommandID)

	// Paths → SelectedSubFieldIDs；映射通过 sub_field.ParamID ↔ cmd.Params.Tr069Path
	require.Len(t, stmt.SelectedSubFieldIDs, 2)
	want := map[uuid.UUID]bool{fx.sfIDs[0]: true, fx.sfIDs[2]: true}
	for _, id := range stmt.SelectedSubFieldIDs {
		assert.True(t, want[id], "unexpected sf id %s", id)
	}
}

func TestStructuredToStatement_MOD_ValuesKeyRekeyed(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "MOD",
		Values: map[string]string{
			"Device.IP.Address": "192.168.1.1",
			"Device.IP.Enable":  "true",
		},
	})
	require.NoError(t, err)
	// values key 从 standardPath 反查 sub_field.MMLCode
	assert.Equal(t, "192.168.1.1", stmt.Values["ADDR"])
	assert.Equal(t, "true", stmt.Values["ENBL"])
	assert.Len(t, stmt.Values, 2)
}

func TestStructuredToStatement_RMV_SingleInstancePassThrough(t *testing.T) {
	// 用户决策 2026-05-20：RMV 保持单实例。
	fx := newStructuredFixture()
	svc := fx.install()
	idx := 7
	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "RMV",
		RmvInstance:   &idx,
	})
	require.NoError(t, err)
	require.NotNil(t, stmt.RmvInstanceIndex)
	assert.Equal(t, 7, *stmt.RmvInstanceIndex)
}

func TestStructuredToStatement_RMV_InstanceIndicesWireCompatibility(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()
	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:       fx.cmd.ID,
		OperationType:   "RMV",
		InstanceIndices: []int{3},
	})
	require.NoError(t, err)
	require.NotNil(t, stmt.RmvInstanceIndex)
	assert.Equal(t, 3, *stmt.RmvInstanceIndex)
}

func TestStructuredToStatement_RMV_RejectsMultipleInstanceIndices(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()
	_, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:       fx.cmd.ID,
		OperationType:   "RMV",
		InstanceIndices: []int{2, 3},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one instance index")
}

func TestStructuredToStatement_UnknownPaths_Aggregated(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	_, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "MOD",
		Paths: []string{
			"Device.IP.Address",    // 已知
			"Device.Unknown.PathA", // 未知
		},
		Values: map[string]string{
			"Device.IP.Netmask":    "255.255.255.0", // 已知
			"Device.Unknown.PathB": "v",             // 未知
			"Device.Unknown.PathA": "v",             // 与 Paths 重复未知 — 去重
		},
	})
	require.Error(t, err)
	var unk *ErrUnknownPaths
	require.True(t, errors.As(err, &unk),
		"should be *ErrUnknownPaths, got %T", err)
	// 去重 + 排序
	assert.Equal(t, []string{"Device.Unknown.PathA", "Device.Unknown.PathB"}, unk.Paths)
	assert.Equal(t, fx.cmd.ID, unk.CommandID)
}

func TestStructuredToStatement_InstanceSelectorsPassThrough(t *testing.T) {
	// R-4：adapter 直通 selectors，substitution 在 executor 中执行。
	fx := newStructuredFixture()
	svc := fx.install()

	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:         fx.cmd.ID,
		OperationType:     "LST",
		InstanceSelectors: map[string]string{"iα": "1", "iβ": "2"},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"iα": "1", "iβ": "2"}, stmt.InstanceSelectors)
}

func TestStructuredToStatement_CommandIDRequired(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	_, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		OperationType: "LST",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command_id required")
}

func TestStructuredToStatement_CommandNotFound(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	_, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     uuid.New(), // 不在 fixture
		OperationType: "LST",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotFound),
		"should wrap ErrCommandNotFound, got: %v", err)
}

func TestStructuredToStatement_LST_EmptyPathsAllowed(t *testing.T) {
	// LST 允许 Paths 为空（既有 executor 默认全选）；adapter 不报错。
	fx := newStructuredFixture()
	svc := fx.install()

	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "LST",
	})
	require.NoError(t, err)
	assert.Empty(t, stmt.SelectedSubFieldIDs)
}

func TestExecuteStructured_DelegatesToExecuteStatements(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	creator := &fakeTaskCreator{}
	task, err := svc.ExecuteStructured(context.Background(), StructuredExecuteRequest{
		Statements: []StructuredStatement{
			{
				CommandID:     fx.cmd.ID,
				OperationType: "MOD",
				Values: map[string]string{
					"Device.IP.Address": "10.0.0.1",
				},
			},
		},
		DeviceSNs: []string{"DEV1"},
		Creator:   "tester",
	}, creator)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NotNil(t, creator.captured)
	// #196：MOD with values → 2 commands（SPV 下发 + GPV 回读核实下发的 PATH）
	require.Len(t, creator.captured.Commands, 2)
	entry := creator.captured.Commands[0]
	assert.Equal(t, "SetParameterValues", entry["rpc_method"])
	params := entry["parameters"].(map[string]interface{})
	// 校验 values key 已经从 standardPath 翻译为 MMLCode
	assert.Equal(t, "10.0.0.1", params["ADDR"])
	// 第 2 条：回读 LST，compound_phase 标记，且只回读本次下发的 ADDR（不含未填的 MASK/ENBL）
	readback := creator.captured.Commands[1]
	assert.Equal(t, "GetParameterValues", readback["rpc_method"])
	assert.Equal(t, "lst_after_mod", readback["compound_phase"])
	rRefs := readback["param_refs"].([]MMLParamRef)
	require.Len(t, rRefs, 1)
	assert.Equal(t, "ADDR", rRefs[0].ParamCode)
}

// 2026-05-22 防回归：模拟 PgCommandRepository.GetByID 返回 cmd.Params=nil 的
// 生产场景，验证 attachParams (via SetCmdParamRepo) 能填上 Params 让 path 匹配成功。
//
// 历史 bug：ConsoleService 漏装 cmdParamRepo → cmd.Params 永远 nil →
// buildPathToSubFieldIndex 返空 → 全部 path 误报 R-9.2 unknown_paths。
func TestStructuredToStatement_CmdParamRepoEnrichesNilParams(t *testing.T) {
	fx := newStructuredFixture()
	// 模拟生产：commandRepo 返回的 cmd.Params 为 nil（PgCommandRepository.GetByID
	// 不查 params 列）。clone cmd 以避免修改 fixture 状态影响其他 case。
	cmdNoParams := *fx.cmd
	cmdNoParams.Params = nil

	cmdRepo := newFakeCommandRepo()
	cmdRepo.byID[fx.cmd.ID] = &cmdNoParams
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[fx.cmd.ID] = fx.subFields
	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	// 装上 cmdParamRepo enrichment（这是 modules.go 在生产应做的，旧版漏装）。
	// 注意：MMLParamRef.ID == sub_field.id（paramRefSelectExpr.csf.id），故用 sfIDs 而非 pIDs。
	svc.SetCmdParamRepo(&stubCmdParamRepo{refs: map[uuid.UUID][]MMLParamRef{
		fx.cmd.ID: {
			{ID: fx.sfIDs[0], ParamCode: "ADDR", Tr069Path: "Device.IP.Address", ValueType: "string"},
			{ID: fx.sfIDs[1], ParamCode: "MASK", Tr069Path: "Device.IP.Netmask", ValueType: "string"},
			{ID: fx.sfIDs[2], ParamCode: "ENBL", Tr069Path: "Device.IP.Enable", ValueType: "boolean"},
		},
	}})

	stmt, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "LST",
		Paths: []string{
			"Device.IP.Address",
			"Device.IP.Enable",
		},
	})
	require.NoError(t, err, "attachParams 应该让 path 命中而非误报 unknown_paths")
	require.Len(t, stmt.SelectedSubFieldIDs, 2)
}

// 2026-05-22 防回归：cmdParamRepo 未装配且 cmd.Params 为空时，attachParams 短路
// 不阻塞调用 — 但 buildPathToSubFieldIndex 拿不到映射会照旧返 unknown_paths。
// 此用例确认短路语义（保留旧部署退化兜底），不掩盖错误（仍报 R-9.2）。
func TestStructuredToStatement_NoCmdParamRepoFallsBackToUnknown(t *testing.T) {
	fx := newStructuredFixture()
	cmdNoParams := *fx.cmd
	cmdNoParams.Params = nil

	cmdRepo := newFakeCommandRepo()
	cmdRepo.byID[fx.cmd.ID] = &cmdNoParams
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[fx.cmd.ID] = fx.subFields
	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	// 不调 SetCmdParamRepo — 模拟旧部署 / 测试环境

	_, err := svc.StructuredToStatement(context.Background(), StructuredStatement{
		CommandID:     fx.cmd.ID,
		OperationType: "LST",
		Paths:         []string{"Device.IP.Address"},
	})
	require.Error(t, err)
	var unk *ErrUnknownPaths
	require.True(t, errors.As(err, &unk),
		"未装 cmdParamRepo 时仍走旧逻辑报 unknown_paths（不掩盖错误）")
}

func TestExecuteStructured_StatementErrorWrappedWithIndex(t *testing.T) {
	fx := newStructuredFixture()
	svc := fx.install()

	_, err := svc.ExecuteStructured(context.Background(), StructuredExecuteRequest{
		Statements: []StructuredStatement{
			{
				CommandID:     fx.cmd.ID,
				OperationType: "MOD",
				Values:        map[string]string{"Device.IP.Address": "10.0.0.1"}, // 合法
			},
			{
				CommandID:     fx.cmd.ID,
				OperationType: "MOD",
				Paths:         []string{"Device.Bogus"}, // 非法
			},
		},
		DeviceSNs: []string{"DEV1"},
	}, &fakeTaskCreator{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "statement[1]",
		"error should carry failing statement index")
	var unk *ErrUnknownPaths
	assert.True(t, errors.As(err, &unk),
		"inner error should be *ErrUnknownPaths")
}
