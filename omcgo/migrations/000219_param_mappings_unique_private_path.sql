-- +goose Up
-- ============================================================
-- 000219_param_mappings_unique_private_path.sql
--
-- 把 param_mappings 的唯一约束从 (param_model_id, standard_path) 改为
-- (param_model_id, private_path)。
--
-- 根因(2026-05-29):
--   - 旧约束 uniq_param_mappings_model_standard 假设 standard_path 在 paramModel
--     内唯一,但 BM.xml 等 XML 有意保留多 private_path 别名同一 standard_path
--     (例:X_COM_EUTRAULEarfcn 与 EUTRACarrierARFCN 两条 private 都映射到
--      standard EUTRACarrierARFCN),由 wangyong 2026-05-29 commit 0bad520d
--     的 TestBMNeighborListHasPrivateArfcnAlias 显式保护此设计。
--   - 旧约束让 Loader batch INSERT 撞 SQLSTATE 23505,整个 paramModel reload
--     失败 → dictload 启动期 module init failed → app 拒启动。
--   - Translator.standardToPrivate map 在重复 standardPath 时 last-wins 写入,
--     业务侧已接受"多 private 别名,反查任选一个"的语义。
--
-- 业务真实约束:
--   - private_path 是 paramModel 内的物理路径,**必须唯一**(物理位置不会重)
--   - standard_path 是规范路径,**允许 alias**(多 private 映射同一 standard)
--
-- 改造:
--   1. DROP 旧约束 uniq_param_mappings_model_standard
--   2. ADD  新约束 uniq_param_mappings_model_private(param_model_id, private_path)
--   3. 保留普通索引(性能):
--      - idx_param_mappings_model_private_active 已存在,不动
--      - 新增 idx_param_mappings_model_standard 给 standard_path 反查走索引
--        (非 UNIQUE,但替代旧 UNIQUE INDEX 的查询性能)
--
-- 升级风险:
--   - 若 DB 中已有 (model_id, private_path) 撞重的脏数据 → 本 migration 报错。
--     若发生,先手工查 SELECT param_model_id, private_path, count(*) FROM
--     param_mappings GROUP BY 1,2 HAVING count(*) > 1; 删除多余,再重跑。
--   - dev DB 通常干净,直接通过。
-- ============================================================

DROP INDEX IF EXISTS uniq_param_mappings_model_standard;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_param_mappings_model_private
    ON param_mappings(param_model_id, private_path);

CREATE INDEX IF NOT EXISTS idx_param_mappings_model_standard
    ON param_mappings(param_model_id, standard_path);


-- +goose Down
-- 回滚到旧约束。注:回滚前若有别名数据存在,DROP 新约束后 CREATE 旧约束会失败
-- (撞 standard_path 重复)。需先手工清理别名行才能回滚。
DROP INDEX IF EXISTS idx_param_mappings_model_standard;
DROP INDEX IF EXISTS uniq_param_mappings_model_private;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_param_mappings_model_standard
    ON param_mappings(param_model_id, standard_path);
