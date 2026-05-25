-- +goose Up
-- ============================================================
-- 000186_products_tech_normalize.sql
-- 把 products.tech 列的存量值统一到 canonical 小写代码（lte / nr / gsm）。
--
-- 背景：products.xml 用人类可读的 "4G" / "5G" / "2G"（与基带产品标签习惯一致），
-- Loader 直接灌进 DB → DB 列里也是 "4G" 等。但前端 API 参数 / devices.technology
-- 列 / global.Technology 常量都用 canonical "lte" / "nr" / "gsm"。
-- 结果 GET /api/v1/products?tech=lte 永远查不到任何记录。
--
-- 修复方向（用户决策 2026-05-25）：以前端为准 —— DB 改成 canonical。
--   - Loader 端已加 model.NormalizeTechnology() 写前归一（同步本迁移落地）
--   - 本迁移做存量数据一次性转换
--
-- 转换表（与 global.NormalizeTechnology 同款）：
--   '4G' / 'LTE'  → 'lte'
--   '5G' / 'NR'   → 'nr'
--   '2G' / 'GSM'  → 'gsm'
--   其它值原样保留（大小写归一 + trim），让上游 IsValid 判定是否拒收。
--
-- 重跑安全：纯 UPDATE，幂等 — 已是 lte/nr/gsm 的行 LOWER+TRIM 后值不变。
-- ============================================================

UPDATE products
SET tech = CASE LOWER(TRIM(tech))
    WHEN '4g'       THEN 'lte'
    WHEN 'lte'      THEN 'lte'
    WHEN '4g lte'   THEN 'lte'
    WHEN 'lte fdd'  THEN 'lte'
    WHEN 'lte tdd'  THEN 'lte'
    WHEN 'fdd-lte'  THEN 'lte'
    WHEN 'tdd-lte'  THEN 'lte'
    WHEN '5g'       THEN 'nr'
    WHEN 'nr'       THEN 'nr'
    WHEN '5g nr'    THEN 'nr'
    WHEN '5g nr sa' THEN 'nr'
    WHEN '5g sa'    THEN 'nr'
    WHEN '2g'       THEN 'gsm'
    WHEN 'gsm'      THEN 'gsm'
    WHEN '2g gsm'   THEN 'gsm'
    ELSE LOWER(TRIM(tech))
END
WHERE tech IS NOT NULL;

-- +goose Down
-- 回滚不可逆 —— 历史 "4G" 别名都已折成 "lte"，原值无法区分（如 "lte fdd"/"lte tdd"
-- 都被合并）。本 down 段把 canonical 值粗略回写为大写代码（"lte" → "LTE" 等），
-- 让 API 层"严格匹配 'lte'"的代码立刻失效，方便排障定位回滚后果。生产请勿轻易跑。
UPDATE products
SET tech = CASE tech
    WHEN 'lte' THEN '4G'
    WHEN 'nr'  THEN '5G'
    WHEN 'gsm' THEN '2G'
    ELSE tech
END
WHERE tech IS NOT NULL;
