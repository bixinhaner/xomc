package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// console_service_test.go — T-0123-P1
//
// 覆盖：
//   - assembleHierarchy 树形组装（path 分层 + 排序）
//   - parentLTreePath / sortByDepthThenLex helper 边界
//   - ConsoleService.RenderMML 透传调用 + logical_code 派生兜底
//   - ConsoleService.ParseMML 调用 parser + 注入 CommandLookup
//   - ConsoleService.LookupByLogicalCode 双策略查找（op_ 前缀 + 退化 logical_code）
//   - GetCommandSubFields lang 派生
//
// 不依赖 PG（用 mock repo）。
// ============================================================

// ============================================================
// mock repos
// ============================================================

type fakeGroupTreeRepo struct {
	tree []GroupTreeNode
	err  error
}

func (f *fakeGroupTreeRepo) BuildTree(_ context.Context, _, _ string) ([]GroupTreeNode, error) {
	return f.tree, f.err
}

type fakeFlatGroupTreeRepo struct {
	groups []FlatGroup
	err    error
}

func (f *fakeFlatGroupTreeRepo) BuildFlatTree(context.Context) ([]FlatGroup, error) {
	return f.groups, f.err
}

type fakeSubFieldRepo struct {
	byCommandList     map[uuid.UUID][]MMLCommandSubField
	byCommandEnriched map[uuid.UUID][]MMLCommandSubFieldEnriched
	listErr           error
}

func newFakeSubFieldRepo() *fakeSubFieldRepo {
	return &fakeSubFieldRepo{
		byCommandList:     map[uuid.UUID][]MMLCommandSubField{},
		byCommandEnriched: map[uuid.UUID][]MMLCommandSubFieldEnriched{},
	}
}

func (f *fakeSubFieldRepo) Create(_ context.Context, _ *MMLCommandSubField) error { return nil }
func (f *fakeSubFieldRepo) Update(_ context.Context, _ *MMLCommandSubField) error { return nil }
func (f *fakeSubFieldRepo) Delete(_ context.Context, _ uuid.UUID) error           { return nil }
func (f *fakeSubFieldRepo) GetByID(_ context.Context, _ uuid.UUID) (*MMLCommandSubField, error) {
	return nil, nil
}
func (f *fakeSubFieldRepo) ListByCommand(_ context.Context, c uuid.UUID) ([]MMLCommandSubField, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.byCommandList[c], nil
}
func (f *fakeSubFieldRepo) ListEnrichedByCommand(_ context.Context, c uuid.UUID, _ *uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.byCommandEnriched[c], nil
}
func (f *fakeSubFieldRepo) CountByParam(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeSubFieldRepo) MarkUnsupportedByStandardPath(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (f *fakeSubFieldRepo) ListAdminByCommand(_ context.Context, _ uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	return nil, nil
}
func (f *fakeSubFieldRepo) BatchCreate(_ context.Context, _ []*MMLCommandSubField) error {
	return nil
}

type fakeCommandRepo struct {
	byID   map[uuid.UUID]*MMLCommand
	byCode map[string]*MMLCommand
	getErr error
}

func newFakeCommandRepo() *fakeCommandRepo {
	return &fakeCommandRepo{
		byID:   map[uuid.UUID]*MMLCommand{},
		byCode: map[string]*MMLCommand{},
	}
}

func (f *fakeCommandRepo) List(_ context.Context, _ CommandFilter) (*model.ListResponse[MMLCommand], error) {
	return &model.ListResponse[MMLCommand]{}, nil
}
func (f *fakeCommandRepo) GetByID(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.byID[id]
	if !ok {
		return nil, ErrCommandNotFound
	}
	return c, nil
}
func (f *fakeCommandRepo) GetByCode(_ context.Context, code string) (*MMLCommand, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.byCode[code]
	if !ok {
		return nil, ErrCommandNotFound
	}
	return c, nil
}
func (f *fakeCommandRepo) ListByGroupID(_ context.Context, _ uuid.UUID) ([]MMLCommand, error) {
	return nil, nil
}

func (f *fakeCommandRepo) addCommand(cmd *MMLCommand) {
	f.byID[cmd.ID] = cmd
	f.byCode[cmd.CommandCode] = cmd
}

// ============================================================
// assembleHierarchy 树形组装测试
// ============================================================

func TestParentLTreePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"BSC_CONFIGURATION", ""},
		{"BSC_CONFIGURATION.BASIC_INFO", "BSC_CONFIGURATION"},
		{"BSC_CONFIGURATION.BTS.SUB", "BSC_CONFIGURATION.BTS"},
		{"", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, parentLTreePath(c.in), "in=%q", c.in)
	}
}

func TestSortByDepthThenLex(t *testing.T) {
	paths := []string{"B.X", "A", "A.Y", "B", "A.Z"}
	sortByDepthThenLex(paths)
	assert.Equal(t, []string{"A", "B", "A.Y", "A.Z", "B.X"}, paths)
}

func TestAssembleHierarchy_TwoLevel(t *testing.T) {
	g1ID := uuid.New()
	g2ID := uuid.New()
	g3ID := uuid.New()
	groups := map[uuid.UUID]*GroupTreeNode{
		g1ID: {ID: g1ID, GroupCode: "ROOT", Path: "ROOT", DisplayOrder: 1, Commands: []GroupTreeCommand{}},
		g2ID: {ID: g2ID, GroupCode: "ROOT_A", Path: "ROOT.A", DisplayOrder: 1, Commands: []GroupTreeCommand{}},
		g3ID: {ID: g3ID, GroupCode: "ROOT_B", Path: "ROOT.B", DisplayOrder: 2, Commands: []GroupTreeCommand{}},
	}
	pathMap := map[string]uuid.UUID{
		"ROOT":   g1ID,
		"ROOT.A": g2ID,
		"ROOT.B": g3ID,
	}
	nodes := assembleHierarchy(groups, pathMap)
	require.Len(t, nodes, 1)
	assert.Equal(t, "ROOT", nodes[0].GroupCode)
	require.Len(t, nodes[0].Children, 2)
	// A display_order=1, B display_order=2 — order respected
	assert.Equal(t, "ROOT_A", nodes[0].Children[0].GroupCode)
	assert.Equal(t, "ROOT_B", nodes[0].Children[1].GroupCode)
}

func TestAssembleHierarchy_PartialSubtree_OrphanNodes(t *testing.T) {
	// 部分子树（缺父节点）— 孤儿节点应作顶级返回
	g2ID := uuid.New()
	groups := map[uuid.UUID]*GroupTreeNode{
		g2ID: {ID: g2ID, GroupCode: "ORPHAN", Path: "MISSING_PARENT.ORPHAN", DisplayOrder: 1},
	}
	pathMap := map[string]uuid.UUID{
		"MISSING_PARENT.ORPHAN": g2ID,
	}
	nodes := assembleHierarchy(groups, pathMap)
	require.Len(t, nodes, 1)
	assert.Equal(t, "ORPHAN", nodes[0].GroupCode)
}

func TestAssembleHierarchy_DisplayOrderRespected(t *testing.T) {
	g1ID := uuid.New()
	g2ID := uuid.New()
	g3ID := uuid.New()
	groups := map[uuid.UUID]*GroupTreeNode{
		g1ID: {ID: g1ID, GroupCode: "C", Path: "C", DisplayOrder: 3},
		g2ID: {ID: g2ID, GroupCode: "A", Path: "A", DisplayOrder: 1},
		g3ID: {ID: g3ID, GroupCode: "B", Path: "B", DisplayOrder: 2},
	}
	pathMap := map[string]uuid.UUID{"A": g2ID, "B": g3ID, "C": g1ID}
	nodes := assembleHierarchy(groups, pathMap)
	require.Len(t, nodes, 3)
	assert.Equal(t, "A", nodes[0].GroupCode)
	assert.Equal(t, "B", nodes[1].GroupCode)
	assert.Equal(t, "C", nodes[2].GroupCode)
}

func TestBuildGroupTreeFilteredByDeviceUsesSupportedPaths(t *testing.T) {
	groupID := uuid.New()
	commandA := GroupTreeCommand{ID: uuid.New(), OperationType: "LST"}
	commandA.SetTargetPathsRaw([]byte(`["Device.A"]`))
	commandB := GroupTreeCommand{ID: uuid.New(), OperationType: "LST"}
	commandB.SetTargetPathsRaw([]byte(`["Device.B"]`))

	pmID := uuid.New()
	svc := NewConsoleService(&fakeGroupTreeRepo{tree: []GroupTreeNode{
		{ID: groupID, GroupCode: "ROOT", Commands: []GroupTreeCommand{commandA, commandB}},
	}}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	svc.SetParamModelByDeviceResolver(func(context.Context, string) (*uuid.UUID, error) {
		return &pmID, nil
	})
	svc.SetParamModelPathsResolver(func(context.Context, uuid.UUID) (map[string]struct{}, error) {
		return map[string]struct{}{"Device.A": {}}, nil
	})

	got, err := svc.BuildGroupTreeFilteredByDevice(context.Background(), "", "zh-CN", "SN-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Commands, 1)
	assert.Equal(t, commandA.ID, got[0].Commands[0].ID)
}

func TestBuildFlatGroupTreeFilteredByDeviceUsesSupportedPaths(t *testing.T) {
	commandA := FlatCommand{ID: uuid.New(), Name: "LST A", ObjectPath: []string{"Device.A"}}
	commandB := FlatCommand{ID: uuid.New(), Name: "LST B", ObjectPath: []string{"Device.B"}}
	pmID := uuid.New()
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	svc.SetFlatTreeRepo(&fakeFlatGroupTreeRepo{groups: []FlatGroup{{
		Code:     "SA",
		Name:     "设备信息参数管理",
		Commands: []FlatCommand{commandA, commandB},
	}}})
	svc.SetParamModelByDeviceResolver(func(context.Context, string) (*uuid.UUID, error) {
		return &pmID, nil
	})
	svc.SetParamModelPathsResolver(func(context.Context, uuid.UUID) (map[string]struct{}, error) {
		return map[string]struct{}{"Device.A": {}}, nil
	})

	got, err := svc.BuildFlatGroupTreeFiltered(context.Background(), "", "SN-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Commands, 1)
	assert.Equal(t, commandA.ID, got[0].Commands[0].ID)
}

// ============================================================
// ConsoleService.RenderMML
// ============================================================

func TestConsoleService_RenderMML_LST_Success(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()

	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST DEVICE_INFO",
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
	})

	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = []MMLCommandSubField{
		{ID: sfID, MMLCode: "LTE_GSM_MODEL_NAME", SortOrder: 1},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	got, err := svc.RenderMML(context.Background(), RenderRequest{
		CommandID:           cmdID,
		OperationType:       "LST",
		SelectedSubFieldIDs: []uuid.UUID{sfID},
	})
	require.NoError(t, err)
	assert.Equal(t, "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME}", got)
}

func TestConsoleService_RenderMML_LogicalCodeFallback(t *testing.T) {
	cmdID := uuid.New()
	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST DEVICE_INFO",
		LogicalCode:   "", // 空 logical_code → 派生
		OperationType: "LST",
	})

	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), cmdRepo, nil)
	got, err := svc.RenderMML(context.Background(), RenderRequest{
		CommandID:     cmdID,
		OperationType: "LST",
	})
	require.NoError(t, err)
	// command_code "LST DEVICE_INFO" 去 op 前缀 "LST " → "DEVICE_INFO"
	assert.Equal(t, "LST DEVICE_INFO", got)
}

func TestConsoleService_RenderMML_CommandNotFound(t *testing.T) {
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	_, err := svc.RenderMML(context.Background(), RenderRequest{
		CommandID:     uuid.New(),
		OperationType: "LST",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotFound))
}

// ============================================================
// ConsoleService.ParseMML（透传 + lookup 注入）
// ============================================================

func TestConsoleService_ParseMML_WithLookup_ResolvesCommandID(t *testing.T) {
	cmdID := uuid.New()
	sfID := uuid.New()

	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            cmdID,
		CommandCode:   "LST DEVICE_INFO",
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
	})

	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandList[cmdID] = []MMLCommandSubField{
		{ID: sfID, MMLCode: "LTE_GSM_MODEL_NAME", SortOrder: 1},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, cmdRepo, nil)
	resp, err := svc.ParseMML(context.Background(), ParseRequest{
		MMLString: "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME};",
	})
	require.NoError(t, err)
	require.Empty(t, resp.ParseErrors)
	require.Len(t, resp.Statements, 1)
	require.NotNil(t, resp.Statements[0].CommandID)
	assert.Equal(t, cmdID, *resp.Statements[0].CommandID)
	assert.Equal(t, []uuid.UUID{sfID}, resp.Statements[0].SelectedSubFieldIDs)
}

func TestConsoleService_ParseMML_UnknownCommand_AccumulatesError(t *testing.T) {
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	resp, err := svc.ParseMML(context.Background(), ParseRequest{
		MMLString: "LST UNKNOWN_CMD:lstId={X};",
	})
	require.NoError(t, err)
	require.Len(t, resp.ParseErrors, 1)
	assert.Contains(t, resp.ParseErrors[0].Reason, "not found")
}

// ============================================================
// LookupByLogicalCode 双策略查找
// ============================================================

func TestLookupByLogicalCode_OpPrefixHit(t *testing.T) {
	cmdRepo := newFakeCommandRepo()
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "MOD DEVICE_INFO",
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
	}
	cmdRepo.addCommand(cmd)

	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), cmdRepo, nil)
	got, _, err := svc.LookupByLogicalCode(context.Background(), "MOD", "DEVICE_INFO")
	require.NoError(t, err)
	assert.Equal(t, cmd.ID, got.ID)
}

func TestLookupByLogicalCode_FallbackToBareCode(t *testing.T) {
	cmdRepo := newFakeCommandRepo()
	cmd := &MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "ADMIN_CUSTOM", // 没有 op 前缀（admin 创建的）
		LogicalCode:   "ADMIN_CUSTOM",
		OperationType: "LST",
	}
	cmdRepo.addCommand(cmd)

	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), cmdRepo, nil)
	got, _, err := svc.LookupByLogicalCode(context.Background(), "LST", "ADMIN_CUSTOM")
	require.NoError(t, err)
	assert.Equal(t, cmd.ID, got.ID)
}

func TestLookupByLogicalCode_OpMismatch_ReturnsNotFound(t *testing.T) {
	cmdRepo := newFakeCommandRepo()
	cmdRepo.addCommand(&MMLCommand{
		ID:            uuid.New(),
		CommandCode:   "LST DEVICE_INFO",
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
	})

	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), cmdRepo, nil)
	// 请求 MOD，但只有 LST 命中
	_, _, err := svc.LookupByLogicalCode(context.Background(), "MOD", "DEVICE_INFO")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotFound))
}

func TestLookupByLogicalCode_NotFound(t *testing.T) {
	svc := NewConsoleService(&fakeGroupTreeRepo{}, newFakeSubFieldRepo(), newFakeCommandRepo(), nil)
	_, _, err := svc.LookupByLogicalCode(context.Background(), "LST", "DOES_NOT_EXIST")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotFound))
}

// ============================================================
// GetCommandSubFields lang 派生
// ============================================================

func TestGetCommandSubFields_LangPicksZhCN(t *testing.T) {
	cmdID := uuid.New()
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandEnriched[cmdID] = []MMLCommandSubFieldEnriched{
		{
			MMLCommandSubField: MMLCommandSubField{
				ID:        uuid.New(),
				CommandID: cmdID,
				MMLCode:   "LTE_GSM_MODEL_NAME",
				LabelI18n: map[string]string{"zh-CN": "型号", "en-US": "Model Name"},
				SortOrder: 1,
			},
			Tr069Path:          "Device.X",
			ValueType:          "string",
			AccessType:         "READ_ONLY",
			ConstraintTextI18n: map[string]string{"zh-CN": "只读字符串", "en-US": "Read-only string"},
		},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, newFakeCommandRepo(), nil)
	got, err := svc.GetCommandSubFields(context.Background(), cmdID, "", "", "zh-CN")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "型号", got[0].Label)
	assert.Equal(t, "只读字符串", got[0].ConstraintText)
}

func TestGetCommandSubFields_LangFallbackToEn(t *testing.T) {
	cmdID := uuid.New()
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandEnriched[cmdID] = []MMLCommandSubFieldEnriched{
		{
			MMLCommandSubField: MMLCommandSubField{
				ID:        uuid.New(),
				CommandID: cmdID,
				MMLCode:   "X",
				LabelI18n: map[string]string{"en-US": "Only EN"}, // 缺 zh-CN
				SortOrder: 1,
			},
		},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, newFakeCommandRepo(), nil)
	got, err := svc.GetCommandSubFields(context.Background(), cmdID, "", "", "zh-CN")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Only EN", got[0].Label) // fallback
}

func TestGetCommandSubFields_DeviceResolverErrorFailsClosed(t *testing.T) {
	cmdID := uuid.New()
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandEnriched[cmdID] = []MMLCommandSubFieldEnriched{
		{
			MMLCommandSubField: MMLCommandSubField{ID: uuid.New(), CommandID: cmdID, MMLCode: "A"},
			Tr069Path:          "Device.A",
		},
	}

	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, newFakeCommandRepo(), nil)
	svc.SetParamModelByDeviceResolver(func(context.Context, string) (*uuid.UUID, error) {
		return nil, errors.New("redis unavailable")
	})

	got, err := svc.GetCommandSubFields(context.Background(), cmdID, "SN-1", "", "zh-CN")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "resolve param_model by device")
}

func TestGetCommandSubFields_DeviceReturnsSupportedMMLIntersection(t *testing.T) {
	cmdID := uuid.New()
	sfRepo := newFakeSubFieldRepo()
	sfRepo.byCommandEnriched[cmdID] = []MMLCommandSubFieldEnriched{
		{
			MMLCommandSubField: MMLCommandSubField{ID: uuid.New(), CommandID: cmdID, MMLCode: "A"},
			Tr069Path:          "Device.A",
		},
		{
			MMLCommandSubField: MMLCommandSubField{ID: uuid.New(), CommandID: cmdID, MMLCode: "B"},
			Tr069Path:          "Device.B",
		},
	}
	pmID := uuid.New()
	svc := NewConsoleService(&fakeGroupTreeRepo{}, sfRepo, newFakeCommandRepo(), nil)
	svc.SetParamModelByDeviceResolver(func(context.Context, string) (*uuid.UUID, error) {
		return &pmID, nil
	})
	svc.SetParamModelPathsResolver(func(context.Context, uuid.UUID) (map[string]struct{}, error) {
		return map[string]struct{}{"Device.A": {}, "Device.C": {}}, nil
	})

	got, err := svc.GetCommandSubFields(context.Background(), cmdID, "SN-1", "", "zh-CN")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Device.A", got[0].Tr069Path)
}

// ============================================================
// buildDisplayName / strOrEmpty / pickI18n 小 helper
// ============================================================

func TestBuildDisplayName(t *testing.T) {
	// 业务化命名（T-0123 v3）：verb 中文/英文 + logicalName
	assert.Equal(t, "查询 设备信息",
		buildDisplayName("设备信息", "LST", "DEVICE_INFO", "zh-CN"))
	assert.Equal(t, "Query Device Info",
		buildDisplayName("Device Info", "LST", "DEVICE_INFO", "en-US"))
	// logicalName 空时 fallback 到 logicalCode
	assert.Equal(t, "查询 DEVICE_INFO",
		buildDisplayName("", "LST", "DEVICE_INFO", "zh-CN"))
	// 未知 op 时保留原值
	assert.Equal(t, "FOO BAR",
		buildDisplayName("BAR", "FOO", "BAR_CODE", "zh-CN"))
}

func TestDeriveLogicalCodeFromCommandCode(t *testing.T) {
	// command_code 是 "<OP> <LOGICAL>" 空格分隔 → 去 "OP " 前缀
	assert.Equal(t, "DEVICE_INFO",
		deriveLogicalCodeFromCommandCode("LST DEVICE_INFO", "LST"))
	// op 不匹配前缀但有空格 → 退化取第二段
	assert.Equal(t, "BAR",
		deriveLogicalCodeFromCommandCode("FOO BAR", "LST"))
	// 无空格（admin 创建的裸 command_code）→ 原样返回
	assert.Equal(t, "ADMIN_CUSTOM",
		deriveLogicalCodeFromCommandCode("ADMIN_CUSTOM", "LST"))
}

func TestPickI18n_LangHit(t *testing.T) {
	m := map[string]string{"zh-CN": "中", "en-US": "EN"}
	assert.Equal(t, "中", pickI18n(m, "zh-CN", "", "", ""))
	assert.Equal(t, "EN", pickI18n(m, "en-US", "", "", ""))
}

func TestPickI18n_LongKeyOnly(t *testing.T) {
	// issue #67 §5：i18n 键已 seed/000039 统一为长码（短键 zh/en 迁移为 zh-CN/en-US），
	// pickI18n 删除短/长兼容分支，只认长码。
	longKey := map[string]string{"zh-CN": "中长", "en-US": "ENLong"}
	assert.Equal(t, "ENLong", pickI18n(longKey, "en-US", "", "", ""))
	assert.Equal(t, "中长", pickI18n(longKey, "zh-CN", "", "", ""))

	// 跨语言族 fallback：请求语言缺失 → zh-CN 优先（系统主语言），再 en-US。
	onlyEn := map[string]string{"en-US": "ENOnly"}
	assert.Equal(t, "ENOnly", pickI18n(onlyEn, "zh-CN", "", "", ""), "zh-CN 无 zh-CN 键时回退 en-US")
	onlyZh := map[string]string{"zh-CN": "ZhOnly"}
	assert.Equal(t, "ZhOnly", pickI18n(onlyZh, "en-US", "", "", ""), "en-US 无 en-US 键时回退 zh-CN")
}

func TestPickI18n_NoChineseResidueForEnglish(t *testing.T) {
	// 英文 locale 命中 en-US 时必须返回英文，绝不漏出中文（issue #67 §6 验收要点）。
	m := map[string]string{"zh-CN": "查询设备信息", "en-US": "Query Device Info"}
	got := pickI18n(m, "en-US", "", "", "")
	assert.Equal(t, "Query Device Info", got)
	assert.False(t, containsHan(got), "英文 locale 的 DisplayName 不应含中文残留")
}

// containsHan 判断字符串是否含 CJK 统一表意文字（中文残留检测）。
func containsHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

func TestPickI18n_FallbackChain(t *testing.T) {
	// 完全未命中 → 走 fallback chain
	assert.Equal(t, "fallback_zh",
		pickI18n(nil, "fr-FR", "fallback_zh", "fallback_en", "code"))
	assert.Equal(t, "fallback_en",
		pickI18n(nil, "fr-FR", "", "fallback_en", "code"))
	assert.Equal(t, "code",
		pickI18n(nil, "fr-FR", "", "", "code"))
}
