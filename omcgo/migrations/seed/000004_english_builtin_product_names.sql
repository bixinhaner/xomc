-- +goose Up
WITH mapping(old_name, new_name, new_vendor, new_description) AS (
    VALUES
        ('BAIBLQ 产品', 'BAIBLQ', 'Baicells', 'Baicells BAIBLQ'),
        ('BLX 产品', 'BLX', 'Baicells', 'Baicells BLX'),
        ('QRTB 系列', 'QRTB Series', 'Baicells', 'Baicells QRTB; supports SC/CA/DC radio modes'),
        ('MLQ 产品', 'MLQ', 'Baicells', 'Baicells MLQ'),
        ('MLN 系列', 'MLN Series', 'Baicells', 'Baicells MLN; supports SC/CA/DC radio modes'),
        ('BM 产品', 'BM', 'Baicells', 'Baicells BM'),
        ('BSC 产品', 'BSC', 'Baicells', 'Baicells BSC'),
        ('BTS 产品', 'BTS', 'Baicells', 'Baicells BTS; 2G base station; KPIs are reported by BSC'),
        ('BaiBNQ 5G 产品', 'BaiBNQ 5G', 'Baicells', 'Baicells BaiBNQ; 5G NR base station; TR-069 parameters and KPI formulas use BaiBNQ'),
        ('CICT SC3400(L1821) 产品', 'CICT SC3400(L1821)', 'CICT', 'CICT SC3400(L1821)'),
        ('大唐 fBS3251 系列', 'Datang fBS3251 Series', 'Datang', 'Datang fBS3251; includes 2 model variants'),
        ('第三方 FDD-LTE-Enterprise', 'Third-party FDD-LTE-Enterprise', 'Third-party', 'Third-party FDD-LTE-Enterprise'),
        ('华为 TCELL 系列', 'Huawei TCELL Series', 'Huawei', 'Huawei TCELL; includes 6 model variants'),
        ('京信 LTE-FDD_N 系列', 'Comba LTE-FDD_N Series', 'Comba', 'Comba LTE-FDD_N; includes 4 model variants'),
        ('京信 femto_au 产品', 'Comba femto_au', 'Comba', 'Comba femto_au')
)
UPDATE public.products AS p
SET product_name = mapping.new_name,
    vendor = mapping.new_vendor,
    description = mapping.new_description,
    updated_at = now()
FROM mapping
WHERE p.product_name = mapping.old_name;

WITH mapping(old_name, new_name) AS (
    VALUES
        ('BAIBLQ 产品', 'BAIBLQ'),
        ('BLX 产品', 'BLX'),
        ('QRTB 系列', 'QRTB Series'),
        ('MLQ 产品', 'MLQ'),
        ('MLN 系列', 'MLN Series'),
        ('BM 产品', 'BM'),
        ('BSC 产品', 'BSC'),
        ('BTS 产品', 'BTS'),
        ('BaiBNQ 5G 产品', 'BaiBNQ 5G'),
        ('CICT SC3400(L1821) 产品', 'CICT SC3400(L1821)'),
        ('大唐 fBS3251 系列', 'Datang fBS3251 Series'),
        ('第三方 FDD-LTE-Enterprise', 'Third-party FDD-LTE-Enterprise'),
        ('华为 TCELL 系列', 'Huawei TCELL Series'),
        ('京信 LTE-FDD_N 系列', 'Comba LTE-FDD_N Series'),
        ('京信 femto_au 产品', 'Comba femto_au')
)
UPDATE public.ufte_task_types AS tt
SET product_scope = (
        SELECT COALESCE(jsonb_agg(to_jsonb(COALESCE(mapping.new_name, entry.value)) ORDER BY entry.ord), '[]'::jsonb)
        FROM jsonb_array_elements_text(tt.product_scope) WITH ORDINALITY AS entry(value, ord)
        LEFT JOIN mapping ON mapping.old_name = entry.value
    ),
    updated_at = now()
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(tt.product_scope) AS entry(value)
    JOIN mapping ON mapping.old_name = entry.value
);

-- +goose Down
WITH mapping(new_name, old_name, old_vendor, old_description) AS (
    VALUES
        ('BAIBLQ', 'BAIBLQ 产品', 'Baicells', 'Baicells BAIBLQ'),
        ('BLX', 'BLX 产品', 'Baicells', 'Baicells BLX'),
        ('QRTB Series', 'QRTB 系列', 'Baicells', 'Baicells QRTB；含 SC/CA/DC 多种射频模式'),
        ('MLQ', 'MLQ 产品', 'Baicells', 'Baicells MLQ'),
        ('MLN Series', 'MLN 系列', 'Baicells', 'Baicells MLN；含 SC/CA/DC 多种射频模式'),
        ('BM', 'BM 产品', 'Baicells', 'Baicells BM'),
        ('BSC', 'BSC 产品', 'Baicells', 'Baicells BSC 产品'),
        ('BTS', 'BTS 产品', 'Baicells', 'Baicells BTS；2G 基站；KPI 由 BSC 统一上报'),
        ('BaiBNQ 5G', 'BaiBNQ 5G 产品', 'Baicells', 'Baicells BaiBNQ；5G NR 基站；TR069 参数集和 KPI 公式集统一命名为 BaiBNQ'),
        ('CICT SC3400(L1821)', 'CICT SC3400(L1821) 产品', 'CICT', 'CICT SC3400(L1821)'),
        ('Datang fBS3251 Series', '大唐 fBS3251 系列', '大唐', '大唐 fBS3251；含 2 个型号变体'),
        ('Third-party FDD-LTE-Enterprise', '第三方 FDD-LTE-Enterprise', '第三方', '第三方 FDD-LTE-Enterprise'),
        ('Huawei TCELL Series', '华为 TCELL 系列', '华为', '华为 TCELL；含 6 个型号变体'),
        ('Comba LTE-FDD_N Series', '京信 LTE-FDD_N 系列', '京信', '京信 LTE-FDD_N；含 4 个型号变体'),
        ('Comba femto_au', '京信 femto_au 产品', '京信', '京信 femto_au')
)
UPDATE public.products AS p
SET product_name = mapping.old_name,
    vendor = mapping.old_vendor,
    description = mapping.old_description,
    updated_at = now()
FROM mapping
WHERE p.product_name = mapping.new_name;

WITH mapping(new_name, old_name) AS (
    VALUES
        ('BAIBLQ', 'BAIBLQ 产品'),
        ('BLX', 'BLX 产品'),
        ('QRTB Series', 'QRTB 系列'),
        ('MLQ', 'MLQ 产品'),
        ('MLN Series', 'MLN 系列'),
        ('BM', 'BM 产品'),
        ('BSC', 'BSC 产品'),
        ('BTS', 'BTS 产品'),
        ('BaiBNQ 5G', 'BaiBNQ 5G 产品'),
        ('CICT SC3400(L1821)', 'CICT SC3400(L1821) 产品'),
        ('Datang fBS3251 Series', '大唐 fBS3251 系列'),
        ('Third-party FDD-LTE-Enterprise', '第三方 FDD-LTE-Enterprise'),
        ('Huawei TCELL Series', '华为 TCELL 系列'),
        ('Comba LTE-FDD_N Series', '京信 LTE-FDD_N 系列'),
        ('Comba femto_au', '京信 femto_au 产品')
)
UPDATE public.ufte_task_types AS tt
SET product_scope = (
        SELECT COALESCE(jsonb_agg(to_jsonb(COALESCE(mapping.old_name, entry.value)) ORDER BY entry.ord), '[]'::jsonb)
        FROM jsonb_array_elements_text(tt.product_scope) WITH ORDINALITY AS entry(value, ord)
        LEFT JOIN mapping ON mapping.new_name = entry.value
    ),
    updated_at = now()
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(tt.product_scope) AS entry(value)
    JOIN mapping ON mapping.new_name = entry.value
);
