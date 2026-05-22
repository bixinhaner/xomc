-- 000158_fix_admin_user_source.sql
--
-- 修复 fresh deploy 中 admin 用户 source 错为 'admin' 的问题（应该是 'builtIn'）。
--
-- 根因：
--   1. migration 000053 (users_source.sql) 加 source 列 DEFAULT 'admin'，然后
--      UPDATE users SET source='builtIn' WHERE username='admin'。
--   2. 但 fresh deploy 的 seed (omcgo-seed apply / migrations/seed/000001) 在
--      000053 之后才跑，UPDATE 时 users 表是空的 → 无效。
--   3. seed 插入 admin 时没设 source → 用 DB DEFAULT 'admin' → IsSuperAdmin()
--      返回 false → 所有需 RBAC 的端点对 admin 都返 403。
--
-- 修法：
--   · seed 文件已修（users.json + migrations/seed/000001 都加 source='builtIn'），
--     新装机器 admin 一开始就是 builtIn。
--   · 本 migration 兜底已上线的 broken DB —— 找出仍是 admin / NULL 的内置 admin
--     用户改回 builtIn。判定条件：username='admin' AND id 等于已知 seed UUID
--     （避免误改与 admin 同名的其他用户）。

-- +goose Up

UPDATE users
   SET source = 'builtIn'
 WHERE username = 'admin'
   AND id = '20000000-0000-0000-0000-000000000001'
   AND source <> 'builtIn';

-- +goose Down

-- 回滚无业务意义（builtIn 状态丢失会让 admin 失去超管权限）。
-- 留空：Down 时不改回 'admin'，避免破坏权限。
SELECT 1;
