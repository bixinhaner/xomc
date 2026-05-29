-- 一次性清理:删除 param_models 中"无加载源"的历史数据(用户拍板 2026-05-29)。
--
-- 背景:
--   - T-0098 P1-06(2026-04) Loader 入库时 loaded_from = filepath.Base(path),裸 basename
--   - T-0178(2026-05) 加 loaded_from 前缀语义:"param-mappings/X.xml" 或
--     "param-mappings-custom/Y.xml";migration 000215 已回填存量带 builtin 前缀
--   - 仍可能残留:① migration 215 前手工 INSERT 的行 ② 测试期插入未经 Loader 的行
--     ③ loaded_from 列空或非标准前缀的历史孤儿
--   - 这些行 ClassifySource 返 SourceUnknown,IsDeletable=false 永远不可在线删,
--     在 product/param-model 页面以"未知"Tag 显示噪声,运维无法清理
--
-- 决策(2026-05-29):
--   1. 前端不显示 unknown source 模型
--   2. 后端 API 不返回 unknown source 模型
--   3. 升级当前版本时,清空一次历史 unknown 数据(本迁移)
--
-- 行为:
--   - DELETE param_models WHERE COALESCE(loaded_from,'') 既不带 builtin 前缀也不带 custom 前缀
--   - FK 约束自动级联(migrations/000058 已定义):
--     · param_mappings.param_model_id REFERENCES param_models(id) ON DELETE CASCADE
--       → 被删 param_model 的所有 mapping 一并删
--     · products.param_model_id REFERENCES param_models(id) ON DELETE SET NULL
--       → 关联 product 的 param_model_id 置空(产品行保留)
--     · 其他引用方(若有)同样 ON DELETE SET NULL
--   - 删除前 RAISE NOTICE 统计;Loader 下次扫描发现 XML 仍在会重建带正确前缀的行(self-healing)
--
-- 幂等性:
--   - WHERE 子句自然幂等(满足条件的行被删后,重跑零匹配 0 行)
--   - 重跑无副作用
--
-- 不可回滚:
--   - Down 段无法还原已删数据(数据已物理 DELETE)
--   - Down 仅做 no-op + 显式说明(避免 goose down 报错也避免误以为可恢复)

-- +goose Up

-- +goose StatementBegin
DO $$
DECLARE
    cnt INT;
BEGIN
    SELECT COUNT(*) INTO cnt
      FROM param_models
     WHERE COALESCE(loaded_from, '') NOT LIKE 'param-mappings/%'
       AND COALESCE(loaded_from, '') NOT LIKE 'param-mappings-custom/%';
    RAISE NOTICE 'T-0178 follow-up: 即将删除 % 个无加载源的 param_models 行(FK 自动级联 param_mappings ON DELETE CASCADE / products.param_model_id ON DELETE SET NULL)', cnt;
END $$;
-- +goose StatementEnd

DELETE FROM param_models
 WHERE COALESCE(loaded_from, '') NOT LIKE 'param-mappings/%'
   AND COALESCE(loaded_from, '') NOT LIKE 'param-mappings-custom/%';

-- +goose Down

-- 不可回滚:已 DELETE 的行无法还原。
-- 若需恢复,请从备份还原或重跑 Loader(builtin XML 会自动重建,custom 需重新上传)。
-- 留空 SQL 以让 goose down 正常返回(no-op),记录到 goose_db_version 但不动数据。
SELECT 1;
