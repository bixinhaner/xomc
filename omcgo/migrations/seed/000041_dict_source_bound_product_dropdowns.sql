-- +goose Up
-- #241 / T-0182：新增产品「参数模型名称 / KPI指标名称 / 告警名称」三个下拉改为数据字典数据源绑定。
-- 建 3 个 source-bound 字典(只建字典壳,不灌明细);明细由同步引擎运行期写入:
--   - worker daily cron SyncSourceBoundAll(DefaultDictSourceDailyCron,默认每日 02:00 BJT)
--     + 启动期 30s catch-up(cmd/worker/main.go),首次部署即可填充;
--   - 三库导入 XML 成功后按 source_table 自动刷新(internal/admin DictionaryService.RefreshSourceBoundByTable)。
-- 绑定来源(label=value 同字段):
--   param_model_name  → param_models.name
--   alarm_ne_type     → alarm_definitions.ne_type
--   kpi_platform_enb  → rela_platform_indicator_formula_enb.platform_name(本期仅 enb,gnb/gsm 留后续)
--     注:平台名权威来源是公式表的 platform_name 列,perf_indicators_* 本身不存平台维度
--     (见 internal/product/pg_repository.go:89-90),与产品下拉现有口径一致。
-- 白名单:前两项早已在 internal/admin/dictsource/sources.yaml;rela_platform_indicator_formula_enb.platform_name 由同 PR 补入。
-- 幂等:WHERE NOT EXISTS 按 type(deleted_at IS NULL)去重,对全新库可重复前向应用(镜像 seed/000007 明细插入式)。

INSERT INTO public.sys_dictionaries
  (name, type, status, description, name_i18n, description_i18n,
   source_table, source_label_field, source_value_field)
SELECT v.cn, v.dtype, true, v.descr,
       jsonb_build_object('zh-CN', v.cn, 'en-US', v.en), '{}'::jsonb,
       v.src_table, v.src_field, v.src_field
FROM (VALUES
  ('param_model_name',  '参数模型名称',       'Param Model Name',
   '新增产品「参数模型名称」下拉源(param_models.name,T-0182 #241)',
   'param_models',        'name'),
  ('alarm_ne_type',     '告警名称(网元类型)', 'Alarm NE Type',
   '新增产品「告警名称」下拉源(alarm_definitions.ne_type,T-0182 #241)',
   'alarm_definitions',   'ne_type'),
  ('kpi_platform_enb',  'KPI平台(eNB)',       'KPI Platform (eNB)',
   '新增产品「KPI指标名称」下拉源(rela_platform_indicator_formula_enb.platform_name,T-0182 #241)',
   'rela_platform_indicator_formula_enb', 'platform_name')
) AS v(dtype, cn, en, descr, src_table, src_field)
WHERE NOT EXISTS (
  SELECT 1 FROM public.sys_dictionaries d
  WHERE d.type = v.dtype AND d.deleted_at IS NULL
);

-- +goose Down
-- 删除 3 个绑定字典及其 auto 同步明细(先 details 后主表,避免 FK 残留)。
DELETE FROM public.sys_dictionary_details
WHERE sys_dictionary_id IN (
  SELECT id FROM public.sys_dictionaries
  WHERE type IN ('param_model_name', 'alarm_ne_type', 'kpi_platform_enb')
);

DELETE FROM public.sys_dictionaries
WHERE type IN ('param_model_name', 'alarm_ne_type', 'kpi_platform_enb');
