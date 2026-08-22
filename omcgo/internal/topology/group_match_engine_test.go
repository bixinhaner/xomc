package topology

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/event"
)

// fakeLister 是 group_match_engine 单测专用的 DeviceLister，
// 直接返回预设的设备切片，无需碰 PG。
type fakeLister struct {
	devices       []DeviceForMatch
	err           error
	requestedFrom *uuid.UUID
}

func (f *fakeLister) ListAllForRuleEval(_ context.Context) ([]DeviceForMatch, error) {
	return f.devices, f.err
}

func (f *fakeLister) ListForRuleEval(_ context.Context, sourceGroupID uuid.UUID) ([]DeviceForMatch, error) {
	f.requestedFrom = &sourceGroupID
	return f.devices, f.err
}

// GetByID 在 devices 切片里线性扫，无命中返 (nil, nil)。供 device.attributes.changed
// 单设备订阅路径的单测覆盖。
func (f *fakeLister) GetByID(_ context.Context, deviceID uuid.UUID) (*DeviceForMatch, error) {
	if f.err != nil {
		return nil, f.err
	}
	for i := range f.devices {
		if f.devices[i].ID == deviceID {
			return &f.devices[i], nil
		}
	}
	return nil, nil
}

func newEngineWithMocks(t *testing.T) (*GroupMatchEngine, *MockDeviceGroupRepository, *fakeLister, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	repo := NewMockDeviceGroupRepository(ctrl)
	lister := &fakeLister{}
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))
	engine := NewGroupMatchEngine(matcher, lister, repo, zaptest.NewLogger(t))
	return engine, repo, lister, ctrl
}

type invalidatingDeviceGroupRepo struct {
	*MockDeviceGroupRepository
	invalidateCalls int
}

func (r *invalidatingDeviceGroupRepo) InvalidateDeviceGroupCounts() {
	r.invalidateCalls++
}

// ── MatchGroup ──────────────────────────────────────────────────────────────

// L2 + 有匹配规则 → 只读取源组，命中设备按源组条件原子移动。
func TestGroupMatchEngine_MatchGroup_L2_AssignsMatchedDevices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := NewMockDeviceGroupRepository(ctrl)
	repo := &invalidatingDeviceGroupRepo{MockDeviceGroupRepository: repoMock}
	lister := &fakeLister{}
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))
	engine := NewGroupMatchEngine(matcher, lister, repo, zaptest.NewLogger(t))

	groupID := uuid.New()
	sourceGroupID := uuid.New()
	group := &DeviceGroup{
		ID: groupID, Level: 2, Name: "Beijing-L2",
		SourceGroupID: &sourceGroupID,
		MatchingMode:  MatchingModeDeviceName,
		NameRuleList:  []NameRule{{Condition: "contain", Value: "BJ"}},
	}
	matchedID := uuid.New()
	missedID := uuid.New()
	lister.devices = []DeviceForMatch{
		{ID: matchedID, Name: "BJ-SITE-01"},
		{ID: missedID, Name: "SH-SITE-99"},
	}

	repoMock.EXPECT().GetByID(gomock.Any(), groupID).Return(group, nil)
	repoMock.EXPECT().GetByID(gomock.Any(), sourceGroupID).Return(&DeviceGroup{ID: sourceGroupID, Level: 2}, nil)
	// 关键契约：matched 设备走带源组条件的原子移动，missed 不调。
	repoMock.EXPECT().MoveDeviceAutoMatched(gomock.Any(), sourceGroupID, groupID, matchedID).Return(int64(1), nil)

	require.NoError(t, engine.MatchGroup(context.Background(), groupID))
	require.NotNil(t, lister.requestedFrom)
	assert.Equal(t, sourceGroupID, *lister.requestedFrom)
	assert.Equal(t, 1, repo.invalidateCalls)
}

func TestGroupMatchEngine_MatchGroup_NoSource_NoOp(t *testing.T) {
	engine, repo, _, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	groupID := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), groupID).Return(&DeviceGroup{
		ID: groupID, Level: 2, Name: "legacy-rule-without-source",
		MatchingMode: MatchingModeTAC,
		TACList:      []int{1},
	}, nil)

	require.NoError(t, engine.MatchGroup(context.Background(), groupID))
}

func TestGroupMatchEngine_MatchGroup_SourceIsNoLongerL2_NoOp(t *testing.T) {
	engine, repo, _, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	targetID := uuid.New()
	sourceID := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), targetID).Return(&DeviceGroup{
		ID: targetID, Level: 2, SourceGroupID: &sourceID,
		MatchingMode: MatchingModeTAC, TACList: []int{1},
	}, nil)
	repo.EXPECT().GetByID(gomock.Any(), sourceID).Return(&DeviceGroup{ID: sourceID, Level: 1}, nil)

	require.NoError(t, engine.MatchGroup(context.Background(), targetID))
}

// L1 分组 → no-op，不查 lister、不写入。
func TestGroupMatchEngine_MatchGroup_L1_NoOp(t *testing.T) {
	engine, repo, lister, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	groupID := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), groupID).Return(&DeviceGroup{
		ID: groupID, Level: 1, Name: "Root-L1",
		MatchingMode: MatchingModeDeviceName,
		NameRuleList: []NameRule{{Condition: "contain", Value: "X"}},
	}, nil)
	// 不应调 lister / MoveDeviceAutoMatched —— 无 EXPECT 即不可调。
	lister.devices = []DeviceForMatch{{ID: uuid.New(), Name: "X-ANY"}}

	require.NoError(t, engine.MatchGroup(context.Background(), groupID))
}

// L2 但未配匹配规则 → no-op。
func TestGroupMatchEngine_MatchGroup_NoRule_NoOp(t *testing.T) {
	engine, repo, _, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	groupID := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), groupID).Return(&DeviceGroup{
		ID: groupID, Level: 2, Name: "L2-no-rule",
		MatchingMode: "", // 未配
	}, nil)
	// 不应有任何后续调用。

	require.NoError(t, engine.MatchGroup(context.Background(), groupID))
}

// ── handleDeviceRegistered ──────────────────────────────────────────────────

// 非法 JSON payload → 返回 decode 错误，不调任何 repo。
func TestGroupMatchEngine_HandleDeviceRegistered_InvalidPayload(t *testing.T) {
	engine, _, _, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	evt := event.Event{Subject: event.SubjectDeviceRegistered, Payload: json.RawMessage("not json")}

	err := engine.handleDeviceRegistered(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode device.registered payload")
}

// payload 缺 device_id → 返回 missing device_id 错误。
func TestGroupMatchEngine_HandleDeviceRegistered_MissingDeviceID(t *testing.T) {
	engine, _, _, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]any{
		"serial_number": "SN-001",
	})
	require.NoError(t, err)

	err = engine.handleDeviceRegistered(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing device_id")
}

// 正常 payload + 无任何分组配规则 → 走完不报错（AssignDeviceToGroup 返 nil）。
func TestGroupMatchEngine_HandleDeviceRegistered_NoConfiguredGroups_NoOp(t *testing.T) {
	engine, repo, lister, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	lister.devices = []DeviceForMatch{{ID: deviceID, SerialNumber: "SN-001"}}
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{}, nil)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]any{
		"device_id":     deviceID,
		"serial_number": "SN-001",
	})
	require.NoError(t, err)

	require.NoError(t, engine.handleDeviceRegistered(context.Background(), evt))
}

func TestGroupMatchEngine_HandleDeviceRegistered_UsesPersistedCurrentGroup(t *testing.T) {
	engine, repo, lister, ctrl := newEngineWithMocks(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	sourceID := uuid.New()
	targetID := uuid.New()
	lister.devices = []DeviceForMatch{{ID: deviceID, SerialNumber: "SN-001", CurrentGroupID: &sourceID}}
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{
		{ID: sourceID, Level: 2},
		{ID: targetID, Level: 2, SourceGroupID: &sourceID, MatchingMode: MatchingModeSerialNumber, SerialNumberList: []string{"SN-001"}},
	}, nil)
	repo.EXPECT().MoveDeviceAutoMatched(gomock.Any(), sourceID, targetID, deviceID).Return(int64(1), nil)

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]any{
		"device_id": deviceID, "serial_number": "SN-001",
	})
	require.NoError(t, err)
	require.NoError(t, engine.handleDeviceRegistered(context.Background(), evt))
}

// ── MatchDevice 排序：最近编辑的 L2 分组优先 ───────────────────────────────

// 反退化：两个 L2 分组都命中同一设备时，updated_at 更晚的分组胜出，
// 实现"分组最后新增/编辑为优先"。
func TestMatchDevice_PrefersMostRecentlyUpdatedL2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))

	sourceGroupID := uuid.New()
	older := DeviceGroup{
		ID: uuid.New(), Level: 2, Name: "older-group",
		SourceGroupID: &sourceGroupID,
		MatchingMode:  MatchingModeDeviceName,
		NameRuleList:  []NameRule{{Condition: "contain", Value: "BJ"}},
		UpdatedAt:     time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	newer := DeviceGroup{
		ID: uuid.New(), Level: 2, Name: "newer-group",
		SourceGroupID: &sourceGroupID,
		MatchingMode:  MatchingModeDeviceName,
		NameRuleList:  []NameRule{{Condition: "contain", Value: "BJ"}},
		UpdatedAt:     time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC),
	}
	// 故意把更早编辑的放前面，验证排序生效（不是按插入顺序）。
	// PgDeviceGroupRepository.GetTreeWithCounts 实际返回扁平 list（Level 字段区分
	// L1/L2），MatchDevice 按 Level==2 过滤，所以 mock 要镜像真实行为：返 flat。
	tree := []DeviceGroup{
		{ID: uuid.New(), Level: 1, Name: "root-L1"},
		{ID: sourceGroupID, Level: 2, Name: "source-group"},
		older,
		newer,
	}
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return(tree, nil)

	result, err := matcher.MatchDevice(context.Background(), MatchRequest{
		DeviceID:       uuid.New(),
		DeviceName:     "BJ-SITE-01",
		CurrentGroupID: &sourceGroupID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, newer.ID, result.GroupID, "最近编辑（updated_at 更晚）的分组应胜出")
	assert.Equal(t, "newer-group", result.GroupName)
	assert.Equal(t, sourceGroupID, result.SourceGroupID)
}

func TestMatchDevice_IgnoresRuleForDifferentSourceGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))

	ruleSourceID := uuid.New()
	currentGroupID := uuid.New()
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{{
		ID: uuid.New(), Level: 2, Name: "target",
		SourceGroupID: &ruleSourceID,
		MatchingMode:  MatchingModeTAC,
		TACList:       []int{1},
	}}, nil)

	tac := 1
	result, err := matcher.MatchDevice(context.Background(), MatchRequest{
		DeviceID: uuid.New(), TAC: &tac, CurrentGroupID: &currentGroupID,
	})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMatchDevice_IgnoresRuleWhoseSourceIsNoLongerL2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))

	sourceID := uuid.New()
	targetID := uuid.New()
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{
		{ID: sourceID, Level: 1},
		{ID: targetID, Level: 2, SourceGroupID: &sourceID, MatchingMode: MatchingModeTAC, TACList: []int{1}},
	}, nil)
	tac := 1
	result, err := matcher.MatchDevice(context.Background(), MatchRequest{
		DeviceID: uuid.New(), TAC: &tac, CurrentGroupID: &sourceID,
	})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMatchDevice_DefaultSourceMatchesExplicitDefaultMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))

	defaultGroupID := uuid.MustParse(global.DefaultLevel2GroupID)
	targetID := uuid.New()
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{
		{ID: defaultGroupID, Level: 2, Name: "default"},
		{ID: targetID, Level: 2, Name: "target",
			SourceGroupID: &defaultGroupID,
			MatchingMode:  MatchingModeSerialNumber,
			SerialNumberList: []string{
				"SN-001",
			},
		},
	}, nil)

	result, err := matcher.MatchDevice(context.Background(), MatchRequest{
		DeviceID: uuid.New(), SerialNumber: "SN-001", CurrentGroupID: &defaultGroupID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, targetID, result.GroupID)
	assert.Equal(t, defaultGroupID, result.SourceGroupID)
}

func TestHeartbeatAssigner_UsesCurrentGroupFromLister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := NewMockDeviceGroupRepository(ctrl)
	matcher := NewDeviceMatcher(repo, nil, zaptest.NewLogger(t))

	sourceGroupID := uuid.New()
	targetGroupID := uuid.New()
	deviceID := uuid.New()
	tac := 1
	lister := &fakeLister{devices: []DeviceForMatch{{
		ID: deviceID, TAC: &tac, CurrentGroupID: &sourceGroupID,
	}}}
	repo.EXPECT().GetTreeWithCounts(gomock.Any()).Return([]DeviceGroup{{
		ID: sourceGroupID, Level: 2, Name: "source",
	}, {
		ID: targetGroupID, Level: 2, Name: "target", SourceGroupID: &sourceGroupID,
		MatchingMode: MatchingModeTAC, TACList: []int{1},
	}}, nil)
	repo.EXPECT().MoveDeviceAutoMatched(gomock.Any(), sourceGroupID, targetGroupID, deviceID).Return(int64(1), nil)

	assigner := NewHeartbeatAssigner(matcher, lister)
	require.NoError(t, assigner.AssignByHeartbeat(context.Background(), HeartbeatRequest{DeviceID: deviceID}))
}
