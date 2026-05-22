-- ============================================================
-- 000155_standard_params_description.sql
-- standard_params 加 description 列（path 含义说明，用于 MML 控制台 tooltip / 行内提示）
--
-- 用户决策（2026-05-22 Bundle B）：
--   · MML 控制台操作面板的 path 行需要显示中文含义说明，目前 standard_params 表
--     只有 schema 元数据（standard_path / data_type / access 等），缺业务可读名称
--   · 数据源：omcgo/internal/config/parammodel/mmlstandardloader/seeds/cmcc_tdlte_v23.json
--     中 groups[].commands[].params[].name 字段（如 "用户友好名" / "DN前缀"）
--   · 后续可由 admin UI / parammodel Loader 维护其它运营商 / 制式的 path 描述
--
-- 改动：
--   · 加 description TEXT NOT NULL DEFAULT '' 列
--   · 加索引（可选）— path 描述用于 tooltip 展示，不做搜索基准，不加索引
--   · 数据回填由配套 seed/000156_standard_params_descriptions_zh.sql 完成
--     （624 条来自 cmcc_tdlte_v23.json 的 path → 中文名 UPDATE）
-- ============================================================

-- +goose Up

ALTER TABLE standard_params
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN standard_params.description IS 'TR-181 path 的中文含义说明（来自规范 JSON seed 的 params[].name 字段；前端 MML 控制台 path 行 tooltip / 行内提示用）';


-- +goose Down

ALTER TABLE standard_params DROP COLUMN IF EXISTS description;
