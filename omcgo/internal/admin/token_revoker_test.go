package admin

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRevoker(t *testing.T) (*TokenRevoker, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewTokenRevoker(client, time.Hour), mr
}

func Test_TokenRevoker_NilSafe(t *testing.T) {
	// rdb 为 nil（降级 / 测试）时 IsRevoked 静默返回 false，不 panic。
	var r *TokenRevoker
	revoked, err := r.IsRevoked(context.Background(), uuid.New(), time.Now().Unix())
	require.NoError(t, err)
	assert.False(t, revoked)
}

func Test_TokenRevoker_NoRecordNotRevoked(t *testing.T) {
	// 失败路径：从未撤销过的用户，任何 token 都不应被判为已撤销。
	r, _ := newTestRevoker(t)
	revoked, err := r.IsRevoked(context.Background(), uuid.New(), time.Now().Unix())
	require.NoError(t, err)
	assert.False(t, revoked)
}

func Test_TokenRevoker_IsRevoked_Boundary(t *testing.T) {
	// 核心回归（issue #139 / #141）：撤销时间戳取 now（见 Revoke），配合 `iat<val` 严格小于，
	// 边界语义为「严格早于撤销时刻的 token 被踢（force-logout 生效），与撤销同秒签发的 token
	// 不被踢（单点登录保命）」。存储值 val 即撤销时刻 now。
	r, _ := newTestRevoker(t)
	ctx := context.Background()
	userID := uuid.New()

	require.NoError(t, r.Revoke(ctx, userID))

	// 取回 Redis 里写入的撤销时间戳 val（= 撤销时刻 now）。
	key := tokenRevokerKeyPrefix + userID.String()
	val, err := r.rdb.Get(ctx, key).Int64()
	require.NoError(t, err)
	now := val // 撤销时刻所在秒

	tests := []struct {
		name     string
		issuedAt int64
		want     bool
	}{
		{"iat 早于撤销秒前一秒 → 已撤销", now - 2, true},
		{"iat 恰为撤销时刻前一秒 → 已撤销", now - 1, true},
		{"iat 与撤销同秒（新 token 自踢场景） → 不撤销", now, false},
		{"iat 晚于撤销秒 → 不撤销", now + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.IsRevoked(ctx, userID, tt.issuedAt)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_TokenRevoker_SameSecondLoginNotSelfKicked(t *testing.T) {
	// 单点登录闭环回归（issue #139）：撤销与新 token 签发落在同一秒（iat == 撤销时刻 now），
	// 新 token 不得被自己刚写的撤销记录误判已撤销，否则登录后首请求 401。
	r, _ := newTestRevoker(t)
	ctx := context.Background()
	userID := uuid.New()

	// 模拟 service.go Login 单点登录：先撤销该用户现存 token。
	require.NoError(t, r.Revoke(ctx, userID))

	// 还原撤销发生的那一秒（存储值即撤销时刻 now）。
	key := tokenRevokerKeyPrefix + userID.String()
	revokedAt, err := r.rdb.Get(ctx, key).Int64()
	require.NoError(t, err)
	now := revokedAt

	// 同一秒签发的新 token（iat == now）不应被撤销。
	revoked, err := r.IsRevoked(ctx, userID, now)
	require.NoError(t, err)
	assert.False(t, revoked, "单点登录同秒签发的新 token 不应被自己踢下线")

	// 而撤销前（前一秒及更早）签发的旧 token 仍被踢。
	revokedOld, err := r.IsRevoked(ctx, userID, now-1)
	require.NoError(t, err)
	assert.True(t, revokedOld, "撤销前签发的旧 token 应被踢下线")
}

func Test_TokenRevoker_RevokeBatch(t *testing.T) {
	// 批量撤销后，每个用户在撤销时刻前签发的 token 都失效；同秒签发的不失效。
	r, _ := newTestRevoker(t)
	ctx := context.Background()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	require.NoError(t, r.RevokeBatch(ctx, ids))

	for _, id := range ids {
		key := tokenRevokerKeyPrefix + id.String()
		val, err := r.rdb.Get(ctx, key).Int64()
		require.NoError(t, err)
		now := val
		// 撤销前签发（iat < now）失效。
		gotOld, err := r.IsRevoked(ctx, id, now-1)
		require.NoError(t, err)
		assert.True(t, gotOld, "撤销前签发的 token 应失效")
		// 同秒签发（iat == now）不失效。
		gotNew, err := r.IsRevoked(ctx, id, now)
		require.NoError(t, err)
		assert.False(t, gotNew, "撤销同秒签发的新 token 不应失效")
	}
}
