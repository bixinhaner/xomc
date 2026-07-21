-- +goose Up
-- Normalize TD-LTE SF / 小区服务参数管理 catalog bindings.

UPDATE public.mml_commands
SET command_code = 'LST FAP_SERVICE',
    logical_name_i18n = '{"en-US":"FAP Service","zh-CN":"FAP 载波基本配置"}'::jsonb,
    updated_at = now()
WHERE command_code = 'LST ';

UPDATE public.mml_commands
SET command_code = 'MOD FAP_SERVICE',
    logical_name_i18n = '{"en-US":"FAP Service","zh-CN":"FAP 载波基本配置"}'::jsonb,
    updated_at = now()
WHERE command_code = 'MOD ';

WITH keep(command_code, standard_path) AS (
  VALUES
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310'),
    ('LST FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310'),
    ('MOD FAP_SERVICE', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311'),
    ('LST PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID'),
    ('LST PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse'),
    ('LST PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable'),
    ('LST PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary'),
    ('MOD PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID'),
    ('MOD PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse'),
    ('MOD PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable'),
    ('MOD PLMN_LIST', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary')
),
affected AS (
  SELECT id, command_code
  FROM public.mml_commands
  WHERE command_code IN ('LST FAP_SERVICE', 'MOD FAP_SERVICE', 'LST PLMN_LIST', 'MOD PLMN_LIST')
)
DELETE FROM public.mml_command_sub_fields csf
USING affected a, public.standard_params sp
WHERE csf.command_id = a.id
  AND csf.standard_path_id = sp.id
  AND NOT EXISTS (
    SELECT 1
    FROM keep k
    WHERE k.command_code = a.command_code
      AND k.standard_path = sp.standard_path
  );

WITH add_back(command_code, mml_code, standard_path, sort_order) AS (
  VALUES
    ('LST PLMN_LIST', 'ENABLE', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable', 3),
    ('LST PLMN_LIST', 'IS_PRIMARY', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary', 4),
    ('MOD PLMN_LIST', 'ENABLE', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable', 3),
    ('MOD PLMN_LIST', 'IS_PRIMARY', 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary', 4)
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
  a.mml_code,
  jsonb_build_object('en-US', initcap(replace(lower(a.mml_code), '_', ' ')), 'zh-CN', a.mml_code),
  true,
  c.operation_type IN ('MOD', 'ADD'),
  a.sort_order,
  sp.id,
  CASE WHEN sp.access = 'READ_WRITE' THEN 'RW' ELSE 'RO' END,
  true
FROM add_back a
JOIN public.mml_commands c ON c.command_code = a.command_code
JOIN public.standard_params sp ON sp.standard_path = a.standard_path
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
    'LST FAP_SERVICE',
    'MOD FAP_SERVICE',
    'LST PLMN_LIST',
    'MOD PLMN_LIST',
    'LST PDCP_INIT_PARAM',
    'MOD PDCP_INIT_PARAM'
  )
)
UPDATE public.mml_commands c
SET target_paths = COALESCE(paths.paths, '[]'::jsonb),
    tree_node_refs = COALESCE(paths.paths, '[]'::jsonb),
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

UPDATE public.mml_commands
SET target_paths = jsonb_build_array(target_object),
    tree_node_refs = jsonb_build_array(target_object),
    updated_at = now()
WHERE command_code IN ('ADD PLMN_LIST', 'RMV PLMN_LIST', 'ADD PDCP_INIT_PARAM', 'RMV PDCP_INIT_PARAM')
  AND target_object IS NOT NULL
  AND target_object <> '';

-- +goose Down
-- Intentionally left as a no-op. Restoring old SF bindings would reintroduce
-- hundreds of non-standard/private paths into the normalized catalog.
