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

// =============================================================================
// T-Mml-Admin — admin List 读路径 + path autofill 派生单测
// =============================================================================

// fakeCommandLister 实现 CommandLister 接口；只关心 filter 透传是否正确。
type fakeCommandLister struct {
	lastFilter CommandFilter
	items      []MMLCommand
}

func (f *fakeCommandLister) List(_ context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	f.lastFilter = filter
	return model.NewListResponse(f.items, int64(len(f.items)), 1, 20), nil
}

func (f *fakeCommandLister) GetByID(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			return &f.items[i], nil
		}
	}
	return nil, ErrCommandNotFound
}

// fakeStandardParamRepo 桩 standard_params 仓储。autofill 派生测试用。
type fakeStandardParamRepo struct {
	byID map[uuid.UUID]*StandardParamView
}

func (f *fakeStandardParamRepo) List(_ context.Context, _ StandardParamFilter) (*model.ListResponse[StandardParamView], error) {
	items := make([]StandardParamView, 0, len(f.byID))
	for _, v := range f.byID {
		items = append(items, *v)
	}
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}

func (f *fakeStandardParamRepo) GetByID(_ context.Context, id uuid.UUID) (*StandardParamView, error) {
	if v, ok := f.byID[id]; ok {
		return v, nil
	}
	return nil, ErrCommandNotFound
}

// 构造 service：注入空 repos，然后 SetCommandReader / SetStandardParamRepo。
func newAdminServiceForList() (*AdminService, *fakeCommandLister, *fakeStandardParamRepo, *mockSubFieldRepo) {
	groupRepo := &mockGroupRepo{groups: map[uuid.UUID]*CommandGroup{}}
	cmdRepo := &mockAdminCmdRepo{commands: map[uuid.UUID]*MMLCommand{}}
	sfRepo := &mockSubFieldRepo{subFields: map[uuid.UUID]*MMLCommandSubField{}}
	svc := NewAdminService(groupRepo, cmdRepo, sfRepo, nil, zap.NewNop())
	cmdLister := &fakeCommandLister{}
	spRepo := &fakeStandardParamRepo{byID: map[uuid.UUID]*StandardParamView{}}
	svc.SetCommandReader(cmdLister)
	svc.SetStandardParamRepo(spRepo)
	return svc, cmdLister, spRepo, sfRepo
}

// V1 — ListCommands 默认追加 source='standard' 过滤
func TestListCommands_DefaultsToStandardSource(t *testing.T) {
	svc, lister, _, _ := newAdminServiceForList()
	gid := uuid.New()
	_, err := svc.ListCommands(context.Background(), ListCommandsReq{GroupID: &gid})
	require.NoError(t, err)
	require.NotNil(t, lister.lastFilter.Source)
	assert.Equal(t, SourceStandard, *lister.lastFilter.Source)
	require.NotNil(t, lister.lastFilter.GroupID)
	assert.Equal(t, gid, *lister.lastFilter.GroupID)
}

// V2 — source='all' 显式不过滤
func TestListCommands_SourceAll_NoFilter(t *testing.T) {
	svc, lister, _, _ := newAdminServiceForList()
	_, err := svc.ListCommands(context.Background(), ListCommandsReq{Source: "all"})
	require.NoError(t, err)
	assert.Nil(t, lister.lastFilter.Source)
}

// V3 — ListStandardParams 未装配 repo 时返 ErrAdminReaderNotConfigured
func TestListStandardParams_NotConfigured(t *testing.T) {
	groupRepo := &mockGroupRepo{groups: map[uuid.UUID]*CommandGroup{}}
	cmdRepo := &mockAdminCmdRepo{commands: map[uuid.UUID]*MMLCommand{}}
	sfRepo := &mockSubFieldRepo{}
	svc := NewAdminService(groupRepo, cmdRepo, sfRepo, nil, zap.NewNop())
	_, err := svc.ListStandardParams(context.Background(), StandardParamFilter{})
	assert.ErrorIs(t, err, ErrAdminReaderNotConfigured)
}

// V4 — BatchCreateSubFields 按 standard_params 元数据派生默认 mml_code / label
func TestBatchCreateSubFields_DerivesAutofillFromStandardParams(t *testing.T) {
	svc, _, sp, sf := newAdminServiceForList()
	cmdID := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()
	sp.byID[p1] = &StandardParamView{
		ID: p1, StandardPath: "Device.DeviceInfo.UserLabel",
		EntryType: "parameter", Access: "READ_WRITE", DataType: "string",
		Description: "用户友好名",
	}
	sp.byID[p2] = &StandardParamView{
		ID: p2, StandardPath: "Device.X.3GPPSpecVersion",
		EntryType: "parameter", Access: "READ_ONLY", DataType: "string",
	}
	out, err := svc.BatchCreateSubFields(context.Background(), cmdID, BatchCreateSubFieldsReq{
		StandardPathIDs: []uuid.UUID{p1, p2},
	})
	require.NoError(t, err)
	require.Len(t, out, 2)

	// p1: zh 用 description 兜底
	assert.Equal(t, "USER_LABEL", out[0].MMLCode)
	assert.Equal(t, "用户友好名", out[0].LabelI18n["zh-CN"])
	assert.Equal(t, "User Label", out[0].LabelI18n["en-US"])
	assert.True(t, out[0].DefaultSelected)
	assert.False(t, out[0].IsRequired)
	assert.Equal(t, 1, out[0].SortOrder)

	// p2: description 空 → zh 用 mml_code 兜底
	assert.Equal(t, "3GPPSPEC_VERSION", out[1].MMLCode)
	assert.Equal(t, "3GPPSPEC_VERSION", out[1].LabelI18n["zh-CN"])
	assert.Equal(t, "3GPPSpec Version", out[1].LabelI18n["en-US"])
	assert.Equal(t, 2, out[1].SortOrder)

	// 持久化（mockSubFieldRepo.BatchCreate）写到 sf.subFields
	assert.Len(t, sf.subFields, 2)
}

// V5 — derivePathLeafCode / humanizePathLeaf 单元覆盖
func TestDerivePathLeafCode_Variations(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Device.DeviceInfo.UserLabel", "USER_LABEL"},
		{"Device.X.AntennaAzimuth", "ANTENNA_AZIMUTH"},
		{"Device.DeviceInfo.1588_Status", "1588_STATUS"},
		{"foo.bar.simple_path", "SIMPLE_PATH"},
		{"NoDot", "NO_DOT"},
		{"", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, derivePathLeafCode(c.in), "input=%q", c.in)
	}
}

func TestHumanizePathLeaf_Variations(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Device.DeviceInfo.UserLabel", "User Label"},
		{"Device.X.AntennaAzimuth", "Antenna Azimuth"},
		{"Device.DeviceInfo.first_use_date", "first use date"},
		{"NoDot", "No Dot"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, humanizePathLeaf(c.in), "input=%q", c.in)
	}
}

// V6 — BatchCreateSubFields 空入参 → 错误
func TestBatchCreateSubFields_EmptyPathList_Errors(t *testing.T) {
	svc, _, _, _ := newAdminServiceForList()
	_, err := svc.BatchCreateSubFields(context.Background(), uuid.New(), BatchCreateSubFieldsReq{})
	assert.Error(t, err)
}
