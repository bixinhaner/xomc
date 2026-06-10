package nedirect

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// IDOR 回归测试：不同操作员凭 session_id 横向越权必须被拒。

func principalFor(id uuid.UUID, super bool) *Principal {
	return &Principal{UserID: id, Username: "u-" + id.String()[:8], IsSuperAdmin: super}
}

// 一个属于 owner 用户的活跃会话。
func ownedSession(sessionID uuid.UUID, ownerID uuid.UUID) *Session {
	return &Session{
		ID:       sessionID,
		DeviceSN: "SN-CMCC-001",
		UserID:   ownerID.String(),
		Status:   SessionActive,
	}
}

func TestDisconnect_IDOR_OtherUserDenied(t *testing.T) {
	owner := uuid.New()
	attacker := uuid.New()
	sessionID := uuid.New()

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Session, error) {
			return ownedSession(id, owner), nil
		},
	}
	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	// attacker（非超管）断开 owner 的会话 → 越权拒绝。
	err := svc.Disconnect(context.Background(), principalFor(attacker, false), sessionID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSessionOwnership))
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden))
	// 不应产生断开事件（操作未执行）。
	assert.Empty(t, eventBus.published)
}

func TestDisconnect_IDOR_OwnerAllowed(t *testing.T) {
	owner := uuid.New()
	sessionID := uuid.New()

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Session, error) {
			return ownedSession(id, owner), nil
		},
	}
	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	err := svc.Disconnect(context.Background(), principalFor(owner, false), sessionID)
	require.NoError(t, err)
}

func TestDisconnect_IDOR_SuperAdminAllowed(t *testing.T) {
	owner := uuid.New()
	superAdmin := uuid.New()
	sessionID := uuid.New()

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Session, error) {
			return ownedSession(id, owner), nil
		},
	}
	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	// 超管可断开任意会话。
	err := svc.Disconnect(context.Background(), principalFor(superAdmin, true), sessionID)
	require.NoError(t, err)
}

func TestSendCommand_IDOR_OtherUserDenied(t *testing.T) {
	owner := uuid.New()
	attacker := uuid.New()
	sessionID := uuid.New()

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Session, error) {
			return ownedSession(id, owner), nil
		},
	}
	svc, eventBus := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	cmd, err := svc.SendCommand(context.Background(), principalFor(attacker, false), sessionID, "RST BTS;")
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.True(t, errors.Is(err, ErrSessionOwnership))
	// 命令未下发，无事件。
	assert.Empty(t, eventBus.published)
}

func TestSendCommand_IDOR_OwnerAllowed(t *testing.T) {
	owner := uuid.New()
	sessionID := uuid.New()

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*Session, error) {
			return ownedSession(id, owner), nil
		},
	}
	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	cmd, err := svc.SendCommand(context.Background(), principalFor(owner, false), sessionID, "DSP CELLINFO;")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "DSP CELLINFO;", cmd.CommandStr)
}

// ListSessions 对非超管强制收紧到本人 user_id —— 防止越权枚举他人会话。
func TestListSessions_NonSuperAdmin_ScopedToSelf(t *testing.T) {
	caller := uuid.New()
	var capturedFilter SessionFilter
	sessionRepo := &mockSessionRepo{
		listFn: func(_ context.Context, f SessionFilter) (*model.ListResponse[Session], error) {
			capturedFilter = f
			return model.NewListResponse([]Session{}, 0, 1, 20), nil
		},
	}
	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	// 即便请求显式传了别人的 user_id，也会被覆盖为 caller。
	other := uuid.New().String()
	_, err := svc.ListSessions(context.Background(), principalFor(caller, false), SessionFilter{UserID: &other})
	require.NoError(t, err)
	require.NotNil(t, capturedFilter.UserID)
	assert.Equal(t, caller.String(), *capturedFilter.UserID)
}

// 超管不被收紧，可按任意条件列出。
func TestListSessions_SuperAdmin_NotScoped(t *testing.T) {
	superAdmin := uuid.New()
	var capturedFilter SessionFilter
	sessionRepo := &mockSessionRepo{
		listFn: func(_ context.Context, f SessionFilter) (*model.ListResponse[Session], error) {
			capturedFilter = f
			return model.NewListResponse([]Session{}, 0, 1, 20), nil
		},
	}
	svc, _ := newTestService(sessionRepo, &mockCommandRepo{}, nil)

	_, err := svc.ListSessions(context.Background(), principalFor(superAdmin, true), SessionFilter{})
	require.NoError(t, err)
	assert.Nil(t, capturedFilter.UserID, "超管不应被强制收紧 user_id")
}
