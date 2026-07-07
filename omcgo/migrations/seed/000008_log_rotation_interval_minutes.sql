-- +goose Up
-- #883：运行期日志文件定时轮转间隔，页面按分钟配置。

INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public, created_at, updated_at, description_i18n)
VALUES ('log.rotation', 'rotate_interval_minutes', '5', 'int', '日志文件定时轮转间隔（分钟）', false, now(), now(), '{}'::jsonb)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs WHERE category = 'log.rotation' AND key = 'rotate_interval_minutes';
