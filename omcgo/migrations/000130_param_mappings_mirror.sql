-- +goose Up
-- T-0159: 给 param_mappings + discovered_param_mappings 加 mirror_with 列，
-- 让 dictloader 能从 paramModel XML 的 mirrorWith 属性解析"交叉镜像约束"，
-- 透出到前端 quicksettings：改 A 字段自动同步 B 字段（典型场景 TDD 上下行带宽必须相等）。
--
-- 字段语义：
--   mirror_with: 完整 standardPath（与 standard_path 同格式，含 {i} 占位符），
--                指向被镜像字段；为 NULL 表示无镜像约束。
--                语义对称：两端均填对方路径（声明显式 + 解析无歧义）。
ALTER TABLE param_mappings            ADD COLUMN IF NOT EXISTS mirror_with VARCHAR(256);
ALTER TABLE discovered_param_mappings ADD COLUMN IF NOT EXISTS mirror_with VARCHAR(256);

-- +goose Down
ALTER TABLE param_mappings            DROP COLUMN IF EXISTS mirror_with;
ALTER TABLE discovered_param_mappings DROP COLUMN IF EXISTS mirror_with;
