package admin

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword_LengthBounds(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 8, MaxLength: 32}

	cases := []struct {
		name     string
		password string
		want     error
	}{
		{"7 chars too short", "abcdef1", ErrPasswordTooShort},
		{"8 chars boundary", "abcdef12", nil},
		{"32 chars boundary", strings.Repeat("a", 32), nil},
		{"33 chars too long", strings.Repeat("a", 33), ErrPasswordTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password, policy)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

// fail-open：MinLength/MaxLength=0 时不校验长度 — 让"未配置"语义安全。
func TestValidatePassword_ZeroBoundsSkipped(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 0, MaxLength: 0}
	assert.NoError(t, ValidatePassword("a", policy), "MinLength=0 不校验")
	assert.NoError(t, ValidatePassword(strings.Repeat("x", 1000), policy), "MaxLength=0 不校验")
}

func TestValidatePassword_ComplexityRequiresThreeClasses(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 4, MaxLength: 64, RequireMix: true}

	cases := []struct {
		name     string
		password string
		ok       bool
	}{
		{"全小写 1 类", "abcdefghi", false},
		{"小写+数字 2 类", "abcd1234", false},
		{"小写+大写+数字 3 类", "Abcd1234", true},
		{"4 类齐全", "Abc1@xyz", true},
		{"含中文符号也算 3 类", "Abc1中文", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password, policy)
			if tc.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrPasswordMissingClass)
			}
		})
	}
}

// RequireMix=false 时不校验复杂度（即使全单一字符也通过）。
func TestValidatePassword_ComplexityDisabled(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 4, MaxLength: 64, RequireMix: false}
	assert.NoError(t, ValidatePassword("aaaaaaaa", policy))
}

// 长度先于复杂度（先返长度错），让用户先理解长度问题。
func TestValidatePassword_LengthCheckedBeforeComplexity(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 16, MaxLength: 32, RequireMix: true}
	err := ValidatePassword("ab", policy) // 既短又单一字符类
	assert.ErrorIs(t, err, ErrPasswordTooShort,
		"应先报长度错而不是复杂度错")
}

// Snapshot 从 SecurityPolicy 派生时正确映射字段。
func TestPasswordPolicySnapshotFromSecurityPolicy(t *testing.T) {
	v := &securityPolicyValues{
		PwdMinLength:          10,
		PwdMaxLength:          50,
		PasswordPolicyEnabled: true,
	}
	s := PasswordPolicySnapshotFromSecurityPolicy(v)
	assert.Equal(t, int64(10), s.MinLength)
	assert.Equal(t, int64(50), s.MaxLength)
	assert.True(t, s.RequireMix)

	// nil 安全：fallback 到 default policy
	s2 := PasswordPolicySnapshotFromSecurityPolicy(nil)
	assert.Equal(t, int64(defaultPwdMinLength), s2.MinLength)
}

// Unicode 字符按 rune 计数（中文密码"长度 8"不应被 byte 数算成 24）
func TestValidatePassword_UnicodeRuneCount(t *testing.T) {
	policy := PasswordPolicySnapshot{MinLength: 8, MaxLength: 16}
	pw := "你好世界1234" // 8 runes / 16 bytes
	assert.NoError(t, ValidatePassword(pw, policy))
}

// =========== Login 密码过期/首次改密响应字段 annotation ===========

func TestAnnotatePasswordPolicyState_NoOpWhenDisabled(t *testing.T) {
	user := &User{PasswordChangedAt: nil}
	tp := &TokenPair{}
	policy := defaultPolicy() // PasswordExpiresEnabled=false by default
	annotatePasswordPolicyState(nil, user, policy, tp)
	assert.False(t, tp.MustChangePassword)
	assert.Nil(t, tp.PasswordExpiresInDays)
}

func TestAnnotatePasswordPolicyState_MustChangeFlagPreserved(t *testing.T) {
	user := &User{MustChangePassword: true}
	tp := &TokenPair{}
	annotatePasswordPolicyState(nil, user, defaultPolicy(), tp)
	assert.True(t, tp.MustChangePassword, "user.MustChangePassword=true 必须透传")
}

func TestAnnotatePasswordPolicyState_PasswordChangedAtNil_ForcesChange(t *testing.T) {
	user := &User{PasswordChangedAt: nil}
	tp := &TokenPair{}
	policy := &securityPolicyValues{PasswordExpiresEnabled: true, PasswordValidDays: 90}
	annotatePasswordPolicyState(nil, user, policy, tp)
	assert.True(t, tp.MustChangePassword,
		"password_changed_at NULL → 视为初始密码强制改密")
}

func TestAnnotatePasswordPolicyState_PasswordExpired(t *testing.T) {
	past := time.Now().AddDate(0, 0, -100) // 100 天前
	user := &User{PasswordChangedAt: &past}
	tp := &TokenPair{}
	policy := &securityPolicyValues{PasswordExpiresEnabled: true, PasswordValidDays: 90}
	annotatePasswordPolicyState(nil, user, policy, tp)
	assert.True(t, tp.MustChangePassword, "100天 > validPeriod=90 → 已过期")
	assert.Nil(t, tp.PasswordExpiresInDays)
}

func TestAnnotatePasswordPolicyState_NearExpiry_WithPrompt(t *testing.T) {
	// 85 天前改的密码 + 90 天有效期 + 7 天提前提示 → 剩 5 天 ≤ 7 → 附 days
	past := time.Now().AddDate(0, 0, -85)
	user := &User{PasswordChangedAt: &past}
	tp := &TokenPair{}
	policy := &securityPolicyValues{
		PasswordExpiresEnabled: true,
		PasswordValidDays:      90,
		PasswordPromptDays:     7,
	}
	annotatePasswordPolicyState(nil, user, policy, tp)
	assert.False(t, tp.MustChangePassword, "还没过期不强制改")
	if assert.NotNil(t, tp.PasswordExpiresInDays) {
		assert.LessOrEqual(t, *tp.PasswordExpiresInDays, 5)
	}
}

func TestAnnotatePasswordPolicyState_OutsidePromptWindow(t *testing.T) {
	// 30 天前改的密码 + 90 天有效期 + 7 天提前提示 → 剩 60 天 > 7 → 不附 days
	past := time.Now().AddDate(0, 0, -30)
	user := &User{PasswordChangedAt: &past}
	tp := &TokenPair{}
	policy := &securityPolicyValues{
		PasswordExpiresEnabled: true,
		PasswordValidDays:      90,
		PasswordPromptDays:     7,
	}
	annotatePasswordPolicyState(nil, user, policy, tp)
	assert.False(t, tp.MustChangePassword)
	assert.Nil(t, tp.PasswordExpiresInDays, "60 天 > 7 天提示窗口 → 不提示")
}

// 防回归：fmt.Errorf wrap 后 errors.Is 仍可识别原型错误
func TestValidatePassword_ErrorsIsCompatible(t *testing.T) {
	err := ValidatePassword("a", PasswordPolicySnapshot{MinLength: 8})
	assert.True(t, errors.Is(err, ErrPasswordTooShort))

	err2 := ValidatePassword(strings.Repeat("a", 100), PasswordPolicySnapshot{MaxLength: 10})
	assert.True(t, errors.Is(err2, ErrPasswordTooLong))
}
