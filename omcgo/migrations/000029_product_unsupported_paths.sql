-- +goose Up
-- product_unsupported_paths —— 按 product_id 记录「设备实际不支持的参数 PATH」的自学习表，
-- 读 / 写分别标记（有些 path 不支持写但支持读，如只读参数）。
--
-- 触发：MML 控制台（/mml/console-v2）执行失败、CPE 返回「path 不支持」类故障时，按
-- (product_id, standardPath) upsert：
--   - 9005 Invalid Parameter Name（path 在数据模型中不存在）→ read_unsupported=true AND write_unsupported=true
--   - 9008 Non-writable Parameter（参数只读）            → write_unsupported=true（读仍可用）
-- 消费：前端「选择命令 / 配置参数」按所选产品 + 命令读/写类型过滤 path 展示。
--
-- 设计：按 **product_id** 维度（前端产品下拉直接给出，无需由 SN 反算）。不按 param_model
-- （多个产品可能共用同一 param_model，按模型标记会互相污染）；也不按 product_class
-- （同一 product 可对应多个 product_class，product_id 才是规范主键）。
CREATE TABLE IF NOT EXISTS product_unsupported_paths (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID         NOT NULL,
    standard_path     VARCHAR(512) NOT NULL,
    read_unsupported  BOOLEAN      NOT NULL DEFAULT false,
    write_unsupported BOOLEAN      NOT NULL DEFAULT false,
    last_fault_code   INTEGER,
    last_device_sn    VARCHAR(64),
    hit_count         INTEGER      NOT NULL DEFAULT 1,
    first_seen_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_product_unsupported_path UNIQUE (product_id, standard_path)
);

CREATE INDEX IF NOT EXISTS idx_product_unsupported_paths_product ON product_unsupported_paths (product_id);

-- +goose Down
DROP TABLE IF EXISTS product_unsupported_paths;
