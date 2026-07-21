-- +goose Up
-- Add CMCC 5G NR SF catalog as a sibling top-level group with second-level groups.

WITH upsert_params(standard_path, access, data_type, min_value, max_value, description) AS (
  VALUES
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfName', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, 'AMFName'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID', 'READ_WRITE', 'STRING', NULL::bigint, NULL::bigint, 'AMFPLMN标识'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfRegionID', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, 'AMFRegionID'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfSetID', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, 'AMFSetID'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfPointer', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, 'AMFPointer'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.RelativeAmfCapacity', 'READ_ONLY', 'U_INT', 0::bigint, 255::bigint, 'RelativeAMFCapacity'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1', 'READ_WRITE', 'STRING', NULL::bigint, NULL::bigint, 'AmfIP1'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2', 'READ_WRITE', 'STRING', NULL::bigint, NULL::bigint, 'AmfIP2'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.TAC', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, '跟踪区编码'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.BroadcastPLMNs.{i}.PLMNID', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, 'PLMNID'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID', 'READ_WRITE', 'U_INT', NULL::bigint, NULL::bigint, 'gNBID'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress', 'READ_WRITE', 'STRING', NULL::bigint, NULL::bigint, '对端IP'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.SubnetMask', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, '子网掩码'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.LocalAddress', 'READ_ONLY', 'STRING', NULL::bigint, NULL::bigint, '本端地址'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength', 'READ_WRITE', 'U_INT', 22::bigint, 32::bigint, 'gNB标识长度'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId', 'READ_WRITE', 'U_INT', 0::bigint, 4294967295::bigint, 'gNB标识'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName', 'READ_WRITE', 'STRING', NULL::bigint, NULL::bigint, 'gNB名称'),
    ('Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState', 'READ_WRITE', 'U_INT', 1::bigint, 3::bigint, '基站的管理状态')
)
INSERT INTO public.standard_params (
  standard_path,
  entry_type,
  access,
  data_type,
  change_applies,
  min_value,
  max_value,
  description
)
SELECT
  standard_path,
  'parameter',
  access,
  data_type,
  'Immediate',
  min_value,
  max_value,
  description
FROM upsert_params
ON CONFLICT (standard_path) DO UPDATE
SET access = EXCLUDED.access,
    data_type = EXCLUDED.data_type,
    change_applies = EXCLUDED.change_applies,
    min_value = EXCLUDED.min_value,
    max_value = EXCLUDED.max_value,
    description = EXCLUDED.description,
    updated_at = now();

WITH groups(group_code, group_name_zh, group_name_en, path_text, display_order, object_path_template, instance_arity, instance_levels) AS (
  VALUES
    ('chapter:SF_NR', '基站配置参数管理', 'Base Station Configuration Parameters', 'chapter_SF_NR', 6, NULL::varchar, 0::smallint, ARRAY[]::text[])
)
INSERT INTO public.mml_command_groups (
  group_code,
  group_name_zh,
  group_name_en,
  name_i18n,
  path,
  param_version,
  display_order,
  source,
  catalog_protected,
  object_path_template,
  chapter_code,
  instance_arity,
  instance_levels
)
SELECT
  group_code,
  group_name_zh,
  group_name_en,
  jsonb_build_object('zh-CN', group_name_zh, 'en-US', group_name_en),
  path_text::ltree,
  'cmcc-td-lte-v2.3',
  display_order,
  'standard',
  true,
  object_path_template,
  'SF_NR',
  instance_arity,
  instance_levels
FROM groups
ON CONFLICT (param_version, group_code) DO UPDATE
SET group_name_zh = EXCLUDED.group_name_zh,
    group_name_en = EXCLUDED.group_name_en,
    name_i18n = EXCLUDED.name_i18n,
    path = EXCLUDED.path,
    param_version = EXCLUDED.param_version,
    display_order = EXCLUDED.display_order,
    source = EXCLUDED.source,
    catalog_protected = EXCLUDED.catalog_protected,
    object_path_template = EXCLUDED.object_path_template,
    chapter_code = EXCLUDED.chapter_code,
    instance_arity = EXCLUDED.instance_arity,
    instance_levels = EXCLUDED.instance_levels,
    deprecated_at = NULL,
    deleted_at = NULL,
    updated_at = now();

WITH cmd_defs(command_code, command_name, logical_name, operation_type, rpc_method, group_code, target_object) AS (
  VALUES
    ('LST NR_AMF_POOL_CONFIG_PARAM', '查询 AMF地址', 'AMF地址', 'LST', 'GetParameterValues', 'chapter:SF_NR', NULL::varchar),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', '修改 AMF地址', 'AMF地址', 'MOD', 'SetParameterValues', 'chapter:SF_NR', NULL::varchar),
    ('ADD NR_AMF_POOL_CONFIG_PARAM', '添加 AMF地址', 'AMF地址', 'ADD', 'AddObject', 'chapter:SF_NR', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.'),
    ('RMV NR_AMF_POOL_CONFIG_PARAM', '删除 AMF地址', 'AMF地址', 'RMV', 'DeleteObject', 'chapter:SF_NR', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.'),
    ('LST NR_XN_IP_ADDR_MAP_INFO', '查询 Xn链接信息', 'Xn链接信息', 'LST', 'GetParameterValues', 'chapter:SF_NR', NULL::varchar),
    ('MOD NR_XN_IP_ADDR_MAP_INFO', '修改 Xn链接信息', 'Xn链接信息', 'MOD', 'SetParameterValues', 'chapter:SF_NR', NULL::varchar),
    ('ADD NR_XN_IP_ADDR_MAP_INFO', '添加 Xn链接信息', 'Xn链接信息', 'ADD', 'AddObject', 'chapter:SF_NR', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.'),
    ('RMV NR_XN_IP_ADDR_MAP_INFO', '删除 Xn链接信息', 'Xn链接信息', 'RMV', 'DeleteObject', 'chapter:SF_NR', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.'),
    ('LST NR_RAN_COMMON', '查询 基站配置参数管理', '基站配置参数管理', 'LST', 'GetParameterValues', 'chapter:SF_NR', NULL::varchar),
    ('MOD NR_RAN_COMMON', '修改 基站配置参数管理', '基站配置参数管理', 'MOD', 'SetParameterValues', 'chapter:SF_NR', NULL::varchar)
)
INSERT INTO public.mml_commands (
  command_name,
  command_code,
  category,
  description,
  rpc_method,
  operation_type,
  target_paths,
  target_object,
  tree_node_refs,
  group_id,
  command_name_i18n,
  logical_name_i18n,
  source,
  catalog_protected,
  help_doc
)
SELECT
  c.command_name,
  c.command_code,
  'FAPService',
  c.command_name,
  c.rpc_method,
  c.operation_type,
  CASE WHEN c.target_object IS NULL THEN '[]'::jsonb ELSE jsonb_build_array(c.target_object) END,
  c.target_object,
  CASE WHEN c.target_object IS NULL THEN '[]'::jsonb ELSE jsonb_build_array(c.target_object) END,
  g.id,
  jsonb_build_object('en-US', c.command_code, 'zh-CN', c.command_name),
  jsonb_build_object('en-US', c.command_code, 'zh-CN', c.logical_name),
  'standard',
  true,
  ''
FROM cmd_defs c
JOIN public.mml_command_groups g ON g.group_code = c.group_code
ON CONFLICT (command_code) DO UPDATE
SET command_name = EXCLUDED.command_name,
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    rpc_method = EXCLUDED.rpc_method,
    operation_type = EXCLUDED.operation_type,
    target_object = EXCLUDED.target_object,
    group_id = EXCLUDED.group_id,
    command_name_i18n = EXCLUDED.command_name_i18n,
    logical_name_i18n = EXCLUDED.logical_name_i18n,
    source = EXCLUDED.source,
    catalog_protected = EXCLUDED.catalog_protected,
    deprecated_at = NULL,
    updated_at = now();

DELETE FROM public.mml_command_groups
WHERE group_code IN ('NR_SF_AMF_ADDRESS', 'NR_SF_XN_LINK_INFO', 'NR_SF_GNB_PARAM');

WITH wanted(command_code, mml_code, standard_path, sort_order) AS (
  VALUES
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfName', 1),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID', 2),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_REGION_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfRegionID', 3),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_SET_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfSetID', 4),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_POINTER', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfPointer', 5),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'RELATIVE_AMF_CAPACITY', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.RelativeAmfCapacity', 6),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP1', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1', 7),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP2', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2', 8),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID', 2),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP1', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1', 7),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP2', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2', 8),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'TAC', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.TAC', 1),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.BroadcastPLMNs.{i}.PLMNID', 2),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID', 3),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'REMOTE_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress', 4),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'SUBNET_MASK', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.SubnetMask', 5),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'LOCAL_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.LocalAddress', 6),
    ('MOD NR_XN_IP_ADDR_MAP_INFO', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID', 3),
    ('MOD NR_XN_IP_ADDR_MAP_INFO', 'REMOTE_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress', 4),
    ('LST NR_RAN_COMMON', 'GNB_ID_LENGTH', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength', 1),
    ('LST NR_RAN_COMMON', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId', 2),
    ('LST NR_RAN_COMMON', 'GNB_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName', 3),
    ('LST NR_RAN_COMMON', 'ADMIN_STATE', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState', 4),
    ('MOD NR_RAN_COMMON', 'GNB_ID_LENGTH', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength', 1),
    ('MOD NR_RAN_COMMON', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId', 2),
    ('MOD NR_RAN_COMMON', 'GNB_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName', 3),
    ('MOD NR_RAN_COMMON', 'ADMIN_STATE', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState', 4)
),
affected AS (
  SELECT id
  FROM public.mml_commands
  WHERE command_code IN (
    'LST NR_AMF_POOL_CONFIG_PARAM',
    'MOD NR_AMF_POOL_CONFIG_PARAM',
    'ADD NR_AMF_POOL_CONFIG_PARAM',
    'RMV NR_AMF_POOL_CONFIG_PARAM',
    'LST NR_XN_IP_ADDR_MAP_INFO',
    'MOD NR_XN_IP_ADDR_MAP_INFO',
    'ADD NR_XN_IP_ADDR_MAP_INFO',
    'RMV NR_XN_IP_ADDR_MAP_INFO',
    'LST NR_RAN_COMMON',
    'MOD NR_RAN_COMMON'
  )
)
DELETE FROM public.mml_command_sub_fields csf
USING affected a
WHERE csf.command_id = a.id
  AND NOT EXISTS (
    SELECT 1
    FROM wanted w
    JOIN public.standard_params sp ON sp.standard_path = w.standard_path
    WHERE w.command_code = (SELECT command_code FROM public.mml_commands WHERE id = a.id)
      AND sp.id = csf.standard_path_id
  );

WITH wanted(command_code, mml_code, standard_path, sort_order) AS (
  VALUES
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfName', 1),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID', 2),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_REGION_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfRegionID', 3),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_SET_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfSetID', 4),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_POINTER', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.GUAMI.{i}.AmfPointer', 5),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'RELATIVE_AMF_CAPACITY', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.RelativeAmfCapacity', 6),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP1', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1', 7),
    ('LST NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP2', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2', 8),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.PLMNID', 2),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP1', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP1', 7),
    ('MOD NR_AMF_POOL_CONFIG_PARAM', 'AMF_IP2', 'Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}.AmfIP2', 8),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'TAC', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.TAC', 1),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'PLMN_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.TAISupportList.{i}.BroadcastPLMNs.{i}.PLMNID', 2),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID', 3),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'REMOTE_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress', 4),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'SUBNET_MASK', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.SubnetMask', 5),
    ('LST NR_XN_IP_ADDR_MAP_INFO', 'LOCAL_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.LocalAddress', 6),
    ('MOD NR_XN_IP_ADDR_MAP_INFO', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.gNBID', 3),
    ('MOD NR_XN_IP_ADDR_MAP_INFO', 'REMOTE_ADDRESS', 'Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}.RemoteAddress', 4),
    ('LST NR_RAN_COMMON', 'GNB_ID_LENGTH', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength', 1),
    ('LST NR_RAN_COMMON', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId', 2),
    ('LST NR_RAN_COMMON', 'GNB_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName', 3),
    ('LST NR_RAN_COMMON', 'ADMIN_STATE', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState', 4),
    ('MOD NR_RAN_COMMON', 'GNB_ID_LENGTH', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBIdLength', 1),
    ('MOD NR_RAN_COMMON', 'GNB_ID', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBId', 2),
    ('MOD NR_RAN_COMMON', 'GNB_NAME', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.gNBName', 3),
    ('MOD NR_RAN_COMMON', 'ADMIN_STATE', 'Device.Services.FAPService.{i}.FAPControl.NR.RAN.Common.AdminState', 4)
)
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
  w.mml_code,
  jsonb_build_object('en-US', initcap(replace(lower(w.mml_code), '_', ' ')), 'zh-CN', w.mml_code),
  true,
  c.operation_type = 'MOD',
  w.sort_order,
  sp.id,
  CASE WHEN sp.access = 'READ_WRITE' THEN 'RW' ELSE 'RO' END,
  true
FROM wanted w
JOIN public.mml_commands c ON c.command_code = w.command_code
JOIN public.standard_params sp ON sp.standard_path = w.standard_path
ON CONFLICT (command_id, standard_path_id) DO UPDATE
SET deprecated_at = NULL,
    mml_code = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    default_selected = EXCLUDED.default_selected,
    is_required = EXCLUDED.is_required,
    sort_order = EXCLUDED.sort_order,
    access_type = EXCLUDED.access_type,
    is_supported = EXCLUDED.is_supported,
    updated_at = now();

WITH affected AS (
  SELECT id
  FROM public.mml_commands
  WHERE command_code IN (
    'LST NR_AMF_POOL_CONFIG_PARAM',
    'MOD NR_AMF_POOL_CONFIG_PARAM',
    'ADD NR_AMF_POOL_CONFIG_PARAM',
    'RMV NR_AMF_POOL_CONFIG_PARAM',
    'LST NR_XN_IP_ADDR_MAP_INFO',
    'MOD NR_XN_IP_ADDR_MAP_INFO',
    'ADD NR_XN_IP_ADDR_MAP_INFO',
    'RMV NR_XN_IP_ADDR_MAP_INFO',
    'LST NR_RAN_COMMON',
    'MOD NR_RAN_COMMON'
  )
)
UPDATE public.mml_commands c
SET target_paths = CASE
        WHEN c.operation_type IN ('ADD', 'RMV') AND COALESCE(c.target_object, '') <> ''
          THEN jsonb_build_array(c.target_object)
        ELSE COALESCE(paths.paths, '[]'::jsonb)
    END,
    tree_node_refs = CASE
        WHEN c.operation_type IN ('ADD', 'RMV') AND COALESCE(c.target_object, '') <> ''
          THEN jsonb_build_array(c.target_object)
        ELSE COALESCE(paths.paths, '[]'::jsonb)
    END,
    updated_at = now()
FROM affected a
LEFT JOIN LATERAL (
  SELECT jsonb_agg(sp.standard_path ORDER BY csf.sort_order, sp.standard_path) AS paths
  FROM public.mml_command_sub_fields csf
  JOIN public.standard_params sp ON sp.id = csf.standard_path_id
  WHERE csf.command_id = a.id
    AND csf.deprecated_at IS NULL
) paths ON true
WHERE c.id = a.id;

-- +goose Down
-- Keep as no-op; this seed migration only adds NR SF catalog coverage.
