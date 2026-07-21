-- +goose Up
-- Existing development databases created before the 2026-07-20 consolidated
-- baseline can have standard_params without updated_fields. The param-model
-- admin API reads this column, so converge those databases in place.
ALTER TABLE standard_params
    ADD COLUMN IF NOT EXISTS updated_fields text[] NOT NULL DEFAULT '{}'::text[];

COMMENT ON COLUMN standard_params.updated_fields IS '最近一次人工编辑发生变化的字段名；新增及尚未人工编辑的记录为空数组';

-- +goose Down
ALTER TABLE standard_params
    DROP COLUMN IF EXISTS updated_fields;
