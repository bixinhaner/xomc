package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builtin role UUIDs（与 service.go builtInRoleIDs / seed/000036 一致）。
const (
	builtinAdminRoleID    = "10000000-0000-0000-0000-000000000001"
	builtinOperatorRoleID = "10000000-0000-0000-0000-000000000002"
	builtinViewerRoleID   = "10000000-0000-0000-0000-000000000003"
)

// TestDeleteRole_BuiltInProtected 覆盖 issue #136：三个内置角色（admin/operator/viewer）
// 即便 is_system 标记在某环境被错误改写，DeleteRole 也必须拒绝（固定 UUID 白名单兜底），
// 返回包 ErrForbidden 的错误 → handler 经 HTTPStatusFromError 映射 403。
func TestDeleteRole_BuiltInProtected(t *testing.T) {
	tests := []struct {
		name   string
		roleID string
	}{
		{"admin 内置角色不可删", builtinAdminRoleID},
		{"operator 内置角色不可删（历史 is_system 误标 false）", builtinOperatorRoleID},
		{"viewer 内置角色不可删（历史 is_system 误标 false）", builtinViewerRoleID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCalled := false
			roleRepo := &mockRoleRepo{
				deleteFn: func(_ context.Context, _ uuid.UUID) error {
					deleteCalled = true
					return nil
				},
			}
			svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})

			err := svc.DeleteRole(context.Background(), uuid.MustParse(tt.roleID))

			require.Error(t, err, "内置角色删除必须被拒绝")
			assert.True(t, errors.Is(err, commonerrors.ErrForbidden),
				"应返回包 ErrForbidden 的错误以便 handler 映射 403，实际 %v", err)
			assert.Equal(t, commonerrors.ErrForbidden, err)
			assert.False(t, deleteCalled, "白名单兜底应在调 repo.Delete 之前短路")
		})
	}
}

// TestDeleteRole_CustomRoleAllowed 失败路径对照：非内置（自建）角色删除应正常透传到 repo，
// 不被白名单误伤。
func TestDeleteRole_CustomRoleAllowed(t *testing.T) {
	customID := uuid.New()
	deleteCalled := false
	roleRepo := &mockRoleRepo{
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			deleteCalled = true
			assert.Equal(t, customID, id)
			return nil
		},
	}
	svc := newTestService(&mockUserRepo{}, roleRepo, &mockAuditRepo{})

	err := svc.DeleteRole(context.Background(), customID)
	require.NoError(t, err)
	assert.True(t, deleteCalled, "自建角色删除应透传到 repo.Delete")
}

// TestIsBuiltInRole 单测白名单判定本身（含一个反例）。
func TestIsBuiltInRole(t *testing.T) {
	assert.True(t, isBuiltInRole(uuid.MustParse(builtinAdminRoleID)))
	assert.True(t, isBuiltInRole(uuid.MustParse(builtinOperatorRoleID)))
	assert.True(t, isBuiltInRole(uuid.MustParse(builtinViewerRoleID)))
	assert.False(t, isBuiltInRole(uuid.New()), "随机自建角色 ID 不在白名单")
	assert.False(t, isBuiltInRole(uuid.Nil), "Nil UUID 不在白名单")
}

// TestSwitchRole_UnassignedRoleForbidden 覆盖 issue #125 第 1 项：切换到未分配给该用户的
// 角色，service 返回 BusinessError(7003) 包 ErrForbidden，HTTPStatusFromError 应映射 403
// （而非 handler 旧逻辑的一律 500）。
func TestSwitchRole_UnassignedRoleForbidden(t *testing.T) {
	userID := uuid.New()
	assignedRoleID := uuid.New()
	targetRoleID := uuid.New() // 未分配给用户的角色

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Username: "alice"}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
			return []Role{{ID: assignedRoleID, Name: "operator"}}, nil
		},
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	_, err := svc.SwitchRole(context.Background(), userID, targetRoleID)
	require.Error(t, err)

	// 错误应可映射为 403（包 ErrForbidden），且 biz_code=7003。
	assert.Equal(t, 403, commonerrors.HTTPStatusFromError(err),
		"未分配角色应映射 403，而非 500")
	var bizErr *commonerrors.BusinessError
	require.True(t, errors.As(err, &bizErr), "应为 BusinessError")
	assert.Equal(t, 7003, bizErr.Code)
}

// TestSwitchRole_AssignedRoleSucceeds 成功路径对照：切换到已分配角色返回 token pair，无错误。
func TestSwitchRole_AssignedRoleSucceeds(t *testing.T) {
	userID := uuid.New()
	targetRoleID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: id, Username: "alice"}, nil
		},
	}
	roleRepo := &mockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
			return []Role{{ID: targetRoleID, Name: "operator"}}, nil
		},
	}
	svc := newTestService(userRepo, roleRepo, &mockAuditRepo{})

	pair, err := svc.SwitchRole(context.Background(), userID, targetRoleID)
	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.NotEmpty(t, pair.AccessToken)
}
