package agentruntime

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type memoryConversationStore struct {
	mu     sync.Mutex
	values map[string]admin.SysConfig
}

func newMemoryConversationStore() *memoryConversationStore {
	return &memoryConversationStore{values: map[string]admin.SysConfig{}}
}

func (s *memoryConversationStore) GetByKey(_ context.Context, category, key string) (*admin.SysConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.values[category+"\x00"+key]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return &item, nil
}

func (s *memoryConversationStore) BatchUpsert(_ context.Context, category string, items []admin.BatchItem) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range items {
		s.values[category+"\x00"+item.Key] = admin.SysConfig{
			ID:        uuid.New(),
			Category:  category,
			Key:       item.Key,
			Value:     item.Value,
			ValueType: item.ValueType,
		}
	}
	return len(items), nil
}

func TestConversationServiceKeepsActiveConversationUntilRotated(t *testing.T) {
	ctx := context.Background()
	svc := NewConversationService(newMemoryConversationStore())
	claims := &admin.Claims{UserID: uuid.New(), Username: "operator"}

	first, err := svc.Active(ctx, "connector-1", claims)
	require.NoError(t, err)
	second, err := svc.Active(ctx, "connector-1", claims)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.True(t, isAgentConversationID(first))

	rotated, err := svc.Rotate(ctx, "connector-1", claims)
	require.NoError(t, err)
	require.NotEqual(t, first, rotated)

	afterRotate, err := svc.Active(ctx, "connector-1", claims)
	require.NoError(t, err)
	require.Equal(t, rotated, afterRotate)
}

func TestConversationServiceIsolatesConnectorAndUser(t *testing.T) {
	ctx := context.Background()
	store := newMemoryConversationStore()
	svc := NewConversationService(store)
	userA := &admin.Claims{UserID: uuid.New(), Username: "operator-a"}
	userB := &admin.Claims{UserID: uuid.New(), Username: "operator-b"}

	userAConnectorA, err := svc.Active(ctx, "connector-a", userA)
	require.NoError(t, err)
	userAConnectorB, err := svc.Active(ctx, "connector-b", userA)
	require.NoError(t, err)
	userBConnectorA, err := svc.Active(ctx, "connector-a", userB)
	require.NoError(t, err)

	require.NotEqual(t, userAConnectorA, userAConnectorB)
	require.NotEqual(t, userAConnectorA, userBConnectorA)
	require.NotContains(t, userAConnectorA, userA.Username)
	require.NotContains(t, userBConnectorA, userB.UserID.String())
}

func TestConversationServiceRepairsMissingInstanceIDWhenConversationExists(t *testing.T) {
	ctx := context.Background()
	store := newMemoryConversationStore()
	svc := NewConversationService(store)
	claims := &admin.Claims{UserID: uuid.New(), Username: "operator"}
	existingConversationID := conversationIDPrefix + "legacy_123"

	_, err := store.BatchUpsert(ctx, conversationCategory, []admin.BatchItem{
		{
			Key:       conversationConfigKey("connector-1", claims.UserID.String()),
			Value:     existingConversationID,
			ValueType: "string",
		},
	})
	require.NoError(t, err)

	active, err := svc.Active(ctx, "connector-1", claims)
	require.NoError(t, err)
	require.Equal(t, existingConversationID, active)

	instanceItem, err := store.GetByKey(ctx, conversationCategory, instanceConfigKey)
	require.NoError(t, err)
	require.True(t, isAgentInstanceID(instanceItem.Value))
}

func TestConversationServiceKeepsStableInstanceID(t *testing.T) {
	ctx := context.Background()
	store := newMemoryConversationStore()
	svc := NewConversationService(store)

	first, err := svc.InstanceID(ctx)
	require.NoError(t, err)
	second, err := svc.InstanceID(ctx)
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.True(t, isAgentInstanceID(first))
}
