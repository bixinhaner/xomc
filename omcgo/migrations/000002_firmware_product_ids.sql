-- +goose Up
-- +goose StatementBegin

-- #638：固件升级文件上传支持多产品复选。
-- 同一份固件镜像往往适配多个产品型号，原 firmware_versions.product_id 单 UUID 只能挂一个产品，
-- 此处新增 product_ids uuid[] 承载多产品。语义：
--   · product_ids = 该固件适用的全部产品 ID 列表（products.id）
--   · product_id  = 兼容主产品 = product_ids 列表首项（写入时由 handler/repo 同步维护，
--                   保留旧唯一索引 idx_firmware_unique_version(product_id, version, file_type) 不变）
--   · 列表/任务按产品过滤命中条件改为 :pid = ANY(product_ids)（已 backfill，覆盖历史行）
ALTER TABLE public.firmware_versions
    ADD COLUMN IF NOT EXISTS product_ids uuid[] NOT NULL DEFAULT '{}'::uuid[];

UPDATE public.firmware_versions
   SET product_ids = ARRAY[product_id]
 WHERE product_id IS NOT NULL
   AND (product_ids IS NULL OR product_ids = '{}'::uuid[]);

CREATE INDEX IF NOT EXISTS idx_firmware_versions_product_ids_gin
    ON public.firmware_versions USING gin (product_ids);

COMMENT ON COLUMN public.firmware_versions.product_ids IS
    '#638 固件适用的多个产品 ID 列表（products.id）。上传支持多选；product_id 保留为兼容主产品=列表首项。列表/任务按产品过滤命中 ANY(product_ids)。';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_firmware_versions_product_ids_gin;
ALTER TABLE public.firmware_versions DROP COLUMN IF EXISTS product_ids;

-- +goose StatementEnd
