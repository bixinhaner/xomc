-- 隐藏一级菜单"配置管理"
-- 前端 navConfig.ts 里该目录已注释，但 DB 中 show_status='show' 仍会在动态菜单中渲染。
-- 此处把目录及其唯一子菜单"批量参数模板"一并置 hide。

-- +goose Up
UPDATE menus SET show_status = 'hide', updated_at = NOW()
 WHERE id IN (
     'aaaa0120-0000-0000-0000-000000000001'::uuid,  -- 配置管理（目录）
     'aaaa0120-1000-0000-0000-000000000001'::uuid   -- 批量参数模板
 );

-- +goose Down
UPDATE menus SET show_status = 'show', updated_at = NOW()
 WHERE id IN (
     'aaaa0120-0000-0000-0000-000000000001'::uuid,
     'aaaa0120-1000-0000-0000-000000000001'::uuid
 );
