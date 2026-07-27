-- +goose Up
-- MML 全局“基站网管参数管理-查询基站网关连接”补充 SSL 状态只读参数。
--
-- 这两个 path 同时存在于标准参数模型和 MML 命令树中：
--   Device.ManagementServer.sslStatus.endDate
--   Device.ManagementServer.sslStatus.startDate
--
-- 迁移按 path / command_code 定位，避免依赖不同环境中的 UUID；可重复执行。
-- 参数映射由 XML 字典加载器维护，本次同步修复历史 BLQ 行及其他已有模型行的
-- is_supported 状态，避免设备上下文过滤时把全局 MML 字段隐藏。

-- +goose StatementBegin
DO $$
DECLARE
    lst_command_id uuid;
    end_date_id uuid;
    start_date_id uuid;
BEGIN
    INSERT INTO public.standard_params (
        id, standard_path, entry_type, access, data_type, change_applies
    ) VALUES (
        gen_random_uuid(),
        'Device.ManagementServer.sslStatus.endDate',
        'parameter', 'READ_ONLY', 'STRING', 'Immediate'
    )
    ON CONFLICT (standard_path) DO NOTHING;

    INSERT INTO public.standard_params (
        id, standard_path, entry_type, access, data_type, change_applies
    ) VALUES (
        gen_random_uuid(),
        'Device.ManagementServer.sslStatus.startDate',
        'parameter', 'READ_ONLY', 'STRING', 'Immediate'
    )
    ON CONFLICT (standard_path) DO NOTHING;

    SELECT id
      INTO end_date_id
      FROM public.standard_params
     WHERE standard_path = 'Device.ManagementServer.sslStatus.endDate';

    SELECT id
      INTO start_date_id
      FROM public.standard_params
     WHERE standard_path = 'Device.ManagementServer.sslStatus.startDate';

    SELECT id
      INTO lst_command_id
      FROM public.mml_commands
     WHERE command_code = 'LST MANAGEMENT_SERVER'
       AND deprecated_at IS NULL
     ORDER BY created_at
     LIMIT 1;

    IF lst_command_id IS NULL THEN
        RAISE EXCEPTION 'MML command LST MANAGEMENT_SERVER is missing';
    END IF;

    INSERT INTO public.mml_command_sub_fields (
        id, command_id, mml_code, label_i18n,
        default_selected, is_required, sort_order,
        standard_path_id, access_type, is_supported
    ) VALUES (
        gen_random_uuid(),
        lst_command_id,
        'END_DATE',
        '{"zh-CN":"SSL状态结束时间","en-US":"End Date"}'::jsonb,
        true, false, 10006,
        end_date_id, 'RO', true
    )
    ON CONFLICT (command_id, standard_path_id) DO UPDATE
       SET mml_code = EXCLUDED.mml_code,
           label_i18n = EXCLUDED.label_i18n,
           default_selected = EXCLUDED.default_selected,
           is_required = EXCLUDED.is_required,
           sort_order = EXCLUDED.sort_order,
           access_type = EXCLUDED.access_type,
           is_supported = EXCLUDED.is_supported,
           deprecated_at = NULL,
           updated_at = NOW();

    INSERT INTO public.mml_command_sub_fields (
        id, command_id, mml_code, label_i18n,
        default_selected, is_required, sort_order,
        standard_path_id, access_type, is_supported
    ) VALUES (
        gen_random_uuid(),
        lst_command_id,
        'START_DATE',
        '{"zh-CN":"SSL状态开始时间","en-US":"Start Date"}'::jsonb,
        true, false, 10007,
        start_date_id, 'RO', true
    )
    ON CONFLICT (command_id, standard_path_id) DO UPDATE
       SET mml_code = EXCLUDED.mml_code,
           label_i18n = EXCLUDED.label_i18n,
           default_selected = EXCLUDED.default_selected,
           is_required = EXCLUDED.is_required,
           sort_order = EXCLUDED.sort_order,
           access_type = EXCLUDED.access_type,
           is_supported = EXCLUDED.is_supported,
           deprecated_at = NULL,
           updated_at = NOW();

    UPDATE public.param_mappings
       SET is_supported = true,
           updated_at = NOW()
     WHERE standard_path IN (
               'Device.ManagementServer.sslStatus.endDate',
               'Device.ManagementServer.sslStatus.startDate'
           )
       AND is_active = true;

    UPDATE public.mml_command_sub_fields AS csf
       SET deprecated_at = NOW(),
           updated_at = NOW()
      FROM public.mml_commands AS command,
           public.standard_params AS sp
     WHERE csf.command_id = command.id
       AND sp.id = csf.standard_path_id
       AND command.command_code = 'MOD MANAGEMENT_SERVER'
       AND command.deprecated_at IS NULL
       AND sp.standard_path IN (
               'Device.ManagementServer.sslStatus.endDate',
               'Device.ManagementServer.sslStatus.startDate'
           )
       AND csf.deprecated_at IS NULL;

    UPDATE public.mml_commands AS command
       SET target_paths = COALESCE((
               SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order)
                 FROM public.mml_command_sub_fields AS csf
                 JOIN public.standard_params AS sp
                   ON sp.id = csf.standard_path_id
                WHERE csf.command_id = command.id
                  AND csf.deprecated_at IS NULL
           ), '[]'::jsonb),
           tree_node_refs = COALESCE((
               SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order)
                 FROM public.mml_command_sub_fields AS csf
                 JOIN public.standard_params AS sp
                   ON sp.id = csf.standard_path_id
                WHERE csf.command_id = command.id
                  AND csf.deprecated_at IS NULL
           ), '[]'::jsonb),
           updated_at = NOW()
     WHERE command.command_code IN (
               'LST MANAGEMENT_SERVER',
               'MOD MANAGEMENT_SERVER'
           )
       AND command.deprecated_at IS NULL;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- 标准参数和 MML 子字段可能已被其他部署数据引用，不能在回滚时破坏性删除。
SELECT 1;
