-- +goose Up
-- 指标单位字典初始化(单一来源化):把单位数据以「字面 SQL」落入系统字典
-- (sys_dictionaries type='indicator_unit'),并 DROP 掉 indicator_unit 表。
--
-- 背景:原先单位由 indicator Loader 从指标 XML 收集 distinct unitId 写入 indicator_unit 表。
-- 现改为:单位以数据字典为唯一来源,不再依赖 XML。Loader 已移除写 indicator_unit 的逻辑
-- (internal/pm/indicator/loader.go);新增单位 = 在 system/data-dictionary「指标单位」字典里增明细。
--
-- 新环境部署:schema 建 indicator_unit 表 → seed/000001 灌历史行 → 本迁移用字面数据建字典
-- + DROP indicator_unit。字典数据**不依赖** indicator_unit 表内容(纯字面),故 XML/表为空也能初始化。
--
-- 必须放 seed/(在 seed/000001 之后);幂等:字典按 type 去重、明细按 (dict,value) 去重;DROP IF EXISTS。

-- +goose StatementBegin
DO $$
DECLARE
  dict_id bigint;
BEGIN
  SELECT id INTO dict_id FROM sys_dictionaries WHERE type = 'indicator_unit' AND deleted_at IS NULL;
  IF dict_id IS NULL THEN
    INSERT INTO sys_dictionaries (name, type, status, description, name_i18n, description_i18n)
    VALUES ('指标单位', 'indicator_unit', true, 'KPI 指标单位(数据字典维护,单一来源)',
            '{"zh-CN":"指标单位","en-US":"Indicator Unit"}'::jsonb, '{}'::jsonb)
    RETURNING id INTO dict_id;
  END IF;

  INSERT INTO sys_dictionary_details
    (label, label_i18n, value, extend, status, sort, sys_dictionary_id, level, origin, created_at, updated_at)
  SELECT v.cn, jsonb_build_object('zh-CN', v.cn, 'en-US', v.en), v.id, '', true, 0, dict_id, 0, 'manual', now(), now()
  FROM (VALUES
    ('%',            '百分比',    '%'),
    ('Byte',         '字节',      'Byte'),
    ('Byte/s',       '字节/秒',   'Byte/s'),
    ('Erl',          'Erl',       'Erl'),
    ('GByte',        '千兆字节',  'GByte'),
    ('Gbit',         '千兆位',    'Gbit'),
    ('KByte',        '千字节',    'KByte'),
    ('KByte/s',      '千字节/秒', 'KByte/s'),
    ('Kb/PRB',       'Kb/PRB',    'Kb/PRB'),
    ('Kbit',         '千位',      'Kbit'),
    ('Kbps',         '千位/秒',   'Kbps'),
    ('MByte',        '兆字节',    'MByte'),
    ('Mbps',         '兆位/秒',   'Mbps'),
    ('W',            '瓦特',      'W'),
    ('bit',          '位',        'bit'),
    ('char',         '字符串',    'char'),
    ('dBm',          'dBm',       'dBm'),
    ('mW',           '毫瓦',      'mW'),
    ('milliseconds', '毫秒',      'milliseconds'),
    ('ms',           '毫秒',      'ms'),
    ('no',           '无',        'no'),
    ('number',       '个',        'number'),
    ('ppm',          '百万分比',  'ppm'),
    ('pps',          '速率/秒',   'pps'),
    ('s',            '秒',        's'),
    ('seconds',      '秒',        'seconds'),
    ('time',         '次数',      'time')
  ) AS v(id, cn, en)
  WHERE NOT EXISTS (
    SELECT 1 FROM sys_dictionary_details d
    WHERE d.sys_dictionary_id = dict_id AND d.value = v.id AND d.deleted_at IS NULL
  );
END $$;
-- +goose StatementEnd

-- 单位不再由 Loader 写表,indicator_unit 已无读写方,删除之。
DROP TABLE IF EXISTS public.indicator_unit;

-- +goose Down
-- 回滚:重建 indicator_unit 空表(供旧版 Loader),删除「指标单位」字典及明细(单向迁移)。
CREATE TABLE IF NOT EXISTS public.indicator_unit (
    id character varying(50) NOT NULL,
    en_name character varying(100),
    cn_name character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pk_indicator_unit PRIMARY KEY (id)
);

-- +goose StatementBegin
DO $$
DECLARE
  dict_id bigint;
BEGIN
  SELECT id INTO dict_id FROM sys_dictionaries WHERE type = 'indicator_unit';
  IF dict_id IS NOT NULL THEN
    DELETE FROM sys_dictionary_details WHERE sys_dictionary_id = dict_id;
    DELETE FROM sys_dictionaries WHERE id = dict_id;
  END IF;
END $$;
-- +goose StatementEnd
