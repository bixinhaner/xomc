-- +goose Up
-- 告警库自定义 XML 上传(严格对标 T-0180 indicator)前置数据迁移。
--
-- 背景:告警 Loader 改为 builtin(alarm-definitions/) + custom(alarm-definitions-custom/)
-- 双目录合并扫描后,loaded_from 列改存"含目录前缀"的相对路径(如 alarm-definitions/ENB.xml);
-- ClassifySource 据前缀判定 builtin/custom。历史行只存 basename(ENB.xml),需回填前缀,
-- 否则会被判为 unknown(不可删,且前端来源列显示异常)。
--
-- 规则:仅回填"非空且不含 /"的 basename 行(即旧 Loader 写入的 builtin 文件);
-- 空值(运维手工新增、无 XML)保持不动;已含前缀的行(本迁移重复执行)跳过 → 幂等。
UPDATE alarm_definitions
   SET loaded_from = 'alarm-definitions/' || loaded_from
 WHERE loaded_from IS NOT NULL
   AND loaded_from <> ''
   AND loaded_from NOT LIKE '%/%';

COMMENT ON COLUMN public.alarm_definitions.loaded_from IS
'Loader 来源 XML 相对路径(含目录前缀,如 "alarm-definitions/ENB.xml" 或 "alarm-definitions-custom/MY.xml");NULL/空 表示运维手工新增(无 XML 来源)。ClassifySource 据前缀判定 builtin/custom。';

-- +goose Down
-- 回滚:剥离 builtin 前缀,恢复为 basename(custom 行不受影响,因前缀不匹配)。
UPDATE alarm_definitions
   SET loaded_from = regexp_replace(loaded_from, '^alarm-definitions/', '')
 WHERE loaded_from LIKE 'alarm-definitions/%';

COMMENT ON COLUMN public.alarm_definitions.loaded_from IS
'T-0179 Loader 来源 XML 文件名(如 "ENB.xml");NULL 表示历史数据未回填,Reload 后会自动写入';
