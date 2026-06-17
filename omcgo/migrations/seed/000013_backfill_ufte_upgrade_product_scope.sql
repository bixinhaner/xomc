-- +goose Up
-- #492：内置升级模板的 product_scope 用当前对应制式(products.tech)的产品英文名列表回填。
-- 映射：ENB_*(4G)→tech='lte'、GNB_*(5G)→tech='nr'、GSM_*(2G)→tech='gsm'。
-- 仅当 product_scope 仍为空（未被运维在「模板配置」里编辑过）时回填，幂等 + 不覆盖人工编辑。
-- jsonb_agg 在该制式无产品时返回 NULL → COALESCE 兜底为 '[]'。
-- migrate-seed 是「先全部 schema 再全部 seed」，本 seed 跑在 000001(products 已 seed) 之后，故能查到产品。
UPDATE public.ufte_task_types t
SET product_scope = COALESCE(
        (SELECT jsonb_agg(p.product_name ORDER BY p.product_name)
         FROM public.products p
         WHERE p.tech = m.tech),
        '[]'::jsonb)
FROM (VALUES
    ('ENB_IMG_UPGRADE',   'lte'),
    ('ENB_PATCH_UPGRADE', 'lte'),
    ('ENB_FPGA_UPGRADE',  'lte'),
    ('GNB_IMG_UPGRADE',   'nr'),
    ('GSM_IMG_UPGRADE',   'gsm')
) AS m(type_code, tech)
WHERE t.type_code = m.type_code
  AND t.product_scope = '[]'::jsonb;

-- +goose Down
UPDATE public.ufte_task_types
SET product_scope = '[]'::jsonb
WHERE type_code IN ('ENB_IMG_UPGRADE', 'ENB_PATCH_UPGRADE', 'ENB_FPGA_UPGRADE', 'GNB_IMG_UPGRADE', 'GSM_IMG_UPGRADE');
