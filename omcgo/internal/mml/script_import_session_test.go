package mml

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newRedisImportSessionStore(t *testing.T, ttl time.Duration) (*RedisImportSessionStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return NewRedisImportSessionStore(client, ttl), mr
}

func validImportSession() *ImportSession {
	return &ImportSession{
		OriginalFilename:  "script.txt",
		NormalizedContent: "LST DEVICE_INFO;SN1\n",
		ContentSHA256:     "abc",
		ValidationVersion: "v1",
		Validation: ScriptValidationResult{
			PlanItems: []MMLPlanItem{{LineNo: 1, DeviceSN: "SN1", Order: 1, CommandCode: "LST DEVICE_INFO"}},
			Summary:   ScriptValidationSummary{TotalLines: 1, ValidLines: 1, DeviceCount: 1},
		},
	}
}

func TestRedisImportSessionStore_PutUsesRandomTokenHashedRedisKeyAndTTL(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisImportSessionStore(t, 15*time.Minute)

	token, err := store.Put(ctx, "alice", validImportSession())
	require.NoError(t, err)
	require.Len(t, token, 64)

	digest := sha256.Sum256([]byte(token))
	key := redisx.Keys.MMLScriptImportSession(hex.EncodeToString(digest[:]))
	require.True(t, mr.Exists(key))
	require.False(t, mr.Exists(redisx.Keys.MMLScriptImportSession(token)))
	require.Equal(t, 15*time.Minute, mr.TTL(key))

	session, err := store.Get(ctx, token, "alice")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, session.ID)
	require.Equal(t, validImportSession().NormalizedContent, session.NormalizedContent)
	require.Equal(t, validImportSession().Validation.PlanItems, session.Validation.PlanItems)
}

func TestRedisImportSessionStore_ClaimAndReleasePreserveEmptyJSONArrays(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisImportSessionStore(t, 15*time.Minute)
	session := validImportSession()
	session.Validation.PlanItems = make([]MMLPlanItem, 0)
	session.Validation.Issues = make([]ScriptIssue, 0)

	token, err := store.Put(ctx, "alice", session)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte(token))
	activeKey := redisx.Keys.MMLScriptImportSession(hex.EncodeToString(digest[:]))
	rawBefore, err := mr.Get(activeKey)
	require.NoError(t, err)
	require.Contains(t, rawBefore, `"plan_items":[]`)
	require.Contains(t, rawBefore, `"issues":[]`)

	claimed, err := store.Claim(ctx, token, "alice", "request-1")
	require.NoError(t, err)
	require.NotNil(t, claimed.Validation.PlanItems)
	require.Empty(t, claimed.Validation.PlanItems)
	require.NotNil(t, claimed.Validation.Issues)
	require.Empty(t, claimed.Validation.Issues)

	rawAfterClaim, err := mr.Get(activeKey)
	require.NoError(t, err)
	require.Equal(t, rawBefore, rawAfterClaim)

	require.NoError(t, store.Release(ctx, token, "alice", "request-1"))
	released, err := store.Get(ctx, token, "alice")
	require.NoError(t, err)
	require.NotNil(t, released.Validation.PlanItems)
	require.Empty(t, released.Validation.PlanItems)
	require.NotNil(t, released.Validation.Issues)
	require.Empty(t, released.Validation.Issues)
	rawAfterRelease, err := mr.Get(activeKey)
	require.NoError(t, err)
	require.Equal(t, rawBefore, rawAfterRelease)
}

func TestRedisImportSessionStore_ActiveAndConsumedKeysShareRedisClusterHashTag(t *testing.T) {
	digest := strings.Repeat("a", 64)
	activeKey := redisx.Keys.MMLScriptImportSession(digest)
	consumedKey := redisx.Keys.MMLScriptImportConsumed(digest)
	claimKey := redisx.Keys.MMLScriptImportClaim(digest)

	require.Equal(t, digest, redisClusterHashTag(activeKey))
	require.Equal(t, redisClusterHashTag(activeKey), redisClusterHashTag(consumedKey))
	require.Equal(t, redisClusterHashTag(activeKey), redisClusterHashTag(claimKey))
	require.NotContains(t, activeKey, "raw-token")
	require.NotContains(t, consumedKey, "raw-token")
	require.NotContains(t, claimKey, "raw-token")
}

func redisClusterHashTag(key string) string {
	start := strings.IndexByte(key, '{')
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(key[start+1:], '}')
	if end < 0 {
		return ""
	}
	return key[start+1 : start+1+end]
}

func TestRedisImportSessionStore_ClaimOnceAndBindUser(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisImportSessionStore(t, 15*time.Minute)
	token, err := store.Put(ctx, "alice", validImportSession())
	require.NoError(t, err)

	_, err = store.Get(ctx, token, "bob")
	require.ErrorIs(t, err, ErrImportTokenOwnerMismatch)

	session, err := store.Claim(ctx, token, "alice", "request-1")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, session.ID)
	_, err = store.Claim(ctx, token, "alice", "request-2")
	require.ErrorIs(t, err, ErrImportTokenClaimed)

	require.NoError(t, store.Finalize(ctx, token, "alice", "request-1"))
	_, err = store.Claim(ctx, token, "alice", "request-1")
	require.ErrorIs(t, err, ErrImportTokenConsumed)
}

func TestRedisImportSessionStore_ReleaseOnlyAllowsClaimingRequest(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisImportSessionStore(t, 15*time.Minute)
	token, err := store.Put(ctx, "alice", validImportSession())
	require.NoError(t, err)
	_, err = store.Claim(ctx, token, "alice", "request-1")
	require.NoError(t, err)

	err = store.Release(ctx, token, "alice", "request-2")
	require.ErrorIs(t, err, ErrImportTokenClaimed)
	_, err = store.Claim(ctx, token, "alice", "request-2")
	require.ErrorIs(t, err, ErrImportTokenClaimed)

	require.NoError(t, store.Release(ctx, token, "alice", "request-1"))
	_, err = store.Claim(ctx, token, "alice", "request-2")
	require.NoError(t, err)
}

func TestRedisImportSessionStore_Expires(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisImportSessionStore(t, time.Minute)
	token, err := store.Put(ctx, "alice", validImportSession())
	require.NoError(t, err)

	mr.FastForward(time.Minute)
	_, err = store.Get(ctx, token, "alice")
	require.ErrorIs(t, err, ErrImportTokenExpired)
}

func TestRedisImportSessionStore_ConcurrentClaimAllowsExactlyOneRequest(t *testing.T) {
	ctx := context.Background()
	store, _ := newRedisImportSessionStore(t, 15*time.Minute)
	token, err := store.Put(ctx, "alice", validImportSession())
	require.NoError(t, err)

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, requestID := range []string{"request-1", "request-2"} {
		wg.Add(1)
		go func(requestID string) {
			defer wg.Done()
			_, err := store.Claim(ctx, token, "alice", requestID)
			results <- err
		}(requestID)
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		require.ErrorIs(t, err, ErrImportTokenClaimed)
	}
	require.Equal(t, 1, successes)
}

func TestRedisImportSessionStore_FinalizeCreatesConsumedTombstoneWithTTL(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisImportSessionStore(t, 15*time.Minute)
	session := validImportSession()
	session.Validation.PlanItems = make([]MMLPlanItem, 0)
	session.Validation.Issues = make([]ScriptIssue, 0)
	token, err := store.Put(ctx, "alice", session)
	require.NoError(t, err)
	_, err = store.Claim(ctx, token, "alice", "request-1")
	require.NoError(t, err)
	require.NoError(t, store.Finalize(ctx, token, "alice", "request-1"))

	digest := sha256.Sum256([]byte(token))
	tombstoneKey := redisx.Keys.MMLScriptImportConsumed(hex.EncodeToString(digest[:]))
	require.True(t, mr.Exists(tombstoneKey))
	require.Equal(t, 15*time.Minute, mr.TTL(tombstoneKey))
	_, err = store.Get(ctx, token, "alice")
	require.ErrorIs(t, err, ErrImportTokenConsumed)
	consumed, err := store.GetConsumed(ctx, token, "alice")
	require.NoError(t, err)
	require.NotNil(t, consumed.Validation.PlanItems)
	require.Empty(t, consumed.Validation.PlanItems)
	require.NotNil(t, consumed.Validation.Issues)
	require.Empty(t, consumed.Validation.Issues)
}
