package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/events"
)

// noopMessageStore implements events.MessageStore as a black hole for tests.
type noopMessageStore struct {
	storeErr error
	stored   []*events.SSEMessage
}

func (n *noopMessageStore) Store(_ context.Context, _ string, msg *events.SSEMessage) error {
	if n.storeErr != nil {
		return n.storeErr
	}
	n.stored = append(n.stored, msg)
	return nil
}

func (n *noopMessageStore) GetSince(_ context.Context, _ string, _ string, _ int) ([]*events.SSEMessage, error) {
	return nil, nil
}

func newTestService(repo Repository, hub *events.MessageHub) *Service {
	return NewService(repo, hub, zap.NewNop())
}

func newTestHub() (*events.MessageHub, *noopMessageStore) {
	store := &noopMessageStore{}
	hub := events.NewMessageHub(store, zap.NewNop())
	return hub, store
}

// ---------- NewService ----------

func TestNewService_Defaults(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop())
	require.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.Nil(t, svc.hub)
	assert.NotNil(t, svc.logger)
}

// ---------- List ----------

func TestService_List_OK(t *testing.T) {
	repo := newMockRepository()
	repo.seed(&Notification{UserID: "alice", Type: NotifTypeAlarm, Title: "t1"})
	repo.seed(&Notification{UserID: "alice", Type: NotifTypeSystem, Title: "t2"})
	repo.seed(&Notification{UserID: "bob", Type: NotifTypeAlarm, Title: "t3"})

	svc := newTestService(repo, nil)
	resp, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
}

func TestService_List_Error(t *testing.T) {
	repo := newMockRepository()
	repo.listErr = errBoom
	svc := newTestService(repo, nil)
	_, err := svc.List(context.Background(), NotificationFilter{UserID: "alice"})
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
}

// ---------- GetByID ----------

func TestService_GetByID_OK(t *testing.T) {
	repo := newMockRepository()
	n := repo.seed(&Notification{UserID: "alice", Title: "t"})
	svc := newTestService(repo, nil)

	got, err := svc.GetByID(context.Background(), n.ID)
	require.NoError(t, err)
	assert.Equal(t, n.ID, got.ID)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)
	_, err := svc.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrNotFound)
}

// ---------- CreateNotification ----------

func TestService_CreateNotification_NoHub(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	n := &Notification{
		UserID: "alice",
		Type:   NotifTypeAlarm,
		Title:  "hello",
	}
	created, err := svc.CreateNotification(context.Background(), n)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
}

func TestService_CreateNotification_WithHub(t *testing.T) {
	repo := newMockRepository()
	hub, store := newTestHub()
	svc := newTestService(repo, hub)

	n := &Notification{
		UserID: "alice",
		Type:   NotifTypeSystem,
		Title:  "Persisted",
	}
	created, err := svc.CreateNotification(context.Background(), n)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	// SSE is fire-and-forget but the store should have received the message
	// because the hub.Publish persists even when no channel is subscribed.
	require.Len(t, store.stored, 1)
	assert.Equal(t, "alice", store.stored[0].UserID)
	assert.Equal(t, "notification", store.stored[0].Event)
}

func TestService_CreateNotification_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = errBoom
	svc := newTestService(repo, nil)

	_, err := svc.CreateNotification(context.Background(), &Notification{UserID: "alice"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errBoom))
}

// ---------- MarkRead ----------

func TestService_MarkRead_OK(t *testing.T) {
	repo := newMockRepository()
	n := repo.seed(&Notification{UserID: "alice", Title: "t"})

	svc := newTestService(repo, nil)
	err := svc.MarkRead(context.Background(), n.ID, "alice")
	require.NoError(t, err)
}

func TestService_MarkRead_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	err := svc.MarkRead(context.Background(), uuid.New(), "alice")
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrNotFound)
}

// ---------- MarkAllRead ----------

func TestService_MarkAllRead_OK(t *testing.T) {
	repo := newMockRepository()
	repo.seed(&Notification{UserID: "alice", Title: "t1"})
	repo.seed(&Notification{UserID: "alice", Title: "t2"})

	svc := newTestService(repo, nil)
	err := svc.MarkAllRead(context.Background(), "alice")
	require.NoError(t, err)

	cnt, err := svc.GetUnreadCount(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, int64(0), cnt)
}

func TestService_MarkAllRead_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.markAllReadErr = errBoom
	svc := newTestService(repo, nil)
	err := svc.MarkAllRead(context.Background(), "alice")
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
}

// ---------- GetUnreadCount ----------

func TestService_GetUnreadCount(t *testing.T) {
	repo := newMockRepository()
	repo.seed(&Notification{UserID: "alice", IsRead: false})
	repo.seed(&Notification{UserID: "alice", IsRead: false})
	repo.seed(&Notification{UserID: "alice", IsRead: true})
	repo.seed(&Notification{UserID: "bob", IsRead: false})

	svc := newTestService(repo, nil)
	cnt, err := svc.GetUnreadCount(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, int64(2), cnt)
}

func TestService_GetUnreadCount_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.getUnreadCountErr = errBoom
	svc := newTestService(repo, nil)
	_, err := svc.GetUnreadCount(context.Background(), "alice")
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
}

// ---------- Delete ----------

func TestService_Delete_OK(t *testing.T) {
	repo := newMockRepository()
	n := repo.seed(&Notification{UserID: "alice"})

	svc := newTestService(repo, nil)
	err := svc.Delete(context.Background(), n.ID, "alice")
	require.NoError(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)
	err := svc.Delete(context.Background(), uuid.New(), "alice")
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrNotFound)
}

func TestService_Delete_WrongOwner(t *testing.T) {
	repo := newMockRepository()
	n := repo.seed(&Notification{UserID: "alice"})

	svc := newTestService(repo, nil)
	err := svc.Delete(context.Background(), n.ID, "bob")
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrNotFound)
}

// ---------- SendNotification ----------

func TestService_SendNotification_OK(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	err := svc.SendNotification(context.Background(), "alice", NotifTypeSystem, PriorityNormal, "title", "content", "/x")
	require.NoError(t, err)
	cnt, _ := svc.GetUnreadCount(context.Background(), "alice")
	assert.Equal(t, int64(1), cnt)
}

func TestService_SendNotification_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = errBoom
	svc := newTestService(repo, nil)
	err := svc.SendNotification(context.Background(), "alice", NotifTypeSystem, PriorityNormal, "t", "c", "")
	require.Error(t, err)
}

// ---------- SendGlobalNotification ----------

func TestService_SendGlobalNotification(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	err := svc.SendGlobalNotification(context.Background(), []string{"alice", "bob"}, NotifTypeSystem, PriorityHigh, "broadcast", "msg", "")
	require.NoError(t, err)

	for _, u := range []string{"alice", "bob"} {
		c, err := svc.GetUnreadCount(context.Background(), u)
		require.NoError(t, err)
		assert.Equal(t, int64(1), c)
	}
}

func TestService_SendGlobalNotification_PartialFailure_NoAbort(t *testing.T) {
	// Even when create fails for every recipient, SendGlobalNotification
	// should aggregate via warnings and return nil.
	repo := newMockRepository()
	repo.createErr = errBoom
	svc := newTestService(repo, nil)

	err := svc.SendGlobalNotification(context.Background(), []string{"a", "b", "c"}, NotifTypeAlarm, PriorityCritical, "t", "c", "")
	require.NoError(t, err)
}

// ---------- CreateAndBroadcast ----------

func TestService_CreateAndBroadcast_NoHub(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)

	n := &Notification{UserID: "alice", Type: NotifTypeAlarm, Title: "t"}
	err := svc.CreateAndBroadcast(context.Background(), n)
	require.NoError(t, err)
}

func TestService_CreateAndBroadcast_WithHub(t *testing.T) {
	repo := newMockRepository()
	hub, store := newTestHub()
	svc := newTestService(repo, hub)

	n := &Notification{UserID: "alice", Type: NotifTypeAlarm, Title: "t"}
	err := svc.CreateAndBroadcast(context.Background(), n)
	require.NoError(t, err)
	assert.Len(t, store.stored, 1)
}

func TestService_CreateAndBroadcast_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = errBoom
	svc := newTestService(repo, nil)
	err := svc.CreateAndBroadcast(context.Background(), &Notification{UserID: "alice"})
	require.Error(t, err)
}

// ---------- pushSSEEvent ----------

func TestService_pushSSEEvent_StoreError_LogsButNoPanic(t *testing.T) {
	// hub with store that fails — pushSSEEvent should swallow the error.
	store := &noopMessageStore{storeErr: errBoom}
	hub := events.NewMessageHub(store, zap.NewNop())

	repo := newMockRepository()
	svc := newTestService(repo, hub)

	n := &Notification{
		ID:     uuid.New(),
		UserID: "alice",
		Type:   NotifTypeSystem,
		Title:  "t",
	}
	// Direct call should not panic
	svc.pushSSEEvent(n)
}

// ---------- type / priority constants smoke test ----------

func TestNotificationConstants(t *testing.T) {
	// Ensure the exported constants have the expected wire values.
	assert.Equal(t, NotificationType("alarm"), NotifTypeAlarm)
	assert.Equal(t, NotificationType("task_complete"), NotifTypeTaskComplete)
	assert.Equal(t, NotificationType("system"), NotifTypeSystem)
	assert.Equal(t, NotificationType("approval"), NotifTypeApproval)
	assert.Equal(t, NotificationType("device_status"), NotifTypeDeviceStatus)

	assert.Equal(t, NotificationPriority("critical"), PriorityCritical)
	assert.Equal(t, NotificationPriority("high"), PriorityHigh)
	assert.Equal(t, NotificationPriority("normal"), PriorityNormal)
	assert.Equal(t, NotificationPriority("low"), PriorityLow)
}
