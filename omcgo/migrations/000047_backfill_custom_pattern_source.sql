-- +goose Up
-- issue #206：保留数据升级后用户自建产品「正则」丢失 —— 存量来源标记修正（防御）。
--
-- 根因：product_class_patterns.source 列在加入时定义为 DEFAULT 'builtin' NOT NULL 且
-- 无 backfill（pre-baseline 000031 → consolidated 000001_init_schema.sql:5225）。在该列
-- 加入之前创建的「用户自建产品」正则被静默打成 'builtin'；而 loader 销毁式 reload 会删
-- builtin 正则、再从 products.xml 重插——用户产品不在 products.xml，删后不重插 → 正则列空。
--
-- 本迁移把误标 'builtin' 的「用户自建产品」（is_builtin=FALSE）正则改回 'custom'，
-- 救「已过加列迁移但尚未触发 reload」的部署（已被 DELETE 的行不可逆，不在此恢复）。
-- 与 loader.go 的 reload 守卫（只删内置产品的 builtin 正则）叠加即闭环：本迁移修存量来源
-- 标记，loader 守卫防再删。
--
-- 幂等：WHERE source='builtin' AND product_id IN (is_builtin=FALSE 产品) 守卫，
-- 重复前向应用对已修正的行无副作用（命中集合稳定收敛）。
UPDATE public.product_class_patterns
SET source = 'custom'
WHERE source = 'builtin'
  AND product_id IN (SELECT id FROM public.products WHERE is_builtin = FALSE);

-- +goose Down
-- no-op：数据修正不可逆回滚。把 source 从 'custom' 改回 'builtin' 既无法区分本迁移
-- 修正的行与用户后续手工建的 custom 正则，也会重新引入 #206 的丢失风险，故不提供回滚。
SELECT 1;
