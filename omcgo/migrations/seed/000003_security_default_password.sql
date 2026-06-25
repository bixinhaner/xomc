-- +goose Up
-- issue #649：「默认密码」配置项接通 — 预置 sys_configs.security.defaultPasswd = 'OMC@123456'。
--
-- 背景：service 层 P1 / handler P2 已接通 use_default_password 路径，消费 sys_configs
-- 取默认密码。在 baseline 000001_init_seed.sql 中 sys_configs 完全没有 category='security'
-- 任何行（设计 §0 现状证据已确认），首次部署管理员未配置时只能靠 admin.defaultPolicy()
-- 的 fail-safe 兜底常量 "OMC@123456"。本 seed 把该默认值显式写入 DB，让：
--   1. 管理员在「系统配置 → 安全设置」页面打开即可看到当前默认密码（而不是空表单字段）；
--   2. service 层消费走"DB 读到非空值"路径（与生产语义一致），不依赖代码兜底常量；
--   3. 升级既有库幂等：ON CONFLICT 已存在则不动（管理员可能已通过 UI 改成自定义值）。
--
-- 不预置 modifyPWD：该配置项已被 issue #649 硬规则下线（管理员创建/重置 = 一律强制改密），
-- service 层不再消费。老库残留行（若有）也忽略。

INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public)
VALUES ('security', 'defaultPasswd', 'OMC@123456', 'string', '系统默认密码（管理员创建用户/重置密码时可选用；使用者首次登录强制改密）', false)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
-- 仅删与 seed 默认值完全一致的行；管理员若已通过 UI 改成自定义值，回滚不动它（防数据丢失）。
DELETE FROM public.sys_configs
WHERE category = 'security'
  AND key = 'defaultPasswd'
  AND value = 'OMC@123456';
