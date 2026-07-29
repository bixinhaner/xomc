-- +goose Up
WITH mapping(old_name, new_name, new_vendor, new_description) AS (
    VALUES
        ('BAIBLQ', 'BLQ', 'Baicells', 'Baicells BLQ'),
        ('QRTB Series', 'QRTB', 'Baicells', 'Baicells QRTB; supports SC/CA/DC radio modes'),
        ('MLN Series', 'MLN', 'Baicells', 'Baicells MLN; supports SC/CA/DC radio modes'),
        ('BaiBNQ 5G', 'BNQ', 'Baicells', 'Baicells BNQ; 5G NR base station; TR-069 parameters and KPI formulas use BaiBNQ')
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
        ('BAIBLQ', 'BLQ'),
        ('QRTB Series', 'QRTB'),
        ('MLN Series', 'MLN'),
        ('BaiBNQ 5G', 'BNQ')
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
        ('BLQ', 'BAIBLQ', 'Baicells', 'Baicells BAIBLQ'),
        ('QRTB', 'QRTB Series', 'Baicells', 'Baicells QRTB; supports SC/CA/DC radio modes'),
        ('MLN', 'MLN Series', 'Baicells', 'Baicells MLN; supports SC/CA/DC radio modes'),
        ('BNQ', 'BaiBNQ 5G', 'Baicells', 'Baicells BaiBNQ; 5G NR base station; TR-069 parameters and KPI formulas use BaiBNQ')
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
        ('BLQ', 'BAIBLQ'),
        ('QRTB', 'QRTB Series'),
        ('MLN', 'MLN Series'),
        ('BNQ', 'BaiBNQ 5G')
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
