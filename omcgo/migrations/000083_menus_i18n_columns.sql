-- 菜单多语言支持：name_i18n（多语言译文 JSONB） + i18n_key（react-intl 兼容字段）
-- 设计依据：方案 C（DB JSONB 存译文 + i18n_key 兼容前端 i18n 包）。
-- 前端渲染优先级：i18n_key 命中前端 messages > name_i18n[locale] > name_i18n['zh-CN'] > name fallback

-- +goose Up
ALTER TABLE menus
    ADD COLUMN IF NOT EXISTS name_i18n JSONB,
    ADD COLUMN IF NOT EXISTS i18n_key  VARCHAR(128);

COMMENT ON COLUMN menus.name_i18n IS '多语言译文 JSONB，键为 locale code（如 zh-CN/en-US），值为对应译文；NULL 表示未配置多语言，前端回退到 name 字段';
COMMENT ON COLUMN menus.i18n_key  IS 'react-intl 翻译键（如 nav.ops.command），命中前端 messages 时优先于 name_i18n；为空时走 name_i18n / name 链路';

-- name_i18n 不建索引：菜单数量级 100~1000，查询路径恒为按 id / parent_id / role 过滤，
-- 译文字段不参与 WHERE。GIN(name_i18n) 写放大无收益，故略。

-- +goose Down
ALTER TABLE menus
    DROP COLUMN IF EXISTS i18n_key,
    DROP COLUMN IF EXISTS name_i18n;
