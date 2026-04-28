package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newRedisStoreFor(t *testing.T) (*RedisMessageStore, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisMessageStore(client, time.Hour)
	store.SetLogger(zap.NewNop())
	return store, s
}

func TestRedisStore_StoreAndGetSince_AllMessages(t *testing.T) {
	store, _ := newRedisStoreFor(t)
	ctx := context.Background()

	for _, id := range []string{"m1", "m2", "m3"} {
		require.NoError(t, store.Store(ctx, "u1", &SSEMessage{
			ID:    id,
			Event: "test",
			Data:  json.RawMessage(`{}`),
		}))
		// Sleep is unnecessary because miniredis monotonically advances
		// per command, but we use distinct IDs which is enough.
	}

	got, err := store.GetSince(ctx, "u1", "", 10)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "m1", got[0].ID)
	assert.Equal(t, "m3", got[2].ID)
}

func TestRedisStore_GetSince_AfterIDFiltersCorrectly(t *testing.T) {
	store, _ := newRedisStoreFor(t)
	ctx := context.Background()

	for _, id := range []string{"m1", "m2", "m3", "m4"} {
		require.NoError(t, store.Store(ctx, "u2", &SSEMessage{ID: id}))
	}

	got, err := store.GetSince(ctx, "u2", "m2", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "m3", got[0].ID)
	assert.Equal(t, "m4", got[1].ID)
}

func TestRedisStore_GetSince_LimitRespected(t *testing.T) {
	store, _ := newRedisStoreFor(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, store.Store(ctx, "u3", &SSEMessage{ID: string(rune('a' + i))}))
	}

	got, err := store.GetSince(ctx, "u3", "", 3)
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestRedisStore_GetSince_EmptyUserReturnsNil(t *testing.T) {
	store, _ := newRedisStoreFor(t)
	got, err := store.GetSince(context.Background(), "no-such", "", 10)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisStore_TTLApplied(t *testing.T) {
	store, mr := newRedisStoreFor(t)
	ctx := context.Background()

	require.NoError(t, store.Store(ctx, "u4", &SSEMessage{ID: "x"}))

	// Verify TTL was set on the key.
	ttl := mr.TTL("sse:pending:u4")
	assert.True(t, ttl > 0, "expected positive TTL on pending key")
}
