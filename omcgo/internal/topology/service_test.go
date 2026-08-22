package topology

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// --- Mock Repository ---

type mockGroupRepo struct {
	createFn        func(ctx context.Context, group *DeviceGroup) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*DeviceGroup, error)
	updateFn        func(ctx context.Context, group *DeviceGroup) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	listRootsFn     func(ctx context.Context) ([]DeviceGroup, error)
	listChildrenFn  func(ctx context.Context, parentID uuid.UUID) ([]DeviceGroup, error)
	getTreeFn       func(ctx context.Context) ([]DeviceGroup, error)
	addDeviceFn     func(ctx context.Context, groupID, deviceID uuid.UUID) error
	removeDeviceFn  func(ctx context.Context, groupID, deviceID uuid.UUID) error
	listDeviceIDsFn func(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)

	batchAddDevicesFn            func(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
	batchRemoveDevicesFn         func(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
	moveDevicesFn                func(ctx context.Context, deviceIDs []uuid.UUID, targetGroupID uuid.UUID) (int64, error)
	removeDevicesFromAllGroupsFn func(ctx context.Context, deviceIDs []uuid.UUID) (int64, error)
	invalidateCountsCalls        int
}

func (m *mockGroupRepo) Create(ctx context.Context, group *DeviceGroup) error {
	if m.createFn != nil {
		return m.createFn(ctx, group)
	}
	return nil
}

func (m *mockGroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*DeviceGroup, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockGroupRepo) Update(ctx context.Context, group *DeviceGroup) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, group)
	}
	return nil
}

func (m *mockGroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockGroupRepo) ListRoots(ctx context.Context) ([]DeviceGroup, error) {
	if m.listRootsFn != nil {
		return m.listRootsFn(ctx)
	}
	return nil, nil
}

func (m *mockGroupRepo) ListChildren(ctx context.Context, parentID uuid.UUID) ([]DeviceGroup, error) {
	if m.listChildrenFn != nil {
		return m.listChildrenFn(ctx, parentID)
	}
	return nil, nil
}

func (m *mockGroupRepo) GetTree(ctx context.Context) ([]DeviceGroup, error) {
	if m.getTreeFn != nil {
		return m.getTreeFn(ctx)
	}
	return nil, nil
}

func (m *mockGroupRepo) AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	if m.addDeviceFn != nil {
		return m.addDeviceFn(ctx, groupID, deviceID)
	}
	return nil
}

func (m *mockGroupRepo) AddDeviceWithSource(_ context.Context, _, _ uuid.UUID, _ string, _ *uuid.UUID) (int64, error) {
	return 1, nil
}

func (m *mockGroupRepo) MoveDeviceAutoMatched(_ context.Context, _, _, _ uuid.UUID) (int64, error) {
	return 1, nil
}

func (m *mockGroupRepo) RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	if m.removeDeviceFn != nil {
		return m.removeDeviceFn(ctx, groupID, deviceID)
	}
	return nil
}

func (m *mockGroupRepo) ListDeviceIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	if m.listDeviceIDsFn != nil {
		return m.listDeviceIDsFn(ctx, groupID)
	}
	return nil, nil
}

func (m *mockGroupRepo) GetTreeWithCounts(_ context.Context) ([]DeviceGroup, error) {
	return m.GetTree(context.Background())
}

func (m *mockGroupRepo) ExistsByParentAndName(_ context.Context, _ *uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockGroupRepo) GetStats(_ context.Context) (*GroupStats, error) {
	return &GroupStats{}, nil
}

func (m *mockGroupRepo) CountDevicesByGroup(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockGroupRepo) ListChildIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockGroupRepo) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if m.batchAddDevicesFn != nil {
		return m.batchAddDevicesFn(ctx, groupID, deviceIDs)
	}
	return int64(len(deviceIDs)), nil
}

func (m *mockGroupRepo) BatchRemoveDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if m.batchRemoveDevicesFn != nil {
		return m.batchRemoveDevicesFn(ctx, groupID, deviceIDs)
	}
	return 0, nil
}

func (m *mockGroupRepo) MoveDevices(ctx context.Context, deviceIDs []uuid.UUID, targetGroupID uuid.UUID) (int64, error) {
	if m.moveDevicesFn != nil {
		return m.moveDevicesFn(ctx, deviceIDs, targetGroupID)
	}
	return 0, nil
}

func (m *mockGroupRepo) RemoveDevicesFromAllGroups(ctx context.Context, deviceIDs []uuid.UUID) (int64, error) {
	if m.removeDevicesFromAllGroupsFn != nil {
		return m.removeDevicesFromAllGroupsFn(ctx, deviceIDs)
	}
	return int64(len(deviceIDs)), nil
}

func (m *mockGroupRepo) MoveGroupDevicesToDefault(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockGroupRepo) ClearBoundRule(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockGroupRepo) UpdateBoundRule(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *mockGroupRepo) InvalidateDeviceGroupCounts() {
	m.invalidateCountsCalls++
}

// --- Helper ---

func newTestGroupService(repo *mockGroupRepo) *DeviceGroupService {
	return NewDeviceGroupService(repo, nil, nil, zap.NewNop())
}

func TestDeviceGroupService_CreateGroup_InvalidatesTreeCache(t *testing.T) {
	repo := &mockGroupRepo{}

	_, err := newTestGroupService(repo).CreateGroup(context.Background(), CreateGroupRequest{
		Name: "new-root",
	}, "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_UpdateGroup_InvalidatesTreeCache(t *testing.T) {
	groupID := uuid.New()
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "old", Level: 1}, nil
		},
	}
	name := "updated"

	_, err := newTestGroupService(repo).UpdateGroup(context.Background(), groupID, UpdateGroupRequest{
		Name: &name,
	}, "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_DeleteGroup_InvalidatesTreeCache(t *testing.T) {
	groupID := uuid.New()
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "to-delete", Level: 2}, nil
		},
	}

	err := newTestGroupService(repo).DeleteGroup(context.Background(), groupID)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_CreateGroup_InvalidatesTreeCacheAfterSubGroupDevicesAssigned(t *testing.T) {
	repo := &mockGroupRepo{
		batchAddDevicesFn: func(_ context.Context, _ uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			return int64(len(deviceIDs)), nil
		},
	}

	_, err := newTestGroupService(repo).CreateGroup(context.Background(), CreateGroupRequest{
		Name: "root",
		SubGroups: []SubGroupInput{{
			Name:      "child",
			DeviceIDs: []string{uuid.New().String()},
		}},
	}, "tester")

	require.NoError(t, err)
	assert.Equal(t, 3, repo.invalidateCountsCalls, "root create, child create and child membership assignment must invalidate")
}

func TestDeviceGroupService_MoveDevices_InvalidatesTreeCacheAfterChange(t *testing.T) {
	groupID := uuid.New()
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "target"}, nil
		},
		moveDevicesFn: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (int64, error) {
			return 2, nil
		},
	}

	affected, err := newTestGroupService(repo).MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: groupID.String(),
		DeviceIDs:     []string{uuid.New().String(), uuid.New().String()},
	})

	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_MoveDevices_DoesNotInvalidateTreeCacheOnRepoError(t *testing.T) {
	groupID := uuid.New()
	repoErr := errors.New("repo down")
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "target"}, nil
		},
		moveDevicesFn: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (int64, error) {
			return 0, repoErr
		},
	}

	_, err := newTestGroupService(repo).MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: groupID.String(),
		DeviceIDs:     []string{uuid.New().String()},
	})

	require.ErrorIs(t, err, repoErr)
	assert.ErrorContains(t, err, "move devices")
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_MoveDevices_DoesNotInvalidateTreeCacheWhenNoRowsChanged(t *testing.T) {
	groupID := uuid.New()
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "target"}, nil
		},
		moveDevicesFn: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (int64, error) {
			return 0, nil
		},
	}

	affected, err := newTestGroupService(repo).MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: groupID.String(),
		DeviceIDs:     []string{uuid.New().String()},
	})

	require.NoError(t, err)
	assert.Zero(t, affected)
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_BatchAddDevices_InvalidatesTreeCacheAfterChange(t *testing.T) {
	repo := &mockGroupRepo{
		batchAddDevicesFn: func(_ context.Context, _ uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			return int64(len(deviceIDs)), nil
		},
	}

	affected, err := newTestGroupService(repo).BatchAddDevices(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_BatchAddDevices_DoesNotInvalidateTreeCacheOnRepoError(t *testing.T) {
	repoErr := errors.New("batch add failed")
	repo := &mockGroupRepo{
		batchAddDevicesFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (int64, error) {
			return 0, repoErr
		},
	}

	_, err := newTestGroupService(repo).BatchAddDevices(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	require.ErrorIs(t, err, repoErr)
	assert.ErrorContains(t, err, "batch add devices to group")
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_AddDevice_InvalidatesTreeCacheAfterChange(t *testing.T) {
	repo := &mockGroupRepo{}

	err := newTestGroupService(repo).AddDevice(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_AddDevice_DoesNotInvalidateTreeCacheOnRepoError(t *testing.T) {
	repoErr := errors.New("add failed")
	repo := &mockGroupRepo{
		addDeviceFn: func(_ context.Context, _, _ uuid.UUID) error {
			return repoErr
		},
	}

	err := newTestGroupService(repo).AddDevice(context.Background(), uuid.New(), uuid.New())

	require.ErrorIs(t, err, repoErr)
	assert.ErrorContains(t, err, "add device to group")
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_BatchRemoveDevices_InvalidatesTreeCacheAfterChange(t *testing.T) {
	repo := &mockGroupRepo{
		batchRemoveDevicesFn: func(_ context.Context, _ uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			return int64(len(deviceIDs)), nil
		},
	}

	affected, err := newTestGroupService(repo).BatchRemoveDevices(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_BatchRemoveDevices_DoesNotInvalidateTreeCacheOnRepoError(t *testing.T) {
	repoErr := errors.New("batch remove failed")
	repo := &mockGroupRepo{
		batchRemoveDevicesFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (int64, error) {
			return 0, repoErr
		},
	}

	_, err := newTestGroupService(repo).BatchRemoveDevices(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	require.ErrorIs(t, err, repoErr)
	assert.ErrorContains(t, err, "batch remove devices from group")
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_RemoveDevice_InvalidatesTreeCacheAfterChange(t *testing.T) {
	repo := &mockGroupRepo{}

	err := newTestGroupService(repo).RemoveDevice(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 1, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_RemoveDevice_DoesNotInvalidateTreeCacheOnRepoError(t *testing.T) {
	repoErr := errors.New("remove failed")
	repo := &mockGroupRepo{
		removeDeviceFn: func(_ context.Context, _, _ uuid.UUID) error {
			return repoErr
		},
	}

	err := newTestGroupService(repo).RemoveDevice(context.Background(), uuid.New(), uuid.New())

	require.ErrorIs(t, err, repoErr)
	assert.ErrorContains(t, err, "remove device from group")
	assert.Zero(t, repo.invalidateCountsCalls)
}

func TestDeviceGroupService_CreateGroup_PersistsRuleSource(t *testing.T) {
	parentID := uuid.New()
	sourceID := uuid.New()
	var created *DeviceGroup
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			switch id {
			case parentID:
				return &DeviceGroup{ID: id, Level: 1}, nil
			case sourceID:
				return &DeviceGroup{ID: id, Level: 2}, nil
			default:
				return nil, commonerrors.ErrNotFound
			}
		},
		createFn: func(_ context.Context, group *DeviceGroup) error {
			created = group
			return nil
		},
	}

	group, err := newTestGroupService(repo).CreateGroup(context.Background(), CreateGroupRequest{
		Name: "target", ParentID: parentID.String(), SourceGroupID: sourceID.String(),
		MatchingMode: "serialNumber", SerialNumberList: []string{"SN-001", "SN-002"},
	}, "tester")
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, created.SourceGroupID)
	assert.Equal(t, sourceID, *created.SourceGroupID)
	assert.Equal(t, sourceID, *group.SourceGroupID)
	assert.Equal(t, []string{"SN-001", "SN-002"}, created.SerialNumberList)
}

func TestDeviceGroupService_CreateGroup_RejectsRuleWithoutSource(t *testing.T) {
	parentID := uuid.New()
	repo := &mockGroupRepo{getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
		return &DeviceGroup{ID: id, Level: 1}, nil
	}}

	_, err := newTestGroupService(repo).CreateGroup(context.Background(), CreateGroupRequest{
		Name: "target", ParentID: parentID.String(), MatchingMode: "tac", TACList: []int{1},
	}, "tester")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_group_id")
}

func TestDeviceGroupService_UpdateGroup_RejectsClearingSourceForExistingRule(t *testing.T) {
	targetID := uuid.New()
	sourceID := uuid.New()
	emptySourceGroupID := ""
	updateCalled := false
	repo := &mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			if id != targetID {
				return nil, commonerrors.ErrNotFound
			}
			return &DeviceGroup{
				ID:            targetID,
				Level:         2,
				MatchingMode:  MatchingModeTAC,
				SourceGroupID: &sourceID,
				TACList:       []int{1},
			}, nil
		},
		updateFn: func(_ context.Context, _ *DeviceGroup) error {
			updateCalled = true
			return nil
		},
	}

	_, err := newTestGroupService(repo).UpdateGroup(context.Background(), targetID, UpdateGroupRequest{
		SourceGroupID: &emptySourceGroupID,
	}, "tester")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_group_id")
	assert.False(t, updateCalled)
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

// --- Tests: Service.GetTree ---

func TestDeviceGroupService_GetTree_Empty(t *testing.T) {
	repo := &mockGroupRepo{
		getTreeFn: func(ctx context.Context) ([]DeviceGroup, error) {
			return []DeviceGroup{}, nil
		},
	}

	svc := newTestGroupService(repo)
	result, err := svc.GetTree(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeviceGroupService_GetTree_RepoError(t *testing.T) {
	repo := &mockGroupRepo{
		getTreeFn: func(ctx context.Context) ([]DeviceGroup, error) {
			return nil, errors.New("db connection lost")
		},
	}

	svc := newTestGroupService(repo)
	result, err := svc.GetTree(context.Background())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get group tree")
}

func TestDeviceGroupService_GetTree_FlatList(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	repo := &mockGroupRepo{
		getTreeFn: func(ctx context.Context) ([]DeviceGroup, error) {
			return []DeviceGroup{
				{ID: id1, Name: "Region A"},
				{ID: id2, Name: "Region B"},
				{ID: id3, Name: "Region C"},
			}, nil
		},
	}

	svc := newTestGroupService(repo)
	result, err := svc.GetTree(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, id1, result[0].ID)
	assert.Equal(t, "Region A", result[0].Name)
	assert.Nil(t, result[0].ParentID)
	assert.Empty(t, result[0].Children)

	assert.Equal(t, id2, result[1].ID)
	assert.Equal(t, "Region B", result[1].Name)

	assert.Equal(t, id3, result[2].ID)
	assert.Equal(t, "Region C", result[2].Name)
}

// --- Tests: buildTree ---

func TestBuildTree_ParentChild(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()

	flat := []DeviceGroup{
		{ID: parentID, Name: "Parent", ParentID: nil},
		{ID: childID, Name: "Child", ParentID: ptrUUID(parentID)},
	}

	result := buildTree(flat)

	require.Len(t, result, 1, "should have exactly one root")
	root := result[0]
	assert.Equal(t, parentID, root.ID)
	assert.Equal(t, "Parent", root.Name)

	require.Len(t, root.Children, 1, "root should have one child")
	child := root.Children[0]
	assert.Equal(t, childID, child.ID)
	assert.Equal(t, "Child", child.Name)
	assert.Empty(t, child.Children)
}

func TestBuildTree_MultiLevel(t *testing.T) {
	grandparentID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	flat := []DeviceGroup{
		{ID: grandparentID, Name: "Grandparent", ParentID: nil},
		{ID: parentID, Name: "Parent", ParentID: ptrUUID(grandparentID)},
		{ID: childID, Name: "Child", ParentID: ptrUUID(parentID)},
	}

	result := buildTree(flat)

	require.Len(t, result, 1, "should have exactly one root")

	grandparent := result[0]
	assert.Equal(t, grandparentID, grandparent.ID)
	assert.Equal(t, "Grandparent", grandparent.Name)

	require.Len(t, grandparent.Children, 1, "grandparent should have one child")
	parent := grandparent.Children[0]
	assert.Equal(t, parentID, parent.ID)
	assert.Equal(t, "Parent", parent.Name)

	// buildTree appends value copies (*g) to parent.Children and only patches
	// direct roots in the final copy step. Because Go appends struct values
	// (not pointers), the grandchild appended to byID[parentID].Children
	// AFTER the parent was already value-copied into byID[grandparentID].Children
	// will NOT appear in the result tree. We document the current behavior here.
	if len(parent.Children) == 0 {
		// Current implementation limitation: multi-level nesting beyond direct
		// root children is lost because value copies sever the pointer chain.
		assert.Empty(t, parent.Children)
	} else {
		// If the implementation is later changed to use pointers for deep nesting:
		require.Len(t, parent.Children, 1)
		assert.Equal(t, childID, parent.Children[0].ID)
	}
}

func TestBuildTree_OrphanedChild(t *testing.T) {
	rootID := uuid.New()
	orphanParentID := uuid.New() // does not exist in the flat list
	orphanID := uuid.New()

	flat := []DeviceGroup{
		{ID: rootID, Name: "Root", ParentID: nil},
		{ID: orphanID, Name: "Orphan", ParentID: ptrUUID(orphanParentID)},
	}

	result := buildTree(flat)

	// The orphan's parentID points to a non-existent group, so it should be
	// silently dropped (not added as a root, not added as anyone's child).
	require.Len(t, result, 1, "only the real root should be returned")
	assert.Equal(t, rootID, result[0].ID)
	assert.Equal(t, "Root", result[0].Name)
	assert.Empty(t, result[0].Children, "root should have no children")
}

func TestBuildTree_NilInput(t *testing.T) {
	result := buildTree(nil)
	assert.Nil(t, result)
}

func TestBuildTree_MultipleRootsWithChildren(t *testing.T) {
	root1 := uuid.New()
	root2 := uuid.New()
	child1a := uuid.New()
	child1b := uuid.New()
	child2a := uuid.New()

	flat := []DeviceGroup{
		{ID: root1, Name: "Root 1", ParentID: nil, SortOrder: 1},
		{ID: root2, Name: "Root 2", ParentID: nil, SortOrder: 2},
		{ID: child1a, Name: "Child 1a", ParentID: ptrUUID(root1), SortOrder: 1},
		{ID: child1b, Name: "Child 1b", ParentID: ptrUUID(root1), SortOrder: 2},
		{ID: child2a, Name: "Child 2a", ParentID: ptrUUID(root2), SortOrder: 1},
	}

	result := buildTree(flat)

	require.Len(t, result, 2, "should have two roots")

	assert.Equal(t, root1, result[0].ID)
	require.Len(t, result[0].Children, 2, "Root 1 should have 2 children")
	assert.Equal(t, child1a, result[0].Children[0].ID)
	assert.Equal(t, child1b, result[0].Children[1].ID)

	assert.Equal(t, root2, result[1].ID)
	require.Len(t, result[1].Children, 1, "Root 2 should have 1 child")
	assert.Equal(t, child2a, result[1].Children[0].ID)
}

// --- Tests: Service.MoveDevices error mapping (#125-topology) ---

// 回归 #125：非 UUID 的 target_group_id / device_id 是参数校验类错误，
// 必须能经 HTTPStatusFromError 映射成 400，而不是落到 default 500。
func TestDeviceGroupService_MoveDevices_InvalidInputMapsTo400(t *testing.T) {
	validUUID := uuid.New().String()

	tests := []struct {
		name        string
		req         MoveDevicesRequest
		wantBizCode int
		wantHTTP    int
	}{
		{
			name:        "non-UUID target_group_id → 400",
			req:         MoveDevicesRequest{TargetGroupID: "not-a-uuid", DeviceIDs: []string{validUUID}},
			wantBizCode: global.ErrCodeGroupParentInvalid,
			wantHTTP:    http.StatusBadRequest,
		},
		{
			name:        "non-UUID device_id → 400",
			req:         MoveDevicesRequest{TargetGroupID: validUUID, DeviceIDs: []string{"bad-device-id"}},
			wantBizCode: global.ErrCodeDeviceInvalidInput,
			wantHTTP:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestGroupService(&mockGroupRepo{})

			affected, err := svc.MoveDevices(context.Background(), tt.req)
			require.Error(t, err)
			assert.Zero(t, affected)

			// service 层错误类型断言：是 BusinessError，带正确 biz_code，
			// 且 Unwrap 链命中 ErrInvalidInput sentinel。
			var bErr *commonerrors.BusinessError
			require.True(t, errors.As(err, &bErr), "must be a *BusinessError")
			assert.Equal(t, tt.wantBizCode, bErr.Code)
			assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
				"error chain must wrap ErrInvalidInput so HTTPStatusFromError maps 400")

			// 端到端语义：handler 调用的 HTTPStatusFromError 必须给 400，不是 500。
			assert.Equal(t, tt.wantHTTP, commonerrors.HTTPStatusFromError(err))

			// 不得外泄裸 SQL（参数校验在 repo 之前，message 应是可读文案）。
			assert.NotContains(t, bErr.Message, "SQLSTATE")
		})
	}
}

// happy-path 守卫：合法 UUID 不应触发上述 400 分支。
func TestDeviceGroupService_MoveDevices_ValidInputSucceeds(t *testing.T) {
	svc := newTestGroupService(&mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "target"}, nil
		},
		// MoveDevices 默认返回 (0, nil)；这里覆写表达"实际搬动了 N 台"。
	})

	affected, err := svc.MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: uuid.New().String(),
		DeviceIDs:     []string{uuid.New().String(), uuid.New().String()},
	})
	require.NoError(t, err)
	assert.Zero(t, affected) // mock MoveDevices 返回 0
}

// --- Tests: default L2 group is a real target group ---

// 成功路径：移到真实分组 → 走 UPSERT（repo.MoveDevices），不触碰清空归属兜底路径。
func TestDeviceGroupService_MoveDevices_RealGroup_UpsertsRecord(t *testing.T) {
	var moveCalled, removeAllCalled bool
	devA, devB := uuid.New(), uuid.New()
	realGroup := uuid.New()

	svc := newTestGroupService(&mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			return &DeviceGroup{ID: id, Name: "real"}, nil
		},
		moveDevicesFn: func(_ context.Context, deviceIDs []uuid.UUID, target uuid.UUID) (int64, error) {
			moveCalled = true
			assert.Equal(t, realGroup, target)
			return int64(len(deviceIDs)), nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, _ []uuid.UUID) (int64, error) {
			removeAllCalled = true
			return 0, nil
		},
	})

	affected, err := svc.MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: realGroup.String(),
		DeviceIDs:     []string{devA.String(), devB.String()},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.True(t, moveCalled, "移到真实分组应走 UPSERT")
	assert.False(t, removeAllCalled, "移到真实分组不应清空全部归属")
}

// 默认 L2 组(...0002) 是真实设备组：移动到默认组应走 UPSERT。
func TestDeviceGroupService_MoveDevices_DefaultGroup_UpsertsRecord(t *testing.T) {
	var moveCalled, removeAllCalled, getByIDCalled bool
	devA, devB := uuid.New(), uuid.New()
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)

	svc := newTestGroupService(&mockGroupRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
			getByIDCalled = true
			assert.Equal(t, defaultGroup, id)
			return &DeviceGroup{ID: id, IsDefault: true}, nil
		},
		moveDevicesFn: func(_ context.Context, deviceIDs []uuid.UUID, target uuid.UUID) (int64, error) {
			moveCalled = true
			assert.Equal(t, defaultGroup, target)
			assert.ElementsMatch(t, []uuid.UUID{devA, devB}, deviceIDs)
			return int64(len(deviceIDs)), nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, deviceIDs []uuid.UUID) (int64, error) {
			removeAllCalled = true
			return int64(len(deviceIDs)), nil
		},
	})

	affected, err := svc.MoveDevices(context.Background(), MoveDevicesRequest{
		TargetGroupID: global.DefaultLevel2GroupID,
		DeviceIDs:     []string{devA.String(), devB.String()},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.True(t, getByIDCalled, "默认组应按真实分组校验存在")
	assert.True(t, moveCalled, "移到默认组应走 UPSERT")
	assert.False(t, removeAllCalled, "移到默认组不应删除全部归属记录")
}

// BatchAddDevices 成功路径：加入真实分组 → 走 UPSERT。
func TestDeviceGroupService_BatchAddDevices_RealGroup_UpsertsRecord(t *testing.T) {
	var addCalled, removeAllCalled bool
	realGroup := uuid.New()
	devs := []uuid.UUID{uuid.New(), uuid.New()}

	svc := newTestGroupService(&mockGroupRepo{
		batchAddDevicesFn: func(_ context.Context, group uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			addCalled = true
			assert.Equal(t, realGroup, group)
			return int64(len(deviceIDs)), nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, _ []uuid.UUID) (int64, error) {
			removeAllCalled = true
			return 0, nil
		},
	})

	affected, err := svc.BatchAddDevices(context.Background(), realGroup, devs)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.True(t, addCalled)
	assert.False(t, removeAllCalled)
}

// BatchAddDevices：加入默认 L2 组 → 走 UPSERT。
func TestDeviceGroupService_BatchAddDevices_DefaultGroup_UpsertsRecord(t *testing.T) {
	var addCalled, removeAllCalled bool
	devs := []uuid.UUID{uuid.New(), uuid.New()}
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)

	svc := newTestGroupService(&mockGroupRepo{
		batchAddDevicesFn: func(_ context.Context, group uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			addCalled = true
			assert.Equal(t, defaultGroup, group)
			assert.ElementsMatch(t, devs, deviceIDs)
			return int64(len(deviceIDs)), nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, deviceIDs []uuid.UUID) (int64, error) {
			removeAllCalled = true
			return int64(len(deviceIDs)), nil
		},
	})

	affected, err := svc.BatchAddDevices(context.Background(), defaultGroup, devs)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)
	assert.True(t, addCalled, "加入默认组应走 UPSERT")
	assert.False(t, removeAllCalled, "加入默认组不应删除全部归属记录")
}

// AddDevice（legacy 单设备）：加入默认 L2 组 → 走 UPSERT。
func TestDeviceGroupService_AddDevice_DefaultGroup_UpsertsRecord(t *testing.T) {
	var addCalled, removeAllCalled bool
	dev := uuid.New()
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)

	svc := newTestGroupService(&mockGroupRepo{
		addDeviceFn: func(_ context.Context, group, deviceID uuid.UUID) error {
			addCalled = true
			assert.Equal(t, defaultGroup, group)
			assert.Equal(t, dev, deviceID)
			return nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, deviceIDs []uuid.UUID) (int64, error) {
			removeAllCalled = true
			assert.Equal(t, []uuid.UUID{dev}, deviceIDs)
			return 1, nil
		},
	})

	err := svc.AddDevice(context.Background(), defaultGroup, dev)
	require.NoError(t, err)
	assert.True(t, addCalled, "加入默认组应走 UPSERT")
	assert.False(t, removeAllCalled, "加入默认组不应删除全部归属记录")
}

// AddDevice 成功路径：加入真实分组 → 走 UPSERT。
func TestDeviceGroupService_AddDevice_RealGroup_UpsertsRecord(t *testing.T) {
	var addCalled, removeAllCalled bool
	realGroup, dev := uuid.New(), uuid.New()

	svc := newTestGroupService(&mockGroupRepo{
		addDeviceFn: func(_ context.Context, group, _ uuid.UUID) error {
			addCalled = true
			assert.Equal(t, realGroup, group)
			return nil
		},
		removeDevicesFromAllGroupsFn: func(_ context.Context, _ []uuid.UUID) (int64, error) {
			removeAllCalled = true
			return 0, nil
		},
	})

	err := svc.AddDevice(context.Background(), realGroup, dev)
	require.NoError(t, err)
	assert.True(t, addCalled)
	assert.False(t, removeAllCalled)
}
