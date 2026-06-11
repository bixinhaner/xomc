-- 内置产品改名：BSC 网关 → BSC 产品（含 description），并清理重复孤儿。#154
--
-- 为什么用 seed DML 而非只改 products.xml：
--   loader.go 的 UPSERT 冲突键是 product_name（ON CONFLICT (product_name)），改"名字"
--   在 loader 眼里是新增产品而非改名 —— 会 INSERT 新 BSC 产品(新 id)，旧 BSC 网关 行不被
--   删、两条并存、设备仍挂旧 id。本 in-place UPDATE 保留 id，关联(devices / discovered_
--   param_mappings / PM 结果按 id)全部完好。
--
-- 目标终态：products 仅一行 'BSC 产品'，且保留有关联的那一行(id 不变)。兼顾两种现网态：
--   (1) 干净升级：只有 'BSC 网关' → 下方 UPDATE in-place 改名。
--   (2) 陷阱态：'BSC 网关' 与 'BSC 产品' 并存(dictloader 曾用过时 host XML 重建过名字)。
--       删掉"无设备且无发现映射"的孤儿那一行，保留有关联的行，再统一为 'BSC 产品'。
--       注意：devices.product_id 无 FK(删错会留悬挂引用)，discovered_param_mappings 是
--       CASCADE 的真实用户数据 —— 故孤儿判定须同时排除这两者；product_class_patterns 由
--       loader 重建、CASCADE 删除无碍。
--
-- 配套：install.sh 升级刷新 host param-mappings/products.xml(builtin)为新名，避免 loader
--   再次用旧 XML 重建 'BSC 网关'。幂等可重跑；Down 改名回退(已删孤儿不恢复,本是误建副本)。

-- +goose Up
-- (2-a) 删 'BSC 网关' 孤儿：仅当 'BSC 产品' 也在(兜底不删到唯一行)、且本行无 devices 无
--       discovered_param_mappings 时删除。
DELETE FROM products p
WHERE p.product_name = 'BSC 网关'
  AND EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 产品')
  AND NOT EXISTS (SELECT 1 FROM devices d WHERE d.product_id = p.id)
  AND NOT EXISTS (SELECT 1 FROM discovered_param_mappings m WHERE m.product_id = p.id);

-- (2-b) 删 'BSC 产品' 孤儿：仅当 'BSC 网关' 仍在(说明网关有关联被 2-a 保留)、且本行无关联时
--       删除。两者皆无关联时,2-a 已删网关 → 此处 EXISTS 网关 为假 → 保留产品(不会两行都删)。
DELETE FROM products p
WHERE p.product_name = 'BSC 产品'
  AND EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 网关')
  AND NOT EXISTS (SELECT 1 FROM devices d WHERE d.product_id = p.id)
  AND NOT EXISTS (SELECT 1 FROM discovered_param_mappings m WHERE m.product_id = p.id);

-- (1) 改名：剩余的 'BSC 网关' → 'BSC 产品'(in-place,保留 id + 关联)。幂等：目标已存在则不动。
UPDATE products
SET product_name = 'BSC 产品',
    description   = 'Baicells BSC 产品'
WHERE product_name = 'BSC 网关'
  AND NOT EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 产品');

-- +goose Down
-- 改名回退。注意：Up 删除的重复孤儿行无法恢复(本就是 dictloader 误建的副本,不应恢复)。
UPDATE products
SET product_name = 'BSC 网关',
    description   = 'Baicells BSC；皮基站 GSM 网关'
WHERE product_name = 'BSC 产品'
  AND NOT EXISTS (SELECT 1 FROM products WHERE product_name = 'BSC 网关');
