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

func Test_TokenRevoker_IsRevoked_ClockSkewBoundary(t *testing.T) {
	// 核心回归（issue #6 token_revoker.go:67 由 < 改为 <=）：
	// 撤销时间戳与 token.iat 同为秒级 Unix 时间，二者相等（同一秒签发与撤销）
	// 必须判为已撤销，消除最长 1 秒的绕过窗口。
	r, _ := newTestRevoker(t)
	ctx := context.Background()
	userID := uuid.New()

	require.NoError(t, r.Revoke(ctx, userID))

	// 取回 Redis 里写入的撤销时间戳，构造三个边界。
	key := tokenRevokerKeyPrefix + userID.String()
	revokedAt, err := r.rdb.Get(ctx, key).Int64()
	require.NoError(t, err)

	tests := []struct {
		name     string
		issuedAt int64
		want     bool
	}{
		{"iat 早于撤销 → 已撤销", revokedAt - 1, true},
		{"iat 等于撤销秒（边界） → 已撤销", revokedAt, true},
		{"iat 晚于撤销 → 未撤销", revokedAt + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.IsRevoked(ctx, userID, tt.issuedAt)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_TokenRevoker_RevokeBatch(t *testing.T) {
	// 批量撤销后，每个用户在撤销秒（含边界）签发的 token 都失效。
	r, _ := newTestRevoker(t)
	ctx := context.Background()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	require.NoError(t, r.RevokeBatch(ctx, ids))

	for _, id := range ids {
		key := tokenRevokerKeyPrefix + id.String()
		revokedAt, err := r.rdb.Get(ctx, key).Int64()
		require.NoError(t, err)
		got, err := r.IsRevoked(ctx, id, revokedAt) // 边界相等
		require.NoError(t, err)
		assert.True(t, got, "撤销同秒签发的 token 应失效")
	}
}
