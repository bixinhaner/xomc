package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// tokenRevokerKeyPrefix Redis key 前缀，存储某用户最近一次 ForceLogout 时间戳（Unix 微秒）。
// JWT 中间件每次验证 access token 后查询该 key：若 token.issued_at_us < 该时间戳，token 视为失效。
const tokenRevokerKeyPrefix = "auth:revoked_at:user:"

// TokenRevoker 通过在 Redis 写入"用户级撤销时间戳"实现 JWT 强制下线。
// TTL 与 refresh token 寿命对齐：超过该时间后所有受影响 token 已自然过期，无需保留撤销记录。
type TokenRevoker struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewTokenRevoker 构造 TokenRevoker。
// refreshTTL 通常等于 JWTService.refreshTokenTTL，确保撤销标记寿命覆盖最长 token 寿命。
func NewTokenRevoker(rdb redis.UniversalClient, refreshTTL time.Duration) *TokenRevoker {
	return &TokenRevoker{rdb: rdb, ttl: refreshTTL}
}

// Revoke 把"撤销时间戳"写入用户撤销 key；之前签发的所有 token 在中间件下次校验时会被拒绝。
// rdb 为 nil 时静默成功（便于测试 / 降级运行）。
//
// 时间戳取 now（微秒），配合 IsRevoked 的严格小于 `issuedAtMicros < val` 比较，实现：
//   - 撤销时刻之前签发的 token（issued_at_us < now_us）→ 判为已撤销 ✓（管理员强制下线生效）
//   - 与撤销同一微秒签发的 token（issued_at_us == now_us）→ `now < now` 为假 → 不被撤销 ✓（单点登录保命）
//
// 单点登录先写撤销时间戳、再签发新 token，两步在微秒级几乎总能区分；即使完全同微秒，
// 严格小于比较仍可保证新 token 不会被自己踢下线。旧 token 若缺少 issued_at_us，
// IsRevoked 会回退到 JWT iat 秒级值换算为微秒继续兼容。
func (r *TokenRevoker) Revoke(ctx context.Context, userID uuid.UUID) error {
	if r == nil || r.rdb == nil {
		return nil
	}
	key := tokenRevokerKeyPrefix + userID.String()
	revokedAt := time.Now().UnixMicro()
	if err := r.rdb.Set(ctx, key, revokedAt, r.ttl).Err(); err != nil {
		return fmt.Errorf("set revocation: %w", err)
	}
	return nil
}

// RevokeBatch 批量调用 Revoke，遇到错误立即返回（前面已写入的不回滚 — Redis 命令本身已生效）。
func (r *TokenRevoker) RevokeBatch(ctx context.Context, userIDs []uuid.UUID) error {
	for _, id := range userIDs {
		if err := r.Revoke(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// IsRevoked 判定某 token 是否已被撤销。
// rdb 为 nil 或 key 不存在时返回 (false, nil)，对绝大多数请求零额外延迟。
func (r *TokenRevoker) IsRevoked(ctx context.Context, userID uuid.UUID, issuedAt, issuedAtMicros int64) (bool, error) {
	if r == nil || r.rdb == nil {
		return false, nil
	}
	key := tokenRevokerKeyPrefix + userID.String()
	val, err := r.rdb.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("get revocation: %w", err)
	}
	tokenIssuedAt := issuedAtMicros
	if tokenIssuedAt == 0 {
		tokenIssuedAt = issuedAt * int64(time.Second/time.Microsecond)
	}
	// 用严格小于：撤销前签发的 token 被踢；单点登录紧随撤销签发的新 token 不会被自己踢。
	return tokenIssuedAt < val, nil
}
