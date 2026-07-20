-- +goose Up
-- Prune DEVICE_INFO MML command bindings according to
-- /Users/wangyong/OBJECT/Codex/MML配置参数分组分析报告-设备信息重分组-20260720.md
-- Also normalize SB/software-version bindings against the CMCC 5G v1.9.4 and
-- TD-LTE V2.3 southbound model specs.
-- Also normalize SC/management-server bindings against the same specs.
-- Also normalize SE/log-management bindings against the same specs.
-- Also normalize SD/alarm-parameter command bindings to read-only query pages:
-- keep only current-alarm and history-alarm LST commands; remove write actions
-- and unsupported auxiliary alarm views from the MML page.
--
-- Policy: do not migrate these paths into other groups here. Other groups already
-- own their copies. Keep only true device-info paths on DEVICE_INFO commands and
-- delete all extra bindings from this group to avoid duplicate command fields.

WITH keep(command_code, standard_path) AS (
    VALUES
        ('LST DEVICE_INFO', 'Device.DeviceInfo.3GPPSpecVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.AdditionalHardwareVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.AdditionalSoftwareVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.DataModelSpecVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.DnPrefix'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.FaultLogURL'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.FirstUseDate'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.HardwarePlatform'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.HardwareVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.HardwareVersion_OLD'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.MODULE_TYPE'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.MODULE_TYPE_OLD'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.Manufacturer'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.ManufacturerOUI'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.ModelName'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.ProductClass'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.ProvisioningCode'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.RunningStatus'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.SerialNumber'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.SiteId'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.SoftwareVersion'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.UpTime'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.UpTime_OLD'),
        ('LST DEVICE_INFO', 'Device.DeviceInfo.UserLabel'),
        ('LST DEVICE_INFO_SW_UPGRADE', 'Device.DeviceInfo.SwUpgrade.FailureCause'),
        ('LST DEVICE_INFO_SW_UPGRADE', 'Device.DeviceInfo.SwUpgrade.Stage'),
        ('LST DEVICE_INFO_SW_UPGRADE', 'Device.DeviceInfo.SwUpgrade.Status'),
        ('MOD DEVICE_INFO', 'Device.DeviceInfo.DnPrefix'),
        ('MOD DEVICE_INFO', 'Device.DeviceInfo.FaultLogURL'),
        ('MOD DEVICE_INFO', 'Device.DeviceInfo.RunningStatus'),
        ('MOD DEVICE_INFO', 'Device.DeviceInfo.SiteId'),
        ('MOD DEVICE_INFO', 'Device.DeviceInfo.UserLabel')
), affected_commands AS (
    SELECT c.id
    FROM public.mml_commands c
    WHERE c.command_code IN ('LST DEVICE_INFO', 'MOD DEVICE_INFO', 'LST DEVICE_INFO_SW_UPGRADE')
), deleted AS (
    DELETE FROM public.mml_command_sub_fields csf
    USING public.mml_commands c, public.standard_params sp
    WHERE csf.command_id = c.id
      AND csf.standard_path_id = sp.id
      AND c.id IN (SELECT id FROM affected_commands)
      AND NOT EXISTS (
          SELECT 1
          FROM keep k
          WHERE k.command_code = c.command_code
            AND k.standard_path = sp.standard_path
      )
    RETURNING csf.command_id
), touched AS (
    SELECT id AS command_id FROM affected_commands
    UNION
    SELECT command_id FROM deleted
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM touched t
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = t.command_id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = t.command_id;

-- Normalize SB/software-version command bindings:
-- - add the 5G v1.9.4-only PatchInfo read-only field to LST SOFTWARE_CTRL
-- - remove local extension fields not present in either referenced SB spec
WITH upsert_patch_info AS (
    INSERT INTO public.standard_params (
        standard_path,
        entry_type,
        access,
        data_type,
        change_applies,
        description
    )
    VALUES (
        'Device.SoftwareCtrl.PatchInfo',
        'parameter',
        'READ_ONLY',
        'STRING',
        'Immediate',
        '补丁信息,用于表示当前网元加载的补丁名称列表。'
    )
    ON CONFLICT (standard_path) DO UPDATE
    SET entry_type = EXCLUDED.entry_type,
        access = EXCLUDED.access,
        data_type = EXCLUDED.data_type,
        change_applies = EXCLUDED.change_applies,
        description = EXCLUDED.description,
        updated_at = now()
    RETURNING id
), list_software_command AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code = 'LST SOFTWARE_CTRL'
), software_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST SOFTWARE_CTRL', 'MOD SOFTWARE_CTRL')
), inserted_patch_info AS (
    INSERT INTO public.mml_command_sub_fields (
        command_id,
        mml_code,
        label_i18n,
        default_selected,
        is_required,
        sort_order,
        standard_path_id,
        access_type,
        is_supported
    )
    SELECT
        c.id,
        'PATCH_INFO',
        '{"zh": "补丁信息", "en": "PatchInfo"}'::jsonb,
        true,
        false,
        6,
        p.id,
        'RO',
        true
    FROM list_software_command c
    CROSS JOIN upsert_patch_info p
    ON CONFLICT (command_id, standard_path_id) DO UPDATE
    SET mml_code = EXCLUDED.mml_code,
        label_i18n = EXCLUDED.label_i18n,
        sort_order = EXCLUDED.sort_order,
        access_type = EXCLUDED.access_type,
        is_supported = EXCLUDED.is_supported,
        deprecated_at = NULL,
        updated_at = now()
    RETURNING command_id
), deleted_extra_software_fields AS (
    DELETE FROM public.mml_command_sub_fields csf
    USING public.mml_commands c, public.standard_params sp
    WHERE csf.command_id = c.id
      AND csf.standard_path_id = sp.id
      AND c.id IN (SELECT id FROM software_commands)
      AND sp.standard_path IN (
          'Device.SoftwareCtrl.AccCard1PpsDelay',
          'Device.SoftwareCtrl.N48N78SharedRfEnable'
      )
    RETURNING csf.command_id
), touched_software_commands AS (
    SELECT id AS command_id FROM software_commands
    UNION
    SELECT command_id FROM inserted_patch_info
    UNION
    SELECT command_id FROM deleted_extra_software_fields
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM touched_software_commands t
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = t.command_id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = t.command_id;

-- mml_command_sub_fields triggers may refresh target_paths during the statement
-- above. Run an independent final sync so tree_node_refs cannot retain stale
-- software-version paths.
WITH affected_software_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST SOFTWARE_CTRL', 'MOD SOFTWARE_CTRL')
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM affected_software_commands a
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = a.id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = a.id;

-- Normalize SC/management-server command bindings:
-- - both referenced specs define the same 19 Device.ManagementServer fields
-- - keep local operational extensions required by the product:
--   X_COM_tr069_port, sslStatus.startDate, sslStatus.endDate
-- - remove other local extension fields not present in either referenced SC spec
-- - refresh tree_node_refs, which may be empty/stale independently of target_paths
WITH management_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST MANAGEMENT_SERVER', 'MOD MANAGEMENT_SERVER')
), management_keep_extensions(command_code, standard_path, mml_code, label_i18n, sort_order, access_type) AS (
    VALUES
        (
            'LST MANAGEMENT_SERVER',
            'Device.ManagementServer.X_COM_tr069_port',
            'X_COM_TR069_PORT',
            '{"zh-CN": "X_COM_TR069_PORT", "en-US": "X COM tr069 port"}'::jsonb,
            10001,
            'RW'
        ),
        (
            'LST MANAGEMENT_SERVER',
            'Device.ManagementServer.sslStatus.endDate',
            'END_DATE',
            '{"zh-CN": "END_DATE", "en-US": "End Date"}'::jsonb,
            10004,
            'RO'
        ),
        (
            'LST MANAGEMENT_SERVER',
            'Device.ManagementServer.sslStatus.startDate',
            'START_DATE',
            '{"zh-CN": "START_DATE", "en-US": "Start Date"}'::jsonb,
            10005,
            'RO'
        ),
        (
            'MOD MANAGEMENT_SERVER',
            'Device.ManagementServer.X_COM_tr069_port',
            'X_COM_TR069_PORT',
            '{"zh-CN": "X_COM_TR069_PORT", "en-US": "X COM tr069 port"}'::jsonb,
            10001,
            'RW'
        ),
        (
            'MOD MANAGEMENT_SERVER',
            'Device.ManagementServer.sslStatus.endDate',
            'END_DATE',
            '{"zh-CN": "END_DATE", "en-US": "End Date"}'::jsonb,
            10004,
            'RO'
        ),
        (
            'MOD MANAGEMENT_SERVER',
            'Device.ManagementServer.sslStatus.startDate',
            'START_DATE',
            '{"zh-CN": "START_DATE", "en-US": "Start Date"}'::jsonb,
            10005,
            'RO'
        )
), inserted_management_keep_extensions AS (
    INSERT INTO public.mml_command_sub_fields (
        command_id,
        mml_code,
        label_i18n,
        default_selected,
        is_required,
        sort_order,
        standard_path_id,
        access_type,
        is_supported
    )
    SELECT
        c.id,
        e.mml_code,
        e.label_i18n,
        true,
        false,
        e.sort_order,
        sp.id,
        e.access_type,
        true
    FROM management_keep_extensions e
    JOIN public.mml_commands c ON c.command_code = e.command_code
    JOIN public.standard_params sp ON sp.standard_path = e.standard_path
    ON CONFLICT (command_id, standard_path_id) DO UPDATE
    SET mml_code = EXCLUDED.mml_code,
        label_i18n = EXCLUDED.label_i18n,
        sort_order = EXCLUDED.sort_order,
        access_type = EXCLUDED.access_type,
        is_supported = EXCLUDED.is_supported,
        deprecated_at = NULL,
        updated_at = now()
    RETURNING command_id
), deleted_extra_management_fields AS (
    DELETE FROM public.mml_command_sub_fields csf
    USING public.mml_commands c, public.standard_params sp
    WHERE csf.command_id = c.id
      AND csf.standard_path_id = sp.id
      AND c.id IN (SELECT id FROM management_commands)
      AND sp.standard_path IN (
          'Device.ManagementServer.tfcsManagerPrimsrc',
          'Device.ManagementServer.tfcsSyncState'
      )
    RETURNING csf.command_id
), touched_management_commands AS (
    SELECT id AS command_id FROM management_commands
    UNION
    SELECT command_id FROM inserted_management_keep_extensions
    UNION
    SELECT command_id FROM deleted_extra_management_fields
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM touched_management_commands t
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = t.command_id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = t.command_id;

-- mml_command_sub_fields triggers may refresh target_paths during the statement
-- above. Run an independent final sync so tree_node_refs cannot retain stale
-- management-server paths.
WITH affected_management_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST MANAGEMENT_SERVER', 'MOD MANAGEMENT_SERVER')
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM affected_management_commands a
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = a.id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = a.id;

-- Normalize SE/log-management command bindings:
-- - add the 5G v1.9.4-only LogLevel RW field to LST/MOD LOG_MGMT
-- - add the same path as a custom product-param-model mapping for every product
--   model that already has the 5 common LogMgmt paths; source=custom keeps the
--   supplement from being removed by XML param-model reloads
-- - refresh tree_node_refs, which was empty while target_paths had 5 entries
WITH upsert_log_level AS (
    INSERT INTO public.standard_params (
        standard_path,
        entry_type,
        access,
        data_type,
        change_applies,
        description
    )
    VALUES (
        'Device.LogMgmt.LogLevel',
        'parameter',
        'READ_WRITE',
        'STRING',
        'Immediate',
        '日志重要等级。Enumerate{ASSERT_LOG;ERROR;WARN;INFO;DEBUG;VERBOSE}'
    )
    ON CONFLICT (standard_path) DO UPDATE
    SET entry_type = EXCLUDED.entry_type,
        access = EXCLUDED.access,
        data_type = EXCLUDED.data_type,
        change_applies = EXCLUDED.change_applies,
        description = EXCLUDED.description,
        updated_at = now()
    RETURNING id
), log_mgmt_param_models AS (
    SELECT pm.param_model_id
    FROM public.param_mappings pm
    WHERE pm.is_active = true
      AND pm.is_supported = true
      AND pm.standard_path IN (
          'Device.LogMgmt.PeriodicUploadEnable',
          'Device.LogMgmt.URL',
          'Device.LogMgmt.Username',
          'Device.LogMgmt.Password',
          'Device.LogMgmt.PeriodicUploadInterval'
      )
    GROUP BY pm.param_model_id
    HAVING COUNT(DISTINCT pm.standard_path) = 5
), inserted_log_level_mappings AS (
    INSERT INTO public.param_mappings (
        param_model_id,
        standard_path,
        private_path,
        entry_type,
        access,
        data_type,
        change_applies,
        is_storable,
        is_active,
        is_supported,
        enum_values,
        source
    )
    SELECT
        pm.param_model_id,
        'Device.LogMgmt.LogLevel',
        'Device.LogMgmt.LogLevel',
        'parameter',
        'READ_WRITE',
        'STRING',
        'Immediate',
        true,
        true,
        true,
        'ASSERT_LOG,ERROR,WARN,INFO,DEBUG,VERBOSE',
        'custom'
    FROM log_mgmt_param_models pm
    ON CONFLICT (param_model_id, private_path) DO UPDATE
    SET standard_path = EXCLUDED.standard_path,
        entry_type = EXCLUDED.entry_type,
        access = EXCLUDED.access,
        data_type = EXCLUDED.data_type,
        change_applies = EXCLUDED.change_applies,
        is_storable = EXCLUDED.is_storable,
        is_active = EXCLUDED.is_active,
        is_supported = EXCLUDED.is_supported,
        enum_values = EXCLUDED.enum_values,
        source = EXCLUDED.source,
        updated_at = now()
    RETURNING param_model_id
), inserted_discovered_log_level_mappings AS (
    INSERT INTO public.discovered_param_mappings (
        product_id,
        software_version,
        standard_path,
        private_path,
        entry_type,
        access,
        data_type,
        change_applies,
        is_storable,
        is_active,
        is_supported,
        enum_values
    )
    SELECT
        d.product_id,
        d.software_version,
        'Device.LogMgmt.LogLevel',
        'Device.LogMgmt.LogLevel',
        'parameter',
        'READ_WRITE',
        'STRING',
        'Immediate',
        true,
        true,
        true,
        'ASSERT_LOG,ERROR,WARN,INFO,DEBUG,VERBOSE'
    FROM public.discovered_param_mappings d
    WHERE d.is_active = true
      AND d.is_supported = true
      AND d.standard_path IN (
          'Device.LogMgmt.PeriodicUploadEnable',
          'Device.LogMgmt.URL',
          'Device.LogMgmt.Username',
          'Device.LogMgmt.Password',
          'Device.LogMgmt.PeriodicUploadInterval'
      )
    GROUP BY d.product_id, d.software_version
    HAVING COUNT(DISTINCT d.standard_path) = 5
    ON CONFLICT (product_id, software_version, standard_path) DO UPDATE
    SET private_path = EXCLUDED.private_path,
        entry_type = EXCLUDED.entry_type,
        access = EXCLUDED.access,
        data_type = EXCLUDED.data_type,
        change_applies = EXCLUDED.change_applies,
        is_storable = EXCLUDED.is_storable,
        is_active = EXCLUDED.is_active,
        is_supported = EXCLUDED.is_supported,
        enum_values = EXCLUDED.enum_values,
        updated_at = now()
    RETURNING product_id
), log_mgmt_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST LOG_MGMT', 'MOD LOG_MGMT')
), inserted_log_level AS (
    INSERT INTO public.mml_command_sub_fields (
        command_id,
        mml_code,
        label_i18n,
        default_selected,
        is_required,
        sort_order,
        standard_path_id,
        access_type,
        is_supported
    )
    SELECT
        c.id,
        'LOG_LEVEL',
        '{"zh-CN": "日志重要等级", "en-US": "LogLevel"}'::jsonb,
        true,
        false,
        6,
        p.id,
        'RW',
        true
    FROM log_mgmt_commands c
    CROSS JOIN upsert_log_level p
    ON CONFLICT (command_id, standard_path_id) DO UPDATE
    SET mml_code = EXCLUDED.mml_code,
        label_i18n = EXCLUDED.label_i18n,
        sort_order = EXCLUDED.sort_order,
        access_type = EXCLUDED.access_type,
        is_supported = EXCLUDED.is_supported,
        deprecated_at = NULL,
        updated_at = now()
    RETURNING command_id
), touched_log_mgmt_commands AS (
    SELECT id AS command_id FROM log_mgmt_commands
    UNION
    SELECT command_id FROM inserted_log_level
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM touched_log_mgmt_commands t
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = t.command_id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = t.command_id;

-- mml_command_sub_fields triggers may refresh target_paths during the statement
-- above. Run an independent final sync so tree_node_refs cannot retain stale or
-- empty log-management paths.
WITH affected_log_mgmt_commands AS (
    SELECT id
    FROM public.mml_commands
    WHERE command_code IN ('LST LOG_MGMT', 'MOD LOG_MGMT')
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM affected_log_mgmt_commands a
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = a.id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = a.id;

-- Normalize SD/alarm-parameter command bindings:
-- - the MML page only keeps current-alarm and history-alarm queries
-- - remove overview/realtime/queued/supported-alarm commands and object mutations
WITH alarm_commands_to_delete(command_code) AS (
    VALUES
        ('LST FAULT_MGMT'),
        ('LST FAULT_MGMT_EXPEDITED_EVENT'),
        ('ADD FAULT_MGMT_EXPEDITED_EVENT'),
        ('RMV FAULT_MGMT_EXPEDITED_EVENT'),
        ('LST FAULT_MGMT_QUEUED_EVENT'),
        ('ADD FAULT_MGMT_QUEUED_EVENT'),
        ('RMV FAULT_MGMT_QUEUED_EVENT'),
        ('ADD SUPPORTED_ALARM'),
        ('LST FAULT_MGMT_SUPPORTED_ALARM'),
        ('MOD FAULT_MGMT_SUPPORTED_ALARM'),
        ('ADD FAULT_MGMT_SUPPORTED_ALARM'),
        ('RMV FAULT_MGMT_SUPPORTED_ALARM'),
        ('ADD FAULT_MGMT_CURRENT_ALARM'),
        ('RMV FAULT_MGMT_CURRENT_ALARM'),
        ('ADD FAULT_MGMT_HISTORY_EVENT'),
        ('RMV FAULT_MGMT_HISTORY_EVENT')
), deleted_alarm_commands AS (
    DELETE FROM public.mml_commands c
    USING alarm_commands_to_delete d
    WHERE c.command_code = d.command_code
    RETURNING c.id
), kept_alarm_commands AS (
    SELECT id AS command_id
    FROM public.mml_commands
    WHERE command_code IN (
        'LST FAULT_MGMT_CURRENT_ALARM',
        'LST FAULT_MGMT_HISTORY_EVENT'
    )
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
    updated_at = now()
FROM kept_alarm_commands k
LEFT JOIN LATERAL (
    SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
    FROM public.mml_command_sub_fields csf
    JOIN public.standard_params sp ON sp.id = csf.standard_path_id
    WHERE csf.command_id = k.command_id
      AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = k.command_id;

-- +goose Down
-- No downgrade path: this is a forward-only data cleanup. The removed bindings
-- were duplicates or non-device-info fields, and should not be recreated.
SELECT 1;
