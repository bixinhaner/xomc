-- +goose Up
-- ISSUE-454：把「OMC 名称」配置接通为系统品牌标题（左上角 + 登录页，三皮肤跟随）。
--
-- 背景：系统配置 > 基本设置页有「OMC名称」输入项（前端字段 mrOMCName，存
-- sys_configs.category='basic'），但 sys_configs 没有该 key 的种子记录，前端
-- 也无代码读取——属半成品死字段。本单接通该配置驱动品牌标题。
--
-- 登录页大标题要在登录前（无 token）即显示，走免登录公开通道
-- /admin/public/configs（仅返回 is_public=true 的项），故种子行必须 is_public=true。
--
-- 幂等：ON CONFLICT DO NOTHING——若运维已在配置页填过该 key（行已存在），
-- 不覆盖其值；行不存在则灌入默认名「OMC 统一网管系统」（与 v1 i18n app.title 对齐）。
--
-- 关键正确性点（与本迁移配套，无需改本文件）：配置页保存（BatchUpsert）的
-- ON CONFLICT 只 UPDATE value + updated_at，不触碰 is_public——故种子设的
-- is_public=true 在用户多次保存后仍保留，公开通道始终可读（见 sys_config.go BatchUpsert）。
INSERT INTO public.sys_configs (category, key, value, value_type, description, is_public) VALUES
  ('basic', 'mrOMCName', 'OMC 统一网管系统', 'string', 'OMC 名称（系统品牌标题，左上角+登录页跟随显示）（#454）', true)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
-- 删除本迁移灌入的种子行。若运维已自定义过该名称，回滚一并删除（无从区分默认/自定义，
-- 且该 key 仅本单引入）。
DELETE FROM public.sys_configs WHERE category = 'basic' AND key = 'mrOMCName';
