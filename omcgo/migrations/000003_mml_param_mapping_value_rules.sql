-- +goose Up

ALTER TABLE param_mappings
    ADD COLUMN IF NOT EXISTS default_value text,
    ADD COLUMN IF NOT EXISTS validation_pattern text;

ALTER TABLE discovered_param_mappings
    ADD COLUMN IF NOT EXISTS default_value text,
    ADD COLUMN IF NOT EXISTS validation_pattern text;

COMMENT ON COLUMN param_mappings.default_value IS '参数模型 XML defaultValue；用于 MML 控制台默认填值提示';
COMMENT ON COLUMN param_mappings.validation_pattern IS '参数模型 XML validationPattern；用于 MML 控制台输入校验';
COMMENT ON COLUMN discovered_param_mappings.default_value IS '从 param_mappings 继承的 defaultValue';
COMMENT ON COLUMN discovered_param_mappings.validation_pattern IS '从 param_mappings 继承的 validationPattern';

-- +goose Down

ALTER TABLE discovered_param_mappings
    DROP COLUMN IF EXISTS validation_pattern,
    DROP COLUMN IF EXISTS default_value;

ALTER TABLE param_mappings
    DROP COLUMN IF EXISTS validation_pattern,
    DROP COLUMN IF EXISTS default_value;
