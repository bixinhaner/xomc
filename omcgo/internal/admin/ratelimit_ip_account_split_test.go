package admin

// issue #220 回归测试：登录限流"IP 连坐"问题的两项最小修复
//   ① 抬高 IP 限流阈值，使其明显高于账号锁定阈值（IP 不再先于账号触发）
//   ② 拆分文案：IP 闸门说"当前网络访问过于频繁"、账号锁定说"该账号已锁定"
//
// 死判项：
//   [threshold]    默认 IP 限流阈值 > 默认账号锁定阈值
//   [b-not-locked] 同 IP 连错触发 IP 闸门后，另一账号 B 首次登录收到"网络访问过于频繁"
//                  文案（biz_code 7014）而非账号锁定文案（7012）
//   [isolation]    账号维度隔离不回归：A 的失败计数不会污染 B 的计数

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// ─── 死判 threshold ────────────────────────────────────────────────────────
// 默认 IP 限流阈值必须明显高于默认账号锁定阈值，确保正常人偶尔输错几次
// 不会先把整个来源 IP 拉黑，而是先撞上"账号锁定"这道更窄的闸门。
func TestIPThresholdAboveAccountLockThreshold(t *testing.T) {
	p := defaultPolicy()
	assert.Greater(t, p.IPLimitCount, p.LockThreshold,
		"IP 限流阈值必须高于账号锁定阈值，否则 IP 闸门会先于账号锁定触发，导致同 IP 连坐")
}

// loginErrMsg 从登录失败响应体里取 msg + biz_code，供文案断言。
func loginErrMsg(t *testing.T, body []byte) (msg string, bizCode int) {
	t.Helper()
	var resp commonerrors.ErrorResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp.Msg, resp.BizCode
}

// newSplitTestRouter 构造一个带真实（miniredis）IPGuard 的登录路由。
// ipLimitCount=触发 IP 锁所需失败次数；密码恒为 "correct"，所有用户都存在。
func newSplitTestRouter(t *testing.T, ipLimitCount int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, username string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				Username:     username,
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
			}, nil
		},
	}
	roleRepo := &handlerMockRoleRepo{
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]Role, error) {
			return []Role{{Name: "admin"}}, nil
		},
	}

	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, roleRepo, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.SetAllowPlaintextPassword(true)

	ipPolicy := NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
		"security.limitMinus": "1",
		"security.limitCount": strconv.Itoa(ipLimitCount),
		"security.limitTimes": "30",
	}})
	ipGuard := NewIPGuard(client)
	ipGuard.SetPolicy(ipPolicy)
	h.SetIPGuard(ipGuard)

	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	return r
}

func doLogin(t *testing.T, r *gin.Engine, username, ip string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		handlerJSON(LoginRequest{Username: username, Password: "wrong"}))
	req.Header.Set("Content-Type", "application/json")
	// gin ClientIP 默认取 RemoteAddr；显式设定来源 IP。
	req.RemoteAddr = ip + ":12345"
	r.ServeHTTP(w, req)
	return w
}

// ─── 死判 b-not-locked ─────────────────────────────────────────────────────
// 同一来源 IP 下，A 连错触发 IP 闸门后，账号 B 首次登录收到的应是
// "当前网络访问过于频繁"（biz_code 7014），而非账号锁定文案。
func TestSameIP_BGetsNetworkBusy_NotAccountLock(t *testing.T) {
	const ipThreshold = 3
	const ip = "203.0.113.7"
	r := newSplitTestRouter(t, ipThreshold)

	// A 连续输错 ipThreshold 次 → 该 IP 进黑名单
	for i := 0; i < ipThreshold; i++ {
		w := doLogin(t, r, "alice", ip)
		// 前 ipThreshold 次是密码错误（401），第 ipThreshold 次失败后 IP 被锁
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"A 的前 %d 次应是密码错误 401", ipThreshold)
	}

	// B 从同一 IP 首次登录 → 一进入口就撞上 IP 黑名单
	w := doLogin(t, r, "bob", ip)
	require.Equal(t, http.StatusTooManyRequests, w.Code,
		"同 IP 下 B 首次登录应被 IP 闸门挡住（429）")

	msg, bizCode := loginErrMsg(t, w.Body.Bytes())
	assert.Contains(t, msg, "当前网络访问过于频繁",
		"B 收到的应是 IP 闸门文案而非账号锁定文案")
	assert.Equal(t, 7014, bizCode, "IP 闸门 biz_code 应为 7014")
	assert.NotContains(t, msg, "该账号已锁定",
		"B 的账号是干净的，不应出现账号锁定文案")
}

// ─── 死判 isolation ────────────────────────────────────────────────────────
// 账号维度隔离不回归：LoginGuard 按用户名独立计数，A 的失败不污染 B。
func TestLoginGuard_PerUsernameIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	lg := NewLoginGuard(client)
	lg.SetPolicy(NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
		"security.sumTimes":   "5",
		"security.unlockMinu": "30",
	}}))
	ctx := context.Background()

	// A 失败 5 次
	var aCount int64
	for i := 0; i < 5; i++ {
		c, err := lg.RecordFailure(ctx, "alice")
		require.NoError(t, err)
		aCount = c
	}
	assert.Equal(t, int64(5), aCount, "A 计数累加到 5")
	assert.True(t, lg.ShouldLock(ctx, aCount), "A 达阈值应锁")

	// B 首次失败 → 计数应为 1（不受 A 的 5 次影响）
	bCount, err := lg.RecordFailure(ctx, "bob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), bCount, "B 的失败计数独立，不被 A 污染")
	assert.False(t, lg.ShouldLock(ctx, bCount), "B 仅 1 次不应锁")
}

func TestLockedAccount_LoginDoesNotIncrementFailureCount(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	const username = "admin"
	lockedUntil := time.Now().Add(30 * time.Minute)
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, gotUsername string) (*User, error) {
			assert.Equal(t, username, gotUsername)
			return &User{
				ID:           uuid.New(),
				Username:     username,
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
				LockedUntil:  &lockedUntil,
			}, nil
		},
	}
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, &handlerMockRoleRepo{}, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.SetAllowPlaintextPassword(true)

	loginGuard := NewLoginGuard(client)
	loginGuard.SetPolicy(NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{
		"security.sumTimes":   "5",
		"security.unlockMinu": "30",
	}}))
	h.SetLoginGuard(loginGuard)
	for i := 0; i < 5; i++ {
		_, err := loginGuard.RecordFailure(context.Background(), username)
		require.NoError(t, err)
	}

	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	w := doLogin(t, r, username, "203.0.113.74")
	require.Equal(t, http.StatusForbidden, w.Code)
	_, bizCode := loginErrMsg(t, w.Body.Bytes())
	assert.Equal(t, 7012, bizCode)
	assert.Equal(t, int64(5), loginGuard.GetFailedCount(context.Background(), username),
		"锁定期间再次登录不得增加失败计数，否则会再次触发 LockUserByUsername 并延长锁定")
}

func TestWrongPassword_LoginStillIncrementsFailureCount(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	const username = "alice"
	userRepo := &handlerMockUserRepo{
		getByUsernameFn: func(_ context.Context, gotUsername string) (*User, error) {
			return &User{
				ID:           uuid.New(),
				Username:     gotUsername,
				PasswordHash: handlerHashPassword("correct"),
				Status:       UserStatusActive,
			}, nil
		},
	}
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(userRepo, &handlerMockRoleRepo{}, &handlerMockMenuRepo{}, &handlerMockAuditRepo{}, jwt, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.SetAllowPlaintextPassword(true)
	loginGuard := NewLoginGuard(client)
	h.SetLoginGuard(loginGuard)

	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	w := doLogin(t, r, username, "203.0.113.75")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, int64(1), loginGuard.GetFailedCount(context.Background(), username),
		"真实密码错误仍必须进入账号级暴力破解计数")
}
