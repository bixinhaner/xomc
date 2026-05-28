-- +goose Up
-- ============================================================
-- 2026-05-28 修正 devices.technology 与 ProductRegistry 字典(products.tech)不一致的存量数据。
--
-- 背景:
--   device_service.applyProductMetadata / batch_processor.applyProductMetadataInline 之前
--   只回填 ModelName,没回填 Technology。设备首次 Inform 时 Technology 由 carrier/默认推断
--   写成 LTE,即使 ProductRegistry 知道这是 NR 产品也没纠正过。
--   后果:UFTE 5G 升级页面 device-candidates 过滤(WHERE technology='nr')0 行,
--        界面上选择 productClass=FAP/BSC7041C243 看不到任何设备。
--
-- 修正算法:
--   product_class_patterns.product_class 是 PostgreSQL 正则,与 devices.product_class
--   做 ~ 匹配,关联 products.tech 取权威值。
--   一台设备可能命中多条 pattern(sort_order 表达优先级)→ 用 DISTINCT ON + ORDER BY 取首条。
--
-- 之后再有新设备入库,代码已同步修复(同次提交),自然走对。
-- ============================================================

-- +goose StatementBegin
WITH device_correct_tech AS (
    SELECT DISTINCT ON (d.id)
           d.id           AS device_id,
           d.technology   AS old_tech,
           p.tech         AS new_tech,
           d.product_class
      FROM devices d
      JOIN product_class_patterns pcp
        ON pcp.is_active
       AND d.product_class IS NOT NULL
       AND d.product_class <> ''
       AND d.product_class ~ pcp.product_class
      JOIN products p
        ON p.id = pcp.product_id
       AND p.tech IS NOT NULL
       AND p.tech <> ''
     ORDER BY d.id, pcp.sort_order
)
UPDATE devices d
   SET technology = c.new_tech,
       updated_at = NOW()
  FROM device_correct_tech c
 WHERE d.id = c.device_id
   AND d.technology <> c.new_tech;
-- +goose StatementEnd


-- +goose Down
-- 反向:不回滚 — 把"修正过的对的值"再改回错的没有意义,且无法精确还原老 Technology
-- (老值已被覆盖,nothing to restore to)。
SELECT 1;
