package admin

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 简化版 mock — 用 map 模拟 sys_configs 表。
type policyMockQuerier struct {
	entries map[string]string
}

func (m *policyMockQuerier) GetByKey(_ context.Context, category, key string) (*SysConfig, error) {
	if v, ok := m.entries[category+"."+key]; ok {
		return &SysConfig{ID: uuid.New(), Category: category, Key: key, Value: v}, nil
	}
	return nil, nil
}

// 21 个字段都按 FE 表单语义解码。
func TestSecurityPolicy_AllFieldsDecoded(t *testing.T) {
	q := &policyMockQuerier{
		entries: map[string]string{
			// ⑤
			"security.attemptTimes": "2",
			"security.sumTimes":     "6",
			"security.unlockMinu":   "15",
			"security.verifyEnable": "true",
			// ⑥
			"security.limitMinus": "5",
			"security.limitCount": "20",
			"security.limitTimes": "60",
			// ⑩
			"security.isOnlyOneUserLoginEnable": "false", // false = 单点登录
			// ②
			"security.passwordContent": "true",
			"security.pwdMinLength":    "12",
			"security.pwdMaxLength":    "64",
			// ④
			"security.expires":          "true",
			"security.validPeriod":      "30",
			"security.promptBeforeDays": "3",
			// ①
			"security.modifyPWD":     "true",
			"security.defaultPasswd": "MyDefault@2026",
			// ⑦
			"security.userSessionExpirationMin": "10",
			// ⑧
			"security.isBrowserAutoRecordPass": "true",
			// ⑨
			"security.autoLockUserDayEnable": "true",
			"security.autoLockUserDay":       "60",
			// ⑪
			"security.enabledFlag": "true",
			"security.msg":         "请遵守安全规范",
		},
	}
	p := NewSecurityPolicy(q)
	v := p.Get(context.Background())

	// ⑤
	assert.Equal(t, int64(2), v.CaptchaThreshold)
	assert.Equal(t, int64(6), v.LockThreshold)
	assert.Equal(t, 15*time.Minute, v.LockDuration)
	assert.True(t, v.VerifyEnable)
	// ⑥
	assert.Equal(t, 5*time.Minute, v.IPLimitWindow)
	assert.Equal(t, int64(20), v.IPLimitCount)
	assert.Equal(t, 60*time.Minute, v.IPLockMinutes)
	// ⑩
	assert.False(t, v.AllowConcurrent, "isOnlyOneUserLoginEnable=false → 单点登录")
	// ②
	assert.True(t, v.PasswordPolicyEnabled)
	assert.Equal(t, int64(12), v.PwdMinLength)
	assert.Equal(t, int64(64), v.PwdMaxLength)
	// ④
	assert.True(t, v.PasswordExpiresEnabled)
	assert.Equal(t, int64(30), v.PasswordValidDays)
	assert.Equal(t, int64(3), v.PasswordPromptDays)
	// ①
	assert.True(t, v.MustChangePasswordOnFirstLogin)
	assert.Equal(t, "MyDefault@2026", v.DefaultPassword)
	// ⑦
	assert.Equal(t, int64(10), v.IdleLockMinutes)
	// ⑧
	assert.True(t, v.PreventBrowserAutofill)
	// ⑨
	assert.True(t, v.AutoLockUnusedEnabled)
	assert.Equal(t, int64(60), v.AutoLockUnusedDays)
	// ⑪
	assert.True(t, v.LoginNotifyEnabled)
	assert.Equal(t, "请遵守安全规范", v.LoginNotifyMsg)
}

// 未注入 querier 时 21 个字段全部 default。
func TestSecurityPolicy_NoQuerier_Defaults(t *testing.T) {
	p := NewSecurityPolicy(nil)
	v := p.Get(context.Background())
	assert.Equal(t, int64(defaultCaptchaThreshold), v.CaptchaThreshold)
	assert.Equal(t, int64(defaultLockThreshold), v.LockThreshold)
	assert.Equal(t, defaultLockDuration, v.LockDuration)
	assert.False(t, v.VerifyEnable)
	assert.True(t, v.AllowConcurrent, "default 允许多端登录")
	assert.Equal(t, defaultIPLimitWindow, v.IPLimitWindow)
	assert.Equal(t, defaultDefaultPassword, v.DefaultPassword)
}

// 缓存命中 — InvalidateCache 后强制重新加载。
func TestSecurityPolicy_CacheInvalidation(t *testing.T) {
	q := &policyMockQuerier{
		entries: map[string]string{"security.sumTimes": "7"},
	}
	p := NewSecurityPolicy(q)
	ctx := context.Background()

	v1 := p.Get(ctx)
	assert.Equal(t, int64(7), v1.LockThreshold)

	// 修改"配置"，但缓存还没过期 — 返还原值
	q.entries["security.sumTimes"] = "99"
	v2 := p.Get(ctx)
	assert.Equal(t, int64(7), v2.LockThreshold, "缓存命中")

	// 失效后下次重新加载
	p.InvalidateCache()
	v3 := p.Get(ctx)
	assert.Equal(t, int64(99), v3.LockThreshold)
}

// 验证 verifyEnable=false 时 RequiresCaptcha 永远返 false（即使失败次数已超）。
func TestLoginGuard_RequiresCaptcha_DisabledByPolicy(t *testing.T) {
	q := &policyMockQuerier{
		entries: map[string]string{
			"security.verifyEnable": "false", // 总开关关
			"security.attemptTimes": "1",     // 失败 1 次本应弹验证码
		},
	}
	g := &LoginGuard{}
	g.SetPolicy(NewSecurityPolicy(q))

	// 由于无 Redis 也跑不了 GetFailedCount；这里只断言 verifyEnable=false 短路返 false
	assert.False(t, g.RequiresCaptcha(context.Background(), "anyone"),
		"verifyEnable=false 时 captcha 永远不弹")
}

func TestLoginGuard_RequiresCaptcha_EnabledByPolicy_RequiresRedis(t *testing.T) {
	// verifyEnable=true 且阈值低，但没 Redis 没法走 GetFailedCount；
	// 此处仅测 short-circuit 不在 verifyEnable=true 上发生。
	q := &policyMockQuerier{
		entries: map[string]string{
			"security.verifyEnable": "true",
			"security.attemptTimes": "1",
		},
	}
	g := &LoginGuard{}
	g.SetPolicy(NewSecurityPolicy(q))

	p := g.snapshot(context.Background())
	require.True(t, p.VerifyEnable)
	require.Equal(t, int64(1), p.CaptchaThreshold)
}

// "0 表示禁用，非 0 表示分钟" — userSessionExpirationMin 边界。
func TestSecurityPolicy_IdleLockMinutes_ZeroPreserved(t *testing.T) {
	// 显式设 0
	q := &policyMockQuerier{
		entries: map[string]string{"security.userSessionExpirationMin": "0"},
	}
	v := NewSecurityPolicy(q).Get(context.Background())
	// 0 是合法的"禁用屏幕锁"状态；不应被 fallback 覆盖
	// 实测 readInt(0) 返回 0 后 if n>=0 + 显式 GetByKey 确认存在 → 设置为 0
	assert.Equal(t, int64(0), v.IdleLockMinutes)

	// 未设置 → default
	v2 := NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{}}).Get(context.Background())
	assert.Equal(t, int64(defaultIdleLockMinutes), v2.IdleLockMinutes)
}

// 类型测试：所有 default 值为合理的"安全保守值"。
func TestSecurityPolicy_DefaultsAreSafe(t *testing.T) {
	v := defaultPolicy()
	assert.GreaterOrEqual(t, v.LockThreshold, int64(5), "锁阈值至少 5，避免误锁")
	assert.GreaterOrEqual(t, v.LockDuration.Minutes(), 10.0, "锁定时长至少 10 分钟")
	assert.GreaterOrEqual(t, v.PwdMinLength, int64(8), "密码至少 8 位")
	assert.True(t, v.AllowConcurrent, "默认允许多端 — 单点登录是高风险，需 admin 显式开")
	// captchaThreshold 不约束 — 取决于业务

	// 一致性：default 常量与 defaultPolicy() 字段对齐
	assert.Equal(t, int64(defaultLockThreshold), v.LockThreshold)
	assert.Equal(t, int64(defaultCaptchaThreshold), v.CaptchaThreshold)
}

// 防回归：FE 改一个数字未来如果不小心写 sumTimes=-5 这种负数，不能让锁机制失效。
func TestSecurityPolicy_NegativeValueIgnored(t *testing.T) {
	q := &policyMockQuerier{
		entries: map[string]string{"security.sumTimes": "-5"},
	}
	v := NewSecurityPolicy(q).Get(context.Background())
	assert.Equal(t, int64(defaultLockThreshold), v.LockThreshold,
		"负数被忽略，回到 default 10 — 不能让恶意/错误配置让账号永远锁不上")
}

// SecuritySettings.tsx 默认值 + 当前 DB seed 实测：sumTimes=8, unlockMinu=2
func TestSecurityPolicy_RealSeedValues(t *testing.T) {
	q := &policyMockQuerier{
		entries: map[string]string{
			"security.sumTimes":   "8",
			"security.unlockMinu": "2",
		},
	}
	v := NewSecurityPolicy(q).Get(context.Background())
	assert.Equal(t, int64(8), v.LockThreshold)
	assert.Equal(t, 2*time.Minute, v.LockDuration)
}

// readBool 的几个状态机：true / false / 1 / 0 / 空 / 非法
func TestReadBool_Variants(t *testing.T) {
	cases := []struct {
		raw      string
		want     bool
		hasValue bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"1", true, true},
		{"0", false, true},
		{"TRUE", true, true},
		{"yes", false, false}, // strconv.ParseBool 不接受 yes
		{"", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			q := &policyMockQuerier{entries: map[string]string{"security.flag": tc.raw}}
			b, ok := readBool(context.Background(), q, "security", "flag")
			assert.Equal(t, tc.hasValue, ok)
			if ok {
				assert.Equal(t, tc.want, b)
			}
		})
	}
}

// 防 string 配置漏读（defaultPasswd / msg）
func TestReadString(t *testing.T) {
	q := &policyMockQuerier{entries: map[string]string{
		"security.defaultPasswd": "hello",
		"security.empty":         "",
	}}
	assert.Equal(t, "hello", readString(context.Background(), q, "security", "defaultPasswd"))
	assert.Equal(t, "", readString(context.Background(), q, "security", "empty"))
	assert.Equal(t, "", readString(context.Background(), q, "security", "missing"))
}

// 验证 strconv.ParseInt 64 位足够大 — 极端配置（如锁定 10000 分钟）也能解码
func TestSecurityPolicy_LargeNumbers(t *testing.T) {
	q := &policyMockQuerier{entries: map[string]string{
		"security.unlockMinu": strconv.Itoa(10000),
	}}
	v := NewSecurityPolicy(q).Get(context.Background())
	assert.Equal(t, 10000*time.Minute, v.LockDuration)
}
