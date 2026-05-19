-- T-DRULE-SN: 通过 device_groups.matching_mode='serialNumber' 给指定 SN 列表
--             自动归组。
--
-- 业务目的：用户提供两个具体 SN（120200024719AAB0027 / 120200024719AAB0039）
--           需要自动划分到同一个分组；设备每次上报心跳时由 inform_handler
--           异步调用 topology.DeviceMatcher 完成匹配 + 入组。
--
-- 依赖：
--   - migration 000124 已加 serial_number_list TEXT[] 列 + 'serialNumber'
--     matching_mode 选项
--   - seed/000001 已建默认一级分组 '00000000-0000-0000-0000-000000000001'
--
-- 设计：
--   - L2 子组挂在默认 L1 父组下（符合 chk_dg_level_parent CHECK）
--   - matching_mode='serialNumber' + serial_number_list 精确成员列表
--   - 固定 UUID 便于幂等（重复执行 ON CONFLICT DO UPDATE 刷新 SN 列表）
--   - 加 SN 时只需 UPDATE 本行 serial_number_list，不必新增分组

-- +goose Up

INSERT INTO device_groups (
    id, name, parent_id, level, is_default, status, sort_order,
    description, remark, created_by,
    matching_mode, serial_number_list
) VALUES (
    'd0d0d0d0-aab0-0027-0039-000000000001'::uuid,
    'AAB0027-0039 自动分组',
    '00000000-0000-0000-0000-000000000001'::uuid,
    2,
    FALSE,
    'active',
    100,
    '通过 SN 列表自动匹配的设备分组（T-DRULE-SN）',
    '心跳路径自动归组：matching_mode=serialNumber + serial_number_list',
    'system',
    'serialNumber',
    ARRAY['120200024719AAB0027', '120200024719AAB0039']::TEXT[]
)
ON CONFLICT (id) DO UPDATE SET
    matching_mode      = EXCLUDED.matching_mode,
    serial_number_list = EXCLUDED.serial_number_list,
    description        = EXCLUDED.description,
    updated_at         = NOW();

-- 立即把已存在的设备（如果 SN 已在 devices 表）入组，免等下次心跳。
-- 与运行时 matcher 一致：device_group_members UNIQUE(device_id) 强制单组归属，
-- ON CONFLICT (device_id) DO UPDATE 切换分组到新规则组（与 matcher 行为对齐）。
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT 'd0d0d0d0-aab0-0027-0039-000000000001'::uuid, d.id, NOW()
FROM devices d
WHERE d.serial_number IN ('120200024719AAB0027', '120200024719AAB0039')
  AND d.deleted_at IS NULL
ON CONFLICT (device_id) DO UPDATE SET
    group_id = EXCLUDED.group_id,
    added_at = NOW();

-- +goose Down

DELETE FROM device_group_members
WHERE group_id = 'd0d0d0d0-aab0-0027-0039-000000000001'::uuid;

DELETE FROM device_groups
WHERE id = 'd0d0d0d0-aab0-0027-0039-000000000001'::uuid;
