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
	"github.com/omcgo/omcgo/internal/core/event"
)

// fakeServerRepo 简易内存实现，便于 table-driven 测试。
type fakeServerRepo struct {
	servers      []Server
	listErr      error
	getErr       error
	setActiveErr error
	setActiveLog []ServerRole
	updateErr    error
	updateLog    []updateCall
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

type updateCall struct {
	role        ServerRole
	host        string
	port        int
	description string
}

func (r *fakeServerRepo) Update(_ context.Context, role ServerRole, host string, port int, description string) error {
	r.updateLog = append(r.updateLog, updateCall{role, host, port, description})
	if r.updateErr != nil {
		return r.updateErr
	}
	for i := range r.servers {
		if r.servers[i].Role == role {
			r.servers[i].Host = host
			r.servers[i].Port = port
			r.servers[i].Description = description
			return nil
		}
	}
	return commonerrors.ErrNotFound
}

func newTestServers(activeRole ServerRole) []Server {
	return []Server{
		{ID: uuid.New(), Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: activeRole == ServerRolePrimary},
		{ID: uuid.New(), Role: ServerRoleStandby, Host: "10.0.0.2", Port: 8081, IsActive: activeRole == ServerRoleStandby},
	}
}

func TestServerService_List(t *testing.T) {
	repo := &fakeServerRepo{servers: newTestServers(ServerRolePrimary)}
	svc := NewServerService(repo, nil, zap.NewNop())

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
			svc := NewServerService(repo, nil, zap.NewNop())
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
			svc := NewServerService(repo, nil, zap.NewNop())
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

func TestServerService_Update(t *testing.T) {
	tests := []struct {
		name    string
		role    ServerRole
		req     UpdateServerRequest
		wantErr bool
	}{
		{
			name:    "valid update primary",
			role:    ServerRolePrimary,
			req:     UpdateServerRequest{Host: "10.1.1.1", Port: 9000, Description: "primary updated"},
			wantErr: false,
		},
		{
			name:    "valid update standby with empty description",
			role:    ServerRoleStandby,
			req:     UpdateServerRequest{Host: "10.1.1.2", Port: 9000},
			wantErr: false,
		},
		{name: "invalid role", role: ServerRole("foo"), req: UpdateServerRequest{Host: "x", Port: 80}, wantErr: true},
		{name: "empty host rejected", role: ServerRolePrimary, req: UpdateServerRequest{Host: "", Port: 80}, wantErr: true},
		{name: "port=0 rejected", role: ServerRolePrimary, req: UpdateServerRequest{Host: "x", Port: 0}, wantErr: true},
		{name: "port=70000 rejected", role: ServerRolePrimary, req: UpdateServerRequest{Host: "x", Port: 70000}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeServerRepo{servers: newTestServers(ServerRolePrimary)}
			svc := NewServerService(repo, nil, zap.NewNop())
			err := svc.Update(context.Background(), tc.role, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.updateLog, 1)
			assert.Equal(t, tc.role, repo.updateLog[0].role)
			assert.Equal(t, tc.req.Host, repo.updateLog[0].host)
			assert.Equal(t, tc.req.Port, repo.updateLog[0].port)
			assert.Equal(t, tc.req.Description, repo.updateLog[0].description)
		})
	}
}

func TestServerService_Update_RoleNotFound(t *testing.T) {
	// 仿 repo.Update 返回 ErrNotFound（仿真：servers 列表里没目标 role）
	repo := &fakeServerRepo{servers: []Server{
		{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: true},
	}}
	svc := NewServerService(repo, nil, zap.NewNop())
	err := svc.Update(context.Background(), ServerRoleStandby, UpdateServerRequest{Host: "x", Port: 80})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestServerService_SetActive_RoleNotFound(t *testing.T) {
	// 仿 GetByRole 返回 ErrNotFound 的场景：仓库无目标 role
	repo := &fakeServerRepo{servers: []Server{
		{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: true},
	}}
	svc := NewServerService(repo, nil, zap.NewNop())
	err := svc.SetActive(context.Background(), ServerRoleStandby)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"expected ErrNotFound, got %v", err)
}

// ============================================================
// T-0099 — EventBus publish + GetActiveForPush 适配器测试
// ============================================================

// captureEventBus 捕获 publish 调用用于断言。
type captureEventBus struct {
	published []capturedEvent
}

type capturedEvent struct {
	subject string
	payload []byte
}

func (b *captureEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	b.published = append(b.published, capturedEvent{subject: subject, payload: evt.Payload})
	return nil
}

func (b *captureEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (b *captureEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (b *captureEventBus) Close() error { return nil }

func TestServerService_SetActive_PublishesEvent(t *testing.T) {
	repo := &fakeServerRepo{servers: []Server{
		{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: true},
		{Role: ServerRoleStandby, Host: "10.0.0.2", Port: 8082, IsActive: false},
	}}
	bus := &captureEventBus{}
	svc := NewServerService(repo, bus, zap.NewNop())

	err := svc.SetActive(context.Background(), ServerRoleStandby)
	require.NoError(t, err)
	require.Len(t, bus.published, 1, "expect one event published")
	assert.Equal(t, event.SubjectNorthboundServerChanged, bus.published[0].subject)
	// payload 含 role=standby + action=active_switch
	assert.Contains(t, string(bus.published[0].payload), `"role":"standby"`)
	assert.Contains(t, string(bus.published[0].payload), `"action":"active_switch"`)
}

func TestServerService_Update_PublishesEvent(t *testing.T) {
	repo := &fakeServerRepo{servers: []Server{
		{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: true},
	}}
	bus := &captureEventBus{}
	svc := NewServerService(repo, bus, zap.NewNop())

	err := svc.Update(context.Background(), ServerRolePrimary, UpdateServerRequest{
		Host: "10.0.0.99", Port: 9999,
	})
	require.NoError(t, err)
	require.Len(t, bus.published, 1)
	assert.Contains(t, string(bus.published[0].payload), `"action":"update"`)
	assert.Contains(t, string(bus.published[0].payload), `"host":"10.0.0.99"`)
	assert.Contains(t, string(bus.published[0].payload), `"port":9999`)
}

func TestServerService_GetActiveForPush(t *testing.T) {
	t.Run("returns active info", func(t *testing.T) {
		repo := &fakeServerRepo{servers: []Server{
			{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: false},
			{Role: ServerRoleStandby, Host: "10.0.0.2", Port: 8082, IsActive: true},
		}}
		svc := NewServerService(repo, nil, zap.NewNop())

		info, err := svc.GetActiveForPush(context.Background())
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "standby", info.Role)
		assert.Equal(t, "10.0.0.2", info.Host)
		assert.Equal(t, 8082, info.Port)
	})
	t.Run("returns nil when no active", func(t *testing.T) {
		repo := &fakeServerRepo{servers: []Server{
			{Role: ServerRolePrimary, Host: "10.0.0.1", Port: 8081, IsActive: false},
		}}
		svc := NewServerService(repo, nil, zap.NewNop())
		info, err := svc.GetActiveForPush(context.Background())
		require.NoError(t, err)
		assert.Nil(t, info)
	})
}
