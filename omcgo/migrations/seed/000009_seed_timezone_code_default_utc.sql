-- +goose Up
-- ISSUE-456（子单 A / 母单 #455）：建立"系统时区"唯一源的默认种子。
--
-- 系统时区存于 sys_configs（category='basic', key='timezoneCode'），默认 UTC。
-- app 响应层（子单 B）与 worker 聚合（子单 C）共用统一读取入口
-- internal/core/systimezone.Provider 从此读取，解析失败回落 UTC。
--
-- 幂等：ON CONFLICT DO UPDATE，仅当现值为空白（空串/纯空格）时回填默认 UTC——
--   · 缺失行    → INSERT 默认 UTC；
--   · 空白残行  → 回填 UTC（既有库里可能先于本种子存在一条 value='' 的半残行，
--                 它既不是有效运维值也不是合法默认值，必须修正为默认 UTC，否则
--                 统一读取入口会回落 UTC 但落库值仍为空，"系统时区唯一源"形同虚设）；
--   · 有效运维值（运维已设过具体时区）→ WHERE 不命中，保留不覆盖（幂等不破坏运维设置）。
-- 号段：本分支 seed 现有最大 000006，但 #453(000007)/#454(000008) 并行 PR 已占用，
-- 本单避让取 000009（参照并行 PR 迁移号撞号教训）。
INSERT INTO public.sys_configs (category, key, value, value_type)
VALUES ('basic', 'timezoneCode', 'UTC', 'string')
ON CONFLICT (category, key) DO UPDATE
   SET value = EXCLUDED.value,
       updated_at = NOW()
 WHERE btrim(public.sys_configs.value) = '';

-- +goose Down
-- 回滚删除该种子行。配对 .up，演练用。
DELETE FROM public.sys_configs WHERE category = 'basic' AND key = 'timezoneCode';
