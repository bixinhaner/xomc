-- +goose Up
-- #492：UFTE 模板「适用产品」对齐产品目录。
-- 新增 product_scope（产品英文名列表，引用 products.product_name）。语义：
--   · 非空 → 设备候选匹配按"设备 productClass → ProductRegistry → product.Name ∈ 列表"精确放行；
--   · 空   → 回退旧 platform_scope 子串 + tech 关键字匹配（灰度兼容，见 internal/ufte deviceMatchesTaskType）。
-- 制式(2G/4G/5G)由所选产品的 tech 派生，不再依赖硬编码 techHint。
-- 内置升级模板的初始 product_scope 由 seed/000012 按制式回填。
ALTER TABLE public.ufte_task_types
    ADD COLUMN IF NOT EXISTS product_scope jsonb NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN public.ufte_task_types.product_scope IS
    '#492 适用产品：产品英文名列表（引用 products.product_name）。非空时设备匹配按产品名精确匹配，空则回退 platform_scope。';

-- +goose Down
ALTER TABLE public.ufte_task_types DROP COLUMN IF EXISTS product_scope;
