-- +goose Up
-- #492：固件库产品名中心化。固件改为关联产品（product_id → products.id），
-- 上传时选产品（前端按 product_id 提交、按产品名展示），不再以 product_class 为准。
-- 唯一约束从 (product_class, version, file_type) 改为 (product_id, version, file_type)：
-- 同一产品下 版本+文件类型 唯一；不同产品可同版本。product_class 列保留（设备侧
-- 下载/兼容仍可用），但产品归属以 product_id 为权威。历史行 product_id 为 NULL（不兼容历史，
-- PG 唯一索引中多个 NULL 互不冲突，建索引不会因旧行报错）。
ALTER TABLE public.firmware_versions ADD COLUMN IF NOT EXISTS product_id uuid;
CREATE INDEX IF NOT EXISTS idx_firmware_versions_product_id ON public.firmware_versions USING btree (product_id);

DROP INDEX IF EXISTS idx_firmware_unique_version;
CREATE UNIQUE INDEX idx_firmware_unique_version ON public.firmware_versions USING btree (product_id, version, file_type);

COMMENT ON COLUMN public.firmware_versions.product_id IS '#492 固件所属产品（products.id）。上传选产品名 → 存此列；升级/库列表按产品名展示与过滤。';

-- +goose Down
DROP INDEX IF EXISTS idx_firmware_unique_version;
CREATE UNIQUE INDEX idx_firmware_unique_version ON public.firmware_versions USING btree (COALESCE(product_class, (''::character varying)::text), version, file_type);
DROP INDEX IF EXISTS idx_firmware_versions_product_id;
ALTER TABLE public.firmware_versions DROP COLUMN IF EXISTS product_id;
