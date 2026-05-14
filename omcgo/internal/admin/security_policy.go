package admin

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// SecurityPolicy 缓存 system/config 安全设置页全部字段的解码结果。
//
// 设计目的：让 LoginGuard / IP 限流 / 单点登录 / 密码策略 / 长期未登录 cron 等
// 多个特性共享同一份 30s in-memory 缓存，避免每个特性各自重复实现 sys_configs
// 读取逻辑。
//
// 数据来源：sys_configs (category='security')，由 FE pages/system/SystemConfig/
// SecuritySettings.tsx 表单 batch upsert 写入。
//
// 缓存策略：30s TTL；atomic.Pointer 无锁读 + double-checked mutex 写。
// 调 InvalidateCache() 可主动失效（如 sys_configs Update Hook 调用）。
//
// fallback：读不到 / 解析失败 / 值非法 → 退化到包级 default 常量。
type SecurityPolicy struct {
	cfg     SysConfigQuerier
	cacheMu sync.Mutex
	cache   atomic.Pointer[securityPolicyValues]
}

// securityPolicyValues 持有所有解码后的运行时值。每个字段对应 SecuritySettings.tsx
// 一个表单字段（FE key 见注释）。
type securityPolicyValues struct {
	// ⑤ 登录锁定
	CaptchaThreshold int64         // FE: attemptTimes（失败 N 次弹验证码）
	LockThreshold    int64         // FE: sumTimes（失败 N 次锁账号）
	LockDuration     time.Duration // FE: unlockMinu（锁定分钟）
	VerifyEnable     bool          // FE: verifyEnable（图形验证码总开关）

	// ⑥ IP 限流
	IPLimitWindow time.Duration // FE: limitMinus（滑动窗口分钟数）
	IPLimitCount  int64         // FE: limitCount（窗口内最大失败次数）
	IPLockMinutes time.Duration // FE: limitTimes（IP 锁定时长分钟）

	// ⑩ 最大会话限制
	// 注意 FE 字段名是反向语义："isOnlyOneUserLoginEnable=true" 表示"允许多端并发"，
	// "false" 表示"单点登录"。这里 BE 取正向语义 AllowConcurrent 减少阅读心智负担。
	AllowConcurrent bool // ← 直接复制自 FE isOnlyOneUserLoginEnable

	// ② 密码强度
	PasswordPolicyEnabled bool  // FE: passwordContent
	PwdMinLength          int64 // FE: pwdMinLength
	PwdMaxLength          int64 // FE: pwdMaxLength

	// ④ 密码有效期
	PasswordExpiresEnabled bool  // FE: expires
	PasswordValidDays      int64 // FE: validPeriod
	PasswordPromptDays     int64 // FE: promptBeforeDays

	// ① 默认密码
	MustChangePasswordOnFirstLogin bool   // FE: modifyPWD
	DefaultPassword                string // FE: defaultPasswd

	// ⑦ 屏幕锁定
	IdleLockMinutes int64 // FE: userSessionExpirationMin（0=禁用）

	// ⑧ 浏览器记密码
	PreventBrowserAutofill bool // FE: isBrowserAutoRecordPass

	// ⑨ 账户长期未登录自动锁
	AutoLockUnusedEnabled bool  // FE: autoLockUserDayEnable
	AutoLockUnusedDays    int64 // FE: autoLockUserDay

	// ⑪ 登录提示
	LoginNotifyEnabled bool   // FE: enabledFlag
	LoginNotifyMsg     string // FE: msg

	expiresAt time.Time
}

// security 类 default 值（fail-safe — 任何配置加载失败都不会让安全机制失活）。
const (
	defaultCaptchaThreshold = 3
	defaultLockThreshold    = 10
	defaultLockDuration     = 30 * time.Minute
	defaultVerifyEnable     = false

	defaultIPLimitWindow = 1 * time.Minute
	defaultIPLimitCount  = 5
	defaultIPLockMinutes = 30 * time.Minute

	defaultAllowConcurrent = true // 默认允许多端，避免上线后用户误踢

	defaultPwdMinLength = 8
	defaultPwdMaxLength = 32

	defaultPasswordValidDays  = 90
	defaultPasswordPromptDays = 7

	defaultDefaultPassword = "OMC@123456"
	defaultIdleLockMinutes = 0 // 0 = 禁用屏幕锁

	defaultAutoLockUnusedDays = 90

	securityPolicyCacheTTL = 30 * time.Second
)

// NewSecurityPolicy creates a new policy with an injected SysConfigQuerier.
// nil 安全：未注入查询器时所有字段返回 default。
func NewSecurityPolicy(q SysConfigQuerier) *SecurityPolicy {
	return &SecurityPolicy{cfg: q}
}

// Get returns the current policy snapshot. 30s 缓存命中时无锁。
func (p *SecurityPolicy) Get(ctx context.Context) *securityPolicyValues {
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return cached
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	if cached := p.cache.Load(); cached != nil && time.Now().Before(cached.expiresAt) {
		return cached
	}

	v := defaultPolicy()
	v.expiresAt = time.Now().Add(securityPolicyCacheTTL)
	if p.cfg != nil {
		v.loadFromSysConfig(ctx, p.cfg)
	}
	p.cache.Store(v)
	return v
}

// InvalidateCache 让下一次 Get 重新加载（FE 保存配置后可调）。
func (p *SecurityPolicy) InvalidateCache() {
	p.cache.Store(nil)
}

// defaultPolicy 返回所有字段 default 的 snapshot。
func defaultPolicy() *securityPolicyValues {
	return &securityPolicyValues{
		CaptchaThreshold: defaultCaptchaThreshold,
		LockThreshold:    defaultLockThreshold,
		LockDuration:     defaultLockDuration,
		VerifyEnable:     defaultVerifyEnable,

		IPLimitWindow: defaultIPLimitWindow,
		IPLimitCount:  defaultIPLimitCount,
		IPLockMinutes: defaultIPLockMinutes,

		AllowConcurrent: defaultAllowConcurrent,

		PwdMinLength: defaultPwdMinLength,
		PwdMaxLength: defaultPwdMaxLength,

		PasswordValidDays:  defaultPasswordValidDays,
		PasswordPromptDays: defaultPasswordPromptDays,

		DefaultPassword: defaultDefaultPassword,
		IdleLockMinutes: defaultIdleLockMinutes,

		AutoLockUnusedDays: defaultAutoLockUnusedDays,
	}
}

// loadFromSysConfig 把 sys_configs 中可读到的字段覆盖到 default 之上。
// 单个字段读失败不影响其它字段加载（部分配置仍可生效）。
func (v *securityPolicyValues) loadFromSysConfig(ctx context.Context, q SysConfigQuerier) {
	// ⑤ 登录锁定
	if n := readInt(ctx, q, "security", "attemptTimes"); n > 0 {
		v.CaptchaThreshold = n
	}
	if n := readInt(ctx, q, "security", "sumTimes"); n > 0 {
		v.LockThreshold = n
	}
	if n := readInt(ctx, q, "security", "unlockMinu"); n > 0 {
		v.LockDuration = time.Duration(n) * time.Minute
	}
	if b, ok := readBool(ctx, q, "security", "verifyEnable"); ok {
		v.VerifyEnable = b
	}

	// ⑥ IP 限流
	if n := readInt(ctx, q, "security", "limitMinus"); n > 0 {
		v.IPLimitWindow = time.Duration(n) * time.Minute
	}
	if n := readInt(ctx, q, "security", "limitCount"); n > 0 {
		v.IPLimitCount = n
	}
	if n := readInt(ctx, q, "security", "limitTimes"); n > 0 {
		v.IPLockMinutes = time.Duration(n) * time.Minute
	}

	// ⑩ 最大会话限制
	// FE isOnlyOneUserLoginEnable=true 表示"允许多端并发"（FE 文案"允许多端同时登录"勾选）
	if b, ok := readBool(ctx, q, "security", "isOnlyOneUserLoginEnable"); ok {
		v.AllowConcurrent = b
	}

	// ② 密码强度
	if b, ok := readBool(ctx, q, "security", "passwordContent"); ok {
		v.PasswordPolicyEnabled = b
	}
	if n := readInt(ctx, q, "security", "pwdMinLength"); n > 0 {
		v.PwdMinLength = n
	}
	if n := readInt(ctx, q, "security", "pwdMaxLength"); n > 0 {
		v.PwdMaxLength = n
	}

	// ④ 密码有效期
	if b, ok := readBool(ctx, q, "security", "expires"); ok {
		v.PasswordExpiresEnabled = b
	}
	if n := readInt(ctx, q, "security", "validPeriod"); n > 0 {
		v.PasswordValidDays = n
	}
	if n := readInt(ctx, q, "security", "promptBeforeDays"); n > 0 {
		v.PasswordPromptDays = n
	}

	// ① 默认密码
	if b, ok := readBool(ctx, q, "security", "modifyPWD"); ok {
		v.MustChangePasswordOnFirstLogin = b
	}
	if s := readString(ctx, q, "security", "defaultPasswd"); s != "" {
		v.DefaultPassword = s
	}

	// ⑦ 屏幕锁定
	if n := readInt(ctx, q, "security", "userSessionExpirationMin"); n >= 0 {
		// 注意：0 表示禁用而非"未配置"；用 >= 0 + 显式存在判定
		if cfg, err := q.GetByKey(ctx, "security", "userSessionExpirationMin"); err == nil && cfg != nil && cfg.Value != "" {
			v.IdleLockMinutes = n
		}
	}

	// ⑧ 浏览器记密码
	if b, ok := readBool(ctx, q, "security", "isBrowserAutoRecordPass"); ok {
		v.PreventBrowserAutofill = b
	}

	// ⑨ 账户长期未登录自动锁
	if b, ok := readBool(ctx, q, "security", "autoLockUserDayEnable"); ok {
		v.AutoLockUnusedEnabled = b
	}
	if n := readInt(ctx, q, "security", "autoLockUserDay"); n > 0 {
		v.AutoLockUnusedDays = n
	}

	// ⑪ 登录提示
	if b, ok := readBool(ctx, q, "security", "enabledFlag"); ok {
		v.LoginNotifyEnabled = b
	}
	if s := readString(ctx, q, "security", "msg"); s != "" {
		v.LoginNotifyMsg = s
	}
}

// readInt / readBool / readString 是 readIntConfig（bruteforce.go）的扩展同类辅助。
// 任何错误（key 不存在 / 解析失败）都返 0 / false / 空串让 caller 走 default。

func readInt(ctx context.Context, q SysConfigQuerier, category, key string) int64 {
	return readIntConfig(ctx, q, category, key) // 复用 bruteforce.go 已有的实现
}

func readBool(ctx context.Context, q SysConfigQuerier, category, key string) (bool, bool) {
	cfg, err := q.GetByKey(ctx, category, key)
	if err != nil || cfg == nil || cfg.Value == "" {
		return false, false
	}
	b, err := strconv.ParseBool(cfg.Value)
	if err != nil {
		return false, false
	}
	return b, true
}

func readString(ctx context.Context, q SysConfigQuerier, category, key string) string {
	cfg, err := q.GetByKey(ctx, category, key)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Value
}
