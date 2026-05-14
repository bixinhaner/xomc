package admin

import (
	"errors"
	"fmt"
	"unicode"
)

// 密码强度策略错误 — 业务层用 errors.Is 区分原因，handler 把每种错回到相应
// 用户提示。所有错都映射到 HTTP 400 + biz_code 7020+。
var (
	ErrPasswordTooShort     = errors.New("password too short")
	ErrPasswordTooLong      = errors.New("password too long")
	ErrPasswordMissingClass = errors.New("password must contain mixed character classes")
)

// PasswordPolicySnapshot 持有密码校验所需的全部字段（取自 SecurityPolicy）。
// 单独提一个类型让 ValidatePassword 不依赖 securityPolicyValues 整体形态，便于
// 测试 + 未来策略字段扩展不破坏校验函数签名。
type PasswordPolicySnapshot struct {
	MinLength   int64 // 0 / 负数 → 不校验长度下限（fail-open）
	MaxLength   int64 // 0 / 负数 → 不校验长度上限
	RequireMix  bool  // 启用"含大写 / 小写 / 数字 / 符号 ≥3 类"复杂度要求
}

// PasswordPolicySnapshotFromSecurityPolicy 提取密码相关字段方便 caller 复用。
func PasswordPolicySnapshotFromSecurityPolicy(v *securityPolicyValues) PasswordPolicySnapshot {
	if v == nil {
		v = defaultPolicy()
	}
	return PasswordPolicySnapshot{
		MinLength:  v.PwdMinLength,
		MaxLength:  v.PwdMaxLength,
		RequireMix: v.PasswordPolicyEnabled,
	}
}

// ValidatePassword 按 policy 校验明文密码。
//
// 校验顺序固定（长度 → 复杂度），同一密码触发多条规则时返第一条匹配的错。
// 返 fmt.Errorf wrap 后的错（保 errors.Is(err, ErrPassword*) 可区分）。
//
// fail-open：policy 字段 ≤0 时跳过该项 — 让"未配置"语义安全，避免新部署因
// 默认值过严锁住所有改密接口。
func ValidatePassword(password string, policy PasswordPolicySnapshot) error {
	pwLen := int64(len([]rune(password)))

	if policy.MinLength > 0 && pwLen < policy.MinLength {
		return fmt.Errorf("%w: min=%d, got=%d",
			ErrPasswordTooShort, policy.MinLength, pwLen)
	}
	if policy.MaxLength > 0 && pwLen > policy.MaxLength {
		return fmt.Errorf("%w: max=%d, got=%d",
			ErrPasswordTooLong, policy.MaxLength, pwLen)
	}

	if policy.RequireMix {
		classes := countCharClasses(password)
		if classes < 3 {
			return fmt.Errorf("%w: need ≥3 of (uppercase/lowercase/digit/symbol), got=%d",
				ErrPasswordMissingClass, classes)
		}
	}
	return nil
}

// countCharClasses 计有几类字符（大写 / 小写 / 数字 / 符号）出现至少一次。
func countCharClasses(s string) int {
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			// 任何非字母数字都算"符号"（空格 / 中文 / 标点）
			hasSymbol = true
		}
	}
	n := 0
	if hasUpper {
		n++
	}
	if hasLower {
		n++
	}
	if hasDigit {
		n++
	}
	if hasSymbol {
		n++
	}
	return n
}
