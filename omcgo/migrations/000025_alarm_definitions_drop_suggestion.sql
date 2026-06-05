-- +goose Up
-- 删除告警定义的「处置建议」字段（cn_suggestion / en_suggestion）。
-- 背景：该字段仅存在于告警库目录与编辑表单，无任何下游消费（告警运行时、北向、
-- 当前告警详情页均不读取），属无明确用途的死字段，按 2026-06-05 决策全栈删除。
ALTER TABLE alarm_definitions DROP COLUMN IF EXISTS cn_suggestion;
ALTER TABLE alarm_definitions DROP COLUMN IF EXISTS en_suggestion;

-- +goose Down
-- 回滚：恢复两列（仅结构；原 XML 数据已从源文件移除，无法自动回填）。
ALTER TABLE alarm_definitions ADD COLUMN IF NOT EXISTS cn_suggestion text;
ALTER TABLE alarm_definitions ADD COLUMN IF NOT EXISTS en_suggestion text;
