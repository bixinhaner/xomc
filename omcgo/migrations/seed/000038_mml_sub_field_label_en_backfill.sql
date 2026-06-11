-- +goose Up
-- issue #67 §2：mml_command_sub_fields.label_i18n 的 'en' 键此前由 catalog 导入
-- (internal/mml/specparser/sql_gen.go) 直接灌中文 ParamName（约 619 行 en=中文），
-- 导致命令执行表单参数名在英文 locale 下仍显示中文。
--
-- 本迁移把「en 缺失或仍含中文（CJK）」的 sub-field 回填为 ASCII 英文标签：
-- 取关联 standard_params.standard_path 的叶子段（如 Device.DeviceInfo.SoftwareVersion
-- → SoftwareVersion）作为可读英文名；standard_path 为空时退回 mml_code（ASCII 助记符）。
-- 该「无英文源 → ASCII/音译兜底」策略与 seed/000002_mml_i18n_en.sql:121-128 处理命令叶子
-- （replace(logical_code,'_',' )）一致：英文形态优先于中文残留。
--
-- 仍写**短键 'en'**（与当前数据形态一致）；随后 seed/000039 统一把短键升级为 'en-US'/'zh-CN'。
-- 幂等：只更新 en 缺失或含中文的行；多次执行收敛到同一结果。
-- 注意：本迁移须在 seed/000039（short→long）之前执行，故用短键。

-- +goose StatementBegin
UPDATE mml_command_sub_fields csf
SET label_i18n = jsonb_set(
        COALESCE(csf.label_i18n, '{}'::jsonb),
        '{en}',
        to_jsonb(
            COALESCE(
                NULLIF(regexp_replace(sp.standard_path, '^.*\.', ''), ''),  -- standard_path 叶子段
                csf.mml_code                                                -- 退回 mml_code
            )
        )
    )
FROM standard_params sp
WHERE sp.id = csf.standard_path_id
  AND (
        (csf.label_i18n->>'en') IS NULL
        OR (csf.label_i18n->>'en') = ''
        OR (csf.label_i18n->>'en') ~ '[一-鿿]'  -- en 仍含中文（CJK 统一表意文字区）
      );
-- +goose StatementEnd

-- +goose Down
-- 单向数据修复，无法精确还原 catalog 导入时的中文 en；Down 不回退（保持英文标签）。
-- 若确需回到导入态，重跑 catalog import（sql_gen 生成的 seed）即可覆盖。
SELECT 1;
