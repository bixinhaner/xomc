-- +goose Up
-- ============================================================
-- 000171_mml_tasks_path_translation_audit.sql
-- T-0168 — mml_tasks / device_tasks 加路径翻译审计列（与 000114 互补）
--
-- 设计：T-0168 PRD §3 GWT-1/GWT-2 — 任务级翻译来源 + 产品解析状态列存
-- 化，避免审计查询 join + JSONB 解析；与 000114 has_path_translation_miss
-- （per-device 路径未命中标记）正交。
--
-- 5 列：
--   mml_tasks.product_resolved        BOOLEAN — 设备 productClass 是否命中 product；
--                                     false=orphan（激进路线下走原路径下发）
--   mml_tasks.matched_product_id      UUID    — 命中的 product.id；NULL=orphan
--   mml_tasks.matched_product_class   VARCHAR — productClass 冗余列（免 join devices）
--   mml_tasks.path_translation_source VARCHAR — 任务级翻译来源汇总
--   device_tasks.path_translation_source VARCHAR — per-device 翻译来源
-- ============================================================

ALTER TABLE mml_tasks
    ADD COLUMN IF NOT EXISTS product_resolved        BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS matched_product_id      UUID NULL,
    ADD COLUMN IF NOT EXISTS matched_product_class   VARCHAR(64) NULL,
    ADD COLUMN IF NOT EXISTS path_translation_source VARCHAR(32) NULL;

COMMENT ON COLUMN mml_tasks.product_resolved IS
    'T-0168: 设备 product_class 是否通过 ProductRegistry.MatchProductClass 命中 product。'
    'false=orphan（激进路线下 path 走 orphan_passthrough 原路径下发，触发 Prometheus 告警 '
    'mml_path_translation_orphan_total）。历史数据默认 true（不回填，假设旧任务非 orphan）。';

COMMENT ON COLUMN mml_tasks.matched_product_id IS
    'T-0168: 命中的 product.id（UUID）。NULL=product_resolved=false 或非 MML 翻译路径。'
    '便于审计查询 join products 表拿厂商/参数模型信息。';

COMMENT ON COLUMN mml_tasks.matched_product_class IS
    'T-0168: 翻译时使用的设备 product_class 字符串（取 device_sns[0].product_class）。'
    '冗余存储避免 join devices 表；R-8.4 保证 task 内一致。';

COMMENT ON COLUMN mml_tasks.path_translation_source IS
    'T-0168: 任务级翻译来源汇总。枚举值：'
    'discovered（全部走 discovered_param_mappings）/ '
    'default（全部走 param_mappings 默认表）/ '
    'passthrough（mapping 缺失，原路径下发）/ '
    'orphan_passthrough（product 未识别，激进路线下发）/ '
    'mixed（任务内多种来源混合）。NULL=非 MML 翻译路径或 PathTranslator 未注入。';

ALTER TABLE device_tasks
    ADD COLUMN IF NOT EXISTS path_translation_source VARCHAR(32) NULL;

COMMENT ON COLUMN device_tasks.path_translation_source IS
    'T-0168: per-device 翻译来源（同 mml_tasks.path_translation_source 枚举）。'
    '首版由 fanout 从 task 直接复制（R-8.4 保证 task 内 product_class 一致）；'
    'D27 弹性保留：未来若引入 per-device swVersion 差异化翻译，本列由 per-device translate 重写。';

-- +goose Down
ALTER TABLE device_tasks DROP COLUMN IF EXISTS path_translation_source;
ALTER TABLE mml_tasks
    DROP COLUMN IF EXISTS path_translation_source,
    DROP COLUMN IF EXISTS matched_product_class,
    DROP COLUMN IF EXISTS matched_product_id,
    DROP COLUMN IF EXISTS product_resolved;
