-- +goose Up
-- issue #67 §5：统一 MML i18n JSONB 键为长码 zh-CN / en-US。
--
-- 历史成因：catalog 导入（internal/mml/specparser/sql_gen.go）与 seed/000002 写的是短键
-- {"en":..,"zh":..}，而 API/前端约定长码 zh-CN/en-US。group_tree_repository.go::pickI18n
-- 曾用短/长兼容分支抹平差异。本迁移把存量短键升级为长码，配合代码侧删除该兼容分支，
-- 使 i18n 键全仓单一形态（长码）。
--
-- 覆盖列（全部 MML i18n JSONB 列）：
--   mml_command_groups.name_i18n
--   mml_commands.command_name_i18n / confirm_msg_i18n / logical_name_i18n
--   mml_command_sub_fields.label_i18n
--   mml_command_sub_field_overrides.label_i18n_override
--
-- 规则：'en' → 'en-US'，'zh' → 'zh-CN'。若长码已存在则保留长码（长码优先），仅在长码缺失
-- 时从短键搬迁；最后删除短键。幂等：无短键时为 no-op，可重复执行。

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION pg_temp.mml_i18n_short_to_long(src jsonb)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
  out jsonb := COALESCE(src, '{}'::jsonb);
BEGIN
  IF out IS NULL OR jsonb_typeof(out) <> 'object' THEN
    RETURN src;
  END IF;
  -- en → en-US（长码缺失时才搬迁），随后删短键
  IF out ? 'en' THEN
    IF NOT (out ? 'en-US') THEN
      out := jsonb_set(out, '{en-US}', out->'en');
    END IF;
    out := out - 'en';
  END IF;
  -- zh → zh-CN
  IF out ? 'zh' THEN
    IF NOT (out ? 'zh-CN') THEN
      out := jsonb_set(out, '{zh-CN}', out->'zh');
    END IF;
    out := out - 'zh';
  END IF;
  RETURN out;
END;
$$;
-- +goose StatementEnd

UPDATE mml_command_groups
SET name_i18n = pg_temp.mml_i18n_short_to_long(name_i18n)
WHERE name_i18n ?| array['en', 'zh'];

UPDATE mml_commands
SET command_name_i18n = pg_temp.mml_i18n_short_to_long(command_name_i18n),
    confirm_msg_i18n   = pg_temp.mml_i18n_short_to_long(confirm_msg_i18n),
    logical_name_i18n  = pg_temp.mml_i18n_short_to_long(logical_name_i18n)
WHERE command_name_i18n ?| array['en', 'zh']
   OR confirm_msg_i18n   ?| array['en', 'zh']
   OR logical_name_i18n  ?| array['en', 'zh'];

UPDATE mml_command_sub_fields
SET label_i18n = pg_temp.mml_i18n_short_to_long(label_i18n)
WHERE label_i18n ?| array['en', 'zh'];

UPDATE mml_command_sub_field_overrides
SET label_i18n_override = pg_temp.mml_i18n_short_to_long(label_i18n_override)
WHERE label_i18n_override ?| array['en', 'zh'];

-- +goose Down
-- 回滚：长码降级回短键（en-US → en，zh-CN → zh）。同样长码缺失时不动；
-- 仅在需要恢复旧兼容分支语义时使用。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION pg_temp.mml_i18n_long_to_short(src jsonb)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
  out jsonb := COALESCE(src, '{}'::jsonb);
BEGIN
  IF out IS NULL OR jsonb_typeof(out) <> 'object' THEN
    RETURN src;
  END IF;
  IF out ? 'en-US' THEN
    IF NOT (out ? 'en') THEN
      out := jsonb_set(out, '{en}', out->'en-US');
    END IF;
    out := out - 'en-US';
  END IF;
  IF out ? 'zh-CN' THEN
    IF NOT (out ? 'zh') THEN
      out := jsonb_set(out, '{zh}', out->'zh-CN');
    END IF;
    out := out - 'zh-CN';
  END IF;
  RETURN out;
END;
$$;
-- +goose StatementEnd

UPDATE mml_command_groups
SET name_i18n = pg_temp.mml_i18n_long_to_short(name_i18n)
WHERE name_i18n ?| array['en-US', 'zh-CN'];

UPDATE mml_commands
SET command_name_i18n = pg_temp.mml_i18n_long_to_short(command_name_i18n),
    confirm_msg_i18n   = pg_temp.mml_i18n_long_to_short(confirm_msg_i18n),
    logical_name_i18n  = pg_temp.mml_i18n_long_to_short(logical_name_i18n)
WHERE command_name_i18n ?| array['en-US', 'zh-CN']
   OR confirm_msg_i18n   ?| array['en-US', 'zh-CN']
   OR logical_name_i18n  ?| array['en-US', 'zh-CN'];

UPDATE mml_command_sub_fields
SET label_i18n = pg_temp.mml_i18n_long_to_short(label_i18n)
WHERE label_i18n ?| array['en-US', 'zh-CN'];

UPDATE mml_command_sub_field_overrides
SET label_i18n_override = pg_temp.mml_i18n_long_to_short(label_i18n_override)
WHERE label_i18n_override ?| array['en-US', 'zh-CN'];
