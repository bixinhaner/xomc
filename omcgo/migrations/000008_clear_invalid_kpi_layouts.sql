-- +goose Up
-- Issue #817 (Follow-up): 清除 dashboard_kpi_layouts 表中使用了旧字符串别名（如 LTE_PDCP_VOLUME_DL）的脏数据。
-- 安全策略：利用 PostgreSQL 原生 JSONB 展开函数校验每一个 metric
-- 若任一 metric 不符合标准的 K/C 系统编号规则（如不以 K 或 C 或 KG 开头），则将该行制式的 layout 删掉。

DELETE FROM public.dashboard_kpi_layouts
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements(
        CASE WHEN jsonb_typeof(layout->'panels') = 'array' 
             THEN layout->'panels' 
             ELSE '[]'::jsonb 
        END
    ) AS p,
    jsonb_array_elements_text(
        CASE WHEN jsonb_typeof(p->'metrics') = 'array' 
             THEN p->'metrics' 
             ELSE '[]'::jsonb 
        END
    ) AS m
    WHERE m NOT SIMILAR TO '[KC][0-9]{9}|KGNB[0-9]{4}|KGSM[0-9]{4}'
);

-- +goose Down
-- 向下迁移不做任何操作，因为脏数据被清理是安全的单向操作。
