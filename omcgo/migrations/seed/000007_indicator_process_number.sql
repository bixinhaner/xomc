-- +goose Up
-- #865：PM 结果值规范化的 number 单位取整策略。
-- 可选值：intUp / intDown / intHalfUp / none。未知值由业务规范化逻辑按 intHalfUp 处理。

INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public, created_at, updated_at, description_i18n)
VALUES ('indicator.process', 'number', 'intHalfUp', 'string', 'PM 结果值规范化 number 单位取整策略（#865）', false, now(), now(), '{}'::jsonb)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs WHERE category = 'indicator.process' AND key = 'number';
