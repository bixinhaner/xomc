-- +goose Up
-- fix: mrOMCName 种子默认值为中文 'OMC 统一网管系统'，导致前端 i18n fallback 失效。
-- 将其重置为空字符串，前端 usePublicOmcName 遇到空值时自动回退到 t('app.title')，
-- 从而在英文环境下显示 'OMC Network Management'，中文环境显示 'OMC 统一网管系统'。
-- 管理员后续可在「系统配置 > 基本设置」中设置自定义品牌名称。
UPDATE public.sys_configs
SET value = '', updated_at = NOW()
WHERE category = 'basic'
  AND key = 'mrOMCName'
  AND value = 'OMC 统一网管系统';

-- +goose Down
UPDATE public.sys_configs
SET value = 'OMC 统一网管系统', updated_at = NOW()
WHERE category = 'basic'
  AND key = 'mrOMCName'
  AND value = '';
