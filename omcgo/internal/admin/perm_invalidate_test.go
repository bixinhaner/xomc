package admin

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPermInvalidator 记录 InvalidateUserCache 的调用，用于断言次数（PRD §10 DoD）。
type mockPermInvalidator struct {
	mu    sync.Mutex
	calls []uuid.UUID
}

func (m *mockPermInvalidator) InvalidateUserCache(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, userID)
	return nil
}

func (m *mockPermInvalidator) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// TestInvalidatePermCache_AssignRole 断言：AssignRole 成功后 InvalidateUserCache 调用 1 次。
func TestInvalidatePermCache_AssignRole(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	userRepo := &mockUserRepo{getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
		return &User{ID: id, Status: UserStatusActive}, nil
	}}
	roleRepo := &mockRoleRepo{
		getByIDFn:    func(_ context.Context, id uuid.UUID) (*Role, error) { return &Role{ID: id}, nil },
		assignRoleFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.AssignRole(context.Background(), userID, roleID))
	assert.Equal(t, 1, inv.callCount(), "AssignRole 应调 InvalidateUserCache 恰好 1 次")
}

// TestInvalidatePermCache_RemoveRole 断言：RemoveRole 成功后 InvalidateUserCache 调用 1 次。
func TestInvalidatePermCache_RemoveRole(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	userRepo := &mockUserRepo{}
	roleRepo := &mockRoleRepo{
		removeRoleFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.RemoveRole(context.Background(), userID, roleID))
	assert.Equal(t, 1, inv.callCount(), "RemoveRole 应调 InvalidateUserCache 恰好 1 次")
}

// TestInvalidatePermCache_UpdateUser_RoleChange 断言：role_ids 变更触发 1 次失效。
func TestInvalidatePermCache_UpdateUser_RoleChange(t *testing.T) {
	userID := uuid.New()
	newRoleID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Status: UserStatusActive}, nil
		},
		updateFn: func(_ context.Context, _ *User) error { return nil },
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) { return nil, nil },
		assignRoleFn:   func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	roleIDs := []uuid.UUID{newRoleID}
	_, err := svc.UpdateUser(context.Background(), userID, UpdateUserRequest{RoleIDs: &roleIDs})
	require.NoError(t, err)
	assert.Equal(t, 1, inv.callCount(), "UpdateUser(role_ids) 应调 InvalidateUserCache 1 次")
}

// TestInvalidatePermCache_UpdateUser_CarrierChange removed in v1.0:
// users.carrier column dropped, UpdateUserRequest no longer has Carrier field.
// See PRD §11.11.

// TestInvalidatePermCache_UpdateUser_NoVisibilityChange 断言：仅改 email 不触发缓存失效。
func TestInvalidatePermCache_UpdateUser_NoVisibilityChange(t *testing.T) {
	userID := uuid.New()
	newEmail := "x@y.cn"

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Status: UserStatusActive}, nil
		},
		updateFn: func(_ context.Context, _ *User) error { return nil },
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) { return nil, nil },
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	_, err := svc.UpdateUser(context.Background(), userID, UpdateUserRequest{Email: &newEmail})
	require.NoError(t, err)
	assert.Equal(t, 0, inv.callCount(), "UpdateUser(仅 email) 不应调 InvalidateUserCache")
}

// TestInvalidatePermCache_DeleteUser 断言：DeleteUser 成功后调 InvalidateUserCache 1 次。
func TestInvalidatePermCache_DeleteUser(t *testing.T) {
	userID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Source: UserSourceAdmin}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(userRepo, &mockRoleRepo{}, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.DeleteUser(context.Background(), userID))
	assert.Equal(t, 1, inv.callCount(), "DeleteUser 应调 InvalidateUserCache 1 次")
}

// TestInvalidatePermCache_BatchAssignRoles 断言：N 个用户应调 InvalidateUserCache 恰好 N 次。
func TestInvalidatePermCache_BatchAssignRoles(t *testing.T) {
	userIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	roleIDs := []uuid.UUID{uuid.New()}

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Status: UserStatusActive}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) { return nil, nil },
		assignRoleFn:   func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.BatchAssignRoles(context.Background(), userIDs, roleIDs))
	assert.Equal(t, len(userIDs), inv.callCount(),
		"BatchAssignRoles 应对每个用户调 1 次（共 %d 次）", len(userIDs))
}

// TestInvalidatePermCacheByRole 断言：roleID 持有 N 用户时，InvalidateUserCache 调 N 次。
// PRD roles.md §10 DoD。
func TestInvalidatePermCacheByRole(t *testing.T) {
	roleID := uuid.New()
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}

	roleRepo := &mockRoleRepo{
		listUserIDsByRoleFn: func(_ context.Context, rid uuid.UUID) ([]uuid.UUID, error) {
			assert.Equal(t, roleID, rid)
			return userIDs, nil
		},
	}
	svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	svc.InvalidatePermCacheByRole(context.Background(), roleID)
	assert.Equal(t, len(userIDs), inv.callCount(),
		"InvalidatePermCacheByRole 应对每个用户调 1 次（共 %d 次）", len(userIDs))
}

// TestInvalidatePermCache_DeleteRole 断言：无用户引用的角色删除成功后，删除前查得的用户列表
// （此处为空）逐一失效。issue #191 后，仍有用户引用的角色已被 in-use 拦截删除，故"删除成功"
// 仅发生在无用户引用时，此时失效循环 0 次。仍有用户引用→被拦截的路径由
// TestDeleteRole_InUseBlocked 覆盖。
func TestInvalidatePermCache_DeleteRole(t *testing.T) {
	roleID := uuid.New()

	roleRepo := &mockRoleRepo{
		listUserIDsByRoleFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return nil, nil // 无用户引用 → 允许删除
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.DeleteRole(context.Background(), roleID))
	assert.Equal(t, 0, inv.callCount(),
		"无用户引用时删除成功，失效循环 0 次")
}

// TestInvalidatePermCache_SetRoleMenus 断言：SetRoleMenus 成功后该角色下用户全部失效。
// mockMenuRepo.SetRoleMenus 默认返回 nil（成功），不需要自定义 fn。
func TestInvalidatePermCache_SetRoleMenus(t *testing.T) {
	roleID := uuid.New()
	operatorID := uuid.New()
	userIDs := []uuid.UUID{uuid.New()}

	roleRepo := &mockRoleRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Role, error) {
			return &Role{ID: id}, nil
		},
		listUserIDsByRoleFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return userIDs, nil
		},
	}
	svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})
	inv := &mockPermInvalidator{}
	svc.SetPermissionInvalidator(inv)

	require.NoError(t, svc.SetRoleMenus(context.Background(), roleID, []uuid.UUID{uuid.New()}, operatorID))
	assert.Equal(t, len(userIDs), inv.callCount(),
		"SetRoleMenus 应对每个用户调 InvalidateUserCache 1 次")
}
