package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisNonceStore(t *testing.T, mr *miniredis.Miniredis, ttl time.Duration) NonceStore {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return NewRedisNonceStore(rdb, ttl)
}

// TestRedisNonceStore_GetDelOneTime 验证 Redis nonce store 的一次性消费语义：
// 第一次 Consume 命中，第二次同一 nonce 必失败（GETDEL 原子删除）。
func TestRedisNonceStore_GetDelOneTime(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := newTestRedisNonceStore(t, mr, 5*time.Minute)

	s.Store(ctx, "nonce-abc")
	assert.True(t, s.Consume(ctx, "nonce-abc"), "first consume must hit")
	assert.False(t, s.Consume(ctx, "nonce-abc"), "second consume of same nonce must fail (one-time)")
}

// TestRedisNonceStore_UnknownNonce 验证未写入的 nonce 直接拒绝。
func TestRedisNonceStore_UnknownNonce(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := newTestRedisNonceStore(t, mr, 5*time.Minute)
	assert.False(t, s.Consume(ctx, "never-issued"))
}

// TestDigestAuth_CrossInstance_ChallengeOnA_AuthenticateOnB 是 issue #65 的核心场景：
// 在无亲和部署下 Challenge 落在实例 A、Authenticate 落在实例 B，两实例共享同一 Redis，
// nonce 不再丢失 → 认证通过；且二次使用同一 nonce 失败（一次性）。
func TestDigestAuth_CrossInstance_ChallengeOnA_AuthenticateOnB(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	// 两个独立的 ACS 实例，各自的 DigestAuthenticator，但共享同一 Redis nonce store。
	instanceA := NewAuthenticatorWithRedis("digest", "cpe-user", "cpe-pass", rdb)
	instanceB := NewAuthenticatorWithRedis("digest", "cpe-user", "cpe-pass", rdb)

	// 1) Challenge 落在 A，取回 nonce。
	wA := httptest.NewRecorder()
	instanceA.Challenge(wA)
	nonce := extractDigestField(wA.Header().Values("WWW-Authenticate")[1], "nonce") // MD5 header
	require.NotEmpty(t, nonce)

	// 2) CPE 用该 nonce 计算 MD5 digest 响应，POST 到 B。
	uri := "/acs"
	method := http.MethodPost
	realm := "ACS"
	ha1 := testMD5(fmt.Sprintf("%s:%s:%s", "cpe-user", realm, "cpe-pass"))
	ha2 := testMD5(fmt.Sprintf("%s:%s", method, uri))
	response := testMD5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	authHeader := fmt.Sprintf(
		`Digest username="cpe-user", realm="%s", nonce="%s", uri="%s", response="%s", algorithm=MD5`,
		realm, nonce, uri, response,
	)
	r := httptest.NewRequest(method, uri, nil)
	r.Header.Set("Authorization", authHeader)

	// 3) Authenticate 落在 B —— 跨实例 nonce 命中，认证成功（修复前会 401 死循环）。
	identity, err := instanceB.Authenticate(r)
	require.NoError(t, err, "cross-instance digest auth must succeed via shared Redis nonce")
	require.NotNil(t, identity)
	assert.Equal(t, "cpe-user", identity.CredentialID)

	// 4) 同一 nonce 再次在任一实例使用必失败（一次性消费）。
	r2 := httptest.NewRequest(method, uri, nil)
	r2.Header.Set("Authorization", authHeader)
	_, err2 := instanceA.Authenticate(r2)
	assert.Error(t, err2, "reused nonce must be rejected (one-time)")
}
