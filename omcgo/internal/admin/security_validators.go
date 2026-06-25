package admin

import (
	"context"
)

// RegisterSecurityValidators 为 sys_configs 安全设置类配置项注册 BatchUpsert 前的值校验器。
//
// 当前注册项（issue #649）：
//   - (security, defaultPasswd)：非空时必须满足当前生效的密码强度规则（passwordContent /
//     pwdMinLength / pwdMaxLength）；允许空字符串（清空 = 关闭"用默认密码"特性，重置走 fallback）。
//
// 设计依据（设计方案 §3.1 + §2）：
//   - defaultPasswd 是"管理员注入凭据"的源头，写入侧强校验后即作为受信任运维设定值；
//     service 层消费默认密码时跳过 ValidatePassword（避免规则调高后存量默认密码触发 UX 死结）。
//   - 取 policy.Get(ctx) 是 BatchUpsert "前"的当前 policy；若同批保存修改了 pwdMinLength 等
//     强度参数，default password 按"旧规则"校验，与 FE 提示「调整规则前请确保默认密码满足
//     新规则」一致，避免同批次内"先调高规则、再以旧规则校验"的竞态歧义。
//
// 调用契约：仅在进程启动 wiring 阶段调用一次（参考 minio_presign_bridge 模板），svc nil 时 no-op。
func RegisterSecurityValidators(svc *SysConfigService, policy *SecurityPolicy) {
	if svc == nil {
		return
	}
	svc.RegisterValidator("security", "defaultPasswd", func(value string) error {
		if value == "" {
			return nil
		}
		snap := PasswordPolicySnapshotFromSecurityPolicy(policy.Get(context.Background()))
		return ValidatePassword(value, snap)
	})
}
