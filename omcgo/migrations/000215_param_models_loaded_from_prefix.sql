-- T-0178 P1: param_models.loaded_from 加目录前缀,与 source.go::ClassifySource 契约对齐。
--
-- 背景:
--   - T-0098 P1-06 Loader 入库时 loaded_from = filepath.Base(path),写裸 basename 如 "BTS.xml"
--   - T-0178 改 Loader.loadParamModelFile 入库为 filepath.Rel(base, path) | ToSlash,
--     写 "param-mappings/BTS.xml" / "param-mappings-custom/CBQQ.xml"
--   - 历史数据无前缀 → ClassifySource 返 SourceUnknown → IsDeletable=false(保守拒删)
--     此迁移前提下行为退化为"内置不可删,且无法识别为内置",对存量没有功能性破坏,
--     但前端 source/deletable 字段全显示 "unknown",UX 不佳
--
-- 行为:
--   - 仅给"既不带 builtin 前缀 又不带 custom 前缀"的非空 loaded_from 加 "param-mappings/" 前缀
--   - 等同于"已存量历史数据 = 出厂内置 XML"的合理假设(custom XML 在 T-0178 之前不存在,
--     custom 目录从 T-0178 起才存在)
--
-- 幂等:NOT LIKE 守门 → 重跑 Up 不会重复加前缀
--
-- Down:逆向脱前缀(用 regexp_replace 一次性剥离两种前缀之一)
--      用于回滚到 T-0098 P1-06 行为(loader.go::loadParamModelFile 也需要回退,否则
--      下次 Reload 又会加上前缀。Down 仅供紧急回滚用,通常不会调)。

-- +goose Up

UPDATE param_models
   SET loaded_from = 'param-mappings/' || loaded_from
 WHERE loaded_from IS NOT NULL
   AND loaded_from <> ''
   AND loaded_from NOT LIKE 'param-mappings/%'
   AND loaded_from NOT LIKE 'param-mappings-custom/%';

-- +goose Down

UPDATE param_models
   SET loaded_from = regexp_replace(loaded_from, '^param-mappings(-custom)?/', '')
 WHERE loaded_from LIKE 'param-mappings%/%';
