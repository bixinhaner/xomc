-- 删除 menus.i18n_key 列
--
-- 背景：000083 引入 i18n_key 作为「让某菜单走前端 i18n 包翻译」的兼容字段。
-- 但实际方案确定走 DB JSONB 主路径（name_i18n + 菜单管理 UI 编辑），i18n_key 在 UI
-- 上对运营产生困惑（"我该填吗？"），又无任何菜单需要走代码翻译。按"偶然零件即删"
-- 原则下线，避免长期残留误导。
--
-- 风险：若已写入 i18n_key 数据，DROP 会丢；当前生产/dev 均未使用此字段，零数据损失。
-- 未来真需要"代码翻译路径"再单独引入，前端 resolveMenuLabel 的 fallback 链路自洽。

-- +goose Up
ALTER TABLE menus DROP COLUMN IF EXISTS i18n_key;

-- +goose Down
-- 回滚仅恢复列结构，不恢复数据（数据已永久丢失）
ALTER TABLE menus ADD COLUMN IF NOT EXISTS i18n_key VARCHAR(128);
COMMENT ON COLUMN menus.i18n_key IS '已废弃字段（migration 000085 删除）。仅在 down 回滚时占位。';
