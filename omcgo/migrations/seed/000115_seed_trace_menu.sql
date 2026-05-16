-- +goose Up
-- T-0137 TR069 报文跟踪：注入运维管理下的二级菜单 + 三角色（admin/operator/viewer）绑定
-- +goose StatementBegin
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order,
                   route_path, component_path, icon, show_status, status, name_i18n)
VALUES (
    'aaaa000a-1000-0000-0000-000000000006'::uuid,
    'TR069报文跟踪',
    'menu',
    'ops:message-trace',
    'aaaa000a-0000-0000-0000-000000000001'::uuid,
    6,
    '/ops/message-trace',
    'ops/MessageTrace',
    'monitor',
    'show',
    'normal',
    '{"en-US": "TR069 Message Trace", "zh-CN": "TR069 报文跟踪"}'::jsonb
) ON CONFLICT (id) DO NOTHING;

INSERT INTO role_menus (role_id, menu_id)
SELECT r.id, 'aaaa000a-1000-0000-0000-000000000006'::uuid
FROM roles r
WHERE r.name IN ('admin','operator','viewer')
ON CONFLICT (role_id, menu_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_menus WHERE menu_id = 'aaaa000a-1000-0000-0000-000000000006'::uuid;
DELETE FROM menus WHERE id = 'aaaa000a-1000-0000-0000-000000000006'::uuid;
-- +goose StatementEnd
