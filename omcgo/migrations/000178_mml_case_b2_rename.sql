-- +goose Up
-- ============================================================
-- 000178_mml_case_b2_rename.sql
-- MML Case B-2 21 簇命令重命名 — 让同名但功能不同的命令在 UI 上可区分
--
-- 背景：000177 去重后剩 21 个 (command_name, op_type) 簇，共 45 行。这些行
-- target_paths 真实不同（不同硬件层级 / 不同子树 / 不同模式），不能删除；但 UI
-- 上显示同名 → 用户无法区分。本次仅 UPDATE command_name 和 command_name_i18n
-- 字段，按技术差异加后缀。**不删行、不动 target_paths/target_object/sub_field**。
--
-- 命名后缀策略：
--   - VlanInterface_i in command_code → " (VLAN)"，否则 " (物理口)"  [IPv4/IPv6]
--   - ConnMode IRAT vs IdleMode IRAT  → " (连接态)" / " (空闲态)"
--   - FAPService_i_Capabilities vs CellConfig_Capabilities → " (FAP)" / " (小区)"
--   - MRMgmt vs PerfMgmt → "MR " / "PM " 前缀替换
--   - SwUpgrade 5 个嵌套层级 → 按 MU/Slot/EU/RU 标注
-- ============================================================

-- 用 VALUES 物化"command_code → 新 command_name"映射，便于审计
-- +goose StatementBegin
DO $$
DECLARE
    rec    RECORD;
    rename_map CONSTANT TEXT[][] := ARRAY[
        -- IPv4 地址 (8 行：ADD/LST/MOD/RMV × 物理口/VLAN)
        ['ADD_Device_Ethernet_Interface_i_IPv4Address_i',                     'ADD IPv4 地址 (物理口)'],
        ['ADD_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'ADD IPv4 地址 (VLAN)'],
        ['LST_Device_Ethernet_Interface_i_IPv4Address_i',                     'LST IPv4 地址 (物理口)'],
        ['LST_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'LST IPv4 地址 (VLAN)'],
        ['MOD_Device_Ethernet_Interface_i_IPv4Address_i',                     'MOD IPv4 地址 (物理口)'],
        ['MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'MOD IPv4 地址 (VLAN)'],
        ['RMV_Device_Ethernet_Interface_i_IPv4Address_i',                     'RMV IPv4 地址 (物理口)'],
        ['RMV_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'RMV IPv4 地址 (VLAN)'],

        -- IPv6 地址 (8 行：同上)
        ['ADD_Device_Ethernet_Interface_i_IPv6Address_i',                     'ADD IPv6 地址 (物理口)'],
        ['ADD_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'ADD IPv6 地址 (VLAN)'],
        ['LST_Device_Ethernet_Interface_i_IPv6Address_i',                     'LST IPv6 地址 (物理口)'],
        ['LST_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'LST IPv6 地址 (VLAN)'],
        ['MOD_Device_Ethernet_Interface_i_IPv6Address_i',                     'MOD IPv6 地址 (物理口)'],
        ['MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'MOD IPv6 地址 (VLAN)'],
        ['RMV_Device_Ethernet_Interface_i_IPv6Address_i',                     'RMV IPv6 地址 (物理口)'],
        ['RMV_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'RMV IPv6 地址 (VLAN)'],

        -- IRAT 测量 (8 行：ADD/LST/MOD/RMV × 连接态/空闲态)
        ['ADD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'ADD IRAT 测量 (连接态)'],
        ['ADD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'ADD IRAT 测量 (空闲态)'],
        ['LST_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'LST IRAT 测量 (连接态)'],
        ['LST_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'LST IRAT 测量 (空闲态)'],
        ['MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'MOD IRAT 测量 (连接态)'],
        ['MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'MOD IRAT 测量 (空闲态)'],
        ['RMV_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'RMV IRAT 测量 (连接态)'],
        ['RMV_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'RMV IRAT 测量 (空闲态)'],

        -- 能力集 (8 行：ADD/LST/MOD/RMV × FAP 级/小区配置级)
        ['ADD_Device_Services_FAPService_i_Capabilities',                'ADD FAP 能力集'],
        ['ADD_Device_Services_FAPService_i_CellConfig_Capabilities',     'ADD 小区能力集'],
        ['LST_Device_Services_FAPService_i_Capabilities',                'LST FAP 能力集'],
        ['LST_Device_Services_FAPService_i_CellConfig_Capabilities',     'LST 小区能力集'],
        ['MOD_Device_Services_FAPService_i_Capabilities',                'MOD FAP 能力集'],
        ['MOD_Device_Services_FAPService_i_CellConfig_Capabilities',     'MOD 小区能力集'],
        ['RMV_Device_Services_FAPService_i_Capabilities',                'RMV FAP 能力集'],
        ['RMV_Device_Services_FAPService_i_CellConfig_Capabilities',     'RMV 小区能力集'],

        -- 配置 (8 行：ADD/LST/MOD/RMV × MR/PM)
        ['ADD_Device_FAP_MRMgmt_Config_i',                               'ADD MR 配置'],
        ['ADD_Device_FAP_PerfMgmt_Config_i',                             'ADD PM 配置'],
        ['LST_Device_FAP_MRMgmt_Config_i',                               'LST MR 配置'],
        ['LST_Device_FAP_PerfMgmt_Config_i',                             'LST PM 配置'],
        ['MOD_Device_FAP_MRMgmt_Config_i',                               'MOD MR 配置'],
        ['MOD_Device_FAP_PerfMgmt_Config_i',                             'MOD PM 配置'],
        ['RMV_Device_FAP_MRMgmt_Config_i',                               'RMV MR 配置'],
        ['RMV_Device_FAP_PerfMgmt_Config_i',                             'RMV PM 配置'],

        -- 设备版本升级 (5 行：仅 LST，按 MU/Slot/EU/RU 嵌套层级)
        ['LST_Device_DeviceInfo_SwUpgrade',                              'LST 设备版本升级 (整机)'],
        ['LST_Device_DeviceInfo_MU_i_SwUpgrade',                         'LST 设备版本升级 (MU)'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_SwUpgrade',                  'LST 设备版本升级 (MU/Slot)'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_EU_i_SwUpgrade',             'LST 设备版本升级 (MU/Slot/EU)'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_EU_i_RU_i_SwUpgrade',        'LST 设备版本升级 (MU/Slot/EU/RU)']
    ];
    rows_updated INT := 0;
    affected     INT;
BEGIN
    FOR i IN 1..array_length(rename_map, 1) LOOP
        UPDATE mml_commands
           SET command_name = rename_map[i][2],
               command_name_i18n = CASE
                   WHEN command_name_i18n IS NULL OR command_name_i18n = '{}'::jsonb
                       THEN jsonb_build_object('zh-CN', rename_map[i][2])
                   ELSE jsonb_set(command_name_i18n, '{zh-CN}',
                                  to_jsonb(rename_map[i][2]))
               END,
               updated_at = NOW()
         WHERE command_code = rename_map[i][1];
        GET DIAGNOSTICS affected = ROW_COUNT;
        rows_updated := rows_updated + affected;
    END LOOP;

    RAISE NOTICE '000178: renamed % rows across 21 Case B-2 clusters', rows_updated;

    IF rows_updated <> 45 THEN
        RAISE WARNING '000178: expected 45 renames, got %; possible data drift', rows_updated;
    END IF;
END $$;
-- +goose StatementEnd

-- 自检：21 簇全部消除（新 command_name 唯一）
-- +goose StatementBegin
DO $$
DECLARE
    leftover INT;
BEGIN
    SELECT COUNT(*) INTO leftover
      FROM (SELECT command_name, operation_type
              FROM mml_commands
             GROUP BY command_name, operation_type
            HAVING COUNT(*) > 1) x;

    RAISE NOTICE '000178 post-check: remaining duplicate clusters by name = %', leftover;

    IF leftover > 0 THEN
        RAISE EXCEPTION '000178: rename did not fully clear duplicates: % clusters remain', leftover;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：将 45 行的 command_name 还原为带后缀前的原值。
-- 注意 command_name_i18n.zh-CN 同步还原。
-- ============================================================
-- +goose StatementBegin
DO $$
DECLARE
    revert_map CONSTANT TEXT[][] := ARRAY[
        ['ADD_Device_Ethernet_Interface_i_IPv4Address_i',                     'ADD IPv4 地址'],
        ['ADD_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'ADD IPv4 地址'],
        ['LST_Device_Ethernet_Interface_i_IPv4Address_i',                     'LST IPv4 地址'],
        ['LST_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'LST IPv4 地址'],
        ['MOD_Device_Ethernet_Interface_i_IPv4Address_i',                     'MOD IPv4 地址'],
        ['MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'MOD IPv4 地址'],
        ['RMV_Device_Ethernet_Interface_i_IPv4Address_i',                     'RMV IPv4 地址'],
        ['RMV_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i',     'RMV IPv4 地址'],
        ['ADD_Device_Ethernet_Interface_i_IPv6Address_i',                     'ADD IPv6 地址'],
        ['ADD_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'ADD IPv6 地址'],
        ['LST_Device_Ethernet_Interface_i_IPv6Address_i',                     'LST IPv6 地址'],
        ['LST_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'LST IPv6 地址'],
        ['MOD_Device_Ethernet_Interface_i_IPv6Address_i',                     'MOD IPv6 地址'],
        ['MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'MOD IPv6 地址'],
        ['RMV_Device_Ethernet_Interface_i_IPv6Address_i',                     'RMV IPv6 地址'],
        ['RMV_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i',     'RMV IPv6 地址'],
        ['ADD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'ADD IRAT 测量'],
        ['ADD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'ADD IRAT 测量'],
        ['LST_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'LST IRAT 测量'],
        ['LST_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'LST IRAT 测量'],
        ['MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'MOD IRAT 测量'],
        ['MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'MOD IRAT 测量'],
        ['RMV_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT', 'RMV IRAT 测量'],
        ['RMV_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT', 'RMV IRAT 测量'],
        ['ADD_Device_Services_FAPService_i_Capabilities',                'ADD 能力集'],
        ['ADD_Device_Services_FAPService_i_CellConfig_Capabilities',     'ADD 能力集'],
        ['LST_Device_Services_FAPService_i_Capabilities',                'LST 能力集'],
        ['LST_Device_Services_FAPService_i_CellConfig_Capabilities',     'LST 能力集'],
        ['MOD_Device_Services_FAPService_i_Capabilities',                'MOD 能力集'],
        ['MOD_Device_Services_FAPService_i_CellConfig_Capabilities',     'MOD 能力集'],
        ['RMV_Device_Services_FAPService_i_Capabilities',                'RMV 能力集'],
        ['RMV_Device_Services_FAPService_i_CellConfig_Capabilities',     'RMV 能力集'],
        ['ADD_Device_FAP_MRMgmt_Config_i',                               'ADD 配置'],
        ['ADD_Device_FAP_PerfMgmt_Config_i',                             'ADD 配置'],
        ['LST_Device_FAP_MRMgmt_Config_i',                               'LST 配置'],
        ['LST_Device_FAP_PerfMgmt_Config_i',                             'LST 配置'],
        ['MOD_Device_FAP_MRMgmt_Config_i',                               'MOD 配置'],
        ['MOD_Device_FAP_PerfMgmt_Config_i',                             'MOD 配置'],
        ['RMV_Device_FAP_MRMgmt_Config_i',                               'RMV 配置'],
        ['RMV_Device_FAP_PerfMgmt_Config_i',                             'RMV 配置'],
        ['LST_Device_DeviceInfo_SwUpgrade',                              'LST 设备版本升级'],
        ['LST_Device_DeviceInfo_MU_i_SwUpgrade',                         'LST 设备版本升级'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_SwUpgrade',                  'LST 设备版本升级'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_EU_i_SwUpgrade',             'LST 设备版本升级'],
        ['LST_Device_DeviceInfo_MU_i_Slot_i_EU_i_RU_i_SwUpgrade',        'LST 设备版本升级']
    ];
BEGIN
    FOR i IN 1..array_length(revert_map, 1) LOOP
        UPDATE mml_commands
           SET command_name = revert_map[i][2],
               command_name_i18n = jsonb_set(
                   COALESCE(command_name_i18n, '{}'::jsonb),
                   '{zh-CN}',
                   to_jsonb(revert_map[i][2])
               ),
               updated_at = NOW()
         WHERE command_code = revert_map[i][1];
    END LOOP;
END $$;
-- +goose StatementEnd
