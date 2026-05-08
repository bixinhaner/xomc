package northbound

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// fakeServerRepo 简易内存实现，便于 table-driven 测试。
type fakeServerRepo struct {
	servers      []Server
	listErr      error
	getErr       error
	setActiveErr error
	setActiveLog []ServerRole
}

func (r *fakeServerRepo) List(_ context.Context) ([]Server, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]Server, len(r.servers))
	copy(out, r.servers)
	return out, nil
}

func (r *fakeServerRepo) GetByRole(_ context.Context, role ServerRole) (*Server, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	for i := range r.servers {
		if r.servers[i].Role == role {
			s := r.servers[i]
			return &s, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeServerRepo) SetActive(_ context.Context, role ServerRole) error {
	r.setActiveLog = append(r.setActiveLog, role)
	if r.setActiveErr != nil {
		return r.setActiveErr
	}
	for i := range r.servers {
		r.servers[i].IsActive = r.servers[i].Role == role
	}
	return nil
}

func newTestServers(activeRole ServerRole) []Server {
	return []Server{
		{ID: uuid.New(), Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: activeRole == ServerRolePrimary},
		{ID: uuid.New(), Role: ServerRoleStandby, Host: "10.0.0.2", Port: 8081, IsActive: activeRole == ServerRoleStandby},
	}
}

func TestServerService_List(t *testing.T) {
	repo := &fakeServerRepo{servers: newTestServers(ServerRolePrimary)}
	svc := NewServerService(repo, zap.NewNop())

	got, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, ServerRolePrimary, got[0].Role)
	assert.True(t, got[0].IsActive)
	assert.Equal(t, ServerRoleStandby, got[1].Role)
	assert.False(t, got[1].IsActive)
}

func TestServerService_GetActive(t *testing.T) {
	tests := []struct {
		name     string
		active   ServerRole
		wantRole ServerRole
		wantNil  bool
	}{
		{"primary active", ServerRolePrimary, ServerRolePrimary, false},
		{"standby active", ServerRoleStandby, ServerRoleStandby, false},
		{"none active", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeServerRepo{servers: newTestServers(tc.active)}
			svc := NewServerService(repo, zap.NewNop())
			got, err := svc.GetActive(context.Background())
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tc.wantRole, got.Role)
			}
		})
	}
}

func TestServerService_SetActive(t *testing.T) {
	tests := []struct {
		name           string
		initialActive  ServerRole
		target         ServerRole
		wantSetCalls   int   // 期望 repo.SetActive 被调几次（noop 时为 0）
		wantErr        bool
	}{
		{"switch primary→standby", ServerRolePrimary, ServerRoleStandby, 1, false},
		{"switch standby→primary", ServerRoleStandby, ServerRolePrimary, 1, false},
		{"already active noop", ServerRolePrimary, ServerRolePrimary, 0, false},
		{"invalid role rejected", ServerRolePrimary, ServerRole("foo"), 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeServerRepo{servers: newTestServers(tc.initialActive)}
			svc := NewServerService(repo, zap.NewNop())
			err := svc.SetActive(context.Background(), tc.target)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, repo.setActiveLog, tc.wantSetCalls,
				"unexpected SetActive call count")
		})
	}
}

func TestServerService_SetActive_RoleNotFound(t *testing.T) {
	// 仿 GetByRole 返回 ErrNotFound 的场景：仓库无目标 role
	repo := &fakeServerRepo{servers: []Server{
		{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: true},
	}}
	svc := NewServerService(repo, zap.NewNop())
	err := svc.SetActive(context.Background(), ServerRoleStandby)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"expected ErrNotFound, got %v", err)
}
