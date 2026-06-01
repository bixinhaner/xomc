-- +goose Up
-- system 内部账号改为非内置（source: builtIn → admin），使其在「系统管理 / 用户」
-- 页面可删除/可禁用。
--
-- 背景：system（UUID 00000000-...001，密码 !disabled-no-password-login! 不可登录）是
-- 内部 API Key（omc-internal，omcctl / worker 调内部接口用）的属主。原 source='builtIn'
-- 让它同时具备「超管旁路 + 不可删除/不可禁用」两重语义（见 internal/admin/middleware.go
-- 的 IsSuperAdmin 旁路、service.go DeleteUser/SetUserStatus 的 builtIn 守门）。
--
-- 产品决策：内置账号只保留 admin 一个；system 改为普通来源(admin) → 前端
-- isBuiltInUser(source==='builtIn') 与后端 DeleteUser 守门(source==UserSourceBuiltIn)
-- 均不再将其视为内置，故可删除。
--
-- 影响：
--   · system 由此失去「超管旁路」（IsSuperAdmin 派生自 source='builtIn'）。
--     内部 API Key 仍由 UUID 常量签发（admin.EnsureInternalAPIKey 不看 source），
--     鉴权时继承角色权限（system 绑 admin 角色），不影响 Key 本身的签发与校验。
--   · 删除 system 才会真正断掉内部 API Key（属主丢失 → 401、app 无法再签发）；
--     本迁移仅放开「可删除」，是否删除由运维决定。
--
-- 幂等：仅当当前为 builtIn 时更新；重复执行 / 已是 admin 时为 no-op。
UPDATE users
   SET source = 'admin', updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000001'
   AND username = 'system'
   AND source = 'builtIn';

-- +goose Down
-- 回滚：恢复 system 为内置账号（重新获得超管旁路 + 受删除/禁用保护）。
UPDATE users
   SET source = 'builtIn', updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000001'
   AND username = 'system'
   AND source = 'admin';
