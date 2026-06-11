-- +goose Up
-- issue #67 §4：指标 data_type 受控码字典 + 存量脏值归一化。
--
-- 背景：data/indicator-library/*.xml 的 dataType 是中英混杂枚举（整数/实数/浮点数
-- 与 Integer/number/整数n）。代码侧 Loader 已改为落库受控码（int/real/float，见
-- internal/pm/indicator/data_type.go::normalizeDataType）；本迁移：
--   1) 建 sys_dictionaries(type='indicator_data_type') + zh-CN/en-US label_i18n（镜像 seed/000007）；
--   2) 把存量 perf_indicators_{enb,gnb,gsm}.data_type 的旧枚举一次性归一化为受控码。
--
-- 幂等：字典按 type 去重、明细按 (dict,value) 去重；归一化 UPDATE 命中即收敛，可重复执行。

-- +goose StatementBegin
DO $$
DECLARE
  dict_id bigint;
BEGIN
  SELECT id INTO dict_id FROM sys_dictionaries WHERE type = 'indicator_data_type' AND deleted_at IS NULL;
  IF dict_id IS NULL THEN
    INSERT INTO sys_dictionaries (name, type, status, description, name_i18n, description_i18n)
    VALUES ('指标数据类型', 'indicator_data_type', true, 'KPI 指标数据类型受控码（int/real/float），单一来源',
            '{"zh-CN":"指标数据类型","en-US":"Indicator Data Type"}'::jsonb, '{}'::jsonb)
    RETURNING id INTO dict_id;
  END IF;

  INSERT INTO sys_dictionary_details
    (label, label_i18n, value, extend, status, sort, sys_dictionary_id, level, origin, created_at, updated_at)
  SELECT v.cn, jsonb_build_object('zh-CN', v.cn, 'en-US', v.en), v.code, '', true, v.sort, dict_id, 0, 'manual', now(), now()
  FROM (VALUES
    ('int',   '整数',   'Integer', 0),
    ('real',  '实数',   'Real',    1),
    ('float', '浮点数', 'Float',   2)
  ) AS v(code, cn, en, sort)
  WHERE NOT EXISTS (
    SELECT 1 FROM sys_dictionary_details d
    WHERE d.sys_dictionary_id = dict_id AND d.value = v.code AND d.deleted_at IS NULL
  );
END $$;
-- +goose StatementEnd

-- 存量数据归一化：旧枚举（中英混杂）→ 受控码。三制式表同款映射。
-- +goose StatementBegin
DO $$
DECLARE
  tbl text;
BEGIN
  FOREACH tbl IN ARRAY ARRAY['perf_indicators_enb', 'perf_indicators_gnb', 'perf_indicators_gsm'] LOOP
    EXECUTE format($f$
      UPDATE %I SET data_type = CASE
        WHEN lower(data_type) IN ('整数', '整数n', 'integer', 'int')    THEN 'int'
        WHEN lower(data_type) IN ('实数', 'real')                       THEN 'real'
        WHEN lower(data_type) IN ('浮点数', 'number', 'float', 'double') THEN 'float'
        ELSE data_type
      END
      WHERE data_type IS NOT NULL
        AND data_type NOT IN ('int', 'real', 'float')
    $f$, tbl);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- 回滚：删字典及明细（受控码→旧枚举无法精确还原，data_type 保持受控码，单向数据迁移）。
-- +goose StatementBegin
DO $$
DECLARE
  dict_id bigint;
BEGIN
  SELECT id INTO dict_id FROM sys_dictionaries WHERE type = 'indicator_data_type';
  IF dict_id IS NOT NULL THEN
    DELETE FROM sys_dictionary_details WHERE sys_dictionary_id = dict_id;
    DELETE FROM sys_dictionaries WHERE id = dict_id;
  END IF;
END $$;
-- +goose StatementEnd
