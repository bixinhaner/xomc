-- +goose Up
-- +goose StatementBegin

-- T-0158: 把"设备异常日志"从"日志管理"挪到"设备管理"下，改名"异常重启记录"。
-- 菜单 ID 保持不变（aaaa0007-1000-0000-0000-000000000002），避免破坏已绑定的
-- role_menus 关联（按 menu_id 关联，与 permission_key 字符串解耦）。

UPDATE menus
   SET parent_id      = '11111111-1111-1111-1111-111111111101',  -- 设备管理
       name           = '重启记录',
       name_i18n      = jsonb_build_object(
                            'zh-CN', '重启记录',
                            'en-US', 'Reboot Records'
                        ),
       route_path     = '/device/abnormal-reboot',
       permission_key = 'device:abnormal-reboot',
       sort_order     = 50,
       icon           = 'WarningOutlined',
       component_path = 'device/AbnormalReboot',
       updated_at     = NOW()
 WHERE id = 'aaaa0007-1000-0000-0000-000000000002';

-- 按钮权限码同步：log:exception:* → device:abnormal-reboot:*
UPDATE menus SET permission_key = 'device:abnormal-reboot:query',  updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'log:exception:query';
UPDATE menus SET permission_key = 'device:abnormal-reboot:add',    updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'log:exception:add';
UPDATE menus SET permission_key = 'device:abnormal-reboot:edit',   updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'log:exception:edit';
UPDATE menus SET permission_key = 'device:abnormal-reboot:delete', updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'log:exception:delete';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE menus
   SET parent_id      = 'aaaa0007-0000-0000-0000-000000000001',  -- 日志管理
       name           = '设备异常日志',
       name_i18n      = jsonb_build_object(
                            'zh-CN', '设备异常日志',
                            'en-US', 'Device Exception Log'
                        ),
       route_path     = '/log/exception',
       permission_key = 'log:exception',
       sort_order     = 2,
       icon           = 'WarningOutlined',
       component_path = 'log/ExceptionLog',
       updated_at     = NOW()
 WHERE id = 'aaaa0007-1000-0000-0000-000000000002';

UPDATE menus SET permission_key = 'log:exception:query',  updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'device:abnormal-reboot:query';
UPDATE menus SET permission_key = 'log:exception:add',    updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'device:abnormal-reboot:add';
UPDATE menus SET permission_key = 'log:exception:edit',   updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'device:abnormal-reboot:edit';
UPDATE menus SET permission_key = 'log:exception:delete', updated_at = NOW() WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_key = 'device:abnormal-reboot:delete';

-- +goose StatementEnd
