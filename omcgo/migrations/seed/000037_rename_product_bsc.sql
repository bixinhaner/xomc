-- 内置产品改名：BSC 网关 → BSC 产品（含 description）。#154
--
-- 为什么用 seed DML 而非只改 products.xml：
--   loader.go 的 UPSERT 冲突键是 product_name（ON CONFLICT (product_name)），改"名字"
--   在 loader 眼里是新增产品而非改名 —— 会 INSERT 新 BSC 产品(新 product_id)，旧 BSC 网关
--   行不被删，两条并存、设备仍挂旧 id。本 in-place UPDATE 保留 product_id 不变，关联的
--   devices 路由 / discovered_param_mappings / PM 结果按 product_id 的引用全部保持完好。
--
-- 配套：install.sh 升级时刷新 host param-mappings/products.xml(builtin,无 .custom sidecar)
--   为新名，避免 dictloader 用过时 host XML 把旧名 BSC 网关 重新 UPSERT 回来。
--
-- 幂等：NOT EXISTS 守卫使重跑安全；目标名已存在则不动。名字带空格，WHERE 精确匹配。

-- +goose Up
UPDATE products
SET product_name = 'BSC 产品',
    description  = 'Baicells BSC 产品'
WHERE product_name = 'BSC 网关'
  AND NOT EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 产品');

-- +goose Down
UPDATE products
SET product_name = 'BSC 网关',
    description  = 'Baicells BSC；皮基站 GSM 网关'
WHERE product_name = 'BSC 产品'
  AND NOT EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 网关');
